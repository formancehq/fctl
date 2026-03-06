# fctl Plugin Architecture

## Overview

fctl uses a **plugin system** to let each Formance product (ledger, payments, etc.) provide its own CLI commands. Plugins are **separate binaries** that communicate with fctl over gRPC using HashiCorp's [go-plugin](https://github.com/hashicorp/go-plugin) framework.

This keeps product-specific logic out of fctl's core, allows independent release cycles, and lets each product team own their CLI experience.

```
┌─────────────────────────────────────────────────────────────┐
│  fctl (host process)                                        │
│                                                             │
│  ┌─────────┐  ┌───────────────┐  ┌───────────────────────┐ │
│  │ Profile  │  │ Auth resolver │  │ Cobra command tree    │ │
│  │ manager  │  │ (membership,  │  │                       │ │
│  │          │  │  stack tokens)│  │  fctl                 │ │
│  └─────────┘  └───────────────┘  │  ├── ledger (plugin)  │ │
│                                   │  ├── payments (plugin)│ │
│                                   │  ├── plugin install   │ │
│                                   │  └── ...              │ │
│                                   └───────────────────────┘ │
│                        │                                     │
│              ┌─────────┴──────────┐                         │
│              │  Plugin Manager    │                         │
│              │  (discover, load,  │                         │
│              │   dispatch, kill)  │                         │
│              └────┬──────────┬───┘                         │
└───────────────────┼──────────┼──────────────────────────────┘
                    │ gRPC     │ gRPC
         ┌──────────▼───┐  ┌──▼──────────────┐
         │ fctl-plugin-  │  │ fctl-plugin-     │
         │ ledger        │  │ payments         │
         │ (separate     │  │ (separate        │
         │  process)     │  │  process)        │
         └──────────────┘  └─────────────────┘
```

## Plugin protocol

### gRPC service

Each plugin implements a single gRPC service with two RPCs:

```protobuf
service PluginService {
  rpc GetManifest(GetManifestRequest) returns (GetManifestResponse);
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
}
```

- **GetManifest**: returns the plugin's metadata and full command tree. Called once at startup.
- **Execute**: runs a specific command. Called each time the user invokes a plugin command.

### Handshake

fctl and plugins use a magic cookie for process-level validation:

```go
HandshakeConfig{
    ProtocolVersion:  1,
    MagicCookieKey:   "FCTL_PLUGIN",
    MagicCookieValue: "formance",
}
```

The plugin binary is spawned by fctl. Communication happens over gRPC via the plugin's stdin/stdout (managed by go-plugin).

### Plugin manifest

```protobuf
message PluginManifest {
    string name         = 1;  // e.g. "ledger"
    string version      = 2;  // e.g. "1.0.0"
    string description  = 3;
    CommandSpec root_command = 4;  // full command tree
}
```

The manifest declares the entire command hierarchy. fctl converts it into cobra commands dynamically.

### Command spec

```protobuf
message CommandSpec {
    string use                  = 1;   // "transactions"
    repeated string aliases     = 2;   // ["tx", "t"]
    string short                = 3;   // short help
    string long                 = 4;   // long help
    repeated FlagSpec flags     = 5;
    repeated FlagSpec persistent_flags = 6;
    repeated CommandSpec subcommands   = 7;
    bool runnable               = 8;   // has an Execute handler
    CommandType command_type    = 9;   // BASIC, MEMBERSHIP, or STACK
    bool hidden                 = 10;
    string deprecated           = 11;
    string args_constraint      = 12;  // "exact:1", "none", "max:2", etc.
    bool confirm                = 13;  // require --confirm flag
}
```

### Command types

The `CommandType` tells fctl what authentication context to resolve before calling the plugin:

| Type | Auth resolved | Use case |
|------|--------------|----------|
| `COMMAND_TYPE_BASIC` | None | Commands that don't need auth |
| `COMMAND_TYPE_MEMBERSHIP` | Membership token + org ID | Org-level management |
| `COMMAND_TYPE_STACK` | Stack access token + service URL | Product API calls (most common) |

### Execute request

```protobuf
message ExecuteRequest {
    string command_path             = 1;  // "transactions/list"
    repeated string args            = 2;  // positional args
    map<string, string> flags       = 3;  // resolved flag values
    AuthContext auth_context        = 4;
    string output_format            = 5;  // "plain" or "json"
}
```

fctl collects all flags as `map[string, string]`, resolves auth based on the command type, and sends everything to the plugin.

### Auth context

```protobuf
message AuthContext {
    string stack_url        = 1;   // service gRPC address
    string access_token     = 2;   // JWT bearer token for the service
    string organization_id  = 3;
    string stack_id         = 4;
    string membership_url   = 5;
    string membership_token = 6;
    bool   insecure_tls     = 7;
    bool   debug            = 8;
}
```

