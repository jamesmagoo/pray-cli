package prayers

import (
	"errors"
	"slices"
	"testing"
)

// These tests pin down the matching rules using invented prayers, so they
// never change when real prayers are added. The real prayers are checked
// separately in data_test.go.
var fixture = []Prayer{
	{ID: "angelus", Title: "The Angelus"},
	{ID: "chaplet-st-michael", Title: "Chaplet of St. Michael"},
	{ID: "hail-holy-queen", Title: "Hail, Holy Queen", Aliases: []string{"salve regina"}},
	{ID: "hail-mary", Title: "Hail Mary", Aliases: []string{"ave"}},
	{ID: "magnificat", Title: "Magnificat (Canticle of Mary)"},
	{ID: "regina-caeli", Title: "Regina Caeli", Aliases: []string{"angelus"}}, // said instead of the Angelus at Easter
	{ID: "st-joseph", Title: "St. Joseph's Prayer for Workers"},
	{ID: "st-michael", Title: "Prayer to St. Michael the Archangel", Aliases: []string{"michael"}},
}

func TestMatchRules(t *testing.T) {
	cases := []struct{ query, want, rule string }{
		// tier 1: id
		{"hail-mary", "hail-mary", "exact id"},
		{"Hail, Holy Queen", "hail-holy-queen", "id, ignoring case and punctuation"},
		{"Saint Joseph", "st-joseph", "saint is the same as st"},
		{"angelus", "angelus", "an id beats another prayer's alias"},
		// tier 2: title or alias
		{"ave", "hail-mary", "exact alias"},
		{"salve regina", "hail-holy-queen", "multi-word alias"},
		{"michael", "st-michael", "an alias beats another prayer's word match"},
		// tier 3: word prefixes
		{"archangel", "st-michael", "a word from the title"},
		{"chap", "chaplet-st-michael", "the start of a word"},
		{"hail m", "hail-mary", "every word must match, short ones too"},
		{"canticle", "magnificat", "brackets don't stick to words"},
		{"canticle of mary", "magnificat", "stop words are skipped"},
		{"joseph's", "st-joseph", "apostrophes are dropped"},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			p, err := match(c.query, fixture)
			if err != nil {
				t.Fatalf("%s: %v", c.rule, err)
			}
			if p.ID != c.want {
				t.Fatalf("%s: got %s, want %s", c.rule, p.ID, c.want)
			}
		})
	}
}

func TestMatchAmbiguous(t *testing.T) {
	cases := map[string][]string{
		"hail": {"hail-holy-queen", "hail-mary"},
		"mary": {"hail-mary", "magnificat"},
	}
	for query, want := range cases {
		t.Run(query, func(t *testing.T) {
			_, err := match(query, fixture)
			var amb *AmbiguousError
			if !errors.As(err, &amb) {
				t.Fatalf("want *AmbiguousError, got %v", err)
			}
			if !slices.Equal(amb.IDs, want) {
				t.Fatalf("got %v, want %v", amb.IDs, want)
			}
		})
	}
}

func TestMatchNotFound(t *testing.T) {
	// Empty, unknown, and too vague: stop words and single letters on
	// their own would otherwise match nearly everything.
	for _, query := range []string{"", "  ", "zzz", "o", "of", "the", "a"} {
		t.Run(query, func(t *testing.T) {
			if _, err := match(query, fixture); !errors.Is(err, ErrNotFound) {
				t.Fatalf("want ErrNotFound, got %v", err)
			}
		})
	}
}
