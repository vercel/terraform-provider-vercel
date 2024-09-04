package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
						Description: "Enable the owasp managed rulesets and select ruleset behaviors",
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
				Description: "Custom rules to apply to the project",
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
									Description: "Name to identify the rule",
									Required:    true,
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
									Description: "Whether the rule is active or not",
									Optional:    true,
								},
								"action": schema.SingleNestedAttribute{
									Description: "Actions to take when the condition groups match a request",
									Required:    true,
									Attributes: map[string]schema.Attribute{
										"action": schema.StringAttribute{
											Description: "Base action",
											Required:    true,
											Validators: []validator.String{
												stringvalidator.OneOf("bypass", "log", "challenge", "deny", "rate_limit", "redirect"),
											},
										},
										"rate_limit": schema.SingleNestedAttribute{
											Description: "Behavior or a rate limiting action. Required if action is rate_limit",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"algo": schema.StringAttribute{
													Description: "Rate limiting algorithm",
													Required:    true,
												},
												"window": schema.Int64Attribute{
													Description: "Time window in seconds",
													Required:    true,
												},
												"limit": schema.Int64Attribute{
													Description: "number of requests allowed in the window",
													Required:    true,
												},
												"keys": schema.ListAttribute{
													Description: "Keys used to bucket an individual client",
													Required:    true,
													ElementType: types.StringType,
												},
												"action": schema.StringAttribute{
													Description: "Action taken when rate limit is exceeded",
													Required:    true,
													Validators: []validator.String{
														stringvalidator.OneOf("bypass", "log", "challenge", "deny", "rate_limit"),
													},
												},
											},
										},
										"redirect": schema.SingleNestedAttribute{
											Description: "How to redirect a request. Required if action is redirect",
											Optional:    true,
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
											Description: "Forward persistence of a rule aciton",
											Optional:    true,
										},
									},
								},
								"condition_group": schema.ListNestedAttribute{
									Description: "Sets of conditions that may match a request",
									Required:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"conditions": schema.ListNestedAttribute{
												Description: "Conditions that must all match within a group",
												Required:    true,
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Description: "Request key type to match against",
															Required:    true,
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
															Description: "How to comparse type to value",
															Required:    true,
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
															Description: "Key within type to match against",
															Optional:    true,
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
									Description: "Hosts to apply these rules to",
									Required:    true,
								},
								"notes": schema.StringAttribute{
									Optional: true,
								},
								"ip": schema.StringAttribute{
									Description: "IP or CIDR to block",
									Required:    true,
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

func (r *FirewallRule) Mitigate() (client.Mitigate, error) {
	mit := client.Mitigate{
		Action: r.Action.Action.ValueString(),
	}
	if !r.Action.RateLimit.IsNull() {
		rl := &client.RateLimit{}
		diags := r.Action.RateLimit.As(context.Background(), rl, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    false,
			UnhandledUnknownAsEmpty: false,
		})
		if diags.HasError() {
			return mit, fmt.Errorf("error converting rate limit: %s - %s", diags[0].Summary(), diags[0].Detail())
		}
		mit.RateLimit = rl
	}

	if !r.Action.Redirect.IsNull() {
		rd := &client.Redirect{}
		diags := r.Action.Redirect.As(context.Background(), rd, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})
		if diags.HasError() {
			return mit, fmt.Errorf("error converting rate limit: %s - %s", diags[0].Summary(), diags[0].Detail())
		}
		mit.Redirect = rd
	}
	if !r.Action.ActionDuration.IsNull() {
		mit.ActionDuration = r.Action.ActionDuration.ValueString()
	}
	return mit, nil
}

func fromFirewallRule(rule client.FirewallRule, ref FirewallRule) (FirewallRule, error) {
	var err error
	r := FirewallRule{
		ID:          types.StringValue(rule.ID),
		Name:        types.StringValue(rule.Name),
		Description: types.StringValue(rule.Description),
		Active:      types.BoolValue(rule.Active),
	}
	if rule.Active && ref.Active == types.BoolNull() {
		r.Active = ref.Active
	}

	r.Action, err = fromMitigate(rule.Action.Mitigate, ref.Action)
	if err != nil {
		return r, err
	}
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
	if rule.Active && ref.Active == types.BoolNull() {
		r.Active = ref.Active
	}

	return r, nil
}

/*
	type Mitigate struct {
		Action         types.String `tfsdk:"action"`
		RateLimit      *RateLimit   `tfsdk:"rate_limit"`
		Redirect       *Redirect    `tfsdk:"redirect"`
		ActionDuration types.String `tfsdk:"action_duration"`
	}
*/
var redirectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"location":  types.StringType,
		"permanent": types.BoolType,
	},
}

var ratelimitType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"algo":   types.StringType,
		"window": types.Int64Type,
		"limit":  types.Int64Type,
		"keys": types.ListType{
			ElemType: types.StringType,
		},
		"action": types.StringType,
	},
}

type Mitigate struct {
	Action         types.String `tfsdk:"action"`
	RateLimit      types.Object `tfsdk:"rate_limit"`
	Redirect       types.Object `tfsdk:"redirect"`
	ActionDuration types.String `tfsdk:"action_duration"`
}

