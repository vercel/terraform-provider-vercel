package vercel_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

const manualPolicyResponse = `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":10,"requireApproval":true},{"targetPercentage":50,"requireApproval":true},{"targetPercentage":100}]}}`

func rollingPolicyResource(t *testing.T, handler http.HandlerFunc) (resource.Resource, tfsdk.State) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	ctx := context.Background()
	for _, factory := range vercel.New().Resources(ctx) {
		r := factory()
		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "vercel"}, &metadata)
		if metadata.TypeName != "vercel_project_rolling_release" {
			continue
		}
		var configured resource.ConfigureResponse
		r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{
			ProviderData: client.New("fixture-only").WithBaseURL(server.URL),
		}, &configured)
		if configured.Diagnostics.HasError() {
			t.Fatal(configured.Diagnostics)
		}
		var schema resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &schema)
		state := tfsdk.State{Schema: schema.Schema}
		stages, diags := types.ListValueFrom(ctx, vercel.RollingReleaseStageElementType, []vercel.RollingReleaseStage{
			{TargetPercentage: types.Int64Value(10), Duration: types.Int64Null()},
			{TargetPercentage: types.Int64Value(50), Duration: types.Int64Null()},
			{TargetPercentage: types.Int64Value(100), Duration: types.Int64Null()},
		})
		if diags.HasError() {
			t.Fatal(diags)
		}
		diags = state.Set(ctx, vercel.RollingReleaseInfo{
			ID: types.StringValue("prj_fixture"), ProjectID: types.StringValue("prj_fixture"),
			TeamID: types.StringValue("team_fixture"), AdvancementType: types.StringValue("manual-approval"), Stages: stages,
		})
		if diags.HasError() {
			t.Fatal(diags)
		}
		return r, state
	}
	t.Fatal("rolling release resource is not registered")
	return nil, tfsdk.State{}
}

func TestProjectRollingReleasePATCHBody(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			bodies := make(chan []byte, 1)
			var applied atomic.Bool
			r, state := rollingPolicyResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/v1/projects/prj_fixture/rolling-release/config" || req.URL.Query().Get("teamId") != "team_fixture" {
					t.Errorf("unexpected path: %s", req.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				switch req.Method {
				case http.MethodGet:
					if applied.Load() {
						fmt.Fprint(w, manualPolicyResponse)
					} else {
						fmt.Fprint(w, `{"rollingRelease":null}`)
					}
				case http.MethodPatch:
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Error(err)
					}
					bodies <- body
					applied.Store(true)
					fmt.Fprint(w, manualPolicyResponse)
				default:
					t.Errorf("unexpected method: %s", req.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			})
			plan := tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
			ctx := context.Background()
			if operation == "create" {
				response := resource.CreateResponse{State: state}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
			} else {
				response := resource.UpdateResponse{State: state}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
			}
			var got, want any
			if err := json.Unmarshal(<-bodies, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(`{"enabled":true,"advancementType":"manual-approval","stages":[{"targetPercentage":10},{"targetPercentage":50},{"targetPercentage":100}]}`), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("PATCH body = %v, want %v", got, want)
			}
		})
	}
}

