package repo

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/rbansal42/bb/internal/git"
	"github.com/rbansal42/bb/internal/iostreams"
)

// LocalConfig represents the .bb.yml file structure
type LocalConfig struct {
	DefaultRepo string `yaml:"default_repo,omitempty"`
}

// SetDefaultOptions holds the options for the set-default command
type SetDefaultOptions struct {
	RepoArg string
	View    bool
	Unset   bool
	Streams *iostreams.IOStreams
}

// NewCmdSetDefault creates the repo set-default command
func NewCmdSetDefault(streams *iostreams.IOStreams) *cobra.Command {
	opts := &SetDefaultOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "set-default [<workspace/repo>]",
		Short: "Set the default repository for the current directory",
		Long: `Set the default repository for the current directory.

The default repository is stored in a .bb.yml file in the current directory,
or in git config (bb.repo) if inside a git repository.

This default is used when no repository is specified for commands that
require a repository context.`,
		Example: `  # Set default repository
  bb repo set-default myworkspace/myrepo

  # Detect from git remote and set as default
  bb repo set-default

  # View current default repository
  bb repo set-default --view

  # Remove default repository
  bb repo set-default --unset`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RepoArg = args[0]
			}
			return runSetDefault(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVar(&opts.View, "view", false, "Show the current default repository")
	cmd.Flags().BoolVar(&opts.Unset, "unset", false, "Remove the default repository")

	return cmd
}

func runSetDefault(ctx context.Context, opts *SetDefaultOptions) error {
	// Check for mutually exclusive flags
	if opts.View && opts.Unset {
		return fmt.Errorf("cannot specify both --view and --unset")
	}

	// Handle --view flag
	if opts.View {
		return viewDefault(opts)
	}

	// Handle --unset flag
	if opts.Unset {
		return unsetDefault(opts)
	}

	// Determine the repository to set
	var workspace, repoSlug string
	var err error

	if opts.RepoArg != "" {
		// Parse provided argument
		workspace, repoSlug, err = parseRepoArg(opts.RepoArg)
		if err != nil {
			return err
		}
	} else {
		// Detect from git remote
		remote, err := git.GetDefaultRemote()
		if err != nil {
			return fmt.Errorf("could not detect repository from git remote: %w\nProvide repository as argument: bb repo set-default <workspace/repo>", err)
		}

		workspace = remote.Workspace
		repoSlug = remote.RepoSlug

		// Require TTY for interactive confirmation
		if !opts.Streams.IsStdinTTY() {
			return fmt.Errorf("cannot confirm: stdin is not a terminal\nProvide repository as argument: bb repo set-default <workspace/repo>")
		}

		// Confirm with user
		fullRepo := fmt.Sprintf("%s/%s", workspace, repoSlug)
		if !confirmSetDefault(opts.Streams, fullRepo) {
			opts.Streams.Info("Aborted")
			return nil
		}
	}

	// Validate the repository exists (optional - requires API call)
	fullRepo := fmt.Sprintf("%s/%s", workspace, repoSlug)

	// Try to validate repository exists if authenticated
	client, err := getAPIClient()
	if err == nil {
		validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		_, err := client.GetRepository(validateCtx, workspace, repoSlug)
		if err != nil {
			opts.Streams.Warning("Could not verify repository exists: %s", err)
		}
	}

	// Store the default
	if err := storeDefault(fullRepo); err != nil {
		return err
	}

	opts.Streams.Success("Set default repository to %s", fullRepo)
	return nil
}

func viewDefault(opts *SetDefaultOptions) error {
	repo, source, err := getDefault()
	if err != nil {
		return err
	}

	if repo == "" {
		opts.Streams.Info("No default repository set")
		return nil
	}

	opts.Streams.Info("Default repository: %s (from %s)", repo, source)
	return nil
}

