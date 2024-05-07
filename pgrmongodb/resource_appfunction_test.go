package pgrmongodb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var function_1 = `exports = async (changeEvent) => {
	console.log('Function Code 1');
}
`

var function_2 = `exports = async (changeEvent) => {
	console.log('Function Code 2');
}
`

func TestAccPGRMongoDBAppFunction(t *testing.T) {
	project_id := "000000000000000000000000"
	appservices_app_id := "000000000000000000000000"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccCheckPGRMongoDBAppFunctionConfig(project_id, appservices_app_id, "my_tf_function", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("pgrmongodb_appfunction.test", "project_id"),
					resource.TestCheckResourceAttr("pgrmongodb_appfunction.test", "function_name", "my_tf_function"),
					resource.TestCheckResourceAttr("pgrmongodb_appfunction.test", "function_code", function_1),
					resource.TestCheckResourceAttrSet("pgrmongodb_appfunction.test", "appservices_app_id"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccCheckPGRMongoDBAppFunctionConfig(project_id, appservices_app_id, "my_tf_function", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("pgrmongodb_appfunction.test", "project_id"),
					resource.TestCheckResourceAttr("pgrmongodb_appfunction.test", "function_name", "my_tf_function"),
					resource.TestCheckResourceAttr("pgrmongodb_appfunction.test", "function_code", function_2),
					resource.TestCheckResourceAttrSet("pgrmongodb_appfunction.test", "appservices_app_id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCheckPGRMongoDBAppFunctionConfig(project_id string, appservices_app_id string, function_name string, function_id int) string {
	if function_id == 1 {
		return fmt.Sprintf(`
		resource "pgrmongodb_appfunction" "test" {
			project_id = "%s"
			appservices_app_id = "%s"
			function_name = "%s"
			function_code = <<EOT
exports = async (changeEvent) => {
	console.log('Function Code 1');
}
EOT
		}
		`, project_id, appservices_app_id, function_name)
	} else {
		return fmt.Sprintf(`
		resource "pgrmongodb_appfunction" "test" {
			project_id = "%s"
			appservices_app_id = "%s"
			function_name = "%s"
			function_code = <<EOT
exports = async (changeEvent) => {
	console.log('Function Code 2');
}
EOT
		}
		`, project_id, appservices_app_id, function_name)
	}
}
