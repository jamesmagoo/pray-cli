package rosary

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// The colours of RENDERING.md, kept here while the rosary is the only thing
// using them. When internal/render lands they move to render/theme.go and this
// file imports them instead, so there is one definition of gold.
var gold = lipgloss.Color("#C9A227")

// beadBlue is the small beads — the Hail Marys and the pendant's chain.
//
// The only colour in the rosary that is not gold. The smalls are most of the ring
// (50 of 55 beads), so colouring them apart from the five big ones separates the
// decades from their junctions by hue rather than by size alone, which the two
// circle glyphs do only quietly.
var beadBlue = lipgloss.Color("#5B8DD6")

// currentRest is the colour the bead under the cursor sits at between moves:
// brighter than the ring so the eye finds it at a glance.
var currentRest = lipgloss.Lighten(gold, 0.35)

// render turns the model into the text on screen.
//
// It is deliberately a plain function of the model rather than a method: it
// takes state and returns a string, so a test can call it with any model and
// compare the result without starting a program.
func render(m model) string {
	switch m.phase {
	case choosing:
		return place(m, chooser(m))
	case finished:
		return placeFinished(m, completion(m))
	case departing:
		return placeFinished(m, farewell(m))
	}

	// INSIDE the ring: the mystery.
	//
	// Written as plain text onto the grid and styled afterwards, because the grid
	// counts cells and an escape sequence is not one cell wide. styleGrid colours
	// these rows by position.
	inside := mysteryLines(m.set, m.announced())

	rosary := drawRosary(m.ring, m.beads, m.cursor, m.glow, m.fade, inside)

	// BESIDE the ring, to its right: the prayer. Joined horizontally and centred
	// vertically, so the prayer's middle lines up with the ring's middle while the
	// ring itself keeps its own geometry.
	//
	// To move the prayer elsewhere, change this one JoinHorizontal — the ring does
	// not know the prayer exists.
	block := lipgloss.JoinHorizontal(lipgloss.Center, rosary, gutterCols, prayerPanel(m))

	return place(m, block)
}

// gutterCols is the space between the ring and the prayer beside it.
const gutterCols = "    "

