package plugins

import (
	"crypto/ed25519"
	"crypto/sha256"
	"time"
)

func clean(c Change) (Change, string, bool) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangePublisher:
		out.Publisher = c.Publisher
		return out, registerScope, validID(c.Publisher.ID) && validText(c.Publisher.Name)
	case ChangeKey:
		k := c.Key
		k.NotBefore, k.NotAfter = k.NotBefore.UTC().Round(0), k.NotAfter.UTC().Round(0)
		k.Public, k.Proof = append([]byte(nil), k.Public...), append([]byte(nil), k.Proof...)
		out.Key = k
		return out, registerScope, validID(k.Publisher) && validID(k.Version) && !k.NotBefore.IsZero() && k.NotBefore.Before(k.NotAfter) && len(k.Proof) == ed25519.SignatureSize
	case ChangeVersion:
		out.Manifest, out.Signature = copyManifest(c.Manifest), append([]byte(nil), c.Signature...)
		return out, registerScope, c.Manifest.valid() && len(c.Signature) == ed25519.SignatureSize
	case ChangeWorkspace:
		out.Workspace = c.Workspace
		return out, registerScope, validID(c.Workspace.ID) && validID(c.Workspace.Runtime)
	case ChangeRuntime:
		out.Workspace = Workspace{ID: c.Workspace.ID, Runtime: c.Workspace.Runtime}
		return out, c.Workspace.ID, validID(c.Workspace.ID) && validID(c.Workspace.Runtime)
	case ChangeInstall, ChangeUpgrade, ChangeRollback:
		out.Workspace, out.Plugin, out.Version = Workspace{ID: c.Workspace.ID}, c.Plugin, c.Version
		_, ok := semver(c.Version)
		return out, c.Workspace.ID, validID(c.Workspace.ID) && validID(c.Plugin) && ok
	case ChangeEnable, ChangeRemove:
		out.Workspace, out.Plugin = Workspace{ID: c.Workspace.ID}, c.Plugin
		return out, c.Workspace.ID, validID(c.Workspace.ID) && validID(c.Plugin)
	case ChangeEndpoint:
		out.Workspace, out.Plugin, out.Endpoint = Workspace{ID: c.Workspace.ID}, c.Plugin, c.Endpoint
		return out, c.Workspace.ID, validID(c.Workspace.ID) && validID(c.Plugin) && validID(c.Endpoint.ID) && validHex(c.Endpoint.Fingerprint) && validHex(c.Endpoint.Evidence)
	}
	return out, registerScope, false
}

func encodeChange(c Change) []byte {
	var e encoder
	e.text(c.Kind)
	e.text(c.Publisher.ID)
	e.text(c.Publisher.Name)
	e.text(c.Key.Publisher)
	e.text(c.Key.Version)
	e.text(string(c.Key.Public))
	for _, at := range []time.Time{c.Key.NotBefore, c.Key.NotAfter} {
		e.number(at.Unix())
		e.number(int64(at.Nanosecond()))
	}
	e.text(string(c.Key.Proof))
	e.text(string(c.Manifest.encode()))
	e.text(string(c.Signature))
	e.text(c.Workspace.ID)
	e.text(c.Workspace.Runtime)
	if c.Workspace.Verification {
		e.text("verification")
	}
	e.text(c.Plugin)
	e.text(c.Version)
	e.text(c.Endpoint.ID)
	e.text(c.Endpoint.Fingerprint)
	e.text(c.Endpoint.Evidence)
	return e.out.Bytes()
}

