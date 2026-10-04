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
			lines := mysteryLines(m.set, m.announced())
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

// No two beads may occupy the same cell by ACCIDENT: one would be drawn over the
// other and a bead would silently vanish from the rosary.
//
// A bead that deliberately shares another's place (SameAs — the rosary's close
// returning to its first bead) is exempt: that is one bead on screen visited
// twice, not two beads colliding.
//
// Adjacent is allowed — on a real rosary the beads touch, and requiring a gap
// forced the ring far larger than its contents needed.
func TestNoTwoBeadsShareACell(t *testing.T) {
	m := praying(t)

	seen := make(map[[2]int]int, len(m.ring.pos))
	shared := 0
	for i, p := range m.ring.pos {
		if m.beads[i].SameAs > 0 {
			shared++
			// It must land exactly on the bead it names, not merely near it.
			if want := m.ring.pos[m.beads[i].SameAs-1]; p != want {
				t.Errorf("bead %d shares bead %d's place but is at %v, not %v",
					i, m.beads[i].SameAs-1, p, want)
			}
			continue
		}
		if prev, clash := seen[p]; clash {
			t.Errorf("beads %d and %d are both at %v", prev, i, p)
		}
		seen[p] = i
	}

	if len(seen) != len(m.beads)-shared {
		t.Errorf("%d beads (%d shared) occupy %d cells", len(m.beads), shared, len(seen))
	}
}

