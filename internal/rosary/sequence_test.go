package rosary

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// The ring must be the same for the whole rosary. This is the point of
// fixedRing: a ring that resized per bead would breathe as you moved through the
// prayers, which reads as a glitch rather than as a design.
func TestRingIsFixedForTheWholeRosary(t *testing.T) {
	m := praying(t)

	for i := range m.beads {
		for j := range m.beads[i].Says {
			m.cursor, m.say = i, j
			lines := frameLines(m.bead(), m.words(), hint())
			out := drawRosary(m.ring, m.beads, m.cursor, 0, 0, lines)

			// Every frame must be exactly the canvas size, whatever is in it.
			if w := widthOf(out); w != m.ring.w {
				t.Fatalf("bead %d prayer %d: frame is %d wide, ring is %d", i, j, w, m.ring.w)
			}
			if h := linesIn(out); h != m.ring.h {
				t.Fatalf("bead %d prayer %d: frame is %d tall, ring is %d", i, j, h, m.ring.h)
			}
		}
	}
}

// Space advances through a bead's prayers first, then to the next bead.
func TestSpaceWalksPrayersThenBeads(t *testing.T) {
	m := model{beads: []Bead{
		{Name: "A", Says: []Words{Text("A1", "a1"), Text("A2", "a2")}},
		{Name: "B", Says: []Words{Text("B1", "b1")}},
	}}

	want := []struct{ cursor, say int }{{0, 0}, {0, 1}, {1, 0}}
	for i, w := range want {
		if m.cursor != w.cursor || m.say != w.say {
			t.Fatalf("step %d: at bead %d prayer %d, want bead %d prayer %d", i, m.cursor, m.say, w.cursor, w.say)
		}
		m.next()
	}

	// At the very end, next() reports that there is nowhere further to go.
	if m.next() {
		t.Error("next() advanced past the end of the rosary")
	}
}

// Going back lands on the LAST prayer of the previous bead, not its first:
// stepping back should undo exactly one space press.
func TestPrevLandsOnTheLastPrayerOfTheBeadBefore(t *testing.T) {
	m := model{beads: []Bead{
		{Name: "A", Says: []Words{Text("A1", "a1"), Text("A2", "a2")}},
		{Name: "B", Says: []Words{Text("B1", "b1")}},
	}, cursor: 1}

	if !m.prev() {
		t.Fatal("prev() refused to move")
	}
	if m.cursor != 0 || m.say != 1 {
		t.Errorf("landed on bead %d prayer %d, want bead 0 prayer 1", m.cursor, m.say)
	}

	// And it stops at the start rather than going negative.
	m.prev()
	if m.prev() {
		t.Error("prev() moved before the start of the rosary")
	}
	if m.cursor != 0 || m.say != 0 {
		t.Errorf("ended at bead %d prayer %d, want 0/0", m.cursor, m.say)
	}
}

// Say() must resolve against internal/prayers/data. If a prayer id in
// sequence.go is wrong, that is a startup error, not a blank bead.
func TestSayResolvesRealPrayers(t *testing.T) {
	m := praying(t)

	for i, b := range m.beads {
		for j, w := range b.Says {
			if w.Title == "" {
				t.Errorf("bead %d prayer %d has no title", i, j)
			}
			if len(w.Lines) == 0 {
				t.Errorf("bead %d (%s) prayer %d has no text", i, b.Name, j)
			}
		}
	}
}

func TestBadPrayerIDIsAStartupError(t *testing.T) {
	w := Say("no-such-prayer")
	if _, err := w.resolve("en"); err == nil {
		t.Error("an unknown prayer id resolved without error")
	}
}

// helpers

func widthOf(s string) int {
	w := 0
	for _, l := range strings.Split(s, "\n") {
		if x := lipgloss.Width(l); x > w {
			w = x
		}
	}
	return w
}

func linesIn(s string) int {
	n := 1
	for _, r := range s {
		if r == '\n' {
			n++
		}
	}
	return n
}

// No two beads may occupy the SAME cell: one would be drawn over the other and a
// bead would silently vanish from the rosary.
//
// Adjacent is allowed — on a real rosary the beads touch, and requiring a gap
// forced the ring far larger than its contents needed.
func TestNoTwoBeadsShareACell(t *testing.T) {
	m := praying(t)

	seen := make(map[[2]int]int, len(m.ring.pos))
	for i, p := range m.ring.pos {
		if prev, clash := seen[p]; clash {
			t.Errorf("beads %d and %d are both at %v", prev, i, p)
		}
		seen[p] = i
	}

	if len(seen) != len(m.beads) {
		t.Errorf("%d beads occupy only %d cells", len(m.beads), len(seen))
	}
}

// The crucifix belongs on the pendant, below the ring, and must appear once.
func TestCrucifixIsOnThePendantOnly(t *testing.T) {
	m := praying(t)
	m.cursor, m.say = 20, 0
	out := drawRosary(m.ring, m.beads, m.cursor, 0, 0, frameLines(m.bead(), m.words(), hint()))

	if n := strings.Count(out, Cross.Glyph()); n != 1 {
		t.Errorf("found %d crucifixes, want exactly 1", n)
	}

	// And it must sit below the ring's centre, not on the ring.
	for i, b := range m.beads {
		if b.Kind == Cross && m.ring.pos[i][1] <= m.ring.cy {
			t.Errorf("the crucifix is at row %d, which is not below the ring centre %d",
				m.ring.pos[i][1], m.ring.cy)
		}
	}
}

// The pendant must be a continuous chain attached to the ring. A gap between the
// ring and the first pendant bead makes the crucifix look like it is floating
// free of the rosary.
func TestPendantIsAttachedAndContinuous(t *testing.T) {
	m := praying(t)
	g := m.ring

	// Every pendant bead is in the same column.
	for i := 0; i < pendantLen; i++ {
		if g.pos[i][0] != g.cx {
			t.Errorf("pendant bead %d is at column %d, not on the centre line %d", i, g.pos[i][0], g.cx)
		}
	}

	// And each is exactly one row from the next, with no gaps.
	for i := 1; i < pendantLen; i++ {
		if d := g.pos[i-1][1] - g.pos[i][1]; d != 1 {
			t.Errorf("pendant beads %d and %d are %d rows apart, want 1", i-1, i, d)
		}
	}

	// The bead nearest the ring must touch the ring's bottom row, or the pendant
	// hangs detached.
	top := g.pos[pendantLen-1][1]
	if want := g.cy + g.ry; top != want {
		t.Errorf("the pendant starts at row %d but the ring's bottom is %d: there is a gap", top, want)
	}

	// The crucifix is the far end.
	if m.beads[0].Kind != Cross {
		t.Errorf("the first bead is %v, want Cross", m.beads[0].Kind)
	}
}
