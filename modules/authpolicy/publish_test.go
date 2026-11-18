package authpolicy

import (
	"errors"
	"math/rand/v2"
	"testing"
	"time"
)

func TestPublish(t *testing.T) {
	f := setup(t)
	good := Publication{Revision: open(), Epoch: 1<<32 + 1, Number: 7, Proposer: "ann", Validator: "ann", Approver: "bob", Activator: "cat"}
	bad := open()
	bad.MaxAge = 0
	for name, p := range map[string]Publication{
		"an invalid revision": {Revision: bad, Epoch: 1, Number: 1},
		"no epoch number":     {Revision: open(), Epoch: 1 << 32, Number: 1},
		"no policy number":    {Revision: open(), Epoch: 1, Number: 0},
		"an invalid actor":    {Revision: open(), Epoch: 1, Number: 1, Activator: "a b"},
	} {
		expect(t, name, second(f.r.Publish(p)), ErrInvalid)
		expect(t, name+" can publish", f.r.CanPublish(p), ErrInvalid)
	}
	f.must(f.r.CanPublish(good))
	v, err := f.r.Publish(good)
	f.must(err)
	if v.Epoch != good.Epoch || v.Number != 7 || v.Validator != "ann" || v.Activator != "cat" || v.Approver != "bob" || v.Proposer != "ann" || !v.At.Equal(start) {
		t.Errorf("published %+v", v)
	}
	if got, _ := f.r.Policy("w1"); got.Epoch != good.Epoch || got.Number != 7 {
		t.Errorf("policy %+v", got)
	}
	p, err := f.r.Propose(planOf("u1"), ann, f.strong())
	f.must(err)
	for name, next := range map[string]Publication{
		"the same epoch":       {Revision: open(), Epoch: good.Epoch, Number: 8},
		"an older epoch":       {Revision: open(), Epoch: 5, Number: 8},
		"the same number":      {Revision: open(), Epoch: good.Epoch + 1, Number: 7},
		"an older number":      {Revision: open(), Epoch: good.Epoch + 1, Number: 6},
		"anothers same epoch?": {Revision: open(), Epoch: good.Epoch, Number: 7},
	} {
		reason(t, name, second(f.r.Publish(next)), ReasonOutOfOrder)
		reason(t, name+" can publish", f.r.CanPublish(next), ReasonOutOfOrder)
	}
	if _, err := f.r.Proposal(p.Number); err != nil {
		t.Error("a refused publication dropped a pending proposal")
	}
	v2, err := f.r.Publish(Publication{Revision: open(), Epoch: good.Epoch + 1, Number: 8})
	f.must(err)
	if v2.Proposer != "" || v2.Activator != "" {
		t.Errorf("seed publication %+v", v2)
	}
	expect(t, "a publication drops pending proposals", second(f.r.Proposal(p.Number)), ErrNotFound)
	if EpochGeneration(good.Epoch) != 1 || EpochNumber(good.Epoch) != 1 || EpochNumber(1<<32-1) != 1<<32-1 || EpochGeneration(1<<32-1) != 0 {
		t.Error("epoch parts")
	}
}

