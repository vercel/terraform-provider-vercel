package client

type VercelAuthentication struct {
	DeploymentType string `json:"deploymentType"`
}

type PasswordProtection struct {
	DeploymentType string `json:"deploymentType"`
}
type PasswordProtectionWithPassword struct {
	DeploymentType string `json:"deploymentType"`
	Password       string `json:"password"`
}

type TrustedIpAddress struct {
	Value string `json:"value"`
	Note  string `json:"note"`
}

type TrustedIps struct {
	DeploymentType string             `json:"deploymentType"`
	Addresses      []TrustedIpAddress `json:"addresses"`
	ProtectionMode string             `json:"protectionMode"`
}

type ProtectionBypass struct {
	Scope string `json:"scope"`
}
