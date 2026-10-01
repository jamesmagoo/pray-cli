package prayers

import (
	"strings"
	"testing"
)

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

func TestPlain(t *testing.T) {
	hail, err := Get("hail-mary", "en")
	if err != nil {
		t.Fatal(err)
	}
	got := hail.Plain("my mum")
	if !strings.HasPrefix(got, "Hail Mary\n\nFor my mum\n\nHail Mary, full of grace,") {
		t.Fatalf("want title, intention line, then prayer; got:\n%s", got)
	}
	if strings.HasSuffix(got, "\n") {
		t.Fatal("plain text should not end with a newline")
	}

	carlo, err := Get("st-carlo-acutis", "en")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(carlo.Plain("the deploy"), "\nFor the deploy\n") {
		t.Fatal("a prayer with its own intention slot shouldn't get a separate line")
	}
}
