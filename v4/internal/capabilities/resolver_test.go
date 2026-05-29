package capabilities

import (
	"errors"
	"testing"
)

func TestResolveAPIVersionSelectsHighestCompatible(t *testing.T) {
	selected, err := ResolveAPIVersion(VersionResolutionRequest{
		Product:          "ledger",
		Feature:          "listTransactions",
		ComponentVersion: "2.3.4",
		Compatibility: ComponentCompatibility{
			{Product: "ledger", Range: ">=2.0.0 <3.0.0", APIVersions: []APIVersion{"v1", "v2"}},
		},
		HandlerVersions: []APIVersion{"v1", "v2", "v3"},
		Policy:          VersionPolicyLatestCompatible,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if selected != "v2" {
		t.Fatalf("expected v2, got %q", selected)
	}
}

func TestResolveAPIVersionHonorsPinned(t *testing.T) {
	selected, err := ResolveAPIVersion(VersionResolutionRequest{
		Product:          "ledger",
		Feature:          "listTransactions",
		ComponentVersion: "2.3.4",
		Compatibility: ComponentCompatibility{
			{Product: "ledger", Range: ">=2.0.0 <3.0.0", APIVersions: []APIVersion{"v1", "v2"}},
		},
		HandlerVersions: []APIVersion{"v1", "v2"},
		Policy:          VersionPolicyPinned,
		PinnedVersion:   "v1",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if selected != "v1" {
		t.Fatalf("expected v1, got %q", selected)
	}
}

func TestResolveAPIVersionReturnsUnsupportedFeatureError(t *testing.T) {
	_, err := ResolveAPIVersion(VersionResolutionRequest{
		Product:          "ledger",
		Feature:          "explainTransaction",
		ComponentVersion: "2.3.4",
		Compatibility: ComponentCompatibility{
			{Product: "ledger", Range: ">=2.0.0 <3.0.0", APIVersions: []APIVersion{"v1", "v2"}},
		},
		HandlerVersions: []APIVersion{"v3"},
		Policy:          VersionPolicyLatestCompatible,
	})
	if err == nil {
		t.Fatal("expected unsupported feature error")
	}
	var unsupported *UnsupportedFeatureError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedFeatureError, got %T %v", err, err)
	}
}
