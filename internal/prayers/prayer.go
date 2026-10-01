package prayers

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"
)

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

// splitTitle takes "# Title" off the first line; the rest is the body.
func splitTitle(s string) (title, body string) {
	s = strings.TrimSpace(s)
	first, rest, _ := strings.Cut(s, "\n")
	if t, ok := strings.CutPrefix(first, "# "); ok {
		return strings.TrimSpace(t), strings.TrimSpace(rest)
	}
	return "", s
}
