resource "google_pubsub_topic" "topics" {
  for_each = var.topics

  name                        = each.key
  project                     = var.project_id
  message_retention_duration  = each.value.message_retention_duration
  kms_key_name                = try(each.value.kms_key_name, null)
  labels                      = try(each.value.labels, null)
}

resource "google_pubsub_subscription" "subscriptions" {
  for_each = {
    for topic_name, topic in var.topics :
    for sub_name, sub in lookup(topic, "subscriptions", {}) :
    "${topic_name}:${sub_name}" => {
      topic_name = topic_name
      subscription_name = sub_name
      config = sub
    }
  }

  name  = replace(each.value.subscription_name, " ", "-")
  topic = google_pubsub_topic.topics[each.value.topic_name].name

  ack_deadline_seconds = try(each.value.config.ack_deadline_seconds, 30)
  labels               = try(each.value.config.labels, null)

  dynamic "retry_policy" {
    for_each = [1]
    content {
      minimum_backoff = try(each.value.config.retry_min_backoff, "10s")
      maximum_backoff = try(each.value.config.retry_max_backoff, "600s")
    }
  }

  dynamic "dead_letter_policy" {
    for_each = try(each.value.config.dead_letter_topic, null) == null ? [] : [each.value.config.dead_letter_topic]
    content {
      dead_letter_topic     = each.value.config.dead_letter_topic
      max_delivery_attempts = try(each.value.config.max_delivery_attempts, 5)
    }
  }

  dynamic "push_config" {
    for_each = try(each.value.config.push_endpoint, null) == null ? [] : [each.value.config.push_endpoint]
    content {
      push_endpoint = each.value.config.push_endpoint

      dynamic "oidc_token" {
        for_each = try(each.value.config.oidc_service_account, null) == null ? [] : [each.value.config.oidc_service_account]
        content {
          service_account_email = each.value.config.oidc_service_account
        }
      }
    }
  }
}
