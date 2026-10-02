package rosary

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

// Drawing the rosary as a ring with the prayer inside it.
//
// The ring is not a fixed picture with a hole in the middle. It is computed from
// the text it has to enclose: given the prayer's longest line, grow the ring
// until no bead sits in a row the text occupies. That way a short prayer gets a
// snug ring and a long one a wide one, and a bead never lands on a word, which
// is the thing that makes a hand-placed ring fall apart as soon as the text
// changes.

// gutter is the blank cells kept between a bead and the text beside it.
const gutter = 3

// pendantLen is how many beads of the sequence hang below the ring rather than
// sitting on it: the crucifix and the short chain up to the first decade.
//
// They are the FIRST beads of Sequence(), because that is the order they are
// prayed. Drawing them on the pendant rather than on the ring is a display
// choice, so it lives here and not in sequence.go — but they are the same beads,
// with the same cursor, so the crucifix is navigable like anything else.
//
// Set it with Pendant() in sequence.go rather than editing this: the pendant's
// length is part of the shape you are describing, and a number here that has to
// be kept in sync with the sequence by hand is a bug waiting to happen.
var pendantLen = Pendant()

// canvas is a fixed grid of cells we place glyphs on, then join into lines.
//
// Drawing onto a grid rather than building strings is what lets the ring and the
// text be positioned independently and still compose: each knows its own
// coordinates and neither has to know the other's layout.
type canvas struct {
	cells [][]rune
	w, h  int
}

func newCanvas(w, h int) *canvas {
	cells := make([][]rune, h)
	for i := range cells {
		cells[i] = []rune(strings.Repeat(" ", w))
	}
	return &canvas{cells: cells, w: w, h: h}
}

// set writes one rune, ignoring anything outside the grid so callers don't each
// have to bounds-check.
func (c *canvas) set(x, y int, r rune) {
	if y >= 0 && y < c.h && x >= 0 && x < c.w {
		c.cells[y][x] = r
	}
}

// text writes a line left-to-right from (x, y).
func (c *canvas) text(x, y int, s string) {
	for i, r := range s {
		c.set(x+i, y, r)
	}
}

// The ring is a CLOSED circle. Angles are measured the usual way,
// counter-clockwise from east, with y flipped because rows grow downwards.
//
// ringStart is straight down, where the pendant hangs: the loop begins and ends
// at the bottom big bead, and the pendant descends vertically from it.
//
// ringArc is negative so the sequence runs counter-clockwise — the beads travel
// away from the crucifix, up the left side, round, and back down, the direction
// the fingers go.
//
// It used to be an open arc with a 51.6 degree gap at the bottom, on the theory
// that a rosary's loop has a mouth where the pendant joins it. That was the thing
// stopping the five big beads from forming a pentagon: 55 beads spread over a
// partial arc give 62.8 degrees between every eleventh one, and no amount of even
// spacing fixes it. A CLOSED loop of 55 divides by five exactly, so the pentagon
// falls out at 72.00 degrees with nothing arranging it. The pendant hangs from the
// bottom bead rather than through a hole beside it, which is also how a real
// rosary is strung.
const (
	ringStart = -math.Pi / 2 // straight down: the bottom of the loop
	ringArc   = -2 * math.Pi // a full turn, counter-clockwise
)

// pendantRows is how many rows the pendant occupies: one per bead plus its gaps.
func pendantRows() int {
	rows := pendantLen
	for i := 0; i < pendantLen; i++ {
		if PendantGapAfter(i) {
			rows++
		}
	}
	return rows
}

// centre returns the ring's centre for the given radii. Both the sizing check
// and the drawing go through this, so they cannot drift apart: two independent
// derivations of the centre is how a bead silently lands on a letter.
func centre(rx, ry int) (cx, cy int) { return rx + 2, ry + 1 }

// ringGeometry is where each bead of the ring sits, and how big the ring is.
type ringGeometry struct {
	rx, ry int      // radii in cells; rx is larger because cells are tall
	pos    [][2]int // bead index -> x, y on the canvas
	w, h   int      // canvas size the ring needs
	cx, cy int      // the ring's centre
}

