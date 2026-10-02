package rosary

import (
	"math"
	"testing"
)

// ringAngle is where a bead sits on the ring, in degrees, normalised to (-360, 0]
// measuring counter-clockwise from east — the same convention as ringStart.
func ringAngle(g ringGeometry, p [2]int) float64 {
	dx := float64(p[0]-g.cx) / float64(g.rx)
	dy := float64(g.cy-p[1]) / float64(g.ry)
	a := math.Atan2(dy, dx) * 180 / math.Pi
	if a > 0 {
		a -= 360
	}
	return a
}

// bigBeadAngles is where the five big beads sit, in the order prayed.
func bigBeadAngles(t *testing.T, m model) []float64 {
	t.Helper()
	var out []float64
	for i, b := range m.beads {
		if b.Kind != Large || b.SameAs > 0 {
			continue
		}
		out = append(out, ringAngle(m.ring, m.ring.pos[i]))
	}
	return out
}

// The five big beads form a pentagon: 72 degrees apart, all the way round.
//
// Nothing arranges them. Each decade is eleven beads of fifty-five, so evenly
// spacing fifty-five slots around a CLOSED loop puts every eleventh one at a fifth
// of the circle. The pentagon is a consequence of the loop being closed and the
// spacing being even — which is why this test is the one that would catch either
// of those being undone.
func TestTheBigBeadsFormAPentagon(t *testing.T) {
	m := praying(t)

	angles := bigBeadAngles(t, m)
	if len(angles) != 5 {
		t.Fatalf("found %d big beads on the ring, want 5", len(angles))
	}

	// The first sits at the bottom, where the pendant hangs.
	if got := angles[0]; math.Abs(got-(-90)) > 0.001 {
		t.Errorf("the first big bead is at %.2f°, want -90° (straight down)", got)
	}

	// Each step round the loop is 72 degrees, the fifth back to the first included.
	//
	// The step is taken modulo a full turn: angles are reported in (-360, 0], so
	// walking past -360 wraps to near 0 and a plain subtraction would read as -288
	// rather than 72. Normalising here is what makes the fifth gap comparable to
	// the other four instead of needing a special case.
	//
	// The tolerance is for cell rounding only: the maths is exact (see
	// TestTheSpacingMathsIsExact), but a bead can only sit on a whole cell, and at
	// rx=18 that is worth up to about 1.5 degrees.
	const tolerance = 5.0
	for i := range angles {
		from := angles[i]
		to := angles[(i+1)%len(angles)]
		step := math.Mod(from-to+360, 360)
		if math.Abs(step-72) > tolerance {
			t.Errorf("big beads %d to %d span %.2f°, want 72° (±%.0f)",
				i, (i+1)%len(angles), step, tolerance)
		}
	}
}

// The underlying angles are exact, before cells round them.
//
// Separated from the test above so a failure says WHICH broke: the geometry, or
// the rounding onto a coarse grid. Tightening steps in arcAngles was what brought
// this from 71.82 to 72.00.
func TestTheSpacingMathsIsExact(t *testing.T) {
	angles := arcAngles(55, 18, 9)
	if len(angles) != 55 {
		t.Fatalf("arcAngles returned %d angles, want 55", len(angles))
	}

	deg := func(i int) float64 { return angles[i] * 180 / math.Pi }

	if math.Abs(deg(0)-(-90)) > 0.0001 {
		t.Errorf("slot 0 is at %.4f°, want exactly -90°", deg(0))
	}
	for i := 11; i < 55; i += 11 {
		gap := deg(i-11) - deg(i)
		if math.Abs(gap-72) > 0.01 {
			t.Errorf("slots %d to %d span %.4f°, want 72.0000°", i-11, i, gap)
		}
	}
}

