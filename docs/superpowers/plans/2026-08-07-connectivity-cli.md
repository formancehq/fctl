# Fctl Connectivity CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a stack-authenticated, domain-oriented `fctl connectivity` command tree for discovering plugins and managing Connectivity instances, including schema-aware configuration and dynamic shell completion.

**Architecture:** A handwritten `internal/connectivityclient` implements the seven Connectivity API operations used by the CLI. Resource-specific Cobra controllers receive a mockable client factory and reuse `fctl` authentication, rendering, pagination, confirmation, and JSON output.

**Tech Stack:** Go 1.25, Cobra, pterm, testify, YAML v3, `net/http`, Nix, Just.

## Global Constraints

- Use `formancehq/connectivity@develop` commit `2ec12564c02c040b3f9da5fbdf40c880e67c1137`, API `0.1.0`, as the contract.
- Keep `fctl payments` unchanged and use `fctl connectivity <resource> <action>`.
- Call `<selected-stack-uri>/api/connectivity` through existing stack authentication.
- Add no Connectivity profile, token, context, API URL flag, generated client, or new dependency.
- Never print inline configuration values in plain output; JSON retains the API model.
- Dynamic completion has a two-second timeout, fails silently, and never prompts.
- Unit tests use mocks; HTTP integration test names use Given/When/Then.
- New handwritten packages maintain at least 80% statement coverage.
- Run all tooling through Nix and Just, including tests and precommit before each commit.

Approved design: `docs/superpowers/specs/2026-08-07-connectivity-cli-design.md`.

## File map

- `internal/connectivityclient/{models,client}.go`: API contract and HTTP adapter.
- `cmd/connectivity/internal/client.go`: authenticated client factory.
- `cmd/connectivity/plugins`: list/show/render/complete plugins.
- `cmd/connectivity/instances/config.go`: config sources, schema, merge, validation.
- `cmd/connectivity/instances`: list/show/install/configure/uninstall and completion.
- `cmd/connectivity/root.go`, `cmd/root.go`: public command wiring.
- `completions/fctl.{bash,zsh,fish}`: regenerated shell scripts.

---

### Task 1: Connectivity API client

**Files:**
- Create: `internal/connectivityclient/models.go`
- Create: `internal/connectivityclient/client.go`
- Test: `internal/connectivityclient/client_test.go`
- Test: `internal/connectivityclient/integration_test.go`

**Interfaces:**
- Consumes: stack URI and authenticated `*http.Client`.
- Produces: models, `ListOptions`, `InstancePatch`, `APIError`, `Client`, and `New`.

- [ ] **Step 1: Write failing HTTP contract tests**

Use a mocked `RoundTripper` to assert method, escaped path, query, headers, body, response decoding, empty success, and structured errors. Include this representative test and table cases for all seven methods:

```go
func TestListPluginsBuildsStackConnectivityRequest(t *testing.T) {
	var seen *http.Request
	httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		seen = req.Clone(req.Context())
		return jsonResponse(200, `{"items":[{"metadata":{"name":"stripe"},"spec":{"image":"img"}}],"continue":"next"}`), nil
	})}
	got, err := New("https://stack.example/base", httpClient).ListPlugins(context.Background(), ListOptions{Limit: 25, Continue: "cursor"})
	require.NoError(t, err)
	require.Equal(t, "/base/api/connectivity/plugins", seen.URL.Path)
	require.Equal(t, "25", seen.URL.Query().Get("limit"))
	require.Equal(t, "cursor", seen.URL.Query().Get("continue"))
	require.Equal(t, "stripe", *got.Items[0].Metadata.Name)
	require.Equal(t, "next", got.Continue)
}
```

Define these test-only helpers in `client_test.go`:

```go
type roundTripperFunc func(*http.Request) (*http.Response, error)
func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }
func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
```

Add `TestGivenConnectivityServer_WhenLifecycleMethodsRun_ThenContractIsRespected` with `httptest.Server` and Given/When/Then comment sections.

