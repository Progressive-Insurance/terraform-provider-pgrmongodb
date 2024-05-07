package main

import (
	"context"
	"terraform-provider-pgrmongodb/pgrmongodb"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	providerserver.Serve(context.Background(), pgrmongodb.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/progressive/pgrmongodb",
	})
}
