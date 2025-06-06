package test

import (
	"context"
	"os"
	"testing"

	"github.com/vercel/terraform-provider-vercel/v3/client"
)

func TestRollingReleaseAPI(t *testing.T) {
	token := os.Getenv("VERCEL_API_TOKEN")
	if token == "" {
		t.Skip("VERCEL_API_TOKEN not set")
	}

	c := client.New(token)
	ctx := context.Background()

	projectID := "prj_9lRsbRoK8DCtxa4CmUu5rWfSaS86"
	teamID := "team_4FWx5KQoszRi0ZmM9q9IBoKG"

	// First, get the current state
	current, err := c.GetRollingRelease(ctx, projectID, teamID)
	if err != nil {
		t.Fatalf("Failed to get current state: %v", err)
	}

	t.Logf("Current state: %+v", current)

	// Define default stages that meet API requirements
	defaultStages := []client.RollingReleaseStage{
		{
			TargetPercentage: 10,
			Duration:         60, // 1 hour in minutes
		},
		{
			TargetPercentage: 50,
			Duration:         120, // 2 hours in minutes
		},
		{
			TargetPercentage: 100, // Final stage must be 100%
		},
	}

	// Try different combinations
	tests := []struct {
		name string
		config client.RollingRelease
	}{
		{
			name: "disable rolling release",
			config: client.RollingRelease{
				Enabled: false,
			},
		},
		{
			name: "enable with automatic advancement",
			config: client.RollingRelease{
				Enabled: true,
				AdvancementType: "automatic",
				Stages: defaultStages,
			},
		},
		{
			name: "enable with manual approval",
			config: client.RollingRelease{
				Enabled: true,
				AdvancementType: "manual-approval",
				Stages: []client.RollingReleaseStage{
					{
						TargetPercentage: 5,
					},
					{
						TargetPercentage: 25,
					},
					{
						TargetPercentage: 60,
					},
					{
						TargetPercentage: 100,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := client.UpdateRollingReleaseRequest{
				RollingRelease: tc.config,
				ProjectID: projectID,
				TeamID: teamID,
			}

			// Update the configuration
			t.Logf("Sending request: %+v", request)
			updated, err := c.UpdateRollingRelease(ctx, request)
			if err != nil {
				t.Fatalf("Update failed: %v", err)
			}

			t.Logf("Update response: %+v", updated)

			// Get the state again to verify
			final, err := c.GetRollingRelease(ctx, projectID, teamID)
			if err != nil {
				t.Fatalf("Failed to get final state: %v", err)
			}

			t.Logf("Final state: %+v", final)
		})
	}
} 