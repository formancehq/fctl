# DeployServer SDK

## Overview

### Available Operations

* [ListApps](#listapps) - List organization apps
* [CreateApp](#createapp) - Create a new app
* [UpdateApp](#updateapp) - Update an app
* [ReadApp](#readapp) - read app details
* [DeleteApp](#deleteapp) - Delete an app
* [ReadAppCurrentStateVersion](#readappcurrentstateversion) - Get the current state version of an app
* [ReadAppVariables](#readappvariables) - Get all variables of an app
* [CreateAppVariable](#createappvariable) - Create variable for an app
* [DeleteAppVariable](#deleteappvariable) - Delete a variable from an app
* [ReadAppRuns](#readappruns) - Get runs of an app
* [ReadAppVersions](#readappversions) - Get versions of an app
* [ReadAppManifest](#readappmanifest) - Get the last valid deployed manifest of an app
* [DeployAppConfigurationRaw](#deployappconfigurationraw) - Deploy a new configuration for an app
* [DeployAppConfiguration](#deployappconfiguration) - Deploy a new configuration for an app
* [ReadCurrentRun](#readcurrentrun) - Get the current run of an app
* [ReadVersion](#readversion) - Get a specific version
* [ReadRun](#readrun) - Get the run of a version
* [ReadCurrentRunLogs](#readcurrentrunlogs) - Get logs of the current run of an app
* [ReadCurrentAppVersion](#readcurrentappversion) - Get the current version of an app
* [DownloadAppVersion](#downloadappversion) - Download a specific version of an app

## ListApps

List organization apps

### Example Usage

<!-- UsageSnippet language="go" operationID="listApps" method="get" path="/apps" -->
```go
package main

import(
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

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `organizationID`                                         | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `pageNumber`                                             | **int64*                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `pageSize`                                               | **int64*                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ListAppsResponse](../../models/operations/listappsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateApp

Create a new app

### Example Usage

<!-- UsageSnippet language="go" operationID="createApp" method="post" path="/apps" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.CreateApp(ctx, components.CreateAppRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.AppResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                  | Type                                                                       | Required                                                                   | Description                                                                |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `ctx`                                                                      | [context.Context](https://pkg.go.dev/context#Context)                      | :heavy_check_mark:                                                         | The context to use for the request.                                        |
| `request`                                                                  | [components.CreateAppRequest](../../models/components/createapprequest.md) | :heavy_check_mark:                                                         | The request object to use for the request.                                 |
| `opts`                                                                     | [][operations.Option](../../models/operations/option.md)                   | :heavy_minus_sign:                                                         | The options for this request.                                              |

### Response

**[*operations.CreateAppResponse](../../models/operations/createappresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpdateApp

Update an app

### Example Usage

<!-- UsageSnippet language="go" operationID="updateApp" method="put" path="/apps/{id}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.UpdateApp(ctx, "<id>", components.UpdateAppRequest{})
    if err != nil {
        log.Fatal(err)
    }
    if res.Error != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                  | Type                                                                       | Required                                                                   | Description                                                                |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `ctx`                                                                      | [context.Context](https://pkg.go.dev/context#Context)                      | :heavy_check_mark:                                                         | The context to use for the request.                                        |
| `id`                                                                       | *string*                                                                   | :heavy_check_mark:                                                         | N/A                                                                        |
| `updateAppRequest`                                                         | [components.UpdateAppRequest](../../models/components/updateapprequest.md) | :heavy_check_mark:                                                         | N/A                                                                        |
| `opts`                                                                     | [][operations.Option](../../models/operations/option.md)                   | :heavy_minus_sign:                                                         | The options for this request.                                              |

### Response

**[*operations.UpdateAppResponse](../../models/operations/updateappresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadApp

read app details

### Example Usage

<!-- UsageSnippet language="go" operationID="readApp" method="get" path="/apps/{id}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadApp(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.AppResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadAppResponse](../../models/operations/readappresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteApp

Delete an app

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteApp" method="delete" path="/apps/{id}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.DeleteApp(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.Error != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.DeleteAppResponse](../../models/operations/deleteappresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadAppCurrentStateVersion

Get the current state version of an app

### Example Usage

<!-- UsageSnippet language="go" operationID="readAppCurrentStateVersion" method="get" path="/apps/{id}/current-state-version" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadAppCurrentStateVersion(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadStateResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadAppCurrentStateVersionResponse](../../models/operations/readappcurrentstateversionresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadAppVariables

Get all variables of an app

### Example Usage

<!-- UsageSnippet language="go" operationID="readAppVariables" method="get" path="/apps/{id}/variables" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadAppVariables(ctx, "<id>", nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadVariablesResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `pageNumber`                                             | **int64*                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `pageSize`                                               | **int64*                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadAppVariablesResponse](../../models/operations/readappvariablesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateAppVariable

Create variable for an app

### Example Usage

<!-- UsageSnippet language="go" operationID="createAppVariable" method="post" path="/apps/{id}/variables" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.CreateAppVariable(ctx, "<id>", components.CreateVariableRequest{
        Variable: components.VariableData{
            Key: "<key>",
            Value: "<value>",
            Sensitive: false,
            Category: components.VariableDataCategoryTerraform,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateVariableResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                            | Type                                                                                 | Required                                                                             | Description                                                                          |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| `ctx`                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                | :heavy_check_mark:                                                                   | The context to use for the request.                                                  |
| `id`                                                                                 | *string*                                                                             | :heavy_check_mark:                                                                   | N/A                                                                                  |
| `createVariableRequest`                                                              | [components.CreateVariableRequest](../../models/components/createvariablerequest.md) | :heavy_check_mark:                                                                   | N/A                                                                                  |
| `opts`                                                                               | [][operations.Option](../../models/operations/option.md)                             | :heavy_minus_sign:                                                                   | The options for this request.                                                        |

### Response

**[*operations.CreateAppVariableResponse](../../models/operations/createappvariableresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteAppVariable

Delete a variable from an app

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteAppVariable" method="delete" path="/apps/{id}/variables/{variableId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.DeleteAppVariable(ctx, "<id>", "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.Error != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `variableID`                                             | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.DeleteAppVariableResponse](../../models/operations/deleteappvariableresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadAppRuns

Get runs of an app

### Example Usage

<!-- UsageSnippet language="go" operationID="readAppRuns" method="get" path="/apps/{id}/runs" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadAppRuns(ctx, "<id>", nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListRunsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `pageNumber`                                             | **int64*                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `pageSize`                                               | **int64*                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadAppRunsResponse](../../models/operations/readapprunsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadAppVersions

Get versions of an app

### Example Usage

<!-- UsageSnippet language="go" operationID="readAppVersions" method="get" path="/apps/{id}/versions" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadAppVersions(ctx, "<id>", nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListVersionsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `pageNumber`                                             | **int64*                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `pageSize`                                               | **int64*                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadAppVersionsResponse](../../models/operations/readappversionsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadAppManifest

Get the last valid deployed manifest of an app

### Example Usage

<!-- UsageSnippet language="go" operationID="readAppManifest" method="get" path="/apps/{id}/manifest" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadAppManifest(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.ResponseStream != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `from`                                                   | [*operations.From](../../models/operations/from.md)      | :heavy_minus_sign:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadAppManifestResponse](../../models/operations/readappmanifestresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeployAppConfigurationRaw

Deploy a new configuration for an app

### Example Usage

<!-- UsageSnippet language="go" operationID="deployAppConfiguration_raw" method="post" path="/apps/{id}/deploy" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    example, fileErr := os.Open("example.file")
    if fileErr != nil {
        panic(fileErr)
    }

    res, err := s.DeployAppConfigurationRaw(ctx, "<id>", example)
    if err != nil {
        log.Fatal(err)
    }
    if res.RunResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `application`                                            | *any*                                                    | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.DeployAppConfigurationRawResponse](../../models/operations/deployappconfigurationrawresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeployAppConfiguration

Deploy a new configuration for an app

### Example Usage

<!-- UsageSnippet language="go" operationID="deployAppConfiguration" method="post" path="/apps/{id}/deploy" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.DeployAppConfiguration(ctx, "<id>", components.Application{})
    if err != nil {
        log.Fatal(err)
    }
    if res.RunResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                        | Type                                                             | Required                                                         | Description                                                      |
| ---------------------------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------- |
| `ctx`                                                            | [context.Context](https://pkg.go.dev/context#Context)            | :heavy_check_mark:                                               | The context to use for the request.                              |
| `id`                                                             | *string*                                                         | :heavy_check_mark:                                               | N/A                                                              |
| `application`                                                    | [components.Application](../../models/components/application.md) | :heavy_check_mark:                                               | N/A                                                              |
| `opts`                                                           | [][operations.Option](../../models/operations/option.md)         | :heavy_minus_sign:                                               | The options for this request.                                    |

### Response

**[*operations.DeployAppConfigurationResponse](../../models/operations/deployappconfigurationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadCurrentRun

Get the current run of an app

### Example Usage

<!-- UsageSnippet language="go" operationID="readCurrentRun" method="get" path="/apps/{id}/run" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadCurrentRun(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.RunResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadCurrentRunResponse](../../models/operations/readcurrentrunresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadVersion

Get a specific version

### Example Usage

<!-- UsageSnippet language="go" operationID="readVersion" method="get" path="/versions/{id}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadVersion(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.AppVersionResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadVersionResponse](../../models/operations/readversionresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadRun

Get the run of a version

### Example Usage

<!-- UsageSnippet language="go" operationID="readRun" method="get" path="/runs/{id}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadRun(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.RunResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadRunResponse](../../models/operations/readrunresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadCurrentRunLogs

Get logs of the current run of an app

### Example Usage

<!-- UsageSnippet language="go" operationID="readCurrentRunLogs" method="get" path="/apps/{id}/run/logs" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadCurrentRunLogs(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadLogsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadCurrentRunLogsResponse](../../models/operations/readcurrentrunlogsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadCurrentAppVersion

Get the current version of an app

### Example Usage

<!-- UsageSnippet language="go" operationID="readCurrentAppVersion" method="get" path="/apps/{id}/version" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadCurrentAppVersion(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.AppVersionResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadCurrentAppVersionResponse](../../models/operations/readcurrentappversionresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DownloadAppVersion

Download a specific version of an app

### Example Usage

<!-- UsageSnippet language="go" operationID="downloadAppVersion" method="get" path="/apps/{id}/version/download" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.DownloadAppVersion(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.ResponseStream != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.DownloadAppVersionResponse](../../models/operations/downloadappversionresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |