package server

import (
	"log/slog"
	"sync"
	"time"
)

const (
	clockDiagnostic = time.Minute
	sweepEvery      = 30 * time.Second
	cleanupEvery    = time.Second
)

type wallLead struct {
	wall   time.Time
	server time.Time
}

type serverClock struct {
	mu        sync.Mutex
	wall      func() time.Time
	source    func() time.Duration
	mark      time.Duration
	base      time.Time
	offset    time.Duration
	tolerance time.Duration
	beyond    bool
	lead      *wallLead
	log       *slog.Logger
}

func systemWall() time.Time { return time.Now() }

func monotonic() func() time.Duration {
	origin := time.Now()
	return func() time.Duration { return time.Since(origin) }
}

func newServerClock(wall func() time.Time, elapsed func() time.Duration, log *slog.Logger) *serverClock {
	c := &serverClock{wall: wall, source: elapsed, tolerance: 24 * time.Hour, log: log}
	if wall == nil {
		c.wall, c.source = systemWall, monotonic()
	}
	if c.source == nil {
		origin := c.wallNow()
		c.source = func() time.Duration { return c.wallNow().Sub(origin) }
	}
	c.rebase(c.wallNow(), c.tolerance)
	return c
}

func (c *serverClock) wallNow() time.Time { return c.wall().UTC().Round(0) }

func (c *serverClock) Wall() time.Time { return c.wallNow() }

func (c *serverClock) rebase(base time.Time, tolerance time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.base, c.mark, c.offset, c.tolerance, c.beyond, c.lead = base.UTC().Round(0), c.source(), 0, tolerance, false, nil
}

func (c *serverClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	proj := c.base.Add(c.source() - c.mark)
	w := c.wallNow()
	lead := w.Sub(proj.Add(c.offset))
	switch {
	case lead > c.tolerance:
		if !c.beyond {
			c.log.Error("clock_wall_lead", "wall_time", stamp(w), "server_time", stamp(proj.Add(c.offset)),
				"advice", "confirm the host clock is correct, then restart the server")
		}
		c.beyond, c.lead = true, &wallLead{wall: w, server: proj.Add(c.offset)}
	case lead > 0:
		c.offset, c.beyond = w.Sub(proj), false
	default:
		c.beyond = false
	}
	return proj.Add(c.offset).UTC().Round(0)
}

func (c *serverClock) leadBeyond() (wallLead, bool) {
	if c == nil {
		return wallLead{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lead == nil {
		return wallLead{}, false
	}
	return *c.lead, true
}

func every(d time.Duration, stop <-chan struct{}, wake <-chan struct{}, run func()) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		case <-wake:
		}
		run()
	}
}
