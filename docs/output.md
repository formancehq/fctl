# Output and diagnostics

The host renders command JSON through `presentation.Render`. Plugins return
raw JSON; terminal styling does not change service requests or plugin schemas.

| Setting | Default | Behavior |
| --- | --- | --- |
| `-o`, `--output` | `auto` | `auto` selects tables on a terminal and JSON otherwise; explicit values are `json` and `table` |
| `--color` | `auto` | `auto` colors terminal output; `always` forces color and `never` disables it |
| `NO_COLOR` | Unset | A nonempty value disables automatic color |
| `TERM=dumb` | Environment-dependent | Disables automatic color |
| `-d`, `--debug` | `false` | Writes HTTP diagnostics to stderr |

`--color always` overrides automatic color suppression. JSON output never adds
ANSI styling, even when color is forced. Full Cobra help remains available;
color emphasizes its standard section labels, including Additional Commands,
and every declared Cobra command group, including Plugins. Only complete
heading lines are styled; descriptions and examples retain their original text.

```sh
fctl cloud stack list
fctl cloud stack list -o json > stacks.json
fctl ledger list -o table --color never
NO_COLOR=1 fctl cloud organizations list
```

## JSON and human tables

JSON output preserves all response fields and exact JSON numbers, with
indentation. Use it for scripts and complete payload inspection.

Human lists select at most six scalar columns that fit the terminal width,
favoring IDs, names, state/status, region and version. Nested configuration and
additional fields are omitted from list cells. Details display top-level fields;
large nested objects and arrays use bounded summaries, with total item counts
for arrays. Tables do not add omitted-field counts or JSON reminders. Empty
nested containers display `—`. Empty `schema`, `mirrorSource` and
`mirrorSyncProgress` configurations also display `—`, including null values or
objects containing only empty strings, nulls and empty containers. Booleans and
numbers, including `false` and `0`, remain meaningful details. Top-level scalar
values and numeric tokens are preserved and wrapped. Empty lists have a friendly
message; pagination displays cursor and `hasMore` information.

Plain tables use ASCII borders. Colored tables add a title and status colors.
Control characters in values are escaped; width calculations account for wide
characters and generated ANSI styling. The renderer is a presentation layer,
so human lists are summaries rather than a replacement for complete JSON.

## HTTP diagnostics

```sh
fctl login --no-browser -d
fctl cloud stack list -o json -d
```

Debug traces include method, sanitized URL, headers, response status and bounded
body previews. Each preview is limited to 64 KiB. Unknown, malformed, incomplete
or oversized bodies are summarized instead of printed raw. Request bodies are
inspected through `GetBody` when available; tracing does not consume a request
stream. Response tracing follows normal reads and close, preserving the stream.
Concurrent trace output is synchronized.

Authorization, cookies and sensitive query, form and nested JSON fields are
redacted, including tokens, secrets, passwords, device codes and Auth's `clear`
secret field. Device verification URLs receive the same query filtering.
Known OAuth `error` codes remain visible, including `invalid_grant`,
`invalid_client`, `server_error` and `temporarily_unavailable`; arbitrary error
text and `error_description` are redacted.

Diagnostic redaction does not alter command results. Commands that intentionally
return credentials, such as secret creation or personal-token generation, still
write those results to stdout.
