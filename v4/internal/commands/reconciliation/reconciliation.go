package reconciliation

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const (
	ProductReconciliation capabilities.Product = "reconciliation"

	FeatureCreatePolicy        capabilities.Feature = "createPolicy"
	FeatureDeletePolicy        capabilities.Feature = "deletePolicy"
	FeatureGetPolicy           capabilities.Feature = "getPolicy"
	FeatureGetReconciliation   capabilities.Feature = "getReconciliation"
	FeatureListPolicies        capabilities.Feature = "listPolicies"
	FeatureListReconciliations capabilities.Feature = "listReconciliations"
	FeatureReconcile           capabilities.Feature = "reconcile"
)

type PolicySummary struct {
	ID             string         `json:"id" yaml:"id"`
	Name           string         `json:"name" yaml:"name"`
	LedgerName     string         `json:"ledgerName" yaml:"ledgerName"`
	PaymentsPoolID string         `json:"paymentsPoolID" yaml:"paymentsPoolID"`
	LedgerQuery    map[string]any `json:"ledgerQuery,omitempty" yaml:"ledgerQuery,omitempty"`
	CreatedAt      time.Time      `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
}

type ReconciliationSummary struct {
	ID                   string              `json:"id" yaml:"id"`
	PolicyID             string              `json:"policyID" yaml:"policyID"`
	Status               string              `json:"status" yaml:"status"`
	Error                string              `json:"error,omitempty" yaml:"error,omitempty"`
	LedgerBalances       map[string]*big.Int `json:"ledgerBalances,omitempty" yaml:"ledgerBalances,omitempty"`
	PaymentsBalances     map[string]*big.Int `json:"paymentsBalances,omitempty" yaml:"paymentsBalances,omitempty"`
	DriftBalances        map[string]*big.Int `json:"driftBalances,omitempty" yaml:"driftBalances,omitempty"`
	ReconciledAtLedger   time.Time           `json:"reconciledAtLedger,omitempty" yaml:"reconciledAtLedger,omitempty"`
	ReconciledAtPayments time.Time           `json:"reconciledAtPayments,omitempty" yaml:"reconciledAtPayments,omitempty"`
	CreatedAt            time.Time           `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
}

type ListPoliciesInput struct {
	PageSize int64
	Cursor   string
	Query    map[string]any
}

type ListPoliciesOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Policies   []PolicySummary         `json:"policies" yaml:"policies"`
	HasMore    bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize   int64                   `json:"pageSize" yaml:"pageSize"`
	Next       *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous   *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type CreatePolicyInput struct {
	Payload []byte
}

type CreatePolicyOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Policy     PolicySummary           `json:"policy" yaml:"policy"`
}

type GetPolicyInput struct {
	PolicyID string
}

type GetPolicyOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Policy     PolicySummary           `json:"policy" yaml:"policy"`
}

type DeletePolicyInput struct {
	PolicyID string
}

type DeletePolicyOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	PolicyID   string                  `json:"policyID" yaml:"policyID"`
}

type ListReconciliationsInput struct {
	PageSize int64
	Cursor   string
	Query    map[string]any
}

type ListReconciliationsOutput struct {
	APIVersion      capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Reconciliations []ReconciliationSummary `json:"reconciliations" yaml:"reconciliations"`
	HasMore         bool                    `json:"hasMore" yaml:"hasMore"`
	PageSize        int64                   `json:"pageSize" yaml:"pageSize"`
	Next            *string                 `json:"next,omitempty" yaml:"next,omitempty"`
	Previous        *string                 `json:"previous,omitempty" yaml:"previous,omitempty"`
}

type GetReconciliationInput struct {
	ReconciliationID string
}

type GetReconciliationOutput struct {
	APIVersion     capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Reconciliation ReconciliationSummary   `json:"reconciliation" yaml:"reconciliation"`
}

type ReconcileInput struct {
	PolicyID             string
	ReconciledAtLedger   time.Time
	ReconciledAtPayments time.Time
}

