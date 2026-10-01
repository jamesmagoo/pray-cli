bin := "bin/pray"

# list available recipes
default:
    @just --list

# build the binary
build:
    go build -o {{bin}} ./cmd/pray

# run the cli, passing through args, e.g. `just run pray`
run *args:
    go run ./cmd/pray {{args}}

# run tests
test:
    go test ./...

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
