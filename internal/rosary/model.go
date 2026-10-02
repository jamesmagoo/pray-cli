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

	// phase is which screen is showing: the chooser, the rosary, or the close.
	// All three are phases of the SAME model rather than separate programs — one
	// Update, one View, and the rosary is already built behind the chooser, so
	// beginning is instant and beginning again needs no reload.
	phase  phase
	choice int // which set is highlighted in the chooser

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

	// depart is the farewell's fade-out: departFrames -> 0, after which the program
	// quits itself. It is separate from fade because it runs the other way (lit to
	// dark) and because it must reach 0 — the other two may be cut short by a key
	// press, this one cannot be, or the program would quit mid-fade.
	depart int
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
		beads:  beads,
		set:    set,
		ring:   fixedRing(beads),
		panelW: widestPrayer(beads),
		phase:  choosing,
	}, nil
}

// begin starts the rosary from the top with the chosen set.
//
// It is also how beginning AGAIN works, and that is the reason it exists as a
// method: the close offers another rosary, and "again" must mean exactly what
// "begin" meant the first time. Writing the reset twice is how the second one
// ends up forgetting a field — the mystery is derived by looking BACK from the
// cursor (see announced), so a stale cursor would carry the last decade's mystery
// into the new rosary's opening prayers.
//
// Note what it does NOT touch: beads, ring and panelW. Those are the rosary as an
// object — measured once in newModel, the same on the second time through as on
// the first — so beginning again is instant and the ring does not resize.
func (m *model) begin(set MysterySet) tea.Cmd {
	m.set = set
	m.cursor = 0
	m.say = 0
	m.phase = atPrayer
	return m.startStep()
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

// leave begins the farewell: the fade-out, after which the program quits itself.
//
// Only `x` comes through here — the considered exit, the one the hint calls
// "finish". It is a method so that every screen's `x` means the same thing and
// none can quit abruptly by forgetting the farewell.
//
// `esc` and `ctrl+c` deliberately do NOT: they are escape hatches, and someone
// reaching for either wants out now, not in 1.2 seconds. A farewell is a courtesy
// on the way out, not a toll on it.
func (m *model) leave() tea.Cmd {
	m.phase = departing
	m.depart = departFrames
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
		// The farewell is its own case: it is the one animation whose END does
		// something rather than merely stopping, so it must not fall through to the
		// shared "stop if nothing is running" test below.
		if m.phase == departing {
			m.depart--
			if m.depart <= 0 {
				// Last frame. View is called for this one before the program is torn
				// down — verified against the runtime — so the fade is seen through to
				// black rather than cut off one frame early.
				return m, tea.Quit
			}
			return m, tick()
		}

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
		// Each phase reads the keys differently, so each gets its own handler and
		// returns early. Sharing one switch between them is how a key ends up
		// meaning two things at once — a space that both chooses a set and advances
		// the first prayer.
		switch m.phase {
		case choosing:
			return m.updateChoosing(msg)
		case finished:
			return m.updateFinished(msg)
		case departing:
			// The farewell ignores every key but the escape hatches. It lasts 1.2s
			// and ends by itself; letting a stray space cancel it or hurry it along
			// would make leaving feel uncertain. esc and ctrl+c still cut it short,
			// because an escape hatch that stops working for 1.2s is not one.
			switch msg.String() {
			case "esc", "ctrl+c":
				return m, tea.Quit
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
			// next() returned false: that was the last prayer on the last bead, so
			// the rosary is complete. Space carried the user through all 61 beads and
			// it carries them over the finish too — the same key, so no new gesture
			// has to be learned at the one moment the hands are not looking.
			//
			// The animations are started again deliberately: the closing screen fades
			// in exactly as every prayer did, so arriving at the end feels like one
			// more step rather than a different program taking over.
			m.phase = finished
			return m, m.startStep()
		case "left", "h", "backspace":
			if m.prev() {
				return m, m.startStep()
			}
		case "esc", "ctrl+c":
			// tea.Quit is a command, not a function call: we hand it back and
			// the runtime shuts the program down cleanly, restoring the
			// terminal. Never call os.Exit from inside a Bubble Tea program.
			//
			// These two quit straight away, with no farewell. They are escape
			// hatches: ctrl+c is an interrupt, and esc is the key people hit when
			// they want out of a full-screen program NOW. Making either sit through
			// a 1.2s animation would be the wrong answer to "stop".
			return m, tea.Quit
		case "x":
			// x is the considered exit — the one the hint offers as "finish" — so it
			// gets the farewell. Leaving is an animation, not an event: see leave().
			return m, m.leave()
		}
	}

	// Any message we don't care about leaves the state untouched. Returning m
	// unchanged is normal and cheap.
	return m, nil
}

// updateChoosing handles keys on the opening screen: move the highlight, or begin.
func (m model) updateChoosing(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		return m, tea.Quit // escape hatches: out now, no animation
	case "x":
		return m, m.leave()
	case "up", "k":
		if m.choice > 0 {
			m.choice--
		}
	case "down", "j":
		if m.choice < len(Sets())-1 {
			m.choice++
		}
	case "space", " ", "enter", "right", "l":
		return m, m.begin(Sets()[m.choice])
	}
	return m, nil
}

// updateFinished handles keys on the closing screen.
//
// Two ways on from here, and the distinction is the point of the screen:
//
//	space  — begin again, back at the chooser, so the set can be changed
//	x      — finish, and leave
//
// Space returns to the CHOOSER rather than straight into another rosary. Praying a
// second set is the usual reason to go again (the Joyful after the Sorrowful), so
// the choice has to be offered; and a space pressed out of habit at the end should
// not silently commit the user to another five decades.
//
// The left arrow still steps back into the rosary. Finishing is not a wall: the
// user may want to re-read the Hail Holy Queen, and the key that has meant "back"
// all the way round should not stop meaning it at the end.
func (m model) updateFinished(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		return m, tea.Quit
	case "x":
		return m, m.leave()
	case "space", " ", "enter":
		// Back to the chooser, with the rosary reset behind it so the beads are
		// already at the start when a set is picked.
		m.phase = choosing
		m.cursor = 0
		m.say = 0
		return m, m.startStep()
	case "left", "h", "backspace":
		// Back into the last prayer of the rosary.
		m.phase = atPrayer
		return m, m.startStep()
	}
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
