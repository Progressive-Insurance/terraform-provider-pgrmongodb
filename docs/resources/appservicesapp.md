---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "pgrmongodb_appservicesapp Resource - terraform-provider-pgrmongodb"
subcategory: ""
description: |-
  Manages a MongoDB Atlas App Services App
---

# pgrmongodb_appservicesapp (Resource)

Manages a MongoDB Atlas App Services App

## Example Usage

```terraform
resource "pgrmongodb_appservicesapp" "appservicesapp" {
  project_id = "<MONGODB ATLAS PROJECT/GROUP ID>"
  cluster_name = "<MONGODB ATLAS CLUSTER NAME>"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cluster_name` (String) Name of the MongoDB Atlas cluster deployed to project.
- `project_id` (String) MongoDB Atlas project identifier. Sometime referred to as group id.

### Optional

- `appservices_app_name` (String) Name of MongoDB Atlas App Services app name to be managed.

### Read-Only

- `id` (String) identifier for resource.
- `linked_datasource_id` (String) Identifier for linked datasource associated to this App Services app.
