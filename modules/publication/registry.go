package publication

import (
	"math/big"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var (
	defaults = Limits{ChunkRows: 10000, VersionRows: 1000000, StagedRows: 2000000, TenantStagedRows: 1000000, HeldRows: 10000000, TenantHeldRows: 5000000,
		Groups: 100000, Retained: 2, Page: 1000, TenantDatasets: 10000, Datasets: 100000, StagingTTL: 24 * time.Hour}
	maximums = Limits{ChunkRows: 100000, VersionRows: 10000000, StagedRows: 20000000, TenantStagedRows: 20000000, HeldRows: 100000000, TenantHeldRows: 100000000,
		Groups: 1000000, Retained: 16, Page: 10000, TenantDatasets: 100000, Datasets: 1000000, StagingTTL: 30 * 24 * time.Hour}
)

type partial struct {
	coefficient *big.Int
	scale       int
}

type chunk struct {
	digest   string
	rows     []Row
	partials map[TotalKey]*partial
}

type version struct {
	view   View
	chunks []chunk
	rows   []Row
	totals []Total
}

type dataset struct {
	tenant      string
	ids         map[string]*version
	highest     string
	sequence    uint64
	current     *version
	retained    []*version
	unpublished *version
}

type queued struct {
	dataset  *dataset
	sequence uint64
	deadline time.Time
}

type Registry struct {
	mu           sync.Mutex
	profile      decimal.Profile
	limits       Limits
	clock        func() time.Time
	now          time.Time
	datasets     map[[2]string]*dataset
	tenantSets   map[string]int
	staged       int
	held         int
	tenantStaged map[string]int
	tenantHeld   map[string]int
	queue        []queued
	open         int
}

func settle(value, fallback, maximum int) (int, bool) {
	if value == 0 {
		return fallback, true
	}
	return value, value > 0 && value <= maximum
}

func NewRegistry(config Config, clock func() time.Time) (*Registry, error) {
	if clock == nil || !config.Profile.Valid() {
		return nil, ErrInvalid
	}
	l, d, m := config.Limits, defaults, maximums
	fields := []struct {
		value          *int
		fallback, most int
	}{
		{&l.ChunkRows, d.ChunkRows, m.ChunkRows}, {&l.VersionRows, d.VersionRows, m.VersionRows}, {&l.StagedRows, d.StagedRows, m.StagedRows},
		{&l.TenantStagedRows, d.TenantStagedRows, m.TenantStagedRows}, {&l.HeldRows, d.HeldRows, m.HeldRows}, {&l.TenantHeldRows, d.TenantHeldRows, m.TenantHeldRows},
		{&l.Groups, d.Groups, m.Groups}, {&l.Retained, d.Retained, m.Retained}, {&l.Page, d.Page, m.Page},
		{&l.TenantDatasets, d.TenantDatasets, m.TenantDatasets}, {&l.Datasets, d.Datasets, m.Datasets},
	}
	for _, f := range fields {
		value, ok := settle(*f.value, f.fallback, f.most)
		if !ok {
			return nil, ErrInvalid
		}
		*f.value = value
	}
	switch {
	case l.StagingTTL == 0:
		l.StagingTTL = d.StagingTTL
	case l.StagingTTL < 0 || l.StagingTTL > m.StagingTTL:
		return nil, ErrInvalid
	}
	return &Registry{profile: config.Profile, limits: l, clock: clock, datasets: map[[2]string]*dataset{}, tenantSets: map[string]int{},
		tenantStaged: map[string]int{}, tenantHeld: map[string]int{}}, nil
}

func (r *Registry) enter() time.Time {
	r.mu.Lock()
	now := r.clock().UTC().Round(0)
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func refuse(reason string) error { return Refusal{Reason: reason} }

func (r *Registry) reserve(tenant string, rows int) bool {
	l := r.limits
	if r.staged+rows > l.StagedRows || r.held+rows > l.HeldRows || r.tenantStaged[tenant]+rows > l.TenantStagedRows || r.tenantHeld[tenant]+rows > l.TenantHeldRows {
		return false
	}
	r.staged += rows
	r.held += rows
	r.tenantStaged[tenant] += rows
	r.tenantHeld[tenant] += rows
	return true
}

func (r *Registry) unstage(tenant string, rows int) {
	r.staged -= rows
	r.tenantStaged[tenant] -= rows
}

func (r *Registry) unhold(tenant string, rows int) {
	r.held -= rows
	r.tenantHeld[tenant] -= rows
}

func (r *Registry) expire(d *dataset, now time.Time) {
	if v := d.unpublished; v != nil && !now.Before(v.view.Deadline) {
		r.drop(d, v)
	}
}

func (r *Registry) drop(d *dataset, v *version) {
	if v.view.State == StateStaging {
		r.unstage(d.tenant, v.view.Rows)
	}
	r.unhold(d.tenant, v.view.Rows)
	delete(d.ids, v.view.Version)
	v.chunks, v.rows, v.totals = nil, nil, nil
	d.unpublished = nil
	r.open--
}

func (r *Registry) sweep(now time.Time) {
	for len(r.queue) > 0 {
		q := r.queue[0]
		if v := q.dataset.unpublished; v != nil && v.view.Sequence == q.sequence {
			if now.Before(q.deadline) {
				return
			}
			r.drop(q.dataset, v)
		}
		r.queue[0] = queued{}
		r.queue = r.queue[1:]
	}
}

func (r *Registry) enqueue(d *dataset, v *version) {
	if len(r.queue) > 2*r.open+64 {
		kept := make([]queued, 0, r.open+1)
		for _, q := range r.queue {
			if u := q.dataset.unpublished; u != nil && u.view.Sequence == q.sequence {
				kept = append(kept, q)
			}
		}
		r.queue = kept
	}
	r.queue = append(r.queue, queued{dataset: d, sequence: v.view.Sequence, deadline: v.view.Deadline})
	r.open++
}

func (r *Registry) lookup(tenant, name string, now time.Time) *dataset {
	d := r.datasets[[2]string{tenant, name}]
	if d != nil {
		r.expire(d, now)
	}
	return d
}

func (r *Registry) version(tenant, name, id string, now time.Time) (*dataset, *version, error) {
	if !validID(tenant) || !validID(name) || !validID(id) {
		return nil, nil, ErrInvalid
	}
	d := r.lookup(tenant, name, now)
	if d == nil {
		return nil, nil, ErrNotFound
	}
	v := d.ids[id]
	switch {
	case v != nil:
		return d, v, nil
	case id <= d.highest:
		return nil, nil, refuse(ReasonStale)
	}
	return nil, nil, ErrNotFound
}
