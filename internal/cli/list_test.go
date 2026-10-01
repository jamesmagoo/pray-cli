package cli

import (
	"strings"
	"testing"

	"github.com/jamesmagoo/pray-cli/internal/prayers"
)

func TestListShowsEveryPrayer(t *testing.T) {
	out := run(t, "list")

	ids, err := prayers.IDs()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		p, err := prayers.Get(id, "en")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, id) || !strings.Contains(out, p.Title) {
			t.Errorf("%s (%s) missing from list:\n%s", id, p.Title, out)
		}
	}
}

func TestListByTag(t *testing.T) {
	out := run(t, "list", "--tag", "marian")
	if !strings.Contains(out, "hail-mary") || strings.Contains(out, "st-carlo-acutis") {
		t.Fatalf("want only marian prayers, got:\n%s", out)
	}
}

func TestListInLatin(t *testing.T) {
	out := run(t, "list", "--lang", "la")
	if !strings.Contains(out, "Ave Maria") {
		t.Fatalf("want Latin titles, got:\n%s", out)
	}
	// Prayers without a Latin text keep their English title.
	if !strings.Contains(out, "Prayer of Intercession for Technical Problems") {
		t.Fatalf("want English fallback title, got:\n%s", out)
	}
}

func TestBarePrayShowsList(t *testing.T) {
	out := run(t)
	if !strings.Contains(out, "hail-mary") || !strings.Contains(out, "pray <name>") {
		t.Fatalf("want the list and a hint, got:\n%s", out)
	}
}
