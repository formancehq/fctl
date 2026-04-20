# DeployServer SDK

## Overview

### Available Operations

* [ListApps](#listapps) - List organization apps
* [CreateApp](#createapp) - Create a new app
* [UpdateApp](#updateapp) - Update an app
* [ReadApp](#readapp) - read app details
* [DeleteApp](#deleteapp) - Delete an app
* [ReadAppVariables](#readappvariables) - Get all variables of an app
* [CreateAppVariable](#createappvariable) - Create variable for an app
* [DeleteAppVariable](#deleteappvariable) - Delete a variable from an app
* [CreateManifestRaw](#createmanifestraw) - Create a new manifest
* [CreateManifest](#createmanifest) - Create a new manifest
* [ListManifests](#listmanifests) - List manifests in the organization
* [ReadManifest](#readmanifest) - Read a manifest
* [UpdateManifest](#updatemanifest) - Update manifest metadata
* [DeleteManifest](#deletemanifest) - Delete a manifest and all its versions
* [PushManifestVersionRaw](#pushmanifestversionraw) - Push a new version of a manifest
* [PushManifestVersion](#pushmanifestversion) - Push a new version of a manifest
* [ListManifestVersions](#listmanifestversions) - List versions of a manifest
* [ReadManifestVersion](#readmanifestversion) - Get a specific manifest version with content
* [CreateDeployment](#createdeployment) - Create a deployment (triggers a run)
* [CreateDeploymentRaw](#createdeploymentraw) - Create a deployment (triggers a run)
* [ListDeployments](#listdeployments) - List deployments
* [ReadDeployment](#readdeployment) - Get a single deployment
* [ReadDeploymentLogs](#readdeploymentlogs) - Get run logs for a deployment

## ListApps

List organization apps

### Example Usage

<!-- UsageSnippet language="go" operationID="listApps" method="get" path="/apps" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ListApps(ctx, nil, nil)
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
| `pageNumber`                                             | `*int64`                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `pageSize`                                               | `*int64`                                                 | :heavy_minus_sign:                                       | N/A                                                      |
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
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.CreateApp(ctx, components.CreateAppRequest{
        Name: "<value>",
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
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.UpdateApp(ctx, "<id>", components.UpdateAppRequest{
        Name: "<value>",
    })
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
| `id`                                                                       | `string`                                                                   | :heavy_check_mark:                                                         | N/A                                                                        |
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
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadApp(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.AppResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                        | Type                                                                                                             | Required                                                                                                         | Description                                                                                                      |
| ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                            | [context.Context](https://pkg.go.dev/context#Context)                                                            | :heavy_check_mark:                                                                                               | The context to use for the request.                                                                              |
| `id`                                                                                                             | `string`                                                                                                         | :heavy_check_mark:                                                                                               | N/A                                                                                                              |
| `include`                                                                                                        | [][operations.ReadAppInclude](../../models/operations/readappinclude.md)                                         | :heavy_minus_sign:                                                                                               | Comma-separated list of related resources to include.<br/>- `state`: Include the current Terraform workspace state.<br/> |
| `opts`                                                                                                           | [][operations.Option](../../models/operations/option.md)                                                         | :heavy_minus_sign:                                                                                               | The options for this request.                                                                                    |

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
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
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
| `id`                                                     | `string`                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.DeleteAppResponse](../../models/operations/deleteappresponse.md), error**

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
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
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
| `id`                                                     | `string`                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `pageNumber`                                             | `*int64`                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `pageSize`                                               | `*int64`                                                 | :heavy_minus_sign:                                       | N/A                                                      |
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
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
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
| `id`                                                                                 | `string`                                                                             | :heavy_check_mark:                                                                   | N/A                                                                                  |
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
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
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
| `id`                                                     | `string`                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `variableID`                                             | `string`                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.DeleteAppVariableResponse](../../models/operations/deleteappvariableresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateManifestRaw

Create a new manifest

### Example Usage

<!-- UsageSnippet language="go" operationID="createManifest_raw" method="post" path="/manifests" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
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

    res, err := s.CreateManifestRaw(ctx, "<value>", example, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateManifestResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `name`                                                   | `string`                                                 | :heavy_check_mark:                                       | Name for the manifest                                    |
| `requestBody`                                            | `any`                                                    | :heavy_check_mark:                                       | N/A                                                      |
| `appID`                                                  | `*string`                                                | :heavy_minus_sign:                                       | Optional app ID to scope the manifest to a specific app  |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.CreateManifestRawResponse](../../models/operations/createmanifestrawresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateManifest

Create a new manifest

### Example Usage

<!-- UsageSnippet language="go" operationID="createManifest" method="post" path="/manifests" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.CreateManifest(ctx, "<value>", operations.CreateManifestRequestBody{}, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateManifestResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                    | Type                                                                                         | Required                                                                                     | Description                                                                                  |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `ctx`                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                        | :heavy_check_mark:                                                                           | The context to use for the request.                                                          |
| `name`                                                                                       | `string`                                                                                     | :heavy_check_mark:                                                                           | Name for the manifest                                                                        |
| `requestBody`                                                                                | [operations.CreateManifestRequestBody](../../models/operations/createmanifestrequestbody.md) | :heavy_check_mark:                                                                           | N/A                                                                                          |
| `appID`                                                                                      | `*string`                                                                                    | :heavy_minus_sign:                                                                           | Optional app ID to scope the manifest to a specific app                                      |
| `opts`                                                                                       | [][operations.Option](../../models/operations/option.md)                                     | :heavy_minus_sign:                                                                           | The options for this request.                                                                |

### Response

**[*operations.CreateManifestResponse](../../models/operations/createmanifestresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListManifests

List manifests in the organization

### Example Usage

<!-- UsageSnippet language="go" operationID="listManifests" method="get" path="/manifests" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ListManifests(ctx, nil, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListManifestsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                            | Type                                                                 | Required                                                             | Description                                                          |
| -------------------------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `ctx`                                                                | [context.Context](https://pkg.go.dev/context#Context)                | :heavy_check_mark:                                                   | The context to use for the request.                                  |
| `pageNumber`                                                         | `*int64`                                                             | :heavy_minus_sign:                                                   | N/A                                                                  |
| `pageSize`                                                           | `*int64`                                                             | :heavy_minus_sign:                                                   | N/A                                                                  |
| `appID`                                                              | `*string`                                                            | :heavy_minus_sign:                                                   | Filter manifests by app ID (includes org-wide manifests as fallback) |
| `opts`                                                               | [][operations.Option](../../models/operations/option.md)             | :heavy_minus_sign:                                                   | The options for this request.                                        |

### Response

**[*operations.ListManifestsResponse](../../models/operations/listmanifestsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadManifest

Read a manifest

### Example Usage

<!-- UsageSnippet language="go" operationID="readManifest" method="get" path="/manifests/{manifestId}" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadManifest(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.ManifestResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                | Type                                                                     | Required                                                                 | Description                                                              |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| `ctx`                                                                    | [context.Context](https://pkg.go.dev/context#Context)                    | :heavy_check_mark:                                                       | The context to use for the request.                                      |
| `manifestID`                                                             | `string`                                                                 | :heavy_check_mark:                                                       | N/A                                                                      |
| `include`                                                                | `*string`                                                                | :heavy_minus_sign:                                                       | Comma-separated includes (e.g. "latest" to embed latest version content) |
| `opts`                                                                   | [][operations.Option](../../models/operations/option.md)                 | :heavy_minus_sign:                                                       | The options for this request.                                            |

### Response

**[*operations.ReadManifestResponse](../../models/operations/readmanifestresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpdateManifest

Update manifest metadata

### Example Usage

<!-- UsageSnippet language="go" operationID="updateManifest" method="patch" path="/manifests/{manifestId}" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.UpdateManifest(ctx, "<id>", components.UpdateManifestRequest{
        Name: "<value>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ManifestResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                            | Type                                                                                 | Required                                                                             | Description                                                                          |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| `ctx`                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                | :heavy_check_mark:                                                                   | The context to use for the request.                                                  |
| `manifestID`                                                                         | `string`                                                                             | :heavy_check_mark:                                                                   | N/A                                                                                  |
| `updateManifestRequest`                                                              | [components.UpdateManifestRequest](../../models/components/updatemanifestrequest.md) | :heavy_check_mark:                                                                   | N/A                                                                                  |
| `opts`                                                                               | [][operations.Option](../../models/operations/option.md)                             | :heavy_minus_sign:                                                                   | The options for this request.                                                        |

### Response

**[*operations.UpdateManifestResponse](../../models/operations/updatemanifestresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteManifest

Delete a manifest and all its versions

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteManifest" method="delete" path="/manifests/{manifestId}" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.DeleteManifest(ctx, "<id>")
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
| `manifestID`                                             | `string`                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.DeleteManifestResponse](../../models/operations/deletemanifestresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.Error    | 409                | application/json   |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## PushManifestVersionRaw

Push a new version of a manifest

### Example Usage

<!-- UsageSnippet language="go" operationID="pushManifestVersion_raw" method="post" path="/manifests/{manifestId}/versions" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
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

    res, err := s.PushManifestVersionRaw(ctx, "<id>", example)
    if err != nil {
        log.Fatal(err)
    }
    if res.ManifestVersionResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `manifestID`                                             | `string`                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `requestBody`                                            | `any`                                                    | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.PushManifestVersionRawResponse](../../models/operations/pushmanifestversionrawresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## PushManifestVersion

Push a new version of a manifest

### Example Usage

<!-- UsageSnippet language="go" operationID="pushManifestVersion" method="post" path="/manifests/{manifestId}/versions" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.PushManifestVersion(ctx, "<id>", operations.PushManifestVersionRequestBody{})
    if err != nil {
        log.Fatal(err)
    }
    if res.ManifestVersionResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                              | Type                                                                                                   | Required                                                                                               | Description                                                                                            |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                                  | :heavy_check_mark:                                                                                     | The context to use for the request.                                                                    |
| `manifestID`                                                                                           | `string`                                                                                               | :heavy_check_mark:                                                                                     | N/A                                                                                                    |
| `requestBody`                                                                                          | [operations.PushManifestVersionRequestBody](../../models/operations/pushmanifestversionrequestbody.md) | :heavy_check_mark:                                                                                     | N/A                                                                                                    |
| `opts`                                                                                                 | [][operations.Option](../../models/operations/option.md)                                               | :heavy_minus_sign:                                                                                     | The options for this request.                                                                          |

### Response

**[*operations.PushManifestVersionResponse](../../models/operations/pushmanifestversionresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListManifestVersions

List versions of a manifest

### Example Usage

<!-- UsageSnippet language="go" operationID="listManifestVersions" method="get" path="/manifests/{manifestId}/versions" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ListManifestVersions(ctx, "<id>", nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListManifestVersionsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `manifestID`                                             | `string`                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `pageNumber`                                             | `*int64`                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `pageSize`                                               | `*int64`                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ListManifestVersionsResponse](../../models/operations/listmanifestversionsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadManifestVersion

Get a specific manifest version with content

### Example Usage

<!-- UsageSnippet language="go" operationID="readManifestVersion" method="get" path="/manifests/{manifestId}/versions/{version}" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadManifestVersion(ctx, "<id>", "<value>")
    if err != nil {
        log.Fatal(err)
    }
    if res.ManifestVersionResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `manifestID`                                             | `string`                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `version`                                                | `string`                                                 | :heavy_check_mark:                                       | Version number or "latest"                               |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadManifestVersionResponse](../../models/operations/readmanifestversionresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateDeployment

Create a deployment (triggers a run)

### Example Usage

<!-- UsageSnippet language="go" operationID="createDeployment" method="post" path="/deployments" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.CreateDeployment(ctx, components.CreateDeploymentRequest{
        AppID: "<id>",
    }, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.DeploymentResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `createDeploymentRequest`                                                                | [components.CreateDeploymentRequest](../../models/components/createdeploymentrequest.md) | :heavy_check_mark:                                                                       | N/A                                                                                      |
| `appID`                                                                                  | `*string`                                                                                | :heavy_minus_sign:                                                                       | App ID (required for inline YAML deploys)                                                |
| `opts`                                                                                   | [][operations.Option](../../models/operations/option.md)                                 | :heavy_minus_sign:                                                                       | The options for this request.                                                            |

### Response

**[*operations.CreateDeploymentResponse](../../models/operations/createdeploymentresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateDeploymentRaw

Create a deployment (triggers a run)

### Example Usage

<!-- UsageSnippet language="go" operationID="createDeployment_raw" method="post" path="/deployments" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"bytes"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.CreateDeploymentRaw(ctx, bytes.NewBuffer([]byte("{\"appId\":\"<id>\"}")), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.DeploymentResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `requestBody`                                            | `any`                                                    | :heavy_check_mark:                                       | N/A                                                      |
| `appID`                                                  | `*string`                                                | :heavy_minus_sign:                                       | App ID (required for inline YAML deploys)                |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.CreateDeploymentRawResponse](../../models/operations/createdeploymentrawresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListDeployments

List deployments

### Example Usage

<!-- UsageSnippet language="go" operationID="listDeployments" method="get" path="/deployments" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ListDeployments(ctx, nil, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListDeploymentsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `appID`                                                  | `*string`                                                | :heavy_minus_sign:                                       | N/A                                                      |
| `pageNumber`                                             | `*int64`                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `pageSize`                                               | `*int64`                                                 | :heavy_minus_sign:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ListDeploymentsResponse](../../models/operations/listdeploymentsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadDeployment

Get a single deployment

### Example Usage

<!-- UsageSnippet language="go" operationID="readDeployment" method="get" path="/deployments/{deploymentId}" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadDeployment(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.DeploymentResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                        | Type                                                                                                                             | Required                                                                                                                         | Description                                                                                                                      |
| -------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                            | [context.Context](https://pkg.go.dev/context#Context)                                                                            | :heavy_check_mark:                                                                                                               | The context to use for the request.                                                                                              |
| `deploymentID`                                                                                                                   | `string`                                                                                                                         | :heavy_check_mark:                                                                                                               | N/A                                                                                                                              |
| `include`                                                                                                                        | [][operations.ReadDeploymentInclude](../../models/operations/readdeploymentinclude.md)                                           | :heavy_minus_sign:                                                                                                               | Comma-separated list of related resources to include.<br/>- `state`: Include the Terraform state produced by this deployment's run.<br/> |
| `opts`                                                                                                                           | [][operations.Option](../../models/operations/option.md)                                                                         | :heavy_minus_sign:                                                                                                               | The options for this request.                                                                                                    |

### Response

**[*operations.ReadDeploymentResponse](../../models/operations/readdeploymentresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadDeploymentLogs

Get run logs for a deployment

### Example Usage

<!-- UsageSnippet language="go" operationID="readDeploymentLogs" method="get" path="/deployments/{deploymentId}/logs" -->
```go
package main

import(
	"context"
	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"log"
)

func main() {
    ctx := context.Background()

    s := deployserverclient.New()

    res, err := s.ReadDeploymentLogs(ctx, "<id>")
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
| `deploymentID`                                           | `string`                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadDeploymentLogsResponse](../../models/operations/readdeploymentlogsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |