package server

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
)

const (
	recoveryPrimary   = "sample-primary"
	recoveryStandby   = "sample-standby"
	recoveryERPWriter = "sample-erp-writer"
	recoveryIssuer    = "sample-issuance-writer"
	recoveryStandbyW  = "sample-standby-writer"
	recoveryVerifier  = "sample-verifier"
	recoveryLedger    = "sample-erp-ledger"
	recoveryRegister  = "sample-issuance-register"
	recoverySeries    = "sample-invoice-series"
	recoveryLockText  = "Administrator lock"
	recoveryRetired   = "retired"
	recoverySkew      = time.Minute
)

var recoveryDomains = []string{recovery.DomainDatabase, recovery.DomainEvidence, recovery.DomainKeys, recovery.DomainPolicies, recovery.DomainModules,
	recovery.DomainNumbering, recovery.DomainSubmissions, recovery.DomainOutbox, recovery.DomainDeletions}

type recoveryLockRecord struct {
	by        string
	rationale string
	at        time.Time
}

type recoveryInstall struct {
	id        string
	tenant    string
	skew      time.Duration
	writers   []string
	externals []recovery.External
	verifier  string
	locks     []recoveryLockRecord
}

type recoveryEntry struct {
	view      recovery.ProposalView
	created   time.Time
	rationale string
	closedBy  string
	closing   string
}

type recoveryStore struct {
	installs []*recoveryInstall
	entries  []*recoveryEntry
	dropped  map[string]int
}

func newRecoveryStore() *recoveryStore { return &recoveryStore{dropped: map[string]int{}} }

func (c *recoveryStore) install(id string) *recoveryInstall {
	for _, i := range c.installs {
		if i.id == id {
			return i
		}
	}
	return nil
}

func (c *recoveryStore) entry(number uint64) *recoveryEntry {
	for _, e := range c.entries {
		if e.view.Number == number {
			return e
		}
	}
	return nil
}

func (c *recoveryStore) keep(e *recoveryEntry) {
	c.entries = append(c.entries, e)
	tenant := c.tenantOf(e)
	if c.held(tenant) <= installKept {
		return
	}
	for index, old := range c.entries {
		if old.view.State != recovery.ProposalPending && c.tenantOf(old) == tenant {
			c.entries = append(c.entries[:index], c.entries[index+1:]...)
			c.dropped[tenant]++
			return
		}
	}
}

func (c *recoveryStore) of(tenant string) []*recoveryEntry {
	out := []*recoveryEntry{}
	for _, e := range c.entries {
		if c.tenantOf(e) == tenant {
			out = append(out, e)
		}
	}
	return out
}

func (c *recoveryStore) held(tenant string) int { return len(c.of(tenant)) }

func changeScope(c recovery.Change) string {
	return firstOf(c.Installation.ID, c.Writer.Installation, c.External.Installation, c.Verifier.Installation, c.Begin.Installation, c.Unlock.Installation,
		c.Prune.Installation, c.Abandon.Installation)
}

func (c *recoveryStore) tenantOf(e *recoveryEntry) string {
	if e.view.Change.Kind == recovery.ChangeInstallation {
		return e.view.Change.Installation.Tenant
	}
	if i := c.install(changeScope(e.view.Change)); i != nil {
		return i.tenant
	}
	return ""
}

func (c *recoveryStore) pendingUnlock(id string) bool {
	for _, e := range c.entries {
		if v := e.view; v.State == recovery.ProposalPending && v.Change.Kind == recovery.ChangeUnlock && v.Change.Unlock.Installation == id {
			return true
		}
	}
	return false
}

func (s *Server) recoveryCall(name string, call func() error) error {
	return s.moduleCall("recovery-"+name, recovery.ErrInvalid, call)
}

func (s *Server) recoveryPropose(c recovery.Change, by recovery.Actor, rationale string) (*recoveryEntry, error) {
	var view recovery.ProposalView
	if err := s.recoveryCall("propose", func() (err error) { view, err = s.recovery.Propose(c, by); return }); err != nil {
		return nil, err
	}
	e := s.rec.entry(view.Number)
	if e == nil {
		e = &recoveryEntry{view: view, created: s.clock(), rationale: rationale}
		s.rec.keep(e)
	}
	return e, nil
}

