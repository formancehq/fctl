# ADR 0001: Contexts Are The Primary Target Selector

Status: Accepted for v4 planning

## Context

The current profile model is centered on Formance Cloud membership. That makes stack usage depend on Cloud identity even when the user wants to talk to a local or self-hosted stack.

## Decision

Use named contexts as the primary target selector. A context describes the target endpoint, authentication method, defaults, and API version policy.

## Consequences

- Stack commands can run without Cloud membership.
- Cloud workflows remain possible through Cloud-specific context kinds.
- `--context` and `FCTL_CONTEXT` can override the current context.
- Context export/import becomes possible later.
