package client

// Passport selects the Connect OAuth application used for deployment protection.
type Passport struct {
	ConnectorID    string `json:"connectorId"`
	DeploymentType string `json:"deploymentType"`
}
