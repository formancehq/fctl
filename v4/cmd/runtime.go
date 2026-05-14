package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	v4auth "github.com/formancehq/fctl/v4/internal/auth"
	"github.com/formancehq/fctl/v4/internal/capabilities"
	cloudcmd "github.com/formancehq/fctl/v4/internal/commands/cloud"
	v4config "github.com/formancehq/fctl/v4/internal/config"
	"github.com/formancehq/fctl/v4/internal/credentials"
	"github.com/formancehq/fctl/v4/internal/runtime"
)

func runtimeFromCommand(cmd *cobra.Command) (*runtime.Runtime, error) {
	path, err := configPath(cmd)
	if err != nil {
		return nil, err
	}
	contextName, err := contextNameFromCommand(cmd)
	if err != nil {
		return nil, err
	}
	organization, stack, err := organizationAndStackFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	store, err := credentialStoreFromCommand(cmd)
	if err != nil {
		return nil, err
	}
	authOptions, err := authOptionsFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	rt, err := runtime.New(cmd.Context(), runtime.Options{
		ConfigPath:      path,
		ContextOverride: v4config.ContextOverride{Name: contextName, Organization: organization, Stack: stack},
		Credentials:     store,
		Auth:            authOptions,
		Manifest:        capabilities.GeneratedManifest,
		Compatibility:   capabilities.DefaultComponentCompatibility,
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, missingConfigError(path)
		}
		return nil, err
	}
	if debug, err := cmd.Root().PersistentFlags().GetBool(debugFlag); err != nil {
		return nil, err
	} else if debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "debug: context=%s target=%s url=%s\n", rt.ContextName, rt.Target.Kind, rt.Target.URL)
	}
	return rt, nil
}

func stackRuntimeFromCommand(cmd *cobra.Command) (*runtime.Runtime, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return nil, err
	}
	switch rt.Target.Kind {
	case runtime.TargetKindStack:
		return rt, nil
	case runtime.TargetKindCloud:
		return resolveCloudProfileStackRuntime(cmd, rt)
	case runtime.TargetKindCloudStack:
		return resolveExplicitCloudStackRuntime(cmd, rt)
	default:
		return rt, nil
	}
}

func resolveCloudProfileStackRuntime(cmd *cobra.Command, rt *runtime.Runtime) (*runtime.Runtime, error) {
	if rt.Context.CloudURL == "" {
		return nil, fmt.Errorf("cloud profiles require a cloud URL")
	}
	if rt.Target.Organization == "" {
		return nil, fmt.Errorf("stack target commands on Cloud profiles require --organization and --stack, or a cloud-stack profile")
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return nil, err
	}
	output, err := cloudcmd.ListStacksService{Client: newMembershipClient(rt.Context.CloudURL, httpClient)}.Run(cmd.Context(), cloudcmd.ListStacksInput{
		OrganizationID: rt.Target.Organization,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve cloud stack target: %w", err)
	}
	if len(output.Stacks) == 0 {
		return nil, fmt.Errorf("resolve cloud stack target: no stacks found in organization %q", rt.Target.Organization)
	}
	if len(output.Stacks) > 1 {
		return nil, fmt.Errorf("resolve cloud stack target: --stack is required when organization %q has multiple stacks; available stacks: %s", rt.Target.Organization, stackLabels(output.Stacks))
	}
	return useCloudStackRuntime(cmd, rt, httpClient, output.Stacks[0])
}

func resolveExplicitCloudStackRuntime(cmd *cobra.Command, rt *runtime.Runtime) (*runtime.Runtime, error) {
	if rt.Context.CloudURL == "" {
		return nil, fmt.Errorf("cloud-stack profiles require a cloud URL")
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return nil, err
	}
	output, err := cloudcmd.ReadStackService{Client: newMembershipClient(rt.Context.CloudURL, httpClient)}.Run(cmd.Context(), cloudcmd.StackIDInput{
		OrganizationID: rt.Target.Organization,
		StackID:        rt.Target.Stack,
	})
	if err != nil {
		if notFoundErr := cloudStackNotFoundError(cmd, rt, httpClient); notFoundErr != nil {
			return nil, notFoundErr
		}
		return nil, fmt.Errorf("resolve cloud stack target: %w", err)
	}
	return useCloudStackRuntime(cmd, rt, httpClient, output.Stack)
}

func useCloudStackRuntime(cmd *cobra.Command, rt *runtime.Runtime, httpClient *http.Client, stack cloudcmd.StackSummary) (*runtime.Runtime, error) {
	if stack.ID == "" {
		return nil, fmt.Errorf("resolve cloud stack target: stack id is empty")
	}
	if stack.URI == "" {
		return nil, fmt.Errorf("resolve cloud stack target: stack %s has no URI", stack.ID)
	}
	rt.Target.Kind = runtime.TargetKindCloudStack
	rt.Target.Stack = stack.ID
	rt.Context.Kind = v4config.ContextKindCloudStack
	rt.Context.Stack = stack.ID
	rt.Target.URL = stack.URI
	rt.Context.StackURL = stack.URI
	assertionAuth := rt.Context.Auth
	if assertionAuth.Method == v4config.AuthMethodClientCredentials {
		assertionAuth.Scopes = []string{"openid", "offline_access"}
		assertionAuth.Resources = []string{v4auth.StackResource(rt.Target.Organization, rt.Target.Stack)}
	}
	source, err := v4auth.NewTokenSource(assertionAuth, rt.Credentials, rt.AuthOptions)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return rt, nil
	}
	assertion, err := source.Token(cmd.Context())
	if err != nil {
		return nil, fmt.Errorf("resolve cloud stack target token: %w", err)
	}
	stackToken, err := v4auth.ExchangeStackToken(cmd.Context(), baseHTTPClient(rt.AuthOptions), stack.URI, assertion.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("resolve cloud stack target token: %w", err)
	}
	store := credentials.NewMemoryStore()
	const tokenRef = "runtime/cloud-stack-token"
	if err := store.Set(cmd.Context(), tokenRef, stackToken.AccessToken); err != nil {
		return nil, err
	}
	rt.Credentials = store
	rt.Context.Auth = v4config.Auth{Method: v4config.AuthMethodToken, TokenRef: tokenRef}
	return rt, nil
}

