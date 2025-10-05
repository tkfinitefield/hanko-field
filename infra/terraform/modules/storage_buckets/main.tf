locals {
  bucket_configs = {
    for key, value in var.buckets : key => merge(value, {
      name = lookup(var.name_override, key, key)
    })
  }
}

resource "google_storage_bucket" "this" {
  for_each = local.bucket_configs

  name          = each.value.name
  location      = try(each.value.location, null) != null ? each.value.location : "asia"
  project       = var.project_id
  storage_class = try(each.value.storage_class, "STANDARD")

  uniform_bucket_level_access = try(each.value.uniform_access, true)

  dynamic "retention_policy" {
    for_each = try(each.value.retention_period_seconds, null) == null ? [] : [each.value.retention_period_seconds]
    content {
      retention_period = retention_policy.value
    }
  }

  versioning {
    enabled = try(each.value.versioning, false)
  }

  labels = try(each.value.labels, null)

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      age = 365
    }
  }
}
