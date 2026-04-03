package components

type DeploymentResponse struct {
	Data Deployment `json:"data"`
}

func (d *DeploymentResponse) GetData() Deployment {
	if d == nil {
		return Deployment{}
	}
	return d.Data
}
