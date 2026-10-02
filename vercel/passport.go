package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var passportAttrTypes = map[string]attr.Type{
	"enabled":         types.BoolType,
	"connector_id":    types.StringType,
	"deployment_type": types.StringType,
}

type passportConfig struct {
	Enabled        types.Bool   `tfsdk:"enabled"`
	ConnectorID    types.String `tfsdk:"connector_id"`
	DeploymentType types.String `tfsdk:"deployment_type"`
}

func passportResourceSchema(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description:   description + " Requires an eligible Enterprise plan and team owner permissions. Omit this attribute to preserve existing settings; set enabled to false to disable Passport. Disabling does not delete the Connect application or its project connections.",
		Optional:      true,
		Computed:      true,
		PlanModifiers: []planmodifier.Object{passportPlanModifier{}},
		Validators:    []validator.Object{passportValidator{}},
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether Passport is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
			},
			"connector_id": schema.StringAttribute{
				Description: "The stable ID of an existing Vercel Connect OAuth application, available from the vercel_connect_application data source. Required when enabled; omit when disabled.",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"deployment_type": schema.StringAttribute{
				Description: "Deployments to protect: all, preview, prod_deployment_urls_and_all_previews, or all_except_custom_domains. Defaults to all.",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf("all", "preview", "prod_deployment_urls_and_all_previews", "all_except_custom_domains")},
			},
		},
	}
}

func passportDataSourceSchema(description string) datasourceschema.SingleNestedAttribute {
	return datasourceschema.SingleNestedAttribute{
		Description: description,
		Computed:    true,
		Attributes: map[string]datasourceschema.Attribute{
			"enabled":         datasourceschema.BoolAttribute{Computed: true, Description: "Whether Passport is enabled."},
			"connector_id":    datasourceschema.StringAttribute{Computed: true, Description: "The stable ID of the Vercel Connect OAuth application. Null when disabled."},
			"deployment_type": datasourceschema.StringAttribute{Computed: true, Description: "The deployment scope protected when Passport is enabled. Reports all when disabled as a default value; Passport does not protect any deployments when enabled is false."},
		},
	}
}

// The outer pointer distinguishes an omitted PATCH field from an explicit null.
func passportUpdate(ctx context.Context, value types.Object) (**client.Passport, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var config passportConfig
	diags := value.As(ctx, &config, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}
	var passport *client.Passport
	if config.Enabled.ValueBool() {
		passport = &client.Passport{ConnectorID: config.ConnectorID.ValueString(), DeploymentType: config.DeploymentType.ValueString()}
	}
	return &passport, nil
}

func passportState(passport *client.Passport) types.Object {
	connectorID := types.StringNull()
	deploymentType := "all"
	if passport != nil {
		connectorID = types.StringValue(passport.ConnectorID)
		if passport.DeploymentType != "" {
			deploymentType = passport.DeploymentType
		}
	}
	return types.ObjectValueMust(passportAttrTypes, map[string]attr.Value{
		"enabled":         types.BoolValue(passport != nil),
		"connector_id":    connectorID,
		"deployment_type": types.StringValue(deploymentType),
	})
}

type passportValidator struct{}

func (passportValidator) Description(context.Context) string {
	return "Passport requires connector_id when enabled, and no connector_id or deployment scope when disabled."
}

func (v passportValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (passportValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var config passportConfig
	resp.Diagnostics.Append(req.ConfigValue.As(ctx, &config, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() || config.Enabled.IsUnknown() {
		return
	}
	if config.Enabled.IsNull() || config.Enabled.ValueBool() {
		if config.ConnectorID.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtName("connector_id"), "Missing Passport Connector", "connector_id is required when Passport is enabled.")
		}
		return
	}
	if !config.ConnectorID.IsNull() && !config.ConnectorID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(req.Path.AtName("connector_id"), "Disabled Passport Connector", "Omit connector_id when enabled is false.")
	}
	if !config.DeploymentType.IsNull() && !config.DeploymentType.IsUnknown() && config.DeploymentType.ValueString() != "all" {
		resp.Diagnostics.AddAttributeError(req.Path.AtName("deployment_type"), "Disabled Passport Scope", "Omit deployment_type or use all when enabled is false.")
	}
}

// Nested static defaults also run when an optional/computed parent is omitted.
// Apply defaults only to configured blocks so omission preserves observed state.
type passportPlanModifier struct{}

func (passportPlanModifier) Description(context.Context) string {
	return "Defaults configured Passport blocks and preserves omitted settings."
}
func (m passportPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (passportPlanModifier) PlanModifyObject(_ context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if req.ConfigValue.IsNull() {
		if !req.State.Raw.IsNull() && !req.StateValue.IsUnknown() {
			resp.PlanValue = req.StateValue
		}
		return
	}
	if req.ConfigValue.IsUnknown() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	values := req.ConfigValue.Attributes()
	if values["enabled"].IsNull() {
		values["enabled"] = types.BoolValue(true)
	}
	if values["deployment_type"].IsNull() {
		values["deployment_type"] = types.StringValue("all")
	}
	resp.PlanValue = types.ObjectValueMust(passportAttrTypes, values)
}