func (s *Server) retireOn(e *recoveryEntry, err error) error {
	if errors.Is(err, recovery.ErrNotFound) && e.view.State == recovery.ProposalPending {
		e.view.State = recoveryRetired
	}
	return err
}

func (s *Server) recordApplied(c recovery.Change) {
	if c.Kind == recovery.ChangeInstallation {
		s.rec.installs = append(s.rec.installs, &recoveryInstall{id: c.Installation.ID, tenant: c.Installation.Tenant, skew: c.Installation.Skew})
		return
	}
	i := s.rec.install(changeScope(c))
	if i == nil {
		return
	}
	switch c.Kind {
	case recovery.ChangeWriter:
		i.writers = append(i.writers, c.Writer.ID)
	case recovery.ChangeExternal:
		i.externals = append(i.externals, c.External)
	case recovery.ChangeVerifier:
		i.verifier = c.Verifier.ID
	}
}

func (s *Server) recoveryApprove(e *recoveryEntry, by recovery.Actor, rationale string) error {
	var view recovery.ProposalView
	err := s.recoveryCall("approve", func() (err error) { view, err = s.recovery.Approve(e.view.Number, by); return })
	if err != nil {
		return s.retireOn(e, err)
	}
	if e.view.State == recovery.ProposalPending {
		e.closedBy, e.closing = by.ID, rationale
		s.recordApplied(view.Change)
	}
	e.view = view
	return nil
}

func (s *Server) recoveryWithdraw(e *recoveryEntry, by recovery.Actor, rationale string) error {
	var view recovery.ProposalView
	err := s.recoveryCall("withdraw", func() (err error) { view, err = s.recovery.Withdraw(e.view.Number, by); return })
	if err != nil {
		return s.retireOn(e, err)
	}
	e.view, e.closedBy, e.closing = view, by.ID, rationale
	return nil
}

func (s *Server) recoveryLock(i *recoveryInstall, by recovery.Actor, rationale string) error {
	if err := s.recoveryCall("lock", func() error { return s.recovery.Lock(i.id, recoveryLockText, by) }); err != nil {
		return err
	}
	i.locks = append(i.locks, recoveryLockRecord{by: by.ID, rationale: rationale, at: s.clock()})
	return nil
}

func (s *Server) recoveryStatus(id string) (string, uint64, time.Time, error) {
	var state string
	var generation uint64
	var stopped time.Time
	err := s.recoveryCall("status", func() (err error) { state, generation, stopped, err = s.recovery.Status(id); return })
	return state, generation, stopped, err
}

func recoveryManifest(tampered string) []recovery.Inventory {
	out := []recovery.Inventory{}
	for _, d := range recoveryDomains {
		second := sampleHash("sample-" + d + "-b")
		if d == tampered {
			second = sampleHash("sample-outbox-tampered")
		}
		out = append(out, recovery.Inventory{Domain: d, Items: 10, Chunks: []recovery.Chunk{{Start: 0, End: 4, Checksum: sampleHash("sample-" + d + "-a")},
			{Start: 4, End: 10, Checksum: second}}})
	}
	return out
}

