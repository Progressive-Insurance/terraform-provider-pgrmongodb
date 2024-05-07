data "pgrmongodb_appfunctionexecute" "executor" {
	project_id = "<MONGODB ATLAS PROJECT/GROUP ID>"
	appservices_app_id = pgrmongodb_appservicesapp.app.id
	function_name = "my_terraform_mongodbatlas_function"
	function_args = ["arg1", "arg2"]
	execute_next_run = true
	execution_timeout = 10
}
