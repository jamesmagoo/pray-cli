package rosary

// ─────────────────────────────────────────────────────────────────────────────
//  THE MYSTERIES — the other file to edit.
//
//  A set has one mystery per decade: five names, in order.
//
//  WHERE each is announced is not decided here — it is decided by Announcing(n, …)
//  in sequence.go, on whichever prayer should name it. This file only says what
//  the mysteries ARE.
//
//  To add a set, write it below and add it to Sets().
// ─────────────────────────────────────────────────────────────────────────────

// MysterySet is five mysteries, one for each decade, under a name.
type MysterySet struct {
	Name string

	// Days is when this set is traditionally prayed, as it should read on screen:
	// "Mondays and Saturdays". It is guidance, not a rule — any set may be prayed
	// on any day — so nothing enforces it and nothing selects by it; the chooser
	// just says it under whichever set you are looking at.
	Days string

	// Mysteries[0] is contemplated during the first decade, and so on. Five of
	// them, matching the five decades.
	Mysteries [5]string
}

// Joyful is prayed on Mondays and Saturdays, and in Advent.
var Joyful = MysterySet{
	Name: "The Joyful Mysteries",
	Days: "Mondays and Saturdays",
	Mysteries: [5]string{
		"The Annunciation",
		"The Visitation",
		"The Nativity",
		"The Presentation in the Temple",
		"The Finding in the Temple",
	},
}

// Sorrowful is prayed on Tuesdays and Fridays, and in Lent.
var Sorrowful = MysterySet{
	Name: "The Sorrowful Mysteries",
	Days: "Tuesdays and Fridays",
	Mysteries: [5]string{
		"The Agony in the Garden",
		"The Scourging at the Pillar",
		"The Crowning with Thorns",
		"The Carrying of the Cross",
		"The Crucifixion",
	},
}

// Glorious is prayed on Wednesdays and Sundays.
var Glorious = MysterySet{
	Name: "The Glorious Mysteries",
	Days: "Wednesdays and Sundays",
	Mysteries: [5]string{
		"The Resurrection",
		"The Ascension",
		"The Descent of the Holy Spirit",
		"The Assumption",
		"The Coronation of the Blessed Virgin Mary",
	},
}

// Luminous is prayed on Thursdays. The newest set: given by John Paul II in 2002,
// where the other three are centuries older.
var Luminous = MysterySet{
	Name: "The Luminous Mysteries",
	Days: "Thursdays",
	Mysteries: [5]string{
		"The Baptism in the Jordan",
		"The Wedding at Cana",
		"The Proclamation of the Kingdom",
		"The Transfiguration",
		"The Institution of the Eucharist",
	},
}

// Sets is every set the user can choose from.
//
// In the order they are traditionally listed — Joyful, Sorrowful, Glorious,
// Luminous — which is also the order the chooser shows them in. Not the order of
// the week: a list the eye already knows beats one that has to be read.
func Sets() []MysterySet {
	return []MysterySet{Joyful, Sorrowful, Glorious, Luminous}
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
