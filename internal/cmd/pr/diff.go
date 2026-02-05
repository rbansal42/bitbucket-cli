package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/iostreams"
)

type diffOptions struct {
	streams *iostreams.IOStreams
	repo    string
	noColor bool
}

// NewCmdDiff creates the diff command
func NewCmdDiff(streams *iostreams.IOStreams) *cobra.Command {
	opts := &diffOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "diff [<number>]",
		Short: "View the diff for a pull request",
		Long: `Display the diff for a pull request.

Shows the changes introduced by the pull request. Color output is enabled
by default when stdout is a terminal, and disabled when piped.`,
		Example: `  # View diff for pull request #123
  bb pr diff 123

  # View diff without color
  bb pr diff 123 --no-color

  # Pipe diff to a file
  bb pr diff 123 > changes.diff`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff(opts, args)
		},
	}

	cmd.Flags().BoolVar(&opts.noColor, "no-color", false, "Disable color output")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runDiff(opts *diffOptions, args []string) error {
	prNum, err := parsePRNumber(args)
	if err != nil {
		return err
	}

	workspace, repoSlug, err := parseRepository(opts.repo)
	if err != nil {
		return err
	}

	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Get the PR to get the diff link
	pr, err := getPullRequest(ctx, client, workspace, repoSlug, prNum)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	// Fetch the diff using the diff link
	diffURL := pr.Links.Diff.Href
	if diffURL == "" {
		diffURL = fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s/pullrequests/%d/diff", workspace, repoSlug, prNum)
	}

	// Make HTTP request for diff (using raw HTTP since it returns text/plain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, diffURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Get token for auth
	token, err := getTokenForRequest()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch diff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch diff: %s", resp.Status)
	}

	// Read diff content
	diffContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read diff: %w", err)
	}

	// Determine if we should colorize
	useColor := opts.streams.IsStdoutTTY() && !opts.noColor

	if useColor {
		colorizedDiff := colorizeDiff(string(diffContent))
		fmt.Fprint(opts.streams.Out, colorizedDiff)
	} else {
		fmt.Fprint(opts.streams.Out, string(diffContent))
	}

	return nil
}

// colorizeDiff adds ANSI colors to a diff
func colorizeDiff(diff string) string {
	var result strings.Builder
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			// File headers - bold
			result.WriteString(iostreams.Bold + line + iostreams.Reset + "\n")
		case strings.HasPrefix(line, "+"):
			// Additions - green
			result.WriteString(iostreams.Green + line + iostreams.Reset + "\n")
		case strings.HasPrefix(line, "-"):
			// Deletions - red
			result.WriteString(iostreams.Red + line + iostreams.Reset + "\n")
		case strings.HasPrefix(line, "@@"):
			// Hunk headers - cyan
			result.WriteString(iostreams.Cyan + line + iostreams.Reset + "\n")
		case strings.HasPrefix(line, "diff "):
			// Diff headers - bold blue
			result.WriteString(iostreams.BoldBlue + line + iostreams.Reset + "\n")
		default:
			result.WriteString(line + "\n")
		}
	}

	return result.String()
}

// getTokenForRequest gets the access token for making requests
func getTokenForRequest() (string, error) {
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return "", err
	}

	user := hosts.GetActiveUser(config.DefaultHost)
	if user == "" {
		return "", fmt.Errorf("not logged in")
	}

	tokenData, _, err := config.GetTokenFromEnvOrKeyring(config.DefaultHost, user)
	if err != nil {
		return "", err
	}

	// Parse token if JSON
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(tokenData), &tokenResp); err == nil && tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, nil
	}

	return tokenData, nil
}
