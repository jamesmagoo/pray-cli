package rosary

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// What to do when the terminal is smaller than the rosary.
//
// The honest answer is: say so. lipgloss.Place pads a block out to the window's
// width rather than clipping it, so a window narrower than the rosary does not
// crop the edges — it emits over-long rows and lets the terminal wrap every one,
// which turns the ring into a scrolling mess of fragments. Measured at 80x24: 34
// rows of 98 columns, all of them wrapped.
//
// So the layout cannot degrade gracefully, and pretending otherwise would mean
// drawing something unreadable and leaving the user to work out why. A sentence
// naming the size needed is more use than a corrupted rosary.

// tooSmall reports whether the window cannot hold the screen, and is false before
// the first WindowSizeMsg arrives — a zero size means "not yet known", not "tiny",
// and blanking the rosary on startup would be worse than briefly overflowing.
func tooSmall(m model, w, h int) bool {
	if m.width == 0 || m.height == 0 {
		return false
	}
	return m.width < w || m.height < h
}

// tooSmallNotice is what to show instead of the rosary.
//
// Deliberately plain text with no frame: a box is one more thing that has to fit,
// and this is shown precisely when things do not fit. It names the size needed and
// the size present, so the fix is obvious without counting anything.
func tooSmallNotice(m model, w, h int) string {
	// Two forms, because the notice has to fit windows the rosary does not — and a
	// notice that overflows is the same fault one level down. The short form is
	// what is left when even the sentence will not fit.
	long := []string{
		titleStyle.Render("This rosary needs more room."),
		"",
		dimStyle.Render(fmt.Sprintf("%d × %d is needed; this window is %d × %d.",
			w, h, m.width, m.height)),
		"",
		dimStyle.Render("Resize the terminal, or press x to finish."),
	}
	short := []string{
		titleStyle.Render("Needs more room."),
		dimStyle.Render(fmt.Sprintf("%d × %d", w, h)),
		dimStyle.Render("x to finish"),
	}

	block := lipgloss.JoinVertical(lipgloss.Center, long...)
	if lipgloss.Width(block) > m.width || lipgloss.Height(block) > m.height {
		block = lipgloss.JoinVertical(lipgloss.Center, short...)
	}

	// Truncate as a last resort. At a few columns wide nothing readable fits, and
	// emitting rows wider than the terminal is worse than emitting fragments.
	block = lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(block)

	// Centre only if there is room to centre in: Place PADS to the window size,
	// which is precisely the behaviour that misbehaves when content exceeds it.
	if m.width >= lipgloss.Width(block) && m.height >= lipgloss.Height(block) {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block)
	}
	return block
}
