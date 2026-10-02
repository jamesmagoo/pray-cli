package rosary

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// An animation is just state plus a timer, so it can be tested without waiting:
// feed Update the frameMsg directly instead of letting tea.Tick deliver it. The
// production path sleeps; the test does not have to.

func TestMovingABeadStartsTheGlow(t *testing.T) {
	m := praying(t)

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'l'})
	got := next.(model)

	if got.glow != glowFrames {
		t.Errorf("glow = %d, want %d", got.glow, glowFrames)
	}
	if cmd == nil {
		t.Fatal("moving a bead returned no command, so no frame was scheduled")
	}
}

// Both animations must stop on their own. They do so by not re-arming the timer,
// so the test is: step through every frame and check the last one returns nil.
//
// The two counters are independent and have different lengths, so the timer must
// keep ticking while EITHER is running and stop only when both are done.
func TestAnimationsFadeAndStop(t *testing.T) {
	m := praying(t)
	next, _ := m.Update(tea.KeyPressMsg{Code: 'l'})
	cur := next.(model)

	longest := max(glowFrames, fadeFrames)
	for i := 1; i <= longest; i++ {
		n, cmd := cur.Update(frameMsg{})
		cur = n.(model)

		if want := max(glowFrames-i, 0); cur.glow != want {
			t.Fatalf("after %d frames: glow = %d, want %d", i, cur.glow, want)
		}
		if want := max(fadeFrames-i, 0); cur.fade != want {
			t.Fatalf("after %d frames: fade = %d, want %d", i, cur.fade, want)
		}

		last := i == longest
		if last && cmd != nil {
			t.Fatal("the final frame re-armed the timer; the animation would never stop")
		}
		if !last && cmd == nil {
			t.Fatalf("frame %d did not schedule the next one", i)
		}
	}
}

// A frame arriving when nothing is animating must not drive glow negative, which
// would index outside the ramp.
func TestStrayFrameIsHarmless(t *testing.T) {
	m := praying(t)
	next, cmd := m.Update(frameMsg{})

	if got := next.(model).glow; got != 0 {
		t.Errorf("glow = %d, want 0", got)
	}
	if got := next.(model).fade; got != 0 {
		t.Errorf("fade = %d, want 0", got)
	}
	if cmd != nil {
		t.Error("a stray frame started an animation")
	}
}

// Pressing keys mid-animation must not stack overlapping animations: the counter
// resets and the in-flight timer is simply absorbed.
func TestRapidMovesResetRatherThanStack(t *testing.T) {
	m := praying(t)

	cur := tea.Model(m)
	for i := 0; i < 3; i++ {
		cur, _ = cur.Update(tea.KeyPressMsg{Code: 'l'})
		cur, _ = cur.Update(frameMsg{}) // a frame lands between presses
	}

	if got := cur.(model).glow; got != glowFrames-1 {
		t.Errorf("glow = %d, want %d: the counter should reset, not accumulate", got, glowFrames-1)
	}
	if got := cur.(model).fade; got != fadeFrames-1 {
		t.Errorf("fade = %d, want %d: the counter should reset, not accumulate", got, fadeFrames-1)
	}
}

// glowStyle is indexed by a counter, so it must be safe at and beyond both ends.
func TestGlowStyleIsSafeAtTheEdges(t *testing.T) {
	for _, glow := range []int{-1, 0, 1, glowFrames, glowFrames + 5} {
		if got := glowStyle(glow).Render("○"); got == "" {
			t.Errorf("glow=%d rendered nothing", glow)
		}
	}
}

// The flare must hand over seamlessly to the resting style. If the ramp's last
// colour differs from currentRest, the bead jumps brightness on the final frame
// — a flicker that is easy to introduce and easy to miss by eye.
func TestGlowLandsOnTheRestingColour(t *testing.T) {
	last := glowRamp[len(glowRamp)-1]

	lr, lg, lb, _ := last.RGBA()
	rr, rg, rb, _ := currentRest.RGBA()

	if lr != rr || lg != rg || lb != rb {
		t.Errorf("ramp ends at #%02X%02X%02X but the bead rests at #%02X%02X%02X: the last frame would flicker",
			lr>>8, lg>>8, lb>>8, rr>>8, rg>>8, rb>>8)
	}
}

