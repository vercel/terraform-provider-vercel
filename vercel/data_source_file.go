package vercel

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceFileType struct{}

func (r dataSourceFileType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides information about a file on disk.

-> This is intended to be used with the ` + "`vercel_deployment` resource only.",
		Attributes: map[string]tfsdk.Attribute{
			"path": {
				Description: "The path to the file on your filesystem. Note that the path is relative to the root of the terraform files.",
				Required:    true,
				Type:        types.StringType,
			},
			"file": {
				Description: "A map of filename to metadata about the file. The metadata contains the file size and hash, and allows a deployment to be created if the file changes.",
				Computed:    true,
				Type: types.MapType{
					ElemType: types.StringType,
				},
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
		},
	}, nil
}

func (r dataSourceFileType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceFile{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceFile struct {
	p provider
}

type FileData struct {
	Path types.String      `tfsdk:"path"`
	ID   types.String      `tfsdk:"id"`
	File map[string]string `tfsdk:"file"`
}

func (r dataSourceFile) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var config FileData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := os.ReadFile(config.Path.Value)
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
	config.File = map[string]string{
		config.Path.Value: fmt.Sprintf("%d~%s", len(content), sha),
	}
	config.ID = config.Path

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
