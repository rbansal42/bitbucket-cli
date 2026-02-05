package pipeline

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdPipeline creates the pipeline command and its subcommands
func NewCmdPipeline(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline <command>",
		Short: "Manage pipelines",
		Long: `View and manage Bitbucket Pipelines.

Pipelines are Bitbucket's integrated CI/CD service that allows you to automatically
build, test, and deploy your code based on a configuration file in your repository.`,
		Example: `  # List pipelines
  bb pipeline list

  # View a specific pipeline
  bb pipeline view 123

  # List failed pipelines
  bb pipeline list --status FAILED

  # View pipeline in browser
  bb pipeline view 123 --web

  # List steps in a pipeline
  bb pipeline steps 123

  # View step logs
  bb pipeline logs 123 --step 2`,
		Aliases: []string{"pipelines"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdRun(streams))
	cmd.AddCommand(NewCmdStop(streams))
	cmd.AddCommand(NewCmdSteps(streams))
	cmd.AddCommand(NewCmdLogs(streams))

	return cmd
}
