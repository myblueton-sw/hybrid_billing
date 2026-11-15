package installer

import (
	"sort"
	"strings"
	"time"
)

const (
	maxDetached = 32
	maxHistory  = 100
	upgradeTag  = "hybrid-billing/install-upgrade/v1"
	rollbackTag = "hybrid-billing/install-rollback/v1"
)

func acceptance(kind string) bool {
	return kind == ChangePlan || kind == ChangeUpgrade || kind == ChangeRollback
}

func listed(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func sortedKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) observed(s *site, now time.Time) error {
	switch {
	case s.observation == nil:
		return ErrNotFound
	case now.After(s.observed.Add(r.limits.ObservationLife)):
		return refuse(ReasonObservationStale)
	}
	return nil
}

func (r *Registry) activeRun(s *site) bool {
	for _, other := range s.runs {
		if active(other.view.State) {
			return true
		}
	}
	return false
}

func sortConflicts(list []Item) {
	sort.Slice(list, func(a, b int) bool {
		x, y := list[a], list[b]
		return x.Conflict < y.Conflict || x.Conflict == y.Conflict && (x.Kind < y.Kind || x.Kind == y.Kind && x.Target < y.Target)
	})
}

func (r *Registry) upgradePreview(s *site, rel *release, now time.Time, allowed map[string]bool) (PreviewView, error) {
	rec := s.record
	if rec == nil {
		return PreviewView{}, refuse(ReasonNotInstalled)
	}
	supported := false
	for _, p := range rel.release.Platforms {
		supported = supported || p == s.view.Platform
	}
	switch {
	case r.revoked(rel):
		return PreviewView{}, refuse(ReasonRevoked)
	case !supported:
		return PreviewView{}, refuse(ReasonUnsupported)
	case compare(rel.release.Version, rec.version) <= 0:
		return PreviewView{}, refuse(ReasonDirection)
	case !listed(rel.release.UpgradeFrom, rec.version):
		return PreviewView{}, refuse(ReasonNotUpgradable)
	case s.touched && rel.release.Version != s.touchedAt && !listed(rel.release.UpgradeFrom, s.touchedAt):
		return PreviewView{}, refuse(ReasonDataForm)
	}
	if err := r.observed(s, now); err != nil {
		return PreviewView{}, err
	}
	o := s.observation
	plan := r.resolve(s, rel.release)
	v := PreviewView{Kind: RunUpgrade, Basis: BasisRecordName, Site: s.view.ID, Version: rel.release.Version, Services: []Item{}, Ports: []Item{}, Volumes: []Item{},
		Data: []DataEffect{}, Permissions: []string{}, Conflicts: []Item{}, Leftovers: []Item{}}
	conflict := func(item *Item, code string) {
		item.Conflict = code
		v.Conflicts = append(v.Conflicts, *item)
	}
	used := map[string]bool{}
	for _, p := range o.Ports {
		used[p.key()] = true
	}
	owner := map[string]string{}
	for name, keys := range rec.services {
		for _, key := range keys {
			owner[key] = name
		}
	}
	adopted := map[string]string{}
	for name, keys := range s.created {
		for _, key := range keys {
			adopted[key] = name
		}
	}
	counts := map[string]int{}
	target := map[string]bool{}
	for _, svc := range rel.release.Services {
		target[svc.Name] = true
		for _, key := range plan.ports[svc.Name] {
			counts[key]++
		}
	}
	for _, svc := range rel.release.Services {
		item := Item{Kind: "service", Target: svc.Name, Action: ActionCreate}
		_, owned := rec.services[svc.Name]
		_, made := s.created[svc.Name]
		switch {
		case owned:
			item.Action = ActionReplace
		case made:
			item.Action = ActionAdopt
		case contains(o.Services, svc.Name) && !allowed["service:"+svc.Name]:
			conflict(&item, ConflictServiceExists)
		}
		v.Services = append(v.Services, item)
		for _, key := range plan.ports[svc.Name] {
			port := Item{Kind: "port", Target: svc.Name + ":" + key, Action: ActionBind}
			switch {
			case owner[key] == svc.Name || adopted[key] == svc.Name:
				port.Action = ActionReuse
			case owner[key] != "":
				conflict(&port, ConflictPortMoved)
			case used[key] && !allowed["port:"+key]:
				conflict(&port, ConflictPortInUse)
			}
			if counts[key] > 1 {
				conflict(&port, ConflictPortDuplicate)
			}
			v.Ports = append(v.Ports, port)
		}
	}
	for _, name := range sortedKeys(rec.services) {
		if !target[name] {
			v.Services = append(v.Services, Item{Kind: "service", Target: name, Action: ActionRemove})
		}
	}
	hosts := map[string]int{}
	for _, host := range plan.volumes {
		hosts[host]++
	}
	targetHosts := map[string]bool{}
	for _, vol := range rel.release.Volumes {
		host := plan.volumes[vol.Name]
		targetHosts[host] = true
		item := Item{Kind: "volume", Target: host, Action: ActionCreate}
		effect := ""
		owned, made := rec.volumes[host], s.createdVols[host]
		switch {
		case owned != nil && owned.logical != vol.Name, owned == nil && made != nil && made.logical != vol.Name:
			conflict(&item, ConflictVolumeRemapped)
		case owned != nil && owned.data && !vol.Data:
			conflict(&item, ConflictDataRemoved)
		case owned != nil && owned.detached && !contains(o.Volumes, host):
			if vol.Data {
				effect = EffectNew
			}
		case owned != nil && owned.detached:
			item.Action, effect = ActionReattach, EffectStale
		case owned != nil:
			item.Action = ActionRetain
			if vol.Data {
				effect = EffectRetained
				if !owned.data {
					effect = EffectStale
				}
			}
		case made != nil && contains(o.Volumes, host):
			item.Action = ActionAdopt
			if vol.Data {
				effect = EffectNew
			}
		case contains(o.Volumes, host) && !allowed["volume:"+host]:
			conflict(&item, ConflictExistingVolume)
			effect = EffectUntouched
		case vol.Data:
			effect = EffectNew
		}
		if hosts[host] > 1 {
			conflict(&item, ConflictVolumeDuplicate)
		}
		if effect != "" {
			v.Data = append(v.Data, DataEffect{Volume: host, Effect: effect})
		}
		v.Volumes = append(v.Volumes, item)
	}
	detached := 0
	for _, host := range sortedKeys(rec.volumes) {
		owned := rec.volumes[host]
		switch {
		case targetHosts[host]:
		case owned.data:
			item := Item{Kind: "volume", Target: host, Action: ActionRetain}
			conflict(&item, ConflictDataRemoved)
		case owned.detached:
			if contains(o.Volumes, host) {
				detached++
			}
		default:
			detached++
			v.Volumes = append(v.Volumes, Item{Kind: "volume", Target: host, Action: ActionDetach})
		}
	}
	if detached > maxDetached {
		return PreviewView{}, ErrLimit
	}
	for _, name := range sortedKeys(rec.services) {
		if !contains(o.Services, name) && !s.removed[name] && !allowed["service:"+name] {
			item := Item{Kind: "service", Target: name, Action: ActionReplace}
			conflict(&item, ConflictOwnedMissing)
		}
	}
	for _, host := range sortedKeys(rec.volumes) {
		if !rec.volumes[host].detached && !contains(o.Volumes, host) && !allowed["volume:"+host] {
			item := Item{Kind: "volume", Target: host, Action: ActionRetain}
			conflict(&item, ConflictOwnedMissing)
		}
	}
	if s.config.Version != "" && s.config.Version != rel.release.Version && len(s.config.Ports)+len(s.config.Volumes) > 0 {
		item := Item{Kind: "config", Target: s.config.Version}
		conflict(&item, ConflictConfigMismatch)
	}
	sort.Slice(v.Data, func(a, b int) bool { return v.Data[a].Volume < v.Data[b].Volume })
	for _, p := range rel.release.permissions() {
		if !contains(o.Granted, p) {
			v.Permissions = append(v.Permissions, p)
		}
	}
	sortConflicts(v.Conflicts)
	v.Hash = r.transitionHash(upgradeTag, s, rel, v)
	return v, nil
}

