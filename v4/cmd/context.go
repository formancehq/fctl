package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	v4config "github.com/formancehq/fctl/v4/internal/config"
)

func newContextCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "context",
		Short: "Manage fctl v4 contexts",
	}

	command.AddCommand(
		newContextListCommand(),
		newContextShowCommand(),
		newContextUseCommand(),
		newContextCreateCommand(),
	)

	return command
}

func newContextListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured contexts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := loadConfig(cmd, true)
			if err != nil {
				return err
			}

			names := cfg.ContextNames()
			result := contextListOutput{
				Current:  cfg.CurrentContext,
				Contexts: names,
			}
			if handled, err := writeStructuredOutput(cmd, result); handled || err != nil {
				return err
			}
			if len(names) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No contexts found.")
				return err
			}
			for _, name := range names {
				prefix := " "
				if name == cfg.CurrentContext {
					prefix = "*"
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", prefix, name); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newContextShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show a configured context",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd, false)
			if err != nil {
				return err
			}

			override := v4config.ContextOverride{}
			if len(args) == 1 {
				override.Name = args[0]
			}
			name, context, err := v4config.ResolveCurrentContext(cfg, override)
			if err != nil {
				return err
			}

			result := contextShowOutput{Name: name, Current: name == cfg.CurrentContext, Context: context}
			if handled, err := writeStructuredOutput(cmd, result); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\nKind: %s\n", name, context.Kind)
			return err
		},
	}
}

func newContextUseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the current context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfig(cmd, false)
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := cfg.Contexts[name]; !ok {
				return fmt.Errorf("context %q does not exist", name)
			}
			cfg.CurrentContext = name
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, map[string]string{"currentContext": name}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Current context set to %s.\n", name)
			return err
		},
	}
}

func newContextCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "Create contexts",
	}
	command.AddCommand(newContextCreateStackCommand())
	return command
}

func newContextCreateStackCommand() *cobra.Command {
	var stackURL string
	var authMethod string
	var issuerURL string
	var clientID string
	var secretRef string
	var defaultLedger string

	command := &cobra.Command{
		Use:   "stack <name>",
		Short: "Create a direct stack context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfig(cmd, true)
			if err != nil {
				return err
			}
			if cfg.Contexts == nil {
				cfg.Contexts = map[string]v4config.Context{}
			}

			name := args[0]
			if _, exists := cfg.Contexts[name]; exists {
				return fmt.Errorf("context %q already exists", name)
			}

			auth := v4config.Auth{Method: v4config.AuthMethod(authMethod)}
			switch auth.Method {
			case v4config.AuthMethodClientCredentials:
				auth.IssuerURL = issuerURL
				auth.ClientID = clientID
				auth.SecretRef = secretRef
			case v4config.AuthMethodNone:
			default:
				return fmt.Errorf("unsupported stack auth method %q", authMethod)
			}

			defaults := map[string]string{}
			if defaultLedger != "" {
				defaults["ledger"] = defaultLedger
			}
			if len(defaults) == 0 {
				defaults = nil
			}

			cfg.Contexts[name] = v4config.Context{
				Kind:     v4config.ContextKindStack,
				StackURL: stackURL,
				Auth:     auth,
				Defaults: defaults,
				API:      map[string]string{"ledger": string(v4config.APIPolicyLatestCompatible)},
			}
			if cfg.CurrentContext == "" {
				cfg.CurrentContext = name
			}

			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, contextShowOutput{
				Name:    name,
				Current: name == cfg.CurrentContext,
				Context: cfg.Contexts[name],
			}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Context %s created.\n", name)
			return err
		},
	}

	command.Flags().StringVar(&stackURL, "stack-url", "", "Stack API URL")
	command.Flags().StringVar(&authMethod, "auth-method", string(v4config.AuthMethodNone), "Authentication method (none, client_credentials)")
	command.Flags().StringVar(&issuerURL, "issuer-url", "", "OIDC issuer URL for client credentials")
	command.Flags().StringVar(&clientID, "client-id", "", "Client ID for client credentials")
	command.Flags().StringVar(&secretRef, "secret-ref", "", "Credential reference for client secret")
	command.Flags().StringVar(&defaultLedger, "default-ledger", "", "Default ledger for this context")

	return command
}

type contextListOutput struct {
	Current  string   `json:"currentContext"`
	Contexts []string `json:"contexts"`
}

type contextShowOutput struct {
	Name    string           `json:"name"`
	Current bool             `json:"current"`
	Context v4config.Context `json:"context"`
}
