package rosary

import (
	"charm.land/lipgloss/v2"

	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Update and View are ordinary functions of the model, so they can be tested
// without a terminal: build a state, send it a message, look at what comes back.
// That is the practical payoff of keeping all state in one value.

// esc and ctrl+c quit immediately, with no farewell.
//
// They are escape hatches: someone reaching for either wants out of a full-screen
// program NOW. x is the considered exit and is tested separately — it starts the
// farewell instead of quitting, which is the whole distinction.
func TestEscapeHatchesQuitImmediately(t *testing.T) {
	m := praying(t)

	for _, key := range []string{"esc", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			// tea.Key is what the runtime builds from a real key press; here we
			// build one directly. Code is the rune or special key pressed.
			var msg tea.KeyPressMsg
			switch key {
			case "esc":
				msg = tea.KeyPressMsg{Code: tea.KeyEscape}
			case "ctrl+c":
				msg = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
			default:
				msg = tea.KeyPressMsg{Code: rune(key[0])}
			}

			if msg.String() != key {
				t.Fatalf("built the wrong key: got %q, want %q", msg.String(), key)
			}

			_, cmd := m.Update(msg)
			if cmd == nil {
				t.Fatalf("%q did not finish", key)
			}
			// A Cmd is a function returning a Msg. Running it tells us which
			// command it was: tea.Quit's message is tea.QuitMsg.
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("%q returned a command, but not Quit", key)
			}
		})
	}
}

func TestOtherKeysDoNotFinish(t *testing.T) {
	m := praying(t)
	// 'z' is bound to nothing. ('x' finishes, so it cannot be used here.)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'z'})
	if cmd != nil {
		t.Fatal("an unhandled key returned a command; it should leave the state alone")
	}
}

func TestWindowSizeIsStored(t *testing.T) {
	m := praying(t)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	got := next.(model)

	if got.width != 80 || got.height != 24 {
		t.Fatalf("size not stored: got %dx%d, want 80x24", got.width, got.height)
	}
	// The original is untouched: Update returns a new value rather than
	// mutating, which is what makes each state independent.
	if m.width != 0 {
		t.Fatal("Update mutated the model it was given")
	}
}

