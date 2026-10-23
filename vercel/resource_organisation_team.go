package vercel

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &organisationTeamResource{}
	_ resource.ResourceWithConfigure   = &organisationTeamResource{}
	_ resource.ResourceWithImportState = &organisationTeamResource{}
)

type organisationTeamResource struct{ client *client.Client }
type organisationTeam struct {
	ID             types.String `tfsdk:"id"`
	OrganisationID types.String `tfsdk:"organisation_id"`
	Name           types.String `tfsdk:"name"`
	Slug           types.String `tfsdk:"slug"`
	BillingPlan    types.String `tfsdk:"billing_plan"`
}

func newOrganisationTeamResource() resource.Resource { return &organisationTeamResource{} }

func (r *organisationTeamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organisation_team"
}

func (r *organisationTeamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *organisationTeamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a team within an existing Vercel organisation. The organisation must already have Enterprise Flex billing configured. Use the returned id with vercel_team_config and other team-scoped resources. When using vercel_team_config, reference this resource's name and slug so both resources stay consistent. Requires a user access token with organisation administrator permissions.\n\nDestroying this resource deletes the team, including its projects; it does not detach it. The API currently refuses self-service deletion of Enterprise child teams. Deletion failures retain the resource in state. Consider lifecycle { prevent_destroy = true } for teams that should not be deleted. Changing organisation_id or billing_plan requires replacement, which is also subject to these deletion restrictions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the created Vercel team.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organisation_id": schema.StringAttribute{
				Description:   "The ID of the existing organisation that owns the team.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "The display name of the team.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 256), stringvalidator.RegexMatches(regexp.MustCompile(`^[À-žA-Za-z0-9_'-](?:[ À-žA-Za-z0-9_'-]*[À-žA-Za-z0-9_'-])?$`), "must contain letters, numbers, spaces, underscores, apostrophes or dashes, without leading or trailing spaces")},
			},
			"slug": schema.StringAttribute{
				Description: "The team's slug, used in its dashboard URL.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 48), stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`), "must contain lowercase letters, numbers or dashes and start and end with a letter or number")},
			},
			"billing_plan": schema.StringAttribute{
				Description:   "The child team's billing profile: enterprise for internal teams using the organisation's Flex commitment, or platform for tenant teams. This is separate from a standalone team's billing plan.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("enterprise", "platform")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *organisationTeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organisationTeam
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.CreateOrganisationTeam(ctx, plan.OrganisationID.ValueString(), plan.Name.ValueString(), plan.Slug.ValueString(), plan.BillingPlan.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating Organisation Team", "Could not create team: "+err.Error())
		return
	}
	// The create response contains the link, but no name or slug. Avoid an immediate
	// list read because the organisation's team index is eventually consistent.
	plan.ID = types.StringValue(out.TeamID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organisationTeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organisationTeam
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetOrganisationTeam(ctx, state.OrganisationID.ValueString(), state.ID.ValueString())
	// The list endpoint uses an eventually consistent index. Do not forget a newly
	// created team on the first empty result; actual API 404s are not retried.
	for attempt := 0; attempt < 10; attempt++ {
		var apiErr client.APIError
		if !errors.As(err, &apiErr) || apiErr.Code != "organisation_team_not_found" {
			break
		}
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Error reading Organisation Team", ctx.Err().Error())
			return
		case <-time.After(time.Second):
		}
		out, err = r.client.GetOrganisationTeam(ctx, state.OrganisationID.ValueString(), state.ID.ValueString())
	}

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading Organisation Team", "Could not read team: "+err.Error())
		return
	}
	state.Name = types.StringValue(out.Name)
	state.Slug = types.StringValue(out.Slug)
	state.BillingPlan = types.StringValue(out.BillingPlan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organisationTeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organisationTeam
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateTeam(ctx, client.UpdateTeamRequest{TeamID: plan.ID.ValueString(), Name: plan.Name.ValueString(), Slug: plan.Slug.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Organisation Team", "Could not update team: "+err.Error())
		return
	}
	plan.Name = types.StringValue(out.Name)
	plan.Slug = types.StringValue(out.Slug)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organisationTeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organisationTeam
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteOrganisationTeam(ctx, state.OrganisationID.ValueString(), state.ID.ValueString())
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error deleting Organisation Team", "Could not delete team. Enterprise child teams cannot currently be deleted through the API. The team remains managed in Terraform state: "+err.Error())
	}
}

func (r *organisationTeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Error importing Organisation Team", "Expected an import ID in the format organisation_id/team_id.")
		return
	}
	rootTeamID, err := r.client.GetOrganisationRootTeamID(ctx, parts[0])
	if err != nil {
		resp.Diagnostics.AddError("Error importing Organisation Team", "Could not read organisation: "+err.Error())
		return
	}
	if rootTeamID == parts[1] {
		resp.Diagnostics.AddError("Error importing Organisation Team", "The organisation's root team cannot be imported as a child team. Use vercel_team_config to manage its settings.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organisation_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
