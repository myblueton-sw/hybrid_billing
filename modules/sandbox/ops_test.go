package sandbox

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAllow(t *testing.T) {
	f := standard(t)
	ok := Request{Environment: "demo", Kind: DestinationERP, Host: "erp.mock.test", Port: 8069}
	f.must(f.r.Allow(ok))
	refusals := map[string]func(*Request){
		"real host":      func(q *Request) { q.Host = "erp.realcorp.com" },
		"other port":     func(q *Request) { q.Port = 8070 },
		"other kind":     func(q *Request) { q.Kind = DestinationPayment },
		"other demo":     func(q *Request) { q.Environment = "demo2" },
		"other reserved": func(q *Request) { q.Host = "mail.mock.test" },
	}
	for name, mutate := range refusals {
		q := ok
		mutate(&q)
		reason(t, name, f.r.Allow(q), ReasonNotMock)
	}
	invalid := map[string]func(*Request){
		"env":       func(q *Request) { q.Environment = "Demo" },
		"kind":      func(q *Request) { q.Kind = "ftp" },
		"upper":     func(q *Request) { q.Host = "ERP.mock.test" },
		"dot":       func(q *Request) { q.Host = "erp.mock.test." },
		"mapped":    func(q *Request) { q.Host = "::ffff:127.0.0.1" },
		"port0":     func(q *Request) { q.Port = 0 },
		"port high": func(q *Request) { q.Port = 65536 },
	}
	for name, mutate := range invalid {
		q := ok
		mutate(&q)
		expect(t, name, f.r.Allow(q), ErrInvalid)
	}
	edge := ok
	edge.Port = 65535
	reason(t, "max port", f.r.Allow(edge), ReasonNotMock)
	edge.Port = 1
	reason(t, "min port", f.r.Allow(edge), ReasonNotMock)
	missing := ok
	missing.Environment = "nope"
	expect(t, "missing", f.r.Allow(missing), ErrNotFound)
	production := Request{Environment: "prod", Kind: DestinationERP, Host: "erp.realcorp.com", Port: 443}
	f.must(f.r.Allow(production))
	for _, host := range []string{"erp.mock.test", "x.invalid", "y.example"} {
		q := production
		q.Host = host
		reason(t, "production "+host, f.r.Allow(q), ReasonMockDestination)
	}
	registered := production
	registered.Host, registered.Port = "erp.mock.test", 8069
	reason(t, "registered mock", f.r.Allow(registered), ReasonMockDestination)
	f.must(f.r.DisableDestination("erp-mock", ann))
	reason(t, "disabled", f.r.Allow(ok), ReasonNotMock)
	reason(t, "disabled still mock in production", f.r.Allow(registered), ReasonMockDestination)
}