- [ ] **Step 2: Verify RED**

Run `nix develop -c go test ./internal/connectivityclient -v`.

Expected: compilation fails because client symbols do not exist.

- [ ] **Step 3: Add exact public models and interface**

```go
type ObjectMeta struct { Name *string `json:"name,omitempty"`; Namespace *string `json:"namespace,omitempty"`; ResourceVersion *string `json:"resourceVersion,omitempty"`; UID *string `json:"uid,omitempty"`; CreationTimestamp *time.Time `json:"creationTimestamp,omitempty"`; Labels map[string]string `json:"labels,omitempty"`; Annotations map[string]string `json:"annotations,omitempty"` }
type KeyRef struct { Name string `json:"name"`; Key string `json:"key"` }
type EnvValue struct { Value *string `json:"value,omitempty"`; SecretRef *KeyRef `json:"secretRef,omitempty"`; ConfigMapRef *KeyRef `json:"configMapRef,omitempty"` }
type FileMount struct { Path string `json:"path"`; Value *string `json:"value,omitempty"`; SecretRef *KeyRef `json:"secretRef,omitempty"`; ConfigMapRef *KeyRef `json:"configMapRef,omitempty"`; Mode *int32 `json:"mode,omitempty"` }
type InstanceConfig struct { Env map[string]EnvValue `json:"env,omitempty"`; Files []FileMount `json:"files,omitempty"` }
type VersionEntry struct { Version string `json:"version"`; Digest *string `json:"digest,omitempty"`; Image *string `json:"image,omitempty"` }
type PluginSpec struct { Image string `json:"image"`; Version *string `json:"version,omitempty"`; Description *string `json:"description,omitempty"`; DocsURL *string `json:"docsURL,omitempty"`; Capabilities []string `json:"capabilities,omitempty"`; ConfigSchema map[string]any `json:"configSchema,omitempty"`; DefaultVersion *string `json:"defaultVersion,omitempty"`; Versions []VersionEntry `json:"versions,omitempty"`; Defaults *InstanceConfig `json:"defaults,omitempty"` }
type PluginStatus struct { Phase *string `json:"phase,omitempty"`; Message *string `json:"message,omitempty"` }
type Plugin struct { Metadata ObjectMeta `json:"metadata"`; Spec PluginSpec `json:"spec"`; Status *PluginStatus `json:"status,omitempty"` }
type PluginList struct { Items []Plugin `json:"items"`; Continue string `json:"continue,omitempty"` }
type InstanceSpec struct { Plugin string `json:"plugin"`; Version *string `json:"version,omitempty"`; ConnectivityRef *string `json:"connectivityRef,omitempty"`; Ledger string `json:"ledger"`; StartSequence *int64 `json:"startSequence,omitempty"`; PollInterval *string `json:"pollInterval,omitempty"`; Config *InstanceConfig `json:"config,omitempty"` }
type InstanceStatus struct { Phase *string `json:"phase,omitempty"`; State *string `json:"state,omitempty"`; PluginAddress *string `json:"pluginAddress,omitempty"`; ResolvedImage *string `json:"resolvedImage,omitempty"`; CurrentSequence *int64 `json:"currentSequence,omitempty"`; SourceTipSequence *int64 `json:"sourceTipSequence,omitempty"`; LastError *string `json:"lastError,omitempty"`; Message *string `json:"message,omitempty"` }
type Instance struct { Metadata ObjectMeta `json:"metadata"`; Spec InstanceSpec `json:"spec"`; Status *InstanceStatus `json:"status,omitempty"` }
type InstanceList struct { Items []Instance `json:"items"`; Continue string `json:"continue,omitempty"` }
type InstanceCreate struct { Name string `json:"name"`; Labels map[string]string `json:"labels,omitempty"`; Annotations map[string]string `json:"annotations,omitempty"`; Spec InstanceSpec `json:"spec"` }
type InstancePatch map[string]any
type ListOptions struct { Limit int32; Continue string; Plugin string }
type APIError struct { StatusCode int; Code string; Message string; Details map[string]any }

type Client interface {
	ListPlugins(context.Context, ListOptions) (*PluginList, error)
	GetPlugin(context.Context, string) (*Plugin, error)
	ListInstances(context.Context, ListOptions) (*InstanceList, error)
	CreateInstance(context.Context, InstanceCreate) (*Instance, error)
	GetInstance(context.Context, string) (*Instance, error)
	PatchInstance(context.Context, string, InstancePatch) (*Instance, error)
	DeleteInstance(context.Context, string) error
}
func New(stackURI string, httpClient *http.Client) Client
```

