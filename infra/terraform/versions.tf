# Cloud accounts for binarysust.org, as code: Cloudflare (DNS, tunnel, Pages), Supabase (Postgres) and the
# GitHub Actions secrets that tie them together. The VPS itself and app deploys are not managed here
# (infra/vps/setup.sh and .github/workflows/deploy-api.yml).
#
# State lives in HCP Terraform (free, no card): organization from TF_CLOUD_ORGANIZATION, workspace
# `binarysust-prod` in **Local** execution mode, so plans and applies run in GitHub Actions
# (.github/workflows/terraform.yml) and HCP Terraform only stores the state. State holds secrets
# (tunnel token, database password, deploy token): keep the workspace private.
terraform {
  required_version = ">= 1.17"

  cloud {
    workspaces {
      name = "binarysust-prod"
    }
  }

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.29"
    }
    supabase = {
      source  = "supabase/supabase"
      version = "~> 1.11"
    }
    github = {
      source  = "integrations/github"
      version = "~> 6.13"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.7"
    }
  }
}

# Credentials come from the environment, never from variables or files:
#   CLOUDFLARE_API_TOKEN   SUPABASE_ACCESS_TOKEN   GITHUB_TOKEN (+ GITHUB_OWNER)
provider "cloudflare" {}
provider "supabase" {}
provider "github" {}
