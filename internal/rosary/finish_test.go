package rosary

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// atEnd returns a model standing on the very last prayer of the rosary.
func atEnd(t *testing.T) model {
	t.Helper()
	m := praying(t)
	m.cursor = len(m.beads) - 1
	m.say = len(m.bead().Says) - 1
	return m
}

// press sends one key and returns the model it produced.
func press(t *testing.T, m model, key string) model {
	t.Helper()
	next, _ := m.Update(tea.KeyPressMsg{Code: keyCode(key)})
	got, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T, not a model", next)
	}
	return got
}

// plain strips the colour escapes from a rendered screen.
//
// Necessary for anything drawn on the GRID. styleGrid wraps each rune in its own
// escape sequence, so a line reading "The Crucifixion" on screen is really
// "\x1b[..mT\x1b[m\x1b[..mh\x1b[m..." and strings.Contains can never find the
// plain words in it. Text in the prayer panel IS findable, because that is styled
// a whole line at a time — which is the trap: an assertion about grid text passes
// vacuously while the identical assertion about panel text works.
func plain(screen string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range screen {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc:
			if r == 'm' {
				inEsc = false
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// keyCode maps the few key names these tests need onto v2 key codes.
func keyCode(key string) rune {
	switch key {
	case "space":
		return ' '
	case "left":
		return tea.KeyLeft
	case "x":
		return 'x'
	}
	panic("rosary: unknown test key " + key)
}

// Space on the last prayer finishes the rosary rather than doing nothing.
//
// Before this, next() returned false at the end and the key was swallowed: the
// user was left on the Hail Holy Queen with no sign they had finished.
func TestSpaceOnTheLastPrayerFinishes(t *testing.T) {
	m := atEnd(t)
	if m.phase != atPrayer {
		t.Fatal("did not start at prayer")
	}

	got := press(t, m, "space")

	if got.phase != finished {
		t.Errorf("phase is %v after the last prayer, want finished", got.phase)
	}
	// The cursor must NOT move past the end.
	if got.cursor != len(got.beads)-1 {
		t.Errorf("cursor moved to %d, want it to stay on the last bead", got.cursor)
	}
}

// Finishing must not happen early: space anywhere before the last prayer prays on.
func TestSpaceBeforeTheEndDoesNotFinish(t *testing.T) {
	m := praying(t)

	// The last bead, but NOT its last prayer: the Hail Holy Queen bead carries
	// three, so this is the case a naive "last bead" check would get wrong.
	m.cursor = len(m.beads) - 1
	m.say = 0

	got := press(t, m, "space")
	if got.phase != atPrayer {
		t.Fatalf("finished on prayer 1 of %d on the last bead", len(m.bead().Says))
	}
	if got.say != 1 {
		t.Errorf("say is %d, want 1", got.say)
	}
}

// The closing screen says the rosary is done, names the set, and offers both ways on.
func TestFinishScreenOffersAgainAndFinish(t *testing.T) {
	m := atEnd(t)
	m = press(t, m, "space")

	screen := plain(render(m))
	for _, want := range []string{"The rosary is prayed.", Sorrowful.Name, "pray again", "finish"} {
		if !strings.Contains(screen, want) {
			t.Errorf("the closing screen does not mention %q", want)
		}
	}

	// The ring closes on "Amen." — drawn on the grid, so this only works on the
	// stripped screen.
	if !strings.Contains(screen, "Amen.") {
		t.Error("the closing screen does not show Amen. inside the ring")
	}

	// No mystery is left on screen: the fifth was being contemplated a moment ago
	// and leaving it there reads as still being on it.
	if strings.Contains(screen, Sorrowful.Mysteries[4]) {
		t.Errorf("the closing screen still shows the last mystery")
	}
}

// No bead is lit on the closing screen. That absence IS the transition: through
// the whole rosary exactly one bead was current.
func TestNoBeadIsCurrentWhenFinished(t *testing.T) {
	m := atEnd(t)
	m = press(t, m, "space")

	screen := plain(render(m))

	// The halo glyphs mark the current bead. None may appear.
	for _, glyph := range glowGlyphs {
		if strings.Contains(screen, glyph) {
			t.Errorf("the closing screen still lights a bead (%q)", glyph)
		}
	}

	// The beads themselves are still drawn — the ring is at rest, not gone.
	if !strings.Contains(screen, smallBead) || !strings.Contains(screen, bigBead) {
		t.Error("the closing screen does not draw the rosary")
	}
	if !strings.Contains(screen, Cross.Glyph()) {
		t.Error("the closing screen does not draw the crucifix")
	}
}

// Space on the closing screen goes back to the CHOOSER, so another set can be
// picked, with the rosary reset behind it.
func TestSpaceOnTheFinishScreenBeginsAgain(t *testing.T) {
	m := atEnd(t)
	m = press(t, m, "space") // finish
	got := press(t, m, "space")

	if got.phase != choosing {
		t.Fatalf("phase is %v, want the chooser", got.phase)
	}
	if got.cursor != 0 || got.say != 0 {
		t.Errorf("began again at bead %d prayer %d, want the very start", got.cursor, got.say)
	}
	if !strings.Contains(plain(render(got)), Sorrowful.Name) {
		t.Error("the chooser is not shown again")
	}
}

// Beginning again must clear the mystery too. announced() looks BACK from the
// cursor, so a stale cursor would carry the fifth mystery into the new rosary.
func TestBeginningAgainClearsTheMystery(t *testing.T) {
	m := atEnd(t)
	if n := m.announced(); n != 5 {
		t.Fatalf("the last bead has mystery %d in force, want 5", n)
	}

	m = press(t, m, "space") // finish
	m = press(t, m, "space") // back to the chooser
	got := press(t, m, "space")

	if got.phase != atPrayer {
		t.Fatalf("phase is %v, want at prayer", got.phase)
	}
	if n := got.announced(); n != 0 {
		t.Errorf("the new rosary opens with mystery %d in force, want none", n)
	}
	if _, ok := got.mystery(); ok {
		t.Error("a mystery is contemplated on the opening prayers")
	}
}

// Space, pressed from the very start, carries the user through the whole rosary
// and over the finish — and then through a second one.
//
// This is the only test that drives the real loop end to end rather than placing
// the cursor where it wants it. It is what proves there is no bead the sequence
// cannot get past: a bead with no prayers, or an addAt pointing at the wrong
// place, would stall here while every targeted test still passed.
func TestSpaceAlonePraysTheWholeRosaryTwice(t *testing.T) {
	m, err := newModel("en", Sorrowful)
	if err != nil {
		t.Fatal(err)
	}
	m.width, m.height = 100, 40

	// A bound well above the real count (74), so a stall fails the test instead of
	// hanging it.
	const bound = 500

	var cur tea.Model = m
	presses := 0
	for presses < bound && cur.(model).phase != finished {
		next, _ := cur.Update(tea.KeyPressMsg{Code: ' '})
		cur, presses = next, presses+1
	}

	got := cur.(model)
	if got.phase != finished {
		t.Fatalf("still not finished after %d presses: phase %v, bead %d of %d",
			presses, got.phase, got.cursor, len(got.beads))
	}
	if !strings.Contains(plain(render(got)), "The rosary is prayed.") {
		t.Error("the closing screen is not shown")
	}

	// Again: space to the chooser, space to begin.
	got = press(t, got, "space")
	got = press(t, got, "space")
	if got.phase != atPrayer || got.cursor != 0 || got.say != 0 {
		t.Fatalf("the second rosary began at phase %v bead %d prayer %d",
			got.phase, got.cursor, got.say)
	}
	if _, ok := got.mystery(); ok {
		t.Error("the second rosary opens with a mystery already in force")
	}
}

// begin() resets the rosary itself, not just the phase.
//
// Tested directly rather than only through the chooser: the chooser happens to
// reset the cursor too, so going through it hides whether begin() does its own
// job. It must, because it is the single definition of "start the rosary" — and
// anything that starts one without the chooser (a flag choosing the set, say)
// would otherwise inherit the last rosary's position and mystery.
func TestBeginResetsTheRosary(t *testing.T) {
	m := atEnd(t)

	cmd := m.begin(Sorrowful)

	if cmd == nil {
		t.Error("begin did not start the animations")
	}
	if m.phase != atPrayer {
		t.Errorf("phase is %v, want at prayer", m.phase)
	}
	if m.cursor != 0 || m.say != 0 {
		t.Errorf("begin left the cursor at bead %d prayer %d, want the very start",
			m.cursor, m.say)
	}
	if n := m.announced(); n != 0 {
		t.Errorf("begin left mystery %d in force, want none", n)
	}
}

// x finishes from the closing screen, as it does everywhere else.
func TestXQuitsFromTheFinishScreen(t *testing.T) {
	m := atEnd(t)
	m = press(t, m, "space")

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'x'})
	if cmd == nil {
		t.Fatal("x did not quit from the closing screen")
	}
}

// Back still works at the end: finishing is not a wall.
func TestBackFromTheFinishScreenReturnsToTheRosary(t *testing.T) {
	m := atEnd(t)
	last := m.cursor
	m = press(t, m, "space")

	got := press(t, m, "left")

	if got.phase != atPrayer {
		t.Fatalf("phase is %v, want at prayer", got.phase)
	}
	if got.cursor != last {
		t.Errorf("landed on bead %d, want the last bead %d", got.cursor, last)
	}
	if !strings.Contains(plain(render(got)), "The Collect") {
		t.Error("stepping back does not show the closing prayer again")
	}
}

// The closing screen animates in like every other step, so arriving reads as one
// more move rather than a different program taking over.
func TestFinishingStartsTheAnimations(t *testing.T) {
	m := atEnd(t)

	next, cmd := m.Update(tea.KeyPressMsg{Code: ' '})
	got := next.(model)

	if got.fade != fadeFrames {
		t.Errorf("fade is %d on arrival, want %d", got.fade, fadeFrames)
	}
	if cmd == nil {
		t.Error("no timer was started for the closing screen")
	}
}

// The closing text sits directly under the crucifix, with no dead rows between.
//
// The ring's canvas reserves rows below the crucifix that are blank; they cost
// nothing while the prayer sits beside the ring, but on the closing screen they
// opened a four-row gap between the crucifix and the text. trimBlankRows removes
// them, and this pins it: the screen is already 31 rows against a 24-row terminal,
// so every reclaimed row matters.
func TestNoGapBetweenTheRosaryAndTheClosingText(t *testing.T) {
	m := atEnd(t)
	m = press(t, m, "space")

	rows := strings.Split(plain(completion(m)), "\n")

	// Measure from the crucifix's FOOT, not its crossbar. The cross is a block
	// several rows tall (see cross.go), so Cross.Glyph() finds its middle; the gap
	// that matters is below the last row it occupies.
	text := -1
	for i, r := range rows {
		if strings.Contains(r, "The rosary is prayed.") {
			text = i
			break
		}
	}
	if text < 0 {
		t.Fatal("the closing text is not on the screen")
	}

	// Walk back from the text to the last row the rosary actually occupies.
	foot := text - 1
	for foot >= 0 && blankRow(rows[foot]) {
		foot--
	}
	if foot < 0 {
		t.Fatal("no rosary above the closing text")
	}

	// One blank row between them: a deliberate breath, not four dead rows.
	if gap := text - foot - 1; gap != 1 {
		t.Errorf("%d rows between the rosary and the closing text, want 1", gap)
	}
}

// The keys sit well below the closing words, not tucked against them.
//
// They are chrome: available, but set apart so the eye rests on what was prayed
// rather than being handed a menu. keysGap is the knob, and this pins the rows it
// actually produces — a block of n newlines is n+1 lines, so the repeat that makes
// the gap is one short of it, and off by one here is a visible row.
func TestKeysSitBelowTheClosingWords(t *testing.T) {
	m := atEnd(t)
	m = press(t, m, "space")

	rows := strings.Split(plain(completion(m)), "\n")

	words := -1
	keys := -1
	for i, r := range rows {
		if strings.Contains(r, Sorrowful.Name) {
			words = i
		}
		if strings.Contains(r, "pray again") {
			keys = i
		}
	}
	if words < 0 || keys < 0 {
		t.Fatalf("set name at row %d, keys at row %d; expected both", words, keys)
	}
	if keys < words {
		t.Fatal("the keys are above the closing words")
	}

	if gap := keys - words - 1; gap != keysGap {
		t.Errorf("%d blank rows between the closing words and the keys, want keysGap (%d)",
			gap, keysGap)
	}

	// And they really are blank — not a row of something faint.
	for i := words + 1; i < keys; i++ {
		if !blankRow(rows[i]) {
			t.Errorf("row %d between the words and the keys is not blank: %q", i, rows[i])
		}
	}
}

// The corner hint is not drawn on the closing screen: it would say "space next"
// when there is nothing next, and repeat keys the screen already states.
func TestFinishScreenHasNoCornerHint(t *testing.T) {
	m := atEnd(t)
	m.width, m.height = 100, 40
	m = press(t, m, "space")

	if strings.Contains(plain(render(m)), "next") {
		t.Error("the closing screen shows the praying hint")
	}
}

// ── Leaving ──────────────────────────────────────────────────────────────────

// x starts the farewell rather than quitting on the spot.
func TestXStartsTheFarewell(t *testing.T) {
	m := praying(t)

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'x'})
	got := next.(model)

	if got.phase != departing {
		t.Fatalf("phase is %v after x, want departing", got.phase)
	}
	if got.depart != departFrames {
		t.Errorf("depart is %d, want %d", got.depart, departFrames)
	}
	if cmd == nil {
		t.Fatal("no timer was started for the farewell")
	}
	// Crucially NOT Quit: quitting here would tear the program down before the
	// fade had a single frame.
	if _, quit := cmd().(tea.QuitMsg); quit {
		t.Error("x quit immediately instead of fading out")
	}
}

