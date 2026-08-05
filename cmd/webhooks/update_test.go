package webhooks

import "testing"

func TestNewUpdateConfigRequestIncludesSecretFromEnvironment(t *testing.T) {
	t.Setenv("SECRET", "environment-secret")
	cmd := NewUpdateCommand()

	request := newUpdateConfigRequest(cmd, []string{"config-1", "https://example.com", "event"})
	if request.ConfigUser.Secret == nil || *request.ConfigUser.Secret != "environment-secret" {
		t.Fatalf("secret = %v, want environment-secret", request.ConfigUser.Secret)
	}
}

func TestNewUpdateConfigRequestSecretFlagOverridesEnvironment(t *testing.T) {
	t.Setenv("SECRET", "environment-secret")
	cmd := NewUpdateCommand()
	if err := cmd.Flags().Set(secretFlag, "flag-secret"); err != nil {
		t.Fatalf("set secret flag: %v", err)
	}

	request := newUpdateConfigRequest(cmd, []string{"config-1", "https://example.com", "event"})
	if request.ConfigUser.Secret == nil || *request.ConfigUser.Secret != "flag-secret" {
		t.Fatalf("secret = %v, want flag-secret", request.ConfigUser.Secret)
	}
}

func TestNewUpdateConfigRequestOmitsUnspecifiedSecret(t *testing.T) {
	t.Setenv("SECRET", "")
	cmd := NewUpdateCommand()

	request := newUpdateConfigRequest(cmd, []string{"config-1", "https://example.com", "event"})
	if request.ConfigUser.Secret != nil {
		t.Fatalf("secret = %q, want nil", *request.ConfigUser.Secret)
	}
}
