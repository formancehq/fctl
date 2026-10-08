# Interactive command input

On a terminal, commands ask for missing required values. `fctl ledger create`
opens a form; `fctl ledger stats` lists available ledgers; `fctl cloud stack create`
selects an organization, asks for a name, then lists regions and their exact
catalog versions. Searchable menus show resource names and IDs. Forms use one
field per step, retain values while moving back, and show keyboard help.

Explicit arguments, flags, saved connection defaults and environment settings
remain authoritative. A complete invocation executes directly. An explicit
`--data` body bypasses payload forms, including `@file` and stdin bodies. Partial
forms ask only for omitted values; blank optional fields remain omitted.

Press `Ctrl+C` to cancel. No service mutation occurs until all required
inputs are accepted and the plugin validates the request. Destructive operations
ask for confirmation, defaulting to No. `--confirm` permits the operation directly;
`--confirm=false` rejects it. App commands still require `--experimental`.

## Automation and streams

Use `--no-input`, `FCTL_NO_INPUT=1`, or `CI=true` to disable interaction.
A non-terminal stdin or stderr also disables forms. Missing inputs then produce
a nonzero exit code and the same explicit-argument contract used by SDK callers.

Forms and selections write to stderr. Results write to stdout, so a terminal
input can be combined with `-o json > result.json`. JSON remains free of terminal
controls even with `--color always` or `--color never`. `NO_COLOR` disables form
colors unless overridden by `--color always`. Password fields mask their input.

## Plugin ownership

Each runnable leaf declares serializable `CommandSpec.Inputs` in the public SDK.
The plugin specifies field names, types, defaults, choices, request bindings and
read operations for discovery. It owns payload validation and service behavior.
The core owns terminal rendering, field validation, cancellation and pagination.
Plugins do not import the UI library or core packages.

`CommandSpec.Target` declares inherited identity, organization or stack context.
`InputSpec` binds exactly one flag, argument, JSON pointer or host resource ID.
Alternative arguments and flags bypass a field when the caller provides an
explicit alternative. Explicit body input bypasses JSON-pointer fields, regardless of the declared
body flag name. Body contents never appear in confirmation prompts.

Choice sources run through a client that rejects non-GET requests. Cursor lists
follow `next` while `hasMore` is true. Keyset lists declare `AfterField`; the core
requests pages of 100 and uses the final row's exact value as `--after` until a
shorter page. Plugins can exclude true boolean fields and match exact state fields to offer
only eligible resources. Pagination uses the original rows before filtering.
Duplicate IDs are removed. Invalid or repeated continuation values
fail with guidance to supply an explicit ID. Enumeration is bounded to 1,000
pages and 10,000 unique choices; it never silently truncates the menu.

A preferred version prefix orders suggestions; it does not select a fallback
version without user input. V4 versions appear first in the stack create menu.
The exact selected version is validated again by the Cloud plugin before create.
