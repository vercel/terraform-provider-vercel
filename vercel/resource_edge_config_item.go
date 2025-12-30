package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"math/big"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                     = &edgeConfigItemResource{}
	_ resource.ResourceWithConfigure        = &edgeConfigItemResource{}
	_ resource.ResourceWithConfigValidators = &edgeConfigItemResource{}
)

func newEdgeConfigItemResource() resource.Resource {
	return &edgeConfigItemResource{}
}

type edgeConfigItemResource struct {
	client *client.Client
}

func (r *edgeConfigItemResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_config_item"
}

func (r *edgeConfigItemResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *edgeConfigItemResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("value"),
			path.MatchRoot("value_json"),
		),
	}
}

// Schema returns the schema information for an edgeConfigToken resource.
func (r *edgeConfigItemResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Edge Config Item.

An Edge Config is a global data store that enables experimentation with feature flags, A/B testing, critical redirects, and more.

An Edge Config Item is a value within an Edge Config.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this resource. Format: edge_config_id/key.",
				Computed:    true,
			},
			"edge_config_id": schema.StringAttribute{
				Description:   "The ID of the Edge Config store.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"key": schema.StringAttribute{
				Description:   "The name of the key you want to add to or update within your Edge Config.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						// Alphanumeric, underscore, dash; 1-256 chars
						// Matches Vercel API constraints for item keys
						// ^[A-Za-z0-9_-]{1,256}$
						regexpMustCompile("^[A-Za-z0-9_-]{1,256}$"),
						"Key must be 1-256 chars: letters, numbers, '_' or '-'",
					),
				},
			},
			"value": schema.StringAttribute{
				Description:   "The value you want to assign to the key when using a string.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"value_json": schema.DynamicAttribute{
				Description:   "Structured JSON value to assign to the key (object/array/number/bool/null).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Dynamic{dynamicplanmodifier.RequiresReplaceIfConfigured()},
			},
		},
	}
}

type EdgeConfigItem struct {
	ID           types.String  `tfsdk:"id"`
	EdgeConfigID types.String  `tfsdk:"edge_config_id"`
	TeamID       types.String  `tfsdk:"team_id"`
	Key          types.String  `tfsdk:"key"`
	Value        types.String  `tfsdk:"value"`
	ValueJSON    types.Dynamic `tfsdk:"value_json"`
}

// helper: compile regex once
var reCache = map[string]*regexp.Regexp{}

func regexpMustCompile(expr string) *regexp.Regexp {
	if re, ok := reCache[expr]; ok {
		return re
	}
	re := regexp.MustCompile(expr)
	reCache[expr] = re
	return re
}

// buildJSONRaw constructs json.RawMessage from either a string or Dynamic value
func buildJSONRaw(ctx context.Context, val types.String, dyn types.Dynamic) (json.RawMessage, error) {
	if !val.IsNull() && !val.IsUnknown() {
		b, err := json.Marshal(val.ValueString())
		return json.RawMessage(b), err
	}
	if !dyn.IsNull() && !dyn.IsUnknown() {
		// Handle underlying unknown
		if dyn.IsUnderlyingValueUnknown() {
			return nil, fmt.Errorf("value_json is unknown")
		}
		// Handle underlying null
		if dyn.IsUnderlyingValueNull() {
			return json.RawMessage("null"), nil
		}
		iface, err := attrValueToInterface(dyn.UnderlyingValue())
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(iface)
		return json.RawMessage(b), err
	}
	return nil, fmt.Errorf("either value or value_json must be provided")
}

