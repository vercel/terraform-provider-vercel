package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProjectMembersUsesModifyPlanForSetIdentityCorrelation(t *testing.T) {
	res := newProjectMembersResource()
	if _, ok := res.(resource.ResourceWithModifyPlan); !ok {
		t.Fatal("project members must implement ResourceWithModifyPlan to correlate existing members inside the members set before apply")
	}
}

func TestProjectMembersSetIdentityFieldsDoNotUseStateForUnknown(t *testing.T) {
	res := newProjectMembersResource()
	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned diagnostics: %v", resp.Diagnostics)
	}

	membersAttr, ok := resp.Schema.Attributes["members"].(rschema.SetNestedAttribute)
	if !ok {
		t.Fatalf("members attribute has type %T, want schema.SetNestedAttribute", resp.Schema.Attributes["members"])
	}

	for _, name := range []string{"user_id", "email", "username"} {
		attr, ok := membersAttr.NestedObject.Attributes[name].(rschema.StringAttribute)
		if !ok {
			t.Fatalf("members.%s has type %T, want schema.StringAttribute", name, membersAttr.NestedObject.Attributes[name])
		}
		if len(attr.PlanModifiers) != 0 {
			t.Fatalf("members.%s has %d plan modifier(s); identity fields inside a set must be correlated in ModifyPlan instead of using UseStateForUnknown-style modifiers", name, len(attr.PlanModifiers))
		}
	}
}

func TestProjectMembersModifyPlanCorrelatesExistingSetMemberIdentity(t *testing.T) {
	ctx := context.Background()
	res := &projectMembersResource{}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	stateMember := ProjectMemberItem{
		UserID:   types.StringValue("usr_123"),
		Email:    types.StringValue("doug+test2@vercel.com"),
		Username: types.StringValue("doug"),
		Role:     types.StringValue("PROJECT_VIEWER"),
	}
	plannedExistingMember := ProjectMemberItem{
		UserID:   types.StringUnknown(),
		Email:    types.StringValue("doug+test2@vercel.com"),
		Username: types.StringUnknown(),
		Role:     types.StringValue("PROJECT_DEVELOPER"),
	}
	plannedNewMember := ProjectMemberItem{
		UserID:   types.StringUnknown(),
		Email:    types.StringValue("doug+test3@vercel.com"),
		Username: types.StringUnknown(),
		Role:     types.StringValue("PROJECT_VIEWER"),
	}

	state := ProjectMembersModel{
		ID:        types.StringValue("prj_123"),
		TeamID:    types.StringValue("team_123"),
		ProjectID: types.StringValue("prj_123"),
		Members:   types.SetValueMust(memberAttrType, []attr.Value{projectMemberItemAttrValue(stateMember)}),
	}
	plan := ProjectMembersModel{
		ID:        types.StringValue("prj_123"),
		TeamID:    types.StringValue("team_123"),
		ProjectID: types.StringValue("prj_123"),
		Members: types.SetValueMust(memberAttrType, []attr.Value{
			projectMemberItemAttrValue(plannedExistingMember),
			projectMemberItemAttrValue(plannedNewMember),
		}),
	}

	priorState := tfsdk.State{Schema: schemaResp.Schema}
	diags := priorState.Set(ctx, state)
	if diags.HasError() {
		t.Fatalf("State.Set() returned diagnostics: %v", diags)
	}

	plannedState := tfsdk.Plan{Schema: schemaResp.Schema}
	diags = plannedState.Set(ctx, plan)
	if diags.HasError() {
		t.Fatalf("Plan.Set() returned diagnostics: %v", diags)
	}

	req := resource.ModifyPlanRequest{
		State: priorState,
		Plan:  plannedState,
	}
	resp := &resource.ModifyPlanResponse{
		Plan: plannedState,
	}

	res.ModifyPlan(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan() returned diagnostics: %v", resp.Diagnostics)
	}

	var modified ProjectMembersModel
	diags = resp.Plan.Get(ctx, &modified)
	if diags.HasError() {
		t.Fatalf("Plan.Get() returned diagnostics: %v", diags)
	}

	members, diags := modified.members(ctx)
	if diags.HasError() {
		t.Fatalf("members() returned diagnostics: %v", diags)
	}

	for _, member := range members {
		switch member.Email.ValueString() {
		case "doug+test2@vercel.com":
			if got := member.UserID.ValueString(); got != "usr_123" {
				t.Fatalf("existing member user_id = %q, want usr_123", got)
			}
			if got := member.Username.ValueString(); got != "doug" {
				t.Fatalf("existing member username = %q, want doug", got)
			}
			if got := member.Role.ValueString(); got != "PROJECT_DEVELOPER" {
				t.Fatalf("existing member role = %q, want PROJECT_DEVELOPER", got)
			}
		case "doug+test3@vercel.com":
			if !member.UserID.IsUnknown() {
				t.Fatalf("new member user_id = %v, want unknown", member.UserID)
			}
			if !member.Username.IsUnknown() {
				t.Fatalf("new member username = %v, want unknown", member.Username)
			}
		default:
			t.Fatalf("unexpected member email %q", member.Email.ValueString())
		}
	}
}
