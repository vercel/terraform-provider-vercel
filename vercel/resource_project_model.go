package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Project reflects the state terraform stores internally for a project.
type Project struct {
	BuildCommand                        types.String          `tfsdk:"build_command"`
	DevCommand                          types.String          `tfsdk:"dev_command"`
	Environment                         types.Set             `tfsdk:"environment"`
	Framework                           types.String          `tfsdk:"framework"`
	GitRepository                       *GitRepository        `tfsdk:"git_repository"`
	ID                                  types.String          `tfsdk:"id"`
	IgnoreCommand                       types.String          `tfsdk:"ignore_command"`
	InstallCommand                      types.String          `tfsdk:"install_command"`
	Name                                types.String          `tfsdk:"name"`
	OutputDirectory                     types.String          `tfsdk:"output_directory"`
	PublicSource                        types.Bool            `tfsdk:"public_source"`
	RootDirectory                       types.String          `tfsdk:"root_directory"`
	ServerlessFunctionRegion            types.String          `tfsdk:"serverless_function_region"`
	TeamID                              types.String          `tfsdk:"team_id"`
	VercelAuthentication                *VercelAuthentication `tfsdk:"vercel_authentication"`
	PasswordProtection                  *PasswordProtection   `tfsdk:"password_protection"`
	TrustedIps                          *TrustedIps           `tfsdk:"trusted_ips"`
	ProtectionBypassForAutomation       types.Bool            `tfsdk:"protection_bypass_for_automation"`
	ProtectionBypassForAutomationSecret types.String          `tfsdk:"protection_bypass_for_automation_secret"`
}

var nullProject = Project{
	/* As this is read only, none of these fields are specified - so treat them all as Null */
	BuildCommand:    types.StringNull(),
	DevCommand:      types.StringNull(),
	InstallCommand:  types.StringNull(),
	OutputDirectory: types.StringNull(),
	PublicSource:    types.BoolNull(),
	Environment:     types.SetNull(envVariableElemType),
}

func (p *Project) environment(ctx context.Context) ([]EnvironmentItem, error) {
	if p.Environment.IsNull() {
		return nil, nil
	}

	var vars []EnvironmentItem
	err := p.Environment.ElementsAs(ctx, &vars, true)
	if err != nil {
		return nil, fmt.Errorf("error reading project environment variables: %s", err)
	}
	return vars, nil
}

func parseEnvironment(vars []EnvironmentItem) []client.EnvironmentVariable {
	out := []client.EnvironmentVariable{}
	for _, e := range vars {
		target := []string{}
		for _, t := range e.Target {
			target = append(target, t.ValueString())
		}

		out = append(out, client.EnvironmentVariable{
			Key:       e.Key.ValueString(),
			Value:     e.Value.ValueString(),
			Target:    target,
			GitBranch: toStrPointer(e.GitBranch),
			Type:      "encrypted",
			ID:        e.ID.ValueString(),
		})
	}
	return out
}

func (p *Project) toCreateProjectRequest(envs []EnvironmentItem) client.CreateProjectRequest {
	return client.CreateProjectRequest{
		BuildCommand:                toStrPointer(p.BuildCommand),
		CommandForIgnoringBuildStep: toStrPointer(p.IgnoreCommand),
		DevCommand:                  toStrPointer(p.DevCommand),
		EnvironmentVariables:        parseEnvironment(envs),
		Framework:                   toStrPointer(p.Framework),
		GitRepository:               p.GitRepository.toCreateProjectRequest(),
		InstallCommand:              toStrPointer(p.InstallCommand),
		Name:                        p.Name.ValueString(),
		OutputDirectory:             toStrPointer(p.OutputDirectory),
		PublicSource:                toBoolPointer(p.PublicSource),
		RootDirectory:               toStrPointer(p.RootDirectory),
		ServerlessFunctionRegion:    toStrPointer(p.ServerlessFunctionRegion),
	}
}

func (p *Project) toUpdateProjectRequest(oldName string) client.UpdateProjectRequest {
	var name *string = nil
	if oldName != p.Name.ValueString() {
		n := p.Name.ValueString()
		name = &n
	}
	return client.UpdateProjectRequest{
		BuildCommand:                toStrPointer(p.BuildCommand),
		CommandForIgnoringBuildStep: toStrPointer(p.IgnoreCommand),
		DevCommand:                  toStrPointer(p.DevCommand),
		Framework:                   toStrPointer(p.Framework),
		InstallCommand:              toStrPointer(p.InstallCommand),
		Name:                        name,
		OutputDirectory:             toStrPointer(p.OutputDirectory),
		PublicSource:                toBoolPointer(p.PublicSource),
		RootDirectory:               toStrPointer(p.RootDirectory),
		ServerlessFunctionRegion:    toStrPointer(p.ServerlessFunctionRegion),
		PasswordProtection:          p.PasswordProtection.toUpdateProjectRequest(),
		VercelAuthentication:        p.VercelAuthentication.toUpdateProjectRequest(),
		TrustedIps:                  p.TrustedIps.toUpdateProjectRequest(),
	}
}