func (r *Registry) rollbackPreview(s *site, now time.Time, allowed map[string]bool) (PreviewView, error) {
	rec := s.record
	switch {
	case rec == nil:
		return PreviewView{}, refuse(ReasonNotInstalled)
	case !s.failed:
		return PreviewView{}, refuse(ReasonNotNeeded)
	case s.touched:
		return PreviewView{}, refuse(ReasonDataTouched)
	}
	rel := r.releases[rec.version]
	if r.revoked(rel) {
		return PreviewView{}, refuse(ReasonRevoked)
	}
	if err := r.observed(s, now); err != nil {
		return PreviewView{}, err
	}
	o := s.observation
	v := PreviewView{Kind: RunRollback, Basis: BasisRecordName, Site: s.view.ID, Version: rec.version, Services: []Item{}, Ports: []Item{}, Volumes: []Item{},
		Data: []DataEffect{}, Permissions: []string{}, Conflicts: []Item{}, Leftovers: []Item{}}
	for _, name := range sortedKeys(rec.services) {
		item := Item{Kind: "service", Target: name, Action: ActionRestart}
		if !contains(o.Services, name) && !s.removed[name] && !allowed["service:"+name] {
			item.Conflict = ConflictOwnedMissing
			v.Conflicts = append(v.Conflicts, item)
		}
		v.Services = append(v.Services, item)
		for _, key := range rec.services[name] {
			v.Ports = append(v.Ports, Item{Kind: "port", Target: name + ":" + key, Action: ActionReuse})
		}
	}
	for _, host := range sortedKeys(rec.volumes) {
		owned := rec.volumes[host]
		if owned.detached {
			continue
		}
		item := Item{Kind: "volume", Target: host, Action: ActionRetain}
		if !contains(o.Volumes, host) {
			item.Conflict = ConflictOwnedMissing
			v.Conflicts = append(v.Conflicts, item)
		}
		if owned.data {
			v.Data = append(v.Data, DataEffect{Volume: host, Effect: EffectRetained})
		}
		v.Volumes = append(v.Volumes, item)
	}
	for _, host := range sortedKeys(s.createdVols) {
		if rec.volumes[host] == nil {
			v.Leftovers = append(v.Leftovers, Item{Kind: "volume", Target: host, Action: ActionLeftover})
		}
	}
	sortConflicts(v.Conflicts)
	v.Hash = r.transitionHash(rollbackTag, s, rel, v)
	return v, nil
}

