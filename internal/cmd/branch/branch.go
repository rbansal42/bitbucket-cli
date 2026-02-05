package branch

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdBranch creates the branch command and its subcommands
func NewCmdBranch(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch <command>",
		Short: "Work with repository branches",
		Long: `Create, list, and delete branches in a repository.

Branches allow you to develop features, fix bugs, or safely experiment with
new ideas in a contained area of your repository.`,
		Example: `  # List branches in the current repository
  bb branch list

  # List branches in a specific repository
  bb branch list --repo myworkspace/myrepo

  # Create a new branch from main
  bb branch create feature-branch --target main

  # Delete a branch
  bb branch delete feature-branch`,
		Aliases: []string{"br"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdCreate(streams))
	cmd.AddCommand(NewCmdDelete(streams))

	return cmd
}
