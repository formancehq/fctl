# ADR 0005: Build v4 In An Isolated Directory

Status: Accepted for v4 planning

## Context

`fctl` v4 is intended to be a near-rewrite. The existing v3 code remains useful as a behavioral reference during implementation and review.

## Decision

Build the new CLI under a top-level `v4/` directory during the transition. Keep the existing root implementation intact until v4 has reached feature parity or an explicit cutover point.

## Consequences

- v3 remains available for comparison while v4 is built.
- Review is easier because new code is isolated from old code.
- The v4 module can start with a clean package layout.
- The final cutover will delete or archive the old root implementation and move the v4 implementation to the root.
- Build, release, and test commands must be explicit about whether they target v3 root or `v4/`.
