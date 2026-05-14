package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	v4auth "github.com/formancehq/fctl/v4/internal/auth"
	v4config "github.com/formancehq/fctl/v4/internal/config"
	v4prompt "github.com/formancehq/fctl/v4/internal/prompt"
)

const (
	loginTargetCloud      = "cloud"
	loginTargetEE         = "ee"
	loginTargetOpenSource = "open-source"
	defaultProfileName    = "default"
)

var loginOpenURL = v4auth.OpenURL

func newLoginCommand() *cobra.Command {
	var target string
	var membershipURL string
	var stackURL string
	var issuerURL string
	var clientID string
	var clientSecret string
	var clientSecretStdin bool
	var defaultLedger string

	command := &cobra.Command{
		Use:   "login",
		Short: "Connect fctl to Formance Cloud, Enterprise, or an open-source stack",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			input, err := newLoginInput(cmd)
			if err != nil {
				return err
			}
			if target == "" {
				if nonInteractive, err := cmd.Root().PersistentFlags().GetBool(nonInteractiveFlag); err != nil {
					return err
				} else if nonInteractive {
					return fmt.Errorf("login requires --target in non-interactive mode")
				}
				target, err = input.chooseTarget(cmd)
				if err != nil {
					return err
				}
			}
			normalizedTarget, err := normalizeLoginTarget(target)
			if err != nil {
				return err
			}

			profileName, err := selectedProfileNameForLogin(cmd)
			if err != nil {
				return err
			}
			organization, stack, err := organizationAndStackFromCommand(cmd)
			if err != nil {
				return err
			}

			cfg, path, err := loadConfig(cmd, true)
			if err != nil {
				return err
			}
			if cfg.Contexts == nil {
				cfg.Contexts = map[string]v4config.Context{}
			}

			context, err := loginContextForTarget(cmd, input, loginContextInput{
				Target:            normalizedTarget,
				MembershipURL:     membershipURL,
				StackURL:          stackURL,
				Organization:      organization,
				Stack:             stack,
				DefaultLedger:     defaultLedger,
				IssuerURL:         issuerURL,
				ClientID:          clientID,
				ClientSecret:      clientSecret,
				ClientSecretStdin: clientSecretStdin,
				ProfileName:       profileName,
			})
			if err != nil {
				return err
			}

			cfg.Contexts[profileName] = context
			cfg.CurrentContext = profileName
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, contextShowOutput{Name: profileName, Current: true, Context: context}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Logged in with profile %s.\n", profileName)
			return err
		},
	}
	command.Flags().StringVar(&target, "target", "", "Target type (cloud, ee, open-source)")
	command.Flags().StringVar(&membershipURL, "membership-url", "", "Cloud or Enterprise membership URL")
	command.Flags().StringVar(&stackURL, "stack-url", "", "Open-source stack API URL")
	command.Flags().StringVar(&issuerURL, "issuer-url", "", "OIDC issuer URL for token exchange")
	command.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	command.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth client secret")
	command.Flags().BoolVar(&clientSecretStdin, "client-secret-stdin", false, "Read OAuth client secret from stdin")
	command.Flags().StringVar(&defaultLedger, "default-ledger", "", "Default ledger for this profile")
	return command
}

type loginContextInput struct {
	Target            string
	MembershipURL     string
	StackURL          string
	Organization      string
	Stack             string
	DefaultLedger     string
	IssuerURL         string
	ClientID          string
	ClientSecret      string
	ClientSecretStdin bool
	ProfileName       string
}

