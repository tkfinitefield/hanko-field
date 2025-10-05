variable "project_id" {
  type        = string
  description = "Project ID"
}

variable "topics" {
  description = "Pub/Sub topics and subscriptions"
  type = map(object({
    message_retention_duration = optional(string, "604800s")
    kms_key_name               = optional(string)
    labels                     = optional(map(string), {})
    subscriptions = optional(map(object({
      ack_deadline_seconds = optional(number, 30)
      dead_letter_topic    = optional(string)
      max_delivery_attempts = optional(number, 5)
      retry_min_backoff    = optional(string, "10s")
      retry_max_backoff    = optional(string, "600s")
      push_endpoint        = optional(string)
      oidc_service_account = optional(string)
      labels               = optional(map(string), {})
    })), {})
  }))
}
