package vercel

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
	"github.com/vercel/terraform-provider-vercel/file"
)

func newProjectDirectoryDataSource() datasource.DataSource {
	return &projectDirectoryDataSource{}
}

type projectDirectoryDataSource struct {
	client *client.Client
}

func (d *projectDirectoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_directory"
}

func (d *projectDirectoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// GetSchema returns the schema information for a project directory data source
func (d projectDirectoryDataSource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides information about files within a directory on disk.

This will recursively read files, providing metadata for use with a ` + "`vercel_deployment`." + `

-> If you want to prevent files from being included, this can be done with a [vercelignore file](https://vercel.com/guides/prevent-uploading-sourcepaths-with-vercelignore).
        `,
		Attributes: map[string]tfsdk.Attribute{
			"path": {
				Description: "The path to the directory on your filesystem. Note that the path is relative to the root of the terraform files.",
				Required:    true,
				Type:        types.StringType,
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"files": {
				Description: "A map of filename to metadata about the file. The metadata contains the file size and hash, and allows a deployment to be created if the file changes.",
				Computed:    true,
				Type: types.MapType{
					ElemType: types.StringType,
				},
			},
		},
	}, nil
}

// ProjectDirectoryData represents the information terraform knows about a project directory data source
type ProjectDirectoryData struct {
	Path  types.String      `tfsdk:"path"`
	ID    types.String      `tfsdk:"id"`
	Files map[string]string `tfsdk:"files"`
}

// Read will recursively scan a directory looking for any files that do not match a .vercelignore file (if a
// .vercelignore is present). Metadata about all these files will then be made available to terraform.
// It is called by the provider whenever data source values should be read to update state.
func (d *projectDirectoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectDirectoryData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ignoreRules, err := file.GetIgnores(config.Path.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading .vercelignore file",
			fmt.Sprintf("Could not read file, unexpected error: %s",
				err,
			),
		)
		return
	}

	paths, err := file.GetPaths(config.Path.Value, ignoreRules)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading directory",
			fmt.Sprintf("Could not read files for directory %s, unexpected error: %s",
				config.Path.Value,
				err,
			),
		)
		return
	}

	config.Files = map[string]string{}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading file",
				fmt.Sprintf("Could not read file %s, unexpected error: %s",
					config.Path.Value,
					err,
				),
			)
			return
		}
		rawSha := sha1.Sum(content)
		sha := hex.EncodeToString(rawSha[:])

		config.Files[path] = fmt.Sprintf("%d~%s", len(content), sha)
	}

	config.ID = config.Path
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
