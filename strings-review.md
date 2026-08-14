# fctl User-Facing Strings — Grammar Review

All user-facing strings extracted from `cmd/**/*.go`. Only entries needing a change are listed.
Each entry shows: **file:line**, current string, and suggested replacement.

Legend:
- 🐛 Bug / clearly wrong (typo, wrong verb, wrong description entirely)
- ⚠️ Grammar / style issue
- Strings marked ✅ OK are omitted.

---

## 1. Command Short Descriptions (`WithShortDescription`)

### auth

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/auth/clients/create.go:69` | `"Create client"` | `"Create a client"` |
| `cmd/auth/clients/delete.go:39` | `"Delete client"` | `"Delete a client"` |
| `cmd/auth/clients/root.go:14` | `"Clients management"` | `"Manage clients"` |
| `cmd/auth/clients/secrets/create.go:40` | `"Create secret"` | `"Create a secret"` |
| `cmd/auth/clients/secrets/delete.go:38` | `"Delete secret"` | `"Delete a secret"` |
| `cmd/auth/clients/secrets/root.go:12` | `"Secrets management"` | `"Manage secrets"` |
| `cmd/auth/clients/show.go:40` | `"Show client"` | `"Show a client"` |
| `cmd/auth/clients/update.go:68` | `"Update client"` | `"Update a client"` |
| `cmd/auth/root.go:13` | `"Auth server management"` | `"Manage the auth server"` |
| `cmd/auth/users/root.go:11` | `"Users management"` | `"Manage users"` |
| `cmd/auth/users/show.go:36` | `"Show user"` | `"Show a user"` |

### cloud / apps

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/cloud/apps/create.go:36` | `"Create apps"` | `"Create an app"` |
| `cmd/cloud/apps/delete.go:34` | `"Delete apps"` | `"Delete an app"` |
| `cmd/cloud/apps/deploy.go:42` | `"Deploy apps"` | `"Deploy an app"` |
| `cmd/cloud/apps/root.go:16` | `"* New * Apps manifests management"` | `"Manage app manifests"` |
| `cmd/cloud/apps/runs/root.go:11` | (already OK: `"Manage app runs"`) | — |
| `cmd/cloud/apps/runs/show.go:38` | `"Show run"` | `"Show a run"` |
| `cmd/cloud/apps/show.go:42` | `"Show apps"` | `"Show an app"` |
| `cmd/cloud/apps/variables/create.go:38` | `"Create new variable for an app"` | `"Create a new variable for an app"` |
| `cmd/cloud/apps/versions/manifest.go:28` | `"Manifest versions for an app"` | `"Show the manifest for an app version"` |
| `cmd/cloud/apps/versions/show.go:38` | `"Show version"` | `"Show a version"` |

### cloud / me

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/cloud/me/invitations/accept.go:35` | `"Accept invitation"` | `"Accept an invitation"` |
| `cmd/cloud/me/invitations/decline.go:35` | `"Decline invitation"` | `"Decline an invitation"` |
| `cmd/cloud/me/invitations/root.go:11` | `"Invitations management"` | `"Manage invitations"` |
| `cmd/cloud/me/root.go:12` | `"Current user management"` | `"Manage current user settings"` |

### cloud / organizations

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/cloud/organizations/applications/list.go:43` | `"List applications available for organization"` | `"List applications available for the organization"` |
| `cmd/cloud/organizations/applications/root.go:12` | `"Applications management"` | `"Manage applications"` |
| `cmd/cloud/organizations/authentication-provider/configure.go:39` | `"Configure authorization provider for organization"` | `"Configure the authorization provider for the organization"` |
| `cmd/cloud/organizations/authentication-provider/delete.go:31` | `"Delete authorization provider of organization"` | `"Delete the authorization provider for the organization"` |
| `cmd/cloud/organizations/authentication-provider/root.go:11` | `"Authentication provider management"` | `"Manage the authentication provider"` |
| `cmd/cloud/organizations/authentication-provider/show.go:36` | `"Show authorization provider of organization"` | `"Show the authorization provider for the organization"` |
| `cmd/cloud/organizations/create.go:37` | `"Create organization"` | `"Create an organization"` |
| `cmd/cloud/organizations/delete.go:35` | `"Delete organization"` | `"Delete an organization"` |
| `cmd/cloud/organizations/describe.go:37` | `"Describe organization"` | `"Describe an organization"` |
| `cmd/cloud/organizations/invitations/root.go:12` | `"Invitations management"` | `"Manage invitations"` |
| `cmd/cloud/organizations/oauth-clients/create.go:41` | `"Create organization OAuth client"` | `"Create an organization OAuth client"` |
| `cmd/cloud/organizations/oauth-clients/delete.go:31` | `"Delete organization OAuth client"` | `"Delete an organization OAuth client"` |
| `cmd/cloud/organizations/oauth-clients/root.go:11` | `"Oauth clients management"` | `"Manage OAuth clients"` |
| `cmd/cloud/organizations/oauth-clients/show.go:35` | `"Show organization OAuth client"` | `"Show an organization OAuth client"` |
| `cmd/cloud/organizations/oauth-clients/update.go:37` | `"Update organization OAuth client"` | `"Update an organization OAuth client"` |
| `cmd/cloud/organizations/policies/root.go:11` | `"Policies management"` | `"Manage policies"` |
| `cmd/cloud/organizations/root.go:18` | `"Organizations management"` | `"Manage organizations"` |
| `cmd/cloud/organizations/update.go:35` | `"Update organization"` | `"Update an organization"` |
| `cmd/cloud/organizations/users/root.go:12` | `"Users management"` | `"Manage users"` |
| `cmd/cloud/organizations/users/show.go:38` | `"Show user by id"` | `"Show a user by ID"` |
| `cmd/cloud/organizations/users/unlink.go:35` | `"Unlink user from organization"` | `"Unlink a user from the organization"` |