// attrValueToInterface recursively converts an attr.Value into Go types suitable for json.Marshal
func attrValueToInterface(v attr.Value) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case types.String:
		if val.IsUnknown() {
			return nil, fmt.Errorf("string value is unknown")
		}
		if val.IsNull() {
			return nil, nil
		}
		return val.ValueString(), nil
	case types.Bool:
		if val.IsUnknown() {
			return nil, fmt.Errorf("bool value is unknown")
		}
		if val.IsNull() {
			return nil, nil
		}
		return val.ValueBool(), nil
	case types.Number:
		if val.IsUnknown() {
			return nil, fmt.Errorf("number value is unknown")
		}
		if val.IsNull() {
			return nil, nil
		}
		bf := val.ValueBigFloat()
		f64, _ := bf.Float64()
		return f64, nil
	case types.List:
		if val.IsUnknown() {
			return nil, fmt.Errorf("list value is unknown")
		}
		if val.IsNull() {
			return nil, nil
		}
		elems := val.Elements()
		arr := make([]any, len(elems))
		for i, ev := range elems {
			iv, err := attrValueToInterface(ev)
			if err != nil {
				return nil, err
			}
			arr[i] = iv
		}
		return arr, nil
	case types.Tuple:
		if val.IsUnknown() {
			return nil, fmt.Errorf("tuple value is unknown")
		}
		if val.IsNull() {
			return nil, nil
		}
		elems := val.Elements()
		arr := make([]any, len(elems))
		for i, ev := range elems {
			iv, err := attrValueToInterface(ev)
			if err != nil {
				return nil, err
			}
			arr[i] = iv
		}
		return arr, nil
	case types.Set:
		if val.IsUnknown() {
			return nil, fmt.Errorf("set value is unknown")
		}
		if val.IsNull() {
			return nil, nil
		}
		elems := val.Elements()
		arr := make([]any, len(elems))
		for i, ev := range elems {
			iv, err := attrValueToInterface(ev)
			if err != nil {
				return nil, err
			}
			arr[i] = iv
		}
		return arr, nil
	case types.Map:
		if val.IsUnknown() {
			return nil, fmt.Errorf("map value is unknown")
		}
		if val.IsNull() {
			return nil, nil
		}
		e := val.Elements()
		res := make(map[string]any, len(e))
		for k, ev := range e {
			iv, err := attrValueToInterface(ev)
			if err != nil {
				return nil, err
			}
			res[k] = iv
		}
		return res, nil
	case types.Object:
		if val.IsUnknown() {
			return nil, fmt.Errorf("object value is unknown")
		}
		if val.IsNull() {
			return nil, nil
		}
		attrs := val.Attributes()
		res := make(map[string]any, len(attrs))
		for k, ev := range attrs {
			iv, err := attrValueToInterface(ev)
			if err != nil {
				return nil, err
			}
			res[k] = iv
		}
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported dynamic value type")
	}
}

// interfaceToAttrValue converts a decoded JSON value (via encoding/json) into a framework attr.Value
func interfaceToAttrValue(ctx context.Context, v any) (attr.Value, error) {
	switch vv := v.(type) {
	case nil:
		return types.DynamicNull(), nil
	case string:
		return types.StringValue(vv), nil
	case bool:
		return types.BoolValue(vv), nil
	case float64:
		// JSON numbers decode to float64
		return types.NumberValue(big.NewFloat(vv)), nil
	case []any:
		elements := make([]attr.Value, len(vv))
		elemTypes := make([]attr.Type, len(vv))
		for i, ev := range vv {
			av, err := interfaceToAttrValue(ctx, ev)
			if err != nil {
				return nil, err
			}
			elements[i] = av
			elemTypes[i] = av.Type(ctx)
		}
		// Use tuple to preserve per-element types
		tuple, diags := types.TupleValue(elemTypes, elements)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to build tuple value: %v", diags)
		}
		return tuple, nil
	case map[string]any:
		attrs := make(map[string]attr.Value, len(vv))
		attrTypes := make(map[string]attr.Type, len(vv))
		for k, ev := range vv {
			av, err := interfaceToAttrValue(ctx, ev)
			if err != nil {
				return nil, err
			}
			attrs[k] = av
			// If value is dynamic (e.g., null), set dynamic type; otherwise infer from value
			if _, ok := av.(types.Dynamic); ok {
				attrTypes[k] = types.DynamicType
			} else {
				attrTypes[k] = av.Type(ctx)
			}
		}
		obj, diags := types.ObjectValue(attrTypes, attrs)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to build object value: %v", diags)
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("unsupported JSON type %T", v)
	}
}

