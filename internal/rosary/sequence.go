package rosary

// ─────────────────────────────────────────────────────────────────────────────
//  THIS IS THE FILE TO EDIT.
//
//  Everything about which beads exist, in what order, and what is prayed on each
//  one is defined here, declaratively. Nothing else in the package needs to
//  change when you change the rosary.
//
//  A bead is one stop. Space moves to the next prayer; when a bead's prayers are
//  finished, space moves to the next bead.
//
//  Two ways to give a bead its words, and you can mix them freely:
//
//      Say("our-father")        a prayer from internal/prayers/data/
//      Text("Glory Be", "...")  words written inline, right here
//
//  Text exists so you are never blocked on a prayer not being in data/ yet.
//  Fill them in later and swap Text for Say; nothing else changes.
//
//  s.announce(n) marks the bead just added as declaring mystery n (1..5). The
//  mystery then shows until another bead announces the next one. The mysteries
//  themselves live in mysteries.go.
// ─────────────────────────────────────────────────────────────────────────────

// Pendant says how many of the beads below hang on the pendant — the chain from
// the crucifix up to the ring — rather than sitting on the ring itself.
//
// Count the beads at the top of Sequence() that belong on the chain: here the
// cross, the Creed, an Our Father and three Hail Marys.
func Pendant() int { return 6 }

// Sequence is the rosary, in the order prayed. Edit this.
func Sequence() []Bead {
	var s builder

	// ── The pendant ──────────────────────────────────────────────────────────
	s.add(Cross, "Sign of the Cross", Text("Sign of the Cross",
		"In the name of the Father,",
		"and of the Son,",
		"and of the Holy Spirit.",
		"",
		"Amen."))

	s.add(Large, "Apostles' Creed", Text("Apostles' Creed",
		"I believe in God,",
		"the Father almighty,",
		"Creator of heaven and earth…"))

	s.add(Large, "Our Father", Say("our-father"))

	// Three Hail Marys: faith, hope and charity.
	s.run(Small, 3, "Hail Mary", Say("hail-mary"))

	s.add(Link, "Glory Be", Text("Glory Be",
		"Glory be to the Father,",
		"and to the Son,",
		"and to the Holy Spirit.",
		"",
		"As it was in the beginning, is now,",
		"and ever shall be, world without end.",
		"",
		"Amen."))

	// ── Five decades ─────────────────────────────────────────────────────────
	for decade := 1; decade <= 5; decade++ {
		// The mystery is announced on the Our Father that opens the decade, and
		// stays on screen through the ten Hail Marys that follow. Move this
		// announce() call to a different bead and the mystery is named there
		// instead — that is the whole mechanism.
		s.add(Large, "Our Father", Say("our-father"))
		s.announce(decade)

		s.run(Small, 10, "Hail Mary", Say("hail-mary"))

		// A bead with more than one prayer: space steps through them in turn.
		s.add(Link, "Glory Be", Text("Glory Be",
			"Glory be to the Father,",
			"and to the Son,",
			"and to the Holy Spirit.",
			"",
			"Amen."),
			Text("Fatima Prayer",
				"O my Jesus, forgive us our sins,",
				"save us from the fires of hell,",
				"and lead all souls to Heaven,",
				"especially those in most need of Thy mercy."))
	}

	// ── The close ────────────────────────────────────────────────────────────
	s.add(Large, "Hail Holy Queen", Text("Hail Holy Queen",
		"Hail, Holy Queen, Mother of Mercy,",
		"our life, our sweetness and our hope."))

	return s.beads
}
