# Syntax

## Commands

```
fm select <field>[, <field>]...  from <glob>...  [where <expression>]  [sort by <field>[, <field>]... [desc]]  [limit <n>]
fm update <glob>...  set <assignment>[, <assignment>]...              [where <expression>]
fm alter  <glob>...  drop <field>[, <field>]...                       [where <expression>]
```

### `select`

Prints a table with one row per matching file and one column per field.

```sh
fm select title, date from '*.md' where published=true
fm select title, date from '*.md' sort by date desc
```

When no fields are given, prints only matching filenames.

`sort by` accepts one or more fields and an optional `desc` suffix for descending order. Sorting is lexicographic; files missing the sort field sort last. Dates stored as `YYYY-MM-DD` sort correctly as strings.

`limit <n>` truncates the output to at most `n` rows, applied after sorting.

### `update`

Applies a list of assignments to each matching file. See the [Assignments](#assignments-update-set) table for all forms. A bare `<name>:<type>` (no operator) casts the field to that type; skipped if the field is absent or the type is `any`.

```sh
fm update '*.md' set date:date, rating:int
fm update '*.md' set tags+=cooking where category=recipe
```

### `alter`

Removes fields from matching files. If a field is typed, only removes it when the stored value matches that type.

```sh
fm alter '*.md' drop draft where published=true
```

---

## Fields

```
<name>           -- alias for <name>:any
<name>:<type>
```

## Types

| Type          | Notes                                        |
|:--------------|:---------------------------------------------|
| `any`         | Matches any type (default when omitted)      |
| `string`      |                                              |
| `int`         | Integer numbers                              |
| `number`      | Integer or floating-point                    |
| `date`        | Stored as `YYYY-MM-DD`; also parses `MM/DD/YYYY`, `DD.MM.YYYY`, RFC 3339 |
| `link`        | Wikilink (`[[…]]`) or Markdown link (`[…](…)`) |
| `list`        | Alias for `list:any`                         |
| `list:<type>` | Typed list; recursion allowed                |

---

## Assignments (`update set`)

| Syntax             | Effect                                                                 |
|:-------------------|:-----------------------------------------------------------------------|
| `<field>`          | Cast field to `<field>`'s type                                         |
| `<field>=<value>`  | Set field to value (value may be blank)                                |
| `<field>+=<value>` | int/number: add; string: append; list: append if not present           |
| `<field>-=<value>` | int/number: subtract; list: remove if present                          |

---

## `where` expression

```
[-]<comparison> [ (or | and) [-]<comparison> ]...
```

`or` and `and` are the boolean operators. `and` binds tighter than `or`.

Outer parentheses around the whole expression are stripped, but nested grouping is not supported.

### Comparisons

| Syntax             | Matches when…                                                        |
|:-------------------|:---------------------------------------------------------------------|
| `<field>`          | Field exists (and matches type if typed)                             |
| `<field>=<value>`  | Field equals value; for lists, value is contained in the list        |
| `<field>+=<value>` | List field contains value                                            |
| `<field>-=<value>` | List field does not contain value                                    |

Prefix `-` negates any comparison, e.g. `-published` matches files where `published` is absent.

```sh
fm select title from '*.md' where -draft
```
