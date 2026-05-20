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
	Value string  `json:"value"`
	Note  *string `json:"note,omitempty"`
}

type TrustedIps struct {
	DeploymentType string             `json:"deploymentType"`
	Addresses      []TrustedIpAddress `json:"addresses"`
	ProtectionMode string             `json:"protectionMode"`
}

type TrustedSourcesClaims map[string][]string

type TrustedSourcesEnvMatcher struct {
	Slugs  []string `json:"slugs,omitempty"`
	Preset *string  `json:"preset,omitempty"`
}

type TrustedSourcesTargetAccess struct {
	To TrustedSourcesEnvMatcher `json:"to"`
}

type TrustedSourcesAccessRule struct {
	From TrustedSourcesEnvMatcher `json:"from"`
	To   TrustedSourcesEnvMatcher `json:"to"`
}

type TrustedSourcesProject struct {
	Label       *string                    `json:"label,omitempty"`
	CustomAllow []TrustedSourcesAccessRule `json:"customAllow,omitempty"`
}

type TrustedSourcesOIDCProvider struct {
	TrustedSourcesTargetAccess
	Label  *string              `json:"label,omitempty"`
	Claims TrustedSourcesClaims `json:"claims"`
}

type TrustedSources struct {
	Projects      map[string]TrustedSourcesProject        `json:"projects,omitempty"`
	OIDCProviders map[string][]TrustedSourcesOIDCProvider `json:"oidcProviders,omitempty"`
}

type ProtectionBypass struct {
	Scope           string  `json:"scope"`
	IsEnvVar        *bool   `json:"isEnvVar,omitempty"`
	Note            *string `json:"note,omitempty"`
	CreatedAt       int64   `json:"createdAt,omitempty"`
	CreatedBy       string  `json:"createdBy,omitempty"`
	IntegrationID   string  `json:"integrationId,omitempty"`
	ConfigurationID string  `json:"configurationId,omitempty"`
}

type OptionsAllowlist struct {
	Paths []OptionsAllowlistPath `json:"paths"`
}

type OptionsAllowlistPath struct {
	Value string `json:"value"`
}
