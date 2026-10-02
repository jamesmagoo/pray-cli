package rosary

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// The colours of RENDERING.md, kept here while the rosary is the only thing
// using them. When internal/render lands they move to render/theme.go and this
// file imports them instead, so there is one definition of gold.
//
// Only gold is used so far: the ring is gold and the prayer keeps the terminal's
// own colour, which is RENDERING.md's restraint principle. Purple is reserved
// there for the intention line and arrives with it.
var gold = lipgloss.Color("#C9A227")

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

	// INSIDE the ring: the mystery, in its bordered box.
	//
	// The box is drawn as plain text on the grid (borders included) and styled
	// afterwards, because the grid has to count cells and an escape sequence is not
	// one cell wide. styleGrid colours these rows by position — see mysteryRows.
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
	// Sizing the panel to its contents makes the whole screen jump: the joined
	// block gets narrower for a short prayer, and Place re-centres it, so the
	// rosary slides sideways every time you move to a different prayer. Measured
	// before this fix: the panel swung between 26 and 44 columns and the ring
	// moved 9 columns left and right.
	//
	// Reserving the width of the longest prayer in the whole rosary means the
	// block is the same size on every frame, so nothing moves. Short prayers
	// simply leave empty space on the right.
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

	// No border. A box inside the ring meant the ring had to be large enough to
	// clear it, and the beads ended up sitting on the border — two frames
	// competing for the same space. The ring IS the frame; the mystery just sits
	// inside it, styled.
	//
	// Plain text, because these rows are written onto the grid cell by cell and an
	// escape sequence is not one cell wide. Colour is added in styleGrid.
	return []string{set.Name, "", name}
}

// place centres a block in the window, or returns it unplaced before the first
// WindowSizeMsg arrives — centring inside a 0x0 box would collapse it.
func place(m model, block string) string {
	if m.width == 0 || m.height == 0 {
		return block
	}
	centred := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block)
	return withHint(m, centred)
}

// withHint writes the key hint into the bottom-left corner of a full screen.
//
// It is drawn ONTO the finished screen rather than joined to the rosary, which is
// the point: chrome should not take part in the layout. Inside the ring it was
// competing with the prayer for attention and padding out the ring's height; in a
// corner it is available without being read.
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

// farewell is the last screen: a cross and a closing word, fading to black.
//
// Deliberately almost nothing. Everything else — the ring, the prayers, the
// mystery, the keys — is gone, so the fade has one thing to carry and the screen
// empties as it dims. A farewell that still had the rosary on it would be the
// rosary dimming; this is the rosary already put down.
//
// The cross rather than a bead: it is where the rosary begins and ends, and it is
// the one glyph here that is not a bead among beads.
//
// It is the single-glyph "✠" rather than the ring's line-drawn crucifix. The lines
// exist to keep the crucifix true to the pendant's column (see cross.go); here
// there is no column, just a cross over three words, so the glyph that looks
// better is the right one.
func farewell(m model) string {
	cross := departCrossStyle(m.depart).Render(malteseCross)
	words := departStyle(m.depart).Render(farewellWords)

	// Centred as a block on the widest of the two, so the cross sits over the
	// middle of the words rather than over their first column. JoinVertical centres
	// line by line, so each line must already be padded to the join's width.
	w := max(lipgloss.Width(words), lipgloss.Width(cross))
	centred := lipgloss.NewStyle().Width(w).Align(lipgloss.Center)

	return lipgloss.JoinVertical(lipgloss.Center,
		centred.Render(cross),
		"",
		centred.Render(words),
	)
}

// farewellWords is the last thing on screen. Change it here.
//
// "Go in peace" is the dismissal — what is said at the end of the Mass, so it is
// the words the rosary's own tradition ends on rather than a greeting invented for
// a CLI.
const farewellWords = "Go in peace."

// completion is the closing screen: the rosary is prayed.
//
// It shows the whole rosary one last time with no bead current, which is the
// transition doing the work: the cursor simply lifts off the beads. Through the
// rosary exactly one bead was lit at a time, so a ring with nothing lit reads
// immediately as "no longer in progress" — the same object, at rest.
//
// cursor = -1 is what lifts it: no bead index can equal it, so styleGrid finds no
// current bead and every bead draws in its resting colour. That is cheaper and
// truer than a second drawing path for the finished ring, and it means the ring
// cannot drift out of agreement with the one just prayed — it IS the same call.
func completion(m model) string {
	ring := drawRosary(m.ring, m.beads, -1, 0, 0, finishedLines())

	// THREE blocks, not one: the rosary, what was prayed, and the keys. They are
	// separate because they are three different kinds of thing, and the gaps between
	// them are what says so — the keys are chrome and should not read as part of the
	// closing words.
	said := lipgloss.JoinVertical(lipgloss.Center,
		"",
		titleStyle.Render("The rosary is prayed."),
		// The set just completed, named: what was contemplated is the thing worth
		// stating at the end, not a count of beads or a time taken.
		setNameStyle.Render(m.set.Name),
	)

	keys := keyStyle.Render("space") + hintStyle.Render(" pray again      ") +
		keyStyle.Render("x") + hintStyle.Render(" finish")

	// Stacked, not side by side: the prayer panel is gone, so the ring is the only
	// thing with width and the text belongs under it rather than beside empty
	// space. Centred on the ring's own width so the block has a single axis.
	//
	// JoinVertical centres line by line, not block by block, so each block must
	// already be padded to the join's width or the result comes out ragged. The
	// ring is the wider of the three, so it sets the width.
	w := lipgloss.Width(ring)
	centred := lipgloss.NewStyle().Width(w).Align(lipgloss.Center)

	return lipgloss.JoinVertical(lipgloss.Center,
		trimBlankRows(ring),
		centred.Render(said),
		// keysGap blank ROWS. A block of n newlines is n+1 lines, so the repeat is
		// one short of the gap — off by one here is a visible row.
		strings.Repeat("\n", keysGap-1),
		centred.Render(keys),
	)
}

