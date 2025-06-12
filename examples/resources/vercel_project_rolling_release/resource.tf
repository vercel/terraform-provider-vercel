resource "vercel_project" "example" {
	name = "example-project"
	skew_protection = "12 hours"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	depends_on = [vercel_project.example]
	rolling_release = {
		enabled          = true
		advancement_type = "manual-approval"
		stages = [
			{
				require_approval  = true
				target_percentage = 20
			},
			{
				require_approval  = true
				target_percentage = 50
			},
			{
				require_approval  = true
				target_percentage = 100
			}
		]
	}
}
