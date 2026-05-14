package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"
)

type StackClient interface {
	GetServerInfo(context.Context, ...operations.Option) (*operations.GetServerInfoResponse, error)
	ListStacks(context.Context, operations.ListStacksRequest, ...operations.Option) (*operations.ListStacksResponse, error)
	GetStack(context.Context, operations.GetStackRequest, ...operations.Option) (*operations.GetStackResponse, error)
	CreateStack(context.Context, operations.CreateStackRequest, ...operations.Option) (*operations.CreateStackResponse, error)
	UpdateStack(context.Context, operations.UpdateStackRequest, ...operations.Option) (*operations.UpdateStackResponse, error)
	DeleteStack(context.Context, operations.DeleteStackRequest, ...operations.Option) (*operations.DeleteStackResponse, error)
	EnableStack(context.Context, operations.EnableStackRequest, ...operations.Option) (*operations.EnableStackResponse, error)
	DisableStack(context.Context, operations.DisableStackRequest, ...operations.Option) (*operations.DisableStackResponse, error)
	RestoreStack(context.Context, operations.RestoreStackRequest, ...operations.Option) (*operations.RestoreStackResponse, error)
	UpgradeStack(context.Context, operations.UpgradeStackRequest, ...operations.Option) (*operations.UpgradeStackResponse, error)
	ListModules(context.Context, operations.ListModulesRequest, ...operations.Option) (*operations.ListModulesResponse, error)
	EnableModule(context.Context, operations.EnableModuleRequest, ...operations.Option) (*operations.EnableModuleResponse, error)
	DisableModule(context.Context, operations.DisableModuleRequest, ...operations.Option) (*operations.DisableModuleResponse, error)
	ListStackUsersAccesses(context.Context, operations.ListStackUsersAccessesRequest, ...operations.Option) (*operations.ListStackUsersAccessesResponse, error)
	UpsertStackUserAccess(context.Context, operations.UpsertStackUserAccessRequest, ...operations.Option) (*operations.UpsertStackUserAccessResponse, error)
	DeleteStackUserAccess(context.Context, operations.DeleteStackUserAccessRequest, ...operations.Option) (*operations.DeleteStackUserAccessResponse, error)
}

