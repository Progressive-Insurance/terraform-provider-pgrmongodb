package pgrmongodb

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// note this checks for a container with at least 1 network container in the project
func TestAccPGRMongoDBAtlasContainers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "pgrmongodb_atlasclustercontainer" "test" {
	project_id = "000000000000000000000000"
	cloud_provider = "AWS"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.pgrmongodb_atlasclustercontainer.test", "id"),
					resource.TestCheckResourceAttrSet("data.pgrmongodb_atlasclustercontainer.test", "project_id"),
				),
			},
		},
	})
}
