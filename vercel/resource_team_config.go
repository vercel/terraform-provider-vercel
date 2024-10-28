package vercel

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &teamConfigResource{}
	_ resource.ResourceWithConfigure = &teamConfigResource{}
)

func newTeamConfigResource() resource.Resource {
	return &teamConfigResource{}
}

type teamConfigResource struct {
	client *client.Client
}

func (r *teamConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_config"
}

func (r *teamConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *teamConfigResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the configuration of an existing Vercel Team.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the existing Vercel Team.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description:   "The name of the team.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"slug": schema.StringAttribute{
				Description:   "The slug of the team. Will be used in the URL of the team's dashboard.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"avatar": schema.MapAttribute{
				Description:   "The `avatar` should be a the 'file' attribute from a vercel_file data source.",
				Optional:      true,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
				ElementType:   types.StringType,
				Validators: []validator.Map{
					mapItemsMinCount(1),
					mapItemsMaxCount(1),
				},
			},
			"description": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "A description of the team.",
			},
			"sensitive_environment_variable_policy": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringOneOf("on", "off"),
				},
			},
			"email_domain": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Hostname that'll be matched with emails on sign-up to automatically join the Team.",
			},
			/*
				"saml": schema.SingleNestedAttribute{
					Attributes: map[string]schema.Attribute{
						"enforced": schema.BoolAttribute{
							Description: "Indicates if SAML is enforced for the team.",
							Required:    true,
						},
						"roles": schema.MapAttribute{
							Description: "Directory groups to role or access group mappings.",
							Optional:    true,
						},
						"access_group_id": schema.StringAttribute{
							// TODO - enforce either accessGroupId or roles.
							Description: "The ID of the access group to use for the team.",
							Optional:    true,
							Validators: []validator.String{
								stringRegex(regexp.MustCompile("^ag_[A-z0-9_ -]+$"), "Access group ID must be a valid access group"),
							},
						},
						"connection": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"status": schema.StringAttribute{
									Computed:    true,
									Description: "The current status of the connection.",
								},
							},
							Description: "Info about the SAML connection.",
							Computed:    true,
						},
						"directory": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Computed:    true,
									Description: "The identity provider type.",
								},
								"state": schema.StringAttribute{
									Computed:    true,
									Description: "The current state of the SAML connection.",
								},
							},
							Description: "Info about the SAML directory.",
							Computed:    true,
						},
					},
					Optional:    true,
					Description: "Configuration for SAML authentication.",
				},
			*/
			"invite_code": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "A code that can be used to join this team. Only visible to Team owners.",
			},
			"preview_deployment_suffix": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Computed:      true,
				Description:   "The hostname that is used as the preview deployment suffix.",
			},
			"remote_caching": schema.SingleNestedAttribute{
				Description: "Configuration for Remote Caching.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description:   "Indicates if Remote Caching is enabled.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
				},
			},
			"enable_preview_feedback": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringOneOf("default", "on", "off"),
				},
			},
			"enable_production_feedback": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringOneOf("default", "on", "off"),
				},
			},
			"hide_ip_addresses": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Indicates if ip addresses should be accessible in o11y tooling.",
			},
			"hide_ip_addresses_in_log_drains": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Indicates if ip addresses should be accessible in log drains.",
			},
		},
	}
}

/*
type SamlConnection struct {
	Status types.String `tfsdk:"status"`
}

type SamlDirectory struct {
	Type  types.String `tfsdk:"type"`
	State types.String `tfsdk:"state"`
}

type Saml struct {
	Enforced      types.Bool      `tfsdk:"enforced"`
	Roles         types.Map       `tfsdk:"roles"`
	AccessGroupId types.String    `tfsdk:"access_group_id"`
	Connection    *SamlConnection `tfsdk:"connection"`
	Directory     *SamlDirectory  `tfsdk:"directory"`
}

func (s Saml) toUpdateSamlConfig() *client.UpdateSamlConfig {
}
*/

