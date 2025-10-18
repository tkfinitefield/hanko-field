resource "google_secret_manager_secret" "secrets" {
  for_each = var.secrets

  project   = var.project_id
  secret_id = each.value

  replication {
    automatic = true
  }

  labels = {
    managed-by = "terraform"
  }
}
