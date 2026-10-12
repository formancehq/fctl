package auth_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/auth"
)

func TestManifestOfflineAndSDKContract(t *testing.T) {
	manifest, err := auth.New(nil).GetManifest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip pluginsdk.Manifest
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip.Name != "auth" || roundTrip.Version != "1.0.0" || roundTrip.Service != "auth" || roundTrip.Root.Target != "stack" || roundTrip.Root.Service != "auth" {
		t.Fatalf("manifest = %s", encoded)
	}
	wantPaths := []string{"auth clients create", "auth clients list", "auth clients show", "auth clients update", "auth clients delete", "auth clients secrets create", "auth clients secrets delete", "auth clients users list", "auth clients users show", "auth users list", "auth users show"}
	gotPaths := manifestCommandPaths(t, roundTrip, roundTrip.Root, nil)
	slices.Sort(gotPaths)
	slices.Sort(wantPaths)
	if !slices.Equal(gotPaths, wantPaths) {
		t.Fatalf("paths = %v, want %v", gotPaths, wantPaths)
	}
}
