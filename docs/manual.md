# `fm` Manual
`fm` implements a subset of SQL with syntax tailored for Markdown front matter in YAML, such as that of [Obsidian documents' properties](https://obsidian.md/help/properties). Think of files as rows and fields as columns.


## Usage

```sh
fm [query] [flags]
```

Query is input either as command line arguments or over stdin.

Flags:
- `-h`, `--help`: Show help.
- `-V`, `--version`: Print version and exit.
- `-d`, `--dry-run`: Simulates the operation without editing any files.
- `-s`, `--silent`: Suppress all output.
- `-v`, `--verbose`: Runs a `select` query on affected files and fields after `update` or `alter`.
- `-H`, `--include-hidden`: Include hidden files, see [glob expansion](#Globs).
- `--max-columns N`: Cap number of columns in `select *` output (default 20).

Query is read from `stdin` if omitted.
Query results and logs are written to `stdout`.
Errors are written to `stderr` and return exit code 1.


### Query using command line arguments

Pass the whole query as a single quoted argument:

```sh
fm 'select url from pages/* where tags>="wiki"'
```

Quoting the entire query stops the shell from interpreting operators
(`>`, `*`, `"`), glob characters, and whitespace, so `fm` receives the
query verbatim.


### Query using stdin

Can be done by pipe

```sh
cat script.sql | fm
```

input redirection

```sh
fm < script.sql
```

or manual input

```sh
fm
```
```sql
alter pages/* drop url where deprecated;

-- Confirm successful
select url from pages/*
```
then send EOF with <kbd>Ctrl</kbd> <kbd>D</kbd>.


### Output

`fm` prints normal output to `stdout`.

Execution errors, e.g. failed casting, are written to `stderr` and exits with code 2.

Query parsing errors (e.g. recognized type, malformed expression, static type error)
halt the program before touching any file and exists with code 1.


#### File mutations

Files are modified in-place without any transactional logic or backup.
Version control (e.g. `git`) and `--dry-run` are recommended safe guards.

Fields are sorted on file update.

Errors skips the file entirely and leaves it unchanged, but does not stop processing of other files.


## Syntax

Syntax generally follows the BigQuery SQL dialect with some minor deviations (and much functionality left out).

Semi-colons (`;`) separate multiple statements and double dash (`--`) creates comments (ignored).

Keywords, operators, and functions are case insensitive. Field names are case sensitive.

Additional spacing and newlines may be added anywhere.

Only frontmatter field names are queryable. File path, filename, and similar are not accessible.


### Select query

```
select <expression>[, <expression>]...
from <glob>...
[where <boolean expression>]
[sort by <expression> [asc|desc][, <field> [asc|desc]]...]
[limit <n>]
```

Execution errors produce different behavior depending on clause:
- `select`: Return ERROR.
- `where`: Evaluate as falsey.
- `sort by`: Evaluate to missing values.

Sort default direction is `asc`.
Direction may be set per field.
Missing values sort last.

`limit` is applied after `sort`.


### Update query

```
update <glob>...
set <assignment>[, <assignment>]...
[where <boolean expression>]
```

Execution errors produce different behavior depending on clause:
- `set`: Stops processing of current file (unchanged on disk).
- `where`: Evaluate as falsey.


### Alter query

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

Execution errors produce different behavior depending on clause:
- `drop`, `rename`: Stops processing of current file (unchanged on disk).
- `where`: Evaluate as falsey.


### Fields

```
identifier[:type]
```


#### Identifiers

Unquoted identifiers:

- Must begin with a letter or an underscore (\_) character.
- Subsequent characters can be letters, numbers, or underscores (\_).

Quoted identifiers:

- Must be enclosed by backtick (\`) characters.
- Can contain any characters, including spaces and symbols.
- Can't be empty.
- Have the same escape sequences as string literals.
- If an identifier is the same as a reserved keyword, the identifier must be quoted. For example, the identifier `FROM` must be quoted.


#### Types

| `fm` type     | Obsidian type                                                      | Example                     | Comment                                                                                                |
| ------------- | ------------------------------------------------------------------ | --------------------------- | ------------------------------------------------------------------------------------------------------ |
| `string`      | [Text](https://obsidian.md/help/properties#Text)                   | Foo                         |                                                                                                        |
| `link`        | Text                                                               | \[\[ref]], \[\[ref\|title]] | Wiki style links.                                                                                      |
| `mdlink`      | Text                                                               | \[title](ref)               | Markdown style links.                                                                                  |
| `list`        | [List](https://obsidian.md/help/properties#List)                   | - "bar"<br>- "baz"          | Elements are coerced to strings on cast. Lists of lists are not supported.                             |
| `numeric`     | [Number](https://obsidian.md/help/properties#Number)               | 1.23                        |                                                                                                        |
| `int`         | Number                                                             | 4                           |                                                                                                        |
| `bool`        | [Checkbox](https://obsidian.md/help/properties#Checkbox)           | true                        |                                                                                                        |
| `date`        | [Date](https://obsidian.md/help/properties#Date)                   | 2026-05-14                  |                                                                                                        |
| `datetime`    | [Date & time](https://obsidian.md/help/properties#Date%20&%20time) | 2026-05-14T21:02:30         | [ISO 8601 local time](https://en.wikipedia.org/wiki/ISO_8601#Local_time_(unqualified)) (no UTC offset) |
| `any`         | n/a                                                                |                             | Default type if omitted. Allows any value to handle heterogeneous data across files or in a list.      |
All types are nullable.


#### Evaluation & type casting

Field value is returned if field exists and type matches.

Casted field value is returned if field exists and value is castable. Strict types are always castable to looser. Loose types can be casted to strict if value is of the correct format, otherwise errors (clause-specific behavior is documented in each section).

Strict to loose:
- `bool` < `int` < `number` < `string`
- `link` = `mdlink` < `string`
- `datetime` < `date` < `string`

Cast between `link` and `mdlink` is always possible (reversible format conversion).

Cast from `datetime` to `date` is lossy (time part truncated).

Scalar may be cast to a single-element list and vice versa. When casting to `list`, each element is coerced to `string`.


##### Examples

| Field value | Cast     | Result   |
| ----------- | -------- | -------- |
| 1           | bool     | true     |
| 2           | bool     | ERROR    |
| "3"         | int      | 3        |
| 4           | list     | \["4"]   |
| \["5"]        | int      | 5        |
| \["6","7"]      | int      | ERROR    |
| Null     | int      | Null     |


### Expressions

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


#### Atoms

Atoms are the leaf nodes of an expression — either a field reference or a literal.

**Field** (`identifier[:type]`): reads the value from the current file's frontmatter and optionally casts it (see [Fields](#fields)). Evaluates to **null** if the field does not exist or casting fails.

**Literal**: a constant value embedded directly in the query (see [Literals](#literals)). Literals always exist; they never produce null.

```
title                -- field, any type
price:number         -- field cast to number
`created-at`:date    -- quoted-identifier field cast to date
42                   -- integer literal
"hello world"        -- string literal
true                 -- boolean literal
```

Null propagates through arithmetic — any operation with a null operand produces null. In a boolean context (comparisons, `where`) null is falsey.


#### Literals

##### String literals

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


##### Integer literals

Decimal or hexadecimal:

```
42
-7
0xFF
0x1A2B
```


##### Numeric literals

Decimal with optional fractional part and exponent:

```
3.14
-2.5
1.0e10
6.022E-23
```


##### Boolean literals

`true` and `false` (case insensitive).


##### Null literal

`null` (case insensitive). Represents absence of a value.


##### Date and datetime literals

`fm` does not use BigQuery's typed literal syntax (`DATE '...'`, `DATETIME '...'`). Instead, date and datetime values are encoded as plain strings matching ISO 8601 format, and type is controlled via the field type annotation:

```
created:date = 2026-05-17
modified:datetime = 2026-05-17T21:02:30
```


#### Boolean expressions

```
<field> [<op> <expression>]
```

Unary comparison evaluates to the boolean value of the field (after optional casting). Falsey values are: `null`, `false`, `0`, `0.0`, and a missing field or failed cast (which also produce null). All other values are truthy.

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
| `url = "https://fm.nofuss.io"` | URL matches "https://fm.nofuss.io".                                                                                                                            |
| `tags:list >= "recipe"`                | Tags contain value "recipe"                                                                                                                                    |
| `url = source`                    | URL matches the value of the source field.                                                                                                                     |
| `price = cost`                    | Price = cost. Always ok regardless of types, since all types can be relaxed to `string` and tested for equality.                                               |
| `price:number > cost:number`        | Price > cost. `false` if casting fails.                                                                                                                        |
| `price:int > cost:number`           | Dito. Price is first cast to `int`, then relaxed to `number` for the comparison. If price is 1.23 then whole expression is `falsey` since `int` casting fails. |
| `price:string > cost:number`        | Static type error caught at parse; halts program. `>` does not accept strings.                                                                                 |


### Assignment

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


### Globs

Can be a list of files

```sh
fm 'select url from page/index.md page/account.md'
```

or expressions to expand (hidden files are ignored)

```sh
fm 'select url from page/*.md'
```

`fm` also implements its own glob expansion capability,
identical to glob expansion in POSIX shells (e.g. bash or zsh),
to provide the same functionality when called programatically or run on Windows.

```sh
fm 'select url from page/*.md'  # Ok, even though shell won't expand it
```
