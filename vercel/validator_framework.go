package vercel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func validateFramework() validatorStringOneOf {
	resp, err := http.Get("https://api-frameworks.zeit.sh/")
	if err != nil {
		panic(fmt.Errorf("unable to retrieve Vercel frameworks: unexpected error: %w", err))
	}
	if resp.StatusCode != 200 {
		panic(fmt.Errorf("unable to retrieve Vercel frameworks: unexpected status code %d", resp.StatusCode))
	}

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("error reading api-frameworks.zeit.sh response body: %w", err))
	}
	var fwList []struct {
		Slug string `json:"slug"`
	}
	err = json.Unmarshal(responseBody, &fwList)
	if resp.StatusCode != 200 {
		panic(fmt.Errorf("unable to parse Vercel frameworks response: %w", err))
	}
	var frameworks []string
	for _, fw := range fwList {
		frameworks = append(frameworks, fw.Slug)
	}

	return stringOneOf(frameworks...)
}