// parseRawToState sets either Value (for strings) or ValueJSON (for structured) from API output
func parseRawToState(ctx context.Context, out client.EdgeConfigItem) (EdgeConfigItem, error) {
	res := EdgeConfigItem{
		ID:           types.StringValue(out.EdgeConfigID + "/" + out.Key),
		EdgeConfigID: types.StringValue(out.EdgeConfigID),
		TeamID:       types.StringValue(out.TeamID),
		Key:          types.StringValue(out.Key),
	}
	if len(out.Value) == 0 {
		return res, nil
	}
	// Try to unmarshal as string first for convenience
	var sv string
	if err := json.Unmarshal(out.Value, &sv); err == nil {
		res.Value = types.StringValue(sv)
		res.ValueJSON = types.DynamicNull()
		return res, nil
	}
	// Otherwise, decode arbitrary JSON into framework values and wrap in Dynamic
	var anyVal any
	if err := json.Unmarshal(out.Value, &anyVal); err != nil {
		return res, fmt.Errorf("failed to parse JSON for value_json: %w", err)
	}
	attrVal, err := interfaceToAttrValue(ctx, anyVal)
	if err != nil {
		return res, fmt.Errorf("failed to convert JSON to attr.Value: %w", err)
	}
	res.Value = types.StringNull()
	res.ValueJSON = types.DynamicValue(attrVal)
	return res, nil
}

// Create will create an edgeConfigToken within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r *edgeConfigItemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeConfigItem
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := buildJSONRaw(ctx, plan.Value, plan.ValueJSON)
	if err != nil {
		resp.Diagnostics.AddError("Invalid value for Edge Config Item", err.Error())
		return
	}

	out, err := r.client.CreateEdgeConfigItem(ctx, client.CreateEdgeConfigItemRequest{
		TeamID:       plan.TeamID.ValueString(),
		EdgeConfigID: plan.EdgeConfigID.ValueString(),
		Key:          plan.Key.ValueString(),
		Value:        raw,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Edge Config Item",
			"Could not create Edge Config Item, unexpected error: "+err.Error(),
		)
		return
	}

	result, err := parseRawToState(ctx, out)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing Edge Config Item", err.Error())
		return
	}
	tflog.Info(ctx, "created Edge Config Item", map[string]any{
		"edge_config_id": plan.EdgeConfigID.ValueString(),
		"key":            result.Key.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read edgeConfigToken information by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *edgeConfigItemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeConfigItem
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetEdgeConfigItem(ctx, client.EdgeConfigItemRequest{
		EdgeConfigID: state.EdgeConfigID.ValueString(),
		TeamID:       state.TeamID.ValueString(),
		Key:          state.Key.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config Item",
			fmt.Sprintf("Could not get Edge Config Item %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.Key.ValueString(),
				err,
			),
		)
		return
	}

	result, err := parseRawToState(ctx, out)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing Edge Config Item", err.Error())
		return
	}
	tflog.Info(ctx, "read edge config token", map[string]any{
		"edge_config_id": state.EdgeConfigID.ValueString(),
		"team_id":        state.TeamID.ValueString(),
		"key":            state.Key.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update is the same as Create
func (r *edgeConfigItemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating an Edge Config Item is not supported. If you see this error, this is a bug in the provider.",
		"Updating an Edge Config Item is not supported. If you see this error, this is a bug in the provider.",
	)
}

// Delete deletes an Edge Config Item.
func (r *edgeConfigItemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeConfigItem
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEdgeConfigItem(ctx, client.EdgeConfigItemRequest{
		TeamID:       state.TeamID.ValueString(),
		EdgeConfigID: state.EdgeConfigID.ValueString(),
		Key:          state.Key.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Edge Config Item",
			fmt.Sprintf(
				"Could not delete Edge Config Item %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.EdgeConfigID.ValueString(),
				state.Key.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted edge config token", map[string]any{
		"edge_config_id": state.EdgeConfigID.ValueString(),
		"team_id":        state.TeamID.ValueString(),
		"key":            state.Key.ValueString(),
	})
}

func (r *edgeConfigItemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, edgeConfigId, id, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Edge Config Item",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/edge_config_id/key\" or \"edge_config_id/key\"", req.ID),
		)
	}

	out, err := r.client.GetEdgeConfigItem(ctx, client.EdgeConfigItemRequest{
		EdgeConfigID: edgeConfigId,
		TeamID:       teamID,
		Key:          id,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config Item",
			fmt.Sprintf("Could not get Edge Config Item %s %s %s, unexpected error: %s",
				teamID,
				edgeConfigId,
				id,
				err,
			),
		)
		return
	}

	result, err := parseRawToState(ctx, out)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing Edge Config Item", err.Error())
		return
	}
	tflog.Info(ctx, "import edge config schema", map[string]any{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.EdgeConfigID.ValueString(),
		"key":            result.Key.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