type StackSummary struct {
	ID              string            `json:"id" yaml:"id"`
	Name            string            `json:"name" yaml:"name"`
	OrganizationID  string            `json:"organizationID" yaml:"organizationID"`
	Dashboard       string            `json:"dashboard,omitempty" yaml:"dashboard,omitempty"`
	URI             string            `json:"uri,omitempty" yaml:"uri,omitempty"`
	RegionID        string            `json:"regionID,omitempty" yaml:"regionID,omitempty"`
	Version         string            `json:"version,omitempty" yaml:"version,omitempty"`
	Status          string            `json:"status" yaml:"status"`
	State           string            `json:"state" yaml:"state"`
	ExpectedStatus  string            `json:"expectedStatus" yaml:"expectedStatus"`
	Reachable       bool              `json:"reachable" yaml:"reachable"`
	StargateEnabled bool              `json:"stargateEnabled" yaml:"stargateEnabled"`
	AuditEnabled    *bool             `json:"auditEnabled,omitempty" yaml:"auditEnabled,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt       *time.Time        `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt       *time.Time        `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	DeletedAt       *time.Time        `json:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DisabledAt      *time.Time        `json:"disabledAt,omitempty" yaml:"disabledAt,omitempty"`
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
	Wait           bool
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

type WaitStackReadyInput struct {
	OrganizationID string
	StackID        string
	StackURL       string
	InitialStack   StackSummary
	PollInterval   time.Duration
	Timeout        time.Duration
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

type ModuleSummary struct {
	Name             string    `json:"name" yaml:"name"`
	State            string    `json:"state" yaml:"state"`
	Status           string    `json:"status" yaml:"status"`
	LastStatusUpdate time.Time `json:"lastStatusUpdate" yaml:"lastStatusUpdate"`
	LastStateUpdate  time.Time `json:"lastStateUpdate" yaml:"lastStateUpdate"`
}

type ListModulesOutput struct {
	OrganizationID string          `json:"organizationID" yaml:"organizationID"`
	StackID        string          `json:"stackID" yaml:"stackID"`
	Modules        []ModuleSummary `json:"modules" yaml:"modules"`
}

type ModuleActionInput struct {
	OrganizationID string
	StackID        string
	Name           string
}

type ModuleActionOutput struct {
	OrganizationID string `json:"organizationID" yaml:"organizationID"`
	StackID        string `json:"stackID" yaml:"stackID"`
	Name           string `json:"name" yaml:"name"`
	Action         string `json:"action" yaml:"action"`
}

type StackUserAccessSummary struct {
	StackID  string `json:"stackID" yaml:"stackID"`
	UserID   string `json:"userID" yaml:"userID"`
	Email    string `json:"email" yaml:"email"`
	PolicyID int64  `json:"policyID" yaml:"policyID"`
}

type ListStackUsersOutput struct {
	OrganizationID string                   `json:"organizationID" yaml:"organizationID"`
	StackID        string                   `json:"stackID" yaml:"stackID"`
	Users          []StackUserAccessSummary `json:"users" yaml:"users"`
}

type StackUserAccessInput struct {
	OrganizationID string
	StackID        string
	UserID         string
	PolicyID       int64
}

type StackUserAccessOutput struct {
	OrganizationID string `json:"organizationID" yaml:"organizationID"`
	StackID        string `json:"stackID" yaml:"stackID"`
	UserID         string `json:"userID" yaml:"userID"`
	Action         string `json:"action" yaml:"action"`
	PolicyID       int64  `json:"policyID,omitempty" yaml:"policyID,omitempty"`
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
	dashboard, err := stackDashboardURL(ctx, s.Client)
	if err != nil {
		return ListStacksOutput{}, err
	}
	stacks := make([]StackSummary, 0, len(data))
	for i := range data {
		stacks = append(stacks, stackSummary(&data[i], dashboard))
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
		return StackOutput{}, fmt.Errorf("cloud stacks show returned no stack")
	}
	return StackOutput{
		OrganizationID: input.OrganizationID,
		Stack:          stackSummary(response.GetReadStackResponse().GetData(), defaultConsoleURL),
	}, nil
}

type CreateStackService struct {
	Client     StackClient
	HTTPClient *http.Client
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
		return StackOutput{}, fmt.Errorf("cloud stacks create returned no stack")
	}
	output := StackOutput{OrganizationID: input.OrganizationID, Stack: stackSummary(response.GetReadStackResponse().GetData(), defaultConsoleURL)}
	if input.Wait && output.Stack.Status != string(components.StackStatusReady) {
		return WaitStackReadyService{Client: s.Client, HTTPClient: s.HTTPClient}.Run(ctx, WaitStackReadyInput{
			OrganizationID: input.OrganizationID,
			StackID:        output.Stack.ID,
			StackURL:       output.Stack.URI,
			InitialStack:   output.Stack,
		})
	}
	return output, nil
}

type WaitStackReadyService struct {
	Client     StackClient
	HTTPClient *http.Client
}

func (s WaitStackReadyService) Run(ctx context.Context, input WaitStackReadyInput) (StackOutput, error) {
	if s.Client == nil {
		return StackOutput{}, fmt.Errorf("membership client is required")
	}
	if err := validateStackTarget(input.OrganizationID, input.StackID); err != nil {
		return StackOutput{}, err
	}
	pollInterval := input.PollInterval
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	last := StackOutput{OrganizationID: input.OrganizationID, Stack: input.InitialStack}
	stackURL := input.StackURL
	if stackURL == "" {
		stackURL = input.InitialStack.URI
	}
	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			waitTimer := time.NewTimer(pollInterval)
			select {
			case <-ctx.Done():
				waitTimer.Stop()
				return StackOutput{}, ctx.Err()
			case <-timeoutTimer.C:
				waitTimer.Stop()
				return last, stackReadyTimeoutError(input.OrganizationID, input.StackID, timeout, last.Stack.Status)
			case <-waitTimer.C:
			}
		}

		if stackVersionsEndpointReady(ctx, s.HTTPClient, stackURL) {
			if last.Stack.ID == "" {
				last.Stack.ID = input.StackID
			}
			if last.Stack.OrganizationID == "" {
				last.Stack.OrganizationID = input.OrganizationID
			}
			if last.Stack.URI == "" {
				last.Stack.URI = stackURL
			}
			last.Stack.Status = string(components.StackStatusReady)
			last.Stack.Reachable = true
			return last, nil
		}

		output, err := ReadStackService{Client: s.Client}.Run(ctx, StackIDInput{
			OrganizationID: input.OrganizationID,
			StackID:        input.StackID,
		})
		if err != nil {
			return StackOutput{}, err
		}
		last = output
		if stackURL == "" {
			stackURL = output.Stack.URI
		}
		if output.Stack.Status == string(components.StackStatusReady) {
			return output, nil
		}
	}
}

