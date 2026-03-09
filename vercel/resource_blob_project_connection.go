package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &blobProjectConnectionResource{}
	_ resource.ResourceWithConfigure   = &blobProjectConnectionResource{}
	_ resource.ResourceWithImportState = &blobProjectConnectionResource{}
)

func newBlobProjectConnectionResource() resource.Resource {
	return &blobProjectConnectionResource{}
}

type blobProjectConnectionResource struct {
	client *client.Client
}

func (r *blobProjectConnectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blob_project_connection"
}

func (r *blobProjectConnectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *blobProjectConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a project connection for a Vercel Blob store.

This resource links an existing Blob store to a project and manages the generated ` + "`BLOB_READ_WRITE_TOKEN`" + ` environment variable prefix and target environments.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the Blob store project connection.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"blob_store_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Blob store to connect.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project to connect to the Blob store.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team that owns the Blob store and project. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"env_var_prefix": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultBlobProjectConnectionEnvVarPrefix),
				Description: "The prefix used for the generated Blob environment variable names.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"environments": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(blobConnectionDefaultEnvironmentsValue()),
				Description: "The environments in which the generated Blob environment variables should be created.",
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf("development", "preview", "production")),
				},
			},
			"read_write_token_env_var_name": schema.StringAttribute{
				Computed:    true,
				Description: "The generated environment variable name that contains the Blob read/write token.",
			},
		},
	}
}

func (r *blobProjectConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BlobProjectConnectionModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var environments []string
	diags = plan.Environments.ElementsAs(ctx, &environments, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connection, err := r.client.CreateBlobStoreConnection(ctx, client.CreateBlobStoreConnectionRequest{
		BlobStoreID:  plan.BlobStoreID.ValueString(),
		Environments: environments,
		EnvVarPrefix: plan.EnvVarPrefix.ValueString(),
		ProjectID:    plan.ProjectID.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Blob store project connection",
			fmt.Sprintf("Could not connect Blob store %s to project %s, unexpected error: %s", plan.BlobStoreID.ValueString(), plan.ProjectID.ValueString(), err),
		)
		return
	}

	resolvedTeamID := r.client.TeamID(plan.TeamID.ValueString())
	result, diags := blobProjectConnectionModelFromResponse(ctx, plan.BlobStoreID.ValueString(), resolvedTeamID, connection)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created blob store project connection", map[string]any{
		"blob_store_id": result.BlobStoreID.ValueString(),
		"connection_id": result.ID.ValueString(),
		"project_id":    result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *blobProjectConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BlobProjectConnectionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connection, err := r.client.GetBlobStoreConnection(ctx, state.BlobStoreID.ValueString(), state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Blob store project connection",
			fmt.Sprintf("Could not read Blob store project connection %s %s, unexpected error: %s", state.BlobStoreID.ValueString(), state.ID.ValueString(), err),
		)
		return
	}

	resolvedTeamID := r.client.TeamID(state.TeamID.ValueString())
	result, diags := blobProjectConnectionModelFromResponse(ctx, state.BlobStoreID.ValueString(), resolvedTeamID, connection)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "read blob store project connection", map[string]any{
		"blob_store_id": result.BlobStoreID.ValueString(),
		"connection_id": result.ID.ValueString(),
		"project_id":    result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *blobProjectConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BlobProjectConnectionModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var environments []string
	diags = plan.Environments.ElementsAs(ctx, &environments, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connection, err := r.client.UpdateBlobStoreConnection(ctx, client.UpdateBlobStoreConnectionRequest{
		BlobStoreID:  plan.BlobStoreID.ValueString(),
		ConnectionID: plan.ID.ValueString(),
		Environments: environments,
		EnvVarPrefix: plan.EnvVarPrefix.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Blob store project connection",
			fmt.Sprintf("Could not update Blob store project connection %s %s, unexpected error: %s", plan.BlobStoreID.ValueString(), plan.ID.ValueString(), err),
		)
		return
	}

	resolvedTeamID := r.client.TeamID(plan.TeamID.ValueString())
	result, diags := blobProjectConnectionModelFromResponse(ctx, plan.BlobStoreID.ValueString(), resolvedTeamID, connection)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "updated blob store project connection", map[string]any{
		"blob_store_id": result.BlobStoreID.ValueString(),
		"connection_id": result.ID.ValueString(),
		"project_id":    result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *blobProjectConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BlobProjectConnectionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBlobStoreConnection(ctx, state.BlobStoreID.ValueString(), state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Blob store project connection",
			fmt.Sprintf("Could not delete Blob store project connection %s %s, unexpected error: %s", state.BlobStoreID.ValueString(), state.ID.ValueString(), err),
		)
	}
}

func (r *blobProjectConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, storeID, connectionID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid ID specified",
			fmt.Sprintf("Invalid ID '%s' specified. It should match the following format \"store_id/connection_id\" or \"team_id/store_id/connection_id\"", req.ID),
		)
		return
	}

	connection, err := r.client.GetBlobStoreConnection(ctx, storeID, connectionID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing Blob store project connection",
			fmt.Sprintf("Could not read Blob store project connection %s %s, unexpected error: %s", storeID, connectionID, err),
		)
		return
	}

	result, diags := blobProjectConnectionModelFromResponse(ctx, storeID, r.client.TeamID(teamID), connection)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
