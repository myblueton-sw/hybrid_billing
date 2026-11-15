package corrections

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

type correction struct {
	view    View
	target  *big.Int
	diff    *big.Int
	of      *original
	credits []string
}

type original struct {
	id      string
	snap    issuance.Snapshot
	total   *big.Int
	applied []*correction
	linked  map[string]bool
	notes   map[string]*Note
	open    *correction
	finals  []Record
}

type live struct {
	credits   []string
	effective *big.Int
	unlinked  []string
	snapshots map[string]issuance.Snapshot
}

type space struct {
	corrections map[string]*correction
	originals   map[string]*original
	keys        map[[2]string]*correction
	documents   map[string]bool
	records     int
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
	if config.Documents == nil || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{config: config, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (l Limits) valid() bool {
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Corrections, maxCorrections}, {l.Records, maxRecords}}
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
		return refuse(RefusalAINotAllowed)
	case KindSystem:
		return refuse(RefusalNotHuman)
	}
	return nil
}

func (a Actor) notAssistant() error {
	if a.Kind == KindAssistant {
		return refuse(RefusalAINotAllowed)
	}
	return nil
}

func canonical(text string) bool {
	whole, fraction, dotted := strings.Cut(text, ".")
	if whole == "" || len(whole)+len(fraction) > maxDigits || dotted && (fraction == "" || fraction[len(fraction)-1] == '0') || len(whole) > 1 && whole[0] == '0' {
		return false
	}
	for _, part := range []string{whole, fraction} {
		for index := 0; index < len(part); index++ {
			if part[index] < '0' || part[index] > '9' {
				return false
			}
		}
	}
	return true
}

func scaled(text string, scale int) (*big.Int, bool) {
	whole, fraction, _ := strings.Cut(text, ".")
	if len(fraction) > scale {
		return nil, false
	}
	value, _ := new(big.Int).SetString(whole+fraction+strings.Repeat("0", scale-len(fraction)), 10)
	return value, true
}

func format(value *big.Int, scale int) string {
	sign := ""
	digits := new(big.Int).Abs(value).String()
	if value.Sign() < 0 {
		sign = "-"
	}
	for len(digits) <= scale {
		digits = "0" + digits
	}
	whole, fraction := digits[:len(digits)-scale], strings.TrimRight(digits[len(digits)-scale:], "0")
	if fraction == "" {
		return sign + whole
	}
	return sign + whole + "." + fraction
}

func (r *Registry) document(workspace, id string) (issuance.DocumentView, error) {
	v, err := r.config.Documents.DocumentOf(workspace, id)
	switch {
	case errors.Is(err, issuance.ErrNotFound):
		return issuance.DocumentView{}, ErrNotFound
	case err != nil || v.Snapshot.Workspace != workspace:
		return issuance.DocumentView{}, refuse(RefusalUnavailable)
	}
	return v, nil
}

func (r *Registry) live(workspace string, o *original) (live, error) {
	credits, err := r.config.Documents.CreditsOf(workspace, o.id)
	if err != nil {
		return live{}, refuse(RefusalUnavailable)
	}
	out := live{credits: credits, effective: new(big.Int).Set(o.total), snapshots: map[string]issuance.Snapshot{}}
	for _, id := range credits {
		d, err := r.document(workspace, id)
		if err != nil {
			return live{}, refuse(RefusalUnavailable)
		}
		amount, ok := scaled(d.Snapshot.Draft.Total, o.snap.Draft.Scale)
		if !ok || !canonical(d.Snapshot.Draft.Total) || d.Snapshot.Draft.Original != o.id {
			return live{}, refuse(RefusalUnavailable)
		}
		out.effective.Sub(out.effective, amount)
		out.snapshots[id] = d.Snapshot
		if !o.linked[id] {
			out.unlinked = append(out.unlinked, id)
		}
	}
	return out, nil
}

