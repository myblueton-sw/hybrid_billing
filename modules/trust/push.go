package trust

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"strconv"
	"time"
)

var (
	fieldPrime = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))
	smallOrder = func() [][]byte {
		var out [][]byte
		for _, text := range []string{
			"0100000000000000000000000000000000000000000000000000000000000000",
			"0100000000000000000000000000000000000000000000000000000000000080",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"0000000000000000000000000000000000000000000000000000000000000080",
			"ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
			"ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc05",
			"26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85",
			"c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a",
			"c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac03fa",
		} {
			decoded, _ := hex.DecodeString(text)
			out = append(out, decoded)
		}
		return out
	}()
)

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

func copyKey(k Key) Key {
	k.Public = append([]byte(nil), k.Public...)
	return k
}

func viewKey(k *key) KeyView {
	v := k.view
	v.Key = copyKey(v.Key)
	return v
}

func sameKey(a, b Key) bool {
	return a.Version == b.Version && bytes.Equal(a.Public, b.Public) && a.NotBefore.Equal(b.NotBefore) && a.NotAfter.Equal(b.NotAfter)
}

func (r *Registry) live(workspace, id string) (*connection, error) {
	_, c, err := r.find(workspace, id)
	if err != nil {
		return nil, err
	}
	if c.state != StateActive {
		return nil, refuse(ReasonNotActive)
	}
	return c, nil
}

func (r *Registry) AddKey(workspace, connectionID string, k Key, actor Actor) (KeyView, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor, connectionID, k.Version); err != nil || k.NotBefore.IsZero() || !k.NotBefore.Before(k.NotAfter) {
		return KeyView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return KeyView{}, err
	}
	c, err := r.live(workspace, connectionID)
	if err != nil {
		return KeyView{}, err
	}
	k.NotBefore, k.NotAfter = k.NotBefore.UTC().Round(0), k.NotAfter.UTC().Round(0)
	if found := c.keys[k.Version]; found != nil {
		if found.view.State == StateProposed && found.view.Proposer == actor.ID && sameKey(found.view.Key, k) {
			return viewKey(found), nil
		}
		return KeyView{}, ErrConflict
	}
	switch {
	case weak(k.Public):
		return KeyView{}, refuse(ReasonWeakKey)
	case k.NotAfter.Sub(k.NotBefore) > r.limits.KeyLife:
		return KeyView{}, refuse(ReasonKeyLife)
	}
	open := 0
	for _, found := range c.keys {
		if found.view.State != StateRetired {
			open++
		}
	}
	if open >= r.limits.Keys {
		return KeyView{}, ErrLimit
	}
	made := &key{view: KeyView{Key: copyKey(k), State: StateProposed, Proposer: actor.ID}}
	c.keys[k.Version] = made
	return viewKey(made), nil
}

func (r *Registry) keyOf(workspace, connectionID, version string, actor Actor) (*key, error) {
	if err := r.configure(workspace, actor, connectionID, version); err != nil {
		return nil, err
	}
	if err := actor.human(); err != nil {
		return nil, err
	}
	_, c, err := r.find(workspace, connectionID)
	if err != nil {
		return nil, err
	}
	if c.keys[version] == nil {
		return nil, ErrNotFound
	}
	return c.keys[version], nil
}

func (r *Registry) ApproveKey(workspace, connectionID, version string, actor Actor) (KeyView, error) {
	r.enter()
	defer r.leave()
	k, err := r.keyOf(workspace, connectionID, version, actor)
	if err != nil {
		return KeyView{}, err
	}
	switch {
	case k.view.State == StateActive && k.view.Approver == actor.ID:
		return viewKey(k), nil
	case k.view.State != StateProposed:
		return KeyView{}, refuse(ReasonNotPending)
	case k.view.Proposer == actor.ID:
		return KeyView{}, refuse(ReasonSelfApproval)
	}
	k.view.State, k.view.Approver = StateActive, actor.ID
	return viewKey(k), nil
}

func (r *Registry) RetireKey(workspace, connectionID, version string, actor Actor) (KeyView, error) {
	r.enter()
	defer r.leave()
	k, err := r.keyOf(workspace, connectionID, version, actor)
	if err != nil {
		return KeyView{}, err
	}
	if k.view.State != StateRetired {
		r.retiredKeys++
		k.view.State, k.retired = StateRetired, r.retiredKeys
		r.prune(workspace, connectionID)
	}
	return viewKey(k), nil
}

func (r *Registry) prune(workspace, connectionID string) {
	c := r.workspaces[workspace].connections[connectionID]
	oldest, count := "", 0
	for version, k := range c.keys {
		if k.view.State != StateRetired {
			continue
		}
		count++
		if oldest == "" || k.retired < c.keys[oldest].retired {
			oldest = version
		}
	}
	if count > r.limits.Keys {
		delete(c.keys, oldest)
	}
}

