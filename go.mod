module github.com/formancehq/fctl

go 1.18

require (
	github.com/athul/shelby v1.0.6
	github.com/c-bata/go-prompt v0.2.6
	github.com/formancehq/auth/authclient v0.0.0-20221003212526-20761562e41e
	github.com/formancehq/fctl/membershipclient v0.0.0-20221110205214-79a725a64b70
	github.com/formancehq/payments/client v0.0.0-20221122203707-e41feb711635
	github.com/formancehq/search/client v0.0.0-20221113191621-1f6854b1e5dd
	github.com/formancehq/webhooks/client v0.0.0-20221113191112-15660877c6c0
	github.com/iancoleman/strcase v0.2.0
	github.com/mattn/go-shellwords v1.0.12
	github.com/numary/ledger/client v0.0.0-20221122120003-a06568501a4c
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
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/net v0.1.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/term v0.1.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
)

replace github.com/zitadel/oidc => github.com/formancehq/oidc v0.0.0-20220922145049-a6daec727711

replace github.com/formancehq/fctl/membershipclient => ./membershipclient

replace github.com/spf13/cobra v1.6.1 => github.com/formancehq/cobra v0.0.0-20221112160629-60a6d6d55ef9
