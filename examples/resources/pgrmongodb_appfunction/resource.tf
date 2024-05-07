resource "pgrmongodb_appfunction" "appfunction" {
  project_id = "<MONGODB ATLAS PROJECT/GROUP ID>"
  appservices_app_id = pgrmongodb_appservicesapp.app.id
  function_name = "my_terraform_mongodbatlas_function"
  function_code = <<EOT
exports = async (changeEvent) => {
	console.log('Hello World!');
}
EOT
}
