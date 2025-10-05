resource "google_cloud_scheduler_job" "jobs" {
  for_each = var.jobs

  name        = each.key
  description = "${each.key} job"
  schedule    = each.value.schedule
  time_zone   = try(each.value.time_zone, "UTC")

  http_target {
    http_method = try(each.value.http_method, "POST")
    uri         = each.value.uri
    body        = try(each.value.body, null) == null ? null : base64encode(each.value.body)

    oidc_token {
      service_account_email = try(each.value.oidc_service_account, var.service_account_email)
      audience              = each.value.uri
    }
  }

  project  = var.project_id
  region   = var.location
}
