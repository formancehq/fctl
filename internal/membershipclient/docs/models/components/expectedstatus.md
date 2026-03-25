# ExpectedStatus

## Example Usage

```go
import (
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

value := components.ExpectedStatusReady

// Open enum: custom values can be created with a direct type cast
custom := components.ExpectedStatus("custom_value")
```


## Values

| Name                     | Value                    |
| ------------------------ | ------------------------ |
| `ExpectedStatusReady`    | READY                    |
| `ExpectedStatusDisabled` | DISABLED                 |
| `ExpectedStatusDeleted`  | DELETED                  |