func TestCheckConfig(t *testing.T) {
	f := standard(t)
	good := Config{Environment: "demo", Schema: "erp-demo", Settings: []Setting{{Key: "batch_size", Value: "-50"}, {Key: "dry_run", Value: "true"}, {Key: "erp", Value: "erp-mock"}, {Key: "mode", Value: "odoo-test"}}}
	f.must(f.r.CheckConfig(good))
	f.must(f.r.CheckConfig(Config{Environment: "demo", Schema: "erp-demo"}))
	values := map[string]Setting{
		"integer text":  {Key: "batch_size", Value: "ten"},
		"integer plus":  {Key: "batch_size", Value: "+5"},
		"integer zero":  {Key: "batch_size", Value: "05"},
		"integer large": {Key: "batch_size", Value: "9223372036854775808"},
		"flag":          {Key: "dry_run", Value: "yes"},
		"enum":          {Key: "mode", Value: "erp.realcorp.com"},
		"destination":   {Key: "erp", Value: "nope"},
	}
	for name, s := range values {
		expect(t, name, f.r.CheckConfig(Config{Environment: "demo", Schema: "erp-demo", Settings: []Setting{s}}), ErrInvalid)
	}
	f.must(f.r.CheckConfig(Config{Environment: "demo", Schema: "erp-demo", Settings: []Setting{{Key: "batch_size", Value: "9223372036854775807"}, {Key: "dry_run", Value: "false"}}}))
	reason(t, "undeclared", f.r.CheckConfig(Config{Environment: "demo", Schema: "erp-demo", Settings: []Setting{{Key: "db_url", Value: "x"}}}), ReasonUndeclared)
	reason(t, "production", f.r.CheckConfig(Config{Environment: "prod", Schema: "erp-demo"}), ReasonNotDemo)
	expect(t, "schema missing", f.r.CheckConfig(Config{Environment: "demo", Schema: "nope"}), ErrNotFound)
	expect(t, "env missing", f.r.CheckConfig(Config{Environment: "nope", Schema: "erp-demo"}), ErrNotFound)
	forms := []Config{
		{Environment: "Demo", Schema: "erp-demo"},
		{Environment: "demo", Schema: ""},
		{Environment: "demo", Schema: "erp-demo", Settings: []Setting{{Key: "mode", Value: "file"}, {Key: "dry_run", Value: "true"}}},
		{Environment: "demo", Schema: "erp-demo", Settings: []Setting{{Key: "mode", Value: "file"}, {Key: "mode", Value: "file"}}},
		{Environment: "demo", Schema: "erp-demo", Settings: []Setting{{Key: "Mode", Value: "file"}}},
	}
	for index, c := range forms {
		expect(t, "form "+itoa(index), f.r.CheckConfig(c), ErrInvalid)
	}
	many := Config{Environment: "demo", Schema: "erp-demo"}
	for len(many.Settings) <= maxFields {
		many.Settings = append(many.Settings, Setting{Key: "k" + itoa(1000+len(many.Settings)), Value: "1"})
	}
	expect(t, "too many", f.r.CheckConfig(many), ErrInvalid)
	many.Settings = many.Settings[:maxFields]
	reason(t, "cap fits form", f.r.CheckConfig(many), ReasonUndeclared)
	webhook := Destination{ID: "hook", Environment: "demo", Kind: DestinationWebhook, Host: "hook.mock.test", Port: 80}
	f.approve(dst(webhook))
	expect(t, "wrong kind", f.r.CheckConfig(Config{Environment: "demo", Schema: "erp-demo", Settings: []Setting{{Key: "erp", Value: "hook"}}}), ErrInvalid)
	foreign := Destination{ID: "erp2", Environment: "demo2", Kind: DestinationERP, Host: "erp2.mock.test", Port: 80}
	f.approve(dst(foreign))
	expect(t, "foreign", f.r.CheckConfig(Config{Environment: "demo", Schema: "erp-demo", Settings: []Setting{{Key: "erp", Value: "erp2"}}}), ErrInvalid)
	f.must(f.r.DisableDestination("erp-mock", ann))
	expect(t, "disabled", f.r.CheckConfig(good), ErrInvalid)
}

