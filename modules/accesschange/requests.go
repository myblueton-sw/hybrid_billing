package accesschange

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type request struct {
	view View
	key  string
	code string
}

type Registry struct {
	mu             sync.Mutex
	clock          func() time.Time
	limits         Limits
	requests       map[uint64]*request
	nextRequest    uint64
	revocations    []*revocation
	nextRevocation uint64
	last           Receipt
	version        uint64
}

func NewRegistry(clock func() time.Time, limits Limits) (*Registry, error) {
	if clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{clock: clock, limits: limits, requests: map[uint64]*request{}}, nil
}

func (r *Registry) now() time.Time { return r.clock().UTC() }

func (r *Registry) changed() {
	if r.version < math.MaxUint64 {
		r.version++
	}
}

func digestOf(parts ...string) string {
	sum := sha256.New()
	for _, part := range parts {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(part)))
		sum.Write(size[:])
		sum.Write([]byte(part))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func (c Change) code() string {
	parts := []string{c.Kind, c.Subject, c.Action, c.ResourceKind, strconv.Itoa(len(c.Fields))}
	parts = append(parts, c.Fields...)
	return digestOf(append(parts, c.Customer, strconv.Itoa(c.Days), c.Grant)...)
}

func (c Change) target() string {
	if c.Kind == ChangeAdd {
		return digestOf(ChangeAdd, c.Subject, c.Action, c.ResourceKind)
	}
	return digestOf(ChangeRevoke, c.Grant)
}

func (c Change) valid() bool {
	switch c.Kind {
	case ChangeAdd:
		return c.Subject != "" && c.Action != "" && c.ResourceKind != "" && c.Customer != "" && c.Days >= 1 && c.Grant == ""
	case ChangeRevoke:
		return c.Grant != "" && c.Subject == "" && c.Action == "" && c.ResourceKind == "" && len(c.Fields) == 0 && c.Customer == "" && c.Days == 0
	}
	return false
}

func copyView(v View) View {
	v.Change.Fields = append([]string(nil), v.Change.Fields...)
	return v
}

func (r *Registry) lapsed(q *request, now time.Time) bool {
	return q.view.State == StatePending && !now.Before(q.view.ExpiresAt)
}

func (r *Registry) shown(q *request, now time.Time) View {
	v := copyView(q.view)
	if r.lapsed(q, now) {
		v.State = StateExpired
	}
	return v
}

func (r *Registry) end(q *request, state string, by, reason string, now time.Time) {
	q.view.State, q.view.DecidedBy, q.view.DecidedAt, q.view.DecisionReason = state, by, now, reason
}

