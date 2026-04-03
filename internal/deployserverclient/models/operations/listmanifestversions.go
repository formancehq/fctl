package operations

import (
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
)

type ListManifestVersionsRequest struct {
	ID string `pathParam:"style=simple,explode=false,name=id"`
}

func (l *ListManifestVersionsRequest) GetID() string {
	if l == nil {
		return ""
	}
	return l.ID
}

type ListManifestVersionsResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Manifest versions retrieved successfully
	ListManifestsResponse *components.ListManifestsResponse
	// Error
	Error *components.Error
}

func (l *ListManifestVersionsResponse) GetHTTPMeta() components.HTTPMetadata {
	if l == nil {
		return components.HTTPMetadata{}
	}
	return l.HTTPMeta
}

func (l *ListManifestVersionsResponse) GetListManifestsResponse() *components.ListManifestsResponse {
	if l == nil {
		return nil
	}
	return l.ListManifestsResponse
}

func (l *ListManifestVersionsResponse) GetError() *components.Error {
	if l == nil {
		return nil
	}
	return l.Error
}
