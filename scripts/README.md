# Design scripts

Throwaway-looking but deliberately kept: these render the rosary's shapes and
palettes **in colour, in a terminal**, which is the only place a design decision
about a TUI can honestly be made. Escape sequences do not survive into a chat
log or a diff — they have to be run.

| Script | Shows |
|---|---|
| `shapes.sh` | The rosary at each candidate ring shape, with the two measurements that matter |
| `swatch.sh` | Bead colours, with and without `Faint` |
| `stars.sh` | Star glyphs, as the font actually paints them |

Each is self-contained: run it, look, decide.

## shapes.sh

The ring's shape is a trade between two faults, and the script prints both:

- **widest flat run** — how many beads land on a single row. An ellipse is
  flattest where it crosses the vertical axis, so a wide ring puts six beads in a
  straight line across the top and bottom while the sides curve properly.
- **empty rows** — a taller ring curves better, but spaces consecutive beads more
  than one row apart, and rows with no bead read as a break in the chain.

Neither is removable; the question is only which you would rather have. See
`ringWidth` in `internal/rosary/ring.go` for the chosen answer.
