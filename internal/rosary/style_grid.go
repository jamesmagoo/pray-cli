package rosary

import (
	"strings"
)

// styleGrid colours the finished grid.
//
// Why this is a separate pass: a styled glyph is a run of escape bytes, not one
// cell, so writing it into the grid would break every column measurement after
// it. The grid therefore holds plain runes while the geometry is computed, and
// colour goes on last, by rebuilding each line rune by rune and wrapping the
// runes we care about.
func styleGrid(c *canvas, g ringGeometry, beads []Bead, cursor, glow, fade int) string {
	// Which cells are beads, and which of those is the current one. A map keyed
	// by position keeps this O(1) per cell and avoids re-deriving the geometry.
	type beadAt struct {
		kind    Kind
		current bool
	}
	at := make(map[[2]int]beadAt, len(beads))
	for i, b := range beads {
		at[g.pos[i]] = beadAt{kind: b.Kind, current: i == cursor}
	}

	var out strings.Builder
	for y, row := range c.cells {
		for x, r := range row {
			if r == ' ' {
				out.WriteRune(' ')
				continue
			}
			if b, ok := at[[2]int{x, y}]; ok {
				out.WriteString(beadStyle(b.kind, b.current, glow).Render(string(r)))
				continue
			}
			// Everything else on the grid is the mystery, in the middle of the
			// ring. Its two lines are told apart by row: the mystery sits on the
			// ring's centre line and is the brighter, since it is the thing being
			// contemplated; the set's name above it is quieter.
			if y >= g.cy {
				out.WriteString(mysteryStyle.Render(string(r)))
				continue
			}
			out.WriteString(setNameStyle.Render(string(r)))
		}
		if y < len(c.cells)-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}
