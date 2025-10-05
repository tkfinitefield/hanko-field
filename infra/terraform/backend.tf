terraform {
  backend "gcs" {
    bucket = "REPLACE_WITH_TERRAFORM_STATE_BUCKET"
    prefix = "api/terraform-state"
  }
}
