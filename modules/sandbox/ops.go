package sandbox

import (
	"strconv"
	"strings"
)

func (r *Registry) DisableDestination(id string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validDestID(id) {
		return ErrInvalid
	}
	d := r.destinations[id]
	if d == nil {
		return ErrNotFound
	}
	d.enabled = false
	return nil
}

func (r *Registry) Allow(q Request) error {
	r.enter()
	defer r.leave()
	if !validEnvID(q.Environment) || !oneOf(q.Kind, kinds...) || !validHost(q.Host) || q.Port < 1 || q.Port > 65535 {
		return ErrInvalid
	}
	env := r.environments[q.Environment]
	if env == nil {
		return ErrNotFound
	}
	if env.view.Mode == ModeProduction {
		if isReserved(q.Host) {
			return refuse(ReasonMockDestination)
		}
		return nil
	}
	d := r.destinations[r.endpoints[endpoint(q.Host, q.Port)]]
	if d == nil || !d.enabled || d.view.Environment != q.Environment || d.view.Kind != q.Kind {
		return refuse(ReasonNotMock)
	}
	return nil
}

func (r *Registry) CheckConfig(c Config) error {
	r.enter()
	defer r.leave()
	if !validEnvID(c.Environment) || !validID(c.Schema) || len(c.Settings) > maxFields {
		return ErrInvalid
	}
	for index, s := range c.Settings {
		if !validKey(s.Key) || index > 0 && c.Settings[index-1].Key >= s.Key {
			return ErrInvalid
		}
	}
	env, schema := r.environments[c.Environment], r.schemas[c.Schema]
	switch {
	case env == nil || schema == nil:
		return ErrNotFound
	case env.view.Mode != ModeDemo:
		return refuse(ReasonNotDemo)
	}
	fields := map[string]FieldSpec{}
	for _, f := range schema.Fields {
		fields[f.Key] = f
	}
	for _, s := range c.Settings {
		f, ok := fields[s.Key]
		if !ok {
			return refuse(ReasonUndeclared)
		}
		if !r.fits(f, s.Value, c.Environment) {
			return ErrInvalid
		}
	}
	return nil
}

func (r *Registry) fits(f FieldSpec, value, env string) bool {
	switch f.Kind {
	case FieldEnum:
		return oneOf(value, f.Values...)
	case FieldInteger:
		n, err := strconv.ParseInt(value, 10, 64)
		return err == nil && strconv.FormatInt(n, 10) == value
	case FieldFlag:
		return value == "true" || value == "false"
	}
	d := r.destinations[value]
	return d != nil && d.enabled && d.view.Environment == env && d.view.Kind == f.Values[0]
}

func (t Target) valid() bool {
	switch t.Kind {
	case TargetEvidence:
		return validPath(t.Name)
	case TargetDatabase, TargetKeyRing, TargetNumbers, TargetConnection, TargetVolume:
		return validID(t.Name)
	}
	return false
}

func (r *Registry) inside(env *environment, t Target) bool {
	e := env.view
	switch t.Kind {
	case TargetDatabase:
		return t.Name == e.Database
	case TargetKeyRing:
		return t.Name == e.KeyRing
	case TargetEvidence:
		return within(t.Name, e.EvidencePath)
	case TargetNumbers:
		return t.Name == e.NumberPrefix
	case TargetConnection:
		d := r.destinations[t.Name]
		return d != nil && d.view.Environment == e.ID
	}
	return t.Name == e.ID || strings.HasPrefix(t.Name, e.ID+"-") && len(t.Name) > len(e.ID)+1
}

