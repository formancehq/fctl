// Package config owns the fctl v4 context configuration format, validation,
// persistence, and migration-facing types.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	Version = 4

	EnvContext              = "FCTL_CONTEXT"
	DefaultCloudContextName = "formance-cloud"
	DefaultCloudURL         = "https://app.formance.cloud/api"
)

type ContextKind string

const (
	ContextKindStack      ContextKind = "stack"
	ContextKindCloud      ContextKind = "cloud"
	ContextKindCloudStack ContextKind = "cloud-stack"
)

type AuthMethod string

const (
	AuthMethodCloudDevice       AuthMethod = "cloud_device"
	AuthMethodOIDCDevice        AuthMethod = "oidc_device"
	AuthMethodClientCredentials AuthMethod = "client_credentials"
	AuthMethodToken             AuthMethod = "token"
	AuthMethodNone              AuthMethod = "none"
)

type APIPolicy string

const (
	APIPolicyLatestCompatible APIPolicy = "latest-compatible"
	APIPolicyPinned           APIPolicy = "pinned"
	APIPolicyLatest           APIPolicy = "latest"
)

type Config struct {
	Version        int                `json:"version" yaml:"version"`
	CurrentContext string             `json:"currentContext,omitempty" yaml:"currentContext,omitempty"`
	Contexts       map[string]Context `json:"contexts,omitempty" yaml:"contexts,omitempty"`
}

type Context struct {
	Kind         ContextKind       `json:"kind" yaml:"kind"`
	StackURL     string            `json:"stackURL,omitempty" yaml:"stackURL,omitempty"`
	CloudURL     string            `json:"cloudURL,omitempty" yaml:"cloudURL,omitempty"`
	Organization string            `json:"organization,omitempty" yaml:"organization,omitempty"`
	Stack        string            `json:"stack,omitempty" yaml:"stack,omitempty"`
	Auth         Auth              `json:"auth" yaml:"auth"`
	Defaults     map[string]string `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	API          map[string]string `json:"api,omitempty" yaml:"api,omitempty"`
}

type Auth struct {
	Method    AuthMethod `json:"method" yaml:"method"`
	IssuerURL string     `json:"issuerURL,omitempty" yaml:"issuerURL,omitempty"`
	ClientID  string     `json:"clientID,omitempty" yaml:"clientID,omitempty"`
	SecretRef string     `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	TokenRef  string     `json:"tokenRef,omitempty" yaml:"tokenRef,omitempty"`
	Account   string     `json:"account,omitempty" yaml:"account,omitempty"`
	Scopes    []string   `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

type ContextOverride struct {
	Name         string
	Organization string
	Stack        string
}

func New() Config {
	return Config{
		Version:  Version,
		Contexts: map[string]Context{},
	}
}

func DefaultCloudContext() Context {
	return Context{
		Kind:     ContextKindCloud,
		CloudURL: DefaultCloudURL,
		Auth:     Auth{Method: AuthMethodNone},
	}
}

func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func SaveFile(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (c Config) Validate() error {
	var errs []error

	if c.Version != Version {
		errs = append(errs, fmt.Errorf("unsupported config version %d", c.Version))
	}
	if len(c.Contexts) == 0 {
		errs = append(errs, errors.New("at least one context is required"))
	}
	if c.CurrentContext != "" {
		if _, ok := c.Contexts[c.CurrentContext]; !ok {
			errs = append(errs, fmt.Errorf("current context %q does not exist", c.CurrentContext))
		}
	}

	for name, context := range c.Contexts {
		if strings.TrimSpace(name) == "" {
			errs = append(errs, errors.New("context name cannot be empty"))
			continue
		}
		if err := context.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("context %q: %w", name, err))
		}
	}

	return errors.Join(errs...)
}

func (c Context) Validate() error {
	var errs []error

	switch c.Kind {
	case ContextKindStack:
		if c.StackURL == "" {
			errs = append(errs, errors.New("stackURL is required for stack contexts"))
		}
	case ContextKindCloud:
		if c.CloudURL == "" {
			errs = append(errs, errors.New("cloudURL is required for cloud contexts"))
		}
	case ContextKindCloudStack:
		if c.CloudURL == "" {
			errs = append(errs, errors.New("cloudURL is required for cloud-stack contexts"))
		}
		if c.Organization == "" {
			errs = append(errs, errors.New("organization is required for cloud-stack contexts"))
		}
		if c.Stack == "" {
			errs = append(errs, errors.New("stack is required for cloud-stack contexts"))
		}
	default:
		errs = append(errs, fmt.Errorf("unsupported kind %q", c.Kind))
	}

	if err := c.Auth.Validate(); err != nil {
		errs = append(errs, err)
	}
	for product, policy := range c.API {
		if strings.TrimSpace(product) == "" {
			errs = append(errs, errors.New("api product cannot be empty"))
			continue
		}
		if err := ValidateAPIPolicy(policy); err != nil {
			errs = append(errs, fmt.Errorf("api policy for %q: %w", product, err))
		}
	}

	return errors.Join(errs...)
}

func (a Auth) Validate() error {
	switch a.Method {
	case AuthMethodCloudDevice:
		return nil
	case AuthMethodOIDCDevice:
		if a.IssuerURL == "" {
			return errors.New("issuerURL is required for oidc_device auth")
		}
	case AuthMethodClientCredentials:
		var errs []error
		if a.IssuerURL == "" {
			errs = append(errs, errors.New("issuerURL is required for client_credentials auth"))
		}
		if a.ClientID == "" {
			errs = append(errs, errors.New("clientID is required for client_credentials auth"))
		}
		if a.SecretRef == "" {
			errs = append(errs, errors.New("secretRef is required for client_credentials auth"))
		}
		return errors.Join(errs...)
	case AuthMethodToken:
		if a.TokenRef == "" {
			return errors.New("tokenRef is required for token auth")
		}
	case AuthMethodNone:
		return nil
	default:
		return fmt.Errorf("unsupported auth method %q", a.Method)
	}
	return nil
}

func ValidateAPIPolicy(policy string) error {
	switch APIPolicy(policy) {
	case APIPolicyLatestCompatible, APIPolicyPinned, APIPolicyLatest:
		return nil
	default:
		return fmt.Errorf("unsupported api policy %q", policy)
	}
}

func ResolveCurrentContext(cfg Config, override ContextOverride) (string, Context, error) {
	name := override.Name
	if name == "" {
		name = cfg.CurrentContext
	}
	if name == "" {
		switch len(cfg.Contexts) {
		case 0:
			return "", Context{}, errors.New("no contexts configured")
		case 1:
			for only := range cfg.Contexts {
				name = only
			}
		default:
			return "", Context{}, fmt.Errorf("no current context configured; available contexts: %s", strings.Join(cfg.ContextNames(), ", "))
		}
	}

	context, ok := cfg.Contexts[name]
	if !ok {
		return "", Context{}, fmt.Errorf("context %q does not exist", name)
	}
	return name, context, nil
}

func ContextOverrideFromEnv(getenv func(string) string) ContextOverride {
	if getenv == nil {
		getenv = os.Getenv
	}
	return ContextOverride{Name: getenv(EnvContext)}
}

func (c Config) ContextNames() []string {
	names := make([]string, 0, len(c.Contexts))
	for name := range c.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
