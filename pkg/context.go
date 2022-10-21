package fctl

import (
	"context"
)

type configKeySymbol struct{}

var configContextKey = configKeySymbol{}

func WithConfig(ctx context.Context, config *Config) context.Context {
	return context.WithValue(ctx, configContextKey, config)
}

func ConfigFromContext(ctx context.Context) *Config {
	v := ctx.Value(configContextKey)
	if v == nil {
		panic("no config selected")
	}
	return v.(*Config)
}

type configManagerKeySymbol struct{}

var configManagerContextKey = configManagerKeySymbol{}

func WithConfigManager(ctx context.Context, configManager *ConfigManager) context.Context {
	return context.WithValue(ctx, configManagerContextKey, configManager)
}

func ConfigManagerFromContext(ctx context.Context) *ConfigManager {
	v := ctx.Value(configManagerContextKey)
	if v == nil {
		panic("no config manager defined")
	}
	return v.(*ConfigManager)
}

type currentProfileKeySymbol struct{}

var currentProfileContextKey = currentProfileKeySymbol{}

func WithCurrentProfile(ctx context.Context, profile *Profile) context.Context {
	return context.WithValue(ctx, currentProfileContextKey, profile)
}

func CurrentProfileFromContext(ctx context.Context) *Profile {
	v := ctx.Value(currentProfileContextKey)
	if v == nil {
		panic("no profile selected")
	}
	return v.(*Profile)
}

type currentProfileNameKeySymbol struct{}

var currentProfileNameContextKey = currentProfileNameKeySymbol{}

func WithCurrentProfileName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, currentProfileNameContextKey, name)
}

func CurrentProfileNameFromContext(ctx context.Context) string {
	v := ctx.Value(currentProfileNameContextKey)
	if v == nil {
		panic("no profile selected")
	}
	return v.(string)
}

type debugKeySymbol struct{}

var debugContextKey = debugKeySymbol{}

func WithDebug(ctx context.Context, debug bool) context.Context {
	return context.WithValue(ctx, debugContextKey, debug)
}

func IsDebugFromContext(ctx context.Context) bool {
	v := ctx.Value(debugContextKey)
	if v == nil {
		return false
	}
	return v.(bool)
}

type organizationKeySymbol struct{}

var organizationContextKey = organizationKeySymbol{}

func WithOrganization(ctx context.Context, organization string) context.Context {
	return context.WithValue(ctx, organizationContextKey, organization)
}

func OrganizationFromContext(ctx context.Context) string {
	v := ctx.Value(organizationContextKey)
	if v == nil {
		panic("no organization selected")
	}
	return v.(string)
}

type stackKeySymbol struct{}

var stackContextKey = stackKeySymbol{}

func WithStack(ctx context.Context, stack string) context.Context {
	return context.WithValue(ctx, stackContextKey, stack)
}

func StackFromContext(ctx context.Context) string {
	v := ctx.Value(stackContextKey)
	if v == nil {
		panic("no stack selected")
	}
	return v.(string)
}

type insecureTLSKeySymbol struct{}

var insecureTLSContextKey = insecureTLSKeySymbol{}

func WithInsecureTLS(ctx context.Context, insecureTLS bool) context.Context {
	return context.WithValue(ctx, insecureTLSContextKey, insecureTLS)
}

func InsecureTLSFromContext(ctx context.Context) bool {
	v := ctx.Value(insecureTLSContextKey)
	if v == nil {
		return false
	}
	return v.(bool)
}
