# fctl v4 Versioning And Ownership

This document records the current v4 position after comparing the isolated v4
architecture with the plugin architecture proposal. It focuses on where API
version complexity lives, who owns CLI evolution, and what must be explicit
before cutover.

## Product Surface

`fctl` is not intended to be a full OpenAPI explorer. Commands should expose
stable Formance product intent.

That means:

- a business feature may map to one endpoint, several endpoints, or no direct
  one-to-one OpenAPI operation;
- not every stack endpoint must become a visible CLI command;
- generated SDK namespaces remain implementation details;
- visible command names should stay stable when API paths or generated SDK names
  change;
- hidden commands are acceptable when code exists but the product contract is not
  ready.

This is why `cloud apps ...` and `ledger transactions explain` can exist in code
without being part of the visible v4 command surface.

## Current MVP Choice

The current v4 implementation keeps product commands in the main `fctl` binary.
This is an MVP and operational choice, not a claim that the monolithic approach
is always structurally better than plugins.

The main benefits are:

- zero install friction for users;
- one binary to distribute, debug, sign, and support;
- one place for profile, auth, target resolution, rendering, and command tests;
- easier migration from v3 while the team is still stabilizing ownership.

The main costs are:

- handlers, mappers, and tests accumulate as API versions accumulate;
- static Cobra help can show flags that only some target versions support;
- intra-version capability differences need explicit modeling;
- the main binary can grow if support windows are unbounded.

## Why Plugins Are Not The MVP

Plugins remain a valid future architecture, but they move complexity from code
to operations.

They would require clear answers for:

- plugin packaging and signing;
- plugin registry and discovery;
- automatic installation for Cloud stacks;
- self-hosted and local installation flows;
- per-product ownership and release duties;
- version retirement policy;
- support and debugging when the core CLI and plugin versions drift;
- CI expectations for every product team.

That operational overhead is too high for the current MVP. The team already has
maintenance pressure on shared tools; multiplying binaries and release surfaces
would make ownership harder before the v4 model is validated.

The plugin architecture should be revisited when:

- the core profile/auth/rendering model is stable;
- product teams have explicit CLI ownership expectations;
- plugin packaging and discovery can be made nearly invisible for Cloud users;
- self-hosted/local users have a documented setup flow;
- the monolithic handler surface starts creating measurable maintenance or binary
  size problems.

## Capabilities Gaps

The current capabilities system is intentionally useful but incomplete.

The model is intentionally close to the Console/frontend approach: discover the
running component versions, select compatible generated SDK code, and keep the
user-facing surface stable while stack releases move forward.

It handles coarse API namespace selection:

1. read `/versions`;
2. map component semver ranges to supported API namespaces;
3. intersect target API namespaces with command handler API namespaces;
4. select the highest compatible namespace unless pinned.

Known gaps:

- the generated manifest contains per-feature operation metadata, but command
  execution does not currently query it to prove that a selected API namespace
  exposes the requested feature;
- `Feature` is passed through version resolution mostly for error context;
- capabilities do not model feature availability inside the same API namespace,
  such as a query parameter or response field added in a minor component
  version;
- version-dependent flags are static Cobra flags, so `--help` can show a flag
  that the selected target stack does not support.

Required improvements:

- use the generated manifest at runtime to verify feature availability for the
  selected API namespace;
- add feature-level compatibility ranges for intra-namespace differences;
- centralize version-dependent flag validation instead of scattering ad hoc
  `if` checks inside handlers;
- annotate version-dependent flags in help text as a low-cost short-term fix,
  for example `[ledger v2+]`;
- decide later whether target-aware dynamic help is worth the network call to
  `/versions`.

## SDK And Release Cadence

The v4 CLI may temporarily need SDK code that is ahead of the last public stack
release, especially while a product feature, generated SDK, stack release, and
CLI command are being developed in parallel.

That is acceptable only when the drift is short-lived and explicit:

- prefer generated SDK updates over hand-written request code;
- avoid pinning long-lived commit hashes without a cleanup plan;
- document when a custom or ahead-of-release SDK is required;
- remove temporary SDK workarounds once the stack release and public SDK catch
  up;
- keep capabilities data aligned with the stack versions users can actually run.

Faster SDK generation and release cadence should reduce the time between product
feature implementation and CLI availability, but it does not remove the need for
capability checks.

## Support Window

v4 must not imply unbounded support for every historical stack version forever.

Before cutover, define a support policy such as:

- latest stack major plus one previous supported major;
- product-specific exceptions documented in compatibility data;
- deprecated API handlers removed only at explicit release boundaries;
- command tests covering every supported API namespace in the policy.

Every new product API version should include a CLI impact note:

- new visible command;
- new flag on an existing command;
- changed request or response mapping;
- required compatibility range update;
- tests to add or remove;
- documentation to update.

## Ownership Rule

The initial v4 migration can be bootstrapped centrally, but future product
changes must include CLI work when they affect user-facing operations.

For a product PR that changes an API contract, the expected checklist is:

- update or confirm the OpenAPI spec;
- update generated SDK or custom SDK references as needed;
- update compatibility data;
- update the relevant `fctl` command or explicitly document why no CLI surface is
  exposed;
- update tests for command behavior and version compatibility;
- update `docs/cli-v4/command-reference.md` and related docs.

The CLI should not become a separate backlog owned only by a few maintainers.
Product teams own the CLI surface for their product in the same way they own API
and documentation changes.

## RFC And Review Governance

Architecture changes need a clear review window.

Recommended process:

- publish the RFC or PR with a concrete review deadline;
- ask required reviewers to acknowledge with a comment or approval marker;
- unresolved objections must be written on the PR/RFC, not only discussed in
  Slack;
- if no blocking comments are added before the deadline, the proposal can be
  considered accepted for the stated scope;
- large competing approaches, such as plugins versus monolithic handlers, should
  be recorded as ADR/RFC tradeoffs instead of staying as informal PR comments.

This keeps disagreement visible and prevents important architecture decisions
from depending on private or ephemeral discussion.
