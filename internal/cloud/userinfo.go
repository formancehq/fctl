package cloud

import (
	"context"
	"errors"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Membership refresh grants return no ID token. A historical signed ID token
// binds the login subject; a current signed access JWT and authenticated,
// same-issuer UserInfo response establish current access. Never persist these
// unsigned claims: retrieve them again while the historical ID token is expired.
func (m *targetManager) userInfoAndSave(ctx context.Context, root *Session, save func(*Session) error) (*Session, identityClaims, error) {
	previousID, required := root.IDToken, root.RequireUserInfo
	root.RequireUserInfo = true
	claims, err := m.currentUserInfo(ctx, root)
	if !required || previousID != root.IDToken {
		if saveErr := m.saveRoot(root, save); saveErr != nil {
			return nil, identityClaims{}, saveErr
		}
	}
	return root, claims, err
}

func (m *targetManager) currentUserInfo(ctx context.Context, root *Session) (identityClaims, error) {
	_, historical, err := verifyIdentity(ctx, m.provider, m.options, root.IDToken, true)
	if err != nil {
		return identityClaims{}, err
	}
	token := cleanToken(root.MembershipToken)
	if err := boundToken(token, time.Now()); err != nil {
		root.IDToken = ""
		return identityClaims{}, err
	}
	access, err := m.provider.Verifier(&oidc.Config{ClientID: m.options.ClientID}).Verify(ctx, token.AccessToken)
	if err != nil {
		if ctx.Err() == nil {
			root.IDToken = ""
		}
		return identityClaims{}, identityUserInfoError(ctx)
	}
	if access.Subject == "" || access.Subject != historical.Subject {
		root.IDToken = ""
		return identityClaims{}, errors.New("root access subject differs from verified login identity; log in again")
	}
	token.Expiry = minExpiry(token.Expiry, access.Expiry)
	authCtx := oidc.ClientContext(ctx, authClient(ctx, m.base, m.options.Issuer))
	info, err := m.provider.UserInfo(authCtx, oauth2.StaticTokenSource(token))
	if err != nil {
		return identityClaims{}, identityUserInfoError(ctx)
	}
	if info.Subject != access.Subject {
		root.IDToken = ""
		return identityClaims{}, errors.New("membership userinfo subject differs from verified access token; log in again")
	}
	var claims identityClaims
	if err := info.Claims(&claims); err != nil {
		root.IDToken = ""
		return identityClaims{}, errors.New("invalid Membership userinfo claims")
	}
	if _, err := verifiedIdentityInfo(m.options, claims); err != nil {
		root.IDToken = ""
		return identityClaims{}, err
	}
	root.MembershipToken = cleanToken(token)
	return claims, nil
}

func identityUserInfoError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errors.New("cannot renew cloud identity from Membership userinfo; log in again")
}
