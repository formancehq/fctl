# Streaming legacy Ledger logs

`legacy.New` and `legacy.NewService` return plugins that implement
`pluginsdk.StreamingPlugin` as well as the original unary `Plugin`. Protocol
version remains 1. Only Ledger import/export declare streams. Their existing
`FileSpec` remains unchanged: import reads a raw string source from argument 1
or `--file`; export writes NDJSON through `--file` or stdout. Historical aliases
and other command manifests remain unchanged.

The host uses `ExecuteStream(ctx, request, streams)` for those commands and
provides an `io.Reader` for import or an `io.Writer` for export. Do not supply
`request.Body` with a stream. The host owns and closes these resources. The
plugin never opens paths from arguments or flags. The same service/Stack version
guards run before stream IO and before mutation. Input is read on demand.

## Import

The source contains one JSON log object per line, with a nonnegative integer
`id`. The final line may lack a newline. Blank or malformed records fail. Raw
JSON preserves arbitrary integer precision and unknown fields. Import compacts
each object and appends a newline for the historical octet-stream endpoint.
Each batch contains at most 100 logs and at most 4 MiB, including newlines. A
single source record is limited to 4 MiB including its existing delimiter; its
compacted object plus the added delimiter must fit the batch limit. Oversized
records fail explicitly. Memory use is bounded by a batch, a record and their
encoding/validation buffers, independently of total source size.

Progress events are absolute counters:

| Phase | Unit | Completed | Total |
| --- | --- | --- | --- |
| `read` | `bytes` | Source bytes read, including whitespace/newlines and bounded read-ahead | Optional `InputSize`, including an explicit zero for an empty input |
| `import` | `logs` | Logs in successful, acknowledged batches | Absent |
| `export` | `logs` | Complete validated records written to the host writer | Absent |

Bytes read do not imply successful import. A final import response contains
`{"imported":N,"skipped":N,"batches":N}`. A response also accompanies a failure;
imported/batch counts include only acknowledged successful batches. A structured
service failure adds its raw `error` object. Read, cancellation and reporter
errors stop execution. A pending batch is discarded when input validation fails.

`--resume-from-last-log` fetches the latest backend log via
`GET /v2/NAME/logs?pageSize=1`. The plugin scans until it finds that exact integer
ID, skips through it, then begins writing. It does not compare IDs through
floating point or infer a checkpoint from ordering. An empty backend imports
from the beginning. A missing/invalid checkpoint makes no import writes;
`skipped` remains zero until the checkpoint has been found.

No batch is automatically retried. A lost acknowledgment can mean the backend
already accepted it. Stop, inspect backend state, and start a new invocation
with resume when appropriate. Concurrent import invocations are not coordinated
by this plugin; backend log/hash validation remains authoritative.

## Export

The plugin consumes the HTTP response incrementally. It validates each JSON log
record and its integer ID before writing that record. Records are limited to
4 MiB including an existing newline. It preserves backend bytes exactly,
including spacing, unknown fields, huge numbers, CRLF and a final missing
newline. It never creates a complete JSON array or buffers the full response.
The final response is only `{"exported":N,"bytes":N}` and is separate from the
NDJSON output. The host must bypass table/JSON rendering for raw output.

A malformed late record, backend read/close failure, output error, cancellation
or reporter error fails execution even after a valid prefix has been written.
The host must stage a file export and replace the destination only after full
success, so a failed export preserves an existing destination. A stdout caller
may already have received a valid prefix when a later record fails. Direct SDK
consumers must provide their own staging if they need atomic file replacement.
Arbitrary readers/writers must cooperate with cancellation to interrupt blocked
IO; the plugin checks context between reads, writes, reports and requests.

## Unary compatibility and verification

Existing direct SDK callers may continue using `Execute`. Import accepts a
JSON array or a JSON string of NDJSON within the 4 MiB Body limit and validates
all logs before writing. Unary export remains bounded at 32 MiB and returns a
JSON array. New hosts should prefer `ExecuteStream` for both commands.

Run from this module:

```sh
go test ./internal/ledger . -run '^TestStream'
go test -race ./...
```

Tests generate an import exceeding 64 MiB from one reusable record and stream
an export exceeding 64 MiB into a temporary file. They cover early output before
EOF, both batch bounds, exact integers, resume without checkpoint writes, late
malformed records, failed-batch non-replay, read/write/report/cancellation errors,
zero totals, unchanged file metadata and compatibility checks before stream IO.
