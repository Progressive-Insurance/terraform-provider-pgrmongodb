resource "pgrmongodb_appfunctiondependencies" "appfunctiondependencies" {
  project_id = "<MONGODB ATLAS PROJECT/GROUP ID>"
  appservices_app_id = pgrmongodb_appservicesapp.app.id
  dependencies = [
    "simple-test-package 0.2.2",
    "uuidv1 1.6.14"
  ]
}