### cloud / regions

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| 🐛 `cmd/cloud/regions/create.go:38` | `"Show region details"` | `"Create a region"` ← wrong description (this is the `create` command) |
| `cmd/cloud/regions/root.go:12` | `"Regions management"` | `"Manage regions"` |

### ledger

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/ledger/accounts/delete_metadata.go:34` | `"Delete metadata on account (Start from ledger v2 api)"` | `"Delete metadata from an account (requires ledger v2 or later)"` |
| `cmd/ledger/accounts/root.go:12` | `"Accounts management"` | `"Manage accounts"` |
| 🐛 `cmd/ledger/accounts/set_metadata.go:38` | `"Set metadata on address"` | `"Set metadata on an account"` ← says "address", should be "account" |
| `cmd/ledger/accounts/show.go:38` | `"Show account"` | `"Show an account"` |
| `cmd/ledger/create.go:48` | `"Create a new ledger (starting from ledger v2)"` | `"Create a new ledger (requires ledger v2 or later)"` |
| `cmd/ledger/delete_metadata.go:33` | `"Delete metadata on a ledger (Start from ledger v2 api)"` | `"Delete metadata from a ledger (requires ledger v2 or later)"` |
| `cmd/ledger/list.go:39` | `"List ledgers (starting from ledger v2)"` | `"List ledgers (requires ledger v2 or later)"` |
| `cmd/ledger/root.go:17` | `"Ledger management"` | `"Manage ledgers"` |
| `cmd/ledger/set_metadata.go:33` | `"Set metadata on a ledger (Start from ledger v2 api)"` | `"Set metadata on a ledger (requires ledger v2 or later)"` |
| `cmd/ledger/transactions/delete_metadata.go:34` | `"Delete metadata on transaction (Start from ledger v2 api)"` | `"Delete metadata from a transaction (requires ledger v2 or later)"` |
| `cmd/ledger/transactions/num.go:54` | `"Execute a numscript script on a ledger"` | `"Execute a Numscript program on a ledger"` |
| `cmd/ledger/transactions/root.go:12` | `"Transactions management"` | `"Manage transactions"` |
| `cmd/ledger/transactions/set_metadata.go:35` | `"Set metadata on transaction"` | `"Set metadata on a transaction"` |
| `cmd/ledger/transactions/show.go:34` | `"Print a transaction"` | `"Show a transaction"` |
| `cmd/ledger/volumes/list.go:166` | `"List volumes and balances for a period of time (OOT-PIT)"` | `"List volumes and balances for a time period (OOT–PIT)"` |
| `cmd/ledger/volumes/root.go:12` | `"Get volumes and Balances for accounts"` | `"Get volumes and balances for accounts"` |

### orchestration

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/orchestration/instances/list.go:50` | `"List all workflows instances"` | `"List all workflow instances"` |
| `cmd/orchestration/instances/root.go:12` | `"Instances management"` | `"Manage instances"` |
| `cmd/orchestration/triggers/list.go:39` | `"List all workflows triggers"` | `"List all workflow triggers"` |
| `cmd/orchestration/triggers/occurrences/list.go:38` | `"List all workflows occurrences"` | `"List all workflow occurrences"` |
| `cmd/orchestration/triggers/occurrences/root.go:12` | `"Triggers occurrences management"` | `"Manage trigger occurrences"` |
| `cmd/orchestration/triggers/root.go:13` | `"Triggers management"` | `"Manage triggers"` |
| `cmd/orchestration/workflows/root.go:12` | `"Workflows management"` | `"Manage workflows"` |