func TestReset(t *testing.T) {
	f := standard(t)
	targets := []Target{
		{Kind: TargetConnection, Name: "erp-mock"},
		{Kind: TargetDatabase, Name: "demo_billing"},
		{Kind: TargetEvidence, Name: "/srv/demo/evidence/run1"},
		{Kind: TargetKeyRing, Name: "demo_ring"},
		{Kind: TargetNumbers, Name: "DEMO-"},
		{Kind: TargetVolume, Name: "demo"},
		{Kind: TargetVolume, Name: "demo-db"},
	}
	f.advance(time.Minute)
	v, err := f.r.BeginReset(Plan{Environment: "demo", Targets: targets}, tool)
	if err != nil || v.ID != 1 || v.By != "reset-tool" || !v.At.Equal(start.Add(time.Minute)) || len(v.Done) != 7 || v.Completed {
		t.Fatalf("begin %+v %v", v, err)
	}
	outside := map[string]Target{
		"prod database":    {Kind: TargetDatabase, Name: "billing"},
		"demo2 database":   {Kind: TargetDatabase, Name: "demo2_billing"},
		"prod keyring":     {Kind: TargetKeyRing, Name: "ring"},
		"prod evidence":    {Kind: TargetEvidence, Name: "/srv/evidence"},
		"string prefix":    {Kind: TargetEvidence, Name: "/srv/demo/evidencex"},
		"parent":           {Kind: TargetEvidence, Name: "/srv/demo"},
		"prod numbers":     {Kind: TargetNumbers, Name: "INV-"},
		"demo2 connection": {Kind: TargetConnection, Name: "nope"},
		"prod volume":      {Kind: TargetVolume, Name: "prod-db"},
		"demo2 volume":     {Kind: TargetVolume, Name: "demo2-db"},
		"volume dash":      {Kind: TargetVolume, Name: "demo-"},
		"volume glued":     {Kind: TargetVolume, Name: "demox"},
	}
	for name, target := range outside {
		reason(t, name, second(f.r.BeginReset(Plan{Environment: "demo", Targets: []Target{target}}, tool)), ReasonOutsideEnv)
	}
	reason(t, "production", second(f.r.BeginReset(Plan{Environment: "prod", Targets: []Target{{Kind: TargetDatabase, Name: "billing"}}}, ann)), ReasonNotDemo)
	reason(t, "assistant", second(f.r.BeginReset(Plan{Environment: "demo", Targets: targets}, bot)), ReasonAINotAllowed)
	expect(t, "missing", second(f.r.BeginReset(Plan{Environment: "nope", Targets: targets}, tool)), ErrNotFound)
	forms := []Plan{
		{Environment: "Demo", Targets: targets},
		{Environment: "demo"},
		{Environment: "demo", Key: "a b", Targets: targets},
		{Environment: "demo", Targets: []Target{targets[1], targets[0]}},
		{Environment: "demo", Targets: []Target{targets[0], targets[0]}},
		{Environment: "demo", Targets: []Target{{Kind: "disk", Name: "x"}}},
	}
	for index, p := range forms {
		expect(t, "form "+itoa(index), second(f.r.BeginReset(p, tool)), ErrInvalid)
	}
	expect(t, "actor", second(f.r.BeginReset(Plan{Environment: "demo", Targets: targets}, Actor{})), ErrInvalid)
	sameKind := []Target{{Kind: TargetVolume, Name: "demo-a"}, {Kind: TargetVolume, Name: "demo-b"}}
	f.must(second(f.r.BeginReset(Plan{Environment: "demo", Targets: sameKind}, tool)))
	expect(t, "same kind order", second(f.r.BeginReset(Plan{Environment: "demo", Targets: []Target{sameKind[1], sameKind[0]}}, tool)), ErrInvalid)
	done, err := f.r.CompleteTarget("demo", 1, targets[0], tool)
	if err != nil || !done.Done[0] || done.Completed {
		t.Errorf("complete %+v %v", done, err)
	}
	if again, _ := f.r.CompleteTarget("demo", 1, targets[0], ann); !again.Done[0] {
		t.Error("complete not idempotent")
	}
	rec, err := f.r.ResetRecord("demo", 1)
	if err != nil || !rec.Done[0] || rec.Done[1] {
		t.Errorf("record %+v %v", rec, err)
	}
	rec.Done[1] = true
	rec.Targets[0].Name = "mutated"
	if again, _ := f.r.ResetRecord("demo", 1); again.Done[1] || again.Targets[0].Name != "erp-mock" {
		t.Error("record shares memory")
	}
	for _, target := range targets[1:] {
		done, err = f.r.CompleteTarget("demo", 1, target, tool)
		f.must(err)
	}
	if !done.Completed {
		t.Error("not completed")
	}
	expect(t, "unknown target", second(f.r.CompleteTarget("demo", 1, Target{Kind: TargetVolume, Name: "demo-x"}, tool)), ErrNotFound)
	reason(t, "complete assistant", second(f.r.CompleteTarget("demo", 1, targets[0], bot)), ReasonAINotAllowed)
	expect(t, "complete form", second(f.r.CompleteTarget("demo", 1, Target{Kind: "disk"}, tool)), ErrInvalid)
	expect(t, "complete actor", second(f.r.CompleteTarget("demo", 1, targets[0], Actor{})), ErrInvalid)
	expect(t, "future id", second(f.r.ResetRecord("demo", 99)), ErrNotFound)
	latest := f.r.environments["demo"].next
	if v, err := f.r.ResetRecord("demo", latest); err != nil || v.ID != latest {
		t.Errorf("latest record %+v %v", v, err)
	}
	expect(t, "next id", second(f.r.ResetRecord("demo", latest+1)), ErrNotFound)
	expect(t, "zero id", second(f.r.ResetRecord("demo", 0)), ErrInvalid)
	expect(t, "bad env", second(f.r.ResetRecord("Demo", 1)), ErrInvalid)
	expect(t, "missing env", second(f.r.ResetRecord("nope", 1)), ErrNotFound)
	expect(t, "complete missing env", second(f.r.CompleteTarget("nope", 1, targets[0], tool)), ErrNotFound)
}