type EnableConfig struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type TeamConfig struct {
	ID                                 types.String `tfsdk:"id"`
	Avatar                             types.Map    `tfsdk:"avatar"`
	Name                               types.String `tfsdk:"name"`
	Slug                               types.String `tfsdk:"slug"`
	Description                        types.String `tfsdk:"description"`
	InviteCode                         types.String `tfsdk:"invite_code"`
	SensitiveEnvironmentVariablePolicy types.String `tfsdk:"sensitive_environment_variable_policy"`
	EmailDomain                        types.String `tfsdk:"email_domain"`
	PreviewDeploymentSuffix            types.String `tfsdk:"preview_deployment_suffix"`
	RemoteCaching                      types.Object `tfsdk:"remote_caching"`
	EnablePreviewFeedback              types.String `tfsdk:"enable_preview_feedback"`
	EnableProductionFeedback           types.String `tfsdk:"enable_production_feedback"`
	HideIPAddresses                    types.Bool   `tfsdk:"hide_ip_addresses"`
	HideIPAddressesInLogDrains         types.Bool   `tfsdk:"hide_ip_addresses_in_log_drains"`
	// Saml                               types.Object `tfsdk:"saml"`
}

type RemoteCaching struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

var remoteCachingAttrTypes = map[string]attr.Type{
	"enabled": types.BoolType,
}

func (r *RemoteCaching) toUpdateTeamRequest() *client.RemoteCaching {
	if r == nil {
		return nil
	}
	return &client.RemoteCaching{
		Enabled: r.Enabled.ValueBoolPointer(),
	}
}

