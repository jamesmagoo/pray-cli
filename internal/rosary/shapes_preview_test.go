package rosary

import (
	"fmt"
	"math"
	"testing"

	"charm.land/lipgloss/v2"
)

// A PREVIEW, not an assertion. It renders the rosary at each candidate ring shape
// so the choice can be made by eye, in a terminal, where escape sequences actually
// paint — which is the only honest place to judge a TUI's look.
//
// Kept in the tree rather than thrown away because the decision recurs: changing
// the bead count, the glyphs or the text inside the ring all move the trade it
// shows. Run it with scripts/shapes.sh.
//
// It asserts nothing and cannot fail. The numbers under each heading are the two
// faults in tension:
//
//   - flat run: how many beads land on one row. An ellipse is flattest where it
//     crosses the vertical axis, so a wide ring draws a straight line of beads
//     across its top and bottom while the sides curve.
//   - empty rows: a taller ring curves better, but spaces consecutive beads more
//     than a row apart and the chain reads as broken strands.
//
// Pentagon error is reported too, and is NOT a fault to minimise: the five big
// beads land 72 degrees apart only when the ring is a visual circle, which is the
// ratio that causes the flat runs. It is shown so the cost of roundness is visible,
// not so it can be chased.
func TestPreviewShapes(t *testing.T) {
	m := praying(t)

	for _, v := range []struct {
		name   string
		rx, ry int
	}{
		{"0  CURRENT (committed): rx=18 ry=9, ratio 2.00", 18, 9},
		{"1  rx=16 ry=10, ratio 1.60", 16, 10},
		{"2  rx=17 ry=11, ratio 1.55", 17, 11},
		{"3  rx=15 ry=10, ratio 1.50", 15, 10},
		{"4  rx=18 ry=11, ratio 1.64", 18, 11},
		{"5  rx=16 ry=9,  ratio 1.78", 16, 9},
		{"6  rx=14 ry=10, ratio 1.40", 14, 10},
	} {
		g := shapeAt(m.beads, v.rx, v.ry)
		stats := shapeStats(m.beads, g)
		fmt.Printf("\n\n  ═══ %s ═══\n  %s\n\n%s\n",
			v.name, stats, drawRosary(g, m.beads, 30, 0, 0, nil))
	}
}

// shapeAt builds a geometry at a forced size, with the pendant placed as the
// fixed version does.
func shapeAt(beads []Bead, rx, ry int) ringGeometry {
	n := len(beads)
	onRingSlots := 0
	for i := pendantLen; i < n; i++ {
		if beads[i].SameAs == 0 {
			onRingSlots++
		}
	}
	g := ringGeometry{rx: rx, ry: ry,
		w: rx*2 + 5,
		h: ry*2 + 3 + pendantRows() + crossRows() + 2,
	}
	g.cx, g.cy = centre(rx, ry)
	g.pos = make([][2]int, n)

	row := g.cy + g.ry + pendantRows() - 1
	if PendantGapAfter(pendantLen - 1) {
		row++
	}
	for i := 0; i < pendantLen && i < n; i++ {
		g.pos[i] = [2]int{g.cx, row}
		row--
		if PendantGapAfter(i) {
			row--
		}
	}
	var shared, own []int
	for i := pendantLen; i < n; i++ {
		if beads[i].SameAs > 0 {
			shared = append(shared, i)
			continue
		}
		own = append(own, i)
	}
	for i, a := range arcAngles(len(own), rx, ry) {
		g.pos[own[i]] = [2]int{
			g.cx + int(math.Round(float64(rx)*math.Cos(a))),
			g.cy - int(math.Round(float64(ry)*math.Sin(a))),
		}
	}
	for _, i := range shared {
		g.pos[i] = g.pos[beads[i].SameAs-1]
	}
	_ = onRingSlots
	return g
}

func shapeStats(beads []Bead, g ringGeometry) string {
	byRow := map[int]int{}
	cells := map[[2]int]bool{}
	for i := pendantLen; i < len(beads); i++ {
		if beads[i].SameAs > 0 {
			continue
		}
		byRow[g.pos[i][1]]++
		cells[g.pos[i]] = true
	}
	worst, empty := 0, 0
	for y := g.cy - g.ry; y <= g.cy+g.ry; y++ {
		if byRow[y] > worst {
			worst = byRow[y]
		}
		if byRow[y] == 0 {
			empty++
		}
	}
	note := ""
	if empty > 0 {
		note = fmt.Sprintf("  ⚠ %d EMPTY ROWS (chain breaks)", empty)
	}
	return lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf(
		"widest flat run: %d beads   distinct cells: %d/55%s", worst, len(cells), note))
}
