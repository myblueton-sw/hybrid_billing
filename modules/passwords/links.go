package passwords

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"
)

func digest(token string) ([32]byte, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return [32]byte{}, false
	}
	return sha256.Sum256(raw), true
}

func (r *Registry) dropLink(k Key) {
	if d, ok := r.byKey[k]; ok {
		delete(r.links, d)
		delete(r.byKey, k)
	}
}

func (r *Registry) Issue(k Key, purpose string, validity time.Duration) (string, error) {
	if !validKey(k) || (purpose != PurposeSet && purpose != PurposeReset) || validity < r.params.MinValidity || validity > r.params.MaxValidity {
		return "", ErrInvalid
	}
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	now := r.clock()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dropLink(k)
	if len(r.links) >= r.params.Links {
		for d, l := range r.links {
			if !now.Before(l.view.Expires) || l.used {
				delete(r.links, d)
				delete(r.byKey, l.view.Key)
			}
		}
	}
	if len(r.links) >= r.params.Links {
		return "", ErrLimit
	}
	d := sha256.Sum256(raw)
	r.links[d] = &link{view: Link{Key: k, Purpose: purpose, Expires: now.Add(validity)}}
	r.byKey[k] = d
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (r *Registry) live(token string, now time.Time) (*link, [32]byte, bool) {
	d, ok := digest(token)
	if !ok {
		return nil, d, false
	}
	found := r.links[d]
	if found == nil || found.used || now.Before(found.reserved) || !now.Before(found.view.Expires) {
		return nil, d, false
	}
	return found, d, true
}

func (r *Registry) Peek(token string) (Link, error) {
	now := r.clock()
	r.mu.Lock()
	defer r.mu.Unlock()
	found, _, ok := r.live(token, now)
	if !ok {
		return Link{}, Refusal{Reason: ReasonLinkInvalid}
	}
	return found.view, nil
}

func (r *Registry) Reserve(token string) (Link, error) {
	now := r.clock()
	r.mu.Lock()
	defer r.mu.Unlock()
	found, _, ok := r.live(token, now)
	if !ok {
		return Link{}, Refusal{Reason: ReasonLinkInvalid}
	}
	found.reserved = now.Add(reservation)
	view := found.view
	view.Hold = found.reserved
	return view, nil
}

const reservation = 5 * time.Minute

func (r *Registry) reserved(token string, now time.Time) (*link, bool) {
	d, ok := digest(token)
	if !ok {
		return nil, false
	}
	found := r.links[d]
	return found, found != nil && now.Before(found.reserved) && !found.used
}

func (r *Registry) Release(token string, hold time.Time) {
	now := r.clock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if found, ok := r.reserved(token, now); ok && found.reserved.Equal(hold) {
		found.reserved = time.Time{}
	}
}

func (r *Registry) Commit(token string, record Record) (Key, error) {
	if !r.validRecord(record) {
		return Key{}, ErrInvalid
	}
	now := r.clock()
	r.mu.Lock()
	defer r.mu.Unlock()
	found, ok := r.reserved(token, now)
	if !ok {
		return Key{}, Refusal{Reason: ReasonLinkInvalid}
	}
	k := found.view.Key
	if r.accounts[k] == nil && len(r.accounts) >= r.params.Records {
		return Key{}, ErrLimit
	}
	found.used = true
	r.store(k, record)
	r.dropLink(k)
	return k, nil
}
