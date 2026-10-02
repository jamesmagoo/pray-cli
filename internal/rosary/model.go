// Package rosary is the interactive rosary: a Bubble Tea program that walks
// through the beads one at a time.
//
// Bubble Tea is built on one idea, borrowed from Elm: all state lives in a
// single value (the Model), and the only way state changes is by a message
// arriving. The runtime owns the loop; we only describe the three pieces of it:
//
//	Init()        once, at the start: what to do before any key is pressed
//	Update(msg)   on every message (a key, a resize, a tick): the next state
//	View()        after every Update: the state drawn as text
//
// Nothing in here prints. View returns a string and the runtime puts it on
// screen, which is the same discipline RENDERING.md sets for the static
// renderer, and the reason both can be tested by comparing strings.
package rosary

import (
	tea "charm.land/bubbletea/v2"
)

// model is the whole state of the interactive rosary. Right now that is one
// bead; it grows as the rosary does.
//
// It is a value, not a pointer. Bubble Tea hands us the model and takes back a
// new one from Update, so each state is a distinct value and there is no shared
// mutable state to reason about.
type model struct {
	beads  []Bead // the whole rosary, in the order prayed
	cursor int    // which bead we are on: an index into beads
	say    int    // which prayer ON that bead: an index into bead.Says

	// set is the mysteries being contemplated, chosen before praying begins.
	set MysterySet

	// choosing is true before the user has picked a set. The chooser is a phase of
	// the same model rather than a separate program: one Update, one View, and the
	// rosary is already built behind it, so starting is instant.
	choosing bool
	choice   int // which set is highlighted in the chooser

	// panelW is the prayer panel's fixed width: the widest prayer in the whole
	// rosary. Computed once, like the ring, and for the same reason — a panel that
	// resizes per prayer makes the whole layout slide sideways as you pray.
	panelW int

	// ring is computed ONCE, in newModel, and never again. The rosary is a
	// physical object: it does not change shape because a longer prayer came
	// along. Every bead therefore draws in the same ring, in the same place.
	ring ringGeometry

	width  int // terminal width, 0 until the first resize message
	height int // terminal height, 0 until the first resize message

	// Two independent countdowns, one per animation, so each can be tuned on its
	// own: glowFrames and fadeFrames are separate knobs and changing one does not
	// touch the other.
	//
	// Both tick on the same timer — a single frameMsg decrements whichever are
	// still running — so there is one clock but two durations.
	glow int // the bead's flare:     glowFrames -> 0
	fade int // the prayer's arrival: fadeFrames -> 0
}

// bead is the bead being prayed now.
func (m model) bead() Bead { return m.beads[m.cursor] }

// mystery is the mystery being contemplated now, and whether there is one.
//
// It is found by looking BACK from the cursor for the most recent bead that
// announced one. That is what makes a mystery persist: it is announced once, on
// one bead, and remains in force until another bead announces the next, with
// nothing to clear and no state to keep in sync.
func (m model) mystery() (string, bool) {
	return m.set.Mystery(m.announced())
}

// announced is the number of the mystery in force now, or 0 before the first
// announcement.
//
// It is found by looking BACK from the cursor for the most recent bead that
// announced one. That is what makes a mystery persist: it is announced once, on
// one bead, and remains in force until another bead announces the next, with
// nothing to clear and no state to keep in sync.
func (m model) announced() int {
	// Walk back over PRAYERS, not beads: a junction bead finishes the previous
	// decade before opening the new one, so the announcement sits partway through
	// it. On the current bead only the prayers up to and including the current one
	// count — a mystery announced later on this same bead has not been reached.
	for i := m.cursor; i >= 0; i-- {
		says := m.beads[i].Says
		last := len(says) - 1
		if i == m.cursor {
			last = min(m.say, last)
		}
		for j := last; j >= 0; j-- {
			if n := says[j].Announces; n > 0 {
				return n
			}
		}
	}
	return 0
}

// words is the prayer being said now: the say'th prayer on the current bead.
func (m model) words() Words {
	b := m.bead()
	if len(b.Says) == 0 {
		return Words{Title: b.Name}
	}
	return b.Says[min(m.say, len(b.Says)-1)]
}

// next advances one step: through the prayers on this bead, then to the next
// bead. Returns false at the very end of the rosary.
//
// This is the whole navigation model, and keeping it in one place is why space,
// enter and the arrow key can all mean "next" without duplicating the rule.
func (m *model) next() bool {
	if m.say < len(m.bead().Says)-1 {
		m.say++ // another prayer on the same bead
		return true
	}
	if m.cursor < len(m.beads)-1 {
		m.cursor++
		m.say = 0
		return true
	}
	return false
}

// prev steps back the same way.
func (m *model) prev() bool {
	if m.say > 0 {
		m.say--
		return true
	}
	if m.cursor > 0 {
		m.cursor--
		m.say = len(m.bead().Says) - 1 // land on the LAST prayer of that bead
		if m.say < 0 {
			m.say = 0
		}
		return true
	}
	return false
}

