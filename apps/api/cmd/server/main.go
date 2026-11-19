// Command server is the Binary SUST API.
//
//	server [serve]            serve on :$PORT (default 8080)
//	server serve --migrate    apply migrations, then serve (Render's free plan has no pre-deploy step)
//	server migrate            apply migrations
//	server openapi [-o file]  print (or write) the OpenAPI document, enums included (no database needed)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // the "pgx" database/sql driver

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/auth"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/mail"
	"github.com/sakib412/ekman/storage"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/access"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/club"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/media"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/republish"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/site"
)

// Config is read from the environment (see .env.example at the repository root and infra/vps/binarysust-api.env.example).
type Config struct {
	DatabaseURL        string // DATABASE_URL
	PublicURL          string // PUBLIC_URL: the site's origin; /api/* is proxied there
	GoogleClientID     string // GOOGLE_CLIENT_ID
	GoogleClientSecret string // GOOGLE_CLIENT_SECRET
	DevAdminEmail      string // DEV_ADMIN_EMAIL: dev sign-in, honored only on a local PUBLIC_URL
	MediaDir           string // MEDIA_DIR: local uploads (development)
	MediaBaseURL       string // MEDIA_BASE_URL: where uploads are served
	BrevoAPIKey        string // BREVO_API_KEY: empty logs mail instead
	MailFrom           string // MAIL_FROM
	DispatchToken      string // GITHUB_DISPATCH_TOKEN: repository_dispatch on content changes
	Repository         string // GITHUB_REPOSITORY: owner/name of the site repo
	MediaToken         string // GITHUB_MEDIA_TOKEN: contents:write on the media repo
	MediaRepository    string // GITHUB_MEDIA_REPOSITORY: owner/name; set it to store uploads on GitHub
	MediaBranch        string // GITHUB_MEDIA_BRANCH: default main
	CloudflareAccount  string // CLOUDFLARE_ACCOUNT_ID
	SnapshotKV         string // SNAPSHOT_KV_NAMESPACE_ID: set it (and SNAPSHOT_KV_TOKEN) to publish snapshots to Workers KV
	SnapshotKVToken    string // SNAPSHOT_KV_TOKEN: "Workers KV Storage: Edit"
	CORS               []string

	cloudflareAPI, githubAPI string // tests point these at fakes
}

// snapshotKV reports whether snapshots are published to Workers KV.
func (c Config) snapshotKV() bool {
	return c.CloudflareAccount != "" && c.SnapshotKV != "" && c.SnapshotKVToken != ""
}

// LoadConfig reads Config from getenv.
func LoadConfig(getenv func(string) string) Config {
	var cors []string
	for _, o := range strings.Split(getenv("CORS_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cors = append(cors, o)
		}
	}
	return Config{
		DatabaseURL:        getenv("DATABASE_URL"),
		PublicURL:          strings.TrimSuffix(getenv("PUBLIC_URL"), "/"),
		GoogleClientID:     getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: getenv("GOOGLE_CLIENT_SECRET"),
		DevAdminEmail:      getenv("DEV_ADMIN_EMAIL"),
		MediaDir:           getenv("MEDIA_DIR"),
		MediaBaseURL:       getenv("MEDIA_BASE_URL"),
		BrevoAPIKey:        getenv("BREVO_API_KEY"),
		MailFrom:           getenv("MAIL_FROM"),
		DispatchToken:      getenv("GITHUB_DISPATCH_TOKEN"),
		Repository:         getenv("GITHUB_REPOSITORY"),
		MediaToken:         getenv("GITHUB_MEDIA_TOKEN"),
		MediaRepository:    getenv("GITHUB_MEDIA_REPOSITORY"),
		MediaBranch:        getenv("GITHUB_MEDIA_BRANCH"),
		CloudflareAccount:  getenv("CLOUDFLARE_ACCOUNT_ID"),
		SnapshotKV:         getenv("SNAPSHOT_KV_NAMESPACE_ID"),
		SnapshotKVToken:    getenv("SNAPSHOT_KV_TOKEN"),
		CORS:               cors,
	}
}

// Server is the wired API.
type Server struct {
	App       *ekman.App
	Svc       *club.Services
	Publisher *republish.Publisher // nil unless snapshots go to Workers KV; Start it when serving
}