- [ ] **Step 4: Implement minimal HTTP wrappers**

Use one JSON request helper. Join `/api/connectivity`, escape names with `url.PathEscape`, use `url.Values`, set `Accept`, set POST content type to `application/json`, PATCH to `application/merge-patch+json`, parse `{code,message,details}` into `APIError`, and reject malformed or empty object responses. Accept 200 for reads and patch, 201 for create, and 204 for delete.

- [ ] **Step 5: Verify GREEN, coverage, repository gates, and commit**

```text
nix develop -c go test ./internal/connectivityclient -coverprofile=/tmp/connectivity-client.cover -v
nix develop -c go tool cover -func=/tmp/connectivity-client.cover
nix develop -c just tests
nix develop -c just pre-commit
git add internal/connectivityclient
git commit -m "feat(connectivity): add API client"
```

Expected: tests pass and package coverage is at least 80%.

---

### Task 2: Schema-aware configuration

**Files:**
- Create: `cmd/connectivity/instances/config.go`
- Test: `cmd/connectivity/instances/test_helpers_test.go`
- Test: `cmd/connectivity/instances/config_test.go`

**Interfaces:**
- Consumes: plugin schema/defaults, current instance config, Cobra stdin, and injected file reader.
- Produces: `SchemaFields`, `BuildInstallConfig`, and `BuildConfigureConfig`.

- [ ] **Step 1: Write failing parser and merge tests**

Use a map-backed `ReadFileFunc`, never the real filesystem. Test flat/structured YAML, dotenv, later-source precedence, later `--set`, inline values, `@file`, `@-`, secret/configmap refs, env/file routing, missing required keys, malformed refs, unknown keys, and configure preserving untouched files.

```go
func TestBuildConfigureConfigPreservesUntouchedFiles(t *testing.T) {
	current := &connectivityclient.InstanceConfig{Files: []connectivityclient.FileMount{
		{Path: "/etc/a", Value: stringPtr("a")},
		{Path: "/etc/b", Value: stringPtr("b")},
	}}
	got, err := BuildConfigureConfig(&cobra.Command{}, pluginWithFileSchema(), current,
		InputOptions{SetValues: []string{"/etc/a=@new"}},
		func(cmd *cobra.Command, name string) (string, error) { return "changed", nil })
	require.NoError(t, err)
	require.Len(t, got.Files, 2)
	require.Equal(t, "changed", *got.Files[0].Value)
	require.Equal(t, "b", *got.Files[1].Value)
}
```

Define `stringPtr`, `pluginWithFileSchema`, and `pluginWithSchemaAndDefaults` in `test_helpers_test.go`. The first returns `&value`; the plugin fixtures return API models with explicit env/file properties, required arrays, password formats, defaults, and two version entries.

- [ ] **Step 2: Verify RED**

Run `nix develop -c go test ./cmd/connectivity/instances -run 'Test(Build|Parse|Schema)' -v`.

Expected: compilation fails because configuration symbols do not exist.

- [ ] **Step 3: Implement exact configuration contracts**

