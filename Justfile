set dotenv-load

default:
  @just --list

pre-commit: tidy generate lint
pc: pre-commit completions

lint:
    cd misc/fctl-plugin && golangci-lint run --config ../../.golangci.yml --fix --timeout 5m
    golangci-lint run --fix --timeout 5m
    cd pkg/pluginsdk && golangci-lint run --config ../../.golangci.yml --fix --timeout 5m

tidy:
    cd misc/fctl-plugin && go mod tidy
    cd pkg/pluginsdk && go mod tidy
    go mod tidy

generate:
    @cd misc/fctl-plugin && go generate ./...
    @cd pkg/pluginsdk && go generate ./...
    @go generate ./...
g: generate

install:
    go install -v .

tests:
    cd misc/fctl-plugin && go test -race ./...
    cd pkg/pluginsdk && go test -race ./...
    go test -race ./...

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

build-legacy-plugin:
    cd misc/fctl-plugin && go build -trimpath -o ../../build/fctl-plugin-legacy ./cmd/fctl-plugin-legacy
