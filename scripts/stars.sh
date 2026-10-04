#!/bin/bash
# Show the crown in each candidate star, in colour, so the glyphs can be judged
# as the terminal actually paints them.
gold='\033[38;2;224;196;120m'
off='\033[m'
arc() {  # $1 = glyph
  local g="$1"
  printf "${gold}       %s  %s %s  %s       ${off}\n" "$g" "$g" "$g" "$g"
  printf "${gold}    %s             %s    ${off}\n" "$g" "$g"
  printf "${gold}  %s                 %s  ${off}\n" "$g" "$g"
  printf "${gold} %s                   %s ${off}\n" "$g" "$g"
  printf "${gold}%s                     %s${off}\n" "$g" "$g"
}
for row in \
  "☆|U+2606 WHITE STAR (current)" \
  "★|U+2605 BLACK STAR" \
  "✦|U+2726 BLACK FOUR POINTED" \
  "✧|U+2727 WHITE FOUR POINTED" \
  "✶|U+2736 SIX POINTED BLACK" \
  "✷|U+2737 EIGHT POINTED RECTILINEAR" \
  "✹|U+2739 TWELVE POINTED BLACK" \
  "⭑|U+2B51 BLACK SMALL STAR" \
  "⭒|U+2B52 WHITE SMALL STAR" \
  "∗|U+2217 ASTERISK OPERATOR" \
  "·|U+00B7 MIDDLE DOT" \
  "•|U+2022 BULLET"
do
  g="${row%%|*}"; name="${row#*|}"
  printf "\n  === %s  %s ===\n" "$g" "$name"
  arc "$g"
done
printf "\n"
