package snippet

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdSnippet creates the snippet command and its subcommands
func NewCmdSnippet(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snippet <command>",
		Short: "Work with code snippets",
		Long: `Create, list, view, edit, and delete code snippets.

Snippets are Bitbucket's equivalent to GitHub Gists - small pieces of
code that can be shared and versioned.`,
		Example: `  # List snippets in a workspace
  bb snippet list --workspace myworkspace

  # View a snippet
  bb snippet view abc123 --workspace myworkspace

  # Create a new snippet
  bb snippet create --title "My Snippet" --file script.py --workspace myworkspace`,
		Aliases: []string{"snip"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdCreate(streams))
	cmd.AddCommand(NewCmdEdit(streams))
	cmd.AddCommand(NewCmdDelete(streams))

	return cmd
}
