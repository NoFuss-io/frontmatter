# Syntax
`fm` implements a subset of SQL with syntax tailored for Markdown front matter in YAML, such as that of [Obsidian documents' properties](https://obsidian.md/help/properties).

All is case sensitive.

Spacing around operators is optional.

## Commands
### Select
```
fm select <field>[, <field>]...
from <glob>...
[where <expression>]
[sort by <field>[, <field>]... [desc]]
[limit <n>]
```

Field specification determines output stream:
- Field exists and type match on all fields -> `stdout`
- Field missing or type mismatch on any field -> `stderr`

`any` type (default if omitted) is always considered a match.

### Update
```
fm update <glob>...
set <assignment>[, <assignment>]...
[where <expression>]
```

### Alter
```
fm alter <glob>...
drop <field>[, <field>]...
[where <expression>]
```

Field specification determines action:
- Field exists and type matches -> drop and logged to `stdout`
- Field missing or type mismatch -> no-op and logged to `stderr`

## Fields
```
name:[type (default:any)]
```

| `fm` type     | Obsidian type                                                      | Example             | Comment                                                                                                |
| ------------- | ------------------------------------------------------------------ | ------------------- | ------------------------------------------------------------------------------------------------------ |
| `string`      | [Text](https://obsidian.md/help/properties#Text)                   | Foo                 |                                                                                                        |
| `link`        | Text                                                               | \[\[Page]]          | Wiki style links.                                                                                      |
| `mdlink`      | Text                                                               | \[title](ref)       | Markdown style links.                                                                                  |
| `list[:type]` | [List](https://obsidian.md/help/properties#List)                   | - "bar"<br>- "baz"  | Default list type is `any`.<br>Lists of lists are not supported.                                       |
| `number`      | [Number](https://obsidian.md/help/properties#Number)               | 1.23                |                                                                                                        |
| `int`         | Number                                                             | 4                   |                                                                                                        |
| `bool`        | [Checkbox](https://obsidian.md/help/properties#Checkbox)           | true                |                                                                                                        |
| `date`        | [Date](https://obsidian.md/help/properties#Date)                   | 2026-05-14          |                                                                                                        |
| `datetime`    | [Date & time](https://obsidian.md/help/properties#Date%20&%20time) | 2026-05-14T21:02:30 | [ISO 8601 local time](https://en.wikipedia.org/wiki/ISO_8601#Local_time_(unqualified)) (no UTC offset) |
| `any`         | n/a                                                                |                     |                                                                                                        |

### Evaluation & type casting
Field value is returned if field exists and type matches.

Casted field value is returned if field exists and value is castable. Strict types are always castable to looser. Loose types can be casted to strict if value is of the correct format, otherwise errors.

Strict to loose:
- `bool` < `int` < `number` < `string`
- `link` = `mdlink` < `string`
- `datetime` < `date` < `string`

So, all types can be cast to string. `any` is not a type in itself but means strictest possible type in the given situation.

#### Example
```markdown
---
count: "4"
---
```

- `count:` evaluates to the string `"4"`.
- `count:int` evaluates to `4` since value's format allows casting to ìnt`.
- `count:date` evaluates to `null` since type casting is not supported.
- `x` evaluates to `null` since not present

## Comparison
```
<field> [<op> <field|value>]
```

Unary comparison is truthy if field exist with matching or stricter type. Otherwise falsey.

Binary comparison is truthy if LHS and RHS fields both exist with comparable types and values meet criteria. Types are comparable if they can be relaxed to a matching type (otherwise error).

| Operator             | LHS type | RHS type | Meaning        |
| -------------------- | -------- | -------- | -------------- |
| `=`, `!=`            | Any      | Any      | Equality       |
| `=`, `!=`            | List     | Not list | Set membership |
| `>`, `>=`, `<`, `<=` | Ordinal  | Ordinal  | Size           |

| Example                             | Tests if                                                                                                                                                       |
| ----------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `url:string = https://fm.nofuss.io` | URL matches "https://fm.nofuss.io".                                                                                                                            |
| `tags:list = recipe`                | Tags contain value "recipe"                                                                                                                                    |
| `url: = source:`                    | URL matches the value of the source field.                                                                                                                     |
| `price: = cost:`                    | Price = cost. Always ok regardless of types, since all types can be relaxed to `string` and tested for equality.                                               |
| `price:number > cost:number`        | Price > cost. `false` if casting fails.                                                                                                                        |
| `price:int > cost:number`           | Dito. Price is first cast to `int`, then relaxed to `number` for the comparison. If price is 1.23 then whole expression is `falsey` since `int` casting fails. |
| `price:string > cost:number`        | Syntax error that halts program. `>` operator does not  accept strings.                                                                                        |


## Assignment
```
<field> [<op> <field|value>]
```

RHS value is cast to type of LHS if given. If casting fails assignment is no-op (error).

| Operator   | Type                       | Meaning                      |
| ---------- | -------------------------- | ---------------------------- |
| `=`        | All                        | Set value                    |
| `+=`, `-=` | List LHS with non-list RHS | Set addition and subtraction |


## Expressions
Boolean expressions composed of comparisons, grouping parentheses, and logical operators.

Operator precedence:
1. `not`
2. `and`, `nand`
3. `or`, `nor`, `xor`

## Globs
Can be a list of files or expressions to expand, identical to glob expansion in POSIX shells (e.g. bash or zsh).

`fm` implements its own glob expansion (rather than just accepting list of files) to provide the same functionality on Windows and if called programmatically.

## Logging and error reporting
`fm` edits files in place.

Output of statement execution is written to `stdout`.
Amount of output is controlled using the verbosity flags `-v` and `-vv`.

Errors are logged to `stderr`.
