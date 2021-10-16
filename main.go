package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/vercel/terraform-provider-vercel/vercel"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: vercel.Provider,
	})
}
