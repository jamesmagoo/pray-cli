bin := "bin/pray"

# list available recipes
default:
    @just --list

# build the binary
build:
    go build -o {{bin}} ./cmd/pray

# run the cli, passing through args, e.g. `just run hail mary --for "Mam & Dad"`
# (positional-arguments passes each arg through untouched, quotes and all)
[positional-arguments]
run *args:
    go run ./cmd/pray "$@"

# run tests
test:
    go test ./... -v

# print every prayer and its metadata (id, title, lang, aliases, tags, body)
debug:
    go run ./internal/prayers/cmd/debug

# run tests with coverage report
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# format code
fmt:
    gofmt -l -w .

# vet code
vet:
    go vet ./...

# tidy go.mod/go.sum
tidy:
    go mod tidy

# fmt, vet, and test
check: fmt vet test

# remove build artifacts
clean:
    rm -rf bin coverage.out coverage.html

# install the binary to $GOPATH/bin
install:
    go install ./cmd/pray

# record the demo: demo/rosary.gif and demo/rosary.webm
#
# The tape drives the `pray` on PATH, so install first.
#
# The tape renders at 1808x1288 — about twice the size the README shows — because
# VHS rasterises at exactly the size it is given, with no supersampling. A small
# font there is genuinely low-resolution, and no amount of compression tuning
# afterwards puts the detail back. So: render big, then scale down with lanczos.
# The result is sharper than rendering small ever was.
#
# That leaves the webm at full size (it is small anyway, and a website can use
# the real resolution) and the gif at half, which is what the README embeds.
#
# stats_mode=full, NOT diff. diff weights colours by what changes between frames,
# and what changes most here is the near-black background — it spent the whole
# palette on greys, turned the cornflower beads grey and washed the gold out to
# cream. full looks at every frame whole, which keeps both.
#
# Needs vhs and ffmpeg (brew install vhs ffmpeg).
demo: install
    vhs demo/rosary.tape
    ffmpeg -y -i demo/rosary.gif \
        -vf "scale=904:-1:flags=lanczos,palettegen=max_colors=128:stats_mode=full" \
        -frames:v 1 -update 1 demo/.palette.png -loglevel error
    ffmpeg -y -i demo/rosary.gif -i demo/.palette.png \
        -filter_complex "scale=904:-1:flags=lanczos[x];[x][1:v]paletteuse=dither=none:diff_mode=rectangle[o]" \
        -map "[o]" demo/.rosary-opt.gif -loglevel error
    mv demo/.rosary-opt.gif demo/rosary.gif
    rm -f demo/.palette.png
    @ls -lh demo/rosary.gif demo/rosary.webm
