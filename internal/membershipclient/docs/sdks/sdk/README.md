# SDK

## Overview

### Available Operations

* [GetServerInfo](#getserverinfo) - Get server info
* [ListOrganizations](#listorganizations) - List organizations of the connected user
* [CreateOrganization](#createorganization) - Create organization
* [~~ListOrganizationsExpanded~~](#listorganizationsexpanded) - List organizations of the connected user with expanded data :warning: **Deprecated**
* [ReadOrganization](#readorganization) - Read organization
* [UpdateOrganization](#updateorganization) - Update organization
* [DeleteOrganization](#deleteorganization) - Delete organization
* [ReadAuthenticationProvider](#readauthenticationprovider) - Read authentication provider
* [UpsertAuthenticationProvider](#upsertauthenticationprovider) - Upsert an authentication provider
* [DeleteAuthenticationProvider](#deleteauthenticationprovider) - Delete authentication provider
* [ListFeatures](#listfeatures) - List features
* [AddFeatures](#addfeatures) - Add Features
* [DeleteFeature](#deletefeature) - Delete feature
* [~~ReadOrganizationClient~~](#readorganizationclient) - Read organization client (DEPRECATED) (until 12/31/2025) :warning: **Deprecated**
* [~~CreateOrganizationClient~~](#createorganizationclient) - Create organization client (DEPRECATED) (until 12/31/2025) :warning: **Deprecated**
* [~~DeleteOrganizationClient~~](#deleteorganizationclient) - Delete organization client (DEPRECATED) (until 12/31/2025) :warning: **Deprecated**
* [OrganizationClientsRead](#organizationclientsread) - Read organization clients
* [OrganizationClientCreate](#organizationclientcreate) - Create organization client
* [OrganizationClientRead](#organizationclientread) - Read organization client
* [OrganizationClientDelete](#organizationclientdelete) - Delete organization client
* [OrganizationClientUpdate](#organizationclientupdate) - Update organization client
* [ListLogs](#listlogs) - List logs
* [ListUsersOfOrganization](#listusersoforganization) - List users of organization
* [ReadUserOfOrganization](#readuseroforganization) - Read user of organization
* [UpsertOrganizationUser](#upsertorganizationuser) - Update user within an organization
* [DeleteUserFromOrganization](#deleteuserfromorganization) - delete user from organization
* [ListPolicies](#listpolicies) - List policies of organization
* [CreatePolicy](#createpolicy) - Create policy
* [ReadPolicy](#readpolicy) - Read policy with scopes
* [UpdatePolicy](#updatepolicy) - Update policy
* [DeletePolicy](#deletepolicy) - Delete policy
* [AddScopeToPolicy](#addscopetopolicy) - Add scope to policy
* [RemoveScopeFromPolicy](#removescopefrompolicy) - Remove scope from policy
* [ListStacks](#liststacks) - List stacks
* [CreateStack](#createstack) - Create stack
* [ListModules](#listmodules) - List modules of a stack
* [EnableModule](#enablemodule) - enable module
* [DisableModule](#disablemodule) - disable module
* [UpgradeStack](#upgradestack) - Upgrade stack
* [GetStack](#getstack) - Find stack
* [UpdateStack](#updatestack) - Update stack
* [DeleteStack](#deletestack) - Delete stack
* [ListStackUsersAccesses](#liststackusersaccesses) - List stack users accesses within an organization
* [ReadStackUserAccess](#readstackuseraccess) - Read stack user access within an organization
* [DeleteStackUserAccess](#deletestackuseraccess) - Delete stack user access within an organization
* [UpsertStackUserAccess](#upsertstackuseraccess) - Update stack user access within an organization
* [DisableStack](#disablestack) - Disable stack
* [EnableStack](#enablestack) - Enable stack
* [RestoreStack](#restorestack) - Restore stack
* [EnableStargate](#enablestargate) - Enable stargate on a stack
* [DisableStargate](#disablestargate) - Disable stargate on a stack
* [ListInvitations](#listinvitations) - List invitations of the user
* [AcceptInvitation](#acceptinvitation) - Accept invitation
* [DeclineInvitation](#declineinvitation) - Decline invitation
* [ListOrganizationInvitations](#listorganizationinvitations) - List invitations of the organization
* [CreateInvitation](#createinvitation) - Create invitation
* [DeleteInvitation](#deleteinvitation) - Delete invitation
* [ListRegions](#listregions) - List regions
* [CreatePrivateRegion](#createprivateregion) - Create a private region
* [GetRegion](#getregion) - Get region
* [DeleteRegion](#deleteregion) - Delete region
* [GetRegionVersions](#getregionversions) - Get region versions
* [ListOrganizationApplications](#listorganizationapplications) - List applications enabled for organization
* [GetOrganizationApplication](#getorganizationapplication) - Get application for organization
* [EnableApplicationForOrganization](#enableapplicationfororganization) - Enable application for organization
* [DisableApplicationForOrganization](#disableapplicationfororganization) - Disable application for organization
* [ListApplications](#listapplications) - List applications
* [CreateApplication](#createapplication) - Create application
* [GetApplication](#getapplication) - Get application
* [UpdateApplication](#updateapplication) - Update application
* [DeleteApplication](#deleteapplication) - Delete application
* [CreateApplicationScope](#createapplicationscope) - Create application scope
* [DeleteApplicationScope](#deleteapplicationscope) - Delete application scope
* [CreateUser](#createuser) - Create user
* [ReadConnectedUser](#readconnecteduser) - Read user

## GetServerInfo

Get server info

### Example Usage

<!-- UsageSnippet language="go" operationID="getServerInfo" method="get" path="/_info" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
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

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.GetServerInfoResponse](../../models/operations/getserverinforesponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListOrganizations

List organizations of the connected user

### Example Usage

<!-- UsageSnippet language="go" operationID="listOrganizations" method="get" path="/organizations" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListOrganizations(ctx, operations.ListOrganizationsRequest{})
    if err != nil {
        log.Fatal(err)
    }
    if res.ListOrganizationExpandedResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                  | Type                                                                                       | Required                                                                                   | Description                                                                                |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| `ctx`                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                      | :heavy_check_mark:                                                                         | The context to use for the request.                                                        |
| `request`                                                                                  | [operations.ListOrganizationsRequest](../../models/operations/listorganizationsrequest.md) | :heavy_check_mark:                                                                         | The request object to use for the request.                                                 |
| `opts`                                                                                     | [][operations.Option](../../models/operations/option.md)                                   | :heavy_minus_sign:                                                                         | The options for this request.                                                              |

### Response

**[*operations.ListOrganizationsResponse](../../models/operations/listorganizationsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateOrganization

Create organization

### Example Usage

<!-- UsageSnippet language="go" operationID="createOrganization" method="post" path="/organizations" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.CreateOrganization(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateOrganizationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                    | Type                                                                                         | Required                                                                                     | Description                                                                                  |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `ctx`                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                        | :heavy_check_mark:                                                                           | The context to use for the request.                                                          |
| `request`                                                                                    | [components.CreateOrganizationRequest](../../models/components/createorganizationrequest.md) | :heavy_check_mark:                                                                           | The request object to use for the request.                                                   |
| `opts`                                                                                       | [][operations.Option](../../models/operations/option.md)                                     | :heavy_minus_sign:                                                                           | The options for this request.                                                                |

### Response

**[*operations.CreateOrganizationResponse](../../models/operations/createorganizationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ~~ListOrganizationsExpanded~~

List organizations of the connected user with expanded data

> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

### Example Usage

<!-- UsageSnippet language="go" operationID="listOrganizationsExpanded" method="get" path="/organizations/expanded" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListOrganizationsExpanded(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListOrganizationExpandedResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ListOrganizationsExpandedResponse](../../models/operations/listorganizationsexpandedresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadOrganization

Read organization

### Example Usage

<!-- UsageSnippet language="go" operationID="readOrganization" method="get" path="/organizations/{organizationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ReadOrganization(ctx, operations.ReadOrganizationRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadOrganizationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `request`                                                                                | [operations.ReadOrganizationRequest](../../models/operations/readorganizationrequest.md) | :heavy_check_mark:                                                                       | The request object to use for the request.                                               |
| `opts`                                                                                   | [][operations.Option](../../models/operations/option.md)                                 | :heavy_minus_sign:                                                                       | The options for this request.                                                            |

### Response

**[*operations.ReadOrganizationResponse](../../models/operations/readorganizationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpdateOrganization

Update organization

### Example Usage

<!-- UsageSnippet language="go" operationID="updateOrganization" method="put" path="/organizations/{organizationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.UpdateOrganization(ctx, operations.UpdateOrganizationRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadOrganizationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                    | Type                                                                                         | Required                                                                                     | Description                                                                                  |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `ctx`                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                        | :heavy_check_mark:                                                                           | The context to use for the request.                                                          |
| `request`                                                                                    | [operations.UpdateOrganizationRequest](../../models/operations/updateorganizationrequest.md) | :heavy_check_mark:                                                                           | The request object to use for the request.                                                   |
| `opts`                                                                                       | [][operations.Option](../../models/operations/option.md)                                     | :heavy_minus_sign:                                                                           | The options for this request.                                                                |

### Response

**[*operations.UpdateOrganizationResponse](../../models/operations/updateorganizationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteOrganization

Delete organization

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteOrganization" method="delete" path="/organizations/{organizationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteOrganization(ctx, operations.DeleteOrganizationRequest{
        OrganizationID: "<id>",
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

| Parameter                                                                                    | Type                                                                                         | Required                                                                                     | Description                                                                                  |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `ctx`                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                        | :heavy_check_mark:                                                                           | The context to use for the request.                                                          |
| `request`                                                                                    | [operations.DeleteOrganizationRequest](../../models/operations/deleteorganizationrequest.md) | :heavy_check_mark:                                                                           | The request object to use for the request.                                                   |
| `opts`                                                                                       | [][operations.Option](../../models/operations/option.md)                                     | :heavy_minus_sign:                                                                           | The options for this request.                                                                |

### Response

**[*operations.DeleteOrganizationResponse](../../models/operations/deleteorganizationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadAuthenticationProvider

Read authentication provider

### Example Usage

<!-- UsageSnippet language="go" operationID="readAuthenticationProvider" method="get" path="/organizations/{organizationId}/authentication-provider" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ReadAuthenticationProvider(ctx, operations.ReadAuthenticationProviderRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.AuthenticationProviderResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                    | Type                                                                                                         | Required                                                                                                     | Description                                                                                                  |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                                        | :heavy_check_mark:                                                                                           | The context to use for the request.                                                                          |
| `request`                                                                                                    | [operations.ReadAuthenticationProviderRequest](../../models/operations/readauthenticationproviderrequest.md) | :heavy_check_mark:                                                                                           | The request object to use for the request.                                                                   |
| `opts`                                                                                                       | [][operations.Option](../../models/operations/option.md)                                                     | :heavy_minus_sign:                                                                                           | The options for this request.                                                                                |

### Response

**[*operations.ReadAuthenticationProviderResponse](../../models/operations/readauthenticationproviderresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpsertAuthenticationProvider

Upsert an authentication provider

### Example Usage

<!-- UsageSnippet language="go" operationID="upsertAuthenticationProvider" method="put" path="/organizations/{organizationId}/authentication-provider" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.UpsertAuthenticationProvider(ctx, operations.UpsertAuthenticationProviderRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.AuthenticationProviderResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                        | Type                                                                                                             | Required                                                                                                         | Description                                                                                                      |
| ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                            | [context.Context](https://pkg.go.dev/context#Context)                                                            | :heavy_check_mark:                                                                                               | The context to use for the request.                                                                              |
| `request`                                                                                                        | [operations.UpsertAuthenticationProviderRequest](../../models/operations/upsertauthenticationproviderrequest.md) | :heavy_check_mark:                                                                                               | The request object to use for the request.                                                                       |
| `opts`                                                                                                           | [][operations.Option](../../models/operations/option.md)                                                         | :heavy_minus_sign:                                                                                               | The options for this request.                                                                                    |

### Response

**[*operations.UpsertAuthenticationProviderResponse](../../models/operations/upsertauthenticationproviderresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteAuthenticationProvider

Delete authentication provider

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteAuthenticationProvider" method="delete" path="/organizations/{organizationId}/authentication-provider" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteAuthenticationProvider(ctx, operations.DeleteAuthenticationProviderRequest{
        OrganizationID: "<id>",
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

| Parameter                                                                                                        | Type                                                                                                             | Required                                                                                                         | Description                                                                                                      |
| ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                            | [context.Context](https://pkg.go.dev/context#Context)                                                            | :heavy_check_mark:                                                                                               | The context to use for the request.                                                                              |
| `request`                                                                                                        | [operations.DeleteAuthenticationProviderRequest](../../models/operations/deleteauthenticationproviderrequest.md) | :heavy_check_mark:                                                                                               | The request object to use for the request.                                                                       |
| `opts`                                                                                                           | [][operations.Option](../../models/operations/option.md)                                                         | :heavy_minus_sign:                                                                                               | The options for this request.                                                                                    |

### Response

**[*operations.DeleteAuthenticationProviderResponse](../../models/operations/deleteauthenticationproviderresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListFeatures

List features

### Example Usage

<!-- UsageSnippet language="go" operationID="listFeatures" method="get" path="/organizations/{organizationId}/features" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListFeatures(ctx, operations.ListFeaturesRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.Object != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.ListFeaturesRequest](../../models/operations/listfeaturesrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.ListFeaturesResponse](../../models/operations/listfeaturesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## AddFeatures

Add Features

### Example Usage

<!-- UsageSnippet language="go" operationID="addFeatures" method="post" path="/organizations/{organizationId}/features" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.AddFeatures(ctx, operations.AddFeaturesRequest{
        OrganizationID: "<id>",
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

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.AddFeaturesRequest](../../models/operations/addfeaturesrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |

### Response

**[*operations.AddFeaturesResponse](../../models/operations/addfeaturesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteFeature

Delete feature

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteFeature" method="delete" path="/organizations/{organizationId}/features/{name}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteFeature(ctx, operations.DeleteFeatureRequest{
        OrganizationID: "<id>",
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

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.DeleteFeatureRequest](../../models/operations/deletefeaturerequest.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |
| `opts`                                                                             | [][operations.Option](../../models/operations/option.md)                           | :heavy_minus_sign:                                                                 | The options for this request.                                                      |

### Response

**[*operations.DeleteFeatureResponse](../../models/operations/deletefeatureresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ~~ReadOrganizationClient~~

Read organization client (DEPRECATED) (until 12/31/2025)

> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

### Example Usage

<!-- UsageSnippet language="go" operationID="readOrganizationClient" method="get" path="/organizations/{organizationId}/client" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ReadOrganizationClient(ctx, operations.ReadOrganizationClientRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateClientResponseResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                            | Type                                                                                                 | Required                                                                                             | Description                                                                                          |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                                | :heavy_check_mark:                                                                                   | The context to use for the request.                                                                  |
| `request`                                                                                            | [operations.ReadOrganizationClientRequest](../../models/operations/readorganizationclientrequest.md) | :heavy_check_mark:                                                                                   | The request object to use for the request.                                                           |
| `opts`                                                                                               | [][operations.Option](../../models/operations/option.md)                                             | :heavy_minus_sign:                                                                                   | The options for this request.                                                                        |

### Response

**[*operations.ReadOrganizationClientResponse](../../models/operations/readorganizationclientresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ~~CreateOrganizationClient~~

Create organization client (DEPRECATED) (until 12/31/2025)

> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

### Example Usage

<!-- UsageSnippet language="go" operationID="createOrganizationClient" method="put" path="/organizations/{organizationId}/client" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.CreateOrganizationClient(ctx, operations.CreateOrganizationClientRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateClientResponseResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                | Type                                                                                                     | Required                                                                                                 | Description                                                                                              |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                    | :heavy_check_mark:                                                                                       | The context to use for the request.                                                                      |
| `request`                                                                                                | [operations.CreateOrganizationClientRequest](../../models/operations/createorganizationclientrequest.md) | :heavy_check_mark:                                                                                       | The request object to use for the request.                                                               |
| `opts`                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                 | :heavy_minus_sign:                                                                                       | The options for this request.                                                                            |

### Response

**[*operations.CreateOrganizationClientResponse](../../models/operations/createorganizationclientresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ~~DeleteOrganizationClient~~

Delete organization client (DEPRECATED) (until 12/31/2025)

> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteOrganizationClient" method="delete" path="/organizations/{organizationId}/client" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteOrganizationClient(ctx, operations.DeleteOrganizationClientRequest{
        OrganizationID: "<id>",
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

| Parameter                                                                                                | Type                                                                                                     | Required                                                                                                 | Description                                                                                              |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                    | :heavy_check_mark:                                                                                       | The context to use for the request.                                                                      |
| `request`                                                                                                | [operations.DeleteOrganizationClientRequest](../../models/operations/deleteorganizationclientrequest.md) | :heavy_check_mark:                                                                                       | The request object to use for the request.                                                               |
| `opts`                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                 | :heavy_minus_sign:                                                                                       | The options for this request.                                                                            |

### Response

**[*operations.DeleteOrganizationClientResponse](../../models/operations/deleteorganizationclientresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## OrganizationClientsRead

Read organization clients

### Example Usage

<!-- UsageSnippet language="go" operationID="organizationClientsRead" method="get" path="/organizations/{organizationId}/clients" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.OrganizationClientsRead(ctx, operations.OrganizationClientsReadRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadOrganizationClientsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                              | Type                                                                                                   | Required                                                                                               | Description                                                                                            |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                                  | :heavy_check_mark:                                                                                     | The context to use for the request.                                                                    |
| `request`                                                                                              | [operations.OrganizationClientsReadRequest](../../models/operations/organizationclientsreadrequest.md) | :heavy_check_mark:                                                                                     | The request object to use for the request.                                                             |
| `opts`                                                                                                 | [][operations.Option](../../models/operations/option.md)                                               | :heavy_minus_sign:                                                                                     | The options for this request.                                                                          |

### Response

**[*operations.OrganizationClientsReadResponse](../../models/operations/organizationclientsreadresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## OrganizationClientCreate

Create organization client

### Example Usage

<!-- UsageSnippet language="go" operationID="organizationClientCreate" method="post" path="/organizations/{organizationId}/clients" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.OrganizationClientCreate(ctx, operations.OrganizationClientCreateRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateOrganizationClientResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                | Type                                                                                                     | Required                                                                                                 | Description                                                                                              |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                    | :heavy_check_mark:                                                                                       | The context to use for the request.                                                                      |
| `request`                                                                                                | [operations.OrganizationClientCreateRequest](../../models/operations/organizationclientcreaterequest.md) | :heavy_check_mark:                                                                                       | The request object to use for the request.                                                               |
| `opts`                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                 | :heavy_minus_sign:                                                                                       | The options for this request.                                                                            |

### Response

**[*operations.OrganizationClientCreateResponse](../../models/operations/organizationclientcreateresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## OrganizationClientRead

Read organization client

### Example Usage

<!-- UsageSnippet language="go" operationID="organizationClientRead" method="get" path="/organizations/{organizationId}/clients/{clientId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.OrganizationClientRead(ctx, operations.OrganizationClientReadRequest{
        OrganizationID: "<id>",
        ClientID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadOrganizationClientResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                            | Type                                                                                                 | Required                                                                                             | Description                                                                                          |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                                | :heavy_check_mark:                                                                                   | The context to use for the request.                                                                  |
| `request`                                                                                            | [operations.OrganizationClientReadRequest](../../models/operations/organizationclientreadrequest.md) | :heavy_check_mark:                                                                                   | The request object to use for the request.                                                           |
| `opts`                                                                                               | [][operations.Option](../../models/operations/option.md)                                             | :heavy_minus_sign:                                                                                   | The options for this request.                                                                        |

### Response

**[*operations.OrganizationClientReadResponse](../../models/operations/organizationclientreadresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## OrganizationClientDelete

Delete organization client

### Example Usage

<!-- UsageSnippet language="go" operationID="organizationClientDelete" method="delete" path="/organizations/{organizationId}/clients/{clientId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.OrganizationClientDelete(ctx, operations.OrganizationClientDeleteRequest{
        OrganizationID: "<id>",
        ClientID: "<id>",
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

| Parameter                                                                                                | Type                                                                                                     | Required                                                                                                 | Description                                                                                              |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                    | :heavy_check_mark:                                                                                       | The context to use for the request.                                                                      |
| `request`                                                                                                | [operations.OrganizationClientDeleteRequest](../../models/operations/organizationclientdeleterequest.md) | :heavy_check_mark:                                                                                       | The request object to use for the request.                                                               |
| `opts`                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                 | :heavy_minus_sign:                                                                                       | The options for this request.                                                                            |

### Response

**[*operations.OrganizationClientDeleteResponse](../../models/operations/organizationclientdeleteresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## OrganizationClientUpdate

Update organization client

### Example Usage

<!-- UsageSnippet language="go" operationID="organizationClientUpdate" method="put" path="/organizations/{organizationId}/clients/{clientId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.OrganizationClientUpdate(ctx, operations.OrganizationClientUpdateRequest{
        OrganizationID: "<id>",
        ClientID: "<id>",
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

| Parameter                                                                                                | Type                                                                                                     | Required                                                                                                 | Description                                                                                              |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                    | :heavy_check_mark:                                                                                       | The context to use for the request.                                                                      |
| `request`                                                                                                | [operations.OrganizationClientUpdateRequest](../../models/operations/organizationclientupdaterequest.md) | :heavy_check_mark:                                                                                       | The request object to use for the request.                                                               |
| `opts`                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                 | :heavy_minus_sign:                                                                                       | The options for this request.                                                                            |

### Response

**[*operations.OrganizationClientUpdateResponse](../../models/operations/organizationclientupdateresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListLogs

List logs

### Example Usage

<!-- UsageSnippet language="go" operationID="listLogs" method="get" path="/organizations/{organizationId}/logs" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListLogs(ctx, operations.ListLogsRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.LogCursor != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                | Type                                                                     | Required                                                                 | Description                                                              |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| `ctx`                                                                    | [context.Context](https://pkg.go.dev/context#Context)                    | :heavy_check_mark:                                                       | The context to use for the request.                                      |
| `request`                                                                | [operations.ListLogsRequest](../../models/operations/listlogsrequest.md) | :heavy_check_mark:                                                       | The request object to use for the request.                               |
| `opts`                                                                   | [][operations.Option](../../models/operations/option.md)                 | :heavy_minus_sign:                                                       | The options for this request.                                            |

### Response

**[*operations.ListLogsResponse](../../models/operations/listlogsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListUsersOfOrganization

List users of organization

### Example Usage

<!-- UsageSnippet language="go" operationID="listUsersOfOrganization" method="get" path="/organizations/{organizationId}/users" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListUsersOfOrganization(ctx, operations.ListUsersOfOrganizationRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ListUsersResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                              | Type                                                                                                   | Required                                                                                               | Description                                                                                            |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                                  | :heavy_check_mark:                                                                                     | The context to use for the request.                                                                    |
| `request`                                                                                              | [operations.ListUsersOfOrganizationRequest](../../models/operations/listusersoforganizationrequest.md) | :heavy_check_mark:                                                                                     | The request object to use for the request.                                                             |
| `opts`                                                                                                 | [][operations.Option](../../models/operations/option.md)                                               | :heavy_minus_sign:                                                                                     | The options for this request.                                                                          |

### Response

**[*operations.ListUsersOfOrganizationResponse](../../models/operations/listusersoforganizationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadUserOfOrganization

Read user of organization

### Example Usage

<!-- UsageSnippet language="go" operationID="readUserOfOrganization" method="get" path="/organizations/{organizationId}/users/{userId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ReadUserOfOrganization(ctx, operations.ReadUserOfOrganizationRequest{
        OrganizationID: "<id>",
        UserID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadOrganizationUserResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                            | Type                                                                                                 | Required                                                                                             | Description                                                                                          |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                                | :heavy_check_mark:                                                                                   | The context to use for the request.                                                                  |
| `request`                                                                                            | [operations.ReadUserOfOrganizationRequest](../../models/operations/readuseroforganizationrequest.md) | :heavy_check_mark:                                                                                   | The request object to use for the request.                                                           |
| `opts`                                                                                               | [][operations.Option](../../models/operations/option.md)                                             | :heavy_minus_sign:                                                                                   | The options for this request.                                                                        |

### Response

**[*operations.ReadUserOfOrganizationResponse](../../models/operations/readuseroforganizationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpsertOrganizationUser

Update user within an organization

### Example Usage

<!-- UsageSnippet language="go" operationID="upsertOrganizationUser" method="put" path="/organizations/{organizationId}/users/{userId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.UpsertOrganizationUser(ctx, operations.UpsertOrganizationUserRequest{
        OrganizationID: "<id>",
        UserID: "<id>",
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

| Parameter                                                                                            | Type                                                                                                 | Required                                                                                             | Description                                                                                          |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                                | :heavy_check_mark:                                                                                   | The context to use for the request.                                                                  |
| `request`                                                                                            | [operations.UpsertOrganizationUserRequest](../../models/operations/upsertorganizationuserrequest.md) | :heavy_check_mark:                                                                                   | The request object to use for the request.                                                           |
| `opts`                                                                                               | [][operations.Option](../../models/operations/option.md)                                             | :heavy_minus_sign:                                                                                   | The options for this request.                                                                        |

### Response

**[*operations.UpsertOrganizationUserResponse](../../models/operations/upsertorganizationuserresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteUserFromOrganization

The owner of the organization can remove anyone while each user can leave any organization where it is not owner.


### Example Usage

<!-- UsageSnippet language="go" operationID="deleteUserFromOrganization" method="delete" path="/organizations/{organizationId}/users/{userId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteUserFromOrganization(ctx, operations.DeleteUserFromOrganizationRequest{
        OrganizationID: "<id>",
        UserID: "<id>",
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

| Parameter                                                                                                    | Type                                                                                                         | Required                                                                                                     | Description                                                                                                  |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                                        | :heavy_check_mark:                                                                                           | The context to use for the request.                                                                          |
| `request`                                                                                                    | [operations.DeleteUserFromOrganizationRequest](../../models/operations/deleteuserfromorganizationrequest.md) | :heavy_check_mark:                                                                                           | The request object to use for the request.                                                                   |
| `opts`                                                                                                       | [][operations.Option](../../models/operations/option.md)                                                     | :heavy_minus_sign:                                                                                           | The options for this request.                                                                                |

### Response

**[*operations.DeleteUserFromOrganizationResponse](../../models/operations/deleteuserfromorganizationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListPolicies

List policies of organization

### Example Usage

<!-- UsageSnippet language="go" operationID="listPolicies" method="get" path="/organizations/{organizationId}/policies" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListPolicies(ctx, operations.ListPoliciesRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ListPoliciesResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.ListPoliciesRequest](../../models/operations/listpoliciesrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.ListPoliciesResponse](../../models/operations/listpoliciesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreatePolicy

Create policy

### Example Usage

<!-- UsageSnippet language="go" operationID="createPolicy" method="post" path="/organizations/{organizationId}/policies" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.CreatePolicy(ctx, operations.CreatePolicyRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.CreatePolicyResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.CreatePolicyRequest](../../models/operations/createpolicyrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.CreatePolicyResponse](../../models/operations/createpolicyresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadPolicy

Read policy with scopes

### Example Usage

<!-- UsageSnippet language="go" operationID="readPolicy" method="get" path="/organizations/{organizationId}/policies/{policyId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ReadPolicy(ctx, operations.ReadPolicyRequest{
        OrganizationID: "<id>",
        PolicyID: 831591,
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadPolicyResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                    | Type                                                                         | Required                                                                     | Description                                                                  |
| ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `ctx`                                                                        | [context.Context](https://pkg.go.dev/context#Context)                        | :heavy_check_mark:                                                           | The context to use for the request.                                          |
| `request`                                                                    | [operations.ReadPolicyRequest](../../models/operations/readpolicyrequest.md) | :heavy_check_mark:                                                           | The request object to use for the request.                                   |
| `opts`                                                                       | [][operations.Option](../../models/operations/option.md)                     | :heavy_minus_sign:                                                           | The options for this request.                                                |

### Response

**[*operations.ReadPolicyResponse](../../models/operations/readpolicyresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpdatePolicy

Update policy

### Example Usage

<!-- UsageSnippet language="go" operationID="updatePolicy" method="put" path="/organizations/{organizationId}/policies/{policyId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.UpdatePolicy(ctx, operations.UpdatePolicyRequest{
        OrganizationID: "<id>",
        PolicyID: 127460,
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.UpdatePolicyResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.UpdatePolicyRequest](../../models/operations/updatepolicyrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.UpdatePolicyResponse](../../models/operations/updatepolicyresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeletePolicy

Delete policy

### Example Usage

<!-- UsageSnippet language="go" operationID="deletePolicy" method="delete" path="/organizations/{organizationId}/policies/{policyId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeletePolicy(ctx, operations.DeletePolicyRequest{
        OrganizationID: "<id>",
        PolicyID: 114294,
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

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.DeletePolicyRequest](../../models/operations/deletepolicyrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.DeletePolicyResponse](../../models/operations/deletepolicyresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## AddScopeToPolicy

Add scope to policy

### Example Usage

<!-- UsageSnippet language="go" operationID="addScopeToPolicy" method="put" path="/organizations/{organizationId}/policies/{policyId}/scopes/{scopeId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.AddScopeToPolicy(ctx, operations.AddScopeToPolicyRequest{
        OrganizationID: "<id>",
        PolicyID: 328027,
        ScopeID: 675877,
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

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `request`                                                                                | [operations.AddScopeToPolicyRequest](../../models/operations/addscopetopolicyrequest.md) | :heavy_check_mark:                                                                       | The request object to use for the request.                                               |
| `opts`                                                                                   | [][operations.Option](../../models/operations/option.md)                                 | :heavy_minus_sign:                                                                       | The options for this request.                                                            |

### Response

**[*operations.AddScopeToPolicyResponse](../../models/operations/addscopetopolicyresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## RemoveScopeFromPolicy

Remove scope from policy

### Example Usage

<!-- UsageSnippet language="go" operationID="removeScopeFromPolicy" method="delete" path="/organizations/{organizationId}/policies/{policyId}/scopes/{scopeId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.RemoveScopeFromPolicy(ctx, operations.RemoveScopeFromPolicyRequest{
        OrganizationID: "<id>",
        PolicyID: 995736,
        ScopeID: 485996,
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

| Parameter                                                                                          | Type                                                                                               | Required                                                                                           | Description                                                                                        |
| -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                              | [context.Context](https://pkg.go.dev/context#Context)                                              | :heavy_check_mark:                                                                                 | The context to use for the request.                                                                |
| `request`                                                                                          | [operations.RemoveScopeFromPolicyRequest](../../models/operations/removescopefrompolicyrequest.md) | :heavy_check_mark:                                                                                 | The request object to use for the request.                                                         |
| `opts`                                                                                             | [][operations.Option](../../models/operations/option.md)                                           | :heavy_minus_sign:                                                                                 | The options for this request.                                                                      |

### Response

**[*operations.RemoveScopeFromPolicyResponse](../../models/operations/removescopefrompolicyresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListStacks

List stacks

### Example Usage

<!-- UsageSnippet language="go" operationID="listStacks" method="get" path="/organizations/{organizationId}/stacks" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListStacks(ctx, operations.ListStacksRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ListStacksResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                    | Type                                                                         | Required                                                                     | Description                                                                  |
| ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `ctx`                                                                        | [context.Context](https://pkg.go.dev/context#Context)                        | :heavy_check_mark:                                                           | The context to use for the request.                                          |
| `request`                                                                    | [operations.ListStacksRequest](../../models/operations/liststacksrequest.md) | :heavy_check_mark:                                                           | The request object to use for the request.                                   |
| `opts`                                                                       | [][operations.Option](../../models/operations/option.md)                     | :heavy_minus_sign:                                                           | The options for this request.                                                |

### Response

**[*operations.ListStacksResponse](../../models/operations/liststacksresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateStack

Create stack

### Example Usage

<!-- UsageSnippet language="go" operationID="createStack" method="post" path="/organizations/{organizationId}/stacks" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.CreateStack(ctx, operations.CreateStackRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadStackResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.CreateStackRequest](../../models/operations/createstackrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |

### Response

**[*operations.CreateStackResponse](../../models/operations/createstackresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListModules

List modules of a stack

### Example Usage

<!-- UsageSnippet language="go" operationID="listModules" method="get" path="/organizations/{organizationId}/stacks/{stackId}/modules" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListModules(ctx, operations.ListModulesRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ListModulesResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.ListModulesRequest](../../models/operations/listmodulesrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |

### Response

**[*operations.ListModulesResponse](../../models/operations/listmodulesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## EnableModule

enable module

### Example Usage

<!-- UsageSnippet language="go" operationID="enableModule" method="post" path="/organizations/{organizationId}/stacks/{stackId}/modules" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.EnableModule(ctx, operations.EnableModuleRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
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

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.EnableModuleRequest](../../models/operations/enablemodulerequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.EnableModuleResponse](../../models/operations/enablemoduleresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DisableModule

disable module

### Example Usage

<!-- UsageSnippet language="go" operationID="disableModule" method="delete" path="/organizations/{organizationId}/stacks/{stackId}/modules" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DisableModule(ctx, operations.DisableModuleRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
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

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.DisableModuleRequest](../../models/operations/disablemodulerequest.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |
| `opts`                                                                             | [][operations.Option](../../models/operations/option.md)                           | :heavy_minus_sign:                                                                 | The options for this request.                                                      |

### Response

**[*operations.DisableModuleResponse](../../models/operations/disablemoduleresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpgradeStack

Upgrade stack

### Example Usage

<!-- UsageSnippet language="go" operationID="upgradeStack" method="put" path="/organizations/{organizationId}/stacks/{stackId}/upgrade" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.UpgradeStack(ctx, operations.UpgradeStackRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
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

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.UpgradeStackRequest](../../models/operations/upgradestackrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.UpgradeStackResponse](../../models/operations/upgradestackresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## GetStack

Find stack

### Example Usage

<!-- UsageSnippet language="go" operationID="getStack" method="get" path="/organizations/{organizationId}/stacks/{stackId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.GetStack(ctx, operations.GetStackRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadStackResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                | Type                                                                     | Required                                                                 | Description                                                              |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| `ctx`                                                                    | [context.Context](https://pkg.go.dev/context#Context)                    | :heavy_check_mark:                                                       | The context to use for the request.                                      |
| `request`                                                                | [operations.GetStackRequest](../../models/operations/getstackrequest.md) | :heavy_check_mark:                                                       | The request object to use for the request.                               |
| `opts`                                                                   | [][operations.Option](../../models/operations/option.md)                 | :heavy_minus_sign:                                                       | The options for this request.                                            |

### Response

**[*operations.GetStackResponse](../../models/operations/getstackresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpdateStack

Update stack

### Example Usage

<!-- UsageSnippet language="go" operationID="updateStack" method="put" path="/organizations/{organizationId}/stacks/{stackId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.UpdateStack(ctx, operations.UpdateStackRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadStackResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.UpdateStackRequest](../../models/operations/updatestackrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |

### Response

**[*operations.UpdateStackResponse](../../models/operations/updatestackresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteStack

Delete stack

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteStack" method="delete" path="/organizations/{organizationId}/stacks/{stackId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteStack(ctx, operations.DeleteStackRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
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

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.DeleteStackRequest](../../models/operations/deletestackrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |

### Response

**[*operations.DeleteStackResponse](../../models/operations/deletestackresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListStackUsersAccesses

List stack users accesses within an organization

### Example Usage

<!-- UsageSnippet language="go" operationID="listStackUsersAccesses" method="get" path="/organizations/{organizationId}/stacks/{stackId}/users" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListStackUsersAccesses(ctx, operations.ListStackUsersAccessesRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.StackUserAccessResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                            | Type                                                                                                 | Required                                                                                             | Description                                                                                          |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                                | :heavy_check_mark:                                                                                   | The context to use for the request.                                                                  |
| `request`                                                                                            | [operations.ListStackUsersAccessesRequest](../../models/operations/liststackusersaccessesrequest.md) | :heavy_check_mark:                                                                                   | The request object to use for the request.                                                           |
| `opts`                                                                                               | [][operations.Option](../../models/operations/option.md)                                             | :heavy_minus_sign:                                                                                   | The options for this request.                                                                        |

### Response

**[*operations.ListStackUsersAccessesResponse](../../models/operations/liststackusersaccessesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadStackUserAccess

Read stack user access within an organization

### Example Usage

<!-- UsageSnippet language="go" operationID="readStackUserAccess" method="get" path="/organizations/{organizationId}/stacks/{stackId}/users/{userId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ReadStackUserAccess(ctx, operations.ReadStackUserAccessRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
        UserID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadStackUserAccess != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                      | Type                                                                                           | Required                                                                                       | Description                                                                                    |
| ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `ctx`                                                                                          | [context.Context](https://pkg.go.dev/context#Context)                                          | :heavy_check_mark:                                                                             | The context to use for the request.                                                            |
| `request`                                                                                      | [operations.ReadStackUserAccessRequest](../../models/operations/readstackuseraccessrequest.md) | :heavy_check_mark:                                                                             | The request object to use for the request.                                                     |
| `opts`                                                                                         | [][operations.Option](../../models/operations/option.md)                                       | :heavy_minus_sign:                                                                             | The options for this request.                                                                  |

### Response

**[*operations.ReadStackUserAccessResponse](../../models/operations/readstackuseraccessresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteStackUserAccess

Delete stack user access within an organization

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteStackUserAccess" method="delete" path="/organizations/{organizationId}/stacks/{stackId}/users/{userId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteStackUserAccess(ctx, operations.DeleteStackUserAccessRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
        UserID: "<id>",
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

| Parameter                                                                                          | Type                                                                                               | Required                                                                                           | Description                                                                                        |
| -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                              | [context.Context](https://pkg.go.dev/context#Context)                                              | :heavy_check_mark:                                                                                 | The context to use for the request.                                                                |
| `request`                                                                                          | [operations.DeleteStackUserAccessRequest](../../models/operations/deletestackuseraccessrequest.md) | :heavy_check_mark:                                                                                 | The request object to use for the request.                                                         |
| `opts`                                                                                             | [][operations.Option](../../models/operations/option.md)                                           | :heavy_minus_sign:                                                                                 | The options for this request.                                                                      |

### Response

**[*operations.DeleteStackUserAccessResponse](../../models/operations/deletestackuseraccessresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpsertStackUserAccess

Update stack user access within an organization

### Example Usage

<!-- UsageSnippet language="go" operationID="upsertStackUserAccess" method="put" path="/organizations/{organizationId}/stacks/{stackId}/users/{userId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.UpsertStackUserAccess(ctx, operations.UpsertStackUserAccessRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
        UserID: "<id>",
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

| Parameter                                                                                          | Type                                                                                               | Required                                                                                           | Description                                                                                        |
| -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                              | [context.Context](https://pkg.go.dev/context#Context)                                              | :heavy_check_mark:                                                                                 | The context to use for the request.                                                                |
| `request`                                                                                          | [operations.UpsertStackUserAccessRequest](../../models/operations/upsertstackuseraccessrequest.md) | :heavy_check_mark:                                                                                 | The request object to use for the request.                                                         |
| `opts`                                                                                             | [][operations.Option](../../models/operations/option.md)                                           | :heavy_minus_sign:                                                                                 | The options for this request.                                                                      |

### Response

**[*operations.UpsertStackUserAccessResponse](../../models/operations/upsertstackuseraccessresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DisableStack

Disable stack

### Example Usage

<!-- UsageSnippet language="go" operationID="disableStack" method="put" path="/organizations/{organizationId}/stacks/{stackId}/disable" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DisableStack(ctx, operations.DisableStackRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
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

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.DisableStackRequest](../../models/operations/disablestackrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.DisableStackResponse](../../models/operations/disablestackresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## EnableStack

Enable stack

### Example Usage

<!-- UsageSnippet language="go" operationID="enableStack" method="put" path="/organizations/{organizationId}/stacks/{stackId}/enable" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.EnableStack(ctx, operations.EnableStackRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
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

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.EnableStackRequest](../../models/operations/enablestackrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |

### Response

**[*operations.EnableStackResponse](../../models/operations/enablestackresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## RestoreStack

Restore stack

### Example Usage

<!-- UsageSnippet language="go" operationID="restoreStack" method="put" path="/organizations/{organizationId}/stacks/{stackId}/restore" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.RestoreStack(ctx, operations.RestoreStackRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadStackResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.RestoreStackRequest](../../models/operations/restorestackrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.RestoreStackResponse](../../models/operations/restorestackresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## EnableStargate

Enable stargate on a stack

### Example Usage

<!-- UsageSnippet language="go" operationID="enableStargate" method="put" path="/organizations/{organizationId}/stacks/{stackId}/stargate/enable" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.EnableStargate(ctx, operations.EnableStargateRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
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

| Parameter                                                                            | Type                                                                                 | Required                                                                             | Description                                                                          |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| `ctx`                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                | :heavy_check_mark:                                                                   | The context to use for the request.                                                  |
| `request`                                                                            | [operations.EnableStargateRequest](../../models/operations/enablestargaterequest.md) | :heavy_check_mark:                                                                   | The request object to use for the request.                                           |
| `opts`                                                                               | [][operations.Option](../../models/operations/option.md)                             | :heavy_minus_sign:                                                                   | The options for this request.                                                        |

### Response

**[*operations.EnableStargateResponse](../../models/operations/enablestargateresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DisableStargate

Disable stargate on a stack

### Example Usage

<!-- UsageSnippet language="go" operationID="disableStargate" method="put" path="/organizations/{organizationId}/stacks/{stackId}/stargate/disable" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DisableStargate(ctx, operations.DisableStargateRequest{
        OrganizationID: "<id>",
        StackID: "<id>",
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

| Parameter                                                                              | Type                                                                                   | Required                                                                               | Description                                                                            |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `ctx`                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                  | :heavy_check_mark:                                                                     | The context to use for the request.                                                    |
| `request`                                                                              | [operations.DisableStargateRequest](../../models/operations/disablestargaterequest.md) | :heavy_check_mark:                                                                     | The request object to use for the request.                                             |
| `opts`                                                                                 | [][operations.Option](../../models/operations/option.md)                               | :heavy_minus_sign:                                                                     | The options for this request.                                                          |

### Response

**[*operations.DisableStargateResponse](../../models/operations/disablestargateresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListInvitations

List invitations of the user

### Example Usage

<!-- UsageSnippet language="go" operationID="listInvitations" method="get" path="/me/invitations" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListInvitations(ctx, operations.ListInvitationsRequest{})
    if err != nil {
        log.Fatal(err)
    }
    if res.ListInvitationsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                              | Type                                                                                   | Required                                                                               | Description                                                                            |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `ctx`                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                  | :heavy_check_mark:                                                                     | The context to use for the request.                                                    |
| `request`                                                                              | [operations.ListInvitationsRequest](../../models/operations/listinvitationsrequest.md) | :heavy_check_mark:                                                                     | The request object to use for the request.                                             |
| `opts`                                                                                 | [][operations.Option](../../models/operations/option.md)                               | :heavy_minus_sign:                                                                     | The options for this request.                                                          |

### Response

**[*operations.ListInvitationsResponse](../../models/operations/listinvitationsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## AcceptInvitation

Accept invitation

### Example Usage

<!-- UsageSnippet language="go" operationID="acceptInvitation" method="post" path="/me/invitations/{invitationId}/accept" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.AcceptInvitation(ctx, operations.AcceptInvitationRequest{
        InvitationID: "<id>",
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

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `request`                                                                                | [operations.AcceptInvitationRequest](../../models/operations/acceptinvitationrequest.md) | :heavy_check_mark:                                                                       | The request object to use for the request.                                               |
| `opts`                                                                                   | [][operations.Option](../../models/operations/option.md)                                 | :heavy_minus_sign:                                                                       | The options for this request.                                                            |

### Response

**[*operations.AcceptInvitationResponse](../../models/operations/acceptinvitationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeclineInvitation

Decline invitation

### Example Usage

<!-- UsageSnippet language="go" operationID="declineInvitation" method="post" path="/me/invitations/{invitationId}/reject" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeclineInvitation(ctx, operations.DeclineInvitationRequest{
        InvitationID: "<id>",
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

| Parameter                                                                                  | Type                                                                                       | Required                                                                                   | Description                                                                                |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| `ctx`                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                      | :heavy_check_mark:                                                                         | The context to use for the request.                                                        |
| `request`                                                                                  | [operations.DeclineInvitationRequest](../../models/operations/declineinvitationrequest.md) | :heavy_check_mark:                                                                         | The request object to use for the request.                                                 |
| `opts`                                                                                     | [][operations.Option](../../models/operations/option.md)                                   | :heavy_minus_sign:                                                                         | The options for this request.                                                              |

### Response

**[*operations.DeclineInvitationResponse](../../models/operations/declineinvitationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListOrganizationInvitations

List invitations of the organization

### Example Usage

<!-- UsageSnippet language="go" operationID="listOrganizationInvitations" method="get" path="/organizations/{organizationId}/invitations" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListOrganizationInvitations(ctx, operations.ListOrganizationInvitationsRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ListInvitationsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                      | Type                                                                                                           | Required                                                                                                       | Description                                                                                                    |
| -------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                          | [context.Context](https://pkg.go.dev/context#Context)                                                          | :heavy_check_mark:                                                                                             | The context to use for the request.                                                                            |
| `request`                                                                                                      | [operations.ListOrganizationInvitationsRequest](../../models/operations/listorganizationinvitationsrequest.md) | :heavy_check_mark:                                                                                             | The request object to use for the request.                                                                     |
| `opts`                                                                                                         | [][operations.Option](../../models/operations/option.md)                                                       | :heavy_minus_sign:                                                                                             | The options for this request.                                                                                  |

### Response

**[*operations.ListOrganizationInvitationsResponse](../../models/operations/listorganizationinvitationsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateInvitation

Create invitation

### Example Usage

<!-- UsageSnippet language="go" operationID="createInvitation" method="post" path="/organizations/{organizationId}/invitations" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.CreateInvitation(ctx, operations.CreateInvitationRequest{
        OrganizationID: "<id>",
        Email: "Manley_Hoeger@hotmail.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateInvitationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `request`                                                                                | [operations.CreateInvitationRequest](../../models/operations/createinvitationrequest.md) | :heavy_check_mark:                                                                       | The request object to use for the request.                                               |
| `opts`                                                                                   | [][operations.Option](../../models/operations/option.md)                                 | :heavy_minus_sign:                                                                       | The options for this request.                                                            |

### Response

**[*operations.CreateInvitationResponse](../../models/operations/createinvitationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteInvitation

Delete invitation

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteInvitation" method="delete" path="/organizations/{organizationId}/invitations/{invitationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteInvitation(ctx, operations.DeleteInvitationRequest{
        OrganizationID: "<id>",
        InvitationID: "<id>",
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

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `request`                                                                                | [operations.DeleteInvitationRequest](../../models/operations/deleteinvitationrequest.md) | :heavy_check_mark:                                                                       | The request object to use for the request.                                               |
| `opts`                                                                                   | [][operations.Option](../../models/operations/option.md)                                 | :heavy_minus_sign:                                                                       | The options for this request.                                                            |

### Response

**[*operations.DeleteInvitationResponse](../../models/operations/deleteinvitationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListRegions

List regions

### Example Usage

<!-- UsageSnippet language="go" operationID="listRegions" method="get" path="/organizations/{organizationId}/regions" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListRegions(ctx, operations.ListRegionsRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ListRegionsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.ListRegionsRequest](../../models/operations/listregionsrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |

### Response

**[*operations.ListRegionsResponse](../../models/operations/listregionsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreatePrivateRegion

Create a private region

### Example Usage

<!-- UsageSnippet language="go" operationID="createPrivateRegion" method="post" path="/organizations/{organizationId}/regions" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.CreatePrivateRegion(ctx, operations.CreatePrivateRegionRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.CreatedPrivateRegionResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                      | Type                                                                                           | Required                                                                                       | Description                                                                                    |
| ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `ctx`                                                                                          | [context.Context](https://pkg.go.dev/context#Context)                                          | :heavy_check_mark:                                                                             | The context to use for the request.                                                            |
| `request`                                                                                      | [operations.CreatePrivateRegionRequest](../../models/operations/createprivateregionrequest.md) | :heavy_check_mark:                                                                             | The request object to use for the request.                                                     |
| `opts`                                                                                         | [][operations.Option](../../models/operations/option.md)                                       | :heavy_minus_sign:                                                                             | The options for this request.                                                                  |

### Response

**[*operations.CreatePrivateRegionResponse](../../models/operations/createprivateregionresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## GetRegion

Get region

### Example Usage

<!-- UsageSnippet language="go" operationID="getRegion" method="get" path="/organizations/{organizationId}/regions/{regionID}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.GetRegion(ctx, operations.GetRegionRequest{
        OrganizationID: "<id>",
        RegionID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.GetRegionResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                  | Type                                                                       | Required                                                                   | Description                                                                |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `ctx`                                                                      | [context.Context](https://pkg.go.dev/context#Context)                      | :heavy_check_mark:                                                         | The context to use for the request.                                        |
| `request`                                                                  | [operations.GetRegionRequest](../../models/operations/getregionrequest.md) | :heavy_check_mark:                                                         | The request object to use for the request.                                 |
| `opts`                                                                     | [][operations.Option](../../models/operations/option.md)                   | :heavy_minus_sign:                                                         | The options for this request.                                              |

### Response

**[*operations.GetRegionResponse](../../models/operations/getregionresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteRegion

Delete region

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteRegion" method="delete" path="/organizations/{organizationId}/regions/{regionID}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteRegion(ctx, operations.DeleteRegionRequest{
        OrganizationID: "<id>",
        RegionID: "<id>",
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

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.DeleteRegionRequest](../../models/operations/deleteregionrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.DeleteRegionResponse](../../models/operations/deleteregionresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## GetRegionVersions

Get region versions

### Example Usage

<!-- UsageSnippet language="go" operationID="getRegionVersions" method="get" path="/organizations/{organizationId}/regions/{regionID}/versions" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.GetRegionVersions(ctx, operations.GetRegionVersionsRequest{
        OrganizationID: "<id>",
        RegionID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.GetRegionVersionsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                  | Type                                                                                       | Required                                                                                   | Description                                                                                |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| `ctx`                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                      | :heavy_check_mark:                                                                         | The context to use for the request.                                                        |
| `request`                                                                                  | [operations.GetRegionVersionsRequest](../../models/operations/getregionversionsrequest.md) | :heavy_check_mark:                                                                         | The request object to use for the request.                                                 |
| `opts`                                                                                     | [][operations.Option](../../models/operations/option.md)                                   | :heavy_minus_sign:                                                                         | The options for this request.                                                              |

### Response

**[*operations.GetRegionVersionsResponse](../../models/operations/getregionversionsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListOrganizationApplications

List applications enabled for organization

### Example Usage

<!-- UsageSnippet language="go" operationID="listOrganizationApplications" method="get" path="/organizations/{organizationId}/applications" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListOrganizationApplications(ctx, operations.ListOrganizationApplicationsRequest{
        OrganizationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.ListApplicationsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                        | Type                                                                                                             | Required                                                                                                         | Description                                                                                                      |
| ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                            | [context.Context](https://pkg.go.dev/context#Context)                                                            | :heavy_check_mark:                                                                                               | The context to use for the request.                                                                              |
| `request`                                                                                                        | [operations.ListOrganizationApplicationsRequest](../../models/operations/listorganizationapplicationsrequest.md) | :heavy_check_mark:                                                                                               | The request object to use for the request.                                                                       |
| `opts`                                                                                                           | [][operations.Option](../../models/operations/option.md)                                                         | :heavy_minus_sign:                                                                                               | The options for this request.                                                                                    |

### Response

**[*operations.ListOrganizationApplicationsResponse](../../models/operations/listorganizationapplicationsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## GetOrganizationApplication

Get application for organization

### Example Usage

<!-- UsageSnippet language="go" operationID="getOrganizationApplication" method="get" path="/organizations/{organizationId}/applications/{applicationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.GetOrganizationApplication(ctx, operations.GetOrganizationApplicationRequest{
        OrganizationID: "<id>",
        ApplicationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.GetApplicationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                    | Type                                                                                                         | Required                                                                                                     | Description                                                                                                  |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                                        | :heavy_check_mark:                                                                                           | The context to use for the request.                                                                          |
| `request`                                                                                                    | [operations.GetOrganizationApplicationRequest](../../models/operations/getorganizationapplicationrequest.md) | :heavy_check_mark:                                                                                           | The request object to use for the request.                                                                   |
| `opts`                                                                                                       | [][operations.Option](../../models/operations/option.md)                                                     | :heavy_minus_sign:                                                                                           | The options for this request.                                                                                |

### Response

**[*operations.GetOrganizationApplicationResponse](../../models/operations/getorganizationapplicationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## EnableApplicationForOrganization

Enable application for organization

### Example Usage

<!-- UsageSnippet language="go" operationID="enableApplicationForOrganization" method="put" path="/organizations/{organizationId}/applications/{applicationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.EnableApplicationForOrganization(ctx, operations.EnableApplicationForOrganizationRequest{
        OrganizationID: "<id>",
        ApplicationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.EnableApplicationForOrganizationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                | Type                                                                                                                     | Required                                                                                                                 | Description                                                                                                              |
| ------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                    | :heavy_check_mark:                                                                                                       | The context to use for the request.                                                                                      |
| `request`                                                                                                                | [operations.EnableApplicationForOrganizationRequest](../../models/operations/enableapplicationfororganizationrequest.md) | :heavy_check_mark:                                                                                                       | The request object to use for the request.                                                                               |
| `opts`                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                 | :heavy_minus_sign:                                                                                                       | The options for this request.                                                                                            |

### Response

**[*operations.EnableApplicationForOrganizationResponse](../../models/operations/enableapplicationfororganizationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DisableApplicationForOrganization

Disable application for organization

### Example Usage

<!-- UsageSnippet language="go" operationID="disableApplicationForOrganization" method="delete" path="/organizations/{organizationId}/applications/{applicationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DisableApplicationForOrganization(ctx, operations.DisableApplicationForOrganizationRequest{
        OrganizationID: "<id>",
        ApplicationID: "<id>",
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

| Parameter                                                                                                                  | Type                                                                                                                       | Required                                                                                                                   | Description                                                                                                                |
| -------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                                                      | :heavy_check_mark:                                                                                                         | The context to use for the request.                                                                                        |
| `request`                                                                                                                  | [operations.DisableApplicationForOrganizationRequest](../../models/operations/disableapplicationfororganizationrequest.md) | :heavy_check_mark:                                                                                                         | The request object to use for the request.                                                                                 |
| `opts`                                                                                                                     | [][operations.Option](../../models/operations/option.md)                                                                   | :heavy_minus_sign:                                                                                                         | The options for this request.                                                                                              |

### Response

**[*operations.DisableApplicationForOrganizationResponse](../../models/operations/disableapplicationfororganizationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ListApplications

List applications

### Example Usage

<!-- UsageSnippet language="go" operationID="listApplications" method="get" path="/applications" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ListApplications(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListApplicationsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ListApplicationsResponse](../../models/operations/listapplicationsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateApplication

Create application

### Example Usage

<!-- UsageSnippet language="go" operationID="createApplication" method="post" path="/applications" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.CreateApplication(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateApplicationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                | Type                                                                     | Required                                                                 | Description                                                              |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| `ctx`                                                                    | [context.Context](https://pkg.go.dev/context#Context)                    | :heavy_check_mark:                                                       | The context to use for the request.                                      |
| `request`                                                                | [components.ApplicationData](../../models/components/applicationdata.md) | :heavy_check_mark:                                                       | The request object to use for the request.                               |
| `opts`                                                                   | [][operations.Option](../../models/operations/option.md)                 | :heavy_minus_sign:                                                       | The options for this request.                                            |

### Response

**[*operations.CreateApplicationResponse](../../models/operations/createapplicationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## GetApplication

Get application

### Example Usage

<!-- UsageSnippet language="go" operationID="getApplication" method="get" path="/applications/{applicationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.GetApplication(ctx, operations.GetApplicationRequest{
        ApplicationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.GetApplicationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                            | Type                                                                                 | Required                                                                             | Description                                                                          |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| `ctx`                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                | :heavy_check_mark:                                                                   | The context to use for the request.                                                  |
| `request`                                                                            | [operations.GetApplicationRequest](../../models/operations/getapplicationrequest.md) | :heavy_check_mark:                                                                   | The request object to use for the request.                                           |
| `opts`                                                                               | [][operations.Option](../../models/operations/option.md)                             | :heavy_minus_sign:                                                                   | The options for this request.                                                        |

### Response

**[*operations.GetApplicationResponse](../../models/operations/getapplicationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## UpdateApplication

Update application

### Example Usage

<!-- UsageSnippet language="go" operationID="updateApplication" method="put" path="/applications/{applicationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.UpdateApplication(ctx, operations.UpdateApplicationRequest{
        ApplicationID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.UpdateApplicationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                  | Type                                                                                       | Required                                                                                   | Description                                                                                |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| `ctx`                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                      | :heavy_check_mark:                                                                         | The context to use for the request.                                                        |
| `request`                                                                                  | [operations.UpdateApplicationRequest](../../models/operations/updateapplicationrequest.md) | :heavy_check_mark:                                                                         | The request object to use for the request.                                                 |
| `opts`                                                                                     | [][operations.Option](../../models/operations/option.md)                                   | :heavy_minus_sign:                                                                         | The options for this request.                                                              |

### Response

**[*operations.UpdateApplicationResponse](../../models/operations/updateapplicationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteApplication

Delete application

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteApplication" method="delete" path="/applications/{applicationId}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteApplication(ctx, operations.DeleteApplicationRequest{
        ApplicationID: "<id>",
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

| Parameter                                                                                  | Type                                                                                       | Required                                                                                   | Description                                                                                |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| `ctx`                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                      | :heavy_check_mark:                                                                         | The context to use for the request.                                                        |
| `request`                                                                                  | [operations.DeleteApplicationRequest](../../models/operations/deleteapplicationrequest.md) | :heavy_check_mark:                                                                         | The request object to use for the request.                                                 |
| `opts`                                                                                     | [][operations.Option](../../models/operations/option.md)                                   | :heavy_minus_sign:                                                                         | The options for this request.                                                              |

### Response

**[*operations.DeleteApplicationResponse](../../models/operations/deleteapplicationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateApplicationScope

Create application scope

### Example Usage

<!-- UsageSnippet language="go" operationID="createApplicationScope" method="post" path="/applications/{applicationId}/scopes" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.CreateApplicationScope(ctx, operations.CreateApplicationScopeRequest{
        ApplicationID: "550e8400-e29b-41d4-a716-446655440000",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateApplicationScopeResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                            | Type                                                                                                 | Required                                                                                             | Description                                                                                          |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                                | :heavy_check_mark:                                                                                   | The context to use for the request.                                                                  |
| `request`                                                                                            | [operations.CreateApplicationScopeRequest](../../models/operations/createapplicationscoperequest.md) | :heavy_check_mark:                                                                                   | The request object to use for the request.                                                           |
| `opts`                                                                                               | [][operations.Option](../../models/operations/option.md)                                             | :heavy_minus_sign:                                                                                   | The options for this request.                                                                        |

### Response

**[*operations.CreateApplicationScopeResponse](../../models/operations/createapplicationscoperesponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## DeleteApplicationScope

Delete a specific scope from an application. This operation requires system administrator privileges.

### Example Usage

<!-- UsageSnippet language="go" operationID="deleteApplicationScope" method="delete" path="/applications/{applicationId}/scopes/{scopeID}" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.DeleteApplicationScope(ctx, operations.DeleteApplicationScopeRequest{
        ApplicationID: "550e8400-e29b-41d4-a716-446655440000",
        ScopeID: 115177,
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

| Parameter                                                                                            | Type                                                                                                 | Required                                                                                             | Description                                                                                          |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                                | :heavy_check_mark:                                                                                   | The context to use for the request.                                                                  |
| `request`                                                                                            | [operations.DeleteApplicationScopeRequest](../../models/operations/deleteapplicationscoperequest.md) | :heavy_check_mark:                                                                                   | The request object to use for the request.                                                           |
| `opts`                                                                                               | [][operations.Option](../../models/operations/option.md)                                             | :heavy_minus_sign:                                                                                   | The options for this request.                                                                        |

### Response

**[*operations.DeleteApplicationScopeResponse](../../models/operations/deleteapplicationscoperesponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.Error    | 400, 404           | application/json   |
| apierrors.Error    | 500                | application/json   |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## CreateUser

Create a new user in the system. This operation requires system administrator privileges.

### Example Usage

<!-- UsageSnippet language="go" operationID="createUser" method="post" path="/users" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.CreateUser(ctx, components.CreateUserRequest{
        Email: "user@example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateUserResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                    | Type                                                                         | Required                                                                     | Description                                                                  |
| ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `ctx`                                                                        | [context.Context](https://pkg.go.dev/context#Context)                        | :heavy_check_mark:                                                           | The context to use for the request.                                          |
| `request`                                                                    | [components.CreateUserRequest](../../models/components/createuserrequest.md) | :heavy_check_mark:                                                           | The request object to use for the request.                                   |
| `opts`                                                                       | [][operations.Option](../../models/operations/option.md)                     | :heavy_minus_sign:                                                           | The options for this request.                                                |

### Response

**[*operations.CreateUserResponse](../../models/operations/createuserresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.Error    | 400                | application/json   |
| apierrors.Error    | 500                | application/json   |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |

## ReadConnectedUser

Read user

### Example Usage

<!-- UsageSnippet language="go" operationID="readConnectedUser" method="get" path="/me" -->
```go
package main

import(
	"context"
	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"log"
)

func main() {
    ctx := context.Background()

    s := membershipclient.New(
        membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
    )

    res, err := s.ReadConnectedUser(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadUserResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.ReadConnectedUserResponse](../../models/operations/readconnecteduserresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| apierrors.APIError | 4XX, 5XX           | \*/\*              |