func TestViewShowsThePrayer(t *testing.T) {
	m := praying(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// View returns a tea.View; its Content is the text on screen.
	out := next.(model).View().Content

	// The rosary opens on the Sign of the Cross. The keys hint is checked
	// separately (TestHintIsInTheCornerNotTheRing): its words are individually
	// styled now, so a literal match on the whole phrase would not survive.
	for _, want := range []string{"Sign of the Cross", "Amen.", "finish"} {
		if !strings.Contains(out, want) {
			t.Errorf("the view is missing %q", want)
		}
	}
}

// Before the first WindowSizeMsg we don't know the terminal size. The view must
// still be drawable rather than collapsing to nothing.
func TestViewBeforeFirstResize(t *testing.T) {
	m := praying(t)
	if out := m.View().Content; !strings.Contains(out, "Sign of the Cross") {
		t.Fatalf("the first frame lost the prayer: %q", out)
	}
}

// The line breaks of the .md file are the line breaks on screen: the renderer
// never joins or reflows. This guards the principle, not the styling.
func TestLineBreaksAreNotReflowed(t *testing.T) {
	m := praying(t)

	// Every line of every prayer must appear intact among the lines drawn:
	// nothing is reflowed or shortened.
	for i, b := range m.beads {
		// j matters: a bead can hold several prayers, so the cursor must point at
		// the one being checked. Leaving m.say at 0 would compare every prayer
		// against the first one's panel.
		for j, w := range b.Says {
			// The prayer is drawn in its own panel beside the ring, so that is
			// where its lines must appear, intact.
			m.cursor, m.say = i, j
			panel := stripEscapes(prayerPanel(m))
			for _, want := range w.Lines {
				if want == "" {
					continue
				}
				if !strings.Contains(panel, want) {
					t.Errorf("bead %d (%s) prayer %d (%s): line was altered or dropped: %q",
						i, b.Name, j, w.Title, want)
				}
			}
		}
	}
}

// The prayer's lines inside the ring share a left edge: they are placed from one
// common left margin, so the stanzas read as a block rather than a ragged
// diamond. Beads sit outside that margin and are ignored here.
func TestStanzasShareALeftEdge(t *testing.T) {
	m := praying(t)
	m.cursor = 3 // a Hail Mary: several lines, so there is a left edge to check

	// The prayer lives in its own panel beside the ring now, so that is what is
	// checked. JoinVertical(Left, ...) is what keeps the lines flush, and it is
	// easy to break by switching that to Center.
	panel := prayerPanel(m)

	col := -1
	checked := 0
	for _, row := range strings.Split(panel, "\n") {
		plain := stripEscapes(row)
		if strings.TrimSpace(plain) == "" {
			continue
		}
		at := len(plain) - len(strings.TrimLeft(plain, " "))
		if col == -1 {
			col = at
			continue
		}
		if at != col {
			t.Errorf("%q starts at column %d, expected %d", strings.TrimSpace(plain), at, col)
		}
		checked++
	}

	if checked < 3 {
		t.Fatalf("only checked %d prayer lines; expected several", checked)
	}
}

// No bead may land on a letter of the MYSTERY, which is what sits inside the ring
// now. This is the invariant the ring-sizing loop exists to maintain, so it is
// checked for every mystery of every set.
func TestNoBeadCollidesWithTheMystery(t *testing.T) {
	m := praying(t)
	g := m.ring

	for _, set := range Sets() {
		for n := 1; n <= len(set.Mysteries); n++ {
			lines := mysteryLines(set, n)

			textW, textH := 0, len(lines)
			for _, l := range lines {
				if x := lipgloss.Width(l); x > textW {
					textW = x
				}
			}

			left, right := g.cx-textW/2, g.cx+textW/2
			top, bottom := g.cy-textH/2, g.cy+textH/2
			for bi, p := range g.pos {
				if p[1] >= top && p[1] <= bottom && p[0] >= left && p[0] <= right {
					t.Fatalf("bead %d sits inside mystery %d of %s at %v", bi, n, set.Name, p)
				}
			}
		}
	}
}

// The highlight must follow the cursor, and land on exactly one bead.
func TestHighlightFollowsCursor(t *testing.T) {
	m := praying(t)
	// The window must be at least the ring's size: lipgloss.Place CLIPS rather
	// than scaling, so a small window silently cuts off the beads at the edges
	// and the highlight can vanish with them.
	next, _ := m.Update(tea.WindowSizeMsg{Width: m.ring.w + 4, Height: m.ring.h + 4})
	m = next.(model)

	for i := range m.beads {
		m.cursor, m.say = i, 0
		out := m.View().Content

		// The current bead is marked by its glyph, a halo, not by a colour: that
		// is what survives NO_COLOR, and asserting on the glyph rather than on an
		// escape code keeps this test about the design intent.
		//
		// The crucifix is the exception: it keeps its own shape when selected, so
		// that the cross never disappears from the rosary.
		if m.beads[i].Kind == Cross {
			if got := strings.Count(out, Cross.Glyph()); got != 1 {
				t.Errorf("cursor %d: the crucifix should stay a cross when selected, found %d", i, got)
			}
		} else {
			want := currentGlyph(m.glow)
			if got := strings.Count(out, want); got != 1 {
				t.Errorf("cursor %d: want exactly 1 %q, got %d", i, want, got)
			}
		}

		// And it must be bold, so it stands out once colour is stripped. Bold is
		// emitted as "1" at the head of the parameter list, e.g.
		// "\x1b[1;38;2;...m◎".
		if !strings.Contains(out, "\x1b[1;") {
			t.Errorf("cursor %d: the current bead is not bold", i)
		}
	}
}
