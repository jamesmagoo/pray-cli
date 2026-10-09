package rosary

import (
	_ "embed"
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// The opening screen: Our Lady, and "Ave Maria, ora pro nobis", rising out of
// the dark before the chooser.
//
// It is a transition, not a menu. It plays by itself and hands over to the
// chooser when it is done; any key hands over at once, so it is never something
// to wait through on the way to praying.
//
// The art is a painting rendered to braille by scripts/opening-art.sh, as plain
// text with no colour in it. It is tinted gold here, as one block: a single
// colour reads the same in every terminal — 16 colours, 256 or truecolour — and
// with colour off it is still an outline, because braille draws with its dots
// rather than its colours. That is why it was chosen over a full-colour
// rendering, which needed truecolour to be recognisable at all.

//go:embed art/our-lady-large.txt
var ourLadyLarge string

//go:embed art/our-lady-small.txt
var ourLadySmall string

// ourLady is the art at each size, largest first. The opening uses the largest
// that fits the window, then none: a smaller terminal still gets the words.
var ourLady = []picture{newPicture(ourLadyLarge), newPicture(ourLadySmall)}

// artNudge is how many columns the picture is drawn left of centre, judged by
// eye: the painting is not symmetric, and centring its box sat it a touch right
// of the words. Only the art moves — the box keeps its width, so the words stay
// exactly where they were.
const artNudge = 2

// The words under the picture. Latin, as the prayer is: "Hail Mary, pray for us."
//
// The salutation is letter-spaced, which is the nearest a terminal comes to the
// small capitals an inscription would use; the petition is not, so the two read
// as a title and the line beneath it.
const (
	aveMaria    = "A V E   M A R I A"
	oraProNobis = "ora pro nobis"
)

// The opening's timeline, in frames of frameRate (50ms). The counter counts
// DOWN from openingFrames like every other animation here; openingAt turns it
// into the elapsed frame, which is easier to lay a timeline against.
//
//	0 ─ picture rises ─ 36
//	        24 ─ Ave Maria ─ 48
//	               36 ─ ora pro nobis ─ 60
//	                                     ... held ...
//	                                    104 ─ all fade ─ 120
//
// 6s in all: unhurried, a moment to rest on before praying. It plays on every
// rosary, so any key skips it.
const (
	openingFrames = 120
	openingOut    = 16 // the fade to the chooser at the end
)

// openingAt is how many frames of the opening have elapsed.
func openingAt(open int) int { return openingFrames - open }

// rise is how far a fade-in starting at frame from, lasting n frames, has got by
// frame at: 0 before it starts, 1 once it is done, eased in between.
//
// Ease-out (fast then slow): light arriving should settle, not snap on at the
// end, which is what a linear ramp looks like in a terminal.
func rise(at, from, n int) float64 {
	t := float64(at-from) / float64(n)
	switch {
	case t <= 0:
		return 0
	case t >= 1:
		return 1
	}
	return 1 - (1-t)*(1-t)
}

// openingLevel is the brightness of something that starts rising at frame from,
// taking n frames, and fades out with everything else at the end.
func openingLevel(open, from, n int) float64 {
	at := openingAt(open)
	out := 1 - rise(at, openingFrames-openingOut, openingOut)
	return rise(at, from, n) * out
}

// openingScreen is the screen itself: the largest picture that fits, and the words.
func openingScreen(m model) string {
	words := lipgloss.JoinVertical(lipgloss.Center,
		faded(aveMaria, gold, openingLevel(m.open, 24, 24), lipgloss.NewStyle().Bold(true)),
		"",
		faded(oraProNobis, beadBlue, openingLevel(m.open, 36, 24), lipgloss.NewStyle().Italic(true)),
	)

	block := words
	if pic, ok := m.pictureThatFits(words); ok {
		art := faded(pic.text, currentRest, openingLevel(m.open, 0, 36), lipgloss.NewStyle())
		w := max(pic.w, lipgloss.Width(words))
		centred := lipgloss.NewStyle().Width(w).Align(lipgloss.Center)
		block = lipgloss.JoinVertical(lipgloss.Center,
			centred.Render(art),
			"",
			centred.Render(words),
		)
	}

	if m.width == 0 || m.height == 0 {
		return block
	}
	// No "too small" notice: there is nothing here that needs room. The words
	// alone fit any window worth praying in, and below that they are clipped
	// rather than wrapped, since this screen is on its way out anyway.
	block = lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(block)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block)
}

// pictureThatFits is the largest picture that fits the window with the words
// under it, and whether there is one at all.
func (m model) pictureThatFits(words string) (picture, bool) {
	for _, pic := range ourLady {
		w := max(pic.w, lipgloss.Width(words))
		h := pic.h + 1 + lipgloss.Height(words)
		// Before the first resize the size is unknown, and the largest is assumed.
		if m.width == 0 || m.height == 0 || (w <= m.width && h <= m.height) {
			return pic, true
		}
	}
	return picture{}, false
}

// faded is text in colour c at brightness level (0 dark, 1 as given).
//
// At 0 it is blank rather than black: black is darker than most dark themes'
// background, so "not yet risen" text drawn in it shows as a shadow of itself.
// The shape is kept, so nothing moves when it appears.
//
// Text of several lines is styled line by line by lipgloss, so the picture fades
// as one block through this same function.
func faded(text string, c color.Color, level float64, style lipgloss.Style) string {
	if level <= 0 {
		lines := strings.Split(text, "\n")
		for i, l := range lines {
			lines[i] = strings.Repeat(" ", lipgloss.Width(l))
		}
		return strings.Join(lines, "\n")
	}
	r, g, b, _ := c.RGBA()
	return style.Foreground(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X",
		dim(uint8(r>>8), level), dim(uint8(g>>8), level), dim(uint8(b>>8), level)))).
		Render(text)
}

// dim scales one channel towards black.
//
// Towards BLACK, which is the same assumption every fade here makes: a terminal
// has no transparency, so a fade has to blend to a guessed background, and dark is
// the common case. See textRamp.
func dim(v uint8, level float64) uint8 { return uint8(float64(v)*level + 0.5) }

// ── The picture ──────────────────────────────────────────────────────────────

// picture is text art and its size in cells.
type picture struct {
	text string
	w, h int
}

// newPicture reads art as opening-art.sh writes it, padding every row to the
// widest so the picture is a true rectangle and centres as one — the script
// trims trailing spaces, and a ragged block would centre each row on its own.
//
// It then moves the art artNudge columns left inside that rectangle, taking blank
// columns from the left edge and adding them on the right. Only as many as are
// blank on every row: past that it would cut off the picture.
func newPicture(s string) picture {
	rows := strings.Split(strings.TrimRight(s, "\n"), "\n")
	w, blank := 0, artNudge
	for _, r := range rows {
		w = max(w, lipgloss.Width(r))
		if strings.TrimSpace(r) != "" {
			blank = min(blank, len(r)-len(strings.TrimLeft(r, " ")))
		}
	}
	for i, r := range rows {
		r = r[min(blank, len(r)-len(strings.TrimLeft(r, " "))):]
		rows[i] = r + strings.Repeat(" ", w-lipgloss.Width(r))
	}
	return picture{text: strings.Join(rows, "\n"), w: w, h: len(rows)}
}
