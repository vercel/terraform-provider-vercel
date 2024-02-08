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
	if teamID == "" {
		panic("VERCEL_TERRAFORM_TESTING_TEAM environment variable not set")
	}
	domain := os.Getenv("VERCEL_TERRAFORM_TESTING_DOMAIN")
	if domain == "" {
		panic("VERCEL_TERRAFORM_TESTING_DOMAIN environment variable not set")
	}
	ctx := context.Background()

	// delete both for the testing team, and for without a team
	err := deleteAllProjects(ctx, c, teamID)
	if err != nil {
		panic(err)
	}
	err = deleteAllDNSRecords(ctx, c, domain, teamID)
	if err != nil {
		panic(err)
	}
	err = deleteAllSharedEnvironmentVariables(ctx, c, teamID)
	if err != nil {
		panic(err)
	}
}

func deleteAllSharedEnvironmentVariables(ctx context.Context, c *client.Client, teamID string) error {
	sharedEnvironmentVariables, err := c.ListSharedEnvironmentVariables(ctx, teamID)
	if err != nil {
		return fmt.Errorf("error listing shared environment variables: %w", err)
	}
	for _, d := range sharedEnvironmentVariables {
		if !strings.HasPrefix(d.Key, "test_acc") {
			// Don't delete actual shared environment variables - only testing ones
			continue
		}

		err = c.DeleteSharedEnvironmentVariable(ctx, teamID, d.ID)
		if err != nil {
			return fmt.Errorf("error deleting shared env var %s: %w", d.Key, err)
		}
	}

	return nil
}

func deleteAllDNSRecords(ctx context.Context, c *client.Client, domain, teamID string) error {
	dnsRecords, err := c.ListDNSRecords(ctx, domain, teamID)
	if err != nil {
		return fmt.Errorf("error listing dns records: %w", err)
	}
	for _, d := range dnsRecords {
		if !strings.HasPrefix(d.Name, "test-acc") {
			// Don't delete actual dns records - only testing ones
			continue
		}

		err = c.DeleteDNSRecord(ctx, domain, d.ID, teamID)
		if err != nil {
			return fmt.Errorf("error deleting dns record %s %s for domain %s: %w", d.ID, teamID, d.Domain, err)
		}
	}

	return nil
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
