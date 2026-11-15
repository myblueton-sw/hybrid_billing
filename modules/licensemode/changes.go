package licensemode

import (
	"crypto/ed25519"
	"encoding/hex"
	"time"
)

func validTime(t int64) bool { return t >= 1 && t <= maxTime }

func (k Key) valid() bool {
	return validID(k.ID) && validID(k.Issuer) && validTime(k.NotBefore) && validTime(k.NotAfter) && validTime(k.AcceptUntil) && k.NotAfter > k.NotBefore &&
		k.AcceptUntil >= k.NotAfter && len(k.Public) == 2*ed25519.PublicKeySize && validHexBytes(k.Public) && len(k.Proof) == 2*ed25519.SignatureSize && validHexBytes(k.Proof)
}

func validHexBytes(text string) bool {
	_, err := hex.DecodeString(text)
	return err == nil && text == hexLower(text)
}

func hexLower(text string) string {
	out := []byte(text)
	for index, c := range out {
		if c >= 'A' && c <= 'F' {
			out[index] = c + 'a' - 'A'
		}
	}
	return string(out)
}

func clean(c Change) (Change, error) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangeKey:
		out.Key = c.Key
		if len(c.Key.Public) == 2*ed25519.PrivateKeySize {
			return out, refuse(ReasonPrivateKey)
		}
		if !c.Key.valid() {
			return out, ErrInvalid
		}
		return out, nil
	case ChangeRetire:
		out.Retire = c.Retire
		if validID(c.Retire.Key) && validTime(c.Retire.At) {
			return out, nil
		}
	case ChangeCompromise:
		out.Compromise = c.Compromise
		if validID(c.Compromise) {
			return out, nil
		}
	case ChangeReset:
		out.Reset = c.Reset
		if validTime(c.Reset.At) && validText(c.Reset.Reason) {
			return out, nil
		}
	}
	return out, ErrInvalid
}

func copyChange(c Change) Change { return c }

func copyProposal(p ProposalView) ProposalView { return p }

func encodeChange(c Change) []byte {
	var e encoder
	k := c.Key
	for _, value := range []string{c.Kind, k.ID, k.Issuer, k.Public, k.Proof, timeText(k.NotBefore), timeText(k.NotAfter), timeText(k.AcceptUntil), c.Retire.Key,
		timeText(c.Retire.At), c.Compromise, timeText(c.Reset.At), c.Reset.Reason} {
		e.text(value)
	}
	return e.out.Bytes()
}

func (r *Registry) check(c Change, _, _ string) error {
	switch c.Kind {
	case ChangeKey:
		public, _ := hex.DecodeString(c.Key.Public)
		proof, _ := hex.DecodeString(c.Key.Proof)
		switch {
		case r.keys[c.Key.ID] != nil:
			return ErrConflict
		case !ed25519.Verify(public, proofMessage(c.Key.ID, c.Key.Issuer), proof):
			return refuse(ReasonNoProof)
		case r.active >= maxKeys:
			return ErrLimit
		}
		return nil
	case ChangeCompromise:
		k := r.keys[c.Compromise]
		switch {
		case k == nil:
			return ErrNotFound
		case k.compromised:
			return refuse(ReasonCompromised)
		}
		return nil
	}
	if c.Kind == ChangeRetire {
		now, outcome := r.authoritative(r.now)
		if outcome != "" {
			return refuse(outcome)
		}
		k := r.keys[c.Retire.Key]
		switch {
		case k == nil:
			return ErrNotFound
		case k.compromised:
			return refuse(ReasonCompromised)
		case k.retired:
			return refuse(ReasonRetired)
		case c.Retire.At < now.Unix():
			return refuse(ReasonRetirePast)
		case c.Retire.At > k.view.NotAfter || c.Retire.At <= k.view.NotBefore:
			return ErrInvalid
		}
		return nil
	}
	for _, a := range r.grants {
		if c.Reset.At < a.view.Start.Unix() {
			return refuse(ReasonResetTooEarly)
		}
	}
	return nil
}

func (r *Registry) apply(c Change, _ people, _ time.Time) {
	switch c.Kind {
	case ChangeKey:
		public, _ := hex.DecodeString(c.Key.Public)
		r.keys[c.Key.ID] = &key{view: c.Key, public: public}
		r.active++
	case ChangeRetire:
		k := r.keys[c.Retire.Key]
		k.view.NotAfter, k.retired = c.Retire.At, true
		r.active--
	case ChangeCompromise:
		k := r.keys[c.Compromise]
		if !k.retired {
			r.active--
		}
		k.compromised = true
	case ChangeReset:
		r.high = time.Unix(c.Reset.At, 0).UTC()
	}
}
