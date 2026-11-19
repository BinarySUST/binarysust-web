output "api_origin" {
  description = "Pages Function proxy target (apps/web/wrangler.toml API_ORIGIN)."
  value       = "https://${local.api_host}"
}

output "tunnel_token" {
  description = "Install cloudflared on the VPS with it: sudo cloudflared service install <token>"
  value       = data.cloudflare_zero_trust_tunnel_cloudflared_token.api.token
  sensitive   = true
}

output "supabase_project_ref" {
  value = supabase_project.main.id
}

output "supabase_pooler" {
  description = "Pooler connection strings; use the session pooler for DATABASE_URL."
  value       = data.supabase_pooler.main.url
  sensitive   = true
}

output "database_password" {
  description = "Initial postgres password (ignored after creation; an imported project keeps its own)."
  value       = random_password.db.result
  sensitive   = true
}
