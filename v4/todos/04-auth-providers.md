# Goal 04 - v4 auth providers

```text
/goal
Implement initial v4 authentication providers for stack targets.

Read first:
- docs/adr/0002-auth-is-decoupled-from-cloud.md
- docs/cli-v4/config-format.md
- v4/internal/credentials from prior goals.

Deliverables:
- auth provider interface in v4/internal/runtime or v4/internal/auth if the split is warranted.
- client_credentials provider using issuer URL, client ID, and secret reference.
- token provider using token from env, stdin, or credential reference.
- explicit none provider for local development targets.
- credential storage through the credentials interface.
- clear errors for missing credentials.

Constraints:
- do not implement Cloud device flow yet unless it is needed for tests.
- do not store secrets directly in config.
- none auth must be explicit.
- keep CI/non-interactive behavior deterministic.
- commit each provider separately when practical.
- run git diff --check before each commit.

Tests:
- unit tests for provider selection.
- unit tests with fake HTTP token endpoint for client_credentials.
- no tests should require real Formance Cloud.

Done when:
- stack contexts can resolve an HTTP client/token source for client_credentials, token, and none.
```
