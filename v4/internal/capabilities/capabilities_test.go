package capabilities

import "testing"

func TestMatchVersionRange(t *testing.T) {
	tests := []struct {
		name      string
		rangeExpr string
		version   string
		want      bool
	}{
		{name: "inside lower inclusive upper exclusive", rangeExpr: ">=2.0.0 <3.0.0", version: "2.3.4", want: true},
		{name: "below lower", rangeExpr: ">=2.0.0 <3.0.0", version: "1.9.9", want: false},
		{name: "at upper exclusive", rangeExpr: ">=2.0.0 <3.0.0", version: "3.0.0", want: false},
		{name: "accepts v prefix", rangeExpr: ">=v2.0.0 <v3.0.0", version: "v2.0.0", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchVersionRange(tt.version, tt.rangeExpr)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestAPIVersionsFor(t *testing.T) {
	compatibility := ComponentCompatibility{
		{Product: "ledger", Range: ">=1.0.0 <2.0.0", APIVersions: []APIVersion{"v1"}},
		{Product: "ledger", Range: ">=2.0.0 <3.0.0", APIVersions: []APIVersion{"v1", "v2"}},
		{Product: "payments", Range: ">=3.0.0", APIVersions: []APIVersion{"v1", "v3"}},
	}

	versions, err := compatibility.APIVersionsFor("ledger", "2.3.4")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertAPIVersions(t, versions, []APIVersion{"v1", "v2"})

	versions, err = compatibility.APIVersionsFor("payments", "3.1.0")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertAPIVersions(t, versions, []APIVersion{"v1", "v3"})
}

func TestHighestAPIVersion(t *testing.T) {
	highest, ok := HighestAPIVersion([]APIVersion{"v1", "v3", "v2"})
	if !ok {
		t.Fatal("expected highest api version")
	}
	if highest != "v3" {
		t.Fatalf("expected v3, got %q", highest)
	}
}

func assertAPIVersions(t *testing.T, got []APIVersion, want []APIVersion) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
