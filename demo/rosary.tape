# A few steps through the rosary: choose the mysteries, begin at the crucifix,
# and arrive at the first decade, where the mystery is announced inside the ring.
#
#   vhs demo/rosary.tape
#
# `pray` must be on PATH (just install). The terminal must be at least 98x34
# cells or the TUI shows its "window too small" notice instead — see
# internal/rosary/toosmall.go.

Output demo/rosary.gif

# Also a video: GIF is capped at 256 colours a frame, where webm keeps the full
# palette and lands far smaller. The README uses the gif, since GitHub will not
# play video in markdown; a website should prefer the webm.
Output demo/rosary.webm

Set Shell zsh

# Rows and Columns fix the terminal GRID; Width and Height are then just the
# pixel size that grid is drawn at. Setting them explicitly is what guarantees
# the rosary fits: it needs at least 98x34 and draws a "window too small" notice
# below that (internal/rosary/toosmall.go).
Set Columns 100
Set Rows 36

# With Columns and Rows set, the pixel size follows from the font size — VHS
# refuses a tape that sets both the grid and Width/Height.
#
# 28 is deliberately large: VHS rasterises at exactly this size with no
# supersampling, so a small font is genuinely low-resolution. Rendering at about
# twice the displayed size keeps the text sharp on a normal screen and crisp on
# a retina one.
Set FontSize 28

Set Padding 40
Set BorderRadius 12
Set Framerate 24
Set TypingSpeed 60ms
Set Theme { "name": "pray", "background": "#16161A", "foreground": "#D8D6D1", "black": "#16161A", "red": "#C76A6A", "green": "#8FA86B", "yellow": "#C9A227", "blue": "#5B8DD6", "magenta": "#A98BC4", "cyan": "#6FAFA6", "white": "#D8D6D1", "brightBlack": "#5C5C66", "brightRed": "#D98585", "brightGreen": "#A6BE85", "brightYellow": "#E0BC4A", "brightBlue": "#7BA5E3", "brightMagenta": "#BFA5D6", "brightCyan": "#8CC5BC", "brightWhite": "#F2F0EB" }

Hide
Type "export PS1=''"
Enter
Type "clear"
Enter
Show

Type "pray rosary"
Sleep 400ms
Enter

# The opening screen: pick the mysteries.
Sleep 1200ms
Down
Sleep 700ms
Up
Sleep 700ms

# The crucifix: the Sign of the Cross, then the Creed.
Space
Sleep 1300ms
Space
Sleep 1500ms

# Up the pendant.
Space
Sleep 1000ms
Space
Sleep 800ms

# Onto the ring: the Our Father opens the first decade and the mystery appears.
Space
Sleep 1100ms
Space
Sleep 1900ms

# The first Hail Mary of the decade.
Space
Sleep 1600ms
