package ledger

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestImportLogsServiceSelectsResolvedHandler(t *testing.T) {
	service := ImportLogsService{
		Handlers: []ImportLogsHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ImportLogsInput) (ImportLogsOutput, error) {
					if input.Ledger != "default" || string(input.Data) != "entry\n" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ImportLogsOutput{Ledger: input.Ledger, Imported: true}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), ImportLogsInput{Ledger: "default", Data: []byte("entry\n")})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.Ledger != "default" || !output.Imported {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestImportLogsServiceRequiresLedger(t *testing.T) {
	service := ImportLogsService{
		Handlers: []ImportLogsHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), ImportLogsInput{Data: []byte("entry\n")}); err == nil {
		t.Fatal("expected ledger validation error")
	}
}

func TestImportLogsServiceRequiresFileOrData(t *testing.T) {
	service := ImportLogsService{
		Handlers: []ImportLogsHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), ImportLogsInput{Ledger: "default"}); err == nil {
		t.Fatal("expected file validation error")
	}
}

func TestImportOffsetAfterLogID(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "logs-*.jsonl")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString("{\"id\":1}\n{\"id\":2}\n{\"id\":3}\n")
	if err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("seek temp file: %v", err)
	}

	offset, err := importOffsetAfterLogID(file, big.NewInt(2))
	if err != nil {
		t.Fatalf("find offset: %v", err)
	}
	if offset != int64(len("{\"id\":1}\n{\"id\":2}\n")) {
		t.Fatalf("unexpected offset: %d", offset)
	}
}
