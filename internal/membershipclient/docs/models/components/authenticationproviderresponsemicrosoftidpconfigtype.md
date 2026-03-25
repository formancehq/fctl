# AuthenticationProviderResponseMicrosoftIDPConfigType

Type of the authentication provider

## Example Usage

```go
import (
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

value := components.AuthenticationProviderResponseMicrosoftIDPConfigTypeOidc

// Open enum: custom values can be created with a direct type cast
custom := components.AuthenticationProviderResponseMicrosoftIDPConfigType("custom_value")
```


## Values

| Name                                                            | Value                                                           |
| --------------------------------------------------------------- | --------------------------------------------------------------- |
| `AuthenticationProviderResponseMicrosoftIDPConfigTypeOidc`      | oidc                                                            |
| `AuthenticationProviderResponseMicrosoftIDPConfigTypeGoogle`    | google                                                          |
| `AuthenticationProviderResponseMicrosoftIDPConfigTypeGithub`    | github                                                          |
| `AuthenticationProviderResponseMicrosoftIDPConfigTypeMicrosoft` | microsoft                                                       |