func copyChange(c Change) Change {
	c.Key.Public, c.Key.Proof = append([]byte(nil), c.Key.Public...), append([]byte(nil), c.Key.Proof...)
	c.Manifest, c.Signature = copyManifest(c.Manifest), append([]byte(nil), c.Signature...)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func (r *Registry) keyOf(m Manifest) (*publisher, *key) {
	p := r.publishers[m.Publisher]
	if p == nil {
		return nil, nil
	}
	return p, p.keys[m.Key]
}

func (r *Registry) runnable(w *workspace, v *version) string {
	p, k := r.keyOf(v.manifest)
	switch {
	case p.revocation != nil || k.compromise != nil:
		return ReasonRevoked
	case !r.permitted(v.manifest.License):
		return ReasonLicenseNotPermitted
	case v.vulnerable != nil && !w.view.Verification:
		return ReasonVulnerable
	}
	return ""
}

func (r *Registry) permitted(license string) bool {
	for _, value := range r.config.Licenses {
		if value == license {
			return true
		}
	}
	return false
}

func (r *Registry) compatible(m Manifest) bool {
	return compare(m.Core.Min, r.config.Core) <= 0 && compare(r.config.Core, m.Core.Max) <= 0
}

func (r *Registry) check(c Change, now time.Time) error {
	switch c.Kind {
	case ChangePublisher:
		return r.checkPublisher(c.Publisher)
	case ChangeKey:
		return r.checkKey(c.Key)
	case ChangeVersion:
		return r.checkVersion(c.Manifest, c.Signature, now)
	case ChangeWorkspace:
		return r.checkWorkspace(c.Workspace)
	case ChangeRuntime:
		return r.checkRuntime(c.Workspace)
	case ChangeInstall:
		return r.checkInstall(c, now)
	case ChangeUpgrade, ChangeRollback:
		return r.checkSwitch(c)
	case ChangeEnable:
		return r.checkEnable(c)
	case ChangeRemove:
		_, err := r.live(c.Workspace.ID, c.Plugin)
		return err
	}
	return r.checkEndpoint(c)
}

func (r *Registry) checkPublisher(p Publisher) error {
	switch {
	case r.publishers[p.ID] != nil:
		return ErrConflict
	case len(r.publishers) >= r.limits.Publishers:
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkKey(k PublisherKey) error {
	p := r.publishers[k.Publisher]
	if p == nil {
		return ErrNotFound
	}
	switch {
	case p.names[k.Version]:
		return ErrConflict
	case p.revocation != nil:
		return refuse(ReasonRevoked)
	case weak(k.Public):
		return refuse(ReasonWeakKey)
	case k.NotAfter.Sub(k.NotBefore) > r.limits.KeyLife:
		return refuse(ReasonKeyLife)
	case !ed25519.Verify(k.Public, challenge(k.Publisher, k.Version), k.Proof):
		return refuse(ReasonKeyMismatch)
	}
	active := 0
	for _, other := range p.keys {
		if !other.view.Retired {
			active++
		}
	}
	if len(p.names) >= r.limits.Keys*namesPerKey || active >= r.limits.Keys {
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkVersion(m Manifest, signature []byte, now time.Time) error {
	p, k := r.keyOf(m)
	if k == nil {
		return ErrNotFound
	}
	owner := r.plugins[m.Plugin]
	switch {
	case owner != nil && owner.versions[m.Version] != nil:
		return ErrConflict
	case owner != nil && owner.owner != m.Publisher:
		return refuse(ReasonPublisherMismatch)
	case p.revocation != nil:
		return refuse(ReasonPublisherInactive)
	case k.view.Retired || now.Before(k.view.NotBefore) || now.After(k.view.NotAfter):
		return refuse(ReasonKeyInactive)
	case !ed25519.Verify(k.view.Public, m.encode(), signature):
		return refuse(ReasonSignatureInvalid)
	case !r.compatible(m):
		return refuse(ReasonIncompatible)
	case owner == nil && len(r.plugins) >= r.limits.Plugins || r.versions >= r.limits.Versions:
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkWorkspace(w Workspace) error {
	switch {
	case r.workspaces[w.ID] != nil:
		return ErrConflict
	case r.runtimes[w.Runtime] != "":
		return refuse(ReasonRuntimeTaken)
	case len(r.workspaces) >= r.limits.Workspaces:
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkRuntime(w Workspace) error {
	switch {
	case r.runtimes[w.Runtime] != "":
		return refuse(ReasonRuntimeTaken)
	case r.workspaces[w.ID].runtimes >= runtimesPer:
		return ErrLimit
	}
	return nil
}

func (r *Registry) target(c Change) (*workspace, *version, error) {
	w := r.workspaces[c.Workspace.ID]
	p := r.plugins[c.Plugin]
	if p == nil || p.versions[c.Version] == nil {
		return nil, nil, ErrNotFound
	}
	return w, p.versions[c.Version], nil
}

func (r *Registry) live(workspace, plugin string) (*installation, error) {
	i := r.workspaces[workspace].installations[plugin]
	switch {
	case i == nil:
		return nil, ErrNotFound
	case i.state == StateRemoved:
		return nil, refuse(ReasonNotAllowedState)
	}
	return i, nil
}

func held(i *installation, now time.Time) bool {
	return i.state != StateRemoved || i.disposal.Deletion == RetainForEvidence && !now.After(i.removed.AddDate(0, 0, i.disposal.RetentionDays))
}

func (r *Registry) taken(w *workspace, plugin, namespace string, now time.Time) bool {
	for name, other := range w.installations {
		if name != plugin && other.disposal.Namespace == namespace && held(other, now) {
			return true
		}
	}
	return false
}

func (r *Registry) checkInstall(c Change, now time.Time) error {
	w, v, err := r.target(c)
	if err != nil {
		return err
	}
	i := w.installations[c.Plugin]
	if i != nil && i.state != StateRemoved {
		return refuse(ReasonNotAllowedState)
	}
	if reason := r.runnable(w, v); reason != "" {
		return refuse(reason)
	}
	if r.taken(w, c.Plugin, v.manifest.Storage.Namespace, now) {
		return refuse(ReasonNamespaceTaken)
	}
	if next := v.manifest.Storage; i != nil && held(i, now) && (next.Namespace != i.disposal.Namespace || next.RetentionDays < i.disposal.RetentionDays) {
		return refuse(ReasonStorageChanged)
	}
	count := 0
	for _, other := range w.installations {
		if other.state != StateRemoved {
			count++
		}
	}
	if count >= r.limits.Installations {
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkSwitch(c Change) error {
	w, v, err := r.target(c)
	if err != nil {
		return err
	}
	i, err := r.live(c.Workspace.ID, c.Plugin)
	if err != nil {
		return err
	}
	order := compare(c.Version, i.version)
	if c.Kind == ChangeUpgrade && order <= 0 || c.Kind == ChangeRollback && order >= 0 {
		return refuse(ReasonDirection)
	}
	if reason := r.runnable(w, v); reason != "" {
		return refuse(reason)
	}
	old, next := i.disposal, v.manifest.Storage
	if old.Namespace != next.Namespace || old.Deletion != next.Deletion || next.RetentionDays < old.RetentionDays {
		return refuse(ReasonStorageChanged)
	}
	return nil
}

func (r *Registry) checkEnable(c Change) error {
	i, err := r.live(c.Workspace.ID, c.Plugin)
	if err != nil {
		return err
	}
	if i.state != StateInstalled && i.state != StateStopped {
		return refuse(ReasonNotAllowedState)
	}
	if reason := r.runnable(r.workspaces[c.Workspace.ID], r.plugins[c.Plugin].versions[i.version]); reason != "" {
		return refuse(reason)
	}
	return nil
}

func (r *Registry) checkEndpoint(c Change) error {
	i, err := r.live(c.Workspace.ID, c.Plugin)
	if err != nil {
		return err
	}
	w := r.workspaces[c.Workspace.ID]
	current := i.endpoints[c.Endpoint.ID]
	switch {
	case w.view.Verification:
		return refuse(ReasonIsolated)
	case !declared(r.plugins[c.Plugin].versions[i.version].manifest.Endpoints, c.Endpoint.ID):
		return refuse(ReasonNotDeclared)
	case r.fingerprints[c.Endpoint.Fingerprint]:
		return ErrConflict
	case current == nil && len(i.endpoints) >= r.limits.Endpoints || w.fingerprints >= r.limits.Fingerprints:
		return ErrLimit
	}
	return nil
}

func declared(list []string, id string) bool {
	for _, value := range list {
		if value == id {
			return true
		}
	}
	return false
}

func (r *Registry) apply(c Change, actor Actor, now time.Time) {
	switch c.Kind {
	case ChangePublisher:
		r.publishers[c.Publisher.ID] = &publisher{view: c.Publisher, keys: map[string]*key{}, names: map[string]bool{}}
	case ChangeKey:
		p := r.publishers[c.Key.Publisher]
		p.keys[c.Key.Version] = &key{view: KeyView{Version: c.Key.Version, Public: append([]byte(nil), c.Key.Public...), NotBefore: c.Key.NotBefore, NotAfter: c.Key.NotAfter}}
		p.names[c.Key.Version] = true
	case ChangeVersion:
		p := r.plugins[c.Manifest.Plugin]
		if p == nil {
			p = &plugin{owner: c.Manifest.Publisher, versions: map[string]*version{}}
			r.plugins[c.Manifest.Plugin] = p
		}
		p.versions[c.Manifest.Version] = &version{manifest: copyManifest(c.Manifest), signature: append([]byte(nil), c.Signature...)}
		r.versions++
	case ChangeWorkspace:
		r.workspaces[c.Workspace.ID] = &workspace{view: c.Workspace, installations: map[string]*installation{}, runtimes: 1}
		r.runtimes[c.Workspace.Runtime] = c.Workspace.ID
	case ChangeRuntime:
		w := r.workspaces[c.Workspace.ID]
		w.view.Runtime = c.Workspace.Runtime
		w.runtimes++
		r.runtimes[c.Workspace.Runtime] = c.Workspace.ID
		for _, i := range w.installations {
			i.revision++
		}
	case ChangeInstall:
		r.applyInstall(c)
	case ChangeUpgrade, ChangeRollback:
		i := r.workspaces[c.Workspace.ID].installations[c.Plugin]
		i.version = c.Version
		i.revision++
		m := r.plugins[c.Plugin].versions[c.Version].manifest
		i.disposal = m.Storage
		for id := range i.endpoints {
			if !declared(m.Endpoints, id) {
				delete(i.endpoints, id)
			}
		}
	case ChangeEnable:
		i := r.workspaces[c.Workspace.ID].installations[c.Plugin]
		i.state, i.reason = StateEnabled, ""
		i.revision++
	case ChangeRemove:
		r.applyRemove(c, now)
	case ChangeEndpoint:
		w := r.workspaces[c.Workspace.ID]
		w.installations[c.Plugin].endpoints[c.Endpoint.ID] = &EndpointView{Endpoint: c.Endpoint, Approver: actor.ID}
		w.fingerprints++
		r.fingerprints[c.Endpoint.Fingerprint] = true
	}
}

func (r *Registry) applyInstall(c Change) {
	w := r.workspaces[c.Workspace.ID]
	i := w.installations[c.Plugin]
	if i == nil {
		i = &installation{plugin: c.Plugin}
		w.installations[c.Plugin] = i
	}
	i.version, i.state, i.reason = c.Version, StateInstalled, ""
	i.revision++
	i.endpoints, i.tokens = map[string]*EndpointView{}, map[[sha256.Size]byte]bool{}
	i.disposal = r.plugins[c.Plugin].versions[c.Version].manifest.Storage
}

func (r *Registry) applyRemove(c Change, now time.Time) {
	i := r.workspaces[c.Workspace.ID].installations[c.Plugin]
	i.state, i.reason, i.removed = StateRemoved, "", now
	i.revision++
	for hash := range i.tokens {
		delete(r.tokens, hash)
	}
	i.tokens, i.endpoints = map[[sha256.Size]byte]bool{}, map[string]*EndpointView{}
}
