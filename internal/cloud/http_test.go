package cloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"golang.org/x/oauth2"
)

func TestSafeOAuthErrorCodes(t *testing.T) {
	for _, code := range []string{
		"invalid_grant", "invalid_client", "invalid_scope", "invalid_request",
		"access_denied", "expired_token", "unauthorized_client", "unsupported_grant_type",
		"authorization_pending", "slow_down", "server_error", "temporarily_unavailable",
		"", "provider-secret", "invalid_grant\nsecret",
	} {
		t.Run(code, func(t *testing.T) {
			wantSuffix := ": " + code
			if code == "" || code == "provider-secret" || code == "invalid_grant\nsecret" {
				wantSuffix = ""
			}
			checkSafeOAuthCode(t, code, wantSuffix)
		})
	}
}

func checkSafeOAuthCode(t *testing.T, code, wantSuffix string) {
	t.Helper()
	for _, response := range []*http.Response{nil, {StatusCode: http.StatusBadRequest}} {
		retrieve := &oauth2.RetrieveError{Response: response, ErrorCode: code,
			ErrorDescription: "description-secret", ErrorURI: "https://secret.invalid", Body: []byte("body-secret")}
		wrapped := fmt.Errorf("wrapper-secret: %w", retrieve)
		want := "token failed"
		if response != nil {
			want += " (HTTP 400)"
		}
		if got := safeError(t.Context(), "token", wrapped).Error(); got != want+wantSuffix {
			t.Fatalf("unsafe or incorrect OAuth error: %q", got)
		}
	}
}

func TestSafeOAuthErrorCancellationTakesPrecedence(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := safeError(ctx, "token", &oauth2.RetrieveError{ErrorCode: "invalid_grant"})
	if !errors.Is(err, context.Canceled) || err.Error() != "token: context canceled" {
		t.Fatalf("OAuth code masked cancellation: %v", err)
	}
}

func TestSafeErrorPreservesOnlyTrustedInnerLabels(t *testing.T) {
	provider := &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusBadRequest}, ErrorCode: "invalid_grant", ErrorDescription: "description-secret", Body: []byte("body-secret")}
	inner := safeError(t.Context(), "Membership refresh", provider)
	wrapped := fmt.Errorf("unsafe wrapper-secret: %w", inner)
	got := safeError(t.Context(), "coordinate cloud authentication", wrapped)
	if got.Error() != "coordinate cloud authentication: Membership refresh failed (HTTP 400): invalid_grant" {
		t.Fatalf("trusted error lost its label or leaked provider/wrapper data: %v", got)
	}
}
