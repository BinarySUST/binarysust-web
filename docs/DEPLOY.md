# Deploying Binary SUST ($0, no credit card)

Set this up once with a **club-owned Google account** (for example binary.sust@gmail.com), not a
personal one, so the next committee inherits it. Use the same account for every service below.

| Piece                      | Service                                                           | Free tier used                               |
| -------------------------- | ----------------------------------------------------------------- | -------------------------------------------- |
| Public site + `/api` proxy | Cloudflare Pages                                                  | static hosting, unlimited bandwidth          |
| Go API                     | Render                                                            | 1 free web service (sleeps when idle)        |
| Database                   | Neon                                                              | 0.5 GB Postgres, suspends instead of billing |
| Media                      | GitHub repo `BinarySUST/binarysust-media` + its own Pages project |                                              |
| Builds                     | GitHub Actions                                                    | unlimited on public repos                    |
| Sign-in                    | Google OAuth                                                      | no billing account needed                    |
| Mail (optional)            | Brevo                                                             | 300 emails/day                               |

## 1. Neon (database)

1. Sign up at neon.tech, create project `binarysust`, region **Singapore**.
2. Copy the **pooled** connection string. This is `DATABASE_URL` for Render.
3. Create a read-only role for the site build (Neon SQL editor):
   ```sql
   CREATE ROLE site_reader LOGIN PASSWORD '<generate one>';
   GRANT CONNECT ON DATABASE neondb TO site_reader;
   GRANT USAGE ON SCHEMA public TO site_reader;
   GRANT SELECT ON ALL TABLES IN SCHEMA public TO site_reader;
   ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO site_reader;
   ```
   The same connection string with `site_reader` is `SNAPSHOT_DATABASE_URL` for GitHub Actions.

## 2. Google sign-in

1. console.cloud.google.com → new project `binarysust` (do **not** enable billing).
2. APIs & Services → OAuth consent screen → External, app name "Binary SUST", then publish it.
3. Credentials → Create OAuth client ID → Web application. Authorized redirect URIs:
   - `https://binarysust.org/api/v1/auth/google/callback`
   - `http://binarysust.localhost/api/v1/auth/google/callback` (local dev; if Google rejects it, use `make dev` with `DEV_ADMIN_EMAIL` instead)
4. Copy the client ID and secret.

## 3. Render (Go API)

1. Sign up at render.com with GitHub. New → **Blueprint** → pick `BinarySUST/binarysust-web`. Render reads `render.yaml`.
2. Fill in the secret env vars it asks for:
   - `DATABASE_URL`: from Neon (owner role)
   - `EKMAN_SECRET`: `openssl rand -base64 48`
   - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
   - `GITHUB_DISPATCH_TOKEN`, `GITHUB_MEDIA_TOKEN`: see step 5
   - `BREVO_API_KEY`: optional
3. Deploy. The container runs migrations on boot. Check `https://binarysust-api.onrender.com/healthz`.

## 4. Cloudflare Pages (site)

1. Sign up at dash.cloudflare.com. Workers & Pages → Create → Pages → **Direct Upload**, named `binarysust`. Create a second project named `binarysust-media` the same way.
2. In `binarysust` → Settings → Variables: set `API_ORIGIN` = `https://binarysust-api.onrender.com`.
3. My Profile → API Tokens → Create token with the template **Edit Cloudflare Workers** (it includes Pages). Copy it, and copy your Account ID from the dashboard sidebar.

### Domain: binarysust.org

The club owns `binarysust.org` at Namecheap (expires 2027-09-20). Cloudflare's free plan hosts its DNS, and no card is needed.

1. Cloudflare dashboard → **Add a domain** → `binarysust.org` → Free plan. Cloudflare shows two nameservers.
2. Namecheap → Domain List → `binarysust.org` → Nameservers → **Custom DNS**, then paste both Cloudflare nameservers. It usually takes effect within an hour, and at most 24 hours.
3. Pages project `binarysust` → Custom domains → add `binarysust.org` and `www.binarysust.org`. Cloudflare creates the DNS records and certificates.
4. Pages project `binarysust-media` → Custom domains → add `media.binarysust.org`.
5. Rules → **Redirect Rules** → redirect `www.binarysust.org/*` to `https://binarysust.org/${1}` (301), so there is one canonical URL.
6. Keep `binary.sust@gmail.com` as the Namecheap contact, and turn on auto-renew. If the domain lapses, the site, sign-in and media all go down.

## 5. GitHub

In `BinarySUST/binarysust-web` → Settings → Secrets and variables → Actions:

- `SNAPSHOT_DATABASE_URL`: Neon `site_reader` string
- `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`

Tokens the API uses (paste them into Render): create **fine-grained personal access tokens** from the club's
GitHub account, owned by the `BinarySUST` org:

- `GITHUB_DISPATCH_TOKEN`: repository `binarysust-web`, permission **Contents: read and write** (needed for `repository_dispatch`).
- `GITHUB_MEDIA_TOKEN`: repository `binarysust-media`, permission **Contents: read and write**.

Set their expiry to one year and add renewing them to the committee handover checklist. A GitHub App is
the long-term replacement (see ARCHITECTURE.md, open questions).

`binarysust-media` deploys itself to the `binarysust-media` Pages project on every push. It needs the same two Cloudflare secrets.

## 6. First publish

Actions → **publish** → Run workflow. Open `https://binarysust.org` and sign in with Google once.
Then make yourself the first admin from the Neon SQL editor. Roles belong to a _person_ (a public
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
- Rotate `GITHUB_*` tokens if they expire within 3 months.
- Check the `binarysust.org` renewal date at Namecheap (next: 2027-09-20).
- Make sure at least two people know the club Google account's password and have recovery set up.
