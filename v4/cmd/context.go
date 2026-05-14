package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	v4config "github.com/formancehq/fctl/v4/internal/config"
)

func newContextCommand() *cobra.Command {
	command := &cobra.Command{
		Use:    "context",
		Short:  "Manage fctl v4 contexts",
		Hidden: true,
	}

	command.AddCommand(
		newContextListCommand(),
		newContextShowCommand(),
		newContextUseCommand(),
		newContextDeleteCommand(),
		newContextRenameCommand(),
		newContextCreateCommand(),
		newContextSetCommand(),
		newContextUnsetDefaultsCommand(),
		newContextWizardCommand(),
	)

	return command
}

func newProfileCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "profile",
		Short: "Manage profiles",
	}

	command.AddCommand(
		newContextListCommand(),
		newContextShowCommand(),
		newContextUseCommand(),
		newContextDeleteCommand(),
		newContextRenameCommand(),
		newContextCreateCommand(),
		newContextSetCommand(),
		newContextUnsetDefaultsCommand(),
	)

	return command
}

func newProfilesCommand() *cobra.Command {
	command := &cobra.Command{
		Use:    "profiles",
		Short:  "Deprecated alias for profile",
		Hidden: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Command profiles has been deprecated, use profile")
		},
	}
	command.Deprecated = "use profile"
	command.AddCommand(
		newContextListCommand(),
		newContextShowCommand(),
		newContextUseCommand(),
		newContextDeleteCommand(),
		newContextRenameCommand(),
		newProfilesResetCommand(),
		newProfilesSetDefaultOrganizationCommand(),
		newProfilesSetDefaultStackCommand(),
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
				_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No contexts found."))
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
			return writeStyledColonKeyValues(cmd,
				styledKeyValue{Label: "Name", Value: name},
				styledKeyValue{Label: "Kind", Value: string(context.Kind)},
			)
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
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Current context set to %s.", name)))
			return err
		},
	}
}

func newContextDeleteCommand() *cobra.Command {
	var force bool
	var confirm bool

	command := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a configured context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("context delete requires --confirm")
			}
			cfg, path, err := loadConfig(cmd, false)
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := cfg.Contexts[name]; !ok {
				return fmt.Errorf("context %q does not exist", name)
			}
			if len(cfg.Contexts) == 1 {
				return fmt.Errorf("cannot delete the last context")
			}
			if cfg.CurrentContext == name && !force {
				return fmt.Errorf("context %q is current; pass --force to delete it", name)
			}
			delete(cfg.Contexts, name)
			if cfg.CurrentContext == name {
				cfg.CurrentContext = ""
			}
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, map[string]string{"deleted": name}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Context %s deleted.", name)))
			return err
		},
	}
	command.Flags().BoolVar(&force, "force", false, "Delete the current context")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm context deletion")
	return command
}

func newContextRenameCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a configured context",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfig(cmd, false)
			if err != nil {
				return err
			}
			oldName := args[0]
			newName := args[1]
			context, ok := cfg.Contexts[oldName]
			if !ok {
				return fmt.Errorf("context %q does not exist", oldName)
			}
			if _, exists := cfg.Contexts[newName]; exists {
				return fmt.Errorf("context %q already exists", newName)
			}
			delete(cfg.Contexts, oldName)
			cfg.Contexts[newName] = context
			if cfg.CurrentContext == oldName {
				cfg.CurrentContext = newName
			}
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, map[string]string{"renamed": oldName, "name": newName}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Context %s renamed to %s.", oldName, newName)))
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
	command.AddCommand(newContextCreateCloudCommand())
	command.AddCommand(newContextCreateCloudStackCommand())
	return command
}

