# AuthenticationProviderResponseGithubIDPConfigType

Type of the authentication provider

## Example Usage

```go
import (
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

value := components.AuthenticationProviderResponseGithubIDPConfigTypeOidc

// Open enum: custom values can be created with a direct type cast
custom := components.AuthenticationProviderResponseGithubIDPConfigType("custom_value")
```


## Values

| Name                                                         | Value                                                        |
| ------------------------------------------------------------ | ------------------------------------------------------------ |
| `AuthenticationProviderResponseGithubIDPConfigTypeOidc`      | oidc                                                         |
| `AuthenticationProviderResponseGithubIDPConfigTypeGoogle`    | google                                                       |
| `AuthenticationProviderResponseGithubIDPConfigTypeGithub`    | github                                                       |
| `AuthenticationProviderResponseGithubIDPConfigTypeMicrosoft` | microsoft                                                    |