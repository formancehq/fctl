package cloud

import (
	"context"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Historical providers put stack accesses in the signed ID token. Membership
// now supplies an ordinary ID token and a resource-bound, signed access JWT.
// Both contracts require verified identity; no URL comes from unsigned claims.
func verifyTargetSession(ctx context.Context, provider *oidc.Provider, target targetAccess, child *Session, storedID, storedAccess bool) (bool, error) {
	if child.MembershipToken == nil {
		return false, authenticationFailure("missing scoped Membership token")
	}
	options, err := child.Options.normalized()
	if err != nil || options != target.Options || child.StackURL != target.URI {
		return false, authenticationFailure("scoped session target differs from verified target")
	}
	if err := validateEndpoint(target.URI); err != nil {
		return false, authenticationFailure("invalid scoped stack URL")
	}
	claims, id, err := verifyIdentity(ctx, provider, target.Options, child.IDToken, storedID)
	if err != nil {
		return false, err
	}
	if target.Subject != "" && id.Subject != target.Subject {
		return false, authenticationFailure("scoped identity subject differs from root identity")
	}
	if err := verifyScopedTokenHash(id, child.MembershipToken.AccessToken, storedID); err != nil {
		return false, err
	}
	return verifyTargetAccess(ctx, provider, target, child, claims, id.Subject, storedAccess)
}

func verifyTargetAccess(ctx context.Context, provider *oidc.Provider, target targetAccess, child *Session, claims identityClaims, subject string, storedAccess bool) (bool, error) {
	historicalMatches := true
	if len(claims.Organizations) != 0 {
		var err error
		historicalMatches, err = historicalStackClaimsMatch(target, claims)
		if err != nil {
			return false, err
		}
		if strings.Count(child.MembershipToken.AccessToken, ".") != 2 {
			return historicalMatches, nil
		}
	}
	matches, err := verifyStackAccess(ctx, provider, target, child, subject, storedAccess)
	return matches && historicalMatches, err
}

func verifyScopedTokenHash(id *oidc.IDToken, access string, storedID bool) error {
	// A historical ID token's hash binds the original access token, not a new
	// refresh token response which deliberately omits an ID token.
	if !storedID && id.AccessTokenHash != "" {
		if err := id.VerifyAccessToken(access); err != nil {
			return authenticationFailure("scoped identity does not match its access token")
		}
	}
	return nil
}

func historicalStackClaimsMatch(target targetAccess, claims identityClaims) (bool, error) {
	signed, err := resolveTarget(target.Options, claims, target.Options)
	if err != nil {
		return false, err
	}
	if signed.URI != target.URI {
		return false, authenticationFailure("session stack URL differs from verified claims; log in again")
	}
	return target.Scopes == nil || slices.Equal(signed.Scopes, target.Scopes), nil
}

func verifyStackAccess(ctx context.Context, provider *oidc.Provider, target targetAccess, child *Session, subject string, stored bool) (bool, error) {
	access, err := provider.Verifier(&oidc.Config{ClientID: target.URI + "/api/auth", SkipExpiryCheck: stored}).Verify(ctx, child.MembershipToken.AccessToken)
	if err != nil {
		return false, safeError(ctx, "verify scoped stack access", err)
	}
	var claims struct {
		Organization string `json:"organization_id"`
		Stack        string `json:"stack_id"`
		Scope        string `json:"scope"`
	}
	if err := access.Claims(&claims); err != nil {
		return false, authenticationFailure("invalid scoped stack access claims")
	}
	if subject == "" || access.Subject != subject || claims.Organization != target.Options.Organization || claims.Stack != target.Options.Stack {
		return false, authenticationFailure("scoped access does not authorize the requested identity, organization and stack")
	}
	child.MembershipToken.Expiry = minExpiry(child.MembershipToken.Expiry, access.Expiry)
	granted := stackPermissions(strings.Fields(claims.Scope))
	return len(granted) != 0 && (target.Scopes == nil || slices.Equal(granted, target.Scopes)), nil
}

func stackPermissions(scopes []string) []string {
	read, write := false, false
	for _, scope := range scopes {
		service, permission, ok := strings.Cut(scope, ":")
		if !ok || !validIdentifier(service) {
			return nil
		}
		switch permission {
		case "read":
			read = true
		case "write":
			write = true
		default:
			return nil
		}
	}
	var result []string
	if read {
		result = append(result, "stack:Read")
	}
	if write {
		result = append(result, "stack:Write")
	}
	return result
}
