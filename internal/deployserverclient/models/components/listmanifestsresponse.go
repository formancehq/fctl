package components

type ListManifestsResponse struct {
	Data []ManifestVersion `json:"data"`
}

func (l *ListManifestsResponse) GetData() []ManifestVersion {
	if l == nil {
		return nil
	}
	return l.Data
}