// EnvironmentItem reflects the state terraform stores internally for a project's environment variable.
type EnvironmentItem struct {
	Target    []types.String `tfsdk:"target"`
	GitBranch types.String   `tfsdk:"git_branch"`
	Key       types.String   `tfsdk:"key"`
	Value     types.String   `tfsdk:"value"`
	ID        types.String   `tfsdk:"id"`
}

func (e *EnvironmentItem) toEnvironmentVariableRequest() client.EnvironmentVariableRequest {
	target := []string{}
	for _, t := range e.Target {
		target = append(target, t.ValueString())
	}
	return client.EnvironmentVariableRequest{
		Key:       e.Key.ValueString(),
		Value:     e.Value.ValueString(),
		Target:    target,
		GitBranch: toStrPointer(e.GitBranch),
		Type:      "encrypted",
	}
}

// GitRepository reflects the state terraform stores internally for a nested git_repository block on a project resource.
type GitRepository struct {
	Type             types.String `tfsdk:"type"`
	Repo             types.String `tfsdk:"repo"`
	ProductionBranch types.String `tfsdk:"production_branch"`
}

func (g *GitRepository) toCreateProjectRequest() *client.GitRepository {
	if g == nil {
		return nil
	}
	return &client.GitRepository{
		Type: g.Type.ValueString(),
		Repo: g.Repo.ValueString(),
	}
}

type VercelAuthentication struct {
	DeploymentType types.String `tfsdk:"deployment_type"`
}

func (v *VercelAuthentication) toUpdateProjectRequest() *client.VercelAuthenticationRequest {
	if v == nil {
		return nil
	}

	return &client.VercelAuthenticationRequest{
		DeploymentType: v.DeploymentType.ValueString(),
	}
}

type PasswordProtection struct {
	Password       types.String `tfsdk:"password"`
	DeploymentType types.String `tfsdk:"deployment_type"`
}

func (p *PasswordProtection) toUpdateProjectRequest() *client.PasswordProtectionRequest {
	if p == nil {
		return nil
	}

	return &client.PasswordProtectionRequest{
		DeploymentType: p.DeploymentType.ValueString(),
		Password:       p.Password.ValueString(),
	}
}

type TrustedIpsAddress struct {
	Value types.String `tfsdk:"value"`
	Note  types.String `tfsdk:"note"`
}

type TrustedIps struct {
	Addresses      []TrustedIpsAddress `tfsdk:"addresses"`
	DeploymentType types.String        `tfsdk:"deployment_type"`
	ProtectionMode types.String        `tfsdk:"protection_mode"`
}

func (t *TrustedIps) toUpdateProjectRequest() *client.TrustedIpsRequest {
	if t == nil {
		return nil
	}

	var addresses = []client.TrustedIpAddress{}
	for _, address := range t.Addresses {
		addresses = append(addresses, client.TrustedIpAddress{
			Value: address.Value.ValueString(),
			Note:  address.Note.ValueString(),
		})
	}

	return &client.TrustedIpsRequest{
		Addresses:      addresses,
		DeploymentType: t.DeploymentType.ValueString(),
		ProtectionMode: t.ProtectionMode.ValueString(),
	}
}

/*
* In the Vercel API the following fields are coerced to null during project creation

* This causes an issue when they are specified, but falsy, as the
* terraform configuration explicitly sets a value for them, but the Vercel
* API returns a different value. This causes an inconsistent plan error.

* We avoid this issue by choosing to use values from the terraform state,
* but only if they are _explicitly stated_ *and* they are _falsy_ values
* *and* the response value was null. This is important as drift detection
* would fail to work if the value was always selected, so this is as stringent
* as possible to allow drift-detection in the majority of scenarios.

* This is implemented in the below uncoerceString and uncoerceBool functions.
 */
type projectCoercedFields struct {
	BuildCommand    types.String
	DevCommand      types.String
	InstallCommand  types.String
	OutputDirectory types.String
	PublicSource    types.Bool
}

func (p *Project) coercedFields() projectCoercedFields {
	return projectCoercedFields{
		BuildCommand:    p.BuildCommand,
		DevCommand:      p.DevCommand,
		InstallCommand:  p.InstallCommand,
		OutputDirectory: p.OutputDirectory,
		PublicSource:    p.PublicSource,
	}
}

func uncoerceString(plan, res types.String) types.String {
	if plan.ValueString() == "" && !plan.IsNull() && res.IsNull() {
		return plan
	}
	return res
}
func uncoerceBool(plan, res types.Bool) types.Bool {
	if !plan.ValueBool() && !plan.IsNull() && res.IsNull() {
		return plan
	}
	return res
}

var envVariableElemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
		"target": types.SetType{
			ElemType: types.StringType,
		},
		"git_branch": types.StringType,
		"id":         types.StringType,
	},
}

func convertResponseToProject(response client.ProjectResponse, plan Project) Project {
	fields := plan.coercedFields()

	var gr *GitRepository
	if repo := response.Repository(); repo != nil {
		gr = &GitRepository{
			Type:             types.StringValue(repo.Type),
			Repo:             types.StringValue(repo.Repo),
			ProductionBranch: types.StringNull(),
		}
		if repo.ProductionBranch != nil {
			gr.ProductionBranch = types.StringValue(*repo.ProductionBranch)
		}
	}

	var pp *PasswordProtection
	if response.PasswordProtection != nil {
		pass := types.StringValue("")
		if plan.PasswordProtection != nil {
			pass = plan.PasswordProtection.Password
		}
		pp = &PasswordProtection{
			Password:       pass,
			DeploymentType: types.StringValue(response.PasswordProtection.DeploymentType),
		}
	}

	var va *VercelAuthentication
	if response.VercelAuthentication != nil {
		va = &VercelAuthentication{
			DeploymentType: types.StringValue(response.VercelAuthentication.DeploymentType),
		}
	}

	var tip *TrustedIps
	if response.TrustedIps != nil {
		var addresses []TrustedIpsAddress
		for _, address := range response.TrustedIps.Addresses {
			addresses = append(addresses, TrustedIpsAddress{
				Value: types.StringValue(address.Value),
				Note:  types.StringValue(address.Note),
			})
		}
		tip = &TrustedIps{
			DeploymentType: types.StringValue(response.TrustedIps.DeploymentType),
			Addresses:      addresses,
			ProtectionMode: types.StringValue(response.TrustedIps.ProtectionMode),
		}
	}

	var env []attr.Value
	for _, e := range response.EnvironmentVariables {
		target := []attr.Value{}
		for _, t := range e.Target {
			target = append(target, types.StringValue(t))
		}
		env = append(env, types.ObjectValueMust(
			map[string]attr.Type{
				"key":   types.StringType,
				"value": types.StringType,
				"target": types.SetType{
					ElemType: types.StringType,
				},
				"git_branch": types.StringType,
				"id":         types.StringType,
			},
			map[string]attr.Value{
				"key":        types.StringValue(e.Key),
				"value":      types.StringValue(e.Value),
				"target":     types.SetValueMust(types.StringType, target),
				"git_branch": fromStringPointer(e.GitBranch),
				"id":         types.StringValue(e.ID),
			},
		))
	}

	protectionBypassSecret := types.StringNull()
	protectionBypass := types.BoolNull()
	for k, v := range response.ProtectionBypass {
		if v.Scope == "automation-bypass" {
			protectionBypass = types.BoolValue(true)
			protectionBypassSecret = types.StringValue(k)
			break
		}
	}
	if !plan.ProtectionBypassForAutomation.IsNull() && !plan.ProtectionBypassForAutomation.ValueBool() {
		protectionBypass = types.BoolValue(false)
	}

	environmentEntry := types.SetValueMust(envVariableElemType, env)
	if len(response.EnvironmentVariables) == 0 && plan.Environment.IsNull() {
		environmentEntry = types.SetNull(envVariableElemType)
	}

	return Project{
		BuildCommand:                        uncoerceString(fields.BuildCommand, fromStringPointer(response.BuildCommand)),
		DevCommand:                          uncoerceString(fields.DevCommand, fromStringPointer(response.DevCommand)),
		Environment:                         environmentEntry,
		Framework:                           fromStringPointer(response.Framework),
		GitRepository:                       gr,
		ID:                                  types.StringValue(response.ID),
		IgnoreCommand:                       fromStringPointer(response.CommandForIgnoringBuildStep),
		InstallCommand:                      uncoerceString(fields.InstallCommand, fromStringPointer(response.InstallCommand)),
		Name:                                types.StringValue(response.Name),
		OutputDirectory:                     uncoerceString(fields.OutputDirectory, fromStringPointer(response.OutputDirectory)),
		PublicSource:                        uncoerceBool(fields.PublicSource, fromBoolPointer(response.PublicSource)),
		RootDirectory:                       fromStringPointer(response.RootDirectory),
		ServerlessFunctionRegion:            fromStringPointer(response.ServerlessFunctionRegion),
		TeamID:                              toTeamID(response.TeamID),
		PasswordProtection:                  pp,
		VercelAuthentication:                va,
		TrustedIps:                          tip,
		ProtectionBypassForAutomation:       protectionBypass,
		ProtectionBypassForAutomationSecret: protectionBypassSecret,
	}
}
