package fx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"sort"
	"strconv"
	"time"
)

type printer struct{ sum hash.Hash }

func (p printer) field(text string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(text)))
	p.sum.Write(size[:])
	p.sum.Write([]byte(text))
}

func (p printer) number(value int) { p.field(strconv.Itoa(value)) }

func (p printer) hex() string { return hex.EncodeToString(p.sum.Sum(nil)) }

func contentDigest(workspace string, entries []Entry) string {
	p := printer{sum: sha256.New()}
	p.field(contentLabel)
	p.field(workspace)
	p.number(len(entries))
	for _, e := range entries {
		n := e.Need
		for _, text := range []string{n.ID, n.Purpose, n.Source, n.Target, n.Customer, n.Contract, n.Date} {
			p.field(text)
		}
		p.number(n.Offset)
		for _, text := range []string{e.Kind, e.Reason, e.Level, e.Policy} {
			p.field(text)
		}
		p.number(e.Version)
		p.field(e.AllowingPolicy)
		p.number(e.AllowingVersion)
		for _, text := range []string{e.Method, e.Construction, e.Pivot, e.Month, e.Averaging} {
			p.field(text)
		}
		p.number(e.LeastDays)
		p.field(e.Rule)
		p.number(e.MaxDays)
		p.number(len(e.Observations))
		for _, o := range e.Observations {
			for _, text := range []string{o.Key.Source, o.Key.Series, o.Key.Base, o.Key.Quote, o.Key.Date} {
				p.field(text)
			}
			p.number(o.Revision)
			p.number(o.Unit)
			for _, text := range []string{o.Value, o.Origin, o.Evidence, o.Reference} {
				p.field(text)
			}
		}
		p.number(e.FixedUnit)
		for _, text := range []string{e.FixedValue, e.FixedEvidence, e.FixedReference, e.DateUsed} {
			p.field(text)
		}
		p.number(e.DaysElapsed)
		p.number(e.VoidsPassed)
		p.field(e.QuotedRate)
		p.number(e.Unit)
		for _, text := range []string{e.PerSource, e.Spread, e.ExactQuote, e.SalesQuote} {
			p.field(text)
		}
		p.number(e.QuoteScale)
		for _, text := range []string{e.QuoteMode, e.Tolerance, e.Difference, e.SalesRate, e.Fee, e.Quantum, e.AmountMode, e.Order} {
			p.field(text)
		}
	}
	return p.hex()
}

func snapshotDigest(id, content string) string {
	p := printer{sum: sha256.New()}
	p.field(snapshotLabel)
	p.field(id)
	p.field(content)
	return p.hex()
}

func copyEntries(entries []Entry) []Entry {
	out := make([]Entry, len(entries))
	for index, entry := range entries {
		out[index] = entry
		out[index].Observations = append([]Used{}, entry.Observations...)
	}
	return out
}

func copySnapshot(s Snapshot) Snapshot {
	s.Entries = copyEntries(s.Entries)
	return s
}

func validNeed(need Need) bool {
	_, dated := parseDate(need.Date)
	return validID(need.ID) && purposes[need.Purpose] && validCurrency(need.Source) && validCurrency(need.Target) &&
		validID(need.Customer) && validID(need.Contract) && dated && need.Offset >= minOffset && need.Offset <= maxOffset
}

func sameNeeds(a, b []Need) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func (r *Registry) Prepare(workspace, id string, needs []Need) (Snapshot, error) {
	if len(needs) > r.limits.Needs {
		return Snapshot{}, ErrLimit
	}
	seen := map[string]bool{}
	for _, need := range needs {
		if !validNeed(need) || seen[need.ID] {
			return Snapshot{}, ErrInvalid
		}
		seen[need.ID] = true
	}
	if !validID(workspace) || !validID(id) || len(needs) == 0 {
		return Snapshot{}, ErrInvalid
	}
	copied := append([]Need{}, needs...)
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s != nil && s.snapshots[id] != nil {
		if !sameNeeds(s.snapshots[id].needs, copied) {
			return Snapshot{}, ErrConflict
		}
		return copySnapshot(s.snapshots[id].snapshot), nil
	}
	if s != nil && len(s.snapshots) >= r.limits.Snapshots {
		return Snapshot{}, ErrLimit
	}
	made := stored{needs: copied, snapshot: Snapshot{ID: id, PreparedAt: now, Entries: make([]Entry, 0, len(copied))}}
	for _, need := range copied {
		offset := now.Add(time.Duration(need.Offset) * time.Minute)
		local := time.Date(offset.Year(), offset.Month(), offset.Day(), 0, 0, 0, 0, time.UTC)
		entry, rated, err := (&resolver{r: r, s: s, local: local}).resolve(need)
		if err != nil {
			return Snapshot{}, err
		}
		if entry.Kind == EntryHeld {
			made.snapshot.Held++
		}
		made.snapshot.Entries = append(made.snapshot.Entries, entry)
		made.rates = append(made.rates, rated)
	}
	made.snapshot.ContentDigest = contentDigest(workspace, made.snapshot.Entries)
	made.snapshot.Digest = snapshotDigest(id, made.snapshot.ContentDigest)
	r.space(workspace).snapshots[id] = &made
	return copySnapshot(made.snapshot), nil
}

func (r *Registry) Snapshot(workspace, id string) (Snapshot, error) {
	if !validID(workspace) || !validID(id) {
		return Snapshot{}, ErrInvalid
	}
	r.enter()
	s := r.workspaces[workspace]
	var found *stored
	if s != nil {
		found = s.snapshots[id]
	}
	r.leave()
	if found == nil {
		return Snapshot{}, ErrNotFound
	}
	return copySnapshot(found.snapshot), nil
}

func (r *Registry) Snapshots(workspace, after string, limit int) ([]Listed, error) {
	if !validID(workspace) || (after != "" && !validID(after)) || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	out := []Listed{}
	if s == nil {
		return out, nil
	}
	ids := make([]string, 0, len(s.snapshots))
	for id := range s.snapshots {
		if id > after {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) > limit {
		ids = ids[:limit]
	}
	for _, id := range ids {
		snap := s.snapshots[id].snapshot
		out = append(out, Listed{ID: id, ContentDigest: snap.ContentDigest, Digest: snap.Digest})
	}
	return out, nil
}

func (r *Registry) Observation(workspace string, key Key) ([]RevisionView, error) {
	if !validID(workspace) || !validKey(key) {
		return nil, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || len(s.keys[key]) == 0 {
		return nil, ErrNotFound
	}
	out := make([]RevisionView, 0, len(s.keys[key]))
	for _, rev := range s.keys[key] {
		out = append(out, viewRevision(rev))
	}
	return out, nil
}

func (r *Registry) Policy(workspace, id string) (PolicyView, error) {
	if !validID(workspace) || !validID(id) {
		return PolicyView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.policies[id] == nil {
		return PolicyView{}, ErrNotFound
	}
	p := s.policies[id]
	out := PolicyView{ID: id, Attributes: p.attrs, Versions: make([]VersionView, 0, len(p.versions))}
	for _, v := range p.versions {
		out.Versions = append(out.Versions, viewVersion(v))
	}
	return out, nil
}