### payments

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/payments/accounts/balances.go:142` | `"List accounts balances"` | `"List account balances"` |
| `cmd/payments/accounts/create.go:45` | `"Create an account on formance platform"` | `"Create an account on the Formance platform"` |
| `cmd/payments/accounts/root.go:12` | `"Accounts management"` | `"Manage accounts"` |
| `cmd/payments/accounts/show.go:37` | `"Get account"` | `"Get an account"` |
| `cmd/payments/bankaccounts/root.go:12` | `"Bank Accounts management"` | `"Manage bank accounts"` |
| `cmd/payments/bankaccounts/show.go:45` | `"Get bank account"` | `"Get a bank account"` |
| `cmd/payments/bankaccounts/update_metadata.go:45` | `"Set metadata on bank account"` | `"Set metadata on a bank account"` |
| `cmd/payments/connectors/configs/adyen.go:53` | `"Update the config of a Adyen connector"` | `"Update the config of an Adyen connector"` |
| `cmd/payments/connectors/install/adyen.go:44` | `"Install an adyen connector"` | `"Install an Adyen connector"` |
| `cmd/payments/connectors/install/atlar.go:44` | `"Install an atlar connector"` | `"Install an Atlar connector"` |
| `cmd/payments/connectors/install/column.go:49` | `"Install a column connector"` | `"Install a Column connector"` |
| `cmd/payments/connectors/install/generic.go:41` | `"Install a Generic connector"` | `"Install a generic connector"` |
| `cmd/payments/connectors/install/qonto.go:49` | `"Install a qonto connector"` | `"Install a Qonto connector"` |
| `cmd/payments/connectors/install/stripe.go:43` | `"Install a stripe connector"` | `"Install a Stripe connector"` |
| `cmd/payments/connectors/root.go:14` | `"Connectors management"` | `"Manage connectors"` |
| `cmd/payments/payments/create.go:45` | `"Create a payment on formance platform"` | `"Create a payment on the Formance platform"` |
| `cmd/payments/payments/root.go:12` | `"Payments management"` | `"Manage payments"` |
| `cmd/payments/payments/set_metadata.go:37` | `"Set metadata on paymentID"` | `"Set metadata on a payment"` |
| `cmd/payments/payments/show.go:38` | `"Get payment"` | `"Get a payment"` |
| `cmd/payments/pools/add_accounts.go:46` | `"Add account to pool"` | `"Add an account to a pool"` |
| `cmd/payments/pools/remove_account.go:45` | `"Remove account from pool"` | `"Remove an account from a pool"` |
| `cmd/payments/pools/root.go:12` | `"Pools management"` | `"Manage pools"` |
| `cmd/payments/pools/show.go:45` | `"Get pool"` | `"Get a pool"` |
| `cmd/payments/root.go:18` | `"Payments management"` | `"Manage payments"` |
| `cmd/payments/tasks/root.go:12` | `"Tasks management"` | `"Manage tasks"` |
| `cmd/payments/tasks/show.go:44` | `"Get task"` | `"Get a task"` |
| `cmd/payments/transferinitiation/delete.go:46` | `"Delete a transfer Initiation"` | `"Delete a transfer initiation"` |
| `cmd/payments/transferinitiation/list.go:165` | `"List transfer initiation"` | `"List transfer initiations"` |
| `cmd/payments/transferinitiation/reverse.go:48` | `"Reverse a transfer Initiation"` | `"Reverse a transfer initiation"` |
| `cmd/payments/transferinitiation/root.go:12` | `"Transfer Initiation management"` | `"Manage transfer initiations"` |
| `cmd/payments/transferinitiation/show.go:45` | `"Get transfer initiation"` | `"Get a transfer initiation"` |

### profiles

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/profiles/reset.go:53` | `"Reset a profile keeping the environment"` | `"Reset a profile while keeping the environment"` |
| `cmd/profiles/root.go:12` | `"Profiles management"` | `"Manage profiles"` |
| `cmd/profiles/setdefaultorganization.go:77` | `"Set default organization"` | `"Set the default organization"` |
| `cmd/profiles/setdefaultstack.go:75` | `"Set default stack"` | `"Set the default stack"` |
| `cmd/profiles/show.go:74` | `"Show profile"` | `"Show a profile"` |
| `cmd/profiles/use.go:78` | `"Use profile"` | `"Switch to a profile"` |

