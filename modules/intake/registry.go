package intake

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type artifact struct {
	spec          Spec
	print         string
	declared      map[string]decimal.Decimal
	state         State
	reasons       []string
	abandon       AbandonReason
	retire        RetireReason
	chunks        map[int]string
	rows          map[string]row
	count         int
	held          int
	sums          map[string]*exact
	digest        string
	sequence      uint64
	verified      *Verification
	contradiction int
	registered    time.Time
	changed       time.Time
	original      *artifact
}

type document struct {
	id        string
	subject   Subject
	kind      KeyKind
	highest   uint64
	artifacts []*artifact
}

type manifest struct {
	view    ManifestView
	sums    map[string]string
	records []ManifestRecord
}

type subject struct {
	documents []*document
	manifests []*manifest
}

type workspace struct {
	artifacts map[string]*artifact
	documents map[string]*document
	subjects  map[Subject]*subject
	records   int
	sequence  uint64
}

type Registry struct {
	mu         sync.Mutex
	built      bool
	limits     Limits
	profile    decimal.Profile
	clock      func() time.Time
	last       time.Time
	workspaces map[string]*workspace
}

func NewRegistry(limits Limits, profile decimal.Profile, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || !profile.Valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, profile: profile, clock: clock, workspaces: map[string]*workspace{}}, nil
}

func (r *Registry) enter() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	if !r.built {
		return false
	}
	if read := r.clock(); read.After(r.last) {
		r.last = read
	}
	return true
}

func (r *Registry) leave() {
	if r != nil {
		r.mu.Unlock()
	}
}

func (r *Registry) find(workspaceID, id string) (*workspace, *artifact) {
	ws := r.workspaces[workspaceID]
	if ws == nil {
		return nil, nil
	}
	return ws, ws.artifacts[id]
}

