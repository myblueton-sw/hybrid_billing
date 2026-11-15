package migration

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	batchTag  = "hybrid-billing/migration-batch/v1"
	effectTag = "hybrid-billing/migration-effect/v1"
)

var documentKinds = map[string]bool{DocumentInvoice: true, DocumentCredit: true, DocumentDebit: true, DocumentReceipt: true}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func fenceKey(prefix string, parts ...string) string {
	var e encoder
	for _, p := range parts {
		e.text(p)
	}
	return prefix + digest(e.out.Bytes())[:32]
}

func printable(text string, limit int) bool {
	if text == "" || len(text) > limit {
		return false
	}
	for index := 0; index < len(text); index++ {
		if text[index] < 0x20 || text[index] > 0x7e {
			return false
		}
	}
	return true
}

func secret(values ...string) bool {
	for _, v := range values {
		upper := strings.ToUpper(v)
		if strings.Contains(upper, "PRIVATE") || strings.Contains(upper, "BEGIN") {
			return true
		}
	}
	return false
}

func currency(code string) bool {
	if len(code) != 3 {
		return false
	}
	for index := 0; index < 3; index++ {
		if code[index] < 'A' || code[index] > 'Z' {
			return false
		}
	}
	return true
}

func (d Document) valid(now time.Time) bool {
	return validID(d.SourceID) && documentKinds[d.Kind] && validID(d.Workspace) && validID(d.SourceSeries) && d.Number >= 1 && d.Number <= maxNumber &&
		printable(d.Display, 64) && !d.IssuedAt.Before(time.Unix(0, 0)) && !d.IssuedAt.After(now) && currency(d.Currency) && d.Total >= 0 && d.Total <= maxNumber &&
		validHex(d.Content) && (d.ExternalKey == "" || validID(d.ExternalKey)) && (!d.Submitted || d.ExternalKey != "")
}

func boolText(flag bool) string {
	if flag {
		return "1"
	}
	return "0"
}

func (d Document) code() string {
	var e encoder
	for _, value := range []string{"document", d.SourceID, d.Kind, d.Workspace, d.SourceSeries, strconv.FormatInt(d.Number, 10), d.Display,
		strconv.FormatInt(d.IssuedAt.UTC().UnixNano(), 10), d.Currency, strconv.FormatInt(d.Total, 10), d.Content, boolText(d.Delivered), boolText(d.Submitted), d.ExternalKey} {
		e.text(value)
	}
	return string(e.out.Bytes())
}

func keyView(k KeyReference) (KeyView, error) {
	if !validID(k.SourceID) || len(k.DER) > maxDER || k.Kind != KeyCertificate && k.Kind != KeyPublic {
		return KeyView{}, ErrInvalid
	}
	if _, err := x509.ParsePKCS8PrivateKey(k.DER); err == nil {
		return KeyView{}, refuse(ReasonSecretMaterial)
	}
	if _, err := x509.ParsePKCS1PrivateKey(k.DER); err == nil {
		return KeyView{}, refuse(ReasonSecretMaterial)
	}
	if _, err := x509.ParseECPrivateKey(k.DER); err == nil {
		return KeyView{}, refuse(ReasonSecretMaterial)
	}
	var err error
	if k.Kind == KeyCertificate {
		_, err = x509.ParseCertificate(k.DER)
	} else {
		_, err = x509.ParsePKIXPublicKey(k.DER)
	}
	if err != nil {
		return KeyView{}, ErrInvalid
	}
	return KeyView{SourceID: k.SourceID, Kind: k.Kind, DER: append([]byte(nil), k.DER...), Fingerprint: digest(k.DER)}, nil
}

func keyCode(k KeyView) string {
	var e encoder
	for _, value := range []string{"key", k.SourceID, k.Kind, k.Fingerprint} {
		e.text(value)
	}
	return string(e.out.Bytes())
}

func copyDocument(d Document) *Document { return &d }

