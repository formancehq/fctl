VERSION 0.8
PROJECT FormanceHQ/fctl

IMPORT github.com/formancehq/earthly:tags/v0.19.1 AS core

FROM core+base-image

CACHE --sharing=shared --id go-mod-cache /go/pkg/mod
CACHE --sharing=shared --id go-cache /root/.cache/go-build

sources:
    WORKDIR /src
    COPY go.* .
    COPY --dir cmd pkg membershipclient .
    COPY main.go .
    SAVE ARTIFACT /src
