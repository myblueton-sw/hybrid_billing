package authpolicy

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestForms(t *testing.T) {
	expect(t, "clock", second(NewRegistry(nil)), ErrInvalid)
	good := Revision{Workspace: "w1", Methods: []string{MethodOIDC}, Issuers: []string{"a", "b"}, MFA: MFANone, MaxAge: time.Hour,
		StepUp: []StepUp{{Action: "a", MaxAge: time.Minute}, {Action: "b", MaxAge: time.Hour, MFA: true}}}
	if _, err := clean(Change{Kind: ChangeRevision, Revision: good}); err != nil {
		t.Fatal(err)
	}
	long := good
	long.MaxAge = 30 * 24 * time.Hour
	if _, err := clean(Change{Kind: ChangeRevision, Revision: long}); err != nil {
		t.Error("30 days")
	}
	rules := func(n int) []StepUp {
		out := []StepUp{}
		for i := 0; i < n; i++ {
			out = append(out, StepUp{Action: "a" + itoa(100+i), MaxAge: time.Minute})
		}
		return out
	}
	issuers := func(n int) []string {
		out := []string{}
		for i := 0; i < n; i++ {
			out = append(out, "i"+itoa(100+i))
		}
		return out
	}
	edge := good
	edge.StepUp, edge.Issuers, edge.Methods = rules(64), issuers(64), []string{MethodLDAP, MethodOIDC, MethodPassword, MethodSAML}
	if _, err := clean(Change{Kind: ChangeRevision, Revision: edge}); err != nil {
		t.Error("64 rules and issuers")
	}
	for name, mutate := range map[string]func(*Revision){
		"workspace": func(x *Revision) { x.Workspace = "" }, "no methods": func(x *Revision) { x.Methods = nil },
		"method": func(x *Revision) { x.Methods = []string{"kerberos"} }, "method order": func(x *Revision) { x.Methods = []string{MethodOIDC, MethodLDAP} },
		"method repeat": func(x *Revision) { x.Methods = []string{MethodOIDC, MethodOIDC} }, "five methods": func(x *Revision) { x.Methods = append(Methods[:4:4], "zz") },
		"issuer": func(x *Revision) { x.Issuers = []string{"a b"} }, "issuer order": func(x *Revision) { x.Issuers = []string{"b", "a"} },
		"65 issuers": func(x *Revision) { x.Issuers = issuers(65) }, "mfa": func(x *Revision) { x.MFA = "some" },
		"short age": func(x *Revision) { x.MaxAge = time.Minute - 1 }, "long age": func(x *Revision) { x.MaxAge = 30*24*time.Hour + 1 },
		"65 rules": func(x *Revision) { x.StepUp = rules(65) }, "rule action": func(x *Revision) { x.StepUp[0].Action = "" },
		"rule order": func(x *Revision) { x.StepUp[0].Action = "c" }, "rule repeat": func(x *Revision) { x.StepUp[0].Action = "b" }, "rule short": func(x *Revision) { x.StepUp[0].MaxAge = time.Minute - 1 },
		"rule over max": func(x *Revision) { x.StepUp[1].MaxAge = time.Hour + 1 },
	} {
		x := good
		x.StepUp = append([]StepUp{}, good.StepUp...)
		mutate(&x)
		expect(t, name, second(clean(Change{Kind: ChangeRevision, Revision: x})), ErrInvalid)
	}
	plan := Plan{Workspace: "w1", Users: []string{"a", "b"}, Chunk: 1000, Reason: "r"}
	if _, err := clean(Change{Kind: ChangePlan, Plan: plan}); err != nil {
		t.Fatal(err)
	}
	many := make([]string, 100001)
	for i := range many {
		many[i] = "u" + itoa(1000000+i)
	}
	for name, mutate := range map[string]func(*Plan){
		"id": func(p *Plan) { p.ID = "w1/1" }, "workspace": func(p *Plan) { p.Workspace = "" }, "users": func(p *Plan) { p.Users = nil },
		"order": func(p *Plan) { p.Users = []string{"b", "a"} }, "user form": func(p *Plan) { p.Users = []string{"a b"} },
		"chunk zero": func(p *Plan) { p.Chunk = 0 }, "chunk big": func(p *Plan) { p.Chunk = 1001 }, "reason": func(p *Plan) { p.Reason = "" },
		"too many": func(p *Plan) { p.Users = many },
	} {
		x := plan
		mutate(&x)
		expect(t, "plan "+name, second(clean(Change{Kind: ChangePlan, Plan: x})), ErrInvalid)
	}
	plan.Users, plan.Chunk = many[:100000], 1
	if _, err := clean(Change{Kind: ChangePlan, Plan: plan}); err != nil {
		t.Error("100000 users")
	}
	expect(t, "lift reason", second(clean(Change{Kind: ChangeLift, Lift: Lift{Plan: "w1/1"}})), ErrInvalid)
	expect(t, "kind", second(clean(Change{Kind: "other"})), ErrInvalid)
	if c, _ := clean(Change{Kind: ChangeLift, Lift: Lift{Plan: "w1/1", Reason: "r"}, Revision: good}); c.Revision.Workspace != "" {
		t.Error("lift keeps other fields")
	}
	for _, id := range []string{"w1/1", "w/9223372036854775807", "a:b/42", strings.Repeat("w", 128) + "/1"} {
		if !validPlanID(id) {
			t.Errorf("plan id %s", id)
		}
	}
	for _, id := range []string{"w1/0", "w1/01", "w1/9223372036854775808", "w1/", "/1", "w1", "w1/-1", "w1/+1", "w1/1a", "a b/1", "a/b/1", strings.Repeat("w", 129) + "/1"} {
		if validPlanID(id) {
			t.Errorf("plan id %s accepted", id)
		}
	}
	if !validText(strings.Repeat("a", 200)) || validText(strings.Repeat("a", 201)) || validText("a\x01") || validText("a\x7f") || validText("a\u0080") || validText("a\u009f") || !validText("a ") || validText("\xff") || !validText("a\x20") {
		t.Error("text")
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", ".", "_", ":", "-", strings.Repeat("a", 128)} {
		if !validID(id) {
			t.Errorf("id %q", id)
		}
	}
	for _, id := range []string{"", "/", "a b", "@", "[", "`", "{", "!", strings.Repeat("a", 129)} {
		if validID(id) {
			t.Errorf("id %q accepted", id)
		}
	}
	if !contains([]string{"a", "c", "e"}, "e") || contains([]string{"a", "c", "e"}, "d") || contains(nil, "a") || contains([]string{"a", "c", "e"}, "f") || !contains([]string{"a", "c", "e"}, "a") {
		t.Error("contains")
	}
}

func TestConcurrency(t *testing.T) {
	f := standard(t)
	id := f.block([]string{"x1", "x2", "x3", "x4"}, 1)
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for m := 0; m < 30; m++ {
				f.r.Evaluate(member("u"+itoa(n)), f.weak(), "")
				f.r.Complete(id, m%4, worker)
				f.r.Current("w1", 2)
				f.r.Propose(Change{Kind: ChangeRevision, Revision: Revision{Workspace: "w1", Methods: Methods, MFA: MFANone, MaxAge: time.Duration(n*100+m+2) * time.Minute}}, admin("p"+itoa(n)), f.strong())
			}
		}(n)
	}
	wg.Wait()
	if v, _ := f.r.Plan(id); v.State != PlanComplete || len(f.r.pending) != 100 {
		t.Errorf("state %+v %d", v.State, len(f.r.pending))
	}
}