func TestResetKeysAndEviction(t *testing.T) {
	f := standard(t)
	targets := []Target{{Kind: TargetDatabase, Name: "demo_billing"}}
	first, err := f.r.BeginReset(Plan{Environment: "demo", Key: "run-1", Targets: targets}, tool)
	f.must(err)
	again, err := f.r.BeginReset(Plan{Environment: "demo", Key: "run-1", Targets: targets}, tool)
	if err != nil || again.ID != first.ID || again.Key != "run-1" {
		t.Errorf("key repeat %+v %v", again, err)
	}
	expect(t, "key conflict", second(f.r.BeginReset(Plan{Environment: "demo", Key: "run-1", Targets: []Target{{Kind: TargetKeyRing, Name: "demo_ring"}}}, tool)), ErrConflict)
	expect(t, "key conflict longer", second(f.r.BeginReset(Plan{Environment: "demo", Key: "run-1", Targets: []Target{targets[0], {Kind: TargetKeyRing, Name: "demo_ring"}}}, tool)), ErrConflict)
	other, _ := f.r.BeginReset(Plan{Environment: "demo2", Key: "run-1", Targets: []Target{{Kind: TargetDatabase, Name: "demo2_billing"}}}, tool)
	if other.ID != 1 {
		t.Errorf("keys are per environment %+v", other)
	}
	if plain, _ := f.r.BeginReset(Plan{Environment: "demo", Targets: targets}, tool); plain.ID != 2 {
		t.Errorf("unkeyed %+v", plain)
	}
	env := f.r.environments["demo"]
	for len(env.resets) < maxResets {
		f.must(second(f.r.BeginReset(Plan{Environment: "demo", Targets: targets}, tool)))
	}
	if env.watermark != 0 {
		t.Fatalf("evicted early %d", env.watermark)
	}
	f.must(second(f.r.BeginReset(Plan{Environment: "demo", Targets: targets}, tool)))
	if env.watermark != 1 || len(env.resets) != maxResets {
		t.Fatalf("watermark %d %d", env.watermark, len(env.resets))
	}
	reason(t, "stale", second(f.r.ResetRecord("demo", 1)), ReasonStale)
	reason(t, "stale complete", second(f.r.CompleteTarget("demo", 1, targets[0], tool)), ReasonStale)
	if _, err := f.r.ResetRecord("demo", 2); err != nil {
		t.Errorf("live record %v", err)
	}
	fresh, err := f.r.BeginReset(Plan{Environment: "demo", Key: "run-1", Targets: []Target{{Kind: TargetKeyRing, Name: "demo_ring"}}}, tool)
	if err != nil || fresh.ID != maxResets+2 {
		t.Errorf("evicted key reusable %+v %v", fresh, err)
	}
}

