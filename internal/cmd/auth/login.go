package auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/browser"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

const (
	// OAuth endpoints
	authorizationURL = "https://bitbucket.org/site/oauth2/authorize"
	tokenURL         = "https://bitbucket.org/site/oauth2/access_token"

	// Default scopes for bb CLI
	defaultScopes = "account repository repository:write pullrequest pullrequest:write issue issue:write pipeline pipeline:write snippet snippet:write webhook"

	// Callback path for OAuth redirect
	callbackPath = "/callback"
)

type loginOptions struct {
	streams   *iostreams.IOStreams
	withToken bool
	hostname  string
	scopes    string
}

// NewCmdLogin creates the login command
func NewCmdLogin(streams *iostreams.IOStreams) *cobra.Command {
	opts := &loginOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Bitbucket",
		Long: `Authenticate with Bitbucket to enable API access.

The default authentication mode is interactive and uses OAuth 2.0.
This will open a browser window for you to authorize bb.

Alternatively, you can use --with-token to read a token from stdin.
This is useful for automation or when using workspace/repository access tokens.`,
		Example: `  # Interactive OAuth login
  $ bb auth login

  # Login with an access token
  $ echo "your_token" | bb auth login --with-token

  # Login with a token from a file
  $ bb auth login --with-token < token.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.withToken, "with-token", false, "Read token from stdin")
	cmd.Flags().StringVar(&opts.hostname, "hostname", config.DefaultHost, "Bitbucket hostname")
	cmd.Flags().StringVar(&opts.scopes, "scopes", defaultScopes, "OAuth scopes to request")

	return cmd
}

func runLogin(opts *loginOptions) error {
	if opts.withToken {
		return loginWithToken(opts)
	}
	return loginWithOAuth(opts)
}

func loginWithToken(opts *loginOptions) error {
	opts.streams.Info("Paste your access token:")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("no token provided")
	}

	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		return fmt.Errorf("empty token provided")
	}

	// Validate token by making an API request
	client := api.NewClient(api.WithToken(token))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Store token in keyring
	if err := config.SetToken(opts.hostname, user.Username, token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Update hosts config
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return fmt.Errorf("failed to load hosts config: %w", err)
	}

	hosts.SetActiveUser(opts.hostname, user.Username)

	if err := config.SaveHostsConfig(hosts); err != nil {
		return fmt.Errorf("failed to save hosts config: %w", err)
	}

	opts.streams.Success("Logged in as %s", user.Username)
	return nil
}

func loginWithOAuth(opts *loginOptions) error {
	// Check if OAuth client credentials are available
	clientID := os.Getenv("BB_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("BB_OAUTH_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf(`OAuth client credentials not configured.

To use OAuth authentication, you need to create an OAuth consumer in Bitbucket:

1. Go to your Workspace settings > OAuth consumers
   (https://bitbucket.org/YOUR_WORKSPACE/workspace/settings/oauth-consumers)
   
2. Click "Add consumer" and configure:
   - Name: bb CLI
   - Callback URL: http://localhost:8372/callback
   - Permissions: Account (Read), Repositories (Read/Write), 
     Pull requests (Read/Write), Pipelines (Read/Write), etc.
   
3. After creating, copy the Key and Secret, then set environment variables:
   export BB_OAUTH_CLIENT_ID="your_key"
   export BB_OAUTH_CLIENT_SECRET="your_secret"

4. Run 'bb auth login' again

Alternatively, use --with-token to authenticate with a Repository Access Token:
  bb auth login --with-token`)
	}

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Start local server to receive callback
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("failed to start local server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://localhost:%d%s", port, callbackPath)

	// Build authorization URL
	authURL, err := url.Parse(authorizationURL)
	if err != nil {
		return err
	}

	q := authURL.Query()
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", callbackURL)
	q.Set("state", state)
	// Note: Bitbucket doesn't use scope parameter in the same way as GitHub
	// Permissions are set in the OAuth consumer configuration
	authURL.RawQuery = q.Encode()

	// Channel to receive authorization code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Set up HTTP server
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != callbackPath {
				http.NotFound(w, r)
				return
			}

			// Verify state
			if r.URL.Query().Get("state") != state {
				errChan <- fmt.Errorf("state mismatch, possible CSRF attack")
				http.Error(w, "State mismatch", http.StatusBadRequest)
				return
			}

			// Check for error
			if errMsg := r.URL.Query().Get("error"); errMsg != "" {
				errDesc := r.URL.Query().Get("error_description")
				errChan <- fmt.Errorf("authorization failed: %s - %s", errMsg, errDesc)
				http.Error(w, errMsg, http.StatusBadRequest)
				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				errChan <- fmt.Errorf("no authorization code received")
				http.Error(w, "No code received", http.StatusBadRequest)
				return
			}

			// Success response
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>bb - Authentication Successful</title></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #0052cc;">
<div style="text-align: center; color: white;">
<h1>Authentication Successful!</h1>
<p>You can close this window and return to the terminal.</p>
</div>
</body>
</html>`)

			codeChan <- code
		}),
	}

	// Start server in background
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Open browser
	opts.streams.Info("Opening browser for authentication...")
	opts.streams.Info("If browser doesn't open, visit: %s", authURL.String())

	if err := browser.Open(authURL.String()); err != nil {
		opts.streams.Warning("Failed to open browser: %v", err)
	}

	opts.streams.Info("Waiting for authentication...")

	// Wait for code or error
	var code string
	select {
	case code = <-codeChan:
		// Success
	case err := <-errChan:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authentication timed out")
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	// Exchange code for token
	opts.streams.Info("Exchanging authorization code for token...")

	tokenResp, err := exchangeCodeForToken(clientID, clientSecret, code, callbackURL)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Validate token and get user info
	client := api.NewClient(api.WithToken(tokenResp.AccessToken))
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Store tokens in keyring
	// We store both access and refresh tokens as JSON
	tokenData, err := json.Marshal(tokenResp)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := config.SetToken(opts.hostname, user.Username, string(tokenData)); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Update hosts config
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return fmt.Errorf("failed to load hosts config: %w", err)
	}

	hosts.SetActiveUser(opts.hostname, user.Username)

	if err := config.SaveHostsConfig(hosts); err != nil {
		return fmt.Errorf("failed to save hosts config: %w", err)
	}

	opts.streams.Success("Logged in as %s", user.Username)
	return nil
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scopes       string `json:"scopes"`
}

func exchangeCodeForToken(clientID, clientSecret, code, redirectURI string) (*oauthTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
