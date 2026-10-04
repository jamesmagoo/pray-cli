#!/bin/bash
# Print the small-bead candidates in real colour. Run this and your terminal
# paints them, unlike output quoted into a chat transcript.
beads() {  # $1 = r;g;b, $2 = faint?
  local f=""; [ "$2" = faint ] && f="2;"
  printf "\033[${f}38;2;${1}m○ ○ ○ ○ ○ ○\033[m"
}
gold() { printf "\033[38;2;201;162;39m●\033[m"; }

printf "\n  small beads, as drawn (Faint):\n\n"
while IFS='|' read -r name rgb hex; do
  printf "  %-26s " "$name"
  beads "$rgb" faint; printf "  "; gold; printf "  "; beads "$rgb" faint
  printf "   %s\n" "$hex"
done <<ROWS
current lavender (G)|143;169;208|#8FA9D0
G/ a shade lighter|157;175;216|#9DAFD8
G+ lighter|165;184;220|#A5B8DC
G++ lightest|180;194;228|#B4C2E4
B cornflower (rejected)|91;141;214|#5B8DD6
old gold|201;162;39|#C9A227
ROWS

printf "\n  the same, WITHOUT Faint (full strength):\n\n"
while IFS='|' read -r name rgb hex; do
  printf "  %-26s " "$name"
  beads "$rgb"; printf "  "; gold; printf "  "; beads "$rgb"
  printf "   %s\n" "$hex"
done <<ROWS
current lavender (G)|143;169;208|#8FA9D0
G/ a shade lighter|157;175;216|#9DAFD8
G+ lighter|165;184;220|#A5B8DC
G++ lightest|180;194;228|#B4C2E4
ROWS
printf "\n"
