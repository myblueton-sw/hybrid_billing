package server

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
)

type scripted struct {
	mu      sync.Mutex
	wall    time.Time
	origin  time.Time
	elapsed time.Duration
}

func (c *scripted) Wall() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.wall.Add(time.Duration(time.Now().Nanosecond() % 1))
}

func (c *scripted) Elapsed() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.elapsed
}

func (c *scripted) step(wall, elapsed time.Duration) {
	c.mu.Lock()
	c.wall, c.elapsed = c.wall.Add(wall), c.elapsed+elapsed
	c.mu.Unlock()
}

func TestTheServerClock(t *testing.T) {
	var logs bytes.Buffer
	src := &scripted{wall: time.Now()}
	c := newServerClock(src.Wall, src.Elapsed, slog.New(slog.NewJSONHandler(&logs, nil)))
	first := c.now()
	if first != first.Round(0) || !first.Equal(src.Wall().UTC().Round(0)) {
		t.Fatalf("first %v", first)
	}
	src.step(-time.Hour, time.Minute)
	back := c.now()
	if !back.Equal(first.Add(time.Minute)) {
		t.Errorf("a backward wall step froze or moved the clock: %v", back.Sub(first))
	}
	src.step(2*time.Hour, time.Minute)
	ahead := c.now()
	if !ahead.Equal(src.Wall().UTC().Round(0)) {
		t.Errorf("a forward step within the tolerance was not absorbed %v", ahead.Sub(first))
	}
	src.step(-30*time.Minute, time.Minute)
	if got := c.now(); !got.Equal(ahead.Add(time.Minute)) {
		t.Errorf("after a catch-up a backward step %v", got.Sub(ahead))
	}
	src.step(10*time.Minute, 0)
	src.step(10*time.Minute, 0)
	if got := c.now(); !got.Equal(ahead.Add(time.Minute)) {
		t.Errorf("repeated small steps accumulated %v", got.Sub(ahead))
	}
	src.step(30*time.Minute, 0)
	if got := c.now(); !got.Equal(src.Wall().UTC().Round(0)) {
		t.Errorf("a pause of the monotonic clock is not caught up %v", got)
	}
	if _, beyond := c.leadBeyond(); beyond {
		t.Error("flagged within the tolerance")
	}
	before := c.now()
	src.step(25*time.Hour, 0)
	if got := c.now(); !got.Equal(before) {
		t.Errorf("a lead beyond the tolerance was absorbed %v", got.Sub(before))
	}
	c.now()
	lead, beyond := c.leadBeyond()
	if !beyond || lead.wall != lead.wall.Round(0) || lead.wall.Sub(lead.server) < 24*time.Hour {
		t.Errorf("lead %+v %v", lead, beyond)
	}
	if n := strings.Count(logs.String(), "clock_wall_lead"); n != 1 || !strings.Contains(logs.String(), "restart the server") {
		t.Errorf("logged %d %s", n, logs.String())
	}
	src.step(-25*time.Hour, 0)
	c.now()
	src.step(25*time.Hour, 0)
	c.now()
	if n := strings.Count(logs.String(), "clock_wall_lead"); n != 2 {
		t.Errorf("a second occurrence logged %d", n)
	}
	system := newServerClock(nil, nil, slog.New(slog.DiscardHandler))
	if got := system.now(); got.IsZero() || got != got.Round(0) || system.Wall().IsZero() {
		t.Error("the system clock")
	}
	var none *serverClock
	if _, beyond := none.leadBeyond(); beyond {
		t.Error("a missing clock")
	}
	stop, wake, ran := make(chan struct{}), make(chan struct{}), make(chan struct{}, 4)
	go every(time.Hour, stop, wake, func() { ran <- struct{}{} })
	wake <- struct{}{}
	<-ran
	close(stop)
}

func TestAWallLeadRefusesIncreases(t *testing.T) {
	src := &scripted{wall: start}
	f := setupWith(t, func(c *Config) { c.Clock, c.Elapsed = src.Wall, src.Elapsed })
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "suspend", f.run(root, "identity.suspend_user", SampleMember, nil), "user_suspended", "")
	src.step(25*time.Hour, time.Minute)
	f.failed(t, "an increase after a lead", f.run(root, "identity.reinstate_user", SampleMember, nil), commands.DependencyUnavailable)
	rows := f.list(root, "").body
	if rows["wall_lead_beyond_tolerance"] == nil || rows["clock_behind"] != false || rows["clock_ahead"] != true {
		t.Errorf("read %+v", rows)
	}
	f.ok(t, "a decrease after a lead", f.run(root, "identity.suspend_membership", SampleAdmin3, nil), "membership_suspended", "")
}

