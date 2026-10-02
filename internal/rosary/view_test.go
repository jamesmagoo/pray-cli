package rosary

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// No counts anywhere on screen. "3 of 10" and "bead 11 of 68" turn praying into
// progress-watching, so their absence is a design decision worth pinning.
func TestNoCountsOnScreen(t *testing.T) {
	m := praying(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: m.ring.w + 6, Height: m.ring.h + 6})
	m = next.(model)

	for i := range m.beads {
		m.cursor, m.say = i, 0
		out := m.View().Content

		for _, bad := range []string{" of 10", " of 3", "bead ", "prayer "} {
			if strings.Contains(out, bad) {
				t.Fatalf("bead %d shows a count: %q", i, bad)
			}
		}
	}
}

// The hint lives in a corner of the screen, not inside the rosary, and must not
// change the rosary's own layout.
func TestHintIsInTheCornerNotTheRing(t *testing.T) {
	m := praying(t)

	// Not in the ring: the lines handed to drawRosary carry no hint text.
	for _, l := range mysteryLines(m.set, m.announced()) {
		if strings.Contains(l, "finish") {
			t.Error("the hint is inside the ring")
		}
	}

	next, _ := m.Update(tea.WindowSizeMsg{Width: m.ring.w + 6, Height: m.ring.h + 6})
	m = next.(model)
	rows := strings.Split(m.View().Content, "\n")

	// On screen: in the lower half, near the left edge.
	found := -1
	for i, r := range rows {
		if strings.Contains(r, "finish") {
			found = i
		}
	}
	if found < 0 {
		t.Fatal("the hint is not on screen at all")
	}
	if found < len(rows)/2 {
		t.Errorf("the hint is at row %d of %d; expected the bottom", found, len(rows))
	}
}

// The overlay must not change a row's display width, or the screen shears.
func TestOverlayPreservesRowWidth(t *testing.T) {
	cases := []string{
		strings.Repeat(" ", 40),
		"  " + lipgloss.NewStyle().Foreground(gold).Render("○") + strings.Repeat(" ", 37),
	}

	for _, row := range cases {
		before := lipgloss.Width(row)
		after := lipgloss.Width(overlay(row, "abcdef", 2))
		if before != after {
			t.Errorf("width changed from %d to %d", before, after)
		}
	}
}

// The overlay counts display columns, not bytes, so escapes already in the row
// must not shift where the text lands.
func TestOverlayIgnoresEscapesWhenCounting(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(gold).Render("○")
	row := styled + strings.Repeat(" ", 20) // 1 visible cell, many bytes

	out := overlay(row, "XY", 5)
	plain := stripEscapes(out)

	// Count DISPLAY COLUMNS, not bytes. strings.Index would return a byte offset,
	// and "○" is three bytes — the very confusion overlay() exists to avoid, and
	// an easy way to write a test that fails on correct code.
	col := lipgloss.Width(plain[:strings.Index(plain, "XY")])
	if col != 5 {
		t.Errorf("overlay landed at column %d, want 5 (row: %q)", col, plain)
	}
}

func stripEscapes(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
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

// The mystery goes INSIDE the ring and the prayer BESIDE it, to the right. This
// is the layout decision, so it is pinned: a regression would put the prayer back
// in the middle, where it used to be.
func TestMysteryInsideRingPrayerToTheRight(t *testing.T) {
	m := praying(t)
	for i, b := range m.beads {
		if b.Announces() == 1 {
			m.cursor = i + 2 // a Hail Mary within the first decade
			break
		}
	}

	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 60})
	m = next.(model)
	rows := strings.Split(stripEscapes(m.View().Content), "\n")

	// Byte offsets would be wrong here too: rows contain multi-byte bead glyphs,
	// so each match is converted to a display column.
	// The mystery's name may be wrapped onto two rows, so this looks for its first
	// line rather than the whole string.
	_, mysteryCol := mysteryRow(rows, Sorrowful.Mysteries[0])
	prayerCol := -1
	for _, row := range rows {
		if i := strings.Index(row, "full of grace"); i >= 0 {
			prayerCol = lipgloss.Width(row[:i])
		}
	}

	if mysteryCol < 0 {
		t.Fatal("the mystery is not on screen")
	}
	if prayerCol < 0 {
		t.Fatal("the prayer is not on screen")
	}
	if prayerCol <= mysteryCol {
		t.Errorf("the prayer starts at column %d, which is not to the right of the mystery at %d",
			prayerCol, mysteryCol)
	}

	// The prayer must be clear of the ring entirely, not overlapping it.
	if prayerCol < m.ring.w {
		t.Errorf("the prayer starts at column %d, inside the ring's %d columns", prayerCol, m.ring.w)
	}
}

