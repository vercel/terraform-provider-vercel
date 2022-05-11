package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/vercel/terraform-provider-vercel/vercel"
)

func main() {
	err := providerserver.Serve(context.Background(), vercel.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/vercel/vercel",
	})
	if err != nil {
		log.Fatalf("unable to serve provider: %s", err)
	}
}
