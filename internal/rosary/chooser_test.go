package rosary

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// The opening screen names the devotion, in full.
func TestTheChooserNamesTheDevotion(t *testing.T) {
	m, err := newModel("en", Sorrowful)
	if err != nil {
		t.Fatal(err)
	}

	screen := plain(chooser(m))

	for _, want := range []string{rosaryTitle, rosarySubtitle} {
		if !strings.Contains(screen, want) {
			t.Errorf("the opening screen does not say %q:\n%s", want, screen)
		}
	}

	// The title comes above the choices, not below them.
	title := strings.Index(screen, rosaryTitle)
	choice := strings.Index(screen, Sorrowful.Name)
	if title < 0 || choice < 0 {
		t.Fatalf("title at %d, choice at %d; expected both", title, choice)
	}
	if title > choice {
		t.Error("the title is below the mystery list; it should head the screen")
	}
}

// The heading's cross is the rosary's own, so the two cannot drift apart.
//
// Not a copy of the art: chooserHeading renders crossArt directly. Restyling the
// crucifix in cross.go restyles the opening screen with it, which is the point of
// testing this rather than just testing that "a cross is shown".
func TestTheChooserShowsTheRosarysOwnCross(t *testing.T) {
	m, _ := newModel("en", Sorrowful)
	screen := plain(chooser(m))

	for i, line := range crossArt {
		if !strings.Contains(screen, strings.TrimSpace(line)) {
			t.Errorf("row %d of the crucifix (%q) is not on the opening screen",
				i, strings.TrimSpace(line))
		}
	}
}

// The heading is centred and the choices are left-aligned.
//
// Two different alignments on purpose: a title centres, a list does not — a list
// needs a common left edge to scan down. Both blocks are padded to one width
// before joining, because JoinVertical centres line by line rather than block by
// block, which is how a ragged box happens.
func TestTheChooserCentresItsTitleAndAlignsItsChoices(t *testing.T) {
	m, _ := newModel("en", Sorrowful)
	rows := strings.Split(plain(chooser(m)), "\n")

	find := func(text string) (row int, col int) {
		for i, r := range rows {
			if k := strings.Index(r, text); k >= 0 {
				return i, lipgloss.Width(r[:k])
			}
		}
		return -1, -1
	}

	// Every row of the box is the same width, or the border is ragged.
	w := lipgloss.Width(rows[0])
	for i, r := range rows {
		if got := lipgloss.Width(r); got != w {
			t.Errorf("row %d is %d wide, want %d; the box is ragged", i, got, w)
		}
	}

	// The title is centred: the space left of it matches the space right of it,
	// within a column for odd widths.
	tRow, tCol := find(rosaryTitle)
	if tRow < 0 {
		t.Fatal("the title is not on the screen")
	}
	left := tCol
	right := w - tCol - lipgloss.Width(rosaryTitle)
	if diff := left - right; diff > 1 || diff < -1 {
		t.Errorf("the title sits %d columns from the left and %d from the right; it is not centred",
			left, right)
	}

	// The subtitle is centred under it too.
	sRow, sCol := find(rosarySubtitle)
	if sRow < 0 {
		t.Fatal("the subtitle is not on the screen")
	}
	if sRow <= tRow {
		t.Error("the subtitle is not below the title")
	}
	sLeft := sCol
	sRight := w - sCol - lipgloss.Width(rosarySubtitle)
	if diff := sLeft - sRight; diff > 1 || diff < -1 {
		t.Errorf("the subtitle is not centred: %d left, %d right", sLeft, sRight)
	}
}

// The selected set is marked by a character, not by colour alone, so the choice
// survives NO_COLOR and a screenshot.
func TestTheChooserMarksTheSelectionWithAGlyph(t *testing.T) {
	m := mustModel(t)

	// The HIGHLIGHTED set, which is whichever m.choice points at — not the set
	// newModel was handed. Those are different things: the argument is only a
	// default for the model to carry until a choice is made.
	screen := plain(chooser(m))
	if want := "▸ " + Sets()[m.choice].Name; !strings.Contains(screen, want) {
		t.Errorf("the selected set is not marked with %q:\n%s", want, screen)
	}

	// And only one set is marked, however many there are.
	if n := strings.Count(screen, "▸"); n != 1 {
		t.Errorf("%d sets are marked as selected, want exactly 1", n)
	}
}

// Every set is listed, so a new one in Sets() needs no other change to appear.
func TestTheChooserListsEverySet(t *testing.T) {
	screen := plain(chooser(mustModel(t)))

	for _, set := range Sets() {
		if !strings.Contains(screen, set.Name) {
			t.Errorf("the chooser does not list %q:\n%s", set.Name, screen)
		}
	}
	if len(Sets()) != 4 {
		t.Errorf("Sets() has %d entries; there are four sets of mysteries", len(Sets()))
	}
}

// Moving down the list changes which set is marked, and stops at the ends.
func TestTheChooserMovesThroughTheSets(t *testing.T) {
	m := mustModel(t)

	// Down through every set.
	for i := 1; i < len(Sets()); i++ {
		next, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
		m = next.(model)
		if m.choice != i {
			t.Fatalf("after %d presses of j the highlight is on %d, want %d", i, m.choice, i)
		}
	}

	// And no further: the list does not wrap, so holding the key cannot run off it.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	if got := next.(model); got.choice != len(Sets())-1 {
		t.Errorf("the highlight moved past the last set, to %d", got.choice)
	}

	// Back up to the top, and no further.
	for i := len(Sets()) - 2; i >= 0; i-- {
		next, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
		m = next.(model)
		if m.choice != i {
			t.Fatalf("moving up, the highlight is on %d, want %d", m.choice, i)
		}
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	if got := next.(model); got.choice != 0 {
		t.Errorf("the highlight moved above the first set, to %d", got.choice)
	}
}

