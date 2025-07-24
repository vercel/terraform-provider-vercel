resource "vercel_project" "example" {
	name = "example-project"
	skew_protection = "12 hours"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	advancement_type = "manual-approval"
	stages = [
		{
			target_percentage = 20
		},
		{
			target_percentage = 50
		}
	]
}