```go
type ConfigKind string
const ( ConfigEnv ConfigKind = "env"; ConfigFile ConfigKind = "file" )
type SchemaField struct { Key string; Kind ConfigKind; Required bool; Password bool; Description string }
type InputOptions struct { ConfigFile string; EnvFiles []string; SetValues []string }
type ReadFileFunc func(cmd *cobra.Command, path string) (string, error)
func SchemaFields(plugin *connectivityclient.Plugin) (map[string]SchemaField, error)
func BuildInstallConfig(cmd *cobra.Command, plugin *connectivityclient.Plugin, inputs InputOptions, read ReadFileFunc) (*connectivityclient.InstanceConfig, error)
func BuildConfigureConfig(cmd *cobra.Command, plugin *connectivityclient.Plugin, current *connectivityclient.InstanceConfig, inputs InputOptions, read ReadFileFunc) (*connectivityclient.InstanceConfig, error)
```

Install merges defaults → config → env files in order → set values in order. Configure replaces defaults with a deep copy of current config. Replace file mounts by path without losing untouched mounts. Required-key and unknown-key errors sort keys for deterministic output.

- [ ] **Step 4: Verify GREEN, coverage, gates, and commit**

```text
nix develop -c go test ./cmd/connectivity/instances -run 'Test(Build|Parse|Schema)' -coverprofile=/tmp/connectivity-config.cover -v
nix develop -c go tool cover -func=/tmp/connectivity-config.cover
nix develop -c just tests
nix develop -c just pre-commit
git add cmd/connectivity/instances/config.go cmd/connectivity/instances/config_test.go docs/superpowers/specs/2026-08-07-connectivity-cli-design.md
git commit -m "feat(connectivity): assemble instance configuration"
```

---

### Task 3: Plugin catalog commands

**Files:**
- Create: `cmd/connectivity/internal/client.go`
- Create: `cmd/connectivity/plugins/{root,list,show,completion}.go`
- Test: `cmd/connectivity/plugins/test_helpers_test.go`
- Test: `cmd/connectivity/plugins/{list,show,completion}_test.go`

**Interfaces:**
- Consumes: client API, stack clients, pagination, renderer.
- Produces: `ClientFactory`, plugin root, list/show, `CompletePluginNames`.

- [ ] **Step 1: Write failing mocked command tests**

Embed `connectivityclient.Client` in mocks and override only called methods. Assert pagination, continuation rendering, JSON store, versions/capabilities/schema detail rendering, prefix filtering, completion descriptions, two-second deadline, and silent errors.

```go
type pluginClientMock struct { connectivityclient.Client; list func(context.Context, connectivityclient.ListOptions) (*connectivityclient.PluginList, error); get func(context.Context, string) (*connectivityclient.Plugin, error) }
func (m pluginClientMock) ListPlugins(ctx context.Context, opts connectivityclient.ListOptions) (*connectivityclient.PluginList, error) { return m.list(ctx, opts) }
func (m pluginClientMock) GetPlugin(ctx context.Context, name string) (*connectivityclient.Plugin, error) { return m.get(ctx, name) }
func pluginFixture(name string) connectivityclient.Plugin { return connectivityclient.Plugin{Metadata: connectivityclient.ObjectMeta{Name: fctl.Ptr(name)}, Spec: connectivityclient.PluginSpec{Image: "registry/plugin", Description: fctl.Ptr("Plugin description")}} }
```

- [ ] **Step 2: Verify RED**

Run `nix develop -c go test ./cmd/connectivity/plugins -v`.

Expected: compilation fails because plugin packages do not exist.

- [ ] **Step 3: Implement authenticated factory and commands**

```go
type ClientFactory func(*cobra.Command) (connectivityclient.Client, error)
func NewClientFactory() ClientFactory
func NewCommand(factory connectivityinternal.ClientFactory) *cobra.Command
func NewListCommand(factory connectivityinternal.ClientFactory) *cobra.Command
func NewShowCommand(factory connectivityinternal.ClientFactory) *cobra.Command
func CompletePluginNames(factory connectivityinternal.ClientFactory) cobra.CompletionFunc
```

