variable "cloudflare_account_id" {
  description = "Cloudflare account ID (dashboard sidebar). Not a secret."
  type        = string
}

variable "supabase_organization_id" {
  description = "Supabase organization slug (Organization settings → General)."
  type        = string
}

variable "domain" {
  description = "The site's apex domain, already added to Cloudflare."
  type        = string
  default     = "binarysust.org"
}

variable "api_port" {
  description = "Port the API listens on at the VPS (infra/vps/binarysust-api.service)."
  type        = number
  default     = 8081
}

variable "github_repositories" {
  description = "Repositories (in GITHUB_OWNER) that deploy to Cloudflare Pages and get its secrets. Add binarysust-media once it exists."
  type        = set(string)
  default     = ["binarysust-web"]
}