func (r *Registry) importing(id, batchID string, actor Actor) (*source, *batch, error) {
	if !actor.valid() || !validID(id) || !validID(batchID) {
		return nil, nil, ErrInvalid
	}
	s := r.sources[id]
	if s == nil {
		return nil, nil, ErrNotFound
	}
	if actor.Kind != KindSystem || actor.ID != s.view.Importer {
		return nil, nil, refuse(ReasonNotImporter)
	}
	b := s.batches[batchID]
	if b == nil {
		return nil, nil, ErrNotFound
	}
	if b.state != BatchOpen {
		return nil, nil, refuse(ReasonNotOpen)
	}
	return s, b, nil
}

func (r *Registry) Add(id, batchID string, item Item, actor Actor) error {
	now := r.enter()
	defer r.leave()
	var sourceID, code string
	switch {
	case item.Document != nil && item.Key == nil:
		d := *item.Document
		if !d.valid(now) {
			return ErrInvalid
		}
		d.IssuedAt = d.IssuedAt.UTC().Round(0)
		item = Item{Document: copyDocument(d)}
		sourceID, code = d.SourceID, d.code()
	case item.Key != nil && item.Document == nil:
		k, err := keyView(*item.Key)
		if err != nil {
			return err
		}
		item = Item{Key: &KeyReference{SourceID: k.SourceID, Kind: k.Kind, DER: k.DER}}
		sourceID, code = k.SourceID, keyCode(k)
	default:
		return ErrInvalid
	}
	s, b, err := r.importing(id, batchID, actor)
	if err != nil {
		return err
	}
	if d := item.Document; d != nil && secret(d.SourceID, d.Display, d.ExternalKey, d.Workspace, d.SourceSeries) || item.Key != nil && secret(item.Key.SourceID) {
		return refuse(ReasonSecretMaterial)
	}
	if item.Document != nil && !listed(s.view.Workspaces, item.Document.Workspace) {
		return refuse(ReasonWorkspaceNotAllowed)
	}
	if existing, ok := b.codes[sourceID]; ok {
		if existing == code {
			return nil
		}
		return ErrConflict
	}
	if len(b.codes) >= b.spec.Count {
		return ErrLimit
	}
	b.items[sourceID], b.codes[sourceID] = item, code
	return nil
}

