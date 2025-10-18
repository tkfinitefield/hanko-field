variable "project_id" {
  type        = string
  description = "Project ID"
}

variable "region" {
  type        = string
  description = "Region for Cloud Run"
}

variable "service_name" {
  type        = string
  description = "Cloud Run service name"
}

variable "image" {
  type        = string
  description = "Container image"
}

variable "service_account_email" {
  type        = string
  description = "Service account to run the service"
}

variable "vpc_connector" {
  type        = string
  description = "Optional VPC connector name"
  default     = null
}

variable "min_instances" {
  type        = number
  default     = 1
}

variable "max_instances" {
  type        = number
  default     = 10
}

variable "ingress" {
  type        = string
  default     = "INGRESS_TRAFFIC_INTERNAL_ONLY"
}

variable "env_vars" {
  type        = map(string)
  default     = {}
}

variable "secrets" {
  type = map(object({
    secret  = string
    version = optional(string, "latest")
    env     = string
  }))
  default = {}
}

variable "invokers" {
  type        = list(string)
  description = "List of identities granted run.invoker"
  default     = []
}

variable "environment" {
  type        = string
  description = "Deployment environment"
}
