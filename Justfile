default:
    @just --list

run:
    go run ./cmd/apicker

fmt:
    gofmt -w cmd/apicker/*.go cmd/apicker/harnesses/*.go

build:
    go build -o apicker ./cmd/apicker

test:
    go test ./...

vet:
    go vet ./...

check: test vet

install:
    mkdir -p ~/.local/bin
    go build -o ~/.local/bin/apicker ./cmd/apicker