func (t *TeamConfig) toUpdateTeamRequest(ctx context.Context, avatar string, stateSlug types.String) (client.UpdateTeamRequest, diag.Diagnostics) {
	slug := t.Slug.ValueString()
	if stateSlug.ValueString() == t.Slug.ValueString() {
		// Prevent updating slug if it hasn't changed, as this has an aggressive rate limit.
		slug = ""
	}
	var rc *RemoteCaching
	diags := t.RemoteCaching.As(ctx, &rc, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	if diags.HasError() {
		return client.UpdateTeamRequest{}, diags
	}

	return client.UpdateTeamRequest{
		TeamID:                             t.ID.ValueString(),
		Avatar:                             avatar,
		EmailDomain:                        t.EmailDomain.ValueString(),
		Name:                               t.Name.ValueString(),
		Description:                        t.Description.ValueString(),
		PreviewDeploymentSuffix:            t.PreviewDeploymentSuffix.ValueString(),
		Slug:                               slug,
		EnablePreviewFeedback:              t.EnablePreviewFeedback.ValueString(),
		EnableProductionFeedback:           t.EnableProductionFeedback.ValueString(),
		SensitiveEnvironmentVariablePolicy: t.SensitiveEnvironmentVariablePolicy.ValueString(),
		RemoteCaching:                      rc.toUpdateTeamRequest(),
		HideIPAddresses:                    t.HideIPAddresses.ValueBoolPointer(),
		HideIPAddressesInLogDrains:         t.HideIPAddressesInLogDrains.ValueBoolPointer(),
		// Saml:                               t.Saml.toUpdateSamlConfig(),
	}, nil
}

func convertResponseToTeamConfig(ctx context.Context, response client.Team, avatar types.Map) (TeamConfig, diag.Diagnostics) {
	remoteCaching := types.ObjectNull(remoteCachingAttrTypes)
	if response.RemoteCaching != nil {
		var diags diag.Diagnostics
		remoteCaching, diags = types.ObjectValueFrom(ctx, remoteCachingAttrTypes, &RemoteCaching{
			Enabled: types.BoolPointerValue(response.RemoteCaching.Enabled),
		})
		if diags.HasError() {
			return TeamConfig{}, diags
		}
	}
	return TeamConfig{
		Avatar:                             avatar,
		ID:                                 types.StringValue(response.ID),
		Name:                               types.StringValue(response.Name),
		Slug:                               types.StringValue(response.Slug),
		Description:                        types.StringPointerValue(response.Description),
		InviteCode:                         types.StringPointerValue(response.InviteCode),
		SensitiveEnvironmentVariablePolicy: types.StringPointerValue(response.SensitiveEnvironmentVariablePolicy),
		EmailDomain:                        types.StringPointerValue(response.EmailDomain),
		PreviewDeploymentSuffix:            types.StringPointerValue(response.PreviewDeploymentSuffix),
		EnablePreviewFeedback:              types.StringPointerValue(response.EnablePreviewFeedback),
		EnableProductionFeedback:           types.StringPointerValue(response.EnableProductionFeedback),
		HideIPAddresses:                    types.BoolPointerValue(response.HideIPAddresses),
		HideIPAddressesInLogDrains:         types.BoolPointerValue(response.HideIPAddressesInLogDrains),
		RemoteCaching:                      remoteCaching,
		// Saml:                               types.StringValue(response.Saml),
	}, nil
}

func (r *teamConfigResource) uploadAvatarIfPresent(ctx context.Context, plan TeamConfig) (avatar string, diags diag.Diagnostics) {
	if !plan.Avatar.IsNull() && !plan.Avatar.IsUnknown() {
		var unparsedFiles map[string]string
		diags = plan.Avatar.ElementsAs(ctx, &unparsedFiles, false)
		if diags.HasError() {
			return avatar, diags
		}
		for filename, rawSizeAndSha := range unparsedFiles {
			sizeSha := strings.Split(rawSizeAndSha, "~")
			if len(sizeSha) != 2 {
				diags.AddError(
					"Error creating team config",
					"Could not parse avatar, unexpected error: expected avatar to have format filename: size~sha, but could not parse",
				)
				return avatar, diags
			}

			sha := sizeSha[1]
			content, err := os.ReadFile(filename)
			if err != nil {
				diags.AddError(
					"Error reading avatar",
					fmt.Sprintf(
						"Could not read file %s, unexpected error: %s",
						filename,
						err,
					),
				)
				return avatar, diags
			}
			err = r.client.CreateFile(ctx, client.CreateFileRequest{
				Filename: normaliseFilename(filename, types.StringNull()),
				SHA:      sha,
				Content:  string(content),
				TeamID:   plan.ID.ValueString(),
			})
			if err != nil {
				diags.AddError(
					"Error uploading avatar",
					fmt.Sprintf(
						"Could not upload avatar %s, unexpected error: %s",
						filename,
						err,
					),
				)
				return avatar, diags
			}
			return sha, nil
		}
	}
	return "", nil
}

func (r *teamConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamConfig
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	avatar, diags := r.uploadAvatarIfPresent(ctx, plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	request, diags := plan.toUpdateTeamRequest(ctx, avatar, types.StringNull())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	response, err := r.client.UpdateTeam(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Team Config",
			"Could not create Team Configuration, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "updated Team Configuration", map[string]interface{}{
		"team_id": response.ID,
	})

	teamConfig, diags := convertResponseToTeamConfig(ctx, response, plan.Avatar)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	diags = resp.State.Set(ctx, teamConfig)
	resp.Diagnostics.Append(diags...)
}

func (r *teamConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamConfig
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetTeam(ctx, state.ID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Team",
			"Could not read Team Configuration, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := convertResponseToTeamConfig(ctx, out, state.Avatar)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *teamConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TeamConfig
	diags := req.Plan.Get(ctx, &plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var state TeamConfig
	diags = req.State.Get(ctx, &state)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	avatar, diags := r.uploadAvatarIfPresent(ctx, plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	request, diags := plan.toUpdateTeamRequest(ctx, avatar, state.Slug)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	response, err := r.client.UpdateTeam(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Team Config",
			"Could not create Team Configuration, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "updated Team configuration", map[string]interface{}{
		"team_id": response.ID,
	})

	teamConfig, diags := convertResponseToTeamConfig(ctx, response, plan.Avatar)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	diags = resp.State.Set(ctx, teamConfig)
	resp.Diagnostics.Append(diags...)
}

func (r *teamConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamConfig
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// We don't actually delete the team, just remove it from state
	resp.State.RemoveResource(ctx)
}