// fixedRing sizes the ring ONCE, for the whole rosary.
//
// This is the important decision in here. A self-sizing ring — one re-measured
// per bead — looks broken in motion: the ring breathes in and out as you move
// between a long prayer and a short one, and the beads crawl. A rosary is a
// physical object, so the ring is measured to the LONGEST prayer in the whole
// sequence and then never changes. Short prayers simply sit in more space.
// The ring now encloses the MYSTERY, not the prayer, so it is sized against the
// widest mystery box of any set — a much smaller block than a prayer, which is why
// the ring is tighter than it used to be.
//
// It is still sized once and frozen: the box is a fixed size per set, so the ring
// does not move as the mysteries change either.
func fixedRing(beads []Bead) ringGeometry {
	textW, textH := 0, 0
	for _, set := range Sets() {
		for n := 1; n <= len(set.Mysteries); n++ {
			lines := mysteryLines(set, n)
			if h := len(lines); h > textH {
				textH = h
			}
			for _, l := range lines {
				if x := lipgloss.Width(l); x > textW {
					textW = x
				}
			}
		}
	}
	return layout(beads, textW, textH)
}

// layout sizes a ring that holds n beads around a text block of the given size.
//
// The loop is the heart of it: start with a ring roughly as wide as the text and
// widen it one cell at a time until clears reports no bead overlaps the text.
// Solving the ellipse algebraically would be exact but far harder to read, and
// this runs a few dozen iterations on a ring of a dozen beads.
func layout(beads []Bead, textW, textH int) ringGeometry {
	n := len(beads)

	// How many positions the ring actually has to hold.
	//
	// NOT len(beads) minus the pendant: a bead with SameAs is drawn on top of
	// another and takes no place of its own. Counting it made the sizing checks
	// test a ring one bead denser than the one being drawn, so the ring was grown
	// to clear a crowding that never happened — and the extra size pushed the five
	// big beads off their even spacing.
	onRingSlots := 0
	for i := pendantLen; i < n; i++ {
		if beads[i].SameAs == 0 {
			onRingSlots++
		}
	}
	// Grow the ring until no bead sits on a letter.
	//
	// Both radii have to be free to grow, not just rx. With many beads the ring is
	// crowded: beads land only a cell or two apart, so widening alone cannot help
	// if the ring is too SHORT — there will always be a bead on a text row. Each
	// pass widens, and every few passes also heightens, so the ring escapes in
	// whichever direction it is stuck.
	// Grow outwards keeping a 2:1 ratio, because a terminal cell is about twice as
	// tall as it is wide: rx = 2*ry is what reads as a CIRCLE rather than as a long
	// hoop. (See ROSARY-TUI.md §1.)
	//
	// With the mystery inside — a short label, not a prayer — the contents no longer
	// constrain the ring much; its size is set almost entirely by the beads needing
	// room not to touch. Growing on a fixed ratio keeps it round while that happens,
	// where growing rx faster than ry stretched it sideways.
	ry := textH/2 + 2
	for ; ry < 200; ry++ {
		rx := ry * 2
		if clears(rx, ry, n, textW, textH) && spacedOut(onRingSlots, rx, ry) {
			break
		}
	}
	rx := ry * 2

	g := ringGeometry{
		rx: rx, ry: ry,
		w: rx*2 + 5,
		// crossRows() is the crucifix hanging BELOW its own bead: it is a block, not
		// a single cell, and canvas.set silently ignores out-of-bounds writes, so a
		// canvas one row short loses the cross's foot with no error anywhere.
		h: ry*2 + 3 + pendantRows() + crossRows() + 1,
	}
	g.cx, g.cy = centre(rx, ry)
	// The first pendantLen beads hang below the ring, in a line from the gap; the
	// rest are spaced around the arc.
	//
	// The chain starts at the ring's own bottom row, not below it, so the pendant
	// is visibly ATTACHED. A one-row gap here makes the crucifix look like it is
	// floating free of the rosary, which is wrong: on a real rosary the pendant
	// hangs from the ring.
	// Drawn bottom-up: the crucifix is furthest from the ring. Gaps are blank rows
	// of chain between beads, so the pendant reads as
	//
	//	crucifix — bead — gap — three beads — gap — the ring
	//
	// rather than as one unbroken column of beads.
	g.pos = make([][2]int, n)
	row := g.cy + g.ry + pendantRows() - 1
	for i := 0; i < pendantLen && i < n; i++ {
		g.pos[i] = [2]int{g.cx, row}
		row--
		if PendantGapAfter(i) {
			row--
		}
	}
	onRing := n - pendantLen
	if onRing < 1 {
		return g
	}

	// Beads that share another bead's place take no slot of their own: a second
	// visit to a bead is drawn as that same bead. They are filled in last, once
	// every real position is known.
	var shared []int
	var own []int
	for i := pendantLen; i < n; i++ {
		if beads[i].SameAs > 0 {
			shared = append(shared, i)
			continue
		}
		own = append(own, i)
	}

	// Every ring bead, spaced evenly around the closed loop in ONE run.
	//
	// The first of them lands at ringStart — straight down, directly above the
	// pendant — because that is where the walk begins, so the bead the loop opens
	// and closes on sits at the join without being placed by hand.
	//
	// It used to be pinned there explicitly, with the rest spaced over the arc
	// above it and one extra slot requested and dropped to stop the run's two ends
	// colliding. All of that was scaffolding for the open arc. A closed loop needs
	// none of it: ask for exactly as many slots as there are beads and they come
	// back evenly spaced with the first at the bottom.
	for i, a := range arcAngles(len(own), rx, ry) {
		g.pos[own[i]] = [2]int{
			g.cx + int(math.Round(float64(rx)*math.Cos(a))),
			g.cy - int(math.Round(float64(ry)*math.Sin(a))),
		}
	}

	for _, i := range shared {
		g.pos[i] = g.pos[beads[i].SameAs-1]
	}
	return g
}

// cellAspect is how many columns tall one terminal row is: a cell is about twice
// as tall as it is wide.
//
// It is the conversion between the grid's coordinates and what the eye actually
// sees. One step down the screen covers twice the visual distance of one step
// across, so any measurement of "how far apart do these look?" has to scale the
// vertical by this before it means anything.
//
// The same 2:1 is why rx = 2*ry draws a circle rather than a hoop (see layout);
// this constant names it for the one other place that needs it.
const cellAspect = 2.0

// arcAngles returns the angle for each of n beads, spaced evenly by VISUAL
// distance around the ellipse.
//
// Visual, not geometric, and that distinction is the whole point of this function.
// Walking the ellipse by its own arc length clumps the beads at the top and bottom
// and stretches them down the sides, because the ellipse is flat where it crosses
// the vertical axis and steep where it crosses the horizontal: equal arc length
// buys you very different numbers of CELLS depending on where you spend it.
// Measured on the real rosary, that gave a worst-to-best gap ratio of 1.98 — beads
// twice as far apart in some places as others, which is exactly what it looked
// like. Scaling dy by cellAspect while walking brings the ratio to 1.01.
//
// A consequence, not a coincidence: with 55 ring beads in five decades of eleven,
// evenly spaced slots around a CLOSED loop put every eleventh bead — the big ones
// — at exactly 72 degrees from the last. The five big beads form a true pentagon
// with no code to arrange them; they are simply every 11th of 55 evenly spaced
// points on a circle. This only works because the loop is closed: see ringArc.
//
// The ellipse has no closed form for arc length, so walk it in fine steps, add up
// the distance travelled, then place beads at equal fractions of the total. Two
// thousand steps is far more than the ~55 beads need and still trivial to compute
// once at startup.
func arcAngles(n, rx, ry int) []float64 {
	if n <= 1 {
		return []float64{ringStart}
	}

	// 20000 steps, not 2000: the walk quantises every angle to a step boundary, and
	// at 2000 that was enough to shift a big bead 0.18 degrees off the pentagon.
	// Cheap insurance — this runs once, at startup.
	const steps = 20000
	// Walk the circle, recording the cumulative VISUAL distance at each step.
	dist := make([]float64, steps+1)
	px := float64(rx) * math.Cos(ringStart)
	py := float64(ry) * math.Sin(ringStart)
	for k := 1; k <= steps; k++ {
		a := ringStart + ringArc*float64(k)/float64(steps)
		x := float64(rx) * math.Cos(a)
		y := float64(ry) * math.Sin(a)
		// dy scaled to columns: this one multiplication is the fix. Without it the
		// walk measures grid units, which is not what the eye is looking at.
		dist[k] = dist[k-1] + math.Hypot(x-px, (y-py)*cellAspect)
		px, py = x, y
	}
	total := dist[steps]

	// For each bead, find where along that walk it falls.
	//
	// Divided by n, not n-1: the loop is CLOSED, so the last bead must stop one
	// interval short of the first rather than landing on top of it. n-1 is right
	// for an open arc with two distinct ends, and was right when this was one —
	// using it on a closed loop puts a bead at 360 degrees, i.e. back at the start.
	angles := make([]float64, n)
	k := 0
	for i := 0; i < n; i++ {
		want := total * float64(i) / float64(n)
		for k < steps && dist[k+1] < want {
			k++
		}
		angles[i] = ringStart + ringArc*float64(k)/float64(steps)
	}
	return angles
}

