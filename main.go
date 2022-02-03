package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/vercel/terraform-provider-vercel/vercel"
)

func main() {
	err := tfsdk.Serve(context.Background(), vercel.New, tfsdk.ServeOpts{
		Name: "registry.terraform.io/vercel/vercel",
	})
	if err != nil {
		log.Fatalf("unable to serve provider: %s", err)
	}
}
