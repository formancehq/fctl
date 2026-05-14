package ledger

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestListSchemasServiceSelectsResolvedHandler(t *testing.T) {
	service := ListSchemasService{
		Handlers: []ListSchemasHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input ListSchemasInput) (ListSchemasOutput, error) {
					if input.Ledger != "default" || input.PageSize != 15 {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListSchemasOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), ListSchemasInput{Ledger: "default", PageSize: 15})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.PageSize != 15 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetSchemaServiceRequiresVersion(t *testing.T) {
	service := GetSchemaService{
		Handlers: []GetSchemaHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetSchemaInput{Ledger: "default"}); err == nil {
		t.Fatal("expected schema version validation error")
	}
}

func TestInsertSchemaServiceRequiresData(t *testing.T) {
	service := InsertSchemaService{
		Handlers: []InsertSchemaHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), InsertSchemaInput{Ledger: "default", Version: "v1"}); err == nil {
		t.Fatal("expected schema data validation error")
	}
}
