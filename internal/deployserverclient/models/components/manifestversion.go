package components

import "time"

type ManifestVersion struct {
	// ID of the parent app
	AppID string `json:"appId"`
	// Manifest version number
	Version int `json:"version"`
	// Raw manifest content
	Content string `json:"content"`
	// When this version was pushed
	CreatedAt time.Time `json:"createdAt"`
}

func (m *ManifestVersion) GetAppID() string {
	if m == nil {
		return ""
	}
	return m.AppID
}

func (m *ManifestVersion) GetVersion() int {
	if m == nil {
		return 0
	}
	return m.Version
}

func (m *ManifestVersion) GetContent() string {
	if m == nil {
		return ""
	}
	return m.Content
}

func (m *ManifestVersion) GetCreatedAt() time.Time {
	if m == nil {
		return time.Time{}
	}
	return m.CreatedAt
}
