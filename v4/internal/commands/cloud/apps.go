package cloud

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"
)

type DeployClient interface {
	ListApps(context.Context, string, *int64, *int64, ...operations.Option) (*operations.ListAppsResponse, error)
	CreateApp(context.Context, components.CreateAppRequest, ...operations.Option) (*operations.CreateAppResponse, error)
	ReadApp(context.Context, string, ...operations.Option) (*operations.ReadAppResponse, error)
	DeleteApp(context.Context, string, ...operations.Option) (*operations.DeleteAppResponse, error)
	DeployAppConfigurationRaw(context.Context, string, any, ...operations.Option) (*operations.DeployAppConfigurationRawResponse, error)
	ReadAppCurrentStateVersion(context.Context, string, ...operations.Option) (*operations.ReadAppCurrentStateVersionResponse, error)
	ReadAppRuns(context.Context, string, *int64, *int64, ...operations.Option) (*operations.ReadAppRunsResponse, error)
	ReadRun(context.Context, string, ...operations.Option) (*operations.ReadRunResponse, error)
	ReadRunLogs(context.Context, string, ...operations.Option) (*operations.ReadRunLogsResponse, error)
	ReadAppVersions(context.Context, string, *int64, *int64, ...operations.Option) (*operations.ReadAppVersionsResponse, error)
	ReadVersion(context.Context, string, ...operations.Option) (*operations.ReadVersionResponse, error)
	ReadAppVariables(context.Context, string, *int64, *int64, ...operations.Option) (*operations.ReadAppVariablesResponse, error)
	CreateAppVariable(context.Context, string, components.CreateVariableRequest, ...operations.Option) (*operations.CreateAppVariableResponse, error)
	DeleteAppVariable(context.Context, string, string, ...operations.Option) (*operations.DeleteAppVariableResponse, error)
}

type CloudAppSummary struct {
	ID                            string         `json:"id" yaml:"id"`
	Name                          string         `json:"name" yaml:"name"`
	RunStatus                     string         `json:"runStatus,omitempty" yaml:"runStatus,omitempty"`
	CurrentConfigurationVersionID string         `json:"currentConfigurationVersionID,omitempty" yaml:"currentConfigurationVersionID,omitempty"`
	CurrentRunID                  string         `json:"currentRunID,omitempty" yaml:"currentRunID,omitempty"`
	StateStack                    map[string]any `json:"stateStack,omitempty" yaml:"stateStack,omitempty"`
}

type ListCloudAppsInput struct {
	OrganizationID string
	Page           int64
	PageSize       int64
}

type ListCloudAppsOutput struct {
	OrganizationID string            `json:"organizationID" yaml:"organizationID"`
	Apps           []CloudAppSummary `json:"apps" yaml:"apps"`
	CurrentPage    *int64            `json:"currentPage,omitempty" yaml:"currentPage,omitempty"`
	NextPage       *int64            `json:"nextPage,omitempty" yaml:"nextPage,omitempty"`
	PreviousPage   *int64            `json:"previousPage,omitempty" yaml:"previousPage,omitempty"`
	TotalPages     *int64            `json:"totalPages,omitempty" yaml:"totalPages,omitempty"`
	TotalCount     *int64            `json:"totalCount,omitempty" yaml:"totalCount,omitempty"`
}

type CloudAppInput struct {
	OrganizationID string
	AppID          string
}

type CloudAppOutput struct {
	OrganizationID string          `json:"organizationID" yaml:"organizationID"`
	App            CloudAppSummary `json:"app" yaml:"app"`
}

type CloudAppActionOutput struct {
	OrganizationID string `json:"organizationID" yaml:"organizationID"`
	AppID          string `json:"appID" yaml:"appID"`
	Action         string `json:"action" yaml:"action"`
}

type CloudRunSummary struct {
	ID                     string    `json:"id" yaml:"id"`
	CreatedAt              time.Time `json:"createdAt" yaml:"createdAt"`
	Status                 string    `json:"status" yaml:"status"`
	Message                string    `json:"message,omitempty" yaml:"message,omitempty"`
	ConfigurationVersionID string    `json:"configurationVersionID,omitempty" yaml:"configurationVersionID,omitempty"`
	AutoApply              bool      `json:"autoApply" yaml:"autoApply"`
	HasChanges             bool      `json:"hasChanges" yaml:"hasChanges"`
	IsDestroy              bool      `json:"isDestroy" yaml:"isDestroy"`
}

