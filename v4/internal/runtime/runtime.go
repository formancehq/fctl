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
	Manifest        capabilities.Manifest
	Compatibility   capabilities.ComponentCompatibility
}

type Runtime struct {
	Config      config.Config
	ContextName string
	Context     config.Context
	Target      Target

	Credentials   credentials.Store
	AuthOptions   auth.Options
	Manifest      capabilities.Manifest
	Compatibility capabilities.ComponentCompatibility
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

	target, err := TargetFromContext(selectedContext)
	if err != nil {
		return nil, err
	}

	return &Runtime{
		Config:        cfg,
		ContextName:   contextName,
		Context:       selectedContext,
		Target:        target,
		Credentials:   options.Credentials,
		AuthOptions:   options.Auth,
		Manifest:      options.Manifest,
		Compatibility: options.Compatibility,
	}, nil
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
			Kind: TargetKindCloud,
			URL:  context.CloudURL,
		}, nil
	case config.ContextKindCloudStack:
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
	return auth.NewHTTPClient(ctx, r.Context.Auth, r.Credentials, r.AuthOptions)
}
