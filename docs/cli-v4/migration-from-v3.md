# Migration From fctl v3

The v4 CLI should import v3 configuration without deleting or rewriting it in place.

## Mapping

Current v3 profile fields:

- `membershipURI`
- `rootTokens`
- `defaultOrganization`
- `defaultStack`

Suggested v4 mapping:

- `membershipURI` -> `cloudURL`
- `defaultOrganization` -> context `organization`
- `defaultStack` -> context `stack`
- `rootTokens` -> keyring credential, referenced by `tokenRef`

## Command

Provide an explicit migration command:

```bash
fctl config migrate-v3
```

By default the command reads the v3 configuration from
`$HOME/.config/formance/fctl`. Use `--from <dir>` only when the v3 config lives
elsewhere. The source directory must contain the v3 `config.yml` file and the
`profiles/` directory.

The command writes the v4 config to the platform user config directory as
`config.yaml` (`$HOME/Library/Application Support/formance/fctl-v4/config.yaml`
on macOS, or `--config-dir <dir>/config.yaml` when provided). It does not create
or mutate the v3 `config.yml` source file.

The command should:

1. Read v3 config and profiles.
2. Show the contexts that will be created.
3. Move secrets to keyring when possible.
4. Write v4 config.
5. Leave v3 files untouched.

## Compatibility

During early v4 releases, support a read-only fallback that can detect v3 profiles and suggest migration. Do not silently mutate v3 profile files during normal command execution.
