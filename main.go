package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/vercel/terraform-provider-vercel/vercel"
)

func main() {
	tfsdk.Serve(context.Background(), vercel.New, tfsdk.ServeOpts{
		Name: "vercel",
	})
}
