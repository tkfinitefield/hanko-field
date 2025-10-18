variable "project_id" {
  type        = string
  description = "Project ID"
}

variable "buckets" {
  description = "Bucket definitions keyed by short name"
  type = map(object({
    location                 = optional(string)
    uniform_access           = optional(bool, true)
    retention_period_seconds = optional(number)
    storage_class            = optional(string, "STANDARD")
    versioning               = optional(bool, false)
    labels                   = optional(map(string), {})
  }))
}

variable "name_override" {
  description = "Map overriding bucket names"
  type        = map(string)
  default     = {}
}