The factory calls `LoadAndAuthenticateCurrentProfile` and `NewStackClientsFromFlags`, then constructs the API client. Use aliases `plugins: plugin,p`, `list: ls,l`, `show: get,g,sh,s`. `ListStore` carries plugins and `fctl.Cursor`; `ShowStore` carries the complete plugin. Render the approved columns and schema summary.

- [ ] **Step 4: Verify GREEN, coverage, gates, and commit**

```text
nix develop -c go test ./cmd/connectivity/plugins -coverprofile=/tmp/connectivity-plugins.cover -v
nix develop -c go tool cover -func=/tmp/connectivity-plugins.cover
nix develop -c just tests
nix develop -c just pre-commit
git add cmd/connectivity/internal cmd/connectivity/plugins
git commit -m "feat(connectivity): add plugin catalog commands"
```

---

### Task 4: Instance read commands and completion

**Files:**
- Create: `cmd/connectivity/instances/{root,list,show,completion}.go`
- Modify: `cmd/connectivity/instances/test_helpers_test.go`
- Test: `cmd/connectivity/instances/{list,show,completion}_test.go`

**Interfaces:**
- Consumes: shared client factory, API models, cursor flags, schema fields.
- Produces: instance root, list/show, instance/version/set/path completion.

- [ ] **Step 1: Write failing list/show tests**

Mock `ListInstances` and `GetInstance`. Assert `--plugin`, cursor/page size, approved columns, progression fields, JSON store, and that plain output names config keys/source types but never inline values.

```go
func TestShowPlainOutputNeverPrintsInlineConfigValues(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.Spec.Config = &connectivityclient.InstanceConfig{Env: map[string]connectivityclient.EnvValue{
		"API_KEY": {Value: stringPtr("must-not-leak")},
	}}
	cmd := NewShowCommand(factoryWithInstance(instance))
	cmd.SetArgs([]string{"stripe-eu"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "API_KEY")
	require.Contains(t, out.String(), "inline")
	require.NotContains(t, out.String(), "must-not-leak")
}
```

- [ ] **Step 2: Write failing completion tests**

Assert instance-name and version candidates, required schema keys before optional keys, descriptions after a tab, already supplied keys omitted, `KEY=@` candidates preserving the prefix, `NoSpace` for `KEY=`, and silence on timeout/error. Inject a mock path completer rather than reading disk.

Define reusable test helpers with these signatures in `test_helpers_test.go`:

```go
func stringPtr(value string) *string
func instanceFixture(name string) connectivityclient.Instance
func instanceWithTwoFiles() *connectivityclient.Instance
func pluginWithFileSchema() *connectivityclient.Plugin
func pluginWithSchemaAndDefaults() *connectivityclient.Plugin
func factoryReturning(client connectivityclient.Client) connectivityinternal.ClientFactory
func factoryWithInstance(instance connectivityclient.Instance) connectivityinternal.ClientFactory
func mockReadFile(files map[string]string) ReadFileFunc
func mockPathCompleter(paths []string) PathCompleter
```

The package-specific mock structs embed `connectivityclient.Client` and implement only the methods exercised by their test file; every function-valued field is invoked directly by its matching method.

- [ ] **Step 3: Verify RED**

Run `nix develop -c go test ./cmd/connectivity/instances -run 'Test(ListInstances|Show|Complete)' -v`.

Expected: compilation fails because read commands and completion functions do not exist.

- [ ] **Step 4: Implement read commands and renderers**

```go
func NewCommand(factory connectivityinternal.ClientFactory, read ReadFileFunc, paths PathCompleter) *cobra.Command
func NewListCommand(factory connectivityinternal.ClientFactory) *cobra.Command
func NewShowCommand(factory connectivityinternal.ClientFactory) *cobra.Command
```

Use aliases `instances: instance,i`, `list: ls,l`, and `show: get,g,sh,s`. `ListStore` contains instances and `fctl.Cursor`; `ShowStore` contains the complete instance. Render config sources as `inline`, `secret:name/key`, or `configmap:name/key` without rendering `Value` fields.

- [ ] **Step 5: Implement exact completion API**

