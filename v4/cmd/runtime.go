package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	v4auth "github.com/formancehq/fctl/v4/internal/auth"
	"github.com/formancehq/fctl/v4/internal/capabilities"
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
		ContextOverride: v4config.ContextOverride{Name: contextName},
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
