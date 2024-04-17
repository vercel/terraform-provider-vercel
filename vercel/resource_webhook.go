package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &webhookResource{}
	_ resource.ResourceWithConfigure = &webhookResource{}
)

func newWebhookResource() resource.Resource {
	return &webhookResource{}
}

type webhookResource struct {
	client *client.Client
}

func (r *webhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *webhookResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
A webhook is a trigger-based HTTP endpoint configured to receive HTTP POST requests through events.

When an event happens, a webhook is sent to a third-party app, which can then take appropriate action.

~> Only Pro and Enterprise teams are able to configure these webhooks at the account level.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Webhook.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Webhook should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"events": schema.SetAttribute{
				Description: "A list of the events the webhook will listen to. At least one must be present.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					stringSetItemsIn(
						"deployment.created",
						"deployment.error",
						"deployment.canceled",
						"deployment.succeeded",
						"project.created",
						"project.removed",
					),
					stringSetMinCount(1),
				},
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			"endpoint": schema.StringAttribute{
				Description:   "Webhooks events will be sent as POST request to this URL.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project_ids": schema.SetAttribute{
				Description:   "A list of project IDs that the webhook should be associated with. These projects should send events to the specified endpoint.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			"secret": schema.StringAttribute{
				Description:   "A secret value which will be provided in the `x-vercel-signature` header and can be used to verify the authenticity of the webhook. See https://vercel.com/docs/observability/webhooks-overview/webhooks-api#securing-webhooks for further details.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

type Webhook struct {
	ID         types.String `tfsdk:"id"`
	TeamID     types.String `tfsdk:"team_id"`
	Endpoint   types.String `tfsdk:"endpoint"`
	Secret     types.String `tfsdk:"secret"`
	ProjectIDs types.Set    `tfsdk:"project_ids"`
	Events     types.Set    `tfsdk:"events"`
}

func responseToWebhook(ctx context.Context, out client.Webhook) (Webhook, diag.Diagnostics) {
	projectIDs, diags := types.SetValueFrom(ctx, types.StringType, out.ProjectIDs)
	if diags.HasError() {
		return Webhook{}, diags
	}
	events, diags := types.SetValueFrom(ctx, types.StringType, out.Events)
	if diags.HasError() {
		return Webhook{}, diags
	}

	return Webhook{
		ID:         types.StringValue(out.ID),
		TeamID:     types.StringValue(out.TeamID),
		Endpoint:   types.StringValue(out.Endpoint),
		Secret:     types.StringValue(out.Secret),
		ProjectIDs: projectIDs,
		Events:     events,
	}, diags
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Webhook
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var events []string
	diags = plan.Events.ElementsAs(ctx, &events, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var projectIDs []string
	diags = plan.ProjectIDs.ElementsAs(ctx, &projectIDs, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateWebhook(ctx, client.CreateWebhookRequest{
		TeamID:     plan.TeamID.ValueString(),
		Events:     events,
		Endpoint:   plan.Endpoint.ValueString(),
		ProjectIDs: projectIDs,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Webhook",
			"Could not create Webhook, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := responseToWebhook(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "created webhook", map[string]interface{}{
		"team_id":    plan.TeamID.ValueString(),
		"webhook_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read webhook information by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Webhook
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetWebhook(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Webhook",
			fmt.Sprintf("Could not get Webhook %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	// Override the secret with state as this is not returned by the 'GET' endpoint.
	out.Secret = state.Secret.ValueString()
	result, diags := responseToWebhook(ctx, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read webhook", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"webhook_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update does nothing.
func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating a Webhook is not supported",
		"Updating a Webhook is not supported",
	)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Webhook
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWebhook(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Webhook",
			fmt.Sprintf(
				"Could not delete Webhook %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Webhook", map[string]interface{}{
		"team_id":    state.TeamID.ValueString(),
		"webhook_id": state.ID.ValueString(),
	})
}