func TestProjectRollingReleaseReadNativeState(t *testing.T) {
	for _, test := range []struct {
		name       string
		body       string
		status     int
		mode       string
		first      int64
		duration   int64
		removed    bool
		diagnostic bool
	}{
		{name: "changed manual", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":25,"requireApproval":true},{"targetPercentage":100}]}}`, mode: "manual-approval", first: 25},
		{name: "zero first stage", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":0,"requireApproval":true},{"targetPercentage":100}]}}`, mode: "manual-approval", first: 0},
		{name: "changed automatic", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":20,"duration":5},{"targetPercentage":100}]}}`, mode: "automatic", first: 20, duration: 5},
		{name: "disabled null", body: `{"rollingRelease":null}`, removed: true},
		{name: "not found", status: http.StatusNotFound, body: `{"error":{"code":"not_found","message":"fixture"}}`, removed: true},
		{name: "forbidden", status: http.StatusForbidden, body: `{"error":{"code":"forbidden","message":"fixture"}}`, diagnostic: true},
		{name: "missing wrapper", body: `{}`, diagnostic: true},
		{name: "missing target", body: `{"rollingRelease":{"stages":[{"targetPercentage":25,"requireApproval":true},{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "missing percentage", body: `{"rollingRelease":{"target":"production","stages":[{"requireApproval":true},{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "invalid percentage", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":"25"},{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "single final", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "empty policy", body: `{"rollingRelease":{"target":"production","stages":[]}}`, diagnostic: true},
		{name: "unrecognized disabled object", body: `{"rollingRelease":{"enabled":false}}`, diagnostic: true},
		{name: "contradictory disabled", body: `{"rollingRelease":{"enabled":false,"target":"production","stages":[{"targetPercentage":25,"requireApproval":true},{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "contradictory mode", body: `{"rollingRelease":{"advancementType":"automatic","target":"production","stages":[{"targetPercentage":25,"requireApproval":true},{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "ambiguous advancement", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":25},{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "null stages", body: `{"rollingRelease":{"target":"production","stages":null}}`, diagnostic: true},
		{name: "final approval", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":25,"requireApproval":true},{"targetPercentage":100,"requireApproval":true}]}}`, diagnostic: true},
		{name: "linear shift", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":25,"duration":5,"linearShift":true},{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "gate", body: `{"rollingRelease":{"target":"production","gate":{"enabled":true},"stages":[{"targetPercentage":25,"requireApproval":true},{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "mixed stage modes", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":25,"requireApproval":true},{"targetPercentage":50,"duration":5},{"targetPercentage":100}]}}`, diagnostic: true},
		{name: "mixed advancement", body: `{"rollingRelease":{"target":"production","stages":[{"targetPercentage":25,"requireApproval":true,"duration":5},{"targetPercentage":100}]}}`, diagnostic: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, state := rollingPolicyResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet {
					t.Errorf("Read issued %s", req.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				if test.status != 0 {
					w.WriteHeader(test.status)
				}
				fmt.Fprint(w, test.body)
			})
			ctx := context.Background()
			response := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &response)
			if response.Diagnostics.HasError() != test.diagnostic {
				t.Fatalf("diagnostics = %v, want error %t", response.Diagnostics, test.diagnostic)
			}
			if test.diagnostic {
				return
			}
			if response.State.Raw.IsNull() != test.removed {
				t.Fatalf("state removed = %t, want %t", response.State.Raw.IsNull(), test.removed)
			}
			if test.removed {
				return
			}
			var got vercel.RollingReleaseInfo
			if diags := response.State.Get(ctx, &got); diags.HasError() {
				t.Fatal(diags)
			}
			var stages []vercel.RollingReleaseStage
			if diags := got.Stages.ElementsAs(ctx, &stages, false); diags.HasError() {
				t.Fatal(diags)
			}
			if (test.duration == 0 && !stages[0].Duration.IsNull()) || stages[0].Duration.ValueInt64() != test.duration {
				t.Fatalf("duration = %v, want %d", stages[0].Duration, test.duration)
			}
			if got.AdvancementType.ValueString() != test.mode || len(stages) != 2 || stages[0].TargetPercentage.ValueInt64() != test.first {
				t.Fatalf("native state = %s %v, want %s starting at %d", got.AdvancementType, stages, test.mode, test.first)
			}
		})
	}
}

func TestProjectRollingReleaseUnknownWrite(t *testing.T) {
	for _, lostReply := range []bool{true, false} {
		t.Run(fmt.Sprintf("lost reply %t", lostReply), func(t *testing.T) {
			var patches, reads atomic.Int32
			r, state := rollingPolicyResource(t, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if req.Method == http.MethodPatch {
					patches.Add(1)
					if lostReply {
						connection, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						connection.Close()
						return
					}
					fmt.Fprint(w, `{}`)
					return
				}
				if req.Method != http.MethodGet {
					t.Errorf("unexpected method: %s", req.Method)
				}
				reads.Add(1)
				fmt.Fprint(w, `{"rollingRelease":null}`)
			})
			response := resource.UpdateResponse{State: state}
			r.Update(context.Background(), resource.UpdateRequest{
				Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}, State: state,
			}, &response)
			if !response.Diagnostics.HasError() || patches.Load() != 1 {
				t.Fatalf("diagnostics = %v, PATCH count = %d", response.Diagnostics, patches.Load())
			}
			if lostReply && reads.Load() != 0 {
				t.Fatal("unknown write was followed by more requests")
			}
		})
	}
}

func TestProjectRollingReleaseRequiresAdvancementStage(t *testing.T) {
	_, state := rollingPolicyResource(t, func(http.ResponseWriter, *http.Request) {
		t.Error("schema validation contacted the API")
	})
	ctx := context.Background()
	stages, diags := types.ListValueFrom(ctx, vercel.RollingReleaseStageElementType, []vercel.RollingReleaseStage{
		{TargetPercentage: types.Int64Value(100), Duration: types.Int64Null()},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	var response validator.ListResponse
	for _, check := range state.Schema.(schema.Schema).Attributes["stages"].(schema.ListNestedAttribute).Validators {
		check.ValidateList(ctx, validator.ListRequest{Path: path.Root("stages"), ConfigValue: stages}, &response)
	}
	if !response.Diagnostics.HasError() {
		t.Fatal("a final stage alone cannot establish the native advancement mode")
	}
}

func TestProjectRollingReleaseCreateRequiresImport(t *testing.T) {
	var patches atomic.Int32
	r, state := rollingPolicyResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPatch {
			patches.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, manualPolicyResponse)
	})
	response := resource.CreateResponse{State: state}
	r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}}, &response)
	if !response.Diagnostics.HasError() || patches.Load() != 0 {
		t.Fatalf("existing native policy: diagnostics = %v, PATCH count = %d", response.Diagnostics, patches.Load())
	}
}

func TestProjectRollingReleaseCreateReadFailurePreservesIdentity(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "unavailable", status: http.StatusServiceUnavailable, body: `{"error":{"code":"unavailable","message":"try again"}}`},
		{name: "malformed", body: `{}`},
		{name: "absent", body: `{"rollingRelease":null}`},
		{name: "not found", status: http.StatusNotFound, body: `{"error":{"code":"not_found","message":"fixture"}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var patches, reads atomic.Int32
			var recoverRead atomic.Bool
			r, planned := rollingPolicyResource(t, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch req.Method {
				case http.MethodPatch:
					patches.Add(1)
					fmt.Fprint(w, `{}`)
				case http.MethodGet:
					reads.Add(1)
					if patches.Load() == 0 {
						fmt.Fprint(w, `{"rollingRelease":null}`)
					} else if recoverRead.Load() {
						fmt.Fprint(w, manualPolicyResponse)
					} else {
						if test.status != 0 {
							w.WriteHeader(test.status)
						}
						fmt.Fprint(w, test.body)
					}
				default:
					t.Errorf("unexpected method: %s", req.Method)
				}
			})
			ctx := context.Background()
			// Create starts with no prior state, as it does under Terraform.
			response := resource.CreateResponse{State: tfsdk.State{Schema: planned.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: planned.Schema, Raw: planned.Raw}}, &response)
			if !response.Diagnostics.HasError() || patches.Load() != 1 || reads.Load() != 2 {
				t.Fatalf("diagnostics = %v, PATCH count = %d, GET count = %d", response.Diagnostics, patches.Load(), reads.Load())
			}
			var saved vercel.RollingReleaseInfo
			if diags := response.State.Get(ctx, &saved); diags.HasError() {
				t.Fatal(diags)
			}
			if saved.ID.ValueString() != "prj_fixture" || saved.ProjectID.ValueString() != "prj_fixture" || saved.TeamID.ValueString() != "team_fixture" {
				t.Fatalf("creation identity was not saved: %+v", saved)
			}
			if !saved.AdvancementType.IsNull() || !saved.Stages.IsNull() {
				t.Fatalf("unverified settings were saved: %+v", saved)
			}
			recoverRead.Store(true)
			refreshed := resource.ReadResponse{State: response.State}
			r.Read(ctx, resource.ReadRequest{State: response.State}, &refreshed)
			if refreshed.Diagnostics.HasError() {
				t.Fatal(refreshed.Diagnostics)
			}
			if !refreshed.State.Raw.Equal(planned.Raw) {
				t.Fatalf("refresh did not recover native state: %v", refreshed.State.Raw)
			}
		})
	}
}

func TestProjectRollingReleaseCreateLostReplyDoesNotSaveIdentity(t *testing.T) {
	var patches, reads atomic.Int32
	r, planned := rollingPolicyResource(t, func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			reads.Add(1)
			fmt.Fprint(w, `{"rollingRelease":null}`)
		case http.MethodPatch:
			patches.Add(1)
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			connection.Close()
		default:
			t.Errorf("unexpected method: %s", req.Method)
		}
	})
	response := resource.CreateResponse{State: tfsdk.State{Schema: planned.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan{Schema: planned.Schema, Raw: planned.Raw}}, &response)
	if !response.Diagnostics.HasError() || patches.Load() != 1 || reads.Load() != 1 {
		t.Fatalf("diagnostics = %v, PATCH count = %d, GET count = %d", response.Diagnostics, patches.Load(), reads.Load())
	}
	if !response.State.Raw.IsNull() {
		t.Fatalf("unacknowledged creation saved state: %v", response.State.Raw)
	}
}
