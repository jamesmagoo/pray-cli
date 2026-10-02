package rosary

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Animation: the bead you move to flares bright, then settles back to gold.
//
// Bubble Tea has no animation API, and doesn't need one. An animation is just
// state that changes over time, so it works like every other state change:
//
//	1. a timer command sends a message after a delay
//	2. Update advances a counter and re-arms the timer
//	3. View draws whatever that counter currently means
//
// Nothing sleeps and nothing loops. The runtime keeps calling Update as the
// messages arrive, so the UI stays responsive to keys throughout — which is the
// whole reason to do it this way rather than with time.Sleep.

// frameRate is how often the animation advances. ~60fps is pointless in a
// terminal: the eye cannot resolve it in text, and every frame is a full
// redraw. 20fps (50ms) reads as smooth for a colour fade and costs a third as
// much work.
const frameRate = 50 * time.Millisecond

// glowFrames is how long the bead's flare lasts. 16 x 50ms = 800ms: long enough to
// dwell on the bead you have moved to rather than blinking and vanishing.
//
// Independent of fadeFrames — the two animations share a timer but not a duration.
const glowFrames = 16

// The current bead is drawn as a ring — a bead with a halo around it — rather
// than as a filled block. Reverse video would colour the whole cell, which reads
// as an ugly rectangle sitting on the string rather than as a lit bead.
//
// The flare swaps the GLYPH as well as the colour, which is what sells the glow:
// the halo blooms outward and settles. A colour change alone on a fixed glyph
// looks like a bulb dimming; changing the shape looks like light spreading.
//
// Brightest first, so the index matches glowRamp. The bloom is short — four
// glyphs — and then it holds on the halo for the rest of the flare, so lengthening
// glowFrames makes the bead dwell rather than making the shape-change slower.
//
// Built from glowFrames rather than written out, so the two can never drift apart
// when a duration is tuned.
var glowGlyphs = buildGlowGlyphs()

func buildGlowGlyphs() []string {
	bloom := []string{"✹", "✻", "◉"}
	out := make([]string, glowFrames)
	for i := range out {
		if i < len(bloom) && i < glowFrames-1 {
			out[i] = bloom[i]
			continue
		}
		out[i] = "◎" // settled
	}
	return out
}

// currentGlyph is the bead drawn at the cursor, at this point in the flare.
func currentGlyph(glow int) string {
	i := glowFrames - glow
	if i < 0 {
		i = 0
	}
	if i >= len(glowGlyphs) {
		i = len(glowGlyphs) - 1
	}
	return glowGlyphs[i]
}

// frameMsg says the animation should advance. It carries the time purely because
// that is the signature tea.Tick expects.
type frameMsg time.Time

// tick schedules the next animation frame.
//
// tea.Tick fires ONCE. It is not an interval: to keep animating, the handler for
// frameMsg must call tick again, which is what makes the animation stop by
// itself simply by not re-arming. This is the single most important thing to
// understand about timers here.
func tick() tea.Cmd {
	return tea.Tick(frameRate, func(t time.Time) tea.Msg {
		return frameMsg(t)
	})
}

// glowRamp is the colours the current bead passes through, brightest first.
//
// Built once at startup rather than per frame: blending is cheap but not free,
// and a fixed ramp means the animation is a lookup, not a computation.
//
// It ends on currentRest, NOT on gold. That matters: the ramp has to land exactly
// where the resting style sits, or the bead dips below its resting brightness and
// then jumps back up when the animation stops — a visible flicker on the last
// frame. The end of an animation must agree with the static state it hands over to.
var glowRamp = lipgloss.Blend1D(glowFrames, lipgloss.Color("#FFF8E0"), currentRest)

// glowStyle is the style for the current bead at a given point in the flare.
//
// glow counts DOWN from glowFrames to 0, so it is an index from the bright end
// of the ramp. At 0 the animation is over and the bead sits at resting gold.
func glowStyle(glow int) lipgloss.Style {
	i := glowFrames - glow
	if i < 0 {
		i = 0
	}
	if i >= len(glowRamp) {
		i = len(glowRamp) - 1
	}
	// Bold, but deliberately NOT Reverse: reversing fills the cell background and
	// is what made the cursor look like a block. The ring glyph plus bold is what
	// marks the position, and both survive NO_COLOR.
	return lipgloss.NewStyle().Foreground(glowRamp[i]).Bold(true)
}