// The mystery is centred in the ring, so it must straddle the ring's centre line.
func TestMysteryIsCentredInTheRing(t *testing.T) {
	m := praying(t)
	for i, b := range m.beads {
		if b.Announces() == 1 {
			m.cursor = i
			// The announcement sits partway through a junction bead, so stand on
			// the prayer that makes it.
			for j, w := range b.Says {
				if w.Announces > 0 {
					m.say = j
				}
			}
			break
		}
	}

	lines := mysteryLines(m.set, m.announced())
	out := drawRosary(m.ring, m.beads, m.cursor, 0, 0, lines)

	// The name may be wrapped across rows; the first of its lines is what to find.
	line := wrapMystery(Sorrowful.Mysteries[0])[0]

	for _, row := range strings.Split(out, "\n") {
		plain := stripEscapes(row)
		i := strings.Index(plain, line)
		if i < 0 {
			continue
		}
		// COLUMNS, not bytes: the row has bead glyphs before the text and each is
		// three bytes. strings.Index gives a byte offset, so it must be converted
		// with lipgloss.Width — the same trap as in overlay().
		//
		// Measured against the LINE found, not the whole mystery name: a long name
		// is wrapped across rows, and each row is centred on its own, so the full
		// name's width is not the width of anything on screen.
		col := lipgloss.Width(plain[:i])
		mid := col + lipgloss.Width(line)/2
		if d := mid - m.ring.cx; d < -1 || d > 1 {
			t.Errorf("the mystery's centre is %d columns from the ring's centre", d)
		}
		return
	}
	t.Fatal("the mystery was not drawn inside the ring")
}

// The layout must not move as you pray. A panel sized to its contents makes the
// joined block change width, and Place then re-centres it, so the whole rosary
// slides sideways on every step. Measured before the fix: the panel swung between
// 26 and 44 columns and the ring moved 9 columns.
func TestLayoutDoesNotShiftBetweenPrayers(t *testing.T) {
	m := praying(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 44})
	m = next.(model)

	wantPanel := -1
	wantCross := -1

	for i := range m.beads {
		for j := range m.beads[i].Says {
			m.cursor, m.say = i, j

			if w := lipgloss.Width(prayerPanel(m)); wantPanel == -1 {
				wantPanel = w
			} else if w != wantPanel {
				t.Fatalf("bead %d prayer %d: panel is %d wide, every other frame is %d",
					i, j, w, wantPanel)
			}

			// The crucifix is a fixed point of the drawing, so if it moves, the
			// whole rosary moved.
			col := -1
			for _, row := range strings.Split(m.View().Content, "\n") {
				plain := stripEscapes(row)
				if k := strings.Index(plain, Cross.Glyph()); k >= 0 {
					col = lipgloss.Width(plain[:k])
				}
			}
			if wantCross == -1 {
				wantCross = col
			} else if col != wantCross {
				t.Fatalf("bead %d prayer %d: the rosary is at column %d, elsewhere %d",
					i, j, col, wantCross)
			}
		}
	}
}

// The mystery is styled — it is the thing being contemplated, so it must not be
// plain text even though it has no box around it.
func TestMysteryIsStyled(t *testing.T) {
	m := praying(t)
	for i, b := range m.beads {
		if b.Announces() == 1 {
			m.cursor = i
			// The announcement sits partway through a junction bead, so stand on
			// the prayer that makes it.
			for j, w := range b.Says {
				if w.Announces > 0 {
					m.say = j
				}
			}
			break
		}
	}

	out := drawRosary(m.ring, m.beads, m.cursor, 0, 0, mysteryLines(m.set, m.announced()))

	first := wrapMystery(Sorrowful.Mysteries[0])[0]
	for _, row := range strings.Split(out, "\n") {
		if !strings.Contains(stripEscapes(row), first) {
			continue
		}
		if !strings.Contains(row, "\x1b[") {
			t.Error("the mystery is drawn unstyled")
		}
		return
	}
	t.Fatal("the mystery was not drawn in the ring")
}

// The heading names the PRAYER being said, not the bead.
//
// A junction bead carries the Glory Be, the Fatima Prayer and the next Our
// Father. Naming the bead showed "Our Father" above the words of the Glory Be —
// the bead is where you are, the prayer is what you are saying.
func TestHeadingNamesThePrayerNotTheBead(t *testing.T) {
	m := praying(t)

	checked := 0
	for i, b := range m.beads {
		if len(b.Says) < 2 {
			continue // only multi-prayer beads can show the wrong name
		}

		for j, w := range b.Says {
			m.cursor, m.say = i, j

			head := strings.TrimSpace(strings.Split(stripEscapes(prayerPanel(m)), "\n")[0])
			if head != w.Title {
				t.Errorf("bead %d (%s) prayer %d: heading is %q, want %q",
					i, b.Name, j, head, w.Title)
			}
			checked++
		}
	}

	if checked == 0 {
		t.Fatal("no multi-prayer beads found; this test checked nothing")
	}
}