// prayerPanel is the prayer as it appears to the right of the rosary: the bead's
// name, then the words.
//
// Unlike the mystery inside the ring, this is styled here rather than on the grid,
// because it never touches the grid — it is joined to the finished ring as a
// block, so escapes in it cannot disturb any cell arithmetic.
func prayerPanel(m model) string {
	w := m.words()

	// The heading names the PRAYER being said, not the bead.
	//
	// A bead can carry several prayers — the junction bead holds the Glory Be, the
	// Fatima Prayer and the next Our Father — so naming the bead showed "Our
	// Father" above the words of the Glory Be. The bead is where you are; the
	// prayer is what you are saying, and that is what the heading is for.
	rows := []string{
		titleStyle.Render(w.Title),
		"",
	}
	// The prayer's own line breaks, never reflowed.
	for _, l := range w.Lines {
		rows = append(rows, prayerStyle(m.fade).Render(l))
	}

	// A FIXED width, not the width of this prayer.
	//
	// Sizing the panel to its contents makes the whole screen jump: the block gets
	// narrower for a short prayer and Place re-centres it, sliding the rosary
	// sideways on every move. Reserving the longest prayer's width keeps every
	// frame the same size; short prayers just leave space on the right.
	return lipgloss.NewStyle().
		Width(m.panelW).
		MaxWidth(m.panelW).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// prayerStyle is the prayer's colour at this point in its fade-in. Past the fade
// it is unstyled, so the text sits in the terminal's own foreground colour.
func prayerStyle(fade int) lipgloss.Style {
	if fade > 1 {
		return fadeStyle(fade)
	}
	return lipgloss.NewStyle()
}

// mysteryLines is the mystery as plain text for the middle of the ring: the set's
// name, then the mystery itself.
//
// Plain, not styled, because it is written onto the grid cell by cell — see
// drawRosary. Returns nothing before the first mystery is announced, so the ring
// is simply empty for the pendant prayers.
func mysteryLines(set MysterySet, announced int) []string {
	name, ok := set.Mystery(announced)
	if !ok {
		return nil
	}

	// The ring IS the frame, so there is no border here: one inside another put the
	// beads on the border, two frames competing for the same space.
	//
	// The set's name is NOT wrapped: it is a title and breaks badly ("The Glorious /
	// Mysteries"), where a mystery's name has natural seams. The ring is sized to
	// clear whatever goes in it, so the set name is simply the widest line it has to
	// accommodate — see mysteryWidth.
	rows := []string{set.Name, ""}
	return append(rows, wrapMystery(name)...)
}

// mysteryWidth is the widest a mystery's name may be before it wraps.
//
// The ring is sized to clear whatever goes inside it, so an over-long name grows
// the whole rosary until beads sit more than a row apart and the chain breaks
// into strands.
//
// 20 by measurement, not taste: the width at which the ring comes out 16x10, the
// roundest shape keeping every bead on its own cell with no row left empty.
// Narrower does not shrink it further — the beads set the size below that. So
// this is really a dial on the RING. Change it and check TestTheRingHasNoBreaks.
//
// It bounds the SET's name too, which at 23 cells is otherwise the widest line
// inside the ring.
const mysteryWidth = 20

// wrapMystery breaks a mystery's name onto as few lines as will fit mysteryWidth,
// splitting only at spaces and BALANCING the lines it produces.
//
// Balanced, not greedy. Greedy wrapping fills each line before breaking, which is
// right for prose and wrong for a title centred in a ring: it gives "The Crowning
// with" over a lone "Thorns", where balancing puts the break where a person would,
// at "The Crowning" over "with Thorns".
//
// It tries every split point for two lines, then every pair for three, keeping the
// arrangement whose longest line is shortest. Twenty names of five or six words:
// the cost is nothing and no name needs a special case.
func wrapMystery(name string) []string {
	if lipgloss.Width(name) <= mysteryWidth {
		return []string{name}
	}

	words := strings.Fields(name)
	for lines := 2; lines <= len(words); lines++ {
		if best := balancedSplit(words, lines); best != nil {
			return best
		}
	}
	return words // one word per line: nothing shorter is possible
}

// balancedSplit arranges words onto exactly n lines, each within mysteryWidth,
// choosing the split whose longest line is shortest. Returns nil if n lines
// cannot hold them.
func balancedSplit(words []string, n int) []string {
	var best []string
	bestWidest := 1 << 30

	// cuts holds the index after each line break; walk every combination.
	var walk func(start int, cuts []int)
	walk = func(start int, cuts []int) {
		if len(cuts) == n-1 {
			lines := linesFrom(words, cuts)
			widest := 0
			for _, l := range lines {
				w := lipgloss.Width(l)
				if w > mysteryWidth {
					return // does not fit
				}
				if w > widest {
					widest = w
				}
			}
			if widest < bestWidest {
				best, bestWidest = lines, widest
			}
			return
		}
		for i := start + 1; i < len(words); i++ {
			walk(i, append(cuts, i))
		}
	}
	walk(0, nil)
	return best
}

// linesFrom joins words into lines broken at the given indices.
func linesFrom(words []string, cuts []int) []string {
	lines := make([]string, 0, len(cuts)+1)
	prev := 0
	for _, c := range append(append([]int{}, cuts...), len(words)) {
		lines = append(lines, strings.Join(words[prev:c], " "))
		prev = c
	}
	return lines
}

// place centres a block in the window, or says the window is too small. Before the
// first WindowSizeMsg it returns the block unplaced: centring inside a 0x0 box
// would collapse it.
//
// The size needed is MEASURED from the block rather than declared as a constant,
// so it cannot drift as the ring, the panel or the prayers change.
func place(m model, block string) string {
	if m.width == 0 || m.height == 0 {
		return block
	}
	w, h := lipgloss.Width(block), lipgloss.Height(block)
	if tooSmall(m, w, h) {
		return tooSmallNotice(m, w, h)
	}
	centred := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block)
	return withHint(m, centred)
}

// withHint writes the key hint into the bottom-left corner of a full screen.
//
// It is drawn ONTO the finished screen rather than joined to the rosary, which is
// the point: chrome should not take part in the layout, or it competes with the
// prayer and pads out the ring's height. In a corner it is available without
// being read.
func withHint(m model, screen string) string {
	rows := strings.Split(screen, "\n")
	if len(rows) < 2 {
		return screen
	}

	// The key names are brighter than what they do, so the eye picks out "space"
	// and "x" at a glance without the line as a whole competing with the prayer.
	text := keyStyle.Render("space") + hintStyle.Render(" next    ") +
		keyStyle.Render("←") + hintStyle.Render(" back    ") +
		keyStyle.Render("x") + hintStyle.Render(" finish")

	// Second row from the bottom, two columns in: clear of the very edge, where
	// terminals sometimes put scrollbars or shells put a prompt.
	y := len(rows) - 2
	rows[y] = overlay(rows[y], text, 2)
	return strings.Join(rows, "\n")
}

// overlay writes s onto row at display column x, keeping the row's width.
//
// It walks the row counting DISPLAY columns rather than bytes, so escape sequences
// already in the row (a bead's colour, say) do not shift the position. This is the
// same reason the grid holds plain runes: once escapes are in a string, byte
// offsets and columns are different things.
func overlay(row, s string, x int) string {
	w := lipgloss.Width(s)

	var b strings.Builder
	col := 0
	inEsc := false
	for _, r := range row {
		switch {
		case r == 0x1b:
			inEsc = true
			b.WriteRune(r)
			continue
		case inEsc:
			b.WriteRune(r)
			if r == 'm' {
				inEsc = false
			}
			continue
		}

		if col == x {
			b.WriteString(s)
		}
		// Skip the cells the overlay covers, so the row keeps its width.
		if col < x || col >= x+w {
			b.WriteRune(r)
		}
		col++
	}
	return b.String()
}

