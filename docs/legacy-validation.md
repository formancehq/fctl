# Legacy plugin validation

The legacy SDK plugin was exercised against a real Cloud sandbox on Stack line
**v3.2** on 2026-10-10. The recorded checks cover service commands, host forms,
file handling, authentication and provider selection. They establish the results
listed below; they do not establish complete coverage of every historical command.

See [legacy plugin architecture](legacy-plugin.md) for provider selection and
compatibility. The [synthetic fixture guide](../misc/fctl-plugin/docs/live-fixtures.md)
describes candidate commands and prerequisites. Its prepared examples are not,
by themselves, evidence that an operation ran successfully.

## Live target

These are the actual module versions on the tested v3.2 sandbox:

| Service | Version |
| --- | --- |
| Ledger | v2.4.15 |
| Auth | v2.5.1 |
| Payments | v3.4.8 |
| Orchestration | v2.7.0 |
| Reconciliation | v2.5.0 |
| Wallets | v2.2.1 |
| Webhooks | v2.5.4 |

Only this historical Stack line received live validation. Other declared
compatibility ranges have contract-test coverage, not equivalent live coverage.
A separate **v4.0-beta** Cloud target was used to check the modern-provider boundary.

## Recorded results

The following local reports are in `/private/tmp/`. Counts are checks in each
report, not distinct commands or percentages of the command tree. Some reports
repeat operations from earlier runs. `PASS` must be interpreted with the stated
assertion boundary.

| Report | Recorded result | Coverage |
| --- | --- | --- |
| `fctl-legacy-live-report.json` | 36 PASS | Ledger create/list/stats, metadata, large-integer send, account and transaction reads/metadata, volumes and revert; Auth client/secret lifecycle and user listing; reads across the other five services; provider selections. |
| `fctl-legacy-payment-webhook-report.json` | 11 PASS | Payments pool create/get/balances/delete, bank-account create/get/metadata; Webhook create/deactivate/update/delete. |
| `fctl-legacy-auth-modes-report.json` | 3 PASS | Standalone client-credentials authentication against real Auth, synthetic credential cleanup, and standalone `none` mode through the real CLI. Cloud session authentication is exercised by the main run. |
| `fctl-legacy-form-report.json` | 3 PASS | Ledger QA sequence: create and stats use terminal forms; list uses explicit arguments. Form results are valid stdout JSON. |
| `fctl-legacy-auth-form-report.json` | 2 PASS | Auth QA sequence: client creation uses a terminal form; deletion uses explicit arguments and confirmation. Creation returns valid stdout JSON. |
| `fctl-legacy-workflow-reconciliation-report.json` | 18 HTTP/CLI PASS | Workflow YAML create/show/run, instance show/history, trigger create/show/test/occurrences/delete and workflow delete; Reconciliation pool/policy create/read/reconcile/result read/delete and pool cleanup. Initial `--wait` completion semantics needed the follow-up below. |
| `fctl-legacy-workflow-assertions-report.json` | 3 PASS | Additional workflow create/run/delete checks. These do not replace the terminal-state assertion. |
| `fctl-legacy-workflow-wait-report.json` | 5 PASS | Final direct Cloud workflow create, run with `--wait`, terminal-state assertion (`terminated=true`), instance show readback and workflow delete. |
| `fctl-legacy-https-schema-report.json` | 2 PASS | Schema insertion from the published HTTPS fixture, then an exact version and chart readback. An empty creation response is valid. |
| `fctl-legacy-extra-report.json` | 9 PASS, 1 FAIL | Wallet create/show/update, balance create, credit/read/debit/transactions; bank-account create. The initial Payments pool payload failed validation because `query` was required. The later pool run passed. |
| `fctl-legacy-wallet-holds-report.json` | 7 PASS, 1 FAIL | Pending hold create/show/void, a second pending hold and confirm, and balance reads. An initial drain failed with insufficient funds; a subsequent balance read alone was not proof of zero. |
| `fctl-legacy-wallet-drain-report.json` | 6 PASS | Final reads, drain operations and explicit zero-balance checks for both tested wallet balances (`main` and the QA balance). |
| `fctl-legacy-clarity-report.json` | 8 PASS, 1 FAIL | Initial Clarity lifecycle attempt: evaluation failed because the rule was disabled. Its workflow-state label does not establish the original `--wait` semantics. |
| `fctl-legacy-clarity-enabled-report.json` | 8 PASS | With Clarity enabled: pool create, rule create/show/update/evaluate, evaluations list, rule delete and pool delete. |