### reconciliation

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/reconciliation/policies/root.go:12` | `"Policies management"` | `"Manage policies"` |
| `cmd/reconciliation/policies/show.go:38` | `"Get policy"` | `"Get a policy"` |
| `cmd/reconciliation/root.go:12` | `"Reconciliation management"` | `"Manage reconciliations"` |
| `cmd/reconciliation/show.go:38` | `"Get reconciliation"` | `"Get a reconciliation"` |

### search

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/search/root.go:182` | `"Search in all services (Default: ANY), or in a specific service (ACCOUNT, TRANSACTION, ASSET, PAYMENT)"` | `"Search across all services (default: ANY) or within a specific service (ACCOUNT, TRANSACTION, ASSET, PAYMENT)"` |

### stack

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/stack/modules/disable.go:32` | `"disable a module"` | `"Disable a module"` |
| `cmd/stack/show.go:49` | `"Show stack"` | `"Show a stack"` |
| `cmd/stack/update.go:44` | `"Update a created stack, name, or metadata"` | `"Update a stack's name or metadata"` |
| `cmd/stack/upgrade.go:42` | `"Upgrade a stack to specified version"` | `"Upgrade a stack to a specified version"` |
| `cmd/stack/users/link.go:39` | `"Link stack user with properties"` | `"Link a user to a stack with access properties"` |
| `cmd/stack/users/list.go:39` | `"List Stack Access Role within an organization by stacks"` | `"List stack access roles within an organization"` |
| `cmd/stack/users/root.go:12` | `"Stack users management within an organization"` | `"Manage stack user access within an organization"` |
| `cmd/stack/users/unlink.go:40` | `"Unlink stack user within an organization"` | `"Unlink a user from a stack within an organization"` |

### version

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/version/version.go:43` | `"Get version"` | `"Show the CLI version"` |

### wallets

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/wallets/balances/root.go:12` | `"Wallet balances"` | `"Manage wallet balances"` |
| `cmd/wallets/credit.go:48` | `"Credit a wallets"` | `"Credit a wallet"` |
| `cmd/wallets/holds/list.go:40` | `"List holds of a wallets"` | `"List holds for a wallet"` |
| `cmd/wallets/holds/root.go:12` | `"Wallets holds management"` | `"Manage wallet holds"` |
| `cmd/wallets/root.go:15` | `"Wallets management"` | `"Manage wallets"` |
| `cmd/wallets/show.go:39` | `"Show a wallets"` | `"Show a wallet"` |
| `cmd/wallets/transactions/root.go:12` | `"Wallet transactions"` | `"Manage wallet transactions"` |
| `cmd/wallets/update.go:40` | `"Update a wallets"` | `"Update a wallet"` |

### webhooks

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/webhooks/activate.go:70` | `"Activate one config"` | `"Activate a webhook config"` |
| `cmd/webhooks/create.go:89` | `"Create a new config. At least one event type is required."` | `"Create a new webhook config. At least one event type is required."` |
| `cmd/webhooks/deactivate.go:77` | `"Deactivate one config"` | `"Deactivate a webhook config"` |
| `cmd/webhooks/delete.go:89` | `"Delete a config"` | `"Delete a webhook config"` |
| `cmd/webhooks/list.go:91` | `"List all configs"` | `"List all webhook configs"` |
| `cmd/webhooks/root.go:12` | `"Webhooks management"` | `"Manage webhooks"` |
| `cmd/webhooks/secret.go:85` | `"Change the signing secret of a config. You can bring your own secret. If not passed or empty, a secret is automatically generated. The format is a string of bytes of size 24, base64 encoded. (larger size after encoding)"` | `"Change the signing secret of a webhook config. Optionally provide your own secret; if omitted, one is generated automatically (24-byte, base64-encoded string)."` |

