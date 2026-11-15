package recovery

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"time"
)

const manifestTag = "hybrid-billing/recovery-manifest/v1"

var (
	domains     = []string{DomainDatabase, DomainEvidence, DomainKeys, DomainPolicies, DomainModules, DomainNumbering, DomainSubmissions, DomainOutbox, DomainDeletions}
	writerKinds = map[string]bool{WriterERP: true, WriterIssuance: true, WriterDelivery: true, WriterPayment: true, WriterMessage: true}
)

func (inv Inventory) valid() bool {
	if len(inv.Chunks) > maxChunks {
		return false
	}
	var next uint64
	for _, c := range inv.Chunks {
		if c.Start != next || c.End <= c.Start || !validHex(c.Checksum) {
			return false
		}
		next = c.End
	}
	return next == inv.Items
}

func manifestDigest(manifest []Inventory) string {
	var e encoder
	e.text(manifestTag)
	e.count(len(manifest))
	for _, inv := range manifest {
		e.text(inv.Domain)
		e.text(strconv.FormatUint(inv.Items, 10))
		e.count(len(inv.Chunks))
		for _, c := range inv.Chunks {
			e.text(strconv.FormatUint(c.Start, 10))
			e.text(strconv.FormatUint(c.End, 10))
			e.text(c.Checksum)
		}
	}
	sum := sha256.Sum256(e.out.Bytes())
	return hex.EncodeToString(sum[:])
}

func copyManifest(manifest []Inventory) []Inventory {
	out := make([]Inventory, len(manifest))
	for index, inv := range manifest {
		inv.Chunks = append([]Chunk(nil), inv.Chunks...)
		out[index] = inv
	}
	return out
}

func (p RestorePoint) valid() bool {
	if !validID(p.Installation) || p.Generation == 0 || !validHex(p.Backup) || !validHex(p.ManifestDigest) || p.AsOf.IsZero() || len(p.Manifest) != len(domains) {
		return false
	}
	for index, inv := range p.Manifest {
		if inv.Domain != domains[index] || !inv.valid() {
			return false
		}
	}
	return true
}

func clean(c Change) (Change, string, bool) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangeInstallation:
		out.Installation = c.Installation
		i := c.Installation
		return out, registerScope, validID(i.ID) && validID(i.Tenant) && i.Skew >= 0 && i.Skew <= maxSkew
	case ChangeWriter:
		out.Writer = c.Writer
		w := c.Writer
		return out, w.Installation, validID(w.Installation) && validID(w.ID) && writerKinds[w.Kind] && w.MaxCall >= minMaxCall && w.MaxCall <= maxMaxCall
	case ChangeExternal:
		out.External = c.External
		x := c.External
		return out, x.Installation, validID(x.Installation) && validID(x.ID) && writerKinds[x.Kind]
	case ChangeVerifier:
		out.Verifier = c.Verifier
		return out, c.Verifier.Installation, validID(c.Verifier.Installation) && validID(c.Verifier.ID)
	case ChangeBegin:
		out.Begin = c.Begin
		out.Begin.AsOf = c.Begin.AsOf.UTC().Round(0)
		out.Begin.Manifest = copyManifest(c.Begin.Manifest)
		return out, c.Begin.Installation, c.Begin.valid()
	case ChangeUnlock:
		out.Unlock = c.Unlock
		return out, c.Unlock.Installation, validID(c.Unlock.Installation) && c.Unlock.Generation > 0 && validHex(c.Unlock.Lost)
	case ChangePrune:
		out.Prune = c.Prune
		out.Prune.Before = c.Prune.Before.UTC().Round(0)
		return out, c.Prune.Installation, validID(c.Prune.Installation) && !c.Prune.Before.IsZero()
	case ChangeAbandon:
		out.Abandon = c.Abandon
		return out, c.Abandon.Installation, validID(c.Abandon.Installation) && validID(c.Abandon.Recovery) && c.Abandon.Generation > 0
	}
	return out, registerScope, false
}

func encodeChange(c Change) []byte {
	var e encoder
	for _, value := range []string{c.Kind, c.Installation.ID, c.Installation.Tenant, c.Writer.Installation, c.Writer.ID, c.Writer.Kind, c.External.Installation,
		c.External.ID, c.External.Kind, c.Verifier.Installation, c.Verifier.ID, c.Begin.Installation, c.Begin.Backup, c.Begin.ManifestDigest,
		manifestDigest(c.Begin.Manifest), c.Unlock.Installation, c.Unlock.Lost, c.Prune.Installation, c.Abandon.Installation, c.Abandon.Recovery} {
		e.text(value)
	}
	for _, n := range []int64{int64(c.Installation.Skew), int64(c.Writer.MaxCall), int64(c.Begin.Generation), c.Begin.AsOf.UnixNano(), int64(c.Unlock.Generation),
		c.Prune.Before.UnixNano(), int64(c.Abandon.Generation)} {
		e.number(n)
	}
	return e.out.Bytes()
}

