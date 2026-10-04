package rosary

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// praying builds a model already past the chooser, which is what almost every
// test wants: the chooser is its own phase, and testing it means setting that
// phase deliberately.
func praying(t *testing.T) model {
	t.Helper()
	m, err := newModel("en", Sorrowful)
	if err != nil {
		t.Fatal(err)
	}
	m.phase = atPrayer
	return m
}

// showsMystery reports whether screen displays the given mystery's name,
// allowing for it being wrapped across lines.
//
// A long name is broken onto two rows inside the ring (see wrapMystery), so
// strings.Contains on the whole name finds nothing however correct the drawing
// is. Matching the wrapped pieces is what the eye does, and what these tests
// mean.
func showsMystery(screen, name string) bool {
	for _, part := range wrapMystery(name) {
		if !strings.Contains(screen, part) {
			return false
		}
	}
	return true
}

// mysteryRow is the row index at which the given mystery's name begins, or -1.
//
// Its FIRST line: a wrapped name occupies several rows, and the tests that care
// about position care where it starts.
func mysteryRow(rows []string, name string) (row, col int) {
	first := wrapMystery(name)[0]
	for i, r := range rows {
		if k := strings.Index(r, first); k >= 0 {
			return i, lipgloss.Width(r[:k])
		}
	}
	return -1, -1
}
