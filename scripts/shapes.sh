#!/bin/bash
# Show the rosary at each candidate ring shape, in colour.
#
# The shape is a trade: a wider ring is flat at top and bottom (beads land in a
# straight run), a taller one breaks the chain (rows with no bead on them). Both
# measurements print under each heading. See scripts/README.md.
set -e
cd "$(dirname "$0")/.."
go test ./internal/rosary/ -run TestPreviewShapes -v 2>&1 \
  | sed -e '1d' -e '/^--- PASS/,$d' -e '/^=== RUN/d' -e '/^ok /d'
