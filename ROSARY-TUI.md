# The rosary TUI: concepts and implementation

Everything you need to understand `internal/rosary`, from how a terminal
actually works up to why each function in this package is shaped the way it is.

This document is in two halves. **Part I** is the concepts — they apply to any
terminal UI you ever write, in any language. **Part II** is this
implementation, function by function, with the reasoning and the known bugs.

Read Part I once. Part II is a reference to come back to.

---

## If you only read one thing

**To change the rosary — prayers, order, how many beads — edit
`internal/rosary/sequence.go`. To change the mysteries, edit
`internal/rosary/mysteries.go`. Nothing else.**

```go
s.add(Large, "Our Father",
    gloryBe, fatima,                            // two prayers, one bead
    Announcing(decade, Say("our-father")))      // the mystery changes here
s.run(Small, 10, "Hail Mary", Say("hail-mary")) // ten of them
```

- `Say("id")` takes the prayer from `internal/prayers/data/<id>/`
- `Text("Title", "line", "line")` is words written inline, for anything not in
  `data/` yet
- Give a bead several prayers and **space** steps through them before moving on
- `Announcing(n, prayer)` marks that **prayer** as declaring mystery `n` (1–5)

The ring draws itself around whatever you define. You do not need to touch the
geometry, the colours, or the drawing code.

