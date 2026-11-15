package authpolicy

import (
	"math/rand"
	"testing"
	"time"
)

type modelPlan struct {
	users     []string
	chunk     int
	completed int
	lifted    bool
}

func pick(rng *rand.Rand, list []string, min int) []string {
	out := []string{}
	for _, item := range list {
		if rng.Intn(2) == 0 {
			out = append(out, item)
		}
	}
	for len(out) < min {
		out = append(out[:0], list[rng.Intn(len(list))])
	}
	return out
}

func modelDecide(rev *Revision, role string, a Authentication, action string, now time.Time) string {
	has := func(list []string, v string) bool {
		for _, x := range list {
			if x == v {
				return true
			}
		}
		return false
	}
	if rev == nil {
		return ReasonNoPolicy
	}
	mfa := false
	for _, f := range a.Factors {
		if f == FactorOTP || f == FactorWebAuthn || f == FactorIdP && (a.Method == MethodOIDC || a.Method == MethodSAML) {
			mfa = true
		}
	}
	age := now.Sub(a.At)
	switch {
	case !has(rev.Methods, a.Method):
		return ReasonMethodNotAllowed
	case len(rev.Issuers) > 0 && a.Method != MethodPassword && !has(rev.Issuers, a.Issuer):
		return ReasonIssuerNotAllowed
	case age < 0:
		return ReasonFutureAuthentication
	case age > rev.MaxAge:
		return ReasonStale
	case !mfa && (rev.MFA == MFAEveryone || rev.MFA == MFAAdministrators && role == RoleAdministrator):
		return ReasonMFARequired
	}
	for _, s := range rev.StepUp {
		if s.Action == action && age > s.MaxAge {
			return ReasonStepUpRequired
		}
		if s.Action == action && s.MFA && !mfa {
			return ReasonStepUpMFARequired
		}
	}
	return ""
}

func TestProperty(t *testing.T) {
	pool := []string{"u0", "u1", "u2", "u3", "u4", "u5", "u6", "u7"}
	for seed := int64(1); seed <= 150; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := setup(t)
		var rev *Revision
		version := uint64(0)
		blocked := map[string]int{}
		plans := map[string]*modelPlan{}
		ids := []string{}
		for step := 0; step < 60; step++ {
			switch op := rng.Intn(10); {
			case op == 0 || rev == nil:
				x := Revision{Workspace: "w1", Methods: pick(rng, []string{MethodLDAP, MethodPassword, MethodSAML}, 0), MFA: MFAScopes[rng.Intn(3)], MaxAge: time.Duration(2+rng.Intn(120)) * time.Minute}
				x.Methods = append(x.Methods, MethodOIDC)
				for i := len(x.Methods) - 1; i > 0 && x.Methods[i] < x.Methods[i-1]; i-- {
					x.Methods[i], x.Methods[i-1] = x.Methods[i-1], x.Methods[i]
				}
				if rng.Intn(2) == 0 {
					x.Issuers = []string{"idp", "zz"}
				}
				for _, action := range []string{"export", ActionPolicyChange, "view"} {
					if rng.Intn(2) == 0 {
						x.StepUp = append(x.StepUp, StepUp{Action: action, MaxAge: time.Duration(1+rng.Intn(int(x.MaxAge/time.Minute))) * time.Minute, MFA: rng.Intn(2) == 0})
					}
				}
				f.revise(x)
				rev = &x
				version++
			case op == 1:
				users := pick(rng, pool, 1)
				chunk := 1 + rng.Intn(3)
				f.by(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users, Chunk: chunk, Reason: "r"}}, cat, dan)
				id := "w1/" + itoa(len(ids)+1)
				ids = append(ids, id)
				plans[id] = &modelPlan{users: users, chunk: chunk}
				for _, u := range users {
					blocked[u]++
				}
				version++
			case op == 2 && len(ids) > 0:
				id := ids[rng.Intn(len(ids))]
				m := plans[id]
				err := f.try(Change{Kind: ChangeLift, Lift: Lift{Plan: id, Reason: "x"}})
				if m.lifted {
					reason(t, "lift twice", err, ReasonLifted)
					continue
				}
				f.must(err)
				m.lifted = true
				for _, u := range m.users {
					blocked[u]--
				}
				version++
			case op == 3 && len(ids) > 0:
				id := ids[rng.Intn(len(ids))]
				m := plans[id]
				count := (len(m.users) + m.chunk - 1) / m.chunk
				index := rng.Intn(count + 1)
				v, err := f.r.Complete(id, index, worker)
				switch {
				case index >= count:
					expect(t, "range", err, ErrInvalid)
				case m.lifted:
					reason(t, "lifted", err, ReasonLifted)
				case index > m.completed:
					reason(t, "order", err, ReasonOutOfOrder)
				default:
					if index == m.completed {
						m.completed++
					}
					if err != nil || v.Completed != m.completed || (v.State == PlanComplete) != (m.completed == count) {
						t.Fatalf("seed %d complete %+v %v", seed, v, err)
					}
				}
			case op == 4:
				f.advance(time.Duration(rng.Intn(30)) * time.Minute)
			default:
				user := pool[rng.Intn(len(pool))]
				role := []string{RoleMember, RoleAdministrator}[rng.Intn(2)]
				method := Methods[rng.Intn(4)]
				a := Authentication{Method: method, At: f.at().Add(-time.Duration(rng.Intn(150)-5) * time.Minute)}
				if method != MethodPassword {
					a.Issuer = []string{"idp", "other"}[rng.Intn(2)]
				}
				for _, factor := range Factors {
					if rng.Intn(3) == 0 && (factor != FactorIdP || method == MethodOIDC || method == MethodSAML) {
						a.Factors = append(a.Factors, factor)
					}
				}
				action := []string{"", "export", "view", ActionPolicyChange}[rng.Intn(4)]
				want := modelDecide(rev, role, a, action, f.at())
				if blocked[user] > 0 {
					want = ReasonBlocked
				}
				d, err := f.r.Evaluate(Principal{User: user, Workspace: "w1", Role: role, Kind: KindHuman}, a, action)
				var got string
				if err != nil {
					got = err.(Refusal).Reason
				}
				if got != want {
					t.Fatalf("seed %d step %d: got %q want %q for %+v %s %s", seed, step, got, want, a, action, role)
				}
				if err == nil && (d.Version != version || !f.r.Current("w1", version)) {
					t.Fatalf("seed %d version %d %d", seed, d.Version, version)
				}
			}
		}
	}
}