func loginContextForTarget(cmd *cobra.Command, input loginInput, options loginContextInput) (v4config.Context, error) {
	switch options.Target {
	case loginTargetCloud:
		if options.MembershipURL == "" {
			options.MembershipURL = v4config.DefaultCloudURL
		}
		return platformLoginContext(cmd, input, options)
	case loginTargetEE:
		if options.MembershipURL == "" {
			if input.nonInteractive {
				return v4config.Context{}, fmt.Errorf("login ee requires --membership-url")
			}
			membershipURL, err := input.prompt(cmd, "Membership URL")
			if err != nil {
				return v4config.Context{}, err
			}
			options.MembershipURL = membershipURL
		}
		return platformLoginContext(cmd, input, options)
	case loginTargetOpenSource:
		if options.StackURL == "" {
			if input.nonInteractive {
				return v4config.Context{}, fmt.Errorf("login open-source requires --stack-url")
			}
			stackURL, err := input.prompt(cmd, "Stack URL")
			if err != nil {
				return v4config.Context{}, err
			}
			options.StackURL = stackURL
		}
		if options.StackURL == "" {
			return v4config.Context{}, fmt.Errorf("login open-source requires --stack-url")
		}
		auth, err := authFromLoginOptions(cmd, input, options, false)
		if err != nil {
			return v4config.Context{}, err
		}
		return v4config.Context{
			Kind:     v4config.ContextKindStack,
			StackURL: options.StackURL,
			Auth:     auth,
			Defaults: defaultsFromLogin(options.DefaultLedger),
			API:      map[string]string{"ledger": string(v4config.APIPolicyLatestCompatible)},
		}, nil
	default:
		return v4config.Context{}, fmt.Errorf("unsupported login target %q", options.Target)
	}
}

func platformLoginContext(cmd *cobra.Command, input loginInput, options loginContextInput) (v4config.Context, error) {
	if options.MembershipURL == "" {
		return v4config.Context{}, fmt.Errorf("login %s requires --membership-url", options.Target)
	}
	auth, err := authFromLoginOptions(cmd, input, options, true)
	if err != nil {
		return v4config.Context{}, err
	}
	kind := v4config.ContextKindCloud
	if options.Stack != "" {
		kind = v4config.ContextKindCloudStack
	}
	context := v4config.Context{
		Kind:         kind,
		CloudURL:     options.MembershipURL,
		Organization: options.Organization,
		Stack:        options.Stack,
		Auth:         auth,
		Defaults:     defaultsFromLogin(options.DefaultLedger),
		API:          map[string]string{"ledger": string(v4config.APIPolicyLatestCompatible)},
	}
	if kind == v4config.ContextKindCloudStack && context.Organization == "" {
		return v4config.Context{}, fmt.Errorf("login with --stack requires --organization")
	}
	return context, nil
}

func authFromLoginOptions(cmd *cobra.Command, input loginInput, options loginContextInput, platform bool) (v4config.Auth, error) {
	if platform && options.ClientID == "" && options.ClientSecret == "" && !options.ClientSecretStdin {
		authMethod, err := input.choosePlatformAuth(cmd)
		if err != nil {
			return v4config.Auth{}, err
		}
		switch authMethod {
		case "1", "browser", "browser/device login", "device":
			return cloudDeviceAuthFromLoginOptions(cmd, options)
		case "2", "client credentials", "client-credentials":
			clientID, err := input.prompt(cmd, "Client ID")
			if err != nil {
				return v4config.Auth{}, err
			}
			clientSecret, err := input.secretPrompt(cmd, "Client secret")
			if err != nil {
				return v4config.Auth{}, err
			}
			options.ClientID = clientID
			options.ClientSecret = clientSecret
		default:
			return v4config.Auth{}, fmt.Errorf("unsupported authentication choice %q", authMethod)
		}
	}

	if options.ClientID != "" || options.ClientSecret != "" || options.ClientSecretStdin {
		if options.ClientSecret != "" && options.ClientSecretStdin {
			return v4config.Auth{}, fmt.Errorf("--client-secret and --client-secret-stdin are mutually exclusive")
		}
		if options.ClientID == "" {
			return v4config.Auth{}, fmt.Errorf("login client credentials requires --client-id")
		}
		issuerURL := options.IssuerURL
		if issuerURL == "" {
			issuerURL = options.MembershipURL
		}
		if issuerURL == "" {
			return v4config.Auth{}, fmt.Errorf("login client credentials requires --issuer-url")
		}
		secret := options.ClientSecret
		if options.ClientSecretStdin {
			data, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return v4config.Auth{}, err
			}
			secret = strings.TrimSpace(string(data))
		}
		if secret == "" {
			return v4config.Auth{}, fmt.Errorf("login client credentials requires --client-secret or --client-secret-stdin")
		}
		ref := "contexts/" + options.ProfileName + "/client-secret"
		store, err := persistentCredentialStoreFromCommand(cmd)
		if err != nil {
			return v4config.Auth{}, err
		}
		if err := store.Set(cmd.Context(), ref, secret); err != nil {
			return v4config.Auth{}, err
		}
		return v4config.Auth{
			Method:    v4config.AuthMethodClientCredentials,
			IssuerURL: issuerURL,
			ClientID:  options.ClientID,
			SecretRef: ref,
			Scopes:    clientCredentialsScopesForPlatform(platform),
		}, nil
	}

	if platform {
		return cloudDeviceAuthFromLoginOptions(cmd, options)
	}
	return v4config.Auth{Method: v4config.AuthMethodNone}, nil
}

