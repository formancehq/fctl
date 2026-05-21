package plugin

import (
	"testing"
)

func TestFindBestVersion(t *testing.T) {
	p := &RegistryPlugin{
		Repo: "formancehq/ledger",
		Versions: map[string]RegistryPluginVersion{
			"3.0.0": {CompatibleWith: ">= 3.0.0, < 3.3.0"},
			"3.3.0": {CompatibleWith: ">= 3.3.0, < 4.0.0"},
			"4.0.0": {CompatibleWith: ">= 4.0.0"},
		},
	}

	tests := []struct {
		serviceVersion string
		wantVersion    string
		wantErr        bool
	}{
		{"3.1.0", "3.0.0", false},
		{"3.0.0", "3.0.0", false},
		{"3.3.0", "3.3.0", false},
		{"3.5.2", "3.3.0", false},
		{"4.0.0", "4.0.0", false},
		{"2.9.0", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.serviceVersion, func(t *testing.T) {
			got, _, err := p.FindBestVersion(tt.serviceVersion)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.serviceVersion)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.serviceVersion, err)
			}
			if got != tt.wantVersion {
				t.Fatalf("FindBestVersion(%q): got=%q, want %q", tt.serviceVersion, got, tt.wantVersion)
			}
		})
	}
}

func TestBinaryURL(t *testing.T) {
	p := &RegistryPlugin{
		Repo: "formancehq/ledger",
	}

	url := p.BinaryURL("ledger", "3.2.0")

	// We can't test exact URL since it depends on runtime.GOOS/GOARCH
	// but we can verify it contains the expected parts
	if url == "" {
		t.Fatal("expected non-empty URL")
	}

	expected := "https://github.com/formancehq/ledger/releases/download/v3.2.0/fctl-plugin-ledger-"
	if len(url) < len(expected) || url[:len(expected)] != expected {
		t.Fatalf("unexpected URL prefix: %s", url)
	}
}
