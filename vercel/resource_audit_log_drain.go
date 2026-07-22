package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                     = &auditLogDrainResource{}
	_ resource.ResourceWithConfigure        = &auditLogDrainResource{}
	_ resource.ResourceWithConfigValidators = &auditLogDrainResource{}
	_ resource.ResourceWithImportState      = &auditLogDrainResource{}
)

func newAuditLogDrainResource() resource.Resource {
	return &auditLogDrainResource{}
}

type auditLogDrainResource struct {
	client *client.Client
}

func (r *auditLogDrainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_audit_log_drain"
}

func (r *auditLogDrainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	configuredClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = configuredClient
}

func (r *auditLogDrainResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("http"),
			path.MatchRoot("s3"),
		),
	}
}

func (r *auditLogDrainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Audit Log Drain resource.

Audit Log Drains forward team activity events to an HTTP endpoint or Amazon S3. They apply to the whole team and do not support project selection, filtering, or sampling.

~> Audit Log Drains are only available to Enterprise teams.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Audit Log Drain.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Description:   "The ID of the team the Audit Log Drain should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description:   "A human-readable name for the Audit Log Drain.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"http": schema.SingleNestedAttribute{
				Description:   "Configuration for delivery to a custom HTTP endpoint. Exactly one of `http` or `s3` must be configured.",
				Optional:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"endpoint": schema.StringAttribute{
						Description: "The HTTPS endpoint that receives Audit Log events.",
						Required:    true,
					},
					"encoding": schema.StringAttribute{
						Description: "The format used to deliver Audit Log events. Can be `json` or `ndjson`.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("json", "ndjson"),
						},
					},
					"compression": schema.StringAttribute{
						Description: "The compression applied to HTTP request bodies. Can be `gzip` or `none`.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("gzip", "none"),
						},
					},
					"headers": schema.MapAttribute{
						Description: "Custom headers included in requests to the HTTP endpoint.",
						ElementType: types.StringType,
						Optional:    true,
						Sensitive:   true,
					},
					"secret": schema.StringAttribute{
						Description: "A custom secret used to sign Audit Log events. If omitted, Vercel generates one.",
						Optional:    true,
						Computed:    true,
						Sensitive:   true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(32),
						},
					},
				},
			},
			"s3": schema.SingleNestedAttribute{
				Description:   "Configuration for delivery to an Amazon S3 bucket. Exactly one of `http` or `s3` must be configured.",
				Optional:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"endpoint": schema.StringAttribute{
						Description: "The S3 bucket and optional prefix where Audit Log events are written, using the format `s3://bucket[/prefix]`.",
						Required:    true,
					},
					"encoding": schema.StringAttribute{
						Description: "The format used to write Audit Log events. Can be `json` or `ndjson`.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("json", "ndjson"),
						},
					},
					"role_arn": schema.StringAttribute{
						Description: "The ARN of the AWS IAM role Vercel assumes to write objects.",
						Required:    true,
					},
					"region": schema.StringAttribute{
						Description: "The AWS region containing the S3 bucket.",
						Required:    true,
					},
					"server_side_encryption": schema.StringAttribute{
						Description: "The server-side encryption header sent when writing objects. Can be `AES256`, `aws:kms`, or `aws:kms:dsse`.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("AES256", "aws:kms", "aws:kms:dsse"),
						},
					},
					"object_acl": schema.StringAttribute{
						Description: "The canned ACL sent when writing objects. Can be `private`, `bucket-owner-read`, or `bucket-owner-full-control`.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("private", "bucket-owner-read", "bucket-owner-full-control"),
						},
					},
				},
			},
		},
	}
}

type AuditLogDrain struct {
	ID     types.String             `tfsdk:"id"`
	TeamID types.String             `tfsdk:"team_id"`
	Name   types.String             `tfsdk:"name"`
	HTTP   *AuditLogDrainHTTPConfig `tfsdk:"http"`
	S3     *AuditLogDrainS3Config   `tfsdk:"s3"`
}

type AuditLogDrainHTTPConfig struct {
	Endpoint    types.String `tfsdk:"endpoint"`
	Encoding    types.String `tfsdk:"encoding"`
	Compression types.String `tfsdk:"compression"`
	Headers     types.Map    `tfsdk:"headers"`
	Secret      types.String `tfsdk:"secret"`
}

type AuditLogDrainS3Config struct {
	Endpoint             types.String `tfsdk:"endpoint"`
	Encoding             types.String `tfsdk:"encoding"`
	RoleARN              types.String `tfsdk:"role_arn"`
	Region               types.String `tfsdk:"region"`
	ServerSideEncryption types.String `tfsdk:"server_side_encryption"`
	ObjectACL            types.String `tfsdk:"object_acl"`
}

func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueString()
	return &result
}

