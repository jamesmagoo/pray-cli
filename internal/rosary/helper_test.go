package rosary

import "testing"

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
