package operations

import (
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
)

type DeleteDeploymentRequest struct {
	ID   string `pathParam:"style=simple,explode=false,name=id"`
	Name string `pathParam:"style=simple,explode=false,name=name"`
}

func (d *DeleteDeploymentRequest) GetID() string {
	if d == nil {
		return ""
	}
	return d.ID
}

func (d *DeleteDeploymentRequest) GetName() string {
	if d == nil {
		return ""
	}
	return d.Name
}

type DeleteDeploymentResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Error
	Error *components.Error
}

func (d *DeleteDeploymentResponse) GetHTTPMeta() components.HTTPMetadata {
	if d == nil {
		return components.HTTPMetadata{}
	}
	return d.HTTPMeta
}

func (d *DeleteDeploymentResponse) GetError() *components.Error {
	if d == nil {
		return nil
	}
	return d.Error
}