// spacedOut reports whether no two beads share or touch a cell.
//
// Without this the ring stops growing as soon as the text fits, and with many
// beads they end up shoulder to shoulder — "○○" reads as a smear rather than as
// two beads. Requiring a gap makes the ring grow until the perimeter is actually
// long enough for the beads on it, which is the honest constraint: you cannot fit
// 68 distinct beads on a ring of 60 cells.
// n is how many beads sit ON THE RING — the pendant excluded, and beads that
// share another's place excluded too, since they take no position of their own.
func spacedOut(n, rx, ry int) bool {
	if n < 2 {
		return true
	}
	prev := [2]int{-99, -99}
	for _, a := range arcAngles(n, rx, ry) {
		p := [2]int{
			rx + 2 + int(math.Round(float64(rx)*math.Cos(a))),
			ry + 1 - int(math.Round(float64(ry)*math.Sin(a))),
		}
		// Beads must not land on the SAME cell — that loses a bead entirely — but
		// adjacent is fine: on a real rosary the beads touch. Requiring a gap
		// instead forced the ring far larger than its contents needed, which with
		// the mystery inside (a short label) left a vast empty hoop.
		if p == prev {
			return false
		}
		prev = p
	}
	return true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// clears reports whether every bead sits clear of the text block, that is: for
// any bead on a row the text occupies, the bead is at least half the text's
// width plus a gutter away from the centre.
// It must agree exactly with where drawRosary puts things, so it works in the
// same integer cell coordinates rather than in floats: a half-row disagreement
// between the two is precisely how a bead ends up on a letter.
func clears(rx, ry, n, textW, textH int) bool {
	cx, cy := centre(rx, ry)
	left, right := cx-textW/2-gutter, cx+textW/2+gutter
	top, bottom := cy-textH/2, cy+textH/2

	onRing := n - pendantLen
	if onRing < 1 {
		return true
	}
	for _, a := range arcAngles(onRing, rx, ry) {
		x := cx + int(math.Round(float64(rx)*math.Cos(a)))
		y := cy - int(math.Round(float64(ry)*math.Sin(a)))
		if y >= top && y <= bottom && x >= left && x <= right {
			return false
		}
	}
	return true
}

// drawRosary draws the ring, the pendant and the prayer inside, highlighting the
// bead at cursor.
//
// Two passes, and the order matters: this one places plain runes on the grid, so
// every cell is exactly one column wide and the arithmetic holds. styleGrid then
// adds colour. Doing it the other way round — styling a glyph as it is placed —
// puts escape bytes in a cell, and a cell that is not one column wide breaks
// every measurement after it.
func drawRosary(g ringGeometry, beads []Bead, cursor, glow, fade int, lines []string) string {
	// The geometry is handed in, already fixed: this function draws, it does not
	// decide how big the ring is. That separation is what keeps the ring still
	// while the prayers change.
	textW := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > textW {
			textW = w
		}
	}
	textH := len(lines)

	c := newCanvas(g.w, g.h)

	// cursor may be out of range deliberately: the closing screen passes -1 to draw
	// the ring with NO bead current, every bead at rest. Resolving that here rather
	// than at each use keeps the three cursor tests below from each needing a bounds
	// check, and means g.pos[cursor] is only ever indexed when there is a cursor.
	onBead := cursor >= 0 && cursor < len(beads)

	// The ring. The bead at the cursor is drawn as a halo rather than as its own
	// kind of bead, so the eye finds it by shape and not only by colour.
	for i, b := range beads {
		// Two beads can share a cell — the close of the rosary is a second visit to
		// the bead it began on. Whichever is drawn LAST would win, so the one that
		// is not current yields: otherwise standing on the first of them would see
		// its halo overwritten by the second's plain glyph.
		//
		// With no cursor neither yields; they draw the same resting glyph in the
		// same cell, so the result is identical either way.
		if onBead && i != cursor && g.pos[i] == g.pos[cursor] {
			continue
		}

		// The crucifix is several cells, not one: it is drawn from box-drawing lines
		// so that it lines up with the pendant (see cross.go). It is stamped as a
		// block and never becomes a halo — its shape carries meaning, and swapping
		// it for a bead would make the cross vanish exactly when you are praying the
		// Sign of the Cross on it. It shows selection by colour and weight instead.
		if b.Kind == Cross {
			for cell, r := range crossCells(g.pos[i]) {
				c.set(cell[0], cell[1], r)
			}
			continue
		}

		glyph := b.Kind.Glyph()
		if onBead && i == cursor {
			glyph = currentGlyph(glow)
		}
		c.set(g.pos[i][0], g.pos[i][1], []rune(glyph)[0])
	}

	// The mystery, centred in the ring. Each line is centred on its own here
	// rather than sharing a left edge: the box is a short label, not a prayer, and
	// a centred label sits better inside a circle.
	top := g.cy - textH/2
	for i, l := range lines {
		c.text(g.cx-lipgloss.Width(l)/2, top+i, l)
	}
	_ = textW

	return styleGrid(c, g, beads, cursor, glow, fade)
}