func (r *Registry) transitionHash(tag string, s *site, rel *release, v PreviewView) string {
	var e encoder
	for _, value := range []string{tag, s.view.ID, s.view.Tenant, s.view.Probe, rel.release.Version, rel.digest, s.observation.digest(), s.record.version, s.touchedAt} {
		e.text(value)
	}
	e.number(int64(s.revision))
	for _, flag := range []bool{s.failed, s.touched} {
		if flag {
			e.text("yes")
		} else {
			e.text("no")
		}
	}
	for _, set := range []map[string][]string{s.record.services, s.created} {
		e.count(len(set))
		for _, name := range sortedKeys(set) {
			e.text(name)
			e.count(len(set[name]))
			for _, key := range set[name] {
				e.text(key)
			}
		}
	}
	for _, set := range []map[string]*ownedVolume{s.record.volumes, s.createdVols} {
		e.count(len(set))
		for _, host := range sortedKeys(set) {
			vol := set[host]
			e.text(host)
			e.text(vol.logical)
			e.text(strings.Join([]string{boolText(vol.data), boolText(vol.detached)}, ","))
		}
	}
	removed := sortedKeys(s.removed)
	e.count(len(removed))
	for _, name := range removed {
		e.text(name)
	}
	e.count(len(s.failedRuns))
	for _, id := range s.failedRuns {
		e.text(id)
	}
	e.count(len(s.backups))
	for _, b := range s.backups {
		e.text(b.run)
		e.text(b.digest)
		e.count(len(b.scope))
		for _, host := range b.scope {
			e.text(host)
		}
	}
	for _, list := range [][]Item{v.Services, v.Ports, v.Volumes, v.Conflicts, v.Leftovers} {
		e.count(len(list))
		for _, item := range list {
			e.text(item.Kind)
			e.text(item.Target)
			e.text(item.Action)
			e.text(item.Conflict)
		}
	}
	e.count(len(v.Data))
	for _, d := range v.Data {
		e.text(d.Volume)
		e.text(d.Effect)
	}
	e.count(len(v.Permissions))
	for _, p := range v.Permissions {
		e.text(p)
	}
	return digestOf(e.out.Bytes())
}

