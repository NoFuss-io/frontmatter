# Syntax

## Commands

```
fm select <field>... from <glob>... [where <expression>]
fm update <glob>... set <field|assignment>... [where <expression>]
fm alter  <glob>... drop <field>...           [where <expression>]
```

### `select`

Prints a table with one row per matching file and one column per field.

```sh
fm select title, date from '*.md' where published=true
```

When no fields are given, prints only matching filenames.

### `update`

Casts or sets fields on matching files.

- `<field>` (no `=`) — cast the field to the given type; skipped if the field is absent or already that type
- `<assignment>` — set, add to, or remove from the field value

```sh
fm update '*.md' set date:date rating:int
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