func stackVersionsEndpointReady(ctx context.Context, client *http.Client, stackURL string) bool {
	if stackURL == "" {
		return false
	}
	if client == nil {
		client = http.DefaultClient
	}
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, stackVersionsURL(stackURL), nil)
	if err != nil {
		return false
	}
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false
	}
	var versions stackVersionsResponse
	if err := json.NewDecoder(response.Body).Decode(&versions); err != nil {
		return false
	}
	if len(versions.Versions) == 0 {
		return false
	}
	for _, version := range versions.Versions {
		if !version.Health {
			return false
		}
	}
	return true
}

type stackVersionsResponse struct {
	Versions []stackVersionHealth `json:"versions"`
}

type stackVersionHealth struct {
	Health bool `json:"health"`
}

func stackVersionsURL(stackURL string) string {
	for len(stackURL) > 0 && stackURL[len(stackURL)-1] == '/' {
		stackURL = stackURL[:len(stackURL)-1]
	}
	return stackURL + "/versions"
}

func stackReadyTimeoutError(organizationID string, stackID string, timeout time.Duration, lastStatus string) error {
	if lastStatus == "" {
		lastStatus = "unknown"
	}
	return fmt.Errorf("stack %s did not become READY after %s (last status: %s); check stack status with: fctl cloud stacks show %s --organization %s", stackID, timeout, lastStatus, stackID, organizationID)
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
		return StackOutput{}, fmt.Errorf("cloud stacks update returned no stack")
	}
	return StackOutput{OrganizationID: input.OrganizationID, Stack: stackSummary(response.GetReadStackResponse().GetData(), defaultConsoleURL)}, nil
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
			stack := stackSummary(response.GetReadStackResponse().GetData(), defaultConsoleURL)
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

type ListModulesService struct {
	Client StackClient
}

func (s ListModulesService) Run(ctx context.Context, input StackIDInput) (ListModulesOutput, error) {
	if err := validateStackTarget(input.OrganizationID, input.StackID); err != nil {
		return ListModulesOutput{}, err
	}
	response, err := s.Client.ListModules(ctx, operations.ListModulesRequest{OrganizationID: input.OrganizationID, StackID: input.StackID})
	if err != nil {
		return ListModulesOutput{}, err
	}
	data := response.GetListModulesResponse().GetData()
	modules := make([]ModuleSummary, 0, len(data))
	for _, module := range data {
		modules = append(modules, ModuleSummary{
			Name:             module.Name,
			State:            string(module.State),
			Status:           string(module.Status),
			LastStatusUpdate: module.LastStatusUpdate,
			LastStateUpdate:  module.LastStateUpdate,
		})
	}
	return ListModulesOutput{OrganizationID: input.OrganizationID, StackID: input.StackID, Modules: modules}, nil
}

type ModuleActionService struct {
	Client StackClient
	Action string
}

func (s ModuleActionService) Run(ctx context.Context, input ModuleActionInput) (ModuleActionOutput, error) {
	if err := validateStackTarget(input.OrganizationID, input.StackID); err != nil {
		return ModuleActionOutput{}, err
	}
	if input.Name == "" {
		return ModuleActionOutput{}, fmt.Errorf("module name is required")
	}
	output := ModuleActionOutput{OrganizationID: input.OrganizationID, StackID: input.StackID, Name: input.Name, Action: s.Action}
	switch s.Action {
	case "enable":
		_, err := s.Client.EnableModule(ctx, operations.EnableModuleRequest{OrganizationID: input.OrganizationID, StackID: input.StackID, Name: input.Name})
		return output, err
	case "disable":
		_, err := s.Client.DisableModule(ctx, operations.DisableModuleRequest{OrganizationID: input.OrganizationID, StackID: input.StackID, Name: input.Name})
		return output, err
	default:
		return ModuleActionOutput{}, fmt.Errorf("unsupported module action %q", s.Action)
	}
}

