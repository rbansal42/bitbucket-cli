package browse

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/git"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdBrowse creates the browse command
func NewCmdBrowse(streams *iostreams.IOStreams) *cobra.Command {
	var (
		branch     string
		commit     string
		noBrowser  bool
		repo       string
		settings   bool
		wiki       bool
		issues     bool
		prs        bool
		pipelines  bool
		downloads  bool
	)

	cmd := &cobra.Command{
		Use:   "browse [<path>]",
		Short: "Open the repository in the browser",
		Long: `Open the Bitbucket repository in your web browser.

With no arguments, opens the repository's home page. If a path is provided,
opens that file or directory in the repository.

Use flags to open specific sections like issues, pull requests, or settings.`,
		Example: `  # Open repository home page
  bb browse

  # Open a specific file
  bb browse src/main.go

  # Open the issues page
  bb browse --issues

  # Open pull requests page
  bb browse --prs

  # Open repository settings
  bb browse --settings

  # Open a specific branch
  bb browse --branch feature/my-feature

  # Print the URL instead of opening browser
  bb browse --no-browser`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get repository from flag or detect from git
			repoPath := repo
			if repoPath == "" {
				var err error
				repoPath, err = detectRepository()
				if err != nil {
					return fmt.Errorf("could not detect repository: %w\nUse --repo WORKSPACE/REPO to specify", err)
				}
			}

			// Parse workspace and repo name
			parts := strings.SplitN(repoPath, "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected workspace/repo)", repoPath)
			}
			workspace, repoName := parts[0], parts[1]

			// Build the URL
			baseURL := fmt.Sprintf("https://bitbucket.org/%s/%s", workspace, repoName)
			var url string

			switch {
			case settings:
				url = baseURL + "/admin"
			case wiki:
				url = baseURL + "/wiki"
			case issues:
				url = baseURL + "/issues"
			case prs:
				url = baseURL + "/pull-requests"
			case pipelines:
				url = baseURL + "/pipelines"
			case downloads:
				url = baseURL + "/downloads"
			case commit != "":
				url = baseURL + "/commits/" + commit
			case len(args) > 0:
				// Path specified
				path := args[0]
				ref := branch
				if ref == "" {
					// Try to detect current branch, fall back to "main"
					if currentBranch, err := git.GetCurrentBranch(); err == nil {
						ref = currentBranch
					} else {
						ref = "main"
					}
				}
				url = fmt.Sprintf("%s/src/%s/%s", baseURL, ref, path)
			case branch != "":
				url = fmt.Sprintf("%s/src/%s", baseURL, branch)
			default:
				url = baseURL
			}

			// Print or open URL
			if noBrowser {
				fmt.Fprintln(streams.Out, url)
				return nil
			}

			// Get configured browser or use system default
			browser := getBrowser()
			if err := openBrowser(browser, url); err != nil {
				return fmt.Errorf("could not open browser: %w", err)
			}

			streams.Success("Opened %s in your browser", url)
			return nil
		},
	}

	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Open a specific branch")
	cmd.Flags().StringVarP(&commit, "commit", "c", "", "Open a specific commit")
	cmd.Flags().BoolVarP(&noBrowser, "no-browser", "n", false, "Print the URL instead of opening browser")
	cmd.Flags().StringVarP(&repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")
	cmd.Flags().BoolVarP(&settings, "settings", "s", false, "Open repository settings")
	cmd.Flags().BoolVarP(&wiki, "wiki", "w", false, "Open repository wiki")
	cmd.Flags().BoolVar(&issues, "issues", false, "Open issues page")
	cmd.Flags().BoolVar(&prs, "prs", false, "Open pull requests page")
	cmd.Flags().BoolVar(&pipelines, "pipelines", false, "Open pipelines page")
	cmd.Flags().BoolVar(&downloads, "downloads", false, "Open downloads page")

	return cmd
}

// detectRepository attempts to detect the repository from git remote
// Uses the shared git package for URL parsing
func detectRepository() (string, error) {
	remote, err := git.GetDefaultRemote()
	if err != nil {
		return "", fmt.Errorf("not in a git repository or no Bitbucket remote found")
	}

	return remote.Workspace + "/" + remote.RepoSlug, nil
}

// getBrowser returns the configured browser or empty string for system default
func getBrowser() string {
	// Check environment variable
	if browser := os.Getenv("BB_BROWSER"); browser != "" {
		return browser
	}

	// Check config
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Browser != "" {
		return cfg.Browser
	}

	return ""
}

// openBrowser opens a URL in the browser
func openBrowser(browser, url string) error {
	var cmd *exec.Cmd

	if browser != "" {
		cmd = exec.Command(browser, url)
	} else {
		// Use system default
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
		}
	}

	return cmd.Start()
}
