# Deploying Binary SUST ($0, no credit card)

Set this up once with a **club-owned Google account** (for example binary.sust@gmail.com), not a
personal one, so the next committee inherits it. Use the same account for every service below.

| Piece                      | Service                                                           | Free tier used                      |
| -------------------------- | ----------------------------------------------------------------- | ----------------------------------- |
| Public site + `/api` proxy | Cloudflare Pages                                                  | static hosting, unlimited bandwidth |
| Go API                     | Oracle Cloud VPS + Cloudflare Tunnel                              | Always Free ARM VM; tunnel is free  |
| Database                   | Supabase                                                          | 500 MB Postgres                     |
| Media                      | GitHub repo `BinarySUST/binarysust-media` + its own Pages project |                                     |
| Builds                     | GitHub Actions                                                    | unlimited on public repos           |
| Sign-in                    | Google OAuth                                                      | no billing account needed           |
| Mail (optional)            | Brevo                                                             | 300 emails/day                      |

## 0. Terraform (Cloudflare, Supabase, GitHub secrets)

`infra/terraform/` creates the cloud side: the Supabase project, the Cloudflare Tunnel and its `api.` DNS record,
both Pages projects with their custom domains and DNS, the www → apex redirect, and a Pages-only deploy token that
it writes to the repository's `CLOUDFLARE_API_TOKEN` / `CLOUDFLARE_ACCOUNT_ID` secrets. It does not touch the VPS
or deploy code. Actions → **terraform** plans on every pull request that changes it, and applies only when run by
hand from `main` with **apply** ticked.

Do this once, after the domain is on Cloudflare (step 4, "Domain") and before steps 1, 3 and 4:

1. **HCP Terraform** (app.terraform.io, free, no card): create an organization, then a workspace named
   `binarysust-prod` of type **CLI-driven**, and set its Settings → Execution Mode to **Local** (HCP only stores
   state; state holds secrets, so keep the organization to the committee). Create a team or user API token.
2. **Tokens** (all from the club accounts):
   - Cloudflare → My Profile → API Tokens → custom token with Account: _Cloudflare Tunnel: Edit_,
     _Cloudflare Pages: Edit_, _Account Settings: Read_; Zone (binarysust.org): _DNS: Edit_,
     _Single Redirect: Edit_, _Zone: Read_; User: _API Tokens: Edit_ (to create the deploy token).
   - Supabase → Account → Access Tokens → new token.
   - GitHub → fine-grained PAT on `BinarySUST/binarysust-web`: **Secrets: read and write**.
3. In `BinarySUST/binarysust-web` → Settings → Secrets and variables → Actions:
   - Secrets: `TF_API_TOKEN`, `TF_CLOUDFLARE_API_TOKEN`, `SUPABASE_ACCESS_TOKEN`, `TF_GITHUB_TOKEN`
   - Variables: `TF_CLOUD_ORGANIZATION`, `CLOUDFLARE_ACCOUNT_ID`, `SUPABASE_ORGANIZATION_ID` (the org slug)
4. Actions → **terraform** → Run workflow on `main`, **apply** ticked. Check the plan in the run summary first
   by running it once without **apply**.

**Already created by hand?** Import before the first apply instead of letting Terraform create duplicates
(from `infra/terraform`, with the same env vars as the workflow):
`terraform import supabase_project.main <project-ref>`,
`terraform import cloudflare_pages_project.site <account-id>/binarysust`,
`terraform import 'cloudflare_pages_domain.this["apex"]' <account-id>/binarysust/binarysust.org`,
`terraform import 'cloudflare_dns_record.pages["apex"]' <zone-id>/<record-id>`, and so on for `www`, `media`
and the redirect ruleset (`cloudflare_ruleset.redirects`, `zones/<zone-id>/<ruleset-id>`). Delete the
manually created tunnel instead of importing it; Terraform makes a fresh one.

Read secrets back from state with `terraform output -raw <name>`: `tunnel_token`, `database_password`,
`supabase_pooler`.

## 1. Supabase (database only)

