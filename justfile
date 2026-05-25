_default: dev

help:
    just --list

commit  := `git rev-parse --short HEAD`
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

ldflags := "-X main.Version=" + version + " -X main.Commit=" + commit

build:
    go build -ldflags "{{ ldflags }}" -o fm ./cmd/fm

lint:
    go fmt ./...
    golangci-lint run ./...

test FLAGS="":
    go test ./internal/... {{ FLAGS }}
    go test -count=1 ./test/... {{ FLAGS }}

new-e2e-test NAME:
    mkdir -p test/e2e/cases/{{ NAME }}/input
    touch test/e2e/cases/{{ NAME }}/input/a.md
    touch test/e2e/cases/{{ NAME }}/input/b.md
    touch test/e2e/cases/{{ NAME }}/cmd
    touch test/e2e/cases/{{ NAME }}/expected
    touch test/e2e/cases/{{ NAME }}/expected_stderr

vendor:
    go mod tidy
    go mod vendor

dev *args:
    go run ./cmd/fm {{ args }}

install:
    go build -ldflags "{{ ldflags }}" -o $(go env GOPATH)/bin/fm ./cmd/fm

install-skill: install
    mkdir -p ~/.claude/skills/fm/docs
    cp SKILL.md ~/.claude/skills/fm/SKILL.md
    cp docs/manual.md ~/.claude/skills/fm/docs/manual.md