func boolText(flag bool) string {
	if flag {
		return "1"
	}
	return "0"
}

func (r *Registry) kindPreview(s *site, kind string, a Acceptance, now time.Time, allowed map[string]bool) (PreviewView, error) {
	switch kind {
	case ChangeUpgrade:
		return r.upgradePreview(s, r.releases[a.Version], now, allowed)
	case ChangeRollback:
		v, err := r.rollbackPreview(s, now, allowed)
		if err == nil && (a.Version != s.record.version || len(s.failedRuns) == 0 || a.Run != s.failedRuns[len(s.failedRuns)-1]) {
			return PreviewView{}, refuse(ReasonPlanChanged)
		}
		return v, err
	}
	return r.preview(s, r.releases[a.Version], now, allowed)
}

func upgradeSteps(rel Release, v PreviewView) []string {
	out := []string{StepPreflight, StepQuiesce, StepBackup}
	for _, a := range rel.Artifacts {
		out = append(out, "artifact:"+a.Name)
	}
	for _, item := range v.Volumes {
		if item.Action == ActionCreate {
			out = append(out, "volume:"+item.Target)
		}
	}
	for _, item := range v.Services {
		if item.Action == ActionRemove {
			out = append(out, "remove:"+item.Target)
		}
	}
	for _, svc := range rel.Services {
		out = append(out, "service:"+svc.Name)
	}
	return append(out, StepVerify)
}

func rollbackSteps(rel Release, rec *record) []string {
	out := []string{StepPreflight}
	for _, a := range rel.Artifacts {
		out = append(out, "artifact:"+a.Name)
	}
	for _, name := range sortedKeys(rec.services) {
		out = append(out, "service:"+name)
	}
	return append(out, StepVerify)
}

func (r *Registry) layoutOf(s *site, rel Release, made *run) {
	plan := r.resolve(s, rel)
	made.logical, made.data, made.ports = map[string]string{}, map[string]bool{}, map[string][]string{}
	for _, vol := range rel.Volumes {
		host := plan.volumes[vol.Name]
		made.logical[host], made.data[host] = vol.Name, vol.Data
	}
	for _, svc := range rel.Services {
		made.ports[svc.Name] = append([]string{}, plan.ports[svc.Name]...)
	}
}

func (r *Registry) recordFrom(made *run) *record {
	rec := &record{version: made.view.Version, services: map[string][]string{}, volumes: map[string]*ownedVolume{}}
	for name, keys := range made.ports {
		rec.services[name] = append([]string{}, keys...)
	}
	for host, logical := range made.logical {
		rec.volumes[host] = &ownedVolume{logical: logical, data: made.data[host]}
	}
	return rec
}

func (s *site) remember(entry History) {
	s.history = append(s.history, entry)
	if len(s.history) > maxHistory {
		s.history = s.history[1:]
	}
}

func (s *site) clearFailure() {
	s.failed, s.touched, s.touchedAt = false, false, ""
	s.created, s.createdVols, s.removed = map[string][]string{}, map[string]*ownedVolume{}, map[string]bool{}
	s.failedRuns, s.backups = nil, nil
}

