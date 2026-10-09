set dotenv-load

default:
  @just --list

pre-commit: tidy generate lint
pc: pre-commit completions

lint:
    golangci-lint run --fix --timeout 5m
    cd pkg/pluginsdk && golangci-lint run --config ../../.golangci.yml --fix --timeout 5m

tidy:
    cd pkg/pluginsdk && go mod tidy
    go mod tidy

generate:
    @cd pkg/pluginsdk && go generate ./...
    @go generate ./...
g: generate

install:
    go install -v .

tests:
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