**For anything else — bead glyphs, ring size and shape, the current-step
indicator — see [Part IV: how to change things](#part-iv-how-to-change-things).**
It is a recipe per question, with the constraints that will bite you.

| Want to change | File | Knob |
|---|---|---|
| Bead glyphs / "size" | `beads.go` | `Kind.Glyph()` |
| Bead colours | `view.go` | `beadDim`, `beadLarge`, `crossBead`, `beadCurrent` |
| Bead order, prayers | `sequence.go` | `Sequence()`, `Pendant()` |
| Pendant gaps | `sequence.go` | `PendantGapAfter(i)` |
| Mysteries | `mysteries.go` | the set, and `Announcing(n, …)` for where |
| Ring roundness | `ring.go` | `rx := ry * 2` |
| Ring tightness | `ring.go` | `const gutter = 3` |
| Start / direction | `ring.go` | `ringStart`, `ringArc` |
| Step indicator | `animate.go` | `glowFrames`, `bloom`, `glowRamp` |
| Fade-in speed | `animate.go` | `fadeFrames`, `frameRate` |
| Fade-out on exit | `animate.go` | `departFrames`, `departRamp` |
| Farewell words | `view.go` | `farewellWords` |
| Keys | `model.go` | the `tea.KeyPressMsg` switch in `Update` |

### The keys

While praying:

| Key | Does |
|---|---|
| `space` (or `enter`, `→`, `l`) | next prayer; at a bead's last prayer, the next bead; on the **last** prayer, finishes |
| `←` (or `h`, `backspace`) | back one prayer |
| `x` | finish — fades out, then quits |
| `esc`, `ctrl+c` | quit immediately, no farewell |

On the chooser: `↑`/`↓` pick the mysteries, `space` begins.

On the closing screen: `space` prays again (back to the chooser), `←` steps back
into the rosary, `x` leaves.

`x` is the considered exit and gets the farewell; `esc` and `ctrl+c` are escape
hatches and skip it. Someone reaching for either wants out *now*, and a hatch that
takes 1.2 seconds to open is not one.

Note that `space` means "go on" on all three screens — the chooser begins, the
rosary advances, the close starts another. That is deliberate: it is the key the
thumb finds without looking, and the one moment not to require a new gesture is
the end.

**Space is `"space"`, not `" "`.** See Part I §6 — matching the wrong one fails
silently.

---

# Part I: the concepts

## 1. The terminal is a grid of cells

A terminal is not a canvas of pixels. It is a grid of **cells**, each holding
one character. A "80×24 terminal" is 80 columns by 24 rows — 1,920 cells.

Everything else follows from this single fact:

- You cannot draw a diagonal line. You can only choose characters that
  *suggest* one.
- You cannot place something at half a column. Positions are integers.
- A circle is an approximation: cells lit in a roughly circular arrangement.

The rosary's ring is a set of cells chosen to look circular. There is no
circle-drawing primitive, and there never will be.

### Cells are taller than they are wide

This is the single most important geometric fact for terminal drawing, and the
one most likely to catch you out.

A typical terminal cell is about **twice as tall as it is wide** (roughly 8×16
or 9×18 pixels, depending on font). So:

```
A "circle" of radius 10 in cell units:          What you actually see:

        ○ ○ ○                                        ○ ○ ○
      ○       ○                                    ○       ○
    ○           ○                                 ○         ○      <- squashed
    ○           ○                                  ○       ○          vertically
      ○       ○                                      ○ ○ ○
        ○ ○ ○
       (what you asked for)                      (a tall ellipse)
```

To get something that *looks* circular you must make the horizontal radius
roughly **twice** the vertical radius. In this code that is why `rx` is always
much larger than `ry` (`ring.go`): for the Our Father bead, `rx=39` and
`ry=11`. Those are not a circle's radii. They are an ellipse in cell units that
reads as a circle to the eye.

> **Rule of thumb:** for a visual circle, `rx ≈ 2 × ry`.

## 2. Characters are not all one cell wide

You will assume one character = one column. That assumption is wrong, and it
will break your layout in ways that look baffling.

Three categories matter:

| Category | Examples | Width in cells |
|---|---|---|
| ASCII, most symbols | `A` `○` `●` `◦` `✠` | 1 |
| East Asian, most emoji | `漢` `⚪` `⚫` `🙏` | **2** |
| Combining marks, zero-width | `◌́`, ZWJ | **0** |

This is defined by the Unicode standard (East Asian Width, UAX #11), not by
your font, though fonts and terminals disagree in practice at the edges.

**This decided the bead glyphs in this project.** Before designing the ring I
measured candidates:

```
BULLET U+2022            "•"  width=1
WHITE CIRCLE U+25CB      "○"  width=1    <- used for Hail Mary beads
BLACK CIRCLE U+25CF      "●"  width=1    <- used for Our Father beads
MED WHITE CIRCLE U+26AA  "⚪"  width=2    <- rejected
MED BLACK CIRCLE U+26AB  "⚫"  width=2    <- rejected
SMALL WHITE CIRCLE U+25E6 "◦" width=1    <- used for chain links
CROSS U+2720             "✠"  width=1    <- used for the crucifix
```

`⚪` and `⚫` are the obvious, prettier choice — they look like beads. They are
also 2 cells wide, so every one of them would shift the rest of its row one
column right, shearing the ring. That is why the beads in this project look
understated: correctness won.

**Never use `len()` on a string to get its display width.** `len("○")` is 3
(it is three UTF-8 bytes). `len([]rune("○"))` is 1, which is right here but
wrong for `⚪`. Use a width-aware function — in this codebase,
`lipgloss.Width()`.

## 3. ANSI escape sequences: how colour works

Colour is not a property of a cell you set through an API. You send the
terminal **in-band control codes** mixed into the text stream.

A styled bead is really this byte sequence:

```
\x1b[2;38;2;201;162;39m ○ \x1b[m
└─────────────────────┘ │  └────┘
   "dim, fg = RGB         │   "reset
    201,162,39"           │    everything"
                    the actual character
```

- `\x1b` is ESC (27). `\x1b[` starts a Control Sequence Introducer.
- `38;2;R;G;B` sets a 24-bit foreground colour. `2` means dim/faint.
- `m` ends the sequence. `\x1b[m` resets.

Measured in this codebase:

```
plain bead bytes=3   styled bead bytes=26   display width both=1
```

**This is the crux of terminal rendering.** 26 bytes, 1 column. The string
length and the visual width have nothing to do with each other. Every layout
bug you hit in a TUI traces back to confusing the two.

### The consequence: style last, measure first

Because escapes are invisible-but-present, you cannot do arithmetic on styled
text:

```go
// WRONG — the escape bytes are in the cell, so column maths is now nonsense
grid[y][x] = styled("○")   // "cell" is now 26 bytes, not 1 column
columnsUsed := len(row)    // meaningless

// RIGHT — plain runes while computing geometry, colour afterwards
grid[y][x] = '○'           // exactly 1 cell
...compute everything...
output = applyColours(grid) // escapes added only on the way out
```

This is precisely the two-pass split in this package: `drawRosary` places plain
runes, then `styleGrid` adds colour. It is not stylistic tidiness; getting it
backwards breaks the ring.

### `NO_COLOR` and degradation

Not every terminal does 24-bit colour, and users may set `NO_COLOR=1`. Lip Gloss
detects the terminal's capability and downgrades (24-bit → 256 → 16 → none)
automatically.

A design implication: **never encode meaning in colour alone.** If the only thing
marking the current bead were its colour, a `NO_COLOR` user would lose their
place entirely.

`Reverse(true)` (swapping foreground and background) is the usual answer, and it
does survive colour stripping — but it fills the **whole cell**, so a one-cell
bead becomes a solid rectangle sitting on the string. It reads as a text cursor,
not as a lit bead.

This package instead marks the current bead by **glyph**: a halo, `◎`, where the
others are `○`, plus bold. Both survive `NO_COLOR` — a different character is a
different character no matter what the terminal supports — and it looks like a
bead rather than a block.

> **The general rule:** encode state in *shape* first, weight (bold) second,
> colour last. Shape degrades to nothing.

## 4. The alternate screen

Terminals keep two buffers:

- the **main** buffer, with your scrollback history
- the **alternate** buffer: one screen, no scrollback

Full-screen programs (`vim`, `less`, `htop`) switch to the alternate buffer,
draw there, and switch back on exit — which is why your prompt reappears
untouched, with no trace of the program in the scrollback.

In Bubble Tea v2 this is a field on the view (see §6), and the rosary sets it:
the prayer fills the window and leaves nothing behind in your shell history.

## 5. Grid-based drawing (the technique this package uses)

For anything geometric — rings, boxes, diagrams, charts — build a 2-D array of
runes, place glyphs by coordinate, then join rows into a string.

```go
cells := make([][]rune, height)        // allocate the grid, filled with spaces
cells[y][x] = '○'                      // place by coordinate
strings.Join(rowsAsStrings, "\n")      // flatten once, at the end
```

**Why this beats string concatenation.** With a grid, the ring and the text
each know only their own coordinates; neither needs to know the other's layout,
and they compose automatically because they write into the same array. Building
the same picture by assembling strings means every row must know about every
element that appears on it — and the moment one element moves, every row's
padding changes.

The trade-off is that a grid cannot express "centre this relative to that".
Grids are for absolute positioning; use a layout library (§7) for relative.

## 6. Bubble Tea: the Elm architecture

Bubble Tea is a direct port of the Elm architecture. All state is in one value
and the only way it changes is a message arriving.

```
      ┌──────────────── tea.Msg ───────────────┐
      │   (keypress, resize, timer, I/O result) │
      ↓                                         │
   Update(msg) ──→ returns (newModel, Cmd) ──────┘
      │                              │
      ↓                              └─→ runtime runs the Cmd,
    View()                               its result returns as a Msg
      │
      ↓
  text on screen
```

Three methods you implement; the runtime owns the loop:

| Method | When | Returns |
|---|---|---|
| `Init()` | once, before the first draw | a `Cmd`, or nil |
| `Update(msg)` | every message | the next model, and a `Cmd` |
| `View()` | after every `Update` | the state as text |

Rules that matter:

1. **`Update` is the only place state changes.** It takes a model and returns a
   new one. The model is a *value*, so each state is independent — no shared
   mutable state to reason about.
2. **`View` must be pure.** No I/O, no clock, no randomness. Same model in,
   same text out. This is what makes a TUI testable: you can assert on frames
   without a terminal.
3. **Never `os.Exit` inside a program.** Return the `tea.Quit` command so the
   runtime restores the terminal. Exiting directly leaves the terminal in raw
   mode with the alternate screen active — a wedged shell.
4. **A `Cmd` is a function returning a `Msg`.** It is how side effects happen:
   you don't do the work, you hand the runtime a function and its result comes
   back as a message. Keeps `Update` pure and testable.

### v1 vs v2 — three differences that will trip you up

This project uses **v2** (`charm.land/bubbletea/v2`). Nearly every tutorial
online is v1 (`github.com/charmbracelet/bubbletea`). The differences:

| | v1 | v2 |
|---|---|---|
| View returns | `string` | `tea.View` (a struct) |
| Key message | `tea.KeyMsg` | `tea.KeyPressMsg` (press/release split) |
| Alt screen | `tea.WithAltScreen()` program option | `view.AltScreen = true` field |

### The space bar is `"space"`, not `" "`

A v2 trap that cost a real bug here. `KeyPressMsg.String()` for the space bar
returns **`"space"`**:

```go
case " ":       // never fires — the key silently does nothing
case "space":   // correct
```

Nothing in the build catches this: it compiles, it runs, and the key just does
not work. The lesson is that **key handling needs a test**, because a missed case
is invisible rather than loud:

```go
next, cmd := m.Update(tea.KeyPressMsg{Code: ' '})
if next.(model).cursor == m.cursor { t.Error("space did not advance") }
```

The alt-screen change is the interesting one. In v1 full-window mode was fixed
at startup. In v2 it is a field on the thing you return *every frame*, so the
model owns it and can change its mind — a program can move in and out of
full-screen as its state changes. `tea.WithAltScreen` does not exist in v2.

## 7. Lip Gloss: styling and layout

Lip Gloss does two separate jobs. Keep them distinct in your head.

**Styling** — produce styled strings:

```go
style := lipgloss.NewStyle().Foreground(lipgloss.Color("#C9A227")).Bold(true)
style.Render("○")   // returns the string with escapes wrapped around it
```

A `Style` is a value and every setter returns a copy, so styles are safe to
share, build once at package level, and reuse.

**Layout** — position blocks of text:

| Function | Does |
|---|---|
| `lipgloss.Width(s)` | display width in cells, ignoring escapes |
| `lipgloss.Place(w, h, hPos, vPos, s)` | position a block in a w×h box |
| `lipgloss.JoinVertical(pos, ...)` | stack blocks, aligned |
| `lipgloss.JoinHorizontal(pos, ...)` | place blocks side by side |

### The `JoinVertical` trap

This cost a real bug during this build and is worth internalising.

`JoinVertical(lipgloss.Center, blocks...)` centres **line by line, not block by
block.** Hand it a 9-line prayer and it centres each of those nine lines
independently against the widest — so left-aligned text comes out as a ragged
diamond:

```
Intended (block centred,          Actual (each line centred
left edge preserved):             independently):

    Hail Mary, full of grace,        Hail Mary, full of grace,
    the Lord is with thee.            the Lord is with thee.
    Blessed art thou amongst...     Blessed art thou amongst...
    and blessed is the fruit...     and blessed is the fruit of...
```

**The fix:** pad the multi-line block to its own width *first*, so all its lines
are equal length and there is nothing left to shift:

```go
text = lipgloss.NewStyle().Width(lipgloss.Width(text)).Render(text)
```

(The current ring implementation sidesteps this entirely by placing text on the
grid by coordinate — but you will meet it the moment you use `JoinVertical`.)

## 8. Trigonometry for cell grids

To place `n` beads around an ellipse:

```go
angle := startAngle + arcSpan*float64(i)/float64(n-1)
x := cx + int(math.Round(float64(rx) * math.Cos(angle)))
y := cy - int(math.Round(float64(ry) * math.Sin(angle)))
//      ↑ MINUS
```

Two things to get right:

**1. The y-axis is flipped.** Screen rows grow *downward*; mathematical y grows
*upward*. So `y` uses `-` where you would expect `+`. Forget this and your
shape is mirrored vertically.

**2. Angles.** `0` is east (right), and angles increase counter-clockwise.
`-π/2` is straight down. A **negative** arc span makes the sequence run
clockwise; a positive one, counter-clockwise.

In this package the beads must run away from the crucifix (the direction fingers
travel), which is counter-clockwise from the bottom of the loop — hence
`ringStart = -π/2` (straight down) and a negative `ringArc`. Getting the sign
wrong was one of the real bugs in this build: the decade ran backwards.

**3. Round, don't truncate.** `int(x)` truncates toward zero, which biases
every position and visibly distorts the shape. `int(math.Round(x))` is correct.

---

# Part II: how this implementation works

## File map

| File | Responsibility |
|---|---|
| **`sequence.go`** | **The rosary itself — the one file you edit to change prayers or order** |
| **`mysteries.go`** | **The mystery sets — what the mysteries ARE** |
| `beads.go` | The *types* behind it: `Bead`, `Words`, `Say`, `Text`, `builder` |
| `ring.go` | *Geometry*: where each bead goes; places plain runes on a grid |
| `style_grid.go` | *Colour*: second pass, adds escapes to the finished grid |
| `view.go` | *Content*: the mystery inside the ring, the prayer panel beside it, colours |
| `animate.go` | *Time*: the bead's flare, the prayer's fade-in, the farewell's fade-out |
| `model.go` | *State and the loop*: the Bubble Tea `Model` |
| `phase.go` | Which of the three screens is showing: chooser, rosary, close |
| `cross.go` | The crucifix: line-drawn art, and the cells it occupies |
| `run.go` | Entry point; the only thing `internal/cli` touches |

The ordering is deliberate: structure → geometry → colour. Each layer depends
only on the ones above it.

## The pipeline for one frame

```
model (cursor=3)
    │
    ├─ contentLines(m) ───────→ []string, PLAIN text
    │     "Hail Mary 3 of 10"     (title, prayer, status line)
    │     "Hail Mary, full of..."
    │
    ├─ drawRosary(beads, cursor, lines)
    │     │
    │     ├─ measure textW (widest line) and textH (line count)
    │     ├─ layout(n, textW, textH) ──→ ringGeometry: rx, ry, cx, cy, pos[]
    │     │     └─ grows rx until clears() is true
    │     ├─ newCanvas(w, h)            grid of spaces
    │     ├─ place bead runes at pos[]  PLAIN runes
    │     ├─ place pendant runes
    │     ├─ place text lines
    │     └─ styleGrid(...) ──────────→ string WITH escapes
    │
    └─ lipgloss.Place(width, height, ...) ──→ centred in the window
```

Note where plain text becomes styled: only in `styleGrid`, the last step before
the string leaves. Everything upstream is measurable.

## `sequence.go` — the rosary (edit this one)

Everything about which beads exist, in what order, and what is prayed on each is
in one file, written as a description rather than as code:

```go
func Sequence() []Bead {
    var s builder

    s.add(Cross, "Sign of the Cross", Text("Sign of the Cross",
        "In the name of the Father,",
        "and of the Son,",
        "and of the Holy Spirit."))

    s.add(Large, "Our Father", Say("our-father"))
    s.run(Small, 3, "Hail Mary", Say("hail-mary"))   // three identical beads

    return s.beads
}
```

Four things to know, and that's the whole API:

| | Does |
|---|---|
| `s.add(kind, name, prayers...)` | one bead |
| `s.run(kind, n, name, prayers...)` | `n` identical beads, numbered "3 of 10" |
| `Say("our-father")` | text from `internal/prayers/data/<id>/` |
| `Text("Glory Be", "line", "line")` | text written inline, right here |

### Why both `Say` and `Text`

`Say` pulls from `internal/prayers/data/`, which is where prayers belong. But
only three prayers are in there, and a rosary needs the Glory Be, the Creed, the
Fatima Prayer and Hail Holy Queen.

`Text` means **you are never blocked**. Write the words inline now; when a prayer
lands in `data/`, swap `Text(...)` for `Say("glory-be")` and nothing else
changes. Both produce the same `Words` value, so the rest of the package cannot
tell them apart.

### Several prayers on one bead

A bead takes as many prayers as you give it, and space steps through them in
turn before moving to the next bead:

```go
s.add(Large, "Our Father",
    gloryBe, fatima,                              // closing the decade before
    Announcing(decade, Say("our-father")))        // opening this one
```

That is two presses of space on one bead.

### How it is resolved

`Sequence()` is a plain declaration — no file reading, no errors. Every `Say` is
turned into real text once, in `newModel`, at startup:

```go
for i, b := range beads {
    for j, w := range b.Says {
        resolved, err := w.resolve(lang)   // reads data/ only for Say()
        ...
    }
}
```

So a wrong prayer id is an ordinary CLI error before the screen clears, never a
blank bead mid-prayer, and no file is read while you are praying.

## Mysteries — announced, not derived

A set is five names, one per decade, in `mysteries.go`:

```go
var Sorrowful = MysterySet{
    Name: "The Sorrowful Mysteries",
    Mysteries: [5]string{"The Agony in the Garden", ...},
}

func Sets() []MysterySet { return []MysterySet{Sorrowful} }
```

### Where a mystery is named is a declaration

The obvious implementation is to work out the mystery from the bead's position —
"bead 8 is in decade 1, so mystery 1". That is wrong, because it buries a
liturgical decision inside loop arithmetic: change how `Sequence()` is written and
the mysteries move with it.

Instead a **prayer** announces a mystery, explicitly, in `sequence.go`:

```go
s.add(Large, "Our Father",
    gloryBe, fatima,                            // closing the decade before
    Announcing(decade, Say("our-father")))      // ← the mystery changes HERE
s.run(Small, 10, "Hail Mary", Say("hail-mary"))
```

Wrap a different prayer in `Announcing()` and the mystery is named there instead.
That is the entire mechanism, and it is the one knob you need.

**Why the prayer and not the bead.** It was on the bead first, which changed the
mystery the moment you arrived — so the Glory Be and Fatima Prayer, which *close
the previous decade*, already showed the next decade's mystery. A junction bead
spans two decades, so a per-bead announcement cannot express when the change
happens. Per-prayer can.

### How it persists

`model.announced()` looks **backwards** from the cursor for the most recent
announcement — over prayers, not beads, and on the current bead only as far as the
prayer being said:

```go
for i := m.cursor; i >= 0; i-- {
    says := m.beads[i].Says
    last := len(says) - 1
    if i == m.cursor {
        last = min(m.say, last)   // a later announcement on THIS bead is not yet reached
    }
    for j := last; j >= 0; j-- {
        if n := says[j].Announces; n > 0 {
            return n
        }
    }
}
return 0   // before the first announcement
```

That `min(m.say, last)` is the whole point: standing on the Glory Be of a junction
bead, the Our Father two prayers later has not been prayed, so its announcement
must not count yet.

So a mystery is announced once and stays in force until another bead announces
the next. There is nothing to clear, no "current mystery" field to keep in sync,
and no way for the display and the position to disagree — the beads are the only
source of truth.

Before the first announcement (the pendant prayers) there is no mystery, and the
box is not drawn, rather than showing a wrong one.

### The mystery is inside the ring; the prayer sits beside it

```go
inside := mysteryLines(m.set, m.announced())          // in the middle of the ring
rosary := drawRosary(m.ring, m.beads, ..., inside)

block := lipgloss.JoinHorizontal(lipgloss.Center,     // prayer to the RIGHT
    rosary, gutterCols, prayerPanel(m))
```

The ring encloses the mystery — a short label — and the prayer is joined to its
right as a separate block. To move the prayer, change that one `JoinHorizontal`:
the ring does not know the prayer exists.

**What this changed about sizing.** The ring used to be sized to the longest
*prayer* (44 columns). Sizing it to a *mystery* (27 columns) made the contents stop
being the binding constraint — the ring's size is now set almost entirely by the
beads needing somewhere to sit. Two consequences followed:

- **Grow on a fixed 2:1 ratio.** The old loop grew `rx` three times faster than
  `ry`, which with small contents produced a wide flat hoop. Growing `rx = 2*ry`
  keeps it round (§1).
- **Beads may touch, but not collide.** The old rule demanded a blank cell between
  neighbours, which with 62 ring beads forced an 81×48 frame around a 27-column
  label — a vast empty hoop. The rule is now only that no two beads share a cell
  (which would silently lose one), and the frame is **41×28**. On a real rosary the
  beads touch anyway.

The prayer, being outside the ring, is styled normally rather than written onto
the grid — escapes in it cannot disturb any cell arithmetic, because it never
touches a cell.

### Reserve a fixed width, or the layout walks

The prayer panel has an explicit width:

```go
return lipgloss.NewStyle().Width(m.panelW).MaxWidth(m.panelW).Render(...)
```

Without it the panel is as wide as whatever prayer is showing, so the joined block
changes width, `Place` re-centres it, and **the whole rosary slides sideways on
every step.** Measured: the panel swung between 26 and 44 columns and the ring
moved 9 columns left and right.

`panelW` is the widest prayer in the whole rosary, computed once in `newModel` —
the same approach as the fixed ring, for the same reason. Short prayers leave empty
space on the right, which is the price of nothing moving.

> **The general rule for TUIs:** anything whose content changes between frames
> needs a reserved size, or it drags its neighbours around. Centring amplifies it:
> a one-column change in content moves the block half a column, and everything
> shifts.

### No box inside the ring

The mystery was briefly given a bordered box. It did not work: the ring then had
to be large enough to clear the border, and the beads ended up **drawn on top of
the border** — two frames competing for the same space, with the beads punching
holes through the box's edge.

The ring *is* the frame. The mystery sits inside it with styling but no border of
its own: the set's name quiet, the mystery itself brighter, since that is the thing
being contemplated.

(The chooser still uses a box — it is a panel on an otherwise empty screen, with
no ring to conflict with.)

### Three screens, one program

The chooser, the rosary and the close are all *phases of the same model* — not
separate programs, not separate Bubble Tea instances:

```go
type phase int

const (
	choosing phase = iota
	atPrayer
	finished
	departing
)
```

`Update` switches on it and each phase gets its own key handler, returning early,
so the phases share no key logic. That matters because `space` means something
different on each screen, and one switch trying to serve all three is how a key
ends up meaning two things at once.

**Why a `phase` and not two booleans.** The first version had `choosing bool`.
Adding the close would make it `choosing bool, done bool` — four states, one of
which (`choosing && done`) is nonsense, and every handler would have to check both
in the right order. One `phase` has exactly the three states that exist, so the
impossible screen cannot be represented.

The rosary is built once, behind the chooser, so beginning is instant — and
*beginning again* needs no reload at all:

```go
func (m *model) begin(set MysterySet) tea.Cmd {
	m.set = set
	m.cursor = 0
	m.say = 0
	m.phase = atPrayer
	return m.startStep()
}
```

Note what it does **not** reset: `beads`, `ring`, `panelW`. Those are the rosary
as a physical object, measured once in `newModel` — the same on the second time
through as on the first.

Resetting the cursor is not cosmetic. `announced()` finds the current mystery by
looking **back** from the cursor, so a stale cursor would carry the fifth mystery
into the new rosary's opening prayers. `TestBeginResetsTheRosary` calls `begin`
directly for exactly this reason: going through the chooser hides the bug, because
the chooser resets the cursor itself.

Adding Joyful and Glorious needs no new code: write them in `mysteries.go`, add
them to `Sets()`, and the chooser lists them.

### The close: the transition is an absence

Reaching the last prayer used to do nothing — `next()` returned false and the key
was swallowed, leaving the user on the Hail Holy Queen with no sign they had
finished. Now `space` there moves to the `finished` phase.

The closing screen draws **the same ring, with no bead current**:

```go
ring := drawRosary(m.ring, m.beads, -1, 0, 0, finishedLines())
```

`cursor = -1` is what lifts the cursor off: no bead index can equal it, so no halo
is drawn and every bead sits in its resting colour. Through the whole rosary
exactly one bead was lit, so a ring with *nothing* lit reads immediately as "no
longer in progress" — the same object, at rest. That is the whole transition, and
it needs no second drawing path that could drift out of agreement with the ring
just prayed.

`drawRosary` needed one guard for this, since it indexed `g.pos[cursor]` to resolve
beads sharing a cell:

```go
onBead := cursor >= 0 && cursor < len(beads)
```

The mystery is cleared from the middle of the ring and replaced with `Amen.` —
leaving the fifth mystery there would read as still being on it.

The closing screen also animates in using the same counters, so arriving at the
end feels like one more step rather than a different program taking over.

### The farewell: an animation that must outlive its own trigger

`x` does not quit. It starts a fade-out, and the program quits itself when the
fade reaches black.

This is the one animation here whose **end does something** rather than merely
stopping, and that makes it structurally different from the other three. Bubble
Tea has no "quit when this is done" command — `tea.Quit` tears the program down on
the spot — so leaving has to be a *phase*:

```go
func (m *model) leave() tea.Cmd {
	m.phase = departing
	m.depart = departFrames
	return tick()
}
```

and the frame handler ends it:

```go
if m.phase == departing {
	m.depart--
	if m.depart <= 0 {
		return m, tea.Quit   // the last frame
	}
	return m, tick()
}
```

**The question that had to be answered first:** does the runtime call `View` for
the frame whose `Update` returns `tea.Quit`? If not, the fade would be cut off one
frame early and the last thing on screen would be a half-faded word. I probed it
with a throwaway program that logged every `View` call:

```
VIEW CALLED n=4
VIEW CALLED n=5
VIEW CALLED n=6   ← the frame that returned tea.Quit
program returned cleanly
```

It does. Worth knowing in general: **a self-quitting animation's final frame is
drawn.**

Note the `departing` case sits *above* the shared counter logic and returns early.
It must not fall through to the "stop if nothing is running" test, which would
leave the phase stuck with nothing re-arming the timer.

**The fade runs the other way.** `fadeStyle` starts dim and arrives; this one
starts lit and leaves:

```go
var departRamp = lipgloss.Blend1D(departFrames, textRest, lipgloss.Color("#000000"))
```

Blended to black for the same reason `textRamp` blends up from `#1C1C1C` — a
terminal has no transparency, so a fade is always an explicit blend between two
opaque colours (Part I §7). On a light terminal the words darken rather than
dissolve; still a fade, just not the same one.

**What the screen shows:** the cross and three words, and nothing else. The ring,
the prayers, the mystery and the keys are all gone, so the fade has one thing to
carry and the screen empties as it dims. A farewell that still had the rosary on it
would be the rosary dimming; this is the rosary already put down.

`rampAt` exists because every one of these animations needs the same clamp, and
writing it out at each use is how one of them ends up missing it and panicking on
the frame where a duration was retuned but a ramp was not.

**Three blocks, not one.** The ring, the closing words and the keys are joined as
separate blocks, because they are three different kinds of thing and the gaps
between them are what says so — the keys are chrome and should not read as part of
the closing words. `keysGap` sets that distance:

```go
strings.Repeat("\n", keysGap-1),
```

The `-1` is not a fudge: a block of n newlines is n+1 lines, and `JoinVertical`
adds its own separator, so the repeat has to be one short of the gap you want.
Off by one here is a visible row. `TestKeysSitBelowTheClosingWords` pins the rows
actually produced and fails in both directions.

**Two rows reclaimed.** The ring's canvas is as tall as the geometry reserved, and
the rows below the crucifix are blank — invisible while the prayer sits *beside*
the ring, but a four-row gap between the crucifix and the closing text. The fix is
`trimBlankRows`, and it carries a trap worth knowing:

```go
// WRONG: the canvas pads every row to the full ring width,
// so a "blank" row is 41 spaces and this trims nothing.
for lipgloss.Width(rows[len(rows)-1]) == 0 { ... }
```

A row is blank when it holds nothing but **spaces**, not when its width is zero.
This was written the wrong way first and silently did nothing.

## Chrome belongs outside the layout

Two things are drawn on the screen but are not part of the rosary: the mystery box
and the keys hint. Neither takes part in the ring's geometry.

The hint is **overlaid onto the finished screen**, in the bottom-left corner:

```go
centred := lipgloss.Place(m.width, m.height, Center, Center, block)
return withHint(m, centred)      // writes into the second-to-last row
```

It was originally a line inside the ring, which was wrong twice over: it competed
with the prayer for attention, and it padded out the ring's height — chrome
changing the shape of the thing it annotates.

### `overlay` counts columns, not bytes

Writing text into a row that already contains escape sequences means walking it by
**display column**:

```go
for _, r := range row {
    if r == 0x1b { inEsc = true; ... continue }   // escapes cost no columns
    if col == x { b.WriteString(s) }
    if col < x || col >= x+w { b.WriteRune(r) }   // skip covered cells
    col++
}
```

Skipping the covered cells is what keeps the row's width unchanged — otherwise the
overlay pushes the rest of the row right and the screen shears.

> **This trap has now bitten three times in this build, every time in a *test*
> rather than in the code.** `strings.Index` returns a byte offset; a row of the
> rosary contains bead glyphs at three bytes each. So a test that checks "what
> column is this text at" reports a number two or more too high and fails on
> correct code.
>
> Convert every byte offset to a column: `lipgloss.Width(row[:i])`. Never compare
> `strings.Index` to a column.

### No counts

"Hail Mary 3 of 10" and "bead 11 of 61" were removed deliberately. A count on
screen turns praying into progress-watching: the eye goes to the number instead of
the words. The beads already show where you are, which is the right place for it —
in the object, not in text.

`Bead.Nth` and `Bead.Of` still exist, because they describe the structure; they
are simply not displayed.

## `beads.go` — the types behind it

```go
type Bead struct {
    Kind Kind      // Large | Small | Cross — decides the glyph
    Name string    // "Hail Mary"
    Says []Words   // one or more prayers, said in order
    Nth, Of int    // "3 of 10", set by run()
}

type Words struct {
    Title string
    Lines []string   // kept exactly as written, never reflowed
    prayerID string  // set by Say(); resolved at startup
}
```

`builder` exists only so `sequence.go` reads as a rosary rather than as slice
appends. You shouldn't need to touch this file to change the rosary — only to
change what a bead *is*.

## `ring.go` — geometry

### The canvas

```go
type canvas struct {
    cells [][]rune
    w, h  int
}
func (c *canvas) set(x, y int, r rune)       // bounds-checked, so callers needn't be
func (c *canvas) text(x, y int, s string)    // writes a line left-to-right
```

`set` silently ignores out-of-bounds writes. That is deliberate: a dozen call
sites would otherwise each need the same bounds check, and a bead slightly off
the edge should be clipped, not a panic.

### The ring is FIXED — sized once, never again

This is the important decision in `ring.go`, and it changed during the build.

The ring was originally re-measured for every bead. That was wrong: it made the
ring **breathe**. Move from a long prayer to a short one and the ring shrank, the
beads crawled inward, then crawled back out. It read as a glitch, not a design.

A rosary is a physical object. It does not change shape because the next prayer
is shorter. So:

```go
// Called ONCE, in newModel. Never again.
func fixedRing(beads []Bead) ringGeometry {
    // measure every prayer in the whole rosary...
    // ...and size the ring to the largest
    return layout(len(beads), textW, textH)
}
```

The ring is sized to the **longest prayer in the entire sequence** and then
frozen into `model.ring`. Short prayers simply sit in more space. `drawRosary`
takes the geometry as an argument rather than computing one, so it draws and does
not decide — which is what keeps the ring still.

### Two constraints the ring must satisfy

The growth loop widens the ring until **both** are true:

```go
for pass := 0; pass < 400; pass++ {
    if clears(rx, ry, n, textW, textH) && spacedOut(n, rx, ry) {
        break
    }
    rx++
    if pass%3 == 2 { ry++ }   // grow vertically too, every third pass
}
```

**1. `clears` — no bead on a letter.** The original check.

**2. `spacedOut` — no two beads in the same cell.** Added when the full rosary
arrived: without it the ring stops growing the moment the contents fit, and beads
can land on top of one another, losing one entirely.

It once demanded a blank cell *between* neighbours, which read better but forced
the ring far larger than its contents needed:

```
Before:   ○○  ◦●  ○○ ○ ○○        ← touching pairs read as a smear
After:    ○ ○ ○  ○ ○ ○  ○ ○ ○     ← distinct beads
```

It uses Chebyshev distance (`max(|dx|, |dy|) >= 2`), so diagonal neighbours count
as touching too.

**Why both radii grow.** With many beads the ring is crowded, and widening alone
cannot help if the ring is too *short* — there will always be a bead on a text
row. Growing in both directions lets it escape whichever way it is stuck.

### Spacing by distance, not by angle

This took **three** goes, and each fixed a different mistake.

**First version: equal angles.** On a circle that is also equal distance; on an
ellipse it is not. Equal angles cover much less ground where the curve is tight,
so beads clumped at the narrow ends.

**Second version: equal arc length.** Better, but still visibly clumped — `●●●
●●●` across the top and bottom, a thin string down each side. The reason is the
cell aspect ratio. A cell is twice as tall as it is wide, so a step of vertical
travel costs **twice** the visual distance of a step across. The ellipse is flat
where it crosses the vertical axis and steep where it crosses the horizontal, so
equal arc length buys very different numbers of CELLS depending on where you spend
it. Measured: worst-to-best gap ratio **1.98**.

**Third version: equal VISUAL distance.** One multiplication:

```go
dist[k] = dist[k-1] + math.Hypot(x-px, (y-py)*cellAspect)
```

Ratio **1.01**. The lesson generalises: when you measure distance on a terminal
grid, you are measuring in a space that is stretched 2:1, and any metric that
ignores that is measuring the wrong thing.

```go
want := total * float64(i) / float64(n)   // bead i goes here
```

Divided by `n`, not `n-1`: the loop is **closed**, so the last bead must stop one
interval short of the first rather than landing on top of it. `n-1` is right for an
open arc with two distinct ends, and was right while this was one.

The ellipse has no closed form for arc length, which is why this is a numeric
walk rather than a formula. It runs once at startup, at 20,000 steps — the walk
quantises every angle to a step boundary, and 2,000 was enough to shift a big bead
0.18° off true.

### The pentagon is a consequence, not a construction

The five big beads sit at exactly **72.00°** from each other. Nothing arranges
them.

Each decade is eleven beads, and there are 55 on the ring, so evenly spacing 55
slots around a **closed** loop puts every eleventh one at a fifth of the circle.

That word *closed* is load-bearing, and it was the whole reason the pentagon did
not appear sooner. The ring used to be an open arc with a 51.6° gap at the bottom,
on the reasoning that a rosary's loop has a mouth where the pendant joins it. But
55 beads spread over a partial arc give **62.8°** between every eleventh, and no
amount of even spacing fixes that — 55 divides by 5 only over a full turn.

Closing the loop and hanging the pendant from the bottom bead (rather than through
a gap beside it) made the arithmetic exact, and deleted a pile of scaffolding with
it — see below.

**Why iterate rather than solve?** The closed forms here are long, easy to get
subtly wrong, and hard to read six months later. A few hundred iterations of an
obviously-correct predicate is the better trade. If it ever shows up in a
profile, solve it then.

### The cursor does not replace the crucifix

The current bead is drawn as a halo instead of its own glyph (§Part III). There is
one exception:

```go
if i == cursor && b.Kind != Cross {
    glyph = currentGlyph(glow)
}
```

Without the `!= Cross` guard the crucifix was replaced by a bead glyph at step 1
— so the cross vanished from the rosary at exactly the moment you were praying the
Sign of the Cross on it. The cross shows selection by colour and weight instead.

> **The general point:** a selection indicator that *replaces* content is fine
> when the content is interchangeable, and wrong when the shape carries meaning.

### The pendant's shape

The pendant is **crucifix — bead — gap — three beads — gap — the ring**, where the
ring's first big bead is the start of the loop:

```
    ●  ●  ●     ← ring; the loop begins on its first big bead
   ...
       ●        ← three Hail Marys
       ●
       ●
                ← gap
       ●        ← Our Father
       ✠        ← crucifix: Sign of the Cross, then the Apostles' Creed
```

Note the crucifix carries **two** prayers. A bead is a place you hold, not a
single prayer — so which prayers sit on which bead is part of the structure, and
`TestPendantShape` pins it by prayer title rather than only by bead kind. Bead
kinds alone would not notice a prayer moving from one bead to its neighbour.

Two knobs in `sequence.go`:

```go
func Pendant() int            { return 5 }            // beads below the ring
func PendantGapAfter(i) bool  { return i == 1 || i == 4 }  // blank rows of chain
```

`Pendant()` must match the beads you put at the top of `Sequence()`.
`PendantGapAfter` is indexed from the crucifix (bead 0), and a gap after the last
pendant bead is what separates the chain from the ring.

### The rosary closes where it began

A rosary has **exactly five big beads**, one per decade. The closing prayers — the
fifth decade's Glory Be and Fatima Prayer, then the Hail Holy Queen — are said
back at the bead praying *started* on: the fingers come round the loop and arrive
where they began.

Giving them beads of their own put **seven** big beads on the ring, two of them
stranded beside the join.

Appending them to the starting bead fixed the count but broke the praying: that
bead then carried all five prayers, so you prayed the entire close *before* the
first decade.

The answer is a second bead at the **same position**:

```go
s.addAt(Pendant(), Large, "Hail Holy Queen", gloryBe, fatima, hailHolyQueen)
```

`Bead.SameAs` marks it as sharing another bead's cell, so the geometry gives it no
slot of its own. One bead on screen, two stops in the sequence: the opening bead
says Glory Be and Our Father, and the closing stop — the same bead, visited again
— says Glory Be, Fatima and Hail Holy Queen.

> **A flat sequence cannot revisit an entry.** Rather than model revisits, a
> revisit *is* a second entry that shares the first's position. That keeps the
> cursor a plain index while letting the drawing show one bead.

**The trap this creates:** two beads at one cell means whichever is drawn last
wins. Standing on the opening bead, the closing bead was drawn after it and
overwrote the halo — the highlight simply disappeared. So the drawing skips any
bead that shares the current bead's cell:

```go
if i != cursor && g.pos[i] == g.pos[cursor] {
    continue   // the bead that is not current yields the cell
}
```

### The first ring bead places itself

The bead where the loop begins and ends sits **directly above the pendant**, at the
bottom of the circle — and it gets there on its own, because `ringStart` is
straight down and the walk starts there:

```go
for i, a := range arcAngles(len(own), rx, ry) {   // one run, all of them
	g.pos[own[i]] = ...
}
```

This used to be three things: the bottom bead pinned by hand, the rest spaced
separately over the arc above it, and one extra slot requested and then dropped so
the run's two ends would not collide. All of it was scaffolding for the open arc.
A closed loop needs none of it — ask for exactly as many slots as there are beads
and they come back evenly spaced with the first at the bottom.

A good sign when a model is corrected: the special cases that propped up the old
one stop being necessary.

### The pendant comes from the sequence

The crucifix and the chain below the ring used to be a hardcoded decoration
(`var pendant = []string{"◦", "●", "◦", "✠"}`) drawn separately. That caused a
visible bug — **the crucifix appeared twice**, once in the ring and once below —
and those beads could not be navigated to.

Now the first `pendantLen` beads of `Sequence()` are *drawn* below the ring
instead of on it:

```go
const pendantLen = 5

for i := 0; i < pendantLen; i++ {
    g.pos[i] = [2]int{g.cx, g.cy + g.ry + (pendantLen - i)}   // in a line below
}
// the rest go round the arc
```

They are the same beads with the same cursor, so the crucifix is navigable like
anything else. Where a bead is *drawn* is a display choice and lives in
`ring.go`; which beads *exist* is structure and lives in `sequence.go`.

### The one-definition rule for the centre

```go
func centre(rx, ry int) (cx, cy int) { return rx + 2, ry + 1 }
```

Both `layout` (which draws) and `clears` (which checks) call this. They
**must** agree: if the collision check computes the centre even slightly
differently from the drawing, it validates a layout that is not the one drawn,
and a bead lands on a letter.

This was a real bug in this build. `layout` used `(ry*2+3)/2` while `clears`
used `ry+1` — equal only by coincidence of integer division, a trap waiting for
someone to change one of them. Now there is one function.

### Integer cells, not floats

The first version of `clears` compared float half-heights (`float64(textH)/2`)
while the drawing used integer rounding. They disagreed by a row, and bead 4
landed on a letter. `clears` now works in exactly the integer cell coordinates
the drawing uses.

> **Lesson:** a collision check must run in the *same* coordinate space, with
> the *same* rounding, as the thing that draws. Approximating "close enough" is
> how off-by-one visual bugs are born.

## `style_grid.go` — the colour pass

```go
func styleGrid(c *canvas, g ringGeometry, beads []Bead, cursor int) string
```

It builds a map of `position → {kind, isCurrent}`, then walks every cell:

- space → write a space
- a position in the map → write the rune wrapped in its bead style
- anything else (prayer text) → **write the rune bare**

That last case is intentional on two counts. It keeps the output small (941
bytes for a full frame rather than ~26 bytes per character), and it honours
`RENDERING.md`'s restraint principle: the prayer keeps the terminal's own
colour, and only the beads are gold.

Why a map rather than recomputing: it is O(1) per cell and, more importantly,
it reuses the *same* `g.pos` the drawing used, so the colour cannot land
somewhere the glyph is not.

## `view.go` — content and colour definitions

```go
func contentLines(m model) []string   // title, prayer, status — all PLAIN
func render(m model) string           // draw, then centre in the window
```

`contentLines` returns plain strings because the grid needs true cell widths
(§3). It splits the prayer on `\n` and **never reflows or shortens a line** —
`RENDERING.md` calls the file's line breaks sacred, and the ring grows to fit
the prayer rather than the prayer shrinking to fit the ring.

Bead styles:

```go
beadDim     = Foreground(gold).Faint(true)                 // Hail Mary beads
beadLarge   = Foreground(gold)                             // Our Father beads
beadCurrent = Foreground(gold).Bold(true).Reverse(true)    // where you are
crossBead   = Foreground(gold).Bold(true)                  // the crucifix
```

`Reverse(true)` for the current bead, as explained in §3, so your place on the
rosary survives `NO_COLOR`.

## `model.go` — state and the loop

```go
type model struct {
    beads  []Bead
    cursor int                            // index into beads
    loaded map[string]prayers.Prayer      // prayers cached by id
    width, height int                     // 0 until the first WindowSizeMsg
}
```

Design notes:

- **Prayers load once, in `newModel`, before the program starts.** A decade says
  the same ten Hail Marys; re-reading files on every keypress would be waste.
  More importantly, a missing prayer becomes an ordinary CLI error rather than a
  failure inside a full-screen program the user then has to escape from.
- **`width`/`height` start at 0.** The size arrives as a `WindowSizeMsg` at
  startup, so there is one frame where it is unknown. `render` handles that by
  returning the block unplaced — centring inside a 0×0 box would collapse it.
  There is a test for exactly this frame.
- **Two cursors, not one.** `cursor` is the bead; `say` is which prayer *on* that
  bead. Both are needed because a bead can hold several prayers.
- **Navigation is one function, `next()`.** Space is the primary key — one press,
  one prayer — with enter and `→` doing the same for people who reach for them.
  The rule lives in one place so every key means exactly the same thing:

```go
func (m *model) next() bool {
    if m.say < len(m.bead().Says)-1 {
        m.say++            // another prayer on the same bead
        return true
    }
    if m.cursor < len(m.beads)-1 {
        m.cursor++         // on to the next bead
        m.say = 0
        return true
    }
    return false           // the end of the rosary
}
```

  `prev()` mirrors it, landing on the **last** prayer of the previous bead so
  stepping back undoes exactly one space press.

## Tests — what they pin, and why

Four tests are about rendering specifically, and each exists because something
actually broke:

| Test | Guards |
|---|---|
| `TestNoBeadCollidesWithTheMystery` | No bead on a letter, for **every** bead of the decade. The invariant the whole sizing loop exists to maintain. |
| `TestHighlightFollowsCursor` | Exactly one bead highlighted, and it tracks the cursor. Counts `\x1b[1;7` (reverse video). |
| `TestStanzasShareALeftEdge` | Prayer lines share a left margin — catches the ragged-diamond failure. |
| `TestViewBeforeFirstResize` | The one frame before the size is known still draws. |

`TestNoBeadCollidesWithTheMystery` loops over all twelve beads rather than checking
one, because each bead shows a different prayer, so each produces a differently
sized ring — twelve geometries, not one.

**How to verify a test can actually fail.** When a test passes first time, prove
it is not vacuous by breaking the code deliberately:

```
$ # (temporarily remove the block-padding line)
--- FAIL: TestStanzasShareALeftEdge
    stanza lines are not flush: "the Lord is with thee." starts at 24, expected 23
```

A test that cannot fail is worse than no test, because it buys false
confidence.

> ### Trap: `strings.Contains` cannot find text drawn on the grid
>
> `styleGrid` wraps **each rune in its own escape sequence**. A line reading
> `The Crucifixion` on screen is really:
>
> ```
> \x1b[1;38;2;...mT\x1b[m\x1b[1;38;2;...mh\x1b[m\x1b[1;38;2;...me\x1b[m ...
> ```
>
> so `strings.Contains(screen, "The Crucifixion")` is **always false**. Text in
> the prayer *panel* is findable, because that is styled a whole line at a time —
> and that is precisely the trap: the same assertion works for panel text and
> passes vacuously for grid text.
>
> This cost a vacuous assertion here. `TestFinishScreenOffersAgainAndFinish`
> checked that the last mystery was no longer shown, and passed even when the
> mystery was deliberately put back. The fix is a `plain()` helper that strips the
> escapes before matching.
>
> Same family as the byte-offset trap above, and the same lesson: **once escapes
> are in a string, it is no longer the text you think you are searching.**

> ### Trap: a test that goes through the UI can hide the bug
>
> `TestBeginningAgainClearsTheMystery` presses space through the chooser and
> checks the new rosary has no mystery. It passed even with `begin()`'s cursor
> reset deleted — because the chooser resets the cursor itself, so `begin()`'s own
> reset was never exercised.
>
> The fix was `TestBeginResetsTheRosary`, which calls `begin()` **directly**. When
> a test drives a path through several steps, ask which step is actually under
> test — and whether an earlier one is quietly doing its job for it.

---

# Part III: animation

## The core idea: there is no animation API

Bubble Tea has no animation system, and doesn't need one. An animation is
**state that changes over time**, so it uses the same loop as everything else:

```
  key press ──→ Update sets a counter, returns a timer Cmd
                   │
                   ↓
            runtime waits 50ms, sends frameMsg
                   │
                   ↓
       Update decrements the counter, returns ANOTHER timer Cmd
                   │                             │
                   ↓                             │
                 View draws                      │
                   │                             │
                   └──── repeat ─────────────────┘
                   
            ...until Update stops re-arming. Then it's over.
```

Three pieces, no more:

1. **A number in the model** that represents "where in the animation are we"
2. **A timer command** that sends a message after a delay
3. **`View` drawing whatever that number currently means**

## Never sleep in a TUI

The instinct is to write this:

```go
// WRONG — the whole program is frozen for 300ms
for i := 6; i > 0; i-- {
    drawBead(brightness(i))
    time.Sleep(50 * time.Millisecond)
}
```

That blocks `Update`. While it sleeps the program processes no keys, no resizes,
no quit signal. The user presses `q`, nothing happens, they press `ctrl+c`
harder. And `View` isn't even being called — you're drawing from inside
`Update`, which breaks the architecture's one rule.

The timer approach never blocks. Each frame is one ordinary pass through
`Update`, so keys keep working *during* the animation.

## `tea.Tick` fires exactly once

This is the thing to internalise:

```go
func tick() tea.Cmd {
    return tea.Tick(frameRate, func(t time.Time) tea.Msg {
        return frameMsg(t)
    })
}
```

`tea.Tick` is **not** an interval. It sends one message, once. To keep animating,
the handler for that message must call `tick()` again:

```go
case frameMsg:
    if m.glow > 0 {
        m.glow--
        if m.glow > 0 {
            return m, tick()   // ← re-arm: this is what continues the animation
        }
    }
    return m, nil              // ← don't re-arm: the animation is now over
```

**This self-terminating property is a feature.** The animation stops by simply
not re-arming, so there is nothing to cancel, no goroutine to kill, no cleanup
on quit. Compare a `time.Ticker`, which you must remember to `Stop()`.

### `Tick` vs `Every`

| | Behaviour | Use for |
|---|---|---|
| `tea.Tick(d, fn)` | fires `d` from *now* | animation frames |
| `tea.Every(d, fn)` | fires on the next wall-clock boundary | clocks; keeping several things in sync |

For animation you want `Tick`. `Every(time.Second)` at 12:34:20 fires at
12:35:00 — 40 seconds later, not one. That's right for a clock display and wrong
for a fade.

## Frame rate: slower than you think

```go
const frameRate = 50 * time.Millisecond   // 20fps
```

60fps is pointless in a terminal. The eye can't resolve smoothness in character
cells the way it can in pixels, and every frame is a **full redraw plus a diff**
of the whole screen. 20fps reads as smooth for a colour fade at a third of the
cost.

Bubble Tea's renderer has its own cap, default 60fps, settable with
`tea.WithFPS(n)` (clamped to 1–120). That's the *ceiling* on redraws, not your
animation's rate — your rate is the `Tick` duration. Leave the renderer default
alone and control pace with your timer.

## Interpolating colour

Lip Gloss v2 ships the colour maths, so don't write your own:

| Function | Does |
|---|---|
| `Blend1D(steps, stops...)` | a ramp of `steps` colours through the stops |
| `Lighten(c, pct)` / `Darken(c, pct)` | shift one colour |
| `Alpha(c, a)` | apply transparency |
| `Blend2D(w, h, angle, stops...)` | a 2-D gradient |

The rosary's flare is one line:

```go
var glowRamp = lipgloss.Blend1D(glowFrames, lipgloss.Color("#FFF8E0"), currentRest)
```

Measured output — bright flare settling to the bead's resting colour:

```
#FFF8E0 → #FFF9CE → #FFFABB → #FFFAA8 → #FFFB94 → #FFFB80
```

### The ramp must END where the static state begins

Note it blends to `currentRest`, **not** to `gold`. This was a real bug: blending
to `gold` made the flare fade *below* the bead's resting brightness and then jump
back up on the final frame — a flicker that is obvious once you know to look for
it and easy to miss otherwise.

> **Rule:** an animation's last frame must equal the static state it hands over
> to. There is a test for exactly this (`TestGlowLandsOnTheRestingColour`),
> because eyes are bad at catching a one-frame jump.

### Two animations, two counters, one timer

The bead's flare and the prayer's fade-in are separate animations with separate
durations, so each can be tuned without touching the other:

```go
const glowFrames = 16   // the bead's flare:     800ms
const fadeFrames = 20   // the prayer arriving: 1000ms

glow int   // counts down glowFrames -> 0
fade int   // counts down fadeFrames -> 0
```

One `frameMsg` decrements whichever are still running, and the timer re-arms
**while either is above zero**:

```go
case frameMsg:
    if m.glow > 0 { m.glow-- }
    if m.fade > 0 { m.fade-- }
    if m.glow > 0 || m.fade > 0 { return m, tick() }
    return m, nil
```

So there is one clock and two independent durations. The alternative — a single
shared counter read through two curves — saves a field but couples the two:
lengthening one silently changes where the other sits.

**Derive the ramps from the constants.** `glowGlyphs` is built from `glowFrames`
rather than written out, because a hand-written list silently becomes the wrong
length the moment a duration is tuned. There is a test for exactly this drift, and
it caught it during this build.

### Terminals have no transparency

The obvious way to fade text in is alpha. It does not work:

```go
lipgloss.Alpha(c, 0.1)   // RGBA says alpha=25...
// ...but renders "[38;2;224;224;224m" — identical to alpha=1.0
```

The alpha channel is **dropped from the escape sequence entirely**. Measured: 0.1,
0.4, 0.7 and 1.0 all produce the same bytes.

A fade must therefore be an explicit blend between two **opaque** colours — from
something near the background up to the text's resting colour:

```go
var textRamp = lipgloss.Blend1D(fadeFrames, lipgloss.Color("#1C1C1C"), textRest)
```

This assumes a dark background. Detecting the real one needs the tty
(`lipgloss.HasDarkBackground`), which a pure `View` cannot reach — so it is a
deliberate assumption, not an oversight.

### The fade hands over to the terminal's own colour

The settled prayer is written as a **bare rune** with no escape at all, so it sits
in the terminal's foreground colour (RENDERING.md's restraint rule) and the frame
stays small. But that colour is unknown, so fading all the way to a specific grey
and then dropping to it would jump on the last frame.

The fix is to stop one frame early — `if fade > 1` — so the handover happens while
the ramp is still close, and the jump is invisible.

### Animating shape, not just colour

Colour alone on a fixed glyph looks like a bulb dimming. Changing the **glyph**
across the flare looks like light spreading:

```go
var glowGlyphs = []string{"✹", "✻", "◉", "◎", "◎", "◎"}
//                         bloom ──────→ settle
```

The halo blooms outward and settles to a steady ring. Two constraints:

1. **Every glyph must be the same cell width** (§2), or the ring shears mid-animation.
2. **It must settle before the animation ends** — a glyph still changing on the
   final frame makes the bead twitch after it has visibly stopped. Hence the
   repeated `◎` at the tail, and `TestGlyphSettles`.

**Build the ramp once, at package level.** Blending per frame is wasted work;
a fixed ramp makes each frame a lookup, not a computation.

### Easing

The ramp above is linear. For motion, linear looks mechanical — real things
accelerate. Apply an easing function to the *index* before you look up:

```go
// ease-out: fast at first, slow at the end
t := float64(i) / float64(steps)        // 0..1
eased := 1 - (1-t)*(1-t)                // quadratic ease-out
idx := int(eased * float64(steps-1))
```

For a colour fade linear is fine, which is why this project doesn't bother. For
anything that *moves* — a sliding panel, a progress bar — easing is the
difference between "animated" and "polished".

## Kinds of animation, and what each costs

| Kind | Technique | Cost |
|---|---|---|
| **Colour fade** (this project) | interpolate a colour over N frames | cheapest; no layout change |
| **Glyph swap** (this project) | cycle through glyphs of the *same width* | cheapest; the shape change is what reads as "glow" |
| **Spinner** | cycle through glyphs `⠋⠙⠹⠸⠼⠴` | cheap; fixed width, so no reflow |
| **Progress bar** | redraw N filled cells of M | cheap if width is fixed |
| **Reveal / typewriter** | draw the first N characters | careful: width changes each frame |
| **Movement** | recompute positions per frame | most expensive; full re-layout |

The first three don't change *layout*, only what's drawn in cells that already
exist. The last two do, and that's where terminal animation gets expensive —
every frame re-runs your geometry. In this codebase, movement would mean
re-running `layout()` (with its growth loop) 20 times a second.

**Design implication:** prefer animating *appearance* over *position*. A bead
that brightens costs nothing; a bead that slides re-lays out the ring.

## Why the terminal makes this cheap anyway

Bubble Tea diffs frames and sends only the cells that changed. The flare touches
**one cell**, so each frame writes a couple of dozen bytes, not the whole 83×30
screen. That's why a 20fps animation in a terminal is essentially free compared
to the same thing in a browser.

It also means a *badly* designed animation is expensive in a way you can feel:
if every frame changes the layout, the diff is the whole screen, 20 times a
second.

## Testing animation without waiting

The payoff of "animation is just state": you can test it by injecting the
message directly instead of letting the timer deliver it.

```go
// Don't wait 50ms — hand Update the message yourself.
next, cmd := m.Update(frameMsg{})
```

The production path sleeps; the test doesn't have to. The whole animation test
suite here runs in milliseconds.

What's worth pinning:

| Test | Guards |
|---|---|
| `TestMovingABeadStartsTheGlow` | a move sets the counter and schedules a frame |
| `TestAnimationsFadeAndStop` | the counter decrements **and the last frame returns nil** |
| `TestStrayFrameIsHarmless` | a frame arriving when idle doesn't drive the counter negative |
| `TestRapidMovesResetRatherThanStack` | fast keys reset the animation, not stack it |
| `TestGlowStyleIsSafeAtTheEdges` | the ramp index is clamped at both ends |

The second is the important one. **An animation that never stops re-arming is an
infinite timer loop** — it burns CPU forever and you may not notice, because it
looks fine. Assert that the final frame returns `nil`.

I verified both of these catch real bugs by breaking the code deliberately:

```
# forced the re-arm to always happen:
--- FAIL: TestAnimationsFadeAndStop
    the final frame re-armed the timer; the animation would never stop

# let the counter go negative:
--- FAIL: TestStrayFrameIsHarmless
    glow = -1, want 0
```

## The overlapping-animation trap

What happens if the user presses a key *while* a flare is in flight? There's
already a timer out there that will deliver a `frameMsg`.

Naively you might try to cancel it. You can't, and you don't need to:

```go
func (m *model) startGlow() tea.Cmd {
    m.glow = glowFrames   // just reset the counter
    return tick()
}
```

The in-flight message still arrives, decrements the (freshly reset) counter, and
re-arms. So pressing keys rapidly does not stack animations — **state decides
what happens, not the number of timers in flight.**

There is a subtle cost: each press adds a timer, so holding a key could put
several in flight at once, making the fade run faster than 20fps. For a 300ms
flare that's invisible. If it mattered, you'd stamp each animation with an id and
ignore frames whose id is stale:

```go
type frameMsg struct{ id int }   // ignore if msg.id != m.animID
```

This project doesn't need that, and adding it pre-emptively would be machinery
without a problem. Know the pattern for when you *do* need it.

## How it works in this codebase

| File | Role |
|---|---|
| `animate.go` | `frameRate`, `glowFrames`, `frameMsg`, `tick()`, `glowRamp`, `glowStyle()` |
| `model.go` | the `glow` counter; `startGlow()`; the `frameMsg` case in `Update` |
| `view.go` | `beadStyle(kind, current, glow)` picks the animated style |

The flow for one keypress:

```
'l' pressed
  → Update: m.cursor++, m.glow = 6, returns tick()
  → 50ms later: frameMsg → m.glow = 5, returns tick()
  → ...
  → frameMsg → m.glow = 0, returns nil        ← animation over
```

`beadStyle` is the one style built per frame rather than once at startup,
because its colour depends on `glow`. That is the entire runtime cost of the
animation.

Note what did **not** change: `ring.go` has no idea an animation exists. The
geometry is identical on every frame; only the colour of one cell differs. That
separation is why the flare was cheap to add — and it's the practical argument
for animating appearance rather than position.

---

# Part IV: how to change things

A recipe per question. Each names the file and what to change, with the
constraints that will bite you.

## 1. Change bead style and size

### The glyph for a kind of bead

`beads.go`, `Kind.Glyph()`:

```go
// beads.go — the two glyphs, and the cross
var (
    bigBead   = "●"   // the five that open the decades
    smallBead = "●"   // Hail Marys, and the pendant's chain
)

func (k Kind) Glyph() string {
    switch k {
    case Cross: return "✠"
    case Small: return smallBead
    default:    return bigBead
    }
}
```

There are exactly **two bead sizes plus the crucifix**. A `Link` kind existed
while the Glory Be had its own bead on the chain; once junctions became single
beads nothing constructed one, and it was removed.

### "Size" is really glyph weight

A terminal cannot scale a character: every cell is the same size. So a "bigger"
bead means a **heavier glyph**.

> **Rule 1: every bead glyph must be 1 cell wide.** `⚪` and `⚫` look like beads
> and are **2 cells** — each one shifts its row a column right and shears the ring.
> `lipgloss.Width("⚪")` → 2. See §2.

> **Rule 2: both bead glyphs must share an East Asian Width CLASS.** This one is
> subtler, and `lipgloss.Width` cannot catch it — read on.

#### The trap: declared width is not painted width

The pair `⬤` / `●` shipped for a while and looked wrong in a way that was hard to
name: the big bead at the top of the pendant sat slightly **right** of the pendant's
column — and snapped into place when selected.

`lipgloss.Width` reports **1** for both. It reports the width Unicode *declares*;
a font may paint a glyph wider than its cell anyway, and the overflow goes
rightward from the cell's left edge. Measured off a screenshot: the big bead's
centre was ~9px right of the pendant's, about half a cell.

The property that predicts this is the **East Asian Width class** (UAX #11):

| Glyph | Codepoint | Class | |
|---|---|---|---|
| `⬤` | U+2B24 | **N** | BLACK **LARGE** CIRCLE — the name says it |
| `●` | U+25CF | **A** | |
| `○` | U+25CB | **A** | |
| `◉` `◍` `⬢` | | **N** | same problem |

Mixing classes is what bit. And it explains the selected-bead behaviour exactly:
the halo glyph `◎` that replaces the bead under the cursor is class **A**, so the
bead appeared to jump into alignment the moment you landed on it.

Same-class pairs, all safe:

| | Big / small | Reads as |
|---|---|---|
| **default** | `●` `○` | filled vs hollow — strongest contrast |
| | `◆` `◇` | diamonds |
| | `⭘` `○` | two rings, more delicate |
| | `●` `·` | smalls recede to a thread |

`TestBeadGlyphsShareAWidthClass` enforces it for the two bead glyphs.

#### The crucifix: drawn from lines, for exactly this reason

`✠` (U+2720 MALTESE CROSS) is class **N**, so it had the same problem — measured
off a screenshot, its painted centre sat about **17px** right of the pendant's
column, roughly half the 42px error the old `⬤` had.

There is no good single-glyph fix. Almost every cross glyph is class N — `✝ ✞ ☩ ✚
✛ ✜ ⸸` all are; the only class-A options are `†` and `‡`, which read as daggers.

So the rosary's crucifix is **assembled from box-drawing characters**, which are
class A throughout and are designed to join across cell boundaries:

```
  ┃
━━╋━━
  ┃
  ┃
```

It lines up with the pendant exactly, and no font can push it off — see
`cross.go`, where the art is declared once and the grid write, the colour pass,
the canvas height and the collision check all derive from it.

**The cost, which is real.** The crucifix is no longer one cell, and three things
had to follow:

1. `drawRosary` stamps it as a block rather than calling `c.set` once.
2. `styleGrid`'s map is keyed by position, so **every** cell of the cross is
   registered — a cross that registered only its stem would have its arms coloured
   by the fallback branch, which treats anything unclaimed as the mystery text.
3. The canvas grew by `crossRows()`. `canvas.set` silently ignores out-of-bounds
   writes, so a canvas one row short loses the cross's foot with no error anywhere.

`Cross.Glyph()` survives, returning the crossbar `╋` — the one cell unique to the
cross — so "find the crucifix on screen" is still a single call.

#### …except on the farewell, which keeps `✠`

The closing screen shows the single glyph. The lines exist to hold the crucifix
true to the pendant's column; the farewell has no column, just a cross above three
words, so nothing can be out of true and the handsomer glyph wins:

```
     ✠

Go in peace.
```

That is `malteseCross` in `cross.go`, deliberately separate from `Cross.Glyph()`.

### The colours

`view.go`:

```go
beadDim     = Foreground(gold).Faint(true)   // Small
beadLarge   = Foreground(gold)               // Large
crossBead   = Foreground(gold).Bold(true)    // Cross
beadCurrent = Foreground(currentRest).Bold(true)
```

`beadStyle(kind, current, glow)` picks between them; `current` wins over kind, and
an active glow wins over both.

### A different glyph per bead, not per kind

If one bead needs its own look, add a field to `Bead` rather than inventing a
`Kind`:

```go
type Bead struct {
    ...
    Glyph string   // overrides Kind.Glyph() when set
}
```

then in `drawRosary` prefer `b.Glyph` when non-empty. Set it in `sequence.go` the
same way `announce()` works.

## 2. Customise bead order

All in `sequence.go`. Four calls, nothing else:

```go
s.add(Large, "Our Father", Say("our-father"))        // one bead
s.run(Small, 10, "Hail Mary", Say("hail-mary"))      // ten identical beads
Announcing(decade, Say("our-father"))                 // this PRAYER names mystery n
func Pendant() int { return 6 }                       // how many hang below the ring
```

Common edits:

| Want | Do |
|---|---|
| A different decade length | `s.run(Small, 7, ...)` — the ring resizes itself |
| Three decades, not five | change the loop bound |
| A prayer between decades | `s.add(...)` inside the loop, after the Glory Be |
| Two prayers on one bead | pass two `Words`: `s.add(Large, "Our Father", gloryBe, fatima)` |
| Announce the mystery elsewhere | wrap a different prayer in `Announcing(decade, …)` |
| A longer pendant | add beads at the top, and raise `Pendant()` to match |

**`Pendant()` must match the beads you put at the top.** It is the count drawn
*below* the ring rather than on it. Get it wrong and a bead that should be on the
chain appears on the ring (or the reverse) — there is a test, but the number is
yours to keep honest.

**Prayers:** `Say("id")` reads `internal/prayers/data/<id>/`; `Text("Title",
"line", ...)` is inline. A bad id is a startup error, not a blank bead.

## 3. Change the ring's size and shape

All in `ring.go`.

### Size

The ring is sized **once** (`fixedRing`) against the largest mystery, then frozen.
Its final size is set by whichever constraint binds last:

```go
ry := textH/2 + 2            // starting height: contents + breathing room
for ; ry < 200; ry++ {
    rx := ry * 2             // ← the aspect ratio
    if clears(...) && spacedOut(...) { break }
}
```

| To make the ring… | Change |
|---|---|
| Rounder / flatter | the `rx := ry * 2` ratio. `2` is a visual circle (cells are ~2:1 tall); `3` is a wide oval, `1.5` a tall one |
| Bigger overall | `ry := textH/2 + 2` → a larger constant; it only ever grows from there |
| Tighter around the text | `const gutter = 3` → smaller. This is the blank cells kept between a bead and the text |
| Hold more beads without touching | `spacedOut` — change `if p == prev` to require a gap (`max(|dx|,|dy|) < 2`). **This made the frame 81×48 instead of 41×28**, so expect it to grow a lot |

**Resist making the ring bigger.** It is the obvious response to beads that touch,
and it is wrong. Measured at the current 55 beads:

| Size | Touching pairs | Blank rows in the chain |
|---|---|---|
| **18×9 (current)** | 4 | **0** |
| 22×11 | 0 | 2 |
| 26×13 | 0 | 5 |

A larger ring spaces consecutive beads more than one row apart, so rows appear with
no bead at all — the chain reads as **broken strands**, which is worse than two
beads touching. `TestTheRingHasNoBreaks` enforces this.

### Shape

```go
ringStart = -math.Pi / 2   // where the first bead sits: straight down
ringArc   = -2 * math.Pi   // a full turn, counter-clockwise
```

- **Direction** is the sign of `ringArc`. Negative runs counter-clockwise — away
  from the crucifix, which is the direction the fingers travel. Flip the sign and
  the decade runs backwards.
- **Starting position** is `ringStart`. `-π/2` is straight down, where the pendant
  hangs.
- **Do not reopen a gap** in `ringArc` without expecting to lose the pentagon.
  `TestTheBigBeadsFormAPentagon` will tell you: the five big beads only land 72°
  apart because 55 slots divide evenly over a *full* turn.

### If you change the geometry, check two things

1. `centre(rx, ry)` is used by **both** the drawing and the collision check. Keep
   it as the single definition — two derivations is how a bead silently lands on a
   letter.
2. `clears()` must stay in the **same integer cell coordinates** as `drawRosary`.
   It already disagrees by a row for an even-height block, which is the bug that
   let beads sit on the mystery box's border.

## 4. Make the current-step indicator more obvious

`animate.go` controls it. The indicator is a **glyph swap plus a colour flare**,
not just a colour — so it survives `NO_COLOR`.

### The quickest wins

```go
const glowFrames = 16                        // how long the flare lasts (×50ms)
bloom := []string{"✹", "✻", "◉"}             // the glyphs it passes through
var glowRamp = Blend1D(glowFrames, Color("#FFF8E0"), currentRest)
```

| To make it… | Change |
|---|---|
| Last longer | `glowFrames` up. 16 = 800ms |
| Flash brighter | the ramp's first colour, `#FFF8E0` → `#FFFFFF` |
| Bloom harder | add glyphs to `bloom`, e.g. `{"✺", "✹", "✻", "◉"}` |
| Rest more boldly | `currentRest` in `view.go` → lighter |

### Stronger options, with their costs

| Option | How | Cost |
|---|---|---|
| **Pulse forever** | don't let the counter reach 0 — re-arm and cycle the ramp | a permanently animating screen; also never stops ticking |
| **Background fill** | `Background(gold).Foreground(black)` on the current bead | fills the whole cell, which is the "ugly rectangle" look that `Reverse` gave |
| **Underline** | `.Underline(true)` | subtle, and some terminals render it oddly next to box-drawing characters |
| **Blink** | `.Blink(true)` | works, but widely disliked and some terminals ignore it |
| **Brackets** `(◎)` | draw into the neighbouring cells | **3 cells wide — it would overwrite the beads either side.** Only safe if the ring is grown to leave a gap (see `spacedOut` above) |

### The two traps

**The ramp must end where the resting style begins.** If `glowRamp`'s last colour
is not `currentRest`, the bead jumps brightness on the final frame — a one-frame
flicker that is easy to miss by eye. `TestGlowLandsOnTheRestingColour` guards it.

**The glyph must settle before the animation ends**, or the bead twitches after it
has visibly stopped. `glowGlyphs` is built from `glowFrames` so the bloom is short
and then holds on `◎` — which also means changing `glowFrames` cannot leave the
list the wrong length.

---

# Known problems

Honest state of the prototype.

### 1. It needs a wide terminal

**Measured:** the rosary plus its prayer panel renders **89 columns × 29 rows**,
so an 80×24 terminal clips it. The ring itself is only 41×29.

The closing screen is **41 × 32**: narrower, since the prayer panel is gone, but
three rows *taller* — the ring, the closing words, and the keys set below them.
Height is the binding constraint there, and `trimBlankRows` already reclaims four
rows below the crucifix. `keysGap` is the one remaining knob; shortening it further
means shortening the ring.

The width is simple arithmetic, not a bug:

```
41 (ring)  +  4 (gutter)  +  44 (prayer panel)  =  89
```

The panel is 44 because that is the longest prayer line in the rosary
(`as we forgive those who trespass against us;`), and it is a **fixed** width so
the layout does not shift between prayers.

`lipgloss.Place` clips rather than scales, so in a narrow terminal the beads at
the edges silently vanish. (This bit me in a test once: a highlight "disappeared"
purely because the test window was too small.)

Options, roughly in order of how well each preserves the design:

- **Re-break the `.md` lines shorter** (~30 columns) — takes ~14 columns straight
  off the panel. `RENDERING.md` already contemplates this, and it keeps line
  breaks an editorial choice rather than an automatic one.
- **Put the prayer below the ring** instead of beside it — one `JoinHorizontal` →
  `JoinVertical` in `render`. Trades width for height.
- **Stack below a width threshold**, keeping the side-by-side layout for wide
  terminals.

Height is 29 rows: the ring (`ry=9` → 19 rows) plus the pendant and its gaps.
Shrinking it means a smaller ring, which `rx = ry * 2` ties to the width.

### 2. The two closing beads sit adjacent on the ring

The fifth decade's Glory Be/Fatima bead and the Hail Holy Queen are two separate
beads, and they end up next to each other at the end of the ring — so there is one
`●●` pair left. Unlike the decade junctions (which were a real modelling error,
now fixed) these genuinely are two stops, so it is a question of how you want the
close prayed, not a bug: merge them in `sequence.go` if a single bead is right.

### 3. Four pairs of beads touch, from cell rounding alone

The spacing itself is even — ratio 1.01 in continuous space, and the five big beads
land at exactly 72°. What remains is the grid: ideal spacing is **2.06 cells**, and
a bead can only sit on a whole one, so some neighbours round to 1 apart and show as
`●●`.

This is not fixable by better spacing maths, and **not** fixable by a bigger ring —
see the table under "If you change the ring's size", where growing it trades these
four touching pairs for blank rows in the chain, which looks worse. It would take
either fewer beads or a finer grid than a terminal has.

### 4. `data/` has only three prayers

`our-father`, `hail-mary`, `st-carlo-acutis`. Everything else in `sequence.go`
uses `Text(...)` with abbreviated words — the Creed and Hail Holy Queen are
placeholders, not the full prayers. Swap each for `Say(...)` as it lands in
`data/`.

---

# Reference

### Pitfalls, collected

1. `len(string)` is bytes, not columns. Use `lipgloss.Width`.
2. Some glyphs are 2 cells wide. Measure before you design around one.
3. Never put styled text in a grid cell — style after layout, never during.
4. Cells are ~2:1 tall, so `rx ≈ 2 × ry` for a visual circle.
5. Screen y grows downward: use `cy - ry*sin(a)`.
6. Cells are ~2:1 tall for **distance**, too. Any "how far apart do these look?"
   must scale dy — equal arc length on an ellipse is not equal visual spacing.
7. `lipgloss.Width` is the width Unicode *declares*, not what the font paints. Two
   glyphs that both report 1 can still render differently — check the East Asian
   Width class and keep a set of glyphs in one class.
8. Spacing `n` items around a **closed** loop divides by `n`, not `n-1`. The `n-1`
   that is right for an open arc puts the last item on top of the first.
6. `int(math.Round(x))`, never `int(x)`.
7. A collision check must use the same coordinates and rounding as the drawing.
8. Derive a shared value (like the centre) in exactly one place.
9. `JoinVertical(Center, ...)` centres line by line — pad blocks to their own
   width first.
10. Don't encode meaning in colour alone; `NO_COLOR` exists. Use reverse/bold.
11. Never `os.Exit` in a Bubble Tea program; return `tea.Quit`.
12. Keep `View` pure, so frames are testable without a terminal.

### Useful commands

```bash
just run rosary          # run it
just test                # all tests
just check               # fmt + vet + test
go doc charm.land/bubbletea/v2 Model    # read the real API, not a tutorial
NO_COLOR=1 just run rosary              # check it degrades
```

To inspect a frame as text, write a temporary test that calls `render(m)` and
prints it — far faster than squinting at a live terminal, and it shows the
escapes:

```go
m, _ := newModel("en")
m.cursor = 3
next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
fmt.Printf("%q\n", render(next.(model)))   // %q reveals the escape sequences
```

### Further reading

- `go doc charm.land/bubbletea/v2` — the authoritative v2 API. Most tutorials
  online are v1; check signatures against this.
- Lip Gloss: https://pkg.go.dev/charm.land/lipgloss/v2
- UAX #11, East Asian Width — why some characters are 2 cells
- `RENDERING.md` in this repo — the static renderer's plan, and the source of
  the colour and line-break principles this package follows