```go
type PathCompleter func(prefix string) ([]string, error)
func CompleteInstanceNames(factory connectivityinternal.ClientFactory) cobra.CompletionFunc
func CompleteVersions(factory connectivityinternal.ClientFactory, pluginArg func(*cobra.Command, []string) string) cobra.CompletionFunc
func CompleteSetValues(factory connectivityinternal.ClientFactory, resolvePlugin func(context.Context, connectivityclient.Client, *cobra.Command, []string) (*connectivityclient.Plugin, error), paths PathCompleter) cobra.CompletionFunc
func OSPathCompleter(prefix string) ([]string, error)
```

Use `context.WithTimeout(cmd.Context(), 2*time.Second)`, list limit 500, prefix filtering, sorted candidates, and silent `NoFileComp` errors. Preserve `KEY=@` when completing paths.

- [ ] **Step 6: Verify GREEN, coverage, gates, and commit**

```text
nix develop -c go test ./cmd/connectivity/instances -run 'Test(ListInstances|Show|Complete)' -coverprofile=/tmp/connectivity-read.cover -v
nix develop -c go tool cover -func=/tmp/connectivity-read.cover
nix develop -c just tests
nix develop -c just pre-commit
git add cmd/connectivity/instances
git commit -m "feat(connectivity): list and show instances"
```

---

### Task 5: Install instances

**Files:**
- Create: `cmd/connectivity/instances/install.go`
- Test: `cmd/connectivity/instances/install_test.go`
- Modify: `cmd/connectivity/instances/root.go`
- Modify: `cmd/connectivity/instances/completion.go`

**Interfaces:**
- Consumes: install config builder and plugin/version/set completion.
- Produces: `NewInstallCommand` and POST workflow.

- [ ] **Step 1: Write failing install tests**

Test required ledger, default name, scalar flags, config assembly, confirmation rejection, API error, success renderer, plugin positional completion, version completion, set completion, and file completion for config/env files.

```go
func TestInstallBuildsInstanceFromPluginSchema(t *testing.T) {
	client := mutationClientMock{
		getPlugin: func(ctx context.Context, name string) (*connectivityclient.Plugin, error) { return pluginWithSchemaAndDefaults(), nil },
		create: func(ctx context.Context, body connectivityclient.InstanceCreate) (*connectivityclient.Instance, error) {
			require.Equal(t, "stripe-eu", body.Name)
			require.Equal(t, "main", body.Spec.Ledger)
			require.Equal(t, "api-key", *body.Spec.Config.Env["API_KEY"].Value)
			return &connectivityclient.Instance{Metadata: connectivityclient.ObjectMeta{Name: stringPtr(body.Name)}, Spec: body.Spec}, nil
		},
	}
	cmd := NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	cmd.SetArgs([]string{"stripe", "--name=stripe-eu", "--ledger=main", "--set=API_KEY=api-key", "--confirm"})
	require.NoError(t, cmd.Execute())
}
```

- [ ] **Step 2: Verify RED**

Run `nix develop -c go test ./cmd/connectivity/instances -run TestInstall -v`.

Expected: compilation fails because `NewInstallCommand` does not exist.

- [ ] **Step 3: Implement flags, completions, controller, and renderer**

Build `install <plugin>` with aliases `create,in`, exact one argument, `--name`, required `--ledger`, optional `--version`, `--start-sequence`, `--poll-interval`, `--config`, repeatable `--env-file`, repeatable `--set`, and confirmation. Fetch plugin, call `BuildInstallConfig`, approve, POST, store the returned instance, and render:

```text
Instance "stripe-eu" installed with plugin "stripe" for ledger "main".
```

Use `Flags().Changed` for optional scalar fields. Send config only when defaults or user inputs produce one.

- [ ] **Step 4: Verify GREEN, coverage, gates, and commit**

