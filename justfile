_default: dev

help:
    just --list

setup:
    git config core.hooksPath .githooks
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
    cargo install lychee

commit := `git rev-parse --short HEAD`
raw_version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
version := trim_start_match(raw_version, "v")
ldflags := "-X main.Version=" + version + " -X main.Commit=" + commit

build:
    go build -ldflags "{{ ldflags }}" -o fm ./cmd/fm

lint:
    go fmt ./...
    golangci-lint run ./...

lint-fix:
    go fmt ./...
    golangci-lint run --fix ./...

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

man:
    mkdir -p docs/man
    pandoc -s -t man docs/fm.1.md -o docs/man/fm.1

completions: build
    mkdir -p completions
    ./fm completion bash > completions/fm.bash
    ./fm completion zsh  > completions/fm.zsh
    ./fm completion fish > completions/fm.fish

release-check:
    goreleaser check

release-snapshot:
    goreleaser release --snapshot --clean

# Check that http(s) links in docs and Go source resolve.

# Requires `lychee` (https://github.com/lycheeverse/lychee).
check-links:
    lychee --no-progress \
        README.md SECURITY.md CONTRIBUTING.md SKILL.md \
        docs/ architecture/ .github/ frontmatter.go \
        --exclude-path docs/tutorial/
