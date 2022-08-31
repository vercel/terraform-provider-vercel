package file

import (
	"encoding/json"
	"fmt"
	"os"
)

type Builds struct {
	Target string    `json:"target"`
	Error  *struct{} `json:"error"`
	Builds []struct {
		Error *struct{} `json:"error"`
	} `json:"builds"`
}

func ReadBuildsJSON(path string) (builds Builds, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return builds, err
	}

	err = json.Unmarshal(content, &builds)
	if err != nil {
		return builds, fmt.Errorf("could not parse file %s: %w", path, err)
	}

	return builds, err
}
