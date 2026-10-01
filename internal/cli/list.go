package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/jamesmagoo/pray-cli/internal/prayers"
	"github.com/spf13/cobra"
)

var listTag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the available prayers",
	Example: `  pray list
  pray list --tag marian
  pray list --lang la`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printList(cmd.OutOrStdout(), listTag)
	},
}

func init() {
	listCmd.Flags().StringVar(&listTag, "tag", "", "only prayers with this tag, e.g. marian")
	rootCmd.AddCommand(listCmd)
}

// printList writes one line per prayer: its id (always works with
// `pray <id>`), its title in the chosen language, and the languages it's
// available in. With a tag, only prayers carrying that tag are listed.
func printList(w io.Writer, tag string) error {
	ids, err := prayers.IDs()
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	for _, id := range ids {
		p, err := prayers.Get(id, lang)
		if err != nil {
			return err
		}
		if tag != "" && !slices.Contains(p.Tags, tag) {
			continue
		}
		langs, err := prayers.Langs(id)
		if err != nil {
			return err
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", id, p.Title, strings.Join(langs, " "))
	}
	return tw.Flush()
}
