package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var projectRouteConditionAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":  types.StringType,
		"key":   types.StringType,
		"value": types.StringType,
	},
}

var projectRouteTransformAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":   types.StringType,
		"op":     types.StringType,
		"target": types.StringType,
		"args":   types.StringType,
		"env": types.ListType{
			ElemType: types.StringType,
		},
	},
}

var projectRouteDefinitionAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"src":            types.StringType,
		"dest":           types.StringType,
		"headers":        types.MapType{ElemType: types.StringType},
		"case_sensitive": types.BoolType,
		"status":         types.Int64Type,
		"has": types.ListType{
			ElemType: projectRouteConditionAttrType,
		},
		"missing": types.ListType{
			ElemType: projectRouteConditionAttrType,
		},
		"transforms": types.ListType{
			ElemType: projectRouteTransformAttrType,
		},
		"respect_origin_cache_control": types.BoolType,
	},
}

var projectRouteAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"description": types.StringType,
		"enabled":     types.BoolType,
		"src_syntax":  types.StringType,
		"route_type":  types.StringType,
		"route":       projectRouteDefinitionAttrType,
	},
}

type ProjectRoutesModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	Rules     types.List   `tfsdk:"rules"`
}

type ProjectRouteResourceModel struct {
	ID          types.String           `tfsdk:"id"`
	ProjectID   types.String           `tfsdk:"project_id"`
	TeamID      types.String           `tfsdk:"team_id"`
	Name        types.String           `tfsdk:"name"`
	Description types.String           `tfsdk:"description"`
	Enabled     types.Bool             `tfsdk:"enabled"`
	SrcSyntax   types.String           `tfsdk:"src_syntax"`
	RouteType   types.String           `tfsdk:"route_type"`
	Position    ProjectRoutePosition   `tfsdk:"position"`
	Route       ProjectRouteDefinition `tfsdk:"route"`
}

type ProjectRoute struct {
	ID          types.String           `tfsdk:"id"`
	Name        types.String           `tfsdk:"name"`
	Description types.String           `tfsdk:"description"`
	Enabled     types.Bool             `tfsdk:"enabled"`
	SrcSyntax   types.String           `tfsdk:"src_syntax"`
	RouteType   types.String           `tfsdk:"route_type"`
	Route       ProjectRouteDefinition `tfsdk:"route"`
}

type ProjectRouteDefinition struct {
	Src                       types.String `tfsdk:"src"`
	Dest                      types.String `tfsdk:"dest"`
	Headers                   types.Map    `tfsdk:"headers"`
	CaseSensitive             types.Bool   `tfsdk:"case_sensitive"`
	Status                    types.Int64  `tfsdk:"status"`
	Has                       types.List   `tfsdk:"has"`
	Missing                   types.List   `tfsdk:"missing"`
	Transforms                types.List   `tfsdk:"transforms"`
	RespectOriginCacheControl types.Bool   `tfsdk:"respect_origin_cache_control"`
}

