package prayers

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

var ErrNotFound = errors.New("no matching prayer")

// AmbiguousError means the query matched several prayers equally well.
type AmbiguousError struct {
	Query string
	IDs   []string
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("%q could mean: %s", e.Query, strings.Join(e.IDs, ", "))
}

// Find loads every prayer and returns the one the query means.
func Find(query, lang string) (Prayer, error) {
	ids, err := IDs()
	if err != nil {
		return Prayer{}, err
	}
	all := make([]Prayer, 0, len(ids))
	for _, id := range ids {
		p, err := Get(id, lang)
		if err != nil {
			return Prayer{}, err
		}
		all = append(all, p)
	}
	return match(query, all)
}

// match picks the prayer a query means. Each prayer is placed in the
// strongest tier it qualifies for:
//
//  1. the query is its id               "hail mary"  → hail-mary
//  2. the query is its title or alias   "ave"        → hail-mary
//  3. every query word starts a word
//     in one of its names               "acut"       → st-carlo-acutis
//
// The first tier with any prayers in it decides: one prayer is the match,
// several is an *AmbiguousError. All tiers empty is ErrNotFound.
func match(query string, prayers []Prayer) (Prayer, error) {
	q := normalize(query)
	if q == "" {
		return Prayer{}, ErrNotFound
	}

	tiers := make([][]Prayer, 3)
	for _, p := range prayers {
		names := []string{normalize(p.ID), normalize(p.Title)}
		for _, a := range p.Aliases {
			names = append(names, normalize(a))
		}

		switch {
		case names[0] == q:
			tiers[0] = append(tiers[0], p)
		case slices.Contains(names, q):
			tiers[1] = append(tiers[1], p)
		case slices.ContainsFunc(names, func(n string) bool { return wordsMatch(q, n) }):
			tiers[2] = append(tiers[2], p)
		}
	}

	for _, hits := range tiers {
		switch len(hits) {
		case 0:
			continue
		case 1:
			return hits[0], nil
		default:
			e := &AmbiguousError{Query: query}
			for _, h := range hits {
				e.IDs = append(e.IDs, h.ID)
			}
			return Prayer{}, e
		}
	}
	return Prayer{}, fmt.Errorf("%w for %q", ErrNotFound, query)
}

// normalize makes names comparable: lowercase, apostrophes dropped
// ("joseph's" → "josephs"), any other character that isn't a letter or digit
// is a word break, and "saint" becomes "st". So "St. Joseph's",
// "saint-josephs" and "st josephs" all compare equal.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("'", "", "’", "").Replace(s)
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for i, w := range words {
		if w == "saint" {
			words[i] = "st"
		}
	}
	return strings.Join(words, " ")
}

// stopWords are skipped in word matching: they appear in so many titles
// ("Prayer to…", "Litany of the…") that they can't identify a prayer.
var stopWords = map[string]bool{
	"a": true, "an": true, "and": true, "at": true, "by": true, "for": true,
	"in": true, "of": true, "on": true, "the": true, "to": true, "with": true,
}

// wordsMatch reports whether every query word (stop words aside) starts some
// word in name. At least one query word must be 2+ letters, so "o" alone
// matches nothing rather than everything, while "hail m" still narrows.
func wordsMatch(q, name string) bool {
	nameWords := strings.Fields(name)
	meaningful := false
	for _, w := range strings.Fields(q) {
		if stopWords[w] {
			continue
		}
		if !slices.ContainsFunc(nameWords, func(n string) bool {
			return strings.HasPrefix(n, w)
		}) {
			return false
		}
		if len([]rune(w)) >= 2 {
			meaningful = true
		}
	}
	return meaningful
}
