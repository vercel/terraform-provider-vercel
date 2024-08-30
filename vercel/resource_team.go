package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &teamResource{}
	_ resource.ResourceWithConfigure   = &teamResource{}
	_ resource.ResourceWithImportState = &teamResource{}
)

func newTeamResource() resource.Resource {
	return &teamResource{}
}

type teamResource struct {
	client *client.Client
}

func (r *teamResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *teamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a Vercel Team resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Team.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the team.",
				Required:    true,
			},
			"plan": schema.StringAttribute{
				Description: "The plan of the team. Can be 'hobby' or 'pro'.",
				Required:    true,
				Validators: []validator.String{
					stringOneOf("hobby", "pro"),
				},
			},
			"avatar": schema.StringAttribute{
				Optional:    true,
				Description: "The hash of an uploaded image to act as the team's avatar.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the team.",
			},
			"slug": schema.StringAttribute{
				Required:      true,
				Description:   "The slug of the team. Will be used in the URL of the team's dashboard.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sensitive_environment_variable_policy": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringOneOf("on", "off"),
				},
			},
			"email_domain": schema.StringAttribute{
				Optional:    true,
				Description: "Hostname that'll be matched with emails on sign-up to automatically join the Team.",
			},
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
			"invite_code": schema.StringAttribute{
				Computed:    true,
				Description: "A code that can be used to join this team. Only visible to Team owners.",
			},
			"billing": schema.SingleNestedAttribute{
				Description: "Billing information for the team.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"currency": schema.StringAttribute{
						Description: "The currency used for billing.",
						Computed:    true,
					},
					"email": schema.StringAttribute{
						Description: "The email address used for billing.",
						Computed:    true,
					},
					"tax_id": schema.StringAttribute{
						Description: "The tax ID for the team.",
						Computed:    true,
					},
					"language": schema.StringAttribute{
						Description: "The language used for billing.",
						Computed:    true,
					},
					"address": schema.SingleNestedAttribute{
						Computed:    true,
						Description: "The billing address for the team.",
						Attributes: map[string]schema.Attribute{
							"line1": schema.StringAttribute{
								Computed: true,
							},
							"line2": schema.StringAttribute{
								Computed: true,
							},
							"postal_code": schema.StringAttribute{
								Computed: true,
							},
							"city": schema.StringAttribute{
								Computed: true,
							},
							"country": schema.StringAttribute{
								Computed: true,
							},
							"state": schema.StringAttribute{
								Computed: true,
							},
						},
					},
				},
			},
			"preview_deployment_suffix": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The hostname that is used as the preview deployment suffix.",
			},
			"remote_caching": schema.SingleNestedAttribute{
				Description: "Configuration for Remote Caching.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Indicates if Remote Caching is enabled.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"enable_preview_feedback": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringOneOf("default", "on", "off"),
				},
			},
			"spaces": schema.SingleNestedAttribute{
				Description: "Configuration for Spaces.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Indicates if Spaces is enabled.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"hide_ip_addresses": schema.BoolAttribute{
				Optional:    true,
				Description: "Indicates if ip addresses should be accessible in o11y tooling.",
			},
		},
	}
}

type SamlConnection struct {
	Status types.String `tfsdk:"status"`
}

type SamlDirectory struct {
	Type  types.String `tfsdk:"type"`
	State types.String `tfsdk:"state"`
}

type BillingAddress struct {
	Line1      types.String `tfsdk:"line1"`
	Line2      types.String `tfsdk:"line2"`
	PostalCode types.String `tfsdk:"postal_code"`
	City       types.String `tfsdk:"city"`
	Country    types.String `tfsdk:"country"`
	State      types.String `tfsdk:"state"`
}

type Billing struct {
	Currency types.String    `tfsdk:"currency"`
	Email    types.String    `tfsdk:"email"`
	TaxID    types.String    `tfsdk:"tax_id"`
	Language types.String    `tfsdk:"language"`
	Address  *BillingAddress `tfsdk:"address"`
}

type Saml struct {
	Enforced      types.Bool      `tfsdk:"enforced"`
	Roles         types.Map       `tfsdk:"roles"`
	AccessGroupId types.String    `tfsdk:"access_group_id"`
	Connection    *SamlConnection `tfsdk:"connection"`
	Directory     *SamlDirectory  `tfsdk:"directory"`
}

type EnableConfig struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type Team struct {
	ID                                 types.String `tfsdk:"id"`
	Name                               types.String `tfsdk:"name"`
	Plan                               types.String `tfsdk:"plan"`
	Avatar                             types.String `tfsdk:"avatar"`
	Description                        types.String `tfsdk:"description"`
	Slug                               types.String `tfsdk:"slug"`
	SensitiveEnvironmentVariablePolicy types.String `tfsdk:"sensitive_environment_variable_policy"`
	EmailDomain                        types.String `tfsdk:"email_domain"`
	Saml                               types.Object `tfsdk:"saml"`
	InviteCode                         types.String `tfsdk:"invite_code"`
	Billing                            types.Object `tfsdk:"billing"`
	PreviewDeploymentSuffix            types.String `tfsdk:"preview_deployment_suffix"`
	RemoteCaching                      types.Object `tfsdk:"remote_caching"`
	EnablePreviewFeedback              types.String `tfsdk:"enable_preview_feedback"`
	Spaces                             types.Object `tfsdk:"spaces"`
	HideIPAddresses                    types.Bool   `tfsdk:"hide_ip_addresses"`
}

func (t *Team) toCreateTeamRequest() client.TeamCreateRequest {
	return client.TeamCreateRequest{
		Slug: t.Slug.String(),
		Name: t.Name.String(),
	}
}

var objectAsOptions = basetypes.ObjectAsOptions{
	UnhandledNullAsEmpty:    true,
	UnhandledUnknownAsEmpty: true,
}

func (r *teamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Team
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var saml *Saml
	if !plan.Saml.IsNull() && !plan.Saml.IsUnknown() {
		diags = plan.Saml.As(ctx, &saml, objectAsOptions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var billing *Billing
	if !plan.Billing.IsNull() && !plan.Billing.IsUnknown() {
		diags = plan.Billing.As(ctx, &billing, objectAsOptions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var remoteCaching *EnableConfig
	if !plan.RemoteCaching.IsNull() && !plan.RemoteCaching.IsUnknown() {
		diags = plan.RemoteCaching.As(ctx, &remoteCaching, objectAsOptions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var spaces *EnableConfig
	if !plan.Spaces.IsNull() && !plan.Spaces.IsUnknown() {
		diags = plan.Spaces.As(ctx, &spaces, objectAsOptions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	out, err := r.client.CreateTeam(ctx, plan.toCreateTeamRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Team",
			"Could not create Team, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := responseToTeam(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "created Team", map[string]interface{}{
		"team_id": out.ID,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
