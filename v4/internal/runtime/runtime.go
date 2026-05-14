// Package runtime resolves contexts, targets, authentication, component
// versions, and API version policy for v4 commands.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/formancehq/fctl/v4/internal/auth"
	"github.com/formancehq/fctl/v4/internal/capabilities"
	"github.com/formancehq/fctl/v4/internal/config"
	"github.com/formancehq/fctl/v4/internal/credentials"
)

type TargetKind string

const (
	TargetKindStack      TargetKind = "stack"
	TargetKindCloud      TargetKind = "cloud"
	TargetKindCloudStack TargetKind = "cloud-stack"
)

type Options struct {
	ConfigPath      string
	ContextOverride config.ContextOverride
	Credentials     credentials.Store
	Auth            auth.Options
	VersionsClient  VersionsClient
	Manifest        capabilities.Manifest
	Compatibility   capabilities.ComponentCompatibility
}

type Runtime struct {
	Config      config.Config
	ContextName string
	Context     config.Context
	Target      Target

	Credentials    credentials.Store
	AuthOptions    auth.Options
	VersionsClient VersionsClient
	Manifest       capabilities.Manifest
	Compatibility  capabilities.ComponentCompatibility
}

type Target struct {
	Kind         TargetKind
	URL          string
	Organization string
	Stack        string
}

func New(ctx context.Context, options Options) (*Runtime, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if options.ConfigPath == "" {
		return nil, errors.New("config path is required")
	}

	cfg, err := config.LoadFile(options.ConfigPath)
	if err != nil {
		return nil, err
	}

	contextName, selectedContext, err := config.ResolveCurrentContext(cfg, options.ContextOverride)
	if err != nil {
		return nil, err
	}
	selectedContext, err = applyTargetOverrides(selectedContext, options.ContextOverride)
	if err != nil {
		return nil, err
	}

	target, err := TargetFromContext(selectedContext)
	if err != nil {
		return nil, err
	}

	return &Runtime{
		Config:         cfg,
		ContextName:    contextName,
		Context:        selectedContext,
		Target:         target,
		Credentials:    options.Credentials,
		AuthOptions:    options.Auth,
		VersionsClient: options.VersionsClient,
		Manifest:       options.Manifest,
		Compatibility:  options.Compatibility,
	}, nil
}

func applyTargetOverrides(selectedContext config.Context, override config.ContextOverride) (config.Context, error) {
	if override.Organization == "" && override.Stack == "" {
		return selectedContext, nil
	}
	if selectedContext.Kind == config.ContextKindStack {
		return config.Context{}, fmt.Errorf("--organization and --stack can only be used with Cloud or EE profiles")
	}
	if override.Organization != "" {
		selectedContext.Organization = override.Organization
	}
	if override.Stack != "" {
		selectedContext.Stack = override.Stack
		if selectedContext.Kind == config.ContextKindCloud {
			selectedContext.Kind = config.ContextKindCloudStack
		}
	}
	return selectedContext, nil
}

func TargetFromContext(context config.Context) (Target, error) {
	switch context.Kind {
	case config.ContextKindStack:
		return Target{
			Kind: TargetKindStack,
			URL:  context.StackURL,
		}, nil
	case config.ContextKindCloud:
		return Target{
			Kind:         TargetKindCloud,
			URL:          context.CloudURL,
			Organization: context.Organization,
			Stack:        context.Stack,
		}, nil
	case config.ContextKindCloudStack:
		if context.Organization == "" {
			return Target{}, fmt.Errorf("organization is required for cloud-stack targets")
		}
		if context.Stack == "" {
			return Target{}, fmt.Errorf("stack is required for cloud-stack targets")
		}
		return Target{
			Kind:         TargetKindCloudStack,
			URL:          context.CloudURL,
			Organization: context.Organization,
			Stack:        context.Stack,
		}, nil
	default:
		return Target{}, fmt.Errorf("unsupported context kind %q", context.Kind)
	}
}

func (r *Runtime) APIPolicyFor(product capabilities.Product) config.APIPolicy {
	if r == nil {
		return config.APIPolicyLatestCompatible
	}
	if policy := r.Context.API[string(product)]; policy != "" {
		return config.APIPolicy(policy)
	}
	return config.APIPolicyLatestCompatible
}

func (r *Runtime) HTTPClient(ctx context.Context) (*http.Client, error) {
	if r == nil {
		return nil, errors.New("runtime is nil")
	}
	return auth.NewHTTPClient(ctx, r.authForTarget(), r.Credentials, r.AuthOptions)
}

func (r *Runtime) authForTarget() config.Auth {
	authConfig := r.Context.Auth
	if authConfig.Method == config.AuthMethodCloudDevice {
		authConfig.Scopes = nil
	}
	if authConfig.Method == config.AuthMethodClientCredentials && len(authConfig.Scopes) == 0 {
		switch r.Target.Kind {
		case TargetKindCloud, TargetKindCloudStack:
			authConfig.Scopes = append([]string(nil), auth.OrganizationScopes...)
		}
	}
	return authConfig
}

func (r *Runtime) ComponentVersions(ctx context.Context) ([]capabilities.ComponentVersion, error) {
	client := r.VersionsClient
	if client == nil {
		httpClient, err := r.HTTPClient(ctx)
		if err != nil {
			return nil, err
		}
		client = HTTPVersionsClient{
			BaseURL:    r.Target.URL,
			HTTPClient: httpClient,
		}
	}
	return client.GetVersions(ctx)
}

func (r *Runtime) ResolveAPIVersion(ctx context.Context, request capabilities.VersionResolutionRequest) (capabilities.APIVersion, error) {
	if request.Compatibility == nil {
		request.Compatibility = r.Compatibility
	}
	if request.Compatibility == nil {
		request.Compatibility = capabilities.DefaultComponentCompatibility
	}
	if request.Policy == "" {
		request.Policy = capabilities.VersionPolicy(r.APIPolicyFor(request.Product))
	}
	if request.ComponentVersion == "" {
		versions, err := r.ComponentVersions(ctx)
		if err != nil {
			return "", err
		}
		componentVersion, ok := componentVersionFor(versions, request.Product)
		if !ok {
			return "", fmt.Errorf("component version for %s not found", request.Product)
		}
		request.ComponentVersion = componentVersion.Version
	}
	return capabilities.ResolveAPIVersion(request)
}
