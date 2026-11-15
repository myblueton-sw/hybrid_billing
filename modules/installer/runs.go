package installer

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	codePrefix = "hbi_"
	codeBytes  = 32
	codeTag    = "hybrid-billing/install-bootstrap/v1"
)

func active(state string) bool { return state == RunOpen || state == RunInterrupted }

func viewRun(run *run) RunView {
	v := run.view
	v.Steps = append([]string(nil), v.Steps...)
	return v
}

func steps(rel Release, plan layout) []string {
	out := []string{StepPreflight}
	for _, a := range rel.Artifacts {
		out = append(out, "artifact:"+a.Name)
	}
	for _, v := range rel.Volumes {
		out = append(out, "volume:"+plan.volumes[v.Name])
	}
	for _, s := range rel.Services {
		out = append(out, "service:"+s.Name)
	}
	return append(out, StepRegisterAdmin)
}

func (r *Registry) site(id string) (*site, error) {
	if !validID(id) {
		return nil, ErrInvalid
	}
	s := r.sites[id]
	if s == nil {
		return nil, ErrNotFound
	}
	return s, nil
}

func (r *Registry) StartRun(id string, number uint64, actor Actor) (RunView, error) {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return RunView{}, err
	}
	s, err := r.site(id)
	if err != nil {
		return RunView{}, err
	}
	p := r.proposals[number]
	switch {
	case p == nil:
		return RunView{}, ErrNotFound
	case p.scope != id || p.view.Change.Kind != ChangePlan || p.view.State != ProposalApplied:
		return RunView{}, refuse(ReasonNotApproved)
	case p.used:
		return RunView{}, refuse(ReasonAcceptanceUsed)
	case p.retired:
		return RunView{}, refuse(ReasonPlanChanged)
	}
	for _, other := range s.runs {
		if active(other.view.State) {
			return RunView{}, refuse(ReasonRunOpen)
		}
	}
	a := p.view.Change.Acceptance
	rel := r.releases[a.Version]
	view, err := r.preview(s, rel, now, nil)
	switch {
	case err != nil:
		return RunView{}, err
	case view.Hash != a.Hash:
		return RunView{}, refuse(ReasonPlanChanged)
	}
	p.used = true
	s.counter++
	made := &run{view: RunView{ID: "run-" + strconv.Itoa(s.counter), Site: id, Version: a.Version, Hash: a.Hash, Revision: s.revision,
		Probe: s.view.Probe, Admin: a.Admin, Steps: steps(rel.release, r.resolve(s, rel.release)), State: RunOpen}}
	s.runs[made.view.ID] = made
	s.order = append(s.order, made.view.ID)
	for len(s.order) > r.limits.Runs {
		delete(s.runs, s.order[0])
		s.order = s.order[1:]
	}
	return viewRun(made), nil
}

func (r *Registry) located(id, runID string) (*site, *run, error) {
	s, err := r.site(id)
	if err != nil {
		return nil, nil, err
	}
	if !validID(runID) {
		return nil, nil, ErrInvalid
	}
	found := s.runs[runID]
	if found == nil {
		return nil, nil, ErrNotFound
	}
	return s, found, nil
}

func (r *Registry) Run(id, runID string) (RunView, error) {
	r.enter()
	defer r.leave()
	_, found, err := r.located(id, runID)
	if err != nil {
		return RunView{}, err
	}
	return viewRun(found), nil
}

func (r *Registry) stepper(id, runID string, actor Actor) (*site, *run, error) {
	s, err := r.probe(id, actor)
	if err != nil {
		return nil, nil, err
	}
	_, found, err := r.located(id, runID)
	if err != nil {
		return nil, nil, err
	}
	switch {
	case !active(found.view.State):
		return nil, nil, refuse(ReasonRunClosed)
	case found.view.State != RunOpen:
		return nil, nil, refuse(ReasonNotAllowedState)
	}
	return s, found, nil
}

func (r *Registry) Complete(id, runID, step string, e Evidence, actor Actor) (RunView, error) {
	r.enter()
	defer r.leave()
	if step == "" || !validHex(e.Digest) {
		return RunView{}, ErrInvalid
	}
	s, found, err := r.stepper(id, runID, actor)
	if err != nil {
		return RunView{}, err
	}
	v := &found.view
	for index := 0; index < v.Done; index++ {
		if v.Steps[index] == step && found.evidence(index) == e.Digest {
			return viewRun(found), nil
		}
	}
	switch {
	case v.Steps[v.Done] != step:
		return RunView{}, refuse(ReasonOutOfOrder)
	case step == StepRegisterAdmin:
		return RunView{}, refuse(ReasonNotAllowedState)
	}
	if name, ok := strings.CutPrefix(step, "artifact:"); ok {
		for _, a := range r.releases[v.Version].release.Artifacts {
			if a.Name == name && a.Digest != e.Digest {
				v.State, v.Reason = RunFailed, ReasonDigestMismatch
				r.forget(s)
				return RunView{}, refuse(ReasonDigestMismatch)
			}
		}
	}
	found.record(e.Digest)
	v.Done++
	return viewRun(found), nil
}

