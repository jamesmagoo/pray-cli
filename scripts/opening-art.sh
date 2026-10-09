#!/bin/bash
# Render the opening screen's Our Lady from demo/our_lady_4.jpg into the two
# embedded art files the rosary draws: internal/rosary/art/our-lady-{large,small}.txt
#
#   just art        (or: scripts/opening-art.sh)
#
# Needs ImageMagick and chafa (brew install imagemagick chafa).
#
# The steps, and why each is there:
#
#   1. Crop to the halo, face and hands. The painting cuts the halo's crown flat
#      at its top edge, so the ellipse in step 2 starts just inside it and rounds
#      the corners off.
#   2. Mask the background away. Two masks, multiplied: a soft ellipse around the
#      figure, and the image's own brightness, so the dark starfield falls to
#      nothing while the bright halo and veil stay. Masked pixels become
#      TRANSPARENT, which chafa leaves blank — without this the stars fill the
#      whole rectangle with stray dots.
#   3. A small median filter. The painting is encrusted with jewels, which at
#      terminal resolution is noise; smoothing it first keeps the face legible.
#   4. chafa in braille, with NO colour: plain text. The app tints it gold
#      itself (internal/rosary/opening.go), so one colour carries the whole
#      picture — it reads the same in 16 colours, 256 or truecolour, and as an
#      outline with colour off. Chosen over full-colour blocks for exactly that;
#      scripts/art-options.sh shows the alternatives.
#   5. U+2800, the EMPTY braille pattern, becomes a plain space. Many fonts draw
#      it as faint dots, which covers the picture in a grid.
set -euo pipefail

cd "$(dirname "$0")/.."
src=demo/our_lady_4.jpg
out=internal/rosary/art
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

magick "$src" -crop 679x900+0+0 +repage "$tmp/crop.png"

magick -size 679x900 xc:black -fill white \
  -draw "ellipse 339,470 330,460 0,360" -blur 0x50 "$tmp/ellipse.png"
magick "$tmp/crop.png" -colorspace gray -level 8%,35% "$tmp/lum.png"
magick "$tmp/ellipse.png" "$tmp/lum.png" -compose lighten -composite \
  "$tmp/ellipse.png" -compose multiply -composite "$tmp/mask.png"

magick "$tmp/crop.png" -statistic median 5 -modulate 108,120 \
  "$tmp/mask.png" -alpha off -compose copy_opacity -composite "$tmp/prep.png"

mkdir -p "$out"
render() { # $1 = size, $2 = name
  chafa -f symbols --symbols braille -c none --polite on -w 9 -t 0.5 \
    --size "$1" "$tmp/prep.png" |
    sed -E $'s/\x1b\\[[0-9;?]*[A-Za-z]//g' |
    sed 's/⠀/ /g; s/ *$//' > "$out/our-lady-$2.txt"
}
render 48x30 large
render 32x20 small

for f in "$out"/our-lady-*.txt; do
  printf '%s: %s rows\n' "$f" "$(wc -l < "$f" | tr -d ' ')"
done
