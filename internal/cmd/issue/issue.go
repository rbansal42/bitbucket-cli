package issue

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdIssue creates the issue command and its subcommands
func NewCmdIssue(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue <command>",
		Short: "Manage issues",
		Long: `Create, view, list, and manage issues.

Issues track bugs, enhancements, proposals, and tasks in your repository.
Use these commands to create new issues, view existing ones, and manage
their lifecycle.`,
		Example: `  # List open issues
  bb issue list

  # View an issue
  bb issue view 123

  # Create a new issue
  bb issue create --title "Bug: login fails"

  # Close an issue
  bb issue close 123

  # Add a comment
  bb issue comment 123 --body "Working on this"`,
		Aliases: []string{"issues"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdCreate(streams))
	cmd.AddCommand(NewCmdEdit(streams))
	cmd.AddCommand(NewCmdComment(streams))
	cmd.AddCommand(NewCmdClose(streams))
	cmd.AddCommand(NewCmdReopen(streams))
	cmd.AddCommand(NewCmdDelete(streams))

	return cmd
}
