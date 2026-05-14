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
- `fctl context unset-defaults [name] --confirm`
- `fctl config migrate-v3`
- `fctl setup`
- `fctl target inspect`
- `fctl target proxy --port 55001`

## Cloud

- `fctl cloud me show`
- `fctl cloud me invitations list`
- `fctl cloud me invitations accept <invitation-id> --confirm`
- `fctl cloud me invitations decline <invitation-id> --confirm`
- `fctl cloud organizations create <name>`
- `fctl cloud organizations list`
- `fctl cloud organizations show <organization-id>`
- `fctl cloud organizations history [organization-id] --action <action> --user-id <user-id> --data key=value`
- `fctl cloud organizations update <organization-id> --name <name>`
- `fctl cloud organizations delete <organization-id> --confirm`
- `fctl cloud organizations applications list --organization <organization-id>`
- `fctl cloud organizations applications show <application-id> --organization <organization-id>`
- `fctl cloud organizations authentication-provider show --organization <organization-id>`
- `fctl cloud organizations authentication-provider configure --type <type> --name <name> --client-id <client-id> --client-secret-stdin --organization <organization-id>`
- `fctl cloud organizations authentication-provider delete --organization <organization-id> --confirm`
- `fctl cloud organizations oauth-clients create --name <name> --organization <organization-id> --confirm`
- `fctl cloud organizations oauth-clients list --organization <organization-id>`
- `fctl cloud organizations oauth-clients show <client-id> --organization <organization-id>`
- `fctl cloud organizations oauth-clients update <client-id> --name <name> --organization <organization-id> --confirm`
- `fctl cloud organizations oauth-clients delete <client-id> --organization <organization-id> --confirm`
- `fctl cloud organizations invitations list --organization <organization-id>`
- `fctl cloud organizations invitations send <email> --organization <organization-id>`
- `fctl cloud organizations invitations delete <invitation-id> --organization <organization-id> --confirm`
- `fctl cloud organizations users list --organization <organization-id>`
- `fctl cloud organizations users show <user-id> --organization <organization-id>`
- `fctl cloud organizations users link <user-id> --organization <organization-id> --policy-id <policy-id>`
- `fctl cloud organizations users unlink <user-id> --organization <organization-id> --confirm`
- `fctl cloud organizations policies create <name> --organization <organization-id>`
- `fctl cloud organizations policies list --organization <organization-id>`
- `fctl cloud organizations policies show <policy-id> --organization <organization-id>`
- `fctl cloud organizations policies update <policy-id> --organization <organization-id> --name <name>`
- `fctl cloud organizations policies delete <policy-id> --organization <organization-id> --confirm`
- `fctl cloud organizations policies add-scope <policy-id> <scope-id> --organization <organization-id>`
- `fctl cloud organizations policies remove-scope <policy-id> <scope-id> --organization <organization-id> --confirm`
- `fctl cloud regions create <name> --organization <organization-id>`
- `fctl cloud regions list --organization <organization-id>`
- `fctl cloud regions show <region-id> --organization <organization-id>`
- `fctl cloud regions delete <region-id> --organization <organization-id> --confirm`
- `fctl cloud apps create --organization <organization-id>`
- `fctl cloud apps list --organization <organization-id>`
- `fctl cloud apps show <app-id> --organization <organization-id>`
- `fctl cloud apps delete <app-id> --organization <organization-id> --confirm`
- `fctl cloud apps deploy <app-id> --file <manifest.yaml>`
- `fctl cloud apps runs list <app-id>`
- `fctl cloud apps runs show <run-id>`
- `fctl cloud apps runs logs <run-id>`
- `fctl cloud apps versions list <app-id>`
- `fctl cloud apps versions show <version-id>`
- `fctl cloud apps versions manifest <version-id>`
- `fctl cloud apps versions archive show <version-id>`
- `fctl cloud apps variables list <app-id>`
- `fctl cloud apps variables create <app-id> --key <key> --value <value>|--value-stdin`
- `fctl cloud apps variables delete <app-id> <variable-id> --confirm`

Cloud commands require a `cloud` or `cloud-stack` context. They are not required
for direct local or self-hosted stack commands. `cloud apps` talks to the Cloud
apps deploy server; use `--deploy-url <url>` to target a non-production deploy
server.

## Cloud Stacks

- `fctl cloud stacks create <name> --region <region-id>`
- `fctl cloud stacks list --organization <organization-id>`
- `fctl cloud stacks show <stack-id> --organization <organization-id>`
- `fctl cloud stacks update <stack-id> --name <name>`
- `fctl cloud stacks delete <stack-id> --confirm`
- `fctl cloud stacks enable <stack-id>`
- `fctl cloud stacks disable <stack-id> --confirm`
- `fctl cloud stacks restore <stack-id> --confirm`
- `fctl cloud stacks upgrade <stack-id> --version <version> --confirm`
- `fctl cloud stacks history <stack-id>`
- `fctl cloud stacks users list <stack-id>`
- `fctl cloud stacks users link <stack-id> <user-id> --policy-id <policy-id>`
- `fctl cloud stacks users unlink <stack-id> <user-id> --confirm`
- `fctl cloud stacks modules list <stack-id>`
- `fctl cloud stacks modules enable <stack-id> <module>`
- `fctl cloud stacks modules disable <stack-id> <module> --confirm`

When the active context is `cloud-stack`, `--organization` defaults to the
context organization. `cloud_stacks`, `stack`, and `stacks` are deprecated
aliases for `cloud stacks`.

## Ledger

- `fctl ledger transactions list`
- `fctl ledger transactions show <transaction-id>`
- `fctl ledger transactions send`
- `fctl ledger transactions run-script --file <path>|-`
- `fctl ledger transactions revert <transaction-id>`
- `fctl ledger transactions count`
- `fctl ledger accounts list`
- `fctl ledger accounts show <address>`
- `fctl ledger accounts query <query-id> --schema-version <version>`
- `fctl ledger set-metadata <ledger> [key=value]... --metadata-file <path>|- --confirm`
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

- `fctl auth login token --token-stdin --credential-dir <dir>`
- `fctl auth login client-credentials --issuer-url <url> --client-id <id> --client-secret-stdin --credential-dir <dir>`
- `fctl auth login none`
- `fctl auth status`
- `fctl auth token`
- `fctl auth logout --confirm`
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
