# Built-in Functions

All functions return `NULL` on invalid arguments unless noted otherwise.

---

## String

| Function | Signature | Description |
|:---------|:----------|:------------|
| `lower` | `lower(s)` | Lowercase string |
| `upper` | `upper(s)` | Uppercase string |
| `length` | `length(s)` | UTF-8 rune count |
| `substr` | `substr(s, pos[, len])` | Substring; 1-based, negative `pos` counts from end |
| `starts_with` | `starts_with(s, prefix)` | True if `s` starts with `prefix` |
| `ends_with` | `ends_with(s, suffix)` | True if `s` ends with `suffix` |
| `contains_substr` | `contains_substr(s, sub)` | True if `s` contains `sub` |
| `trim` | `trim(s[, chars])` | Strip whitespace (or `chars`) from both ends |
| `ltrim` | `ltrim(s[, chars])` | Strip from left end only |
| `rtrim` | `rtrim(s[, chars])` | Strip from right end only |
| `replace` | `replace(s, from, to)` | Replace all occurrences of `from` with `to` |
| `split` | `split(s, sep)` | Split string into a list |
| `concat` | `concat(s, ...)` | Concatenate two or more strings |
| `regexp_contains` | `regexp_contains(s, pattern)` | True if `s` matches regex `pattern` |
| `regexp_extract` | `regexp_extract(s, pattern)` | First match (or capture group 1) of regex; NULL if no match |
| `to_string` | `to_string(v)` | Cast any value to its string representation |

---

## Numeric

| Function | Signature | Description |
|:---------|:----------|:------------|
| `abs` | `abs(n)` | Absolute value; preserves int/float type |
| `ceil` | `ceil(n)` | Ceiling; returns int |
| `floor` | `floor(n)` | Floor; returns int |
| `round` | `round(n[, digits])` | Round to `digits` decimal places (default 0) |
| `mod` | `mod(x, y)` | Integer remainder `x % y`; NULL if `y = 0` |
| `sqrt` | `sqrt(n)` | Square root; returns float |
| `pow` | `pow(base, exp)` | Exponentiation; returns float |
| `greatest` | `greatest(a, b, ...)` | Maximum of non-NULL arguments |
| `least` | `least(a, b, ...)` | Minimum of non-NULL arguments |
| `coalesce` | `coalesce(a, b, ...)` | First non-NULL argument |

---

## List

| Function | Signature | Description |
|:---------|:----------|:------------|
| `array_length` | `array_length(list)` | Length of list; returns 1 for scalar |
| `array_contains` | `array_contains(list, elem)` | True if `list` contains `elem` |
| `array_concat` | `array_concat(list, ...)` | Concatenate lists (scalars treated as single-element lists); NULL if any arg is NULL |
| `distinct` | `distinct(list)` | Remove duplicate elements; scalar → single-element list |
| `array_to_string` | `array_to_string(list, sep)` | Join list elements with separator |

---

## Date

| Function | Signature | Description |
|:---------|:----------|:------------|
| `today` | `today()` | Current date (time truncated to midnight) |
| `year` | `year(d)` | Year component as int |
| `month` | `month(d)` | Month component (1–12) as int |
| `day` | `day(d)` | Day-of-month component as int |
| `date_diff` | `date_diff(a, b)` | Days between two dates (`a − b`) |
