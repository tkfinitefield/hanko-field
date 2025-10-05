variable "project_id" {
  type        = string
  description = "Project ID"
}

variable "location" {
  type        = string
  description = "Region for scheduler"
}

variable "service_account_email" {
  type        = string
  description = "Service account used for OIDC invocation"
}

variable "jobs" {
  description = "Scheduler jobs keyed by name"
  type = map(object({
    schedule             = string
    http_method          = optional(string, "POST")
    uri                  = string
    body                 = optional(string)
    time_zone            = optional(string, "UTC")
    oidc_service_account = string
  }))
}