func TestAdmit(t *testing.T) {
	f := standard(t)
	f.must(f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "billing", Origin: OriginReal, Number: "INV-2026-0001"}))
	f.must(f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "billing", Origin: OriginReal}))
	for _, origin := range []string{OriginSynthetic, OriginPublic, OriginConverted} {
		reason(t, origin, f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "billing", Origin: origin}), ReasonNotReal)
		f.must(f.r.Admit(Input{Environment: "demo", Source: "demo", Database: "demo_billing", Origin: origin, Number: "DEMO-1"}))
	}
	reason(t, "demo source", f.r.Admit(Input{Environment: "prod", Source: "demo", Database: "billing", Origin: OriginReal}), ReasonCrossEnvironment)
	reason(t, "demo database", f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "demo_billing", Origin: OriginReal}), ReasonCrossEnvironment)
	reason(t, "prod into demo", f.r.Admit(Input{Environment: "demo", Source: "prod", Database: "demo_billing", Origin: OriginReal}), ReasonCrossEnvironment)
	reason(t, "demo number", f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "billing", Origin: OriginReal, Number: "DEMO-1"}), ReasonForeignNumber)
	reason(t, "other number", f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "billing", Origin: OriginReal, Number: "X-1"}), ReasonForeignNumber)
	reason(t, "demo foreign", f.r.Admit(Input{Environment: "demo", Source: "demo", Database: "demo_billing", Origin: OriginReal, Number: "INV-1"}), ReasonForeignNumber)
	f.approve(env(Environment{ID: "dp", Mode: ModeProduction, Database: "dp_db", KeyRing: "dp_ring", EvidencePath: "/dp", NumberPrefix: "DP-"}))
	f.must(f.r.Admit(Input{Environment: "dp", Source: "dp", Database: "dp_db", Origin: OriginReal, Number: "DP-1"}))
	forms := []Input{
		{Environment: "Prod", Source: "prod", Database: "billing", Origin: OriginReal},
		{Environment: "prod", Source: "", Database: "billing", Origin: OriginReal},
		{Environment: "prod", Source: "prod", Database: "Billing", Origin: OriginReal},
		{Environment: "prod", Source: "prod", Database: "billing", Origin: "mock"},
		{Environment: "prod", Source: "prod", Database: "billing", Origin: OriginReal, Number: "INV 1"},
	}
	for index, in := range forms {
		expect(t, "form "+itoa(index), f.r.Admit(in), ErrInvalid)
	}
	expect(t, "missing", f.r.Admit(Input{Environment: "nope", Source: "nope", Database: "billing", Origin: OriginReal}), ErrNotFound)
}

func TestNumberOutsidePrefix(t *testing.T) {
	f := standard(t)
	reason(t, "glued", f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "billing", Origin: OriginReal, Number: "INVDEMO-1"}), ReasonForeignNumber)
	reason(t, "bare prefix", f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "billing", Origin: OriginReal, Number: "INV-"}), ReasonForeignNumber)
	f.must(f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "billing", Origin: OriginReal, Number: "INV-1"}))
	reason(t, "numbers target", second(f.r.BeginReset(Plan{Environment: "demo", Targets: []Target{{Kind: TargetNumbers, Name: "DEMO-1"}}}, tool)), ReasonOutsideEnv)
	reason(t, "keyring target", second(f.r.BeginReset(Plan{Environment: "demo", Targets: []Target{{Kind: TargetKeyRing, Name: "demo2_ring"}}}, tool)), ReasonOutsideEnv)
	f.must(f.r.Admit(Input{Environment: "demo", Source: "demo", Database: "demo_billing", Origin: OriginConverted}))
}

