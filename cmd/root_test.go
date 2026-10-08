package cmd_test

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/cmd"
)

func executeRoot(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.ExecuteContext(t.Context())
	return stdout.String(), stderr.String(), err
}

func TestRootHelpWithoutHomeOrProfile(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, args := range [][]string{{}, {"--help"}, {"help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, stderr, err := executeRoot(t, args...)
			if err != nil {
				t.Fatalf("help failed without a home or profile: %v", err)
			}
			for _, want := range []string{"Formance Control CLI", "Usage:", "completion", "help", "version"} {
				if !strings.Contains(stdout, want) {
					t.Errorf("help missing %q: %s", want, stdout)
				}
			}
			if stderr != "" {
				t.Errorf("unexpected stderr: %q", stderr)
			}
		})
	}
}

func TestRootDefaultVersion(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"command", []string{"version"}, "fctl v4.0.0-dev (commit: -, built: -)\n"},
		{"flag", []string{"--version"}, "fctl version v4.0.0-dev\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := executeRoot(t, tc.args...)
			if err != nil {
				t.Fatalf("version failed: %v", err)
			}
			if stdout != tc.want {
				t.Errorf("stdout = %q, want %q", stdout, tc.want)
			}
			if stderr != "" {
				t.Errorf("unexpected stderr: %q", stderr)
			}
		})
	}
}

func TestRootErrorsDoNotPrintUsageOrErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"unknown command", []string{"does-not-exist"}, `unknown command "does-not-exist" for "fctl"`},
		{"version argument", []string{"version", "extra"}, `unknown command "extra" for "fctl version"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := executeRoot(t, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want containing %q", err, tc.want)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("error printed output: stdout=%q stderr=%q", stdout, stderr)
			}
		})
	}
}

func TestRootCompletionOnlyOffersCurrentCommands(t *testing.T) {
	stdout, _, err := executeRoot(t, "__complete", "")
	if err != nil {
		t.Fatalf("completion failed: %v", err)
	}
	var names []string
	for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
		if strings.HasPrefix(line, ":") {
			continue
		}
		name, _, _ := strings.Cut(line, "\t")
		names = append(names, name)
	}
	slices.Sort(names)
	if want := []string{"completion", "help", "version"}; !slices.Equal(names, want) {
		t.Errorf("completed commands = %v, want %v", names, want)
	}
}

func TestRootGeneratesShellCompletions(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			stdout, stderr, err := executeRoot(t, "completion", shell)
			if err != nil {
				t.Fatalf("generate completion: %v", err)
			}
			if !strings.Contains(stdout, "fctl") {
				t.Errorf("completion script does not reference fctl: %q", stdout)
			}
			if stderr != "" {
				t.Errorf("unexpected stderr: %q", stderr)
			}
		})
	}
}

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestRootVersionReturnsWriterError(t *testing.T) {
	want := errors.New("version output unavailable")
	root := cmd.NewRootCommand()
	root.SetArgs([]string{"version"})
	root.SetOut(failingWriter{err: want})
	var stderr bytes.Buffer
	root.SetErr(&stderr)
	if err := root.ExecuteContext(t.Context()); !errors.Is(err, want) {
		t.Errorf("error = %v, want %v", err, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("unexpected stderr: %q", stderr.String())
	}
}
