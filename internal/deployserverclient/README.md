# github.com/formancehq/fctl/internal/deployserverclient

Developer-friendly & type-safe Go SDK specifically catered to leverage *github.com/formancehq/fctl/internal/deployserverclient* API.

<div align="left">
    <a href="https://www.speakeasy.com/?utm_source=github-com/formancehq/fctl/internal/deployserverclient&utm_campaign=go"><img src="https://custom-icon-badges.demolab.com/badge/-Built%20By%20Speakeasy-212015?style=for-the-badge&logoColor=FBE331&logo=speakeasy&labelColor=545454" /></a>
    <a href="https://opensource.org/licenses/MIT">
        <img src="https://img.shields.io/badge/License-MIT-blue.svg" style="width: 100px; height: 28px;" />
    </a>
</div>


<br /><br />
> [!IMPORTANT]
> This SDK is not yet ready for production use. To complete setup please follow the steps outlined in your [workspace](https://app.speakeasy.com/org/formance/formance). Delete this section before > publishing to a package manager.

<!-- Start Summary [summary] -->
## Summary


<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [github.com/formancehq/fctl/internal/deployserverclient](#githubcomformancehqfctlinternaldeployserverclient)
  * [SDK Installation](#sdk-installation)
  * [SDK Example Usage](#sdk-example-usage)
  * [Available Resources and Operations](#available-resources-and-operations)
  * [Retries](#retries)
  * [Error Handling](#error-handling)
  * [Server Selection](#server-selection)
  * [Custom HTTP Client](#custom-http-client)
* [Development](#development)
  * [Maturity](#maturity)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start SDK Installation [installation] -->
## SDK Installation

To add the SDK as a dependency to your project:
```bash
go get github.com/formancehq/fctl/internal/deployserverclient
```
<!-- End SDK Installation [installation] -->

<!-- Start SDK Example Usage [usage] -->
## SDK Example Usage

### Example

```go
package main

import (
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
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

<!-- Start Available Resources and Operations [operations] -->
## Available Resources and Operations

<details open>
<summary>Available methods</summary>

### [DeployServer SDK](docs/sdks/deployserver/README.md)

* [ListApps](docs/sdks/deployserver/README.md#listapps) - List organization apps
* [CreateApp](docs/sdks/deployserver/README.md#createapp) - Create a new app
* [UpdateApp](docs/sdks/deployserver/README.md#updateapp) - Update an app
* [ReadApp](docs/sdks/deployserver/README.md#readapp) - read app details
* [DeleteApp](docs/sdks/deployserver/README.md#deleteapp) - Delete an app
* [ReadAppCurrentStateVersion](docs/sdks/deployserver/README.md#readappcurrentstateversion) - Get the current state version of an app
* [ReadAppVariables](docs/sdks/deployserver/README.md#readappvariables) - Get all variables of an app
* [CreateAppVariable](docs/sdks/deployserver/README.md#createappvariable) - Create variable for an app
* [DeleteAppVariable](docs/sdks/deployserver/README.md#deleteappvariable) - Delete a variable from an app
* [ReadAppRuns](docs/sdks/deployserver/README.md#readappruns) - Get runs of an app
* [ReadAppVersions](docs/sdks/deployserver/README.md#readappversions) - Get versions of an app
* [DeployAppConfigurationRaw](docs/sdks/deployserver/README.md#deployappconfigurationraw) - Deploy a new configuration for an app
* [DeployAppConfiguration](docs/sdks/deployserver/README.md#deployappconfiguration) - Deploy a new configuration for an app
* [ReadCurrentRun](docs/sdks/deployserver/README.md#readcurrentrun) - Get the current run of an app
* [ReadVersion](docs/sdks/deployserver/README.md#readversion) - Get a specific version
* [ReadRun](docs/sdks/deployserver/README.md#readrun) - Get the run of a version
* [ReadRunLogs](docs/sdks/deployserver/README.md#readrunlogs) - Get logs of a run by its ID
* [ReadCurrentRunLogs](docs/sdks/deployserver/README.md#readcurrentrunlogs) - Get logs of the current run of an app
* [ReadCurrentAppVersion](docs/sdks/deployserver/README.md#readcurrentappversion) - Get the current version of an app

</details>
<!-- End Available Resources and Operations [operations] -->

<!-- Start Retries [retries] -->
## Retries

Some of the endpoints in this SDK support retries. If you use the SDK without any configuration, it will fall back to the default retry strategy provided by the API. However, the default retry strategy can be overridden on a per-operation basis, or across the entire SDK.

To change the default retry strategy for a single API call, simply provide a `retry.Config` object to the call by using the `WithRetries` option:
```go
package main

import (
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"github.com/formancehq/fctl/internal/deployserverclient/retry"
	"log"
	"models/operations"
)

func main() {
	ctx := context.Background()

	s := deployserverclient.New()

	res, err := s.ListApps(ctx, "<id>", nil, nil, operations.WithRetries(
		retry.Config{
			Strategy: "backoff",
			Backoff: &retry.BackoffStrategy{
				InitialInterval: 1,
				MaxInterval:     50,
				Exponent:        1.1,
				MaxElapsedTime:  100,
			},
			RetryConnectionErrors: false,
		}))
	if err != nil {
		log.Fatal(err)
	}
	if res.ListAppsResponse != nil {
		// handle response
	}
}

```

If you'd like to override the default retry strategy for all operations that support retries, you can use the `WithRetryConfig` option at SDK initialization:
```go
package main

import (
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"github.com/formancehq/fctl/internal/deployserverclient/retry"
	"log"
)

func main() {
	ctx := context.Background()

	s := deployserverclient.New(
		deployserverclient.WithRetryConfig(
			retry.Config{
				Strategy: "backoff",
				Backoff: &retry.BackoffStrategy{
					InitialInterval: 1,
					MaxInterval:     50,
					Exponent:        1.1,
					MaxElapsedTime:  100,
				},
				RetryConnectionErrors: false,
			}),
	)

	res, err := s.ListApps(ctx, "<id>", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if res.ListAppsResponse != nil {
		// handle response
	}
}

```
<!-- End Retries [retries] -->

<!-- Start Error Handling [errors] -->
## Error Handling

Handling errors in this SDK should largely match your expectations. All operations return a response object or an error, they will never return both.

By Default, an API error will return `apierrors.APIError`. When custom error responses are specified for an operation, the SDK may also return their associated error. You can refer to respective *Errors* tables in SDK docs for more details on possible error types for each operation.

For example, the `ListApps` function may return the following errors:

| Error Type         | Status Code | Content Type |
| ------------------ | ----------- | ------------ |
| apierrors.APIError | 4XX, 5XX    | \*/\*        |

### Example

```go
package main

import (
	"context"
	"errors"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"github.com/formancehq/fctl/internal/deployserverclient/models/apierrors"
	"log"
)

func main() {
	ctx := context.Background()

	s := deployserverclient.New()

	res, err := s.ListApps(ctx, "<id>", nil, nil)
	if err != nil {

		var e *apierrors.APIError
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}
	}
}

```
<!-- End Error Handling [errors] -->

<!-- Start Server Selection [server] -->
## Server Selection

### Select Server by Index

You can override the default server globally using the `WithServerIndex(serverIndex int)` option when initializing the SDK client instance. The selected server will then be used as the default on the operations that use it. This table lists the indexes associated with the available servers:

| #   | Server                                         | Description       |
| --- | ---------------------------------------------- | ----------------- |
| 0   | `https://deploy-server.staging.formance.cloud` | Staging server    |
| 1   | `https://deploy-server.formance.cloud`         | Production server |
| 2   | `http://localhost:8080`                        | Local server      |

#### Example

```go
package main

import (
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
	ctx := context.Background()

	s := deployserverclient.New(
		deployserverclient.WithServerIndex(2),
	)

	res, err := s.ListApps(ctx, "<id>", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if res.ListAppsResponse != nil {
		// handle response
	}
}

```

### Override Server URL Per-Client

The default server can also be overridden globally using the `WithServerURL(serverURL string)` option when initializing the SDK client instance. For example:
```go
package main

import (
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
	ctx := context.Background()

	s := deployserverclient.New(
		deployserverclient.WithServerURL("http://localhost:8080"),
	)

	res, err := s.ListApps(ctx, "<id>", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if res.ListAppsResponse != nil {
		// handle response
	}
}

```
<!-- End Server Selection [server] -->

<!-- Start Custom HTTP Client [http-client] -->
## Custom HTTP Client

The Go SDK makes API calls that wrap an internal HTTP client. The requirements for the HTTP client are very simple. It must match this interface:

```go
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
```

The built-in `net/http` client satisfies this interface and a default client based on the built-in is provided by default. To replace this default with a client of your own, you can implement this interface yourself or provide your own client configured as desired. Here's a simple example, which adds a client with a 30 second timeout.

```go
import (
	"net/http"
	"time"

	"github.com/formancehq/fctl/internal/deployserverclient"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	sdkClient  = deployserverclient.New(deployserverclient.WithClient(httpClient))
)
```

This can be a convenient way to configure timeouts, cookies, proxies, custom headers, and other low-level configuration.
<!-- End Custom HTTP Client [http-client] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This SDK is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

While we value open-source contributions to this SDK, this library is generated programmatically. Any manual changes added to internal files will be overwritten on the next generation. 
We look forward to hearing your feedback. Feel free to open a PR or an issue with a proof of concept and we'll do our best to include it in a future release. 

### SDK Created by [Speakeasy](https://www.speakeasy.com/?utm_source=github-com/formancehq/fctl/internal/deployserverclient&utm_campaign=go)
