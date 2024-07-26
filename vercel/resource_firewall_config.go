package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &firewallConfigResource{}
	_ resource.ResourceWithConfigure   = &firewallConfigResource{}
	_ resource.ResourceWithImportState = &firewallConfigResource{}
)

func newFirewallConfigResource() resource.Resource { return &firewallConfigResource{} }

type firewallConfigResource struct {
	client *client.Client
}

func (r *firewallConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_config"
}

func (r *firewallConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Define Custom Rules to shape the way your traffic is handled by the Vercel Edge Network.`,
		Blocks: map[string]schema.Block{
			"managed_rulesets": schema.SingleNestedBlock{
				Description: "The managed rulesets that are enabled.",
				Blocks: map[string]schema.Block{
					"owasp": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"xss": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
							"sqli": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
							"lfi": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
							"rfi": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
							"rce": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
							"sd": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
							"ma": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
							"php": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
							"gen": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
							"java": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"active": schema.BoolAttribute{
										Optional: true,
									},
									"action": schema.StringAttribute{
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"rules": schema.SingleNestedBlock{
				Blocks: map[string]schema.Block{
					"rule": schema.ListNestedBlock{
						Validators: []validator.List{
							listvalidator.UniqueValues(),
						},
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed: true,
								},
								"name": schema.StringAttribute{
									Required: true,
									Validators: []validator.String{
										stringvalidator.LengthBetween(4, 160),
									},
								},
								"description": schema.StringAttribute{
									Optional: true,
									Validators: []validator.String{
										stringvalidator.LengthAtMost(260),
									},
								},
								"active": schema.BoolAttribute{
									Optional: true,
								},
								"action": schema.SingleNestedAttribute{
									Required: true,
									Attributes: map[string]schema.Attribute{
										"action": schema.StringAttribute{
											Required: true,
											Validators: []validator.String{
												stringvalidator.OneOf("bypass", "log", "challenge", "deny", "rate_limit", "redirect"),
											},
										},
										"rate_limit": schema.SingleNestedAttribute{
											Optional: true,
											Attributes: map[string]schema.Attribute{
												"algo": schema.StringAttribute{
													Required: true,
												},
												"window": schema.Int64Attribute{
													Required: true,
												},
												"limit": schema.Int64Attribute{
													Required: true,
												},
												"keys": schema.ListAttribute{
													Required:    true,
													ElementType: types.StringType,
												},
												"action": schema.StringAttribute{
													Required: true,
													Validators: []validator.String{
														stringvalidator.OneOf("bypass", "log", "challenge", "deny", "rate_limit"),
													},
												},
											},
										},
										"redirect": schema.SingleNestedAttribute{
											Optional: true,
											Attributes: map[string]schema.Attribute{
												"location": schema.StringAttribute{
													Required: true,
												},
												"permanent": schema.BoolAttribute{
													Required: true,
												},
											},
										},
										"action_duration": schema.StringAttribute{
											Optional: true,
										},
									},
								},
								"condition_group": schema.ListNestedAttribute{
									Required: true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"conditions": schema.ListNestedAttribute{
												Required: true,
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Required: true,
															Validators: []validator.String{
																stringvalidator.OneOf(
																	"host",
																	"path",
																	"method",
																	"header",
																	"query",
																	"cookie",
																	"target_path",
																	"ip_address",
																	"region",
																	"protocol",
																	"scheme",
																	"environment",
																	"user_agent",
																	"geo_continent",
																	"geo_country",
																	"geo_country_region",
																	"geo_city",
																	"geo_as_number",
																	"ja4_digest",
																	"ja3_digest",
																),
															},
														},
														"op": schema.StringAttribute{
															Required: true,
															Validators: []validator.String{
																stringvalidator.OneOf(
																	"re",
																	"eq",
																	"neq",
																	"ex",
																	"nex",
																	"inc",
																	"ninc",
																	"pre",
																	"suf",
																	"sub",
																	"gt",
																	"gte",
																	"lt",
																	"lte",
																),
															},
														},
														"neg": schema.BoolAttribute{
															Optional: true,
														},
														"key": schema.StringAttribute{
															Optional: true,
														},
														"value": schema.StringAttribute{
															Required: true,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"ip_rules": schema.SingleNestedBlock{
				Description: "IP rules to apply to the project.",
				Blocks: map[string]schema.Block{
					"rule": schema.ListNestedBlock{
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed: true,
								},
								"hostname": schema.StringAttribute{
									Required: true,
								},
								"notes": schema.StringAttribute{
									Optional: true,
								},
								"ip": schema.StringAttribute{
									Required: true,
								},
								"action": schema.StringAttribute{
									Required: true,
									Validators: []validator.String{
										stringvalidator.OneOf("bypass", "log", "challenge", "deny"),
									},
								},
							},
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description:   "The ID of the project this configuration belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Description:   "The ID of the team this project belongs to.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether firewall is enabled or not.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *firewallConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *client.Client, got: %T. Please report this issue to the provider developers.",
		)
		return
	}

	r.client = client
}

type FirewallConfig struct {
	ProjectID       types.String             `tfsdk:"project_id"`
	TeamID          types.String             `tfsdk:"team_id"`
	Enabled         types.Bool               `tfsdk:"enabled"`
	ManagedRulesets *FirewallManagedRulesets `tfsdk:"managed_rulesets"`

	Rules   *FirewallRules `tfsdk:"rules"`
	IPRules *IPRules       `tfsdk:"ip_rules"`
}

type FirewallManagedRulesets struct {
	OWASP *CRSRule `tfsdk:"owasp"`
}

type CRSRule struct {
	XSS  *CRSRuleConfig `tfsdk:"xss"`
	SQLI *CRSRuleConfig `tfsdk:"sqli"`
	LFI  *CRSRuleConfig `tfsdk:"lfi"`
	RFI  *CRSRuleConfig `tfsdk:"rfi"`
	RCE  *CRSRuleConfig `tfsdk:"rce"`
	SD   *CRSRuleConfig `tfsdk:"sd"`
	MA   *CRSRuleConfig `tfsdk:"ma"`
	PHP  *CRSRuleConfig `tfsdk:"php"`
	GEN  *CRSRuleConfig `tfsdk:"gen"`
	JAVA *CRSRuleConfig `tfsdk:"java"`
}

func (r *CRSRule) ToMap() map[string]*CRSRuleConfig {
	return map[string]*CRSRuleConfig{
		"xss":  r.XSS,
		"sqli": r.SQLI,
		"lfi":  r.LFI,
		"rfi":  r.RFI,
		"rce":  r.RCE,
		"sd":   r.SD,
		"ma":   r.MA,
		"php":  r.PHP,
		"gen":  r.GEN,
		"java": r.JAVA,
	}
}

type CRSRuleConfig struct {
	Active types.Bool   `tfsdk:"active"`
	Action types.String `tfsdk:"action"`
}

type FirewallRules struct {
	Rules []FirewallRule `tfsdk:"rule"`
}

type FirewallRule struct {
	ID             types.String     `tfsdk:"id"`
	Name           types.String     `tfsdk:"name"`
	Description    types.String     `tfsdk:"description"`
	Active         types.Bool       `tfsdk:"active"`
	ConditionGroup []ConditionGroup `tfsdk:"condition_group"`
	Action         Mitigate         `tfsdk:"action"`
}

func (r *FirewallRule) Conditions() []client.ConditionGroup {
	var groups []client.ConditionGroup
	for _, group := range r.ConditionGroup {
		var conditions []client.Condition
		for _, condition := range group.Conditions {
			conditions = append(conditions, client.Condition{
				Type:  condition.Type.ValueString(),
				Op:    condition.Op.ValueString(),
				Neg:   condition.Neg.ValueBool(),
				Key:   condition.Key.ValueString(),
				Value: condition.Value.ValueString(),
			})
		}
		groups = append(groups, client.ConditionGroup{
			Conditions: conditions,
		})
	}
	return groups
}

func (r *FirewallRule) Mitigate() client.Mitigate {
	mit := client.Mitigate{
		Action: r.Action.Action.ValueString(),
	}
	if r.Action.RateLimit != nil {
		keys := make([]string, len(r.Action.RateLimit.Keys))
		for i, k := range r.Action.RateLimit.Keys {
			keys[i] = k.ValueString()
		}
		mit.RateLimit = &client.RateLimit{
			Algo:   r.Action.RateLimit.Algo.ValueString(),
			Window: r.Action.RateLimit.Window.ValueInt64(),
			Limit:  r.Action.RateLimit.Limit.ValueInt64(),
			Keys:   keys,
			Action: r.Action.RateLimit.Action.ValueString(),
		}
	}
	if r.Action.Redirect != nil {
		mit.Redirect = &client.Redirect{
			Location:  r.Action.Redirect.Location.ValueString(),
			Permanent: r.Action.Redirect.Permanent.ValueBool(),
		}
	}
	if !r.Action.ActionDuration.IsNull() {
		mit.ActionDuration = r.Action.ActionDuration.ValueString()
	}
	return mit
}

func fromFirewallRule(rule client.FirewallRule, ref FirewallRule) FirewallRule {
	r := FirewallRule{
		ID:          types.StringValue(rule.ID),
		Name:        types.StringValue(rule.Name),
		Description: types.StringValue(rule.Description),
		Active:      types.BoolValue(rule.Active),
	}
	if rule.Active == true && ref.Active == types.BoolNull() {
		r.Active = ref.Active
	}

	r.Action = fromMitigate(rule.Action.Mitigate, ref.Action)
	var conditionGroups = make([]ConditionGroup, len(rule.ConditionGroup))
	for j, group := range rule.ConditionGroup {
		var conditions = make([]Condition, len(group.Conditions))
		for k, condition := range group.Conditions {
			var cond = Condition{}
			if len(ref.ConditionGroup) > j && len(ref.ConditionGroup[j].Conditions) > k {
				cond = ref.ConditionGroup[j].Conditions[k]
			}
			conditions[k] = fromCondition(condition, cond)
		}
		conditionGroups[j] = ConditionGroup{
			Conditions: conditions,
		}
	}
	r.ConditionGroup = conditionGroups
	// Description and active can be optional
	if rule.Description == "" && ref.Description == types.StringNull() {
		r.Description = ref.Description
	}
	if rule.Active == true && ref.Active == types.BoolNull() {
		r.Active = ref.Active
	}

	return r
}

type Mitigate struct {
	Action         types.String `tfsdk:"action"`
	RateLimit      *RateLimit   `tfsdk:"rate_limit"`
	Redirect       *Redirect    `tfsdk:"redirect"`
	ActionDuration types.String `tfsdk:"action_duration"`
}

func fromMitigate(mitigate client.Mitigate, ref Mitigate) Mitigate {
	m := Mitigate{
		Action:         types.StringValue(mitigate.Action),
		ActionDuration: types.StringValue(mitigate.ActionDuration),
	}

	if mitigate.ActionDuration == "" && ref.ActionDuration == types.StringNull() {
		m.ActionDuration = ref.ActionDuration
	}

	if mitigate.RateLimit != nil {
		keys := make([]types.String, len(mitigate.RateLimit.Keys))
		for i, k := range mitigate.RateLimit.Keys {
			keys[i] = types.StringValue(k)
		}
		m.RateLimit = &RateLimit{
			Algo:   types.StringValue(mitigate.RateLimit.Algo),
			Window: types.Int64Value(mitigate.RateLimit.Window),
			Limit:  types.Int64Value(mitigate.RateLimit.Limit),
			Keys:   keys,
			Action: types.StringValue(mitigate.RateLimit.Action),
		}
	}
	if mitigate.Redirect != nil {
		m.Redirect = &Redirect{
			Location:  types.StringValue(mitigate.Redirect.Location),
			Permanent: types.BoolValue(mitigate.Redirect.Permanent),
		}
	}
	return m
}

type Redirect struct {
	Location  types.String `tfsdk:"location"`
	Permanent types.Bool   `tfsdk:"permanent"`
}

type RateLimit struct {
	Algo   types.String   `tfsdk:"algo"`
	Window types.Int64    `tfsdk:"window"`
	Limit  types.Int64    `tfsdk:"limit"`
	Keys   []types.String `tfsdk:"keys"`
	Action types.String   `tfsdk:"action"`
}

type ConditionGroup struct {
	Conditions []Condition `tfsdk:"conditions"`
}

type Condition struct {
	Type  types.String `tfsdk:"type"`
	Op    types.String `tfsdk:"op"`
	Neg   types.Bool   `tfsdk:"neg"`
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func fromCondition(condition client.Condition, ref Condition) Condition {
	c := Condition{
		Type:  types.StringValue(condition.Type),
		Op:    types.StringValue(condition.Op),
		Value: types.StringValue(condition.Value),
		Key:   types.StringValue(condition.Key),
		Neg:   types.BoolValue(condition.Neg),
	}
	// Neg and Key are optional
	if ref.Neg == types.BoolNull() {
		c.Neg = types.BoolNull()
	}
	if ref.Key == types.StringNull() {
		c.Key = types.StringNull()
	}
	return c
}

type IPRules struct {
	Rules []IPRule `tfsdk:"rule"`
}

type IPRule struct {
	ID       types.String `tfsdk:"id"`
	Hostname types.String `tfsdk:"hostname"`
	IP       types.String `tfsdk:"ip"`
	Notes    types.String `tfsdk:"notes"`
	Action   types.String `tfsdk:"action"`
}

func fromCRS(conf map[string]client.CoreRuleSet, refMr *FirewallManagedRulesets) *CRSRule {
	var ref = &CRSRule{}
	if refMr != nil && refMr.OWASP != nil {
		ref = refMr.OWASP
	}
	if conf == nil || ref == nil {
		return nil
	}
	return &CRSRule{
		XSS:  fromCoreRuleset(conf["xss"], ref.XSS),
		SQLI: fromCoreRuleset(conf["sqli"], ref.SQLI),
		LFI:  fromCoreRuleset(conf["lfi"], ref.LFI),
		RFI:  fromCoreRuleset(conf["rfi"], ref.RFI),
		RCE:  fromCoreRuleset(conf["rce"], ref.RCE),
		SD:   fromCoreRuleset(conf["sd"], ref.SD),
		MA:   fromCoreRuleset(conf["ma"], ref.MA),
		PHP:  fromCoreRuleset(conf["php"], ref.PHP),
		GEN:  fromCoreRuleset(conf["gen"], ref.GEN),
		JAVA: fromCoreRuleset(conf["java"], ref.JAVA),
	}
}

func fromCoreRuleset(crsRule client.CoreRuleSet, ref *CRSRuleConfig) *CRSRuleConfig {
	if ref == nil && crsRule.Active == false && crsRule.Action == "log" {
		return nil
	}
	c := &CRSRuleConfig{
		Active: types.BoolValue(crsRule.Active),
		Action: types.StringValue(crsRule.Action),
	}
	if (ref == nil && crsRule.Active == true) ||
		ref != nil && ref.Active == types.BoolNull() {
		c.Active = types.BoolNull()
	}
	return c
}

func fromClient(conf client.FirewallConfig, state FirewallConfig) FirewallConfig {
	cfg := FirewallConfig{
		ProjectID: state.ProjectID,
		// Take the teamID from the response/provider if it wasn't provided in resource
		TeamID:  types.StringValue(conf.TeamID),
		Enabled: state.Enabled,
	}
	// Enabled can be null
	if conf.Enabled && state.Enabled.IsNull() {
		cfg.Enabled = state.Enabled
	}

	if len(conf.Rules) > 0 {
		rules := make([]FirewallRule, len(conf.Rules))
		for i, rule := range conf.Rules {
			// Set empty optional types
			var stateRule = FirewallRule{
				Active: types.BoolNull(),
			}
			if state.Rules != nil && len(state.Rules.Rules)-1 > i {
				stateRule = state.Rules.Rules[i]
			}
			rules[i] = fromFirewallRule(rule, stateRule)

		}
		cfg.Rules = &FirewallRules{Rules: rules}
	}

	if len(conf.IPRules) > 0 {
		ipRules := make([]IPRule, len(conf.IPRules))
		for i, iprule := range conf.IPRules {
			ipRules[i] = IPRule{
				ID:       types.StringValue(iprule.ID),
				Hostname: types.StringValue(iprule.Hostname),
				IP:       types.StringValue(iprule.IP),
				Notes:    types.StringValue(iprule.Notes),
				Action:   types.StringValue(iprule.Action),
			}
			// notes don't have to be set
			if iprule.Notes == "" && state.IPRules != nil && len(state.IPRules.Rules) > i && state.IPRules.Rules[i].Notes.IsNull() {
				ipRules[i].Notes = state.IPRules.Rules[i].Notes
			}
		}

		cfg.IPRules = &IPRules{Rules: ipRules}
	}

	managedRulesets := &FirewallManagedRulesets{}
	if conf.ManagedRulesets != nil && conf.CRS != nil {
		cfg.ManagedRulesets = managedRulesets
		cfg.ManagedRulesets.OWASP = fromCRS(conf.CRS, state.ManagedRulesets)
	}

	return cfg
}

func (f *FirewallConfig) toClient() client.FirewallConfig {
	conf := client.FirewallConfig{
		ProjectID: f.ProjectID.ValueString(),
		TeamID:    f.TeamID.ValueString(),
		Enabled:   f.Enabled.IsNull() || f.Enabled.ValueBool(),
	}

	if f.ManagedRulesets != nil {
		conf.ManagedRulesets = make(map[string]client.ManagedRule)
		if f.ManagedRulesets.OWASP != nil {
			conf.ManagedRulesets["owasp"] = client.ManagedRule{
				Active: true,
			}
			conf.CRS = make(map[string]client.CoreRuleSet)
			for key, value := range f.ManagedRulesets.OWASP.ToMap() {
				if value != nil {
					conf.CRS[key] = client.CoreRuleSet{
						Action: value.Action.ValueString(),
						Active: value.Active.IsNull() || value.Active.ValueBool(),
					}
				}
			}
		}
	}
	if f.Rules != nil && len(f.Rules.Rules) > 0 {
		for _, rule := range f.Rules.Rules {
			conf.Rules = append(conf.Rules, client.FirewallRule{
				ID:             rule.ID.ValueString(),
				Name:           rule.Name.ValueString(),
				Description:    rule.Description.ValueString(),
				Active:         rule.Active.IsNull() || rule.Active.ValueBool(),
				ConditionGroup: rule.Conditions(),
				Action: client.Action{
					Mitigate: rule.Mitigate(),
				},
			})
		}
	}

	if f.IPRules != nil && len(f.IPRules.Rules) > 0 {
		for _, iprule := range f.IPRules.Rules {
			conf.IPRules = append(conf.IPRules, client.IPRule{
				ID:       iprule.ID.ValueString(),
				Hostname: iprule.Hostname.ValueString(),
				IP:       iprule.IP.ValueString(),
				Notes:    iprule.Notes.ValueString(),
				Action:   iprule.Action.ValueString(),
			})
		}
	}
	return conf
}

func (r *firewallConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan FirewallConfig
	diags := req.Plan.Get(ctx, &plan)
	if resp.Diagnostics.HasError() {
		return
	}

	conf := plan.toClient()

	out, err := r.client.PutFirewallConfig(ctx, conf)
	if err != nil {
		diags.AddError("failed to create firewall config", err.Error())
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg := fromClient(out, plan)
	diags = resp.State.Set(ctx, cfg)
	resp.Diagnostics.Append(diags...)
	return
}

func (r *firewallConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallConfig
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetFirewallConfig(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if err != nil {
		diags.AddError("failed to read firewall config", err.Error())
	}
	cfg := fromClient(out, state)
	diags = resp.State.Set(ctx, cfg)
	resp.Diagnostics.Append(diags...)
	return
}

func (r *firewallConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallConfig
	diags := req.Plan.Get(ctx, &plan)
	if resp.Diagnostics.HasError() {
		return
	}

	conf := plan.toClient()

	out, err := r.client.PutFirewallConfig(ctx, conf)
	if err != nil {
		diags.AddError("failed to create firewall config", err.Error())
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg := fromClient(out, plan)
	diags = resp.State.Set(ctx, cfg)
	resp.Diagnostics.Append(diags...)
	return
}
func (r *firewallConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallConfig
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conf := client.FirewallConfig{
		Enabled:   false,
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
	}

	_, err := r.client.PutFirewallConfig(ctx, conf)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete firewall config", err.Error())
	}
	tflog.Info(ctx, "deleted firewall config", map[string]interface{}{
		"project_id": state.ProjectID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
	})
	return
}
func (r *firewallConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Firewall Config",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}
	out, err := r.client.GetFirewallConfig(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing Firewall Config", err.Error())
		return
	}
	conf := fromClient(out, FirewallConfig{
		ProjectID: types.StringValue(projectID),
		TeamID:    types.StringValue(out.TeamID), // use output teamID if not provided on import
	})
	tflog.Info(ctx, "imported firewall config", map[string]interface{}{
		"team_id":    conf.TeamID.ValueString(),
		"project_id": conf.ProjectID.ValueString(),
	})
	diags := resp.State.Set(ctx, conf)
	resp.Diagnostics.Append(diags...)
}
