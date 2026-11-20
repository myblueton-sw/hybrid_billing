package licensemode

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type checkpointGrant struct {
	View    Activation
	Expired bool
}
type checkpointDocument struct {
	Claims Claims
	Ended  bool
	Order  uint64
}
type checkpointInstallation struct {
	Mode        string
	Modes       []ModeChange
	Submissions []Submission
	Grant       string
	Documents   []string
	Last        string
	LastAt      time.Time
}
type checkpoint struct {
	Version      int
	Installation string
	Product      string
	Now          time.Time
	High         time.Time
	Counter      uint64
	Accepted     uint64
	Keys         map[string]KeyView
	Grants       map[string]checkpointGrant
	Documents    map[string]checkpointDocument
	Proposals    map[uint64]ProposalView
	Pending      []uint64
	Done         []uint64
	State        checkpointInstallation
}
type checkpointEnvelope struct {
	SHA256 string
	State  json.RawMessage
}

func makeCheckpoint(r *Registry, id, product string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inst := r.installations[id]
	if inst == nil || len(r.installations) != 1 || inst.product != product {
		return nil, ErrCheckpoint
	}
	c := checkpoint{Version: 1, Installation: id, Product: product, Now: r.now, High: r.high, Counter: r.counter, Accepted: r.accepted,
		Keys: map[string]KeyView{}, Grants: map[string]checkpointGrant{}, Documents: map[string]checkpointDocument{}, Proposals: map[uint64]ProposalView{},
		Pending: r.pending, Done: r.done, State: checkpointInstallation{Mode: inst.mode, Modes: inst.modes, Submissions: inst.submissions, Last: inst.last, LastAt: inst.lastAt}}
	for id, k := range r.keys {
		c.Keys[id] = KeyView{Key: k.view, Compromised: k.compromised, Retired: k.retired}
	}
	for id, a := range r.grants {
		c.Grants[id] = checkpointGrant{a.view, a.expired}
	}
	for id, e := range r.documents {
		c.Documents[id] = checkpointDocument{e.claims, e.ended, e.order}
	}
	for number, p := range r.proposals {
		c.Proposals[number] = p.view
	}
	if inst.activation != nil {
		c.State.Grant = inst.activation.view.Grant
	}
	for _, e := range inst.accepted {
		c.State.Documents = append(c.State.Documents, e.claims.ID)
	}
	if !validateCheckpoint(c) {
		return nil, ErrCheckpoint
	}
	state, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(state)
	out, err := json.Marshal(checkpointEnvelope{hex.EncodeToString(sum[:]), state})
	if len(out) > maxCheckpointBytes {
		return nil, ErrCheckpoint
	}
	return out, err
}

func canonicalJSON(data []byte, out any) bool {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil {
		return false
	}
	canonical, err := json.Marshal(out)
	return err == nil && bytes.Equal(canonical, data)
}

func restoreCheckpoint(r *Registry, data []byte, id, product string) error {
	if len(data) > maxCheckpointBytes {
		return ErrCheckpoint
	}
	var envelope checkpointEnvelope
	if !canonicalJSON(data, &envelope) {
		return ErrCheckpoint
	}
	sum := sha256.Sum256(envelope.State)
	if envelope.SHA256 != hex.EncodeToString(sum[:]) {
		return ErrCheckpoint
	}
	var c checkpoint
	if !canonicalJSON(envelope.State, &c) || c.Version != 1 || c.Installation != id || c.Product != product {
		return ErrCheckpoint
	}
	if !validateCheckpoint(c) {
		return ErrCheckpoint
	}
	r.now, r.high, r.counter, r.accepted = c.Now, c.High, c.Counter, c.Accepted
	for id, k := range c.Keys {
		public, _ := hex.DecodeString(k.Key.Public)
		r.keys[id] = &key{view: k.Key, public: public, compromised: k.Compromised, retired: k.Retired}
		if !k.Compromised && !k.Retired {
			r.active++
		}
	}
	for id, a := range c.Grants {
		r.grants[id] = &activation{view: a.View, expired: a.Expired}
	}
	for id, e := range c.Documents {
		r.documents[id] = &entitlement{claims: e.Claims, ended: e.Ended, order: e.Order}
	}
	for number, p := range c.Proposals {
		r.proposals[number] = &proposal{view: p, code: string(encodeChange(p.Change))}
	}
	r.pending, r.done = c.Pending, c.Done
	state := c.State
	inst := &installation{id: id, product: product, mode: state.Mode, modes: state.Modes, submissions: state.Submissions, last: state.Last, lastAt: state.LastAt}
	if state.Grant != "" {
		inst.activation = r.grants[state.Grant]
	}
	for _, id := range state.Documents {
		inst.accepted = append(inst.accepted, r.documents[id])
	}
	r.installations[id] = inst
	return nil
}