func newContextSetCommand() *cobra.Command {
	var organization string
	var stack string
	var defaultLedger string

	command := &cobra.Command{
		Use:   "set [name]",
		Short: "Update a configured context",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfig(cmd, false)
			if err != nil {
				return err
			}
			name := cfg.CurrentContext
			if len(args) == 1 {
				name = args[0]
			}
			if name == "" {
				return fmt.Errorf("no context selected")
			}
			context, ok := cfg.Contexts[name]
			if !ok {
				return fmt.Errorf("context %q does not exist", name)
			}
			if organization != "" {
				if context.Kind != v4config.ContextKindCloud && context.Kind != v4config.ContextKindCloudStack {
					return fmt.Errorf("--organization can only be set on cloud or cloud-stack contexts")
				}
				context.Organization = organization
			}
			if stack != "" {
				if context.Kind != v4config.ContextKindCloudStack {
					return fmt.Errorf("--stack can only be set on cloud-stack contexts")
				}
				context.Stack = stack
			}
			if defaultLedger != "" {
				if context.Defaults == nil {
					context.Defaults = map[string]string{}
				}
				context.Defaults["ledger"] = defaultLedger
			}
			cfg.Contexts[name] = context
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, contextShowOutput{Name: name, Current: name == cfg.CurrentContext, Context: context}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Context %s updated.", name)))
			return err
		},
	}
	command.Flags().StringVar(&organization, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&stack, "stack", "", "Cloud stack ID")
	command.Flags().StringVar(&defaultLedger, "default-ledger", "", "Default ledger for this context")
	return command
}

func newContextUnsetDefaultsCommand() *cobra.Command {
	var confirm bool

	command := &cobra.Command{
		Use:   "unset-defaults [name]",
		Short: "Clear defaults from a configured context",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("context unset-defaults requires --confirm")
			}
			cfg, path, err := loadConfig(cmd, false)
			if err != nil {
				return err
			}
			name := cfg.CurrentContext
			if len(args) == 1 {
				name = args[0]
			}
			if name == "" {
				return fmt.Errorf("no context selected")
			}
			context, ok := cfg.Contexts[name]
			if !ok {
				return fmt.Errorf("context %q does not exist", name)
			}
			context.Defaults = nil
			cfg.Contexts[name] = context
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, contextShowOutput{Name: name, Current: name == cfg.CurrentContext, Context: context}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Context %s defaults cleared.", name)))
			return err
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm clearing context defaults")
	return command
}

func newProfilesResetCommand() *cobra.Command {
	command := newContextUnsetDefaultsCommand()
	command.Use = "reset <name>"
	command.Short = "Clear defaults from a configured context"
	command.Args = cobra.ExactArgs(1)
	command.PreRun = func(cmd *cobra.Command, _ []string) {
		fmt.Fprintln(cmd.ErrOrStderr(), "Command profiles reset has been deprecated, use profile unset-defaults <name> --confirm")
	}
	return command
}

func newProfilesSetDefaultOrganizationCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "set-default-organization <organization-id>",
		Aliases: []string{"sdo"},
		Short:   "Set the default organization on the current context",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "Command profiles set-default-organization has been deprecated, use profile set --organization <organization-id>")
			cfg, path, err := loadConfig(cmd, false)
			if err != nil {
				return err
			}
			name := cfg.CurrentContext
			if name == "" {
				return fmt.Errorf("no context selected")
			}
			context, ok := cfg.Contexts[name]
			if !ok {
				return fmt.Errorf("context %q does not exist", name)
			}
			if context.Kind != v4config.ContextKindCloud && context.Kind != v4config.ContextKindCloudStack {
				return fmt.Errorf("--organization can only be set on cloud or cloud-stack contexts")
			}
			context.Organization = args[0]
			cfg.Contexts[name] = context
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, contextShowOutput{Name: name, Current: true, Context: context}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Context %s updated.", name)))
			return err
		},
	}
	return command
}

func newProfilesSetDefaultStackCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "set-default-stack <stack-id>",
		Aliases: []string{"sds"},
		Short:   "Set the default stack on the current context",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "Command profiles set-default-stack has been deprecated, use profile set --stack <stack-id>")
			cfg, path, err := loadConfig(cmd, false)
			if err != nil {
				return err
			}
			name := cfg.CurrentContext
			if name == "" {
				return fmt.Errorf("no context selected")
			}
			context, ok := cfg.Contexts[name]
			if !ok {
				return fmt.Errorf("context %q does not exist", name)
			}
			if context.Kind != v4config.ContextKindCloudStack {
				return fmt.Errorf("--stack can only be set on cloud-stack contexts")
			}
			context.Stack = args[0]
			cfg.Contexts[name] = context
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, contextShowOutput{Name: name, Current: true, Context: context}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Context %s updated.", name)))
			return err
		},
	}
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
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Context %s created.", name)))
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

