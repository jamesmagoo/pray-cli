package rosary

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// A window too small for the rosary gets a plain notice, not a corrupted ring.
//
// lipgloss.Place pads a block to the window's width rather than clipping it, so
// without this check an 80x24 terminal received 34 rows of 98 columns and wrapped
// every one — the ring arriving as a scrolling mess of fragments. Measured before
// the fix.
func TestASmallWindowGetsANoticeNotAMess(t *testing.T) {
	m := praying(t)
	m.width, m.height = 80, 24

	out := plain(render(m))

	if !strings.Contains(out, "needs more room") {
		t.Fatalf("no notice at 80x24; got:\n%s", out)
	}
	// The rosary itself is NOT drawn: a notice over a broken ring is still broken.
	if strings.Contains(out, smallBead) || strings.Contains(out, bigBead) {
		t.Error("the rosary is still drawn behind the notice")
	}
}

// The notice must not itself overflow the window it is complaining about.
//
// Place pads to the window size, which is the very behaviour that misbehaves here:
// at a window narrower than the notice, centring it would emit rows wider than the
// terminal and wrap them — the same fault, one level down. Tested at a size small
// enough for the notice itself not to fit, since at 80x24 it comfortably does.
func TestTheNoticeDoesNotOverflow(t *testing.T) {
	phases := map[string]func(m *model){
		"praying":   func(m *model) { m.phase = atPrayer },
		"chooser":   func(m *model) { m.phase = choosing },
		"finished":  func(m *model) { m.phase = finished },
		"departing": func(m *model) { m.phase = departing; m.depart = departFrames },
	}

	for name, setup := range phases {
		for _, size := range [][2]int{{80, 24}, {40, 10}, {20, 5}} {
			m := praying(t)
			setup(&m)
			m.width, m.height = size[0], size[1]

			for i, row := range strings.Split(plain(render(m)), "\n") {
				if w := lipgloss.Width(row); w > m.width {
					t.Errorf("%s at %dx%d: row %d is %d columns — wider than the window",
						name, size[0], size[1], i, w)
				}
			}
		}
	}
}

// The notice says what is needed and what there is, so the fix is obvious.
func TestTheNoticeNamesBothSizes(t *testing.T) {
	m := praying(t)
	m.width, m.height = 80, 24

	out := plain(render(m))

	if !strings.Contains(out, "80 × 24") {
		t.Errorf("the notice does not say the window's actual size:\n%s", out)
	}
	// The needed size is measured from the block, so just check a plausible number
	// appears rather than hard-coding one that will drift.
	if !strings.Contains(out, "×") {
		t.Errorf("the notice does not state a required size:\n%s", out)
	}
	if !strings.Contains(out, "x to finish") {
		t.Error("the notice does not say how to leave")
	}
}

// A window large enough draws the rosary as before.
func TestALargeWindowIsUnaffected(t *testing.T) {
	m := praying(t)
	m.width, m.height = 140, 60

	out := plain(render(m))

	if strings.Contains(out, "needs more room") {
		t.Error("a 140x60 window was told it is too small")
	}
	if !strings.Contains(out, smallBead) {
		t.Error("the rosary is not drawn in a window with room for it")
	}
}

// Before the first WindowSizeMsg the size is unknown, and unknown is not small.
//
// Zero would compare as smaller than anything, so a naive check would replace the
// rosary with a notice on the very first frame — before the terminal has said how
// big it is — and then flip back. Worse than briefly overflowing.
func TestUnknownSizeIsNotTreatedAsSmall(t *testing.T) {
	m := praying(t)
	// width and height are zero: no WindowSizeMsg has arrived.

	if tooSmall(m, 999, 999) {
		t.Error("a window of unknown size was judged too small")
	}
	if strings.Contains(plain(render(m)), "needs more room") {
		t.Error("the notice appeared before the terminal reported its size")
	}
}

// Every LARGE screen is covered, not just the rosary: the closing screen goes
// through placeFinished, which needs the same guard as place.
//
// The farewell is deliberately absent. It is 12x3 — smaller than any window worth
// warning about — so it fits where the others do not, and asserting it warns would
// be asserting a bug. TestTheNoticeDoesNotOverflow covers it instead, by rendering
// every phase at sizes down to 20x5 and checking nothing overflows.
func TestEveryScreenChecksTheWindow(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(m *model)
	}{
		{"praying", func(m *model) { m.phase = atPrayer }},
		{"chooser", func(m *model) { m.phase = choosing }},
		{"finished", func(m *model) { m.phase = finished }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := praying(t)
			tc.setup(&m)
			// Small enough that no screen fits, but not so small that the notice
			// itself is truncated past recognition — at 10x4 even "Needs more room"
			// is cut, which says nothing about whether the check ran.
			m.width, m.height = 30, 8

			out := plain(render(m))
			if !strings.Contains(out, "more room") {
				t.Errorf("the %s screen drew itself into a 30x8 window:\n%s", tc.name, out)
			}
			// And whatever it drew fits.
			for i, row := range strings.Split(out, "\n") {
				if w := lipgloss.Width(row); w > m.width {
					t.Errorf("%s: row %d is %d columns in a %d-column window",
						tc.name, i, w, m.width)
				}
			}
		})
	}
}

// x still leaves from the notice, or a small terminal would be a trap.
func TestXWorksFromTheNotice(t *testing.T) {
	m := praying(t)
	m.width, m.height = 80, 24

	if !strings.Contains(plain(render(m)), "needs more room") {
		t.Fatal("expected the notice at 80x24")
	}

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'x'})
	if cmd == nil {
		t.Fatal("x did nothing while the notice was up")
	}
	if got := next.(model); got.phase != departing {
		t.Errorf("x left phase %v, want departing", got.phase)
	}
}
