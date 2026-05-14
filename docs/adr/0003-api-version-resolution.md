# ADR 0003: API Version Resolution Belongs In Runtime

Status: Accepted for v4 planning

## Context

The public SDK exposes versioned namespaces such as `Ledger.V1`, `Ledger.V2`, and `Payments.V3`. The stack exposes `/versions`, but not a full capabilities endpoint.

## Decision

Commands declare product features and available handlers. The runtime calls `/versions`, maps component versions to supported API namespaces, intersects that with command handlers, and selects the best compatible handler.

## Consequences

- Commands do not hardcode the oldest API namespace.
- New endpoints can appear as normal product commands and fail cleanly on older targets.
- A small manual compatibility table is still required for component version ranges.
- Most operation metadata can be generated from the OpenAPI spec.
