package cli

import (
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

// writeClipboard is replaced in tests, which can't rely on a real clipboard.
var writeClipboard = clipboard.WriteAll

// copyCmd always copies plain text (Prayer.Plain), never the rendered
// terminal view: no colour codes, frames or crosses end up on the clipboard.
var copyCmd = &cobra.Command{
	Use:   "copy <prayer>",
	Short: "Copy a prayer to the clipboard",
	Example: `  pray copy hail mary
  pray copy carlo --for "the server migration"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := findPrayer(args)
		if err != nil {
			return err
		}

		if err := writeClipboard(p.Plain(intention)); err != nil {
			return fmt.Errorf("couldn't copy to the clipboard: %w", err)
		}

		// Confirmation goes to stderr, keeping stdout empty for scripts.
		fmt.Fprintf(cmd.ErrOrStderr(), "Copied %s to the clipboard\n", p.Title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
