package client

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// The API returns native stage rules, not the request's advancementType.
type rollingReleaseResponse struct {
	RollingRelease RollingRelease
}

type nativeRollingRelease struct {
	Target          string          `json:"target"`
	Enabled         *bool           `json:"enabled"`
	AdvancementType string          `json:"advancementType"`
	Gate            json.RawMessage `json:"gate"`
	Stages          []struct {
		TargetPercentage *int `json:"targetPercentage"`
		RequireApproval  bool `json:"requireApproval"`
		Duration         *int `json:"duration"`
		LinearShift      bool `json:"linearShift"`
	} `json:"stages"`
}

func (response *rollingReleaseResponse) UnmarshalJSON(data []byte) error {
	var envelope struct {
		RollingRelease json.RawMessage `json:"rollingRelease"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("decode rolling release response: %w", err)
	}
	if len(envelope.RollingRelease) == 0 {
		return fmt.Errorf("rolling release response is missing rollingRelease")
	}
	if bytes.Equal(bytes.TrimSpace(envelope.RollingRelease), []byte("null")) {
		response.RollingRelease = RollingRelease{}
		return nil
	}
	var config nativeRollingRelease
	if err := json.Unmarshal(envelope.RollingRelease, &config); err != nil {
		return fmt.Errorf("decode rolling release configuration: %w", err)
	}
	result, err := config.normalize()
	if err != nil {
		return err
	}
	response.RollingRelease = result
	return nil
}

func (config nativeRollingRelease) normalize() (RollingRelease, error) {
	if config.Target != "production" || len(config.Stages) < 2 || len(config.Stages) > 10 || (config.Enabled != nil && !*config.Enabled) {
		return RollingRelease{}, fmt.Errorf("rolling release requires enabled production and two to ten stages")
	}
	if len(config.Gate) != 0 && !bytes.Equal(bytes.TrimSpace(config.Gate), []byte("null")) {
		return RollingRelease{}, fmt.Errorf("rolling release gating cannot be represented by this resource")
	}
	result := RollingRelease{Enabled: true}
	previous := -1
	for index, stage := range config.Stages {
		if stage.TargetPercentage == nil || *stage.TargetPercentage <= previous || *stage.TargetPercentage > 100 || stage.LinearShift {
			return RollingRelease{}, fmt.Errorf("rolling release stage %d has invalid percentage or unsupported linear shift", index)
		}
		percentage := *stage.TargetPercentage
		if index == len(config.Stages)-1 {
			if percentage != 100 || stage.RequireApproval || stage.Duration != nil {
				return RollingRelease{}, fmt.Errorf("rolling release final stage must be 100 without advancement rules")
			}
		} else {
			mode := ""
			switch {
			case stage.RequireApproval && stage.Duration == nil:
				mode = "manual-approval"
			case !stage.RequireApproval && stage.Duration != nil && *stage.Duration > 0 && *stage.Duration <= 10000:
				mode = "automatic"
			default:
				return RollingRelease{}, fmt.Errorf("rolling release stage %d has ambiguous advancement rules", index)
			}
			if result.AdvancementType != "" && result.AdvancementType != mode {
				return RollingRelease{}, fmt.Errorf("rolling release has mixed advancement rules")
			}
			result.AdvancementType = mode
		}
		result.Stages = append(result.Stages, RollingReleaseStage{TargetPercentage: percentage, Duration: stage.Duration})
		previous = percentage
	}
	if config.AdvancementType != "" && config.AdvancementType != result.AdvancementType {
		return RollingRelease{}, fmt.Errorf("rolling release advancement metadata contradicts its stages")
	}
	return result, nil
}