---

## 2. Command Long Descriptions (`WithDescription`)

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/cloud/generate_personal_token.go:34` | `"Generate a personal bearer token"` | Same as short description — consider adding more detail or removing the long description. |

---

## 3. Flag Descriptions

### auth

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/auth/clients/create.go:63` | `"Is client public"` | `"Mark the client as public"` |
| `cmd/auth/clients/create.go:64` | `"Is the client trusted"` | `"Mark the client as trusted"` |
| `cmd/auth/clients/update.go:71` | `"Is client public"` | `"Mark the client as public"` |
| `cmd/auth/clients/update.go:72` | `"Is the client trusted"` | `"Mark the client as trusted"` |

### cloud / apps

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/cloud/apps/runs/logs.go:35` | `"run ID"` | `"Run ID"` |
| `cmd/cloud/apps/variables/create.go:43` | `"Is variable sensitive"` | `"Mark the variable as sensitive"` |
| `cmd/cloud/apps/variables/delete.go:35` | `"Variable id"` | `"Variable ID"` |

### cloud / organizations

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/cloud/organizations/authentication-provider/configure.go:41` | `"Used when type = oidc"` | `"OIDC issuer URL (used when type is 'oidc')"` |
| `cmd/cloud/organizations/authentication-provider/configure.go:42` | `"Used when type = microsoft"` | `"Microsoft tenant ID (used when type is 'microsoft')"` |
| `cmd/cloud/organizations/create.go:40` | `"Default policy id"` | `"Default policy ID"` |
| `cmd/cloud/organizations/create.go:41` | `"Organization Domain"` | `"Organization domain"` |
| `cmd/cloud/organizations/history.go:55` | `"Filter on Action"` | `"Filter by action"` |
| `cmd/cloud/organizations/history.go:56` | `"Filter on UserId, use SYSTEM to filter on system logs"` | `"Filter by user ID (use SYSTEM for system logs)"` |
| `cmd/cloud/organizations/history.go:57` | `"Filter on modified Data with --data key=value, key is a jsonb text path"` | `"Filter by modified data (format: key=value, key is a JSONB text path)"` |
| `cmd/cloud/organizations/history.go:61` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/cloud/organizations/update.go:39` | `"Organization Name"` | `"Organization name"` |
| `cmd/cloud/organizations/update.go:40` | `"Default policy id"` | `"Default policy ID"` |
| `cmd/cloud/organizations/update.go:41` | `"Organization Domain"` | `"Organization domain"` |

### ledger

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/ledger/create.go:49` | `"Bucket on which install the new ledger"` | `"Bucket in which to install the new ledger"` |
| `cmd/ledger/transactions/list.go:59` | `"Filter on account"` | `"Filter by account"` |
| `cmd/ledger/transactions/list.go:60` | `"Filter on destination account"` | `"Filter by destination account"` |
| `cmd/ledger/transactions/list.go:61` | `"Consider transactions before date"` | `"Include only transactions before this date"` |
| `cmd/ledger/transactions/list.go:62` | `"Consider transactions after date"` | `"Include only transactions after this date"` |
| `cmd/ledger/transactions/list.go:63` | `"Filter on source account"` | `"Filter by source account"` |
| `cmd/ledger/transactions/list.go:64` | `"Filter on reference"` | `"Filter by reference"` |
| `cmd/ledger/transactions/revert.go:38` | `"set the timestamp to the original transaction timestamp"` | `"Use the original transaction's timestamp"` |
| `cmd/ledger/volumes/list.go:171` | `"Group by level of segment of the address"` | `"Group by address segment level"` |
| `cmd/ledger/volumes/list.go:172` | `"Filter accounts with address"` | `"Filter accounts by address"` |
| `cmd/ledger/volumes/list.go:174` | `"Cursor pagination"` | `"Pagination cursor"` |

