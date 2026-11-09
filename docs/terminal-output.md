# Terminal output helpers

`pkg` provides helpers that keep plain output and help text inside the width
of the user's terminal. The root command applies `WithTerminalHelp()`, so every
command's help fits the terminal. The other helpers are opt-in per command;
`fctl connectivity` uses all of them.

## Width

`fctl.TerminalWidth(out)` returns the width of the terminal when `out` is the
interactive standard output, and `0` otherwise. Zero means "do not fit": piped
or redirected output keeps every value intact. Tests force a width with
`fctl.SetTerminalWidth(n)`, which returns a restore function.

## Tables and text

| Helper | Use |
|---|---|
| `RenderTable(out, header, columns, rows)` | Header table fitted to the terminal |
| `RenderGroupedTable(out, header, columns, groups)` | Titled row groups sharing column widths |
| `RenderKeyValues(out, rows)` | Label/value block; values wrap under the value column |
| `RenderText(out, text)` / `RenderBullet(out, text)` | Wrapped line, or `- ` bullet with an indented continuation |
| `RenderNextPage(out, cursor)` | `Next page: --cursor <token>`, printed only when more results exist |
| `Truncate`, `Wrap`, `FirstSentence` | Text primitives used by the above |

Each `TableColumn` declares how it gives up width:

- `FitNone` (default): never cut. Use it for identifiers people copy, such as
  names, keys and versions.
- `FitCut`: truncated with `…`, down to `Min` cells (default 8).
- `FitWrap`: wrapped on word boundaries into a multi-line cell, at most
  `MaxLines` lines.

When the table is too wide, the widest shrinkable column loses one cell at a
time. When the minimum widths still overflow, each row prints as a stacked
block of labelled values instead. `Style` colours a cell after fitting, so
widths stay exact.

## Help and completion options

| Option | Effect |
|---|---|
| `WithTerminalHelp()` | In this command's help and its subcommands': wraps flag descriptions to the terminal and, on an interactive stdout without `NO_COLOR`, colours section titles, command names, examples and the `(required)` flag marker |
| `WithExamples(lines...)` | Sets the Examples section, one command per line |
| `WithFlagUsage(name, usage)` | Replaces the help text of a flag declared by a shared option |
| `WithoutFileCompletion(names...)` | Stops the shell offering file names for free-text flags; apply after the flags are declared |