// No counts anywhere on screen. "Hail Mary 3 of 10" and "bead 11 of 68" turn
// praying into progress-watching: the eye goes to the number instead of the words,
// and the rosary becomes a task with a completion bar. The beads already show where
// you are, which is the right place for it — in the object, not in text.
// TestNoCountsOnScreen holds this.

// Bead colours. The big beads and the crucifix are gold, the small beads blue;
// the bead being prayed is brighter and bold so it reads as "you are here" even
// with colour off (NO_COLOR), which a foreground change alone would not survive.
var (
	// The small beads, and the only non-gold thing on the ring.
	//
	// NOT Faint, which would dim the blue to mud — choosing a hue and then half
	// throwing it away. The smalls recede by hue and by the hollow glyph instead.
	beadDim   = lipgloss.NewStyle().Foreground(beadBlue)
	beadLarge = lipgloss.NewStyle().Foreground(gold)
	// The resting current bead: bright and bold, but no Reverse — see glowStyle.
	// currentRest is named separately because glowRamp must END on exactly this
	// colour: if the flare faded to anything else, the bead would visibly jump
	// when the animation stopped.
	beadCurrent = lipgloss.NewStyle().Foreground(currentRest).Bold(true)
	crossBead   = lipgloss.NewStyle().Foreground(gold).Bold(true)
)

// beadStyle picks the style for a bead: current beats kind, since knowing where
// you are matters more than what sort of bead it is.
//
// The current bead's colour depends on glow, so it is the one style that is built
// per frame rather than once at startup. That is the whole cost of the animation.
func beadStyle(k Kind, current bool, glow int) lipgloss.Style {
	switch {
	case current && glow > 0:
		return glowStyle(glow)
	case current:
		return beadCurrent
	case k == Cross:
		return crossBead
	case k == Large:
		return beadLarge
	default:
		return beadDim
	}
}

// Styles for the opening screen, and the text inside the ring.
var (
	// The frame around the chooser. Padding(vertical, horizontal): without the
	// vertical 1 the border sits directly on the crown of the cross and on the
	// keys, which reads as tight however wide the box is.
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(gold).
			Padding(1, 6)

	// The set's name is secondary to the mystery itself, so it is quieter.
	setNameStyle = lipgloss.NewStyle().Foreground(gold).Faint(true)

	// The opening screen's title, in the beads' blue and bold. Blue rather than the
	// gold used for every other title: it is Our Lady's name, and her colour.
	rosaryTitleStyle = lipgloss.NewStyle().Foreground(beadBlue).Bold(true)

	// The second title line, in the same blue but unbolded — a subtitle, not a
	// second title.
	subtitleStyle = lipgloss.NewStyle().Foreground(beadBlue)

	// When a set is traditionally prayed. Faint: it is guidance, and should be
	// available to the eye without asking for it — the choice is still the user's.
	daysStyle = lipgloss.NewStyle().Foreground(beadBlue).Faint(true)

	// The rule under the title: gold like the frame it sits inside, and faint, so
	// it divides without competing with either the title or the list.
	dividerStyle = lipgloss.NewStyle().Foreground(gold).Faint(true)

	mysteryStyle = lipgloss.NewStyle().Foreground(currentRest).Bold(true)

	titleStyle = lipgloss.NewStyle().Foreground(gold).Bold(true)

	dimStyle = lipgloss.NewStyle().Faint(true)

	// Quiet, but legible: the keys should be readable without hunting for them,
	// while still sitting below the prayer and the beads in the visual order.
	hintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8578"))

	// The key names themselves: gold, so they read as the interactive part.
	keyStyle = lipgloss.NewStyle().Foreground(gold).Faint(true)
)

// widestPrayer is the width to reserve for the prayer panel: the widest line of
// any prayer in the rosary, and of any bead's name.
//
// Measured over the whole sequence once, so the panel never changes size and the
// layout never shifts. See prayerPanel.
func widestPrayer(beads []Bead) int {
	w := 0
	for _, b := range beads {
		for _, says := range b.Says {
			// The heading is the prayer's TITLE (see prayerPanel), not the bead's
			// name, so that is what has to be measured: a junction bead is named
			// "Our Father" but also shows "Glory Be" and "Fatima Prayer".
			if x := lipgloss.Width(says.Title); x > w {
				w = x
			}
			for _, l := range says.Lines {
				if x := lipgloss.Width(l); x > w {
					w = x
				}
			}
		}
	}
	return w
}
