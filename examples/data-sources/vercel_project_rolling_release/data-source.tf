data "vercel_project" "example" {
	name = "example-project"
}

data "vercel_project_rolling_release" "example" {
	project_id = data.vercel_project_rolling_release.example.id
}