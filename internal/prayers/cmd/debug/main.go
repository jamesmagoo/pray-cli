// Command debug prints every prayer and its metadata, for development use.
// Run via `just debug`.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jamesmagoo/pray-cli/internal/prayers"
)

func main() {
	ids, err := prayers.IDs()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	for i, id := range ids {
		if i > 0 {
			fmt.Println(strings.Repeat("-", 40))
		}
		p, err := prayers.Get(id, "en")
		if err != nil {
			fmt.Printf("%s: error: %v\n", id, err)
			continue
		}
		fmt.Printf("id:        %s\n", p.ID)
		fmt.Printf("title:     %s\n", p.Title)
		fmt.Printf("lang:      %s\n", p.Lang)
		fmt.Printf("intention: %s\n", p.Intention)
		fmt.Printf("aliases:   %s\n", strings.Join(p.Aliases, ", "))
		fmt.Printf("tags:      %s\n", strings.Join(p.Tags, ", "))
		fmt.Printf("body:\n%s\n", p.Body)
	}
}
