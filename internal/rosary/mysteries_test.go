package rosary

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// sayThatAnnounces is the index of the prayer on bead i that declares a mystery,
// or -1. Tests need it because the announcement sits partway through a junction
// bead: at say 0 the previous decade is still being finished.
func sayThatAnnounces(b Bead) int {
	for j, w := range b.Says {
		if w.Announces > 0 {
			return j
		}
	}
	return -1
}

// Exactly five mysteries are announced, once each, in order 1..5.
func TestEachMysteryIsAnnouncedOnce(t *testing.T) {
	m := praying(t)

	var announced []int
	for _, b := range m.beads {
		if b.Announces() > 0 {
			announced = append(announced, b.Announces())
		}
	}

	if len(announced) != 5 {
		t.Fatalf("%d beads announce a mystery, want 5: %v", len(announced), announced)
	}
	for i, n := range announced {
		if n != i+1 {
			t.Errorf("announcement %d is for mystery %d, want %d", i, n, i+1)
		}
	}
}

// A mystery is announced on one bead and must stay in force until the next
// announcement — that is what lets it persist through the ten Hail Marys.
func TestMysteryPersistsThroughTheDecade(t *testing.T) {
	m := praying(t)

	// Find the bead that announces mystery 1, then walk forward.
	start := -1
	for i, b := range m.beads {
		if b.Announces() == 1 {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatal("no bead announces the first mystery")
	}

	want := Sorrowful.Mysteries[0]
	for i := start; i < len(m.beads); i++ {
		// Stand on the announcing prayer, not the start of the bead: on a junction
		// bead the earlier prayers still belong to the decade before.
		m.cursor = i
		m.say = 0
		if j := sayThatAnnounces(m.beads[i]); j > 0 && i == start {
			m.say = j
		}
		got, ok := m.mystery()
		if !ok {
			t.Fatalf("bead %d has no mystery", i)
		}
		if m.beads[i].Announces() == 2 {
			break // the next decade has begun, correctly
		}
		if got != want {
			t.Fatalf("bead %d shows %q, want %q", i, got, want)
		}
	}
}

// Nothing is contemplated before the first mystery is announced: the pendant
// prayers get no mystery rather than the wrong one.
func TestNoMysteryBeforeTheFirstAnnouncement(t *testing.T) {
	m := praying(t)

	for i := 0; i < len(m.beads); i++ {
		m.cursor = i
		if m.beads[i].Announces() > 0 {
			return // reached the first announcement; everything before was clear
		}
		if _, ok := m.mystery(); ok {
			t.Fatalf("bead %d has a mystery before any was announced", i)
		}
	}
}

// The mystery is shown in the middle of the ring, and only once one is in force.
func TestMysteryAppearsInsideTheRing(t *testing.T) {
	m := praying(t)

	m.cursor = 0
	if got := mysteryLines(m.set, m.announced()); got != nil {
		t.Errorf("a mystery is drawn before any was announced: %q", got)
	}

	for i, b := range m.beads {
		if b.Announces() == 3 {
			m.cursor = i
			m.say = max(sayThatAnnounces(b), 0)
			break
		}
	}

	inside := strings.Join(mysteryLines(m.set, m.announced()), "\n")
	if !showsMystery(inside, Sorrowful.Mysteries[2]) {
		t.Errorf("the ring does not name the third mystery:\n%s", inside)
	}
	if !strings.Contains(inside, Sorrowful.Name) {
		t.Errorf("the ring does not name the set:\n%s", inside)
	}
}

// Mystery() must not index outside the set.
func TestMysteryIndexIsBounded(t *testing.T) {
	for _, n := range []int{-1, 0, 6, 99} {
		if _, ok := Sorrowful.Mystery(n); ok {
			t.Errorf("Mystery(%d) returned a mystery; want none", n)
		}
	}
	for n := 1; n <= 5; n++ {
		if _, ok := Sorrowful.Mystery(n); !ok {
			t.Errorf("Mystery(%d) returned nothing", n)
		}
	}
}

// The chooser must run before any prayer, and space must start the rosary.
func TestChooserRunsFirstAndSpaceBegins(t *testing.T) {
	m, err := newModel("en", Sorrowful)
	if err != nil {
		t.Fatal(err)
	}

	// The opening comes first, and hands over to the chooser — never to a prayer.
	if m.phase != opening {
		t.Fatal("the rosary did not begin with the opening")
	}
	m = press(t, m, "space")
	if m.phase != choosing {
		t.Fatal("the opening did not hand over to the chooser")
	}
	if !strings.Contains(render(m), Sorrowful.Name) {
		t.Error("the chooser does not list the mysteries")
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	got := next.(model)

	if got.phase != atPrayer {
		t.Error("space did not leave the chooser for the rosary")
	}
	// The set the chooser was HIGHLIGHTING, not the one newModel was handed: the
	// argument is a default to carry until a choice is made, and space makes it.
	if want := Sets()[0].Name; got.set.Name != want {
		t.Errorf("began with %q, want the highlighted set %q", got.set.Name, want)
	}
	if got.cursor != 0 || got.say != 0 {
		t.Errorf("began at bead %d prayer %d, want the very start", got.cursor, got.say)
	}
}

// Praying keys must do nothing while the chooser is up, or a stray space would
// both choose and advance.
func TestPrayingKeysAreInertWhileChoosing(t *testing.T) {
	m := mustModel(t)

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if got := next.(model); got.cursor != 0 || got.phase != choosing {
		t.Error("an arrow key moved the rosary while the chooser was up")
	}
}

// The mystery changes AFTER the Glory Be and Fatima Prayer, on the Our Father.
//
// A junction bead finishes the previous decade before opening the new one, so
// announcing on the bead changed the mystery too early: the Glory Be and Fatima
// showed the next decade's mystery while still closing the one before.
func TestMysteryChangesOnTheOurFatherNotOnArrival(t *testing.T) {
	m := praying(t)

	checked := 0
	for i, b := range m.beads {
		n := b.Announces()
		if n < 2 {
			continue // the first decade has no previous one to finish
		}

		announcesAt := sayThatAnnounces(b)
		if announcesAt <= 0 {
			t.Errorf("bead %d announces mystery %d at prayer %d; the Glory Be and Fatima should come first",
				i, n, announcesAt)
			continue
		}

		// Before the announcing prayer, the PREVIOUS mystery is still in force.
		prev, _ := m.set.Mystery(n - 1)
		for say := 0; say < announcesAt; say++ {
			m.cursor, m.say = i, say
			got, _ := m.mystery()
			if got != prev {
				t.Errorf("bead %d prayer %d (%s) shows %q; should still be %q",
					i, say, b.Says[say].Title, got, prev)
			}
		}

		// On it and after, the new one.
		want, _ := m.set.Mystery(n)
		for say := announcesAt; say < len(b.Says); say++ {
			m.cursor, m.say = i, say
			if got, _ := m.mystery(); got != want {
				t.Errorf("bead %d prayer %d shows %q, want %q", i, say, got, want)
			}
		}
		checked++
	}

	if checked != 4 {
		t.Errorf("checked %d junctions, expected 4 (decades 2 to 5)", checked)
	}
}
