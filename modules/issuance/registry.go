package issuance

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type kind struct {
	view KindView
}

type series struct {
	view    SeriesView
	numbers []*NumberView
}

type draft struct {
	revisions []*DraftView
	current   *DraftView
	reserved  *NumberView
	cancelled []*NumberView
	document  *document
}

type document struct {
	view    DocumentView
	file    []byte
	token   int
	running int
	credits []string
}

type space struct {
	kinds     map[string]*kind
	series    map[string]*series
	byKind    map[string]*series
	prefixes  map[string]bool
	drafts    map[string]*draft
	documents map[string]*document
	external  map[string]bool
	numbers   int
}

type Registry struct {
	mu         sync.Mutex
	config     Config
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(config Config, limits Limits, clock func() time.Time) (*Registry, error) {
	if config.Fields == nil || config.Terms == nil || config.Details == nil || config.Renderer == nil || config.Zones == nil || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{config: config, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (l Limits) valid() bool {
	if l.Lease < minLease || l.Lease > maxLease || l.Downloads < 1 || l.Downloads > maxDownloads {
		return false
	}
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Kinds, maxKinds}, {l.Drafts, maxDrafts}, {l.Documents, maxDocuments}, {l.Numbers, maxNumbers}, {l.Attempts, maxAttempts}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return true
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

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) is(id string) bool { return a.Kind == KindHuman && a.ID == id }

func (a Actor) human() error {
	switch a.Kind {
	case KindAssistant:
		return refuse(ReasonAINotAllowed)
	case KindSystem:
		return refuse(ReasonNotHuman)
	}
	return nil
}

func (a Actor) notAssistant() error {
	if a.Kind == KindAssistant {
		return refuse(ReasonAINotAllowed)
	}
	return nil
}

func (r *Registry) space(workspace string) *space {
	return r.workspaces[workspace]
}

func (r *Registry) open(workspace string) (*space, error) {
	if s := r.workspaces[workspace]; s != nil {
		return s, nil
	}
	if len(r.workspaces) >= r.limits.Workspaces {
		return nil, ErrLimit
	}
	s := &space{kinds: map[string]*kind{}, series: map[string]*series{}, byKind: map[string]*series{}, prefixes: map[string]bool{},
		drafts: map[string]*draft{}, documents: map[string]*document{}, external: map[string]bool{}}
	r.workspaces[workspace] = s
	return s, nil
}

var languages = map[string]bool{LanguageEnglish: true, LanguageKorean: true, LanguageArabic: true}

var classes = map[string]bool{ClassSelf: true, ClassInternal: true, ClassERP: true}

func subject(id string) bool { return validID(id) && len(id) <= maxSubject }

func (r *Registry) zone(name string) *time.Location {
	if name == "" || name == "Local" || len(name) > maxIDBytes {
		return nil
	}
	location, err := r.config.Zones(name)
	if err != nil {
		return nil
	}
	return location
}

func (r *Registry) kindForm(k Kind) bool {
	if !subject(k.ID) || r.zone(k.Zone) == nil || !classes[k.Class] || !validID(k.Tax) || len(k.Region) != 2 || len(k.Languages) == 0 || len(k.Languages) > len(languages) {
		return false
	}
	for index := 0; index < 2; index++ {
		if k.Region[index] < 'A' || k.Region[index] > 'Z' {
			return false
		}
	}
	seen := map[string]bool{}
	for _, language := range k.Languages {
		if !languages[language] || seen[language] {
			return false
		}
		seen[language] = true
	}
	return !(k.Statutory && k.Class == ClassInternal)
}

func sameKind(a, b Kind) bool {
	if a.ID != b.ID || a.Class != b.Class || a.Region != b.Region || a.Tax != b.Tax || a.Statutory != b.Statutory || a.Credit != b.Credit || a.Sequential != b.Sequential || a.Zone != b.Zone || len(a.Languages) != len(b.Languages) {
		return false
	}
	for index := range a.Languages {
		if a.Languages[index] != b.Languages[index] {
			return false
		}
	}
	return true
}

func copyKind(k Kind) Kind {
	k.Languages = append([]string(nil), k.Languages...)
	return k
}

func viewKind(k *kind) KindView {
	v := k.view
	v.Kind = copyKind(v.Kind)
	return v
}

func (r *Registry) Define(workspace string, k Kind, actor Actor) (KindView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !r.kindForm(k) {
		return KindView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return KindView{}, err
	}
	k = copyKind(k)
	sort.Strings(k.Languages)
	if s := r.space(workspace); s != nil {
		if found := s.kinds[k.ID]; found != nil {
			if actor.is(found.view.Definer) && sameKind(found.view.Kind, k) {
				return viewKind(found), nil
			}
			return KindView{}, ErrConflict
		}
	}
	s, err := r.open(workspace)
	if err != nil {
		return KindView{}, err
	}
	if len(s.kinds) >= r.limits.Kinds {
		return KindView{}, ErrLimit
	}
	s.kinds[k.ID] = &kind{view: KindView{Kind: k, State: StateProposed, Definer: actor.ID, At: now}}
	return viewKind(s.kinds[k.ID]), nil
}

func (r *Registry) lookupKind(workspace, id string) (*space, *kind, error) {
	s := r.space(workspace)
	if s == nil || s.kinds[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.kinds[id], nil
}

func (r *Registry) ApproveKind(workspace, id string, actor Actor) (KindView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return KindView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return KindView{}, err
	}
	_, k, err := r.lookupKind(workspace, id)
	if err != nil {
		return KindView{}, err
	}
	if k.view.State == StateActive {
		if actor.is(k.view.Approver) {
			return viewKind(k), nil
		}
		return KindView{}, ErrConflict
	}
	if actor.ID == k.view.Definer {
		return KindView{}, refuse(ReasonSelfApproval)
	}
	k.view.State, k.view.Approver, k.view.At = StateActive, actor.ID, now
	return viewKind(k), nil
}

func (r *Registry) VerifyTax(workspace, id, evidence string, actor Actor) (KindView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(evidence) || !actor.valid() {
		return KindView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return KindView{}, err
	}
	_, k, err := r.lookupKind(workspace, id)
	if err != nil {
		return KindView{}, err
	}
	if k.view.TaxReviewer != "" {
		if actor.is(k.view.TaxReviewer) && evidence == k.view.TaxEvidence {
			return viewKind(k), nil
		}
		return KindView{}, ErrConflict
	}
	switch {
	case !k.view.Kind.Statutory:
		return KindView{}, refuse(ReasonNotStatutory)
	case k.view.State != StateActive:
		return KindView{}, refuse(ReasonKindPending)
	case actor.ID == k.view.Definer:
		return KindView{}, refuse(ReasonSelfReview)
	}
	k.view.TaxReviewer, k.view.TaxEvidence, k.view.At = actor.ID, evidence, now
	return viewKind(k), nil
}

func (r *Registry) KindOf(workspace, id string) (KindView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return KindView{}, ErrInvalid
	}
	_, k, err := r.lookupKind(workspace, id)
	if err != nil {
		return KindView{}, err
	}
	return viewKind(k), nil
}

func seriesForm(v Series) bool {
	if !validID(v.Prefix) {
		return false
	}
	last := v.Prefix[len(v.Prefix)-1]
	return validID(v.ID) && validID(v.Kind) && (last < '0' || last > '9') && v.Width >= 1 && v.Width <= maxWidth && v.First >= 1 && v.First <= ceiling(v.Width)
}

func ceiling(width int) int64 {
	limit := int64(1)
	for index := 0; index < width; index++ {
		limit *= 10
	}
	return limit - 1
}

func format(v Series, value int64) string {
	text := strconv.FormatInt(value, 10)
	return v.Prefix + strings.Repeat("0", v.Width-len(text)) + text
}

func (r *Registry) ProposeSeries(workspace string, v Series, actor Actor) (SeriesView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !seriesForm(v) {
		return SeriesView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return SeriesView{}, err
	}
	s, k, err := r.lookupKind(workspace, v.Kind)
	if err != nil {
		return SeriesView{}, err
	}
	if found := s.series[v.ID]; found != nil {
		if actor.is(found.view.Proposer) && found.view.Series == v {
			return found.view, nil
		}
		return SeriesView{}, ErrConflict
	}
	switch {
	case k.view.Kind.Class == ClassERP:
		return SeriesView{}, refuse(ReasonERPNumbered)
	case k.view.State != StateActive:
		return SeriesView{}, refuse(ReasonKindPending)
	case s.byKind[v.Kind] != nil:
		return SeriesView{}, refuse(ReasonSeriesExists)
	case s.prefixes[v.Prefix]:
		return SeriesView{}, refuse(ReasonPrefixInUse)
	}
	created := &series{view: SeriesView{Series: v, State: StateProposed, Proposer: actor.ID, Next: v.First, At: now}}
	s.series[v.ID], s.byKind[v.Kind], s.prefixes[v.Prefix] = created, created, true
	return created.view, nil
}

func (r *Registry) ApproveSeries(workspace, id string, actor Actor) (SeriesView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return SeriesView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return SeriesView{}, err
	}
	s := r.space(workspace)
	if s == nil || s.series[id] == nil {
		return SeriesView{}, ErrNotFound
	}
	v := s.series[id]
	if v.view.State == StateActive {
		if actor.is(v.view.Approver) {
			return v.view, nil
		}
		return SeriesView{}, ErrConflict
	}
	if actor.ID == v.view.Proposer {
		return SeriesView{}, refuse(ReasonSelfApproval)
	}
	v.view.State, v.view.Approver, v.view.At = StateActive, actor.ID, now
	return v.view, nil
}

func (r *Registry) Numbers(workspace, id string) ([]NumberView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return nil, ErrInvalid
	}
	s := r.space(workspace)
	if s == nil || s.series[id] == nil {
		return nil, ErrNotFound
	}
	out := make([]NumberView, 0, len(s.series[id].numbers))
	for _, n := range s.series[id].numbers {
		out = append(out, *n)
	}
	return out, nil
}

func (r *Registry) take(s *space, v *series, draft string, actor Actor, now time.Time) (*NumberView, error) {
	if v.view.Next > ceiling(v.view.Series.Width) || s.numbers >= r.limits.Numbers {
		return nil, ErrLimit
	}
	s.numbers++
	n := &NumberView{Number: format(v.view.Series, v.view.Next), Value: v.view.Next, Draft: draft, State: NumberReserved, Actor: actor.ID, At: now, Taker: actor.ID, Taken: now}
	v.view.Next++
	v.numbers = append(v.numbers, n)
	return n, nil
}
