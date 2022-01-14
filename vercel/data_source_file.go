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
		Attributes: map[string]tfsdk.Attribute{
			"path": {
				Required: true,
				Type:     types.StringType,
			},
			"file": {
				Computed: true,
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
