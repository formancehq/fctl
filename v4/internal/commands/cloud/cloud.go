package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"
)

type MembershipClient interface {
	ReadConnectedUser(context.Context, ...operations.Option) (*operations.ReadConnectedUserResponse, error)
	ListOrganizations(context.Context, operations.ListOrganizationsRequest, ...operations.Option) (*operations.ListOrganizationsResponse, error)
	ReadOrganization(context.Context, operations.ReadOrganizationRequest, ...operations.Option) (*operations.ReadOrganizationResponse, error)
}

type UserSummary struct {
	ID    string `json:"id" yaml:"id"`
	Email string `json:"email" yaml:"email"`
	Role  string `json:"role,omitempty" yaml:"role,omitempty"`
}

type OrganizationSummary struct {
	ID                 string       `json:"id" yaml:"id"`
	Name               string       `json:"name" yaml:"name"`
	OwnerID            string       `json:"ownerID,omitempty" yaml:"ownerID,omitempty"`
	Domain             string       `json:"domain,omitempty" yaml:"domain,omitempty"`
	DefaultPolicyID    *int64       `json:"defaultPolicyID,omitempty" yaml:"defaultPolicyID,omitempty"`
	AvailableStacks    *int64       `json:"availableStacks,omitempty" yaml:"availableStacks,omitempty"`
	AvailableSandboxes *int64       `json:"availableSandboxes,omitempty" yaml:"availableSandboxes,omitempty"`
	TotalStacks        *int64       `json:"totalStacks,omitempty" yaml:"totalStacks,omitempty"`
	TotalUsers         *int64       `json:"totalUsers,omitempty" yaml:"totalUsers,omitempty"`
	CreatedAt          *time.Time   `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt          *time.Time   `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	Owner              *UserSummary `json:"owner,omitempty" yaml:"owner,omitempty"`
}

type MeOutput struct {
	User UserSummary `json:"user" yaml:"user"`
}

type ListOrganizationsInput struct {
	Expand bool
}

type ListOrganizationsOutput struct {
	Organizations []OrganizationSummary `json:"organizations" yaml:"organizations"`
}

type OrganizationIDInput struct {
	OrganizationID string
	Expand         bool
}

type OrganizationOutput struct {
	Organization OrganizationSummary `json:"organization" yaml:"organization"`
}

type MeService struct {
	Client MembershipClient
}

func (s MeService) Run(ctx context.Context) (MeOutput, error) {
	if s.Client == nil {
		return MeOutput{}, fmt.Errorf("membership client is required")
	}
	response, err := s.Client.ReadConnectedUser(ctx)
	if err != nil {
		return MeOutput{}, err
	}
	if response.GetReadUserResponse().GetData() == nil {
		return MeOutput{}, fmt.Errorf("cloud me show returned no user")
	}
	return MeOutput{User: userSummary(response.GetReadUserResponse().GetData())}, nil
}

type ListOrganizationsService struct {
	Client MembershipClient
}

func (s ListOrganizationsService) Run(ctx context.Context, input ListOrganizationsInput) (ListOrganizationsOutput, error) {
	if s.Client == nil {
		return ListOrganizationsOutput{}, fmt.Errorf("membership client is required")
	}
	response, err := s.Client.ListOrganizations(ctx, operations.ListOrganizationsRequest{Expand: &input.Expand})
	if err != nil {
		return ListOrganizationsOutput{}, err
	}
	data := response.GetListOrganizationExpandedResponse().GetData()
	organizations := make([]OrganizationSummary, 0, len(data))
	for i := range data {
		organizations = append(organizations, organizationSummary(&data[i]))
	}
	return ListOrganizationsOutput{Organizations: organizations}, nil
}

type ReadOrganizationService struct {
	Client MembershipClient
}

func (s ReadOrganizationService) Run(ctx context.Context, input OrganizationIDInput) (OrganizationOutput, error) {
	if s.Client == nil {
		return OrganizationOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return OrganizationOutput{}, fmt.Errorf("organization id is required")
	}
	response, err := s.Client.ReadOrganization(ctx, operations.ReadOrganizationRequest{
		OrganizationID: input.OrganizationID,
		Expand:         &input.Expand,
	})
	if err != nil {
		return OrganizationOutput{}, err
	}
	if response.GetReadOrganizationResponse().GetData() == nil {
		return OrganizationOutput{}, fmt.Errorf("cloud organizations show returned no organization")
	}
	return OrganizationOutput{Organization: organizationSummary(response.GetReadOrganizationResponse().GetData())}, nil
}

func organizationSummary(organization *components.OrganizationExpanded) OrganizationSummary {
	if organization == nil {
		return OrganizationSummary{}
	}
	summary := OrganizationSummary{
		ID:                 organization.ID,
		Name:               organization.Name,
		OwnerID:            organization.OwnerID,
		DefaultPolicyID:    organization.DefaultPolicyID,
		AvailableStacks:    organization.AvailableStacks,
		AvailableSandboxes: organization.AvailableSandboxes,
		TotalStacks:        organization.TotalStacks,
		TotalUsers:         organization.TotalUsers,
		CreatedAt:          organization.CreatedAt,
		UpdatedAt:          organization.UpdatedAt,
	}
	if organization.Domain != nil {
		summary.Domain = *organization.Domain
	}
	if organization.Owner != nil {
		owner := userSummary(organization.Owner)
		summary.Owner = &owner
	}
	return summary
}

func userSummary(user *components.User) UserSummary {
	if user == nil {
		return UserSummary{}
	}
	summary := UserSummary{
		ID:    user.ID,
		Email: user.Email,
	}
	if user.Role != nil {
		summary.Role = string(*user.Role)
	}
	return summary
}
