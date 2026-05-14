# fctl v4 Command Reference

This reference lists the current canonical v4 command families implemented under
`v4/`. It should eventually be generated from Cobra help to avoid drift.

## Target and Configuration

- `fctl context create stack <name> --stack-url <url>`
- `fctl context create cloud <name> --cloud-url <url>`
- `fctl context create cloud-stack <name> --cloud-url <url> --organization <organization-id> --stack <stack-id>`
- `fctl context list`
- `fctl context show <name>`
- `fctl context use <name>`
- `fctl context rename <old-name> <new-name>`
- `fctl context delete <name> --confirm`
- `fctl context set [name] --organization <organization-id> --stack <stack-id> --default-ledger <ledger>`
- `fctl config migrate-v3`
- `fctl setup`
- `fctl target inspect`

## Cloud

- `fctl cloud me show`
- `fctl cloud organizations create <name>`
- `fctl cloud organizations list`
- `fctl cloud organizations show <organization-id>`
- `fctl cloud organizations update <organization-id> --name <name>`
- `fctl cloud organizations delete <organization-id> --confirm`

Cloud commands require a `cloud` or `cloud-stack` context. They are not required
for direct local or self-hosted stack commands.

## Cloud Stacks

- `fctl cloud_stacks create <name> --region <region-id>`
- `fctl cloud_stacks list --organization <organization-id>`
- `fctl cloud_stacks show <stack-id> --organization <organization-id>`
- `fctl cloud_stacks update <stack-id> --name <name>`
- `fctl cloud_stacks delete <stack-id> --confirm`
- `fctl cloud_stacks enable <stack-id>`
- `fctl cloud_stacks disable <stack-id> --confirm`
- `fctl cloud_stacks restore <stack-id> --confirm`
- `fctl cloud_stacks upgrade <stack-id> --version <version> --confirm`
- `fctl cloud_stacks users list <stack-id>`
- `fctl cloud_stacks users link <stack-id> <user-id> --policy-id <policy-id>`
- `fctl cloud_stacks users unlink <stack-id> <user-id> --confirm`
- `fctl cloud_stacks modules list <stack-id>`
- `fctl cloud_stacks modules enable <stack-id> <module>`
- `fctl cloud_stacks modules disable <stack-id> <module> --confirm`

When the active context is `cloud-stack`, `--organization` defaults to the
context organization. `stack` and `stacks` are deprecated aliases for
`cloud_stacks`.

## Ledger

- `fctl ledger transactions list`
- `fctl ledger transactions show <transaction-id>`
- `fctl ledger transactions send`
- `fctl ledger transactions revert <transaction-id>`
- `fctl ledger transactions count`
- `fctl ledger accounts list`
- `fctl ledger accounts show <address>`
- `fctl ledger schemas list`
- `fctl ledger schemas show <schema-id>`
- `fctl ledger schemas insert`
- `fctl ledger volumes list`

Ledger commands use service-qualified internal names and adapt canonical CLI
flags to the selected Ledger API version.

## Payments

- `fctl payments versions`
- `fctl payments connectors install <connector> --file <path>|-`
- `fctl payments connectors list`
- `fctl payments connectors config show <connector-id>`
- `fctl payments connectors config update <connector-id> --file <path>|-`
- `fctl payments connectors uninstall <connector-id> --confirm`
- `fctl payments pools create --file <path>|-`
- `fctl payments pools list`
- `fctl payments pools show <pool-id>`
- `fctl payments pools delete <pool-id> --confirm`
- `fctl payments pools add-account <pool-id> <account-id>`
- `fctl payments pools remove-account <pool-id> <account-id> --confirm`
- `fctl payments pools update-query <pool-id> --file <path>|- --confirm`
- `fctl payments pools balances <pool-id> --at <time>`
- `fctl payments pools latest-balances <pool-id>`

Connector configuration commands always target a connector ID.

## Wallets

- `fctl wallets create <name>`
- `fctl wallets list`
- `fctl wallets show <wallet-id>`
- `fctl wallets update <wallet-id>`
- `fctl wallets credit <wallet-id> --amount <amount> --asset <asset> --confirm`
- `fctl wallets debit <wallet-id> --amount <amount> --asset <asset> --confirm`
- `fctl wallets balances create <wallet-id> <name>`
- `fctl wallets balances list <wallet-id>`
- `fctl wallets balances show <wallet-id> <balance-name>`
- `fctl wallets holds list`
- `fctl wallets holds show <hold-id>`
- `fctl wallets holds void <hold-id> --confirm`
- `fctl wallets holds confirm <hold-id> --confirm`
- `fctl wallets transactions list <wallet-id>`

Wallet credit and debit require the wallet target explicitly.

## Flows

- `fctl flows workflows create --file <path>|-`
- `fctl flows workflows list`
- `fctl flows workflows show <workflow-id>`
- `fctl flows workflows run <workflow-id>`
- `fctl flows workflows delete <workflow-id> --confirm`
- `fctl flows instances list`
- `fctl flows instances show <instance-id>`
- `fctl flows instances inspect <instance-id>`
- `fctl flows instances send-event <instance-id> <event>`
- `fctl flows instances stop <instance-id> --confirm`
- `fctl flows triggers create <event> <workflow-id>`
- `fctl flows triggers list`
- `fctl flows triggers show <trigger-id>`
- `fctl flows triggers delete <trigger-id> --confirm`
- `fctl flows triggers test <trigger-id>`
- `fctl flows triggers occurrences list <trigger-id>`

`orchestration` is a deprecated alias for `flows`.

## Reconciliation

- `fctl reconciliation list`
- `fctl reconciliation show <reconciliation-id>`
- `fctl reconciliation policies create --file <path>|- --confirm`
- `fctl reconciliation policies list`
- `fctl reconciliation policies show <policy-id>`
- `fctl reconciliation policies delete <policy-id> --confirm`
- `fctl reconciliation policies reconcile <policy-id> --ledger-at <time> --payments-at <time> --confirm`

## Auth

- `fctl auth clients create <name>`
- `fctl auth clients list`
- `fctl auth clients show <client-id>`
- `fctl auth clients update <client-id> --name <name>`
- `fctl auth clients delete <client-id> --confirm`
- `fctl auth clients secrets create <client-id> <secret-name>`
- `fctl auth clients secrets delete <client-id> <secret-id> --confirm`
- `fctl auth users list`
- `fctl auth users show <user-id>`

`auth` is the canonical service name.

## Webhooks

- `fctl webhooks create <endpoint> <event-type>...`
- `fctl webhooks list`
- `fctl webhooks activate <config-id>`
- `fctl webhooks deactivate <config-id> --confirm`
- `fctl webhooks delete <config-id> --confirm`
- `fctl webhooks secret rotate <config-id> --secret-stdin`

Plain output masks webhook secrets.
