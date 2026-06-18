package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