func TestRoleEndsFireDuringWallSteps(t *testing.T) {
	src := &scripted{wall: start}
	f := setupWith(t, func(c *Config) { c.Clock, c.Elapsed = src.Wall, src.Elapsed })
	root := f.as(AdminLogin)
	member := f.withMember()
	f.ok(t, "an end", f.period(root, SampleMember, stamp(f.s.clock().Add(10*time.Minute).Truncate(time.Minute))), "role_period_set", "")
	src.step(-time.Hour, 11*time.Minute)
	if got := f.me(member); got.status != 401 {
		t.Errorf("an end during a backward step %+v", got)
	}
	if rows := f.list(f.as(AdminLogin), "").body; rows["clock_behind"] != true {
		t.Errorf("diagnostics %+v", rows)
	}
}

func (f world) aheadLedger(t *testing.T, ahead time.Duration) {
	t.Helper()
	f.advance(ahead)
	f.withMember()
	f.ok(t, "a record in the future", f.period(f.as(AdminLogin), SampleMember, f.at(time.Hour)), "role_period_set", "")
	f.advance(-ahead)
}

func TestLedgerTimeAheadOfTheHost(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.aheadLedger(t, 48*time.Hour)
	_, err := f.restartWith(nil)
	if !errors.Is(err, ErrStart) || !strings.Contains(err.Error(), "more than 24 hours ahead of this host's clock 2026-11-16T09:00:00Z") {
		t.Fatalf("start %v", err)
	}
	g := f.restart(func(c *Config) { c.Access.ClockSkewHours = 72 })
	if got := g.s.clock(); got.Before(start.Add(48 * time.Hour)) {
		t.Errorf("the base did not take the ledger time %v", got)
	}
	h := g.restart(func(c *Config) { c.Access.ClockSkewHours, c.Access.ClockBase = 24, ClockBaseLedger })
	if got := h.s.clock(); got.Before(start.Add(48 * time.Hour)) {
		t.Errorf("ledger base %v", got)
	}
	chosen := h.records(t, decisionledger.ClockBaseChoice)
	if len(chosen) != 1 || chosen[0]["choice"] != ClockBaseLedger || chosen[0]["v"] != 2.0 {
		t.Fatalf("clock base %+v", chosen)
	}
	if rows := h.list(h.as(AdminLogin), "").body["clock_base"].(map[string]any); rows["choice"] != ClockBaseLedger || rows["recorded"] != true {
		t.Errorf("read %+v", rows)
	}
	i := h.restart(func(c *Config) { c.Access.ClockBase = "" })
	if got := i.s.clock(); got.Before(start.Add(48 * time.Hour)) {
		t.Errorf("a later start honours the recorded offset %v", got)
	}
	before, _ := readLedger(i)
	if _, err := i.restartWith(func(c *Config) { c.Access.ClockBase = ClockBaseWall }); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "cannot be verified") {
		t.Errorf("a flag left on after a verified recorded choice %v", err)
	}
	if after, _ := readLedger(i); len(after) != len(before) {
		t.Error("a refused flag wrote a record")
	}
}

func TestClockBaseWallAndRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.aheadLedger(t, 48*time.Hour)
	g := f.restart(func(c *Config) { c.Access.ClockBase = ClockBaseWall })
	if got := g.s.clock(); !got.Equal(start) {
		t.Errorf("wall base %v", got)
	}
	h := g.restart(func(c *Config) { c.Access.ClockBase = "" })
	if got := h.s.clock(); !got.Equal(start) {
		t.Errorf("earlier ahead times are ignored after a wall choice %v", got)
	}
	plain := setup(t, SeedSample).ready()
	if _, err := plain.restartWith(func(c *Config) { c.Access.ClockBase = ClockBaseLedger }); !errors.Is(err, ErrConfig) {
		t.Errorf("a flag without the condition %v", err)
	}
	broken := setup(t, SeedSample).ready()
	broken.aheadLedger(t, 48*time.Hour)
	j := broken.restart(func(c *Config) { c.Access.ClockBase = ClockBaseLedger })
	lines := len(j.ledgerLines(t))
	j.breakLine(t, 2)
	before, _ := readLedger(j)
	if _, err := j.restartWith(func(c *Config) { c.Access.ClockBase = "" }); !errors.Is(err, ErrStart) || !strings.Contains(err.Error(), "could not be verified") {
		t.Errorf("an unverified clock base %v (%d lines)", err, lines)
	}
	if after, _ := readLedger(j); !bytes.Equal(before, after) {
		t.Error("the refusal changed the ledger")
	}
	k, err := j.restartWith(func(c *Config) { c.Access.ClockBase = ClockBaseWall })
	if err != nil {
		t.Fatalf("a flag for an isolated ledger %v", err)
	}
	if rows := k.list(k.as(AdminLogin), "").body["clock_base"].(map[string]any); rows["recorded"] != false || rows["choice"] != ClockBaseWall {
		t.Errorf("isolated %+v", rows)
	}
}

func TestPreScanReadFailure(t *testing.T) {
	f := setup(t, SeedSample)
	f.must(osMkdirAll(f.s.cfg.Data + "/" + decisionledger.RelativeFile))
	if err := f.s.Start(); !errors.Is(err, ErrStart) || !strings.Contains(err.Error(), "pre-scan") {
		t.Errorf("start %v", err)
	}
}