func fingerprint(s Spec, declared map[string]decimal.Decimal) string {
	currencies := make([]string, 0, len(declared))
	for currency := range declared {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	fields := []string{s.ID, s.Subject.WorkspaceID, s.Subject.Provider, s.Subject.Scope, s.Subject.Period, s.Document, string(s.Kind),
		strconv.FormatUint(s.Revision, 10), s.Bytes, string(s.Role), string(s.Holds), s.Original, s.Method,
		strconv.Itoa(s.Chunks), strconv.Itoa(s.Rows), string(s.Totals), strconv.FormatInt(s.Size, 10), strconv.FormatInt(s.Expanded, 10),
		s.Location, s.Collection, string(s.Scan), string(s.Syntax), string(s.Bodies)}
	for _, currency := range currencies {
		fields = append(fields, currency, declared[currency].String())
	}
	return strings.Join(fields, "\x00")
}

func (r *Registry) shape(s Spec) (map[string]decimal.Decimal, error) {
	if !s.Subject.valid() || !validID(s.ID) || !validID(s.Document) || !validDigest(s.Bytes) || !validID(s.Location) {
		return nil, ErrInvalid
	}
	if s.Kind != EventKeys && s.Kind != LocatorKeys {
		return nil, ErrInvalid
	}
	if s.Revision < 1 || s.Size <= 0 || s.Expanded < s.Size || s.Chunks < 0 {
		return nil, ErrInvalid
	}
	if s.Collection != "" && !validID(s.Collection) {
		return nil, ErrInvalid
	}
	if !s.Scan.known() || !s.Syntax.known() || !s.Bodies.known() {
		return nil, ErrInvalid
	}
	switch s.Role {
	case Original:
		if s.Holds != Rows && s.Holds != Empty && s.Holds != AwaitingExtraction {
			return nil, ErrInvalid
		}
		if s.Original != "" || s.Method != "" {
			return nil, ErrInvalid
		}
	case Derived:
		if s.Holds != "" || !validID(s.Original) || !validID(s.Method) {
			return nil, ErrInvalid
		}
	default:
		return nil, ErrInvalid
	}
	declared := make(map[string]decimal.Decimal, len(s.Sums))
	for currency, text := range s.Sums {
		if !validCurrency(currency) {
			return nil, ErrInvalid
		}
		sum, err := decimal.Parse(r.profile, text)
		if err != nil {
			return nil, ErrInvalid
		}
		declared[currency] = sum
	}
	if !s.carriesRows() {
		if s.Chunks != 0 || s.Rows != 0 || len(s.Sums) != 0 || s.Totals != "" {
			return nil, ErrInvalid
		}
		return declared, nil
	}
	if s.Totals != SourceStated && s.Totals != CallerComputed {
		return nil, ErrInvalid
	}
	if s.Chunks == 0 || len(s.Sums) == 0 || s.Rows < s.Chunks || len(s.Sums) > s.Rows {
		return nil, ErrInvalid
	}
	if s.Chunks > r.limits.Chunks {
		return nil, ErrLimit
	}
	if s.Rows/s.Chunks > r.limits.ChunkRecords || s.Rows/s.Chunks == r.limits.ChunkRecords && s.Rows%s.Chunks != 0 {
		return nil, ErrInvalid
	}
	return declared, nil
}

func (r *Registry) faults(s Spec) []string {
	var reasons []string
	if s.Size > r.limits.MaxSize {
		reasons = append(reasons, TooLarge)
	}
	if s.Expanded/s.Size > r.limits.MaxRatio || s.Expanded/s.Size == r.limits.MaxRatio && s.Expanded%s.Size != 0 {
		reasons = append(reasons, RatioExceeded)
	}
	for _, check := range []struct {
		result  Check
		failed  string
		missing string
	}{{s.Scan, ScanFailed, ScanNotRun}, {s.Syntax, SyntaxFailed, SyntaxNotRun}, {s.Bodies, BodiesFound, BodiesNotRun}} {
		switch check.result {
		case Failed:
			reasons = append(reasons, check.failed)
		case NotRun:
			reasons = append(reasons, check.missing)
		}
	}
	return reasons
}

func (r *Registry) Register(s Spec) (State, []string, error) {
	defer r.leave()
	if !r.enter() {
		return "", nil, ErrInvalid
	}
	declared, err := r.shape(s)
	if err != nil {
		return "", nil, err
	}
	print := fingerprint(s, declared)
	ws := r.workspaces[s.Subject.WorkspaceID]
	if ws != nil {
		if held := ws.artifacts[s.ID]; held != nil {
			if held.print != print {
				return "", nil, ErrConflict
			}
			return held.state, append([]string(nil), held.reasons...), nil
		}
	}
	var doc *document
	if ws != nil {
		doc = ws.documents[s.Document]
	}
	if doc != nil && (doc.subject != s.Subject || doc.kind != s.Kind) {
		return "", nil, ErrConflict
	}
	var original *artifact
	if s.Role == Derived {
		if ws != nil {
			original = ws.artifacts[s.Original]
		}
		if original == nil {
			return "", nil, ErrNotFound
		}
		if original.spec.Holds != AwaitingExtraction || original.spec.Document != s.Document || original.spec.Revision != s.Revision {
			return "", nil, ErrRejected
		}
		if original.state != Staging && original.state != Sealed {
			return "", nil, ErrRejected
		}
	}
	if ws != nil && len(ws.artifacts) >= r.limits.Artifacts {
		return "", nil, ErrLimit
	}
	if ws == nil {
		ws = &workspace{artifacts: map[string]*artifact{}, documents: map[string]*document{}, subjects: map[Subject]*subject{}}
		r.workspaces[s.Subject.WorkspaceID] = ws
	}
	if doc == nil {
		doc = &document{id: s.Document, subject: s.Subject, kind: s.Kind}
		ws.documents[s.Document] = doc
		sub := ws.subjects[s.Subject]
		if sub == nil {
			sub = &subject{}
			ws.subjects[s.Subject] = sub
		}
		sub.documents = append(sub.documents, doc)
	}
	if s.Revision > doc.highest {
		doc.highest = s.Revision
	}
	stored := s
	stored.Sums = make(map[string]string, len(declared))
	for currency, sum := range declared {
		stored.Sums[currency] = sum.String()
	}
	made := &artifact{spec: stored, print: print, declared: declared, state: Staging, reasons: r.faults(s),
		chunks: map[int]string{}, rows: map[string]row{}, sums: map[string]*exact{}, registered: r.last, changed: r.last, original: original}
	if len(made.reasons) > 0 {
		made.state = Rejected
		made.rows = nil
	}
	ws.artifacts[s.ID] = made
	doc.artifacts = append(doc.artifacts, made)
	return made.state, append([]string(nil), made.reasons...), nil
}

func (r *Registry) drop(ws *workspace, a *artifact) {
	ws.records -= a.held
	a.held = 0
	a.rows = nil
}

func (r *Registry) Stage(workspaceID, id string, number int, records []Record) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || number < 1 || len(records) == 0 || len(records) > r.limits.ChunkRecords {
		return "", ErrInvalid
	}
	keys := make([]string, len(records))
	rows := make([]row, len(records))
	for index, record := range records {
		if !validID(record.Key) || !validDigest(record.Digest) || !validCurrency(record.Currency) {
			return "", ErrInvalid
		}
		amount, err := decimal.Parse(r.profile, record.Amount)
		if err != nil {
			return "", ErrInvalid
		}
		keys[index] = record.Key
		rows[index] = row{amount: amount, currency: record.Currency, digest: record.Digest}
	}
	ws, a := r.find(workspaceID, id)
	if a == nil {
		return "", ErrNotFound
	}
	if number > a.spec.Chunks {
		return "", ErrInvalid
	}
	sent := chunkDigest(keys, rows)
	staged, known := a.chunks[number]
	if known && staged == sent {
		return a.state, nil
	}
	if a.state != Staging {
		return "", ErrRejected
	}
	contradicts := known
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if _, held := a.rows[key]; held || seen[key] {
			contradicts = true
		}
		seen[key] = true
	}
	if contradicts {
		a.state = Inconsistent
		a.contradiction = number
		a.changed = r.last
		r.drop(ws, a)
		return a.state, nil
	}
	if len(records) > r.limits.Records-ws.records {
		return "", ErrLimit
	}
	for index, key := range keys {
		a.rows[key] = rows[index]
		sum := a.sums[rows[index].currency]
		if sum == nil {
			sum = newExact()
			a.sums[rows[index].currency] = sum
		}
		sum.add(rows[index].amount)
	}
	a.chunks[number] = sent
	a.count += len(records)
	a.held += len(records)
	ws.records += len(records)
	return a.state, nil
}

