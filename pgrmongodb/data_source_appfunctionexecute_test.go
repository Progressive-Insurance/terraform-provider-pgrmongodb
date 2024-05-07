package pgrmongodb

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// note this checks for a container with at least 1 network container in the project
func TestAccPGRMongoDBAppFunctionExecute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "pgrmongodb_appfunctionexecute" "test" {
	project_id = "000000000000000000000000"
	appservices_app_id = "000000000000000000000000"
	function_name = "myfunction"
	function_args = ["tfarg1", "tfarg2"]
	execute_next_run = true
	execution_timeout = 10
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.pgrmongodb_appfunctionexecute.test", "id"),
					resource.TestCheckResourceAttrSet("data.pgrmongodb_appfunctionexecute.test", "last_run"),
				),
			},
		},
	})
}
