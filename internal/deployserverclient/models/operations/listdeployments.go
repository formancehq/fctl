package operations

import (
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
)

type ListDeploymentsRequest struct {
	ID string `pathParam:"style=simple,explode=false,name=id"`
}

func (l *ListDeploymentsRequest) GetID() string {
	if l == nil {
		return ""
	}
	return l.ID
}

type ListDeploymentsResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Deployments retrieved successfully
	ListDeploymentsResponse *components.ListDeploymentsResponse
	// Error
	Error *components.Error
}

func (l *ListDeploymentsResponse) GetHTTPMeta() components.HTTPMetadata {
	if l == nil {
		return components.HTTPMetadata{}
	}
	return l.HTTPMeta
}

func (l *ListDeploymentsResponse) GetListDeploymentsResponse() *components.ListDeploymentsResponse {
	if l == nil {
		return nil
	}
	return l.ListDeploymentsResponse
}

func (l *ListDeploymentsResponse) GetError() *components.Error {
	if l == nil {
		return nil
	}
	return l.Error
}
