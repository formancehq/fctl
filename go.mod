module github.com/formancehq/fctl

go 1.24.4

toolchain go1.24.7

replace (
	github.com/formancehq/fctl/internal/deployserverclient => ./internal/deployserverclient
	github.com/formancehq/fctl/membershipclient => ./membershipclient
	github.com/spf13/cobra => github.com/formancehq/cobra v0.0.0-20240315111924-ca456bf9cac9
	github.com/zitadel/oidc/v2 v2.6.1 => github.com/formancehq/oidc/v2 v2.6.2-0.20230526075055-93dc5ecb0149
)

require (
	github.com/TylerBrock/colorjson v0.0.0-20200706003622-8a50f05110d2
	github.com/c-bata/go-prompt v0.2.6
	github.com/formancehq/fctl/internal/deployserverclient v0.0.0-00010101000000-000000000000
	github.com/formancehq/fctl/membershipclient v0.0.0-00010101000000-000000000000
	github.com/formancehq/formance-sdk-go/v3 v3.5.0
	github.com/formancehq/go-libs v1.7.2
	github.com/formancehq/go-libs/v3 v3.2.2-0.20251029133300-46d86f349060
	github.com/iancoleman/strcase v0.3.0
	github.com/mattn/go-shellwords v1.0.12
	github.com/pkg/errors v0.9.1
	github.com/pterm/pterm v0.12.81
	github.com/segmentio/ksuid v1.0.4
	github.com/spf13/cobra v1.9.1
	github.com/zitadel/oidc/v2 v2.12.2
	golang.org/x/mod v0.27.0
	golang.org/x/oauth2 v0.31.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	atomicgo.dev/cursor v0.2.0 // indirect
	atomicgo.dev/keyboard v0.2.9 // indirect
	atomicgo.dev/schedule v0.1.0 // indirect
	dario.cat/mergo v1.0.2 // indirect
	github.com/ThreeDotsLabs/watermill v1.5.1 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/containerd/console v1.0.5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/ericlagergren/decimal v0.0.0-20240411145413-00de7ca16731 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/go-chi/chi/v5 v5.2.3 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/gorilla/schema v1.4.1 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/lithammer/fuzzysearch v1.1.8 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-tty v0.0.5 // indirect
	github.com/muhlemmer/gu v0.3.1 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/term v1.2.0-beta.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/uptrace/bun v1.2.15 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/zitadel/oidc/v3 v3.45.0 // indirect
	github.com/zitadel/schema v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/term v0.34.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.3 // indirect
	gopkg.in/validator.v2 v2.0.1 // indirect
)
