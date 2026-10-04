package rosary

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// The two screens that end the rosary.
//
// completion is reached by praying the last bead: the same ring drawn with no bead
// current, which is the whole transition — through the rosary exactly one bead was
// lit, so a ring with nothing lit reads as "no longer in progress".
//
// farewell is reached by pressing x: a cross and a last word fading to black,
// after which the program quits itself. It is a phase rather than an immediate
// quit because tea.Quit tears the program down on the spot and there is no "quit
// when this animation finishes" command. See phase.go.

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
	w, h := lipgloss.Width(block), lipgloss.Height(block)
	if tooSmall(m, w, h) {
		return tooSmallNotice(m, w, h)
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block)
}
