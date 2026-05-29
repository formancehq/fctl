# Capabilities Manifest

`manifest_generated.go` is generated from the stack OpenAPI document.

Regenerate it from the v4 module directory:

```bash
curl -L https://github.com/formancehq/stack/releases/download/v3.2.4/generate.json -o /tmp/formance-stack-generate.json
go run ./internal/capabilities/genmanifest \
  -input /tmp/formance-stack-generate.json \
  -output internal/capabilities/manifest_generated.go
go test ./internal/capabilities ./internal/capabilities/genmanifest
```

The manual component-version compatibility ranges live in `compatibility.go`.
