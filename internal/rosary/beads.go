package rosary

import (
	"strings"

	"github.com/jamesmagoo/pray-cli/internal/prayers"
)

// The machinery behind sequence.go. You shouldn't need to touch this file to
// change the rosary — only to change what a bead *is*.

// Kind is what sort of bead this is. It decides the glyph, since on a real
// rosary the beads differ in size, not just in which prayer is said.
type Kind int

const (
	// Large is an Our Father bead: bigger, and set apart from its neighbours.
	Large Kind = iota
	// Small is a Hail Mary bead: the ones that make up a decade.
	Small
	// Link is chain rather than a bead — where a Glory Be is said.
	Link
	// Cross is the crucifix.
	Cross
)

// Glyph is the character drawn for a bead of this kind.
//
// Every glyph must be one cell wide or the ring shears. See ROSARY-TUI.md §2.
func (k Kind) Glyph() string {
	switch k {
	case Large:
		return "●"
	case Cross:
		return "✠"
	case Link:
		return "◦"
	default:
		return "○"
	}
}

// Words is one prayer said on a bead: a title and the lines to show.
//
// Lines are kept exactly as given and never reflowed, which is RENDERING.md's
// rule. An empty string is a blank line between stanzas.
type Words struct {
	Title string
	Lines []string

	// prayerID is set by Say instead of Lines: the text is then loaded from
	// internal/prayers/data at startup. Resolving it there rather than here keeps
	// Sequence() a plain declaration, with no I/O and no error handling in it.
	prayerID string
}

// Say takes a prayer from internal/prayers/data by its folder name.
func Say(prayerID string) Words {
	return Words{prayerID: prayerID}
}

// Text is a prayer written inline, for anything not in data/ yet.
func Text(title string, lines ...string) Words {
	return Words{Title: title, Lines: lines}
}

// resolve loads the text for a Say, and returns a Text unchanged.
func (w Words) resolve(lang string) (Words, error) {
	if w.prayerID == "" {
		return w, nil // written inline; nothing to load
	}
	p, err := prayers.Get(w.prayerID, lang)
	if err != nil {
		return w, err
	}
	w.Title = p.Title
	w.Lines = strings.Split(p.Text(""), "\n")
	return w, nil
}

// Bead is one stop on the rosary.
type Bead struct {
	Kind Kind
	Name string  // what to call it in the status line, e.g. "Hail Mary"
	Says []Words // one or more prayers, said in order

	// Nth and Of number a bead within its run, for "Hail Mary 3 of 10". Zero for
	// beads that don't come in runs.
	Nth, Of int

	// Announces is which mystery this bead declares: 1..5, or 0 for a bead that
	// declares none.
	//
	// The mystery is ANNOUNCED at a bead — traditionally the Our Father opening
	// the decade — and is then contemplated until the next announcement. It is
	// declared explicitly in sequence.go rather than worked out from the bead's
	// position, so that where a mystery is named is your decision and not a
	// consequence of how the loop happens to be written.
	Announces int
}

// Label names the bead on screen.
//
// Just the name: Nth and Of are kept on the Bead because they describe the
// structure, but they are deliberately not shown. A count on screen pulls the eye
// to the number instead of the prayer.
func (b Bead) Label() string {
	return b.Name
}

// builder accumulates the sequence, so sequence.go reads as a description of a
// rosary rather than as slice manipulation.
type builder struct {
	beads []Bead
}

// add appends one bead, saying the given prayers in order.
func (s *builder) add(k Kind, name string, says ...Words) {
	s.beads = append(s.beads, Bead{Kind: k, Name: name, Says: says})
}

// announce marks the bead most recently added as declaring mystery n (1..5).
//
// Call it straight after the add() for the bead that names the mystery:
//
//	s.add(Large, "Our Father", Say("our-father"))
//	s.announce(decade)
//
// From that bead onwards the mystery is shown, until another bead announces the
// next one. Nothing is cleared in between, which is what makes the mystery stay
// on screen through the ten Hail Marys that follow.
func (s *builder) announce(n int) {
	if len(s.beads) == 0 {
		panic("rosary: announce() called before any bead was added")
	}
	s.beads[len(s.beads)-1].Announces = n
}

// run appends n identical beads, numbered 1..n so each can say "3 of 10".
func (s *builder) run(k Kind, n int, name string, says ...Words) {
	for i := 1; i <= n; i++ {
		s.beads = append(s.beads, Bead{Kind: k, Name: name, Says: says, Nth: i, Of: n})
	}
}
