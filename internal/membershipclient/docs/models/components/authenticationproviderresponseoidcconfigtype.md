# AuthenticationProviderResponseOIDCConfigType

Type of the authentication provider

## Example Usage

```go
import (
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

value := components.AuthenticationProviderResponseOIDCConfigTypeOidc

// Open enum: custom values can be created with a direct type cast
custom := components.AuthenticationProviderResponseOIDCConfigType("custom_value")
```


## Values

| Name                                                    | Value                                                   |
| ------------------------------------------------------- | ------------------------------------------------------- |
| `AuthenticationProviderResponseOIDCConfigTypeOidc`      | oidc                                                    |
| `AuthenticationProviderResponseOIDCConfigTypeGoogle`    | google                                                  |
| `AuthenticationProviderResponseOIDCConfigTypeGithub`    | github                                                  |
| `AuthenticationProviderResponseOIDCConfigTypeMicrosoft` | microsoft                                               |