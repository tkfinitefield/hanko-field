variable "project_id" {
  type        = string
  description = "Project ID"
}

variable "secrets" {
  description = "Secret IDs keyed by logical name"
  type        = map(string)
}
