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
	CreateOrganization(context.Context, *components.CreateOrganizationRequest, ...operations.Option) (*operations.CreateOrganizationResponse, error)
	UpdateOrganization(context.Context, operations.UpdateOrganizationRequest, ...operations.Option) (*operations.UpdateOrganizationResponse, error)
	DeleteOrganization(context.Context, operations.DeleteOrganizationRequest, ...operations.Option) (*operations.DeleteOrganizationResponse, error)
	ListInvitations(context.Context, operations.ListInvitationsRequest, ...operations.Option) (*operations.ListInvitationsResponse, error)
	AcceptInvitation(context.Context, operations.AcceptInvitationRequest, ...operations.Option) (*operations.AcceptInvitationResponse, error)
	DeclineInvitation(context.Context, operations.DeclineInvitationRequest, ...operations.Option) (*operations.DeclineInvitationResponse, error)
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

type CreateOrganizationInput struct {
	Name            string
	Domain          string
	DefaultPolicyID *int64
	OwnerID         string
}

type UpdateOrganizationInput struct {
	OrganizationID  string
	Name            string
	Domain          string
	DefaultPolicyID *int64
}

type DeleteOrganizationOutput struct {
	OrganizationID string `json:"organizationID" yaml:"organizationID"`
}

type InvitationSummary struct {
	ID             string     `json:"id" yaml:"id"`
	OrganizationID string     `json:"organizationID" yaml:"organizationID"`
	UserEmail      string     `json:"userEmail" yaml:"userEmail"`
	Status         string     `json:"status" yaml:"status"`
	CreationDate   time.Time  `json:"creationDate" yaml:"creationDate"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty" yaml:"expiresAt,omitempty"`
	UserID         string     `json:"userID,omitempty" yaml:"userID,omitempty"`
	CreatorID      string     `json:"creatorID,omitempty" yaml:"creatorID,omitempty"`
}

type ListInvitationsInput struct {
	Status       string
	Organization string
}

type ListInvitationsOutput struct {
	Invitations []InvitationSummary `json:"invitations" yaml:"invitations"`
}

