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
	ListOrganizationInvitations(context.Context, operations.ListOrganizationInvitationsRequest, ...operations.Option) (*operations.ListOrganizationInvitationsResponse, error)
	CreateInvitation(context.Context, operations.CreateInvitationRequest, ...operations.Option) (*operations.CreateInvitationResponse, error)
	DeleteInvitation(context.Context, operations.DeleteInvitationRequest, ...operations.Option) (*operations.DeleteInvitationResponse, error)
	ListUsersOfOrganization(context.Context, operations.ListUsersOfOrganizationRequest, ...operations.Option) (*operations.ListUsersOfOrganizationResponse, error)
	ReadUserOfOrganization(context.Context, operations.ReadUserOfOrganizationRequest, ...operations.Option) (*operations.ReadUserOfOrganizationResponse, error)
	UpsertOrganizationUser(context.Context, operations.UpsertOrganizationUserRequest, ...operations.Option) (*operations.UpsertOrganizationUserResponse, error)
	DeleteUserFromOrganization(context.Context, operations.DeleteUserFromOrganizationRequest, ...operations.Option) (*operations.DeleteUserFromOrganizationResponse, error)
	ListPolicies(context.Context, operations.ListPoliciesRequest, ...operations.Option) (*operations.ListPoliciesResponse, error)
	CreatePolicy(context.Context, operations.CreatePolicyRequest, ...operations.Option) (*operations.CreatePolicyResponse, error)
	ReadPolicy(context.Context, operations.ReadPolicyRequest, ...operations.Option) (*operations.ReadPolicyResponse, error)
	UpdatePolicy(context.Context, operations.UpdatePolicyRequest, ...operations.Option) (*operations.UpdatePolicyResponse, error)
	DeletePolicy(context.Context, operations.DeletePolicyRequest, ...operations.Option) (*operations.DeletePolicyResponse, error)
	AddScopeToPolicy(context.Context, operations.AddScopeToPolicyRequest, ...operations.Option) (*operations.AddScopeToPolicyResponse, error)
	RemoveScopeFromPolicy(context.Context, operations.RemoveScopeFromPolicyRequest, ...operations.Option) (*operations.RemoveScopeFromPolicyResponse, error)
	ListRegions(context.Context, operations.ListRegionsRequest, ...operations.Option) (*operations.ListRegionsResponse, error)
	GetRegion(context.Context, operations.GetRegionRequest, ...operations.Option) (*operations.GetRegionResponse, error)
	CreatePrivateRegion(context.Context, operations.CreatePrivateRegionRequest, ...operations.Option) (*operations.CreatePrivateRegionResponse, error)
	DeleteRegion(context.Context, operations.DeleteRegionRequest, ...operations.Option) (*operations.DeleteRegionResponse, error)
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

type ListOrganizationInvitationsInput struct {
	OrganizationID string
	Status         string
}

type CreateInvitationInput struct {
	OrganizationID string
	Email          string
}

type OrganizationInvitationActionInput struct {
	OrganizationID string
	InvitationID   string
}

type OrganizationInvitationActionOutput struct {
	OrganizationID string `json:"organizationID" yaml:"organizationID"`
	InvitationID   string `json:"invitationID" yaml:"invitationID"`
	Action         string `json:"action" yaml:"action"`
}

type OrganizationUserSummary struct {
	ID       string `json:"id" yaml:"id"`
	Email    string `json:"email" yaml:"email"`
	PolicyID int64  `json:"policyID" yaml:"policyID"`
}

type ListOrganizationUsersOutput struct {
	OrganizationID string                    `json:"organizationID" yaml:"organizationID"`
	Users          []OrganizationUserSummary `json:"users" yaml:"users"`
}

type OrganizationUserOutput struct {
	OrganizationID string                  `json:"organizationID" yaml:"organizationID"`
	User           OrganizationUserSummary `json:"user" yaml:"user"`
}

type OrganizationUserActionInput struct {
	OrganizationID string
	UserID         string
	PolicyID       int64
}

type OrganizationUserActionOutput struct {
	OrganizationID string `json:"organizationID" yaml:"organizationID"`
	UserID         string `json:"userID" yaml:"userID"`
	Action         string `json:"action" yaml:"action"`
	PolicyID       int64  `json:"policyID,omitempty" yaml:"policyID,omitempty"`
}

