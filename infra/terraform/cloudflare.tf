data "cloudflare_zone" "site" {
  filter = {
    name = var.domain
  }
}

locals {
  zone_id  = data.cloudflare_zone.site.id
  api_host = "api.${var.domain}"
}

# --- API: Cloudflare Tunnel to the VPS --------------------------------------------------------------------
# cloudflared on the VPS dials out to Cloudflare, so the box needs no open ports and OpenLiteSpeed keeps 80/443.
# Install it once with the `tunnel_token` output: sudo cloudflared service install <token>

resource "cloudflare_zero_trust_tunnel_cloudflared" "api" {
  account_id = var.cloudflare_account_id
  name       = "binarysust-vps"
  config_src = "cloudflare" # ingress below is pushed to cloudflared; no config file on the VPS
}

resource "cloudflare_zero_trust_tunnel_cloudflared_config" "api" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.api.id

  config = {
    ingress = [
      {
        hostname = local.api_host
        service  = "http://localhost:${var.api_port}"
      },
      {
        service = "http_status:404" # catch-all, required last
      },
    ]
  }
}

resource "cloudflare_dns_record" "api" {
  zone_id = local.zone_id
  name    = local.api_host
  type    = "CNAME"
  content = "${cloudflare_zero_trust_tunnel_cloudflared.api.id}.cfargotunnel.com"
  proxied = true
  ttl     = 1 # automatic
  comment = "Cloudflare Tunnel to the Go API on the VPS (Terraform)"
}

data "cloudflare_zero_trust_tunnel_cloudflared_token" "api" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.api.id
}

# --- Site and media: Cloudflare Pages ----------------------------------------------------------------------
# GitHub Actions uploads builds with `wrangler pages deploy`. apps/web/wrangler.toml owns the runtime config
# (API_ORIGIN, compatibility date), so Terraform ignores deployment_configs instead of fighting it.

resource "cloudflare_pages_project" "site" {
  account_id        = var.cloudflare_account_id
  name              = "binarysust"
  production_branch = "main"

  lifecycle {
    ignore_changes = [deployment_configs, build_config]
  }
}

resource "cloudflare_pages_project" "media" {
  account_id        = var.cloudflare_account_id
  name              = "binarysust-media"
  production_branch = "main"

  lifecycle {
    ignore_changes = [deployment_configs, build_config]
  }
}

locals {
  pages_domains = {
    apex  = { name = var.domain, project = cloudflare_pages_project.site }
    www   = { name = "www.${var.domain}", project = cloudflare_pages_project.site }
    media = { name = "media.${var.domain}", project = cloudflare_pages_project.media }
  }
}

resource "cloudflare_pages_domain" "this" {
  for_each = local.pages_domains

  account_id   = var.cloudflare_account_id
  project_name = each.value.project.name
  name         = each.value.name
}

resource "cloudflare_dns_record" "pages" {
  for_each = local.pages_domains

  zone_id = local.zone_id
  name    = each.value.name
  type    = "CNAME" # flattened at the apex by Cloudflare
  content = "${each.value.project.name}.pages.dev"
  proxied = true
  ttl     = 1
  comment = "Cloudflare Pages (Terraform)"
}

# One canonical URL: www.binarysust.org/* → https://binarysust.org/* (301).
resource "cloudflare_ruleset" "redirects" {
  zone_id     = local.zone_id
  name        = "redirects"
  description = "Single redirects (Terraform)"
  kind        = "zone"
  phase       = "http_request_dynamic_redirect"

  rules = [
    {
      description = "www to apex"
      expression  = "(http.host eq \"www.${var.domain}\")"
      action      = "redirect"
      action_parameters = {
        from_value = {
          status_code           = 301
          preserve_query_string = true
          target_url = {
            expression = "concat(\"https://${var.domain}\", http.request.uri.path)"
          }
        }
      }
    },
  ]
}

# --- Deploy token for GitHub Actions ------------------------------------------------------------------------
# A token that can only edit Pages in this account, written to each repo's CLOUDFLARE_API_TOKEN (github.tf).
# Terraform's own token stays out of the repos.

data "cloudflare_account_api_token_permission_groups_list" "pages_write" {
  account_id = var.cloudflare_account_id
  name       = "Pages Write"
}

resource "cloudflare_api_token" "pages_deploy" {
  name = "binarysust GitHub Actions: Pages deploy (Terraform)"

  policies = [
    {
      effect            = "allow"
      permission_groups = [{ id = data.cloudflare_account_api_token_permission_groups_list.pages_write.result[0].id }]
      resources         = jsonencode({ "com.cloudflare.api.account.${var.cloudflare_account_id}" = "*" })
    },
  ]
}