// x reaches the farewell from every screen, so no screen can quit abruptly.
func TestXFadesOutFromEveryScreen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(m *model)
	}{
		{"chooser", func(m *model) { m.phase = choosing }},
		{"praying", func(m *model) { m.phase = atPrayer }},
		{"finished", func(m *model) { m.phase = finished }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := praying(t)
			tc.setup(&m)

			next, cmd := m.Update(tea.KeyPressMsg{Code: 'x'})
			got := next.(model)

			if got.phase != departing {
				t.Errorf("x from %s left phase %v, want departing", tc.name, got.phase)
			}
			if cmd == nil {
				t.Fatalf("x from %s started no timer", tc.name)
			}
			if _, quit := cmd().(tea.QuitMsg); quit {
				t.Errorf("x from %s quit without fading", tc.name)
			}
		})
	}
}

// The farewell runs exactly departFrames and then quits itself.
//
// This is the one animation whose END does something rather than merely stopping,
// so both halves matter: it must not quit early (the fade would be cut off) and it
// must not fail to quit (the program would hang on a black screen).
func TestTheFarewellRunsItsCourseThenQuits(t *testing.T) {
	m := praying(t)
	next, _ := m.Update(tea.KeyPressMsg{Code: 'x'})

	var cur tea.Model = next
	for i := 1; i <= departFrames*2; i++ {
		n, cmd := cur.Update(frameMsg{})
		cur = n
		if cmd == nil {
			t.Fatalf("frame %d returned no command; the farewell stalled", i)
		}
		if _, quit := cmd().(tea.QuitMsg); quit {
			if i != departFrames {
				t.Errorf("quit on frame %d, want frame %d", i, departFrames)
			}
			return
		}
	}
	t.Fatalf("the farewell never quit after %d frames", departFrames*2)
}

