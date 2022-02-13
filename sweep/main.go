package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/vercel/terraform-provider-vercel/client"
)

func main() {
	// We want to clean up any resources in the account.
	// It's actually pretty easy - everything is tied to a project,
	// so removing a project will remove everything else.
	// This means we only need to delete projects.
	c := client.New(os.Getenv("VERCEL_API_TOKEN"))
	teamID := os.Getenv("VERCEL_TERRAFORM_TESTING_TEAM")
	ctx := context.Background()

	// delete both for the testing team, and for without a team
	err := deleteAllProjects(ctx, c, teamID)
	if err != nil {
		panic(err)
	}
	err = deleteAllProjects(ctx, c, "")
	if err != nil {
		panic(err)
	}
}

func deleteAllProjects(ctx context.Context, c *client.Client, teamID string) error {
	projects, err := c.ListProjects(ctx, teamID)
	if err != nil {
		return fmt.Errorf("error listing projects: %w", err)
	}

	for _, p := range projects {
		if !strings.HasPrefix(p.Name, "test-acc") {
			// Don't delete actual projects - only testing ones
			continue
		}

		err = c.DeleteProject(ctx, p.ID, teamID)
		if err != nil {
			return fmt.Errorf("error deleting project: %w", err)
		}
		log.Printf("Deleted project %s", p.Name)
	}

	return nil
}
