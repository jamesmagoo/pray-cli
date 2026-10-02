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

// The ring is an arc with a gap at the bottom, where the pendant hangs on a real
// rosary. Angles are measured the usual way, counter-clockwise from east, with y
// flipped because rows grow downwards.
//
// gapArc is negative so the sequence runs counter-clockwise: the first bead sits
// just left of the pendant and the beads travel away from the crucifix, up the
// left side, round, and back down — the direction the fingers go.
const (
	gapStart = -math.Pi/2 - 0.45 // just right of straight down, measured CCW
	gapArc   = -(2*math.Pi - 0.9)
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
		if clears(rx, ry, n, textW, textH) && spacedOut(n, rx, ry) {
			break
		}
	}
	rx := ry * 2

	g := ringGeometry{
		rx: rx, ry: ry,
		w: rx*2 + 5,
		h: ry*2 + 3 + pendantRows() + 1,
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

	// The FIRST bead of the ring sits directly above the pendant, at the bottom of
	// the circle — it is where the loop begins and ends, so it belongs at the
	// join, not wherever even spacing happens to put it.
	//
	// It is pinned here explicitly, and the remaining beads are then spaced around
	// the arc above it. Spacing all of them together would walk the first bead off
	// to one side of the pendant.
	g.pos[pendantLen] = [2]int{g.cx, g.cy + g.ry}

	// Beads that share another bead's place take no slot of their own: a second
	// visit to a bead is drawn as that same bead. They are filled in last, once
	// every real position is known.
	var shared []int
	var own []int
	for i := pendantLen + 1; i < n; i++ {
		if beads[i].SameAs > 0 {
			shared = append(shared, i)
			continue
		}
		own = append(own, i)
	}

	// The decades: spaced evenly by arc length all the way round, starting just
	// past the pinned bead and coming back to just before it.
	//
	// One extra slot is asked for and its last entry dropped, so no bead lands on
	// top of the pinned one — the run's two ends are the same point on a closed
	// circle.
	if len(own) > 0 {
		for i, a := range arcAngles(len(own)+1, rx, ry) {
			if i == len(own) {
				break
			}
			g.pos[own[i]] = [2]int{
				g.cx + int(math.Round(float64(rx)*math.Cos(a))),
				g.cy - int(math.Round(float64(ry)*math.Sin(a))),
			}
		}
	}

	for _, i := range shared {
		g.pos[i] = g.pos[beads[i].SameAs-1]
	}
	return g
}

// arcAngles returns the angle for each of n beads, spaced evenly by DISTANCE
// along the ellipse rather than by angle.
//
// This is what stops the beads clumping. The ellipse has no closed form for arc
// length, so walk it in fine steps, add up the distance travelled, then place
// beads at equal fractions of the total. A thousand steps is far more than the
// ~70 beads need and still trivial to compute once at startup.
func arcAngles(n, rx, ry int) []float64 {
	if n <= 1 {
		return []float64{gapStart}
	}

	const steps = 2000
	// Walk the arc, recording the cumulative distance at each step.
	dist := make([]float64, steps+1)
	px := float64(rx) * math.Cos(gapStart)
	py := float64(ry) * math.Sin(gapStart)
	for k := 1; k <= steps; k++ {
		a := gapStart + gapArc*float64(k)/float64(steps)
		x := float64(rx) * math.Cos(a)
		y := float64(ry) * math.Sin(a)
		dist[k] = dist[k-1] + math.Hypot(x-px, y-py)
		px, py = x, y
	}
	total := dist[steps]

	// For each bead, find where along that walk it falls.
	angles := make([]float64, n)
	k := 0
	for i := 0; i < n; i++ {
		want := total * float64(i) / float64(n-1)
		for k < steps && dist[k+1] < want {
			k++
		}
		angles[i] = gapStart + gapArc*float64(k)/float64(steps)
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
func spacedOut(n, rx, ry int) bool {
	n -= pendantLen
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

	// The ring. The bead at the cursor is drawn as a halo rather than as its own
	// kind of bead, so the eye finds it by shape and not only by colour.
	for i, b := range beads {
		// Two beads can share a cell — the close of the rosary is a second visit to
		// the bead it began on. Whichever is drawn LAST would win, so the one that
		// is not current yields: otherwise standing on the first of them would see
		// its halo overwritten by the second's plain glyph.
		if i != cursor && g.pos[i] == g.pos[cursor] {
			continue
		}

		glyph := b.Kind.Glyph()
		// The cursor normally becomes a halo, but not on the crucifix: its shape
		// carries meaning, and swapping it for a bead would make the cross vanish
		// exactly when you are praying the Sign of the Cross on it. The cross
		// shows selection by colour and weight instead.
		if i == cursor && b.Kind != Cross {
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
