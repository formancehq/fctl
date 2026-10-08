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
