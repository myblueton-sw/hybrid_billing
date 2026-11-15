package authpolicy

import (
	"testing"
	"time"
)

func TestEvaluateOrder(t *testing.T) {
	f := setup(t)
	reason(t, "no policy", second(f.r.Evaluate(member("m1"), f.weak(), "")), ReasonNoPolicy)
	f.revise(open())
	strict := Revision{Workspace: "w1", Methods: []string{MethodOIDC, MethodPassword}, Issuers: []string{"idp"}, MFA: MFAAdministrators, MaxAge: time.Hour,
		StepUp: []StepUp{{Action: "export", MaxAge: 10 * time.Minute, MFA: true}, {Action: "view", MaxAge: 5 * time.Minute}}}
	f.revise(strict)
	now := f.at()
	at := func(method, issuer string, age time.Duration, factors ...string) Authentication {
		return Authentication{Method: method, Issuer: issuer, Factors: factors, At: now.Add(-age)}
	}
	d, err := f.r.Evaluate(member("m1"), at(MethodPassword, "", time.Hour), "")
	if err != nil || d.Revision != 2 || d.Version != 2 {
		t.Fatalf("allowed at exact age %+v %v", d, err)
	}
	cases := []struct {
		name string
		p    Principal
		a    Authentication
		act  string
		want string
	}{
		{"method", member("m1"), at(MethodSAML, "idp", 0), "", ReasonMethodNotAllowed},
		{"issuer", member("m1"), at(MethodOIDC, "other", 0), "", ReasonIssuerNotAllowed},
		{"future", member("m1"), at(MethodPassword, "", -time.Nanosecond), "", ReasonFutureAuthentication},
		{"stale", member("m1"), at(MethodPassword, "", time.Hour+time.Nanosecond), "", ReasonStale},
		{"admin mfa", admin("a9"), at(MethodPassword, "", 0), "", ReasonMFARequired},
		{"step-up age", member("m1"), at(MethodPassword, "", 5*time.Minute+time.Nanosecond), "view", ReasonStepUpRequired},
		{"step-up mfa", member("m1"), at(MethodPassword, "", 0), "export", ReasonStepUpMFARequired},
		{"step-up age before mfa", member("m1"), at(MethodPassword, "", 11*time.Minute), "export", ReasonStepUpRequired},
		{"stale before mfa", admin("a9"), at(MethodPassword, "", 2*time.Hour), "", ReasonStale},
		{"method before issuer", member("m1"), at(MethodLDAP, "other", 0), "", ReasonMethodNotAllowed},
	}
	for _, c := range cases {
		reason(t, c.name, second(f.r.Evaluate(c.p, c.a, c.act)), c.want)
	}
	for name, a := range map[string]Authentication{
		"view at bound":            at(MethodPassword, "", 5*time.Minute),
		"export with otp":          at(MethodPassword, "", 10*time.Minute, FactorOTP),
		"unknown action":           at(MethodPassword, "", time.Hour),
		"password ignores issuers": at(MethodPassword, "", 0),
	} {
		act := map[string]string{"view at bound": "view", "export with otp": "export", "unknown action": "other"}[name]
		if _, err := f.r.Evaluate(member("m1"), a, act); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, a := range map[string]Authentication{"otp": at(MethodPassword, "", 0, FactorOTP), "webauthn": at(MethodPassword, "", 0, FactorWebAuthn), "idp": at(MethodOIDC, "idp", 0, FactorIdP)} {
		if _, err := f.r.Evaluate(admin("a9"), a, ""); err != nil {
			t.Errorf("admin %s: %v", name, err)
		}
	}
	f.revise(Revision{Workspace: "w1", Methods: []string{MethodOIDC, MethodPassword, MethodSAML}, MFA: MFAEveryone, MaxAge: time.Hour})
	reason(t, "everyone", second(f.r.Evaluate(member("m1"), at(MethodPassword, "", 0), "")), ReasonMFARequired)
	if _, err := f.r.Evaluate(member("m1"), at(MethodSAML, "any", 0, FactorIdP), ""); err != nil {
		t.Errorf("no issuers allows any: %v", err)
	}
	reason(t, "assistant", second(f.r.Evaluate(Principal{User: "x", Workspace: "w1", Role: RoleMember, Kind: KindAssistant}, f.weak(), "")), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Evaluate(Principal{User: "x", Workspace: "w1", Role: RoleMember, Kind: KindSystem}, f.weak(), "")), ReasonNotHuman)
	expect(t, "kind", second(f.r.Evaluate(Principal{User: "x", Workspace: "w1", Role: RoleMember}, f.weak(), "")), ErrInvalid)
	g := setup(t)
	for name, x := range map[string]struct {
		p   Principal
		a   Authentication
		act string
	}{
		"service":           {Principal{User: "x", Workspace: "w1", Role: "service", Kind: KindHuman}, g.weak(), ""},
		"user":              {Principal{Workspace: "w1", Role: RoleMember, Kind: KindHuman}, g.weak(), ""},
		"workspace":         {Principal{User: "x", Role: RoleMember, Kind: KindHuman}, g.weak(), ""},
		"action":            {member("m1"), g.weak(), "a b"},
		"method":            {member("m1"), Authentication{Method: "kerberos", At: start}, ""},
		"zero time":         {member("m1"), Authentication{Method: MethodPassword}, ""},
		"password issuer":   {member("m1"), Authentication{Method: MethodPassword, Issuer: "x", At: start}, ""},
		"ldap issuer":       {member("m1"), Authentication{Method: MethodLDAP, At: start}, ""},
		"idp with ldap":     {member("m1"), Authentication{Method: MethodLDAP, Issuer: "x", Factors: []string{FactorIdP}, At: start}, ""},
		"idp with password": {member("m1"), Authentication{Method: MethodPassword, Factors: []string{FactorIdP}, At: start}, ""},
		"factor order":      {member("m1"), Authentication{Method: MethodOIDC, Issuer: "x", Factors: []string{FactorOTP, FactorIdP}, At: start}, ""},
		"factor repeat":     {member("m1"), Authentication{Method: MethodOIDC, Issuer: "x", Factors: []string{FactorOTP, FactorOTP}, At: start}, ""},
		"factor kind":       {member("m1"), Authentication{Method: MethodOIDC, Issuer: "x", Factors: []string{"sms"}, At: start}, ""},
	} {
		expect(t, name+" before no_policy", second(g.r.Evaluate(x.p, x.a, x.act)), ErrInvalid)
	}
	if _, err := g.r.Evaluate(member("m1"), Authentication{Method: MethodSAML, Issuer: "x", Factors: []string{FactorIdP, FactorOTP, FactorWebAuthn}, At: start}, ""); ReasonOf(err) != ReasonNoPolicy {
		t.Error("all factors form")
	}
}

