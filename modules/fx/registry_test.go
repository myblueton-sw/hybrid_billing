package fx

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestNewRegistry(t *testing.T) {
	clock := func() time.Time { return start }
	if _, err := NewRegistry(decimal.Profile{}, testLimits(), clock); kind(err) != "invalid" {
		t.Fatal("profile")
	}
	if _, err := NewRegistry(testProfile(t), testLimits(), nil); kind(err) != "invalid" {
		t.Fatal("clock")
	}
	cases := map[string]func(*Limits){
		"keys": func(l *Limits) { l.Keys = 0 }, "keys high": func(l *Limits) { l.Keys = 1000001 },
		"revisions": func(l *Limits) { l.Revisions = 0 }, "revisions high": func(l *Limits) { l.Revisions = 1001 },
		"policies": func(l *Limits) { l.Policies = 0 }, "policies high": func(l *Limits) { l.Policies = 100001 },
		"versions": func(l *Limits) { l.Versions = 0 }, "versions high": func(l *Limits) { l.Versions = 10001 },
		"needs": func(l *Limits) { l.Needs = 0 }, "needs high": func(l *Limits) { l.Needs = 10001 },
		"snapshots": func(l *Limits) { l.Snapshots = 0 }, "snapshots high": func(l *Limits) { l.Snapshots = 100001 },
		"lines": func(l *Limits) { l.Lines = 0 }, "lines high": func(l *Limits) { l.Lines = 100001 },
		"digits": func(l *Limits) { l.Digits = 0 }, "digits high": func(l *Limits) { l.Digits = 10001 },
		"page": func(l *Limits) { l.Page = 0 }, "page high": func(l *Limits) { l.Page = 1001 },
	}
	for name, change := range cases {
		limits := testLimits()
		change(&limits)
		if _, err := NewRegistry(testProfile(t), limits, clock); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	top := Limits{Keys: 1000000, Revisions: 1000, Policies: 100000, Versions: 10000, Needs: 10000, Snapshots: 100000, Lines: 100000, Digits: 10000, Page: 1000}
	if _, err := NewRegistry(testProfile(t), top, clock); err != nil {
		t.Fatalf("bounds: %v", err)
	}
	low := Limits{Keys: 1, Revisions: 1, Policies: 1, Versions: 1, Needs: 1, Snapshots: 1, Lines: 1, Digits: 1, Page: 1}
	if _, err := NewRegistry(testProfile(t), low, clock); err != nil {
		t.Fatalf("ones: %v", err)
	}
}

func TestRecordObservation(t *testing.T) {
	r, _ := open(t)
	good := ObservationInput{Key: key("USD", "KRW", "2026-11-10"), Revision: 1, Unit: 1, Value: "1350.00", Origin: OriginFeed, Recorder: "loader"}
	view, err := r.RecordObservation("ws", good)
	if err != nil || view.Value != "1350" || view.State != ObservationRecorded || !view.RecordedAt.Equal(start) || view.Approval != nil || view.Void != nil {
		t.Fatalf("recorded: %v %+v", err, view)
	}
	repeat := good
	repeat.Value = "1350"
	if _, err := r.RecordObservation("ws", repeat); err != nil {
		t.Fatalf("a repeat in another print of the same value: %v", err)
	}
	change := func(apply func(*ObservationInput)) ObservationInput {
		in := good
		apply(&in)
		return in
	}
	cases := []struct {
		name      string
		workspace string
		in        ObservationInput
		want      string
	}{
		{"workspace", "", good, "invalid"},
		{"source", "ws", change(func(i *ObservationInput) { i.Key.Source = "b ank" }), "invalid"},
		{"series", "ws", change(func(i *ObservationInput) { i.Key.Series = "" }), "invalid"},
		{"base", "ws", change(func(i *ObservationInput) { i.Key.Base = "usd" }), "invalid"},
		{"quote", "ws", change(func(i *ObservationInput) { i.Key.Quote = "KRWX" }), "invalid"},
		{"one currency", "ws", change(func(i *ObservationInput) { i.Key.Quote = "USD" }), "invalid"},
		{"date", "ws", change(func(i *ObservationInput) { i.Key.Date = "2026-02-30" }), "invalid"},
		{"date form", "ws", change(func(i *ObservationInput) { i.Key.Date = "2026-1-10" }), "invalid"},
		{"date early", "ws", change(func(i *ObservationInput) { i.Key.Date = "1999-12-31" }), "invalid"},
		{"date late", "ws", change(func(i *ObservationInput) { i.Key.Date = "2100-01-01" }), "invalid"},
		{"revision zero", "ws", change(func(i *ObservationInput) { i.Revision = 0 }), "invalid"},
		{"revision high", "ws", change(func(i *ObservationInput) { i.Revision = 1001 }), "invalid"},
		{"unit", "ws", change(func(i *ObservationInput) { i.Unit = 5 }), "invalid"},
		{"value zero", "ws", change(func(i *ObservationInput) { i.Value = "0" }), "invalid"},
		{"value negative", "ws", change(func(i *ObservationInput) { i.Value = "-1" }), "invalid"},
		{"value text", "ws", change(func(i *ObservationInput) { i.Value = "1,350" }), "invalid"},
		{"value long", "ws", change(func(i *ObservationInput) { i.Value = strings.Repeat("1", 65) }), "invalid"},
		{"origin", "ws", change(func(i *ObservationInput) { i.Origin = "api" }), "invalid"},
		{"evidence on a feed", "ws", change(func(i *ObservationInput) { i.Evidence = "doc" }), "invalid"},
		{"no evidence on a manual", "ws", change(func(i *ObservationInput) { i.Origin = OriginManual }), "invalid"},
		{"no evidence on a file", "ws", change(func(i *ObservationInput) { i.Origin = OriginFile }), "invalid"},
		{"evidence form", "ws", change(func(i *ObservationInput) { i.Origin, i.Evidence = OriginFile, "d oc" }), "invalid"},
		{"recorder", "ws", change(func(i *ObservationInput) { i.Recorder = "" }), "invalid"},
		{"another value", "ws", change(func(i *ObservationInput) { i.Value = "1351" }), "conflict"},
		{"another recorder", "ws", change(func(i *ObservationInput) { i.Recorder = "other" }), "conflict"},
		{"a gap", "ws", change(func(i *ObservationInput) { i.Revision = 3 }), "rejected"},
		{"a new key not at one", "ws", change(func(i *ObservationInput) { i.Revision, i.Key.Date = 2, "2026-11-11" }), "not_found"},
		{"another workspace", "other", change(func(i *ObservationInput) { i.Revision = 2 }), "not_found"},
		{"form before the lookup", "ws", change(func(i *ObservationInput) { i.Revision, i.Key.Date, i.Unit = 2, "2026-11-11", 3 }), "invalid"},
	}
	for _, c := range cases {
		if _, err := r.RecordObservation(c.workspace, c.in); kind(err) != c.want {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	for revision := 2; revision <= 3; revision++ {
		next := good
		next.Revision, next.Value = revision, "1360"
		if _, err := r.RecordObservation("ws", next); err != nil {
			t.Fatal(err)
		}
	}
	full := good
	full.Revision = 4
	if _, err := r.RecordObservation("ws", full); kind(err) != "limit" {
		t.Fatalf("a full key: %v", err)
	}
	small, _ := NewRegistry(testProfile(t), Limits{Keys: 1, Revisions: 3, Policies: 1, Versions: 1, Needs: 1, Snapshots: 1, Lines: 1, Digits: 60, Page: 1}, func() time.Time { return start })
	if _, err := small.RecordObservation("ws", good); err != nil {
		t.Fatal(err)
	}
	other := good
	other.Key.Date = "2026-11-11"
	if _, err := small.RecordObservation("ws", other); kind(err) != "limit" {
		t.Fatalf("a full workspace: %v", err)
	}
	if _, err := small.RecordObservation("ws2", other); err != nil {
		t.Fatalf("another workspace has its own room: %v", err)
	}
	revisions, err := r.Observation("ws", good.Key)
	if err != nil || len(revisions) != 3 || revisions[2].Value != "1360" || revisions[0].Value != "1350" {
		t.Fatalf("read: %v %+v", err, revisions)
	}
}

func TestApprovalsAndVoids(t *testing.T) {
	r, _ := open(t)
	k := key("USD", "KRW", "2026-11-10")
	feed(t, r, k, 1, 1, "1350")
	manual(t, r, k, 2, 1, "1351")
	for name, c := range map[string]struct {
		workspace string
		k         Key
		revision  int
		approver  string
		ref       string
		want      string
	}{
		"workspace": {"", k, 2, "manager", "ok", "invalid"}, "key": {"ws", key("USD", "USD", "2026-11-10"), 2, "manager", "ok", "invalid"},
		"revision": {"ws", k, 0, "manager", "ok", "invalid"}, "approver": {"ws", k, 2, "", "ok", "invalid"}, "reference": {"ws", k, 2, "manager", "o k", "invalid"},
		"no revision": {"ws", k, 3, "manager", "ok", "not_found"}, "no key": {"ws", key("EUR", "KRW", "2026-11-10"), 1, "manager", "ok", "not_found"},
		"another workspace": {"other", k, 2, "manager", "ok", "not_found"}, "a feed": {"ws", k, 1, "manager", "ok", "rejected"},
		"the recorder": {"ws", k, 2, "clerk", "ok", "rejected"},
	} {
		if _, err := r.RecordObservationApproval(c.workspace, c.k, c.revision, c.approver, c.ref); kind(err) != c.want {
			t.Errorf("approval %s: %v", name, err)
		}
	}
	view, err := r.RecordObservationApproval("ws", k, 2, "manager", "ok")
	if err != nil || view.State != ObservationApproved || view.Approval.Approver != "manager" || !view.Approval.At.Equal(start) {
		t.Fatalf("approved: %v %+v", err, view)
	}
	if _, err := r.RecordObservationApproval("ws", k, 2, "manager", "ok"); err != nil {
		t.Fatalf("a repeat: %v", err)
	}
	if _, err := r.RecordObservationApproval("ws", k, 2, "boss", "ok"); kind(err) != "conflict" {
		t.Fatalf("another approver: %v", err)
	}
	if _, err := r.RecordObservationApproval("ws", k, 2, "manager", "ok-2"); kind(err) != "conflict" {
		t.Fatalf("another reference: %v", err)
	}
	for name, c := range map[string][4]string{"voider": {"", "doc", "mistake", "invalid"}, "evidence": {"clerk", "", "mistake", "invalid"}, "reason": {"clerk", "doc", "typo", "invalid"}} {
		if _, err := r.VoidObservation("ws", k, 2, c[0], c[1], c[2]); kind(err) != c[3] {
			t.Errorf("void %s: %v", name, err)
		}
	}
	if _, err := r.VoidObservation("ws", k, 9, "clerk", "doc", "mistake"); kind(err) != "not_found" {
		t.Fatal("void of nothing")
	}
	if _, err := r.RecordVoidApproval("ws", k, 2, "manager", "ok"); kind(err) != "rejected" {
		t.Fatalf("approval of no void: %v", err)
	}
	voided, err := r.VoidObservation("ws", k, 2, "clerk", "doc", "duplicate")
	if err != nil || voided.State != ObservationVoidAwaiting || voided.Void.Reason != "duplicate" || voided.Approval == nil {
		t.Fatalf("void: %v %+v", err, voided)
	}
	if _, err := r.VoidObservation("ws", k, 2, "clerk", "doc", "duplicate"); err != nil {
		t.Fatal("a repeat of a void")
	}
	for _, c := range [][3]string{{"other", "doc", "duplicate"}, {"clerk", "doc-2", "duplicate"}, {"clerk", "doc", "mistake"}} {
		if _, err := r.VoidObservation("ws", k, 2, c[0], c[1], c[2]); kind(err) != "conflict" {
			t.Errorf("another void %v: %v", c, err)
		}
	}
	if _, err := r.RecordObservationApproval("ws", k, 2, "manager", "ok"); err != nil {
		t.Fatal("a repeat of an approval on a voided revision")
	}
	for _, c := range [][3]string{{"", "ok", "invalid"}, {"manager", "", "invalid"}, {"clerk", "ok", "rejected"}} {
		if _, err := r.RecordVoidApproval("ws", k, 2, c[0], c[1]); kind(err) != c[2] {
			t.Errorf("void approval %v: %v", c, err)
		}
	}
	done, err := r.RecordVoidApproval("ws", k, 2, "manager", "ok-v")
	if err != nil || done.State != ObservationVoid || done.Void.Approval.Reference != "ok-v" {
		t.Fatalf("void approved: %v %+v", err, done)
	}
	if _, err := r.RecordVoidApproval("ws", k, 2, "manager", "ok-v"); err != nil {
		t.Fatal("a repeat of a void approval")
	}
	if _, err := r.RecordVoidApproval("ws", k, 2, "boss", "ok-v"); kind(err) != "conflict" {
		t.Fatal("another void approval")
	}
	if _, err := r.RecordVoidApproval("ws", k, 9, "boss", "ok-v"); kind(err) != "not_found" {
		t.Fatal("void approval of nothing")
	}
	read, _ := r.Observation("ws", k)
	read[1].Void.Approval.Approver = "changed"
	read[1].Approval.Approver = "changed"
	again, _ := r.Observation("ws", k)
	if again[1].Void.Approval.Approver != "manager" || again[1].Approval.Approver != "manager" {
		t.Fatal("a read shares its records")
	}
	if _, err := r.Observation("ws", key("EUR", "KRW", "2026-11-10")); kind(err) != "not_found" {
		t.Fatal("read of nothing")
	}
	if _, err := r.Observation("", k); kind(err) != "invalid" {
		t.Fatal("read form")
	}
}

func TestRecordPolicyVersion(t *testing.T) {
	r, _ := open(t)
	a := attrs(LevelContract, "con-1", "USD", "KRW")
	good := dated(1, "2026-01-01", ConstructionDirect)
	view, err := r.RecordPolicyVersion("ws", "pol", a, good)
	if err != nil || view.State != VersionRecorded || view.Spread != "0" {
		t.Fatalf("recorded: %v %+v", err, view)
	}
	change := func(apply func(*VersionInput)) VersionInput {
		in := good
		apply(&in)
		return in
	}
	attrsWith := func(apply func(*PolicyAttributes)) PolicyAttributes {
		made := a
		apply(&made)
		return made
	}
	cases := []struct {
		name  string
		id    string
		attrs PolicyAttributes
		in    VersionInput
		want  string
	}{
		{"id", "p ol", a, good, "invalid"},
		{"level", "pol", attrsWith(func(p *PolicyAttributes) { p.Level = "group" }), good, "invalid"},
		{"msp with a key", "pol", attrsWith(func(p *PolicyAttributes) { p.Level = LevelMSP }), good, "invalid"},
		{"contract without a key", "pol", attrsWith(func(p *PolicyAttributes) { p.Key = "" }), good, "invalid"},
		{"purpose", "pol", attrsWith(func(p *PolicyAttributes) { p.Purpose = "erp" }), good, "invalid"},
		{"one currency", "pol", attrsWith(func(p *PolicyAttributes) { p.Target = "USD" }), good, "invalid"},
		{"version zero", "pol", a, change(func(v *VersionInput) { v.Version = 0 }), "invalid"},
		{"version high", "pol", a, change(func(v *VersionInput) { v.Version = 10001 }), "invalid"},
		{"effective", "pol", a, change(func(v *VersionInput) { v.Effective = "2026-13-01" }), "invalid"},
		{"author", "pol", a, change(func(v *VersionInput) { v.Author = "" }), "invalid"},
		{"method", "pol", a, change(func(v *VersionInput) { v.Method = "spot" }), "invalid"},
		{"rate source", "pol", a, change(func(v *VersionInput) { v.RateSource = "" }), "invalid"},
		{"series", "pol", a, change(func(v *VersionInput) { v.Series = "" }), "invalid"},
		{"construction", "pol", a, change(func(v *VersionInput) { v.Construction = "chain" }), "invalid"},
		{"pivot on direct", "pol", a, change(func(v *VersionInput) { v.Pivot = "EUR" }), "invalid"},
		{"cross without pivot", "pol", a, change(func(v *VersionInput) { v.Construction = ConstructionCross }), "invalid"},
		{"pivot the source", "pol", a, change(func(v *VersionInput) { v.Construction, v.Pivot = ConstructionCross, "USD" }), "invalid"},
		{"pivot the target", "pol", a, change(func(v *VersionInput) { v.Construction, v.Pivot = ConstructionCross, "KRW" }), "invalid"},
		{"pivot form", "pol", a, change(func(v *VersionInput) { v.Construction, v.Pivot = ConstructionCross, "eur" }), "invalid"},
		{"rule", "pol", a, change(func(v *VersionInput) { v.Rule = "latest" }), "invalid"},
		{"hold with days", "pol", a, change(func(v *VersionInput) { v.MaxDays = 1 }), "invalid"},
		{"previous without days", "pol", a, change(func(v *VersionInput) { v.Rule = RulePreviousRecorded }), "invalid"},
		{"previous beyond days", "pol", a, change(func(v *VersionInput) { v.Rule, v.MaxDays = RulePreviousRecorded, 32 }), "invalid"},
		{"month on dated", "pol", a, change(func(v *VersionInput) { v.Month = MonthSame }), "invalid"},
		{"averaging on dated", "pol", a, change(func(v *VersionInput) { v.Averaging = Averaging }), "invalid"},
		{"least days on dated", "pol", a, change(func(v *VersionInput) { v.LeastDays = 1 }), "invalid"},
		{"unit on dated", "pol", a, change(func(v *VersionInput) { v.Unit = 1 }), "invalid"},
		{"value on dated", "pol", a, change(func(v *VersionInput) { v.Value = "1" }), "invalid"},
		{"evidence on dated", "pol", a, change(func(v *VersionInput) { v.Evidence = "doc" }), "invalid"},
		{"spread one", "pol", a, change(func(v *VersionInput) { v.Spread = "1" }), "invalid"},
		{"spread negative", "pol", a, change(func(v *VersionInput) { v.Spread = "-0.01" }), "invalid"},
		{"spread empty", "pol", a, change(func(v *VersionInput) { v.Spread = "" }), "invalid"},
		{"fee one", "pol", a, change(func(v *VersionInput) { v.Fee = "1" }), "invalid"},
		{"quote scale", "pol", a, change(func(v *VersionInput) { v.QuoteScale = 9 }), "invalid"},
		{"quote scale negative", "pol", a, change(func(v *VersionInput) { v.QuoteScale = -1 }), "invalid"},
		{"quote mode", "pol", a, change(func(v *VersionInput) { v.QuoteMode = "half_down" }), "invalid"},
		{"tolerance zero", "pol", a, change(func(v *VersionInput) { v.Tolerance = "0" }), "invalid"},
		{"tolerance high", "pol", a, change(func(v *VersionInput) { v.Tolerance = "0.0100001" }), "invalid"},
		{"quantum", "pol", a, change(func(v *VersionInput) { v.Quantum = "0.5" }), "invalid"},
		{"amount mode", "pol", a, change(func(v *VersionInput) { v.AmountMode = "" }), "invalid"},
		{"order", "pol", a, change(func(v *VersionInput) { v.Order = "convert_first" }), "invalid"},
		{"another policy at the place", "pol2", a, good, "conflict"},
		{"other attributes", "pol", attrsWith(func(p *PolicyAttributes) { p.Purpose = "display" }), change(func(v *VersionInput) { v.Version = 2; v.Effective = "2026-02-01" }), "conflict"},
		{"another version one", "pol", a, change(func(v *VersionInput) { v.Spread = "0.1" }), "conflict"},
		{"a gap", "pol", a, change(func(v *VersionInput) { v.Version = 3; v.Effective = "2026-02-01" }), "rejected"},
		{"an effective date not later", "pol", a, change(func(v *VersionInput) { v.Version = 2 }), "rejected"},
		{"a new policy not at one", "pol3", attrsWith(func(p *PolicyAttributes) { p.Key = "con-3" }), change(func(v *VersionInput) { v.Version = 2 }), "not_found"},
	}
	for _, c := range cases {
		if _, err := r.RecordPolicyVersion("ws", c.id, c.attrs, c.in); kind(err) != c.want {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	for _, c := range []struct {
		level, method string
		want          string
	}{{LevelMSP, MethodEnded, "invalid"}, {LevelMSP, MethodDefaultAllowed, "invalid"}, {LevelCustomer, MethodDefaultAllowed, "invalid"}, {LevelCustomer, MethodEnded, "ok"}, {LevelContract, MethodEnded, "ok"}} {
		keyed := "x-" + c.level + c.method
		if c.level == LevelMSP {
			keyed = ""
		}
		_, err := r.RecordPolicyVersion("ws", "m-"+c.level+"-"+c.method, PolicyAttributes{Level: c.level, Key: keyed, Purpose: "billing", Source: "EUR", Target: "KRW"},
			VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: c.method})
		if kind(err) != c.want {
			t.Errorf("%s at %s: %v", c.method, c.level, err)
		}
	}
	if _, err := r.RecordPolicyVersion("ws", "ended-with", attrs(LevelContract, "con-9", "EUR", "KRW"), VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodEnded, Spread: "0"}); kind(err) != "invalid" {
		t.Fatalf("an ended version with a field: %v", err)
	}
	fixedOn := func(apply func(*VersionInput)) VersionInput {
		in := pricing(VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodFixed, Construction: ConstructionDirect, Unit: 1, Value: "1300", Evidence: "doc"})
		apply(&in)
		return in
	}
	for name, in := range map[string]VersionInput{
		"no evidence": fixedOn(func(v *VersionInput) { v.Evidence = "" }), "no value": fixedOn(func(v *VersionInput) { v.Value = "0" }),
		"a unit": fixedOn(func(v *VersionInput) { v.Unit = 3 }), "a cross": fixedOn(func(v *VersionInput) { v.Construction, v.Pivot = ConstructionCross, "USD" }),
		"a source": fixedOn(func(v *VersionInput) { v.RateSource = "bank" }), "a rule": fixedOn(func(v *VersionInput) { v.Rule = RuleHold }),
		"a month": fixedOn(func(v *VersionInput) { v.Month = MonthSame }),
	} {
		if _, err := r.RecordPolicyVersion("ws", "fx-"+strings.ReplaceAll(name, " ", "-"), attrs(LevelContract, "con-8", "EUR", "KRW"), in); kind(err) != "invalid" {
			t.Errorf("fixed %s: %v", name, err)
		}
	}
	monthly := func(apply func(*VersionInput)) VersionInput {
		in := pricing(VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodMonthlyAverage, RateSource: "bank", Series: "deal", Construction: ConstructionDirect, Month: MonthSame, Averaging: Averaging, LeastDays: 1})
		apply(&in)
		return in
	}
	for name, in := range map[string]VersionInput{
		"month": monthly(func(v *VersionInput) { v.Month = "next_month" }), "averaging": monthly(func(v *VersionInput) { v.Averaging = "median" }),
		"no days": monthly(func(v *VersionInput) { v.LeastDays = 0 }), "too many days": monthly(func(v *VersionInput) { v.LeastDays = 32 }),
		"a rule": monthly(func(v *VersionInput) { v.Rule = RuleHold }), "days of a rule": monthly(func(v *VersionInput) { v.MaxDays = 1 }),
	} {
		if _, err := r.RecordPolicyVersion("ws", "av-"+strings.ReplaceAll(name, " ", "-"), attrs(LevelContract, "con-7", "EUR", "KRW"), in); kind(err) != "invalid" {
			t.Errorf("monthly %s: %v", name, err)
		}
	}
	for version := 2; version <= 4; version++ {
		next := good
		next.Version, next.Effective = version, "2026-0"+string(rune('0'+version))+"-01"
		if _, err := r.RecordPolicyVersion("ws", "pol", a, next); err != nil {
			t.Fatalf("version %d: %v", version, err)
		}
	}
	fifth := good
	fifth.Version, fifth.Effective = 5, "2026-09-01"
	if _, err := r.RecordPolicyVersion("ws", "pol", a, fifth); kind(err) != "limit" {
		t.Fatalf("a full policy: %v", err)
	}
	read, err := r.Policy("ws", "pol")
	if err != nil || read.Attributes != a || len(read.Versions) != 4 || read.Versions[3].Effective != "2026-04-01" {
		t.Fatalf("read: %v %+v", err, read)
	}
	if _, err := r.Policy("ws", "none"); kind(err) != "not_found" {
		t.Fatal("read of nothing")
	}
	if _, err := r.Policy("ws", ""); kind(err) != "invalid" {
		t.Fatal("read form")
	}
	small, _ := NewRegistry(testProfile(t), Limits{Keys: 1, Revisions: 1, Policies: 1, Versions: 1, Needs: 1, Snapshots: 1, Lines: 1, Digits: 60, Page: 1}, func() time.Time { return start })
	if _, err := small.RecordPolicyVersion("ws", "a", a, good); err != nil {
		t.Fatal(err)
	}
	if _, err := small.RecordPolicyVersion("ws", "b", attrs(LevelContract, "con-2", "USD", "KRW"), good); kind(err) != "limit" {
		t.Fatalf("a full workspace: %v", err)
	}
}

func TestPolicyApprovalAndWithdrawal(t *testing.T) {
	r, _ := open(t)
	fixed := pricing(VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodFixed, Construction: ConstructionDirect, Unit: 1, Value: "1300", Evidence: "doc"})
	policyVersion(t, r, "fix", attrs(LevelContract, "con-1", "USD", "KRW"), fixed)
	policyVersion(t, r, "dat", attrs(LevelContract, "con-2", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	policyVersion(t, r, "gone", attrs(LevelContract, "con-3", "USD", "KRW"), fixed)
	for name, c := range map[string]struct {
		id       string
		version  int
		approver string
		ref      string
		want     string
	}{
		"id": {"", 1, "manager", "ok", "invalid"}, "version": {"fix", 0, "manager", "ok", "invalid"}, "approver": {"fix", 1, "", "ok", "invalid"},
		"no policy": {"none", 1, "manager", "ok", "not_found"}, "no version": {"fix", 2, "manager", "ok", "not_found"},
		"not fixed": {"dat", 1, "manager", "ok", "rejected"}, "the author": {"fix", 1, "author", "ok", "rejected"},
	} {
		if _, err := r.RecordPolicyApproval("ws", c.id, c.version, c.approver, c.ref); kind(err) != c.want {
			t.Errorf("approval %s: %v", name, err)
		}
	}
	if _, err := r.WithdrawPolicyVersion("ws", "gone", 1, "manager", "mistake"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RecordPolicyApproval("ws", "gone", 1, "manager", "ok"); kind(err) != "rejected" {
		t.Fatalf("approval of a withdrawn version: %v", err)
	}
	view, err := r.RecordPolicyApproval("ws", "fix", 1, "manager", "ok")
	if err != nil || view.State != VersionApproved || view.Approval.Reference != "ok" {
		t.Fatalf("approved: %v %+v", err, view)
	}
	if _, err := r.RecordPolicyApproval("ws", "fix", 1, "manager", "ok"); err != nil {
		t.Fatal("repeat")
	}
	if _, err := r.RecordPolicyApproval("ws", "fix", 1, "manager", "ok-2"); kind(err) != "conflict" {
		t.Fatal("another reference")
	}
	for name, c := range map[string][3]string{"withdrawer": {"", "mistake", "invalid"}, "reason": {"manager", "typo", "invalid"}} {
		if _, err := r.WithdrawPolicyVersion("ws", "fix", 1, c[0], c[1]); kind(err) != c[2] {
			t.Errorf("withdraw %s: %v", name, err)
		}
	}
	if _, err := r.WithdrawPolicyVersion("ws", "fix", 2, "manager", "mistake"); kind(err) != "not_found" {
		t.Fatal("withdraw nothing")
	}
	done, err := r.WithdrawPolicyVersion("ws", "fix", 1, "manager", "replaced")
	if err != nil || done.Withdrawal.Reason != "replaced" || done.Approval == nil {
		t.Fatalf("withdrawn: %v %+v", err, done)
	}
	if _, err := r.WithdrawPolicyVersion("ws", "fix", 1, "manager", "replaced"); err != nil {
		t.Fatal("repeat of a withdrawal")
	}
	for _, c := range [][2]string{{"boss", "replaced"}, {"manager", "mistake"}} {
		if _, err := r.WithdrawPolicyVersion("ws", "fix", 1, c[0], c[1]); kind(err) != "conflict" {
			t.Errorf("another withdrawal %v: %v", c, err)
		}
	}
	read, _ := r.Policy("ws", "fix")
	read.Versions[0].Approval.Approver = "changed"
	read.Versions[0].Withdrawal.Reason = "changed"
	again, _ := r.Policy("ws", "fix")
	if again.Versions[0].Approval.Approver != "manager" || again.Versions[0].Withdrawal.Reason != "replaced" {
		t.Fatal("a read shares its records")
	}
}

func TestConversionTable(t *testing.T) {
	r, _ := open(t)
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	policyVersion(t, r, "eur", attrs(LevelContract, "con-1", "EUR", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	prepare(t, r, "run", need("rate", "USD", "KRW", "2026-11-10"), need("held", "EUR", "KRW", "2026-11-10"), need("same", "KRW", "KRW", "2026-11-10"))
	ok := []struct {
		line Line
		want string
	}{
		{line("a", "same", "50000", "KRW", StateSource), "a:not_converted_same_currency:50000 KRW"},
		{line("a", "same", "50000.00", "KRW", StateAlreadyInTarget), "a:not_converted_same_currency:50000.00 KRW"},
		{line("a", "rate", "1215000", "KRW", StateAlreadyInTarget), "a:not_converted_already_in_target:1215000 KRW"},
		{line("a", "held", "-7", "KRW", StateAlreadyInTarget), "a:not_converted_already_in_target:-7 KRW"},
		{line("a", "held", "10", "EUR", StateSource), "a:held:no_rate_recorded"},
		{line("a", "rate", "1", "USD", StateSource), "a=1350 KRW fee 0 total 1350 quoted 1350"},
	}
	for _, c := range ok {
		if got := shown(convert(t, r, "run", c.line)); got != c.want {
			t.Errorf("%+v: %s", c.line, got)
		}
	}
	rejected := []Line{
		line("a", "same", "1", "USD", StateSource),
		line("a", "same", "1", "USD", StateAlreadyInTarget),
		line("a", "rate", "1", "KRW", StateSource),
		line("a", "rate", "1", "USD", StateAlreadyInTarget),
		line("a", "rate", "1", "EUR", StateSource),
		line("a", "held", "1", "KRW", StateSource),
		line("a", "held", "1", "EUR", StateAlreadyInTarget),
	}
	for _, l := range rejected {
		if _, err := r.Convert("ws", "run", []Line{line("ok", "rate", "1", "USD", StateSource), l}); kind(err) != "rejected" {
			t.Errorf("%+v: %v", l, err)
		}
	}
	nine := []Line{}
	for index := 0; index < 9; index++ {
		nine = append(nine, line(string(rune('a'+index)), "none", "x", "usd", "no"))
	}
	cases := []struct {
		name      string
		workspace string
		id        string
		lines     []Line
		want      string
	}{
		{"too many before their form", "", "run", nine, "limit"},
		{"workspace", "", "run", []Line{line("a", "rate", "1", "USD", StateSource)}, "invalid"},
		{"snapshot", "ws", "r un", []Line{line("a", "rate", "1", "USD", StateSource)}, "invalid"},
		{"line id", "ws", "run", []Line{line("", "rate", "1", "USD", StateSource)}, "invalid"},
		{"line twice", "ws", "run", []Line{line("a", "rate", "1", "USD", StateSource), line("a", "rate", "2", "USD", StateSource)}, "invalid"},
		{"need form", "ws", "run", []Line{line("a", "", "1", "USD", StateSource)}, "invalid"},
		{"amount", "ws", "run", []Line{line("a", "rate", "1e3", "USD", StateSource)}, "invalid"},
		{"currency", "ws", "run", []Line{line("a", "rate", "1", "usd", StateSource)}, "invalid"},
		{"state", "ws", "run", []Line{line("a", "rate", "1", "USD", "converted")}, "invalid"},
		{"form before the lookup", "ws", "none", []Line{line("a", "rate", "1", "USD", "x")}, "invalid"},
		{"no snapshot", "ws", "none", []Line{line("a", "rate", "1", "USD", StateSource)}, "not_found"},
		{"another workspace", "other", "run", []Line{line("a", "rate", "1", "USD", StateSource)}, "not_found"},
		{"no need", "ws", "run", []Line{line("a", "none", "1", "USD", StateSource)}, "not_found"},
		{"not found before rejected", "ws", "run", []Line{line("a", "same", "1", "USD", StateSource), line("b", "none", "1", "USD", StateSource)}, "not_found"},
		{"rejected before a limit", "ws", "run", []Line{line("a", "rate", "99999999999999", "USD", StateSource), line("b", "same", "1", "USD", StateSource)}, "rejected"},
		{"an amount the profile cannot hold", "ws", "run", []Line{line("a", "rate", "99999999999999", "USD", StateSource)}, "limit"},
		{"none", "ws", "run", []Line{}, "ok"},
	}
	for _, c := range cases {
		if _, err := r.Convert(c.workspace, c.id, c.lines); kind(err) != c.want {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func printed(fields ...string) string {
	sum := sha256.New()
	for _, field := range fields {
		sum.Write([]byte{0, 0, 0, 0, 0, 0, 0, byte(len(field))})
		sum.Write([]byte(field))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func TestDigestsArePrintsOfEveryField(t *testing.T) {
	r, _ := open(t)
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	snap := prepare(t, r, "run", need("n", "USD", "KRW", "2026-11-10"), need("s", "KRW", "KRW", "2026-11-10"))
	fields := []string{"fx_snapshot_content_v1", "ws", "2",
		"n", "billing", "USD", "KRW", "cust-1", "con-1", "2026-11-10", "0",
		"rate", "contract", "contract", "usd", "1", "", "0",
		"dated", "direct", "", "", "", "0", "hold", "0",
		"1", "bank", "deal", "USD", "KRW", "2026-11-10", "1", "1", "1350", "feed", "", "",
		"0", "", "", "", "", "0", "0", "1350", "1", "1350", "0", "1350", "1350", "2",
		"half_even", "0.0001", "0", "1350", "0", "1", "half_even", Order,
		"s", "billing", "KRW", "KRW", "cust-1", "con-1", "2026-11-10", "0",
		"same_currency", "", "", "", "0", "", "0", "", "", "", "", "", "0", "", "0", "0", "0", "", "", "", "", "0", "0", "", "0", "", "", "", "", "0"}
	fields = append(fields, make([]string, 8)...)
	if want := printed(fields...); snap.ContentDigest != want {
		t.Fatalf("content digest: %s, built %s", snap.ContentDigest, want)
	}
	if snap.Digest != printed("fx_snapshot_v1", "run", snap.ContentDigest) {
		t.Fatal("snapshot digest")
	}
	base := snap.Entries
	seen := map[string]string{snap.ContentDigest: "the base"}
	changes := map[string]func(*Entry){
		"need id": func(e *Entry) { e.Need.ID = "m" }, "offset": func(e *Entry) { e.Need.Offset = 540 }, "customer": func(e *Entry) { e.Need.Customer = "c2" },
		"kind": func(e *Entry) { e.Kind = EntryHeld }, "reason": func(e *Entry) { e.Reason = "msp_default" }, "version": func(e *Entry) { e.Version = 2 },
		"allowing": func(e *Entry) { e.AllowingPolicy = "x" }, "allowing version": func(e *Entry) { e.AllowingVersion = 1 },
		"pivot": func(e *Entry) { e.Pivot = "EUR" }, "least days": func(e *Entry) { e.LeastDays = 1 }, "max days": func(e *Entry) { e.MaxDays = 1 },
		"observation revision": func(e *Entry) {
			e.Observations = []Used{{Key: e.Observations[0].Key, Revision: 2, Unit: 1, Value: "1350", Origin: "feed"}}
		},
		"observation value": func(e *Entry) {
			e.Observations = []Used{{Key: e.Observations[0].Key, Revision: 1, Unit: 1, Value: "1351", Origin: "feed"}}
		},
		"observation reference": func(e *Entry) {
			e.Observations = []Used{{Key: e.Observations[0].Key, Revision: 1, Unit: 1, Value: "1350", Origin: "feed", Reference: "ok"}}
		},
		"no observation": func(e *Entry) { e.Observations = nil }, "fixed reference": func(e *Entry) { e.FixedReference = "ok" },
		"date used": func(e *Entry) { e.DateUsed = "2026-11-09" }, "elapsed": func(e *Entry) { e.DaysElapsed = 1 }, "voids": func(e *Entry) { e.VoidsPassed = 1 },
		"quoted": func(e *Entry) { e.QuotedRate = "1351" }, "unit": func(e *Entry) { e.Unit = 100 }, "per source": func(e *Entry) { e.PerSource = "1351" },
		"spread": func(e *Entry) { e.Spread = "0.01" }, "sales quote": func(e *Entry) { e.SalesQuote = "1351" }, "quote scale": func(e *Entry) { e.QuoteScale = 3 },
		"tolerance": func(e *Entry) { e.Tolerance = "0.001" }, "difference": func(e *Entry) { e.Difference = "1/2" }, "sales rate": func(e *Entry) { e.SalesRate = "1351" },
		"fee": func(e *Entry) { e.Fee = "0.1" }, "quantum": func(e *Entry) { e.Quantum = "0.01" }, "amount mode": func(e *Entry) { e.AmountMode = "floor" },
		"a byte moved": func(e *Entry) { e.Need.Customer, e.Need.Contract = "cust-1c", "on-1" },
	}
	for name, change := range changes {
		entries := copyEntries(base)
		change(&entries[0])
		got := contentDigest("ws", entries)
		if earlier, ok := seen[got]; ok {
			t.Errorf("%s has the digest of %s", name, earlier)
		}
		seen[got] = name
	}
	if contentDigest("other", base) == snap.ContentDigest || contentDigest("ws", base[:1]) == snap.ContentDigest {
		t.Fatal("workspace or count left out")
	}
}

func TestPrepareRefusals(t *testing.T) {
	r, _ := open(t)
	n := need("n", "USD", "KRW", "2026-11-10")
	nine := []Need{}
	for index := 0; index < 9; index++ {
		nine = append(nine, Need{})
	}
	change := func(apply func(*Need)) Need {
		made := n
		apply(&made)
		return made
	}
	cases := []struct {
		name      string
		workspace string
		id        string
		needs     []Need
		want      string
	}{
		{"too many before their form", "", "", nine, "limit"},
		{"workspace", "", "run", []Need{n}, "invalid"},
		{"id", "ws", "", []Need{n}, "invalid"},
		{"none", "ws", "run", nil, "invalid"},
		{"need id", "ws", "run", []Need{change(func(x *Need) { x.ID = "" })}, "invalid"},
		{"need twice", "ws", "run", []Need{n, n}, "invalid"},
		{"purpose", "ws", "run", []Need{change(func(x *Need) { x.Purpose = "erp" })}, "invalid"},
		{"source", "ws", "run", []Need{change(func(x *Need) { x.Source = "us" })}, "invalid"},
		{"target", "ws", "run", []Need{change(func(x *Need) { x.Target = "" })}, "invalid"},
		{"customer", "ws", "run", []Need{change(func(x *Need) { x.Customer = "" })}, "invalid"},
		{"contract", "ws", "run", []Need{change(func(x *Need) { x.Contract = "" })}, "invalid"},
		{"date", "ws", "run", []Need{change(func(x *Need) { x.Date = "2026-11-31" })}, "invalid"},
		{"offset low", "ws", "run", []Need{change(func(x *Need) { x.Offset = -721 })}, "invalid"},
		{"offset high", "ws", "run", []Need{change(func(x *Need) { x.Offset = 841 })}, "invalid"},
		{"offsets at their bounds", "ws", "run", []Need{change(func(x *Need) { x.Offset = -720 }), change(func(x *Need) { x.ID, x.Offset = "m", 840 })}, "ok"},
	}
	for _, c := range cases {
		if _, err := r.Prepare(c.workspace, c.id, c.needs); kind(err) != c.want {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	if _, err := r.Prepare("ws", "run", []Need{n}); kind(err) != "conflict" {
		t.Fatalf("other needs under an identifier: %v", err)
	}
	reordered := []Need{change(func(x *Need) { x.ID, x.Offset = "m", 840 }), change(func(x *Need) { x.Offset = -720 })}
	if _, err := r.Prepare("ws", "run", reordered); kind(err) != "conflict" {
		t.Fatalf("the same needs in another order: %v", err)
	}
	small, _ := NewRegistry(testProfile(t), Limits{Keys: 1, Revisions: 1, Policies: 1, Versions: 1, Needs: 1, Snapshots: 1, Lines: 1, Digits: 60, Page: 1}, func() time.Time { return start })
	if _, err := small.Prepare("ws", "a", []Need{n}); err != nil {
		t.Fatal(err)
	}
	if _, err := small.Prepare("ws", "a", []Need{n}); err != nil {
		t.Fatalf("a repeat in a full workspace: %v", err)
	}
	if _, err := small.Prepare("ws", "b", []Need{n}); kind(err) != "limit" {
		t.Fatalf("a full workspace: %v", err)
	}
	tiny, _ := NewRegistry(testProfile(t), Limits{Keys: 9, Revisions: 1, Policies: 9, Versions: 1, Needs: 2, Snapshots: 9, Lines: 1, Digits: 4, Page: 1}, func() time.Time { return start })
	policyVersion(t, tiny, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	feed(t, tiny, key("USD", "KRW", "2026-11-10"), 1, 1, "13500.5")
	if _, err := tiny.Prepare("ws", "a", []Need{need("s", "KRW", "KRW", "2026-11-10"), n}); kind(err) != "limit" {
		t.Fatalf("a fraction beyond the digits: %v", err)
	}
	if _, err := tiny.Snapshot("ws", "a"); kind(err) != "not_found" {
		t.Fatal("a refused prepare stored a snapshot")
	}
}

func TestSnapshotsAreCopiesAndPaged(t *testing.T) {
	r, _ := open(t)
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	for _, id := range []string{"c", "a", "d", "b"} {
		prepare(t, r, id, need("n", "USD", "KRW", "2026-11-10"))
	}
	snap, err := r.Snapshot("ws", "a")
	if err != nil || len(snap.Entries) != 1 || snap.Held != 0 || !snap.PreparedAt.Equal(start) {
		t.Fatalf("read: %v %+v", err, snap)
	}
	snap.Entries[0].SalesRate = "1"
	snap.Entries[0].Observations[0].Value = "1"
	again, _ := r.Snapshot("ws", "a")
	if again.Entries[0].SalesRate != "1350" || again.Entries[0].Observations[0].Value != "1350" {
		t.Fatal("a read shares its entries")
	}
	if got := convert(t, r, "a", line("x", "n", "1", "USD", StateSource)); got.Lines[0].Amount != "1350" {
		t.Fatal("a changed read changed a conversion")
	}
	list := func(after string, limit int) string {
		listed, err := r.Snapshots("ws", after, limit)
		if err != nil {
			return kind(err)
		}
		ids := []string{}
		for _, item := range listed {
			ids = append(ids, item.ID)
		}
		return strings.Join(ids, " ")
	}
	for _, c := range []struct {
		after string
		limit int
		want  string
	}{{"", 3, "a b c"}, {"b", 3, "c d"}, {"d", 1, ""}, {"", 4, "invalid"}, {"", 0, "invalid"}, {"b c", 1, "invalid"}} {
		if got := list(c.after, c.limit); got != c.want {
			t.Errorf("after %q limit %d: %q", c.after, c.limit, got)
		}
	}
	if listed, _ := r.Snapshots("other", "", 1); listed == nil || len(listed) != 0 {
		t.Fatal("another workspace")
	}
	if listed, _ := r.Snapshots("ws", "", 1); listed[0].Digest != again.Digest || listed[0].ContentDigest != again.ContentDigest {
		t.Fatal("digests in a page")
	}
	for _, c := range [][3]string{{"", "a", "invalid"}, {"ws", "", "invalid"}, {"ws", "z", "not_found"}, {"other", "a", "not_found"}} {
		if _, err := r.Snapshot(c[0], c[1]); kind(err) != c[2] {
			t.Errorf("snapshot %v: %v", c, err)
		}
	}
	if _, err := r.Snapshots("", "", 1); kind(err) != "invalid" {
		t.Fatal("snapshots form")
	}
}

func TestTheTenantBoundaryAndFixedErrors(t *testing.T) {
	r, _ := open(t)
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	if _, err := r.Prepare("other", "run", []Need{need("n", "USD", "KRW", "2026-11-10")}); err != nil {
		t.Fatal(err)
	}
	snap, _ := r.Snapshot("other", "run")
	if snap.Entries[0].Kind != EntryHeld || snap.Entries[0].Reason != ReasonContractNotStated {
		t.Fatalf("another workspace's policy and rate were used: %+v", snap.Entries[0])
	}
	mine := prepare(t, r, "run", need("n", "USD", "KRW", "2026-11-10"))
	if mine.ContentDigest == snap.ContentDigest {
		t.Fatal("two workspaces share a digest")
	}
	secret := "secret-name"
	_, a := r.RecordObservation(secret, ObservationInput{Key: key("USD", "KRW", secret)})
	_, b := r.Snapshot(secret, secret)
	_, c := r.Policy("ws", secret)
	for _, err := range []error{a, b, c} {
		if err == nil || strings.Contains(err.Error(), secret) || (err != ErrInvalid && err != ErrNotFound) {
			t.Fatalf("an error that is not a fixed value: %v", err)
		}
	}
}

func TestTheSourceUsesNoFloatingPointAndHasNoComments(t *testing.T) {
	for _, file := range []string{"types.go", "registry.go", "resolve.go", "snapshot.go", "convert.go"} {
		body, err := readSource(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, word := range []string{"float", "Float", "big.Rat", "//", "/*", "\"net", "os/exec"} {
			if strings.Contains(body, word) {
				t.Errorf("%s contains %q", file, word)
			}
		}
	}
}