// NewApp builds the whole API from cfg over database.
func NewApp(cfg Config, database *db.DB, logger *slog.Logger) (*Server, error) {
	svc := club.New(database, club.Deps{
		Mail:      mailer(cfg, logger),
		Bucket:    bucket(cfg),
		PublicURL: cfg.PublicURL,
		Logger:    logger,
	})
	afterLogin := "/dashboard"
	if cfg.PublicURL != "" {
		afterLogin = cfg.PublicURL + "/dashboard"
	}
	app, err := ekman.New(ekman.Options{
		DB:       database,
		Roles:    club.Roles,
		Baseline: club.Baseline,
		Grants:   svc.Access,
		Auth: &auth.Config{
			Providers: access.Providers(access.ProviderConfig{
				PublicURL:          cfg.PublicURL,
				GoogleClientID:     cfg.GoogleClientID,
				GoogleClientSecret: cfg.GoogleClientSecret,
				DevAdminEmail:      cfg.DevAdminEmail,
			}),
			Directory:  svc.Access,
			PublicURL:  cfg.PublicURL,
			AfterLogin: afterLogin,
		},
		CORS:   cfg.CORS,
		Logger: logger,
		Info: ekman.Info{Title: "Binary SUST API", Version: "0.1.0",
			Description: "The Binary SUST club API. Public pages never call it; they are built from the snapshot."},
	})
	if err != nil {
		return nil, err
	}
	s := &Server{App: app, Svc: svc}
	var n republish.Notifier = notifier(cfg, logger)
	if cfg.snapshotKV() {
		s.Publisher = &republish.Publisher{
			Build: func(ctx context.Context) (republish.Built, error) {
				snap, err := site.Build(ctx, svc.Sources, time.Now())
				if err != nil {
					return republish.Built{}, err
				}
				body, err := json.MarshalIndent(snap, "", "  ") // the same file cmd/snapshot writes
				return republish.Built{Hash: snap.Version.Hash, BuiltAt: snap.Version.BuiltAt, Body: body}, err
			},
			Store: &republish.KV{AccountID: cfg.CloudflareAccount, NamespaceID: cfg.SnapshotKV,
				Token: cfg.SnapshotKVToken, BaseURL: cfg.cloudflareAPI},
			Dispatch: n.(republish.Dispatcher),
			Log:      logger,
		}
		n = s.Publisher
	}
	if s.Publisher != nil {
		svc.Site.SetRepublisher(s.Publisher)
	} else {
		svc.Site.SetRepublisher(dispatchOnly{n})
	}
	app.Use(republish.Middleware(svc.Access, n, logger))
	app.Register(svc.Modules()...)
	return s, nil
}

func mailer(cfg Config, logger *slog.Logger) mail.Sender {
	if cfg.BrevoAPIKey != "" {
		return mail.Brevo(cfg.BrevoAPIKey, cfg.MailFrom, "Binary SUST")
	}
	return mail.Log(logger)
}

func bucket(cfg Config) storage.Bucket {
	switch {
	case cfg.MediaRepository != "":
		return &media.GitHubBucket{Token: cfg.MediaToken, Repository: cfg.MediaRepository,
			Branch: cfg.MediaBranch, PublicBaseURL: cfg.MediaBaseURL}
	case cfg.MediaDir != "":
		return storage.Dir(cfg.MediaDir, cfg.MediaBaseURL)
	}
	return nil
}

// dispatchOnly forces a build without a stored snapshot (KV not configured).
type dispatchOnly struct{ republish.Notifier }

func (d dispatchOnly) Republish(ctx context.Context, reason string) error {
	return d.Notify(ctx, reason)
}

// notifier dispatches without a snapshot (KV not configured), or is the Publisher's Dispatcher.
func notifier(cfg Config, logger *slog.Logger) interface {
	republish.Notifier
	republish.Dispatcher
} {
	if cfg.DispatchToken != "" && cfg.Repository != "" {
		return &republish.GitHub{Token: cfg.DispatchToken, Repository: cfg.Repository, BaseURL: cfg.githubAPI}
	}
	return republish.Log{Logger: logger}
}

func writeOpenAPI(app *ekman.App, args []string) error {
	fs := flag.NewFlagSet("openapi", flag.ContinueOnError)
	out := fs.String("o", "", "write the document to this file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	doc, err := club.OpenAPI(app)
	if err != nil {
		return err
	}
	doc = append(doc, '\n')
	if *out == "" {
		_, err = os.Stdout.Write(doc)
		return err
	}
	return os.WriteFile(*out, doc, 0o644)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	cfg := LoadConfig(os.Getenv)
	database, err := db.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("open database", "err", err.Error())
		os.Exit(1)
	}
	defer database.Close()
	s, err := NewApp(cfg, database, logger)
	if err != nil {
		logger.Error("configure", "err", err.Error())
		os.Exit(1)
	}
	// `openapi [-o file]`: ekman's document plus the enums it cannot express (club.OpenAPI),
	// written to stdout or to file (`pnpm gen` writes openapi.json without a shell redirect).
	if len(os.Args) > 1 && os.Args[1] == "openapi" {
		if err := writeOpenAPI(s.App, os.Args[2:]); err != nil {
			logger.Error("openapi", "err", err.Error())
			os.Exit(1)
		}
		return
	}
	serving := len(os.Args) < 2 || os.Args[1] == "serve"
	if serving && access.DevSignInEnabled(access.ProviderConfig{PublicURL: cfg.PublicURL, DevAdminEmail: cfg.DevAdminEmail}) {
		logger.Warn("dev sign-in enabled: GET /api/v1/auth/dev/start signs in " + cfg.DevAdminEmail)
	}
	// `serve --migrate`: migrate first, then hand over to ekman's serve.
	if len(os.Args) > 2 && os.Args[1] == "serve" && slices.Contains(os.Args[2:], "--migrate") {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		err := s.App.Migrate(ctx)
		stop()
		if err != nil {
			logger.Error("migrate", "err", err.Error())
			os.Exit(1)
		}
		logger.Info("migrations applied")
		os.Args = os.Args[:2]
	}
	if serving && s.Publisher != nil {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		s.Publisher.Start(ctx)
		go func() { // the date rollover changes "upcoming" and "registration open"
			for range republish.Daily(ctx, dhaka) {
				_ = s.Publisher.Notify(ctx, "nightly")
			}
		}()
		defer func() {
			stop()
			sctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.Publisher.Stop(sctx)
		}()
	} else if serving {
		logger.Warn("SNAPSHOT_KV_* not set: content changes dispatch builds without storing a snapshot")
	}
	s.App.Main()
}

// dhaka is the club's day: "upcoming" and "registration open" roll over at 00:05 here.
// Bangladesh has had no daylight saving since 2009, so a fixed zone needs no tzdata.
var dhaka = time.FixedZone("Asia/Dhaka", 6*60*60)
