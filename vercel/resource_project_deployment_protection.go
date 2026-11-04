package vercel

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &projectDeploymentProtectionResource{}
	_ resource.ResourceWithConfigure   = &projectDeploymentProtectionResource{}
	_ resource.ResourceWithImportState = &projectDeploymentProtectionResource{}
)

var standaloneAuthenticationTypes = map[string]attr.Type{"deployment_type": types.StringType}
var standalonePasswordTypes = map[string]attr.Type{"deployment_type": types.StringType, "password_wo": types.StringType, "password_wo_version": types.Int64Type}
var standaloneTrustedIPTypes = map[string]attr.Type{"deployment_type": types.StringType, "addresses": types.SetType{ElemType: types.ObjectType{AttrTypes: trustedIpAddressAttrType.AttrTypes}}, "protection_mode": types.StringType}

type projectDeploymentProtectionResource struct{ client *client.Client }

type projectDeploymentProtectionModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectID      types.String `tfsdk:"project_id"`
	TeamID         types.String `tfsdk:"team_id"`
	Authentication types.Object `tfsdk:"vercel_authentication"`
	Password       types.Object `tfsdk:"password_protection"`
	TrustedIPs     types.Object `tfsdk:"trusted_ips"`
}

type standalonePassword struct {
	DeploymentType    types.String `tfsdk:"deployment_type"`
	PasswordWO        types.String `tfsdk:"password_wo"`
	PasswordWOVersion types.Int64  `tfsdk:"password_wo_version"`
}

type standaloneTrustedIPs struct {
	DeploymentType types.String       `tfsdk:"deployment_type"`
	Addresses      []TrustedIpAddress `tfsdk:"addresses"`
	ProtectionMode types.String       `tfsdk:"protection_mode"`
}

func newProjectDeploymentProtectionResource() resource.Resource {
	return &projectDeploymentProtectionResource{}
}

func (r *projectDeploymentProtectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_deployment_protection"
}

func (r *projectDeploymentProtectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
	}
}

func standaloneProtectionScope(production bool) schema.StringAttribute {
	scopes := []string{"all", "preview", "prod_deployment_urls_and_all_previews", "all_except_custom_domains"}
	if production {
		scopes = append(scopes, "production")
	}
	return schema.StringAttribute{Required: true, Description: "The deployment scope to protect, using API names: all, preview, prod_deployment_urls_and_all_previews, or all_except_custom_domains. Trusted IPs also supports production.", Validators: []validator.String{stringvalidator.OneOf(scopes...)}}
}

func (r *projectDeploymentProtectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages deployment protection as standalone settings on an existing Vercel project. It never creates or deletes the project, and does not manage builds, Git settings, Passport, bypasses, shareable links, or URL exceptions.

Each configured object enables and owns that protection method. Objects never configured are left untouched. Removing a previously managed object disables that method. Destroying this resource disables only the methods recorded in its state, leaving the project intact. Use a removed block with lifecycle { destroy = false } to relinquish ownership without disabling protection.

If vercel_project manages the same project, set manage_deployment_protection = false there and remove its inline vercel_authentication, password_protection, and trusted_ips attributes. Only one standalone resource may own these settings per project. Passwords are write-only, require Terraform 1.11 or later, and cannot be read or checked for drift. Import reads all enabled protection methods; omitted methods in the subsequent configuration will be disabled on apply.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The project ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the existing project. No project is created by this resource.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The team ID. Defaults to the provider team.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"vercel_authentication": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Enable Vercel Authentication. Omission leaves it unmanaged unless previously managed, in which case removal disables it.",
				Attributes: map[string]schema.Attribute{
					"deployment_type": standaloneProtectionScope(false),
				},
			},
			"password_protection": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Enable password protection. Omission leaves it unmanaged unless previously managed, in which case removal disables it. Import preserves the existing password.",
				Attributes: map[string]schema.Attribute{
					"deployment_type": standaloneProtectionScope(false),
					"password_wo": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						WriteOnly:   true,
						Description: "The password, never stored in state. Use an ephemeral variable to keep it out of saved plans. Required when enabling previously disabled password protection. Existing passwords are preserved when omitted. Change password_wo_version to rotate it.",
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 72),
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("password_wo_version")),
						},
					},
					"password_wo_version": schema.Int64Attribute{
						Optional:    true,
						Description: "A non-secret rotation counter. Set with password_wo and increment to send a new password; changing only password_wo does not trigger an update.",
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
							int64validator.AlsoRequires(path.MatchRelative().AtParent().AtName("password_wo")),
						},
					},
				},
			},
			"trusted_ips": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Set the IPv4 allowlist and deployment scope. Removing this object disables Trusted IPs. The API's existing protection mode is reported but not changed.",
				Attributes: map[string]schema.Attribute{
					"deployment_type": standaloneProtectionScope(true),
					"protection_mode": schema.StringAttribute{
						Computed:    true,
						Description: "The API-reported Trusted IP mode: additional requires Trusted IPs alongside other protection; exclusive allows Trusted IPs as an alternative. The public PATCH schema does not expose mode updates, so this resource does not change it.",
					},
					"addresses": schema.SetNestedAttribute{
						Required:    true,
						Description: "Allowed IPv4 addresses or CIDR ranges.",
						Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									Required:    true,
									Description: "An IPv4 address or CIDR range. IPv6 is not supported.",
									Validators:  []validator.String{standaloneIPv4Validator{}},
								},
								"note": schema.StringAttribute{
									Optional:    true,
									Description: "Optional description, up to 20 characters.",
									Validators:  []validator.String{stringvalidator.LengthAtMost(20)},
								},
							},
						},
					},
				},
			},
		},
	}
}

