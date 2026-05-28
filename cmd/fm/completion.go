package main

import (
	"fmt"
	"io"
)

const completionBash = `# bash completion for fm
_fm_complete() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "--help --dry-run --hidden --max-columns" -- "$cur") )
        return
    fi
    COMPREPLY=( $(compgen -f -- "$cur") )
}
complete -o default -F _fm_complete fm
`

const completionZsh = `#compdef fm
_fm() {
    _arguments \
        '(-h --help)'{-h,--help}'[show help]' \
        '(-d --dry-run)'{-d,--dry-run}'[simulate without writing]' \
        '(-H --hidden)'{-H,--hidden}'[include hidden files]' \
        '--max-columns[column cap for select *]:N:' \
        '*:files:_files -g "*.md"'
}
compdef _fm fm
`

const completionFish = `# fish completion for fm
complete -c fm -s h -l help -d 'show help'
complete -c fm -s d -l dry-run -d 'simulate without writing'
complete -c fm -s H -l hidden -d 'include hidden files'
complete -c fm -l max-columns -d 'column cap for select *' -x
complete -c fm -F
`

func writeCompletion(shell string, out io.Writer) error {
	switch shell {
	case "bash":
		_, err := io.WriteString(out, completionBash)
		return err
	case "zsh":
		_, err := io.WriteString(out, completionZsh)
		return err
	case "fish":
		_, err := io.WriteString(out, completionFish)
		return err
	default:
		return fmt.Errorf("unknown shell %q (supported: bash, zsh, fish)", shell)
	}
}
