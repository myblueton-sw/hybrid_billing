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
	v.Scope = append([]string(nil), v.Scope...)
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
	case p.scope != id || !acceptance(p.view.Change.Kind) || p.view.State != ProposalApplied:
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
	a, kind := p.view.Change.Acceptance, p.view.Change.Kind
	if kind == ChangeRollback && s.rolledBack[a.Run] {
		return RunView{}, refuse(ReasonRolledBack)
	}
	rel := r.releases[a.Version]
	view, err := r.kindPreview(s, kind, a, now, nil)
	switch {
	case err != nil:
		return RunView{}, err
	case view.Hash != a.Hash:
		return RunView{}, refuse(ReasonPlanChanged)
	}
	p.used = true
	s.counter++
	made := &run{view: RunView{ID: "run-" + strconv.Itoa(s.counter), Site: id, Version: a.Version, Hash: a.Hash, Revision: s.revision,
		Probe: s.view.Probe, Admin: a.Admin, State: RunOpen}}
	switch kind {
	case ChangePlan:
		made.view.Kind, made.view.Steps = RunInstall, steps(rel.release, r.resolve(s, rel.release))
	case ChangeUpgrade:
		made.view.Kind, made.view.Steps = RunUpgrade, upgradeSteps(rel.release, view)
	default:
		made.view.Kind, made.view.Steps = RunRollback, rollbackSteps(rel.release, s.record)
	}
	r.layoutOf(s, rel.release, made)
	if kind == ChangeRollback {
		made.ports = map[string][]string{}
		for name, keys := range s.record.services {
			made.ports[name] = append([]string{}, keys...)
		}
	}
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
				r.ended(s, found)
				return RunView{}, refuse(ReasonDigestMismatch)
			}
		}
	}
	if step == StepBackup {
		v.Backup, v.Scope = e.Digest, []string{}
		scope := map[string]bool{}
		for host, vol := range s.record.volumes {
			scope[host] = scope[host] || vol.data
		}
		for host, vol := range s.createdVols {
			scope[host] = scope[host] || vol.data
		}
		for _, host := range sortedKeys(scope) {
			if scope[host] {
				v.Scope = append(v.Scope, host)
			}
		}
		s.backups = append(s.backups, backup{run: v.ID, digest: e.Digest, scope: append([]string{}, v.Scope...)})
		if len(s.backups) > maxHistory {
			s.backups = s.backups[1:]
		}
	}
	found.record(e.Digest)
	v.Done++
	switch {
	case v.Done == len(v.Steps):
		r.finish(s, found)
	case v.Kind == RunUpgrade && strings.HasPrefix(v.Steps[v.Done], "service:"):
		s.touched, s.touchedAt = true, v.Version
	}
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
	allowed := map[string]bool{}
	for _, step := range v.Steps[:v.Done] {
		for _, prefix := range []string{"service:", "remove:"} {
			if name, ok := strings.CutPrefix(step, prefix); ok {
				allowed["service:"+name] = true
				for _, key := range found.ports[name] {
					allowed["port:"+key] = true
				}
			}
		}
		if host, ok := strings.CutPrefix(step, "volume:"); ok {
			allowed["volume:"+host] = true
		}
	}
	kinds := map[string]string{RunInstall: ChangePlan, RunUpgrade: ChangeUpgrade, RunRollback: ChangeRollback}
	view, err := r.kindPreview(s, kinds[v.Kind], Acceptance{Version: v.Version, Run: r.lastFailed(s)}, now, allowed)
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
		r.ended(s, found)
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
	switch {
	case found.view.Kind != RunInstall:
		return "", refuse(ReasonNotAllowedState)
	case found.view.Steps[found.view.Done] != StepRegisterAdmin:
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
	s.record = r.recordFrom(found)
	s.remember(History{Kind: RunInstall, To: found.view.Version, Run: found.view.ID})
	return viewRun(found), nil
}

func (r *Registry) lastFailed(s *site) string {
	if len(s.failedRuns) == 0 {
		return ""
	}
	return s.failedRuns[len(s.failedRuns)-1]
}