type ListCloudRunsOutput struct {
	AppID        string            `json:"appID" yaml:"appID"`
	Runs         []CloudRunSummary `json:"runs" yaml:"runs"`
	CurrentPage  *int64            `json:"currentPage,omitempty" yaml:"currentPage,omitempty"`
	NextPage     *int64            `json:"nextPage,omitempty" yaml:"nextPage,omitempty"`
	PreviousPage *int64            `json:"previousPage,omitempty" yaml:"previousPage,omitempty"`
	TotalPages   *int64            `json:"totalPages,omitempty" yaml:"totalPages,omitempty"`
	TotalCount   *int64            `json:"totalCount,omitempty" yaml:"totalCount,omitempty"`
}

type CloudRunOutput struct {
	Run CloudRunSummary `json:"run" yaml:"run"`
}

type CloudLogSummary struct {
	Timestamp string `json:"timestamp" yaml:"timestamp"`
	Module    string `json:"module,omitempty" yaml:"module,omitempty"`
	Message   string `json:"message" yaml:"message"`
	Severity  string `json:"severity,omitempty" yaml:"severity,omitempty"`
	Summary   string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Detail    string `json:"detail,omitempty" yaml:"detail,omitempty"`
}

type CloudRunLogsOutput struct {
	RunID string            `json:"runID" yaml:"runID"`
	Logs  []CloudLogSummary `json:"logs" yaml:"logs"`
}

type CloudVersionSummary struct {
	ID            string `json:"id" yaml:"id"`
	AutoQueueRuns bool   `json:"autoQueueRuns" yaml:"autoQueueRuns"`
	Status        string `json:"status" yaml:"status"`
	Error         string `json:"error,omitempty" yaml:"error,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty" yaml:"errorMessage,omitempty"`
}

type ListCloudVersionsOutput struct {
	AppID        string                `json:"appID" yaml:"appID"`
	Versions     []CloudVersionSummary `json:"versions" yaml:"versions"`
	CurrentPage  *int64                `json:"currentPage,omitempty" yaml:"currentPage,omitempty"`
	NextPage     *int64                `json:"nextPage,omitempty" yaml:"nextPage,omitempty"`
	PreviousPage *int64                `json:"previousPage,omitempty" yaml:"previousPage,omitempty"`
	TotalPages   *int64                `json:"totalPages,omitempty" yaml:"totalPages,omitempty"`
	TotalCount   *int64                `json:"totalCount,omitempty" yaml:"totalCount,omitempty"`
}

type CloudVersionOutput struct {
	Version CloudVersionSummary `json:"version" yaml:"version"`
}

type CloudVersionBlobOutput struct {
	VersionID string `json:"versionID" yaml:"versionID"`
	Data      []byte `json:"data" yaml:"data"`
}

