# StackStatus

## Example Usage

```go
import (
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

value := components.StackStatusUnknown

// Open enum: custom values can be created with a direct type cast
custom := components.StackStatus("custom_value")
```


## Values

| Name                     | Value                    |
| ------------------------ | ------------------------ |
| `StackStatusUnknown`     | UNKNOWN                  |
| `StackStatusProgressing` | PROGRESSING              |
| `StackStatusReady`       | READY                    |
| `StackStatusDisabled`    | DISABLED                 |
| `StackStatusDeleted`     | DELETED                  |