package file

import (
	"encoding/json"
	"fmt"
	"os"
)

// Builds defines some of the information that can be contained within a builds.json file
// as part of the Build API output.
type Builds struct {
	Target string    `json:"target"`
	Error  *struct{} `json:"error"`
	Builds []struct {
		Error *struct{} `json:"error"`
	} `json:"builds"`
}

// ReadBuildsJSON will read a builds.json file and return the parsed content as a Builds struct.
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
