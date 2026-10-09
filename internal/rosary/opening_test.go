package rosary

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// opened builds a fresh model, on the opening, at the given window size.
func opened(t *testing.T, w, h int) model {
	t.Helper()
	m, err := newModel("en", Sorrowful)
	if err != nil {
		t.Fatal(err)
	}
	m.width, m.height = w, h
	return m
}

// The embedded art is real pictures: rectangular, mostly drawn, plain text, and
// the large one larger than the small.
//
// A bad file would not fail loudly — it would draw a blank, or a ragged picture
// that centres crookedly — so the shape is checked rather than assumed.
func TestThePicturesAreSound(t *testing.T) {
	if len(ourLady) < 2 {
		t.Fatalf("%d pictures; want a large and a small", len(ourLady))
	}
	for i, pic := range ourLady {
		rows := strings.Split(pic.text, "\n")
		if pic.w == 0 || pic.h != len(rows) {
			t.Fatalf("picture %d is %dx%d with %d rows", i, pic.w, pic.h, len(rows))
		}
		drawn := 0
		for y, row := range rows {
			if w := lipgloss.Width(row); w != pic.w {
				t.Errorf("picture %d row %d is %d wide, want %d", i, y, w, pic.w)
			}
			drawn += len([]rune(strings.ReplaceAll(row, " ", "")))
		}
		if drawn < pic.w*pic.h/3 {
			t.Errorf("picture %d has only %d drawn cells of %d", i, drawn, pic.w*pic.h)
		}
		// The colour is the app's to give: an escape in the art would fight the
		// gold, and survive the fade as a cell that never dims.
		if strings.Contains(pic.text, "\x1b") {
			t.Errorf("picture %d has escape sequences in it", i)
		}
		// U+2800 is the empty braille pattern, which many fonts draw as faint dots;
		// opening-art.sh turns it into a space.
		if strings.ContainsRune(pic.text, '\u2800') {
			t.Errorf("picture %d has empty braille cells, which some fonts draw as a grid", i)
		}
	}
	if ourLady[0].w <= ourLady[1].w || ourLady[0].h <= ourLady[1].h {
		t.Error("the pictures are not largest first")
	}
}

// The rosary begins with the opening, which runs its course and hands over to
// the chooser by itself.
func TestTheOpeningHandsOverToTheChooser(t *testing.T) {
	m := opened(t, 120, 50)
	if m.phase != opening {
		t.Fatalf("began in phase %v, want the opening", m.phase)
	}
	if m.Init() == nil {
		t.Fatal("Init started no clock; the opening would never end")
	}

	var cur tea.Model = m
	for i := 1; i <= openingFrames*2; i++ {
		next, cmd := cur.Update(frameMsg{})
		cur = next
		if next.(model).phase == choosing {
			if i != openingFrames {
				t.Errorf("handed over on frame %d, want frame %d", i, openingFrames)
			}
			return
		}
		if cmd == nil {
			t.Fatalf("frame %d returned no command; the opening stalled", i)
		}
	}
	t.Fatalf("the opening never handed over after %d frames", openingFrames*2)
}

// Any key skips to the chooser — and the key is spent on the skip.
//
// A space that skipped AND began would start a rosary in a set the user never saw
// highlighted.
func TestAnyKeySkipsTheOpening(t *testing.T) {
	for _, key := range []string{"space", "enter", "j", "left"} {
		got := press(t, opened(t, 120, 50), key)
		if got.phase != choosing {
			t.Errorf("%s left the opening in phase %v, want the chooser", key, got.phase)
		}
		if got.choice != 0 {
			t.Errorf("%s moved the chooser's highlight while skipping", key)
		}
	}
}

// A frame still in flight when a key skipped the opening must not disturb the
// chooser, nor keep a clock running.
func TestAStrayFrameAfterSkippingIsHarmless(t *testing.T) {
	m := press(t, opened(t, 120, 50), "space")

	next, cmd := m.Update(frameMsg{})
	if got := next.(model); got.phase != choosing {
		t.Errorf("a late frame moved the chooser to phase %v", got.phase)
	}
	if cmd != nil {
		t.Error("a late frame re-armed the clock on the chooser")
	}
}

// The words are on the screen once they have risen.
func TestTheOpeningSaysAveMaria(t *testing.T) {
	m := opened(t, 120, 50)
	m.open = openingFrames / 2 // held: everything risen, nothing fading yet

	screen := plain(render(m))
	for _, want := range []string{aveMaria, oraProNobis} {
		if !strings.Contains(screen, want) {
			t.Errorf("the opening does not say %q", want)
		}
	}
}

// The picture is the largest that fits the window, and none when none fits —
// the words are shown regardless, and nothing overflows.
func TestTheOpeningFitsTheWindow(t *testing.T) {
	large, small := ourLady[0], ourLady[1]

	for _, tc := range []struct {
		name string
		w, h int
		want *picture
	}{
		{"roomy", 120, 50, &large},
		{"middling", 80, 24, &small},
		{"cramped", 40, 10, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := opened(t, tc.w, tc.h)
			m.open = openingFrames / 2

			words := plain(render(m))
			pic, ok := m.pictureThatFits(aveMaria)
			switch {
			case tc.want == nil && ok:
				t.Errorf("drew a %dx%d picture in a %dx%d window", pic.w, pic.h, tc.w, tc.h)
			case tc.want != nil && (!ok || pic.h != tc.want.h):
				t.Errorf("drew the %d-row picture, want the %d-row one", pic.h, tc.want.h)
			}

			rows := strings.Split(words, "\n")
			if len(rows) > tc.h {
				t.Errorf("%d rows in a %d-row window", len(rows), tc.h)
			}
			if !strings.Contains(words, aveMaria) {
				t.Error("the words are missing")
			}
		})
	}
}

// The picture starts dark and ends dark: it rises out of the black and sinks
// back into it, so the chooser arrives on an empty screen rather than cutting.
func TestThePictureRisesAndSinks(t *testing.T) {
	if l := openingLevel(openingFrames, 0, 36); l != 0 {
		t.Errorf("the first frame is at brightness %v, want 0", l)
	}
	if l := openingLevel(openingFrames/2, 0, 36); l != 1 {
		t.Errorf("the held frame is at brightness %v, want 1", l)
	}
	if l := openingLevel(0, 0, 36); l != 0 {
		t.Errorf("the last frame is at brightness %v, want 0", l)
	}
}
