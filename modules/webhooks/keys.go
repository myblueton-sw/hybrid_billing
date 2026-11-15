package webhooks

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"math/big"
	"time"
)

var (
	fieldPrime = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))
	smallOrder = [][]byte{
		hexBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		hexBytes("0100000000000000000000000000000000000000000000000000000000000080"),
		hexBytes("0000000000000000000000000000000000000000000000000000000000000000"),
		hexBytes("0000000000000000000000000000000000000000000000000000000000000080"),
		hexBytes("ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f"),
		hexBytes("ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		hexBytes("26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc05"),
		hexBytes("26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85"),
		hexBytes("c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a"),
		hexBytes("c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac03fa"),
	}
)

func hexBytes(text string) []byte {
	out := make([]byte, len(text)/2)
	for index := range out {
		out[index] = nibble(text[2*index])<<4 | nibble(text[2*index+1])
	}
	return out
}

func nibble(c byte) byte {
	if c >= 'a' {
		return c - 'a' + 10
	}
	return c - '0'
}

func weak(public []byte) bool {
	if len(public) != ed25519.PublicKeySize {
		return true
	}
	for _, known := range smallOrder {
		if bytes.Equal(public, known) {
			return true
		}
	}
	reversed := make([]byte, len(public))
	for index, b := range public {
		reversed[len(public)-1-index] = b
	}
	reversed[0] &= 0x7f
	return new(big.Int).SetBytes(reversed).Cmp(fieldPrime) >= 0
}

func copyKey(k KeyView) KeyView {
	k.Key.Public = append([]byte(nil), k.Key.Public...)
	return k
}

func challenge(workspace, subscription, version string) []byte {
	var out bytes.Buffer
	for _, value := range []string{challengeTag, workspace, subscription, version} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		out.Write(size[:])
		out.WriteString(value)
	}
	return out.Bytes()
}

func (r *Registry) ProposeKey(workspace, id string, k Key, actor Actor) (KeyView, error) {
	r.enter()
	defer r.leave()
	_, s, err := r.find(workspace, id, actor)
	if err != nil {
		return KeyView{}, err
	}
	if !validID(k.Version) || k.NotBefore.IsZero() || !k.NotBefore.Before(k.NotAfter) {
		return KeyView{}, ErrInvalid
	}
	k.NotBefore, k.NotAfter = k.NotBefore.UTC().Round(0), k.NotAfter.UTC().Round(0)
	if found := s.keys[k.Version]; found != nil && found.State == KeyProposed && found.Proposer == actor.ID && bytes.Equal(found.Key.Public, k.Public) &&
		found.Key.NotBefore.Equal(k.NotBefore) && found.Key.NotAfter.Equal(k.NotAfter) {
		return copyKey(*found), nil
	}
	switch {
	case s.names[k.Version] || s.keys[k.Version] != nil:
		return KeyView{}, ErrConflict
	case s.view.State == SubscriptionRevoked:
		return KeyView{}, refuse(ReasonRevoked)
	case weak(k.Public):
		return KeyView{}, refuse(ReasonWeakKey)
	case k.NotAfter.Sub(k.NotBefore) > r.limits.KeyLife:
		return KeyView{}, refuse(ReasonKeyLife)
	case len(s.names) >= r.limits.Keys*namesPerKey:
		return KeyView{}, ErrLimit
	}
	proposed := 0
	for _, other := range s.keys {
		if other.State == KeyProposed {
			proposed++
		}
	}
	if proposed >= r.limits.Keys {
		return KeyView{}, ErrLimit
	}
	k.Public = append([]byte(nil), k.Public...)
	made := &KeyView{Key: k, State: KeyProposed, Proposer: actor.ID}
	s.keys[k.Version] = made
	return copyKey(*made), nil
}

func (r *Registry) ApproveKey(workspace, id, version string, actor Actor) (KeyView, error) {
	r.enter()
	_, s, err := r.find(workspace, id, actor)
	if err == nil && !validID(version) {
		err = ErrInvalid
	}
	if err == nil && s.keys[version] == nil {
		err = ErrNotFound
	}
	if err != nil {
		r.leave()
		return KeyView{}, err
	}
	k := s.keys[version]
	switch {
	case k.State == KeyActive && k.Approver == actor.ID:
		defer r.leave()
		return copyKey(*k), nil
	case k.State != KeyProposed:
		r.leave()
		return KeyView{}, refuse(ReasonNotPending)
	case k.Proposer == actor.ID:
		r.leave()
		return KeyView{}, refuse(ReasonSelfApproval)
	}
	public := append([]byte(nil), k.Key.Public...)
	r.leave()
	proof, err := r.config.Signer.Sign(workspace, id, version, challenge(workspace, id, version))
	if err != nil {
		return KeyView{}, refuse(ReasonDependency)
	}
	if !ed25519.Verify(public, challenge(workspace, id, version), proof) {
		return KeyView{}, refuse(ReasonKeyMismatch)
	}
	now := r.enter()
	defer r.leave()
	if s.keys[version] != k || k.State != KeyProposed || s.view.State == SubscriptionRevoked {
		return KeyView{}, ErrStale
	}
	published := 0
	for _, other := range s.keys {
		if other.State == KeyActive && !now.After(other.Key.NotAfter) {
			published++
		}
	}
	if published >= r.limits.Keys {
		return KeyView{}, ErrLimit
	}
	k.State, k.Approver = KeyActive, actor.ID
	s.names[version] = true
	return copyKey(*k), nil
}

func (r *Registry) RetireKey(workspace, id, version string, actor Actor) (KeyView, error) {
	r.enter()
	defer r.leave()
	_, s, err := r.find(workspace, id, actor)
	if err != nil {
		return KeyView{}, err
	}
	if !validID(version) {
		return KeyView{}, ErrInvalid
	}
	k := s.keys[version]
	if k == nil {
		return KeyView{}, ErrNotFound
	}
	if k.State == KeyProposed {
		delete(s.keys, version)
	}
	k.State = KeyRetired
	return copyKey(*k), nil
}

func (r *Registry) Keys(workspace, id string, actor Actor) ([]KeyView, error) {
	r.enter()
	defer r.leave()
	_, s, err := r.listed(workspace, id, actor)
	if err != nil {
		return nil, err
	}
	out := []KeyView{}
	if s.view.State == SubscriptionRevoked {
		return out, nil
	}
	for _, k := range s.keys {
		if k.State == KeyActive {
			out = append(out, copyKey(*k))
		}
	}
	sortKeys(out)
	return out, nil
}

func sortKeys(keys []KeyView) {
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j].Key.Version < keys[j-1].Key.Version; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
}

func published(s *subscription, version string, at time.Time) []byte {
	k := s.keys[version]
	if k == nil || k.State != KeyActive || at.Before(k.Key.NotBefore) || at.After(k.Key.NotAfter) {
		return nil
	}
	return k.Key.Public
}