func (r *Registry) Propose(p Proposal) (View, bool, error) {
	r.mu.Lock()
	r.changed()
	defer r.mu.Unlock()
	now := r.now()
	switch {
	case !p.Proposer.Human:
		return View{}, false, Refusal{ReasonNotHuman}
	case p.Change.Kind == ChangeAdd && p.Proposer.User == p.Change.Subject:
		return View{}, false, Refusal{ReasonProposerSubject}
	case p.Change.Kind == ChangeRevoke && p.Proposer.User == p.Holder:
		return View{}, false, Refusal{ReasonProposerHolder}
	case !p.Allowed:
		return View{}, false, Refusal{ReasonNotAllowed}
	case !p.Change.valid() || p.Workspace == "" || p.Proposer.User == "" || (p.Change.Kind == ChangeRevoke && p.Holder == ""):
		return View{}, false, ErrInvalid
	}
	code, key := p.Change.code(), p.Change.target()
	mine, workspace := 0, 0
	for _, number := range r.numbers() {
		q := r.requests[number]
		if q.view.Workspace != p.Workspace || q.view.State != StatePending || r.lapsed(q, now) {
			continue
		}
		if q.key == key {
			if q.code == code && q.view.Proposer == p.Proposer.User {
				return copyView(q.view), true, nil
			}
			return View{}, false, Refusal{ReasonPendingTarget}
		}
		workspace++
		if q.view.Proposer == p.Proposer.User {
			mine++
		}
	}
	if mine >= r.limits.PerProposer || workspace >= r.limits.PerWorkspace {
		return View{}, false, Refusal{ReasonLimit}
	}
	expires := now.Add(r.limits.Lifetime)
	if p.Change.Kind == ChangeAdd {
		expires = minTime(expires, p.CeilingUntil.UTC().Add(-time.Duration(p.Change.Days)*24*time.Hour))
	}
	if !expires.After(now) {
		return View{}, false, Refusal{ReasonExpiry}
	}
	r.nextRequest++
	number := r.nextRequest
	change := p.Change
	change.Fields = append([]string(nil), change.Fields...)
	v := View{Number: number, ID: RequestPrefix + strconv.FormatUint(number, 10), Workspace: p.Workspace, Change: change, Holder: p.Holder, Basis: p.Basis,
		Digest: digestOf(digestLabel, p.Workspace, code, p.Holder, p.Basis), State: StatePending, Proposer: p.Proposer.User, ProposedAt: now, ExpiresAt: expires,
		Rationale: p.Rationale}
	r.requests[number] = &request{view: v, key: key, code: code}
	return copyView(v), false, nil
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func (r *Registry) numbers() []uint64 {
	out := make([]uint64, 0, len(r.requests))
	for n := range r.requests {
		out = append(out, n)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}

func (r *Registry) find(number uint64, workspace string) (*request, error) {
	q, ok := r.requests[number]
	if !ok || q.view.Workspace != workspace {
		return nil, Refusal{ReasonNotFound}
	}
	return q, nil
}

func (r *Registry) live(q *request, now time.Time) error {
	if q.view.State != StatePending {
		return Refusal{ReasonNotPending}
	}
	if r.lapsed(q, now) {
		r.end(q, StateExpired, "", "", q.view.ExpiresAt)
		r.prune()
		return Refusal{ReasonExpired}
	}
	return nil
}

func (r *Registry) Approve(number uint64, workspace string, approver Actor, digest string, current Standing, apply func(Applier) (string, error)) (View, error) {
	r.mu.Lock()
	r.changed()
	defer r.mu.Unlock()
	now := r.now()
	q, err := r.find(number, workspace)
	if err != nil {
		return View{}, err
	}
	switch {
	case !approver.Human:
		return View{}, Refusal{ReasonNotHuman}
	case approver.User == q.view.Proposer:
		return View{}, Refusal{ReasonSelfDecision}
	case q.view.Change.Kind == ChangeAdd && approver.User == q.view.Change.Subject:
		return View{}, Refusal{ReasonApproverSubject}
	case !current.Allowed:
		return View{}, Refusal{ReasonNotAllowed}
	}
	if err := r.live(q, now); err != nil {
		return View{}, err
	}
	if digest != q.view.Digest {
		return View{}, Refusal{ReasonDigest}
	}
	if current.Basis != q.view.Basis {
		r.end(q, StateSuperseded, "", "", now)
		r.prune()
		return View{}, Refusal{ReasonSuperseded}
	}
	result, err := apply(inner{r})
	if err != nil {
		return View{}, err
	}
	r.end(q, StateApplied, approver.User, "", now)
	q.view.Result = result
	v := copyView(q.view)
	r.prune()
	return v, nil
}

func (r *Registry) Reject(number uint64, workspace string, rejecter Actor, allowed bool, rationale string) (View, error) {
	r.mu.Lock()
	r.changed()
	defer r.mu.Unlock()
	now := r.now()
	q, err := r.find(number, workspace)
	if err != nil {
		return View{}, err
	}
	switch {
	case !rejecter.Human:
		return View{}, Refusal{ReasonNotHuman}
	case rejecter.User == q.view.Proposer:
		return View{}, Refusal{ReasonSelfDecision}
	case q.view.Change.Kind == ChangeRevoke && rejecter.User == q.view.Holder:
		return View{}, Refusal{ReasonHolderReject}
	case !allowed:
		return View{}, Refusal{ReasonNotAllowed}
	}
	if err := r.live(q, now); err != nil {
		return View{}, err
	}
	r.end(q, StateRejected, rejecter.User, rationale, now)
	v := copyView(q.view)
	r.prune()
	return v, nil
}

func (r *Registry) Withdraw(number uint64, workspace string, actor Actor) (View, error) {
	r.mu.Lock()
	r.changed()
	defer r.mu.Unlock()
	now := r.now()
	q, err := r.find(number, workspace)
	if err != nil {
		return View{}, err
	}
	switch {
	case !actor.Human:
		return View{}, Refusal{ReasonNotHuman}
	case actor.User != q.view.Proposer:
		return View{}, Refusal{ReasonNotProposer}
	}
	if err := r.live(q, now); err != nil {
		return View{}, err
	}
	r.end(q, StateWithdrawn, actor.User, "", now)
	v := copyView(q.view)
	r.prune()
	return v, nil
}

func (r *Registry) prune() {
	ended := []*request{}
	for _, n := range r.numbers() {
		if q := r.requests[n]; q.view.State != StatePending {
			ended = append(ended, q)
		}
	}
	for len(ended) > r.limits.EndedKept {
		delete(r.requests, ended[0].view.Number)
		ended = ended[1:]
	}
}

func (r *Registry) Request(number uint64, workspace string) (View, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	q, err := r.find(number, workspace)
	if err != nil {
		return View{}, false
	}
	return r.shown(q, r.now()), true
}

func (r *Registry) Requests(workspace string) []View {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	out := []View{}
	for _, n := range r.numbers() {
		if q := r.requests[n]; q.view.Workspace == workspace {
			out = append(out, r.shown(q, now))
		}
	}
	return out
}

func Number(id, prefix string) (uint64, bool) {
	raw, ok := strings.CutPrefix(id, prefix)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	return n, err == nil && n > 0 && strconv.FormatUint(n, 10) == raw
}

type inner struct{ r *Registry }

func (i inner) Revoke(in Revocation, e Effects) (RevocationView, error) { return i.r.revoke(in, e) }
