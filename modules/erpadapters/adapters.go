package erpadapters

import (
	"strings"
	"time"
)

type adapter struct {
	view    AdapterView
	methods map[string]*MethodView
	targets map[string]*TargetView
}

type space struct {
	profiles  map[string]*ProfileView
	adapters  map[string]*adapter
	diagnoses int
}

func (p Profile) valid() bool {
	if !validID(p.ID) || !vendors[p.Vendor] || !validID(p.Version) || !deployments[p.Deployment] || len(p.Modules) > maxModules {
		return false
	}
	if _, ok := products[p.Product]; !ok {
		return false
	}
	seen := map[string]bool{}
	for _, m := range p.Modules {
		if !validID(m) || seen[m] {
			return false
		}
		seen[m] = true
	}
	return true
}

func sameProfile(a, b Profile) bool {
	return a.ID == b.ID && a.Vendor == b.Vendor && a.Product == b.Product && a.Version == b.Version && a.Deployment == b.Deployment &&
		strings.Join(a.Modules, ",") == strings.Join(b.Modules, ",")
}

func copyProfile(v ProfileView) ProfileView {
	v.Profile.Modules = append([]string(nil), v.Profile.Modules...)
	return v
}

func (r *Registry) Profile(workspace string, p Profile, actor Actor) (ProfileView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !p.valid() || !actor.valid() {
		return ProfileView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s != nil {
		if seen := s.profiles[p.ID]; seen != nil {
			if sameProfile(seen.Profile, p) && actor.is(seen.Author) {
				return copyProfile(*seen), nil
			}
			return ProfileView{}, ErrConflict
		}
	}
	if err := actor.refusal(); err != nil {
		return ProfileView{}, err
	}
	if products[p.Product] != p.Vendor {
		return ProfileView{}, refuse(ReasonVendorMismatch)
	}
	if s == nil {
		if len(r.workspaces) >= r.limits.Workspaces {
			return ProfileView{}, ErrLimit
		}
		s = &space{profiles: map[string]*ProfileView{}, adapters: map[string]*adapter{}}
		r.workspaces[workspace] = s
	} else if len(s.profiles) >= r.limits.Profiles {
		return ProfileView{}, ErrLimit
	}
	p.Modules = append([]string(nil), p.Modules...)
	view := &ProfileView{Profile: p, Author: actor.ID, At: now}
	s.profiles[p.ID] = view
	return copyProfile(*view), nil
}

func (r *Registry) ProfileOf(workspace, id string) (ProfileView, error) {
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.profiles[id] == nil {
		return ProfileView{}, ErrNotFound
	}
	return copyProfile(*s.profiles[id]), nil
}

func knownCommand(name string) bool {
	for _, c := range commands {
		if c == name {
			return true
		}
	}
	return false
}

func (a Adapter) valid() bool {
	if !validID(a.ID) || !validID(a.Profile) || len(a.Methods) == 0 || a.MaxRows < 1 || a.MaxRows > maxRowsPerDoc ||
		a.CallsPerMinute < 1 || a.CallsPerMinute > maxCalls {
		return false
	}
	seen := map[string]bool{}
	for _, m := range a.Methods {
		if !knownCommand(m.Command) || seen[m.Command] || !validID(m.Name) || m.Transport != "api" && m.Transport != "file" && m.Transport != "database" {
			return false
		}
		seen[m.Command] = true
	}
	return true
}

func sameAdapter(a, b Adapter) bool {
	if a.ID != b.ID || a.Profile != b.Profile || a.MaxRows != b.MaxRows || a.CallsPerMinute != b.CallsPerMinute || len(a.Methods) != len(b.Methods) {
		return false
	}
	for index := range a.Methods {
		if a.Methods[index] != b.Methods[index] {
			return false
		}
	}
	return true
}

func (a *adapter) copy() AdapterView {
	v := a.view
	v.Adapter.Methods = append([]Method(nil), v.Adapter.Methods...)
	v.Methods = nil
	for _, m := range a.view.Adapter.Methods {
		v.Methods = append(v.Methods, *a.methods[m.Command])
	}
	return v
}

func (r *Registry) Adapter(workspace string, a Adapter, actor Actor) (AdapterView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !a.valid() || !actor.valid() {
		return AdapterView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil || s.profiles[a.Profile] == nil {
		return AdapterView{}, refuse(ReasonUnknownProfile)
	}
	if seen := s.adapters[a.ID]; seen != nil {
		if sameAdapter(seen.view.Adapter, a) && actor.is(seen.view.Author) {
			return seen.copy(), nil
		}
		return AdapterView{}, ErrConflict
	}
	if err := actor.refusal(); err != nil {
		return AdapterView{}, err
	}
	for _, m := range a.Methods {
		if m.Transport == "database" {
			return AdapterView{}, refuse(ReasonDirectDatabase)
		}
	}
	if len(s.adapters) >= r.limits.Adapters {
		return AdapterView{}, ErrLimit
	}
	a.Methods = append([]Method(nil), a.Methods...)
	declared := EvidenceView{Evidence: Evidence{Level: LevelDeclared}, Actor: actor.ID, At: now}
	made := &adapter{methods: map[string]*MethodView{}, targets: map[string]*TargetView{}}
	made.view = AdapterView{Adapter: a, Rows: declared, Calls: declared, Author: actor.ID, At: now}
	for _, m := range a.Methods {
		made.methods[m.Command] = &MethodView{Method: m, Evidence: declared}
	}
	s.adapters[a.ID] = made
	return made.copy(), nil
}

func (r *Registry) find(workspace, id string) (*space, *adapter, error) {
	s := r.workspaces[workspace]
	if s == nil || s.adapters[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.adapters[id], nil
}

func (r *Registry) AdapterOf(workspace, id string) (AdapterView, error) {
	r.enter()
	defer r.leave()
	_, a, err := r.find(workspace, id)
	if err != nil {
		return AdapterView{}, err
	}
	return a.copy(), nil
}

func (e Evidence) valid() bool {
	_, ok := levels[e.Level]
	return ok && e.Level != LevelDeclared && (e.Reference == "" && e.Level == LevelSimulated || validID(e.Reference))
}

func raise(current *EvidenceView, e Evidence, actor Actor, author string, now time.Time) (bool, error) {
	if current.Revoked {
		if actor.Kind == KindAssistant {
			return false, refuse(ReasonAINotAllowed)
		}
		return false, refuse(ReasonRevoked)
	}
	if current.Evidence == e && actor.is(current.Actor) || current.Evidence == e && e.Level == LevelSimulated && actor.Kind == KindSystem && actor.ID == current.Actor {
		return false, nil
	}
	if actor.Kind == KindAssistant {
		return false, refuse(ReasonAINotAllowed)
	}
	if actor.Kind == KindSystem && e.Level != LevelSimulated {
		return false, refuse(ReasonNotHuman)
	}
	if levels[e.Level] <= levels[current.Evidence.Level] {
		return false, refuse(ReasonLevelDown)
	}
	if levels[e.Level] >= levels[LevelSandbox] && actor.ID == author {
		return false, refuse(ReasonSelfApproval)
	}
	*current = EvidenceView{Evidence: e, Actor: actor.ID, At: now}
	return true, nil
}

func (r *Registry) Evidence(workspace, id, command string, e Evidence, actor Actor) (AdapterView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !knownCommand(command) || !e.valid() || !actor.valid() {
		return AdapterView{}, ErrInvalid
	}
	_, a, err := r.find(workspace, id)
	if err != nil {
		return AdapterView{}, err
	}
	m := a.methods[command]
	if m == nil {
		if actor.Kind == KindAssistant {
			return AdapterView{}, refuse(ReasonAINotAllowed)
		}
		return AdapterView{}, refuse(ReasonNotMapped)
	}
	if _, err := raise(&m.Evidence, e, actor, a.view.Author, now); err != nil {
		return AdapterView{}, err
	}
	return a.copy(), nil
}

func (a *adapter) limit(name string) *EvidenceView {
	switch name {
	case LimitRows:
		return &a.view.Rows
	case LimitCalls:
		return &a.view.Calls
	}
	return nil
}

func (r *Registry) LimitEvidence(workspace, id, limit string, e Evidence, actor Actor) (AdapterView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || limit != LimitRows && limit != LimitCalls || !e.valid() || !actor.valid() {
		return AdapterView{}, ErrInvalid
	}
	_, a, err := r.find(workspace, id)
	if err != nil {
		return AdapterView{}, err
	}
	if _, err := raise(a.limit(limit), e, actor, a.view.Author, now); err != nil {
		return AdapterView{}, err
	}
	return a.copy(), nil
}

func (r *Registry) Revoke(workspace, id, item, reason string, actor Actor) (AdapterView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(item) || !validID(reason) || !actor.valid() {
		return AdapterView{}, ErrInvalid
	}
	_, a, err := r.find(workspace, id)
	if err != nil {
		return AdapterView{}, err
	}
	current := a.limit(item)
	if current == nil {
		if m := a.methods[item]; m != nil {
			current = &m.Evidence
		}
	}
	if current != nil && current.Revoked && current.Reason == reason && actor.is(current.RevokedBy) {
		return a.copy(), nil
	}
	if err := actor.refusal(); err != nil {
		return AdapterView{}, err
	}
	if current == nil {
		return AdapterView{}, refuse(ReasonNotMapped)
	}
	if current.Revoked {
		return AdapterView{}, refuse(ReasonRevoked)
	}
	if actor.ID == a.view.Author {
		return AdapterView{}, refuse(ReasonSelfApproval)
	}
	current.Revoked, current.Reason, current.RevokedBy, current.RevokedAt = true, reason, actor.ID, now
	return a.copy(), nil
}

func level(e EvidenceView) int {
	if e.Revoked {
		return -1
	}
	return levels[e.Evidence.Level]
}

func (r *Registry) Certification(workspace, id string) (Certification, error) {
	r.enter()
	defer r.leave()
	_, a, err := r.find(workspace, id)
	if err != nil {
		return Certification{}, err
	}
	lowest := levels[LevelProduction]
	var blocking []string
	for _, name := range lifecycle {
		m := a.methods[name]
		current := -1
		if m != nil {
			current = level(m.Evidence)
		}
		if current < lowest {
			lowest = current
		}
		if current < levels[LevelSandbox] {
			blocking = append(blocking, name)
		}
	}
	switch {
	case lowest >= levels[LevelProduction]:
		return Certification{Level: LevelProduction}, nil
	case lowest >= levels[LevelSandbox]:
		return Certification{Level: LevelSandbox}, nil
	}
	return Certification{Level: CertUncertified, Blocking: blocking}, nil
}

func (r *Registry) Capabilities(workspace, id, minimum string) ([]string, error) {
	r.enter()
	defer r.leave()
	if minimum != LevelSandbox && minimum != LevelProduction {
		return nil, ErrInvalid
	}
	_, a, err := r.find(workspace, id)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range commands {
		if m := a.methods[name]; m != nil && level(m.Evidence) >= levels[minimum] {
			out = append(out, name)
		}
	}
	return out, nil
}

func (r *Registry) Admits(workspace, id, minimum string, capabilities []string) ([]string, error) {
	r.enter()
	defer r.leave()
	if minimum != LevelSandbox && minimum != LevelProduction {
		return nil, ErrInvalid
	}
	_, a, err := r.find(workspace, id)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range capabilities {
		if m := a.methods[name]; m == nil || level(m.Evidence) < levels[minimum] {
			out = append(out, name)
		}
	}
	return out, nil
}

func (d Delivery) valid() bool {
	seen := map[string]bool{}
	for _, kind := range d.Attachments {
		if !attachments[kind] || seen[kind] {
			return false
		}
		seen[kind] = true
	}
	return (d.RecipientControl == "" || validID(d.RecipientControl)) && (d.AttachmentControl == "" || validID(d.AttachmentControl))
}

func copyDiagnosis(d Diagnosis) Diagnosis {
	d.Delivery.Attachments = append([]string(nil), d.Delivery.Attachments...)
	d.Reasons = append([]string(nil), d.Reasons...)
	return d
}

func sameDelivery(a, b Delivery) bool {
	return a.AutoAttach == b.AutoAttach && a.AutoMail == b.AutoMail && a.Portal == b.Portal && a.RecipientControl == b.RecipientControl &&
		a.AttachmentControl == b.AttachmentControl && strings.Join(a.Attachments, ",") == strings.Join(b.Attachments, ",")
}

func copyTarget(t TargetView) TargetView {
	t.Basis.Attachments = append([]string(nil), t.Basis.Attachments...)
	if t.Diagnosis != nil {
		d := copyDiagnosis(*t.Diagnosis)
		t.Diagnosis = &d
	}
	if t.Enabled != nil {
		e := *t.Enabled
		t.Enabled = &e
	}
	return t
}

func (r *Registry) Diagnose(workspace, id, target string, d Delivery, actor Actor) (Diagnosis, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(target) || !d.valid() || !actor.valid() {
		return Diagnosis{}, ErrInvalid
	}
	s, a, err := r.find(workspace, id)
	if err != nil {
		return Diagnosis{}, err
	}
	if err := actor.refusal(); err != nil {
		return Diagnosis{}, err
	}
	if s.diagnoses >= r.limits.Diagnoses {
		return Diagnosis{}, ErrLimit
	}
	d.Attachments = append([]string(nil), d.Attachments...)
	s.diagnoses++
	out := Diagnosis{Number: s.diagnoses, Target: target, Delivery: d, Result: DeliveryManual, Actor: actor.ID, At: now}
	for _, kind := range d.Attachments {
		if kind == "internal_cost" {
			out.Reasons = append(out.Reasons, ReasonInternalAttachment)
		}
	}
	if d.AutoAttach || d.AutoMail || d.Portal {
		if d.RecipientControl == "" {
			out.Reasons = append(out.Reasons, ReasonRecipientUnproven)
		}
		if d.AttachmentControl == "" {
			out.Reasons = append(out.Reasons, ReasonAttachmentUnproven)
		}
		out.Result = DeliveryAllowed
		if len(out.Reasons) > 0 {
			out.Result = DeliveryBlocked
		}
	}
	t := a.targets[target]
	if t == nil {
		t = &TargetView{Target: target}
		a.targets[target] = t
	}
	if t.Enabled != nil && (out.Result != DeliveryAllowed || !sameDelivery(t.Basis, out.Delivery)) {
		t.Enabled = nil
	}
	t.Diagnosis = &out
	return copyDiagnosis(out), nil
}

func (r *Registry) EnableDelivery(workspace, id, target string, diagnosis int, actor Actor) (TargetView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(target) || diagnosis < 1 || !actor.valid() {
		return TargetView{}, ErrInvalid
	}
	_, a, err := r.find(workspace, id)
	if err != nil {
		return TargetView{}, err
	}
	t := a.targets[target]
	if t == nil {
		return TargetView{}, ErrNotFound
	}
	if t.Enabled != nil && t.Enabled.Diagnosis == diagnosis && actor.is(t.Enabled.Actor) {
		return copyTarget(*t), nil
	}
	if err := actor.refusal(); err != nil {
		return TargetView{}, err
	}
	if t.Diagnosis.Number != diagnosis {
		return TargetView{}, refuse(ReasonStaleDiagnosis)
	}
	if t.Diagnosis.Result != DeliveryAllowed {
		return TargetView{}, refuse(ReasonDeliveryUnsafe)
	}
	if actor.ID == t.Diagnosis.Actor {
		return TargetView{}, refuse(ReasonSelfApproval)
	}
	if t.Enabled != nil {
		return TargetView{}, ErrConflict
	}
	t.Enabled = &Enablement{Diagnosis: diagnosis, Actor: actor.ID, At: now}
	t.Basis = t.Diagnosis.Delivery
	return copyTarget(*t), nil
}

func (r *Registry) TargetOf(workspace, id, target string) (TargetView, error) {
	r.enter()
	defer r.leave()
	_, a, err := r.find(workspace, id)
	if err != nil {
		return TargetView{}, err
	}
	if a.targets[target] == nil {
		return TargetView{}, ErrNotFound
	}
	return copyTarget(*a.targets[target]), nil
}

func (r *Registry) Plan(workspace, id string, rows int) ([]Chunk, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || rows < 1 || rows > maxPlanRows {
		return nil, ErrInvalid
	}
	_, a, err := r.find(workspace, id)
	if err != nil {
		return nil, err
	}
	if level(a.view.Rows) < levels[LevelSandbox] {
		return nil, refuse(ReasonLimitUnproven)
	}
	size := a.view.Adapter.MaxRows
	count := (rows + size - 1) / size
	if count > maxPlanDocuments {
		return nil, ErrLimit
	}
	out := make([]Chunk, 0, count)
	for first := 1; first <= rows; first += size {
		last := first + size - 1
		if last > rows {
			last = rows
		}
		out = append(out, Chunk{First: first, Last: last})
	}
	return out, nil
}
