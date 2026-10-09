#!/bin/bash
# Show the candidate styles for the opening's Our Lady, one at a time, so the
# look can be chosen in a real terminal with a real font — braille especially
# depends on the font, and no screenshot settles it.
#
#   scripts/art-options.sh            step through every option (enter = next)
#   scripts/art-options.sh C          just option C
#   SIZE=40x24 scripts/art-options.sh  a different size (default 48x30)
#
# Needs ImageMagick and chafa. The preprocessing matches opening-art.sh; this
# script only renders, it never touches internal/rosary/art.
set -euo pipefail

cd "$(dirname "$0")/.."
src=demo/our_lady_4.jpg
size=${SIZE:-48x30}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# The painting as it is, cropped to the figure.
magick "$src" -crop 679x900+0+0 +repage "$tmp/crop.png"

# The masked painting: the starfield removed, the edges transparent. Same steps
# as opening-art.sh.
magick -size 679x900 xc:black -fill white \
  -draw "ellipse 339,470 330,460 0,360" -blur 0x50 "$tmp/ellipse.png"
magick "$tmp/crop.png" -colorspace gray -level 8%,35% "$tmp/lum.png"
magick "$tmp/ellipse.png" "$tmp/lum.png" -compose lighten -composite \
  "$tmp/ellipse.png" -compose multiply -composite "$tmp/mask.png"
magick "$tmp/crop.png" -statistic median 5 -modulate 108,120 \
  "$tmp/mask.png" -alpha off -compose copy_opacity -composite "$tmp/masked.png"

gold=$'\033[38;2;201;162;39m'
blue=$'\033[38;2;91;141;214m'
off=$'\033[0m'

# U+2800, the EMPTY braille pattern, is swapped for a plain space: many fonts draw
# it as faint dots, which covers the whole picture in a grid.
c() { chafa -f symbols --polite on -w 9 --size "$size" "$@" | sed 's/⠀/ /g'; }

# Each option: a letter, a name, and how to draw it.
option() {
  case "$1" in
    A) c --symbols braille -c none "$src" ;;
    B) c --symbols braille -c none -t 0.5 "$tmp/masked.png" ;;
    C) c --symbols braille -c none -t 0.5 "$tmp/masked.png" |
         sed -E $'s/\x1b\\[[0-9;?]*[A-Za-z]//g' | while IFS= read -r l; do printf '%s%s%s\n' "$gold" "$l" "$off"; done ;;
    D) c --symbols braille --fg-only -c full -t 0.5 "$tmp/masked.png" ;;
    E) c --symbols block+space -c full -t 0.5 "$tmp/masked.png" ;;
    F) c --symbols block+border+space-wide -c full "$tmp/crop.png" ;;
    G) c --symbols block+braille+space -c full -t 0.5 "$tmp/masked.png" ;;
    H) c --symbols ascii -c full --fg-only -t 0.5 "$tmp/masked.png" ;;
  esac
}

name() {
  case "$1" in
    A) echo "Braille, mono, whole painting  (the original art.txt)" ;;
    B) echo "Braille, mono, starfield masked" ;;
    C) echo "Braille, tinted gold, starfield masked" ;;
    D) echo "Braille, full colour, starfield masked" ;;
    E) echo "Blocks, full colour, starfield masked  (current)" ;;
    F) echo "Blocks, full colour, whole painting in its rectangle" ;;
    G) echo "Blocks + braille mixed, full colour, masked" ;;
    H) echo "ASCII characters, full colour, masked" ;;
  esac
}

words() {
  printf '\n%s%s%s\n\n%s\033[3m%s%s\n' "$gold" "        A V E   M A R I A" "$off" "$blue" "          ora pro nobis" "$off"
}

show() {
  clear
  printf '\033[1m%s  %s\033[0m\n\n' "$1" "$(name "$1")"
  option "$1"
  words
}

if [[ $# -gt 0 ]]; then
  show "$1"
  exit
fi

for o in A B C D E F G H; do
  show "$o"
  [[ $o == H ]] || read -rp $'\n  enter for the next option… ' _
done
