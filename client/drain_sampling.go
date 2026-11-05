package client

// DrainSamplingRule selects the sampling rate for matching logs or traces.
type DrainSamplingRule struct {
	Rate        float64 `json:"rate"`
	Environment string  `json:"env,omitempty"`
	RequestPath string  `json:"requestPath,omitempty"`
}
