// Command seed loads the demo data (internal/demo) into a migrated database and makes
// DEV_ADMIN_EMAIL an admin through a permanent grant. Running it again is safe.
//
//	make seed     # or: DATABASE_URL=... go run ./cmd/seed
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sakib412/ekman/db"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/demo"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
	fmt.Println("seed: done")
}

func run() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	d, err := db.Open("pgx", url)
	if err != nil {
		return err
	}
	defer d.Close()
	ctx := context.Background()
	return d.Tx(ctx, func(ctx context.Context) error {
		return demo.Seed(ctx, d.Q(ctx), time.Now().UTC().Truncate(time.Hour), os.Getenv("DEV_ADMIN_EMAIL"))
	})
}
