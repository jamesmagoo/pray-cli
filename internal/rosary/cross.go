package rosary

// The crucifix, drawn from box-drawing lines rather than as a single glyph.
//
// WHY IT IS NOT ONE CHARACTER. Every cross glyph worth using — ✠ ✝ ✞ ☩ ✚ ✛ —
// is East Asian Width class N, while the bead glyphs are class A. A font may
// paint a class-N glyph wider than its cell, and the overflow goes rightward, so
// the crucifix sat about 17px right of the pendant's column: visibly off the
// centre line that every bead above it sits on. lipgloss.Width cannot catch it,
// because it reports the width Unicode declares, not what is painted. See
// ROSARY-TUI.md §"declared width is not painted width".
//
// The box-drawing block is class A throughout, and those glyphs are designed to
// join across cell boundaries, so a cross assembled from them lines up exactly.
// It costs the crucifix its one-cell simplicity, which is the trade: see
// crossCells for what that means for the grid.

// crossArt is the crucifix, row by row. Change it here and everything follows —
// the grid write, the colour pass and the collision check all derive from it.
//
// It must have an odd width so it has a true centre column, and that column must
// be the stem: crossOrigin centres it on the pendant by that column alone.
//
// Alternatives that also work, if this one ever wants replacing:
//
//	light:      │      arms high:    ┃
//	         ──┼──                   ┃
//	           │                  ━━╋━━
//	           │                    ┃
var crossArt = []string{
	"  ┃  ",
	"━━╋━━",
	"  ┃  ",
	"  ┃  ",
}

// crossW and crossH are the crucifix's size in cells.
func crossW() int { return len([]rune(crossArt[0])) }
func crossH() int { return len(crossArt) }

// crossOrigin is the top-left cell of the crucifix, given where its bead sits.
//
// The bead's position is where the STEM goes — the cell the pendant's column runs
// down — so the art is shifted left by half its width to put the stem there. The
// art hangs downward from that row, because the bead marks where the cross joins
// the chain, not where the cross is centred.
func crossOrigin(pos [2]int) (x, y int) {
	return pos[0] - crossW()/2, pos[1]
}

// crossCells is every cell the crucifix occupies, with the rune in each.
//
// Both passes need this. drawRosary writes the runes; styleGrid has to know which
// cells belong to the cross so it colours them as the crucifix rather than as the
// mystery text — its map is keyed by position, and a cross that registered only
// its stem would have its arms coloured as though they were words inside the ring.
func crossCells(pos [2]int) map[[2]int]rune {
	ox, oy := crossOrigin(pos)
	cells := make(map[[2]int]rune, crossW()*crossH())
	for dy, line := range crossArt {
		for dx, r := range []rune(line) {
			if r == ' ' {
				continue // the art's padding is not part of the cross
			}
			cells[[2]int{ox + dx, oy + dy}] = r
		}
	}
	return cells
}

// crossRows is how many rows below its bead the crucifix needs.
//
// The canvas has to be tall enough for it, or the lower half is silently clipped:
// canvas.set ignores anything out of bounds, so a cross that does not fit simply
// loses its foot with no error anywhere.
func crossRows() int { return crossH() - 1 }

// crossMark is the single cell that identifies the crucifix: the crossing where
// the arms meet the stem.
//
// It is the centre of the art's crossbar row. Useful because it appears exactly
// once in the rosary — the beads are circles and the chain has no lines — so
// "where is the cross?" stays one search even though the cross is a block.
func crossMark() string {
	bar := []rune(crossArt[crossBarRow()])
	return string(bar[crossW()/2])
}

// crossBarRow is which row of the art holds the arms: the first row that has
// anything outside the stem column.
//
// Derived rather than declared, so moving the arms up or down in crossArt does
// not need a constant updated to match — the kind of pairing that silently rots.
func crossBarRow() int {
	stem := crossW() / 2
	for y, line := range crossArt {
		for x, r := range []rune(line) {
			if x != stem && r != ' ' {
				return y
			}
		}
	}
	return 0
}

// malteseCross is the crucifix as a single glyph, for the farewell screen.
//
// It is NOT used on the rosary itself. "✠" is East Asian Width class N while the
// beads are class A, so a font may paint it wider than its cell and the overflow
// pushes it right of the pendant's column — which is the whole reason the rosary's
// crucifix is drawn from lines instead.
//
// None of that matters on the farewell, where the cross stands alone above three
// words with no column to line up with: nothing for it to be out of true against.
// So the handsomer glyph is used in the one place it costs nothing.
const malteseCross = "✠"
