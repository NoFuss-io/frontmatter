_default: dev

help:
    just --list

build:
    go build -o fm ./cmd/fm

lint:
    go fmt ./...
    go vet ./...
    staticcheck ./...

test:
    go test ./...

vendor:
    go mod tidy
    go mod vendor

dev *args:
    go run ./cmd/fm {{ args }}

install:
    go build -o $(go env GOPATH)/bin/fm ./cmd/fm

install-skill: install
    mkdir -p ~/.claude/skills/fm
    cp SKILL.md ~/.claude/skills/fm/SKILL.md
