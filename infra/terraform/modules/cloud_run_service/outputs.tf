output "service_name" {
  value = google_cloud_run_v2_service.this.name
}

output "service_url" {
  value = data.google_cloud_run_service.legacy.status[0].url
}