func stringFromAPI(value *string, prior types.String) types.String {
	if value != nil {
		return types.StringValue(*value)
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	return types.StringNull()
}

func auditLogDrainFromAPI(ctx context.Context, out client.AuditLogDrain, prior *AuditLogDrain) (AuditLogDrain, error) {
	result := AuditLogDrain{
		ID:     types.StringValue(out.ID),
		TeamID: toTeamID(out.TeamID),
		Name:   types.StringValue(out.Name),
	}

	if out.HTTP != nil {
		var priorHTTP AuditLogDrainHTTPConfig
		if prior != nil && prior.HTTP != nil {
			priorHTTP = *prior.HTTP
		} else {
			priorHTTP.Headers = types.MapNull(types.StringType)
			priorHTTP.Secret = types.StringNull()
			priorHTTP.Compression = types.StringNull()
		}

		headers := priorHTTP.Headers
		if out.HTTP.Headers != nil {
			headersFromAPI, diags := types.MapValueFrom(ctx, types.StringType, out.HTTP.Headers)
			if diags.HasError() {
				return AuditLogDrain{}, fmt.Errorf("converting Audit Log Drain HTTP headers: %s", diags.Errors()[0].Detail())
			}
			headers = headersFromAPI
		}
		if len(out.HTTP.Headers) == 0 && (prior == nil || prior.HTTP == nil || prior.HTTP.Headers.IsNull() || prior.HTTP.Headers.IsUnknown()) {
			headers = types.MapNull(types.StringType)
		}

		secret := priorHTTP.Secret
		if secret.IsNull() || secret.IsUnknown() {
			secret = stringFromAPI(out.HTTP.Secret, secret)
		}

		result.HTTP = &AuditLogDrainHTTPConfig{
			Endpoint:    types.StringValue(out.HTTP.Endpoint),
			Encoding:    types.StringValue(out.HTTP.Encoding),
			Compression: stringFromAPI(out.HTTP.Compression, priorHTTP.Compression),
			Headers:     headers,
			Secret:      secret,
		}
	}

	if out.S3 != nil {
		var priorS3 AuditLogDrainS3Config
		if prior != nil && prior.S3 != nil {
			priorS3 = *prior.S3
		} else {
			priorS3.ServerSideEncryption = types.StringNull()
			priorS3.ObjectACL = types.StringNull()
		}

		result.S3 = &AuditLogDrainS3Config{
			Endpoint:             types.StringValue(out.S3.Endpoint),
			Encoding:             types.StringValue(out.S3.Encoding),
			RoleARN:              types.StringValue(out.S3.RoleARN),
			Region:               types.StringValue(out.S3.Region),
			ServerSideEncryption: stringFromAPI(out.S3.ServerSideEncryption, priorS3.ServerSideEncryption),
			ObjectACL:            stringFromAPI(out.S3.ObjectACL, priorS3.ObjectACL),
		}
	}

	return result, nil
}

func (r *auditLogDrainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuditLogDrain
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := client.CreateAuditLogDrainRequest{
		TeamID: plan.TeamID.ValueString(),
		Name:   plan.Name.ValueString(),
	}
	if plan.HTTP != nil {
		var headers map[string]string
		if !plan.HTTP.Headers.IsNull() && !plan.HTTP.Headers.IsUnknown() {
			resp.Diagnostics.Append(plan.HTTP.Headers.ElementsAs(ctx, &headers, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		request.HTTP = &client.AuditLogDrainHTTPDelivery{
			Endpoint:    plan.HTTP.Endpoint.ValueString(),
			Encoding:    plan.HTTP.Encoding.ValueString(),
			Compression: optionalString(plan.HTTP.Compression),
			Headers:     headers,
			Secret:      optionalString(plan.HTTP.Secret),
		}
	}
	if plan.S3 != nil {
		request.S3 = &client.AuditLogDrainS3Delivery{
			Endpoint:             plan.S3.Endpoint.ValueString(),
			Encoding:             plan.S3.Encoding.ValueString(),
			RoleARN:              plan.S3.RoleARN.ValueString(),
			Region:               plan.S3.Region.ValueString(),
			ServerSideEncryption: optionalString(plan.S3.ServerSideEncryption),
			ObjectACL:            optionalString(plan.S3.ObjectACL),
		}
	}

	out, err := r.client.CreateAuditLogDrain(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Audit Log Drain", "Could not create Audit Log Drain, unexpected error: "+err.Error())
		return
	}

	result, err := auditLogDrainFromAPI(ctx, out, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error converting Audit Log Drain response", err.Error())
		return
	}

	tflog.Info(ctx, "created Audit Log Drain", map[string]any{
		"team_id":            result.TeamID.ValueString(),
		"audit_log_drain_id": result.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *auditLogDrainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuditLogDrain
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAuditLogDrain(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Audit Log Drain",
			fmt.Sprintf("Could not get Audit Log Drain %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ID.ValueString(), err),
		)
		return
	}

	result, err := auditLogDrainFromAPI(ctx, out, &state)
	if err != nil {
		resp.Diagnostics.AddError("Error converting Audit Log Drain response", err.Error())
		return
	}

	tflog.Info(ctx, "read Audit Log Drain", map[string]any{
		"team_id":            result.TeamID.ValueString(),
		"audit_log_drain_id": result.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *auditLogDrainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Updating an Audit Log Drain is not supported", "Updating an Audit Log Drain is not supported")
}

func (r *auditLogDrainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuditLogDrain
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAuditLogDrain(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Audit Log Drain",
			fmt.Sprintf("Could not delete Audit Log Drain %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "deleted Audit Log Drain", map[string]any{
		"team_id":            state.TeamID.ValueString(),
		"audit_log_drain_id": state.ID.ValueString(),
	})
}

func (r *auditLogDrainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Audit Log Drain",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/audit_log_drain_id\" or \"audit_log_drain_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetAuditLogDrain(ctx, id, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Audit Log Drain",
			fmt.Sprintf("Could not get Audit Log Drain %s %s, unexpected error: %s", teamID, id, err),
		)
		return
	}

	result, err := auditLogDrainFromAPI(ctx, out, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error converting Audit Log Drain response", err.Error())
		return
	}

	tflog.Info(ctx, "import Audit Log Drain", map[string]any{
		"team_id":            result.TeamID.ValueString(),
		"audit_log_drain_id": result.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
