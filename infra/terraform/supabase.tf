# The production database. The API connects through the session pooler (IPv4, prepared statements);
# the `supabase_pooler` output has its connection strings. Grants (revoking the Data API
# roles) are SQL, applied once from the SQL editor: docs/DEPLOY.md step 1.

resource "random_password" "db" {
  length  = 40
  special = false # URL-safe inside DATABASE_URL
}

resource "supabase_project" "main" {
  organization_id   = var.supabase_organization_id
  name              = "binarysust"
  region            = "ap-southeast-1" # Singapore, closest to Sylhet
  database_password = random_password.db.result

  lifecycle {
    # Changing the password here would not rotate it in /etc/binarysust-api.env on the VPS.
    # An imported project keeps the password it already has.
    ignore_changes  = [database_password, instance_size]
    prevent_destroy = true
  }
}

data "supabase_pooler" "main" {
  project_ref = supabase_project.main.id
}