func sortedIDs(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func batchDigest(tenant, sourceID string, b *batch) string {
	var e encoder
	for _, value := range []string{batchTag, tenant, sourceID, b.spec.ID, strconv.Itoa(b.spec.Count)} {
		e.text(value)
	}
	for _, id := range sortedIDs(b.codes) {
		e.text(b.codes[id])
	}
	return digest(e.out.Bytes())
}

func (r *Registry) Seal(id, batchID string, actor Actor) (BatchView, error) {
	r.enter()
	s, b, err := r.importing(id, batchID, actor)
	if err != nil {
		r.leave()
		return BatchView{}, err
	}
	b.computed = batchDigest(s.view.Tenant, s.view.ID, b)
	if len(b.codes) != b.spec.Count || b.computed != b.spec.Digest {
		b.state, s.open = BatchMismatch, ""
		b.items, b.codes = nil, nil
		defer r.leave()
		return r.view(s, b), nil
	}
	tenant, contents := s.view.Tenant, []string{}
	for _, id := range sortedIDs(b.codes) {
		if d := b.items[id].Document; d != nil {
			contents = append(contents, d.Content)
		}
	}
	r.leave()
	missing := false
	for _, c := range contents {
		present, err := r.config.Evidence.Present(tenant, c)
		if err != nil {
			return BatchView{}, refuse(ReasonDependency)
		}
		if !present {
			missing = true
			break
		}
	}
	r.enter()
	defer r.leave()
	if b.state != BatchOpen || actor.ID != s.view.Importer {
		return BatchView{}, refuse(ReasonStale)
	}
	if missing {
		b.state, s.open = BatchMissing, ""
		b.items, b.codes = nil, nil
	} else {
		b.state = BatchVerified
	}
	return r.view(s, b), nil
}

func (r *Registry) Reject(id, batchID, reason string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validText(reason) || !validID(id) || !validID(batchID) {
		return ErrInvalid
	}
	s := r.sources[id]
	if s == nil || s.batches[batchID] == nil {
		return ErrNotFound
	}
	b := s.batches[batchID]
	switch b.state {
	case BatchOpen, BatchVerified:
		b.state, s.open = BatchRejected, ""
		b.rejectedBy, b.reason = actor.ID, reason
		b.items, b.codes = nil, nil
		r.retireAccepts(id, batchID)
	case BatchRejected:
	default:
		return refuse(ReasonNotOpen)
	}
	return nil
}

func (r *Registry) retireAccepts(scopeName, batchID string) {
	sc := r.scopes[scopeName]
	if sc == nil {
		return
	}
	for _, number := range append([]uint64(nil), sc.pending...) {
		if p := r.proposals[number]; p.view.Change.Kind == ChangeAccept && p.view.Change.Accept.Batch == batchID {
			delete(r.proposals, number)
			sc.pending = drop(sc.pending, number)
		}
	}
}

type plan struct {
	classes []Classified
	newDocs []*Document
	newKeys []KeyView
	series  map[[2]string]int64
	printed map[[2]string]bool
	keys    map[[2]string]bool
	fresh   []Classified
	effect  string
}

func (r *Registry) classify(s *source, b *batch) (plan, error) {
	t := r.tenant(s.view.Tenant)
	p := plan{series: map[[2]string]int64{}, printed: map[[2]string]bool{}, keys: map[[2]string]bool{}}
	numbers, displays, externals := map[[3]string]string{}, map[[2]string]string{}, map[[2]string]string{}
	for _, id := range sortedIDs(b.codes) {
		item, code := b.items[id], b.codes[id]
		c := Classified{SourceID: id, Class: ClassNew, Incoming: digest([]byte(code))}
		stored := s.history[id]
		if stored != nil {
			c.Class, c.Stored = ClassDuplicate, digest([]byte(stored.code))
			if stored.code != code {
				c.Class = ClassConflict
			}
		}
		if d := item.Document; d != nil {
			mapped := s.mappings[[2]string{d.SourceSeries, d.Workspace}]
			if mapped == "" {
				return plan{}, refuse(ReasonUnmappedSeries)
			}
			nk := [3]string{d.Workspace, mapped, strconv.FormatInt(d.Number, 10)}
			dk := [2]string{d.Workspace, d.Display}
			ek := [2]string{d.Workspace, d.ExternalKey}
			if c.Class == ClassNew {
				switch {
				case t.numbers[nk] != "" || numbers[nk] != "":
					c.Class, c.Stored = ClassNumberConflict, t.numbers[nk]+numbers[nk]
				case t.displays[dk] != "" || displays[dk] != "":
					c.Class, c.Stored = ClassNumberConflict, t.displays[dk]+displays[dk]
				case d.ExternalKey != "" && (t.external[ek] != "" || externals[ek] != ""):
					c.Class, c.Stored = ClassExternalConflict, t.external[ek]+externals[ek]
				default:
					numbers[nk], displays[dk] = c.Incoming, c.Incoming
					if d.ExternalKey != "" {
						externals[ek] = c.Incoming
					}
					p.newDocs = append(p.newDocs, d)
				}
			}
			sk := [2]string{d.Workspace, mapped}
			if d.Number > t.high[sk] && d.Number > p.series[sk] {
				p.series[sk] = d.Number
			}
			if pk := [2]string{d.Workspace, fenceKey("dsp-", s.view.Tenant, d.Display)}; !t.printed[pk] {
				p.printed[pk] = true
			}
			delivered, externalKeys := d.Delivered || d.Submitted, []string{d.ExternalKey}
			if stored != nil && stored.document != nil {
				delivered = delivered || stored.document.Delivered || stored.document.Submitted
				externalKeys = append(externalKeys, stored.document.ExternalKey)
			}
			if delivered {
				if k := [2]string{d.Workspace, fenceKey("src-", s.view.Tenant, s.view.ID, id)}; !t.delivery[k] {
					p.keys[k] = true
				}
				for _, x := range externalKeys {
					if k := [2]string{d.Workspace, fenceKey("ext-", s.view.Tenant, x)}; x != "" && !t.delivery[k] {
						p.keys[k] = true
					}
				}
			}
		} else if c.Class == ClassNew {
			k, _ := keyView(*item.Key)
			p.newKeys = append(p.newKeys, k)
		}
		if c.Class != ClassNew && c.Class != ClassDuplicate && !s.seen[quarantineKey(c)] {
			p.fresh = append(p.fresh, c)
		}
		p.classes = append(p.classes, c)
	}
	p.effect = effectDigest(p)
	return p, nil
}

func quarantineKey(c Classified) string {
	return c.SourceID + "\x00" + c.Class + "\x00" + c.Incoming + "\x00" + c.Stored
}

func seriesChanges(m map[[2]string]int64) []SeriesChange {
	out := []SeriesChange{}
	for k, n := range m {
		out = append(out, SeriesChange{Workspace: k[0], Series: k[1], Number: n})
	}
	sort.Slice(out, func(a, b int) bool {
		return out[a].Workspace < out[b].Workspace || out[a].Workspace == out[b].Workspace && out[a].Series < out[b].Series
	})
	return out
}

func effectDigest(p plan) string {
	var e encoder
	e.text(effectTag)
	e.count(len(p.classes))
	for _, c := range p.classes {
		for _, value := range []string{c.SourceID, c.Class, c.Incoming, c.Stored} {
			e.text(value)
		}
	}
	changes := seriesChanges(p.series)
	e.count(len(changes))
	for _, ch := range changes {
		e.text(ch.Workspace)
		e.text(ch.Series)
		e.number(ch.Number)
	}
	e.count(len(p.printed))
	e.count(len(p.keys))
	return digest(e.out.Bytes())
}

func (r *Registry) room(s *source, p plan) error {
	t := r.tenant(s.view.Tenant)
	newSeries := 0
	for k := range p.series {
		if _, ok := t.high[k]; !ok {
			newSeries++
		}
	}
	switch {
	case t.history+len(p.newDocs) > maxHistory, r.history+len(p.newDocs) > maxHistoryTotal,
		t.keys+len(p.newKeys) > maxKeyRefs,
		len(s.quarantine)+len(p.fresh) > maxQuarantine,
		len(t.high)+newSeries > maxSeries, r.series+newSeries > maxSeriesTotal,
		len(t.printed)+len(p.printed) > maxPrinted, r.printed+len(p.printed) > maxPrintedTotal,
		len(t.delivery)+len(p.keys) > maxDelivery, r.delivery+len(p.keys) > maxDeliveryTotal:
		return ErrLimit
	}
	return nil
}

func (r *Registry) commit(s *source, b *batch, p plan) {
	t := r.tenant(s.view.Tenant)
	for _, d := range p.newDocs {
		code := d.code()
		incoming := digest([]byte(code))
		s.history[d.SourceID] = &entry{code: code, document: copyDocument(*d)}
		mapped := s.mappings[[2]string{d.SourceSeries, d.Workspace}]
		t.numbers[[3]string{d.Workspace, mapped, strconv.FormatInt(d.Number, 10)}] = incoming
		t.displays[[2]string{d.Workspace, d.Display}] = incoming
		if d.ExternalKey != "" {
			t.external[[2]string{d.Workspace, d.ExternalKey}] = incoming
		}
	}
	for _, k := range p.newKeys {
		kk := k
		s.history[k.SourceID] = &entry{code: keyCode(k), key: &kk}
	}
	t.history += len(p.newDocs)
	r.history += len(p.newDocs)
	t.keys += len(p.newKeys)
	for _, c := range p.fresh {
		s.quarantine = append(s.quarantine, c)
		s.seen[quarantineKey(c)] = true
	}
	for k, n := range p.series {
		if _, ok := t.high[k]; !ok {
			r.series++
		}
		t.high[k] = n
	}
	for k := range p.printed {
		t.printed[k] = true
		r.printed++
	}
	for k := range p.keys {
		t.delivery[k] = true
		r.delivery++
	}
	b.state, s.open = BatchAccepted, ""
	b.summary = map[string]int{}
	for _, c := range p.classes {
		b.summary[c.Class]++
	}
	b.effect = p.effect
	b.items, b.codes = nil, nil
	r.retireAccepts(s.view.ID, b.spec.ID)
}