// newModel builds the starting state. Loading the prayer here, before the
// program runs, means a missing prayer is an ordinary CLI error rather than a
// failure inside a full-screen program the user then has to escape from.
func newModel(lang string, set MysterySet) (model, error) {
	beads := Sequence()

	// Resolve every Say() into real text now, so a missing prayer is an ordinary
	// CLI error rather than a failure inside a full-screen program the user then
	// has to escape from. It also means no file is read while praying.
	for i, b := range beads {
		for j, w := range b.Says {
			resolved, err := w.resolve(lang)
			if err != nil {
				return model{}, err
			}
			beads[i].Says[j] = resolved
		}
	}

	// The ring is sized once, to the widest prayer in the whole rosary, and then
	// fixed. Sizing it per bead would make the ring breathe as you moved through
	// the prayers, which looks like a glitch rather than a design.
	return model{
		beads:    beads,
		set:      set,
		ring:     fixedRing(beads),
		panelW:   widestPrayer(beads),
		choosing: true,
	}, nil
}

// startStep begins the animations for a new step: the bead's flare and the
// prayer's fade-in.
//
// ONE counter drives both. It runs for the longer of the two (the fade), and each
// animation reads it through its own curve — the bead only looks at the first
// glowFrames, the text uses the whole run. That keeps a single timer and makes it
// impossible for the two to drift out of step.
//
// Pressing a key mid-animation simply resets the counter: the in-flight timer
// still delivers its frameMsg, and that handler re-arms only while the counter is
// above zero, so rapid key presses cannot stack up overlapping animations. State
// decides what happens, not the number of timers in flight.
func (m *model) startStep() tea.Cmd {
	m.glow = glowFrames
	m.fade = fadeFrames
	return tick()
}

// Init runs once before the first View. It returns a command: a function the
// runtime runs for us, whose result comes back as a message to Update. We have
// nothing to do up front, so nil.
//
// (The terminal size arrives on its own as a WindowSizeMsg at startup, so we
// don't have to ask for it.)
func (m model) Init() tea.Cmd {
	return nil
}

// Update is the only place state changes. It receives a message, decides what
// the next state is, and returns it along with any command to run.
//
// The type switch is the heart of it: a Msg is an empty interface, so we ask
// what actually arrived.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Sent once at startup and again on every terminal resize. We store it so
	// View can centre the prayer in the window.
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	// One animation frame has elapsed. Advance every countdown that is still
	// running, and keep ticking while any of them is.
	case frameMsg:
		if m.glow > 0 {
			m.glow--
		}
		if m.fade > 0 {
			m.fade--
		}
		if m.glow > 0 || m.fade > 0 {
			// Still animating, so ask for another frame. The animation stops by
			// simply not re-arming here, which is why no cancellation or cleanup
			// is needed anywhere.
			return m, tick()
		}
		return m, nil

	// In v2 a key press and a key release are separate messages. We want
	// presses; KeyReleaseMsg exists but most terminals never send it.
	case tea.KeyPressMsg:
		// While choosing, the keys mean something different. Handling that here
		// and returning early keeps the two phases from sharing key logic.
		if m.choosing {
			switch msg.String() {
			case "x", "esc", "ctrl+c":
				return m, tea.Quit
			case "up", "k":
				if m.choice > 0 {
					m.choice--
				}
			case "down", "j":
				if m.choice < len(Sets())-1 {
					m.choice++
				}
			case "space", " ", "enter", "right", "l":
				m.set = Sets()[m.choice]
				m.choosing = false
				return m, m.startStep()
			}
			return m, nil
		}

		switch msg.String() {
		// Space is how you pray: one press, one prayer. Enter and the right arrow
		// do the same thing for people who reach for them, but space is the one
		// the thumb finds without looking, which is the point when your eyes are
		// on the words.
		// NOTE: in Bubble Tea v2 the space bar's String() is "space", NOT " ".
		// Matching " " silently never fires — the key just does nothing.
		case "space", " ", "enter", "right", "l":
			if m.next() {
				return m, m.startStep()
			}
		case "left", "h", "backspace":
			if m.prev() {
				return m, m.startStep()
			}
		case "x", "esc", "ctrl+c":
			// tea.Quit is a command, not a function call: we hand it back and
			// the runtime shuts the program down cleanly, restoring the
			// terminal. Never call os.Exit from inside a Bubble Tea program.
			return m, tea.Quit
		}
	}

	// Any message we don't care about leaves the state untouched. Returning m
	// unchanged is normal and cheap.
	return m, nil
}

// View draws the current state. It is called after every Update, so it must be
// a pure function of the model: no I/O, no clock, no randomness. Same model in,
// same pixels out — which is what makes it testable.
//
// In v2 this returns a tea.View rather than a plain string. That struct carries
// the content plus how the terminal should present it, and this is the change
// worth understanding: in v1 full-window mode was a program option fixed at
// startup (tea.WithAltScreen). In v2 it is AltScreen, a field on the thing we
// return every frame, so the model decides it and can change its mind.
//
// The alternate screen is the terminal's second buffer: we draw into a blank
// window and, on quit, the terminal restores the scrollback exactly as it was.
// The prayer leaves no trace in the shell history, which is right here, and it
// is why View centres on the full height above.
func (m model) View() tea.View {
	v := tea.NewView(render(m))
	v.AltScreen = true
	v.WindowTitle = "pray · " + m.words().Title
	return v
}