// esc and ctrl+c skip the farewell: an escape hatch that takes 1.2s is not one.
func TestEscapeHatchesSkipTheFarewell(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{"esc", tea.KeyPressMsg{Code: tea.KeyEscape}},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, phase := range []phase{choosing, atPrayer, finished, departing} {
				m := praying(t)
				m.phase = phase

				_, cmd := m.Update(tc.msg)
				if cmd == nil {
					t.Fatalf("%s in phase %v did nothing", tc.name, phase)
				}
				if _, quit := cmd().(tea.QuitMsg); !quit {
					t.Errorf("%s in phase %v did not quit immediately", tc.name, phase)
				}
			}
		})
	}
}

// The farewell cannot be hurried or cancelled by a stray key.
func TestTheFarewellIgnoresOtherKeys(t *testing.T) {
	m := praying(t)
	next, _ := m.Update(tea.KeyPressMsg{Code: 'x'})
	dep := next.(model)

	for _, code := range []rune{' ', 'l', 'h', 'z'} {
		got, cmd := dep.Update(tea.KeyPressMsg{Code: code})
		if cmd != nil {
			t.Errorf("%q did something during the farewell", string(code))
		}
		if g := got.(model); g.phase != departing || g.depart != dep.depart {
			t.Errorf("%q disturbed the farewell: phase %v depart %d",
				string(code), g.phase, g.depart)
		}
	}
}

