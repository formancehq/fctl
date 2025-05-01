set dotenv-load

default:
  @just --list

pre-commit: tidy generate lint
pc: pre-commit

lint:
    golangci-lint run --fix --build-tags it --timeout 5m
    for d in $(ls tools); do \
        pushd tools/$d; \
        golangci-lint run --fix --build-tags it --timeout 5m; \
        popd; \
    done
    cd {{justfile_directory()}}/deployments/pulumi && golangci-lint run --fix --build-tags it --timeout 5m

tidy:
    go mod tidy

generate: completions
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
g: generate

tests:
    echo "not implemented"

release-local:
    @goreleaser release --nightly --skip=publish --clean

release-ci:
    @goreleaser release --nightly --clean

release:
    @goreleaser release --clean

completions:
    mkdir -p ./completions
    go run main.go completion bash > "./completions/fctl.bash"
    go run main.go completion zsh > "./completions/fctl.zsh"
    go run main.go completion fish > "./completions/fctl.fish"