Supabase is used only as Postgres: no Supabase Auth, Storage or Data API. Sign-in, uploads and the API are ours.
Terraform (step 0) creates project `binarysust` in Singapore (`ap-southeast-1`) with a generated password
(`terraform output -raw database_password`). Then, once:

1. **Turn off the Data API** (Project Settings → Data API → disable). Left on, Supabase serves tables in
   `public` over REST and GraphQL to anyone holding the project's anon key.
2. In the SQL editor, take away what Supabase grants its Data API roles, and create the read-only role for the
   site build:
   ```sql
   -- Supabase-only roles get nothing in the API's schema, now or for tables created later.
   REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon, authenticated;
   REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM anon, authenticated;
   REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM anon, authenticated;
   ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE ALL ON TABLES FROM anon, authenticated;
   ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE ALL ON SEQUENCES FROM anon, authenticated;
   ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE ALL ON FUNCTIONS FROM anon, authenticated;

   CREATE ROLE site_reader LOGIN PASSWORD '<generate one>';
   GRANT CONNECT ON DATABASE postgres TO site_reader;
   GRANT USAGE ON SCHEMA public TO site_reader;
   GRANT SELECT ON ALL TABLES IN SCHEMA public TO site_reader;
   ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT SELECT ON TABLES TO site_reader;
   ```
3. Connect → **Session pooler** connection string (IPv4, supports prepared statements; the direct connection is
   IPv6-only on the free plan). With the `postgres` password it is `DATABASE_URL` for the VPS (step 3). With
   user `site_reader.<project-ref>` and its password it is `SNAPSHOT_DATABASE_URL` for GitHub Actions (step 5).

## 2. Google sign-in

1. console.cloud.google.com → new project `binarysust` (do **not** enable billing).
2. APIs & Services → OAuth consent screen → External, app name "Binary SUST", then publish it.
3. Credentials → Create OAuth client ID → Web application. Authorized redirect URIs:
   - `https://binarysust.org/api/v1/auth/google/callback`
   - `http://binarysust.localhost/api/v1/auth/google/callback` (local dev; if Google rejects it, use `pnpm dev` with `DEV_ADMIN_EMAIL` instead)
4. Copy the client ID and secret.

## 3. VPS (Go API)

The API runs as a static linux/arm64 binary under systemd on the club's Oracle Cloud VM (Ubuntu 22.04,
aarch64). OpenLiteSpeed already owns ports 80/443 there, so the API listens on port 8081
(closed to the internet by ufw) and Cloudflare Tunnel publishes it as `api.binarysust.org`. Nobody visits that
hostname directly: the site's `/api/*` Pages Function proxies to it, so members only ever see `binarysust.org/api`.

1. **One-time setup** from the repository root (needs an SSH user with sudo):
   ```sh
   scp -r infra/vps <user>@<host>:/tmp/binarysust-vps
   ssh <user>@<host> bash /tmp/binarysust-vps/setup.sh
   ```
   It creates `/opt/binarysust-api/`, installs and enables `binarysust-api.service`, and creates
   `/etc/binarysust-api.env` (root, 0600) from `infra/vps/binarysust-api.env.example`. Re-running it is safe.
2. Fill in the env file on the VPS (`sudo nano /etc/binarysust-api.env`):
   - `DATABASE_URL`: Supabase session pooler string (owner role `postgres.<project-ref>`)
   - `EKMAN_SECRET`: `openssl rand -base64 48`
   - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
   - `GITHUB_DISPATCH_TOKEN`, `GITHUB_MEDIA_TOKEN`: see step 5
   - `BREVO_API_KEY`: optional
3. **Cloudflare Tunnel**: Terraform (step 0) creates tunnel `binarysust-vps`, routes `api.binarysust.org` to
   `localhost:8081` and adds the DNS record. Install the connector on the VPS once, with the token from
   `terraform output -raw tunnel_token`:
   ```sh
   curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null
   echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main' \
     | sudo tee /etc/apt/sources.list.d/cloudflared.list
   sudo apt-get update && sudo apt-get install -y cloudflared
   sudo cloudflared service install <token>
   ```
   The tunnel shows as Healthy in Zero Trust → Networks → Tunnels.
