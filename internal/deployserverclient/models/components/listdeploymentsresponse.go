package components

type ListDeploymentsResponse struct {
	Data []Deployment `json:"data"`
}

func (l *ListDeploymentsResponse) GetData() []Deployment {
	if l == nil {
		return nil
	}
	return l.Data
}
