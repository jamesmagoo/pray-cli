/*
Copyright © 2026 James McGauran
*/
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/jamesmagoo/pray-cli/internal/prayers"
	"github.com/spf13/cobra"
)

var lang string
var intention string

var rootCmd = &cobra.Command{
	Use:          "pray <prayer>",
	Short:        "Prayers in your terminal",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := findPrayer(args)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), p.Plain(intention))
		return nil
	},
}

// findPrayer looks up what the user typed, e.g. ["hail", "mary"].
func findPrayer(args []string) (prayers.Prayer, error) {
	return prayers.Find(strings.Join(args, " "), lang)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pray-cli.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().StringVar(&lang, "lang", "en", "prayer language (en, la)")
	rootCmd.PersistentFlags().StringVar(&intention, "for", "", `fill the prayer's intention, e.g. --for "my mother, a job, the servers deploy"`)
}
