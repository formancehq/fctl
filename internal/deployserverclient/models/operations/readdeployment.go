package operations

import (
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
)

type ReadDeploymentRequest struct {
	ID   string `pathParam:"style=simple,explode=false,name=id"`
	Name string `pathParam:"style=simple,explode=false,name=name"`
}

func (r *ReadDeploymentRequest) GetID() string {
	if r == nil {
		return ""
	}
	return r.ID
}

func (r *ReadDeploymentRequest) GetName() string {
	if r == nil {
		return ""
	}
	return r.Name
}

type ReadDeploymentResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Deployment retrieved successfully
	DeploymentResponse *components.DeploymentResponse
	// Error
	Error *components.Error
}

func (r *ReadDeploymentResponse) GetHTTPMeta() components.HTTPMetadata {
	if r == nil {
		return components.HTTPMetadata{}
	}
	return r.HTTPMeta
}

func (r *ReadDeploymentResponse) GetDeploymentResponse() *components.DeploymentResponse {
	if r == nil {
		return nil
	}
	return r.DeploymentResponse
}

func (r *ReadDeploymentResponse) GetError() *components.Error {
	if r == nil {
		return nil
	}
	return r.Error
}
