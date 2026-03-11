package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &blobStoreResource{}
	_ resource.ResourceWithConfigure   = &blobStoreResource{}
	_ resource.ResourceWithImportState = &blobStoreResource{}
)

func newBlobStoreResource() resource.Resource {
	return &blobStoreResource{}
}

type blobStoreResource struct {
	client *client.Client
}

func (r *blobStoreResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blob_store"
}

func (r *blobStoreResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *blobStoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Vercel Blob store.

Blob stores are team-scoped storage resources that back Vercel Blob uploads and can be connected to one or more projects.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the Blob store.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Blob store.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access": schema.StringAttribute{
				Description:   "Whether blobs should be created with `public` or `private` access.",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString(defaultBlobStoreAccess),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("public", "private"),
				},
			},
			"region": schema.StringAttribute{
				Description:   "The region in which the Blob store should be created.",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString(defaultBlobStoreRegion),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf(blobRegions...),
				},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Blob store should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The current status of the Blob store.",
			},
			"size": schema.Int64Attribute{
				Computed:      true,
				Description:   "The size of the Blob store in bytes.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
			},
			"file_count": schema.Int64Attribute{
				Computed:      true,
				Description:   "The number of files currently stored in the Blob store.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
			},
			"created_at": schema.Int64Attribute{
				Computed:      true,
				Description:   "The Unix timestamp, in milliseconds, when the Blob store was created.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
			},
			"updated_at": schema.Int64Attribute{
				Computed:      true,
				Description:   "The Unix timestamp, in milliseconds, when the Blob store was last updated.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
			},
		},
	}
}

func (r *blobStoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BlobStoreModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	store, err := r.client.CreateBlobStore(ctx, client.CreateBlobStoreRequest{
		Access: plan.Access.ValueString(),
		Name:   plan.Name.ValueString(),
		Region: plan.Region.ValueString(),
		TeamID: plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Blob store",
			fmt.Sprintf("Could not create Blob store %s, unexpected error: %s", plan.Name.ValueString(), err),
		)
		return
	}

	result := blobStoreModelFromResponse(store)
	tflog.Info(ctx, "created blob store", map[string]any{
		"blob_store_id": result.ID.ValueString(),
		"team_id":       result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *blobStoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BlobStoreModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	store, err := r.client.GetBlobStore(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Blob store",
			fmt.Sprintf("Could not read Blob store %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ID.ValueString(), err),
		)
		return
	}

	result := blobStoreModelFromResponse(store)
	tflog.Info(ctx, "read blob store", map[string]any{
		"blob_store_id": result.ID.ValueString(),
		"team_id":       result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *blobStoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BlobStoreModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	store, err := r.client.UpdateBlobStore(ctx, client.UpdateBlobStoreRequest{
		Name:    plan.Name.ValueString(),
		StoreID: plan.ID.ValueString(),
		TeamID:  plan.TeamID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Blob store",
			fmt.Sprintf("Could not update Blob store %s %s, unexpected error: %s", plan.TeamID.ValueString(), plan.ID.ValueString(), err),
		)
		return
	}

	result := blobStoreModelFromResponse(store)
	tflog.Info(ctx, "updated blob store", map[string]any{
		"blob_store_id": result.ID.ValueString(),
		"team_id":       result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *blobStoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BlobStoreModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBlobStore(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Blob store",
			fmt.Sprintf("Could not delete Blob store %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ID.ValueString(), err),
		)
	}
}

func (r *blobStoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, storeID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid ID specified",
			fmt.Sprintf("Invalid ID '%s' specified. It should match the following format \"store_id\" or \"team_id/store_id\"", req.ID),
		)
		return
	}

	store, err := r.client.GetBlobStore(ctx, storeID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing Blob store",
			fmt.Sprintf("Could not read Blob store %s %s, unexpected error: %s", teamID, storeID, err),
		)
		return
	}

	diags := resp.State.Set(ctx, blobStoreModelFromResponse(store))
	resp.Diagnostics.Append(diags...)
}
