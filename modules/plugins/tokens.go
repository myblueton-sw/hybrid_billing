package plugins

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"time"
)

const (
	tokenPrefix = "hbp_"
	tokenBytes  = 32
	tokenTag    = "hybrid-billing/plugin-token/v1"
)

func digest(text string) [sha256.Size]byte {
	var e encoder
	e.text(tokenTag)
	e.text(text)
	return sha256.Sum256(e.out.Bytes())
}

func validToken(text string) bool {
	if len(text) != len(tokenPrefix)+base64.RawURLEncoding.EncodedLen(tokenBytes) || text[:len(tokenPrefix)] != tokenPrefix {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(text[len(tokenPrefix):])
	return err == nil && base64.RawURLEncoding.EncodeToString(raw) == text[len(tokenPrefix):]
}

func validScopes(scopes []Capability) bool {
	if len(scopes) == 0 || len(scopes) > maxCapabilities {
		return false
	}
	for index, c := range scopes {
		if !capabilityKind(c.Kind) || !validID(c.Target) || index > 0 && !less(scopes[index-1], c) {
			return false
		}
	}
	return true
}

func (r *Registry) runtime(workspace, plugin string, actor Actor) (*workspace, *installation, *version, error) {
	if !actor.valid() || !validID(workspace) || !validID(plugin) {
		return nil, nil, nil, ErrInvalid
	}
	w := r.workspaces[workspace]
	if w == nil {
		return nil, nil, nil, ErrNotFound
	}
	if actor.Kind != KindSystem || actor.ID != w.view.Runtime {
		return nil, nil, nil, refuse(ReasonNotRuntime)
	}
	i := w.installations[plugin]
	if i == nil {
		return nil, nil, nil, ErrNotFound
	}
	if i.state != StateEnabled {
		return nil, nil, nil, refuse(ReasonNotEnabled)
	}
	v := r.plugins[plugin].versions[i.version]
	if reason := r.runnable(w, v); reason != "" {
		return nil, nil, nil, refuse(reason)
	}
	return w, i, v, nil
}

func (r *Registry) prune(i *installation, now time.Time) {
	for hash := range i.tokens {
		if t := r.tokens[hash]; !now.Before(t.expiry) || t.revision != i.revision {
			delete(r.tokens, hash)
			delete(i.tokens, hash)
		}
	}
}

func (r *Registry) IssueToken(workspace, plugin string, scopes []Capability, ttl time.Duration, actor Actor) (string, Grant, error) {
	now := r.enter()
	defer r.leave()
	if !validScopes(scopes) || ttl < minTokenLife || ttl > r.limits.TokenLife {
		return "", Grant{}, ErrInvalid
	}
	w, i, v, err := r.runtime(workspace, plugin, actor)
	if err != nil {
		return "", Grant{}, err
	}
	for _, c := range scopes {
		switch {
		case !has(v.manifest.Capabilities, c):
			return "", Grant{}, refuse(ReasonScope)
		case c.Kind == CapabilityCallCommand && w.view.Verification:
			return "", Grant{}, refuse(ReasonIsolated)
		}
	}
	r.prune(i, now)
	if len(i.tokens) >= r.limits.Tokens {
		return "", Grant{}, ErrLimit
	}
	raw := make([]byte, tokenBytes)
	if _, err := io.ReadFull(r.config.Random, raw); err != nil {
		return "", Grant{}, refuse(ReasonDependency)
	}
	text := tokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	hash := digest(text)
	if r.tokens[hash] != nil {
		return "", Grant{}, refuse(ReasonDependency)
	}
	t := &issued{workspace: workspace, plugin: plugin, version: i.version, digest: v.manifest.Digest, revision: i.revision,
		scopes: append([]Capability(nil), scopes...), expiry: now.Add(ttl)}
	r.tokens[hash] = t
	i.tokens[hash] = true
	return text, grant(t), nil
}

func grant(t *issued) Grant {
	return Grant{Workspace: t.workspace, Plugin: t.plugin, Version: t.version, Digest: t.digest, Revision: t.revision, Scopes: append([]Capability(nil), t.scopes...)}
}

func (r *Registry) CheckToken(text string, scope Capability) (Grant, error) {
	now := r.enter()
	defer r.leave()
	if !validToken(text) || !capabilityKind(scope.Kind) || !validID(scope.Target) {
		return Grant{}, ErrInvalid
	}
	hash := digest(text)
	t := r.tokens[hash]
	if t == nil {
		return Grant{}, refuse(ReasonUnknownToken)
	}
	w := r.workspaces[t.workspace]
	i := w.installations[t.plugin]
	forget := func(reason string) (Grant, error) {
		delete(r.tokens, hash)
		delete(i.tokens, hash)
		return Grant{}, refuse(reason)
	}
	switch {
	case !now.Before(t.expiry):
		return forget(ReasonExpired)
	case i.revision != t.revision || i.state != StateEnabled || i.version != t.version:
		return forget(ReasonStaleToken)
	case r.runnable(w, r.plugins[t.plugin].versions[t.version]) != "":
		return forget(ReasonStaleToken)
	case !has(t.scopes, scope):
		return Grant{}, refuse(ReasonScope)
	}
	return grant(t), nil
}

func (r *Registry) CheckEndpoint(workspace, plugin, endpoint, fingerprint string, actor Actor) (EndpointView, error) {
	r.enter()
	defer r.leave()
	if !validID(endpoint) || !validHex(fingerprint) {
		return EndpointView{}, ErrInvalid
	}
	_, i, _, err := r.runtime(workspace, plugin, actor)
	if err != nil {
		return EndpointView{}, err
	}
	e := i.endpoints[endpoint]
	switch {
	case e == nil:
		return EndpointView{}, ErrNotFound
	case e.Endpoint.Fingerprint != fingerprint:
		return EndpointView{}, refuse(ReasonFingerprintMismatch)
	}
	return *e, nil
}

func (r *Registry) CheckEgress(workspace, plugin, destination string, actor Actor) error {
	r.enter()
	if !validID(destination) {
		r.leave()
		return ErrInvalid
	}
	w, i, v, err := r.runtime(workspace, plugin, actor)
	switch {
	case err != nil:
	case w.view.Verification:
		err = refuse(ReasonIsolated)
	case !declared(v.manifest.Egress, destination):
		err = refuse(ReasonNotDeclared)
	}
	if err != nil {
		r.leave()
		return err
	}
	revision := i.revision
	r.leave()
	active, err := r.config.Destinations.Active(workspace, destination)
	if err != nil {
		return refuse(ReasonDependency)
	}
	if !active {
		return refuse(ReasonDestinationInactive)
	}
	r.enter()
	defer r.leave()
	if i.revision != revision || i.state != StateEnabled || r.runnable(w, v) != "" {
		return ErrStale
	}
	return nil
}
