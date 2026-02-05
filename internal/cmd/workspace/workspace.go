package workspace

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdWorkspace creates the workspace command and its subcommands
func NewCmdWorkspace(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace <command>",
		Short: "Work with Bitbucket workspaces",
		Long: `List, view, and manage Bitbucket workspaces.

Workspaces are where you organize your repositories and collaborate with
your team. Each workspace can contain multiple repositories and projects.`,
		Example: `  # List workspaces you have access to
  bb workspace list

  # View a specific workspace
  bb workspace view myworkspace

  # List members of a workspace
  bb workspace members myworkspace`,
		Aliases: []string{"ws"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdMembers(streams))

	return cmd
}
