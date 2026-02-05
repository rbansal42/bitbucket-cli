package repo

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdRepo creates the repo command and its subcommands
func NewCmdRepo(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo <command>",
		Short: "Work with repositories",
		Long: `Create, clone, fork, and view repositories.

Repositories contain all of your project's files and each file's revision
history. You can discuss and manage your project's work within the repository.`,
		Example: `  # View the current repository
  bb repo view

  # View a specific repository
  bb repo view myworkspace/myrepo

  # Clone a repository
  bb repo clone myworkspace/myrepo

  # List repositories in a workspace
  bb repo list --workspace myworkspace`,
		Aliases: []string{"repository"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdClone(streams))
	cmd.AddCommand(NewCmdCreate(streams))
	cmd.AddCommand(NewCmdFork(streams))
	cmd.AddCommand(NewCmdDelete(streams))
	cmd.AddCommand(NewCmdSync(streams))
	cmd.AddCommand(NewCmdSetDefault(streams))

	return cmd
}
