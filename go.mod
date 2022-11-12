module github.com/formancehq/fctl

go 1.18

require (
	github.com/athul/shelby v1.0.6
	github.com/c-bata/go-prompt v0.2.6
	github.com/davecgh/go-spew v1.1.1
	github.com/formancehq/auth/authclient v0.0.0-20221003212526-20761562e41e
	github.com/formancehq/fctl/membershipclient v0.0.0-20221110205214-79a725a64b70
	github.com/formancehq/search/client v0.0.0-20221112130150-832c2c17043b
	github.com/formancehq/webhooks/client v0.0.0-20221112150350-abea13654995
	github.com/iancoleman/strcase v0.2.0
	github.com/mattn/go-shellwords v1.0.12
	github.com/numary/ledger/client v0.0.0-20220912105324-6153d9ee752f
	github.com/numary/payments/client v0.0.0-20220912091851-92e6ed3de087
	github.com/pkg/errors v0.9.1
	github.com/pterm/pterm v0.12.49
	github.com/spf13/cobra v1.6.1
	github.com/zitadel/oidc v1.8.0
	golang.org/x/oauth2 v0.0.0-20220909003341-f21342109be1
)

require (
	atomicgo.dev/cursor v0.1.1 // indirect
	atomicgo.dev/keyboard v0.2.8 // indirect
	github.com/containerd/console v1.0.3 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gookit/color v1.5.2 // indirect
	github.com/gorilla/schema v1.2.0 // indirect
	github.com/gorilla/securecookie v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/lithammer/fuzzysearch v1.1.5 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect
	github.com/pkg/term v1.2.0-beta.2 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/talal/go-bits v0.0.0-20200204154716-071e9f3e66e1 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	golang.org/x/crypto v0.0.0-20220926161630-eccd6366d1be // indirect
	golang.org/x/net v0.0.0-20220927155233-aa73b2587036 // indirect
	golang.org/x/sys v0.0.0-20220926163933-8cfa568d3c25 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
)

replace github.com/zitadel/oidc => github.com/formancehq/oidc v0.0.0-20220922145049-a6daec727711

replace github.com/formancehq/fctl/membershipclient => ./membershipclient

replace github.com/spf13/cobra v1.6.1 => github.com/formancehq/cobra v0.0.0-20221112160629-60a6d6d55ef9
