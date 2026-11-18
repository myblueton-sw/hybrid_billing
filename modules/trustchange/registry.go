package trustchange

import (
	"encoding/hex"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Registry struct {
	mu      sync.Mutex
	clock   func() time.Time
	run     string
	counter uint64
	changes map[string]*View
	order   []string
}

func Draw(random io.Reader, taken map[string]bool) (string, error) {
	for {
		var raw [RunBytes]byte
		if _, err := io.ReadFull(random, raw[:]); err != nil {
			return "", err
		}
		if run := hex.EncodeToString(raw[:]); !taken[run] {
			return run, nil
		}
	}
}

func RunOf(id string) (string, bool) {
	rest, ok := strings.CutPrefix(id, "trust-")
	run, number, cut := strings.Cut(rest, "-")
	if !ok || !cut || len(run) != 2*RunBytes || strings.ToLower(run) != run || strings.Trim(number, "0123456789") != "" || number == "" || number[0] == '0' {
		return "", false
	}
	if _, err := hex.DecodeString(run); err != nil {
		return "", false
	}
	return run, true
}

func New(run string, clock func() time.Time) (*Registry, error) {
	if _, ok := RunOf("trust-" + run + "-1"); !ok || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{clock: clock, run: run, changes: map[string]*View{}}, nil
}

func (r *Registry) Run() string { return r.run }

func refuse(reason string) error { return Refusal{Reason: reason} }

func copyView(v View) View {
	v.Candidate = copyCandidate(v.Candidate)
	if v.Validation != nil {
		x := *v.Validation
		x.Items = append([]Item(nil), x.Items...)
		v.Validation = &x
	}
	if v.Activation != nil {
		a := *v.Activation
		v.Activation = &a
	}
	v.History = append([]Entry(nil), v.History...)
	return v
}

func copyCandidate(c Candidate) Candidate {
	c.Methods = append([]string(nil), c.Methods...)
	c.StepUp = append([]Rule(nil), c.StepUp...)
	return c
}

func validCandidate(c Candidate) bool {
	if len(c.Methods) == 0 || c.MFA == "" || c.MaxAge <= 0 {
		return false
	}
	for i, m := range c.Methods {
		if m == "" || i > 0 && c.Methods[i-1] >= m {
			return false
		}
	}
	for _, rule := range c.StepUp {
		if rule.Action == "" || rule.MaxAge <= 0 {
			return false
		}
	}
	return true
}

func (r *Registry) record(v *View, state, by, rationale string, at time.Time) {
	v.State = state
	v.History = append(v.History, Entry{State: state, By: by, At: at, Rationale: rationale})
	if len(v.History) > MaxHistory {
		v.History = append([]Entry(nil), v.History[len(v.History)-MaxHistory:]...)
	}
}

func (r *Registry) open(workspace, proposer string) (int, int) {
	mine, all := 0, 0
	for _, v := range r.changes {
		if v.Workspace == workspace && Open(v.State) {
			all++
			if v.Proposer == proposer {
				mine++
			}
		}
	}
	return mine, all
}

func (r *Registry) prune() {
	for len(r.order) > MaxKept {
		dropped := false
		for i, id := range r.order {
			if !Open(r.changes[id].State) {
				delete(r.changes, id)
				r.order = append(r.order[:i:i], r.order[i+1:]...)
				dropped = true
				break
			}
		}
		if !dropped {
			return
		}
	}
}

func current(v *View, base Base) bool {
	return v.OldEpoch == base.Epoch && v.OldPolicy == base.Policy && v.OldDigest == base.Digest
}

func (r *Registry) Propose(p Proposal) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !validCandidate(p.Candidate) || p.Workspace == "" || p.Route == "" || p.Proposer == "" {
		return View{}, refuse(ReasonInvalid)
	}
	digest := ConfigDigest(p.Candidate)
	if digest == p.Base.Digest {
		return View{}, refuse(ReasonUnchanged)
	}
	for _, v := range r.changes {
		if v.Workspace == p.Workspace && Open(v.State) && v.Proposer == p.Proposer && v.NewDigest == digest && v.OldEpoch == p.Base.Epoch {
			return View{}, refuse(ReasonDuplicate)
		}
	}
	if mine, all := r.open(p.Workspace, p.Proposer); mine >= MaxOpenPerProposer || all >= MaxOpenPerWorkspace {
		return View{}, refuse(ReasonLimit)
	}
	at := r.clock()
	r.counter++
	v := &View{ID: "trust-" + r.run + "-" + strconv.FormatUint(r.counter, 10), Number: r.counter, Run: r.run, Workspace: p.Workspace, Route: p.Route, Revision: 1,
		OldEpoch: p.Base.Epoch, OldPolicy: p.Base.Policy, OldDigest: p.Base.Digest, NewDigest: digest, Candidate: copyCandidate(p.Candidate), Proposer: p.Proposer}
	v.ChangeDigest = changeDigest(*v)
	r.record(v, StateDraft, p.Proposer, p.Rationale, at)
	r.changes[v.ID] = v
	r.order = append(r.order, v.ID)
	r.prune()
	return copyView(*v), nil
}