```text
nix develop -c go test ./cmd/connectivity/instances -run TestInstall -v
nix develop -c go test ./cmd/connectivity/instances -coverprofile=/tmp/connectivity-instances.cover
nix develop -c go tool cover -func=/tmp/connectivity-instances.cover
nix develop -c just tests
nix develop -c just pre-commit
git add cmd/connectivity/instances
git commit -m "feat(connectivity): install plugin instances"
```

---

### Task 6: Configure and uninstall instances

**Files:**
- Create: `cmd/connectivity/instances/configure.go`
- Test: `cmd/connectivity/instances/configure_test.go`
- Create: `cmd/connectivity/instances/uninstall.go`
- Test: `cmd/connectivity/instances/uninstall_test.go`
- Modify: `cmd/connectivity/instances/root.go`
- Modify: `cmd/connectivity/instances/completion.go`

**Interfaces:**
- Consumes: current instance config, plugin schema, configure builder, completion, confirmation.
- Produces: PATCH `configure` and DELETE `uninstall` workflows.

- [ ] **Step 1: Write failing configure tests**

Assert current-config precedence, all untouched file mounts preserved, changed scalar fields included, unchanged scalars omitted, no-change rejection, confirmation rejection, API errors, instance/version/set completion, and success rendering.

```go
func TestConfigureBuildsNestedPatch(t *testing.T) {
	client := configureClientMock{instance: instanceWithTwoFiles(), plugin: pluginWithFileSchema(), patch: func(ctx context.Context, name string, patch connectivityclient.InstancePatch) (*connectivityclient.Instance, error) {
		spec := patch["spec"].(map[string]any)
		require.Equal(t, "15s", spec["pollInterval"])
		config := spec["config"].(*connectivityclient.InstanceConfig)
		require.Len(t, config.Files, 2)
		return &connectivityclient.Instance{Metadata: connectivityclient.ObjectMeta{Name: stringPtr(name)}}, nil
	}}
	cmd := NewConfigureCommand(factoryReturning(client), mockReadFile(map[string]string{"new": "changed"}), mockPathCompleter(nil))
	cmd.SetArgs([]string{"stripe-eu", "--set=/etc/a=@new", "--poll-interval=15s", "--confirm"})
	require.NoError(t, cmd.Execute())
}
```

- [ ] **Step 2: Write failing uninstall tests and verify RED**

Test confirmed delete, rejected confirmation, API error, output, and instance completion. Run `nix develop -c go test ./cmd/connectivity/instances -run 'Test(Configure|Uninstall)' -v`.

Expected: compilation fails because mutation commands do not exist.

- [ ] **Step 3: Implement configure**

Build `configure <instance>` with aliases `config,update,c`, shared config flags, optional scalar flags, exact one argument, and confirmation. Fetch instance then plugin. Merge on current config. Build exactly:

```go
specPatch := map[string]any{}
if configChanged { specPatch["config"] = mergedConfig }
if cmd.Flags().Changed("version") { specPatch["version"] = version }
if cmd.Flags().Changed("ledger") { specPatch["ledger"] = ledger }
if cmd.Flags().Changed("start-sequence") { specPatch["startSequence"] = startSequence }
if cmd.Flags().Changed("poll-interval") { specPatch["pollInterval"] = pollInterval }
patch := connectivityclient.InstancePatch{"spec": specPatch}
```

Reject an empty map, approve, PATCH, and render the returned instance name.

- [ ] **Step 4: Implement uninstall**

Build `uninstall <instance>` with aliases `delete,remove,rm,u`, exact one argument, instance completion, and confirmation. Delete only after approval and render `Instance "<name>" uninstalled.`

- [ ] **Step 5: Verify GREEN, coverage, gates, and commit**

```text
nix develop -c go test ./cmd/connectivity/instances -v
nix develop -c go test ./cmd/connectivity/instances -coverprofile=/tmp/connectivity-instances.cover
nix develop -c go tool cover -func=/tmp/connectivity-instances.cover
nix develop -c just tests
nix develop -c just pre-commit
git add cmd/connectivity/instances
git commit -m "feat(connectivity): configure and uninstall instances"
```

