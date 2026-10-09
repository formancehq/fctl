// Package browser opens login verification URLs in the operating system's default browser.
package browser

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	httpclient "github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

const openTimeout = 5 * time.Second

type commandRunner func(context.Context, string, ...string) error

// Open launches the default browser quietly. HTTPS and loopback HTTP URLs are
// accepted, including verification query parameters. The caller handles fallback.
func Open(ctx context.Context, target string) error {
	return open(ctx, target, runtime.GOOS, runCommand)
}

func open(ctx context.Context, target, goos string, run commandRunner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateURL(target); err != nil {
		return err
	}
	name, args, err := browserCommand(goos, target)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, openTimeout)
	defer cancel()
	err = run(ctx, name, args...)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("open default browser: %w", err)
	}
	return nil
}

func validateURL(target string) error {
	u, err := url.Parse(target)
	if err != nil || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("browser URL must be an absolute HTTP(S) URL without credentials")
	}
	// Shared endpoint validation forbids queries; browser verification URLs need them.
	// Validate the same origin/path while passing the full URL to the OS opener.
	u.RawQuery = ""
	u.ForceQuery = false
	return httpclient.ValidateSecureURL(u.String())
}

func browserCommand(goos, target string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{target}, nil
	case "linux":
		return "xdg-open", []string{target}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", target}, nil
	default:
		return "", nil, fmt.Errorf("default browser opening is unsupported on %s", goos)
	}
}

func runCommand(ctx context.Context, name string, args ...string) error {
	// Both the executable and argument structure are selected from a fixed OS table.
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // Fixed OS opener; validated URL is a separate argument, never shell code.
	// Nil output streams go directly to the null device, avoiding pipe-copy
	// goroutines that could outlive cancellation when an opener forks.
	return cmd.Run()
}