func ReasonOf(err error) string {
	if r, ok := err.(Refusal); ok {
		return r.Reason
	}
	return ""
}

func TestVersions(t *testing.T) {
	f := setup(t)
	if f.r.Current("w1", 0) {
		t.Error("no workspace")
	}
	f.revise(open())
	d, _ := f.r.Evaluate(member("m1"), f.weak(), "")
	if !f.r.Current("w1", d.Version) || d.Version != 1 || f.r.Current("w1", 0) || f.r.Current("w2", 1) {
		t.Fatal("current")
	}
	id := f.block([]string{"x1"}, 1)
	if f.r.Current("w1", 1) || !f.r.Current("w1", 2) {
		t.Error("plan bumps")
	}
	f.by(Change{Kind: ChangeLift, Lift: Lift{Plan: id, Reason: "done"}}, ann, bob)
	if !f.r.Current("w1", 3) {
		t.Error("lift bumps")
	}
	f.revise(open())
	if !f.r.Current("w1", 4) {
		t.Error("revision bumps")
	}
	if v, err := f.r.Policy("w1"); err != nil || v.Number != 2 || v.Proposer != "ann" || v.Approver != "bob" || !v.At.Equal(start) {
		t.Errorf("policy %+v", v)
	}
	expect(t, "policy missing", second(f.r.Policy("w2")), ErrNotFound)
	if v, err := f.r.Revision("w1", 1); err != nil || v.Number != 1 {
		t.Error("revision 1")
	}
	expect(t, "revision missing", second(f.r.Revision("w1", 3)), ErrNotFound)
	expect(t, "revision workspace", second(f.r.Revision("w2", 1)), ErrNotFound)
	v, _ := f.r.Policy("w1")
	v.Revision.Methods[0] = "zzz"
	if w, _ := f.r.Policy("w1"); w.Revision.Methods[0] != MethodLDAP {
		t.Error("policy shares memory")
	}
	for i := 0; i < 101; i++ {
		f.revise(open())
	}
	expect(t, "evicted", second(f.r.Revision("w1", 3)), ErrNotFound)
	if v, err := f.r.Revision("w1", 4); err != nil || v.Number != 4 {
		t.Error("kept 100")
	}
	if w, _ := f.r.Policy("w1"); w.Number != 103 {
		t.Errorf("numbering %d", w.Number)
	}
}
