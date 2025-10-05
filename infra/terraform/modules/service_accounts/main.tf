locals {
  accounts = {
    for key, value in var.service_accounts : key => {
      account_id   = replace("${var.name_prefix}-${key}", "_", "-")
      display_name = title(replace(key, "_", " "))
      description  = coalesce(value.description, "")
      roles        = value.roles
    }
  }

  account_roles = flatten([
    for account_key, value in local.accounts : [
      for role in value.roles : {
        key  = "${account_key}-${replace(role, "roles/", "")}"
        sa   = account_key
        role = role
      }
    ]
  ])
}

resource "google_service_account" "this" {
  for_each     = local.accounts
  account_id   = each.value.account_id
  display_name = each.value.display_name
  description  = each.value.description
}

resource "google_project_iam_member" "bindings" {
  for_each = {
    for item in local.account_roles : item.key => item
  }

  project = var.project_id
  role    = each.value.role
  member  = "serviceAccount:${google_service_account.this[each.value.sa].email}"

  depends_on = [google_service_account.this]
}
