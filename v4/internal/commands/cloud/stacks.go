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
	CreateStack(context.Context, operations.CreateStackRequest, ...operations.Option) (*operations.CreateStackResponse, error)
	UpdateStack(context.Context, operations.UpdateStackRequest, ...operations.Option) (*operations.UpdateStackResponse, error)
	DeleteStack(context.Context, operations.DeleteStackRequest, ...operations.Option) (*operations.DeleteStackResponse, error)
	EnableStack(context.Context, operations.EnableStackRequest, ...operations.Option) (*operations.EnableStackResponse, error)
	DisableStack(context.Context, operations.DisableStackRequest, ...operations.Option) (*operations.DisableStackResponse, error)
	RestoreStack(context.Context, operations.RestoreStackRequest, ...operations.Option) (*operations.RestoreStackResponse, error)
	UpgradeStack(context.Context, operations.UpgradeStackRequest, ...operations.Option) (*operations.UpgradeStackResponse, error)
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

type CreateStackInput struct {
	OrganizationID string
	Name           string
	RegionID       string
	Version        string
	Metadata       map[string]string
}

type UpdateStackInput struct {
	OrganizationID string
	StackID        string
	Name           string
	Metadata       map[string]string
}

type DeleteStackInput struct {
	OrganizationID string
	StackID        string
	Force          bool
}

type StackActionInput struct {
	OrganizationID string
	StackID        string
	Version        string
}

type DeleteStackOutput struct {
	OrganizationID string `json:"organizationID" yaml:"organizationID"`
	StackID        string `json:"stackID" yaml:"stackID"`
}

type StackActionOutput struct {
	OrganizationID string        `json:"organizationID" yaml:"organizationID"`
	StackID        string        `json:"stackID" yaml:"stackID"`
	Action         string        `json:"action" yaml:"action"`
	Version        string        `json:"version,omitempty" yaml:"version,omitempty"`
	Stack          *StackSummary `json:"stack,omitempty" yaml:"stack,omitempty"`
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

type CreateStackService struct {
	Client StackClient
}

func (s CreateStackService) Run(ctx context.Context, input CreateStackInput) (StackOutput, error) {
	if s.Client == nil {
		return StackOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return StackOutput{}, fmt.Errorf("organization id is required")
	}
	if input.Name == "" {
		return StackOutput{}, fmt.Errorf("stack name is required")
	}
	if input.RegionID == "" {
		return StackOutput{}, fmt.Errorf("region id is required")
	}
	body := &components.CreateStackRequest{
		Name:     input.Name,
		RegionID: input.RegionID,
		Metadata: input.Metadata,
	}
	if input.Version != "" {
		body.Version = &input.Version
	}
	response, err := s.Client.CreateStack(ctx, operations.CreateStackRequest{
		OrganizationID: input.OrganizationID,
		Body:           body,
	})
	if err != nil {
		return StackOutput{}, err
	}
	if response.GetReadStackResponse().GetData() == nil {
		return StackOutput{}, fmt.Errorf("cloud_stacks create returned no stack")
	}
	return StackOutput{OrganizationID: input.OrganizationID, Stack: stackSummary(response.GetReadStackResponse().GetData())}, nil
}

type UpdateStackService struct {
	Client StackClient
}

func (s UpdateStackService) Run(ctx context.Context, input UpdateStackInput) (StackOutput, error) {
	if s.Client == nil {
		return StackOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return StackOutput{}, fmt.Errorf("organization id is required")
	}
	if input.StackID == "" {
		return StackOutput{}, fmt.Errorf("stack id is required")
	}
	if input.Name == "" {
		return StackOutput{}, fmt.Errorf("stack name is required")
	}
	response, err := s.Client.UpdateStack(ctx, operations.UpdateStackRequest{
		OrganizationID: input.OrganizationID,
		StackID:        input.StackID,
		Body: &components.StackData{
			Name:     input.Name,
			Metadata: input.Metadata,
		},
	})
	if err != nil {
		return StackOutput{}, err
	}
	if response.GetReadStackResponse().GetData() == nil {
		return StackOutput{}, fmt.Errorf("cloud_stacks update returned no stack")
	}
	return StackOutput{OrganizationID: input.OrganizationID, Stack: stackSummary(response.GetReadStackResponse().GetData())}, nil
}

type DeleteStackService struct {
	Client StackClient
}

func (s DeleteStackService) Run(ctx context.Context, input DeleteStackInput) (DeleteStackOutput, error) {
	if s.Client == nil {
		return DeleteStackOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return DeleteStackOutput{}, fmt.Errorf("organization id is required")
	}
	if input.StackID == "" {
		return DeleteStackOutput{}, fmt.Errorf("stack id is required")
	}
	_, err := s.Client.DeleteStack(ctx, operations.DeleteStackRequest{
		OrganizationID: input.OrganizationID,
		StackID:        input.StackID,
		Force:          &input.Force,
	})
	if err != nil {
		return DeleteStackOutput{}, err
	}
	return DeleteStackOutput{OrganizationID: input.OrganizationID, StackID: input.StackID}, nil
}

type StackActionService struct {
	Client StackClient
	Action string
}

func (s StackActionService) Run(ctx context.Context, input StackActionInput) (StackActionOutput, error) {
	if s.Client == nil {
		return StackActionOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return StackActionOutput{}, fmt.Errorf("organization id is required")
	}
	if input.StackID == "" {
		return StackActionOutput{}, fmt.Errorf("stack id is required")
	}
	output := StackActionOutput{
		OrganizationID: input.OrganizationID,
		StackID:        input.StackID,
		Action:         s.Action,
	}
	switch s.Action {
	case "enable":
		_, err := s.Client.EnableStack(ctx, operations.EnableStackRequest{OrganizationID: input.OrganizationID, StackID: input.StackID})
		return output, err
	case "disable":
		_, err := s.Client.DisableStack(ctx, operations.DisableStackRequest{OrganizationID: input.OrganizationID, StackID: input.StackID})
		return output, err
	case "restore":
		response, err := s.Client.RestoreStack(ctx, operations.RestoreStackRequest{OrganizationID: input.OrganizationID, StackID: input.StackID})
		if err != nil {
			return StackActionOutput{}, err
		}
		if response.GetReadStackResponse().GetData() != nil {
			stack := stackSummary(response.GetReadStackResponse().GetData())
			output.Stack = &stack
		}
		return output, nil
	case "upgrade":
		body := &components.StackVersion{}
		if input.Version != "" {
			body.Version = &input.Version
			output.Version = input.Version
		}
		_, err := s.Client.UpgradeStack(ctx, operations.UpgradeStackRequest{
			OrganizationID: input.OrganizationID,
			StackID:        input.StackID,
			Body:           body,
		})
		return output, err
	default:
		return StackActionOutput{}, fmt.Errorf("unsupported stack action %q", s.Action)
	}
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
