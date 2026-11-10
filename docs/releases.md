# Homebrew release publication

`just release` runs GoReleaser, then `just release-homebrew-pr`. GoReleaser
uploads `fctl.rb` and `fctl@<major>.rb` to the same `fctl-<version>` branch in
`formancehq/homebrew-tap`. Its per-formula PR creation is disabled so it does
not try to open the same PR twice or expose a partial update for review.

The PR step reads `dist/metadata.json` and `dist/homebrew/Formula/`. It verifies
that both uploaded Git blobs match the generated formulas before opening one
PR into `main`. This check is required because a Homebrew publisher warning
does not necessarily fail GoReleaser. GitHub errors and incomplete uploads
fail the PR step.

The step reuses an open PR for the same branch. If another invocation creates
the PR concurrently, it reads and reuses that PR. If both formulas already
match `main`, including after a squash merge, it exits without opening a PR.
It does not approve or merge PRs, or change repository review policy.

To retry only the PR step after a failure, keep the generated `dist/` directory
from the release and run:

```sh
nix develop --impure --command just release-homebrew-pr
```

This command requires the release identity's `GITHUB_TOKEN` with write access
to `formancehq/homebrew-tap`. It does not rebuild or republish release assets.
If a formula upload failed, restore both generated formulas on the release
branch before retrying. `release-local` and `release-ci` do not invoke the PR
step; local builds remain free of GitHub writes.

Run the publication regression tests with:

```sh
nix develop --impure --command just tests-release
```

These tests use a simulated GitHub API. They cover two formulas, retries,
partial uploads, squash merges, competing PR creation, and API failures.
