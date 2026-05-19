_default: dev

help:
    just --list

build:
    go build -o fm ./cli

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
    go run ./cli {{ args }}

install:
    go build -o $(go env GOPATH)/bin/fm ./cli

install-skill: install
    mkdir -p ~/.claude/skills/fm
    cp SKILL.md ~/.claude/skills/fm/SKILL.md