// Whichever set is highlighted is the one prayed.
func TestTheChosenSetIsThePrayedSet(t *testing.T) {
	for i, want := range Sets() {
		m := mustModel(t)
		m.choice = i

		next, _ := m.Update(tea.KeyPressMsg{Code: ' '})
		got := next.(model)

		if got.set.Name != want.Name {
			t.Errorf("choice %d began %q, want %q", i, got.set.Name, want.Name)
		}
		// And the rosary shows that set's mysteries, not another's.
		got.cursor, got.say = 0, 0
		for j, b := range got.beads {
			if b.Announces() == 1 {
				got.cursor = j
				got.say = max(sayThatAnnounces(b), 0)
				break
			}
		}
		if mystery, _ := got.mystery(); mystery != want.Mysteries[0] {
			t.Errorf("choice %d contemplates %q, want %q", i, mystery, want.Mysteries[0])
		}
	}
}

func mustModel(t *testing.T) model {
	t.Helper()
	m, err := newModel("en", Sorrowful)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// Every set says when it is traditionally prayed, and the days are distinct.
//
// Distinct because the week divides between them: a set sharing another's days
// means one has been copied and not edited.
func TestEverySetSaysWhenItIsPrayed(t *testing.T) {
	seen := map[string]string{}
	for _, set := range Sets() {
		if set.Days == "" {
			t.Errorf("%q does not say when it is prayed", set.Name)
			continue
		}
		if other, clash := seen[set.Days]; clash {
			t.Errorf("%q and %q are both prayed on %q", other, set.Name, set.Days)
		}
		seen[set.Days] = set.Name
	}
}

// The chooser shows the days of the HIGHLIGHTED set, and only those.
//
// Only one set's days at a time: it is a note about the choice in front of you,
// not a table of all four to study.
func TestTheChooserShowsTheHighlightedSetsDays(t *testing.T) {
	for i, set := range Sets() {
		m := mustModel(t)
		m.choice = i

		screen := plain(chooser(m))

		if !strings.Contains(screen, set.Days) {
			t.Errorf("with %q highlighted the chooser does not say %q:\n%s",
				set.Name, set.Days, screen)
		}

		// No other set's days are shown — unless another set shares the text, which
		// TestEverySetSaysWhenItIsPrayed forbids.
		for j, other := range Sets() {
			if j == i {
				continue
			}
			if strings.Contains(screen, other.Days) {
				t.Errorf("with %q highlighted the chooser also shows %q's days (%q)",
					set.Name, other.Name, other.Days)
			}
		}
	}
}

// The days line sits below the list and above the keys: a caption on the choice,
// not an entry in it.
func TestTheDaysLineIsACaptionUnderTheList(t *testing.T) {
	m := mustModel(t)
	rows := strings.Split(plain(chooser(m)), "\n")

	lastSet, days, keys := -1, -1, -1
	for i, r := range rows {
		for _, set := range Sets() {
			if strings.Contains(r, set.Name) {
				lastSet = i
			}
		}
		if strings.Contains(r, Sets()[m.choice].Days) {
			days = i
		}
		if strings.Contains(r, "space to begin") {
			keys = i
		}
	}
	if lastSet < 0 || days < 0 || keys < 0 {
		t.Fatalf("list ends at %d, days at %d, keys at %d; expected all three",
			lastSet, days, keys)
	}
	if days <= lastSet {
		t.Error("the days line is inside the list of sets; it should sit below it")
	}
	if days >= keys {
		t.Error("the days line is below the keys; it belongs with the choice, above them")
	}
	// Not marked as selectable.
	if strings.Contains(rows[days], "▸") {
		t.Error("the days line carries the selection marker; it is not a choice")
	}
}

// A rule divides the title from the list of sets.
//
// Between them, not merely present: the whole job of the divider is to say the
// heading and the list are different things, which it only does from that one
// position.
func TestARuleDividesTheTitleFromTheList(t *testing.T) {
	m := mustModel(t)
	rows := strings.Split(plain(chooser(m)), "\n")

	title, rule, firstSet := -1, -1, -1
	for i, r := range rows {
		if strings.Contains(r, rosaryTitle) {
			title = i
		}
		if strings.Contains(r, chooserDivider) {
			rule = i
		}
		if firstSet < 0 && strings.Contains(r, Sets()[0].Name) {
			firstSet = i
		}
	}
	if rule < 0 {
		t.Fatalf("no divider on the opening screen:\n%s", strings.Join(rows, "\n"))
	}
	if title < 0 || firstSet < 0 {
		t.Fatalf("title at %d, first set at %d; expected both", title, firstSet)
	}
	if rule <= title {
		t.Error("the divider is above the title; it belongs between title and list")
	}
	if rule >= firstSet {
		t.Error("the divider is below the first set; it belongs between title and list")
	}

	// It FLOATS: clearly shorter than the content it divides, so it reads as a mark
	// between two parts of one screen rather than as a line splitting two panels.
	//
	// Measured against the list, not the box: the box's width includes the padding,
	// which would let the rule grow to nearly the full inner width and still pass.
	// Two thirds of the widest line is the bound — the current rule is 14 cells
	// against a 26-cell list, so there is room to retune without tripping it.
	widest := 0
	for _, set := range Sets() {
		if w := lipgloss.Width(set.Name) + 3; w > widest {
			widest = w
		}
	}
	if w := lipgloss.Width(chooserDivider); w*3 > widest*2 {
		t.Errorf("the divider is %d cells against a %d-cell list; it should be short "+
			"enough to float rather than span", w, widest)
	}
}