// The farewell screen is the cross and the closing words, and nothing else.
func TestTheFarewellShowsOnlyTheCrossAndTheWords(t *testing.T) {
	m := praying(t)
	next, _ := m.Update(tea.KeyPressMsg{Code: 'x'})
	got := next.(model)

	screen := plain(render(got))
	if !strings.Contains(screen, farewellWords) {
		t.Errorf("the farewell does not say %q", farewellWords)
	}
	// The single-glyph cross, not the rosary's line-drawn one: the lines exist to
	// hold the crucifix true to the pendant's column, and the farewell has no
	// column. See farewell().
	if !strings.Contains(screen, malteseCross) {
		t.Errorf("the farewell does not show the cross %q", malteseCross)
	}
	if strings.Contains(screen, Cross.Glyph()) {
		t.Error("the farewell draws the rosary's line-built crucifix; it should be the single glyph")
	}
	// Everything else is put down: no beads, no prayer, no keys.
	if strings.Contains(screen, smallBead) || strings.Contains(screen, bigBead) {
		t.Error("the farewell still draws the rosary's beads")
	}
	if strings.Contains(screen, "pray again") || strings.Contains(screen, "finish") {
		t.Error("the farewell still offers keys")
	}
}

// The fade reaches black on its last visible frame, so the words sink into the
// terminal rather than being switched off mid-brightness.
func TestTheFarewellFadesAllTheWayOut(t *testing.T) {
	for _, ramp := range [][]color.Color{departRamp, departGoldRamp} {
		r, g, b, _ := ramp[len(ramp)-1].RGBA()
		if r>>8 > 1 || g>>8 > 1 || b>>8 > 1 {
			t.Errorf("the fade ends at rgb(%d,%d,%d), want black", r>>8, g>>8, b>>8)
		}
	}
	// And it starts lit, or there would be nothing to fade.
	r, _, _, _ := departRamp[0].RGBA()
	if r>>8 < 100 {
		t.Errorf("the fade starts at red %d; too dark to read", r>>8)
	}
}

