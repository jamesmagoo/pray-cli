package rosary

import (
	"charm.land/lipgloss/v2"
)

// The chooser: the devotion's name under its crucifix, and the sets of
// mysteries to choose between.
//
// A phase of the same model as the rosary itself (see phase.go), not a separate
// program — the rosary is already built behind it, so beginning is instant and
// beginning again needs no reload.

// chooser is the first screen: the title, then the mysteries to pick from.
//
// Two blocks with different alignments, which is the whole of the layout here:
// the heading is CENTRED, because it is a title; the choices are LEFT-aligned,
// because they are a list and a list needs a common left edge to scan down. Both
// are padded to the same width before joining, since JoinVertical centres line by
// line rather than block by block.
func chooser(m model) string {
	var choices []string
	for i, set := range Sets() {
		line := "   " + set.Name
		if i == m.choice {
			// The marker is a character, not just a colour, so the selection is
			// visible with colour stripped.
			choices = append(choices, mysteryStyle.Render(" ▸ "+set.Name))
			continue
		}
		choices = append(choices, dimStyle.Render(line))
	}
	// When the highlighted set is traditionally prayed. Only the one being looked
	// at, not all four: it is a note about the choice in front of you rather than a
	// table to study, and it changes as the highlight moves.
	//
	// Centred under the list while the list stays left-aligned, so it reads as a
	// caption rather than as a fifth entry you could select.
	days := lipgloss.NewStyle().
		Width(lipgloss.Width(lipgloss.JoinVertical(lipgloss.Left, choices...))).
		Align(lipgloss.Center).
		Render(daysStyle.Render(Sets()[m.choice].Days))

	choices = append(choices, "", days, "", dimStyle.Render("space to begin   x to finish"))
	body := lipgloss.JoinVertical(lipgloss.Left, choices...)

	head := chooserHeading()

	w := max(lipgloss.Width(body), lipgloss.Width(head))
	centred := lipgloss.NewStyle().Width(w).Align(lipgloss.Center)

	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		centred.Render(head),
		"",
		centred.Render(dividerStyle.Render(chooserDivider)),
		"",
		lipgloss.NewStyle().Width(w).Render(body),
	))
}

// chooserDivider separates the title from the list of sets.
//
// Short and centred rather than spanning the box: a full-width rule cuts the
// screen in two, and the two halves are not equals — the title is a heading and
// the list is the thing you came to use. This marks the break without claiming
// they are separate panels.
//
// The tapered ends (╶ ╴) are what keep it from reading as a truncated full rule.
const chooserDivider = "╶────────────╴"

// chooserHeading is the crucifix over the rosary's full name.
//
// The cross is the SAME crossArt the rosary draws (see cross.go), not a copy: the
// opening screen then cannot drift away from the object it introduces, and
// restyling the crucifix restyles this too.
//
// "The Most Holy Rosary / of the Blessed Virgin Mary" is the devotion's proper
// name, both lines in the beads' blue — Our Lady's colour. The second line is
// unbolded so it reads as a subtitle rather than a second title, which is the only
// hierarchy a terminal really affords here.
func chooserHeading() string {
	rows := make([]string, 0, len(crossArt)+4)
	for _, line := range crossArt {
		rows = append(rows, crossBead.Render(line))
	}
	rows = append(rows,
		"",
		rosaryTitleStyle.Render(rosaryTitle),
		subtitleStyle.Render(rosarySubtitle),
	)
	return lipgloss.JoinVertical(lipgloss.Center, rows...)
}

// The devotion's proper name, shown on the chooser.
const (
	rosaryTitle    = "The Most Holy Rosary"
	rosarySubtitle = "of the Blessed Virgin Mary"
)
