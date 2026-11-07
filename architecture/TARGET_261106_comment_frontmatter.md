baseline = 368d7d4

# Target state — frontmatter in source-file comments

Resolves [issue #21](https://github.com/NoFuss-io/frontmatter/issues/21).

`fm` reads and writes YAML frontmatter that lives inside a comment in a
non-Markdown file (scripts, source code, config). The block is found by
looking up the file's comment syntax by extension, stripping the comment
markers, and parsing the remainder as YAML exactly like Markdown frontmatter.
Writes re-emit the block in the same comment style and leave every other byte
of the file untouched.

Markdown behaviour does not change.

Acceptance: the issue's examples in `store/markdown/markdown_test.go`
(`commentCases`) pass for both read and write.

---

## Mental model

```
file ──ext──▶ language row ──▶ candidate fence styles
                                   │
          scan lines top-down ◀────┘
                   │ first valid block
                   ▼
   prefix | open | YAML (markers stripped) | close | suffix
       └── kept verbatim ──┘                   └── kept verbatim
```

A file is split into three segments: everything before the block, the block,
everything after. Read parses the block. Write re-renders only the block, in
the style it was read in, and concatenates the three segments.

---

## Language table

One row per language. A row may have a line comment, one or more block
comments, or both. Extensions are matched case-insensitively.

| Language   | Extensions                                         | Line   | Block                         |
|:-----------|:---------------------------------------------------|:-------|:------------------------------|
| Markdown   | `.md` `.markdown`                                  | —      | — (bare `---`, see below)     |
| Python     | `.py` `.pyi`                                       | `#`    | `"""` `"""`, `'''` `'''`      |
| Shell      | `.sh` `.bash` `.zsh` `.fish`                       | `#`    | —                             |
| Ruby       | `.rb`                                              | `#`    | `=begin` `=end`               |
| Perl       | `.pl` `.pm`                                        | `#`    | —                             |
| R          | `.r`                                               | `#`    | —                             |
| YAML/TOML  | `.yml` `.yaml` `.toml`                             | `#`    | —                             |
| PowerShell | `.ps1`                                             | `#`    | `<#` `#>`                     |
| C family   | `.c` `.h` `.cpp` `.hpp` `.cs` `.java` `.kt` `.go` `.rs` `.swift` `.scala` `.dart` | `//` | `/*` `*/` |
| JavaScript | `.js` `.mjs` `.cjs` `.jsx` `.ts` `.tsx`            | `//`   | `/*` `*/`                     |
| PHP        | `.php`                                             | `//`, `#` | `/*` `*/`                  |
| CSS        | `.css`                                             | —      | `/*` `*/`                     |
| SCSS/Less  | `.scss` `.less`                                    | `//`   | `/*` `*/`                     |
| SQL        | `.sql`                                             | `--`   | `/*` `*/`                     |
| Lua        | `.lua`                                             | `--`   | `--[[` `]]`                   |
| Haskell    | `.hs`                                              | `--`   | `{-` `-}`                     |
| HTML/XML   | `.html` `.htm` `.xml` `.svg`                       | —      | `<!--` `-->`                  |
| TeX        | `.tex`                                             | `%`    | —                             |
| Lisp       | `.el` `.lisp` `.clj`                               | `;`    | —                             |
| INI        | `.ini`                                             | `;`, `#` | —                           |
| Vim        | `.vim`                                             | `"`    | —                             |

The table is data (a Go slice), not code, so adding a language is a one-line
change.

Files with no extension: if the first line is a shebang, the interpreter name
(`python3`, `bash`, `node`, `ruby`, …) selects the row. Otherwise, and for
unknown extensions, Markdown rules apply (today's behaviour).

---

## Recognised block shapes

The fence is exactly `---` (trailing whitespace allowed). Longer runs such as
`----------` (reST underlines, separator comments) are not fences.

### Line comment

```
# ---
# key: value
#
# other: value
# ---
```

Every line from opener to closer starts with the line marker. One space after
the marker is stripped if present. A line consisting of the bare marker is an
empty YAML line. A line without the marker ends the candidate as invalid.

### Block comment

Fence on the marker line, or on its own line — both are accepted:

```
/*---              /*
key: value         ---
---*/              key: value
                   ---
                   */
```

Ruby `=begin`/`=end` only allows the second shape (the markers must be alone on
their line).

### Markdown

Bare `---` on line 1, closing `---` on its own line. Top of file only, never
mid-file: a mid-file `---` in Markdown is a horizontal rule or setext heading.

---

## Locating the block

1. Scan lines top-down.
2. At each line, test every fence style the language row offers.
3. A candidate is valid only if a closing fence is found and the stripped
   content parses as a YAML mapping (or is empty). A candidate that parses as a
   scalar or list, or fails to parse, is skipped and the scan continues.
4. The first valid candidate wins. Later blocks are ignored on read and left
   untouched on write.
5. No valid candidate: empty field map, whole file is the prefix.

Rule 3 is what makes "anywhere in the file" safe: separator comments like
`# ---\n# Helpers\n# ---` parse as a scalar and are skipped.

---

## Write

- Re-render the YAML with the existing encoder (keys sorted, dates kept as
  dates — same as Markdown).
- Re-apply the style that was read: same markers, same fence shape, same
  "space after marker" choice, same indentation.
- Prefix and suffix are written back byte-for-byte.
- The file's original line ending style is preserved.

### File without a block

Insert a new block at the top, after the preamble, using the row's line
comment (block comment if the row has none). Preamble = a leading shebang
line, plus immediately following comment lines that are encoding or magic
comments (`# -*- coding: … -*-`, `# frozen_string_literal: …`,
`// @ts-check`, …). Markdown rows insert at line 1, as today.

---

## Out of scope

- More than one block per file (see "Locating the block", rule 4).
- Shebang detection in globs. `FROM 'dir/*'` matches every extension in the
  language table, but extensionless scripts must be named explicitly.

---

## Implementation notes

- The split lives in `store/markdown.Document` (`Head`, `LinePrefix`, `Body`,
  `LineEndings`). It cannot live in `internal/` because `internal/exec.go`
  imports `store/markdown`.
- `Head` and `Body` are kept byte-for-byte; `LineEndings` (taken from the
  opening fence line) only applies to the rendered block, so files with mixed
  line endings survive a write.
- Invalid YAML in a Markdown block is still an error. In other files an
  invalid candidate is skipped.
- Line markers ending in `-` (`--`) require a space before the fence, so a
  `-----` separator is never a fence.
- An empty field map renders as two fences with nothing between them, instead
  of `{}`.
- Overrides: `-L/--lang` (name or extension), or `--line-comment` and/or
  `--block-open`/`--block-close` (closer defaults to opener). Without them
  the syntax is chosen per file; with any of them it applies to every file
  and `*` matches every file. `--lang` cannot be combined with the others.