type CloudVariableSummary struct {
	ID          string `json:"id" yaml:"id"`
	Key         string `json:"key" yaml:"key"`
	Value       string `json:"value,omitempty" yaml:"value,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Sensitive   bool   `json:"sensitive" yaml:"sensitive"`
}

type ListCloudVariablesOutput struct {
	AppID        string                 `json:"appID" yaml:"appID"`
	Variables    []CloudVariableSummary `json:"variables" yaml:"variables"`
	CurrentPage  *int64                 `json:"currentPage,omitempty" yaml:"currentPage,omitempty"`
	NextPage     *int64                 `json:"nextPage,omitempty" yaml:"nextPage,omitempty"`
	PreviousPage *int64                 `json:"previousPage,omitempty" yaml:"previousPage,omitempty"`
	TotalPages   *int64                 `json:"totalPages,omitempty" yaml:"totalPages,omitempty"`
	TotalCount   *int64                 `json:"totalCount,omitempty" yaml:"totalCount,omitempty"`
}

type CloudVariableInput struct {
	AppID       string
	VariableID  string
	Key         string
	Value       string
	Description string
	Sensitive   bool
}

type CloudVariableOutput struct {
	AppID    string               `json:"appID" yaml:"appID"`
	Variable CloudVariableSummary `json:"variable" yaml:"variable"`
}

type CloudVariableActionOutput struct {
	AppID      string `json:"appID" yaml:"appID"`
	VariableID string `json:"variableID" yaml:"variableID"`
	Action     string `json:"action" yaml:"action"`
}

type ListCloudAppsService struct {
	Client DeployClient
}

func (s ListCloudAppsService) Run(ctx context.Context, input ListCloudAppsInput) (ListCloudAppsOutput, error) {
	if s.Client == nil {
		return ListCloudAppsOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.ListApps(ctx, input.OrganizationID, &input.Page, &input.PageSize)
	if err != nil {
		return ListCloudAppsOutput{}, err
	}
	if response.ListAppsResponse == nil {
		return ListCloudAppsOutput{}, fmt.Errorf("cloud apps list returned no data")
	}
	data := response.ListAppsResponse.Data
	apps := make([]CloudAppSummary, 0, len(data.Items))
	for _, app := range data.Items {
		apps = append(apps, cloudAppSummary(app, nil))
	}
	return ListCloudAppsOutput{
		OrganizationID: input.OrganizationID,
		Apps:           apps,
		CurrentPage:    data.CurrentPage,
		NextPage:       data.NextPage,
		PreviousPage:   data.PreviousPage,
		TotalPages:     data.TotalPages,
		TotalCount:     data.TotalCount,
	}, nil
}

type CreateCloudAppService struct {
	Client DeployClient
}

func (s CreateCloudAppService) Run(ctx context.Context, organizationID string) (CloudAppOutput, error) {
	if s.Client == nil {
		return CloudAppOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.CreateApp(ctx, components.CreateAppRequest{OrganizationID: organizationID})
	if err != nil {
		return CloudAppOutput{}, err
	}
	if response.AppResponse == nil {
		return CloudAppOutput{}, fmt.Errorf("cloud apps create returned no data")
	}
	return CloudAppOutput{OrganizationID: organizationID, App: cloudAppSummary(response.AppResponse.Data, nil)}, nil
}

type GetCloudAppService struct {
	Client DeployClient
}

func (s GetCloudAppService) Run(ctx context.Context, input CloudAppInput) (CloudAppOutput, error) {
	if s.Client == nil {
		return CloudAppOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.ReadApp(ctx, input.AppID)
	if err != nil {
		return CloudAppOutput{}, err
	}
	if response.AppResponse == nil {
		return CloudAppOutput{}, fmt.Errorf("cloud apps show returned no data")
	}
	var stack map[string]any
	if stateResponse, err := s.Client.ReadAppCurrentStateVersion(ctx, input.AppID); err == nil && stateResponse.ReadStateResponse != nil {
		stack = stateResponse.ReadStateResponse.Data.Stack
	}
	return CloudAppOutput{OrganizationID: input.OrganizationID, App: cloudAppSummary(response.AppResponse.Data, stack)}, nil
}

type DeleteCloudAppService struct {
	Client DeployClient
}

func (s DeleteCloudAppService) Run(ctx context.Context, input CloudAppInput) (CloudAppActionOutput, error) {
	if s.Client == nil {
		return CloudAppActionOutput{}, fmt.Errorf("deploy client is required")
	}
	if _, err := s.Client.DeleteApp(ctx, input.AppID); err != nil {
		return CloudAppActionOutput{}, err
	}
	return CloudAppActionOutput{OrganizationID: input.OrganizationID, AppID: input.AppID, Action: "deleted"}, nil
}

type DeployCloudAppService struct {
	Client DeployClient
}

func (s DeployCloudAppService) Run(ctx context.Context, appID string, data []byte) (CloudRunOutput, error) {
	if s.Client == nil {
		return CloudRunOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.DeployAppConfigurationRaw(ctx, appID, data)
	if err != nil {
		return CloudRunOutput{}, err
	}
	if response.RunResponse == nil {
		return CloudRunOutput{}, fmt.Errorf("cloud apps deploy returned no run")
	}
	return CloudRunOutput{Run: cloudRunSummary(response.RunResponse.Data)}, nil
}

type ListCloudRunsService struct {
	Client DeployClient
}

func (s ListCloudRunsService) Run(ctx context.Context, appID string, page int64, pageSize int64) (ListCloudRunsOutput, error) {
	if s.Client == nil {
		return ListCloudRunsOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.ReadAppRuns(ctx, appID, &page, &pageSize)
	if err != nil {
		return ListCloudRunsOutput{}, err
	}
	if response.ListRunsResponse == nil {
		return ListCloudRunsOutput{}, fmt.Errorf("cloud apps runs list returned no data")
	}
	data := response.ListRunsResponse.Data
	runs := make([]CloudRunSummary, 0, len(data.Items))
	for _, run := range data.Items {
		runs = append(runs, cloudRunSummary(run))
	}
	return ListCloudRunsOutput{AppID: appID, Runs: runs, CurrentPage: data.CurrentPage, NextPage: data.NextPage, PreviousPage: data.PreviousPage, TotalPages: data.TotalPages, TotalCount: data.TotalCount}, nil
}

type GetCloudRunService struct {
	Client DeployClient
}

func (s GetCloudRunService) Run(ctx context.Context, runID string) (CloudRunOutput, error) {
	if s.Client == nil {
		return CloudRunOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.ReadRun(ctx, runID)
	if err != nil {
		return CloudRunOutput{}, err
	}
	if response.RunResponse == nil {
		return CloudRunOutput{}, fmt.Errorf("cloud apps runs show returned no data")
	}
	return CloudRunOutput{Run: cloudRunSummary(response.RunResponse.Data)}, nil
}

type GetCloudRunLogsService struct {
	Client DeployClient
}

func (s GetCloudRunLogsService) Run(ctx context.Context, runID string) (CloudRunLogsOutput, error) {
	if s.Client == nil {
		return CloudRunLogsOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.ReadRunLogs(ctx, runID)
	if err != nil {
		return CloudRunLogsOutput{}, err
	}
	if response.ReadLogsResponse == nil {
		return CloudRunLogsOutput{}, fmt.Errorf("cloud apps runs logs returned no data")
	}
	logs := make([]CloudLogSummary, 0, len(response.ReadLogsResponse.Data))
	for _, log := range response.ReadLogsResponse.Data {
		logs = append(logs, cloudLogSummary(log))
	}
	return CloudRunLogsOutput{RunID: runID, Logs: logs}, nil
}

type ListCloudVersionsService struct {
	Client DeployClient
}

func (s ListCloudVersionsService) Run(ctx context.Context, appID string, page int64, pageSize int64) (ListCloudVersionsOutput, error) {
	if s.Client == nil {
		return ListCloudVersionsOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.ReadAppVersions(ctx, appID, &page, &pageSize)
	if err != nil {
		return ListCloudVersionsOutput{}, err
	}
	if response.ListVersionsResponse == nil {
		return ListCloudVersionsOutput{}, fmt.Errorf("cloud apps versions list returned no data")
	}
	data := response.ListVersionsResponse.Data
	versions := make([]CloudVersionSummary, 0, len(data.Items))
	for _, version := range data.Items {
		versions = append(versions, cloudVersionSummary(version))
	}
	return ListCloudVersionsOutput{AppID: appID, Versions: versions, CurrentPage: data.CurrentPage, NextPage: data.NextPage, PreviousPage: data.PreviousPage, TotalPages: data.TotalPages, TotalCount: data.TotalCount}, nil
}

type GetCloudVersionService struct {
	Client DeployClient
}

func (s GetCloudVersionService) Run(ctx context.Context, versionID string) (CloudVersionOutput, error) {
	if s.Client == nil {
		return CloudVersionOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.ReadVersion(ctx, versionID)
	if err != nil {
		return CloudVersionOutput{}, err
	}
	if response.AppVersionResponse == nil {
		return CloudVersionOutput{}, fmt.Errorf("cloud apps versions show returned no data")
	}
	return CloudVersionOutput{Version: cloudVersionSummary(response.AppVersionResponse.Data)}, nil
}

type GetCloudVersionBlobService struct {
	Client DeployClient
	Accept operations.AcceptHeaderEnum
}

func (s GetCloudVersionBlobService) Run(ctx context.Context, versionID string) (CloudVersionBlobOutput, error) {
	if s.Client == nil {
		return CloudVersionBlobOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.ReadVersion(ctx, versionID, operations.WithAcceptHeaderOverride(s.Accept))
	if err != nil {
		return CloudVersionBlobOutput{}, err
	}
	var stream io.ReadCloser
	switch s.Accept {
	case operations.AcceptHeaderEnumApplicationYaml:
		stream = response.TwoHundredApplicationYamlResponseStream
	case operations.AcceptHeaderEnumApplicationGzip:
		stream = response.TwoHundredApplicationGzipResponseStream
	}
	if stream == nil {
		return CloudVersionBlobOutput{}, fmt.Errorf("cloud apps versions returned no stream")
	}
	defer stream.Close()
	data, err := io.ReadAll(stream)
	if err != nil {
		return CloudVersionBlobOutput{}, err
	}
	return CloudVersionBlobOutput{VersionID: versionID, Data: data}, nil
}

type ListCloudVariablesService struct {
	Client DeployClient
}

func (s ListCloudVariablesService) Run(ctx context.Context, appID string, page int64, pageSize int64) (ListCloudVariablesOutput, error) {
	if s.Client == nil {
		return ListCloudVariablesOutput{}, fmt.Errorf("deploy client is required")
	}
	response, err := s.Client.ReadAppVariables(ctx, appID, &page, &pageSize)
	if err != nil {
		return ListCloudVariablesOutput{}, err
	}
	if response.ReadVariablesResponse == nil {
		return ListCloudVariablesOutput{}, fmt.Errorf("cloud apps variables list returned no data")
	}
	data := response.ReadVariablesResponse.Data
	variables := make([]CloudVariableSummary, 0, len(data.Items))
	for _, variable := range data.Items {
		variables = append(variables, cloudVariableSummary(variable))
	}
	return ListCloudVariablesOutput{AppID: appID, Variables: variables, CurrentPage: data.CurrentPage, NextPage: data.NextPage, PreviousPage: data.PreviousPage, TotalPages: data.TotalPages, TotalCount: data.TotalCount}, nil
}

type CreateCloudVariableService struct {
	Client DeployClient
}

func (s CreateCloudVariableService) Run(ctx context.Context, input CloudVariableInput) (CloudVariableOutput, error) {
	if s.Client == nil {
		return CloudVariableOutput{}, fmt.Errorf("deploy client is required")
	}
	var description *string
	if input.Description != "" {
		description = &input.Description
	}
	response, err := s.Client.CreateAppVariable(ctx, input.AppID, components.CreateVariableRequest{
		Variable: components.VariableData{
			Key:         input.Key,
			Value:       input.Value,
			Description: description,
			Sensitive:   input.Sensitive,
		},
	})
	if err != nil {
		return CloudVariableOutput{}, err
	}
	if response.CreateVariableResponse == nil {
		return CloudVariableOutput{}, fmt.Errorf("cloud apps variables create returned no data")
	}
	return CloudVariableOutput{AppID: input.AppID, Variable: cloudVariableSummary(response.CreateVariableResponse.Data)}, nil
}

type DeleteCloudVariableService struct {
	Client DeployClient
}

func (s DeleteCloudVariableService) Run(ctx context.Context, input CloudVariableInput) (CloudVariableActionOutput, error) {
	if s.Client == nil {
		return CloudVariableActionOutput{}, fmt.Errorf("deploy client is required")
	}
	if _, err := s.Client.DeleteAppVariable(ctx, input.AppID, input.VariableID); err != nil {
		return CloudVariableActionOutput{}, err
	}
	return CloudVariableActionOutput{AppID: input.AppID, VariableID: input.VariableID, Action: "deleted"}, nil
}

func cloudAppSummary(app components.App, stateStack map[string]any) CloudAppSummary {
	summary := CloudAppSummary{ID: app.ID, Name: app.Name, StateStack: stateStack}
	if app.CurrentRun != nil {
		summary.CurrentRunID = app.CurrentRun.ID
		summary.RunStatus = app.CurrentRun.Status
	}
	if app.CurrentConfigurationVersion != nil {
		summary.CurrentConfigurationVersionID = app.CurrentConfigurationVersion.ID
	}
	return summary
}

func cloudRunSummary(run components.Run) CloudRunSummary {
	versionID := ""
	if run.ConfigurationVersion != nil {
		versionID = run.ConfigurationVersion.ID
	}
	return CloudRunSummary{
		ID:                     run.ID,
		CreatedAt:              run.CreatedAt,
		Status:                 run.Status,
		Message:                run.Message,
		ConfigurationVersionID: versionID,
		AutoApply:              run.AutoApply,
		HasChanges:             run.HasChanges,
		IsDestroy:              run.IsDestroy,
	}
}

func cloudLogSummary(log components.Log) CloudLogSummary {
	summary := CloudLogSummary{
		Timestamp: log.Timestamp.Format(time.RFC3339),
		Module:    log.Module,
		Message:   log.Message,
	}
	if log.Diagnostic != nil {
		summary.Severity = log.Diagnostic.Severity
		summary.Summary = log.Diagnostic.Summary
		summary.Detail = log.Diagnostic.Detail
	}
	return summary
}

func cloudVersionSummary(version components.ConfigurationVersion) CloudVersionSummary {
	return CloudVersionSummary{
		ID:            version.ID,
		AutoQueueRuns: version.AutoQueueRuns,
		Status:        string(version.Status),
		Error:         version.Error,
		ErrorMessage:  version.ErrorMessage,
	}
}

func cloudVariableSummary(variable components.Variable) CloudVariableSummary {
	value := variable.Value
	if variable.Sensitive {
		value = ""
	}
	description := ""
	if variable.Description != nil {
		description = *variable.Description
	}
	return CloudVariableSummary{
		ID:          variable.ID,
		Key:         variable.Key,
		Value:       value,
		Description: description,
		Sensitive:   variable.Sensitive,
	}
}
