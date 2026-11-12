package intake

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type counted struct {
	kind     KeyKind
	document string
	source   string
	key      string
	row      row
	artifact string
}

type recordKey struct {
	kind     KeyKind
	document string
	key      string
}

type picture struct {
	blockers    []Blocker
	behind      []string
	same        []string
	staging     int
	records     []counted
	sums        map[string]*exact
	content     string
	attribution string
}

func (a *artifact) live() bool {
	return a.state == Sealed && (a.original == nil || a.original.state == Sealed)
}

func (a *artifact) carries() bool {
	return a.spec.Role == Derived || a.spec.Holds != AwaitingExtraction
}

func (s *subject) picture(profile decimal.Profile) picture {
	var pic picture
	type readable struct {
		doc  *document
		from *artifact
	}
	var readables []readable
	bytes := map[string]map[string]bool{}
	for _, doc := range s.documents {
		var counting uint64
		for _, a := range doc.artifacts {
			if a.state == Staging {
				pic.staging++
			}
			if !a.live() {
				continue
			}
			if a.spec.Revision > counting {
				counting = a.spec.Revision
			}
			if bytes[a.spec.Bytes] == nil {
				bytes[a.spec.Bytes] = map[string]bool{}
			}
			bytes[a.spec.Bytes][doc.id] = true
		}
		if counting < doc.highest {
			pic.behind = append(pic.behind, doc.id)
		}
		if counting == 0 {
			continue
		}
		var from *artifact
		differ := false
		for _, a := range doc.artifacts {
			if !a.live() || a.spec.Revision != counting || !a.carries() {
				continue
			}
			if from == nil {
				from = a
				continue
			}
			if a.digest != from.digest {
				differ = true
			}
			if a.sequence < from.sequence {
				from = a
			}
		}
		switch {
		case from == nil:
			pic.blockers = append(pic.blockers, Blocker{What: Unreadable, Document: doc.id})
		case differ:
			pic.blockers = append(pic.blockers, Blocker{What: InConflict, Document: doc.id})
		default:
			readables = append(readables, readable{doc, from})
		}
	}
	sort.Strings(pic.behind)
	for _, documents := range bytes {
		ids := make([]string, 0, len(documents))
		for id := range documents {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for first := 0; first < len(ids); first++ {
			for second := first + 1; second < len(ids); second++ {
				pic.same = append(pic.same, ids[first]+"+"+ids[second])
			}
		}
	}
	sort.Strings(pic.same)
	seen := map[string]bool{}
	unique := pic.same[:0]
	for _, pair := range pic.same {
		if !seen[pair] {
			seen[pair] = true
			unique = append(unique, pair)
		}
	}
	pic.same = unique
	sort.Slice(readables, func(i, j int) bool { return readables[i].from.sequence < readables[j].from.sequence })
	placed := map[recordKey]int{}
	conflicted := map[string]bool{}
	for _, read := range readables {
		for key, held := range read.from.rows {
			at := recordKey{kind: read.doc.kind, key: key}
			if read.doc.kind == LocatorKeys {
				at.document = read.doc.id
			}
			if index, taken := placed[at]; taken {
				if !pic.records[index].row.same(held) && !conflicted[key] {
					conflicted[key] = true
					pic.blockers = append(pic.blockers, Blocker{What: KeyConflict, Document: read.doc.id, Key: key, HeldBy: pic.records[index].source})
				}
				continue
			}
			placed[at] = len(pic.records)
			pic.records = append(pic.records, counted{kind: at.kind, document: at.document, source: read.doc.id, key: key, row: held, artifact: read.from.spec.ID})
		}
	}
	pic.sums = map[string]*exact{}
	for _, record := range pic.records {
		sum := pic.sums[record.row.currency]
		if sum == nil {
			sum = newExact()
			pic.sums[record.row.currency] = sum
		}
		sum.add(record.row.amount)
	}
	for currency, sum := range pic.sums {
		value, scale := sum.reduced()
		if _, err := decimal.New(profile, value, scale); err != nil {
			pic.blockers = append(pic.blockers, Blocker{What: SumOutside, Currency: currency})
		}
	}
	sort.Slice(pic.blockers, func(i, j int) bool {
		a, b := pic.blockers[i], pic.blockers[j]
		if a.What != b.What {
			return a.What < b.What
		}
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		if a.Currency != b.Currency {
			return a.Currency < b.Currency
		}
		return a.Document < b.Document
	})
	sort.Slice(pic.records, func(i, j int) bool {
		a, b := pic.records[i], pic.records[j]
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		if a.document != b.document {
			return a.document < b.document
		}
		return a.key < b.key
	})
	content, attribution := sha256.New(), sha256.New()
	for _, record := range pic.records {
		for _, field := range []string{string(record.kind), record.document, record.key, record.row.amount.String(), record.row.currency, record.row.digest} {
			put(content, field)
			put(attribution, field)
		}
		put(attribution, record.artifact)
	}
	pic.content = hex.EncodeToString(content.Sum(nil))
	pic.attribution = hex.EncodeToString(attribution.Sum(nil))
	return pic
}

func (r *Registry) listed(items []string) Listed {
	shown := items
	if len(shown) > r.limits.Listed {
		shown = shown[:r.limits.Listed]
	}
	return Listed{Total: len(items), Items: shown}
}

func (r *Registry) bounded(blockers []Blocker) []Blocker {
	shown := blockers
	if len(shown) > r.limits.Listed {
		shown = shown[:r.limits.Listed]
	}
	return shown
}

func (r *Registry) subject(at Subject) *subject {
	ws := r.workspaces[at.WorkspaceID]
	if ws == nil {
		return nil
	}
	return ws.subjects[at]
}

func (r *Registry) Publish(at Subject, expected uint64) (Published, error) {
	defer r.leave()
	if !r.enter() || !at.valid() {
		return Published{}, ErrInvalid
	}
	sub := r.subject(at)
	if sub == nil {
		return Published{}, ErrNotFound
	}
	pic := sub.picture(r.profile)
	if len(pic.blockers) > 0 {
		return Published{Blockers: r.bounded(pic.blockers), BlockerTotal: len(pic.blockers)}, ErrRejected
	}
	var latest *manifest
	if len(sub.manifests) > 0 {
		latest = sub.manifests[len(sub.manifests)-1]
		if latest.view.Attribution == pic.attribution {
			return Published{Number: latest.view.Number}, nil
		}
	}
	if expected != uint64(len(sub.manifests)) {
		return Published{}, ErrConflict
	}
	if len(pic.records) == 0 && latest == nil {
		return Published{}, ErrRejected
	}
	if len(sub.manifests) >= r.limits.Manifests {
		return Published{}, ErrLimit
	}
	made := &manifest{sums: texts(pic.sums), records: make([]ManifestRecord, len(pic.records))}
	for index, record := range pic.records {
		made.records[index] = ManifestRecord{Kind: record.kind, Document: record.document, Key: record.key, Amount: record.row.amount.String(),
			Currency: record.row.currency, Digest: record.row.digest, Artifact: record.artifact}
	}
	made.view = ManifestView{Subject: at, Number: expected + 1, Records: len(pic.records), Content: pic.content, Attribution: pic.attribution,
		Staging: pic.staging, Behind: r.listed(pic.behind), SameBytes: r.listed(pic.same), At: r.last}
	sub.manifests = append(sub.manifests, made)
	return Published{Number: made.view.Number, Written: true}, nil
}

func (r *Registry) Status(at Subject) (StatusView, error) {
	defer r.leave()
	if !r.enter() || !at.valid() {
		return StatusView{}, ErrInvalid
	}
	sub := r.subject(at)
	if sub == nil {
		return StatusView{}, ErrNotFound
	}
	pic := sub.picture(r.profile)
	view := StatusView{Latest: uint64(len(sub.manifests)), Blockers: r.bounded(pic.blockers), BlockerTotal: len(pic.blockers),
		Staging: pic.staging, Behind: r.listed(pic.behind), SameBytes: r.listed(pic.same)}
	switch {
	case len(pic.blockers) > 0:
		view.Differs = Blocked
	case len(sub.manifests) == 0:
		if len(pic.records) > 0 {
			view.Differs = DiffersContent
		}
	default:
		latest := sub.manifests[len(sub.manifests)-1].view
		if latest.Content != pic.content {
			view.Differs = DiffersContent
		} else if latest.Attribution != pic.attribution {
			view.Differs = DiffersAttribution
		}
	}
	return view, nil
}

func (r *Registry) manifest(at Subject, number uint64) (*manifest, error) {
	if !at.valid() || number < 1 {
		return nil, ErrInvalid
	}
	sub := r.subject(at)
	if sub == nil || number > uint64(len(sub.manifests)) {
		return nil, ErrNotFound
	}
	return sub.manifests[number-1], nil
}

func (r *Registry) Manifest(at Subject, number uint64) (ManifestView, error) {
	defer r.leave()
	if !r.enter() {
		return ManifestView{}, ErrInvalid
	}
	held, err := r.manifest(at, number)
	if err != nil {
		return ManifestView{}, err
	}
	view := held.view
	view.Sums = make(map[string]string, len(held.sums))
	for currency, sum := range held.sums {
		view.Sums[currency] = sum
	}
	view.Behind.Items = append([]string(nil), view.Behind.Items...)
	view.SameBytes.Items = append([]string(nil), view.SameBytes.Items...)
	return view, nil
}

func (r *Registry) Records(at Subject, number uint64, after, limit int) ([]ManifestRecord, error) {
	defer r.leave()
	if !r.enter() || after < 0 || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	held, err := r.manifest(at, number)
	if err != nil {
		return nil, err
	}
	if after >= len(held.records) {
		return nil, nil
	}
	end := len(held.records)
	if limit < end-after {
		end = after + limit
	}
	return append([]ManifestRecord(nil), held.records[after:end]...), nil
}
