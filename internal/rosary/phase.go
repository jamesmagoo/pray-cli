package rosary

// phase is which of the three screens the program is showing.
//
// It replaces a pair of booleans. Two flags (choosing, done) have four states of
// which one is nonsense — choosing AND done at once — and every key handler would
// have to check both in the right order. One phase has exactly the three states
// that exist, so an impossible screen cannot be represented and Update can switch
// on it.
type phase int

const (
	// choosing is the opening screen: which mysteries to contemplate. The rosary
	// is already built behind it, so beginning is instant.
	choosing phase = iota
	// atPrayer is the rosary itself. (Named for the phase, not "praying", which
	// is the test helper that builds a model already in it.)
	atPrayer
	// finished is the closing screen, reached by praying the last bead. It is a
	// phase rather than an immediate quit because arriving at the end of the
	// rosary is a moment, and because it is where beginning again belongs.
	finished
	// departing is the farewell: a last word fading out, after which the program
	// quits itself.
	//
	// It exists because tea.Quit tears the program down immediately — there is no
	// "quit after this animation" command. So leaving has to be a phase that
	// animates and then returns tea.Quit on its own last frame. Verified: the
	// runtime does call View for the frame that quits, so the final frame is seen.
	departing
)
