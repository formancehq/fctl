package components

type Deployment struct {
	// Name of the deployment
	Name string `json:"name"`
	// ID of the parent app
	AppID string `json:"appId"`
	// ID of the Formance Cloud stack
	StackID string `json:"stackId"`
	// ID of the TFC workspace
	WorkspaceID string `json:"workspaceId"`
}

func (d *Deployment) GetName() string {
	if d == nil {
		return ""
	}
	return d.Name
}

func (d *Deployment) GetAppID() string {
	if d == nil {
		return ""
	}
	return d.AppID
}

func (d *Deployment) GetStackID() string {
	if d == nil {
		return ""
	}
	return d.StackID
}

func (d *Deployment) GetWorkspaceID() string {
	if d == nil {
		return ""
	}
	return d.WorkspaceID
}