func copyChange(c Change) Change {
	c.Begin.Manifest = copyManifest(c.Begin.Manifest)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func (r *Registry) taken(id string) bool {
	_, ok := r.identities[id]
	return ok
}

func (r *Registry) check(c Change, now time.Time) error {
	if c.Kind == ChangeInstallation {
		switch {
		case r.installations[c.Installation.ID] != nil:
			return ErrConflict
		case len(r.installations) >= maxInstallations:
			return ErrLimit
		}
		return nil
	}
	s := r.installations[scopeOf(c)]
	switch c.Kind {
	case ChangeWriter:
		active := 0
		for _, w := range s.writers {
			if !w.removed {
				active++
			}
		}
		switch {
		case r.taken(c.Writer.ID):
			return refuse(ReasonIdentityTaken)
		case active >= maxWriters:
			return ErrLimit
		}
	case ChangeExternal:
		switch {
		case s.externals[c.External.ID].ID != "":
			return ErrConflict
		case len(s.externals) >= maxExternals:
			return ErrLimit
		}
	case ChangeVerifier:
		if r.taken(c.Verifier.ID) {
			return refuse(ReasonIdentityTaken)
		}
	case ChangeBegin:
		p := c.Begin
		switch {
		case p.AsOf.After(now):
			return ErrInvalid
		case p.Generation != s.generation:
			return refuse(ReasonGeneration)
		case manifestDigest(p.Manifest) != p.ManifestDigest:
			return refuse(ReasonManifestMismatch)
		case s.current != nil && s.current.point != nil:
			return refuse(ReasonOpenRecovery)
		case !s.pruned.IsZero() && !p.AsOf.Add(-s.view.Skew).After(s.pruned):
			return refuse(ReasonPruned)
		}
	case ChangeUnlock:
		if c.Unlock.Generation != s.generation {
			return refuse(ReasonGeneration)
		}
		return r.checkUnlock(s, c.Unlock, now)
	case ChangePrune:
		switch {
		case c.Prune.Before.After(now):
			return ErrInvalid
		case s.current != nil:
			return refuse(ReasonOpenRecovery)
		}
	case ChangeAbandon:
		switch {
		case s.current == nil || s.current.point == nil || s.current.id != c.Abandon.Recovery:
			return refuse(ReasonNoRestorePoint)
		case c.Abandon.Generation != s.generation:
			return refuse(ReasonGeneration)
		}
	}
	return nil
}

func scopeOf(c Change) string {
	switch c.Kind {
	case ChangeWriter:
		return c.Writer.Installation
	case ChangeExternal:
		return c.External.Installation
	case ChangeVerifier:
		return c.Verifier.Installation
	case ChangeBegin:
		return c.Begin.Installation
	case ChangeUnlock:
		return c.Unlock.Installation
	case ChangePrune:
		return c.Prune.Installation
	}
	return c.Abandon.Installation
}

func (r *Registry) apply(c Change, now time.Time) {
	if c.Kind == ChangeInstallation {
		r.installations[c.Installation.ID] = &site{view: c.Installation, generation: 1, state: StateServing, writers: map[string]*writer{}, externals: map[string]External{},
			tokens: map[uint64]*permit{}, open: map[string]*call{}, closedByID: map[string]*call{}, high: map[[2]string]int64{}, fenced: map[[2]string]bool{}}
		return
	}
	s := r.installations[scopeOf(c)]
	switch c.Kind {
	case ChangeWriter:
		s.writers[c.Writer.ID] = &writer{view: c.Writer}
		r.identities[c.Writer.ID] = s.view.ID
	case ChangeExternal:
		s.externals[c.External.ID] = c.External
		if s.current != nil && s.current.point != nil {
			s.current.externals[c.External.ID] = &pages{next: 1}
		}
	case ChangeVerifier:
		s.verifier = c.Verifier.ID
		r.identities[c.Verifier.ID] = s.view.ID
	case ChangeBegin:
		r.begin(s, copyChange(c).Begin, now)
	case ChangeUnlock:
		r.unlock(s)
	case ChangePrune:
		kept := s.events[:0]
		for _, e := range s.events {
			if e.view.At.After(c.Prune.Before) {
				kept = append(kept, e)
			} else {
				r.eventsTotal--
			}
		}
		s.events = kept
		if c.Prune.Before.After(s.pruned) {
			s.pruned = c.Prune.Before
		}
	case ChangeAbandon:
		r.abandon(s)
	}
}

func (r *Registry) bump(s *site) {
	s.generation++
	r.retireBegins(s.view.ID)
}

func sortedExternals(s *site) []string {
	out := make([]string, 0, len(s.externals))
	for id := range s.externals {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