type ReconcileOutput struct {
	APIVersion     capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Reconciliation ReconciliationSummary   `json:"reconciliation" yaml:"reconciliation"`
}

type ListPoliciesHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListPoliciesInput) (ListPoliciesOutput, error)
}

type CreatePolicyHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, CreatePolicyInput) (CreatePolicyOutput, error)
}

type GetPolicyHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetPolicyInput) (GetPolicyOutput, error)
}

type DeletePolicyHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, DeletePolicyInput) (DeletePolicyOutput, error)
}

type ListReconciliationsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ListReconciliationsInput) (ListReconciliationsOutput, error)
}

type GetReconciliationHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, GetReconciliationInput) (GetReconciliationOutput, error)
}

type ReconcileHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ReconcileInput) (ReconcileOutput, error)
}

type ListPoliciesService struct {
	Handlers []ListPoliciesHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type CreatePolicyService struct {
	Handlers []CreatePolicyHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetPolicyService struct {
	Handlers []GetPolicyHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type DeletePolicyService struct {
	Handlers []DeletePolicyHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ListReconciliationsService struct {
	Handlers []ListReconciliationsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type GetReconciliationService struct {
	Handlers []GetReconciliationHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

type ReconcileService struct {
	Handlers []ReconcileHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ListPoliciesService) Run(ctx context.Context, input ListPoliciesInput) (ListPoliciesOutput, error) {
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler ListPoliciesHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ListPoliciesOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListPoliciesOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s CreatePolicyService) Run(ctx context.Context, input CreatePolicyInput) (CreatePolicyOutput, error) {
	if len(input.Payload) == 0 {
		return CreatePolicyOutput{}, fmt.Errorf("policy request is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler CreatePolicyHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return CreatePolicyOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return CreatePolicyOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetPolicyService) Run(ctx context.Context, input GetPolicyInput) (GetPolicyOutput, error) {
	if input.PolicyID == "" {
		return GetPolicyOutput{}, fmt.Errorf("policy id is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler GetPolicyHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return GetPolicyOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetPolicyOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s DeletePolicyService) Run(ctx context.Context, input DeletePolicyInput) (DeletePolicyOutput, error) {
	if input.PolicyID == "" {
		return DeletePolicyOutput{}, fmt.Errorf("policy id is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler DeletePolicyHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return DeletePolicyOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return DeletePolicyOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s ListReconciliationsService) Run(ctx context.Context, input ListReconciliationsInput) (ListReconciliationsOutput, error) {
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler ListReconciliationsHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ListReconciliationsOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ListReconciliationsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s GetReconciliationService) Run(ctx context.Context, input GetReconciliationInput) (GetReconciliationOutput, error) {
	if input.ReconciliationID == "" {
		return GetReconciliationOutput{}, fmt.Errorf("reconciliation id is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler GetReconciliationHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return GetReconciliationOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return GetReconciliationOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func (s ReconcileService) Run(ctx context.Context, input ReconcileInput) (ReconcileOutput, error) {
	if input.PolicyID == "" {
		return ReconcileOutput{}, fmt.Errorf("policy id is required")
	}
	if input.ReconciledAtLedger.IsZero() {
		return ReconcileOutput{}, fmt.Errorf("ledger reconciliation time is required")
	}
	if input.ReconciledAtPayments.IsZero() {
		return ReconcileOutput{}, fmt.Errorf("payments reconciliation time is required")
	}
	handler, selected, err := resolveHandler(ctx, s.Handlers, s.Resolve, func(handler ReconcileHandler) capabilities.APIVersion {
		return handler.APIVersion
	})
	if err != nil {
		return ReconcileOutput{}, err
	}
	output, err := handler.Run(ctx, input)
	if err != nil {
		return ReconcileOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func resolveHandler[H any](
	ctx context.Context,
	serviceHandlers []H,
	resolve func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error),
	versionOf func(H) capabilities.APIVersion,
) (H, capabilities.APIVersion, error) {
	var zero H
	handlerVersions := make([]capabilities.APIVersion, 0, len(serviceHandlers))
	handlers := map[capabilities.APIVersion]H{}
	for _, handler := range serviceHandlers {
		version := versionOf(handler)
		handlerVersions = append(handlerVersions, version)
		handlers[version] = handler
	}
	selected, err := resolve(ctx, handlerVersions)
	if err != nil {
		return zero, "", err
	}
	handler, ok := handlers[selected]
	if !ok {
		return zero, "", fmt.Errorf("resolved api version %s has no handler", selected)
	}
	return handler, selected, nil
}

func SDKListPoliciesHandlers(sdk *formance.Formance) []ListPoliciesHandler {
	return []ListPoliciesHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListPoliciesInput) (ListPoliciesOutput, error) {
				response, err := sdk.Reconciliation.V1.ListPolicies(ctx, operations.ListPoliciesRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
					Query:    input.Query,
				})
				if err != nil {
					return ListPoliciesOutput{}, err
				}
				if response.PoliciesCursorResponse == nil {
					return ListPoliciesOutput{}, fmt.Errorf("reconciliation v1 list policies returned no cursor")
				}
				cursor := response.PoliciesCursorResponse.Cursor
				policies := make([]PolicySummary, 0, len(cursor.Data))
				for _, policy := range cursor.Data {
					policies = append(policies, fromPolicy(policy))
				}
				return ListPoliciesOutput{
					Policies: policies,
					HasMore:  cursor.HasMore,
					PageSize: cursor.PageSize,
					Next:     cursor.Next,
					Previous: cursor.Previous,
				}, nil
			},
		},
	}
}

func SDKCreatePolicyHandlers(sdk *formance.Formance) []CreatePolicyHandler {
	return []CreatePolicyHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input CreatePolicyInput) (CreatePolicyOutput, error) {
				var request shared.PolicyRequest
				if err := json.Unmarshal(input.Payload, &request); err != nil {
					return CreatePolicyOutput{}, err
				}
				response, err := sdk.Reconciliation.V1.CreatePolicy(ctx, request)
				if err != nil {
					return CreatePolicyOutput{}, err
				}
				if response.PolicyResponse == nil {
					return CreatePolicyOutput{}, fmt.Errorf("reconciliation v1 create policy returned no data")
				}
				return CreatePolicyOutput{Policy: fromPolicy(response.PolicyResponse.Data)}, nil
			},
		},
	}
}

func SDKGetPolicyHandlers(sdk *formance.Formance) []GetPolicyHandler {
	return []GetPolicyHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetPolicyInput) (GetPolicyOutput, error) {
				response, err := sdk.Reconciliation.V1.GetPolicy(ctx, operations.GetPolicyRequest{PolicyID: input.PolicyID})
				if err != nil {
					return GetPolicyOutput{}, err
				}
				if response.PolicyResponse == nil {
					return GetPolicyOutput{}, fmt.Errorf("reconciliation v1 get policy returned no data")
				}
				return GetPolicyOutput{Policy: fromPolicy(response.PolicyResponse.Data)}, nil
			},
		},
	}
}

func SDKDeletePolicyHandlers(sdk *formance.Formance) []DeletePolicyHandler {
	return []DeletePolicyHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input DeletePolicyInput) (DeletePolicyOutput, error) {
				if _, err := sdk.Reconciliation.V1.DeletePolicy(ctx, operations.DeletePolicyRequest{PolicyID: input.PolicyID}); err != nil {
					return DeletePolicyOutput{}, err
				}
				return DeletePolicyOutput{PolicyID: input.PolicyID}, nil
			},
		},
	}
}

func SDKListReconciliationsHandlers(sdk *formance.Formance) []ListReconciliationsHandler {
	return []ListReconciliationsHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ListReconciliationsInput) (ListReconciliationsOutput, error) {
				response, err := sdk.Reconciliation.V1.ListReconciliations(ctx, operations.ListReconciliationsRequest{
					PageSize: pointer(input.PageSize),
					Cursor:   optionalString(input.Cursor),
					Query:    input.Query,
				})
				if err != nil {
					return ListReconciliationsOutput{}, err
				}
				if response.ReconciliationsCursorResponse == nil {
					return ListReconciliationsOutput{}, fmt.Errorf("reconciliation v1 list reconciliations returned no cursor")
				}
				cursor := response.ReconciliationsCursorResponse.Cursor
				reconciliations := make([]ReconciliationSummary, 0, len(cursor.Data))
				for _, reconciliation := range cursor.Data {
					reconciliations = append(reconciliations, fromReconciliation(reconciliation))
				}
				return ListReconciliationsOutput{
					Reconciliations: reconciliations,
					HasMore:         cursor.HasMore,
					PageSize:        cursor.PageSize,
					Next:            cursor.Next,
					Previous:        cursor.Previous,
				}, nil
			},
		},
	}
}

