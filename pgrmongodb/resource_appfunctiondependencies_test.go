package pgrmongodb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPGRMongoDBAppFunctionDependencies(t *testing.T) {
	project_id := "000000000000000000000000"
	appservices_app_id := "000000000000000000000000"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccCheckPGRMongoDBAppFunctionDependenciesConfig(project_id, appservices_app_id, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("pgrmongodb_appfunctiondependencies.test", "project_id"),
					resource.TestCheckTypeSetElemAttr("pgrmongodb_appfunctiondependencies.test", "dependencies.*", "uuidv1 1.6.14"),
					resource.TestCheckTypeSetElemAttr("pgrmongodb_appfunctiondependencies.test", "dependencies.*", "simple-test-package 0.2.2"),
					resource.TestCheckResourceAttrSet("pgrmongodb_appfunctiondependencies.test", "appservices_app_id"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccCheckPGRMongoDBAppFunctionDependenciesConfig(project_id, appservices_app_id, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("pgrmongodb_appfunctiondependencies.test", "project_id"),
					resource.TestCheckTypeSetElemAttr("pgrmongodb_appfunctiondependencies.test", "dependencies.*", "uuidv1 1.6.14"),
					resource.TestCheckResourceAttrSet("pgrmongodb_appfunctiondependencies.test", "appservices_app_id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCheckPGRMongoDBAppFunctionDependenciesConfig(project_id string, appservices_app_id string, id int) string {
	if id == 1 {
		return fmt.Sprintf(`
		resource "pgrmongodb_appfunctiondependencies" "test" {
			project_id = "%s"
			appservices_app_id = "%s"
			dependencies = ["uuidv1 1.6.14"]
		}
		`, project_id, appservices_app_id)
	} else {
		return fmt.Sprintf(`
		resource "pgrmongodb_appfunctiondependencies" "test" {
			project_id = "%s"
			appservices_app_id = "%s"
			dependencies = ["uuidv1 1.6.14", "simple-test-package 0.2.2"]
		}
		`, project_id, appservices_app_id)
	}
}
