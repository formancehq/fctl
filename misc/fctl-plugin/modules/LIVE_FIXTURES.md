# Synthetic fixtures for parent-run QA

These commands are prepared from the pinned historical CLI and cached SDK v4.0.0. They have **not been executed against Cloud**. Use a built binary and an already authenticated sandbox profile. The host owns authentication, file reads, and YAML decoding.

The supplied `/private/tmp/fctl-legacy-payment-providers.json` contains only a `data` map, with 22 providers and **no `dummypay` entry or version information**. Service versions below are those reported by the parent: Payments 3.4.8, Orchestration 2.7.0, Reconciliation 2.5.0, Wallets 2.2.1, Webhooks 2.5.4. A positive DUMMYPAY install cannot be established from this snapshot. The historical install behavior permits a provider absent from configs; the backend may still reject it. Stop the Payments-dependent sequence if installation fails.

## Common setup

Run in the parent's QA shell. Set `QA_PROFILE` to its existing sandbox profile and `FCTL_BIN` to the built binary. `QA_LEDGER` must identify the parent's dedicated existing Ledger fixture. No credentials are placed in these commands. Do not enable debug tracing or shell tracing.

```sh
: "${FCTL_BIN:?Set the path to the parent-built fctl binary}"
: "${QA_PROFILE:?Set the authenticated sandbox profile name}"
: "${QA_ORGANIZATION:?Set the organization ID}"
: "${QA_STACK:?Set the disposable sandbox Stack ID}"
: "${QA_LEDGER:?Set the existing dedicated QA ledger name}"
QA_TAG="legacy-qa-$(date -u +%Y%m%dT%H%M%SZ)"
QA_DIR="$(mktemp -d)"
QA_ACCOUNT="fctl:legacy:qa:${QA_TAG}"
fctlqa() {
  "$FCTL_BIN" --profile "$QA_PROFILE" --organization "$QA_ORGANIZATION" --stack "$QA_STACK" --no-input --output json "$@"
}
```

Responses remain their API envelopes; successful object IDs are usually `.data.id`. File sources are optional alternatives to `--data`; do not combine them. All examples below use synthetic names and amounts.

## Payments: conditional DUMMYPAY install, account, pool

`directory` is a path **inside the Payments backend**, not the parent's machine. `/tmp` is a synthetic candidate; the parent must select an allowed existing server directory if it differs. DUMMYPAY requires backend development support, which the providers snapshot does not establish. The config has no PSP credentials.

```sh
jq -n --arg name "$QA_TAG" \
  '{name:$name,directory:"/tmp",pollingPeriod:"30m"}' > "$QA_DIR/dummypay.json"
connector_json="$(fctlqa legacy payments connectors install dummypay "$QA_DIR/dummypay.json" --confirm)"
connector_id="$(printf '%s' "$connector_json" | jq -er '.data | select(type == "string" and length > 0)')"

created_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
jq -n --arg connector "$connector_id" --arg reference "$QA_TAG" --arg at "$created_at" \
  '{connectorID:$connector,createdAt:$at,reference:$reference,type:"INTERNAL",accountName:$reference,defaultAsset:"USD/2",metadata:{fixture:$reference}}' \
  > "$QA_DIR/payment-account.json"
account_json="$(fctlqa legacy payments accounts create "$QA_DIR/payment-account.json" --confirm)"
account_id="$(printf '%s' "$account_json" | jq -er '.data.id')"
fctlqa legacy payments accounts get "$account_id"
fctlqa legacy payments accounts balances "$account_id"

jq -n --arg name "$QA_TAG" --arg account "$account_id" \
  '{name:$name,accountIDs:[$account]}' > "$QA_DIR/pool.json"
pool_json="$(fctlqa legacy payments pools create "$QA_DIR/pool.json" --confirm)"
pool_id="$(printf '%s' "$pool_json" | jq -er '.data.id')"
fctlqa legacy payments pools get "$pool_id"
fctlqa legacy payments pools latest-balances "$pool_id"
```

On Payments 3, the module uses V3 for install and pool create/latest-balances, and the historical unversioned V1 endpoints for account create/get/balances and pool get/delete/member operations. Every Payments execution first reads `/_info`. Accounts have no historical delete command; clean up the synthetic connector after removing its pool references. Keep the pool until the Reconciliation fixture finishes.