type ListStackUsersService struct {
	Client StackClient
}

func (s ListStackUsersService) Run(ctx context.Context, input StackIDInput) (ListStackUsersOutput, error) {
	if err := validateStackTarget(input.OrganizationID, input.StackID); err != nil {
		return ListStackUsersOutput{}, err
	}
	response, err := s.Client.ListStackUsersAccesses(ctx, operations.ListStackUsersAccessesRequest{OrganizationID: input.OrganizationID, StackID: input.StackID})
	if err != nil {
		return ListStackUsersOutput{}, err
	}
	data := response.GetStackUserAccessResponse().GetData()
	users := make([]StackUserAccessSummary, 0, len(data))
	for _, user := range data {
		users = append(users, StackUserAccessSummary{
			StackID:  user.StackID,
			UserID:   user.UserID,
			Email:    user.Email,
			PolicyID: user.PolicyID,
		})
	}
	return ListStackUsersOutput{OrganizationID: input.OrganizationID, StackID: input.StackID, Users: users}, nil
}

type StackUserAccessService struct {
	Client StackClient
	Action string
}

func (s StackUserAccessService) Run(ctx context.Context, input StackUserAccessInput) (StackUserAccessOutput, error) {
	if err := validateStackTarget(input.OrganizationID, input.StackID); err != nil {
		return StackUserAccessOutput{}, err
	}
	if input.UserID == "" {
		return StackUserAccessOutput{}, fmt.Errorf("user id is required")
	}
	output := StackUserAccessOutput{OrganizationID: input.OrganizationID, StackID: input.StackID, UserID: input.UserID, Action: s.Action, PolicyID: input.PolicyID}
	switch s.Action {
	case "link":
		if input.PolicyID == 0 {
			return StackUserAccessOutput{}, fmt.Errorf("policy id is required")
		}
		_, err := s.Client.UpsertStackUserAccess(ctx, operations.UpsertStackUserAccessRequest{
			OrganizationID: input.OrganizationID,
			StackID:        input.StackID,
			UserID:         input.UserID,
			Body:           &components.UpdateStackUserRequest{PolicyID: input.PolicyID},
		})
		return output, err
	case "unlink":
		_, err := s.Client.DeleteStackUserAccess(ctx, operations.DeleteStackUserAccessRequest{OrganizationID: input.OrganizationID, StackID: input.StackID, UserID: input.UserID})
		return output, err
	default:
		return StackUserAccessOutput{}, fmt.Errorf("unsupported stack user action %q", s.Action)
	}
}

func validateStackTarget(organizationID string, stackID string) error {
	if organizationID == "" {
		return fmt.Errorf("organization id is required")
	}
	if stackID == "" {
		return fmt.Errorf("stack id is required")
	}
	return nil
}

const defaultConsoleURL = "https://portal.formance.cloud"

func stackDashboardURL(ctx context.Context, client StackClient) (string, error) {
	response, err := client.GetServerInfo(ctx)
	if err != nil {
		return "", err
	}
	info := response.GetServerInfo()
	if info != nil && info.GetConsoleURL() != nil {
		return *info.GetConsoleURL(), nil
	}
	return defaultConsoleURL, nil
}

func stackSummary(stack *components.Stack, dashboard string) StackSummary {
	if stack == nil {
		return StackSummary{}
	}
	summary := StackSummary{
		ID:              stack.ID,
		Name:            stack.Name,
		OrganizationID:  stack.OrganizationID,
		Dashboard:       dashboard,
		URI:             stack.URI,
		RegionID:        stack.RegionID,
		Status:          string(stack.Status),
		State:           string(stack.State),
		ExpectedStatus:  string(stack.ExpectedStatus),
		Reachable:       stack.Reachable,
		StargateEnabled: stack.StargateEnabled,
		AuditEnabled:    stack.AuditEnabled,
		Metadata:        stack.Metadata,
		CreatedAt:       stack.CreatedAt,
		UpdatedAt:       stack.UpdatedAt,
		DeletedAt:       stack.DeletedAt,
		DisabledAt:      stack.DisabledAt,
	}
	if stack.Version != nil {
		summary.Version = *stack.Version
	}
	return summary
}
