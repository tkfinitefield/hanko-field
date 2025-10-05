output "bucket_names" {
  value = {
    for key, bucket in google_storage_bucket.this : key => bucket.name
  }
}
