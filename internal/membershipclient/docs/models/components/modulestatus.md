# ModuleStatus

## Example Usage

```go
import (
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

value := components.ModuleStatusUnknown

// Open enum: custom values can be created with a direct type cast
custom := components.ModuleStatus("custom_value")
```


## Values

| Name                      | Value                     |
| ------------------------- | ------------------------- |
| `ModuleStatusUnknown`     | UNKNOWN                   |
| `ModuleStatusProgressing` | PROGRESSING               |
| `ModuleStatusReady`       | READY                     |
| `ModuleStatusDeleted`     | DELETED                   |