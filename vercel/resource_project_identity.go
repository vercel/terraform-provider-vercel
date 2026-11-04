package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithIdentity = &projectResource{}

type projectIdentity struct {
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
}

func (r *projectResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"project_id": identityschema.StringAttribute{RequiredForImport: true, Description: "The stable project ID."},
		"team_id":    identityschema.StringAttribute{OptionalForImport: true, Description: "The owning team ID. Omit to use the provider's default team or personal account."},
	}}
}

func setProjectIdentity(ctx context.Context, identity *tfsdk.ResourceIdentity, project Project) diag.Diagnostics {
	if identity == nil {
		return nil
	}
	return identity.Set(ctx, projectIdentity{ProjectID: project.ID, TeamID: project.TeamID})
}
