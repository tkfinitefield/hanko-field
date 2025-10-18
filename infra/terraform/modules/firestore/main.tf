resource "google_firestore_database" "default" {
  project     = var.project_id
  name        = "(default)"
  location_id = var.location
  type        = "FIRESTORE_NATIVE"
}

resource "google_firestore_field" "stock_reservations_ttl" {
  project     = var.project_id
  database    = google_firestore_database.default.name
  collection  = "stockReservations"
  field_path  = "expiresAt"

  ttl_config {
    state = "ENABLED"
  }
}

resource "google_firestore_index" "orders_user_created" {
  project    = var.project_id
  database   = google_firestore_database.default.name
  collection = "orders"

  fields {
    field_path = "userRef"
    order      = "ASCENDING"
  }

  fields {
    field_path = "createdAt"
    order      = "DESCENDING"
  }
}

resource "google_firestore_index" "orders_status_updated" {
  project    = var.project_id
  database   = google_firestore_database.default.name
  collection = "orders"

  fields {
    field_path = "status"
    order      = "ASCENDING"
  }

  fields {
    field_path = "updatedAt"
    order      = "DESCENDING"
  }
}
