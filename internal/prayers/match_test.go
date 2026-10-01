package prayers

import (
	"errors"
	"strings"
	"testing"
)

func TestFind(t *testing.T) {
	cases := map[string]string{
		"hail mary":   "hail-mary",
		"hail-mary":   "hail-mary",
		"ave":         "hail-mary",
		"carlo":       "st-carlo-acutis",
		"Saint Carlo": "st-carlo-acutis",
		"st. carlo":   "st-carlo-acutis",
		"acut":        "st-carlo-acutis",
	}
	for q, want := range cases {
		p, err := Find(q, "en")
		if err != nil {
			t.Errorf("Find(%q): %v", q, err)
			continue
		}
		if p.ID != want {
			t.Errorf("Find(%q) = %s, want %s", q, p.ID, want)
		}
	}
}

func TestFindNotFound(t *testing.T) {
	if _, err := Find("nonexistent", "en"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestIntentionFilled(t *testing.T) {
	p, err := Get("st-carlo-acutis", "en")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.Text(""), "{{") {
		t.Fatal("default intention not filled in")
	}
	if !strings.Contains(p.Text("the deploy"), "the deploy") {
		t.Fatal("custom intention not filled in")
	}
}
