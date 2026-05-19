`fm` implements a subset of SQL with syntax tailored for Markdown front matter in YAML, such as that of [Obsidian documents' properties](https://obsidian.md/help/properties). Think of files as rows and fields as columns.


# Usage
```sh
fm [query] [flags]
```

Flags:
- `--dry-run`: Simulates the operation without editing any files.
- `--silent`: Suppress all output.
- `-v`: Runs a `select` query on affected files and fields after `update` or `alter`.

Query is read from `stdin` if omitted.
Query results and logs are written to `stdout`.
Errors are written to `stderr` and return exit code 1.

Parsing happens first. Any parse error (e.g. unrecognized type, malformed expression, static type error) halts the program before any file is touched.

## Scripts

Multiple statements may be combined into a script, separated by `;`.
`--` starts a line comment that runs to the end of the line. Both `;`
and `--` are ignored inside quoted strings and backtick identifiers.

Statements run sequentially. If any statement fails the script halts
and the remaining statements are not executed.

`--dry-run` is rejected for multi-statement scripts because there is no
transactional layer: each statement re-reads files from disk, so later
statements would not observe the mutations that earlier statements would
have made.

```sh
fm < script.sql
```

# Mutations

Files are modified in-place without any transactional logic or backup.
Version control (e.g. `git`) and `--dry-run` are recommended safe guards.

Fields are sorted on file update.

Errors skips the file entirely and leaves it unchanged, but does not stop processing of other files.

# Query
Syntax generally follows the BigQuery SQL dialect with some minor deviations (and much functionality left out).

Keywords, operators, and functions are case insensitive. Field names are case sensitive.

Additional spacing and newlines may be added anywhere.

Only frontmatter field names are queryable. File path, filename, and similar are not accessible.

## Select query
```
select <expression>[, <expression>]...
from <glob>...
[where <boolean expression>]
[sort by <expression> [asc|desc][, <field> [asc|desc]]...]
[limit <n>]
```

Erroneous expressions produce different behavior depending on clause:
- `select`: Return ERROR.
- `where`: Evaluate as falsey.
- `sort by`: Evaluate to missing values.

Sort default direction is `asc`.
Direction may be set per field.
Missing values sort last.

`limit` is applied after `sort`.

## Update query
```
update <glob>...
set <assignment>[, <assignment>]...
[where <boolean expression>]
```

Erroneous expressions produce different behavior depending on clause:
- `set`: Stops processing of current file (unchanged on disk).
- `where`: Evaluate as falsey.

## Alter query
```
alter <glob>...
drop <field>[, <field>]...
[where <boolean expression>]
```

```
alter <glob>...
rename <field> to <field>[, <field> to <field>]...
[where <boolean expression>]
```

Erroneous expressions produce different behavior depending on clause:
- `drop`, `rename`: Stops processing of current file (unchanged on disk).
- `where`: Evaluate as falsey.

## Fields
```
identifier[:type]
```

### Identifiers
Unquoted identifiers:

- Must begin with a letter or an underscore (\_) character.
- Subsequent characters can be letters, numbers, or underscores (\_).

Quoted identifiers:

- Must be enclosed by backtick (\`) characters.
- Can contain any characters, including spaces and symbols.
- Can't be empty.
- Have the same escape sequences as string literals.
- If an identifier is the same as a reserved keyword, the identifier must be quoted. For example, the identifier `FROM` must be quoted.

### Types

| `fm` type     | Obsidian type                                                      | Example                     | Comment                                                                                                |
| ------------- | ------------------------------------------------------------------ | --------------------------- | ------------------------------------------------------------------------------------------------------ |
| `string`      | [Text](https://obsidian.md/help/properties#Text)                   | Foo                         |                                                                                                        |
| `link`        | Text                                                               | \[\[ref]], \[\[ref\|title]] | Wiki style links.                                                                                      |
| `mdlink`      | Text                                                               | \[title](ref)               | Markdown style links.                                                                                  |
| `list[:type]` | [List](https://obsidian.md/help/properties#List)                   | - "bar"<br>- "baz"          | Default list type is `any`.<br>Lists of lists are not supported.                                       |
| `numeric`     | [Number](https://obsidian.md/help/properties#Number)               | 1.23                        |                                                                                                        |
| `int`         | Number                                                             | 4                           |                                                                                                        |
| `bool`        | [Checkbox](https://obsidian.md/help/properties#Checkbox)           | true                        |                                                                                                        |
| `date`        | [Date](https://obsidian.md/help/properties#Date)                   | 2026-05-14                  |                                                                                                        |
| `datetime`    | [Date & time](https://obsidian.md/help/properties#Date%20&%20time) | 2026-05-14T21:02:30         | [ISO 8601 local time](https://en.wikipedia.org/wiki/ISO_8601#Local_time_(unqualified)) (no UTC offset) |
| `any`         | n/a                                                                |                             | Default type if omitted. Allows any value to handle heterogeneous data across files or in a list.      |
All types are nullable.

### Evaluation & type casting
Field value is returned if field exists and type matches.

Casted field value is returned if field exists and value is castable. Strict types are always castable to looser. Loose types can be casted to strict if value is of the correct format, otherwise errors (clause-specific behavior is documented in each section).

Strict to loose:
- `bool` < `int` < `number` < `string`
- `link` = `mdlink` < `string`
- `datetime` < `date` < `string`

Cast between `link` and `mdlink` is always possible (reversible format conversion).

Cast from `datetime` to `date` is lossy (time part truncated). 

Scalar may be cast to a single-element list and vice versa.

`list:X` can be relaxed to `list:Y` if X is looser than Y. The reverse direction errors if any element fails to cast.

#### Examples

| Field value | Cast     | Result   |
| ----------- | -------- | -------- |
| 1           | bool     | true     |
| 2           | bool     | ERROR    |
| "3"         | int      | 3        |
| 4           | list     | \[4]     |
| \[5]        | int      | 5        |
| \[6,7]      | int      | ERROR    |
| \[7,8,"9"]  | list:int | \[7,8,9] |
| Missing     | int      | Void     |

## Expressions

An expression is recursively composed of [fields](#fields), [literals](#literals), operators, and grouping parentheses.

```ebnf
expression = or_expr
or_expr    = and_expr { "or" and_expr }
and_expr   = not_expr { "and" not_expr }
not_expr   = [ "not" ] comparison
comparison = arith { comp_op arith }
arith      = term { ( "+" | "-" ) term }
term       = factor { ( "*" | "/" ) factor }
factor     = [ "-" ] primary
primary    = "(" expression ")" | field | literal
```

Parentheses override precedence: `a or (b and c)` evaluates `b and c` first.

Operator precedence (highest to lowest), following BigQuery ([docs](https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/operators#operator_precedence)):

| Precedence | Operator |
|-----------:|:---------|
| 1 | Unary `-` |
| 2 | `*`, `/` |
| 3 | `+`, `-` |
| 4 | Comparison operators |
| 5 | `not` |
| 6 | `and` |
| 7 | `or` |

### Atoms

Atoms are the leaf nodes of an expression — either a field reference or a literal.

**Field** (`identifier[:type]`): reads the value from the current file's frontmatter and optionally casts it (see [Fields](#fields)). Evaluates to **void** if the field does not exist or casting fails.

**Literal**: a constant value embedded directly in the query (see [Literals](#literals)). Literals always exist; they never produce void.

```
title                -- field, any type
price:number         -- field cast to number
`created-at`:date    -- quoted-identifier field cast to date
42                   -- integer literal
"hello world"        -- string literal
true                 -- boolean literal
```

Void propagates through arithmetic — any operation with a void operand produces void. In a boolean context (comparisons, `where`) void is falsey.

### Literals

#### String literals

Strings can be enclosed in single quotes (`'`), double quotes (`"`), or triple-quoted variants (`'''` or `"""`). Triple-quoted strings can span multiple lines and contain unescaped single or double quotes respectively.

```
'hello world'
"hello world"
'''multi
line'''
"""also
multi-line"""
```

Prefix `r` or `R` creates a raw string where backslash sequences are not interpreted:

```
r'C:\Users\name'
R"no\escape"
```

Escape sequences in non-raw strings:

| Sequence | Description |
|:---------|:------------|
| `\a` | Bell |
| `\b` | Backspace |
| `\f` | Form feed |
| `\n` | Newline |
| `\r` | Carriage return |
| `\t` | Tab |
| `\v` | Vertical tab |
| `\\` | Backslash |
| `\'` | Single quote |
| `\"` | Double quote |
| `` \` `` | Backtick |
| `\?` | Question mark |
| `\NNN` | Octal (3 digits, range `000`–`377`) |
| `\xNN` | Hex (2 digits) |
| `\uNNNN` | Unicode codepoint (4 hex digits) |
| `\UNNNNNNNN` | Unicode codepoint (8 hex digits) |

#### Integer literals

Decimal or hexadecimal:

```
42
-7
0xFF
0x1A2B
```

#### Numeric literals

Decimal with optional fractional part and exponent:

```
3.14
-2.5
1.0e10
6.022E-23
```

#### Boolean literals

`true` and `false` (case insensitive).

#### Null literal

`null` (case insensitive). Represents absence of a value.

#### Date and datetime literals

`fm` does not use BigQuery's typed literal syntax (`DATE '...'`, `DATETIME '...'`). Instead, date and datetime values are encoded as plain strings matching ISO 8601 format, and type is controlled via the field type annotation:

```
created:date = 2026-05-17
modified:datetime = 2026-05-17T21:02:30
```

### Boolean expressions
```
<field> [<op> <expression>]
```

Unary comparison evaluates to the boolean value of the field (after optional casting). Falsey values are: `null`, `false`, `0`, `0.0`, and void (missing field or failed cast). All other values are truthy.

Binary comparison is truthy if both operands exist (literals always exist), with comparable types, and values meeting the criteria. Types are comparable if they can be relaxed to a matching type — otherwise it is a static type error caught at parse and halts the program. Runtime cast failure on values is per-clause (see [Error handling and program flow](#error-handling-and-program-flow)).

| Operator             | LHS type          | RHS type          | Meaning        |
| -------------------- | ----------------- | ----------------- | -------------- |
| `=`, `!=`            | Scalar            | Scalar            | Equality       |
| `>`, `>=`, `<`, `<=` | Bool, int, number | Bool, int, number | Size           |
| `=`, `!=`            | List              | List              | Set equality   |
| `>=`                 | List              | Scalar            | Set membership |
| `<=`                 | Scalar            | List              | Set membership |

| Example                             | Tests if                                                                                                                                                       |
| ----------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `url:string = https://fm.nofuss.io` | URL matches "https://fm.nofuss.io".                                                                                                                            |
| `tags:list = recipe`                | Tags contain value "recipe"                                                                                                                                    |
| `url: = source:`                    | URL matches the value of the source field.                                                                                                                     |
| `price: = cost:`                    | Price = cost. Always ok regardless of types, since all types can be relaxed to `string` and tested for equality.                                               |
| `price:number > cost:number`        | Price > cost. `false` if casting fails.                                                                                                                        |
| `price:int > cost:number`           | Dito. Price is first cast to `int`, then relaxed to `number` for the comparison. If price is 1.23 then whole expression is `falsey` since `int` casting fails. |
| `price:string > cost:number`        | Static type error caught at parse; halts program. `>` does not accept strings.                                                                                 |

QUESTION: Literal syntax — `https://fm.nofuss.io` and `recipe` are unquoted bare values. What if the literal contains spaces, `=`, `,`, parentheses, or shell metacharacters? Are double quotes / single quotes supported as literal delimiters? Same question for `from` glob arguments.
QUESTION: `url: = source:` — both sides default to `any`. The prose says "always relaxed to string for equality". Is that the rule for `any = any`, or only when both fields exist with different concrete types? Make explicit.
QUESTION: `tags:list = recipe` — example uses `=`, but the operator table above says list-vs-scalar set membership uses `>=`/`<=`, not `=`. Either the example is stale or the table is. Reconcile.

## Assignment
```
<field> [<assignment operator> <expression>]
```

Unary assignment casts field to type and creates it if missing with `null` as value.

> [!example] Ensure all documents have field `foo` (`int`):
>```
>update * set foo:int
>```
> Errors if `foo` exist but cannot be casted to `int` (see [[#Fields]]).

Binary assignment assigns new value to field, optionally casts to specific type and creates if missing. `null` is assigned as `<field> = null`.

| Operator   | LHS type          | RHS type     | Meaning                                          |
| ---------- | ----------------- | ------------ | ------------------------------------------------ |
| `=`        | Scalar            | Scalar       | Set value                                        |
| `+=`, `-=` | List              | Scalar, list | Set addition and subtraction.                    |
| `+=`, `-=` | Scalar (implicit) | Scalar       | Cast LHS to list then set addition and subtract. |

> [!example] Create or update field `foo`:
> ```
> update * set foo = bar + 3
> ```
> Errors if `bar` cannot be casted to numeric (required by addition, see [[#Expressions]]).
> 
> Type of `foo` is inferred automatically since type is not explicitly state. This may produce different types in different documents (`int` or `numeric`).
> To ensure field type:
> ```
> update * set foo:int = bar + 3
> ```
> Errors if `bar + 3` cannot be casted to `int`.

Statically invalid assignments raise parse errors and halt program before any file is touched (see [[#Usage]]).

> [!example] Invalid query
> ```
> update * set foo:int = "hello"
> ```
> The string `"hello"` can never be assigned to an integer field, so program will not run.


## Globs
Can be a list of files or expressions to expand, identical to glob expansion in POSIX shells (e.g. bash or zsh).

`fm` implements its own glob expansion (rather than just accepting list of files) to provide the same functionality on Windows and if called programmatically.
QUESTION: POSIX glob, bash, and zsh differ. Does `fm` support `**` (recursive)? Brace expansion `{a,b}`? Character classes `[abc]`? Negation? Dotfiles? Document the exact supported subset.
QUESTION: Are non-`.md` matches filtered out automatically, or treated as errors, or attempted as YAML frontmatter regardless of extension?
QUESTION: Are symlinks followed? Hidden directories (e.g. `.obsidian/`) recursed?
QUESTION: When called from a POSIX shell, the shell will glob-expand first; `fm`'s own expansion only runs on unmatched/quoted arguments. Spell out the interaction.
