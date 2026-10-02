package rosary

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Exactly five mysteries are announced, once each, in order 1..5.
func TestEachMysteryIsAnnouncedOnce(t *testing.T) {
	m := praying(t)

	var announced []int
	for _, b := range m.beads {
		if b.Announces > 0 {
			announced = append(announced, b.Announces)
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
		if b.Announces == 1 {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatal("no bead announces the first mystery")
	}

	want := Sorrowful.Mysteries[0]
	for i := start; i < len(m.beads); i++ {
		m.cursor = i
		got, ok := m.mystery()
		if !ok {
			t.Fatalf("bead %d has no mystery", i)
		}
		if m.beads[i].Announces == 2 {
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
		if m.beads[i].Announces > 0 {
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
		if b.Announces == 3 {
			m.cursor = i
			break
		}
	}

	inside := strings.Join(mysteryLines(m.set, m.announced()), "\n")
	if !strings.Contains(inside, Sorrowful.Mysteries[2]) {
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

	if !m.choosing {
		t.Fatal("the rosary did not open on the chooser")
	}
	if !strings.Contains(render(m), Sorrowful.Name) {
		t.Error("the chooser does not list the mysteries")
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	got := next.(model)

	if got.choosing {
		t.Error("space did not leave the chooser")
	}
	if got.set.Name != Sorrowful.Name {
		t.Errorf("began with %q, want %q", got.set.Name, Sorrowful.Name)
	}
	if got.cursor != 0 || got.say != 0 {
		t.Errorf("began at bead %d prayer %d, want the very start", got.cursor, got.say)
	}
}

// Praying keys must do nothing while the chooser is up, or a stray space would
// both choose and advance.
func TestPrayingKeysAreInertWhileChoosing(t *testing.T) {
	m, _ := newModel("en", Sorrowful)

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if got := next.(model); got.cursor != 0 || !got.choosing {
		t.Error("an arrow key moved the rosary while the chooser was up")
	}
}