The Ledger and Auth form sequences used real PTY interaction. Their totals
include the explicit read or cleanup commands shown above; they are not five
separate interactive forms.

The validation also confirmed the following live checks, which are not
given separate per-case counts in the reports above:

- Ledger file handling for Numscript execution, export, import and schema
  operations. The host reads source files and presents or writes results through
  the SDK body/result boundary.
- Modern Auth and Ledger reads on the v4.0-beta target, with explicit legacy
  execution rejected on that target.
- A native macOS `fctl-plugin-legacy` executable launched through the public SDK
  transport completed a real service read. This checks the independent process
  and host HTTP callback, in addition to the embedded service facades.

## Workflow wait correction

Orchestration v2.7.0 initially returned an active instance from the run request
even with `wait=true`. A successful HTTP response from that request did not prove
that the workflow had terminated. This explains the boundary of the original
18-check workflow/Reconciliation report.

The plugin now follows the returned instance with terminal-state polling when
`--wait` is set. It polls every **500 ms**, with a **five-minute polling deadline**,
and honors context cancellation. The deadline applies to the follow-up wait;
the initial run request still has the host's HTTP timeout. Polling reads the same
instance and does not start another run. Missing state, an unexpected instance ID,
a reported workflow error or a failed read produces an error.

The final direct Cloud run records five PASS checks in
`fctl-legacy-workflow-wait-report.json`. It explicitly verifies `terminated=true`
and reads the instance back with `instances show`. The terminal-state check took
4.264 seconds. The original server behavior remains relevant context, but the
plugin correction and this final run resolve the workflow-wait validation gap.

## Automated validation

The final checkout passed `just pre-commit` (tidy, generation and lint across
the core, SDK and legacy modules), `just tests` with the race detector, and
`goreleaser check`. GoReleaser built the legacy executable for all six Linux,
macOS and Windows amd64/arm64 targets. The rebuilt native macOS executable
then repeated a real Ledger read through the SDK transport. These build checks
do not establish runtime validation on the other platforms.

After the branch was pushed, a fresh independent Go consumer downloaded
`github.com/formancehq/fctl/misc/fctl-plugin` at
`v0.0.0-20261010100529-b96046bc717b`, without a local `replace` or workspace.
It resolved the public SDK dependency and read the seven-service manifest.
This verifies that consumers can import the published module independently
of the fctl checkout; it is separate from binary release publication.

## Reproduce selected checks

Use the built binary and an authenticated sandbox profile. Replace the
variables with a dedicated disposable target and fixture names. These examples
contain no organization IDs, Stack IDs or credentials. Keep shell tracing and
HTTP debug tracing disabled when recording results.

```sh
: "${FCTL_BIN:?Set the path to the built fctl binary}"
: "${QA_PROFILE:?Set the saved sandbox profile name}"
: "${QA_ORGANIZATION:?Set the sandbox organization ID}"
: "${QA_STACK:?Set the v3.2 sandbox Stack ID}"
: "${QA_LEDGER:?Set the dedicated QA ledger name}"
fctlqa() {
  "$FCTL_BIN" --profile "$QA_PROFILE" \
    --organization "$QA_ORGANIZATION" --stack "$QA_STACK" \
    --no-input --output json "$@"
}

fctlqa ledger list
fctlqa ledger stats --ledger "$QA_LEDGER"
fctlqa legacy auth clients list
fctlqa legacy payments pools list
fctlqa plugins selections
```

| Flag | Purpose in validation |
| --- | --- |
| `--profile` | Use an existing saved profile and its host-managed authentication. |
| `--organization`, `--stack` | Select the exact Cloud target. |
| `--no-input` | Disable forms and selectors for scripted checks. |
| `--output json` | Preserve machine-readable results for assertions. |
| `--ledger` | Scope Ledger operations to the dedicated fixture. |
| `--confirm` | Explicitly authorize commands that declare confirmation. |
| `--data` | Supply inline JSON, `@file` or stdin (`-`) for commands that accept an SDK body. |
| `--auth-mode` | Select `cloud`, `client-credentials` or `none`; use separate prepared profiles for standalone checks. |

