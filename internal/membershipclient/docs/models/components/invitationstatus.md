# InvitationStatus

## Example Usage

```go
import (
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

value := components.InvitationStatusPending

// Open enum: custom values can be created with a direct type cast
custom := components.InvitationStatus("custom_value")
```


## Values

| Name                        | Value                       |
| --------------------------- | --------------------------- |
| `InvitationStatusPending`   | PENDING                     |
| `InvitationStatusAccepted`  | ACCEPTED                    |
| `InvitationStatusRejected`  | REJECTED                    |
| `InvitationStatusCancelled` | CANCELLED                   |