# Actions secrets the Pages deploys need, in every repository in var.github_repositories.
# The VPS deploy secrets (VPS_*) and the SNAPSHOT_KV_* pair stay out of Terraform on purpose: an SSH key and an
# API token would otherwise sit in state. Set those with `gh secret set` (docs/DEPLOY.md step 5).

resource "github_actions_secret" "cloudflare_account_id" {
  for_each = var.github_repositories

  repository  = each.value
  secret_name = "CLOUDFLARE_ACCOUNT_ID"
  value       = var.cloudflare_account_id
}

resource "github_actions_secret" "cloudflare_api_token" {
  for_each = var.github_repositories

  repository  = each.value
  secret_name = "CLOUDFLARE_API_TOKEN"
  value       = cloudflare_api_token.pages_deploy.value
}
