package client

import "testing"

func TestFindConflictingEnvIDMatchesGitBranchByValue(t *testing.T) {
	branchFromAPI := "feature"
	branchFromConflict := "feature"

	id, ok := findConflictingEnvID("team_123", "prj_123", EnvConflictError{
		EnvVarKey: "SECRET",
		Target:    []string{"preview"},
		GitBranch: &branchFromConflict,
	}, []EnvironmentVariable{
		{
			ID:        "env_123",
			Key:       "SECRET",
			Target:    []string{"preview"},
			GitBranch: &branchFromAPI,
		},
	})
	if !ok {
		t.Fatal("findConflictingEnvID() did not find conflict")
	}

	if id != "team_123/prj_123/env_123" {
		t.Fatalf("findConflictingEnvID() id = %q, want %q", id, "team_123/prj_123/env_123")
	}
}
