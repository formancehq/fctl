package components

type ManifestVersionResponse struct {
	Data ManifestVersion `json:"data"`
}

func (m *ManifestVersionResponse) GetData() ManifestVersion {
	if m == nil {
		return ManifestVersion{}
	}
	return m.Data
}
