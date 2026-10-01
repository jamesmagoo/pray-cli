package cli

import (
	"bytes"
	"strings"
	"testing"
)

// run executes the root command with args and returns what it printed.
func run(t *testing.T, args ...string) string {
	t.Helper()
	intention, lang = "", "en" // flags are package globals; reset between runs

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
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
