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
// crucifix, one small bead (Creed and Our Father), and three small beads.
func Pendant() int { return 5 }

// Prayers said at more than one place in the rosary, defined once.
var (
	gloryBe = Text("Glory Be",
		"Glory be to the Father,",
		"and to the Son,",
		"and to the Holy Spirit.",
		"",
		"As it was in the beginning, is now,",
		"and ever shall be, world without end.",
		"",
		"Amen.")

	hailHolyQueen = Text("Hail Holy Queen",
		"Hail, Holy Queen, Mother of Mercy,",
		"our life, our sweetness and our hope.")

	fatima = Text("Fatima Prayer",
		"O my Jesus, forgive us our sins,",
		"save us from the fires of hell,",
		"and lead all souls to Heaven,",
		"especially those in most need of Thy mercy.")
)

// PendantGapAfter says whether a blank row of chain follows pendant bead i,
// counting from the crucifix (bead 0).
//
// The pendant is:
//
//	crucifix — bead — GAP — three beads — GAP — the ring
//
// so there is a gap after the crucifix's bead (index 1) and after the last of the
// three Hail Marys (index 4).
func PendantGapAfter(i int) bool {
	return i == 1 || i == 4
}

// Sequence is the rosary, in the order prayed. Edit this.
func Sequence() []Bead {
	var s builder

	// ── The pendant ──────────────────────────────────────────────────────────
	//
	// crucifix — one small bead — gap — three small beads — gap — the ring, whose
	// first big bead is where the loop begins.

	// The crucifix: the Sign of the Cross, then the Apostles' Creed, both said
	// while holding it.
	s.add(Cross, "Sign of the Cross",
		Text("Sign of the Cross",
			"In the name of the Father,",
			"and of the Son,",
			"and of the Holy Spirit.",
			"",
			"Amen."),
		Text("Apostles' Creed",
			"I believe in God,",
			"the Father almighty,",
			"Creator of heaven and earth…"))

	// The first bead of the chain: the Our Father, and nothing else.
	s.add(Small, "Our Father", Say("our-father"))

	// Three Hail Marys: faith, hope and charity.
	s.run(Small, 3, "Hail Mary", Say("hail-mary"))

	// ── Five decades ─────────────────────────────────────────────────────────
	//
	// Each decade is ONE big bead followed by ten small ones.
	//
	// The big bead carries every prayer said at that junction: the Glory Be and
	// Fatima Prayer closing the decade before, then the Our Father opening this
	// one. On a real rosary that is a single bead held once while all three are
	// prayed — adding a separate bead for the Glory Be put two big beads side by
	// side, which is not what the object looks like.
	for decade := 1; decade <= 5; decade++ {
		junction := []Words{}

		// What closes the decade before. The first decade follows the pendant,
		// whose three Hail Marys are closed by a Glory Be alone; later decades are
		// closed by the Glory Be and the Fatima Prayer.
		if decade == 1 {
			junction = append(junction, gloryBe)
		} else {
			junction = append(junction, gloryBe, fatima)
		}
		// The mystery is announced on the OUR FATHER, after the Glory Be and Fatima
		// Prayer that close the decade before. Wrapping a different prayer in
		// Announcing() names the mystery there instead — that is the whole
		// mechanism.
		junction = append(junction, Announcing(decade, Say("our-father")))

		s.add(Large, "Our Father", junction...)

		s.run(Small, 10, "Hail Mary", Say("hail-mary"))
	}

	// ── The close ────────────────────────────────────────────────────────────
	//
	// The rosary ends where it began: the fingers come round the loop and arrive
	// back at the first big bead.
	//
	// That bead is therefore visited TWICE — once to open (Glory Be, Our Father)
	// and once to close (Glory Be, Fatima, Hail Holy Queen). A flat sequence cannot
	// revisit a bead, so this is a second bead at the SAME POSITION: it draws as
	// one bead on screen, and you pray each visit's prayers separately rather than
	// all five at once.
	//
	// samePlaceAs(Pendant()) is what pins it to the same cell. It must NOT be given
	// its own place on the ring, or there would be six big beads.
	s.addAt(Pendant(), Large, "Hail Holy Queen", gloryBe, fatima, hailHolyQueen)

	return s.beads
}