// Every frame of the farewell is the same size, or the screen would jump as it
// faded — the kind of thing that reads as a glitch rather than a design.
func TestTheFarewellDoesNotResizeAsItFades(t *testing.T) {
	m := praying(t)
	m.phase = departing

	m.depart = departFrames
	wantW, wantH := lipgloss.Width(farewell(m)), lipgloss.Height(farewell(m))

	for d := departFrames; d >= 0; d-- {
		m.depart = d
		if w, h := lipgloss.Width(farewell(m)), lipgloss.Height(farewell(m)); w != wantW || h != wantH {
			t.Fatalf("frame at depart=%d is %dx%d, want %dx%d", d, w, h, wantW, wantH)
		}
	}
}

// rampAt clamps rather than panicking, which is what lets a duration be retuned
// without every ramp having to be resized in the same commit.
func TestRampAtClamps(t *testing.T) {
	for _, i := range []int{-99, -1, 0, departFrames - 1, departFrames, 999} {
		if rampAt(departRamp, i) == nil {
			t.Errorf("rampAt(%d) returned nothing", i)
		}
	}
	if rampAt(departRamp, -1) != departRamp[0] {
		t.Error("a negative index does not clamp to the first colour")
	}
	if rampAt(departRamp, 999) != departRamp[len(departRamp)-1] {
		t.Error("a large index does not clamp to the last colour")
	}
}
