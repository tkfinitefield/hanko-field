variable "project_id" {
  description = "Project ID where the service accounts will be created"
  type        = string
}

variable "service_accounts" {
  description = "Map of service accounts to create"
  type = map(object({
    description = optional(string, "")
    roles       = list(string)
  }))
}

variable "name_prefix" {
  description = "Prefix applied to service account IDs"
  type        = string
}
