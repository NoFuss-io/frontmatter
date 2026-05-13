# Windows compatibility

Glob expansion (`expandGlobs`) and file I/O use `filepath.Glob`, `os.Stat`, and `os.ReadFile` — all cross-platform; path separators are handled correctly on Windows.

`ReadFile` assumes `\n` line endings when detecting and splitting frontmatter delimiters (`---\n`). Files authored on Windows with `\r\n` endings will not have their frontmatter parsed.
