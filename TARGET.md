# Markdown frontmatter 
Command Line Interface `fm`

```
Markdown frontmatter batch editor

Usage:
    fm <command>

Commands
    select <field>...            from <glob>... [where <expression>]
    update <glob>... set <field|assignment>...  [where <expression>]
    alter  <glob>... drop <field>...            [where <expression>]

    select -- Output table with filename column and one column per field
    update -- Cast or set fields:
                <field>          casts field to the given type
                <assignment>     sets field to the given value
              If field is typed, only affect files where the type matches
    alter  -- Remove fields; if field is typed, only drop if type matches

Expression
    [!]<comparison> [ (|| | &&) [!]<comparison> ]...

Comparison / Assignment
    <field>          -- In where: match by type; in update: cast to type
    <field>=<value>  -- In where: match by value; in update: set value; value may be blank
    <field>+=<value>  -- In update: if int or number, addition; if string, append; if list, append if missing (treat as a set)
    <field>-=<value>  -- In update: if int or number, subtraction; if list, remove if present (treat as a set)
    

Fields
    <name>:<type>
    <name>           -- Alias for <name>:any

Types:
    any         -- Match any type (default if type is omitted)
    string
    int
    number
    date
    link
    list        -- Alias for list:any
    list:<type> -- Recursion allowed
```