func (r *Registry) Interrupt(id, runID, reason string, actor Actor) (RunView, error) {
	r.enter()
	defer r.leave()
	if !actor.valid() || !validText(reason) {
		return RunView{}, ErrInvalid
	}
	s, found, err := r.located(id, runID)
	if err != nil {
		return RunView{}, err
	}
	switch {
	case actor.Kind == KindAssistant:
		return RunView{}, refuse(ReasonAINotAllowed)
	case actor.Kind == KindSystem && actor.ID != s.view.Probe:
		return RunView{}, refuse(ReasonNotProbe)
	}
	switch found.view.State {
	case RunOpen:
		found.view.State, found.view.Reason, found.view.Note = RunInterrupted, ReasonInterrupted, reason
		found.paused = s.sequence
		r.forget(s)
	case RunInterrupted:
	default:
		return RunView{}, refuse(ReasonRunClosed)
	}
	return viewRun(found), nil
}

func (r *Registry) Resume(id, runID string, actor Actor) (RunView, error) {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return RunView{}, err
	}
	s, found, err := r.located(id, runID)
	if err != nil {
		return RunView{}, err
	}
	v := &found.view
	switch v.State {
	case RunInterrupted:
	case RunOpen:
		return RunView{}, refuse(ReasonNotAllowedState)
	default:
		return RunView{}, refuse(ReasonRunClosed)
	}
	rel := r.releases[v.Version]
	cause := ""
	switch {
	case r.revoked(rel):
		cause = ReasonReleaseRevoked
	case v.Revision != s.revision:
		cause = ReasonConfigChanged
	case v.Probe != s.view.Probe:
		cause = ReasonProbeChanged
	case s.sequence <= found.paused:
		return RunView{}, refuse(ReasonObservationStale)
	}
	if cause != "" {
		r.invalidate(s, found, cause)
		return RunView{}, refuse(cause)
	}
	plan := r.resolve(s, rel.release)
	allowed := map[string]bool{}
	for _, step := range v.Steps[:v.Done] {
		if name, ok := strings.CutPrefix(step, "service:"); ok {
			allowed["service:"+name] = true
			for _, key := range plan.ports[name] {
				allowed["port:"+key] = true
			}
		}
		if host, ok := strings.CutPrefix(step, "volume:"); ok {
			allowed["volume:"+host] = true
		}
	}
	view, err := r.preview(s, rel, now, allowed)
	if err != nil {
		return RunView{}, err
	}
	if len(view.Conflicts) > 0 {
		r.invalidate(s, found, ReasonHostChanged)
		return RunView{}, refuse(ReasonHostChanged)
	}
	v.State, v.Reason, v.Note = RunOpen, "", ""
	return viewRun(found), nil
}

func (r *Registry) Cancel(id, runID string, actor Actor) (RunView, error) {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return RunView{}, err
	}
	s, found, err := r.located(id, runID)
	if err != nil {
		return RunView{}, err
	}
	switch found.view.State {
	case RunOpen, RunInterrupted:
		found.view.State, found.view.Reason, found.view.Note = RunCancelled, "", ""
		r.forget(s)
	case RunCancelled:
	default:
		return RunView{}, refuse(ReasonRunClosed)
	}
	return viewRun(found), nil
}

func codeHash(text string) [sha256.Size]byte {
	var e encoder
	e.text(codeTag)
	e.text(text)
	return sha256.Sum256(e.out.Bytes())
}

func validCode(text string) bool {
	if len(text) != len(codePrefix)+base64.RawURLEncoding.EncodedLen(codeBytes) || text[:len(codePrefix)] != codePrefix {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(text[len(codePrefix):])
	return err == nil && base64.RawURLEncoding.EncodeToString(raw) == text[len(codePrefix):]
}

func (r *Registry) IssueBootstrap(id, runID string, ttl time.Duration, actor Actor) (string, error) {
	now := r.enter()
	defer r.leave()
	if ttl < minBootstrapLife || ttl > r.limits.BootstrapLife {
		return "", ErrInvalid
	}
	s, found, err := r.stepper(id, runID, actor)
	if err != nil {
		return "", err
	}
	if found.view.Steps[found.view.Done] != StepRegisterAdmin {
		return "", refuse(ReasonOutOfOrder)
	}
	raw := make([]byte, codeBytes)
	if _, err := io.ReadFull(r.config.Random, raw); err != nil {
		return "", refuse(ReasonDependency)
	}
	text := codePrefix + base64.RawURLEncoding.EncodeToString(raw)
	hash := codeHash(text)
	if r.codes[hash] != nil {
		return "", refuse(ReasonDependency)
	}
	r.forget(s)
	r.codes[hash] = &code{site: id, run: runID, expiry: now.Add(ttl)}
	s.code = &hash
	return text, nil
}

func (r *Registry) RegisterAdmin(text string, actor Actor) (RunView, error) {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return RunView{}, err
	}
	if !validCode(text) {
		return RunView{}, ErrInvalid
	}
	hash := codeHash(text)
	c := r.codes[hash]
	if c == nil {
		return RunView{}, refuse(ReasonUnknownCode)
	}
	s := r.sites[c.site]
	found := s.runs[c.run]
	switch {
	case !now.Before(c.expiry):
		r.forget(s)
		return RunView{}, refuse(ReasonExpired)
	case found == nil || found.view.State != RunOpen || found.view.Steps[found.view.Done] != StepRegisterAdmin || found.view.Probe != s.view.Probe ||
		found.view.Revision != s.revision:
		r.forget(s)
		return RunView{}, refuse(ReasonStaleCode)
	case actor.ID != found.view.Admin:
		return RunView{}, refuse(ReasonNotNominee)
	}
	r.forget(s)
	s.admin, s.installed = actor.ID, true
	found.record(digestOf([]byte(actor.ID)))
	found.view.Done++
	found.view.State = RunCompleted
	return viewRun(found), nil
}
