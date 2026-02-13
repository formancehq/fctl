# undefined

Developer-friendly & type-safe Go SDK specifically catered to leverage *undefined* API.

<div align="left" style="margin-bottom: 0;">
    <a href="https://www.speakeasy.com/?utm_source=undefined&utm_campaign=go" class="badge-link">
        <span class="badge-container">
            <span class="badge-icon-section">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 30 30" fill="none" style="vertical-align: middle;"><title>Speakeasy Logo</title><path fill="currentColor" d="m20.639 27.548-19.17-2.724L0 26.1l20.639 2.931 8.456-7.336-1.468-.208-6.988 6.062Z"></path><path fill="currentColor" d="m20.639 23.1 8.456-7.336-1.468-.207-6.988 6.06-6.84-.972-9.394-1.333-2.936-.417L0 20.169l2.937.416L0 23.132l20.639 2.931 8.456-7.334-1.468-.208-6.986 6.062-9.78-1.39 1.468-1.273 8.31 1.18Z"></path><path fill="currentColor" d="m20.639 18.65-19.17-2.724L0 17.201l20.639 2.931 8.456-7.334-1.468-.208-6.988 6.06Z"></path><path fill="currentColor" d="M27.627 6.658 24.69 9.205 20.64 12.72l-7.923-1.126L1.469 9.996 0 11.271l11.246 1.596-1.467 1.275-8.311-1.181L0 14.235l20.639 2.932 8.456-7.334-2.937-.418 2.937-2.549-1.468-.208Z"></path><path fill="currentColor" d="M29.095 3.902 8.456.971 0 8.305l20.639 2.934 8.456-7.337Z"></path></svg>
            </span>
            <span class="badge-text badge-text-section">BUILT BY SPEAKEASY</span>
        </span>
    </a>
    <a href="https://opensource.org/licenses/MIT" class="badge-link">
        <span class="badge-container blue">
            <span class="badge-text badge-text-section">LICENSE // MIT</span>
        </span>
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
* [undefined](#undefined)
  * [SDK Installation](#sdk-installation)
  * [SDK Example Usage](#sdk-example-usage)
  * [Authentication](#authentication)
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
go get github.com/formancehq/fctl/internal/membershipclient
```
<!-- End SDK Installation [installation] -->

<!-- Start SDK Example Usage [usage] -->
## SDK Example Usage

### Example

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

<!-- Start Authentication [security] -->
## Authentication

### Per-Client Security Schemes

This SDK supports the following security scheme globally:

| Name     | Type   | Scheme       |
| -------- | ------ | ------------ |
| `Oauth2` | oauth2 | OAuth2 token |

You can configure it using the `WithSecurity` option when initializing the SDK client instance. For example:
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
<!-- End Authentication [security] -->

<!-- Start Available Resources and Operations [operations] -->
## Available Resources and Operations

<details open>
<summary>Available methods</summary>

### [SDK](docs/sdks/sdk/README.md)

* [GetServerInfo](docs/sdks/sdk/README.md#getserverinfo) - Get server info
* [ListOrganizations](docs/sdks/sdk/README.md#listorganizations) - List organizations of the connected user
* [CreateOrganization](docs/sdks/sdk/README.md#createorganization) - Create organization
* [~~ListOrganizationsExpanded~~](docs/sdks/sdk/README.md#listorganizationsexpanded) - List organizations of the connected user with expanded data :warning: **Deprecated**
* [ReadOrganization](docs/sdks/sdk/README.md#readorganization) - Read organization
* [UpdateOrganization](docs/sdks/sdk/README.md#updateorganization) - Update organization
* [DeleteOrganization](docs/sdks/sdk/README.md#deleteorganization) - Delete organization
* [ReadAuthenticationProvider](docs/sdks/sdk/README.md#readauthenticationprovider) - Read authentication provider
* [UpsertAuthenticationProvider](docs/sdks/sdk/README.md#upsertauthenticationprovider) - Upsert an authentication provider
* [DeleteAuthenticationProvider](docs/sdks/sdk/README.md#deleteauthenticationprovider) - Delete authentication provider
* [ListFeatures](docs/sdks/sdk/README.md#listfeatures) - List features
* [AddFeatures](docs/sdks/sdk/README.md#addfeatures) - Add Features
* [DeleteFeature](docs/sdks/sdk/README.md#deletefeature) - Delete feature
* [~~ReadOrganizationClient~~](docs/sdks/sdk/README.md#readorganizationclient) - Read organization client (DEPRECATED) (until 12/31/2025) :warning: **Deprecated**
* [~~CreateOrganizationClient~~](docs/sdks/sdk/README.md#createorganizationclient) - Create organization client (DEPRECATED) (until 12/31/2025) :warning: **Deprecated**
* [~~DeleteOrganizationClient~~](docs/sdks/sdk/README.md#deleteorganizationclient) - Delete organization client (DEPRECATED) (until 12/31/2025) :warning: **Deprecated**
* [OrganizationClientsRead](docs/sdks/sdk/README.md#organizationclientsread) - Read organization clients
* [OrganizationClientCreate](docs/sdks/sdk/README.md#organizationclientcreate) - Create organization client
* [OrganizationClientRead](docs/sdks/sdk/README.md#organizationclientread) - Read organization client
* [OrganizationClientDelete](docs/sdks/sdk/README.md#organizationclientdelete) - Delete organization client
* [OrganizationClientUpdate](docs/sdks/sdk/README.md#organizationclientupdate) - Update organization client
* [ListLogs](docs/sdks/sdk/README.md#listlogs) - List logs
* [ListUsersOfOrganization](docs/sdks/sdk/README.md#listusersoforganization) - List users of organization
* [ReadUserOfOrganization](docs/sdks/sdk/README.md#readuseroforganization) - Read user of organization
* [UpsertOrganizationUser](docs/sdks/sdk/README.md#upsertorganizationuser) - Update user within an organization
* [DeleteUserFromOrganization](docs/sdks/sdk/README.md#deleteuserfromorganization) - delete user from organization
* [ListPolicies](docs/sdks/sdk/README.md#listpolicies) - List policies of organization
* [CreatePolicy](docs/sdks/sdk/README.md#createpolicy) - Create policy
* [ReadPolicy](docs/sdks/sdk/README.md#readpolicy) - Read policy with scopes
* [UpdatePolicy](docs/sdks/sdk/README.md#updatepolicy) - Update policy
* [DeletePolicy](docs/sdks/sdk/README.md#deletepolicy) - Delete policy
* [AddScopeToPolicy](docs/sdks/sdk/README.md#addscopetopolicy) - Add scope to policy
* [RemoveScopeFromPolicy](docs/sdks/sdk/README.md#removescopefrompolicy) - Remove scope from policy
* [ListStacks](docs/sdks/sdk/README.md#liststacks) - List stacks
* [CreateStack](docs/sdks/sdk/README.md#createstack) - Create stack
* [ListModules](docs/sdks/sdk/README.md#listmodules) - List modules of a stack
* [EnableModule](docs/sdks/sdk/README.md#enablemodule) - enable module
* [DisableModule](docs/sdks/sdk/README.md#disablemodule) - disable module
* [UpgradeStack](docs/sdks/sdk/README.md#upgradestack) - Upgrade stack
* [GetStack](docs/sdks/sdk/README.md#getstack) - Find stack
* [UpdateStack](docs/sdks/sdk/README.md#updatestack) - Update stack
* [DeleteStack](docs/sdks/sdk/README.md#deletestack) - Delete stack
* [ListStackUsersAccesses](docs/sdks/sdk/README.md#liststackusersaccesses) - List stack users accesses within an organization
* [ReadStackUserAccess](docs/sdks/sdk/README.md#readstackuseraccess) - Read stack user access within an organization
* [DeleteStackUserAccess](docs/sdks/sdk/README.md#deletestackuseraccess) - Delete stack user access within an organization
* [UpsertStackUserAccess](docs/sdks/sdk/README.md#upsertstackuseraccess) - Update stack user access within an organization
* [DisableStack](docs/sdks/sdk/README.md#disablestack) - Disable stack
* [EnableStack](docs/sdks/sdk/README.md#enablestack) - Enable stack
* [RestoreStack](docs/sdks/sdk/README.md#restorestack) - Restore stack
* [EnableStargate](docs/sdks/sdk/README.md#enablestargate) - Enable stargate on a stack
* [DisableStargate](docs/sdks/sdk/README.md#disablestargate) - Disable stargate on a stack
* [ListInvitations](docs/sdks/sdk/README.md#listinvitations) - List invitations of the user
* [AcceptInvitation](docs/sdks/sdk/README.md#acceptinvitation) - Accept invitation
* [DeclineInvitation](docs/sdks/sdk/README.md#declineinvitation) - Decline invitation
* [ListOrganizationInvitations](docs/sdks/sdk/README.md#listorganizationinvitations) - List invitations of the organization
* [CreateInvitation](docs/sdks/sdk/README.md#createinvitation) - Create invitation
* [DeleteInvitation](docs/sdks/sdk/README.md#deleteinvitation) - Delete invitation
* [ListRegions](docs/sdks/sdk/README.md#listregions) - List regions
* [CreatePrivateRegion](docs/sdks/sdk/README.md#createprivateregion) - Create a private region
* [GetRegion](docs/sdks/sdk/README.md#getregion) - Get region
* [DeleteRegion](docs/sdks/sdk/README.md#deleteregion) - Delete region
* [GetRegionVersions](docs/sdks/sdk/README.md#getregionversions) - Get region versions
* [ListOrganizationApplications](docs/sdks/sdk/README.md#listorganizationapplications) - List applications enabled for organization
* [GetOrganizationApplication](docs/sdks/sdk/README.md#getorganizationapplication) - Get application for organization
* [EnableApplicationForOrganization](docs/sdks/sdk/README.md#enableapplicationfororganization) - Enable application for organization
* [DisableApplicationForOrganization](docs/sdks/sdk/README.md#disableapplicationfororganization) - Disable application for organization
* [ListApplications](docs/sdks/sdk/README.md#listapplications) - List applications
* [CreateApplication](docs/sdks/sdk/README.md#createapplication) - Create application
* [GetApplication](docs/sdks/sdk/README.md#getapplication) - Get application
* [UpdateApplication](docs/sdks/sdk/README.md#updateapplication) - Update application
* [DeleteApplication](docs/sdks/sdk/README.md#deleteapplication) - Delete application
* [CreateApplicationScope](docs/sdks/sdk/README.md#createapplicationscope) - Create application scope
* [DeleteApplicationScope](docs/sdks/sdk/README.md#deleteapplicationscope) - Delete application scope
* [AddDelegationToScope](docs/sdks/sdk/README.md#adddelegationtoscope) - Add delegation to scope
* [RemoveDelegationFromScope](docs/sdks/sdk/README.md#removedelegationfromscope) - Remove delegation from scope
* [CreateApplicationClient](docs/sdks/sdk/README.md#createapplicationclient) - Create application client
* [GetApplicationClient](docs/sdks/sdk/README.md#getapplicationclient) - Get application client
* [UpdateApplicationClient](docs/sdks/sdk/README.md#updateapplicationclient) - Update application client
* [DeleteApplicationClient](docs/sdks/sdk/README.md#deleteapplicationclient) - Delete application client
* [CreateUser](docs/sdks/sdk/README.md#createuser) - Create user
* [ReadConnectedUser](docs/sdks/sdk/README.md#readconnecteduser) - Read user

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
	"github.com/formancehq/fctl/internal/membershipclient"
	"github.com/formancehq/fctl/internal/membershipclient/retry"
	"log"
	"models/operations"
)

func main() {
	ctx := context.Background()

	s := membershipclient.New(
		membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
	)

	res, err := s.GetServerInfo(ctx, operations.WithRetries(
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
	if res.ServerInfo != nil {
		// handle response
	}
}

```

If you'd like to override the default retry strategy for all operations that support retries, you can use the `WithRetryConfig` option at SDK initialization:
```go
package main

import (
	"context"
	"github.com/formancehq/fctl/internal/membershipclient"
	"github.com/formancehq/fctl/internal/membershipclient/retry"
	"log"
)

func main() {
	ctx := context.Background()

	s := membershipclient.New(
		membershipclient.WithRetryConfig(
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
<!-- End Retries [retries] -->

<!-- Start Error Handling [errors] -->
## Error Handling

Handling errors in this SDK should largely match your expectations. All operations return a response object or an error, they will never return both.

By Default, an API error will return `apierrors.APIError`. When custom error responses are specified for an operation, the SDK may also return their associated error. You can refer to respective *Errors* tables in SDK docs for more details on possible error types for each operation.

For example, the `GetServerInfo` function may return the following errors:

| Error Type         | Status Code | Content Type |
| ------------------ | ----------- | ------------ |
| apierrors.APIError | 4XX, 5XX    | \*/\*        |

### Example

```go
package main

import (
	"context"
	"errors"
	"github.com/formancehq/fctl/internal/membershipclient"
	"github.com/formancehq/fctl/internal/membershipclient/models/apierrors"
	"log"
)

func main() {
	ctx := context.Background()

	s := membershipclient.New(
		membershipclient.WithSecurity("<YOUR_OAUTH2_HERE>"),
	)

	res, err := s.GetServerInfo(ctx)
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

### Override Server URL Per-Client

The default server can be overridden globally using the `WithServerURL(serverURL string)` option when initializing the SDK client instance. For example:
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
		membershipclient.WithServerURL("http://localhost:8080"),
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

	"github.com/formancehq/fctl/internal/membershipclient"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	sdkClient  = membershipclient.New(membershipclient.WithClient(httpClient))
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

### SDK Created by [Speakeasy](https://www.speakeasy.com/?utm_source=undefined&utm_campaign=go)

<style>
  :root {
    --badge-gray-bg: #f3f4f6;
    --badge-gray-border: #d1d5db;
    --badge-gray-text: #374151;
    --badge-blue-bg: #eff6ff;
    --badge-blue-border: #3b82f6;
    --badge-blue-text: #3b82f6;
  }

  @media (prefers-color-scheme: dark) {
    :root {
      --badge-gray-bg: #374151;
      --badge-gray-border: #4b5563;
      --badge-gray-text: #f3f4f6;
      --badge-blue-bg: #1e3a8a;
      --badge-blue-border: #3b82f6;
      --badge-blue-text: #93c5fd;
    }
  }
  
  h1 {
    border-bottom: none !important;
    margin-bottom: 4px;
    margin-top: 0;
    letter-spacing: 0.5px;
    font-weight: 600;
  }
  
  .badge-text {
    letter-spacing: 1px;
    font-weight: 300;
  }
  
  .badge-container {
    display: inline-flex;
    align-items: center;
    background: var(--badge-gray-bg);
    border: 1px solid var(--badge-gray-border);
    border-radius: 6px;
    overflow: hidden;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
    font-size: 11px;
    text-decoration: none;
    vertical-align: middle;
  }

  .badge-container.blue {
    background: var(--badge-blue-bg);
    border-color: var(--badge-blue-border);
  }

  .badge-icon-section {
    padding: 4px 8px;
    border-right: 1px solid var(--badge-gray-border);
    display: flex;
    align-items: center;
  }

  .badge-text-section {
    padding: 4px 10px;
    color: var(--badge-gray-text);
    font-weight: 400;
  }

  .badge-container.blue .badge-text-section {
    color: var(--badge-blue-text);
  }
  
  .badge-link {
    text-decoration: none;
    margin-left: 8px;
    display: inline-flex;
    vertical-align: middle;
  }

  .badge-link:hover {
    text-decoration: none;
  }
  
  .badge-link:first-child {
    margin-left: 0;
  }
  
  .badge-icon-section svg {
    color: var(--badge-gray-text);
  }

  .badge-container.blue .badge-icon-section svg {
    color: var(--badge-blue-text);
  }
</style> 