func SDKGetReconciliationHandlers(sdk *formance.Formance) []GetReconciliationHandler {
	return []GetReconciliationHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input GetReconciliationInput) (GetReconciliationOutput, error) {
				response, err := sdk.Reconciliation.V1.GetReconciliation(ctx, operations.GetReconciliationRequest{ReconciliationID: input.ReconciliationID})
				if err != nil {
					return GetReconciliationOutput{}, err
				}
				if response.ReconciliationResponse == nil {
					return GetReconciliationOutput{}, fmt.Errorf("reconciliation v1 get reconciliation returned no data")
				}
				return GetReconciliationOutput{Reconciliation: fromReconciliation(response.ReconciliationResponse.Data)}, nil
			},
		},
	}
}

func SDKReconcileHandlers(sdk *formance.Formance) []ReconcileHandler {
	return []ReconcileHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context, input ReconcileInput) (ReconcileOutput, error) {
				response, err := sdk.Reconciliation.V1.Reconcile(ctx, operations.ReconcileRequest{
					PolicyID: input.PolicyID,
					ReconciliationRequest: shared.ReconciliationRequest{
						ReconciledAtLedger:   input.ReconciledAtLedger,
						ReconciledAtPayments: input.ReconciledAtPayments,
					},
				})
				if err != nil {
					return ReconcileOutput{}, err
				}
				if response.ReconciliationResponse == nil {
					return ReconcileOutput{}, fmt.Errorf("reconciliation v1 reconcile returned no data")
				}
				return ReconcileOutput{Reconciliation: fromReconciliation(response.ReconciliationResponse.Data)}, nil
			},
		},
	}
}

func fromPolicy(policy shared.Policy) PolicySummary {
	return PolicySummary{
		ID:             policy.ID,
		Name:           policy.Name,
		LedgerName:     policy.LedgerName,
		PaymentsPoolID: policy.PaymentsPoolID,
		LedgerQuery:    policy.LedgerQuery,
		CreatedAt:      policy.CreatedAt,
	}
}

func fromReconciliation(reconciliation shared.Reconciliation) ReconciliationSummary {
	errorMessage := ""
	if reconciliation.Error != nil {
		errorMessage = *reconciliation.Error
	}
	return ReconciliationSummary{
		ID:                   reconciliation.ID,
		PolicyID:             reconciliation.PolicyID,
		Status:               reconciliation.Status,
		Error:                errorMessage,
		LedgerBalances:       reconciliation.LedgerBalances,
		PaymentsBalances:     reconciliation.PaymentsBalances,
		DriftBalances:        reconciliation.DriftBalances,
		ReconciledAtLedger:   reconciliation.ReconciledAtLedger,
		ReconciledAtPayments: reconciliation.ReconciledAtPayments,
		CreatedAt:            reconciliation.CreatedAt,
	}
}

func pointer[T any](value T) *T {
	return &value
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