func cloudDeviceAuthFromLoginOptions(cmd *cobra.Command, options loginContextInput) (v4config.Auth, error) {
	store, err := persistentCredentialStoreFromCommand(cmd)
	if err != nil {
		return v4config.Auth{}, err
	}
	authOptions, err := authOptionsFromCommand(cmd)
	if err != nil {
		return v4config.Auth{}, err
	}
	tokens, err := v4auth.DeviceLogin(cmd.Context(), v4auth.DeviceLoginOptions{
		IssuerURL:  options.MembershipURL,
		ClientID:   v4auth.DeviceClientID,
		Scopes:     []string{"openid", "offline_access", "accesses", "on_behalf"},
		Prompt:     []string{"no-org"},
		HTTPClient: authOptions.HTTPClient,
		OpenURL:    loginOpenURL,
		Out:        cmd.OutOrStdout(),
	})
	if err != nil {
		return v4config.Auth{}, err
	}
	ref := "contexts/" + options.ProfileName + "/root-tokens"
	encoded, err := v4auth.MarshalDeviceTokens(tokens)
	if err != nil {
		return v4config.Auth{}, err
	}
	if err := store.Set(cmd.Context(), ref, encoded); err != nil {
		return v4config.Auth{}, err
	}
	return v4config.Auth{
		Method:    v4config.AuthMethodCloudDevice,
		IssuerURL: options.MembershipURL,
		TokenRef:  ref,
		Account:   v4auth.EmailFromIDToken(tokens.IDToken),
	}, nil
}

func defaultsFromLogin(defaultLedger string) map[string]string {
	if defaultLedger == "" {
		return nil
	}
	return map[string]string{"ledger": defaultLedger}
}

func newLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear authentication from the selected profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, path, name, context, err := selectedContextForSession(cmd)
			if err != nil {
				return err
			}
			store, err := credentialStoreFromCommand(cmd)
			if err != nil {
				return err
			}
			deleteCredentialRef(cmd, store, context.Auth.TokenRef)
			deleteCredentialRef(cmd, store, context.Auth.SecretRef)
			context.Auth = v4config.Auth{Method: v4config.AuthMethodNone}
			cfg.Contexts[name] = context
			if err := v4config.SaveFile(path, cfg); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, contextShowOutput{Name: name, Current: true, Context: context}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Logged out from profile %s.\n", name)
			return err
		},
	}
}

func newWhoamiCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the selected profile and authentication state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, name, context, err := selectedContextForSession(cmd)
			if err != nil {
				return err
			}
			output := contextShowOutput{Name: name, Current: name == cfg.CurrentContext, Context: context}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Profile\t%s\n", name); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Target\t%s\n", context.Kind); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Auth\t%s\n", context.Auth.Method); err != nil {
				return err
			}
			if context.Organization != "" {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Organization\t%s\n", context.Organization); err != nil {
					return err
				}
			}
			if context.Stack != "" {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Stack\t%s\n", context.Stack); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

type loginInput struct {
	reader         *bufio.Reader
	wizard         v4prompt.Wizard
	nonInteractive bool
}

func newLoginInput(cmd *cobra.Command) (loginInput, error) {
	nonInteractive, err := cmd.Root().PersistentFlags().GetBool(nonInteractiveFlag)
	if err != nil {
		return loginInput{}, err
	}
	noColor, err := cmd.Root().PersistentFlags().GetBool(noColorFlag)
	if err != nil {
		return loginInput{}, err
	}

	in := cmd.InOrStdin()
	input := loginInput{
		reader:         bufio.NewReader(in),
		nonInteractive: nonInteractive,
	}
	if !nonInteractive && !noColor {
		input.wizard = v4prompt.NewWizard(in, cmd.ErrOrStderr())
	}
	return input, nil
}

func (i loginInput) chooseTarget(cmd *cobra.Command) (string, error) {
	if i.wizard.Available() {
		return i.wizard.Select("What do you want to connect to?", []v4prompt.Choice{
			{Title: "Formance Cloud", Value: loginTargetCloud},
			{Title: "Formance EE Self-Hosted", Value: loginTargetEE},
			{Title: "Formance Open Source / local", Value: loginTargetOpenSource},
		})
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "What do you want to connect to?"); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "1. Formance Cloud"); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "2. Formance EE Self-Hosted"); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "3. Formance Open Source / local"); err != nil {
		return "", err
	}
	answer, err := i.prompt(cmd, "Choice")
	if err != nil {
		return "", err
	}
	switch strings.TrimSpace(strings.ToLower(answer)) {
	case "1", "cloud", "formance cloud":
		return loginTargetCloud, nil
	case "2", "ee", "self-hosted", "enterprise":
		return loginTargetEE, nil
	case "3", "open-source", "oss", "local":
		return loginTargetOpenSource, nil
	default:
		return "", fmt.Errorf("unsupported login choice %q", answer)
	}
}

func (i loginInput) choosePlatformAuth(cmd *cobra.Command) (string, error) {
	if i.nonInteractive {
		return "", fmt.Errorf("login requires --client-id/--client-secret in non-interactive mode")
	}
	if i.wizard.Available() {
		return i.wizard.Select("How do you want to authenticate?", []v4prompt.Choice{
			{Title: "Browser/device login", Value: "browser"},
			{Title: "Client credentials", Value: "client-credentials"},
		})
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "How do you want to authenticate?"); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "1. Browser/device login"); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "2. Client credentials"); err != nil {
		return "", err
	}
	answer, err := i.prompt(cmd, "Choice")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.ToLower(answer)), nil
}

func (i loginInput) prompt(cmd *cobra.Command, label string) (string, error) {
	return i.promptValue(cmd, label, "", false)
}

func (i loginInput) secretPrompt(cmd *cobra.Command, label string) (string, error) {
	return i.promptValue(cmd, label, "", true)
}

func (i loginInput) promptValue(cmd *cobra.Command, label string, placeholder string, secret bool) (string, error) {
	if i.nonInteractive {
		return "", fmt.Errorf("%s is required in non-interactive mode", label)
	}
	if i.wizard.Available() {
		return i.wizard.Input(label, placeholder, secret)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: ", label); err != nil {
		return "", err
	}
	value, err := i.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func normalizeLoginTarget(value string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "cloud", "formance-cloud", "formance cloud":
		return loginTargetCloud, nil
	case "ee", "enterprise", "self-hosted", "selfhosted", "formance-ee":
		return loginTargetEE, nil
	case "open-source", "opensource", "oss", "local", "stack":
		return loginTargetOpenSource, nil
	default:
		return "", fmt.Errorf("unsupported login target %q", value)
	}
}

func selectedProfileNameForLogin(cmd *cobra.Command) (string, error) {
	name, err := contextNameFromCommand(cmd)
	if err != nil {
		return "", err
	}
	if name == "" {
		name = defaultProfileName
	}
	return name, nil
}
