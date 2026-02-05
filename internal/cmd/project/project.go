package project

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdProject creates the project command and its subcommands
func NewCmdProject(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project <command>",
		Short: "Work with Bitbucket projects",
		Long: `Create, view, and manage Bitbucket projects.

Projects provide a way to group repositories within a workspace. You can
organize related repositories together and manage access permissions at
the project level.`,
		Example: `  # List projects in a workspace
  bb project list --workspace myworkspace

  # View a specific project
  bb project view PROJ --workspace myworkspace

  # Create a new project
  bb project create --workspace myworkspace --key PROJ --name "My Project"`,
		Aliases: []string{"proj"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdCreate(streams))

	return cmd
}