func cloudStackNotFoundError(cmd *cobra.Command, rt *runtime.Runtime, httpClient *http.Client) error {
	output, err := cloudcmd.ListStacksService{Client: newMembershipClient(rt.Context.CloudURL, httpClient)}.Run(cmd.Context(), cloudcmd.ListStacksInput{
		OrganizationID: rt.Target.Organization,
	})
	if err != nil {
		return nil
	}
	for _, stack := range output.Stacks {
		if stack.ID == rt.Target.Stack {
			return nil
		}
	}
	if len(output.Stacks) == 0 {
		return fmt.Errorf("resolve cloud stack target: stack %q was not found in organization %q", rt.Target.Stack, rt.Target.Organization)
	}
	return fmt.Errorf("resolve cloud stack target: stack %q was not found in organization %q; available stacks: %s", rt.Target.Stack, rt.Target.Organization, stackLabels(output.Stacks))
}

func stackLabels(stacks []cloudcmd.StackSummary) string {
	available := make([]string, 0, len(stacks))
	for _, stack := range stacks {
		label := stack.ID
		if stack.Name != "" {
			label += " (" + stack.Name + ")"
		}
		available = append(available, label)
	}
	return strings.Join(available, ", ")
}

func baseHTTPClient(options v4auth.Options) *http.Client {
	if options.HTTPClient != nil {
		return options.HTTPClient
	}
	return http.DefaultClient
}

func contextNameFromCommand(cmd *cobra.Command) (string, error) {
	flags := cmd.Root().PersistentFlags()
	profileName, err := flags.GetString(profileFlag)
	if err != nil {
		return "", err
	}
	contextName, err := flags.GetString(contextFlag)
	if err != nil {
		return "", err
	}
	if contextName != "" && profileName != "" {
		return "", fmt.Errorf("--profile and --context are mutually exclusive")
	}
	if contextName != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Flag --context has been deprecated, use --profile")
		return contextName, nil
	}
	return profileName, nil
}

func organizationAndStackFromCommand(cmd *cobra.Command) (string, string, error) {
	flags := cmd.Root().PersistentFlags()
	organization, err := flags.GetString(organizationFlag)
	if err != nil {
		return "", "", err
	}
	stack, err := flags.GetString(stackFlag)
	if err != nil {
		return "", "", err
	}
	return organization, stack, nil
}

func credentialStoreFromCommand(cmd *cobra.Command) (credentials.Store, error) {
	dir, err := credentialDirFromCommand(cmd)
	if err != nil {
		return nil, err
	}
	return credentials.NewInsecureFileStore(dir), nil
}

func persistentCredentialStoreFromCommand(cmd *cobra.Command) (credentials.Store, error) {
	dir, err := credentialDirFromCommand(cmd)
	if err != nil {
		return nil, err
	}
	return credentials.NewInsecureFileStore(dir), nil
}

func credentialDirFromCommand(cmd *cobra.Command) (string, error) {
	dir, err := cmd.Root().PersistentFlags().GetString(credentialDirFlag)
	if err != nil {
		return "", err
	}
	if dir != "" {
		return dir, nil
	}
	path, err := configPath(cmd)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "credentials"), nil
}

func authOptionsFromCommand(cmd *cobra.Command) (v4auth.Options, error) {
	insecureTLS, err := cmd.Root().PersistentFlags().GetBool(insecureTLSFlag)
	if err != nil {
		return v4auth.Options{}, err
	}
	if !insecureTLS {
		return v4auth.Options{}, nil
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // Explicit user opt-in via --insecure-tls.
	return v4auth.Options{
		HTTPClient: &http.Client{Transport: transport},
	}, nil
}
