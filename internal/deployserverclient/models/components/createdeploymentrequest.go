package components

type CreateDeploymentRequest struct {
	// Name for the deployment
	Name string `json:"name"`
	// ID of the Formance Cloud stack to deploy to
	StackID string `json:"stackId"`
}

func (c *CreateDeploymentRequest) GetName() string {
	if c == nil {
		return ""
	}
	return c.Name
}

func (c *CreateDeploymentRequest) GetStackID() string {
	if c == nil {
		return ""
	}
	return c.StackID
}
