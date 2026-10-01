package prayers

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
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

// IDs lists every prayer (one folder per prayer under data/).
func IDs() ([]string, error) {
	entries, err := fs.ReadDir(dataFS, "data")
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

// Find resolves what the user typed to a prayer, trying in order:
//  1. exact id          "hail-mary", "hail mary"
//  2. exact alias/title "ave", "ave maria"
//  3. word prefixes     "carlo", "saint carlo", "acut"
//
// The first tier with any hits wins; several hits in a tier is ambiguous.
func Find(query, lang string) (Prayer, error) {
	q := normalize(query)
	if q == "" {
		return Prayer{}, ErrNotFound
	}
	ids, err := IDs()
	if err != nil {
		return Prayer{}, err
	}

	tiers := make([][]Prayer, 3)
	for _, id := range ids {
		p, err := Get(id, lang)
		if err != nil {
			return Prayer{}, err
		}
		names := []string{normalize(id), normalize(p.Title)}
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

// normalize lowercases, turns punctuation into spaces and "saint" into "st",
// so "St. Carlo", "saint-carlo" and "st carlo" all compare equal.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("-", " ", "_", " ", ".", " ", ",", " ").Replace(s)
	words := strings.Fields(s)
	for i, w := range words {
		if w == "saint" {
			words[i] = "st"
		}
	}
	return strings.Join(words, " ")
}

// wordsMatch reports whether every word in q starts some word in name.
func wordsMatch(q, name string) bool {
	nameWords := strings.Fields(name)
	for _, w := range strings.Fields(q) {
		if !slices.ContainsFunc(nameWords, func(n string) bool {
			return strings.HasPrefix(n, w)
		}) {
			return false
		}
	}
	return true
}
