# Markdown frontmatter 
Command Line Interface `fm`

```
Markdown frontmatter batch editor

Usage:
	fm <glob> --filter <expression> <command>

Commands
	list <field|comparison> -- Outputs table with columns filename and field values
	set <comparison>
	rm <field> -- If field type is given, only rm if name and type match
	cast <field> <type>
	check field 

Expression
	(<comparison>||<comparison>&&<comparison>) -- Boolean expression

Comparison
	<field>=<value> -- Value may be blank

Fields
	<name>:<type>
	<name> -- Alias for <name>:any

Types:
	any -- Default if type is omitted
	string
	int
	number
	date
	link
	list -- Alias for list:any
	list:<type> -- Recursion allowed
```