func part(out *bytes.Buffer, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	out.Write(size[:])
	out.Write(value)
}

func Message(workspace, connectionID string, p Push) []byte {
	var out bytes.Buffer
	digest := sha256.Sum256(p.Body)
	part(&out, []byte(pushTag))
	part(&out, []byte(workspace))
	part(&out, []byte(connectionID))
	part(&out, []byte(p.KeyVersion))
	part(&out, []byte(strconv.Itoa(len(p.EventIDs))))
	for _, id := range p.EventIDs {
		part(&out, []byte(id))
	}
	part(&out, []byte(strconv.FormatInt(p.Signed.UnixNano(), 10)))
	part(&out, digest[:])
	return out.Bytes()
}

func (r *Registry) pushForm(a Admission, p Push) bool {
	if !validID(a.Workspace) || !validID(a.Connection) || a.Revision < 1 || len(p.EventIDs) < 1 || len(p.EventIDs) > r.limits.Items || !validID(p.KeyVersion) ||
		p.Signed.IsZero() || len(p.Body) > r.limits.Body || len(p.Signature) != signatureBytes {
		return false
	}
	ids := map[string]bool{}
	for _, id := range p.EventIDs {
		if !validID(id) || ids[id] {
			return false
		}
		ids[id] = true
	}
	return true
}

func (r *Registry) Verify(a Admission, p Push) error {
	now := r.enter()
	defer r.leave()
	if !r.pushForm(a, p) {
		return ErrInvalid
	}
	s, c, err := r.find(a.Workspace, a.Connection)
	if err != nil {
		return err
	}
	k := c.keys[p.KeyVersion]
	signed := p.Signed.UTC().Round(0)
	digest := sha256.Sum256(p.Body)
	early := ""
	switch {
	case c.state != StateActive || c.revision != a.Revision:
		early = ReasonConnectionChanged
	case k == nil || k.view.State != StateActive:
		early = ReasonUnknownKey
	case !ed25519.Verify(k.view.Key.Public, Message(a.Workspace, a.Connection, p), p.Signature):
		early = ReasonForged
	}
	if early != "" {
		s.counts[early]++
		return refuse(early)
	}
	reason := ""
	switch {
	case signed.Before(k.view.Key.NotBefore) || signed.After(k.view.Key.NotAfter):
		reason = ReasonKeyExpired
	case now.Sub(signed) > r.limits.Skew || signed.Sub(now) > r.limits.Skew:
		reason = ReasonStale
	default:
		reason = r.replay(c, p.EventIDs, digest, signed, now)
	}
	switch reason {
	case "":
		return nil
	case ReasonAlreadyAccepted:
		s.counts[reason]++
		return refuse(reason)
	}
	if len(s.quarantine) >= r.limits.Quarantine {
		s.dropped++
	} else {
		s.quarantine = append(s.quarantine, Entry{Sequence: len(s.quarantine) + 1, Connection: a.Connection, EventIDs: append([]string(nil), p.EventIDs...),
			KeyVersion: p.KeyVersion, Reason: reason, Signed: signed, Received: now, BodyDigest: digest})
	}
	return refuse(reason)
}

func (r *Registry) replay(c *connection, ids []string, digest [32]byte, signed, now time.Time) string {
	same, overlap := 0, 0
	for _, id := range ids {
		found, ok := c.seen[id]
		if !ok || now.After(found.expires) {
			continue
		}
		overlap++
		if found.digest == digest {
			same++
		}
	}
	switch {
	case same == len(ids):
		return ReasonAlreadyAccepted
	case overlap > 0:
		return ReasonDuplicate
	}
	if len(c.seen)+len(ids) > r.limits.Events {
		for id, found := range c.seen {
			if now.After(found.expires) {
				delete(c.seen, id)
			}
		}
		if len(c.seen)+len(ids) > r.limits.Events {
			return ReasonReplayCapacity
		}
	}
	for _, id := range ids {
		c.seen[id] = seen{digest: digest, expires: signed.Add(r.limits.Skew)}
	}
	return ""
}

func (r *Registry) Quarantine(workspace string, after, limit int, actor Actor) ([]Entry, int, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor); err != nil || after < 0 || limit < 1 {
		return nil, 0, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return nil, 0, err
	}
	s := r.workspaces[workspace]
	if s == nil {
		return nil, 0, nil
	}
	var out []Entry
	for index := after; index < len(s.quarantine) && len(out) < limit; index++ {
		entry := s.quarantine[index]
		entry.EventIDs = append([]string(nil), entry.EventIDs...)
		out = append(out, entry)
	}
	return out, s.dropped, nil
}

func (r *Registry) Counts(workspace string, actor Actor) (map[string]int, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor); err != nil {
		return nil, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return nil, err
	}
	out := map[string]int{}
	if s := r.workspaces[workspace]; s != nil {
		for reason, count := range s.counts {
			out[reason] = count
		}
	}
	return out, nil
}
