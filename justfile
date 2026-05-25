_default: dev

help:
    just --list

commit := `git rev-parse --short HEAD`

build:
    go build -ldflags "-X main.Commit={{ commit }}" -o fm ./cmd/fm

lint:
    go fmt ./...
    go vet ./...
    staticcheck ./...

test FLAGS="":
    go test ./internal/... {{ FLAGS }}
    go test -count=1 ./test/... {{ FLAGS }}

new-integration-test NAME:
    mkdir -p test/integration/cases/{{ NAME }}/input
    touch test/integration/cases/{{ NAME }}/input/a.md
    touch test/integration/cases/{{ NAME }}/input/b.md
    touch test/integration/cases/{{ NAME }}/cmd
    touch test/integration/cases/{{ NAME }}/expected
    touch test/integration/cases/{{ NAME }}/expected_stderr

vendor:
    go mod tidy
    go mod vendor

dev *args:
    go run ./cmd/fm {{ args }}

install:
    go build -ldflags "-X main.Commit={{ commit }}" -o $(go env GOPATH)/bin/fm ./cmd/fm

install-skill: install
    mkdir -p ~/.claude/skills/fm/docs
    cp SKILL.md ~/.claude/skills/fm/SKILL.md
    cp docs/manual.md ~/.claude/skills/fm/docs/manual.md
