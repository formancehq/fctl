# ADR 0002: Authentication Is Decoupled From Cloud

Status: Accepted for v4 planning

## Context

The current CLI authenticates through a membership relying party. This prevents a clean local and self-hosted user experience.

## Decision

Model authentication as a target-local strategy. Supported strategies should include Cloud device flow, generic OIDC, client credentials, token from stdin/env, and explicit no-auth development mode.

## Consequences

- Local stacks can use `client_credentials` with default development clients.
- Self-hosted stacks can use their own OIDC issuer.
- CI can use tokens or client credentials without browser flows.
- Cloud membership becomes one auth strategy, not the root abstraction.