4. GitHub secrets for `deploy-api` (step 5), then Actions → **deploy-api** → Run workflow. It runs the Go tests,
   builds, uploads to `/opt/binarysust-api/releases/<sha>/`, switches `current`, restarts the service (migrations
   run on boot) and checks `/healthz`, rolling back to the previous release if it fails. After that, every push
   to `main` that touches `apps/api/**` deploys. Check `https://api.binarysust.org/healthz`.

Useful on the VPS: `sudo systemctl status binarysust-api`, `sudo journalctl -u binarysust-api -f`.

## 4. Cloudflare Pages (site)

Terraform (step 0) creates the `binarysust` and `binarysust-media` Pages projects, their custom domains
(`binarysust.org`, `www.binarysust.org`, `media.binarysust.org`) with DNS records, the www → apex 301 redirect,
and the deploy token in GitHub secrets. `API_ORIGIN` (`https://api.binarysust.org`) lives in
`apps/web/wrangler.toml`, which every deploy uploads. Only the domain itself is set up by hand:

### Domain: binarysust.org

The club owns `binarysust.org` at Namecheap (expires 2027-09-20). Cloudflare's free plan hosts its DNS, and no card is needed.

1. Cloudflare dashboard → **Add a domain** → `binarysust.org` → Free plan. Cloudflare shows two nameservers.
2. Namecheap → Domain List → `binarysust.org` → Nameservers → **Custom DNS**, then paste both Cloudflare nameservers. It usually takes effect within an hour, and at most 24 hours.
3. Keep `binary.sust@gmail.com` as the Namecheap contact, and turn on auto-renew. If the domain lapses, the site, sign-in and media all go down.

## 5. GitHub

In `BinarySUST/binarysust-web` → Settings → Secrets and variables → Actions:

- `SNAPSHOT_DATABASE_URL`: Supabase `site_reader` string
- `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`: written by Terraform (step 0)
- `VPS_HOST`, `VPS_USER`: the VM's address and an SSH user with passwordless sudo
- `VPS_SSH_KEY`: that user's private key
- `VPS_KNOWN_HOSTS`: output of `ssh-keyscan -t ed25519 <host>`, so the workflow pins the host key

Tokens the API uses (paste them into `/etc/binarysust-api.env` on the VPS): create **fine-grained personal access tokens** from the club's
GitHub account, owned by the `BinarySUST` org:

- `GITHUB_DISPATCH_TOKEN`: repository `binarysust-web`, permission **Contents: read and write** (needed for `repository_dispatch`).
- `GITHUB_MEDIA_TOKEN`: repository `binarysust-media`, permission **Contents: read and write**.

Set their expiry to one year and add renewing them to the committee handover checklist. A GitHub App is
the long-term replacement (see ARCHITECTURE.md, open questions).

`binarysust-media` deploys itself to the `binarysust-media` Pages project on every push. It needs the same two Cloudflare secrets: add `binarysust-media` to `github_repositories` in
`infra/terraform/variables.tf` once that repository exists.

## 6. First publish

Actions → **publish** → Run workflow. Open `https://binarysust.org` and sign in with Google once.
Then make yourself the first admin from the Supabase SQL editor. Roles belong to a _person_ (a public
profile), so create yours, link it, and grant a permanent admin role:

```sql
INSERT INTO persons (handle, name, email) VALUES ('your-handle', 'Your Name', 'you@gmail.com');
UPDATE accounts a SET person_id = p.id FROM persons p WHERE p.email = a.email AND a.email = 'you@gmail.com';
INSERT INTO role_grants (person_id, role, term_id)
SELECT id, 'admin', NULL FROM persons WHERE email = 'you@gmail.com';
```

Emails must be lowercase.

After that, give out every other role from the dashboard's **Committee & roles** section.

## Yearly handover checklist

- Dashboard → Committee & roles → create the new term, fill its seats, click **Make current**.
- Rotate `GITHUB_*` and `TF_*` tokens if they expire within 3 months.
- Hand over the HCP Terraform organization (it holds the state) along with the other club accounts.
- Check the `binarysust.org` renewal date at Namecheap (next: 2027-09-20).
- Make sure at least two people know the club Google account's password and have recovery set up.
