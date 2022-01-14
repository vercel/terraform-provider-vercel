package vercel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

type Deployment struct {
	TeamID     types.String      `tfsdk:"team_id"`
	ProjectID  types.String      `tfsdk:"project_id"`
	ID         types.String      `tfsdk:"id"`
	URL        types.String      `tfsdk:"url"`
	Production types.Bool        `tfsdk:"production"`
	Files      map[string]string `tfsdk:"files"`
}

func (d *Deployment) getFiles() ([]client.DeploymentFile, map[string]client.DeploymentFile, error) {
	var files []client.DeploymentFile
	filesBySha := map[string]client.DeploymentFile{}
	for filename, rawSizeAndSha := range d.Files {
		sizeSha := strings.Split(rawSizeAndSha, "~")
		if len(sizeSha) != 2 {
			return nil, nil, fmt.Errorf("expected file to have format `filename: size~sha`, but could not parse")
		}
		size, err := strconv.Atoi(sizeSha[0])
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse file size: %w", err)
		}
		sha := sizeSha[1]

		file := client.DeploymentFile{
			File: filename,
			Sha:  sha,
			Size: size,
		}
		files = append(files, file)
		filesBySha[sha] = file
	}
	return files, filesBySha, nil
}

func convertResponseToDeployment(response client.DeploymentResponse, tid types.String, files map[string]string) Deployment {
	production := types.Bool{Value: false}
	if response.Target != nil && *response.Target == "production" {
		production.Value = true
	}

	return Deployment{
		TeamID:     tid,
		ProjectID:  types.String{Value: response.ProjectID},
		ID:         types.String{Value: response.ID},
		URL:        types.String{Value: response.URL},
		Production: production,
		Files:      files,
	}
}