type ScopeSummary struct {
	ID            int64  `json:"id" yaml:"id"`
	Label         string `json:"label" yaml:"label"`
	Description   string `json:"description,omitempty" yaml:"description,omitempty"`
	ApplicationID string `json:"applicationID,omitempty" yaml:"applicationID,omitempty"`
	Protected     *bool  `json:"protected,omitempty" yaml:"protected,omitempty"`
}

type PolicySummary struct {
	ID             int64          `json:"id" yaml:"id"`
	Name           string         `json:"name" yaml:"name"`
	Description    string         `json:"description,omitempty" yaml:"description,omitempty"`
	OrganizationID string         `json:"organizationID,omitempty" yaml:"organizationID,omitempty"`
	Protected      bool           `json:"protected" yaml:"protected"`
	Scopes         []ScopeSummary `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	CreatedAt      time.Time      `json:"createdAt" yaml:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt" yaml:"updatedAt"`
}

type ListPoliciesOutput struct {
	OrganizationID string          `json:"organizationID" yaml:"organizationID"`
	Policies       []PolicySummary `json:"policies" yaml:"policies"`
}

type PolicyOutput struct {
	OrganizationID string        `json:"organizationID" yaml:"organizationID"`
	Policy         PolicySummary `json:"policy" yaml:"policy"`
}

type PolicyInput struct {
	OrganizationID string
	PolicyID       int64
	Name           string
	Description    string
}

type PolicyActionInput struct {
	OrganizationID string
	PolicyID       int64
	ScopeID        int64
}

type PolicyActionOutput struct {
	OrganizationID string `json:"organizationID" yaml:"organizationID"`
	PolicyID       int64  `json:"policyID" yaml:"policyID"`
	ScopeID        int64  `json:"scopeID,omitempty" yaml:"scopeID,omitempty"`
	Action         string `json:"action" yaml:"action"`
}

type RegionSummary struct {
	ID             string `json:"id" yaml:"id"`
	Name           string `json:"name" yaml:"name"`
	BaseURL        string `json:"baseURL,omitempty" yaml:"baseURL,omitempty"`
	Active         bool   `json:"active" yaml:"active"`
	Public         bool   `json:"public" yaml:"public"`
	Version        string `json:"version,omitempty" yaml:"version,omitempty"`
	OrganizationID string `json:"organizationID,omitempty" yaml:"organizationID,omitempty"`
}

type ListRegionsOutput struct {
	OrganizationID string          `json:"organizationID" yaml:"organizationID"`
	Regions        []RegionSummary `json:"regions" yaml:"regions"`
}

type RegionInput struct {
	OrganizationID string
	RegionID       string
	Name           string
}

type RegionOutput struct {
	OrganizationID string        `json:"organizationID" yaml:"organizationID"`
	Region         RegionSummary `json:"region" yaml:"region"`
}

