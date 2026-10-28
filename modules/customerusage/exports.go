package customerusage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

type ExportLimits struct {
	MaxJobs  int
	MaxRows  int
	MaxBytes int
	TTL      time.Duration
}

type recipient struct {
	installation string
	subject      string
	epoch        uint64
}

func recipientOf(p ca.Principal) recipient {
	return recipient{p.Installation, p.SubjectID, p.Epoch}
}

type exportJob struct {
	owner      recipient
	checkpoint *ca.Checkpoint
	expires    time.Time
	state      string
	data       []byte
	digest     string
}

type Exports struct {
	mu      sync.Mutex
	gateway *Gateway
	limits  ExportLimits
	clock   func() time.Time
	jobs    map[string]*exportJob
}

func NewExports(gateway *Gateway, limits ExportLimits, clock func() time.Time) (*Exports, error) {
	if gateway == nil || gateway.store == nil || clock == nil || limits.MaxJobs <= 0 || limits.MaxRows <= 0 || limits.MaxRows > 10000 || limits.MaxBytes < 128 || limits.MaxBytes > 1048576 || limits.TTL <= 0 {
		return nil, ca.ErrInvalidPolicy
	}
	return &Exports{gateway: gateway, limits: limits, clock: clock, jobs: make(map[string]*exportJob)}, nil
}

func (e *Exports) purge() time.Time {
	now := e.clock()
	for id, job := range e.jobs {
		if !now.Before(job.expires) {
			job.data = nil
			delete(e.jobs, id)
		}
	}
	return now
}

func (e *Exports) PurgeExpired() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.purge()
}

func (e *Exports) Create(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", ca.ErrDenied
	}
	p, err := e.gateway.verify(ctx)
	if err != nil || p.Installation != e.gateway.profile.Installation {
		return "", ca.ErrDenied
	}
	service, err := e.gateway.exportService(func(context.Context) (ca.Principal, error) { return p, nil }, func(context.Context, ca.Selection) ([]ca.Row, error) { return nil, nil }, e.limits.MaxRows)
	if err != nil {
		return "", err
	}
	checkpoint, err := service.Capture(ctx, e.gateway.exportSelection(e.limits.MaxRows))
	if err != nil {
		return "", err
	}
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", ca.ErrUnavailable
	}
	id := hex.EncodeToString(random[:])
	e.mu.Lock()
	defer e.mu.Unlock()
	e.purge()
	if len(e.jobs) >= e.limits.MaxJobs {
		return "", ca.ErrUnavailable
	}
	if _, exists := e.jobs[id]; exists {
		return "", ca.ErrUnavailable
	}
	e.jobs[id] = &exportJob{owner: recipientOf(p), checkpoint: checkpoint, expires: e.clock().Add(e.limits.TTL), state: "pending"}
	return id, nil
}

func (e *Exports) invalidate(id string, job *exportJob) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.jobs[id] == job {
		job.state = "invalid"
		job.data = nil
		job.digest = ""
	}
}

func (e *Exports) authorizedJob(ctx context.Context, id string) (*exportJob, error) {
	if ctx == nil {
		return nil, ca.ErrDenied
	}
	p, err := e.gateway.verify(ctx)
	if err != nil {
		return nil, ca.ErrDenied
	}
	e.mu.Lock()
	e.purge()
	job := e.jobs[id]
	if job == nil || job.owner != recipientOf(p) {
		e.mu.Unlock()
		return nil, ca.ErrDenied
	}
	e.mu.Unlock()
	service, err := e.gateway.exportService(func(context.Context) (ca.Principal, error) { return p, nil }, func(context.Context, ca.Selection) ([]ca.Row, error) { return nil, nil }, e.limits.MaxRows)
	if err == nil {
		_, err = service.QueryPinned(ctx, e.gateway.exportSelection(e.limits.MaxRows), job.checkpoint)
	}
	if err != nil {
		e.invalidate(id, job)
		return nil, err
	}
	return job, nil
}

func (e *Exports) Run(ctx context.Context, id string) error {
	job, err := e.authorizedJob(ctx, id)
	if err != nil {
		return err
	}
	e.mu.Lock()
	now := e.purge()
	if e.jobs[id] != job || job.state != "pending" || !now.Before(job.expires) {
		e.mu.Unlock()
		return ca.ErrUnavailable
	}
	job.state = "running"
	e.mu.Unlock()
	var data []byte
	err = e.gateway.exportSource(ctx, job.checkpoint, e.limits.MaxRows, func(rows []ca.Row) error {
		var err error
		data, err = e.gateway.packageRows(rows, e.limits.MaxBytes)
		return err
	})
	if err != nil {
		e.invalidate(id, job)
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	now = e.purge()
	if e.jobs[id] != job || job.state != "running" || !now.Before(job.expires) {
		job.data = nil
		job.state = "invalid"
		return ca.ErrUnavailable
	}
	job.data = data
	job.digest = hashBytes(data)
	job.state = "ready"
	return nil
}

func (e *Exports) Download(ctx context.Context, id string) (*Download, error) {
	job, err := e.authorizedJob(ctx, id)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	now := e.purge()
	if e.jobs[id] != job || job.state != "ready" || !now.Before(job.expires) {
		e.mu.Unlock()
		return nil, ca.ErrUnavailable
	}
	data := append([]byte(nil), job.data...)
	digest := job.digest
	e.mu.Unlock()
	err = e.gateway.exportSource(ctx, job.checkpoint, e.limits.MaxRows, func(rows []ca.Row) error {
		candidate, err := e.gateway.packageRows(rows, e.limits.MaxBytes)
		if err != nil {
			return err
		}
		if hashBytes(data) != digest || !bytes.Equal(candidate, data) {
			return ca.ErrUnavailable
		}
		return nil
	})
	if err != nil {
		e.invalidate(id, job)
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	now = e.purge()
	if e.jobs[id] != job || job.state != "ready" || job.digest != digest || !now.Before(job.expires) {
		return nil, ca.ErrUnavailable
	}
	return &Download{Filename: "usage.json", ContentType: "application/json", SHA256: digest, Bytes: data}, nil
}