func (s *Server) seedRecovery(now time.Time) error {
	ws := s.Workspace()
	first := recovery.Actor{ID: SampleAdmin, Kind: recovery.KindHuman}
	second := recovery.Actor{ID: SampleAdmin2, Kind: recovery.KindHuman}
	writer := recovery.Actor{ID: recoveryERPWriter, Kind: recovery.KindSystem}
	verifier := recovery.Actor{ID: recoveryVerifier, Kind: recovery.KindSystem}
	approved := func(c recovery.Change) func() error {
		return func() error {
			e, err := s.recoveryPropose(c, first, installSeedReason)
			if err != nil {
				return err
			}
			return s.recoveryApprove(e, second, installSeedReason)
		}
	}
	var token recovery.Token
	var confirmed recovery.CallView
	call := func(key string, number int64, keep *recovery.CallView) func() error {
		return func() error {
			target := recovery.Target{Workspace: ws, Key: key, Series: recoverySeries, Number: number}
			return s.recoveryCall("call", func() (err error) {
				*keep, err = s.recovery.Call(recoveryPrimary, recoveryERPWriter, token, target, writer)
				return
			})
		}
	}
	record := func(e recovery.Event, by recovery.Actor) func() error {
		return func() error {
			return s.recoveryCall("record", func() error { _, err := s.recovery.Record(recoveryPrimary, e, by); return err })
		}
	}
	manifest := recoveryManifest("")
	point := recovery.RestorePoint{Installation: recoveryPrimary, Generation: 1, Backup: sampleHash("sample-backup-1"), ManifestDigest: recovery.ManifestDigest(manifest),
		AsOf: now.Add(-time.Hour), Manifest: manifest}
	var open recovery.CallView
	steps := []func() error{
		approved(recovery.Change{Kind: recovery.ChangeInstallation, Installation: recovery.Installation{ID: recoveryPrimary, Tenant: ws, Skew: recoverySkew}}),
		approved(recovery.Change{Kind: recovery.ChangeWriter, Writer: recovery.Writer{Installation: recoveryPrimary, ID: recoveryERPWriter, Kind: recovery.WriterERP,
			MaxCall: time.Minute}}),
		approved(recovery.Change{Kind: recovery.ChangeWriter, Writer: recovery.Writer{Installation: recoveryPrimary, ID: recoveryIssuer, Kind: recovery.WriterIssuance,
			MaxCall: 2 * time.Minute}}),
		approved(recovery.Change{Kind: recovery.ChangeVerifier, Verifier: recovery.Verifier{Installation: recoveryPrimary, ID: recoveryVerifier}}),
		approved(recovery.Change{Kind: recovery.ChangeExternal, External: recovery.External{Installation: recoveryPrimary, ID: recoveryLedger, Kind: recovery.WriterERP}}),
		approved(recovery.Change{Kind: recovery.ChangeExternal, External: recovery.External{Installation: recoveryPrimary, ID: recoveryRegister,
			Kind: recovery.WriterIssuance}}),
		func() error {
			return s.recoveryCall("permit", func() (err error) {
				token, err = s.recovery.Permit(recoveryPrimary, recoveryERPWriter, writer)
				return
			})
		},
		call("sample-invoice-1", 1, &confirmed),
		func() error {
			return s.recoveryCall("done", func() error {
				_, err := s.recovery.Done(recoveryPrimary, confirmed.ID, recovery.OutcomeConfirmed, writer)
				return err
			})
		},
		call("sample-invoice-2", 2, &open),
		record(recovery.Event{Kind: recovery.EventDelete, Domain: recovery.DomainEvidence, Key: "sample-bundle-1"}, writer),
		record(recovery.Event{Kind: recovery.EventRevokeAccess, Domain: recovery.DomainDatabase, Key: "sample-user-1"}, verifier),
		approved(recovery.Change{Kind: recovery.ChangeBegin, Begin: point}),
	}
	for _, inv := range recoveryManifest(recovery.DomainOutbox) {
		steps = append(steps, func() error {
			return s.recoveryCall("report", func() error { _, err := s.recovery.Report(recoveryPrimary, inv, verifier); return err })
		})
	}
	steps = append(steps,
		approved(recovery.Change{Kind: recovery.ChangeInstallation, Installation: recovery.Installation{ID: recoveryStandby, Tenant: ws, Skew: recoverySkew}}),
		approved(recovery.Change{Kind: recovery.ChangeWriter, Writer: recovery.Writer{Installation: recoveryStandby, ID: recoveryStandbyW, Kind: recovery.WriterERP,
			MaxCall: time.Minute}}),
		func() error { return s.recoveryLock(s.rec.install(recoveryStandby), first, installSeedReason) },
		func() error {
			_, err := s.recoveryPropose(recovery.Change{Kind: recovery.ChangeUnlock, Unlock: recovery.Unlock{Installation: recoveryStandby, Generation: 2,
				Lost: recovery.LostDigest(nil)}}, second, installSeedReason)
			return err
		},
	)
	return run(steps...)
}