func (r *Registry) Staged(workspaceID, id string) (StagedView, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return StagedView{}, ErrInvalid
	}
	_, a := r.find(workspaceID, id)
	if a == nil {
		return StagedView{}, ErrNotFound
	}
	if a.rows == nil {
		return StagedView{}, ErrRejected
	}
	return StagedView{Chunks: len(a.chunks), Rows: a.count, Sums: texts(a.sums), Digest: recordDigest(a.rows), Complete: len(a.chunks) == a.spec.Chunks}, nil
}

func (r *Registry) Verify(workspaceID, id, digest, verifier, evidence string) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !validDigest(digest) || !validID(verifier) || !validID(evidence) {
		return ErrInvalid
	}
	_, a := r.find(workspaceID, id)
	if a == nil {
		return ErrNotFound
	}
	if a.spec.Role != Derived {
		return ErrRejected
	}
	if held := a.verified; held != nil {
		if held.Digest != digest || held.Verifier != verifier || held.Evidence != evidence {
			return ErrConflict
		}
		return nil
	}
	if a.state != Staging || len(a.chunks) != a.spec.Chunks {
		return ErrRejected
	}
	if recordDigest(a.rows) != digest {
		return ErrConflict
	}
	a.verified = &Verification{Digest: digest, Verifier: verifier, Evidence: evidence, At: r.last}
	return nil
}

