package integration_test

import (
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func TestAuthFreshHelpHasNoEmbeddedCommands(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "https://invalid.example/catalogue")
	out, trace, err := executeExternalCLI(t, []string{"--config-dir", t.TempDir(), "auth", "--help"})
	if err != nil || trace != "" || !strings.Contains(out, "plugins sync --service auth") || strings.Contains(out, "clients") {
		t.Fatalf("unprepared Auth help: %s %s %v", out, trace, err)
	}
}

func TestAuthMissingReleaseNeverFallsBack(t *testing.T) {
	catalogue := filepath.Join(t.TempDir(), "empty.json")
	writeAuthCatalogue(t, catalogue)
	t.Setenv("FCTL_PLUGIN_CATALOGUE", catalogue)
	out, _, err := executeExternalCLI(t, []string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--auth-url", "http://127.0.0.1:1", "auth", "clients", "list"})
	// An explicit catalogue first discovers the exact service version. Use a real
	// fixture below to distinguish missing release from a failed service endpoint.
	if err == nil || out != "" {
		t.Fatalf("Auth without a release executed: %s %v", out, err)
	}
	var version atomic.Value
	version.Store("1.0.0")
	var mutations atomic.Int32
	server := authDistributionAPI(t, &version, &mutations)
	out, _, err = executeExternalCLI(t, append(authDistributionArgs(t.TempDir(), server.URL), "auth", "clients", "create"))
	if !errors.Is(err, pluginmanager.ErrNoRelease) || out != "" || mutations.Load() != 0 {
		t.Fatalf("missing Auth release fell back: %s %v writes=%d", out, err, mutations.Load())
	}
}

func TestAuthFirstCommandDiscoversAndCachesExecutable(t *testing.T) {
	registry, reads := authOCIRegistry(t)
	manager, err := pluginmanager.New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	binary := buildDistributionAuth(t, "1.0.0")
	release := publishDistributionAuth(t, manager, registry.URL, binary, "1.0.0", 1)
	catalogue := filepath.Join(t.TempDir(), "catalogue.json")
	writeAuthCatalogue(t, catalogue, release)
	t.Setenv("FCTL_PLUGIN_CATALOGUE", catalogue)
	var version atomic.Value
	version.Store("1.0.0")
	var mutations atomic.Int32
	server := authDistributionAPI(t, &version, &mutations)
	args := authDistributionArgs(t.TempDir(), server.URL)
	out := runAuthDistribution(t, args, "auth", "clients", "list")
	if !strings.Contains(out, "distributed-client") || reads.Load() != 3 {
		t.Fatalf("automatic Auth preparation: %s reads=%d", out, reads.Load())
	}
	out = runAuthDistribution(t, args, "plugins", "show", "--service", "auth")
	if !strings.Contains(out, `"service": "auth"`) {
		t.Fatalf("missing prepared lock: %s", out)
	}
	registry.Close()
	server.Close()
	assertAuthOffline(t, args)
}

func TestAuthUnpreparedFlagsUseStandardParsing(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "https://invalid.example/catalogue")
	for _, tail := range [][]string{{"auth"}, {"auth", "--help=true"}, {"auth", "--help"}} {
		args := append([]string{"--config-dir", t.TempDir()}, tail...)
		out, trace, err := executeExternalCLI(t, args)
		if err != nil || trace != "" || !strings.Contains(out, "plugins sync --service auth") {
			t.Fatalf("unprepared help %v: %s %s %v", tail, out, trace, err)
		}
	}
	out, _, err := executeExternalCLI(t, []string{"--config-dir", t.TempDir(), "auth", "--typo", "--help"})
	if err == nil || !strings.Contains(err.Error(), "unknown flag: --typo") || out != "" {
		t.Fatalf("unknown flag bypassed validation: %s %v", out, err)
	}
}
