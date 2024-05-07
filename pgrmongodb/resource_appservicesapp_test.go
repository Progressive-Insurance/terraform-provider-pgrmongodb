package pgrmongodb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPGRMongoDBAppServicesApp(t *testing.T) {
	project_id := "000000000000000000000000"
	cluster_name := "progressive_is_awesome"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccCheckPGRMongoDBAppServicesAppConfig(project_id, cluster_name, "TerraformApp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("pgrmongodb_appservicesapp.test", "project_id"),
					resource.TestCheckResourceAttr("pgrmongodb_appservicesapp.test", "appservices_app_name", "TerraformApp"),
					resource.TestCheckResourceAttrSet("pgrmongodb_appservicesapp.test", "cluster_name"),
					resource.TestCheckResourceAttrSet("pgrmongodb_appservicesapp.test", "linked_datasource_id"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccCheckPGRMongoDBAppServicesAppConfig(project_id, cluster_name, "TerraformApp2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("pgrmongodb_appservicesapp.test", "project_id"),
					resource.TestCheckResourceAttr("pgrmongodb_appservicesapp.test", "appservices_app_name", "TerraformApp2"),
					resource.TestCheckResourceAttrSet("pgrmongodb_appservicesapp.test", "cluster_name"),
					resource.TestCheckResourceAttrSet("pgrmongodb_appservicesapp.test", "linked_datasource_id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCheckPGRMongoDBAppServicesAppConfig(project_id string, cluster_name string, app_name string) string {
	if app_name == "TerraformApp" {
		return fmt.Sprintf(`
		resource "pgrmongodb_appservicesapp" "test" {
			project_id = "%s"
			cluster_name = "%s"
		}
		`, project_id, cluster_name)
	} else {
		return fmt.Sprintf(`
		resource "pgrmongodb_appservicesapp" "test" {
			project_id = "%s"
			cluster_name = "%s"
			appservices_app_name = "%s"
		}
		`, project_id, cluster_name, app_name)
	}
}
