package republish

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// Built is one snapshot build: the JSON the site is built from and its content hash.
type Built struct {
	Hash    string
	BuiltAt time.Time
	Body    []byte
}

// Store holds snapshots (KV in production).
type Store interface {
	Put(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
}

// Dispatcher asks CI to rebuild the site from the snapshot with hash (GitHub in production).
type Dispatcher interface {
	Dispatch(ctx context.Context, reason, hash string) error
}

// LatestKey points at the newest stored snapshot: {"hash", "builtAt"}. Snapshots themselves live
// under SnapshotKey(hash), so a build always fetches exactly the version it was told about, even
// while Workers KV is still propagating an older "latest".
const LatestKey = "latest"

// SnapshotKey is where the snapshot with hash is stored.
func SnapshotKey(hash string) string { return "snapshot:" + hash }

// Latest is the value stored under LatestKey.
type Latest struct {
	Hash    string    `json:"hash"`
	BuiltAt time.Time `json:"builtAt"`
}

// Publisher makes the API the only producer of the snapshot. Each run builds the snapshot,
// stores it (SnapshotKey, then LatestKey) and dispatches the build, skipping all of it when the
// hash equals the last one published. Runs happen on one background goroutine: signals that
// arrive while a run is in progress collapse into a single next run, so a burst of edits costs
// one build, two KV writes and one CI run.
//
// It implements Notifier, so the republish Middleware signals it after every content write
// without waiting for the build.
type Publisher struct {
	Build    func(ctx context.Context) (Built, error)
	Store    Store
	Dispatch Dispatcher
	Log      *slog.Logger
	TTL      time.Duration // of stored snapshots; default 30 days
	Timeout  time.Duration // of one run; default 2 minutes

	mu      sync.Mutex
	pending bool
	force   bool
	reason  string
	running bool
	idle    chan struct{} // closed while nothing is pending or running
	wake    chan struct{}
	done    chan struct{}
	last    string // hash of the last fully published snapshot
	seeded  bool
}

// Notify implements Notifier: it queues a run and returns at once.
func (p *Publisher) Notify(_ context.Context, reason string) error {
	p.signal(reason, false)
	return nil
}

// Republish queues a run that stores and dispatches even when the hash is unchanged.
func (p *Publisher) Republish(_ context.Context, reason string) error {
	p.signal(reason, true)
	return nil
}

func (p *Publisher) init() {
	if p.wake == nil {
		p.wake = make(chan struct{}, 1)
		p.done = make(chan struct{})
		p.idle = make(chan struct{})
		close(p.idle)
	}
}

func (p *Publisher) signal(reason string, force bool) {
	p.mu.Lock()
	p.init()
	if !p.pending && !p.running {
		p.idle = make(chan struct{})
	}
	p.pending, p.reason = true, reason
	p.force = p.force || force
	p.mu.Unlock()
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// Start runs the publisher until ctx is canceled, beginning with a "boot" run: a deploy that
// changes what Build produces publishes without waiting for an edit.
func (p *Publisher) Start(ctx context.Context) {
	p.signal("boot", false)
	go p.loop(ctx)
}

// Stop waits for the goroutine to finish after Start's ctx is canceled: a run in progress
// completes (bounded by Timeout); a run still pending is dropped, the next boot covers it.
func (p *Publisher) Stop(ctx context.Context) {
	p.mu.Lock()
	p.init()
	done := p.done
	p.mu.Unlock()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// Wait blocks until nothing is pending or running.
func (p *Publisher) Wait(ctx context.Context) error {
	for {
		p.mu.Lock()
		p.init()
		if !p.pending && !p.running {
			p.mu.Unlock()
			return nil
		}
		idle := p.idle
		p.mu.Unlock()
		select {
		case <-idle:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *Publisher) loop(ctx context.Context) {
	p.mu.Lock()
	p.init()
	wake, done := p.wake, p.done
	p.mu.Unlock()
	defer close(done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
		}
		p.mu.Lock()
		if !p.pending {
			p.mu.Unlock()
			continue
		}
		reason, force := p.reason, p.force
		p.pending, p.force, p.running = false, false, true
		p.mu.Unlock()

		p.run(ctx, reason, force)

		p.mu.Lock()
		p.running = false
		if !p.pending {
			close(p.idle)
		}
		p.mu.Unlock()
	}
}

func (p *Publisher) run(parent context.Context, reason string, force bool) {
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), timeout)
	defer cancel()
	log := p.Log
	if log == nil {
		log = slog.Default()
	}
	fail := func(msg string, err error) {
		log.ErrorContext(ctx, "republish: "+msg, "err", err.Error(), "reason", reason)
	}
	if !p.seeded {
		p.seed(ctx, log)
	}
	b, err := p.Build(ctx)
	if err != nil {
		fail("build snapshot", err)
		return
	}
	if b.Hash == p.last && !force {
		return
	}
	ttl := p.TTL
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	if err := p.Store.Put(ctx, SnapshotKey(b.Hash), b.Body, ttl); err != nil {
		fail("store snapshot", err)
		return
	}
	latest, _ := json.Marshal(Latest{Hash: b.Hash, BuiltAt: b.BuiltAt})
	if err := p.Store.Put(ctx, LatestKey, latest, 0); err != nil {
		fail("store latest", err)
		return
	}
	if err := p.Dispatch.Dispatch(ctx, reason, b.Hash); err != nil {
		fail("dispatch failed", err)
		return
	}
	// Only a fully published snapshot counts: after any failure the next trigger tries again.
	p.last = b.Hash
	log.InfoContext(ctx, "republish: published snapshot", "hash", b.Hash, "reason", reason)
}

// seed learns the last published hash from the store, so a restart with unchanged content
// writes nothing and dispatches nothing.
func (p *Publisher) seed(ctx context.Context, log *slog.Logger) {
	raw, err := p.Store.Get(ctx, LatestKey)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			log.WarnContext(ctx, "republish: read latest", "err", err.Error())
			return // try again next run
		}
		p.seeded = true
		return
	}
	var l Latest
	if json.Unmarshal(raw, &l) == nil {
		p.last = l.Hash
	}
	p.seeded = true
}

// NextRollover is the next 00:05 in loc strictly after now: the moment "upcoming" and
// "registration open" change for the club's own day.
func NextRollover(now time.Time, loc *time.Location) time.Time {
	t := now.In(loc)
	next := time.Date(t.Year(), t.Month(), t.Day(), 0, 5, 0, 0, loc)
	if !next.After(t) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// Daily ticks at every NextRollover in loc; the channel closes when ctx is canceled.
func Daily(ctx context.Context, loc *time.Location) <-chan time.Time {
	ch := make(chan time.Time)
	go func() {
		defer close(ch)
		for {
			timer := time.NewTimer(time.Until(NextRollover(time.Now(), loc)))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case t := <-timer.C:
				select {
				case ch <- t:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}
