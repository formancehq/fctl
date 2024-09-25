VERSION 0.8

IMPORT github.com/formancehq/earthly:tags/v0.16.0 AS core

FROM core+base-image

sources:
    WORKDIR /src
    COPY go.* .
    COPY --dir cmd pkg membershipclient .
    COPY main.go .
    SAVE ARTIFACT /src

lint:
    FROM core+builder-image
    COPY (+sources/*) /src
    COPY --pass-args +tidy/go.* .
    WORKDIR /src
    DO --pass-args core+GO_LINT
    SAVE ARTIFACT cmd AS LOCAL cmd
    SAVE ARTIFACT pkg AS LOCAL pkg
    SAVE ARTIFACT main.go AS LOCAL main.go

pre-commit:
    WAIT
        BUILD --pass-args +tidy
    END
    BUILD --pass-args +lint
    BUILD --pass-args +completions

tests:
    RUN echo "not implemented"

completions:
    FROM core+builder-image
    COPY --pass-args (+sources/src) /src
    WORKDIR /src
    RUN mkdir -p ./completions
    RUN go run main.go completion bash > "./completions/fctl.bash"
    RUN go run main.go completion zsh > "./completions/fctl.zsh"
    RUN go run main.go completion fish > "./completions/fctl.fish"
    SAVE ARTIFACT ./completions AS LOCAL completions

tidy:
    FROM core+builder-image
    COPY --pass-args (+sources/src) /src
    WORKDIR /src
    DO --pass-args core+GO_TIDY

membership-openapi:
    FROM core+base-image
    WORKDIR /src
    COPY membership-swagger.yaml openapi.yaml
    SAVE ARTIFACT openapi.yaml

generate-membership-client:
    FROM openapitools/openapi-generator-cli:v6.6.0
    WORKDIR /src
    COPY (+membership-openapi/openapi.yaml) .
    RUN docker-entrypoint.sh generate \
        -i ./openapi.yaml \
        -g go \
        -o ./membershipclient \
        --git-user-id=formancehq \
        --git-repo-id=fctl \
        -p packageVersion=latest \
        -p isGoSubmodule=true \
        -p packageName=membershipclient
    RUN rm -rf ./membershipclient/test
    SAVE ARTIFACT ./membershipclient AS LOCAL membershipclient

release:
    FROM core+builder-image
    ARG mode=local
    COPY --dir . /src
    DO core+GORELEASER --mode=$mode