package cli

import (
	"github.com/jamesmagoo/pray-cli/internal/rosary"
	"github.com/spf13/cobra"
)

var rosaryCmd = &cobra.Command{
	Use:   "rosary",
	Short: "Pray the rosary, one bead at a time",
	Example: `  pray rosary
  pray rosary --lang la`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rosary.Run(cmd.InOrStdin(), cmd.OutOrStdout(), lang)
	},
}

func init() {
	rootCmd.AddCommand(rosaryCmd)
}
