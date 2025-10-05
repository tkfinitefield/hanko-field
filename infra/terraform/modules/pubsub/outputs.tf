output "topic_ids" {
  value = {
    for name, topic in google_pubsub_topic.topics : name => topic.name
  }
}