func (a *artifact) gaps() []Gap {
	var found []Gap
	if len(a.chunks) != a.spec.Chunks {
		found = append(found, Gap{What: GapChunks})
	}
	if a.count != a.spec.Rows {
		found = append(found, Gap{What: GapRows})
	}
	currencies := map[string]bool{}
	for currency := range a.sums {
		currencies[currency] = true
	}
	for currency := range a.declared {
		currencies[currency] = true
	}
	ordered := make([]string, 0, len(currencies))
	for currency := range currencies {
		ordered = append(ordered, currency)
	}
	sort.Strings(ordered)
	for _, currency := range ordered {
		sum, staged := a.sums[currency]
		stated, declared := a.declared[currency]
		switch {
		case !declared:
			found = append(found, Gap{What: GapCurrency, Currency: currency})
		case !staged:
			found = append(found, Gap{What: GapNoRows, Currency: currency})
		case !sum.equals(stated):
			found = append(found, Gap{What: GapSum, Currency: currency})
		}
	}
	if a.spec.Role == Derived {
		if a.original.state != Sealed {
			found = append(found, Gap{What: GapOriginal})
		}
		if a.verified == nil {
			found = append(found, Gap{What: GapVerified})
		}
	}
	return found
}

func (r *Registry) Seal(workspaceID, id string) (State, []Gap, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return "", nil, ErrInvalid
	}
	ws, a := r.find(workspaceID, id)
	if a == nil {
		return "", nil, ErrNotFound
	}
	if a.state == Sealed || a.state == Retired {
		return a.state, nil, nil
	}
	if a.state != Staging {
		return "", nil, ErrRejected
	}
	if found := a.gaps(); len(found) > 0 {
		return "", found, ErrRejected
	}
	ws.sequence++
	a.state = Sealed
	a.sequence = ws.sequence
	a.digest = recordDigest(a.rows)
	a.changed = r.last
	return a.state, nil, nil
}

func (r *Registry) Abandon(workspaceID, id string, reason AbandonReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !reason.known() {
		return ErrInvalid
	}
	ws, a := r.find(workspaceID, id)
	if a == nil {
		return ErrNotFound
	}
	if a.state == Abandoned {
		if a.abandon != reason {
			return ErrConflict
		}
		return nil
	}
	if a.state != Staging {
		return ErrRejected
	}
	a.state = Abandoned
	a.abandon = reason
	a.changed = r.last
	r.drop(ws, a)
	return nil
}

func (r *Registry) Retire(workspaceID, id string, reason RetireReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !reason.known() {
		return ErrInvalid
	}
	ws, a := r.find(workspaceID, id)
	if a == nil {
		return ErrNotFound
	}
	if a.state == Retired {
		if a.retire != reason {
			return ErrConflict
		}
		return nil
	}
	if a.state != Sealed {
		return ErrRejected
	}
	a.state = Retired
	a.retire = reason
	a.changed = r.last
	r.drop(ws, a)
	return nil
}

func (r *Registry) Artifact(workspaceID, id string) (ArtifactView, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return ArtifactView{}, ErrInvalid
	}
	_, a := r.find(workspaceID, id)
	if a == nil {
		return ArtifactView{}, ErrNotFound
	}
	view := ArtifactView{Spec: a.spec, State: a.state, Reasons: append([]string(nil), a.reasons...), AbandonReason: a.abandon, RetireReason: a.retire,
		Rows: a.count, Sums: texts(a.sums), Digest: a.digest, Sequence: a.sequence, Contradiction: a.contradiction, Registered: a.registered, Changed: a.changed}
	view.Spec.Sums = make(map[string]string, len(a.spec.Sums))
	for currency, sum := range a.spec.Sums {
		view.Spec.Sums[currency] = sum
	}
	if a.verified != nil {
		copied := *a.verified
		view.Verification = &copied
	}
	return view, nil
}
