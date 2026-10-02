package rosary

import "testing"

// praying builds a model already past the chooser, which is what almost every
// test wants: the chooser is a separate phase and testing it needs choosing=true
// set deliberately.
func praying(t *testing.T) model {
	t.Helper()
	m, err := newModel("en", Sorrowful)
	if err != nil {
		t.Fatal(err)
	}
	m.choosing = false
	return m
}