func fromMitigate(mitigate client.Mitigate, ref Mitigate) (Mitigate, error) {
	m := Mitigate{
		Action:         types.StringValue(mitigate.Action),
		ActionDuration: types.StringValue(mitigate.ActionDuration),
		Redirect:       types.ObjectNull(redirectType.AttrTypes),
		RateLimit:      types.ObjectNull(ratelimitType.AttrTypes),
	}

	if mitigate.ActionDuration == "" && ref.ActionDuration == types.StringNull() {
		m.ActionDuration = ref.ActionDuration
	}

	if mitigate.RateLimit != nil {
		// TODO diags
		keys, diags := basetypes.NewListValueFrom(context.Background(), types.StringType, mitigate.RateLimit.Keys)
		if diags.HasError() {
			return m, fmt.Errorf("error converting keys: %s - %s", diags[0].Summary(), diags[0].Detail())
		}
		m.RateLimit = types.ObjectValueMust(
			ratelimitType.AttrTypes,
			map[string]attr.Value{
				"algo":   types.StringValue(mitigate.RateLimit.Algo),
				"window": types.Int64Value(mitigate.RateLimit.Window),
				"limit":  types.Int64Value(mitigate.RateLimit.Limit),
				"keys":   keys,
				"action": types.StringValue(mitigate.RateLimit.Action),
			},
		)
	}
	if mitigate.Redirect != nil {
		m.Redirect = types.ObjectValueMust(
			redirectType.AttrTypes,
			map[string]attr.Value{
				"location":  types.StringValue(mitigate.Redirect.Location),
				"permanent": types.BoolValue(mitigate.Redirect.Permanent),
			},
		)
	}
	return m, nil
}

type Redirect struct {
	Location  types.String `tfsdk:"location"`
	Permanent types.Bool   `tfsdk:"permanent"`
}

type RateLimit struct {
	Algo   types.String `tfsdk:"algo"`
	Window types.Int64  `tfsdk:"window"`
	Limit  types.Int64  `tfsdk:"limit"`
	Keys   types.List   `tfsdk:"keys"`
	Action types.String `tfsdk:"action"`
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
	if ref == nil && !crsRule.Active && crsRule.Action == "log" {
		return nil
	}
	c := &CRSRuleConfig{
		Active: types.BoolValue(crsRule.Active),
		Action: types.StringValue(crsRule.Action),
	}
	if (ref == nil && crsRule.Active) ||
		ref != nil && ref.Active == types.BoolNull() {
		c.Active = types.BoolNull()
	}
	return c
}

func fromClient(conf client.FirewallConfig, state FirewallConfig) (FirewallConfig, error) {
	var err error
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
			rules[i], err = fromFirewallRule(rule, stateRule)
			if err != nil {
				return cfg, err
			}
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

	return cfg, nil
}

func (f *FirewallConfig) toClient() (client.FirewallConfig, error) {
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
			mit, err := rule.Mitigate()
			if err != nil {
				return conf, err
			}
			conf.Rules = append(conf.Rules, client.FirewallRule{
				ID:             rule.ID.ValueString(),
				Name:           rule.Name.ValueString(),
				Description:    rule.Description.ValueString(),
				Active:         rule.Active.IsNull() || rule.Active.ValueBool(),
				ConditionGroup: rule.Conditions(),
				Action: client.Action{
					Mitigate: mit,
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
	return conf, nil
}

func (r *firewallConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan FirewallConfig
	diags := req.Plan.Get(ctx, &plan)
	if resp.Diagnostics.HasError() {
		return
	}

	conf, err := plan.toClient()
	if err != nil {
		diags.AddError("failed to convert plan to client", err.Error())
		return
	}

	out, err := r.client.PutFirewallConfig(ctx, conf)
	if err != nil {
		diags.AddError("failed to create firewall config", err.Error())
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := fromClient(out, plan)
	if err != nil {
		diags.AddError("failed to read created firewall config", err.Error())
		return
	}
	diags = resp.State.Set(ctx, cfg)
	resp.Diagnostics.Append(diags...)
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
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	cfg, err := fromClient(out, state)
	if err != nil {
		diags.AddError("failed to read firewall config", err.Error())
		return
	}
	diags = resp.State.Set(ctx, cfg)
	resp.Diagnostics.Append(diags...)
}

func (r *firewallConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallConfig
	diags := req.Plan.Get(ctx, &plan)
	if resp.Diagnostics.HasError() {
		return
	}

	conf, err := plan.toClient()
	if err != nil {
		diags.AddError("failed to convert plan to client", err.Error())
		return
	}

	out, err := r.client.PutFirewallConfig(ctx, conf)
	if err != nil {
		diags.AddError("failed to create firewall config", err.Error())
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := fromClient(out, plan)
	if err != nil {
		diags.AddError("failed to read updated firewall config", err.Error())
		return
	}
	diags = resp.State.Set(ctx, cfg)
	resp.Diagnostics.Append(diags...)
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
	conf, err := fromClient(out, FirewallConfig{
		ProjectID: types.StringValue(projectID),
		TeamID:    types.StringValue(out.TeamID), // use output teamID if not provided on import
	})
	if err != nil {
		resp.Diagnostics.AddError("failed to read firewall config", err.Error())
		return
	}
	tflog.Info(ctx, "imported firewall config", map[string]interface{}{
		"team_id":    conf.TeamID.ValueString(),
		"project_id": conf.ProjectID.ValueString(),
	})
	diags := resp.State.Set(ctx, conf)
	resp.Diagnostics.Append(diags...)
}
