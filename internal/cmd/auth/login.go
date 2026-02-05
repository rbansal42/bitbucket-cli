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

This command will guide you through authentication setup interactively.
You can choose between:
  - API Token: Simple setup, good for CI/CD and automation
  - OAuth: More secure, supports token refresh

Alternatively, use --with-token to read a token directly from stdin.`,
		Example: `  # Interactive login (recommended)
  $ bb auth login

  # Login with a token from stdin (for CI/CD)
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
	// If --with-token flag is set, read token from stdin
	if opts.withToken {
		return loginWithTokenFromStdin(opts)
	}

	// Interactive flow
	return interactiveLogin(opts)
}

func interactiveLogin(opts *loginOptions) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "Welcome to bb CLI! Let's get you authenticated with Bitbucket.")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "How would you like to authenticate?")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "  [1] API Token (simple, good for CI/CD)")
	fmt.Fprintln(opts.streams.Out, "  [2] OAuth (more secure, supports token refresh)")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprint(opts.streams.Out, "Enter choice [1/2]: ")

	choice, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	choice = strings.TrimSpace(choice)

	var loginErr error
	switch choice {
	case "1":
		loginErr = interactiveAPITokenLogin(opts, reader)
	case "2":
		loginErr = interactiveOAuthLogin(opts, reader)
	default:
		return fmt.Errorf("invalid choice: %s (enter 1 or 2)", choice)
	}

	if loginErr != nil {
		return loginErr
	}

	// After successful login, ask about default workspace
	return promptForDefaultWorkspace(opts, reader)
}

func interactiveAPITokenLogin(opts *loginOptions, reader *bufio.Reader) error {
	const apiTokenURL = "https://id.atlassian.com/manage-profile/security/api-tokens"

	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "=== API Token Authentication ===")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "To create an API token:")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintf(opts.streams.Out, "  1. Opening: %s\n", apiTokenURL)
	fmt.Fprintln(opts.streams.Out, "  2. Click 'Create API token'")
	fmt.Fprintln(opts.streams.Out, "  3. Enter a label (e.g., 'bb-cli')")
	fmt.Fprintln(opts.streams.Out, "  4. Click 'Create' and copy the token")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprint(opts.streams.Out, "Press Enter to open browser (or 'n' to skip): ")

	skipBrowser, _ := reader.ReadString('\n')
	skipBrowser = strings.TrimSpace(strings.ToLower(skipBrowser))

	if skipBrowser != "n" && skipBrowser != "no" {
		if err := browser.Open(apiTokenURL); err != nil {
			opts.streams.Warning("Failed to open browser: %v", err)
			fmt.Fprintf(opts.streams.Out, "Please open manually: %s\n", apiTokenURL)
		}
	}

	// Get email for Basic Auth
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprint(opts.streams.Out, "Enter your Atlassian account email: ")

	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read email: %w", err)
	}
	email = strings.TrimSpace(email)

	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	// Token entry loop with retry on invalid token
	for {
		fmt.Fprintln(opts.streams.Out, "")
		fmt.Fprint(opts.streams.Out, "Paste your API token: ")

		token, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}
		token = strings.TrimSpace(token)

		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}

		// Validate and save the token (using Basic Auth)
		err = validateAndSaveAPIToken(opts, email, token)
		if err == nil {
			return nil // Success!
		}

		// Token validation failed - offer to retry
		fmt.Fprintln(opts.streams.Out, "")
		opts.streams.Error("Token validation failed: %v", err)
		fmt.Fprintln(opts.streams.Out, "")
		fmt.Fprintln(opts.streams.Out, "Please ensure:")
		fmt.Fprintln(opts.streams.Out, "  - Your email is correct")
		fmt.Fprintln(opts.streams.Out, "  - The API token was copied correctly (no extra spaces)")
		fmt.Fprintln(opts.streams.Out, "  - The API token has not been revoked")
		fmt.Fprintln(opts.streams.Out, "")
		fmt.Fprint(opts.streams.Out, "Try again? [Y/n]: ")

		retry, _ := reader.ReadString('\n')
		retry = strings.TrimSpace(strings.ToLower(retry))

		if retry == "n" || retry == "no" {
			return fmt.Errorf("authentication cancelled")
		}
		// Loop continues for retry
	}
}