type standaloneIPv4Validator struct{}

func (standaloneIPv4Validator) Description(context.Context) string {
	return "Must be an IPv4 address or CIDR range."
}
func (v standaloneIPv4Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (standaloneIPv4Validator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	address, addressErr := netip.ParseAddr(value)
	prefix, prefixErr := netip.ParsePrefix(value)
	if addressErr == nil && address.Is4() || prefixErr == nil && prefix.Addr().Is4() {
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid Trusted IP address", "Use an IPv4 address or IPv4 CIDR range; IPv6 is not supported.")
}

func emptyProjectDeploymentProtection() projectDeploymentProtectionModel {
	return projectDeploymentProtectionModel{Authentication: types.ObjectNull(standaloneAuthenticationTypes), Password: types.ObjectNull(standalonePasswordTypes), TrustedIPs: types.ObjectNull(standaloneTrustedIPTypes)}
}

// read records only owned methods, so refresh cannot adopt unrelated protection.
func (m *projectDeploymentProtectionModel) read(ctx context.Context, out client.ProjectResponse, importing bool) diag.Diagnostics {
	var diags diag.Diagnostics
	m.ID, m.ProjectID, m.TeamID = types.StringValue(out.ID), types.StringValue(out.ID), toTeamID(out.TeamID)
	if importing || !m.Authentication.IsNull() {
		m.Authentication = types.ObjectNull(standaloneAuthenticationTypes)
		if out.VercelAuthentication != nil {
			m.Authentication = types.ObjectValueMust(standaloneAuthenticationTypes, map[string]attr.Value{"deployment_type": types.StringValue(out.VercelAuthentication.DeploymentType)})
		}
	}
	if importing || !m.Password.IsNull() {
		version := types.Int64Null()
		if !m.Password.IsNull() && !m.Password.IsUnknown() {
			version = m.Password.Attributes()["password_wo_version"].(types.Int64)
		}
		m.Password = types.ObjectNull(standalonePasswordTypes)
		if out.PasswordProtection != nil {
			m.Password = types.ObjectValueMust(standalonePasswordTypes, map[string]attr.Value{"deployment_type": types.StringValue(out.PasswordProtection.DeploymentType), "password_wo": types.StringNull(), "password_wo_version": version})
		}
	}
	if importing || !m.TrustedIPs.IsNull() {
		m.TrustedIPs = types.ObjectNull(standaloneTrustedIPTypes)
		if out.TrustedIps != nil {
			if len(out.TrustedIps.Addresses) == 0 {
				diags.AddError("Incomplete Trusted IP settings", "The API returned Trusted IP protection without its allowlist. Use credentials that can read the full settings before managing or importing them.")
				return diags
			}
			addresses := make([]TrustedIpAddress, 0, len(out.TrustedIps.Addresses))
			for _, address := range out.TrustedIps.Addresses {
				addresses = append(addresses, TrustedIpAddress{Value: types.StringValue(address.Value), Note: types.StringPointerValue(address.Note)})
			}
			value, d := types.ObjectValueFrom(ctx, standaloneTrustedIPTypes, &standaloneTrustedIPs{DeploymentType: types.StringValue(out.TrustedIps.DeploymentType), Addresses: addresses, ProtectionMode: types.StringValue(out.TrustedIps.ProtectionMode)})
			diags.Append(d...)
			m.TrustedIPs = value
		}
	}
	return diags
}

func (m projectDeploymentProtectionModel) patch(ctx context.Context, previous projectDeploymentProtectionModel, password types.String) (client.ProjectDeploymentProtectionRequest, diag.Diagnostics) {
	var request client.ProjectDeploymentProtectionRequest
	var diags diag.Diagnostics
	if !m.Authentication.Equal(previous.Authentication) {
		var authentication *client.VercelAuthentication
		if m.Authentication.IsUnknown() {
			diags.AddError("Unknown authentication settings", "Authentication settings must be known at apply.")
			return request, diags
		}
		if !m.Authentication.IsNull() {
			var config VercelAuthentication
			diags.Append(m.Authentication.As(ctx, &config, basetypes.ObjectAsOptions{})...)
			authentication = &client.VercelAuthentication{DeploymentType: config.DeploymentType.ValueString()}
		}
		request.Authentication = &authentication
	}
	if !m.Password.Equal(previous.Password) {
		var protection *client.PasswordProtectionWithPassword
		if m.Password.IsUnknown() {
			diags.AddError("Unknown password settings", "Password settings must be known at apply.")
			return request, diags
		}
		if !m.Password.IsNull() {
			var config standalonePassword
			diags.Append(m.Password.As(ctx, &config, basetypes.ObjectAsOptions{})...)
			protection = &client.PasswordProtectionWithPassword{DeploymentType: config.DeploymentType.ValueString()}
			var old standalonePassword
			if !previous.Password.IsNull() {
				diags.Append(previous.Password.As(ctx, &old, basetypes.ObjectAsOptions{})...)
			}
			if previous.Password.IsNull() || !config.PasswordWOVersion.Equal(old.PasswordWOVersion) {
				if password.IsUnknown() {
					diags.AddError("Unknown deployment password", "password_wo must be known at apply when enabling or rotating password protection.")
				}
				if !password.IsNull() {
					protection.Password = password.ValueString()
				}
			}
		}
		request.Password = &protection
	}
	// The computed protection mode can become unknown on an unrelated update;
	// compare only fields this resource can write.
	if !sameTrustedIPSettings(m.TrustedIPs, previous.TrustedIPs) {
		var trustedIPs *client.ProjectProtectionTrustedIPs
		if m.TrustedIPs.IsUnknown() {
			diags.AddError("Unknown Trusted IP settings", "Trusted IP settings must be known at apply.")
			return request, diags
		}
		if !m.TrustedIPs.IsNull() {
			var config standaloneTrustedIPs
			diags.Append(m.TrustedIPs.As(ctx, &config, basetypes.ObjectAsOptions{UnhandledUnknownAsEmpty: true})...)
			trustedIPs = &client.ProjectProtectionTrustedIPs{DeploymentType: config.DeploymentType.ValueString()}
			for _, address := range config.Addresses {
				trustedIPs.Addresses = append(trustedIPs.Addresses, client.TrustedIpAddress{Value: address.Value.ValueString(), Note: address.Note.ValueStringPointer()})
			}
		}
		request.TrustedIPs = &trustedIPs
	}
	return request, diags
}

func sameTrustedIPSettings(a, b types.Object) bool {
	if a.IsNull() || a.IsUnknown() || b.IsNull() || b.IsUnknown() {
		return a.Equal(b)
	}
	return a.Attributes()["deployment_type"].Equal(b.Attributes()["deployment_type"]) && a.Attributes()["addresses"].Equal(b.Attributes()["addresses"])
}

func (r *projectDeploymentProtectionResource) apply(ctx context.Context, plan *projectDeploymentProtectionModel, previous projectDeploymentProtectionModel, config projectDeploymentProtectionModel) diag.Diagnostics {
	password := types.StringNull()
	if !config.Password.IsNull() && !config.Password.IsUnknown() {
		password = config.Password.Attributes()["password_wo"].(types.String)
	}
	request, diags := plan.patch(ctx, previous, password)
	if diags.HasError() {
		return diags
	}
	// Read first so adopting existing password protection never requires rotating it.
	out, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if err != nil {
		diags.AddError("Error reading project deployment protection", err.Error())
		return diags
	}
	if request.Password != nil && *request.Password != nil && out.PasswordProtection == nil && (*request.Password).Password == "" {
		diags.AddAttributeError(path.Root("password_protection").AtName("password_wo"), "Password required", "Supply password_wo and password_wo_version when enabling previously disabled password protection.")
		return diags
	}
	if request.Authentication != nil || request.Password != nil || request.TrustedIPs != nil {
		out, err = r.client.UpdateProjectDeploymentProtection(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), request)
		if err != nil {
			diags.AddError("Error updating project deployment protection", err.Error())
			return diags
		}
	}
	diags.Append(plan.read(ctx, out, false)...)
	return diags
}

func (r *projectDeploymentProtectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config projectDeploymentProtectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.apply(ctx, &plan, emptyProjectDeploymentProtection(), config)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *projectDeploymentProtectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectDeploymentProtectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetProject(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading project deployment protection", err.Error())
		return
	}
	resp.Diagnostics.Append(state.read(ctx, out, false)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

func (r *projectDeploymentProtectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, previous, config projectDeploymentProtectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.apply(ctx, &plan, previous, config)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *projectDeploymentProtectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectDeploymentProtectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request, diags := emptyProjectDeploymentProtection().patch(ctx, state, types.StringNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if request.Authentication == nil && request.Password == nil && request.TrustedIPs == nil {
		return
	}
	_, err := r.client.UpdateProjectDeploymentProtection(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), request)
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error disabling project deployment protection", err.Error())
	}
}

func (r *projectDeploymentProtectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", "Use project_id or team_id/project_id.")
		return
	}
	out, err := r.client.GetProject(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing project deployment protection", err.Error())
		return
	}
	state := emptyProjectDeploymentProtection()
	resp.Diagnostics.Append(state.read(ctx, out, true)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}
