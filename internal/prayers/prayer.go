package prayers

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Each prayer is a folder in data/:
//
//	data/<id>/meta.yaml   aliases, tags, default intention (shared by all languages)
//	data/<id>/en.md       "# Title", a blank line, then the text
//	data/<id>/la.md       other languages alongside, named by language code
//
//go:embed data
var dataFS embed.FS

type Prayer struct {
	ID        string
	Lang      string // the language actually used, after fallback
	Title     string
	Body      string
	Intention string   `yaml:"intention"` // default when none is given
	Aliases   []string `yaml:"aliases"`
	Tags      []string `yaml:"tags"`
}

// IDs lists every prayer, sorted.
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

// Langs lists the languages a prayer is written in, e.g. [en la].
func Langs(id string) ([]string, error) {
	files, err := fs.Glob(dataFS, path.Join("data", id, "*.md"))
	if err != nil {
		return nil, err
	}
	langs := make([]string, len(files))
	for i, f := range files {
		langs[i] = strings.TrimSuffix(path.Base(f), ".md")
	}
	return langs, nil
}

// Get loads a prayer by id in the given language, falling back to English.
func Get(id, lang string) (Prayer, error) {
	dir := path.Join("data", id)

	raw, err := fs.ReadFile(dataFS, path.Join(dir, "meta.yaml"))
	if err != nil {
		return Prayer{}, fmt.Errorf("no prayer called %q", id)
	}
	p := Prayer{ID: id}
	if err := yaml.Unmarshal(raw, &p); err != nil {
		return Prayer{}, fmt.Errorf("%s/meta.yaml: %w", id, err)
	}

	text, err := fs.ReadFile(dataFS, path.Join(dir, lang+".md"))
	if err != nil && lang != "en" {
		lang = "en"
		text, err = fs.ReadFile(dataFS, path.Join(dir, "en.md"))
	}
	if err != nil {
		return Prayer{}, fmt.Errorf("%s has no text: %w", id, err)
	}

	p.Lang = lang
	p.Title, p.Body = splitTitle(string(text))
	return p, nil
}

// Text returns the body with {{intention}} filled in, falling back to the
// prayer's default intention.
func (p Prayer) Text(intention string) string {
	if intention == "" {
		intention = p.Intention
	}
	return strings.ReplaceAll(p.Body, "{{intention}}", intention)
}

// HasIntentionSlot reports whether the text has a place for an intention.
// Prayers without one show the intention as a line before the prayer instead.
func (p Prayer) HasIntentionSlot() bool {
	return strings.Contains(p.Body, "{{intention}}")
}

// Plain is the prayer as plain text: the title, the intention line, then the
// prayer, with no trailing newline. It never contains colour or decoration,
// so it is what gets copied to the clipboard, piped or logged. Rendering for
// the terminal is separate and builds on the same parts.
func (p Prayer) Plain(intention string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", p.Title)
	// Name the intention first, as you would aloud, unless the prayer
	// has its own place for it in the text.
	if intention != "" && !p.HasIntentionSlot() {
		fmt.Fprintf(&b, "For %s\n\n", intention)
	}
	b.WriteString(p.Text(intention))
	return b.String()
}

// splitTitle takes "# Title" off the first line; the rest is the body.
func splitTitle(s string) (title, body string) {
	s = strings.TrimSpace(s)
	first, rest, _ := strings.Cut(s, "\n")
	if t, ok := strings.CutPrefix(first, "# "); ok {
		return strings.TrimSpace(t), strings.TrimSpace(rest)
	}
	return "", s
}
