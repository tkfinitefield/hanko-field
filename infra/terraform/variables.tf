variable "project_id" {
  description = "Google Cloud project ID"
  type        = string
}

variable "region" {
  description = "Primary region for regional services"
  type        = string
  default     = "asia-northeast1"
}

variable "location" {
  description = "Multi-regional location for storage/Firestore if different from region"
  type        = string
  default     = "asia"
}

variable "environment" {
  description = "Deployment environment identifier (dev, stg, prod)"
  type        = string
}

variable "cloud_run_image" {
  description = "Container image for Cloud Run service"
  type        = string
}

variable "vpc_connector" {
  description = "Name of the Serverless VPC connector for Cloud Run"
  type        = string
  default     = null
}

variable "min_instances" {
  description = "Minimum Cloud Run instances"
  type        = number
  default     = 1
}

variable "max_instances" {
  description = "Maximum Cloud Run instances"
  type        = number
  default     = 10
}

variable "ingress" {
  description = "Cloud Run ingress setting"
  type        = string
  default     = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"
}

variable "psp_topics" {
  description = "Map of Pub/Sub topic definitions"
  type = map(object({
    subscriptions = optional(map(object({
      ack_deadline_seconds = optional(number, 30)
      dead_letter_topic    = optional(string)
      max_delivery_attempts = optional(number, 5)
      retry_min_backoff    = optional(string, "10s")
      retry_max_backoff    = optional(string, "600s")
      push_endpoint        = optional(string)
      oidc_service_account = optional(string)
    })), {})
  }))
  default = {
    ai_jobs = {
      subscriptions = {
        ai-worker = {
          ack_deadline_seconds = 60
        }
      }
    }
    webhook_retries = {
      subscriptions = {
        webhook-dlq = {
          ack_deadline_seconds = 300
        }
      }
    }
    export_jobs = {
      subscriptions = {}
    }
  }
}

variable "storage_buckets" {
  description = "Storage buckets to create keyed by short name"
  type = map(object({
    location                 = optional(string)
    uniform_access           = optional(bool, true)
    retention_period_seconds = optional(number)
    storage_class            = optional(string, "STANDARD")
    versioning               = optional(bool, false)
    labels                   = optional(map(string), {})
  }))
  default = {
    design_assets = {
      retention_period_seconds = 60 * 60 * 24 * 180
      labels = {
        purpose = "assets"
      }
    }
    ai_suggestions = {
      retention_period_seconds = 60 * 60 * 24 * 30
      labels = {
        purpose = "ai"
      }
    }
    exports = {
      labels = {
        purpose = "exports"
      }
    }
    invoices = {
      versioning = true
      labels = {
        purpose = "invoices"
      }
    }
  }
}

variable "scheduler_jobs" {
  description = "Cloud Scheduler job definitions"
  type = map(object({
    schedule              = string
    http_method           = optional(string, "POST")
    uri                   = string
    oidc_service_account  = string
    body                  = optional(string)
    time_zone             = optional(string, "Asia/Tokyo")
  }))
  default = {
    cleanup_reservations = {
      schedule             = "*/15 * * * *"
      uri                  = "https://api-cleanup-placeholder/run"
      oidc_service_account = "svc-api-scheduler"
    }
    stock_safety_notify = {
      schedule             = "0 * * * *"
      uri                  = "https://api-stock-placeholder/run"
      oidc_service_account = "svc-api-scheduler"
    }
  }
}

variable "secret_ids" {
  description = "Secret Manager secrets keyed by logical name"
  type = map(object({
    replication = optional(string, "automatic")
  }))
  default = {
    stripe_api_key = {}
    stripe_webhook_secret = {}
    paypal_secret = {}
    ai_worker_token = {}
    webhook_signing = {}
  }
}

variable "service_accounts" {
  description = "Service accounts to create keyed by short name"
  type = map(object({
    description = optional(string, "")
    roles       = list(string)
  }))
  default = {
    api_runtime = {
      description = "Cloud Run runtime principal"
      roles = [
        "roles/run.invoker",
        "roles/firestore.user",
        "roles/datastore.user",
        "roles/storage.objectAdmin",
        "roles/pubsub.publisher",
        "roles/logging.logWriter"
      ]
    }
    scheduler_invoker = {
      description = "Cloud Scheduler invoker for internal endpoints"
      roles = [
        "roles/run.invoker"
      ]
    }
    ai_worker = {
      description = "AI worker service account"
      roles = [
        "roles/pubsub.subscriber",
        "roles/storage.objectViewer"
      ]
    }
  }
}
