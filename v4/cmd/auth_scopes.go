package cmd

import (
	v4auth "github.com/formancehq/fctl/v4/internal/auth"
	v4config "github.com/formancehq/fctl/v4/internal/config"
)

func clientCredentialsScopesForPlatform(platform bool) []string {
	if !platform {
		return nil
	}
	return cloneOrganizationScopes()
}

func clientCredentialsScopesForContext(context v4config.Context) []string {
	switch context.Kind {
	case v4config.ContextKindCloud, v4config.ContextKindCloudStack:
		return cloneOrganizationScopes()
	default:
		return nil
	}
}

func cloneOrganizationScopes() []string {
	return append([]string(nil), v4auth.OrganizationScopes...)
}