type RegionActionOutput struct {
	OrganizationID string `json:"organizationID" yaml:"organizationID"`
	RegionID       string `json:"regionID" yaml:"regionID"`
	Action         string `json:"action" yaml:"action"`
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

type ListOrganizationInvitationsService struct {
	Client MembershipClient
}

func (s ListOrganizationInvitationsService) Run(ctx context.Context, input ListOrganizationInvitationsInput) (ListInvitationsOutput, error) {
	if s.Client == nil {
		return ListInvitationsOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return ListInvitationsOutput{}, fmt.Errorf("organization id is required")
	}
	request := operations.ListOrganizationInvitationsRequest{OrganizationID: input.OrganizationID}
	if input.Status != "" {
		request.Status = &input.Status
	}
	response, err := s.Client.ListOrganizationInvitations(ctx, request)
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

type CreateInvitationService struct {
	Client MembershipClient
}

func (s CreateInvitationService) Run(ctx context.Context, input CreateInvitationInput) (InvitationSummary, error) {
	if s.Client == nil {
		return InvitationSummary{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return InvitationSummary{}, fmt.Errorf("organization id is required")
	}
	if input.Email == "" {
		return InvitationSummary{}, fmt.Errorf("invitation email is required")
	}
	response, err := s.Client.CreateInvitation(ctx, operations.CreateInvitationRequest{
		OrganizationID: input.OrganizationID,
		Email:          input.Email,
	})
	if err != nil {
		return InvitationSummary{}, err
	}
	if response.GetCreateInvitationResponse().GetData() == nil {
		return InvitationSummary{}, fmt.Errorf("cloud organizations invitations send returned no invitation")
	}
	return invitationSummary(response.GetCreateInvitationResponse().GetData()), nil
}

type DeleteInvitationService struct {
	Client MembershipClient
}

func (s DeleteInvitationService) Run(ctx context.Context, input OrganizationInvitationActionInput) (OrganizationInvitationActionOutput, error) {
	if s.Client == nil {
		return OrganizationInvitationActionOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return OrganizationInvitationActionOutput{}, fmt.Errorf("organization id is required")
	}
	if input.InvitationID == "" {
		return OrganizationInvitationActionOutput{}, fmt.Errorf("invitation id is required")
	}
	_, err := s.Client.DeleteInvitation(ctx, operations.DeleteInvitationRequest{
		OrganizationID: input.OrganizationID,
		InvitationID:   input.InvitationID,
	})
	if err != nil {
		return OrganizationInvitationActionOutput{}, err
	}
	return OrganizationInvitationActionOutput{OrganizationID: input.OrganizationID, InvitationID: input.InvitationID, Action: "delete"}, nil
}

type ListOrganizationUsersService struct {
	Client MembershipClient
}

func (s ListOrganizationUsersService) Run(ctx context.Context, organizationID string) (ListOrganizationUsersOutput, error) {
	if s.Client == nil {
		return ListOrganizationUsersOutput{}, fmt.Errorf("membership client is required")
	}
	if organizationID == "" {
		return ListOrganizationUsersOutput{}, fmt.Errorf("organization id is required")
	}
	response, err := s.Client.ListUsersOfOrganization(ctx, operations.ListUsersOfOrganizationRequest{OrganizationID: organizationID})
	if err != nil {
		return ListOrganizationUsersOutput{}, err
	}
	data := response.GetListUsersResponse().GetData()
	users := make([]OrganizationUserSummary, 0, len(data))
	for _, user := range data {
		users = append(users, OrganizationUserSummary{
			ID:       user.ID,
			Email:    user.Email,
			PolicyID: user.PolicyID,
		})
	}
	return ListOrganizationUsersOutput{OrganizationID: organizationID, Users: users}, nil
}

type ReadOrganizationUserService struct {
	Client MembershipClient
}

func (s ReadOrganizationUserService) Run(ctx context.Context, input OrganizationUserActionInput) (OrganizationUserOutput, error) {
	if s.Client == nil {
		return OrganizationUserOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return OrganizationUserOutput{}, fmt.Errorf("organization id is required")
	}
	if input.UserID == "" {
		return OrganizationUserOutput{}, fmt.Errorf("user id is required")
	}
	response, err := s.Client.ReadUserOfOrganization(ctx, operations.ReadUserOfOrganizationRequest{OrganizationID: input.OrganizationID, UserID: input.UserID})
	if err != nil {
		return OrganizationUserOutput{}, err
	}
	data := response.GetReadOrganizationUserResponse().GetData()
	if data == nil {
		return OrganizationUserOutput{}, fmt.Errorf("cloud organizations users show returned no user")
	}
	return OrganizationUserOutput{OrganizationID: input.OrganizationID, User: OrganizationUserSummary{ID: data.ID, Email: data.Email, PolicyID: data.PolicyID}}, nil
}

type OrganizationUserActionService struct {
	Client MembershipClient
	Action string
}

func (s OrganizationUserActionService) Run(ctx context.Context, input OrganizationUserActionInput) (OrganizationUserActionOutput, error) {
	if s.Client == nil {
		return OrganizationUserActionOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return OrganizationUserActionOutput{}, fmt.Errorf("organization id is required")
	}
	if input.UserID == "" {
		return OrganizationUserActionOutput{}, fmt.Errorf("user id is required")
	}
	output := OrganizationUserActionOutput{OrganizationID: input.OrganizationID, UserID: input.UserID, Action: s.Action, PolicyID: input.PolicyID}
	switch s.Action {
	case "link":
		body := &components.UpdateOrganizationUserRequest{}
		if input.PolicyID != 0 {
			body.PolicyID = &input.PolicyID
		}
		_, err := s.Client.UpsertOrganizationUser(ctx, operations.UpsertOrganizationUserRequest{OrganizationID: input.OrganizationID, UserID: input.UserID, Body: body})
		return output, err
	case "unlink":
		_, err := s.Client.DeleteUserFromOrganization(ctx, operations.DeleteUserFromOrganizationRequest{OrganizationID: input.OrganizationID, UserID: input.UserID})
		return output, err
	default:
		return OrganizationUserActionOutput{}, fmt.Errorf("unsupported organization user action %q", s.Action)
	}
}

type ListPoliciesService struct {
	Client MembershipClient
}

func (s ListPoliciesService) Run(ctx context.Context, organizationID string) (ListPoliciesOutput, error) {
	if s.Client == nil {
		return ListPoliciesOutput{}, fmt.Errorf("membership client is required")
	}
	if organizationID == "" {
		return ListPoliciesOutput{}, fmt.Errorf("organization id is required")
	}
	response, err := s.Client.ListPolicies(ctx, operations.ListPoliciesRequest{OrganizationID: organizationID})
	if err != nil {
		return ListPoliciesOutput{}, err
	}
	data := response.GetListPoliciesResponse().GetData()
	policies := make([]PolicySummary, 0, len(data))
	for i := range data {
		policies = append(policies, policySummary(&data[i]))
	}
	return ListPoliciesOutput{OrganizationID: organizationID, Policies: policies}, nil
}

type CreatePolicyService struct {
	Client MembershipClient
}

func (s CreatePolicyService) Run(ctx context.Context, input PolicyInput) (PolicyOutput, error) {
	if s.Client == nil {
		return PolicyOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return PolicyOutput{}, fmt.Errorf("organization id is required")
	}
	if input.Name == "" {
		return PolicyOutput{}, fmt.Errorf("policy name is required")
	}
	body := &components.CreatePolicyRequest{Name: input.Name}
	if input.Description != "" {
		body.Description = &input.Description
	}
	response, err := s.Client.CreatePolicy(ctx, operations.CreatePolicyRequest{OrganizationID: input.OrganizationID, Body: body})
	if err != nil {
		return PolicyOutput{}, err
	}
	if response.GetCreatePolicyResponse().GetData() == nil {
		return PolicyOutput{}, fmt.Errorf("cloud organizations policies create returned no policy")
	}
	return PolicyOutput{OrganizationID: input.OrganizationID, Policy: policySummary(response.GetCreatePolicyResponse().GetData())}, nil
}

type ReadPolicyService struct {
	Client MembershipClient
}

func (s ReadPolicyService) Run(ctx context.Context, input PolicyInput) (PolicyOutput, error) {
	if err := validatePolicyTarget(input.OrganizationID, input.PolicyID); err != nil {
		return PolicyOutput{}, err
	}
	response, err := s.Client.ReadPolicy(ctx, operations.ReadPolicyRequest{OrganizationID: input.OrganizationID, PolicyID: input.PolicyID})
	if err != nil {
		return PolicyOutput{}, err
	}
	if response.GetReadPolicyResponse().GetData() == nil {
		return PolicyOutput{}, fmt.Errorf("cloud organizations policies show returned no policy")
	}
	return PolicyOutput{OrganizationID: input.OrganizationID, Policy: policySummary(response.GetReadPolicyResponse().GetData())}, nil
}

type UpdatePolicyService struct {
	Client MembershipClient
}

func (s UpdatePolicyService) Run(ctx context.Context, input PolicyInput) (PolicyOutput, error) {
	if err := validatePolicyTarget(input.OrganizationID, input.PolicyID); err != nil {
		return PolicyOutput{}, err
	}
	if input.Name == "" {
		return PolicyOutput{}, fmt.Errorf("policy name is required")
	}
	body := &components.CreatePolicyRequest{Name: input.Name}
	if input.Description != "" {
		body.Description = &input.Description
	}
	response, err := s.Client.UpdatePolicy(ctx, operations.UpdatePolicyRequest{OrganizationID: input.OrganizationID, PolicyID: input.PolicyID, Body: body})
	if err != nil {
		return PolicyOutput{}, err
	}
	if response.GetUpdatePolicyResponse().GetData() == nil {
		return PolicyOutput{}, fmt.Errorf("cloud organizations policies update returned no policy")
	}
	return PolicyOutput{OrganizationID: input.OrganizationID, Policy: policySummary(response.GetUpdatePolicyResponse().GetData())}, nil
}

type PolicyActionService struct {
	Client MembershipClient
	Action string
}

func (s PolicyActionService) Run(ctx context.Context, input PolicyActionInput) (PolicyActionOutput, error) {
	if err := validatePolicyTarget(input.OrganizationID, input.PolicyID); err != nil {
		return PolicyActionOutput{}, err
	}
	output := PolicyActionOutput{OrganizationID: input.OrganizationID, PolicyID: input.PolicyID, ScopeID: input.ScopeID, Action: s.Action}
	switch s.Action {
	case "delete":
		_, err := s.Client.DeletePolicy(ctx, operations.DeletePolicyRequest{OrganizationID: input.OrganizationID, PolicyID: input.PolicyID})
		return output, err
	case "add-scope":
		if input.ScopeID == 0 {
			return PolicyActionOutput{}, fmt.Errorf("scope id is required")
		}
		_, err := s.Client.AddScopeToPolicy(ctx, operations.AddScopeToPolicyRequest{OrganizationID: input.OrganizationID, PolicyID: input.PolicyID, ScopeID: input.ScopeID})
		return output, err
	case "remove-scope":
		if input.ScopeID == 0 {
			return PolicyActionOutput{}, fmt.Errorf("scope id is required")
		}
		_, err := s.Client.RemoveScopeFromPolicy(ctx, operations.RemoveScopeFromPolicyRequest{OrganizationID: input.OrganizationID, PolicyID: input.PolicyID, ScopeID: input.ScopeID})
		return output, err
	default:
		return PolicyActionOutput{}, fmt.Errorf("unsupported policy action %q", s.Action)
	}
}

type ListRegionsService struct {
	Client MembershipClient
}

func (s ListRegionsService) Run(ctx context.Context, organizationID string) (ListRegionsOutput, error) {
	if s.Client == nil {
		return ListRegionsOutput{}, fmt.Errorf("membership client is required")
	}
	if organizationID == "" {
		return ListRegionsOutput{}, fmt.Errorf("organization id is required")
	}
	response, err := s.Client.ListRegions(ctx, operations.ListRegionsRequest{OrganizationID: organizationID})
	if err != nil {
		return ListRegionsOutput{}, err
	}
	if response.GetListRegionsResponse() == nil {
		return ListRegionsOutput{}, fmt.Errorf("cloud regions list returned no regions")
	}
	data := response.GetListRegionsResponse().GetData()
	regions := make([]RegionSummary, 0, len(data))
	for i := range data {
		regions = append(regions, regionSummaryFromAny(&data[i]))
	}
	return ListRegionsOutput{OrganizationID: organizationID, Regions: regions}, nil
}

type ReadRegionService struct {
	Client MembershipClient
}

func (s ReadRegionService) Run(ctx context.Context, input RegionInput) (RegionOutput, error) {
	if s.Client == nil {
		return RegionOutput{}, fmt.Errorf("membership client is required")
	}
	if err := validateRegionTarget(input.OrganizationID, input.RegionID); err != nil {
		return RegionOutput{}, err
	}
	response, err := s.Client.GetRegion(ctx, operations.GetRegionRequest{
		OrganizationID: input.OrganizationID,
		RegionID:       input.RegionID,
	})
	if err != nil {
		return RegionOutput{}, err
	}
	if response.GetGetRegionResponse() == nil {
		return RegionOutput{}, fmt.Errorf("cloud regions show returned no region")
	}
	region := response.GetGetRegionResponse().GetData()
	return RegionOutput{OrganizationID: input.OrganizationID, Region: regionSummaryFromAny(&region)}, nil
}

type CreateRegionService struct {
	Client MembershipClient
}

func (s CreateRegionService) Run(ctx context.Context, input RegionInput) (RegionOutput, error) {
	if s.Client == nil {
		return RegionOutput{}, fmt.Errorf("membership client is required")
	}
	if input.OrganizationID == "" {
		return RegionOutput{}, fmt.Errorf("organization id is required")
	}
	if input.Name == "" {
		return RegionOutput{}, fmt.Errorf("region name is required")
	}
	response, err := s.Client.CreatePrivateRegion(ctx, operations.CreatePrivateRegionRequest{
		OrganizationID: input.OrganizationID,
		Body:           &components.CreatePrivateRegionRequest{Name: input.Name},
	})
	if err != nil {
		return RegionOutput{}, err
	}
	if response.GetCreatedPrivateRegionResponse() == nil {
		return RegionOutput{}, fmt.Errorf("cloud regions create returned no region")
	}
	region := response.GetCreatedPrivateRegionResponse().GetData()
	return RegionOutput{OrganizationID: input.OrganizationID, Region: regionSummaryFromPrivate(&region)}, nil
}

type DeleteRegionService struct {
	Client MembershipClient
}

func (s DeleteRegionService) Run(ctx context.Context, input RegionInput) (RegionActionOutput, error) {
	if s.Client == nil {
		return RegionActionOutput{}, fmt.Errorf("membership client is required")
	}
	if err := validateRegionTarget(input.OrganizationID, input.RegionID); err != nil {
		return RegionActionOutput{}, err
	}
	_, err := s.Client.DeleteRegion(ctx, operations.DeleteRegionRequest{
		OrganizationID: input.OrganizationID,
		RegionID:       input.RegionID,
	})
	if err != nil {
		return RegionActionOutput{}, err
	}
	return RegionActionOutput{OrganizationID: input.OrganizationID, RegionID: input.RegionID, Action: "delete"}, nil
}

func validateRegionTarget(organizationID string, regionID string) error {
	if organizationID == "" {
		return fmt.Errorf("organization id is required")
	}
	if regionID == "" {
		return fmt.Errorf("region id is required")
	}
	return nil
}

func validatePolicyTarget(organizationID string, policyID int64) error {
	if organizationID == "" {
		return fmt.Errorf("organization id is required")
	}
	if policyID == 0 {
		return fmt.Errorf("policy id is required")
	}
	return nil
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

func policySummary(policy *components.Policy) PolicySummary {
	if policy == nil {
		return PolicySummary{}
	}
	summary := PolicySummary{
		ID:        policy.ID,
		Name:      policy.Name,
		Protected: policy.Protected,
		CreatedAt: policy.CreatedAt,
		UpdatedAt: policy.UpdatedAt,
	}
	if policy.Description != nil {
		summary.Description = *policy.Description
	}
	if policy.OrganizationID != nil {
		summary.OrganizationID = *policy.OrganizationID
	}
	if len(policy.Scopes) > 0 {
		summary.Scopes = make([]ScopeSummary, 0, len(policy.Scopes))
		for i := range policy.Scopes {
			summary.Scopes = append(summary.Scopes, scopeSummary(&policy.Scopes[i]))
		}
	}
	return summary
}

func scopeSummary(scope *components.Scope) ScopeSummary {
	if scope == nil {
		return ScopeSummary{}
	}
	summary := ScopeSummary{ID: scope.ID, Label: scope.Label, Protected: scope.Protected}
	if scope.Description != nil {
		summary.Description = *scope.Description
	}
	if scope.ApplicationID != nil {
		summary.ApplicationID = *scope.ApplicationID
	}
	return summary
}

func regionSummaryFromAny(region *components.AnyRegion) RegionSummary {
	if region == nil {
		return RegionSummary{}
	}
	summary := RegionSummary{
		ID:      region.ID,
		Name:    region.Name,
		BaseURL: region.BaseURL,
		Active:  region.Active,
		Public:  region.Public,
	}
	if region.Version != nil {
		summary.Version = *region.Version
	}
	if region.OrganizationID != nil {
		summary.OrganizationID = *region.OrganizationID
	}
	return summary
}

func regionSummaryFromPrivate(region *components.PrivateRegion) RegionSummary {
	if region == nil {
		return RegionSummary{}
	}
	summary := RegionSummary{
		ID:             region.ID,
		Name:           region.Name,
		BaseURL:        region.BaseURL,
		Active:         region.Active,
		OrganizationID: region.OrganizationID,
	}
	if region.Version != nil {
		summary.Version = *region.Version
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
