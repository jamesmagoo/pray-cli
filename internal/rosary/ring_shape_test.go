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
	// Class N (neutral): the oversized shapes that cause the trouble.
	case c == 0x2B24, // ⬤ BLACK LARGE CIRCLE
		c == 0x2B22, c == 0x2B23, // ⬢ ⬣
		c == 0x25C9,              // ◉ FISHEYE
		c == 0x25CD,              // ◍
		c == 0x25E6,              // ◦
		c == 0x2720,              // ✠ MALTESE CROSS
		c == 0x2739, c == 0x273B, // ✹ ✻
		c == 0x29BF: // ⦿
		return "N"
	}
	return "?"
}