func unsetDefault(opts *SetDefaultOptions) error {
	// Check if we're in a git repo
	if git.IsGitRepository() {
		// Try to unset from git config
		if err := unsetGitConfig(); err != nil {
			// Git config might not have been set, try .bb.yml
			if err := removeLocalConfig(); err != nil {
				opts.Streams.Info("No default repository was set")
				return nil
			}
		}
	} else {
		// Not in git repo, try .bb.yml
		if err := removeLocalConfig(); err != nil {
			opts.Streams.Info("No default repository was set")
			return nil
		}
	}

	opts.Streams.Success("Removed default repository")
	return nil
}

func storeDefault(repo string) error {
	// If in git repository, use git config
	if git.IsGitRepository() {
		return setGitConfig(repo)
	}

	// Otherwise, use .bb.yml
	return setLocalConfig(repo)
}

func getDefault() (repo string, source string, err error) {
	// First, check git config if in a git repo
	if git.IsGitRepository() {
		repo, err = getGitConfig()
		if err == nil && repo != "" {
			return repo, "git config", nil
		}
	}

	// Then, check .bb.yml in current directory
	repo, err = getLocalConfig()
	if err == nil && repo != "" {
		return repo, ".bb.yml", nil
	}

	return "", "", nil
}

func setGitConfig(repo string) error {
	cmd := execCommand("git", "config", "--local", "bb.repo", repo)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git config: %w", err)
	}
	return nil
}

func getGitConfig() (string, error) {
	cmd := execCommand("git", "config", "--local", "--get", "bb.repo")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func unsetGitConfig() error {
	cmd := execCommand("git", "config", "--local", "--unset", "bb.repo")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unset git config: %w", err)
	}
	return nil
}

func setLocalConfig(repo string) error {
	config := LocalConfig{
		DefaultRepo: repo,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(".", ".bb.yml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write .bb.yml: %w", err)
	}

	return nil
}

func getLocalConfig() (string, error) {
	configPath := filepath.Join(".", ".bb.yml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	var config LocalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", err
	}

	return config.DefaultRepo, nil
}

func removeLocalConfig() error {
	configPath := filepath.Join(".", ".bb.yml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("no .bb.yml file found")
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config LocalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		// File exists but invalid, just remove it
		return os.Remove(configPath)
	}

	// If only default_repo was set, remove the file
	config.DefaultRepo = ""

	// Check if config is now empty
	newData, _ := yaml.Marshal(config)
	if strings.TrimSpace(string(newData)) == "{}" || strings.TrimSpace(string(newData)) == "" {
		return os.Remove(configPath)
	}

	// Otherwise, write back without the default_repo
	return os.WriteFile(configPath, newData, 0644)
}

func confirmSetDefault(streams *iostreams.IOStreams, repo string) bool {
	fmt.Fprintf(streams.Out, "Set default repository to %s? [Y/n] ", repo)

	var reader *bufio.Reader
	if r, ok := streams.In.(io.Reader); ok {
		reader = bufio.NewReader(r)
	} else {
		reader = bufio.NewReader(os.Stdin)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "" || input == "y" || input == "yes"
}

// execCommand is a wrapper for exec.Command to allow testing
var execCommand = execCommandImpl

func execCommandImpl(name string, args ...string) *execCmd {
	return &execCmd{name: name, args: args}
}

type execCmd struct {
	name string
	args []string
}

func (c *execCmd) Run() error {
	cmd := newOSExecCommand(c.name, c.args...)
	return cmd.Run()
}

func (c *execCmd) Output() ([]byte, error) {
	cmd := newOSExecCommand(c.name, c.args...)
	return cmd.Output()
}

// newOSExecCommand wraps os/exec.Command
func newOSExecCommand(name string, args ...string) osExecCommand {
	return &realExecCommand{name: name, args: args}
}

type osExecCommand interface {
	Run() error
	Output() ([]byte, error)
}

type realExecCommand struct {
	name string
	args []string
}

func (c *realExecCommand) Run() error {
	cmd := exec.Command(c.name, c.args...)
	return cmd.Run()
}

func (c *realExecCommand) Output() ([]byte, error) {
	cmd := exec.Command(c.name, c.args...)
	return cmd.Output()
}