func interactiveOAuthLogin(opts *loginOptions, reader *bufio.Reader) error {
	// Check if OAuth credentials are already configured
	clientID := os.Getenv("BB_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("BB_OAUTH_CLIENT_SECRET")

	if clientID != "" && clientSecret != "" {
		// Credentials are set, proceed with OAuth flow
		fmt.Fprintln(opts.streams.Out, "")
		fmt.Fprintln(opts.streams.Out, "OAuth credentials found. Starting authentication...")
		return performOAuthFlow(opts, clientID, clientSecret)
	}

	// Need to set up OAuth consumer first
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "=== OAuth Authentication Setup ===")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "OAuth requires a one-time setup of an OAuth consumer in Bitbucket.")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprint(opts.streams.Out, "Enter your workspace name (e.g., 'myteam'): ")

	workspace, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	workspace = strings.TrimSpace(workspace)

	if workspace == "" {
		return fmt.Errorf("workspace name is required")
	}

	// Construct the URL for creating OAuth consumers
	oauthURL := fmt.Sprintf("https://bitbucket.org/%s/workspace/settings/oauth-consumers", workspace)

	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "To create an OAuth consumer:")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintf(opts.streams.Out, "  1. Opening: %s\n", oauthURL)
	fmt.Fprintln(opts.streams.Out, "  2. Click 'Add consumer'")
	fmt.Fprintln(opts.streams.Out, "  3. Fill in:")
	fmt.Fprintln(opts.streams.Out, "     - Name: bb CLI")
	fmt.Fprintln(opts.streams.Out, "     - Callback URL: http://localhost:8372/callback")
	fmt.Fprintln(opts.streams.Out, "     - [x] This is a private consumer")
	fmt.Fprintln(opts.streams.Out, "  4. Select permissions (Account, Repositories, Pull requests, etc.)")
	fmt.Fprintln(opts.streams.Out, "  5. Click 'Save'")
	fmt.Fprintln(opts.streams.Out, "  6. Copy the 'Key' and 'Secret'")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprint(opts.streams.Out, "Press Enter to open browser (or 'n' to skip): ")

	skipBrowser, _ := reader.ReadString('\n')
	skipBrowser = strings.TrimSpace(strings.ToLower(skipBrowser))

	if skipBrowser != "n" && skipBrowser != "no" {
		if err := browser.Open(oauthURL); err != nil {
			opts.streams.Warning("Failed to open browser: %v", err)
			fmt.Fprintf(opts.streams.Out, "Please open manually: %s\n", oauthURL)
		}
	}

	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprint(opts.streams.Out, "Paste your OAuth Key (Client ID): ")
	clientID, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read client ID: %w", err)
	}
	clientID = strings.TrimSpace(clientID)

	if clientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}

	fmt.Fprint(opts.streams.Out, "Paste your OAuth Secret (Client Secret): ")
	clientSecret, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read client secret: %w", err)
	}
	clientSecret = strings.TrimSpace(clientSecret)

	if clientSecret == "" {
		return fmt.Errorf("client secret cannot be empty")
	}

	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "To avoid entering these again, add to your shell profile (~/.zshrc or ~/.bashrc):")
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintf(opts.streams.Out, "  export BB_OAUTH_CLIENT_ID=\"%s\"\n", clientID)
	fmt.Fprintf(opts.streams.Out, "  export BB_OAUTH_CLIENT_SECRET=\"%s\"\n", clientSecret)
	fmt.Fprintln(opts.streams.Out, "")

	// Proceed with OAuth flow
	return performOAuthFlow(opts, clientID, clientSecret)
}

func loginWithTokenFromStdin(opts *loginOptions) error {
	opts.streams.Info("Reading token from stdin...")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("no token provided")
	}

	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		return fmt.Errorf("empty token provided")
	}

	return validateAndSaveToken(opts, token)
}

func validateAndSaveToken(opts *loginOptions, token string) error {
	opts.streams.Info("Validating token...")

	// Validate token by making an API request (Bearer token)
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

	opts.streams.Success("Logged in as: %s (%s)", user.DisplayName, user.Username)
	return nil
}

