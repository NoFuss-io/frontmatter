package markdown

import (
	"path/filepath"
	"regexp"
	"strings"
)

// blockComment is a comment delimited by an opening and closing marker.
type blockComment struct {
	open, close string
}

// language describes how frontmatter can be embedded in one kind of file.
type language struct {
	name string
	exts []string
	// line lists line-comment markers. The first is used when inserting a new
	// block into a file that has none.
	line []string
	// block lists block-comment delimiters. The first is used for insertion
	// when the language has no line comment.
	block []blockComment
	// markdown selects plain "---" fences, recognised on the first line only.
	markdown bool
}

var (
	cBlock  = blockComment{"/*", "*/"}
	docstrs = []blockComment{{`"""`, `"""`}, {"'''", "'''"}}
)

// languages is the extension table. Extensions are matched case-insensitively.
var languages = []language{
	{name: "markdown", exts: []string{".md", ".markdown"}, markdown: true},
	{name: "python", exts: []string{".py", ".pyi"}, line: []string{"#"}, block: docstrs},
	{name: "shell", exts: []string{".sh", ".bash", ".zsh", ".fish"}, line: []string{"#"}},
	{name: "ruby", exts: []string{".rb"}, line: []string{"#"}, block: []blockComment{{"=begin", "=end"}}},
	{name: "perl", exts: []string{".pl", ".pm"}, line: []string{"#"}},
	{name: "r", exts: []string{".r"}, line: []string{"#"}},
	{name: "config", exts: []string{".yml", ".yaml", ".toml"}, line: []string{"#"}},
	{name: "powershell", exts: []string{".ps1"}, line: []string{"#"}, block: []blockComment{{"<#", "#>"}}},
	{name: "c", exts: []string{
		".c", ".h", ".cpp", ".hpp", ".cs", ".java", ".kt", ".go", ".rs", ".swift", ".scala", ".dart",
		".js", ".mjs", ".cjs", ".jsx", ".ts", ".tsx", ".scss", ".less",
	}, line: []string{"//"}, block: []blockComment{cBlock}},
	{name: "php", exts: []string{".php"}, line: []string{"//", "#"}, block: []blockComment{cBlock}},
	{name: "css", exts: []string{".css"}, block: []blockComment{cBlock}},
	{name: "sql", exts: []string{".sql"}, line: []string{"--"}, block: []blockComment{cBlock}},
	{name: "lua", exts: []string{".lua"}, line: []string{"--"}, block: []blockComment{{"--[[", "]]"}}},
	{name: "haskell", exts: []string{".hs"}, line: []string{"--"}, block: []blockComment{{"{-", "-}"}}},
	{name: "html", exts: []string{".html", ".htm", ".xml", ".svg"}, block: []blockComment{{"<!--", "-->"}}},
	{name: "tex", exts: []string{".tex"}, line: []string{"%"}},
	{name: "lisp", exts: []string{".el", ".lisp", ".clj"}, line: []string{";"}},
	{name: "ini", exts: []string{".ini"}, line: []string{";", "#"}},
	{name: "vim", exts: []string{".vim"}, line: []string{`"`}},
}

// interpreters maps a shebang interpreter (version suffix stripped) to a
// language name, for files without an extension.
var interpreters = map[string]string{
	"python": "python", "bash": "shell", "sh": "shell", "zsh": "shell", "fish": "shell",
	"dash": "shell", "ksh": "shell", "ruby": "ruby", "perl": "perl", "Rscript": "r",
	"node": "c", "deno": "c", "bun": "c", "lua": "lua", "php": "php",
}

var (
	byExt  = map[string]language{}
	byName = map[string]language{}
)

func init() {
	for _, l := range languages {
		byName[l.name] = l
		for _, e := range l.exts {
			byExt[e] = l
		}
	}
}

// Extensions lists every file extension in the language table, so that an
// extensionless glob such as `*` matches all supported files.
func Extensions() []string {
	var out []string
	for _, l := range languages {
		out = append(out, l.exts...)
	}
	return out
}

var versionSuffix = regexp.MustCompile(`[0-9.]+$`)

// languageFor picks the language row for a file: by extension, else by
// shebang interpreter for extensionless files, else Markdown.
func languageFor(path, content string) language {
	ext := strings.ToLower(filepath.Ext(path))
	if l, ok := byExt[ext]; ok {
		return l
	}
	if ext == "" {
		if name := shebangInterpreter(content); name != "" {
			if l, ok := byName[interpreters[versionSuffix.ReplaceAllString(name, "")]]; ok {
				return l
			}
		}
	}
	return byName["markdown"]
}

// shebangInterpreter returns the interpreter's base name from a "#!" first
// line, looking through "/usr/bin/env [-S] name".
func shebangInterpreter(content string) string {
	first, _, _ := strings.Cut(content, "\n")
	rest, ok := strings.CutPrefix(strings.TrimRight(first, "\r"), "#!")
	if !ok {
		return ""
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	name := filepath.Base(fields[0])
	if name == "env" {
		for _, f := range fields[1:] {
			if !strings.HasPrefix(f, "-") {
				return filepath.Base(f)
			}
		}
		return ""
	}
	return name
}

// magicComment matches lines that must stay above an inserted block:
// encoding declarations, editor modelines and similar pragmas.
var magicComment = regexp.MustCompile(`^(#|//)\s*(-\*-.*-\*-|.*coding[:=]|frozen_string_literal:|@ts-(no)?check)`)