func (r *Registry) finish(s *site, made *run) {
	v := &made.view
	v.State = RunCompleted
	switch v.Kind {
	case RunUpgrade:
		rec := r.recordFrom(made)
		for host, old := range s.record.volumes {
			if rec.volumes[host] == nil && (!old.detached || contains(s.observation.Volumes, host)) {
				rec.volumes[host] = &ownedVolume{logical: old.logical, data: old.data, detached: true}
			}
		}
		for host, vol := range s.createdVols {
			if rec.volumes[host] == nil && vol.data {
				rec.volumes[host] = &ownedVolume{logical: vol.logical, data: true, detached: true}
			}
		}
		s.remember(History{Kind: RunUpgrade, From: s.record.version, To: v.Version, Run: v.ID, Backup: v.Backup})
		s.record = rec
	case RunRollback:
		for _, id := range s.failedRuns {
			s.rolledBack[id] = true
			s.rolledOrder = append(s.rolledOrder, id)
		}
		for len(s.rolledOrder) > maxHistory {
			delete(s.rolledBack, s.rolledOrder[0])
			s.rolledOrder = s.rolledOrder[1:]
		}
		s.remember(History{Kind: RunRollback, To: v.Version, Run: v.ID, Runs: append([]string{}, s.failedRuns...)})
	}
	s.clearFailure()
}

func (r *Registry) ended(s *site, made *run) {
	v := made.view
	if v.Kind != RunUpgrade || v.Done < 2 {
		return
	}
	s.failed = true
	s.failedRuns = append(s.failedRuns, v.ID)
	if len(s.failedRuns) > maxHistory {
		s.failedRuns = s.failedRuns[1:]
	}
	for _, step := range v.Steps[:v.Done] {
		if host, ok := strings.CutPrefix(step, "volume:"); ok {
			s.createdVols[host] = &ownedVolume{logical: made.logical[host], data: made.data[host]}
		}
		if name, ok := strings.CutPrefix(step, "service:"); ok {
			s.created[name] = append([]string{}, made.ports[name]...)
		}
		if name, ok := strings.CutPrefix(step, "remove:"); ok {
			s.removed[name] = true
		}
	}
}

func (r *Registry) PreviewUpgrade(id, version string) (PreviewView, error) {
	now := r.enter()
	defer r.leave()
	if _, ok := semver(version); !ok || !validID(id) {
		return PreviewView{}, ErrInvalid
	}
	s, rel := r.sites[id], r.releases[version]
	if s == nil || rel == nil {
		return PreviewView{}, ErrNotFound
	}
	if r.activeRun(s) {
		return PreviewView{}, refuse(ReasonRunOpen)
	}
	return r.upgradePreview(s, rel, now, nil)
}

func (r *Registry) PreviewRollback(id string) (PreviewView, error) {
	now := r.enter()
	defer r.leave()
	s, err := r.site(id)
	if err != nil {
		return PreviewView{}, err
	}
	if r.activeRun(s) {
		return PreviewView{}, refuse(ReasonRunOpen)
	}
	return r.rollbackPreview(s, now, nil)
}

func (r *Registry) Installed(id string) (InstalledView, error) {
	r.enter()
	defer r.leave()
	s, err := r.site(id)
	if err != nil {
		return InstalledView{}, err
	}
	if s.record == nil {
		return InstalledView{}, ErrNotFound
	}
	v := InstalledView{Version: s.record.version, State: StateInstalled, Touched: s.touched, TouchedVersion: s.touchedAt, Services: []OwnedService{}, Volumes: []OwnedVolume{},
		History: []History{}, Backups: []string{}}
	for _, entry := range s.history {
		entry.Runs = append([]string{}, entry.Runs...)
		v.History = append(v.History, entry)
	}
	for _, b := range s.backups {
		v.Backups = append(v.Backups, b.digest)
	}
	if s.failed {
		v.State = StateUpgradeFailed
	}
	for _, name := range sortedKeys(s.record.services) {
		v.Services = append(v.Services, OwnedService{Name: name, Ports: append([]string{}, s.record.services[name]...)})
	}
	for _, host := range sortedKeys(s.record.volumes) {
		vol := s.record.volumes[host]
		v.Volumes = append(v.Volumes, OwnedVolume{Host: host, Logical: vol.logical, Data: vol.data, Detached: vol.detached})
	}
	return v, nil
}
