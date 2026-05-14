package payments

import (
	"context"
	"testing"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func TestListPoolsServiceSelectsResolvedHandler(t *testing.T) {
	service := ListPoolsService{
		Handlers: []ListPoolsHandler{
			{
				APIVersion: "v1",
				Run: func(context.Context, ListPoolsInput) (ListPoolsOutput, error) {
					t.Fatal("v1 handler should not run")
					return ListPoolsOutput{}, nil
				},
			},
			{
				APIVersion: "v3",
				Run: func(_ context.Context, input ListPoolsInput) (ListPoolsOutput, error) {
					if input.PageSize != 10 {
						t.Fatalf("unexpected input: %#v", input)
					}
					return ListPoolsOutput{PageSize: input.PageSize}, nil
				},
			},
		},
		Resolve: func(_ context.Context, versions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			assertAPIVersions(t, versions, []capabilities.APIVersion{"v1", "v3"})
			return "v3", nil
		},
	}

	output, err := service.Run(context.Background(), ListPoolsInput{PageSize: 10})
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if output.APIVersion != "v3" || output.PageSize != 10 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestGetPoolServiceRequiresID(t *testing.T) {
	service := GetPoolService{
		Handlers: []GetPoolHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), GetPoolInput{}); err == nil {
		t.Fatal("expected pool id validation error")
	}
}

func TestDeletePoolServiceRequiresID(t *testing.T) {
	service := DeletePoolService{
		Handlers: []DeletePoolHandler{{APIVersion: "v3"}},
		Resolve: func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
			t.Fatal("resolver should not run")
			return "", nil
		},
	}

	if _, err := service.Run(context.Background(), DeletePoolInput{}); err == nil {
		t.Fatal("expected pool id validation error")
	}
}
