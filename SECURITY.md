# Security Policy

`fm` is a local CLI tool. It reads and writes Markdown files in the directories
you point it at. It does not open network connections, execute shell commands,
or load plugins. There is no privilege boundary between the tool and the user
who runs it, so there is no meaningful remote attack surface.

If you find a crash, hang, or incorrect mutation — even on weird or
deliberately malformed input — please file a regular [bug report][issues].
There is no separate private channel; treat security findings like any other
bug.

[issues]: https://github.com/NoFuss-io/frontmatter/issues/new?template=bug_report.yml
