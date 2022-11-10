module github.com/formancehq/fctl

go 1.18

require (
	github.com/formancehq/auth/authclient v0.0.0-20221003212526-20761562e41e
	github.com/formancehq/fctl/membershipclient v0.0.0-20221110205214-79a725a64b70
	github.com/numary/ledger/client v0.0.0-20220912105324-6153d9ee752f
	github.com/numary/payments/client v0.0.0-20220912091851-92e6ed3de087
	github.com/pborman/indent v1.1.1
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.5.0
	github.com/spf13/viper v1.13.0
	github.com/zitadel/oidc v1.8.0
	golang.org/x/oauth2 v0.0.0-20220909003341-f21342109be1
)

require (
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/schema v1.2.0 // indirect
	github.com/gorilla/securecookie v1.1.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	golang.org/x/crypto v0.0.0-20220926161630-eccd6366d1be // indirect
	golang.org/x/net v0.0.0-20220927155233-aa73b2587036 // indirect
	golang.org/x/sys v0.0.0-20220926163933-8cfa568d3c25 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/zitadel/oidc => github.com/formancehq/oidc v0.0.0-20220922145049-a6daec727711

replace github.com/formancehq/fctl/membershipclient => ./membershipclient
