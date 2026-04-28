package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeBuildMachineType(t *testing.T) {
	type testCase struct {
		name              string
		resourceConfig    *ResourceConfigResponse
		expectedType      string
		expectedSelection string
	}

	for _, tc := range []testCase{
		{
			name:           "nil resource config is a no-op",
			resourceConfig: nil,
		},
		{
			name: "selection=elastic overrides concrete type to elastic",
			resourceConfig: &ResourceConfigResponse{
				BuildMachineType:      "enhanced",
				BuildMachineSelection: "elastic",
			},
			expectedType:      "elastic",
			expectedSelection: "elastic",
		},
		{
			name: "selection=elastic overrides standard to elastic",
			resourceConfig: &ResourceConfigResponse{
				BuildMachineType:      "standard",
				BuildMachineSelection: "elastic",
			},
			expectedType:      "elastic",
			expectedSelection: "elastic",
		},
		{
			name: "selection=fixed leaves buildMachineType untouched",
			resourceConfig: &ResourceConfigResponse{
				BuildMachineType:      "enhanced",
				BuildMachineSelection: "fixed",
			},
			expectedType:      "enhanced",
			expectedSelection: "fixed",
		},
		{
			name: "empty selection leaves buildMachineType untouched",
			resourceConfig: &ResourceConfigResponse{
				BuildMachineType:      "turbo",
				BuildMachineSelection: "",
			},
			expectedType:      "turbo",
			expectedSelection: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &ProjectResponse{ResourceConfig: tc.resourceConfig}
			r.normalizeBuildMachineType()

			if tc.resourceConfig == nil {
				if r.ResourceConfig != nil {
					t.Fatalf("expected nil resource config to remain nil")
				}
				return
			}
			if got := r.ResourceConfig.BuildMachineType; got != tc.expectedType {
				t.Errorf("BuildMachineType: got %q, want %q", got, tc.expectedType)
			}
			if got := r.ResourceConfig.BuildMachineSelection; got != tc.expectedSelection {
				t.Errorf("BuildMachineSelection: got %q, want %q", got, tc.expectedSelection)
			}
		})
	}
}

func TestGetProjectNormalizesElasticBuildMachine(t *testing.T) {
	type testCase struct {
		name         string
		responseJSON string
		expectedType string
	}

	for _, tc := range []testCase{
		{
			name: "API returns selection=elastic with concrete type -> provider returns elastic",
			responseJSON: `{
				"id": "proj_1",
				"name": "test",
				"resourceConfig": {
					"buildMachineType": "enhanced",
					"buildMachineSelection": "elastic"
				}
			}`,
			expectedType: "elastic",
		},
		{
			name: "API returns selection=fixed -> provider returns concrete type",
			responseJSON: `{
				"id": "proj_1",
				"name": "test",
				"resourceConfig": {
					"buildMachineType": "turbo",
					"buildMachineSelection": "fixed"
				}
			}`,
			expectedType: "turbo",
		},
		{
			name: "API omits selection -> provider returns concrete type",
			responseJSON: `{
				"id": "proj_1",
				"name": "test",
				"resourceConfig": {
					"buildMachineType": "enhanced"
				}
			}`,
			expectedType: "enhanced",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintln(w, tc.responseJSON)
			}))
			defer h.Close()

			cl := New("INVALID")
			cl.baseURL = fmt.Sprintf("http://%s", h.Listener.Addr().String())

			r, err := cl.GetProject(context.Background(), "proj_1", "")
			if err != nil {
				t.Fatalf("GetProject: %v", err)
			}
			if r.ResourceConfig == nil {
				t.Fatalf("expected resourceConfig to be set")
			}
			if got := r.ResourceConfig.BuildMachineType; got != tc.expectedType {
				t.Errorf("BuildMachineType: got %q, want %q", got, tc.expectedType)
			}
		})
	}
}
