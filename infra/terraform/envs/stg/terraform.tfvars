project_id       = "hanko-field-stg"
region           = "asia-northeast1"
location         = "asia-northeast1"
environment      = "stg"
cloud_run_image  = "asia-northeast1-docker.pkg.dev/hanko-field-stg/api/api:stg"
vpc_connector    = "projects/hanko-field-stg/locations/asia-northeast1/connectors/api-stg"
min_instances    = 2
max_instances    = 10

scheduler_jobs = {
  cleanup_reservations = {
    schedule             = "*/15 * * * *"
    uri                  = "https://api-stg.internal.hanko-field.app/internal/maintenance/cleanup-reservations"
    oidc_service_account = "svc-api-scheduler@hanko-field-stg.iam.gserviceaccount.com"
    time_zone            = "Asia/Tokyo"
  }
  stock_safety_notify = {
    schedule             = "0 */2 * * *"
    uri                  = "https://api-stg.internal.hanko-field.app/internal/maintenance/stock-safety-notify"
    oidc_service_account = "svc-api-scheduler@hanko-field-stg.iam.gserviceaccount.com"
    time_zone            = "Asia/Tokyo"
  }
}