## Wallet: create, credit, pending debit/void, drain

```sh
wallet_json="$(fctlqa legacy wallets create "$QA_TAG" --metadata "fixture=$QA_TAG" --ik "$QA_TAG-create" --confirm)"
wallet_id="$(printf '%s' "$wallet_json" | jq -er '.data.id')"
fctlqa legacy wallets show --id "$wallet_id"
fctlqa legacy wallets credit 100 USD/2 --id "$wallet_id" --source account=world \
  --metadata "fixture=$QA_TAG" --ik "$QA_TAG-credit" --confirm
fctlqa legacy wallets balances show main --id "$wallet_id"

hold_json="$(fctlqa legacy wallets debit 20 USD/2 --id "$wallet_id" --balance main --pending \
  --destination "account=$QA_ACCOUNT:sink" --description "$QA_TAG-hold" --ik "$QA_TAG-hold" --confirm)"
hold_id="$(printf '%s' "$hold_json" | jq -er '.data.id')"
fctlqa legacy wallets holds show "$hold_id"
fctlqa legacy wallets holds void "$hold_id" --ik "$QA_TAG-void" --confirm

fctlqa legacy wallets debit 100 USD/2 --id "$wallet_id" --balance main \
  --destination "account=$QA_ACCOUNT:sink" --description "$QA_TAG-drain" --ik "$QA_TAG-drain" --confirm
fctlqa legacy wallets balances show main --id "$wallet_id"
fctlqa legacy wallets transactions list --id "$wallet_id"
```

`world` is the synthetic credit source. The fixture debits the same 100 units after voiding its pending hold. **Wallet delete does not exist in the historical command tree or SDK/API.** The drained wallet and its transaction history remain; no delete command is fabricated. A repeated run should use a fresh tag, rather than repeat amounts against the same wallet.

## Workflow: historical YAML file, run, trigger, cleanup

```sh
cat > "$QA_DIR/workflow.yaml" <<EOF
name: $QA_TAG
stages:
  - delay:
      duration: 1s
EOF
workflow_json="$(fctlqa legacy orchestration workflows create "$QA_DIR/workflow.yaml")"
workflow_id="$(printf '%s' "$workflow_json" | jq -er '.data.id')"
fctlqa legacy orchestration workflows show "$workflow_id"
instance_json="$(fctlqa legacy orchestration workflows run "$workflow_id" --wait)"
instance_id="$(printf '%s' "$instance_json" | jq -er '.data.id')"
fctlqa legacy orchestration instances show "$instance_id"
fctlqa legacy orchestration instances describe "$instance_id"

trigger_json="$(fctlqa legacy orchestration triggers create fctl.legacy.synthetic "$workflow_id" \
  --name "$QA_TAG" --filter 'true')"
trigger_id="$(printf '%s' "$trigger_json" | jq -er '.data.id')"
fctlqa legacy orchestration triggers show "$trigger_id"
fctlqa legacy orchestration triggers test "$trigger_id" '{"fixture":"synthetic"}'
fctlqa legacy orchestration triggers occurrences list "$trigger_id"
fctlqa legacy orchestration triggers delete "$trigger_id" --confirm
fctlqa legacy orchestration workflows delete "$workflow_id" --confirm
```

The delay stage uses the historical `delay.duration` shape. Trigger test uses `/v2/triggers/{id}/test`; other workflow/trigger commands use the historical unversioned API. Delete the trigger before its workflow. These creation/run commands historically have no confirmation gate.

## Reconciliation: policy, reconcile, rule, evaluate, cleanup

Requires the parent's dedicated existing ledger and `pool_id` from the successful Payments sequence. An account with no balance history may yield empty balances or a backend reconciliation error; account creation alone does not create a PSP balance snapshot. Check the returned reconciliation status/error as well as the HTTP result.