fctl handles all the OAuth/OIDC flows, token refresh, profile management. The plugin receives ready-to-use credentials.

### Execute response

```protobuf
message ExecuteResponse {
    oneof result {
        ExecuteSuccess success = 1;
        ExecuteError error     = 2;
    }
}

message ExecuteSuccess {
    string json_data      = 1;  // structured output (for --output json)
    string rendered_text  = 2;  // human-readable output (tables, etc.)
}

message ExecuteError {
    string message = 1;
    int32 code     = 2;
}
```

## Plugin lifecycle

### 1. Discovery

fctl reads `~/.config/formance/fctl/plugins.json`:

```json
{
  "plugins": [
    {
      "name": "ledger",
      "version": "1.0.0",
      "path": ""
    }
  ]
}
```

If `path` is empty, fctl looks for the binary at:
`~/.config/formance/fctl/plugins/{name}/{version}/fctl-plugin-{name}`

### 2. Loading

For each configured plugin:

1. Spawn the plugin binary using `exec.Command`
2. Perform go-plugin handshake (magic cookie + protocol version)
3. Establish gRPC connection over stdin/stdout
4. Obtain a `FctlPlugin` client interface

### 3. Manifest registration

1. Call `plugin.GetManifest()`
2. Recursively convert the `CommandSpec` tree into cobra commands
3. If a built-in fctl command has the same name, the plugin **overrides** it
4. Add the command tree to fctl's root

### 4. Command execution

When the user runs `fctl ledger transactions list --ledger foo`:

```
User                    fctl                        Plugin
 │                       │                            │
 │  fctl ledger tx list  │                            │
 │  --ledger foo         │                            │
 │ ─────────────────────►│                            │
 │                       │                            │
 │                       │ 1. Resolve CommandType      │
 │                       │    (STACK)                  │
 │                       │                            │
 │                       │ 2. Load profile            │
 │                       │ 3. Obtain stack token      │
 │                       │ 4. Resolve service URL     │
 │                       │                            │
 │                       │ 5. Build ExecuteRequest:   │
 │                       │    path: "transactions/list"│
 │                       │    flags: {ledger: "foo"}  │
 │                       │    auth: {url, token}      │
 │                       │                            │
 │                       │  ── Execute(req) ─────────►│
 │                       │                            │
 │                       │                            │ 6. Create gRPC client
 │                       │                            │    (using auth.stack_url)
 │                       │                            │
 │                       │                            │ 7. Call service API
 │                       │                            │    (with auth.access_token
 │                       │                            │     as bearer token)
 │                       │                            │
 │                       │                            │ 8. Format output
 │                       │                            │
 │                       │  ◄── ExecuteResponse ──────│
 │                       │                            │
 │  ◄─── display output ─│                            │
 │                       │                            │
```

### 5. Shutdown

On fctl exit, `PluginManager.Shutdown()` kills all plugin processes.

## Plugin management CLI

fctl provides built-in commands for managing plugins:

```bash
# Install a plugin from the registry
fctl plugin install ledger
fctl plugin install ledger --version 1.2.0

# List installed plugins
fctl plugin list

# Update plugins
fctl plugin update ledger      # update one
fctl plugin update --all       # update all

# Remove a plugin
fctl plugin remove ledger
```

### Plugin registry

Plugins are distributed via a central registry (GitHub-hosted JSON):
`https://raw.githubusercontent.com/formancehq/fctl-plugin-registry/main/registry.json`

The registry contains per-plugin metadata with platform-specific binary URLs:

```json
{
  "plugins": {
    "ledger": {
      "versions": {
        "1.0.0": {
          "minCoreVersion": "0.1.0",
          "binaries": {
            "linux/amd64": "https://github.com/.../ledger-linux-amd64",
            "darwin/arm64": "https://github.com/.../ledger-darwin-arm64"
          }
        }
      }
    }
  }
}
```

## Plugin SDK

Product teams implement a plugin using the `pluginsdk` Go package:

```go
package pluginsdk

type FctlPlugin interface {
    GetManifest(ctx context.Context) (*pluginpb.PluginManifest, error)
    Execute(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error)
}

func Serve(impl FctlPlugin)
```

### Minimal plugin example

