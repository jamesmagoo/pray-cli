# Design scripts

Throwaway-looking but deliberately kept: these render the rosary's colours and
glyphs **in a terminal**, which is the only place a look-and-feel decision can
honestly be made. Escape sequences do not survive into a chat log or a diff —
`\x1b[38;2;91;141;214m` is all anyone reads — so they have to be run.

| Script | Shows |
|---|---|
| `swatch.sh` | Bead colours, with and without `Faint` |
| `stars.sh` | Star glyphs, as the font actually paints them |

Each is self-contained: run it, look, decide.

There was a `shapes.sh` here too, for choosing the ring's proportions. It is gone
now that the shape is settled — it needed a preview renderer living in the test
tree (to reach the package's unexported geometry), which meant a second copy of
the layout code that could silently drift from the real one. The trade it
illustrated is recorded in ROSARY-TUI.md instead, under "The ring's shape is a
trade"; rebuild the preview from there if the question ever reopens.
