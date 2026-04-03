package operations

import (
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
)

type CreateDeploymentRequest struct {
	ID                      string                              `pathParam:"style=simple,explode=false,name=id"`
	CreateDeploymentRequest components.CreateDeploymentRequest `request:"mediaType=application/json"`
}

func (c *CreateDeploymentRequest) GetID() string {
	if c == nil {
		return ""
	}
	return c.ID
}

func (c *CreateDeploymentRequest) GetCreateDeploymentRequest() components.CreateDeploymentRequest {
	if c == nil {
		return components.CreateDeploymentRequest{}
	}
	return c.CreateDeploymentRequest
}

type CreateDeploymentResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Deployment created successfully
	DeploymentResponse *components.DeploymentResponse
	// Error
	Error *components.Error
}

func (c *CreateDeploymentResponse) GetHTTPMeta() components.HTTPMetadata {
	if c == nil {
		return components.HTTPMetadata{}
	}
	return c.HTTPMeta
}

func (c *CreateDeploymentResponse) GetDeploymentResponse() *components.DeploymentResponse {
	if c == nil {
		return nil
	}
	return c.DeploymentResponse
}

func (c *CreateDeploymentResponse) GetError() *components.Error {
	if c == nil {
		return nil
	}
	return c.Error
}