### login

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/login/login.go:103` | `"service url"` | `"Service URL"` |

### orchestration

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/orchestration/instances/list.go:54` | `"Filter on workflow id"` | `"Filter by workflow ID"` |
| `cmd/orchestration/instances/list.go:55` | `"Filter on running instances"` | `"Show only running instances"` |
| `cmd/orchestration/triggers/create.go:49` | `"Trigger's name"` | `"Trigger name"` |
| `cmd/orchestration/workflows/run.go:48` | `"Wait end of the run"` | `"Wait for the run to complete"` |

### payments

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/payments/accounts/balances.go:143` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/payments/accounts/balances.go:144` | `"PageSize"` | `"Page size"` |
| `cmd/payments/accounts/list.go:145` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/payments/accounts/list.go:146` | `"PageSize"` | `"Page size"` |
| `cmd/payments/bankaccounts/list.go:194` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/payments/bankaccounts/list.go:195` | `"PageSize"` | `"Page size"` |
| `cmd/payments/payments/list.go:149` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/payments/payments/list.go:150` | `"PageSize"` | `"Page size"` |
| `cmd/payments/pools/list.go:157` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/payments/pools/list.go:158` | `"PageSize"` | `"Page size"` |
| `cmd/payments/transferinitiation/list.go:166` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/payments/transferinitiation/list.go:167` | `"PageSize"` | `"Page size"` |

### reconciliation

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/reconciliation/list.go:144` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/reconciliation/list.go:145` | `"PageSize"` | `"Page size"` |
| `cmd/reconciliation/policies/list.go:152` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/reconciliation/policies/list.go:153` | `"PageSize"` | `"Page size"` |

### stack

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/stack/create.go:55` | `"Region on which deploy the stack"` | `"Region in which to deploy the stack"` |
| `cmd/stack/create.go:57` | `"Not wait stack availability"` | `"Do not wait for the stack to become available"` |
| `cmd/stack/delete.go:53` | `"Force to delete a stack without retention policy"` | `"Force deletion, bypassing the retention policy"` |
| `cmd/stack/history.go:55` | `"Filter on Action"` | `"Filter by action"` |
| `cmd/stack/history.go:56` | `"Filter on UserId, use SYSTEM to filter on system logs"` | `"Filter by user ID (use SYSTEM to filter by system logs)"` |
| `cmd/stack/history.go:57` | `"Filter on modified Data with --data key=value, key is a jsonb text path"` | `"Filter by modified data (format: key=value, key is a JSONB text path)"` |
| `cmd/stack/history.go:61` | `"Cursor"` | `"Pagination cursor"` |
| `cmd/stack/list.go:61` | `"Display deleted stacks"` | `"Display all stacks, including deleted ones"` ← duplicate of `--deleted` flag description |
| 🐛 `cmd/stack/restore.go:50` | `""` (empty) | `"Stack name to restore"` |
| 🐛 `cmd/stack/show.go:52` | `""` (empty) | `"Stack name to show"` |
| `cmd/stack/upgrade.go:43` | `"Wait stack availability"` | `"Wait for the stack to become available"` |
| `cmd/stack/users/link.go:38` | `"Policy id"` | `"Policy ID"` |

### wallets

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/wallets/credit.go:55` | `"Idempotency Key"` | `"Idempotency key"` |
| `cmd/wallets/debit.go:61` | `"Idempotency Key"` | `"Idempotency key"` |
| `cmd/wallets/holds/confirm.go:49` | `"Is final debit (close hold)"` | `"Mark as final debit (closes the hold)"` |
| `cmd/wallets/holds/confirm.go:50` | `"Idempotency Key"` | `"Idempotency key"` |
| `cmd/wallets/holds/void.go:43` | `"Idempotency Key"` | `"Idempotency key"` |
| `cmd/wallets/update.go:45` | `"Idempotency Key"` | `"Idempotency key"` |

### webhooks

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/webhooks/create.go:94` | `"Bring your own webhooks signing secret. If not passed or empty, a secret is automatically generated. The format is a string of bytes of size 24, base64 encoded. (larger size after encoding)"` | `"Custom signing secret (24-byte, base64-encoded string). If omitted, one is generated automatically."` |

---

## 4. Success / Error / Warning Messages (`pterm`)

### cloud / apps

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/cloud/apps/deploy.go:183` | `pterm.Info.Println("App Deployment accepted", c.store.ID)` | `"App deployment accepted with ID: %s"` (note: "Deployment" → lowercase) |