func TestJudgeAndReachableAgreeWithEvaluate(t *testing.T) {
	f := standard(t)
	f.block([]string{"blocked"}, 1)
	tight := Revision{Workspace: "w1", Methods: []string{MethodOIDC, MethodPassword}, Issuers: []string{"idp"}, MFA: MFAAdministrators, MaxAge: time.Hour,
		StepUp: []StepUp{{Action: ActionPolicyChange, MaxAge: time.Minute, MFA: true}}}
	now := f.at()
	for name, c := range map[string]struct {
		p    Principal
		a    Authentication
		x    Revision
		want string
	}{
		"blocked":          {admin("blocked"), f.strong(), open(), ReasonBlocked},
		"method":           {ann, Authentication{Method: MethodSAML, Issuer: "idp", At: now}, tight, ReasonMethodNotAllowed},
		"issuer":           {ann, Authentication{Method: MethodOIDC, Issuer: "other", Factors: []string{FactorIdP}, At: now}, tight, ReasonIssuerNotAllowed},
		"future":           {ann, Authentication{Method: MethodPassword, At: now.Add(time.Second)}, tight, ReasonFutureAuthentication},
		"stale":            {ann, Authentication{Method: MethodPassword, At: now.Add(-2 * time.Hour)}, tight, ReasonStale},
		"mfa":              {ann, Authentication{Method: MethodPassword, At: now}, tight, ReasonMFARequired},
		"member without":   {member("m"), Authentication{Method: MethodPassword, At: now}, tight, ""},
		"step-up excluded": {ann, Authentication{Method: MethodPassword, Factors: []string{FactorOTP}, At: now.Add(-30 * time.Minute)}, tight, ""},
	} {
		err := f.r.Judge(c.p, c.a, c.x, now)
		if c.want == "" {
			f.must(err)
		} else {
			reason(t, name, err, c.want)
		}
	}
	reason(t, "assistant", f.r.Judge(Principal{User: "x", Workspace: "w1", Role: RoleAdministrator, Kind: KindAssistant}, f.weak(), open(), now), ReasonAINotAllowed)
	reason(t, "system", f.r.Judge(Principal{User: "x", Workspace: "w1", Role: RoleAdministrator, Kind: KindSystem}, f.weak(), open(), now), ReasonNotHuman)
	expect(t, "unknown kind", f.r.Judge(Principal{User: "x", Workspace: "w1", Role: RoleAdministrator, Kind: "robot"}, f.weak(), open(), now), ErrInvalid)
	expect(t, "invalid candidate", f.r.Judge(ann, f.weak(), Revision{Workspace: "w1"}, now), ErrInvalid)
	expect(t, "invalid authentication", f.r.Judge(ann, Authentication{Method: MethodPassword}, open(), now), ErrInvalid)
	other := open()
	other.Workspace = "w2"
	expect(t, "another workspace", f.r.Judge(ann, f.weak(), other, now), ErrInvalid)
	if !f.r.Blocked("w1", "blocked") || f.r.Blocked("w1", "ann") || f.r.Blocked("w2", "blocked") {
		t.Error("blocked")
	}
	rng := rand.New(rand.NewPCG(7, 11))
	pick := func(list []string) string { return list[rng.IntN(len(list))] }
	for n := 0; n < 4000; n++ {
		x := Revision{Workspace: "w1", MFA: pick(MFAScopes), MaxAge: time.Duration(1+rng.IntN(120)) * time.Minute}
		for _, m := range Methods {
			if rng.IntN(2) == 0 {
				x.Methods = append(x.Methods, m)
			}
		}
		if len(x.Methods) == 0 {
			x.Methods = []string{pick(Methods)}
		}
		if rng.IntN(2) == 0 {
			x.Issuers = []string{"idp"}
		}
		f.revise(x)
		p := Principal{User: pick([]string{"ann", "blocked", "m1"}), Workspace: "w1", Role: pick([]string{RoleAdministrator, RoleMember}), Kind: KindHuman}
		a := Authentication{Method: pick(Methods), At: f.at().Add(time.Duration(rng.IntN(150)-10) * time.Minute)}
		if a.Method != MethodPassword {
			a.Issuer = pick([]string{"idp", "other"})
		}
		if rng.IntN(2) == 0 {
			a.Factors = []string{FactorOTP}
		}
		_, want := f.r.Evaluate(p, a, "")
		got := f.r.Judge(p, a, x, f.at())
		if (want == nil) != (got == nil) || want != nil && want.Error() != got.Error() {
			t.Fatalf("%+v %+v %+v: evaluate %v judge %v", p, a, x, want, got)
		}
		fresh := Authentication{Method: MethodPassword, At: f.at()}
		reach := Reachable(p, x) && !f.r.Blocked("w1", p.User)
		if judged := f.r.Judge(p, fresh, x, f.at()); reach != (judged == nil) {
			t.Fatalf("%+v %+v: reachable %v judge %v", p, x, reach, judged)
		}
	}
}

func TestPublishRacesProposals(t *testing.T) {
	f := standard(t)
	done := make(chan error, 2)
	go func() {
		for n := 0; n < 50; n++ {
			f.r.Propose(planOf("r"+itoa(n)), ann, f.strong())
		}
		done <- nil
	}()
	go func() {
		var err error
		for n := 0; n < 50 && err == nil; n++ {
			_, err = f.r.Publish(Publication{Revision: open(), Epoch: uint64(n + 2), Number: uint64(n + 2)})
		}
		done <- err
	}()
	for range 2 {
		if err := <-done; err != nil && !errors.Is(err, ErrRejected) {
			t.Fatal(err)
		}
	}
}
