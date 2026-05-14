package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"
)

type StackClient interface {
	ListStacks(context.Context, operations.ListStacksRequest, ...operations.Option) (*operations.ListStacksResponse, error)
	GetStack(context.Context, operations.GetStackRequest, ...operations.Option) (*operations.GetStackResponse, error)
}

type StackSummary struct {
	ID             string            `json:"id" yaml:"id"`
	Name           string            `json:"name" yaml:"name"`
	OrganizationID string            `json:"organizationID" yaml:"organizationID"`
	URI            string            `json:"uri,omitempty" yaml:"uri,omitempty"`
	RegionID       string            `json:"regionID,omitempty" yaml:"regionID,omitempty"`
	Version        string            `json:"version,omitempty" yaml:"version,omitempty"`
	Status         string            `json:"status" yaml:"status"`
	State          string            `json:"state" yaml:"state"`
	ExpectedStatus string            `json:"expectedStatus" yaml:"expectedStatus"`
	Reachable      bool              `json:"reachable" yaml:"reachable"`
	Metadata       map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt      *time.Time        `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt      *time.Time        `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	DeletedAt      *time.Time        `json:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DisabledAt     *time.Time        `json:"disabledAt,omitempty" yaml:"disabledAt,omitempty"`
}

type ListStacksInput struct {
	OrganizationID string
	All            bool
}

type ListStacksOutput struct {
	OrganizationID string         `json:"organizationID" yaml:"organizationID"`
	Stacks         []StackSummary `json:"stacks" yaml:"stacks"`
}

type StackIDInput struct {
	OrganizationID string
	StackID        string
}

type StackOutput struct {
	OrganizationID string       `json:"organizationID" yaml:"organizationID"`
	Stack          StackSummary `json:"stack" yaml:"stack"`
}

type ListStacksService struct {
	Client StackClient
}

func (s ListStacksService) Run(ctx context.Context, input ListStacksInput) (ListStacksOutput, error) {
	if s.Client == nil {
		return ListStacksOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return ListStacksOutput{}, fmt.Errorf("organization id is required")
	}
	response, err := s.Client.ListStacks(ctx, operations.ListStacksRequest{
		OrganizationID: input.OrganizationID,
		All:            &input.All,
	})
	if err != nil {
		return ListStacksOutput{}, err
	}
	data := response.GetListStacksResponse().GetData()
	stacks := make([]StackSummary, 0, len(data))
	for i := range data {
		stacks = append(stacks, stackSummary(&data[i]))
	}
	return ListStacksOutput{OrganizationID: input.OrganizationID, Stacks: stacks}, nil
}

type ReadStackService struct {
	Client StackClient
}

func (s ReadStackService) Run(ctx context.Context, input StackIDInput) (StackOutput, error) {
	if s.Client == nil {
		return StackOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return StackOutput{}, fmt.Errorf("organization id is required")
	}
	if input.StackID == "" {
		return StackOutput{}, fmt.Errorf("stack id is required")
	}
	response, err := s.Client.GetStack(ctx, operations.GetStackRequest{
		OrganizationID: input.OrganizationID,
		StackID:        input.StackID,
	})
	if err != nil {
		return StackOutput{}, err
	}
	if response.GetReadStackResponse().GetData() == nil {
		return StackOutput{}, fmt.Errorf("cloud_stacks show returned no stack")
	}
	return StackOutput{
		OrganizationID: input.OrganizationID,
		Stack:          stackSummary(response.GetReadStackResponse().GetData()),
	}, nil
}

func stackSummary(stack *components.Stack) StackSummary {
	if stack == nil {
		return StackSummary{}
	}
	summary := StackSummary{
		ID:             stack.ID,
		Name:           stack.Name,
		OrganizationID: stack.OrganizationID,
		URI:            stack.URI,
		RegionID:       stack.RegionID,
		Status:         string(stack.Status),
		State:          string(stack.State),
		ExpectedStatus: string(stack.ExpectedStatus),
		Reachable:      stack.Reachable,
		Metadata:       stack.Metadata,
		CreatedAt:      stack.CreatedAt,
		UpdatedAt:      stack.UpdatedAt,
		DeletedAt:      stack.DeletedAt,
		DisabledAt:     stack.DisabledAt,
	}
	if stack.Version != nil {
		summary.Version = *stack.Version
	}
	return summary
}
