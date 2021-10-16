package vercel

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vercel/terraform-provider-vercel/client"
)

func dataSourceVercelUser() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVercelUserRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Computed: true,
				Type:     schema.TypeString,
			},
			"username": {
				Computed: true,
				Type:     schema.TypeString,
			},
			"plan": {
				Computed: true,
				Type:     schema.TypeString,
			},
		},
	}
}

func dataSourceVercelUserRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client.Client)

	log.Printf("[DEBUG] Reading User\n")
	user, err := client.GetUser(context.Background())
	if err != nil {
		return fmt.Errorf("error getting user: %w", err)
	}

	if err := d.Set("name", user.Name); err != nil {
		return err
	}
	if err := d.Set("username", user.Username); err != nil {
		return err
	}
	if err := d.Set("plan", user.Billing.Plan); err != nil {
		return err
	}

	// Always read this resource
	d.SetId(user.Username)

	return nil
}