// ── The prayer's fade-in ─────────────────────────────────────────────────────
//
// The text rises from near-invisible to full brightness on every step. It shares
// the glow counter with the bead's flare, so there is ONE timer driving both and
// no second animation system to keep in sync — they just read the same number
// through different curves.

// fadeFrames is how long the text takes to arrive. 20 x 50ms = 1s, slow enough to
// feel like the prayer appearing rather than a redraw.
//
// This and glowFrames are independent: tune either without touching the other.
const fadeFrames = 20

// textRamp is the colours the prayer passes through, dimmest first.
//
// Terminals have no transparency — lipgloss.Alpha sets an alpha channel that is
// dropped from the escape sequence entirely, so every alpha renders identically.
// A fade must therefore be an explicit blend between two opaque colours: from
// something close to the background up to the text's resting colour.
//
// It is built against a dark background, which is the common case for a terminal
// and the only one we can assume without querying the terminal (which needs the
// real tty and so cannot be done from a pure View).
var textRamp = lipgloss.Blend1D(fadeFrames, lipgloss.Color("#1C1C1C"), textRest)

// textRest is the brightest the fade reaches. The last frame is skipped in favour
// of the terminal's own foreground (see style_grid.go), so this only has to be
// close enough that the handover is invisible — a plausible "normal text" grey.
var textRest = lipgloss.Color("#D8D4CA")

// fadeStyle is the prayer's style at this point in the fade.
//
// fade counts DOWN from fadeFrames to 0, so it indexes from the DIM end: at
// fadeFrames the text is barely visible, at 0 it is fully lit. That is the
// opposite direction from glowStyle, which starts bright — the bead flares and
// decays, the text emerges.
func fadeStyle(fade int) lipgloss.Style {
	i := fadeFrames - fade
	if i < 0 {
		i = 0
	}
	if i >= len(textRamp) {
		i = len(textRamp) - 1
	}
	return lipgloss.NewStyle().Foreground(textRamp[i])
}

// ── The farewell's fade-out ──────────────────────────────────────────────────
//
// Leaving is the one animation that must finish before the program ends, and
// Bubble Tea has no "quit when this is done" command: tea.Quit tears the program
// down on the spot. So the fade-out is a phase (see phase.go) that animates and
// returns tea.Quit on its own last frame.
//
// This was verified against the runtime rather than assumed: View IS called for
// the frame whose Update returns tea.Quit, so the final frame is drawn before
// teardown. If it were not, the farewell would end one frame early and the last
// thing on screen would be a half-faded word.

// departFrames is how long the farewell takes. 24 x 50ms = 1.2s — slower than the
// prayer's fade-in, because this one is the last thing seen and hurrying it would
// undo the point of having it.
const departFrames = 24

// departRamp is the colours the farewell passes through: from the resting text
// colour down to black, so the words sink into the terminal rather than being
// switched off.
//
// Blended to #000000 for the same reason textRamp blends UP from #1C1C1C: a
// terminal has no transparency (lipgloss.Alpha is dropped from the escape
// sequence entirely), so a fade has to be an explicit blend between two opaque
// colours. Black is the assumption a dark terminal makes true; on a light terminal
// the words darken instead of lightening, which still reads as a fade, just not as
// a dissolve.
var departRamp = lipgloss.Blend1D(departFrames, textRest, lipgloss.Color("#000000"))

// departGoldRamp is the same fade for the cross, which starts from gold rather
// than from the text colour so it keeps its own character on the way out.
var departGoldRamp = lipgloss.Blend1D(departFrames, currentRest, lipgloss.Color("#000000"))

// departStyle is the farewell's text at this point in the fade-out.
//
// depart counts DOWN from departFrames to 0, and the ramp runs bright to dark, so
// the index is departFrames-depart: the opposite direction from fadeStyle, which
// starts dim and arrives. This one starts lit and leaves.
func departStyle(depart int) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(rampAt(departRamp, departFrames-depart))
}

// departCrossStyle is the cross's colour at this point in the fade-out.
func departCrossStyle(depart int) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(rampAt(departGoldRamp, departFrames-depart)).Bold(true)
}

// rampAt reads a ramp at index i, clamped to its ends.
//
// Every animation here needs this clamp, and writing it out at each use is how one
// of them ends up missing it and panicking on the frame where a duration was
// retuned but a ramp was not.
func rampAt(ramp []color.Color, i int) color.Color {
	if i < 0 {
		return ramp[0]
	}
	if i >= len(ramp) {
		return ramp[len(ramp)-1]
	}
	return ramp[i]
}
