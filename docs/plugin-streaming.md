# Plugin streams and progress

Streaming is an optional extension of the public plugin SDK. Unary plugins keep
their `GetManifest` and `Execute` methods and protocol version 1. A command with
`CommandSpec.Streams` uses `StreamingPlugin.ExecuteStream`; an implementation
without that method fails explicitly, before the host reads an input stream.

## Contract

`StreamSpec.Input` and `StreamSpec.Output` declare raw byte streams.
`ExecutionStreams` supplies an optional reader, input size, writer and progress
reporter. They exist only during that invocation. The host owns sources and
destinations; a plugin must never open the user's file or print terminal output.

`FileSpec` selects the source and destination. Streaming does not convert raw
output into JSON or YAML: its format must match the declared stream format.
Bounded inline `Body` remains available where the plugin supports it, but cannot
be combined with a streamed input. Final `ExecuteResponse.Data` remains a bounded
JSON result and is distinct from streamed output.

For an input stream, a required Body flag describes the inline alternative; it
does not require a second inline body alongside the reader. Plugins validate
that their required stream or supported inline alternative is present.

The external runtime adds a bidirectional `ExecuteStream` RPC. The plugin pulls
input in chunks of at most 64 KiB; the host does not upload or read ahead of its
requests. Output uses bounded chunks and transport backpressure. Progress and
final results have their own frames. There is no automatic retry or replay.
Existing unary requests keep the 4 MiB Body and 32 MiB result limits.

Streaming HTTP callbacks preserve host authentication, endpoint and redirect
boundaries, credential-header filtering, request timeout and diagnostics. They
transfer bodies incrementally instead of buffering the complete HTTP response.

Canceling an external invocation terminates and reaps the executable. Plugins
must propagate cancellation and input, output or reporter failures. An
interruptible input reader or output writer must unblock on Close; the native
transport closes such host streams on cancellation. The CLI interrupts its own
stdin and stdout pipes. Plugins must not retain streams after returning.

## Progress

`ProgressEvent` carries `phase`, `completed`, optional `total`, and `unit`.
Counters are absolute. Absence of a total means indeterminate progress. Labels
are bounded and cannot contain terminal control characters. A total of zero is
valid for an empty operation.

The host renders progress on stderr in an interactive terminal and throttles
updates. Explicit JSON output, CI and disabled interactive input suppress terminal
progress. stdout remains the command result or raw export. Reading input and
committing work are separate phases: bytes consumed do not prove server
acceptance. Report errors must stop execution.

## Legacy Ledger

`legacy ledger import NAME FILE` and `legacy ledger export --ledger NAME`
use streams. Files and stdin no longer pass through a whole-file JSON Body.
The plugin keeps bounded NDJSON records and batches, preserves exact integers
and unknown fields, and reports confirmed import counts. Resume scans to the
backend's latest log ID; a missing checkpoint causes no import writes.

An import can commit successful batches before a later malformed record or
transport failure. Its final partial result counts only acknowledged batches;
a failed or ambiguous batch is never automatically retried. Use an explicit
resume after inspecting the service's state.

For an export to a file, the host streams into a private temporary file beside
the destination. Only successful completion replaces the destination. Failure
or cancellation removes the temporary file and preserves a previous export.
stdout exports are incremental and cannot be rolled back; they contain no
final summary or progress, including on failure.

Modern Ledger APIs do not gain historical log import/export endpoints from this
change. The extension belongs to the plugin contract; individual products must
implement their supported operations.

## Validation

The transport tests exchange 80 MiB through a real plugin executable and check
content, incremental delivery, progress, rejection before input reads, reporter
and writer failures, and process cleanup. Integration fixtures use real files
larger than 64 MiB, verify import/export SHA-256 and stdout separation, and run
the independently built legacy executable through the HTTP callback.

These are local API fixtures, not additional Cloud sandbox validation.
