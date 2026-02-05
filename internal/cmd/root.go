package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/cmd/api"
	"github.com/rbansal42/bb/internal/cmd/auth"
	"github.com/rbansal42/bb/internal/cmd/branch"
	"github.com/rbansal42/bb/internal/cmd/browse"
	"github.com/rbansal42/bb/internal/cmd/completion"
	bbconfigcmd "github.com/rbansal42/bb/internal/cmd/config"
	"github.com/rbansal42/bb/internal/cmd/issue"
	"github.com/rbansal42/bb/internal/cmd/pipeline"
	"github.com/rbansal42/bb/internal/cmd/pr"
	"github.com/rbansal42/bb/internal/cmd/project"
	"github.com/rbansal42/bb/internal/cmd/repo"
	"github.com/rbansal42/bb/internal/cmd/snippet"
	"github.com/rbansal42/bb/internal/cmd/workspace"
	"github.com/rbansal42/bb/internal/iostreams"
)

var (
	// Version is set at build time
	Version = "dev"

	// BuildDate is set at build time
	BuildDate = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bb",
	Short: "Bitbucket CLI - Work seamlessly with Bitbucket from the command line",
	Long: `bb is an unofficial command-line interface for Bitbucket Cloud.

It provides commands for working with pull requests, repositories,
issues, pipelines, and more - all from your terminal.

To get started, authenticate with Bitbucket:
  bb auth login

Then you can start using commands like:
  bb pr list
  bb repo clone workspace/repo
  bb issue create`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// streams is the global IOStreams instance
var streams *iostreams.IOStreams

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	streams = iostreams.New()

	err := rootCmd.Execute()
	if err != nil {
		streams.Error("%s", err)
	}
	return err
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringP("repo", "R", "", "Select a repository using the WORKSPACE/REPO format")

	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number of bb",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("bb version %s (%s)\n", Version, BuildDate)
		},
	})

	// Add subcommands
	rootCmd.AddCommand(auth.NewCmdAuth(GetStreams()))
	rootCmd.AddCommand(api.NewCmdAPI(GetStreams()))
	rootCmd.AddCommand(branch.NewCmdBranch(GetStreams()))
	rootCmd.AddCommand(completion.NewCmdCompletion(GetStreams()))
	rootCmd.AddCommand(browse.NewCmdBrowse(GetStreams()))
	rootCmd.AddCommand(bbconfigcmd.NewCmdConfig(GetStreams()))
	rootCmd.AddCommand(issue.NewCmdIssue(GetStreams()))
	rootCmd.AddCommand(pipeline.NewCmdPipeline(GetStreams()))
	rootCmd.AddCommand(pr.NewCmdPR(GetStreams()))
	rootCmd.AddCommand(project.NewCmdProject(GetStreams()))
	rootCmd.AddCommand(repo.NewCmdRepo(GetStreams()))
	rootCmd.AddCommand(snippet.NewCmdSnippet(GetStreams()))
	rootCmd.AddCommand(workspace.NewCmdWorkspace(GetStreams()))
}

// GetStreams returns the global IOStreams instance
func GetStreams() *iostreams.IOStreams {
	if streams == nil {
		streams = iostreams.New()
	}
	return streams
}