func (r *Registry) find(id, workspace string) (*View, error) {
	v := r.changes[id]
	if v == nil || v.Workspace != workspace {
		return nil, refuse(ReasonNotFound)
	}
	return v, nil
}

func clearDecisions(v *View) {
	v.Validator, v.Validation, v.Approver, v.ApprovedAt = "", nil, "", time.Time{}
}

func (r *Registry) Revise(id, workspace, proposer string, revision uint64, c Candidate, base Base, rationale string) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, err := r.find(id, workspace)
	switch {
	case err != nil:
		return View{}, err
	case v.Proposer != proposer:
		return View{}, refuse(ReasonNotProposer)
	case !Open(v.State):
		return View{}, refuse(ReasonStage)
	case v.Revision != revision:
		return View{}, refuse(ReasonApprovalStale)
	case !current(v, base):
		return View{}, refuse(ReasonEpoch)
	case !validCandidate(c):
		return View{}, refuse(ReasonInvalid)
	}
	digest := ConfigDigest(c)
	switch digest {
	case base.Digest:
		return View{}, refuse(ReasonUnchanged)
	case v.NewDigest:
		return View{}, refuse(ReasonDuplicate)
	}
	v.Candidate, v.NewDigest = copyCandidate(c), digest
	v.Revision++
	v.ChangeDigest = changeDigest(*v)
	clearDecisions(v)
	r.record(v, StateDraft, proposer, rationale, r.clock())
	return copyView(*v), nil
}

func (r *Registry) Validate(id, workspace, validator string, revision uint64, digest string, base Base, ev Evidence, rationale string) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, err := r.find(id, workspace)
	switch {
	case err != nil:
		return View{}, err
	case !Open(v.State):
		return View{}, refuse(ReasonStage)
	case v.Revision != revision || v.ChangeDigest != digest:
		return View{}, refuse(ReasonApprovalStale)
	case !current(v, base):
		return View{}, refuse(ReasonEpoch)
	case ev.Digest == "":
		return View{}, refuse(ReasonInvalid)
	}
	at := r.clock()
	if v.Approver != "" && !ev.Blocked && v.Validation.Digest == ev.Digest {
		v.Validator = validator
		v.Validation = &Validation{Result: ResultValidated, At: at, Digest: ev.Digest, Items: append([]Item(nil), ev.Items...)}
		r.record(v, HistoryKept, validator, rationale, at)
		v.State = StateApproved
		return copyView(*v), nil
	}
	if v.Approver != "" {
		r.record(v, v.State, validator, HistoryInvalidated, at)
	}
	clearDecisions(v)
	v.Validator = validator
	v.Validation = &Validation{Result: ResultValidated, At: at, Digest: ev.Digest, Items: append([]Item(nil), ev.Items...)}
	state := StateValidated
	if ev.Blocked {
		v.Validation.Result, state = ResultBlocked, StateDraft
	}
	r.record(v, state, validator, rationale, at)
	return copyView(*v), nil
}

