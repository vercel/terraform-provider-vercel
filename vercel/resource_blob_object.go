package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource               = &blobObjectResource{}
	_ resource.ResourceWithConfigure  = &blobObjectResource{}
	_ resource.ResourceWithModifyPlan = &blobObjectResource{}
)

func newBlobObjectResource() resource.Resource {
	return &blobObjectResource{}
}

type blobObjectResource struct {
	client *client.Client
}

func (r *blobObjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blob_object"
}

func (r *blobObjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *blobObjectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan BlobObjectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Pathname.IsUnknown() || plan.Pathname.IsNull() {
		return
	}

	if err := validateManagedBlobObjectPathname(plan.Pathname.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("pathname"),
			"Invalid Blob object pathname",
			err.Error(),
		)
		return
	}

	if plan.Source.IsUnknown() || plan.Source.IsNull() {
		return
	}

	_, sourceSHA256, etag, err := readBlobObjectSource(plan.Source.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("source"),
			"Invalid Blob object source",
			fmt.Sprintf("Could not read blob object source file: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("source_sha256"), types.StringValue(sourceSHA256))...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("etag"), types.StringValue(etag))...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ContentType.IsNull() || plan.ContentType.IsUnknown() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("content_type"), types.StringValue(inferBlobObjectContentType(plan.Pathname.ValueString())))...)
	}
}

func (r *blobObjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Vercel Blob object.

This resource uploads a local file into a Blob store using a deterministic pathname so the object can be managed in place by Terraform.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The unique identifier for this Blob object. Format: `store_id/pathname`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"store_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Blob store that should contain the object.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team that owns the Blob store. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"pathname": schema.StringAttribute{
				Required:      true,
				Description:   "The pathname to upload within the Blob store.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"source": schema.StringAttribute{
				Required:    true,
				Description: "The local filesystem path to the file that should be uploaded.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"source_sha256": schema.StringAttribute{
				Computed:    true,
				Description: "The SHA-256 of the local source file content used for drift detection.",
			},
			"content_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The content type to store on the object. When omitted, Vercel infers it from the pathname.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"cache_control_max_age": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(defaultBlobObjectCacheControlMaxAge),
				Description: "The cache max-age, in seconds, to apply to the uploaded object.",
				Validators: []validator.Int64{
					int64validator.AtLeast(60),
				},
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The canonical URL for the Blob object.",
			},
			"download_url": schema.StringAttribute{
				Computed:    true,
				Description: "The Blob object URL with download semantics enabled.",
			},
			"size": schema.Int64Attribute{
				Computed:    true,
				Description: "The size of the Blob object in bytes.",
			},
			"uploaded_at": schema.StringAttribute{
				Computed:    true,
				Description: "The timestamp at which the Blob object was last uploaded.",
			},
			"content_disposition": schema.StringAttribute{
				Computed:    true,
				Description: "The content disposition returned for the Blob object.",
			},
			"cache_control": schema.StringAttribute{
				Computed:    true,
				Description: "The full Cache-Control header stored on the Blob object.",
			},
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "The current ETag for the Blob object.",
			},
		},
	}
}

func (r *blobObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BlobObjectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateManagedBlobObjectPathname(plan.Pathname.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("pathname"), "Invalid Blob object pathname", err.Error())
		return
	}

	content, _, _, err := readBlobObjectSource(plan.Source.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("source"),
			"Invalid Blob object source",
			fmt.Sprintf("Could not read blob object source file: %s", err),
		)
		return
	}

	object, err := r.client.PutBlobObject(ctx, client.PutBlobObjectRequest{
		Body:               content,
		CacheControlMaxAge: plan.CacheControlMaxAge.ValueInt64(),
		ContentType:        plan.ContentType.ValueString(),
		Pathname:           plan.Pathname.ValueString(),
		StoreID:            plan.StoreID.ValueString(),
		TeamID:             plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Blob object",
			fmt.Sprintf("Could not create Blob object %s, unexpected error: %s", plan.Pathname.ValueString(), err),
		)
		return
	}

	result := blobObjectResourceModelFromResponse(plan.Source, plan.SourceSHA256, plan.StoreID.ValueString(), r.client.TeamID(plan.TeamID.ValueString()), object)
	tflog.Info(ctx, "created blob object", map[string]any{
		"blob_object_id": result.ID.ValueString(),
		"store_id":       result.StoreID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *blobObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BlobObjectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	object, err := r.client.GetBlobObject(ctx, client.GetBlobObjectRequest{
		Pathname: state.Pathname.ValueString(),
		StoreID:  state.StoreID.ValueString(),
		TeamID:   state.TeamID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Blob object",
			fmt.Sprintf("Could not read Blob object %s, unexpected error: %s", state.ID.ValueString(), err),
		)
		return
	}

	result := blobObjectResourceModelFromResponse(state.Source, state.SourceSHA256, state.StoreID.ValueString(), r.client.TeamID(state.TeamID.ValueString()), object)
	tflog.Info(ctx, "read blob object", map[string]any{
		"blob_object_id": result.ID.ValueString(),
		"store_id":       result.StoreID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *blobObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BlobObjectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, _, _, err := readBlobObjectSource(plan.Source.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("source"),
			"Invalid Blob object source",
			fmt.Sprintf("Could not read blob object source file: %s", err),
		)
		return
	}

	object, err := r.client.PutBlobObject(ctx, client.PutBlobObjectRequest{
		Body:               content,
		CacheControlMaxAge: plan.CacheControlMaxAge.ValueInt64(),
		ContentType:        plan.ContentType.ValueString(),
		Pathname:           plan.Pathname.ValueString(),
		StoreID:            plan.StoreID.ValueString(),
		TeamID:             plan.TeamID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Blob object",
			fmt.Sprintf("Could not update Blob object %s, unexpected error: %s", plan.ID.ValueString(), err),
		)
		return
	}

	result := blobObjectResourceModelFromResponse(plan.Source, plan.SourceSHA256, plan.StoreID.ValueString(), r.client.TeamID(plan.TeamID.ValueString()), object)
	tflog.Info(ctx, "updated blob object", map[string]any{
		"blob_object_id": result.ID.ValueString(),
		"store_id":       result.StoreID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *blobObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BlobObjectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBlobObject(ctx, state.StoreID.ValueString(), state.Pathname.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Blob object",
			fmt.Sprintf("Could not delete Blob object %s, unexpected error: %s", state.ID.ValueString(), err),
		)
	}
}
