package vercel

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type customAlertQuery struct {
	Metrics  map[string]customAlertQueryMetric `tfsdk:"metrics" json:"metrics"`
	Formulas map[string]string                 `tfsdk:"formulas" json:"formulas,omitempty"`
	Outputs  []string                          `tfsdk:"outputs" json:"outputs"`
	GroupBy  []string                          `tfsdk:"group_by" json:"groupBy,omitempty"`
	Filter   *string                           `tfsdk:"filter" json:"filter,omitempty"`
}

type customAlertQueryMetric struct {
	Metric      string   `tfsdk:"metric" json:"metric"`
	Aggregation string   `tfsdk:"aggregation" json:"aggregation"`
	Per         *string  `tfsdk:"per" json:"per,omitempty"`
	Normalize   *string  `tfsdk:"normalize" json:"normalize,omitempty"`
	Dimensions  []string `tfsdk:"dimensions" json:"dimensions,omitempty"`
	Filter      *string  `tfsdk:"filter" json:"filter,omitempty"`
}

var customAlertQueryMetricAttrTypes = map[string]attr.Type{
	"metric": types.StringType, "aggregation": types.StringType,
	"per": types.StringType, "normalize": types.StringType,
	"dimensions": types.ListType{ElemType: types.StringType}, "filter": types.StringType,
}
var customAlertQueryAttrTypes = map[string]attr.Type{
	"metrics":  types.MapType{ElemType: types.ObjectType{AttrTypes: customAlertQueryMetricAttrTypes}},
	"formulas": types.MapType{ElemType: types.StringType},
	"outputs":  types.ListType{ElemType: types.StringType},
	"group_by": types.ListType{ElemType: types.StringType}, "filter": types.StringType,
}

func customAlertQueryAttribute() schema.SingleNestedAttribute {
	alias := stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`), "Must be a metric alias containing letters, digits, and underscores, starting with a letter or underscore.")
	filter := schema.StringAttribute{Optional: true, MarkdownDescription: "KQL filter. Syntax and metric dimensions are validated by the API.", Validators: []validator.String{stringvalidator.LengthBetween(1, 2048)}}
	return schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "The structured Alerts v3 query. Use one metric without formulas, or two metrics with a division formula named `formula`. Discover supported metrics with `vc metrics schema <metric-or-prefix>`.",
		Validators:          []validator.Object{customAlertQueryValidator{}},
		Attributes: map[string]schema.Attribute{
			"metrics": schema.MapNestedAttribute{
				Required: true, MarkdownDescription: "One or two metric selections keyed by caller-chosen aliases. The alias `formula` is reserved.",
				Validators: []validator.Map{mapvalidator.SizeBetween(1, 2), mapvalidator.KeysAre(alias, stringvalidator.NoneOf("formula"))},
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"metric":      schema.StringAttribute{Required: true, MarkdownDescription: "Semantic metric ID or custom metric name. Availability is validated by the API.", Validators: []validator.String{stringvalidator.LengthAtLeast(1)}},
					"aggregation": schema.StringAttribute{Required: true, MarkdownDescription: "Metric aggregation. Supported combinations depend on the metric.", Validators: []validator.String{stringvalidator.OneOf("count", "sum", "avg", "min", "max", "p50", "p75", "p90", "p95", "p99", "stddev", "unique")}},
					"per":         schema.StringAttribute{Optional: true, MarkdownDescription: "Set to `second` for a per-second rate. Only supported for count or sum; cannot be combined with normalize.", Validators: []validator.String{stringvalidator.OneOf("second")}},
					"normalize":   schema.StringAttribute{Optional: true, MarkdownDescription: "Set to `percent` for percentage normalization. Only supported for count or sum; cannot be combined with per.", Validators: []validator.String{stringvalidator.OneOf("percent")}},
					"dimensions":  schema.ListAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "Dimensions required by the unique aggregation.", Validators: []validator.List{listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1))}},
					"filter":      filter,
				}},
			},
			"formulas": schema.MapAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "Optional division formula, keyed by `formula`, referencing the two metric aliases, for example `errors / requests`.", Validators: []validator.Map{mapvalidator.SizeBetween(1, 1), mapvalidator.KeysAre(stringvalidator.OneOf("formula")), mapvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1))}},
			"outputs":  schema.ListAttribute{Required: true, ElementType: types.StringType, MarkdownDescription: "Exactly one output: the sole metric alias, or `formula` for a ratio.", Validators: []validator.List{listvalidator.SizeBetween(1, 1), listvalidator.ValueStringsAre(alias)}},
			"group_by": schema.ListAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "Optional grouping dimension. Exactly one dimension is supported.", Validators: []validator.List{listvalidator.SizeBetween(1, 1), listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1))}},
			"filter":   filter,
		},
	}
}

func customAlertQueryToJSON(ctx context.Context, value types.Object) (json.RawMessage, diag.Diagnostics) {
	var query customAlertQuery
	diags := value.As(ctx, &query, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}
	encoded, err := json.Marshal(query)
	if err != nil {
		diags.AddError("Invalid custom alert query", err.Error())
	}
	return encoded, diags
}

func customAlertQueryFromJSON(ctx context.Context, value json.RawMessage) (types.Object, diag.Diagnostics) {
	var query customAlertQuery
	var diags diag.Diagnostics
	if err := json.Unmarshal(value, &query); err != nil {
		diags.AddError("Invalid custom alert query response", err.Error())
		return types.ObjectNull(customAlertQueryAttrTypes), diags
	}
	return types.ObjectValueFrom(ctx, customAlertQueryAttrTypes, query)
}

type customAlertQueryValidator struct{}

func (customAlertQueryValidator) Description(context.Context) string {
	return "Requires a single metric output or a ratio of two metrics."
}
func (v customAlertQueryValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (customAlertQueryValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value, err := req.ConfigValue.ToTerraformValue(ctx)
	if err != nil || !value.IsFullyKnown() {
		return
	}
	var query customAlertQuery
	resp.Diagnostics.Append(req.ConfigValue.As(ctx, &query, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() || len(query.Outputs) != 1 {
		return
	}
	if len(query.Metrics) == 1 && len(query.Formulas) == 0 {
		if _, ok := query.Metrics[query.Outputs[0]]; !ok {
			resp.Diagnostics.AddAttributeError(req.Path.AtName("outputs"), "Invalid query output", "The output must reference the sole metric alias.")
		}
		return
	}
	operands := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*/\s*([A-Za-z_][A-Za-z0-9_]*)$`).FindStringSubmatch(query.Formulas["formula"])
	valid := len(query.Metrics) == 2 && len(query.Formulas) == 1 && query.Outputs[0] == "formula" && len(operands) == 3
	if valid {
		_, left := query.Metrics[operands[1]]
		_, right := query.Metrics[operands[2]]
		valid = left && right && operands[1] != operands[2]
	}
	if !valid {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid custom alert query", "Use one metric with its alias as the output, or exactly two metrics divided by a formula named `formula`, with `formula` as the output.")
	}
}
