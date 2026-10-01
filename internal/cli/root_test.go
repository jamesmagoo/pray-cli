package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/jamesmagoo/pray-cli/internal/prayers"
)

// run executes the root command with args and returns what it printed.
func run(t *testing.T, args ...string) string {
	t.Helper()
	intention, lang, listTag = "", "en", "" // flags are package globals; reset between runs

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	if args == nil {
		args = []string{} // nil would make Cobra read the test binary's own os.Args
	}
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pray %s: %v", strings.Join(args, " "), err)
	}
	return out.String()
}

func TestIntentionLineForPrayerWithoutSlot(t *testing.T) {
	out := run(t, "hail", "mary", "--for", "my mum")
	if !strings.Contains(out, "\nFor my mum\n") {
		t.Fatalf("expected an intention line, got:\n%s", out)
	}
}

func TestNoIntentionLineByDefault(t *testing.T) {
	out := run(t, "hail", "mary")
	if strings.Contains(out, "\nFor ") {
		t.Fatalf("unexpected intention line:\n%s", out)
	}
}

func TestIntentionWovenIntoSlot(t *testing.T) {
	out := run(t, "carlo", "--for", "the deploy")
	if !strings.Contains(out, "specifically for the deploy.") {
		t.Fatalf("intention not filled into the prayer:\n%s", out)
	}
	if strings.Contains(out, "\nFor the deploy\n") {
		t.Fatalf("intention shown twice:\n%s", out)
	}
}

// pray copy must always put plain text on the clipboard, whatever pray
// prints in a terminal: no colour codes, and no frames or block-character
// crosses from the renderer.
func TestCopyIsPlainText(t *testing.T) {
	var copied string
	original := writeClipboard
	writeClipboard = func(s string) error { copied = s; return nil }
	t.Cleanup(func() { writeClipboard = original })

	out := run(t, "copy", "hail", "mary", "--for", "my mum")

	p, err := prayers.Find("hail mary", "en")
	if err != nil {
		t.Fatal(err)
	}
	if copied != p.Plain("my mum") {
		t.Fatalf("clipboard isn't the prayer's plain text:\n%s", copied)
	}
	if strings.Contains(copied, "\x1b") {
		t.Fatal("clipboard contains terminal colour codes")
	}
	isDecoration := func(r rune) bool { return r >= 0x2500 && r <= 0x259F } // ═ ║ █ …
	if strings.IndexFunc(copied, isDecoration) >= 0 {
		t.Fatal("clipboard contains frame or block characters")
	}
	if !strings.Contains(out, "Copied Hail Mary") || strings.Contains(out, "full of grace") {
		t.Fatalf("want only a confirmation, got:\n%s", out)
	}
}

// Cobra checks the first word for a subcommand before any prayer lookup, so
// a prayer whose id, alias or title starts with a subcommand name ("list",
// "copy", "rosary"…) could never be reached that way. Subcommands are read
// from rootCmd, so this stays in sync as you add them.
func TestNoPrayerHiddenBySubcommand(t *testing.T) {
	reserved := []string{"help", "completion"} // added by Cobra itself
	for _, c := range rootCmd.Commands() {
		reserved = append(reserved, c.Name())
		reserved = append(reserved, c.Aliases...)
	}

	ids, err := prayers.IDs()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		p, err := prayers.Get(id, "en")
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range append([]string{id, p.Title}, p.Aliases...) {
			words := strings.Fields(name)
			if len(words) == 0 {
				continue
			}
			first := strings.ToLower(words[0])
			if slices.Contains(reserved, first) {
				t.Errorf("prayer %s: %q starts with the subcommand %q", id, name, first)
			}
		}
	}
}