// Beads are spaced evenly by VISUAL distance, not by arc length on the ellipse.
//
// This is the fix for the clumping. Measuring in the ellipse's own units packed
// beads together at the top and bottom (where it is flat) and stretched them down
// the sides (where it is steep) — a worst-to-best ratio of 1.98, which is exactly
// what it looked like: "●●● ●●●" across the top and a thin string down each side.
//
// The ratio here is measured on the ROUNDED positions, so it cannot reach 1.0: a
// bead sits on a whole cell or not at all, and at this size the ideal 2.05-cell
// spacing rounds to 1 or 3 often enough to matter. The bound is set to catch the
// metric being wrong again, not to chase the grid.
func TestBeadsAreSpacedEvenlyAroundTheRing(t *testing.T) {
	m := praying(t)
	g := m.ring

	var prev [2]int
	first := true
	minGap, maxGap := math.Inf(1), 0.0
	for i := pendantLen; i < len(m.beads); i++ {
		if m.beads[i].SameAs > 0 {
			continue // drawn on another bead; takes no place of its own
		}
		p := g.pos[i]
		if !first {
			// Vertical distance scaled to columns: a row is two columns tall, so
			// this is the gap the EYE sees rather than the gap in grid units.
			d := math.Hypot(float64(p[0]-prev[0]), float64(p[1]-prev[1])*cellAspect)
			minGap, maxGap = math.Min(minGap, d), math.Max(maxGap, d)
		}
		prev, first = p, false
	}

	// Before the fix this was 1.00 .. 8.25. The 8.25 was a visible hole in the ring.
	if maxGap > 4.0 {
		t.Errorf("the widest gap between beads is %.2f cells; the ring has a hole in it", maxGap)
	}
	if ratio := maxGap / minGap; ratio > 3.0 {
		t.Errorf("gaps range %.2f..%.2f (ratio %.2f); the beads are unevenly spaced",
			minGap, maxGap, ratio)
	}
}

// Every bead on the ring gets its own cell: none is drawn on top of another.
func TestNoTwoRingBeadsShareACell(t *testing.T) {
	m := praying(t)

	seen := map[[2]int]int{}
	for i := pendantLen; i < len(m.beads); i++ {
		if m.beads[i].SameAs > 0 {
			continue
		}
		p := m.ring.pos[i]
		if prev, clash := seen[p]; clash {
			t.Errorf("beads %d and %d both sit at %v", prev, i, p)
		}
		seen[p] = i
	}
	if len(seen) != 55 {
		t.Errorf("%d distinct ring cells, want 55", len(seen))
	}
}

// The ring is a closed loop with no mouth: every row it spans holds at least one
// bead, so the chain reads as continuous rather than as broken strands.
//
// This is what rules out growing the ring. A larger ring spaces the beads further
// apart than one row, leaving blank rows down each side — a visible break. At
// rx=18 the beads are close enough that every row is covered.
func TestTheRingHasNoBreaks(t *testing.T) {
	m := praying(t)
	g := m.ring

	rows := map[int]bool{}
	for i := pendantLen; i < len(m.beads); i++ {
		if m.beads[i].SameAs == 0 {
			rows[g.pos[i][1]] = true
		}
	}

	for y := g.cy - g.ry; y <= g.cy+g.ry; y++ {
		if !rows[y] {
			t.Errorf("row %d of the ring holds no bead; the chain is broken there", y)
		}
	}
}

// The pendant hangs straight down from the bottom big bead, in one column.
//
// With the loop closed there is no gap for it to hang through: it descends from
// the bead the rosary opens and closes on. Everything therefore has to share the
// ring's centre column, or the pendant visibly kinks where it meets the loop.
func TestThePendantHangsStraightDownFromTheBottomBead(t *testing.T) {
	m := praying(t)
	g := m.ring

	// The first ring bead is at the bottom of the loop, on the centre column.
	bottom := g.pos[pendantLen]
	if bottom[0] != g.cx {
		t.Errorf("the bottom ring bead is at column %d, want the centre %d", bottom[0], g.cx)
	}
	if bottom[1] != g.cy+g.ry {
		t.Errorf("the bottom ring bead is on row %d, want the ring's bottom %d",
			bottom[1], g.cy+g.ry)
	}

	// And every pendant bead sits in that same column, the crucifix included.
	for i := 0; i < pendantLen; i++ {
		if g.pos[i][0] != g.cx {
			t.Errorf("pendant bead %d is at column %d, want %d", i, g.pos[i][0], g.cx)
		}
	}
}

