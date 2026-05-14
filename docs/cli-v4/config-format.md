# fctl v4 Config Format

The v4 config should be explicit, versioned, and free of long-lived secrets.

## Example

```yaml
version: 4
currentContext: local

contexts:
  local:
    kind: stack
    stackURL: http://localhost/api
    auth:
      method: client_credentials
      issuerURL: http://localhost/api/auth
      clientID: testing
      secretRef: keyring://formance/local/testing
    defaults:
      ledger: default
    api:
      ledger: latest-compatible

  cloud-prod:
    kind: cloud-stack
    cloudURL: https://app.formance.cloud/api
    organization: org_x
    stack: stack_y
    auth:
      method: cloud_device
      account: user@example.com
      tokenRef: keyring://formance/cloud-prod/user@example.com
    api:
      ledger: latest-compatible
```

## Context Kinds

- `stack`: direct stack API target.
- `cloud`: Formance Cloud control plane target.
- `cloud-stack`: stack target discovered or authorized through Formance Cloud.

## Default Login Profile

When no v4 config exists yet, `fctl login` creates a default profile named
`default` unless `--profile` selects another name. Formance Cloud defaults to
the production membership URL:

```yaml
currentContext: default
contexts:
  default:
    kind: cloud
    cloudURL: https://app.formance.cloud/api
    auth:
      method: none
```

The login command then replaces `auth` with the selected method. Product stack
commands need either a direct `stack` profile or a Cloud/EE profile plus
`--organization` and `--stack` when the profile does not already store them.

## Auth Methods

- `cloud_device`
- `oidc_device`
- `client_credentials`
- `token`
- `none`

`none` must be explicit and should warn unless the command is non-interactive or configured to suppress warnings.

## Paths

Use XDG-aware locations:

- config: user config directory
- cache: discovery and temporary API tokens
- state: telemetry IDs and non-secret local state

Keep credentials in a system keyring when available. Until the v4 keyring
backend is wired, commands store secrets under the v4 config directory
`credentials/` subdirectory by default, or under `--credential-dir` when
provided. Secret values must not be written to `config.yaml`.
