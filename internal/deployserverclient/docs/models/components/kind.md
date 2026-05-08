# Kind

- `undeployed`: the app has a manifest bound but no successful deployment yet.
- `synced`: deployed (manifestId, version) matches the bound manifest's latest version.
- `behind`: deployed manifest matches the bound manifest but on an older version.
- `rebound`: the app was rebound; deployed manifest is a different manifest.


## Example Usage

```go
import (
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
)

value := components.KindUndeployed
```


## Values

| Name             | Value            |
| ---------------- | ---------------- |
| `KindUndeployed` | undeployed       |
| `KindSynced`     | synced           |
| `KindBehind`     | behind           |
| `KindRebound`    | rebound          |