// Both bead glyphs must sit in the same East Asian Width class.
//
// lipgloss.Width is not enough. It reports the width Unicode DECLARES, and every
// glyph considered here declares 1 — including "⬤", which many fonts nonetheless
// paint wider than a cell. The overflow goes rightward, so the bead appears half a
// cell right of where the grid put it: the big bead at the top of the pendant sat
// visibly off the centre line while every small bead was true, and it snapped into
// place when selected, because the halo glyph that replaces it is a different
// character with a different width.
//
// The property that predicts this is the width CLASS. "⬤" is N (neutral) and "●"
// is A (ambiguous); a terminal may size the two differently and nothing in the
// declared width says so. Keeping both glyphs in one class is what makes them
// agree, whatever the font does.
func TestBeadGlyphsShareAWidthClass(t *testing.T) {
	big := eastAsianWidthClass(bigBead)
	small := eastAsianWidthClass(smallBead)

	if big != small {
		t.Errorf("bigBead %q is width class %s but smallBead %q is %s; "+
			"mixed classes let a font draw one wider than its cell and the bead "+
			"then sits off the centre line",
			bigBead, big, smallBead, small)
	}

	// The crucifix is NOT checked against the beads. It is class N while the
	// circles are class A, but it sits alone on the pendant's bottom row with no
	// bead beside it to disagree with, and the screenshot shows it landing true.
	// A rule that forced it into class A would rule out every cross glyph for no
	// gain.
}

// eastAsianWidthClass returns the UAX #11 class of a single-rune glyph: "A"
// (ambiguous), "N" (neutral), "W" (wide), and so on.
//
// Go's standard library does not expose the property, so the ranges that matter
// for bead glyphs are listed here. It is deliberately narrow: it covers the
// geometric shapes these glyphs come from and reports "?" for anything else,
// which fails the test above rather than guessing.
func eastAsianWidthClass(glyph string) string {
	r := []rune(glyph)
	if len(r) != 1 {
		return "?"
	}
	switch c := r[0]; {
	// Geometric Shapes and the dingbats/symbols that are class A (ambiguous).
	case c == 0x00B7, // ·  MIDDLE DOT
		c == 0x25C6, c == 0x25C7, // ◆ ◇
		c == 0x25CB,                // ○
		c >= 0x25CE && c <= 0x25D1, // ◎ ● ◐ ◑
		c == 0x2022,                // •
		c == 0x2B58:                // ⭘
		return "A"
	// Box drawing (U+2500-257F) is class A throughout: it is what the crucifix is
	// built from, and the reason it lines up with the pendant whatever the font
	// does.
	case c >= 0x2500 && c <= 0x257F:
		return "A"
	// Block elements, which are class A EXCEPT for a few — listed as N below.
	case c >= 0x2580 && c <= 0x258F, c == 0x2592, c == 0x2593, c == 0x2594, c == 0x2595:
		return "A"
	// Class N (neutral): the oversized shapes that cause the trouble.
	case c == 0x2B24, // ⬤ BLACK LARGE CIRCLE
		c == 0x2B22, c == 0x2B23, // ⬢ ⬣
		c == 0x25C9,              // ◉ FISHEYE
		c == 0x25CD,              // ◍
		c == 0x25E6,              // ◦
		c == 0x2720,              // ✠ MALTESE CROSS
		c == 0x2739, c == 0x273B, // ✹ ✻
		c == 0x2590,                // ▐ RIGHT HALF BLOCK — the odd one out of the block elements
		c == 0x2591,                // ░ LIGHT SHADE
		c >= 0x2596 && c <= 0x259F, // the quadrant blocks
		c == 0x29BF:                // ⦿
		return "N"
	}
	return "?"
}

