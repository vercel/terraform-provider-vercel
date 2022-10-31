package vercel

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
	"github.com/vercel/terraform-provider-vercel/file"
)

func newPrebuiltProjectDataSource() datasource.DataSource {
	return &prebuiltProjectDataSource{}
}

type prebuiltProjectDataSource struct {
	client *client.Client
}

func (d *prebuiltProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prebuilt_project"
}

func (d *prebuiltProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *prebuiltProjectDataSource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides the output of a project built via ` + "`vercel build`" + ` and provides metadata for use with a ` + "`vercel_deployment`" + `

The [build command](https://vercel.com/docs/cli#commands/build) can be used to build a project locally or in your own CI environment.
Build artifacts are placed into the ` + "`.vercel/output`" + ` directory according to the [Build Output API](https://vercel.com/docs/build-output-api/v3).

This allows a Vercel Deployment to be created without sharing the Project's source code with Vercel.
`,
		Attributes: map[string]tfsdk.Attribute{
			"path": {
				Description: "The path to the project. Note that this path is relative to the root of your terraform files. This should be the directory that contains the `.vercel/output` directory.",
				Required:    true,
				Type:        types.StringType,
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
			"output": {
				Description: "A map of output file to metadata about the file. The metadata contains the file size and hash, and allows a deployment to be created if the file changes.",
				Computed:    true,
				Type: types.MapType{
					ElemType: types.StringType,
				},
			},
		},
	}, nil
}

// PrebuiltProjectData represents the information terraform knows about a project directory data source
type PrebuiltProjectData struct {
	Path   types.String      `tfsdk:"path"`
	ID     types.String      `tfsdk:"id"`
	Output map[string]string `tfsdk:"output"`
}

func (d *prebuiltProjectDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var config PrebuiltProjectData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Path.IsUnknown() || config.Path.IsNull() {
		return
	}

	// if we know the path, let's do a quick check for prebuilt output valid-ness. i.e. reading the output directory
	// and ensuring no build errors.
	// We want to validate this both here and in the Read method in case the field is Unknown at plan time.
	validatePrebuiltOutput(&resp.Diagnostics, config.Path.ValueString())
}

// AddErrorer defines an interface that contains the AddError method. Most commonly used with Diagnostics.
type AddErrorer interface {
	AddError(summary string, detail string)
}

func validatePrebuiltOutput(diags AddErrorer, path string) {
	outputDir := filepath.Join(path, ".vercel", "output")
	_, err := os.Stat(outputDir)
	if os.IsNotExist(err) {
		diags.AddError(
			"Error reading prebuilt output",
			fmt.Sprintf(
				"A prebuilt project data source was used, but no prebuilt output was found in `%s`. Run `vercel build` to generate a local build",
				path,
			),
		)
		return
	}
	if err != nil {
		diags.AddError(
			"Error reading prebuilt project",
			fmt.Sprintf(
				"An unexpected error occurred when reading the prebuilt directory: %s",
				err,
			),
		)
		return
	}

	// The .vercel/output/builds.json file may exist, and can contain information about failed builds.
	// But it does not _have_ to exist, so we do not rely on its presence.
	builds, err := file.ReadBuildsJSON(filepath.Join(outputDir, "builds.json"))
	if os.IsNotExist(err) {
		// It's okay to not have a builds.json file. So allow this.
		return
	}
	if err != nil {
		diags.AddError(
			"Error reading prebuilt output",
			fmt.Sprintf(
				"An unexpected error occurred reading the prebuilt output builds.json: %s",
				err,
			),
		)
		return
	}

	// The file exists so check if there are any failed builds.
	containsError := builds.Error != nil
	for _, build := range builds.Builds {
		if build.Error != nil {
			containsError = true
		}
	}

	if containsError {
		diags.AddError(
			"Prebuilt deployment cannot be used",
			fmt.Sprintf(
				"The prebuilt deployment at `%s` cannot be used because `vercel build` failed with an error",
				path,
			),
		)
		return
	}
}

// Read will recursively read files from a .vercel/output directory. Metadata about all these files will then be made
// available to terraform.
func (d *prebuiltProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PrebuiltProjectData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	outputDir := filepath.Join(config.Path.ValueString(), ".vercel", "output")
	validatePrebuiltOutput(&resp.Diagnostics, config.Path.ValueString())
	if resp.Diagnostics.HasError() {
		return
	}

	config.Output = map[string]string{}
	err := filepath.WalkDir(
		outputDir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("could not read file %s: %w", path, err)
			}

			rawSha := sha1.Sum(content)
			sha := hex.EncodeToString(rawSha[:])

			config.Output[path] = fmt.Sprintf("%d~%s", len(content), sha)
			return nil
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading prebuilt output",
			fmt.Sprintf(
				"An unexpected error occurred reading files from the .vercel directory: %s",
				err,
			),
		)
		return
	}

	config.ID = config.Path
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