type ProjectRouteCondition struct {
	Type  types.String `tfsdk:"type"`
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type ProjectRouteTransform struct {
	Type   types.String `tfsdk:"type"`
	Op     types.String `tfsdk:"op"`
	Target types.String `tfsdk:"target"`
	Args   types.String `tfsdk:"args"`
	Env    types.List   `tfsdk:"env"`
}

type ProjectRoutePosition struct {
	Placement        types.String `tfsdk:"placement"`
	ReferenceRouteID types.String `tfsdk:"reference_route_id"`
}

func projectRoutesResourceID(teamID, projectID string) string {
	if teamID == "" {
		return projectID
	}

	return fmt.Sprintf("%s/%s", teamID, projectID)
}

func projectRoutesListValue(rules []ProjectRoute) types.List {
	values := make([]attr.Value, 0, len(rules))
	for _, rule := range rules {
		values = append(values, rule.toAttrValue())
	}

	return types.ListValueMust(projectRouteAttrType, values)
}

func (r ProjectRoute) toAttrValue() attr.Value {
	return types.ObjectValueMust(projectRouteAttrType.AttrTypes, map[string]attr.Value{
		"id":          r.ID,
		"name":        r.Name,
		"description": r.Description,
		"enabled":     r.Enabled,
		"src_syntax":  r.SrcSyntax,
		"route_type":  r.RouteType,
		"route":       r.Route.toAttrValue(),
	})
}

func (r ProjectRouteDefinition) toAttrValue() attr.Value {
	return types.ObjectValueMust(projectRouteDefinitionAttrType.AttrTypes, map[string]attr.Value{
		"src":                          r.Src,
		"dest":                         r.Dest,
		"headers":                      r.Headers,
		"case_sensitive":               r.CaseSensitive,
		"status":                       r.Status,
		"has":                          r.Has,
		"missing":                      r.Missing,
		"transforms":                   r.Transforms,
		"respect_origin_cache_control": r.RespectOriginCacheControl,
	})
}

func (c ProjectRouteCondition) toAttrValue() attr.Value {
	return types.ObjectValueMust(projectRouteConditionAttrType.AttrTypes, map[string]attr.Value{
		"type":  c.Type,
		"key":   c.Key,
		"value": c.Value,
	})
}

func (t ProjectRouteTransform) toAttrValue() attr.Value {
	return types.ObjectValueMust(projectRouteTransformAttrType.AttrTypes, map[string]attr.Value{
		"type":   t.Type,
		"op":     t.Op,
		"target": t.Target,
		"args":   t.Args,
		"env":    t.Env,
	})
}

func (m ProjectRouteResourceModel) projectRoute() ProjectRoute {
	return ProjectRoute{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Enabled:     m.Enabled,
		SrcSyntax:   m.SrcSyntax,
		RouteType:   m.RouteType,
		Route:       m.Route,
	}
}

func projectRouteResourceModelFromRoute(projectID, teamID string, route ProjectRoute, position ProjectRoutePosition) ProjectRouteResourceModel {
	return ProjectRouteResourceModel{
		ID:          route.ID,
		ProjectID:   types.StringValue(projectID),
		TeamID:      toTeamID(teamID),
		Name:        route.Name,
		Description: route.Description,
		Enabled:     route.Enabled,
		SrcSyntax:   route.SrcSyntax,
		RouteType:   route.RouteType,
		Position:    position,
		Route:       route.Route,
	}
}

func (p ProjectRoutePosition) isNull() bool {
	return p.Placement.IsNull() && p.ReferenceRouteID.IsNull()
}

func (p ProjectRoutePosition) toClientPosition() *client.ProjectRoutingRulePosition {
	if p.isNull() || p.Placement.IsUnknown() {
		return nil
	}

	request := &client.ProjectRoutingRulePosition{
		Placement: p.Placement.ValueString(),
	}
	if !p.ReferenceRouteID.IsNull() && !p.ReferenceRouteID.IsUnknown() {
		request.ReferenceID = p.ReferenceRouteID.ValueStringPointer()
	}
	return request
}

func (r ProjectRoute) toClientInput(ctx context.Context) (client.ProjectRoutingRuleInput, diag.Diagnostics) {
	route, diags := r.Route.toClientRequest(ctx)
	if diags.HasError() {
		return client.ProjectRoutingRuleInput{}, diags
	}

	request := client.ProjectRoutingRuleInput{
		Name:  r.Name.ValueString(),
		Route: route,
	}

	if !r.Description.IsNull() && !r.Description.IsUnknown() {
		request.Description = r.Description.ValueStringPointer()
	}
	if !r.Enabled.IsNull() && !r.Enabled.IsUnknown() {
		request.Enabled = r.Enabled.ValueBoolPointer()
	}
	if !r.SrcSyntax.IsNull() && !r.SrcSyntax.IsUnknown() {
		request.SrcSyntax = r.SrcSyntax.ValueStringPointer()
	}

	return request, diags
}

func (r ProjectRouteDefinition) toClientRequest(ctx context.Context) (client.ProjectRouteDefinition, diag.Diagnostics) {
	var diags diag.Diagnostics

	request := client.ProjectRouteDefinition{
		Src: r.Src.ValueString(),
	}

	if !r.Dest.IsNull() && !r.Dest.IsUnknown() {
		request.Dest = r.Dest.ValueStringPointer()
	}
	if !r.Headers.IsNull() && !r.Headers.IsUnknown() {
		var headers map[string]string
		diags = r.Headers.ElementsAs(ctx, &headers, false)
		if diags.HasError() {
			return request, diags
		}
		request.Headers = headers
	}
	if !r.CaseSensitive.IsNull() && !r.CaseSensitive.IsUnknown() {
		request.CaseSensitive = r.CaseSensitive.ValueBoolPointer()
	}
	if !r.Status.IsNull() && !r.Status.IsUnknown() {
		request.Status = r.Status.ValueInt64Pointer()
	}

	request.Has, diags = projectRouteConditionsToClient(ctx, r.Has)
	if diags.HasError() {
		return request, diags
	}

	request.Missing, diags = projectRouteConditionsToClient(ctx, r.Missing)
	if diags.HasError() {
		return request, diags
	}

	request.Transforms, diags = projectRouteTransformsToClient(ctx, r.Transforms)
	if diags.HasError() {
		return request, diags
	}

	if !r.RespectOriginCacheControl.IsNull() && !r.RespectOriginCacheControl.IsUnknown() {
		request.RespectOriginCacheControl = r.RespectOriginCacheControl.ValueBoolPointer()
	}

	return request, nil
}

func projectRouteConditionsToClient(ctx context.Context, conditionList types.List) ([]client.ProjectRouteCondition, diag.Diagnostics) {
	if conditionList.IsNull() || conditionList.IsUnknown() {
		return nil, nil
	}

	var conditions []ProjectRouteCondition
	diags := conditionList.ElementsAs(ctx, &conditions, false)
	if diags.HasError() {
		return nil, diags
	}

	requests := make([]client.ProjectRouteCondition, 0, len(conditions))
	for _, condition := range conditions {
		request := client.ProjectRouteCondition{
			Type: condition.Type.ValueString(),
		}
		if !condition.Key.IsNull() && !condition.Key.IsUnknown() {
			request.Key = condition.Key.ValueStringPointer()
		}
		if !condition.Value.IsNull() && !condition.Value.IsUnknown() {
			request.Value = condition.Value.ValueStringPointer()
		}
		requests = append(requests, request)
	}

	return requests, nil
}

func projectRouteTransformsToClient(ctx context.Context, transformList types.List) ([]client.ProjectRouteTransform, diag.Diagnostics) {
	if transformList.IsNull() || transformList.IsUnknown() {
		return nil, nil
	}

	var transforms []ProjectRouteTransform
	diags := transformList.ElementsAs(ctx, &transforms, false)
	if diags.HasError() {
		return nil, diags
	}

	requests := make([]client.ProjectRouteTransform, 0, len(transforms))
	for _, transform := range transforms {
		request := client.ProjectRouteTransform{
			Type: transform.Type.ValueString(),
			Op:   transform.Op.ValueString(),
		}

		if !transform.Target.IsNull() && !transform.Target.IsUnknown() {
			value, err := parseJSONString(transform.Target.ValueString())
			if err != nil {
				diags.AddError("Error preparing route transform target", err.Error())
				return nil, diags
			}
			request.Target = value
		}

		if !transform.Args.IsNull() && !transform.Args.IsUnknown() {
			value, err := parseJSONString(transform.Args.ValueString())
			if err != nil {
				diags.AddError("Error preparing route transform args", err.Error())
				return nil, diags
			}
			request.Args = value
		}

		if !transform.Env.IsNull() && !transform.Env.IsUnknown() {
			var env []string
			diags = transform.Env.ElementsAs(ctx, &env, false)
			if diags.HasError() {
				return nil, diags
			}
			request.Env = env
		}

		requests = append(requests, request)
	}

	return requests, nil
}

func readLiveProjectRoutingRules(ctx context.Context, vercelClient *client.Client, projectID, teamID string) (client.ProjectRoutingRulesResponse, error) {
	versions, err := vercelClient.GetProjectRouteVersions(ctx, projectID, teamID)
	if err != nil {
		return client.ProjectRoutingRulesResponse{}, err
	}

	liveVersionID := ""
	for _, version := range versions {
		if version.IsLive {
			liveVersionID = version.ID
			break
		}
	}

	return vercelClient.GetProjectRoutingRules(ctx, projectID, teamID, liveVersionID)
}

func convertResponseToProjectRoutes(ctx context.Context, response client.ProjectRoutingRulesResponse, projectID, teamID string, preferredRules []ProjectRoute) (ProjectRoutesModel, diag.Diagnostics) {
	preferredByID := map[string]ProjectRoute{}
	for _, rule := range preferredRules {
		if rule.ID.IsNull() || rule.ID.IsUnknown() || rule.ID.ValueString() == "" {
			continue
		}
		preferredByID[rule.ID.ValueString()] = rule
	}

	rules := make([]ProjectRoute, 0, len(response.Routes))
	for _, apiRule := range response.Routes {
		rule, diags := projectRouteFromAPI(ctx, apiRule, preferredByID[apiRule.ID])
		if diags.HasError() {
			return ProjectRoutesModel{}, diags
		}
		rules = append(rules, rule)
	}

	return ProjectRoutesModel{
		ID:        types.StringValue(projectRoutesResourceID(teamID, projectID)),
		ProjectID: types.StringValue(projectID),
		TeamID:    toTeamID(teamID),
		Rules:     projectRoutesListValue(rules),
	}, nil
}

func readProjectRoutes(ctx context.Context, vercelClient *client.Client, projectID, teamID string, preferredRules []ProjectRoute) (ProjectRoutesModel, diag.Diagnostics, error) {
	response, err := readLiveProjectRoutingRules(ctx, vercelClient, projectID, teamID)
	if err != nil {
		return ProjectRoutesModel{}, nil, err
	}

	result, diags := convertResponseToProjectRoutes(ctx, response, projectID, vercelClient.TeamID(teamID), preferredRules)
	return result, diags, nil
}

func readProjectRoute(ctx context.Context, vercelClient *client.Client, routeID, projectID, teamID string, preferredRoute ProjectRoute, preferredPosition ProjectRoutePosition) (ProjectRouteResourceModel, diag.Diagnostics, error) {
	response, err := readLiveProjectRoutingRules(ctx, vercelClient, projectID, teamID)
	if err != nil {
		return ProjectRouteResourceModel{}, nil, err
	}

	for _, apiRule := range response.Routes {
		if apiRule.ID != routeID {
			continue
		}

		route, diags := projectRouteFromAPI(ctx, apiRule, preferredRoute)
		if diags.HasError() {
			return ProjectRouteResourceModel{}, diags, nil
		}

		return projectRouteResourceModelFromRoute(projectID, vercelClient.TeamID(teamID), route, preferredPosition), nil, nil
	}

	return ProjectRouteResourceModel{}, nil, client.APIError{
		StatusCode: 404,
		Code:       "not_found",
		Message:    fmt.Sprintf("Could not find project route %s", routeID),
	}
}

func projectRouteFromAPI(ctx context.Context, apiRule client.ProjectRoutingRule, preferredRule ProjectRoute) (ProjectRoute, diag.Diagnostics) {
	route, diags := projectRouteDefinitionFromAPI(ctx, apiRule, preferredRule.Route)
	if diags.HasError() {
		return ProjectRoute{}, diags
	}

	return ProjectRoute{
		ID:          types.StringValue(apiRule.ID),
		Name:        types.StringValue(apiRule.Name),
		Description: stringValueOrNull(apiRule.Description),
		Enabled:     boolValueOrDefault(apiRule.Enabled, preferredRule.Enabled, true),
		SrcSyntax:   stringValueOrNull(apiRule.SrcSyntax),
		RouteType:   stringValueOrNull(apiRule.RouteType),
		Route:       route,
	}, nil
}

func projectRouteDefinitionFromAPI(ctx context.Context, apiRule client.ProjectRoutingRule, preferredRoute ProjectRouteDefinition) (ProjectRouteDefinition, diag.Diagnostics) {
	var diags diag.Diagnostics

	src := apiRule.Route.Src
	if apiRule.RawSrc != nil && *apiRule.RawSrc != "" {
		src = *apiRule.RawSrc
	}

	dest := apiRule.Route.Dest
	if apiRule.RawDest != nil {
		dest = apiRule.RawDest
	}

	headers, headerDiags := mapValueOrNull(ctx, apiRule.Route.Headers, preferredRoute.Headers)
	diags.Append(headerDiags...)
	if diags.HasError() {
		return ProjectRouteDefinition{}, diags
	}

	hasConditions, conditionDiags := projectRouteConditionsFromAPI(apiRule.Route.Has, preferredRoute.Has)
	diags.Append(conditionDiags...)
	if diags.HasError() {
		return ProjectRouteDefinition{}, diags
	}

	missingConditions, missingDiags := projectRouteConditionsFromAPI(apiRule.Route.Missing, preferredRoute.Missing)
	diags.Append(missingDiags...)
	if diags.HasError() {
		return ProjectRouteDefinition{}, diags
	}

	transforms, transformDiags := projectRouteTransformsFromAPI(ctx, apiRule.Route.Transforms, preferredRoute.Transforms)
	diags.Append(transformDiags...)
	if diags.HasError() {
		return ProjectRouteDefinition{}, diags
	}

	return ProjectRouteDefinition{
		Src:                       types.StringValue(src),
		Dest:                      stringValueOrNull(dest),
		Headers:                   headers,
		CaseSensitive:             boolValueOrNull(apiRule.Route.CaseSensitive, preferredRoute.CaseSensitive),
		Status:                    int64ValueOrNull(apiRule.Route.Status),
		Has:                       hasConditions,
		Missing:                   missingConditions,
		Transforms:                transforms,
		RespectOriginCacheControl: boolValueOrNull(apiRule.Route.RespectOriginCacheControl, preferredRoute.RespectOriginCacheControl),
	}, nil
}

func projectRouteConditionsFromAPI(apiConditions []client.ProjectRouteCondition, preferredList types.List) (types.List, diag.Diagnostics) {
	if len(apiConditions) == 0 {
		return preserveEmptyList(preferredList, projectRouteConditionAttrType), nil
	}

	values := make([]attr.Value, 0, len(apiConditions))
	for _, apiCondition := range apiConditions {
		values = append(values, ProjectRouteCondition{
			Type:  types.StringValue(apiCondition.Type),
			Key:   stringValueOrNull(apiCondition.Key),
			Value: stringValueOrNull(apiCondition.Value),
		}.toAttrValue())
	}

	return types.ListValueMust(projectRouteConditionAttrType, values), nil
}

func projectRouteTransformsFromAPI(ctx context.Context, apiTransforms []client.ProjectRouteTransform, preferredList types.List) (types.List, diag.Diagnostics) {
	if len(apiTransforms) == 0 {
		return preserveEmptyList(preferredList, projectRouteTransformAttrType), nil
	}

	preferredTransforms := map[int]ProjectRouteTransform{}
	if !preferredList.IsNull() && !preferredList.IsUnknown() {
		var decodedTransforms []ProjectRouteTransform
		diags := preferredList.ElementsAs(ctx, &decodedTransforms, false)
		if !diags.HasError() {
			for i, transform := range decodedTransforms {
				preferredTransforms[i] = transform
			}
		}
	}

	values := make([]attr.Value, 0, len(apiTransforms))
	for i, apiTransform := range apiTransforms {
		preferredTransform := preferredTransforms[i]
		target, err := jsonStringValueFromResponse(preferredTransform.Target, apiTransform.Target)
		if err != nil {
			var diags diag.Diagnostics
			diags.AddError("Error reading route transform target", err.Error())
			return types.ListNull(projectRouteTransformAttrType), diags
		}

		args, err := jsonStringValueFromResponse(preferredTransform.Args, apiTransform.Args)
		if err != nil {
			var diags diag.Diagnostics
			diags.AddError("Error reading route transform args", err.Error())
			return types.ListNull(projectRouteTransformAttrType), diags
		}

		env, diags := stringListValueOrNull(ctx, apiTransform.Env, preferredTransform.Env)
		if diags.HasError() {
			return types.ListNull(projectRouteTransformAttrType), diags
		}

		values = append(values, ProjectRouteTransform{
			Type:   types.StringValue(apiTransform.Type),
			Op:     types.StringValue(apiTransform.Op),
			Target: target,
			Args:   args,
			Env:    env,
		}.toAttrValue())
	}

	return types.ListValueMust(projectRouteTransformAttrType, values), nil
}

func mapValueOrNull(ctx context.Context, value map[string]string, preferredMap types.Map) (types.Map, diag.Diagnostics) {
	if len(value) == 0 {
		if preferredMap.IsNull() || preferredMap.IsUnknown() {
			return types.MapNull(types.StringType), nil
		}
	}

	return types.MapValueFrom(ctx, types.StringType, value)
}

func preserveEmptyList(preferredList types.List, elementType types.ObjectType) types.List {
	if preferredList.IsNull() || preferredList.IsUnknown() {
		return types.ListNull(elementType)
	}

	return types.ListValueMust(elementType, []attr.Value{})
}

func stringListValueOrNull(ctx context.Context, values []string, preferredList types.List) (types.List, diag.Diagnostics) {
	if len(values) == 0 {
		if preferredList.IsNull() || preferredList.IsUnknown() {
			return types.ListNull(types.StringType), nil
		}
	}

	return types.ListValueFrom(ctx, types.StringType, values)
}

func stringValueOrNull(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}

	return types.StringValue(*value)
}

