package prayers

import "testing"

func TestGetHailMary(t *testing.T) {
	p, err := Get("hail-mary", "en")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Hail Mary" || p.Body == "" {
		t.Fatalf("got title %q, body %d chars", p.Title, len(p.Body))
	}
}

func TestFallsBackToEnglish(t *testing.T) {
	p, err := Get("hail-mary", "xx")
	if err != nil {
		t.Fatal(err)
	}
	if p.Lang != "en" {
		t.Fatalf("want fallback to en, got %q", p.Lang)
	}
}

// Every prayer with an {{intention}} slot needs a default, or running it
// without --for would print "specifically for ."
func TestSlottedPrayersHaveDefaults(t *testing.T) {
	ids, err := IDs()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		p, err := Get(id, "en")
		if err != nil {
			t.Fatal(err)
		}
		if p.HasIntentionSlot() && p.Intention == "" {
			t.Errorf("%s has {{intention}} but no default intention in meta.yaml", id)
		}
	}
}