func creditDigest(credits []string) string {
	var b strings.Builder
	for _, id := range credits {
		b.WriteString(strconv.Itoa(len(id)))
		b.WriteByte(':')
		b.WriteString(id)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func (r *Registry) base(o *original, l live) Base {
	return Base{Effective: format(l.effective, o.snap.Draft.Scale), Credits: len(l.credits), Digest: creditDigest(l.credits)}
}

func (r *Registry) originalOf(workspace string, s *space, id string) (*original, error) {
	if s != nil && s.originals[id] != nil {
		return s.originals[id], nil
	}
	v, err := r.document(workspace, id)
	if err != nil {
		return nil, err
	}
	snap := v.Snapshot
	if snap.Draft.Original != "" {
		return nil, refuse(RefusalNotOriginal)
	}
	total, ok := scaled(snap.Draft.Total, snap.Draft.Scale)
	if !ok || !canonical(snap.Draft.Total) {
		return nil, refuse(RefusalUnavailable)
	}
	return &original{id: id, snap: snap, total: total, linked: map[string]bool{}, notes: map[string]*Note{}}, nil
}

func (r *Registry) open(workspace string) (*space, error) {
	if s := r.workspaces[workspace]; s != nil {
		return s, nil
	}
	if len(r.workspaces) >= r.limits.Workspaces {
		return nil, ErrLimit
	}
	s := &space{corrections: map[string]*correction{}, originals: map[string]*original{}, keys: map[[2]string]*correction{}, documents: map[string]bool{}}
	r.workspaces[workspace] = s
	return s, nil
}

var reasons = map[string]bool{ReasonLateSource: true, ReasonRetroactive: true, ReasonQuantityError: true, ReasonPriceError: true, ReasonOther: true}

func validDate(text string) bool {
	parsed, err := time.Parse(time.DateOnly, text)
	return err == nil && parsed.Format(time.DateOnly) == text
}

func form(c Correction) bool {
	return validID(c.ID) && validID(c.Original) && validID(c.Key) && canonical(c.Target) && reasons[c.Reason] && validID(c.Evidence) &&
		validDate(c.Period.From) && validDate(c.Period.To) && c.Period.From <= c.Period.To
}

func digestOf(workspace string, v View) string {
	var b strings.Builder
	for _, part := range []string{requestLabel, workspace, v.Correction.ID, v.Correction.Original, v.Correction.Key, v.Correction.Target, v.Correction.Reason,
		v.Correction.Evidence, v.Correction.Period.From, v.Correction.Period.To, v.Base.Effective, strconv.Itoa(v.Base.Credits), v.Base.Digest, v.Difference, v.Direction, v.Requester, v.Requested.Format(time.RFC3339Nano)} {
		b.WriteString(strconv.Itoa(len(part)))
		b.WriteByte(':')
		b.WriteString(part)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func (r *Registry) Request(workspace string, c Correction, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !form(c) {
		return View{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return View{}, err
	}
	s := r.workspaces[workspace]
	o, err := r.originalOf(workspace, s, c.Original)
	if err != nil {
		return View{}, err
	}
	if s != nil {
		if found := s.corrections[c.ID]; found != nil {
			if actor.is(found.view.Requester) && found.view.Correction == c {
				return found.view, nil
			}
			return View{}, ErrConflict
		}
		if found := s.keys[[2]string{c.Original, c.Key}]; found != nil {
			duplicate := found.view.Correction
			duplicate.ID = c.ID
			if duplicate == c {
				v := found.view
				v.Duplicate = true
				return v, nil
			}
			return View{}, refuse(RefusalKeyReused)
		}
	}
	target, ok := scaled(c.Target, o.snap.Draft.Scale)
	if !ok {
		return View{}, refuse(RefusalScale)
	}
	l, err := r.live(workspace, o)
	if err != nil {
		return View{}, err
	}
	switch {
	case len(l.unlinked) > 0:
		return View{}, refuse(RefusalUnlinked)
	case o.open != nil:
		return View{}, refuse(RefusalPending)
	case target.Cmp(l.effective) == 0:
		return View{}, refuse(RefusalNoChange)
	case target.Cmp(l.effective) > 0:
		return View{}, refuse(RefusalDebit)
	}
	if s, err = r.open(workspace); err != nil {
		return View{}, err
	}
	if len(s.corrections) >= r.limits.Corrections {
		return View{}, ErrLimit
	}
	diff := new(big.Int).Sub(target, l.effective)
	made := &correction{target: target, diff: diff, of: o, credits: l.credits, view: View{Correction: c, Base: r.base(o, l),
		Difference: format(diff, o.snap.Draft.Scale), Direction: DirectionCredit, State: StateRequested, Requester: actor.ID, Requested: now}}
	made.view.Digest = digestOf(workspace, made.view)
	s.originals[o.id] = o
	s.corrections[c.ID] = made
	s.keys[[2]string{c.Original, c.Key}] = made
	o.open = made
	return made.view, nil
}

func (r *Registry) lookup(workspace, id string) (*space, *correction, error) {
	s := r.workspaces[workspace]
	if s == nil || s.corrections[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.corrections[id], nil
}

func (r *Registry) Approve(workspace, id, digest string, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || len(digest) != 64 || !actor.valid() {
		return View{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return View{}, err
	}
	_, c, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	if c.view.State == StateApproved && c.view.Digest == digest {
		if actor.is(c.view.Approver) {
			return c.view, nil
		}
		return View{}, ErrConflict
	}
	switch {
	case c.view.State != StateRequested:
		return View{}, refuse(RefusalNotRequested)
	case c.view.Digest != digest:
		return View{}, refuse(RefusalDigestChanged)
	case actor.ID == c.view.Requester:
		return View{}, refuse(RefusalSelfApproval)
	}
	l, err := r.live(workspace, c.of)
	if err != nil {
		return View{}, err
	}
	if r.base(c.of, l) != c.view.Base {
		return View{}, refuse(RefusalBaseChanged)
	}
	c.view.State, c.view.Approver, c.view.Approved = StateApproved, actor.ID, now
	return c.view, nil
}

func (r *Registry) Withdraw(workspace, id string, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return View{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return View{}, err
	}
	s, c, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	switch c.view.State {
	case StateWithdrawn:
		if actor.is(c.view.Closer) {
			return c.view, nil
		}
		return View{}, ErrConflict
	case StateApplied:
		return View{}, refuse(RefusalClosed)
	}
	if c.view.State == StateApproved {
		l, err := r.live(workspace, c.of)
		if err != nil {
			return View{}, err
		}
		if len(l.unlinked) > 0 {
			return View{}, refuse(RefusalCreditIssued)
		}
	}
	c.view.State, c.view.Closer, c.view.CloserKind, c.view.Closed = StateWithdrawn, actor.ID, actor.Kind, now
	c.of.open = nil
	delete(s.keys, [2]string{c.view.Correction.Original, c.view.Correction.Key})
	return c.view, nil
}

func (r *Registry) Apply(workspace, id, documentID string, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(documentID) || !actor.valid() {
		return View{}, ErrInvalid
	}
	if err := actor.notAssistant(); err != nil {
		return View{}, err
	}
	s, c, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	if c.view.State == StateApplied {
		if c.view.Document == documentID && c.view.Closer == actor.ID && c.view.CloserKind == actor.Kind {
			return c.view, nil
		}
		return View{}, refuse(RefusalClosed)
	}
	if c.view.State != StateApproved {
		return View{}, refuse(RefusalNotApproved)
	}
	o := c.of
	if s.documents[documentID] {
		return View{}, refuse(RefusalLinked)
	}
	l, err := r.live(workspace, o)
	if err != nil {
		return View{}, err
	}
	if len(l.credits) != len(c.credits)+1 || l.credits[len(c.credits)] != documentID || creditDigest(l.credits[:len(c.credits)]) != c.view.Base.Digest {
		return View{}, refuse(RefusalBaseChanged)
	}
	if err := matches(o, c, l.snapshots[documentID]); err != nil {
		return View{}, err
	}
	c.view.State, c.view.Closer, c.view.CloserKind, c.view.Closed, c.view.Document = StateApplied, actor.ID, actor.Kind, now, documentID
	s.documents[documentID] = true
	o.linked[documentID] = true
	o.applied = append(o.applied, c)
	o.open = nil
	return c.view, nil
}

func matches(o *original, c *correction, d issuance.Snapshot) error {
	a := o.snap
	amount, _ := scaled(d.Draft.Total, a.Draft.Scale)
	switch {
	case d.Class != a.Class || d.Region != a.Region || d.Tax != a.Tax || d.Statutory != a.Statutory || d.External != a.External ||
		d.Draft.Contract != a.Draft.Contract || d.Draft.Customer != a.Draft.Customer || d.Draft.Entity != a.Draft.Entity ||
		d.Draft.Currency != a.Draft.Currency || d.Draft.Scale != a.Draft.Scale:
		return refuse(RefusalMismatch)
	case amount.Cmp(new(big.Int).Abs(c.diff)) != 0:
		return refuse(RefusalAmount)
	}
	return nil
}

func (r *Registry) Of(workspace, id string) (View, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return View{}, ErrInvalid
	}
	_, c, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	return c.view, nil
}