```sh
: "${pool_id:?Payments pool creation must have succeeded}"
jq -n --arg name "$QA_TAG" --arg ledger "$QA_LEDGER" --arg pool "$pool_id" --arg address "$QA_ACCOUNT" \
  '{name:$name,ledgerName:$ledger,paymentsPoolID:$pool,ledgerQuery:{"$match":{address:$address}}}' \
  > "$QA_DIR/policy.json"
policy_json="$(fctlqa legacy reconciliation policies create "$QA_DIR/policy.json" --confirm)"
policy_id="$(printf '%s' "$policy_json" | jq -er '.data.id')"
fctlqa legacy reconciliation policies get "$policy_id"
at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
reconciliation_json="$(fctlqa legacy reconciliation policies reconcile "$policy_id" "$at" "$at")"
reconciliation_id="$(printf '%s' "$reconciliation_json" | jq -er '.data.id')"
fctlqa legacy reconciliation get "$reconciliation_id"
```

The rule fixture can run independently of DUMMYPAY. It uses two equal queries with opposite signs and an explicit USD tolerance. `ledger_invariant` is a backend template; its spec is forwarded intact. The synthetic account filter confines the rule to the parent's QA ledger.

```sh
jq -n --arg name "$QA_TAG" --arg ledger "$QA_LEDGER" --arg address "$QA_ACCOUNT" \
  '{name:$name,templateKind:"ledger_invariant",enabled:true,severity:"info",cadence:"continuous",periodType:"continuous",schedule:{kind:"on_demand"},labels:{fixture:$name},templateSpec:{terms:[{ledger:$ledger,query:{"$match":{address:$address}},sign:1},{ledger:$ledger,query:{"$match":{address:$address}},sign:-1}],tolerance:{"USD/2":0}}}' \
  > "$QA_DIR/rule.json"
rule_json="$(fctlqa legacy reconciliation rules create "$QA_DIR/rule.json" --confirm)"
rule_id="$(printf '%s' "$rule_json" | jq -er '.data.id')"
fctlqa legacy reconciliation rules get "$rule_id"
evaluation_json="$(fctlqa legacy reconciliation rules evaluate "$rule_id" --safety-margin 1s --confirm)"
evaluation_id="$(printf '%s' "$evaluation_json" | jq -er '.data.id')"
fctlqa legacy reconciliation evaluations get "$evaluation_id"
printf '%s\n' '{"enabled":false}' > "$QA_DIR/rule-patch.json"
fctlqa legacy reconciliation rules update "$rule_id" "$QA_DIR/rule-patch.json" --confirm
fctlqa legacy reconciliation rules delete "$rule_id" --confirm
```

Rule deletion also deletes its evaluations and alerts. If the policy fixture was created, delete it before the pool:

```sh
fctlqa legacy reconciliation policies delete "$policy_id" --confirm
fctlqa legacy payments pools remove-account "$pool_id" "$account_id" --confirm
fctlqa legacy payments pools delete "$pool_id" --confirm
fctlqa legacy payments connectors uninstall --connector-id "$connector_id" --confirm
```

Connector uninstall may complete asynchronously. If its response carries a task ID, check it with `fctlqa legacy payments tasks get TASK_ID` before declaring cleanup complete. There is no independent account delete in the historical CLI.

## Webhook: non-routable URL, create/delete

Capture the create response without printing it because it can contain a generated signing secret. The reserved `.invalid` URL has no real recipient; do not run `webhooks test` or replay commands for this fixture.

```sh
webhook_json="$(fctlqa legacy webhooks create https://fctl-legacy-qa.invalid/webhook fctl.legacy.synthetic --confirm)"
webhook_id="$(printf '%s' "$webhook_json" | jq -er '.data.id')"
fctlqa legacy webhooks delete "$webhook_id" --confirm
```

## Historical underscore compatibility

| Historical caller path | Canonical plugin path |
| --- | --- |
| `payments bank_accounts ...` | `payments bank-accounts ...` |
| `payments transfer_initiation ...` | `payments transfer-initiation ...` |
| `payments transfer_initiation update_status ...` | `payments transfer-initiation update-status ...` |
| `payments payment_initiations ...` | `payments transfer-initiation ...` |

The first three are present in the pinned main command tree. `payment_initiations` is additional compatibility explicitly requested by the parent. No other canonical underscore names were found in the five historical module trees. Aliases are declared in the module manifests. The ambiguous historical `p` remains Payments' payments-group alias, `cr` remains wallet-create's alias, `r` remains transfer-retry's alias; `holds h` is unavailable because the host reserves `h` for help.
