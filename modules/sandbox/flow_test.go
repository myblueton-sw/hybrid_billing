package sandbox

import "testing"

func TestProposalRules(t *testing.T) {
	f := setup(t)
	c := env(prod)
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, tool)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{Kind: KindHuman})), ErrInvalid)
	expect(t, "actor kind", second(f.r.Propose(c, Actor{ID: "x", Kind: "robot"})), ErrInvalid)
	expect(t, "form", second(f.r.Propose(Change{Kind: ChangeEnvironment}, ann)), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if again, _ := f.r.Propose(c, ann); again.Number != p.Number {
		t.Error("repeat not idempotent")
	}
	q, err := f.r.Propose(c, cat)
	f.must(err)
	reason(t, "self", second(f.r.Approve(p.Number, ann)), ReasonSelfApproval)
	reason(t, "approve system", second(f.r.Approve(p.Number, tool)), ReasonNotHuman)
	expect(t, "missing", second(f.r.Approve(99, bob)), ErrNotFound)
	v, err := f.r.Approve(p.Number, bob)
	if err != nil || v.State != ProposalApplied || v.Approver != "bob" {
		t.Errorf("applied %+v %v", v, err)
	}
	if again, err := f.r.Approve(p.Number, bob); err != nil || again.State != ProposalApplied {
		t.Errorf("repeat approve %v", err)
	}
	reason(t, "approve applied", second(f.r.Approve(p.Number, cat)), ReasonNotPending)
	expect(t, "recheck", second(f.r.Approve(q.Number, bob)), ErrConflict)
	reason(t, "withdraw other", second(f.r.Withdraw(q.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw applied", second(f.r.Withdraw(p.Number, ann)), ReasonNotPending)
	reason(t, "withdraw assistant", second(f.r.Withdraw(q.Number, bot)), ReasonAINotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, ann)), ErrNotFound)
	if w, err := f.r.Withdraw(q.Number, cat); err != nil || w.State != ProposalWithdrawn {
		t.Errorf("withdraw %v", err)
	}
	expect(t, "withdrawn read", second(f.r.Proposal(q.Number)), ErrNotFound)
	if got, err := f.r.Proposal(p.Number); err != nil || got.Change.Environment.ID != "prod" {
		t.Errorf("read %v", err)
	}
	s, err := f.r.Propose(Change{Kind: ChangeSchema, Schema: schema()}, ann)
	f.must(err)
	s.Change.Schema.Fields[3].Values[0] = "mutated"
	if got, _ := f.r.Proposal(s.Number); got.Change.Schema.Fields[3].Values[0] != "file" {
		t.Error("proposal shares memory")
	}
}

func TestEnvironmentRules(t *testing.T) {
	f := setup(t)
	reason(t, "no production", f.try(env(demo)), ReasonNoProduction)
	wrong := prod
	wrong.NumberPrefix = "DEMOX-"
	reason(t, "production demo prefix", f.try(env(wrong)), ReasonDemoPrefix)
	f.approve(env(prod))
	expect(t, "repeat", f.try(env(prod)), ErrConflict)
	plain := demo
	plain.NumberPrefix = "DM-"
	reason(t, "demo without prefix", f.try(env(plain)), ReasonDemoPrefix)
	shared := map[string]func(*Environment){
		"database":   func(e *Environment) { e.Database = "billing" },
		"keyring":    func(e *Environment) { e.KeyRing = "ring" },
		"path equal": func(e *Environment) { e.EvidencePath = "/srv/evidence" },
		"path below": func(e *Environment) { e.EvidencePath = "/srv/evidence/demo" },
		"path above": func(e *Environment) { e.EvidencePath = "/srv" },
		"number":     func(e *Environment) { e.NumberPrefix = "DEMO-" },
	}
	f.approve(env(demo))
	for name, mutate := range shared {
		e := Environment{ID: "x", Mode: ModeDemo, Database: "x_db", KeyRing: "x_ring", EvidencePath: "/x", NumberPrefix: "DEMOX-"}
		mutate(&e)
		reason(t, name, f.try(env(e)), ReasonNamespaceShared)
	}
	short := Environment{ID: "x", Mode: ModeDemo, Database: "x_db", KeyRing: "x_ring", EvidencePath: "/x", NumberPrefix: "DEM-"}
	reason(t, "short prefix", f.try(env(short)), ReasonDemoPrefix)
	sibling := Environment{ID: "x", Mode: ModeDemo, Database: "x_db", KeyRing: "x_ring", EvidencePath: "/srv/evidence2", NumberPrefix: "DEMOX-"}
	f.approve(env(sibling))
	prefixed := Environment{ID: "y", Mode: ModeProduction, Database: "y_db", KeyRing: "y_ring", EvidencePath: "/y", NumberPrefix: "IN-"}
	f.approve(env(prefixed))
	longer := Environment{ID: "z", Mode: ModeProduction, Database: "z_db", KeyRing: "z_ring", EvidencePath: "/z", NumberPrefix: "INVZ-"}
	f.approve(env(longer))
	g := setup(t)
	g.approve(env(prod))
	one, err := g.r.Propose(env(Environment{ID: "p1", Mode: ModeProduction, Database: "same", KeyRing: "p1_ring", EvidencePath: "/p1", NumberPrefix: "PA-"}), ann)
	g.must(err)
	g.approve(env(Environment{ID: "p2", Mode: ModeProduction, Database: "same", KeyRing: "p2_ring", EvidencePath: "/p2", NumberPrefix: "PB-"}))
	reason(t, "approval recheck", second(g.r.Approve(one.Number, cat)), ReasonNamespaceShared)
	got, err := f.r.Environment("demo")
	if err != nil || got != demo {
		t.Errorf("read %+v %v", got, err)
	}
	expect(t, "missing", second(f.r.Environment("nope")), ErrNotFound)
}

func TestDestinationRules(t *testing.T) {
	f := setup(t)
	f.approve(env(prod))
	expect(t, "env missing", f.try(dst(erpDst)), ErrNotFound)
	f.approve(env(demo))
	f.approve(env(demo2))
	onProd := erpDst
	onProd.Environment = "prod"
	reason(t, "production", f.try(dst(onProd)), ReasonNotDemo)
	for _, host := range []string{"erp.realcorp.com", "example.com", "localhost", "a.localhost", "127.0.0.1", "::1", "10.0.0.5"} {
		d := erpDst
		d.Host = host
		reason(t, host, f.try(dst(d)), ReasonNotReserved)
	}
	f.approve(dst(erpDst))
	expect(t, "repeat id", f.try(dst(erpDst)), ErrConflict)
	same := erpDst
	same.ID, same.Environment = "other", "demo2"
	expect(t, "endpoint across environments", f.try(dst(same)), ErrConflict)
	port := same
	port.Port = 8070
	f.approve(dst(port))
	reason(t, "disable system", f.r.DisableDestination("erp-mock", tool), ReasonNotHuman)
	expect(t, "disable form", f.r.DisableDestination("", ann), ErrInvalid)
	expect(t, "disable missing", f.r.DisableDestination("nope", ann), ErrNotFound)
	f.must(f.r.DisableDestination("erp-mock", ann))
	expect(t, "reuse after disable", f.try(dst(erpDst)), ErrConflict)
	list, err := f.r.Destinations("demo")
	if err != nil || len(list) != 1 || list[0].Enabled || list[0].Destination != erpDst {
		t.Errorf("list %+v %v", list, err)
	}
	if none, _ := f.r.Destinations("prod"); none == nil || len(none) != 0 {
		t.Error("empty list")
	}
	expect(t, "list missing", second(f.r.Destinations("nope")), ErrNotFound)
}

func TestSchemaRules(t *testing.T) {
	f := standard(t)
	expect(t, "repeat", f.try(Change{Kind: ChangeSchema, Schema: schema()}), ErrConflict)
	for _, key := range []string{"erp_endpoint", "api_token", "smtp_server"} {
		s := Schema{ID: "s-" + key, Fields: []FieldSpec{{Key: "a_plain", Kind: FieldFlag}, {Key: key, Kind: FieldFlag}}}
		reason(t, key, f.try(Change{Kind: ChangeSchema, Schema: s}), ReasonSensitiveKey)
	}
	got, err := f.r.Schema("erp-demo")
	if err != nil || len(got.Fields) != 4 {
		t.Fatalf("read %v", err)
	}
	got.Fields[3].Values[0] = "mutated"
	if again, _ := f.r.Schema("erp-demo"); again.Fields[3].Values[0] != "file" {
		t.Error("schema read shares memory")
	}
	expect(t, "missing", second(f.r.Schema("nope")), ErrNotFound)
}
