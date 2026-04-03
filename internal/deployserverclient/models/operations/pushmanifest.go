package operations

import (
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
)

type PushManifestRequest struct {
	ID string `pathParam:"style=simple,explode=false,name=id"`
	// This field accepts []byte data or io.Reader implementations, such as *os.File.
	RequestBody any `request:"mediaType=application/yaml"`
}

func (p *PushManifestRequest) GetID() string {
	if p == nil {
		return ""
	}
	return p.ID
}

func (p *PushManifestRequest) GetRequestBody() any {
	if p == nil {
		return nil
	}
	return p.RequestBody
}

type PushManifestResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Manifest pushed successfully
	ManifestVersionResponse *components.ManifestVersionResponse
	// Error
	Error *components.Error
}

func (p *PushManifestResponse) GetHTTPMeta() components.HTTPMetadata {
	if p == nil {
		return components.HTTPMetadata{}
	}
	return p.HTTPMeta
}

func (p *PushManifestResponse) GetManifestVersionResponse() *components.ManifestVersionResponse {
	if p == nil {
		return nil
	}
	return p.ManifestVersionResponse
}

func (p *PushManifestResponse) GetError() *components.Error {
	if p == nil {
		return nil
	}
	return p.Error
}
