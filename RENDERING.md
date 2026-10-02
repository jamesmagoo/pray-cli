# Rendering plan

How `pray` will show prayers beautifully in the terminal, built in small
phases that can each be committed on their own.

## Decisions

| Topic | Decision |
|---|---|
| Styling library | [Lip Gloss v2](https://pkg.go.dev/charm.land/lipgloss/v2) (`charm.land/lipgloss/v2`) |
| Terminal detection & width | [`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term) |
| Frame | Double line: `╔═╗ ║ ╚═╝`, as in the README |
| Colours | Gold and purple, fixed (exact shades TBD, see below) |
| Line breaks | **Exactly as written in the `.md` file.** The renderer never reflows text |
| Cross | The single character `✠` (U+2720) |

No library for reading prayer files, golden tests or (later) liturgical
seasons: each is a few dozen lines of plain Go. Glamour, raw ANSI/termenv
and tview were considered and rejected: Glamour makes prayers look like
documentation, raw ANSI means rewriting Lip Gloss, and tview is a full TUI
framework.

Bubble Tea is **not** needed here. It comes in later for interactive
features (rosary, picker), and will reuse this renderer's output.

## Principles

1. **Line breaks are sacred.** Every line in the `.md` file is a line on
   screen. Stanzas are separated by a blank line. The renderer may add a
   hanging indent when a terminal is too narrow (see Phase 2), but it never
   joins or reflows lines.
2. **Plain text is untouched.** `Prayer.Plain()` stays the only plain-text
   form. `pray copy`, pipes and redirects always use it. Nothing in the
   renderer is reachable from `copy`.
3. **The renderer returns a string and does no I/O.** It doesn't print and
   doesn't check whether it's in a terminal. So `pray` can print it, tests
   can compare it, and a Bubble Tea screen can show it later.
4. **Only `root.go` decides framed vs plain.** One place checks "is stdout a
   terminal?", `--plain` and the terminal width, then passes explicit
   values to the renderer.
5. **Restraint.** Body text stays in the terminal's own colour for
   legibility. Gold and purple are accents only.

## The target

`pray hail mary --for "Mam & Dad"` in a terminal:

```
╔════════════════════════════════════════════════════╗
║                                                    ║
║                         ✠                          ║
║                                                    ║
║                     Hail Mary                      ║
║                    ───────────                     ║
║                                                    ║
║                   For Mam & Dad                    ║
║                                                    ║
║    Hail Mary, full of grace,                       ║
║    the Lord is with thee.                          ║
║    Blessed art thou amongst women,                 ║
║    and blessed is the fruit of thy womb, Jesus.    ║
║                                                    ║
║    Holy Mary, Mother of God,                       ║
║    pray for us sinners,                            ║
║    now and at the hour of our death.               ║
║                                                    ║
║                       Amen.                        ║
║                                                    ║
╚════════════════════════════════════════════════════╝
```

| Element | Alignment | Style |
|---|---|---|
| Frame | — | gold |
| Cross `✠` | centred | gold |
| Title | centred | gold, bold |
| Rule under title | centred, a little wider than the title | gold, faint |
| Intention line ("For …") | centred | purple, italic |
| Stanzas | left-aligned, 4-space padding | terminal default |
| Closing (e.g. "Amen.") | centred | terminal default, bold |

The intention line only appears for prayers without an `{{intention}}`
slot, exactly as in plain text.

## Phase 1: stanzas and closing (prayers package)

Give the renderer structure instead of one long string.

- Add `func (p Prayer) Stanzas(intention string) [][]string` to
  `prayer.go`. It takes `p.Text(intention)`, splits on blank lines into
  stanzas, and splits each stanza into lines, trimming only trailing
  spaces. Lines are otherwise kept exactly as written.
- **Closing rule:** if the last stanza is a single line, it's the closing
  and is centred. That covers "Amen." (Hail Mary) and "Through Christ our
  Lord. Amen." (St. Carlo) with no extra markup.
- Tests: Hail Mary has 3 stanzas, the last being `["Amen."]`; line breaks
  match the file exactly; an intention is filled in before splitting.

## Phase 2: the renderer (`internal/render`)

New package, one main function:

```go
// Framed draws a prayer in its double-line frame, no wider than maxWidth
// columns. It returns a string and does no I/O.
func Framed(p prayers.Prayer, intention string, maxWidth int) string
```

Build it top to bottom with Lip Gloss:

1. Style each part (cross, title, rule, intention, stanza lines, closing)
   with the styles in the table above.
2. Join stanzas with one blank line between them, and the parts with
   `lipgloss.JoinVertical`.
3. **Width:** the frame's inner width is the widest of the longest line,
   the title and the intention, plus padding. So a short prayer gets a
   snug frame, never a stretched one.
4. Wrap it all in `Border(lipgloss.DoubleBorder())` with gold border colour
   and padding of 1 line top and bottom, 4 columns left and right.
5. Centre the finished frame within `maxWidth`.

**Narrow terminals:** if the longest line can't fit within `maxWidth`, only
those overlong lines are broken at the last space that fits, and the
continuation is indented 2 spaces. That's a display fallback, the same one
hymnals use, not a change to the text. Shorter lines are untouched.

Colours live in one place, `render/theme.go`, so the TBD shades are a
one-line change.

## Phase 3: wire it into `pray`

All changes are in `internal/cli/root.go`:

- Add `--plain`: always print `Prayer.Plain()`.
- Decide once:
  - stdout is a terminal and no `--plain` → `render.Framed(p, intention, width)`
    printed with `lipgloss.Println`, which reduces colours to what the
    terminal supports and respects `NO_COLOR` (colour off; bold and italic
    stay).
  - otherwise → `p.Plain(intention)`, as today.
- Width comes from `term.GetSize` on stdout, with 80 as the fallback.
- `copy.go` doesn't change.
- Update the README: mention `--plain`, and that piped output is plain.

## Phase 4: golden tests

Lock the look in so any change to it shows up as a diff you review on purpose.

- `internal/render/testdata/<id>-<lang>-<width>.golden` for every prayer
  and language at widths **100** and **50**. Width 50 exercises the
  narrow-terminal fallback.
- Golden files hold the output **with colour codes stripped**, so they're
  readable text that looks like the target above. A separate small test
  checks that colours are actually applied.
- `go test ./internal/render -update` rewrites the golden files. The test
  loops over every prayer, so new prayers are covered automatically, the
  same way as `data_test.go`.
- Keep the existing guarantee: `TestCopyIsPlainText` must still pass
  unchanged.

## Phase 5: polish and real-terminal checks

- **Light and dark backgrounds:** pick a light and a dark variant of each
  colour with `lipgloss.LightDark`, so gold and purple are readable on both.
- **The cross:** check that `✠` doesn't shift the border in Terminal.app,
  iTerm2, Ghostty and VS Code's terminal. If it does in a common one,
  revisit the cross choice.
- **`NO_COLOR=1 pray hail mary`** still looks good, with frame and bold,
  no colour.
- Look at every prayer at a few real widths: 60, 80, 120.

## Open questions

- **Exact colours.** Pick hex values for gold and purple, each with a light-
  and dark-background variant. Candidates to try: gold `#C9A227` / `#E6C35C`,
  purple `#5B2C83` / `#B48CDE`.
- **St. Carlo is wide.** Its longest line is 78 characters before the
  intention is filled in. That's about 90 columns once framed, so it'll hit
  the narrow-terminal fallback in an 80-column terminal. Since line breaks
  follow the file, the fix, if wanted, is to re-break its lines in `en.md`
  (around 60 characters reads best). A data test could enforce a maximum
  line length for all prayers.
- **Rule under the title:** keep it, or let the title stand alone?

## Later, not in this plan

- Rubrics (instructions) in purple italic, and ℣ / ℟ for versicles and
  responses, once a prayer needs them.
- Seasonal colours that follow the liturgical calendar.
- `pray --version` showing the commit it was built from.
- Reusing `render.Framed` inside the Bubble Tea rosary and picker.
