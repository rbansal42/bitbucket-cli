package completion

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdPowerShell creates the powershell completion command
func NewCmdPowerShell(streams *iostreams.IOStreams) *cobra.Command {
	var noDescriptions bool

	cmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell completion script",
		Long: `Generate the autocompletion script for PowerShell.

To load completions in your current shell session:

    bb completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your PowerShell profile.`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noDescriptions {
				return cmd.Root().GenPowerShellCompletion(streams.Out)
			}
			return cmd.Root().GenPowerShellCompletionWithDesc(streams.Out)
		},
	}

	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "Disable completion descriptions")

	return cmd
}
