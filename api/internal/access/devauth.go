package access

import (
	"context"
	"net/url"
	"strings"

	"github.com/sakib412/ekman/auth"
)

// DevProvider signs in one fixed email without any external provider. It is an ordinary
// auth.Provider, so ekman runs its usual flow (signed state cookie, Directory, session cookie):
// GET /auth/dev/start redirects straight to /auth/dev/callback, whose Exchange returns the
// identity. Only ever register it through Providers, which refuses non-local origins.
type DevProvider struct{ Email string }

// Name implements auth.Provider.
func (DevProvider) Name() string { return "dev" }

// AuthCodeURL implements auth.Provider: skip the consent screen and go to the callback.
func (DevProvider) AuthCodeURL(state, _, redirectURL string) string {
	q := url.Values{"state": {state}, "code": {"dev"}}
	return redirectURL + "?" + q.Encode()
}

// Exchange implements auth.Provider.
func (p DevProvider) Exchange(_ context.Context, _, _, _ string) (auth.Identity, error) {
	name := p.Email
	if i := strings.IndexByte(name, '@'); i > 0 {
		name = name[:i]
	}
	return auth.Identity{Provider: "dev", Subject: p.Email, Email: p.Email, EmailVerified: true, Name: name}, nil
}

// ProviderConfig is what Providers needs from the environment.
type ProviderConfig struct {
	PublicURL          string
	GoogleClientID     string
	GoogleClientSecret string
	DevAdminEmail      string
}

// Providers returns the sign-in providers: Google when configured, plus the dev provider ONLY
// when PublicURL is a local origin (localhost, *.localhost) and DevAdminEmail is set.
func Providers(c ProviderConfig) []auth.Provider {
	var ps []auth.Provider
	if c.GoogleClientID != "" {
		ps = append(ps, auth.Google(c.GoogleClientID, c.GoogleClientSecret))
	}
	if DevSignInEnabled(c) {
		ps = append(ps, DevProvider{Email: c.DevAdminEmail})
	}
	return ps
}

// DevSignInEnabled is the one rule for /auth/dev/start.
func DevSignInEnabled(c ProviderConfig) bool {
	return c.DevAdminEmail != "" && c.PublicURL != "" && auth.IsLocalOrigin(c.PublicURL)
}
