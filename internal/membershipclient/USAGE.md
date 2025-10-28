<!-- Start SDK Example Usage [usage] -->
```go
package main

import (
	"context"
	"github.com/formancehq/fctl/internal/membershipclient"
	"log"
)

func main() {
	ctx := context.Background()

	s := membershipclient.New(
		membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
	)

	res, err := s.GetServerInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```
<!-- End SDK Example Usage [usage] -->