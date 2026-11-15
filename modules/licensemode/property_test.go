package licensemode

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

type modelEntitlement struct {
	seq                                int
	id, subject, agreement, supersedes string
	start, end, issued                 int64
	compromised                        bool
	ended                              bool
}

func TestLicenseProperty(t *testing.T) {
	runs := 150
	if testing.Short() {
		runs = 30
	}
	reached := map[string]int{}
	for seed := 0; seed < runs; seed++ {
		runLicenseProperty(t, int64(seed), reached)
	}
	for _, name := range []string{OutcomeBadSupersedes, OutcomeAccepted, OutcomeDuplicateRight, OutcomeSuperseded, OutcomeExpired, OutcomeNotYetValid, OutcomeInvalid, OutcomeUnverifiable,
		StateCommercialActive, StateCommercialExpired, StateUnverified, StateCommercialPending} {
		if reached[name] == 0 {
			t.Errorf("%s never reached", name)
		}
	}
	t.Logf("reached %v", reached)
}

func runLicenseProperty(t *testing.T, seed int64, reached map[string]int) {
	rng := rand.New(rand.NewSource(seed))
	f := standard(t)
	f.approve(Change{Kind: ChangeKey, Key: keySpec(f, "k2", "owner")})
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	model := []*modelEntitlement{}
	keyOf := map[string]string{}
	compromisedKeys := map[string]bool{}
	hour := int64(3600)
	now := start.Unix()
	high := now
	clockOK := true
	status(t, f, "i1")
	for step := 0; step < 30; step++ {
		switch rng.Intn(8) {
		case 0:
			d := time.Duration(rng.Intn(48)) * time.Hour
			f.advance(d)
			now += int64(d / time.Second)
			high, clockOK = now, true
		case 1:
			if rng.Intn(5) == 0 && !compromisedKeys["k2"] {
				f.approve(Change{Kind: ChangeCompromise, Compromise: "k2"})
				compromisedKeys["k2"] = true
				for _, e := range model {
					if keyOf[e.id] == "k2" {
						e.compromised = true
					}
				}
			}
		case 2:
			if rng.Intn(3) == 0 {
				f.auth.set(time.Unix(high-hour, 0), true)
				clockOK = false
			}
		default:
			if !clockOK {
				f.auth.set(time.Unix(now, 0), true)
				clockOK = true
			}
			id := "e" + strconv.Itoa(step)
			subject := []string{"acme", "beta"}[rng.Intn(2)]
			from := now + int64(rng.Intn(5)-3)*hour*24
			to := from + int64(1+rng.Intn(10))*hour*24
			agreements := []string{"agr-1", "agr-2", "agr-3"}
			c := commercial(id, agreements[rng.Intn(3)], time.Unix(from, 0), time.Unix(to, 0))
			c.SubjectID = subject
			c.IssuedAt = now - int64(rng.Intn(3600))
			c.KeyID = []string{"k1", "k2"}[rng.Intn(2)]
			if rng.Intn(3) == 0 {
				c.Supersedes = agreements[rng.Intn(3)]
			}
			doc := f.sign(c)
			want := ""
			switch rng.Intn(6) {
			case 0:
				doc.Signature[0] ^= 1
				want = OutcomeInvalid
			case 1:
				c.KeyID = "k9"
				doc = Signed{Body: Encode(c), Signature: make([]byte, 64)}
				want = OutcomeUnverifiable
			}
			if want == "" {
				want = modelRegister(model, c, now, compromisedKeys)
			}
			got, err := f.r.Register("i1", doc, ann)
			f.must(err)
			reached[got]++
			if got != want {
				t.Fatalf("seed %d step %d register %+v: %s, model %s", seed, step, c, got, want)
			}
			if got == OutcomeAccepted {
				model = append(model, &modelEntitlement{seq: len(model) + 1, id: id, subject: subject, agreement: c.Agreement, supersedes: c.Supersedes, start: from, end: to, issued: c.IssuedAt})
				keyOf[id] = c.KeyID
			}
		}
		v := status(t, f, "i1")
		want := StateUnverified
		if clockOK {
			for _, e := range model {
				if e.end <= now {
					e.ended = true
				}
			}
			want = modelState(model, now)
		}
		reached[v.State]++
		if v.State != want {
			t.Fatalf("seed %d step %d state %s, model %s", seed, step, v.State, want)
		}
	}
}

func live(model []*modelEntitlement) []*modelEntitlement {
	out := []*modelEntitlement{}
	for _, e := range model {
		if !e.compromised {
			out = append(out, e)
		}
	}
	return out
}

func effEnd(e *modelEntitlement, all []*modelEntitlement) int64 {
	out := e.end
	for _, s := range all {
		later := s.seq > e.seq
		if later && s.supersedes == e.agreement && s.subject == e.subject {
			cut := s.start
			if e.start > cut {
				cut = e.start
			}
			if cut < out {
				out = cut
			}
		}
	}
	return out
}

func currentOf(model []*modelEntitlement, now int64) (*modelEntitlement, bool) {
	l := live(model)
	var best *modelEntitlement
	for _, e := range l {
		if !e.ended && e.start <= now && now < effEnd(e, l) && (best == nil || e.issued > best.issued) {
			best = e
		}
	}
	if best != nil {
		return best, true
	}
	for _, e := range l {
		if best == nil || e.issued > best.issued {
			best = e
		}
	}
	return best, false
}

func modelRegister(model []*modelEntitlement, c Claims, now int64, compromised map[string]bool) string {
	if compromised[c.KeyID] {
		return OutcomeUnverifiable
	}
	switch {
	case c.Start > now:
		return OutcomeNotYetValid
	case c.End <= now:
		return OutcomeExpired
	}
	if cur, active := currentOf(model, now); active && c.IssuedAt <= cur.issued {
		return OutcomeSuperseded
	}
	l := live(model)
	if c.Supersedes != "" {
		found := false
		for _, e := range l {
			found = found || e.agreement == c.Supersedes && e.subject == c.SubjectID
		}
		if !found {
			return OutcomeBadSupersedes
		}
	}
	for _, e := range l {
		if e.subject == c.SubjectID && e.agreement != c.Supersedes && e.start < c.End && c.Start < effEnd(e, l) {
			return OutcomeDuplicateRight
		}
	}
	return OutcomeAccepted
}

func modelState(model []*modelEntitlement, now int64) string {
	cur, active := currentOf(model, now)
	switch {
	case cur == nil && len(model) > 0:
		return StateUnverified
	case cur == nil:
		return StateCommercialPending
	case active:
		return StateCommercialActive
	}
	for _, e := range model {
		if e.compromised && !e.ended && e.start <= now && now < e.end {
			return StateUnverified
		}
	}
	return StateCommercialExpired
}
