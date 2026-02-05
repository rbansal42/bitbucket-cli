package completion

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdZsh creates the zsh completion command
func NewCmdZsh(streams *iostreams.IOStreams) *cobra.Command {
	var noDescriptions bool

	cmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: `Generate the autocompletion script for zsh.

To load completions in your current shell session:

    source <(bb completion zsh)

To load completions for every new session, execute once:

Linux:
    bb completion zsh > "${fpath[1]}/_bb"

macOS:
    bb completion zsh > $(brew --prefix)/share/zsh/site-functions/_bb

You will need to start a new shell for this setup to take effect.

If shell completion is not already enabled in your environment, you will need
to enable it. You can execute the following once:

    echo "autoload -U compinit; compinit" >> ~/.zshrc`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noDescriptions {
				return cmd.Root().GenZshCompletionNoDesc(streams.Out)
			}
			return cmd.Root().GenZshCompletion(streams.Out)
		},
	}

	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "Disable completion descriptions")

	return cmd
}