### cloud / me

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/cloud/me/invitations/decline.go:78` | `"Invitation declined! %s"` | `"Invitation %s declined."` |

### cloud / organizations

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| 🐛 `cmd/cloud/organizations/users/link.go:96` | `"User Addd."` | `"User added."` |
| `cmd/cloud/organizations/users/unlink.go:75` | `"User '%s' Deleted from organization '%s'"` | `"User '%s' removed from organization '%s'."` |

### cloud / regions

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/cloud/regions/create.go:102` | `"Your secret is (keep it safe, we will not be able to give it to you again): %s"` | `"Secret (shown once — store it safely): %s"` |

### login

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/login/login.go:97` | `"Logged!"` | `"Logged in successfully."` |

### orchestration

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/orchestration/instances/stop.go:72` | `"Workflow Instance with ID: %s successfully canceled "` | `"Workflow instance %s canceled."` (trailing space, unnecessary "with ID:") |
| `cmd/orchestration/triggers/delete.go:72` | `"Trigger %s Deleted!"` | `"Trigger %s deleted."` |
| `cmd/orchestration/workflows/delete.go:74` | `"Workflow %s Deleted!"` | `"Workflow %s deleted."` |

### payments

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/payments/bankaccounts/create.go:124` | `"Bank Account created with ID: %s"` | `"Bank account created with ID: %s"` |
| `cmd/payments/bankaccounts/forward.go:137` | `"Bank Account %s forwarded to connector %s"` | `"Bank account %s forwarded to connector %s."` |
| `cmd/payments/bankaccounts/forward.go:140` | `"Forwarding Bank Account scheduled with TaskID: %s"` | `"Bank account forwarding scheduled with task ID: %s"` |
| `cmd/payments/connectors/uninstall.go:178` | `"Connector uninstall scheduled with TaskID: %s"` | `"Connector uninstall scheduled with task ID: %s"` |
| `cmd/payments/pools/delete.go:103` | `"Pool %s Deleted!"` | `"Pool %s deleted."` |
| 🐛 `cmd/payments/pools/remove_account.go:96` | `"Successfully removed '%s' to '%s'"` | `"Successfully removed '%s' from '%s'."` |
| `cmd/payments/transferinitiation/approve.go:97` | `"Transfer Initiation scheduled with TaskID %q"` | `"Transfer initiation scheduled with task ID %q."` |
| `cmd/payments/transferinitiation/create.go:105` | `"Transfer Initiation created with ID: %s"` | `"Transfer initiation created with ID: %s"` |
| `cmd/payments/transferinitiation/delete.go:102` | `"Transfer Initiation %s Deleted!"` | `"Transfer initiation %s deleted."` |
| `cmd/payments/transferinitiation/reject.go:96` | `"Transfer Initiation %q was rejected"` | `"Transfer initiation %q was rejected."` |
| `cmd/payments/transferinitiation/retry.go:99` | `"Retry Transfer Initiation with ID: %s"` | `"Transfer initiation %s queued for retry."` |
| `cmd/payments/transferinitiation/reverse.go:115` | `"Transfer Initiation %s reversed!"` | `"Transfer initiation %s reversed."` |
| `cmd/payments/transferinitiation/update_status.go:106` | `"Update Transfer Initiation status with ID: %s and status %s"` | `"Transfer initiation %s status updated to %s."` |

### profiles

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/profiles/reset.go:46` | `"Profile reset on default !"` | `"Profile reset to defaults."` |

### reconciliation

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/reconciliation/policies/delete.go:88` | `"Policy %s Deleted!"` | `"Policy %s deleted."` |

### root (error messages)

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/root.go:111` | `"Your authentication is invalid, please login :)"` | `"Your authentication is invalid. Please log in."` |
| `cmd/root.go:152` | `"Got error with code %s: %s"` | `"Error %s: %s"` |

### stack

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/stack/controller.go:59` | `"You can check fctl stack show %s --organization %s to see the status of the stack"` | `"Check stack status with: fctl stack show %s --organization %s"` |
| `cmd/stack/modules/enable.go:78` | `"Module enabled"` | `"Module enabled."` (missing period, inconsistent with `"Module disabled."`) |

### wallets

| File:Line | Current | Suggested |
|-----------|---------|-----------|
| `cmd/wallets/debit.go:151` | `"Wallet debited successfully with hold id '%s'!"` | `"Wallet debited successfully with hold ID '%s'."` |
