data "vercel_project" "example" {
	name = "example-project"
}

data "vercel_project_routes" "example" {
	project_id = data.vercel_project.example.id
}
