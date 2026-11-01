package demo_test

import (
	"context"
	"testing"
	"time"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/demo"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/testapp"
)

// Seeded events happen at hours people hold events in Sylhet, whatever time the seed runs.
func TestSeededEventsAtLocalDaytime(t *testing.T) {
	dhaka, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		dhaka = time.FixedZone("Asia/Dhaka", 6*60*60)
	}
	a := testapp.New(t)
	ctx := context.Background()
	// Seed at 18:xx UTC (midnight in Dhaka), the hour that used to put every event at 12 am.
	now := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	if err := a.DB.Tx(ctx, func(ctx context.Context) error { return demo.Seed(ctx, a.DB.Q(ctx), now, "") }); err != nil {
		t.Fatal(err)
	}
	rows, err := a.DB.SQL().QueryContext(ctx, `SELECT slug, starts_at, ends_at FROM events`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var slug string
		var starts, ends time.Time
		if err := rows.Scan(&slug, &starts, &ends); err != nil {
			t.Fatal(err)
		}
		n++
		if h := starts.In(dhaka).Hour(); h < 8 || h > 20 {
			t.Errorf("%s starts at %s in Dhaka", slug, starts.In(dhaka).Format("15:04"))
		}
		if !ends.After(starts) {
			t.Errorf("%s ends before it starts", slug)
		}
	}
	if n == 0 {
		t.Fatal("no events seeded")
	}
}
