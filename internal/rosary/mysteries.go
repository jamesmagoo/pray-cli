package rosary

// ─────────────────────────────────────────────────────────────────────────────
//  THE MYSTERIES — the other file to edit.
//
//  A set has one mystery per decade: five names, in order.
//
//  WHERE each is announced is not decided here — it is decided by s.announce(n)
//  in sequence.go, on whichever bead should name it. This file only says what the
//  mysteries ARE.
//
//  To add a set, write it below and add it to Sets().
// ─────────────────────────────────────────────────────────────────────────────

// MysterySet is five mysteries, one for each decade, under a name.
type MysterySet struct {
	Name string
	// Mysteries[0] is contemplated during the first decade, and so on. Five of
	// them, matching the five decades.
	Mysteries [5]string
}

// Sorrowful is the set prayed on Tuesdays and Fridays, and in Lent.
var Sorrowful = MysterySet{
	Name: "The Sorrowful Mysteries",
	Mysteries: [5]string{
		"The Agony in the Garden",
		"The Scourging at the Pillar",
		"The Crowning with Thorns",
		"The Carrying of the Cross",
		"The Crucifixion",
	},
}

// Sets is every set the user can choose from. One for now; add more here and the
// chooser picks them up without further change.
func Sets() []MysterySet {
	return []MysterySet{Sorrowful}
}

// Mystery returns mystery n, 1-based, and whether there is one.
//
// n is whatever a bead announced. Zero means nothing has been announced yet — the
// pendant, before the first decade — and gets no mystery rather than a wrong one.
func (s MysterySet) Mystery(n int) (string, bool) {
	if n < 1 || n > len(s.Mysteries) {
		return "", false
	}
	return s.Mysteries[n-1], true
}