func (r *Registry) BeginReset(p Plan, actor Actor) (ResetView, error) {
	now := r.enter()
	defer r.leave()
	switch {
	case !actor.valid() || !validEnvID(p.Environment) || p.Key != "" && !validID(p.Key) || len(p.Targets) == 0 || len(p.Targets) > maxTargets:
		return ResetView{}, ErrInvalid
	case actor.Kind == KindAssistant:
		return ResetView{}, refuse(ReasonAINotAllowed)
	}
	for index, t := range p.Targets {
		if !t.valid() || index > 0 && !less(p.Targets[index-1], t) {
			return ResetView{}, ErrInvalid
		}
	}
	env := r.environments[p.Environment]
	if env == nil {
		return ResetView{}, ErrNotFound
	}
	if env.view.Mode != ModeDemo {
		return ResetView{}, refuse(ReasonNotDemo)
	}
	if id, ok := env.keys[p.Key]; ok && p.Key != "" {
		x := env.resets[id]
		if !sameTargets(x.view.Targets, p.Targets) {
			return ResetView{}, ErrConflict
		}
		return copyReset(x.view), nil
	}
	for _, t := range p.Targets {
		if !r.inside(env, t) {
			return ResetView{}, refuse(ReasonOutsideEnv)
		}
	}
	env.next++
	if p.Key != "" {
		env.keys[p.Key] = env.next
	}
	x := &reset{view: ResetView{Environment: p.Environment, ID: env.next, Key: p.Key, Targets: append([]Target(nil), p.Targets...), Done: make([]bool, len(p.Targets)), By: actor.ID, At: now}}
	env.resets[x.view.ID] = x
	if len(env.resets) > maxResets {
		env.watermark++
		delete(env.keys, env.resets[env.watermark].view.Key)
		delete(env.resets, env.watermark)
	}
	return copyReset(x.view), nil
}

func sameTargets(a, b []Target) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func less(a, b Target) bool { return a.Kind < b.Kind || a.Kind == b.Kind && a.Name < b.Name }

func copyReset(v ResetView) ResetView {
	v.Targets = append([]Target(nil), v.Targets...)
	v.Done = append([]bool(nil), v.Done...)
	return v
}

func (r *Registry) lookupReset(environment string, id uint64) (*reset, error) {
	if !validEnvID(environment) || id == 0 {
		return nil, ErrInvalid
	}
	env := r.environments[environment]
	switch {
	case env == nil || id > env.next:
		return nil, ErrNotFound
	case id <= env.watermark:
		return nil, refuse(ReasonStale)
	}
	return env.resets[id], nil
}

func (r *Registry) ResetRecord(environment string, id uint64) (ResetView, error) {
	r.enter()
	defer r.leave()
	x, err := r.lookupReset(environment, id)
	if err != nil {
		return ResetView{}, err
	}
	return copyReset(x.view), nil
}

func (r *Registry) CompleteTarget(environment string, id uint64, t Target, actor Actor) (ResetView, error) {
	r.enter()
	defer r.leave()
	switch {
	case !actor.valid() || !t.valid():
		return ResetView{}, ErrInvalid
	case actor.Kind == KindAssistant:
		return ResetView{}, refuse(ReasonAINotAllowed)
	}
	x, err := r.lookupReset(environment, id)
	if err != nil {
		return ResetView{}, err
	}
	for index, target := range x.view.Targets {
		if target == t {
			x.view.Done[index] = true
			x.view.Completed = !contains(x.view.Done, false)
			return copyReset(x.view), nil
		}
	}
	return ResetView{}, ErrNotFound
}

func contains(list []bool, value bool) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func (r *Registry) Admit(in Input) error {
	r.enter()
	defer r.leave()
	if !validEnvID(in.Environment) || !validEnvID(in.Source) || !validStore(in.Database) || !oneOf(in.Origin, OriginReal, OriginPublic, OriginConverted, OriginSynthetic) ||
		in.Number != "" && !validID(in.Number) {
		return ErrInvalid
	}
	env := r.environments[in.Environment]
	if env == nil {
		return ErrNotFound
	}
	switch {
	case in.Source != in.Environment || in.Database != env.view.Database:
		return refuse(ReasonCrossEnvironment)
	case env.view.Mode == ModeProduction && in.Origin != OriginReal:
		return refuse(ReasonNotReal)
	case in.Number != "" && (!strings.HasPrefix(in.Number, env.view.NumberPrefix) || in.Number == env.view.NumberPrefix):
		return refuse(ReasonForeignNumber)
	}
	return nil
}

func (r *Registry) Label(environment, origin, source string) (LabelView, error) {
	r.enter()
	defer r.leave()
	if !validEnvID(environment) || !oneOf(origin, OriginReal, OriginPublic, OriginConverted, OriginSynthetic) || !validID(source) {
		return LabelView{}, ErrInvalid
	}
	env := r.environments[environment]
	if env == nil {
		return LabelView{}, ErrNotFound
	}
	v := LabelView{Environment: environment, Origin: origin, Source: source}
	if env.view.Mode == ModeDemo {
		v.Demo, v.Marking = true, MarkingDemo
	}
	return v, nil
}
