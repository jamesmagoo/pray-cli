package prayers

import (
	"slices"
	"testing"
)

// These tests check the real prayers in data/. They need no updating when
// you add a prayer: every prayer is checked automatically.

func allIDs(t *testing.T) []string {
	t.Helper()
	ids, err := IDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Fatal("no prayers found in data/")
	}
	return ids
}

// Every prayer must be found by its id, by each alias, and by its title in
// each of its languages. This fails when two prayers share an alias or
// title, or when a new prayer makes an existing name ambiguous.
func TestEveryPrayerIsFindable(t *testing.T) {
	for _, id := range allIDs(t) {
		langs, err := Langs(id)
		if err != nil {
			t.Fatal(err)
		}
		for _, lang := range langs {
			p, err := Get(id, lang)
			if err != nil {
				t.Fatal(err)
			}
			names := append([]string{id, p.Title}, p.Aliases...)
			for _, name := range names {
				t.Run(id+"/"+lang+"/"+name, func(t *testing.T) {
					got, err := Find(name, lang)
					if err != nil {
						t.Fatal(err)
					}
					if got.ID != id {
						t.Fatalf("finds %s instead of %s", got.ID, id)
					}
				})
			}
		}
	}
}

// Every prayer needs English (the fallback) and a "# Title" line and some
// text in each language.
func TestEveryPrayerIsComplete(t *testing.T) {
	for _, id := range allIDs(t) {
		langs, err := Langs(id)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(langs, "en") {
			t.Errorf("%s: missing en.md, the fallback language", id)
		}
		for _, lang := range langs {
			p, err := Get(id, lang)
			if err != nil {
				t.Errorf("%s/%s: %v", id, lang, err)
				continue
			}
			if p.Title == "" {
				t.Errorf("%s/%s.md: first line should be \"# Title\"", id, lang)
			}
			if p.Body == "" {
				t.Errorf("%s/%s.md: no text after the title", id, lang)
			}
		}
	}
}

// Every prayer with an {{intention}} slot needs a default, or running it
// without --for would print "specifically for ."
func TestSlottedPrayersHaveDefaults(t *testing.T) {
	for _, id := range allIDs(t) {
		p, err := Get(id, "en")
		if err != nil {
			t.Fatal(err)
		}
		if p.HasIntentionSlot() && p.Intention == "" {
			t.Errorf("%s has {{intention}} but no default intention in meta.yaml", id)
		}
	}
}
