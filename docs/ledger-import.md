# Ledger import

`fctl ledger import` sends a file of ledger logs to the ledger's v2
`ImportLogs` endpoint. The code lives in `cmd/ledger/import.go`.

```bash
fctl ledger import <ledger name> <file path> [--resume-from-last-log]
```

## Input file

One JSON log per line, as exported by the ledger, in ascending log ID
order. Lines have no length limit. A final line without a trailing newline
is still imported.

## Flags

| Flag | Default | Effect |
| --- | --- | --- |
| `--resume-from-last-log` | `false` | Skip the logs the ledger already has and continue after its last log. See [Resuming](#resuming-an-interrupted-import). |
| `--file` | `""` | Declared but not read. The file always comes from the `<file path>` argument. |

## Batching and errors

The file is sent in requests of 100 lines each, plus one for any remainder.
The command stops at the first read error or failed request and returns it.
"Ledger imported!" is printed only after every line has been sent.

Batches sent before a failure stay in the ledger. Running the same import
again without `--resume-from-last-log` then fails: the ledger rejects any
log whose ID is not greater than its last log, with a 400 `IMPORT` error
whose message reads like:

```text
importing logs: log 99 already exists
```

## Resuming an interrupted import

With `--resume-from-last-log`, `fctl`:

1. Fetches the ledger's most recent log.
2. If the ledger has no logs, imports the whole file.
3. Otherwise reads the file from the start until it finds the line whose
   `id` equals that log's ID, then imports everything after it.

The input file must therefore contain the ledger's last log ID, and every
line up to it must be valid JSON. Use the same file as the interrupted
import.

If the ID is not in the file, the command fails without sending anything:

```text
log 4521 not found in ./ledger-export.jsonl
```

This means the file does not match the ledger's current state, for example
a different export or one that ends before the ledger's last log. If the
ledger's last log is the file's last line, there is nothing left to import
and the command succeeds.

### Migration

Before fctl PR #191, a missing ID made the command skip to the end
of the file, import nothing, and print "Ledger imported!". Scripts that
relied on that exit code now get a non-zero exit and should check that the
file and ledger match.
