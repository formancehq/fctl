package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func regularStdinFile(t *testing.T, contents string) *os.File {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdin-*.num")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := io.WriteString(file, contents); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestPluginStdinRegularFileRedirectEOF(t *testing.T) {
	t.Parallel()
	for _, contents := range []string{"", "send [USD 10] (source = @world destination = @alice)\n"} {
		t.Run(contents, func(t *testing.T) {
			t.Parallel()
			file := regularStdinFile(t, contents)
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			cmd := &cobra.Command{}
			cmd.SetContext(ctx)
			cmd.SetIn(file)
			request, err := ReadPluginInput(cmd, &pluginsdk.FileSpec{ReadArgument: new(0), ReadFormat: "string"}, pluginsdk.ExecuteRequest{Args: []string{"-"}})
			if err != nil {
				t.Fatalf("redirected regular stdin did not finish at EOF: %v", err)
			}
			var script string
			if err := json.Unmarshal(request.Body, &script); err != nil || script != contents {
				t.Fatalf("redirected Numscript changed: %q (%v)", script, err)
			}
			if _, err := file.Seek(0, io.SeekStart); err != nil {
				t.Fatalf("the host closed redirected stdin: %v", err)
			}
		})
	}
}

func TestRegularPluginStdinBoundsAndCancellation(t *testing.T) {
	t.Parallel()
	file := regularStdinFile(t, "send [USD 10]")
	data, err := readPluginStdin(t.Context(), file)
	if err != nil || string(data) != "send [USD 10]" {
		t.Fatalf("regular input = %q (%v)", data, err)
	}
	large := regularStdinFile(t, strings.Repeat("x", maxPluginInput+1))
	if _, err := readPluginStdin(t.Context(), large); err == nil {
		t.Fatal("regular stdin bypassed the input bound")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := readPluginStdin(ctx, file); !errors.Is(err, context.Canceled) {
		t.Fatalf("regular stdin lost cancellation: %v", err)
	}
	if data, err := readPluginStdin(t.Context(), strings.NewReader("stream")); err != nil || string(data) != "stream" {
		t.Fatalf("non-file stream changed: %q (%v)", data, err)
	}
}
