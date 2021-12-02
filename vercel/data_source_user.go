package vercel

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vercel/terraform-provider-vercel/client"
)

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceUserRead,
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

func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.Client)

	log.Printf("[DEBUG] Reading User\n")
	user, err := client.GetUser(ctx)
	if err != nil {
		return diag.Errorf("error getting user: %s", err)
	}

	if err := d.Set("name", user.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("username", user.Username); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("plan", user.Billing.Plan); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(user.Username)

	return nil
}