---

### Task 7: Root wiring and final PR verification

**Files:**
- Create: `cmd/connectivity/root.go`
- Test: `cmd/connectivity/root_test.go`
- Modify: `cmd/root.go`
- Modify: `completions/fctl.bash`
- Modify: `completions/fctl.zsh`
- Modify: `completions/fctl.fish`

**Interfaces:**
- Consumes: plugin/instance roots and production dependencies.
- Produces: public `fctl connectivity` tree and shell scripts.

- [ ] **Step 1: Write failing tree tests**

```go
func TestCommandTreeMatchesConnectivityUX(t *testing.T) {
	cmd := NewCommand()
	require.Equal(t, "connectivity", cmd.Name())
	for _, path := range [][]string{{"plugins", "list"}, {"plugins", "show"}, {"instances", "list"}, {"instances", "show"}, {"instances", "install"}, {"instances", "configure"}, {"instances", "uninstall"}} {
		found, _, err := cmd.Find(path)
		require.NoError(t, err)
		require.Equal(t, path[len(path)-1], found.Name())
	}
	require.NotNil(t, cmd.PersistentFlags().Lookup("stack"))
	require.NotNil(t, cmd.PersistentFlags().Lookup("organization"))
}
```

Add a `cmd` package test that `NewRootCommand().Find([]string{"connectivity"})` succeeds.

- [ ] **Step 2: Verify RED**

Run `nix develop -c go test ./cmd/connectivity ./cmd -run 'Test(CommandTree|RootCommand)' -v`.

Expected: compilation or assertion failure because the root is not wired.

- [ ] **Step 3: Implement and register root**

```go
func NewCommand() *cobra.Command {
	factory := connectivityinternal.NewClientFactory()
	return fctl.NewStackCommand("connectivity",
		fctl.WithShortDescription("Manage Connectivity plugins and instances"),
		fctl.WithChildCommands(
			plugins.NewCommand(factory),
			instances.NewCommand(factory, fctl.ReadFile, instances.OSPathCompleter),
		),
	)
}
```

Import Connectivity in `cmd/root.go` and add it without moving existing children.

- [ ] **Step 4: Format, inspect help, and regenerate completions**

```text
nix develop -c gofmt -w internal/connectivityclient cmd/connectivity cmd/root.go
nix develop -c go run main.go connectivity --help
nix develop -c go run main.go connectivity plugins --help
nix develop -c go run main.go connectivity instances --help
nix develop -c just completions
```

Expected: help matches the approved hierarchy and all three scripts contain Connectivity commands and flags.

- [ ] **Step 5: Run final verification**

```text
nix develop -c just tests
nix develop -c go test -race ./internal/connectivityclient ./cmd/connectivity/...
nix develop -c go test -coverprofile=/tmp/connectivity-all.cover ./internal/connectivityclient ./cmd/connectivity/...
nix develop -c go tool cover -func=/tmp/connectivity-all.cover
nix develop -c just pre-commit
git diff --check
git status --short
```

Expected: all commands exit 0, race detector is clean, new code coverage is at least 80%, and no `cmd/payments` file changed.

- [ ] **Step 6: Commit and prepare PR**

```text
git add cmd/root.go cmd/connectivity/root.go cmd/connectivity/root_test.go completions/fctl.bash completions/fctl.zsh completions/fctl.fish docs/superpowers/plans/2026-08-07-connectivity-cli.md
git commit -m "feat(connectivity): expose fctl command tree"
```

PR title and body:

```text
feat(connectivity): add plugin and instance commands

- add a stack-authenticated fctl connectivity module
- expose plugin discovery and instance lifecycle workflows
- support schema-aware --set, --env-file, --config, and @file inputs
- add dynamic completion for plugins, instances, versions, schema keys, and paths

Testing:
- nix develop -c just tests
- nix develop -c go test -race ./internal/connectivityclient ./cmd/connectivity/...
- nix develop -c just completions
- nix develop -c just pre-commit
```
