package operations

import (
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
)

type DeployToDeploymentRequest struct {
	ID      string `pathParam:"style=simple,explode=false,name=id"`
	Name    string `pathParam:"style=simple,explode=false,name=name"`
	Version *int64 `queryParam:"style=form,explode=true,name=version"`
}

func (d *DeployToDeploymentRequest) GetID() string {
	if d == nil {
		return ""
	}
	return d.ID
}

func (d *DeployToDeploymentRequest) GetName() string {
	if d == nil {
		return ""
	}
	return d.Name
}

func (d *DeployToDeploymentRequest) GetVersion() *int64 {
	if d == nil {
		return nil
	}
	return d.Version
}

type DeployToDeploymentResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Manifest deployed successfully
	RunResponse *components.RunResponse
	// Error
	Error *components.Error
}

func (d *DeployToDeploymentResponse) GetHTTPMeta() components.HTTPMetadata {
	if d == nil {
		return components.HTTPMetadata{}
	}
	return d.HTTPMeta
}

func (d *DeployToDeploymentResponse) GetRunResponse() *components.RunResponse {
	if d == nil {
		return nil
	}
	return d.RunResponse
}

func (d *DeployToDeploymentResponse) GetError() *components.Error {
	if d == nil {
		return nil
	}
	return d.Error
}
