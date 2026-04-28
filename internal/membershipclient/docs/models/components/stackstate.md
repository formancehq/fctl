# StackState

## Example Usage

```go
import (
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

value := components.StackStateActive

// Open enum: custom values can be created with a direct type cast
custom := components.StackState("custom_value")
```


## Values

| Name                 | Value                |
| -------------------- | -------------------- |
| `StackStateActive`   | ACTIVE               |
| `StackStateDisabled` | DISABLED             |
| `StackStateDeleted`  | DELETED              |