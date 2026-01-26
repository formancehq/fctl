set dotenv-load

default:
  @just --list

pre-commit: tidy generate lint
pc: pre-commit completions

lint:
    golangci-lint run --fix --timeout 5m

tidy:
    go mod tidy

generate: generate-deploy-server-client generate-membership-client
    @go generate ./...
g: generate

[group('generate')]
generate-deploy-server-client:
    @cd internal/deployserverclient && speakeasy run --skip-versioning --frozen-workflow-lockfile

generate-membership-client:
    @cd internal/membershipclient && speakeasy run --skip-versioning --frozen-workflow-lockfile


tests:
    echo "not implemented"

release-local:
    @goreleaser release --nightly --skip=publish --clean

release-ci:
    @goreleaser release --nightly --clean

release:
    @goreleaser release --clean

completions: generate
    mkdir -p ./completions
    go run main.go completion bash > "./completions/fctl.bash"
    go run main.go completion zsh > "./completions/fctl.zsh"
    go run main.go completion fish > "./completions/fctl.fish"