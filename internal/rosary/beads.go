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

// There are two bead sizes and the crucifix. Nothing else: a rosary's beads differ
// in size, and the cross differs in shape.
const (
	// Large is the bigger bead: the five that open the decades.
	Large Kind = iota
	// Small is the lesser bead: the Hail Marys, and the pendant's chain.
	Small
	// Cross is the crucifix. Not a bead size: its shape carries meaning, so it is
	// always a cross.
	Cross
)

// ── THE TWO BEAD GLYPHS — change these to restyle the rosary ────────────────
//
// There are exactly two bead sizes, one slightly bigger than the other. Both
// MUST be one cell wide or the ring shears (ROSARY-TUI.md §2); every pair below
// has been measured at 1 cell.
//
// Candidates, big / small:
//
//	"◯" "○"   large / medium hollow   — the default: big, and the two sizes read
//	"⬤" "●"   heavy / medium filled   — biggest and solid, but the two look alike
//	"●" "○"   filled / hollow         — strong contrast, though of fill not size
//	"●" "•"   filled / bullet
//	"○" "◦"   medium / small hollow   — most delicate
var (
	bigBead   = "⬤"
	smallBead = "●"
)

// Glyph is the character drawn for a bead of this kind.
//
// Every glyph must be one cell wide or the ring shears. See ROSARY-TUI.md §2.
func (k Kind) Glyph() string {
	switch k {
	case Cross:
		return "✠"
	case Small:
		return smallBead
	default:
		return bigBead
	}
}

// Words is one prayer said on a bead: a title and the lines to show.
//
// Lines are kept exactly as given and never reflowed, which is RENDERING.md's
// rule. An empty string is a blank line between stanzas.
type Words struct {
	Title string
	Lines []string

	// Announces is which mystery this PRAYER declares: 1..5, or 0 for one that
	// declares none.
	//
	// It is on the prayer rather than on the bead because a bead can hold several
	// prayers and the mystery changes partway through: the big bead at a junction
	// says the previous decade's Glory Be and Fatima Prayer first, and only then
	// the Our Father that opens the new decade. Announcing on the bead changed the
	// mystery too early — while the previous decade was still being finished.
	Announces int

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

	// Name identifies the bead — "Hail Mary", "Our Father". It is not what the
	// screen shows: the heading names the PRAYER being said (see prayerPanel),
	// since a bead can hold several. It is the fallback title for a bead with no
	// prayers, and what tests report when one is wrong.
	Name string

	Says []Words // one or more prayers, said in order

	// SameAs, when non-zero, means this bead is drawn at the same place as bead
	// SameAs-1 rather than taking its own place on the ring. It is 1-based so that
	// the zero value means "has its own place".
	//
	// This is how the rosary closes where it began: the final prayers are a second
	// visit to the first big bead, drawn as one bead but prayed separately.
	SameAs int
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

// Announcing returns a copy of w that declares mystery n (1..5).
//
// Wrap the PRAYER at which the mystery changes:
//
//	s.add(Large, "Our Father", gloryBe, fatima, Announcing(1, Say("our-father")))
//
// From that prayer onwards the mystery is shown, until another announces the next
// one. Nothing is cleared in between, which is what keeps the mystery on screen
// through the ten Hail Marys that follow.
//
// Announcing on the prayer rather than the bead matters at a junction: the big
// bead there finishes the previous decade (Glory Be, Fatima) before opening the
// new one, so the mystery must change partway through the bead, not on arrival.
func Announcing(n int, w Words) Words {
	w.Announces = n
	return w
}

// run appends n identical beads.
func (s *builder) run(k Kind, n int, name string, says ...Words) {
	for i := 0; i < n; i++ {
		s.beads = append(s.beads, Bead{Kind: k, Name: name, Says: says})
	}
}

// addAt appends a bead that is drawn at the same place as an existing one.
//
// It exists for the close of the rosary: praying returns to the bead it began on,
// so that bead is visited twice. A flat sequence cannot revisit an entry, so the
// second visit is its own bead sharing the first one's cell — one bead on screen,
// two stops in the sequence. The user prays each visit's prayers in turn instead
// of all of them at once.
//
// SameAs is honoured by the geometry (see ring.go), which gives this bead the
// position of bead `at` rather than a place of its own on the ring.
func (s *builder) addAt(at int, k Kind, name string, says ...Words) {
	if at < 0 || at >= len(s.beads) {
		panic("rosary: addAt refers to a bead that does not exist")
	}
	s.beads = append(s.beads, Bead{Kind: k, Name: name, Says: says, SameAs: at + 1})
}

// Announces is the mystery this bead declares, or 0 for none.
//
// A bead declares whatever its prayers declare. Which prayer does the announcing
// decides WHEN the mystery changes (see Announcing); this reports only whether
// the bead is where it happens.
//
// Nothing in the rendering path uses it — the display asks announced(), which
// needs the prayer, not the bead. It is here because "which bead announces this
// mystery?" is the useful question when checking the sequence's structure.
func (b Bead) Announces() int {
	for _, w := range b.Says {
		if w.Announces > 0 {
			return w.Announces
		}
	}
	return 0
}
