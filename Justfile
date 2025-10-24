set dotenv-load

default:
  @just --list

pre-commit: tidy generate lint
pc: pre-commit completions

lint:
    golangci-lint run --fix --timeout 5m

tidy:
    go mod tidy

generate: 
    openapi-generator-cli generate \
        -i ./membership-swagger.yaml \
        -g go \
        -o ./membershipclient \
        --git-user-id=formancehq \
        --git-repo-id=fctl \
        -p packageVersion=latest \
        -p isGoSubmodule=true \
        -p packageName=membershipclient \
        -p disallowAdditionalPropertiesIfNotPresent=false
    rm -rf ./membershipclient/test
    rm -rf ./membershipclient/docs
g: generate

[group('generate')]
generate-client:
    @cd internal/deployserverclient && speakeasy run --skip-versioning


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