type InvitationActionOutput struct {
	InvitationID string `json:"invitationID" yaml:"invitationID"`
	Action       string `json:"action" yaml:"action"`
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

type CreateOrganizationService struct {
	Client MembershipClient
}

func (s CreateOrganizationService) Run(ctx context.Context, input CreateOrganizationInput) (OrganizationOutput, error) {
	if s.Client == nil {
		return OrganizationOutput{}, fmt.Errorf("membership client is required")
	}
	if input.Name == "" {
		return OrganizationOutput{}, fmt.Errorf("organization name is required")
	}
	body := &components.CreateOrganizationRequest{
		Name:            input.Name,
		DefaultPolicyID: input.DefaultPolicyID,
	}
	if input.Domain != "" {
		body.Domain = &input.Domain
	}
	if input.OwnerID != "" {
		body.OwnerID = &input.OwnerID
	}
	response, err := s.Client.CreateOrganization(ctx, body)
	if err != nil {
		return OrganizationOutput{}, err
	}
	if response.GetCreateOrganizationResponse().GetData() == nil {
		return OrganizationOutput{}, fmt.Errorf("cloud organizations create returned no organization")
	}
	return OrganizationOutput{Organization: organizationSummary(response.GetCreateOrganizationResponse().GetData())}, nil
}

type UpdateOrganizationService struct {
	Client MembershipClient
}

func (s UpdateOrganizationService) Run(ctx context.Context, input UpdateOrganizationInput) (OrganizationOutput, error) {
	if s.Client == nil {
		return OrganizationOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return OrganizationOutput{}, fmt.Errorf("organization id is required")
	}
	if input.Name == "" {
		return OrganizationOutput{}, fmt.Errorf("organization name is required")
	}
	body := &components.OrganizationData{
		Name:            input.Name,
		DefaultPolicyID: input.DefaultPolicyID,
	}
	if input.Domain != "" {
		body.Domain = &input.Domain
	}
	response, err := s.Client.UpdateOrganization(ctx, operations.UpdateOrganizationRequest{
		OrganizationID: input.OrganizationID,
		Body:           body,
	})
	if err != nil {
		return OrganizationOutput{}, err
	}
	if response.GetReadOrganizationResponse().GetData() == nil {
		return OrganizationOutput{}, fmt.Errorf("cloud organizations update returned no organization")
	}
	return OrganizationOutput{Organization: organizationSummary(response.GetReadOrganizationResponse().GetData())}, nil
}

type DeleteOrganizationService struct {
	Client MembershipClient
}

func (s DeleteOrganizationService) Run(ctx context.Context, organizationID string) (DeleteOrganizationOutput, error) {
	if s.Client == nil {
		return DeleteOrganizationOutput{}, fmt.Errorf("membership client is required")
	}
	if organizationID == "" {
		return DeleteOrganizationOutput{}, fmt.Errorf("organization id is required")
	}
	_, err := s.Client.DeleteOrganization(ctx, operations.DeleteOrganizationRequest{OrganizationID: organizationID})
	if err != nil {
		return DeleteOrganizationOutput{}, err
	}
	return DeleteOrganizationOutput{OrganizationID: organizationID}, nil
}

type ListInvitationsService struct {
	Client MembershipClient
}

func (s ListInvitationsService) Run(ctx context.Context, input ListInvitationsInput) (ListInvitationsOutput, error) {
	if s.Client == nil {
		return ListInvitationsOutput{}, fmt.Errorf("membership client is required")
	}
	request := operations.ListInvitationsRequest{}
	if input.Status != "" {
		request.Status = &input.Status
	}
	if input.Organization != "" {
		request.Organization = &input.Organization
	}
	response, err := s.Client.ListInvitations(ctx, request)
	if err != nil {
		return ListInvitationsOutput{}, err
	}
	data := response.GetListInvitationsResponse().GetData()
	invitations := make([]InvitationSummary, 0, len(data))
	for i := range data {
		invitations = append(invitations, invitationSummary(&data[i]))
	}
	return ListInvitationsOutput{Invitations: invitations}, nil
}

type InvitationActionService struct {
	Client MembershipClient
	Action string
}

func (s InvitationActionService) Run(ctx context.Context, invitationID string) (InvitationActionOutput, error) {
	if s.Client == nil {
		return InvitationActionOutput{}, fmt.Errorf("membership client is required")
	}
	if invitationID == "" {
		return InvitationActionOutput{}, fmt.Errorf("invitation id is required")
	}
	switch s.Action {
	case "accept":
		_, err := s.Client.AcceptInvitation(ctx, operations.AcceptInvitationRequest{InvitationID: invitationID})
		return InvitationActionOutput{InvitationID: invitationID, Action: s.Action}, err
	case "decline":
		_, err := s.Client.DeclineInvitation(ctx, operations.DeclineInvitationRequest{InvitationID: invitationID})
		return InvitationActionOutput{InvitationID: invitationID, Action: s.Action}, err
	default:
		return InvitationActionOutput{}, fmt.Errorf("unsupported invitation action %q", s.Action)
	}
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

func invitationSummary(invitation *components.Invitation) InvitationSummary {
	if invitation == nil {
		return InvitationSummary{}
	}
	summary := InvitationSummary{
		ID:             invitation.ID,
		OrganizationID: invitation.OrganizationID,
		UserEmail:      invitation.UserEmail,
		Status:         string(invitation.Status),
		CreationDate:   invitation.CreationDate,
		ExpiresAt:      invitation.ExpiresAt,
	}
	if invitation.UserID != nil {
		summary.UserID = *invitation.UserID
	}
	if invitation.CreatorID != nil {
		summary.CreatorID = *invitation.CreatorID
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
