package vercel

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

const (
	defaultBlobProjectConnectionEnvVarPrefix = "BLOB"
	defaultBlobStoreAccess                   = "public"
	defaultBlobStoreRegion                   = "iad1"
)

var blobRegions = []string{
	"arn1",
	"bom1",
	"cdg1",
	"cle1",
	"cpt1",
	"dub1",
	"dxb1",
	"fra1",
	"gru1",
	"hkg1",
	"hnd1",
	"iad1",
	"icn1",
	"kix1",
	"lhr1",
	"pdx1",
	"sfo1",
	"sin1",
	"syd1",
	"yul1",
}

type BlobStoreModel struct {
	Access    types.String `tfsdk:"access"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	FileCount types.Int64  `tfsdk:"file_count"`
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Region    types.String `tfsdk:"region"`
	Size      types.Int64  `tfsdk:"size"`
	Status    types.String `tfsdk:"status"`
	TeamID    types.String `tfsdk:"team_id"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

type BlobStoreListItem struct {
	Access    types.String `tfsdk:"access"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	FileCount types.Int64  `tfsdk:"file_count"`
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Region    types.String `tfsdk:"region"`
	Size      types.Int64  `tfsdk:"size"`
	Status    types.String `tfsdk:"status"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

type BlobProjectConnectionModel struct {
	BlobStoreID              types.String `tfsdk:"blob_store_id"`
	Environments             types.Set    `tfsdk:"environments"`
	EnvVarPrefix             types.String `tfsdk:"env_var_prefix"`
	ID                       types.String `tfsdk:"id"`
	ProjectID                types.String `tfsdk:"project_id"`
	ReadWriteTokenEnvVarName types.String `tfsdk:"read_write_token_env_var_name"`
	TeamID                   types.String `tfsdk:"team_id"`
}

type BlobProjectConnectionListItem struct {
	Environments             types.Set    `tfsdk:"environments"`
	EnvVarPrefix             types.String `tfsdk:"env_var_prefix"`
	ID                       types.String `tfsdk:"id"`
	ProductionDeploymentID   types.String `tfsdk:"production_deployment_id"`
	ProductionDeploymentURL  types.String `tfsdk:"production_deployment_url"`
	ProjectFramework         types.String `tfsdk:"project_framework"`
	ProjectID                types.String `tfsdk:"project_id"`
	ProjectName              types.String `tfsdk:"project_name"`
	ReadWriteTokenEnvVarName types.String `tfsdk:"read_write_token_env_var_name"`
}

func blobStoreModelFromResponse(store client.BlobStore) BlobStoreModel {
	return BlobStoreModel{
		Access:    types.StringValue(store.Access),
		CreatedAt: types.Int64Value(store.CreatedAt),
		FileCount: types.Int64Value(store.Count),
		ID:        types.StringValue(store.ID),
		Name:      types.StringValue(store.Name),
		Region:    types.StringValue(store.Region),
		Size:      types.Int64Value(store.Size),
		Status:    types.StringValue(store.Status),
		TeamID:    toTeamID(store.TeamID),
		UpdatedAt: types.Int64Value(store.UpdatedAt),
	}
}

func blobStoreListItemFromResponse(store client.BlobStore) BlobStoreListItem {
	return BlobStoreListItem{
		Access:    types.StringValue(store.Access),
		CreatedAt: types.Int64Value(store.CreatedAt),
		FileCount: types.Int64Value(store.Count),
		ID:        types.StringValue(store.ID),
		Name:      types.StringValue(store.Name),
		Region:    types.StringValue(store.Region),
		Size:      types.Int64Value(store.Size),
		Status:    types.StringValue(store.Status),
		UpdatedAt: types.Int64Value(store.UpdatedAt),
	}
}

func blobProjectConnectionModelFromResponse(ctx context.Context, storeID, teamID string, connection client.BlobProjectConnection) (BlobProjectConnectionModel, diag.Diagnostics) {
	environments, diags := types.SetValueFrom(ctx, types.StringType, sortedStrings(connection.EnvVarEnvironments))
	if diags.HasError() {
		return BlobProjectConnectionModel{}, diags
	}

	prefix := resolvedBlobProjectConnectionEnvVarPrefix(connection.EnvVarPrefix)

	return BlobProjectConnectionModel{
		BlobStoreID:              types.StringValue(storeID),
		Environments:             environments,
		EnvVarPrefix:             types.StringValue(prefix),
		ID:                       types.StringValue(connection.ID),
		ProjectID:                types.StringValue(connection.ProjectID),
		ReadWriteTokenEnvVarName: types.StringValue(blobReadWriteTokenEnvVarName(prefix)),
		TeamID:                   toTeamID(teamID),
	}, nil
}

func blobProjectConnectionListItemFromResponse(ctx context.Context, connection client.BlobProjectConnection) (BlobProjectConnectionListItem, diag.Diagnostics) {
	environments, diags := types.SetValueFrom(ctx, types.StringType, sortedStrings(connection.EnvVarEnvironments))
	if diags.HasError() {
		return BlobProjectConnectionListItem{}, diags
	}

	prefix := resolvedBlobProjectConnectionEnvVarPrefix(connection.EnvVarPrefix)

	return BlobProjectConnectionListItem{
		Environments:             environments,
		EnvVarPrefix:             types.StringValue(prefix),
		ID:                       types.StringValue(connection.ID),
		ProductionDeploymentID:   stringPointerValue(connectionProductionDeploymentID(connection.ProductionDeployment)),
		ProductionDeploymentURL:  stringPointerValue(connectionProductionDeploymentURL(connection.ProductionDeployment)),
		ProjectFramework:         stringPointerValue(connection.Project.Framework),
		ProjectID:                types.StringValue(connection.ProjectID),
		ProjectName:              types.StringValue(connection.Project.Name),
		ReadWriteTokenEnvVarName: types.StringValue(blobReadWriteTokenEnvVarName(prefix)),
	}, nil
}

func blobConnectionDefaultEnvironmentsValue() types.Set {
	return types.SetValueMust(types.StringType, []attr.Value{
		types.StringValue("preview"),
		types.StringValue("production"),
	})
}

func resolvedBlobProjectConnectionEnvVarPrefix(prefix *string) string {
	if prefix == nil || *prefix == "" {
		return defaultBlobProjectConnectionEnvVarPrefix
	}

	return *prefix
}

func blobReadWriteTokenEnvVarName(prefix string) string {
	return fmt.Sprintf("%s_READ_WRITE_TOKEN", prefix)
}

func connectionProductionDeploymentID(deployment *client.BlobProjectConnectionDeployment) *string {
	if deployment == nil || deployment.ID == "" {
		return nil
	}

	return &deployment.ID
}

func connectionProductionDeploymentURL(deployment *client.BlobProjectConnectionDeployment) *string {
	if deployment == nil {
		return nil
	}

	return deployment.URL
}

func stringPointerValue(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}

	return types.StringValue(*value)
}

func sortedStrings(values []string) []string {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	return sorted
}

func sortBlobStores(stores []client.BlobStore) {
	slices.SortFunc(stores, func(left, right client.BlobStore) int {
		if left.Name == right.Name {
			return compareStrings(left.ID, right.ID)
		}

		return compareStrings(left.Name, right.Name)
	})
}

func sortBlobConnections(connections []client.BlobProjectConnection) {
	slices.SortFunc(connections, func(left, right client.BlobProjectConnection) int {
		if left.ProjectID == right.ProjectID {
			return compareStrings(left.ID, right.ID)
		}

		return compareStrings(left.ProjectID, right.ProjectID)
	})
}

func compareStrings(left, right string) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
