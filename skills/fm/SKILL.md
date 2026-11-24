---
name: fm
description: Query and batch-edit YAML frontmatter (properties) in Markdown files, e.g. an Obsidian vault, and in header comments of source files (scripts, code) with the fm CLI. Use when the user wants to list, filter, add, change, retype, rename or remove frontmatter fields across many files.
---

# fm — query and batch-edit Markdown frontmatter

`fm` treats files as rows and their frontmatter fields as columns, and runs
SQL-like queries over them. Frontmatter is the YAML block at the top of a
Markdown file, or a YAML block in a comment at the top of a source file. Translate the user's request into `fm` queries.

## Before you start

- Check that `fm` is installed: `fm --version`. If missing, tell the user to
  install it with `brew install nofuss-io/tap/fm` (macOS/Linux),
  `go install github.com/nofuss-io/frontmatter/cmd/fm@latest`, or a binary
  from https://github.com/NoFuss-io/frontmatter/releases.
- `fm` edits files in place with no backup. Prefer working in a git repo, or
  suggest the user commits or backs up first.

## Workflow

1. **Explore** the data: `fm 'select * from <glob> limit 10'` shows which fields
   exist and how values look. Ask if the target files are ambiguous.
2. **Preview** the change: run the same `where` as a `select`, or add
   `--dry-run --verbose` to show the result without writing.
3. **Execute** the mutation.
4. **Verify**: re-run a `select`, or use `--verbose` to print affected fields.

Pass the whole query as one single-quoted shell argument. Use double quotes
inside for string literals. Separate several statements with `;`, or pipe a
script on stdin (`fm < script.sql`).

## Statements

```
select <expr>[, <expr>]... from <glob>... [where <cond>] [sort by <expr> [asc|desc], ...] [limit <n>]
update <glob>... set <assignment>[, <assignment>]... [where <cond>]
alter  <glob>... drop <field>[, <field>]... [where <cond>]
alter  <glob>... rename <field> to <field>[, ...] [where <cond>]
```

Globs: `notes/*.md`, `**/*.md`. A glob without an extension (`scripts/*`)
matches every supported file type (`.md`, `.py`, `.sh`, `.js`, …). Hidden
files are skipped unless `-H`.

## Fields and types

`field` or `field:type`. Quote odd names with backticks: `` `created-at`:date ``.
Field names are case sensitive; keywords and functions are not.

Types: `any` (default), `string`, `int`, `numeric`, `bool`, `date`, `datetime`,
`list`, `link` (`[[ref|title]]`), `mdlink` (`[title](ref)`).

A missing field, or a value that cannot be cast to the type, reads as `null`.

## Conditions

- `field` — truthy (exists, not null/false/0)
- `not field` — missing, null or falsy
- `field = "x"`, `field != "x"`
- `<`, `<=`, `>`, `>=` — numbers, booleans, dates and datetimes (not strings);
  comparing typed fields or literals that can never be ordered is a parse error
- Dates are quoted strings: `created >= "2025-01-01"`,
  `modified > "2025-01-01T09:00:00"`, `due < today()`. Unquoted
  `2025-01-01` is arithmetic (2023).
- `tags >= "recipe"` — list contains value
- `and`, `or`, `not`, parentheses

## Assignments

| Form              | Effect                                          |
|:------------------|:------------------------------------------------|
| `field:type`      | Cast to type (create as null if missing)        |
| `field = expr`    | Set value; `field:type = expr` also sets type   |
| `field = null`    | Clear value (keeps the field)                   |
| `field += value`  | Add element(s) to list (casts scalar to list)   |
| `field -= value`  | Remove element(s) from list                     |

Expressions support `+ - * /`, string literals (`"..."`, `'...'`) and
functions, e.g. `lower`, `replace`, `split`, `concat`, `regexp_contains`,
`coalesce`, `array_length`, `array_contains`, `today`, `date_diff`.

## Examples

```sh
# Overview of fields in a folder
fm 'select * from notes/*.md limit 10'

# Filter, sort and limit
fm 'select title, tags from notes/*.md where published sort by date desc limit 10'

# Notes with a tag
fm 'select title from notes/*.md where tags >= "recipe"'

# Notes created this year, newest first
fm 'select title, created from notes/*.md where created >= "2025-01-01" sort by created desc'

# Notes missing a field
fm 'select title from notes/*.md where not status'

# Normalise types (creates the field as null where missing)
fm 'update notes/*.md set date:date, rating:int'

# Conditional set
fm 'update notes/*.md set status = "published" where status = "draft"'

# Add a tag to a list
fm 'update notes/*.md set tags += "cooking" where category = "recipe"'

# Rename and drop fields
fm 'alter notes/*.md rename veggie to vegetarian'
fm 'alter notes/*.md drop draft where published'

# Preview before writing: simulate and show the resulting values
fm --dry-run --verbose 'update notes/*.md set tags -= "todo"'
```

## Source files

Frontmatter can also live in a comment at the top of a source file. The comment
style is chosen per file by extension (or shebang), and the block may follow a
shebang or encoding line:

```python
#!/usr/bin/env python3
# ---
# author: jane.doe
# description: exports sessions to Excel
# ---
```

Other styles: `// ---` (JS, Go, …), `/*---` … `---*/`, `"""---` … `---"""`,
`-- ---` (SQL). The YAML is always fenced by `---` inside the comment.

```sh
# Find scripts by metadata
fm 'select author, description from scripts/*.py where author = "jane.doe"'

# Extensionless scripts or unknown extensions: force the syntax for all files
fm -L shell 'select description from bin/*'
fm --block-open '{{/*' --block-close '*/}}' 'select * from templates/*.tmpl'
```

`update` only adds a new block to a source file that has none if the file is
named explicitly or a syntax flag (`-L`, `--line-comment`, `--block-open`) is
given. Glob-matched files without a block are reported as errors and left
untouched.

## Notes

- Fields are sorted alphabetically when a file is written.
- Parse and type errors stop `fm` before any file is touched. Runtime errors
  (e.g. a failed cast) skip that file and continue with the others.
- When a query fails or something is unclear, read the reference that ships
  with the installed version: `fm help syntax` (query language) or
  `fm help functions` (built-in functions). Prefer it over memory.
