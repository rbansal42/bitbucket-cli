package pr

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdPR creates the pr command and its subcommands
func NewCmdPR(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr <command>",
		Short: "Work with pull requests",
		Long: `Create, view, and manage pull requests.

Pull requests let you tell others about changes you've pushed to a branch
in a repository. Once a pull request is opened, you can discuss and review
the potential changes with collaborators and add follow-up commits before
your changes are merged.`,
		Example: `  # View a pull request
  bb pr view 123

  # Create a pull request
  bb pr create --title "My feature" --base main

  # List pull requests
  bb pr list`,
		Aliases: []string{"pull-request"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdCreate(streams))
	cmd.AddCommand(NewCmdEdit(streams))
	cmd.AddCommand(NewCmdCheckout(streams))
	cmd.AddCommand(NewCmdMerge(streams))
	cmd.AddCommand(NewCmdClose(streams))
	cmd.AddCommand(NewCmdReopen(streams))
	cmd.AddCommand(NewCmdReview(streams))
	cmd.AddCommand(NewCmdDiff(streams))
	cmd.AddCommand(NewCmdComment(streams))
	cmd.AddCommand(NewCmdChecks(streams))

	return cmd
}
