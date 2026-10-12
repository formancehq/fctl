# Declarative forms and selectors

Read the checkout's `docs/interaction.md` and the existing service's
`interaction.go` before choosing fields. `InputSpec` expresses UI intent; it does
not replace argument, flag or payload validation in direct SDK execution.

## Bindings

Each input binds exactly one of `Flag`, `Argument`, `BodyPointer` or `Context`.
`Argument` is a zero-based index represented by `*int`; zero must not disappear
during JSON serialization. `BodyPointer` is an RFC 6901 pointer into the request.
`Context` is reserved for host organization/stack IDs.

Supported kinds are `input`, `text`, `select` and `confirm`; the value type is
`string`, `json`, `number` or `bool`. Choose types from the actual service
contract. A numeric input does not expand the supported `FlagSpec.Type` set.
Declare every bound flag and keep it consistent with the input's value type.
`number` checks JSON numeric syntax, not an integer range or sign. Enforce
integer, nonnegative and size constraints in plugin payload validation according
to the service contract. Validate exact tokens or `json.Number` with integer
arithmetic instead of converting through `float64`; preserve the API's allowed
numeric notation and bounds.

For example, a leaf creating an object can declare:

```go
pluginsdk.CommandSpec{
    Use: "create", Short: "Create an item", Runnable: true,
    Flags: []pluginsdk.FlagSpec{{
        Name: "data", Type: "string", Body: true, Required: true,
        Usage: "JSON object, @file or - for stdin",
    }},
    Inputs: []pluginsdk.InputSpec{
        {Title: "Item name", Kind: "input", BodyPointer: "/name",
            ValueType: "string", Required: true},
        {Title: "Metadata", Kind: "text", BodyPointer: "/metadata",
            ValueType: "json", Description: "Optional JSON object"},
    },
}
```

The host reads inline/file/stdin body flags; SDK callers provide `Body` directly.
An explicit body bypasses JSON-pointer fields. Use `AlternativeArgument` for a
positional equivalent of a flag/context input. Use `AlternativeFlag` only for a
real declared alternative, such as `data` bypassing a payload-related flag.

Explicit arguments, flags, JSON fields, environment settings and saved targets
take precedence. Without explicit body input, missing required values open the
form. An explicit body bypasses the form even when incomplete: the plugin then
rejects missing required fields. Complete invocations run directly. Optional
blank fields stay omitted. Update forms should preserve
existing options unless the caller explicitly changes or clears them.

## Discover choices

Declare the read operation in the same plugin, then reference it with a full
`CommandPath`. The host injects a GET-only client for choice discovery.
For a runnable `show ITEM_ID` leaf with exactly one argument:

```go
pluginsdk.InputSpec{
    Title: "Item", Kind: "select", Argument: new(0),
    ValueType: "string", Required: true,
    Source: &pluginsdk.ChoiceSource{
        CommandPath: []string{"inventory", "list"},
        ValueField: "id", LabelFields: []string{"name", "id"},
        EmptyMessage: "No items are available; create one first",
    },
}
```

`ValueField` is the identifier passed to execution; labels are display only.
Source arguments and flags can refer to previously resolved `$arg0`, `$flag`,
`$organization` or `$stack`. Propagate any required experimental opt-in flag.
Secret fields use `Secret: true`, kind `input`, a string value type and no
default. Choice labels must not expose credentials.

Return the service's supported list envelope, such as `data` or `cursor.data`.
For cursor pagination preserve `hasMore` and `next`. For keyset pagination use
`AfterField` with the exact final-row identifier and declare the list's
`page-size` flag as `uint32` and `after` as `string`. Cursor lists need a
declared string `cursor` flag for subsequent pages. The core enumerates pages and
deduplicates choices; the plugin still implements the actual pagination API.

`ExcludeTrueFields` hides entries with a true boolean field. `MatchFields` ANDs
the declared fields and accepts any listed value per field. These filters can
restrict eligible lifecycle states. Pagination advances from original rows
before filtering, so an ineligible full page must not terminate discovery.
`PreferredPrefix` orders suggestions; it does not authorize selecting a fallback.

## Host behavior to preserve

Forms require terminal stdin and stderr and write to stderr; results write to
stdout. `--no-input`, `FCTL_NO_INPUT` and `CI` disable interaction. The host owns
cancellation, styles, keyboard navigation and confirmation defaulting to No.
Use metadata to extend these behaviors instead of adding prompts or output
calls to service code. `--confirm=false` and cancellation must cause no mutation.
