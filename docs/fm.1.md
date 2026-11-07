% FM(1) | fm

# NAME

fm - Markdown frontmatter batch editor

# SYNOPSIS

**fm** [_QUERY_] [_FLAGS_]

**fm** **completion** {bash|zsh|fish}

# DESCRIPTION

**fm** queries and mutates YAML frontmatter in Markdown files using an
SQL-style DSL. It is aimed at knowledge-management workflows such as
Obsidian vaults, where structured metadata sits in the `---` YAML block at
the top of `.md` files.

Three top-level statements are supported:

```
select <exprs> from <globs> [where <expr>] [sort by <terms>] [limit <n>]
update <globs> set <assigns> [where <expr>]
alter  <globs> drop   <fields>  [where <expr>]
alter  <globs> rename <pairs>   [where <expr>]
```

Multiple statements may be combined into an SQL-style script separated by
`;`, with `--` as a line comment. Scripts can be read from stdin or passed
as a single positional argument.

# OPTIONS

**-h**, **--help**
:   Show usage and exit.

**-d**, **--dry-run**
:   Simulate the operation without editing any files. Rejected for
    multi-statement scripts.

**-H**, **--hidden**
:   Include hidden files (ignored by default).

**\--max-columns** _N_
:   Column cap for **select \*** output. Default 20.

**-L**, **\--lang** _LANG_
:   Parse every file as language _LANG_, given as a name (`python`, `shell`,
    `c`, `sql`, ...) or an extension (`py`, `js`, ...). By default the
    syntax is chosen per file by extension or shebang. With this flag a glob
    such as **\*** matches files of every type.

**\--line-comment** _STR_
:   Frontmatter is in line comments starting with _STR_, for every file.

**\--block-open** _STR_, **\--block-close** _STR_
:   Frontmatter is in a block comment delimited by these markers, for every
    file. The closer defaults to the opener. May be combined with
    **\--line-comment** but not with **\--lang**.

# SUBCOMMANDS

**fm completion** _SHELL_
:   Print a shell-completion script for _SHELL_ (one of `bash`, `zsh`,
    `fish`) to stdout.

# EXAMPLES

Inspect frontmatter across a vault:

```
fm 'select title, tags from notes/**/*.md'
```

Cast a field to int across many files:

```
fm 'update recipes/* set `Prep time`:int where `Prep time`'
```

Drop deprecated fields:

```
fm 'alter notes/* drop legacy_field'
```

# FILES

**fm** reads and writes the file(s) matched by the **from** clause in
place. There is no transactional layer; consider running on files under
version control or backed up.

# SEE ALSO

Full manual: `docs/manual.md` in the source distribution, or
<https://github.com/NoFuss-io/frontmatter/blob/main/docs/manual.md>.

# AUTHORS

Christofer Bäcklin and contributors.
<https://github.com/NoFuss-io/frontmatter>