// The glyph sequence and the colour ramp are indexed by the same counter, so they
// must be the same length or the two drift apart mid-flare.
func TestGlyphAndColourRampAgree(t *testing.T) {
	if len(glowGlyphs) != len(glowRamp) {
		t.Fatalf("glowGlyphs has %d entries, glowRamp has %d", len(glowGlyphs), len(glowRamp))
	}
	if len(glowGlyphs) != glowFrames {
		t.Fatalf("glowGlyphs has %d entries but the flare is %d frames", len(glowGlyphs), glowFrames)
	}
}

// The halo must settle to a steady glyph rather than still be changing shape when
// the animation ends, or the bead appears to twitch after it has stopped.
func TestGlyphSettles(t *testing.T) {
	if got, want := currentGlyph(0), glowGlyphs[len(glowGlyphs)-1]; got != want {
		t.Errorf("resting glyph is %q, want %q", got, want)
	}
	if currentGlyph(1) != currentGlyph(0) {
		t.Errorf("the glyph is still changing on the last frame: %q then %q", currentGlyph(1), currentGlyph(0))
	}
}

// Space is the primary key for praying, so it gets its own test. In Bubble Tea v2
// the space bar's String() is "space", not " " — matching the wrong one fails
// silently, the key simply does nothing, and nothing in the build catches it.
func TestSpaceAdvances(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{
		{Code: ' '}, // how a terminal reports the space bar
		{Code: tea.KeyEnter},
		{Code: tea.KeyRight},
	} {
		m := praying(t)
		next, cmd := m.Update(key)
		got := next.(model)

		if got.cursor == m.cursor && got.say == m.say {
			t.Errorf("%q did not advance", key.String())
		}
		if cmd == nil {
			t.Errorf("%q advanced but scheduled no glow", key.String())
		}
	}
}

// The crucifix must never be replaced by the cursor's halo: the cross vanishing
// exactly when you pray the Sign of the Cross on it is the worst moment for it to
// disappear.
func TestCrossKeepsItsShapeWhenSelected(t *testing.T) {
	m := praying(t)
	if m.beads[0].Kind != Cross {
		t.Skip("the sequence does not open on the cross")
	}

	m.cursor, m.say = 0, 0
	out := drawRosary(m.ring, m.beads, 0, glowFrames, 0, frameLines(m.bead(), m.words(), hint()))

	if !strings.Contains(out, Cross.Glyph()) {
		t.Error("the crucifix disappeared while it was the current bead")
	}
}

// The two animations are independent, so each must be tunable without the other
// changing. This guards the shape of that: separate counters, separate lengths.
func TestGlowAndFadeAreIndependent(t *testing.T) {
	m := praying(t)
	next, _ := m.Update(tea.KeyPressMsg{Code: 'l'})
	got := next.(model)

	if got.glow != glowFrames {
		t.Errorf("glow = %d, want %d", got.glow, glowFrames)
	}
	if got.fade != fadeFrames {
		t.Errorf("fade = %d, want %d", got.fade, fadeFrames)
	}

	// The glyph sequence is derived from glowFrames, so changing the duration
	// cannot leave it the wrong length.
	if len(glowGlyphs) != glowFrames || len(glowRamp) != glowFrames {
		t.Errorf("glyphs=%d ramp=%d, both should be glowFrames=%d",
			len(glowGlyphs), len(glowRamp), glowFrames)
	}
	if len(textRamp) != fadeFrames {
		t.Errorf("textRamp=%d, want fadeFrames=%d", len(textRamp), fadeFrames)
	}
}

// fadeStyle is indexed by a counter, so it must be safe past both ends.
func TestFadeStyleIsSafeAtTheEdges(t *testing.T) {
	for _, fade := range []int{-1, 0, 1, fadeFrames, fadeFrames + 5} {
		if got := fadeStyle(fade).Render("x"); got == "" {
			t.Errorf("fade=%d rendered nothing", fade)
		}
	}
}

// The fade must brighten, not darken: it is the prayer arriving.
func TestFadeGetsBrighter(t *testing.T) {
	prev := -1
	for fade := fadeFrames; fade >= 1; fade-- {
		r, g, b, _ := textRamp[fadeFrames-fade].RGBA()
		lum := int(r>>8) + int(g>>8) + int(b>>8)
		if lum < prev {
			t.Fatalf("fade=%d is darker than the frame before it", fade)
		}
		prev = lum
	}
}