func newContextCreateCloudCommand() *cobra.Command {
	var cloudURL string
	var authMethod string
	var tokenRef string
	var account string

	command := &cobra.Command{
		Use:   "cloud <name>",
		Short: "Create a Formance Cloud control-plane context",
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
			auth, err := cloudContextAuth(authMethod, tokenRef, account)
			if err != nil {
				return err
			}
			cfg.Contexts[name] = v4config.Context{
				Kind:     v4config.ContextKindCloud,
				CloudURL: cloudURL,
				Auth:     auth,
			}
			if cfg.CurrentContext == "" {
				cfg.CurrentContext = name
			}
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, contextShowOutput{Name: name, Current: name == cfg.CurrentContext, Context: cfg.Contexts[name]}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Context %s created.", name)))
			return err
		},
	}
	command.Flags().StringVar(&cloudURL, "cloud-url", v4config.DefaultCloudURL, "Cloud API URL")
	command.Flags().StringVar(&authMethod, "auth-method", string(v4config.AuthMethodCloudDevice), "Authentication method (cloud_device, token, none)")
	command.Flags().StringVar(&tokenRef, "token-ref", "", "Credential reference for token auth")
	command.Flags().StringVar(&account, "account", "", "Cloud account label")
	return command
}

func newContextCreateCloudStackCommand() *cobra.Command {
	var cloudURL string
	var organization string
	var stack string
	var authMethod string
	var tokenRef string
	var account string
	var defaultLedger string

	command := &cobra.Command{
		Use:   "cloud-stack <name>",
		Short: "Create a Formance Cloud stack context",
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
			auth, err := cloudContextAuth(authMethod, tokenRef, account)
			if err != nil {
				return err
			}
			defaults := map[string]string{}
			if defaultLedger != "" {
				defaults["ledger"] = defaultLedger
			}
			if len(defaults) == 0 {
				defaults = nil
			}
			cfg.Contexts[name] = v4config.Context{
				Kind:         v4config.ContextKindCloudStack,
				CloudURL:     cloudURL,
				Organization: organization,
				Stack:        stack,
				Auth:         auth,
				Defaults:     defaults,
				API:          map[string]string{"ledger": string(v4config.APIPolicyLatestCompatible)},
			}
			if cfg.CurrentContext == "" {
				cfg.CurrentContext = name
			}
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, contextShowOutput{Name: name, Current: name == cfg.CurrentContext, Context: cfg.Contexts[name]}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Context %s created.", name)))
			return err
		},
	}
	command.Flags().StringVar(&cloudURL, "cloud-url", v4config.DefaultCloudURL, "Cloud API URL")
	command.Flags().StringVar(&organization, "organization", "", "Cloud organization ID")
	command.Flags().StringVar(&stack, "stack", "", "Cloud stack ID")
	command.Flags().StringVar(&authMethod, "auth-method", string(v4config.AuthMethodCloudDevice), "Authentication method (cloud_device, token, none)")
	command.Flags().StringVar(&tokenRef, "token-ref", "", "Credential reference for token auth")
	command.Flags().StringVar(&account, "account", "", "Cloud account label")
	command.Flags().StringVar(&defaultLedger, "default-ledger", "", "Default ledger for this context")
	return command
}

func cloudContextAuth(authMethod string, tokenRef string, account string) (v4config.Auth, error) {
	auth := v4config.Auth{Method: v4config.AuthMethod(authMethod), TokenRef: tokenRef, Account: account}
	switch auth.Method {
	case v4config.AuthMethodCloudDevice, v4config.AuthMethodToken, v4config.AuthMethodNone:
		return auth, nil
	default:
		return v4config.Auth{}, fmt.Errorf("unsupported cloud auth method %q", authMethod)
	}
}

type contextListOutput struct {
	Current  string   `json:"currentContext" yaml:"currentContext"`
	Contexts []string `json:"contexts" yaml:"contexts"`
}

type contextShowOutput struct {
	Name    string           `json:"name" yaml:"name"`
	Current bool             `json:"current" yaml:"current"`
	Context v4config.Context `json:"context" yaml:"context"`
}
