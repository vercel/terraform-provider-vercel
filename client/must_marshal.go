package client

import (
	"encoding/json"
)

func (v *VercelAuthentication) MarshalJSON() ([]byte, error) {
	if v.DeploymentType == "none" {
		return []byte(`null`), nil
	}

	return json.Marshal(&struct {
		DeploymentType string `json:"deploymentType"`
	}{
		DeploymentType: v.DeploymentType,
	})
}

// mustMarshal is a helper to remove unnecessary error checking when marshaling a Go
// struct to json. There are only a few instances where marshaling can fail, and they
// are around the shape of the data. e.g. if a struct contains a channel, then it cannot
// be marshaled. As our structs are known ahead of time and are all safe to marshal,
// this simplifies the error checking process.
func mustMarshal(v interface{}) []byte {
	res, err := json.Marshal(v)
	if err != nil {
		//lintignore:R009 // this is okay as we know the shape of the data
		panic(err)
	}
	return res
}