// keysGap is how many blank rows sit between the closing words and the keys.
//
// The keys are chrome: available, but set apart from the rosary and from what was
// prayed, so the eye rests on the words rather than being handed a menu. This is
// the knob for that distance — raise it to push them further down.
//
// Every row here is a row of height on a screen that is already 31 tall against a
// 24-row terminal, which is why it is 2 rather than 4.
const keysGap = 2

// finishedLines is what sits inside the ring on the closing screen.
//
// The last mystery is cleared away. Through the rosary the middle of the ring held
// the mystery being contemplated; at the end there is none, and leaving the fifth
// one there would read as still being on it. "Amen." closes the object instead.
func finishedLines() []string {
	// One line, so it lands on the ring's centre row. mysteryLines returns three
	// (name, gap, mystery) and the drawing centres the block, so a leading blank
	// here would push "Amen." a row below centre.
	return []string{"Amen."}
}

// trimBlankRows drops empty rows from the bottom of a block.
//
// The ring's canvas is as tall as the geometry reserved, and the rows below the
// crucifix are blank — harmless while the prayer sits beside the ring, but four
// dead rows between the crucifix and the closing text.
//
// A row counts as blank when it holds nothing but spaces — NOT when its width is
// zero. The canvas pads every row to the full ring width, so these rows are 41
// spaces each and a width test calls them full. That was the first attempt here,
// and it silently trimmed nothing.
func trimBlankRows(block string) string {
	rows := strings.Split(block, "\n")
	for len(rows) > 0 && blankRow(rows[len(rows)-1]) {
		rows = rows[:len(rows)-1]
	}
	return strings.Join(rows, "\n")
}

// blankRow reports whether a row shows nothing: only spaces, once the escape
// sequences that carry no width are removed.
func blankRow(row string) bool {
	inEsc := false
	for _, r := range row {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc:
			if r == 'm' {
				inEsc = false
			}
		case r != ' ':
			return false
		}
	}
	return true
}

// placeFinished centres the closing screen WITHOUT the key hint in the corner.
//
// The closing screen states its own keys, under the ring where the eye already is.
// The corner hint would repeat them, and it says "space next", which is no longer
// true — there is nothing next.
func placeFinished(m model, block string) string {
	if m.width == 0 || m.height == 0 {
		return block
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block)
}

// chooser is the first screen: pick the mysteries to contemplate.
func chooser(m model) string {
	rows := []string{
		titleStyle.Render("Which mysteries will you contemplate?"),
		"",
	}

	for i, set := range Sets() {
		line := "   " + set.Name
		if i == m.choice {
			// The marker is a character, not just a colour, so the selection is
			// visible with colour stripped.
			line = " ▸ " + set.Name
			rows = append(rows, mysteryStyle.Render(line))
			continue
		}
		rows = append(rows, dimStyle.Render(line))
	}

	rows = append(rows, "", dimStyle.Render("space to begin   x to finish"))
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// hint is the one line of chrome: the keys, and nothing else. It is drawn in a
// corner of the screen, not inside the rosary — see withHint.
//
// No counts. "Hail Mary 3 of 10" and "bead 11 of 68" turn praying into
// progress-watching: the eye goes to the number instead of the words, and the
// rosary becomes a task with a completion bar. The beads already show where you
// are, which is the right place for it — in the object, not in text.
func hint() string {
	return "space next   ← back   x finish"
}

// Bead colours. The ring is gold; the bead being prayed is reversed so it reads
// as "you are here" even with colour off (NO_COLOR), which a foreground change
// alone would not survive.
var (
	beadDim   = lipgloss.NewStyle().Foreground(gold).Faint(true)
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

// Styles for the mystery box and the chooser.
var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(gold).
			Padding(0, 3)

	// The set's name is secondary to the mystery itself, so it is quieter.
	setNameStyle = lipgloss.NewStyle().Foreground(gold).Faint(true)

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
			// name, so that is what has to be measured. Measuring the bead's name
			// would reserve the wrong width for any bead whose prayers are titled
			// differently from it — a junction bead is named "Our Father" but shows
			// "Glory Be" and "Fatima Prayer" too.
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
