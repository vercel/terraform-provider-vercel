# Optional: reject visible entries that have not been approved.
# This does not establish full coverage: complete is currently always false.
variable "approved_protection_entries" {
  type    = set(string)
  default = []
}

data "vercel_project_protection_inventory" "approved" {
  project_id = "prj_example"
  team_id    = "team_example"

  lifecycle {
    postcondition {
      condition = alltrue([
        for entry in self.entries :
        contains(var.approved_protection_entries, entry.approval_key)
      ])
      error_message = "Unapproved protection access exists."
    }
  }
}