func (r *Registry) checkBound(v *View, a Approval) error {
	switch {
	case !current(v, a.Base):
		return refuse(ReasonEpoch)
	case v.ChangeDigest != a.ChangeDigest:
		return refuse(ReasonApprovalStale)
	case v.Validation.Digest != a.EvidenceDigest || a.Recomputed != v.Validation.Digest:
		return refuse(ReasonEvidenceStale)
	}
	return nil
}

func (r *Registry) Approve(a Approval) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, err := r.find(a.ID, a.Workspace)
	switch {
	case err != nil:
		return View{}, err
	case v.Proposer == a.Actor:
		return View{}, refuse(ReasonSelf)
	case v.State != StateValidated:
		return View{}, refuse(ReasonStage)
	}
	if err := r.checkBound(v, a); err != nil {
		return View{}, err
	}
	at := r.clock()
	v.Approver, v.ApprovedAt = a.Actor, at
	r.record(v, StateApproved, a.Actor, a.Rationale, at)
	return copyView(*v), nil
}

func (r *Registry) Activate(a Approval, commit func(View) Outcome) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, err := r.find(a.ID, a.Workspace)
	switch {
	case err != nil:
		return View{}, err
	case v.Proposer == a.Actor || v.Approver == a.Actor:
		return View{}, refuse(ReasonSelf)
	case v.State == StateDraft || v.State == StateValidated:
		return View{}, refuse(ReasonNeedsApproval)
	case v.State != StateApproved:
		return View{}, refuse(ReasonStage)
	}
	if err := r.checkBound(v, a); err != nil {
		return View{}, err
	}
	intent := copyView(*v)
	intent.Activator = a.Actor
	out := commit(intent)
	if out.Err != nil {
		return View{}, out.Err
	}
	at := r.clock()
	v.Activator = a.Actor
	if out.Unknown != "" {
		v.Unknown = out.Unknown
		r.record(v, StateUnknown, a.Actor, a.Rationale, at)
		return copyView(*v), nil
	}
	activation := out.Activation
	v.Activation = &activation
	r.record(v, StateActivated, a.Actor, a.Rationale, at)
	for _, id := range r.order {
		other := r.changes[id]
		if other != v && other.Workspace == v.Workspace && Open(other.State) && other.OldEpoch == v.OldEpoch {
			other.SupersededBy, other.SupersededEpoch = v.ID, activation.Epoch
			r.record(other, StateSuperseded, a.Actor, "", at)
		}
	}
	return copyView(*v), nil
}

func (r *Registry) end(id, workspace, actor string, revision uint64, rationale, state string) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, err := r.find(id, workspace)
	switch {
	case err != nil:
		return View{}, err
	case state == StateCancelled && v.Proposer != actor:
		return View{}, refuse(ReasonNotProposer)
	case state == StateRejected && v.Proposer == actor:
		return View{}, refuse(ReasonSelf)
	case !Open(v.State):
		return View{}, refuse(ReasonStage)
	case v.Revision != revision:
		return View{}, refuse(ReasonApprovalStale)
	}
	r.record(v, state, actor, rationale, r.clock())
	return copyView(*v), nil
}

func (r *Registry) Reject(id, workspace, actor string, revision uint64, rationale string) (View, error) {
	return r.end(id, workspace, actor, revision, rationale, StateRejected)
}

func (r *Registry) Cancel(id, workspace, actor string, revision uint64, rationale string) (View, error) {
	return r.end(id, workspace, actor, revision, rationale, StateCancelled)
}

func (r *Registry) Get(id, workspace string) (View, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, err := r.find(id, workspace)
	if err != nil {
		return View{}, false
	}
	return copyView(*v), true
}

func (r *Registry) List(workspace string) []View {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []View{}
	for i := len(r.order) - 1; i >= 0; i-- {
		if v := r.changes[r.order[i]]; v.Workspace == workspace {
			out = append(out, copyView(*v))
		}
	}
	return out
}

func (r *Registry) Restore(views []View) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, v := range views {
		run, ok := RunOf(v.ID)
		if !ok || run == r.run || r.changes[v.ID] != nil || v.State != StateActivated {
			return ErrInvalid
		}
		x := copyView(v)
		x.Run, x.EarlierRun = run, true
		r.changes[x.ID] = &x
		r.order = append(r.order, x.ID)
	}
	r.prune()
	return nil
}