func validateAndSaveAPIToken(opts *loginOptions, email, apiToken string) error {
	opts.streams.Info("Validating credentials...")

	// Validate using Basic Auth (email:api_token)
	client := api.NewClient(api.WithBasicAuth(email, apiToken))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Store credentials - we store as "email:token" format for Basic Auth
	credentials := email + ":" + apiToken
	if err := config.SetToken(opts.hostname, user.Username, "basic:"+credentials); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
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

	opts.streams.Success("Logged in as: %s (%s)", user.DisplayName, email)
	return nil
}

func promptForDefaultWorkspace(opts *loginOptions, reader *bufio.Reader) error {
	// Check current default workspace
	currentDefault, _ := config.GetDefaultWorkspace()
	if currentDefault != "" {
		fmt.Fprintln(opts.streams.Out, "")
		opts.streams.Info("Current default workspace: %s", currentDefault)
		return nil
	}

	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprint(opts.streams.Out, "Would you like to set a default workspace? [y/N]: ")

	answer, err := reader.ReadString('\n')
	if err != nil {
		return nil // Don't fail login if this fails
	}
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		fmt.Fprintln(opts.streams.Out, "You can set a default workspace later with: bb workspace set-default <workspace>")
		return nil
	}

	// List available workspaces
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "Fetching your workspaces...")

	apiClient, err := getAuthenticatedClient(opts.hostname)
	if err != nil {
		opts.streams.Warning("Could not fetch workspaces: %v", err)
		fmt.Fprintln(opts.streams.Out, "You can set a default workspace later with: bb workspace set-default <workspace>")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := apiClient.ListWorkspaces(ctx, nil)
	if err != nil {
		opts.streams.Warning("Could not fetch workspaces: %v", err)
		fmt.Fprintln(opts.streams.Out, "You can set a default workspace later with: bb workspace set-default <workspace>")
		return nil
	}

	workspaces := result.Values
	if len(workspaces) == 0 {
		opts.streams.Info("No workspaces found")
		return nil
	}

	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprintln(opts.streams.Out, "Available workspaces:")
	for i, membership := range workspaces {
		fmt.Fprintf(opts.streams.Out, "  [%d] %s (%s)\n", i+1, membership.Workspace.Name, membership.Workspace.Slug)
	}
	fmt.Fprintln(opts.streams.Out, "")
	fmt.Fprint(opts.streams.Out, "Enter number to select (or press Enter to skip): ")

	selection, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}
	selection = strings.TrimSpace(selection)

	if selection == "" {
		fmt.Fprintln(opts.streams.Out, "You can set a default workspace later with: bb workspace set-default <workspace>")
		return nil
	}

	var idx int
	if _, err := fmt.Sscanf(selection, "%d", &idx); err != nil || idx < 1 || idx > len(workspaces) {
		opts.streams.Warning("Invalid selection")
		return nil
	}

	selectedWorkspace := workspaces[idx-1].Workspace.Slug
	if err := config.SetDefaultWorkspace(selectedWorkspace); err != nil {
		opts.streams.Warning("Failed to set default workspace: %v", err)
		return nil
	}

	opts.streams.Success("Default workspace set to: %s", selectedWorkspace)
	return nil
}

func getAuthenticatedClient(hostname string) (*api.Client, error) {
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return nil, err
	}

	user := hosts.GetActiveUser(hostname)
	if user == "" {
		return nil, fmt.Errorf("not logged in")
	}

	tokenData, _, err := config.GetTokenFromEnvOrKeyring(hostname, user)
	if err != nil {
		return nil, err
	}

	// Check if this is Basic Auth credentials
	if strings.HasPrefix(tokenData, "basic:") {
		credentials := strings.TrimPrefix(tokenData, "basic:")
		parts := strings.SplitN(credentials, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid credentials format")
		}
		return api.NewClient(api.WithBasicAuth(parts[0], parts[1])), nil
	}

	// Try to parse as JSON (OAuth token)
	var tokenResp oauthTokenResponse
	if err := json.Unmarshal([]byte(tokenData), &tokenResp); err == nil && tokenResp.AccessToken != "" {
		return api.NewClient(api.WithToken(tokenResp.AccessToken)), nil
	}

	return api.NewClient(api.WithToken(tokenData)), nil
}

func performOAuthFlow(opts *loginOptions, clientID, clientSecret string) error {
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

	// Store tokens in keyring (as JSON with refresh token)
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

	opts.streams.Success("Logged in as: %s (%s)", user.DisplayName, user.Username)
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