```go
package main

import (
    "context"
    "github.com/formancehq/fctl/pkg/pluginsdk"
    "github.com/formancehq/fctl/pkg/pluginsdk/pluginpb"
)

type MyPlugin struct{}

func (p *MyPlugin) GetManifest(_ context.Context) (*pluginpb.PluginManifest, error) {
    return &pluginpb.PluginManifest{
        Name:    "my-product",
        Version: "1.0.0",
        RootCommand: &pluginpb.CommandSpec{
            Use:         "my-product",
            Short:       "My product commands",
            CommandType: pluginpb.CommandType_COMMAND_TYPE_STACK,
            Subcommands: []*pluginpb.CommandSpec{
                {
                    Use:      "list",
                    Short:    "List resources",
                    Runnable: true,
                    CommandType: pluginpb.CommandType_COMMAND_TYPE_STACK,
                },
            },
        },
    }, nil
}

func (p *MyPlugin) Execute(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
    // req.CommandPath = "list"
    // req.AuthContext.StackUrl = "grpc.stack-xxx.formance.cloud:443"
    // req.AuthContext.AccessToken = "eyJ..."

    // Call your product's gRPC API using the auth context...

    return &pluginpb.ExecuteResponse{
        Result: &pluginpb.ExecuteResponse_Success{
            Success: &pluginpb.ExecuteSuccess{
                RenderedText: "Resource 1\nResource 2\n",
            },
        },
    }, nil
}

func main() {
    pluginsdk.Serve(&MyPlugin{})
}
```

## Shared command pattern (ledger example)

For products that also have a standalone CLI (e.g., `ledgerctl`), commands can be defined once and built for both targets using an adapter pattern:

```
cmd/shared/commands/          Shared definitions (CommandDef + Handler)
                              Uses a Runtime interface for environment abstraction

cmd/shared/cobradapter/       CommandDef → cobra.Command (for standalone CLI)
                              CobraRuntime implements Runtime

cmd/fctl-plugin/              Separate Go sub-module (imports pluginsdk)
  pluginadapter/              CommandDef → pluginpb.CommandSpec (for fctl)
                              PluginRuntime implements Runtime
  main.go                     pluginsdk.Serve()
```

The `Runtime` interface abstracts flag access, gRPC client creation, output, and auth:

```go
type Runtime interface {
    Flag(name string) string
    BoolFlag(name string) bool
    Args() []string
    Writer() io.Writer
    IsJSON() bool
    Client() (servicepb.Client, *grpc.ClientConn, error)
    Context() (context.Context, context.CancelFunc)
    SignRequests(requests []*servicepb.Request) error
}
```

| | CobraRuntime (standalone CLI) | PluginRuntime (fctl plugin) |
|---|---|---|
| `Writer()` | `os.Stdout` | `bytes.Buffer` (returned in response) |
| `Client()` | From `--server` flag + profiles | From `AuthContext.StackUrl` |
| `Context()` | From `--auth-token` flag | From `AuthContext.AccessToken` |
| `SignRequests()` | Ed25519 signing from `--signing-key` | No-op |
| Interactive features | Spinners, prompts (in CLI-only commands) | Non-interactive, flags required |

The **sub-module** pattern (`cmd/fctl-plugin/go.mod`) keeps the pluginsdk dependency isolated — the main module has no knowledge of fctl.

## Design principles

1. **Process isolation**: each plugin runs as a separate process. A crash in a plugin doesn't take down fctl.

2. **Manifest-driven commands**: plugins declare their CLI tree dynamically. fctl doesn't hardcode product-specific commands.

3. **Auth delegation**: fctl handles all OAuth/OIDC flows, token refresh, profile management. Plugins receive ready-to-use credentials via `AuthContext`.

4. **Command type awareness**: `BASIC`, `MEMBERSHIP`, `STACK` levels let plugins declare what auth context they need. fctl only resolves what's required.

5. **Override built-in commands**: plugins can replace built-in fctl commands. This allows gradual migration from built-in to plugin-based commands.

6. **Registry distribution**: plugins are versioned and distributed via a central registry with per-platform binaries.

7. **Dual-target support**: products with a standalone CLI can define commands once and build for both targets using the adapter pattern + Runtime interface.

## File map

| File | Purpose |
|------|---------|
| `pkg/pluginsdk/sdk.go` | Plugin SDK (FctlPlugin interface + Serve) |
| `pkg/pluginsdk/pluginpb/` | Generated protobuf types |
| `proto/fctl/plugin/v1/plugin.proto` | Protocol definition |
| `pkg/plugin/manager.go` | Plugin discovery, loading, install, lifecycle |
| `pkg/plugin/loader.go` | go-plugin client setup + gRPC connection |
| `pkg/plugin/cobra.go` | Manifest → cobra conversion + auth dispatch |
| `pkg/plugin/config.go` | plugins.json read/write |
| `pkg/plugin/registry.go` | Remote registry client |
| `cmd/plugin/` | `fctl plugin install/list/update/remove` commands |
| `cmd/root.go` | Plugin loading at startup (lines 118-131) |
