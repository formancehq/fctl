package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
)

func TestRunAccountQueryServiceSelectsResolvedHandler(t *testing.T) {
	service := RunAccountQueryService{
		Handlers: []RunAccountQueryHandler{
			{
				APIVersion: "v2",
				Run: func(_ context.Context, input RunAccountQueryInput) (RunAccountQueryOutput, error) {
					if input.Ledger != "default" || input.QueryID != "active_accounts" || input.SchemaVersion != "v1" {
						t.Fatalf("unexpected input: %#v", input)
					}
					return RunAccountQueryOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v2"})
			return "v2", nil
		},
	}

	output, err := service.Run(context.Background(), RunAccountQueryInput{
		Ledger:        "default",
		QueryID:       "active_accounts",
		SchemaVersion: "v1",
		PageSize:      10,
	})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v2" || output.QueryID != "active_accounts" || output.SchemaVersion != "v1" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestRunAccountQueryServiceRequiresInputs(t *testing.T) {
	service := RunAccountQueryService{
		Handlers: []RunAccountQueryHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), RunAccountQueryInput{}); err == nil || err.Error() != "ledger is required" {
		t.Fatalf("expected ledger required error, got %v", err)
	}
	if _, err := service.Run(context.Background(), RunAccountQueryInput{Ledger: "default"}); err == nil || err.Error() != "query id is required" {
		t.Fatalf("expected query id required error, got %v", err)
	}
	if _, err := service.Run(context.Background(), RunAccountQueryInput{Ledger: "default", QueryID: "q"}); err == nil || err.Error() != "schema version is required" {
		t.Fatalf("expected schema version required error, got %v", err)
	}
}

func TestRunAccountQueryServiceReturnsResolverError(t *testing.T) {
	expected := errors.New("requires ledger API v2")
	service := RunAccountQueryService{
		Handlers: []RunAccountQueryHandler{{APIVersion: "v2"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			return "", expected
		},
	}

	_, err := service.Run(context.Background(), RunAccountQueryInput{
		Ledger:        "default",
		QueryID:       "active_accounts",
		SchemaVersion: "v1",
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected resolver error, got %v", err)
	}
}

func TestRunAccountQueryRequestMapsCanonicalInput(t *testing.T) {
	pit := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	request := toV2RunAccountQueryRequest(RunAccountQueryInput{
		Ledger:        "default",
		QueryID:       "active_accounts",
		SchemaVersion: "v1",
		PageSize:      10,
		Cursor:        "cursor",
		Expand:        "metadata",
		Pit:           &pit,
		Reverse:       true,
		Sort:          "address:asc",
		Vars:          map[string]string{"segment": "vip"},
	})

	if request.Ledger != "default" || request.ID != "active_accounts" || request.SchemaVersion != "v1" {
		t.Fatalf("unexpected request identity: %#v", request)
	}
	if request.PageSize == nil || *request.PageSize != 10 || request.Cursor == nil || *request.Cursor != "cursor" {
		t.Fatalf("unexpected pagination mapping: %#v", request)
	}
	if request.Reverse == nil || !*request.Reverse || request.Sort == nil || *request.Sort != "address:asc" {
		t.Fatalf("unexpected order mapping: %#v", request)
	}
	if request.RequestBody.Vars["segment"] != "vip" {
		t.Fatalf("unexpected vars mapping: %#v", request.RequestBody.Vars)
	}
	if request.RequestBody.Params == nil || request.RequestBody.Params.QueryTemplateAccountParams == nil {
		t.Fatalf("expected account query params, got %#v", request.RequestBody.Params)
	}
	params := request.RequestBody.Params.QueryTemplateAccountParams
	if params.Resource == nil || *params.Resource != shared.V2QueryParamsResourceAccounts {
		t.Fatalf("expected account resource, got %#v", params.Resource)
	}
	if params.Pit == nil || !params.Pit.Equal(pit) || params.Expand == nil || *params.Expand != "metadata" {
		t.Fatalf("unexpected params mapping: %#v", params)
	}
}

func TestRunAccountQueryMapping(t *testing.T) {
	next := "next"
	output := fromV2RunAccountQuery(shared.V2AccountsCursorResponseCursor{
		Data: []shared.V2Account{
			{Address: "users:123", Metadata: map[string]string{"tier": "gold"}},
		},
		HasMore:  true,
		Next:     &next,
		PageSize: 10,
	})
	if len(output.Accounts) != 1 || output.Accounts[0].Address != "users:123" || output.Accounts[0].Metadata["tier"] != "gold" {
		t.Fatalf("unexpected accounts: %#v", output.Accounts)
	}
	if !output.HasMore || output.Next == nil || *output.Next != "next" || output.PageSize != 10 {
		t.Fatalf("unexpected cursor mapping: %#v", output)
	}
}
