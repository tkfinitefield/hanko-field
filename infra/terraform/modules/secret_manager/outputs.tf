output "secret_ids" {
  value = {
    for key, secret in google_secret_manager_secret.secrets : key => secret.secret_id
  }
}
