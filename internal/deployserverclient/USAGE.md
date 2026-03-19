<!-- Start SDK Example Usage [usage] -->
```go
package main

import (
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
	ctx := context.Background()

	s := deployserverclient.New()

	res, err := s.ListApps(ctx, "<id>", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if res.ListAppsResponse != nil {
		// handle response
	}
}

```
<!-- End SDK Example Usage [usage] -->