For a PTY form check, invoke the built binary in a terminal, omit `--no-input`
and leave the required field unset. The recorded examples are `ledger create`,
`ledger stats` without a ledger selection, and `auth clients create`, with
`--color always` and `--output json`. Read stdout separately from terminal prompts.

To repeat the file checks, prepare synthetic `transfer.num`, `schema.yaml` and
an empty import target using the host's file boundary. Set `QA_DIR`,
`QA_IMPORT_LEDGER` and `QA_SCHEMA_VERSION` before running:

```sh
fctlqa ledger transactions num "$QA_DIR/transfer.num" --ledger "$QA_LEDGER" --confirm
fctlqa ledger export --ledger "$QA_LEDGER" --file "$QA_DIR/logs.ndjson"
fctlqa ledger create "$QA_IMPORT_LEDGER" --confirm
fctlqa ledger import "$QA_IMPORT_LEDGER" "$QA_DIR/logs.ndjson"
fctlqa ledger schemas insert "$QA_SCHEMA_VERSION" "$QA_DIR/schema.yaml" \
  --ledger "$QA_LEDGER" --confirm
fctlqa ledger schemas get "$QA_SCHEMA_VERSION" --ledger "$QA_LEDGER"
```

For the wait check, create a synthetic workflow from a YAML file containing
`stages: [{delay: {duration: 1s}}]` and a synthetic name. With `jq` available:

```sh
workflow_json="$(fctlqa legacy orchestration workflows create "$QA_DIR/workflow.yaml")"
workflow_id="$(printf '%s' "$workflow_json" | jq -er '.data.id')"
instance_json="$(fctlqa legacy orchestration workflows run "$workflow_id" --wait)"
printf '%s' "$instance_json" | jq -e '.data.terminated == true'
instance_id="$(printf '%s' "$instance_json" | jq -er '.data.id')"
fctlqa legacy orchestration instances show "$instance_id" \
  | jq -e '.instance.data.terminated == true'
fctlqa legacy orchestration workflows delete "$workflow_id" --confirm
```

For the modern boundary, repeat the normal Auth and Ledger reads with a prepared
v4.0-beta target and its modern plugins. An explicit `legacy` service command
must return an error on that target. For standalone authentication, select saved
profiles prepared separately for `client-credentials` and `none`; keep credentials
in the host configuration rather than copied into commands or reports.

## Coverage and delivery boundaries

- **PSP-linked Payments mutations remain untested.** No PSP was available.
  Pool and bank-account checks do not establish connector installation,
  PSP account/payment flows, transfers or transfer-initiation mutations. The
  conditional DUMMYPAY sequence in the fixture guide is not a successful live run.
- **No outside webhook send was performed.** Webhook lifecycle checks used a
  non-routable `.invalid` destination. Test-send, replay and external delivery
  success are outside this record.
- Reconciliation policy operations and enabled Clarity rule evaluation passed
  the recorded checks. They do not establish reconciliation against real PSP
  balance history or every rule template and workflow stage.
- Wallet pending debit, confirm and void were exercised. Final tested balances
  are verified as zero by the drain report. The wallet and transaction history
  remain; the historical API has no wallet-delete command. Cleanup of reported
  fixtures does not imply removal of all sandbox data.
- The initial integration embeds the SDK facades in fctl. The independent
  executable's macOS read passed, but no registry publication or release
  publication is established. This record does not claim live execution on
  Linux, Windows or every advertised architecture.
- Contract tests validate declared API families and failure cases separately
  from live QA. Automated checks and live QA establish separate results; commit, push,
  merge and deployment remain separate delivery states.

Report future results by target version, operation, assertion and remaining
boundary. Preserve failed attempts and their successful follow-ups. Keep secrets,
generated Auth/Webhook secrets and raw authenticated responses out of shared
reports. Do not describe these selected checks as “100% of all commands tested.”