func TestBoundsAtLimit(t *testing.T) {
	f := standard(t)
	targets := []Target{}
	for len(targets) < maxTargets {
		targets = append(targets, Target{Kind: TargetVolume, Name: "demo-" + strings.Repeat("0", 3-len(itoa(len(targets)))) + itoa(len(targets))})
	}
	f.must(second(f.r.BeginReset(Plan{Environment: "demo", Targets: targets}, tool)))
	expect(t, "targets", second(f.r.BeginReset(Plan{Environment: "demo", Targets: append(targets, Target{Kind: TargetVolume, Name: "demo-z"})}, tool)), ErrInvalid)
	key := strings.Repeat("k", maxIDBytes)
	f.must(second(f.r.BeginReset(Plan{Environment: "demo", Key: key, Targets: targets[:1]}, tool)))
	expect(t, "key", second(f.r.BeginReset(Plan{Environment: "demo", Key: key + "k", Targets: targets[:1]}, tool)), ErrInvalid)
	long := "/" + strings.Repeat("a", 64) + "/" + strings.Repeat("b", 64) + "/" + strings.Repeat("c", 64) + "/" + strings.Repeat("d", 4)
	if len(long) != maxPathBytes || !validPath(long) || validPath(long+"e") {
		t.Errorf("path bound %d", len(long))
	}
	if !validDestID(strings.Repeat("a", 64)) || validDestID(strings.Repeat("a", 65)) || validDestID("smtp.realcorp.com:25") || validDestID("A") || validDestID("") || !validDestID("erp_mock-2") {
		t.Error("destination id grammar")
	}
	if !validID(strings.Repeat("a", maxIDBytes)) || validID(strings.Repeat("a", maxIDBytes+1)) {
		t.Error("id bound")
	}
}

func TestLabel(t *testing.T) {
	f := standard(t)
	v, err := f.r.Label("demo", OriginSynthetic, "gen-1")
	if err != nil || !v.Demo || v.Marking != MarkingDemo || v.Origin != OriginSynthetic || v.Source != "gen-1" || v.Environment != "demo" {
		t.Errorf("demo %+v %v", v, err)
	}
	for _, origin := range []string{OriginPublic, OriginConverted, OriginReal} {
		if v, err := f.r.Label("demo", origin, "s"); err != nil || v.Origin != origin {
			t.Errorf("origin %s %v", origin, err)
		}
	}
	v, err = f.r.Label("prod", OriginReal, "aws")
	if err != nil || v.Demo || v.Marking != "" || v.Source != "aws" {
		t.Errorf("prod %+v %v", v, err)
	}
	for _, args := range [][3]string{{"Demo", OriginReal, "s"}, {"demo", "mock", "s"}, {"demo", OriginReal, ""}} {
		expect(t, "form", second(f.r.Label(args[0], args[1], args[2])), ErrInvalid)
	}
	expect(t, "missing", second(f.r.Label("nope", OriginReal, "s")), ErrNotFound)
}

func TestConcurrency(t *testing.T) {
	f := standard(t)
	var wg sync.WaitGroup
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for n := 0; n < 20; n++ {
				f.r.Allow(Request{Environment: "demo", Kind: DestinationERP, Host: "erp.mock.test", Port: 8069})
				f.r.BeginReset(Plan{Environment: "demo", Targets: []Target{{Kind: TargetDatabase, Name: "demo_billing"}}}, tool)
				f.r.Admit(Input{Environment: "prod", Source: "prod", Database: "billing", Origin: OriginReal})
				f.r.CheckConfig(Config{Environment: "demo", Schema: "erp-demo"})
				f.r.Label("demo", OriginSynthetic, "gen")
				if v, err := f.r.BeginReset(Plan{Environment: "demo", Targets: []Target{{Kind: TargetKeyRing, Name: "demo_ring"}}}, tool); err == nil {
					f.r.CompleteTarget("demo", v.ID, v.Targets[0], tool)
					f.r.ResetRecord("demo", v.ID)
				}
				p, err := f.r.Propose(Change{Kind: ChangeSchema, Schema: Schema{ID: "c" + itoa(index) + "-" + itoa(n), Fields: []FieldSpec{{Key: "mode", Kind: FieldFlag}}}}, ann)
				if err == nil {
					f.r.Approve(p.Number, bob)
				}
			}
		}(index)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		f.r.DisableDestination("erp-mock", cat)
	}()
	wg.Wait()
	if f.r.environments["demo"].next != 320 || len(f.r.schemas) != 161 {
		t.Errorf("counts %d %d", f.r.environments["demo"].next, len(f.r.schemas))
	}
}
