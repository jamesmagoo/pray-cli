package rosary

import (
	"io"

	tea "charm.land/bubbletea/v2"
)

// Run starts the interactive rosary and blocks until the user quits.
//
// The cli package calls this and knows nothing about Bubble Tea: the whole TUI
// is behind this one function, so the command stays a thin wrapper and the
// program can be driven by a test with its own input and output.
func Run(in io.Reader, out io.Writer, lang string) error {
	// The set is chosen on the first screen, so a default is all that is needed
	// here; the user's pick replaces it before any prayer is said.
	m, err := newModel(lang, Sets()[0])
	if err != nil {
		return err
	}

	// Input and output are passed in rather than taken from os.Stdin/Stdout so
	// that the caller (and a test) decides where the program is attached.
	p := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out))

	// Run returns the final model, which is how a Bubble Tea program hands
	// results back. We have nothing to report yet; later this is where the
	// rosary says how far the user got.
	_, err = p.Run()
	return err
}
