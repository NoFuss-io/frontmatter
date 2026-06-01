---
name: fm
description: Batch editing of Markdown frontmatter in YAML using natural language
---

# fm — frontmatter refactoring

Use the `fm` CLI to query and mutate YAML frontmatter in Markdown files based on the
user's natural language request.

## Workflow

1. **Clarify** target files if the glob is ambiguous — ask before assuming.
2. **Preview** with `fm select` before any destructive mutation, unless the user's
   request is already precise.
3. **Execute** the mutation command(s).
4. Show a brief summary of what changed (e.g. `git diff --stat` or `fm select` output).

---

## Command syntax

Command uses BigQuery syntax but does not implement it in full.
Notably, it lacks function calls, group-by, and window operations, i.e. all processing is per file.

Full manual is available in @docs/manual.md but only load if anything is unclear or not working
(to save tokens and time).

Examples:
Pass the whole query as a single quoted argument:

```
fm 'select [<field>[, <field>]...] from <glob>... [where <expr>] [sort by <field> [desc]] [limit <n>]'
fm 'update <glob>... set <assignment>[, <assignment>]...          [where <expr>]'
fm 'alter  <glob>... drop <field>[, <field>]...                   [where <expr>]'
```

### Assignments (`update set`)

| Form               | Effect                                               |
|:-------------------|:-----------------------------------------------------|
| `field:type`       | Cast field to type (creates as null if absent)       |
| `field=value`      | Set field to value; `null` clears it                 |
| `field+=value`     | Number: add; string: append; list: append if absent  |
| `field-=value`     | Number: subtract; list: remove if present            |

### Where expressions

```
[-]<comparison> [(or | and) [-]<comparison>]...
```

| Comparison         | Matches when…                                        |
|:-------------------|:-----------------------------------------------------|
| `field`            | Field exists (and matches type if typed)             |
| `field=value`      | Field equals value; for lists: value is in the list  |
| `field=null`       | Field is null                                        |
| `field+=value`     | List contains value                                  |
| `field-=value`     | List does not contain value                          |

Prefix `-` negates, e.g. `-draft` = "draft field is absent".  
`and` binds tighter than `or`.

### Types

`any` · `string` · `bool` · `int` · `number` · `date` · `link` · `list` · `list:<type>`

Dates are stored as `YYYY-MM-DD`; also parsed from `MM/DD/YYYY`, `DD.MM.YYYY`, RFC 3339.

---

## Examples

```sh
# Query
fm 'select title, tags from notes/*.md where published=true sort by date desc limit 10'

# Batch cast
fm 'update notes/*.md set date:date, rating:int'

# Conditional set
fm 'update notes/*.md set status=published where status=draft'

# Append to list
fm 'update notes/*.md set tags+=cooking where category=recipe'

# Remove field when condition met
fm 'alter notes/*.md drop draft where published=true'

# Rename a field (two-step: copy then drop)
fm 'update notes/*.md set category=cooking where from=cooking'
fm 'alter  notes/*.md drop from where from=cooking'
```

---

## Notes

- Field order in frontmatter is always sorted alphabetically after a write.
- `null` is a keyword; to assign the literal string `"null"` you need `sed`.
- Globs with spaces in paths should be quoted.
