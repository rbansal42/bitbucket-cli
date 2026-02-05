package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdAPI creates the api command
func NewCmdAPI(streams *iostreams.IOStreams) *cobra.Command {
	var (
		method      string
		headers     []string
		inputFile   string
		rawFields   []string
		jsonFields  []string
		silent      bool
		includeResp bool
		paginate    bool
	)

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make an authenticated Bitbucket API request",
		Long: `Make an authenticated request to the Bitbucket API.

The endpoint argument should be the path of the API endpoint to call,
such as "repositories/workspace/repo" or "user".

The default HTTP method is GET. Pass --method to specify a different method.

Placeholder values in the endpoint will be substituted with values from
the current repository context when available.

Pass request body using --field for URL-encoded data, --json for JSON data,
or --input for reading from a file.`,
		Example: `  # Get the current user
  bb api user

  # List repositories in a workspace
  bb api repositories/myworkspace

  # Create an issue
  bb api repositories/myworkspace/myrepo/issues --method POST \
    --json title="Bug report" --json priority="major"

  # Get raw response with headers
  bb api user --include`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			endpoint := args[0]

			// Ensure endpoint starts with a slash or is a full URL
			if !strings.HasPrefix(endpoint, "/") && !strings.HasPrefix(endpoint, "http") {
				endpoint = "/" + endpoint
			}

			// Construct full URL if not already a full URL
			var url string
			if strings.HasPrefix(endpoint, "http") {
				url = endpoint
			} else {
				url = "https://api.bitbucket.org/2.0" + endpoint
			}

			// Get authentication token
			token, err := getAuthToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'bb auth login' to authenticate", err)
			}

			// Prepare request body
			var body io.Reader
			contentType := ""

			if inputFile != "" {
				// Read from file
				if inputFile == "-" {
					body = os.Stdin
				} else {
					data, err := os.ReadFile(inputFile)
					if err != nil {
						return fmt.Errorf("could not read input file: %w", err)
					}
					body = bytes.NewReader(data)
				}
				contentType = "application/json"
			} else if len(jsonFields) > 0 {
				// Build JSON body from fields
				jsonBody := make(map[string]interface{})
				for _, field := range jsonFields {
					parts := strings.SplitN(field, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid json field format: %s (expected key=value)", field)
					}
					jsonBody[parts[0]] = parts[1]
				}
				data, err := json.Marshal(jsonBody)
				if err != nil {
					return fmt.Errorf("could not encode JSON: %w", err)
				}
				body = bytes.NewReader(data)
				contentType = "application/json"
			} else if len(rawFields) > 0 {
				// Build form body from fields
				var formParts []string
				for _, field := range rawFields {
					formParts = append(formParts, field)
				}
				body = strings.NewReader(strings.Join(formParts, "&"))
				contentType = "application/x-www-form-urlencoded"
			}

			// Create request
			req, err := http.NewRequest(strings.ToUpper(method), url, body)
			if err != nil {
				return fmt.Errorf("could not create request: %w", err)
			}

			// Set headers
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "application/json")
			if contentType != "" {
				req.Header.Set("Content-Type", contentType)
			}

			// Add custom headers
			for _, h := range headers {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header format: %s (expected Header:Value)", h)
				}
				req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}

			// Execute request
			client := &http.Client{
				Timeout: 30 * time.Second,
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			// Handle pagination if requested
			if paginate && resp.StatusCode == http.StatusOK {
				return handlePagination(streams, client, req, resp, token, includeResp, silent)
			}

			// Print response headers if requested
			if includeResp {
				fmt.Fprintf(streams.Out, "%s %s\n", resp.Proto, resp.Status)
				for key, values := range resp.Header {
					for _, value := range values {
						fmt.Fprintf(streams.Out, "%s: %s\n", key, value)
					}
				}
				fmt.Fprintln(streams.Out)
			}

			// Read and print response body
			if !silent {
				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("could not read response: %w", err)
				}

				// Pretty-print JSON if possible
				if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
					var prettyJSON bytes.Buffer
					if err := json.Indent(&prettyJSON, respBody, "", "  "); err == nil {
						fmt.Fprintln(streams.Out, prettyJSON.String())
					} else {
						fmt.Fprintln(streams.Out, string(respBody))
					}
				} else {
					fmt.Fprintln(streams.Out, string(respBody))
				}
			}

			// Return error for non-2xx status codes
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("API request failed with status %d", resp.StatusCode)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method to use")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Add a custom header (can be specified multiple times)")
	cmd.Flags().StringVar(&inputFile, "input", "", "Read request body from file (use - for stdin)")
	cmd.Flags().StringArrayVarP(&rawFields, "field", "f", nil, "Add a URL-encoded field (can be specified multiple times)")
	cmd.Flags().StringArrayVarP(&jsonFields, "json", "j", nil, "Add a JSON field (can be specified multiple times)")
	cmd.Flags().BoolVarP(&silent, "silent", "s", false, "Do not print response body")
	cmd.Flags().BoolVarP(&includeResp, "include", "i", false, "Include response headers in output")
	cmd.Flags().BoolVar(&paginate, "paginate", false, "Automatically fetch all pages of results")

	return cmd
}

// getAuthToken retrieves the authentication token
func getAuthToken() (string, error) {
	// Check environment variables first (BB_TOKEN takes precedence)
	if token := os.Getenv("BB_TOKEN"); token != "" {
		return token, nil
	}
	if token := os.Getenv("BITBUCKET_TOKEN"); token != "" {
		return token, nil
	}

	// Load hosts config to get active user
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return "", err
	}

	user := hosts.GetActiveUser(config.DefaultHost)
	if user == "" {
		return "", fmt.Errorf("no authenticated user found")
	}

	// Get token from keyring
	token, err := config.GetToken(config.DefaultHost, user)
	if err != nil {
		return "", err
	}

	return token, nil
}

// handlePagination handles paginated responses
func handlePagination(streams *iostreams.IOStreams, client *http.Client, originalReq *http.Request, firstResp *http.Response, token string, includeResp, silent bool) error {
	type paginatedResponse struct {
		Values []json.RawMessage `json:"values"`
		Next   string            `json:"next"`
	}

	allValues := []json.RawMessage{}

	// Process first response
	if includeResp {
		fmt.Fprintf(streams.Out, "%s %s\n", firstResp.Proto, firstResp.Status)
		for key, values := range firstResp.Header {
			for _, value := range values {
				fmt.Fprintf(streams.Out, "%s: %s\n", key, value)
			}
		}
		fmt.Fprintln(streams.Out)
	}

	body, err := io.ReadAll(firstResp.Body)
	if err != nil {
		return fmt.Errorf("could not read response: %w", err)
	}

	var page paginatedResponse
	if err := json.Unmarshal(body, &page); err != nil {
		// Not a paginated response, just print it
		if !silent {
			fmt.Fprintln(streams.Out, string(body))
		}
		return nil
	}

	allValues = append(allValues, page.Values...)
	nextURL := page.Next

	// Fetch remaining pages
	for nextURL != "" {
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return fmt.Errorf("could not create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("could not read response: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}

		if err := json.Unmarshal(body, &page); err != nil {
			return fmt.Errorf("could not parse response: %w", err)
		}

		allValues = append(allValues, page.Values...)
		nextURL = page.Next
	}

	// Print all values
	if !silent {
		result, err := json.MarshalIndent(allValues, "", "  ")
		if err != nil {
			return fmt.Errorf("could not encode results: %w", err)
		}
		fmt.Fprintln(streams.Out, string(result))
	}

	return nil
}