// The crucifix belongs on the pendant, below the ring, and must appear once.
func TestCrucifixIsOnThePendantOnly(t *testing.T) {
	m := praying(t)
	m.cursor, m.say = 20, 0
	out := drawRosary(m.ring, m.beads, m.cursor, 0, 0, mysteryLines(m.set, m.announced()))

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
func TestPendantHangsFromTheRing(t *testing.T) {
	m := praying(t)
	g := m.ring

	// Every pendant bead is on the ring's centre line.
	for i := 0; i < pendantLen; i++ {
		if g.pos[i][0] != g.cx {
			t.Errorf("pendant bead %d is at column %d, not on the centre line %d",
				i, g.pos[i][0], g.cx)
		}
	}

	// Beads run bottom-up, one row apart except where a gap is declared.
	//
	// The expected spacing is written out rather than derived from
	// PendantGapAfter: deriving it from the thing under test makes the assertion
	// vacuous — it passed even with every gap removed.
	//
	// Bottom to top: crucifix(0), bead(1), GAP, three beads(2,3,4), GAP, ring.
	wantApart := []int{1, 2, 1, 1} // between beads 0-1, 1-2, 2-3, 3-4
	for i := 1; i < pendantLen && i-1 < len(wantApart); i++ {
		if got := g.pos[i-1][1] - g.pos[i][1]; got != wantApart[i-1] {
			t.Errorf("pendant beads %d and %d are %d rows apart, want %d",
				i-1, i, got, wantApart[i-1])
		}
	}

	// The top bead hangs BELOW the ring's bottom bead with a blank row between —
	// the gap declared after the last pendant bead.
	//
	// This previously asserted the adjacent row while its own comment said "one row
	// below", and so locked in a bug: the gap was counted into the canvas height
	// but never applied, because the placement loop decrements after the last bead
	// and then ends. The bottom bead sat directly on the pendant.
	top := g.pos[pendantLen-1][1]
	want := g.cy + g.ry + 1
	if PendantGapAfter(pendantLen - 1) {
		want++
	}
	if top != want {
		t.Errorf("the pendant's top bead is at row %d, want %d", top, want)
	}

	// And the row between really is empty, so the gap is visible rather than merely
	// arithmetic.
	if PendantGapAfter(pendantLen - 1) {
		between := g.cy + g.ry + 1
		for i := range m.beads {
			if g.pos[i][1] == between {
				t.Errorf("bead %d sits at row %d, which should be the gap between the "+
					"ring and the pendant", i, between)
			}
		}
	}

	// The crucifix is the far end.
	if m.beads[0].Kind != Cross {
		t.Errorf("the first bead is %v, want Cross", m.beads[0].Kind)
	}
}

// The pendant's shape, as described: crucifix, one small bead, gap, three small
// beads, gap, then the ring's first big bead where the loop begins.
func TestPendantShape(t *testing.T) {
	m := praying(t)

	if got := Pendant(); got != 5 {
		t.Fatalf("Pendant() = %d, want 5 (crucifix, one bead, three beads)", got)
	}

	want := []Kind{Cross, Small, Small, Small, Small}
	for i, k := range want {
		if got := m.beads[i].Kind; got != k {
			t.Errorf("pendant bead %d is %v, want %v", i, got, k)
		}
	}

	// And WHICH prayers sit on which bead. The bead kinds alone would not catch a
	// prayer moving between them, which is how the Creed ended up on the wrong
	// bead: the Sign of the Cross and the Creed are both said on the crucifix,
	// and the first bead of the chain is the Our Father alone.
	wantSays := [][]string{
		{"Sign of the Cross", "Apostles' Creed"},
		{"Our Father"},
		{"Hail Mary"},
		{"Hail Mary"},
		{"Hail Mary"},
	}
	for i, titles := range wantSays {
		got := m.beads[i].Says
		if len(got) != len(titles) {
			t.Errorf("pendant bead %d says %d prayers, want %d", i, len(got), len(titles))
			continue
		}
		for j, title := range titles {
			if got[j].Title != title {
				t.Errorf("pendant bead %d prayer %d is %q, want %q", i, j, got[j].Title, title)
			}
		}
	}

	// The loop begins on a big bead, on the ring.
	if got := m.beads[pendantLen].Kind; got != Large {
		t.Errorf("the ring's first bead is %v, want Large", got)
	}

	// The rosary ENDS where it began: the last bead is a second visit to the first
	// big bead, drawn in the same place.
	last := m.beads[len(m.beads)-1]
	if last.SameAs-1 != Pendant() {
		t.Errorf("the last bead shares place %d, want %d (the first big bead)",
			last.SameAs-1, Pendant())
	}
	if title := last.Says[len(last.Says)-1].Title; title != "The Collect" {
		t.Errorf("the rosary's last prayer is %q, want The Collect", title)
	}
}

// A decade junction is ONE big bead, not two side by side.
//
// On a real rosary the Glory Be, the Fatima Prayer and the next Our Father are
// all said on a single bead, held once. Giving the Glory Be its own bead put two
// big beads next to each other, and they showed in different shades because the
// Glory Be's bead was a different Kind from the Our Father's.
func TestDecadeJunctionIsOneBead(t *testing.T) {
	m := praying(t)

	// Walk the ring. Between two runs of Hail Marys there must be exactly one
	// big bead.
	run := 0
	for i, b := range m.beads {
		if b.Kind == Small {
			run++
			continue
		}
		if b.Kind == Cross {
			run = 0
			continue
		}

		// A big bead that ends a run of ten is a junction. Between two decades it
		// must be followed by Hail Marys again, and must carry the Glory Be, the
		// Fatima Prayer and the next Our Father.
		//
		// The LAST decade is different: no sixth decade follows, so its bead
		// carries only the closing Glory Be and Fatima, and the closing prayers
		// come after it. That is correct, not a double junction.
		if run == 10 {
			more := false
			for _, later := range m.beads[i+1:] {
				if later.Kind == Small {
					more = true
					break
				}
			}

			if more {
				if m.beads[i+1].Kind != Small {
					t.Errorf("bead %d (%s) closes a decade but bead %d (%s) is also big: two beads at one junction",
						i, m.beads[i].Name, i+1, m.beads[i+1].Name)
				}
				if len(b.Says) < 3 {
					t.Errorf("junction bead %d says only %d prayers; expected Glory Be, Fatima and Our Father",
						i, len(b.Says))
				}
			}
		}
		run = 0
	}
}

// Every decade must still be ten Hail Marys, and there must be five of them.
func TestFiveDecadesOfTen(t *testing.T) {
	m := praying(t)

	runs := []int{}
	run := 0
	for _, b := range m.beads {
		if b.Kind == Small {
			run++
			continue
		}
		if run > 0 {
			runs = append(runs, run)
		}
		run = 0
	}
	if run > 0 {
		runs = append(runs, run)
	}

	tens := 0
	for _, r := range runs {
		if r == 10 {
			tens++
		}
	}
	if tens != 5 {
		t.Errorf("found %d decades of ten Hail Marys, want 5 (runs: %v)", tens, runs)
	}
}

// The first bead of the ring sits directly above the pendant.
//
// It is where the loop begins and ends, so it belongs at the join rather than
// wherever even spacing happens to put it. The decades are then spaced around the
// circle from there.
func TestFirstRingBeadSitsAbovePendant(t *testing.T) {
	m := praying(t)
	g := m.ring

	first := g.pos[pendantLen]

	// Same column as the pendant, which hangs on the centre line.
	if first[0] != g.cx {
		t.Errorf("the first ring bead is at column %d, not above the pendant at %d",
			first[0], g.cx)
	}

	// At the bottom of the circle.
	if want := g.cy + g.ry; first[1] != want {
		t.Errorf("the first ring bead is at row %d, want %d (the bottom of the ring)",
			first[1], want)
	}

	// And directly above the pendant's topmost bead, with the gap between them.
	top := g.pos[pendantLen-1]
	if top[0] != first[0] {
		t.Errorf("the pendant hangs at column %d but the first ring bead is at %d",
			top[0], first[0])
	}
	if top[1] <= first[1] {
		t.Errorf("the pendant's top bead (row %d) is not below the first ring bead (row %d)",
			top[1], first[1])
	}
}

// A rosary has exactly FIVE big beads, one per decade.
//
// The closing prayers tempted an extra two: giving the fifth Glory Be and the
// Hail Holy Queen their own beads put seven big beads on the ring. They belong on
// the bead the rosary started on, which the fingers return to.
func TestExactlyFiveBigBeads(t *testing.T) {
	m := praying(t)

	// Count what is DRAWN, not the number of sequence entries: the close is a
	// second visit to the first big bead and shares its cell, so it is the same
	// bead on screen.
	var big []int
	for i, b := range m.beads {
		if b.Kind == Large && b.SameAs == 0 {
			big = append(big, i)
		}
	}

	if len(big) != 5 {
		t.Errorf("found %d big beads on the ring, want 5 (at %v)", len(big), big)
		for _, i := range big {
			t.Logf("  bead %d: %s", i, m.beads[i].Name)
		}
	}

	// And they occupy five distinct cells.
	cells := map[[2]int]bool{}
	for _, i := range big {
		cells[m.ring.pos[i]] = true
	}
	if len(cells) != 5 {
		t.Errorf("the five big beads occupy %d cells, want 5", len(cells))
	}

	// The first of them is the starting bead, at the pendant.
	if len(big) > 0 && big[0] != Pendant() {
		t.Errorf("the first big bead is %d, want %d (the start of the loop)", big[0], Pendant())
	}
}

// The starting bead opens the rosary with a Glory Be and an Our Father, and ONLY
// those: the closing prayers are a separate stop at the same place, so you do not
// pray the whole close before the first decade.
func TestStartingBeadOpensOnly(t *testing.T) {
	m := praying(t)
	start := m.beads[Pendant()]

	titles := make([]string, len(start.Says))
	for i, w := range start.Says {
		titles[i] = w.Title
	}

	want := []string{"Glory Be", "Our Father"}
	if len(titles) != len(want) {
		t.Fatalf("the starting bead says %v, want %v", titles, want)
	}
	for i := range want {
		if titles[i] != want[i] {
			t.Errorf("starting bead prayer %d is %q, want %q", i, titles[i], want[i])
		}
	}
}

// Opening and closing the rosary are two separate stops at the same bead.
//
// Putting all five prayers on one entry meant praying the whole close before the
// first decade. They are now two beads sharing a cell: one bead on screen, two
// stops in the sequence.
func TestOpenAndCloseAreSeparateStops(t *testing.T) {
	m := praying(t)

	start := m.beads[Pendant()]
	last := m.beads[len(m.beads)-1]

	// Same place on screen.
	if m.ring.pos[Pendant()] != m.ring.pos[len(m.beads)-1] {
		t.Errorf("the closing bead is at %v, the opening bead at %v: they should share a cell",
			m.ring.pos[len(m.beads)-1], m.ring.pos[Pendant()])
	}

	// Separate stops: the opening bead must NOT carry the closing prayers.
	for _, w := range start.Says {
		if w.Title == "Hail, Holy Queen" || w.Title == "Fatima Prayer" {
			t.Errorf("the opening bead says %q; the close belongs to the second visit", w.Title)
		}
	}
	if len(start.Says) != 2 {
		t.Errorf("the opening bead says %d prayers, want 2 (Glory Be, Our Father)", len(start.Says))
	}
	if len(last.Says) != 4 {
		t.Errorf("the closing bead says %d prayers, want 4 (Glory Be, Fatima, Hail, Holy Queen, the Collect)",
			len(last.Says))
	}

	// Whichever of the two the cursor is on must show the halo: the one that is
	// not current has to yield the cell, or the highlight vanishes.
	for _, cursor := range []int{Pendant(), len(m.beads) - 1} {
		out := drawRosary(m.ring, m.beads, cursor, 0, 0, mysteryLines(m.set, 1))
		if got := strings.Count(stripEscapes(out), currentGlyph(0)); got != 1 {
			t.Errorf("cursor %d: want exactly 1 halo, got %d", cursor, got)
		}
	}
}

// Pendant() must agree with what Sequence() actually builds.
//
// It is a hand-written constant — `func Pendant() int { return 5 }` — that the
// geometry trusts completely: ring.go slices the bead list at that index and
// treats everything before it as the hanging chain. Nothing made the two agree,
// so adding or removing a pendant bead in Sequence() without editing Pendant()
// would silently mis-slice the rosary: a ring bead drawn down the pendant, or a
// pendant bead flung onto the ring.
//
// The structure is derivable, which is what makes this checkable: the pendant is
// the crucifix and the small beads that follow it, and the ring begins at the
// first Large bead — the one where the loop opens and closes.
func TestPendantCountMatchesTheSequence(t *testing.T) {
	beads := Sequence()

	firstLarge := -1
	for i, b := range beads {
		if b.Kind == Large {
			firstLarge = i
			break
		}
	}
	if firstLarge < 0 {
		t.Fatal("the sequence has no Large bead; the ring has nowhere to begin")
	}

	if Pendant() != firstLarge {
		t.Errorf("Pendant() is %d but the first big bead — where the ring begins — "+
			"is at index %d; the geometry will slice the bead list in the wrong place",
			Pendant(), firstLarge)
	}

	// And everything below that index really is pendant material: the crucifix
	// first, then small beads. A Large bead among them would mean the ring's
	// opening bead is not the first one.
	if beads[0].Kind != Cross {
		t.Errorf("bead 0 is %v, want Cross — the pendant hangs from the crucifix",
			beads[0].Kind)
	}
	for i := 1; i < Pendant(); i++ {
		if beads[i].Kind != Small {
			t.Errorf("pendant bead %d is %v, want Small", i, beads[i].Kind)
		}
	}

	// PendantGapAfter must not point past the pendant either: a gap declared after
	// a bead that does not exist is silently ignored, and reads as a typo that did
	// nothing.
	for i := Pendant(); i < Pendant()+3; i++ {
		if PendantGapAfter(i) {
			t.Errorf("PendantGapAfter(%d) is true, but the pendant is only %d beads; "+
				"that gap can never be drawn", i, Pendant())
		}
	}
}
