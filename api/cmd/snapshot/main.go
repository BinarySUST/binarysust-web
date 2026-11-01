// Command snapshot writes the public-site snapshot. CI runs it against Neon directly (with a
// read-only role), so a site build never waits on the sleeping API.
//
//	DATABASE_URL=... go run ./cmd/snapshot -out ../web
//
// It writes <out>/.snapshot/site.json (the whole site.Snapshot; the web build slices it and
// writes public/version.json from its version field) and prints the version to stdout, so CI can
// compare the hash with the deployed /version.json and skip an unchanged deploy.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sakib412/ekman/db"

	"github.com/BinarySUST/binarysust-web/api/internal/club"
	"github.com/BinarySUST/binarysust-web/api/internal/site"
)

func main() {
	out := flag.String("out", "../web", "the web app directory; writes <out>/.snapshot/site.json")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "snapshot:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	d, err := db.Open("pgx", url)
	if err != nil {
		return err
	}
	defer d.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	svc := club.New(d, club.Deps{}) // read-only: no mail, no storage
	snap, err := site.Build(ctx, svc.Sources, time.Now())
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(out, ".snapshot")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(dir, "site.json"), append(body, '\n')); err != nil {
		return err
	}
	version, _ := json.Marshal(snap.Version)
	fmt.Println(string(version))
	fmt.Fprintf(os.Stderr, "snapshot: %s, %s, %s, %s, %s -> %s\n",
		count(len(snap.People), "person", "people"), count(len(snap.Events), "event", "events"),
		count(len(snap.Posts), "post", "posts"), count(len(snap.Talks), "talk", "talks"),
		count(len(snap.Sponsors), "sponsor", "sponsors"), filepath.Join(dir, "site.json"))
	return nil
}

func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".site-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
