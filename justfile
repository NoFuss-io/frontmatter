default: build

build:
    go build -o fm ./cli

lint:
    go fmt ./...
    go vet ./...
    staticcheck ./...

vendor:
    go mod tidy
    go mod vendor

dev *args:
    go run ./cli {{args}}

install:
    go build -o $(go env GOPATH)/bin/fm ./cli

install-skill: install
    mkdir -p ~/.claude/skills/fm
    cp SKILL.md ~/.claude/skills/fm/SKILL.md

shell-completion:
    @echo 'Bash — add to ~/.bashrc:'
    @echo '  eval "$(fm completion bash)"'
    @echo ''
    @echo 'Zsh — add to ~/.zshrc:'
    @echo '  eval "$(fm completion zsh)"'
    @echo ''
    @echo 'Fish — add to ~/.config/fish/completions/fm.fish:'
    @echo '  fm completion fish | source'
    @echo ''
    @echo 'PowerShell — add to $PROFILE:'
    @echo '  fm completion powershell | Out-String | Invoke-Expression'