func int64ValueOrNull(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*value)
}

func boolValueOrDefault(value *bool, preferredValue types.Bool, defaultValue bool) types.Bool {
	if value == nil {
		if !preferredValue.IsNull() && !preferredValue.IsUnknown() && !preferredValue.ValueBool() {
			return preferredValue
		}
		return types.BoolValue(defaultValue)
	}

	return types.BoolValue(*value)
}

func boolValueOrNull(value *bool, preferredValue types.Bool) types.Bool {
	if value == nil {
		if !preferredValue.IsNull() && !preferredValue.IsUnknown() && !preferredValue.ValueBool() {
			return preferredValue
		}
		return types.BoolNull()
	}

	return types.BoolValue(*value)
}

func parseJSONString(value string) (any, error) {
	var parsed any
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return nil, fmt.Errorf("unable to parse JSON value %q: %w", value, err)
	}
	return parsed, nil
}

func normalizeJSONValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var normalized any
	if err := json.Unmarshal(payload, &normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

func jsonStringValueFromResponse(preferredValue types.String, responseValue any) (types.String, error) {
	if responseValue == nil {
		return types.StringNull(), nil
	}

	normalizedResponse, err := normalizeJSONValue(responseValue)
	if err != nil {
		return types.StringNull(), err
	}

	if !preferredValue.IsNull() && !preferredValue.IsUnknown() {
		normalizedPreferred, err := parseJSONString(preferredValue.ValueString())
		if err == nil {
			normalizedPreferred, err = normalizeJSONValue(normalizedPreferred)
			if err == nil && reflect.DeepEqual(normalizedPreferred, normalizedResponse) {
				return preferredValue, nil
			}
		}
	}

	payload, err := json.Marshal(normalizedResponse)
	if err != nil {
		return types.StringNull(), err
	}

	return types.StringValue(string(payload)), nil
}