// The crucifix's stem sits on the pendant's column, and its arms are symmetric
// about it.
//
// This is the whole reason the cross is drawn from box-drawing lines rather than
// as a single glyph. Every cross glyph worth using (✠ ✝ ✞ ☩ ✚) is East Asian
// Width class N while the beads are class A, and a font may paint a class-N glyph
// wider than its cell — the overflow goes rightward and the cross drifts off the
// centre line. The box-drawing block is class A throughout, so this alignment is
// one the font cannot undo.
func TestTheCrucifixIsCentredOnThePendant(t *testing.T) {
	m := praying(t)
	g := m.ring

	// Find the crucifix's bead.
	cross := -1
	for i, b := range m.beads {
		if b.Kind == Cross {
			cross = i
			break
		}
	}
	if cross < 0 {
		t.Fatal("the rosary has no crucifix")
	}

	cells := crossCells(g.pos[cross])
	if len(cells) == 0 {
		t.Fatal("the crucifix occupies no cells")
	}

	// The stem: every row of the cross must have a cell on the pendant's column.
	minY, maxY := 1<<30, -1
	minX, maxX := 1<<30, -1
	for c := range cells {
		minY, maxY = min(minY, c[1]), max(maxY, c[1])
		minX, maxX = min(minX, c[0]), max(maxX, c[0])
	}
	for y := minY; y <= maxY; y++ {
		if _, ok := cells[[2]int{g.cx, y}]; !ok {
			t.Errorf("row %d of the crucifix has nothing on the pendant's column %d", y, g.cx)
		}
	}

	// And the arms reach equally far either side, so the cross reads as centred
	// rather than merely as touching the column.
	if left, right := g.cx-minX, maxX-g.cx; left != right {
		t.Errorf("the crucifix reaches %d cells left of centre and %d right; it is lopsided",
			left, right)
	}
}

// Every character of the crucifix is East Asian Width class A, like the beads.
//
// This is the rule the whole of cross.go exists to satisfy, so it is worth
// asserting directly rather than trusting the art to stay correct. The trap is
// live: the block-element range is mostly class A, but "▐" RIGHT HALF BLOCK is
// class N, and reaching for it to taper an arm would quietly reintroduce the drift
// the lines were adopted to remove.
func TestTheCrucifixIsBuiltFromClassAGlyphs(t *testing.T) {
	for y, line := range crossArt {
		for x, r := range line {
			if r == ' ' {
				continue
			}
			if class := eastAsianWidthClass(string(r)); class != "A" {
				t.Errorf("crossArt[%d] column %d is %q, width class %s; want A — "+
					"a class-N glyph may be painted wider than its cell and push the "+
					"cross off the pendant's column", y, x, string(r), class)
			}
		}
	}
}

// The art has a true centre column, and it is the stem.
//
// crossOrigin centres the cross by halving its width, so an even width has no
// middle cell to put on the pendant's column — the whole cross would sit half a
// cell off, which is the fault this file was written to cure.
func TestTheCrucifixArtHasACentreColumn(t *testing.T) {
	w := crossW()
	if w%2 == 0 {
		t.Fatalf("crossArt is %d cells wide; an even width has no centre column", w)
	}
	for y, line := range crossArt {
		if got := len([]rune(line)); got != w {
			t.Errorf("crossArt[%d] is %d cells wide, want %d; the rows must be a rectangle",
				y, got, w)
		}
	}
	// And the centre column is the stem: something on every row.
	for y, line := range crossArt {
		if r := []rune(line)[w/2]; r == ' ' {
			t.Errorf("crossArt[%d] has a gap in its centre column; the stem must be unbroken", y)
		}
	}
}

// Every cell of the crucifix is drawn, and none is clipped off the canvas.
//
// canvas.set silently ignores out-of-bounds writes, so a canvas one row too short
// loses the cross's foot with no error anywhere — exactly the kind of fault that
// is invisible until someone looks at the screen.
func TestTheWholeCrucifixFitsOnTheCanvas(t *testing.T) {
	m := praying(t)
	g := m.ring

	cross := -1
	for i, b := range m.beads {
		if b.Kind == Cross {
			cross = i
			break
		}
	}
	if cross < 0 {
		t.Fatal("the rosary has no crucifix")
	}

	for c := range crossCells(g.pos[cross]) {
		if c[0] < 0 || c[0] >= g.w || c[1] < 0 || c[1] >= g.h {
			t.Errorf("the crucifix's cell %v falls outside the %dx%d canvas", c, g.w, g.h)
		}
	}

	// And it really is on screen: count the drawn runes against the art's.
	drawn := 0
	out := plain(drawRosary(g, m.beads, 30, 0, 0, nil))
	for _, r := range out {
		for _, line := range crossArt {
			for _, want := range line {
				if want != ' ' && r == want {
					drawn++
				}
			}
		}
	}
	if drawn == 0 {
		t.Error("none of the crucifix was drawn")
	}
}
