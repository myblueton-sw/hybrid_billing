package formula

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func printed(fields ...string) string {
	sum := sha256.New()
	for _, field := range fields {
		size := []byte{0, 0, 0, 0, 0, 0, 0, byte(len(field))}
		sum.Write(size)
		sum.Write([]byte(field))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func TestNewRegistry(t *testing.T) {
	clock := func() time.Time { return start }
	if _, err := NewRegistry(decimal.Profile{}, registryLimits(), clock); kind(err) != "invalid" {
		t.Fatalf("profile: %v", err)
	}
	if _, err := NewRegistry(testProfile(t), registryLimits(), nil); kind(err) != "invalid" {
		t.Fatalf("clock: %v", err)
	}
	cases := map[string]func(*RegistryLimits){
		"formula limits": func(l *RegistryLimits) { l.Depth = 0 },
		"sets zero":      func(l *RegistryLimits) { l.Sets = 0 }, "sets high": func(l *RegistryLimits) { l.Sets = 100001 },
		"revisions zero": func(l *RegistryLimits) { l.Revisions = 0 }, "revisions high": func(l *RegistryLimits) { l.Revisions = 1000000 },
		"history zero": func(l *RegistryLimits) { l.History = 0 }, "history high": func(l *RegistryLimits) { l.History = 1001 },
		"page zero": func(l *RegistryLimits) { l.Page = 0 }, "page high": func(l *RegistryLimits) { l.Page = 1001 },
	}
	for name, change := range cases {
		limits := registryLimits()
		change(&limits)
		if _, err := NewRegistry(testProfile(t), limits, clock); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	top := RegistryLimits{Limits: testLimits(), Sets: 100000, Revisions: 999999, History: 1000, Page: 1000}
	if _, err := NewRegistry(testProfile(t), top, clock); err != nil {
		t.Fatalf("limits at their bounds: %v", err)
	}
	low := RegistryLimits{Limits: testLimits(), Sets: 1, Revisions: 1, History: 1, Page: 1}
	if _, err := NewRegistry(testProfile(t), low, clock); err != nil {
		t.Fatalf("limits of one: %v", err)
	}
}

func TestTheContentDigestIsAPrintOfEveryField(t *testing.T) {
	r, _ := open(t)
	set := setOf(vars(decl("rate", number), decl("cost", usd)),
		fm("g", usdHour, con("-1.50", usdHour)),
		Formula{ID: "f", Target: "t", Rank: 12, Output: usd, Quantum: "0.01", Mode: "half_even", Root: op(NodeMul, vr("cost"), op(NodeAdd, ref("g2"), vr("rate")))})
	view, err := r.Draft("ws", "set", 1, "author", set)
	want := printed("formula_v1",
		"2", "cost", "money", "USD", "", "rate", "number", "", "",
		"2",
		"f", "t", "12", "money", "USD", "", "0.01", "half_even", "mul", "variable", "cost", "add", "reference", "g2", "variable", "rate",
		"g", "g", "1", "price", "USD", "hour", "0.01", "half_even", "constant", "-1.50", "price", "USD", "hour")
	if err != nil || view.Digest != want {
		t.Fatalf("digest: %v %s", err, view.Digest)
	}
	base := func() Set {
		return setOf(vars(decl("cost", usd), decl("rate", number)),
			fm("f", usd, op(NodeMul, vr("cost"), op(NodeAdd, con("1", number), vr("rate")))),
			fm("g", usd, op(NodeAdd, op(NodeAdd, vr("cost"), vr("cost")), vr("cost"))))
	}
	digest := func(number int, set Set) string {
		view, err := r.Draft("ws", "digests", number, "author", set)
		if err != nil {
			t.Fatalf("draft %d: %v", number, err)
		}
		return view.Digest
	}
	first := digest(1, base())
	reordered := base()
	reordered.Variables[0], reordered.Variables[1] = reordered.Variables[1], reordered.Variables[0]
	reordered.Formulas[0], reordered.Formulas[1] = reordered.Formulas[1], reordered.Formulas[0]
	if _, err := r.Draft("ws", "digests", 1, "author", reordered); err != nil {
		t.Fatalf("another order of declaration is the same content: %v", err)
	}
	changes := map[string]func(*Set){
		"a variable's name":     func(s *Set) { s.Variables = append(s.Variables, decl("extra", number)) },
		"a variable's kind":     func(s *Set) { s.Variables[1].Type = hour },
		"a variable's currency": func(s *Set) { s.Variables[0].Type = krw; s.Formulas[0].Output = krw },
		"a target":              func(s *Set) { s.Formulas[0].Target = "other" },
		"a rank":                func(s *Set) { s.Formulas[0].Rank = 2 },
		"an output":             func(s *Set) { s.Formulas[0].Output = krw },
		"a quantum":             func(s *Set) { s.Formulas[0].Quantum = "0.1" },
		"a mode":                func(s *Set) { s.Formulas[0].Mode = "floor" },
		"a kind of node":        func(s *Set) { s.Formulas[0].Root.Kind = NodeDiv },
		"a constant's text":     func(s *Set) { s.Formulas[0].Root.Operands[1].Operands[0].Number = "1.0" },
		"a constant's type":     func(s *Set) { s.Formulas[0].Root.Operands[1].Operands[0].Type = usd },
		"a variable in a node":  func(s *Set) { s.Formulas[0].Root.Operands[0].Name = "rate" },
		"the order of operands": func(s *Set) {
			s.Formulas[0].Root.Operands[0], s.Formulas[0].Root.Operands[1] = s.Formulas[0].Root.Operands[1], s.Formulas[0].Root.Operands[0]
		},
		"the shape of a tree":        func(s *Set) { s.Formulas[1].Root = op(NodeAdd, vr("cost"), op(NodeAdd, vr("cost"), vr("cost"))) },
		"a reference for a variable": func(s *Set) { s.Formulas[0].Root.Operands[0] = ref("cost") },
		"a byte moved to a neighbour": func(s *Set) {
			s.Formulas[0].ID, s.Formulas[0].Target = "fx", "f"
			s.Formulas[1].ID, s.Formulas[1].Target = "g", "xg"
		},
		"one formula fewer": func(s *Set) { s.Formulas = s.Formulas[:1] },
		"a variable named as a formula": func(s *Set) {
			s.Variables[1].Name = "f"
			s.Formulas[0].Root.Operands[1].Operands[1].Name = "f"
		},
	}
	seen := map[string]string{first: "the base"}
	number := 2
	other, _ := NewRegistry(testProfile(t), RegistryLimits{Limits: testLimits(), Sets: 3, Revisions: 100, History: 2, Page: 3}, func() time.Time { return start })
	for name, change := range changes {
		set := base()
		change(&set)
		view, err := other.Draft("ws", "digests", number, "author", set)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if earlier, ok := seen[view.Digest]; ok {
			t.Errorf("%s has the digest of %s", name, earlier)
		}
		seen[view.Digest] = name
		number++
	}
	if len(first) != 64 || first != strings.ToLower(first) {
		t.Fatalf("form of a digest: %s", first)
	}
}

func TestDraft(t *testing.T) {
	r, clock := open(t)
	view, err := r.Draft("ws", "set", 7, "author", plain("2"))
	want := RevisionView{Set: "set", Number: 7, State: "draft", Author: "author", Digest: view.Digest, Findings: []Finding{}, History: []ImpactRecord{}, DraftedAt: start}
	if err != nil || view.State != want.State || view.Set != want.Set || view.Number != 7 || view.Author != "author" || view.Impact != nil || view.Current ||
		view.Findings == nil || len(view.Findings) != 0 || view.History == nil || len(view.History) != 0 || view.Withdrawal != "" ||
		!view.DraftedAt.Equal(start) || !view.CheckedAt.IsZero() || !view.ActivatedAt.IsZero() || !view.SupersededAt.IsZero() || !view.WithdrawnAt.IsZero() {
		t.Fatalf("draft: %v %+v", err, view)
	}
	clock.add(time.Hour)
	again, err := r.Draft("ws", "set", 7, "author", plain("2"))
	if err != nil || !again.DraftedAt.Equal(start) {
		t.Fatalf("an identical repeat: %v", err)
	}
	if active, _ := r.Active("ws", "set"); active.Sequence != 1 {
		t.Fatalf("a repeat wrote: %d", active.Sequence)
	}
	deep := setOf(nil, fm("f", number, chain(5)))
	limits := registryLimits()
	limits.Nodes = 14
	small, _ := NewRegistry(testProfile(t), limits, clock.read)
	if _, err := small.Draft("ws", "set", 1, "author", setOf(nil, fm("f", number, op(NodeAdd, op(NodeAdd, chain(2), chain(2)), op(NodeAdd, chain(2), chain(2)))))); kind(err) != "limit" {
		t.Fatalf("too large: %v", err)
	}
	fifteen := fm("a", number, op(NodeAdd, op(NodeAdd, chain(2), chain(2)), op(NodeAdd, chain(2), chain(2))))
	if _, err := small.Draft("ws", "set", 1, "author", setOf(nil, fifteen, fm("b", number, con("x", number)))); kind(err) != "limit" {
		t.Fatalf("too large in one formula and malformed in another: %v", err)
	}
	stray := op(NodeAdd, chain(4), con("1", number))
	stray.Name = "x"
	if _, err := r.Draft("ws", "set", 8, "author", setOf(nil, fm("f", number, stray))); kind(err) != "limit" {
		t.Fatalf("a size below a node with a stray field: %v", err)
	}
	operands := make([]Node, 2)
	operands[0], operands[1] = Node{Kind: NodeAdd, Operands: operands}, Node{Kind: NodeAdd, Operands: operands}
	if _, err := r.Draft("ws", "set", 8, "author", setOf(nil, fm("f", number, Node{Kind: NodeAdd, Operands: operands}))); kind(err) != "limit" {
		t.Fatalf("a tree that contains itself: %v", err)
	}
	many := setOf(nil)
	for index := 0; index < 7; index++ {
		many.Variables = append(many.Variables, decl(string(rune('a'+index)), number))
	}
	mixed := setOf(nil, fm("a", number, chain(5)), fm("b", number, con("x", number)))
	semantic := setOf(nil, fm("f", usd, ref("f")))
	cases := []struct {
		name      string
		workspace string
		set       string
		number    int
		author    string
		content   Set
		want      string
	}{
		{"another content under the number", "ws", "set", 7, "author", plain("3"), "conflict"},
		{"another author under the number", "ws", "set", 7, "other", plain("2"), "conflict"},
		{"workspace", "", "set", 8, "author", plain("2"), "invalid"},
		{"set", "ws", "s et", 8, "author", plain("2"), "invalid"},
		{"number zero", "ws", "set", 0, "author", plain("2"), "invalid"},
		{"number high", "ws", "set", 1000000, "author", plain("2"), "invalid"},
		{"author", "ws", "set", 8, "", plain("2"), "invalid"},
		{"malformed content", "ws", "set", 8, "author", setOf(nil, fm("f", number, con("x", number))), "invalid"},
		{"malformed content before the number that exists", "ws", "set", 7, "author", setOf(nil, fm("f", number, con("x", number))), "invalid"},
		{"too many", "ws", "set", 8, "author", many, "limit"},
		{"too deep", "ws", "set", 8, "author", deep, "limit"},
		{"too deep in one formula and malformed in another", "ws", "set", 8, "author", mixed, "limit"},
		{"a size before the form of the request", "", "set", 8, "author", deep, "limit"},
		{"content that will not pass the check", "ws", "set", 8, "author", semantic, "ok"},
		{"the highest number", "ws", "set", 999999, "author", plain("2"), "ok"},
	}
	for _, c := range cases {
		if _, err := r.Draft(c.workspace, c.set, c.number, c.author, c.content); kind(err) != c.want {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	for number := 1; number <= 3; number++ {
		drafted(t, r, number, "2")
	}
	if _, err := r.Draft("ws", "set", 4, "author", plain("2")); kind(err) != "limit" {
		t.Fatalf("a seventh revision: %v", err)
	}
	if _, err := r.Draft("ws", "set", 3, "author", plain("2")); err != nil {
		t.Fatalf("a repeat in a full set: %v", err)
	}
	if _, err := r.Draft("ws", "set", 3, "author", plain("9")); kind(err) != "conflict" {
		t.Fatalf("a conflict in a full set: %v", err)
	}
	for _, set := range []string{"second", "third"} {
		if _, err := r.Draft("ws", set, 1, "author", plain("2")); err != nil {
			t.Fatalf("set %s: %v", set, err)
		}
	}
	if _, err := r.Draft("ws", "fourth", 1, "author", plain("2")); kind(err) != "limit" {
		t.Fatalf("a fourth set: %v", err)
	}
	if _, err := r.Draft("ws", "third", 2, "author", plain("2")); err != nil {
		t.Fatalf("a revision in a full workspace: %v", err)
	}
	if _, err := r.Draft("other", "fourth", 1, "author", plain("2")); err != nil {
		t.Fatalf("another workspace has its own room: %v", err)
	}
	if _, err := r.Active("ws", "fourth"); kind(err) != "not_found" {
		t.Fatalf("a refused draft left a set: %v", err)
	}
}

func TestContentIsACopy(t *testing.T) {
	r, _ := open(t)
	content := plain("2")
	view, _ := r.Draft("ws", "set", 1, "author", content)
	content.Formulas[0].Root.Operands[1] = con("9", number)
	content.Formulas[0].Root.Kind = NodeDiv
	content.Variables[0].Name = "other"
	stored, err := r.Content("ws", "set", 1)
	if err != nil || stored.Variables[0].Name != "cost" || stored.Formulas[0].Root.Kind != NodeMul || stored.Formulas[0].Root.Operands[1].Number != "2" {
		t.Fatalf("after the caller changed its set: %v %+v", err, stored)
	}
	stored.Formulas[0].Root.Operands[1].Number = "9"
	stored.Formulas[0].Root.Operands = stored.Formulas[0].Root.Operands[:1]
	stored.Formulas[0].ID = "other"
	stored.Variables[0].Type = krw
	again, _ := r.Content("ws", "set", 1)
	if again.Variables[0].Type != usd || again.Formulas[0].ID != "price" || len(again.Formulas[0].Root.Operands) != 2 || again.Formulas[0].Root.Operands[1].Number != "2" {
		t.Fatalf("after the caller changed what it read: %+v", again)
	}
	if _, err := r.RunCheck("ws", "set", 1); err != nil {
		t.Fatal(err)
	}
	result, err := r.EvaluateRevision("ws", "set", 1, "price", in("cost", "10"))
	if err != nil || shown(result) != "price=20(20)" || result.Digest != view.Digest {
		t.Fatalf("evaluation of the stored content: %v %s", err, shown(result))
	}
	sorted, _ := r.Draft("ws", "set", 2, "author", setOf(vars(decl("b", number), decl("a", number)), fm("z", number, vr("a")), fm("y", number, vr("b"))))
	listed, _ := r.Content("ws", "set", 2)
	if sorted.State != "draft" || listed.Variables[0].Name != "a" || listed.Formulas[0].ID != "y" {
		t.Fatalf("content in the order of the print: %+v", listed)
	}
}

func TestRunCheck(t *testing.T) {
	r, clock := open(t)
	drafted(t, r, 1, "2")
	if _, err := r.Draft("ws", "set", 2, "author", setOf(nil, fm("g", usd, ref("g")), fm("f", usd, ref("none")))); err != nil {
		t.Fatal(err)
	}
	clock.add(time.Minute)
	view, err := r.RunCheck("ws", "set", 1)
	if err != nil || view.State != "checked" || len(view.Findings) != 0 || !view.CheckedAt.Equal(start.Add(time.Minute)) {
		t.Fatalf("a check that passes: %v %+v", err, view)
	}
	failed, err := r.RunCheck("ws", "set", 2)
	if err != nil || failed.State != "check_failed" || codes(failed.Findings) != "unknown_name@f cycle@g" || !failed.CheckedAt.Equal(start.Add(time.Minute)) {
		t.Fatalf("a check that fails: %v %+v", err, failed)
	}
	before, _ := r.Active("ws", "set")
	clock.add(time.Minute)
	for number, want := range map[int]string{1: "checked", 2: "check_failed"} {
		again, err := r.RunCheck("ws", "set", number)
		if err != nil || again.State != want || !again.CheckedAt.Equal(start.Add(time.Minute)) {
			t.Fatalf("a repeat on %d: %v %+v", number, err, again)
		}
	}
	if again, _ := r.RunCheck("ws", "set", 2); codes(again.Findings) != "unknown_name@f cycle@g" {
		t.Fatalf("findings are kept: %+v", again.Findings)
	}
	if after, _ := r.Active("ws", "set"); after.Sequence != before.Sequence || before.Sequence != 4 {
		t.Fatalf("sequence: %d then %d", before.Sequence, after.Sequence)
	}
	if _, err := r.RecordImpact("ws", "set", 2, full(0)); kind(err) != "rejected" {
		t.Fatalf("an impact on a failed check: %v", err)
	}
	drafted(t, r, 3, "3")
	if _, err := r.Withdraw("ws", "set", 3, "mistake"); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		workspace, set string
		number         int
		want           string
	}{
		{"ws", "set", 3, "rejected"}, {"ws", "set", 4, "not_found"}, {"ws", "none", 1, "not_found"}, {"other", "set", 1, "not_found"},
		{"w s", "set", 1, "invalid"}, {"ws", "", 1, "invalid"}, {"ws", "set", 0, "invalid"}, {"ws", "set", 1000000, "invalid"}, {"ws", "", 3, "invalid"},
	}
	for _, c := range cases {
		if _, err := r.RunCheck(c.workspace, c.set, c.number); kind(err) != c.want {
			t.Errorf("%s %s %d: %v", c.workspace, c.set, c.number, err)
		}
	}
	if _, err := r.Withdraw("ws", "set", 2, "mistake"); err != nil {
		t.Fatalf("a failed check can be withdrawn: %v", err)
	}
}

func TestRecordImpact(t *testing.T) {
	r, clock := open(t)
	checked(t, r, 1, "2")
	bad := map[string]func(*Impact){
		"manifest":            func(i *Impact) { i.Manifest = "" },
		"manifest digest":     func(i *Impact) { i.ManifestDigest = "abc" },
		"digest upper":        func(i *Impact) { i.ManifestDigest = strings.ToUpper(digestA) },
		"digest other":        func(i *Impact) { i.ManifestDigest = strings.Repeat("g", 64) },
		"digest long":         func(i *Impact) { i.ManifestDigest = digestA + "a" },
		"epoch":               func(i *Impact) { i.Epoch = -1 },
		"coverage":            func(i *Impact) { i.Coverage = "all" },
		"coverage empty":      func(i *Impact) { i.Coverage = "" },
		"evaluated":           func(i *Impact) { i.Evaluated = -1 },
		"evaluated high":      func(i *Impact) { i.Evaluated = 1000000001 },
		"changed":             func(i *Impact) { i.Changed = -1 },
		"changed above":       func(i *Impact) { i.Changed = 11 },
		"without":             func(i *Impact) { i.WithoutValue = -1 },
		"failed":              func(i *Impact) { i.Failed = -1; i.WithoutValue = 1 },
		"without and failed":  func(i *Impact) { i.WithoutValue = 6; i.Failed = 5 },
		"epoch high":          func(i *Impact) { i.Epoch = 1000000001 },
		"failed above":        func(i *Impact) { i.Failed = 11 },
		"without value above": func(i *Impact) { i.WithoutValue = 11 },
	}
	for name, change := range bad {
		impact := full(0)
		change(&impact)
		if _, err := r.RecordImpact("ws", "set", 1, impact); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	edge := full(0)
	edge.Evaluated, edge.Changed, edge.WithoutValue, edge.Failed = 1000000000, 1000000000, 999999999, 1
	for name, request := range map[string][3]string{"workspace": {"", "set", "invalid"}, "set": {"ws", "s/et", "invalid"}, "unknown set": {"ws", "none", "not_found"}, "another workspace": {"other", "set", "not_found"}} {
		if _, err := r.RecordImpact(request[0], request[1], 1, full(0)); kind(err) != request[2] {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.RecordImpact("ws", "set", 0, full(0)); kind(err) != "invalid" {
		t.Fatalf("number: %v", err)
	}
	if _, err := r.RecordImpact("ws", "set", 2, full(0)); kind(err) != "not_found" {
		t.Fatalf("unknown revision: %v", err)
	}
	if _, err := r.RecordImpact("ws", "none", 1, Impact{}); kind(err) != "invalid" {
		t.Fatalf("form before the lookup: %v", err)
	}
	if _, err := r.RecordImpact("ws", "set", 1, full(1)); kind(err) != "rejected" {
		t.Fatalf("an epoch that is not the set's: %v", err)
	}
	sample := full(0)
	sample.Coverage = CoverageSample
	clock.add(time.Minute)
	view, err := r.RecordImpact("ws", "set", 1, sample)
	if err != nil || view.State != "impact_recorded" || view.Impact.Impact != sample || !view.Current || !view.Impact.At.Equal(start.Add(time.Minute)) || view.Impact.Approval != nil || len(view.History) != 0 {
		t.Fatalf("a sample: %v %+v", err, view)
	}
	if view.Impact.Subject != printed("formula_approval_subject_v1", "ws", "set", "1", view.Digest, "m-1", digestA, "0", "sample", "10", "4", "0", "0") {
		t.Fatalf("subject digest: %s", view.Impact.Subject)
	}
	sequence, _ := r.Active("ws", "set")
	clock.add(time.Minute)
	if again, err := r.RecordImpact("ws", "set", 1, sample); err != nil || !again.Impact.At.Equal(start.Add(time.Minute)) || len(again.History) != 0 {
		t.Fatalf("an identical repeat: %v", err)
	}
	if after, _ := r.Active("ws", "set"); after.Sequence != sequence.Sequence {
		t.Fatalf("a repeat wrote")
	}
	if _, err := r.RecordApproval("ws", "set", 1, "approver", "ref-1", view.Impact.Subject); kind(err) != "rejected" {
		t.Fatalf("an approval of a sample: %v", err)
	}
	replaced, err := r.RecordImpact("ws", "set", 1, edge)
	if err != nil || replaced.Impact.Impact != edge || len(replaced.History) != 1 || replaced.History[0].Impact != sample || replaced.History[0].Subject != view.Impact.Subject || replaced.Impact.Subject == view.Impact.Subject {
		t.Fatalf("a sample followed by a full impact: %v %+v", err, replaced)
	}
	if _, err := r.RecordApproval("ws", "set", 1, "approver", "ref-1", view.Impact.Subject); kind(err) != "rejected" {
		t.Fatalf("an approval of an impact that was replaced: %v", err)
	}
	third, err := r.RecordImpact("ws", "set", 1, full(0))
	if err != nil || len(third.History) != 2 {
		t.Fatalf("a third impact: %v", err)
	}
	if _, err := r.RecordImpact("ws", "set", 1, sample); kind(err) != "limit" {
		t.Fatalf("a full history: %v", err)
	}
	if now, _ := r.Revision("ws", "set", 1); now.Impact.Impact != full(0) || len(now.History) != 2 || now.State != "impact_recorded" {
		t.Fatalf("a refused impact wrote: %+v", now)
	}
	if _, err := r.RecordImpact("ws", "set", 1, full(0)); err != nil {
		t.Fatalf("a repeat with a full history: %v", err)
	}
	if _, err := r.RecordApproval("ws", "set", 1, "approver", "ref-1", third.Impact.Subject); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RecordImpact("ws", "set", 1, edge); kind(err) != "conflict" {
		t.Fatalf("another impact under an approval of this epoch: %v", err)
	}
	if _, err := r.RecordImpact("ws", "set", 1, full(1)); kind(err) != "conflict" {
		t.Fatalf("a conflict before a rejection: %v", err)
	}
	if again, err := r.RecordImpact("ws", "set", 1, full(0)); err != nil || again.State != "approval_recorded" {
		t.Fatalf("a repeat under an approval: %v", err)
	}
}

func TestStates(t *testing.T) {
	type reach func(t testing.TB, r *Registry)
	states := map[string]reach{
		"draft":   func(t testing.TB, r *Registry) { drafted(t, r, 1, "2") },
		"checked": func(t testing.TB, r *Registry) { checked(t, r, 1, "2") },
		"check_failed": func(t testing.TB, r *Registry) {
			r.Draft("ws", "set", 1, "author", setOf(nil, fm("f", usd, ref("f"))))
			r.RunCheck("ws", "set", 1)
		},
		"impact_recorded":   func(t testing.TB, r *Registry) { impacted(t, r, 1, "2") },
		"approval_recorded": func(t testing.TB, r *Registry) { approved(t, r, 1, "2") },
		"active":            func(t testing.TB, r *Registry) { activated(t, r, 1, "2") },
		"superseded": func(t testing.TB, r *Registry) {
			activated(t, r, 1, "2")
			activated(t, r, 2, "3")
		},
		"withdrawn": func(t testing.TB, r *Registry) {
			checked(t, r, 1, "2")
			r.Withdraw("ws", "set", 1, "mistake")
		},
	}
	want := map[string][6]string{
		"draft":             {"ok", "rejected", "rejected", "rejected", "ok", "rejected"},
		"check_failed":      {"ok", "rejected", "rejected", "rejected", "ok", "rejected"},
		"checked":           {"ok", "ok", "rejected", "rejected", "ok", "ok"},
		"impact_recorded":   {"ok", "ok", "ok", "rejected", "ok", "ok"},
		"approval_recorded": {"ok", "conflict", "conflict", "ok", "ok", "ok"},
		"active":            {"ok", "rejected", "conflict", "ok", "ok", "ok"},
		"superseded":        {"ok", "rejected", "conflict", "rejected", "ok", "ok"},
		"withdrawn":         {"rejected", "rejected", "rejected", "rejected", "conflict", "ok"},
	}
	if len(states) != 8 || len(want) != 8 {
		t.Fatal("states")
	}
	for name, make := range states {
		for operation := 0; operation < 6; operation++ {
			r, _ := open(t)
			make(t, r)
			if got := state(t, r, 1); got != name {
				t.Fatalf("%s reached as %s", name, got)
			}
			view, _ := r.Revision("ws", "set", 1)
			active, _ := r.Active("ws", "set")
			other := full(active.Epoch)
			other.Changed = 5
			var err error
			switch operation {
			case 0:
				_, err = r.RunCheck("ws", "set", 1)
			case 1:
				_, err = r.RecordImpact("ws", "set", 1, other)
			case 2:
				subject := digestA
				if view.Impact != nil {
					subject = view.Impact.Subject
				}
				_, err = r.RecordApproval("ws", "set", 1, "second", "ref-2", subject)
			case 3:
				_, err = r.Activate("ws", "set", 1)
			case 4:
				_, err = r.Withdraw("ws", "set", 1, "stopped")
			case 5:
				_, err = r.EvaluateRevision("ws", "set", 1, "price", in("cost", "1"))
			}
			if expected := want[name][operation]; kind(err) != expected {
				t.Errorf("%s, operation %d: %v", name, operation, err)
			}
		}
	}
}

func TestRecordApproval(t *testing.T) {
	r, clock := open(t)
	subject := impacted(t, r, 1, "2")
	cases := []struct {
		name                                  string
		workspace, set                        string
		number                                int
		approver, reference, subject, outcome string
	}{
		{"workspace", "w/s", "set", 1, "approver", "ref-1", subject, "invalid"},
		{"set", "ws", "", 1, "approver", "ref-1", subject, "invalid"},
		{"number", "ws", "set", 0, "approver", "ref-1", subject, "invalid"},
		{"approver", "ws", "set", 1, "", "ref-1", subject, "invalid"},
		{"reference", "ws", "set", 1, "approver", "ref 1", subject, "invalid"},
		{"subject", "ws", "set", 1, "approver", "ref-1", "abc", "invalid"},
		{"form before the lookup", "ws", "none", 1, "", "ref-1", subject, "invalid"},
		{"unknown set", "ws", "none", 1, "approver", "ref-1", subject, "not_found"},
		{"unknown revision", "ws", "set", 2, "approver", "ref-1", subject, "not_found"},
		{"another workspace", "other", "set", 1, "approver", "ref-1", subject, "not_found"},
		{"another subject", "ws", "set", 1, "approver", "ref-1", digestA, "rejected"},
		{"the author", "ws", "set", 1, "author", "ref-1", subject, "rejected"},
	}
	for _, c := range cases {
		if _, err := r.RecordApproval(c.workspace, c.set, c.number, c.approver, c.reference, c.subject); kind(err) != c.outcome {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	if got := state(t, r, 1); got != "impact_recorded" {
		t.Fatalf("a refusal wrote: %s", got)
	}
	clock.add(time.Minute)
	view, err := r.RecordApproval("ws", "set", 1, "approver", "ref-1", subject)
	recorded := Approval{Approver: "approver", Reference: "ref-1", Subject: subject, At: start.Add(time.Minute)}
	if err != nil || view.State != "approval_recorded" || *view.Impact.Approval != recorded || !view.Current {
		t.Fatalf("approval: %v %+v", err, view)
	}
	before, _ := r.Active("ws", "set")
	clock.add(time.Minute)
	if again, err := r.RecordApproval("ws", "set", 1, "approver", "ref-1", subject); err != nil || *again.Impact.Approval != recorded {
		t.Fatalf("an identical repeat: %v", err)
	}
	if after, _ := r.Active("ws", "set"); after.Sequence != before.Sequence {
		t.Fatal("a repeat wrote")
	}
	for name, other := range map[string][3]string{"approver": {"second", "ref-1", subject}, "reference": {"approver", "ref-2", subject}, "subject": {"approver", "ref-1", digestA}, "the author": {"author", "ref-1", subject}} {
		if _, err := r.RecordApproval("ws", "set", 1, other[0], other[1], other[2]); kind(err) != "conflict" {
			t.Errorf("another %s: %v", name, err)
		}
	}
	view.Impact.Approval.Approver = "changed"
	if now, _ := r.Revision("ws", "set", 1); now.Impact.Approval.Approver != "approver" {
		t.Fatal("a view shares its approval with the registry")
	}
	unclean := map[string]func(*Impact){
		"a sample":                 func(i *Impact) { i.Coverage = CoverageSample },
		"a subject without value":  func(i *Impact) { i.WithoutValue = 1 },
		"a subject that failed":    func(i *Impact) { i.Failed = 1 },
		"nothing evaluated at all": func(i *Impact) { i.Evaluated, i.Changed = 0, 0 },
	}
	number := 2
	for name, change := range unclean {
		checked(t, r, number, "3")
		impact := full(0)
		change(&impact)
		recorded, err := r.RecordImpact("ws", "set", number, impact)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		_, err = r.RecordApproval("ws", "set", number, "approver", "ref-1", recorded.Impact.Subject)
		want := "rejected"
		if name == "nothing evaluated at all" {
			want = "ok"
		}
		if kind(err) != want {
			t.Errorf("%s: %v", name, err)
		}
		number++
	}
}

func TestActivationAndTheEpoch(t *testing.T) {
	r, clock := open(t)
	approved(t, r, 1, "2")
	approved(t, r, 2, "3")
	subject3 := impacted(t, r, 3, "4")
	checked(t, r, 4, "5")
	if active, err := r.Active("ws", "set"); err != nil || active.Revision != 0 || active.Digest != "" || active.Epoch != 0 || active.Sequence != 13 {
		t.Fatalf("nothing active: %v %+v", err, active)
	}
	clock.add(time.Minute)
	first, err := r.Activate("ws", "set", 1)
	if err != nil || first.State != "active" || !first.ActivatedAt.Equal(start.Add(time.Minute)) || first.Current {
		t.Fatalf("activate: %v %+v", err, first)
	}
	active, _ := r.Active("ws", "set")
	if active.Revision != 1 || active.Digest != first.Digest || active.Epoch != 1 || active.Sequence != 14 {
		t.Fatalf("active: %+v", active)
	}
	if again, err := r.Activate("ws", "set", 1); err != nil || again.State != "active" {
		t.Fatalf("a repeat: %v", err)
	}
	if after, _ := r.Active("ws", "set"); after != active {
		t.Fatalf("a repeat wrote: %+v", after)
	}
	if _, err := r.Activate("ws", "set", 2); kind(err) != "rejected" {
		t.Fatalf("an approval of an earlier epoch: %v", err)
	}
	if view, _ := r.Revision("ws", "set", 2); view.State != "approval_recorded" || view.Current {
		t.Fatalf("a past approval is shown as not current: %+v", view)
	}
	if _, err := r.RecordApproval("ws", "set", 3, "approver", "ref-1", subject3); kind(err) != "rejected" {
		t.Fatalf("an approval on an impact of an earlier epoch: %v", err)
	}
	past := full(0)
	past.Changed = 7
	for number, state := range map[int]string{3: "impact_recorded", 4: "checked"} {
		if _, err := r.RecordImpact("ws", "set", number, past); kind(err) != "rejected" {
			t.Fatalf("an impact of a past epoch on a %s revision: %v", state, err)
		}
	}
	if view, err := r.RecordImpact("ws", "set", 4, full(1)); err != nil || !view.Current {
		t.Fatalf("an impact of this epoch on a checked revision: %v", err)
	}
	if again, err := r.RecordImpact("ws", "set", 2, full(0)); err != nil || again.State != "approval_recorded" || again.Current {
		t.Fatalf("an identical repeat of a past impact: %v", err)
	}
	if again, err := r.RecordApproval("ws", "set", 2, "approver", "ref-1", again2(t, r)); err != nil || again.State != "approval_recorded" {
		t.Fatalf("an identical repeat of a past approval: %v", err)
	}
	if _, err := r.RecordImpact("ws", "set", 2, full(0)); err != nil {
		t.Fatal(err)
	}
	other := full(0)
	other.Changed = 9
	if _, err := r.RecordImpact("ws", "set", 2, other); kind(err) != "rejected" {
		t.Fatalf("another impact of a past epoch: %v", err)
	}
	clock.add(time.Minute)
	renewed, err := r.RecordImpact("ws", "set", 2, full(1))
	if err != nil || renewed.State != "impact_recorded" || !renewed.Current || renewed.Impact.Approval != nil || len(renewed.History) != 1 ||
		renewed.History[0].Approval == nil || renewed.History[0].Approval.Approver != "approver" || renewed.History[0].Impact != full(0) {
		t.Fatalf("an impact of the new epoch discards the approval into the history: %v %+v", err, renewed)
	}
	if _, err := r.Activate("ws", "set", 2); kind(err) != "rejected" {
		t.Fatalf("activation without a new approval: %v", err)
	}
	if _, err := r.RecordApproval("ws", "set", 2, "approver", "ref-1", renewed.History[0].Subject); kind(err) != "rejected" {
		t.Fatalf("the subject of the earlier impact: %v", err)
	}
	if _, err := r.RecordApproval("ws", "set", 2, "approver", "ref-1", renewed.Impact.Subject); err != nil {
		t.Fatal(err)
	}
	clock.add(time.Minute)
	second, err := r.Activate("ws", "set", 2)
	if err != nil || second.State != "active" {
		t.Fatalf("activate the second: %v", err)
	}
	old, _ := r.Revision("ws", "set", 1)
	if old.State != "superseded" || !old.SupersededAt.Equal(start.Add(3*time.Minute)) || !old.ActivatedAt.Equal(start.Add(time.Minute)) {
		t.Fatalf("the earlier one: %+v", old)
	}
	if active, _ := r.Active("ws", "set"); active.Revision != 2 || active.Epoch != 2 || active.Digest != second.Digest {
		t.Fatalf("active: %+v", active)
	}
	if _, err := r.Activate("ws", "set", 1); kind(err) != "rejected" {
		t.Fatalf("a superseded revision: %v", err)
	}
	for name, request := range map[string][2]string{"workspace": {"", "invalid"}, "another workspace": {"other", "not_found"}} {
		if _, err := r.Activate(request[0], "set", 1); kind(err) != request[1] {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.Activate("ws", "set", 0); kind(err) != "invalid" {
		t.Fatal("number")
	}
	if _, err := r.Activate("ws", "s et", 1); kind(err) != "invalid" {
		t.Fatal("set")
	}
	if _, err := r.Activate("ws", "set", 9); kind(err) != "not_found" {
		t.Fatal("unknown revision")
	}
	if _, err := r.Activate("ws", "none", 1); kind(err) != "not_found" {
		t.Fatal("unknown set")
	}
}

func again2(t testing.TB, r *Registry) string {
	t.Helper()
	view, err := r.Revision("ws", "set", 2)
	if err != nil || view.Impact == nil {
		t.Fatalf("revision 2: %v", err)
	}
	return view.Impact.Subject
}

func TestAnApprovalDoesNotComeBackWhenNothingIsActiveAgain(t *testing.T) {
	r, _ := open(t)
	approved(t, r, 1, "2")
	approved(t, r, 2, "3")
	if _, err := r.Activate("ws", "set", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Withdraw("ws", "set", 1, "stopped"); err != nil {
		t.Fatal(err)
	}
	active, _ := r.Active("ws", "set")
	if active.Revision != 0 || active.Digest != "" || active.Epoch != 2 {
		t.Fatalf("after the stop: %+v", active)
	}
	if _, err := r.Activate("ws", "set", 2); kind(err) != "rejected" {
		t.Fatalf("an approval from before two changes: %v", err)
	}
	if _, err := r.RecordImpact("ws", "set", 2, full(0)); err != nil {
		t.Fatalf("its repeat: %v", err)
	}
	renewed, err := r.RecordImpact("ws", "set", 2, full(2))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.RecordApproval("ws", "set", 2, "approver", "ref-1", renewed.Impact.Subject); err != nil {
		t.Fatal(err)
	}
	if view, err := r.Activate("ws", "set", 2); err != nil || view.State != "active" {
		t.Fatalf("after a new impact and approval: %v", err)
	}
	if active, _ := r.Active("ws", "set"); active.Revision != 2 || active.Epoch != 3 {
		t.Fatalf("active: %+v", active)
	}
}

func TestWithdraw(t *testing.T) {
	r, clock := open(t)
	activated(t, r, 1, "2")
	approved(t, r, 2, "3")
	before, _ := r.Active("ws", "set")
	clock.add(time.Minute)
	view, err := r.Withdraw("ws", "set", 2, "refused")
	if err != nil || view.State != "withdrawn" || view.Withdrawal != "refused" || !view.WithdrawnAt.Equal(start.Add(time.Minute)) {
		t.Fatalf("withdraw: %v %+v", err, view)
	}
	after, _ := r.Active("ws", "set")
	if after.Epoch != before.Epoch || after.Revision != 1 || after.Sequence != before.Sequence+1 {
		t.Fatalf("a revision that is not active leaves the epoch: %+v", after)
	}
	clock.add(time.Minute)
	if again, err := r.Withdraw("ws", "set", 2, "refused"); err != nil || !again.WithdrawnAt.Equal(start.Add(time.Minute)) {
		t.Fatalf("an identical repeat: %v", err)
	}
	if now, _ := r.Active("ws", "set"); now.Sequence != after.Sequence {
		t.Fatal("a repeat wrote")
	}
	later := full(before.Epoch)
	later.Changed = 6
	if _, err := r.RecordImpact("ws", "set", 2, later); kind(err) != "conflict" {
		t.Fatalf("another impact on a withdrawn revision with an approval of this epoch: %v", err)
	}
	if _, err := r.Withdraw("ws", "set", 2, "mistake"); kind(err) != "conflict" {
		t.Fatalf("another reason: %v", err)
	}
	for _, reason := range []string{"mistake", "replaced", "refused", "stopped"} {
		number := map[string]int{"mistake": 3, "replaced": 4, "refused": 5, "stopped": 6}[reason]
		drafted(t, r, number, "5")
		if _, err := r.Withdraw("ws", "set", number, reason); err != nil {
			t.Errorf("%s: %v", reason, err)
		}
	}
	cases := []struct {
		workspace, set string
		number         int
		reason, want   string
	}{
		{"ws", "set", 1, "other", "invalid"}, {"ws", "set", 1, "", "invalid"}, {"", "set", 1, "mistake", "invalid"}, {"ws", "se t", 1, "mistake", "invalid"},
		{"ws", "set", 0, "mistake", "invalid"}, {"ws", "none", 1, "none", "invalid"}, {"ws", "none", 1, "mistake", "not_found"},
		{"ws", "set", 7, "mistake", "not_found"}, {"other", "set", 1, "mistake", "not_found"},
	}
	for _, c := range cases {
		if _, err := r.Withdraw(c.workspace, c.set, c.number, c.reason); kind(err) != c.want {
			t.Errorf("%s %s %d %s: %v", c.workspace, c.set, c.number, c.reason, err)
		}
	}
	if len(withdrawals) != 4 {
		t.Fatal("reasons")
	}
	stopped, err := r.Withdraw("ws", "set", 1, "stopped")
	if err != nil || stopped.State != "withdrawn" {
		t.Fatalf("the active revision: %v", err)
	}
	if now, _ := r.Active("ws", "set"); now.Revision != 0 || now.Digest != "" || now.Epoch != before.Epoch+1 {
		t.Fatalf("after the stop: %+v", now)
	}
	result, err := r.EvaluateRevision("ws", "set", 1, "price", in("cost", "10"))
	if err != nil || shown(result) != "price=20(20)" {
		t.Fatalf("a withdrawn revision still evaluates: %v", err)
	}
}

func TestASupersededRevisionWithdrawnLeavesTheEpoch(t *testing.T) {
	r, _ := open(t)
	activated(t, r, 1, "2")
	activated(t, r, 2, "3")
	before, _ := r.Active("ws", "set")
	if _, err := r.Withdraw("ws", "set", 1, "replaced"); err != nil {
		t.Fatal(err)
	}
	if after, _ := r.Active("ws", "set"); after.Epoch != before.Epoch || after.Revision != 2 {
		t.Fatalf("after: %+v", after)
	}
}

func TestEvaluateRevisionReproduces(t *testing.T) {
	r, _ := open(t)
	activated(t, r, 1, "2")
	first, err := r.EvaluateRevision("ws", "set", 1, "price", in("cost", "10.555"))
	view, _ := r.Revision("ws", "set", 1)
	if err != nil || shown(first) != "price=21.11(2111/100)" || first.Set != "set" || first.Revision != 1 || first.Digest != view.Digest || first.Semantics != "formula_v1" {
		t.Fatalf("result: %v %+v", err, first)
	}
	activated(t, r, 2, "3")
	activated(t, r, 3, "4")
	r.Withdraw("ws", "set", 1, "replaced")
	second, err := r.EvaluateRevision("ws", "set", 1, "price", in("cost", "10.555"))
	if err != nil || shown(second) != shown(first) || second.Digest != first.Digest || second.Revision != 1 {
		t.Fatalf("after it was superseded and withdrawn: %v %+v", err, second)
	}
	if other, _ := r.EvaluateRevision("ws", "set", 3, "price", in("cost", "10.555")); shown(other) != "price=42.22(2111/50)" || other.Digest == first.Digest || other.Revision != 3 {
		t.Fatalf("another revision: %+v", other)
	}
	seven := in("cost", "1", "cost", "1", "cost", "1", "cost", "1", "cost", "1", "cost", "1", "cost", "1")
	cases := []struct {
		name           string
		workspace, set string
		number         int
		target         string
		inputs         []Input
		want           string
	}{
		{"too many inputs before the form", "", "set", 1, "price", seven, "limit"},
		{"workspace", "", "set", 1, "price", nil, "invalid"},
		{"set", "ws", "", 1, "price", nil, "invalid"},
		{"number", "ws", "set", 0, "price", nil, "invalid"},
		{"number high", "ws", "set", 1000000, "price", nil, "invalid"},
		{"target", "ws", "set", 1, "pri ce", nil, "invalid"},
		{"target before the lookup", "ws", "none", 1, "", nil, "invalid"},
		{"unknown set", "ws", "none", 1, "price", nil, "not_found"},
		{"unknown revision", "ws", "set", 9, "price", nil, "not_found"},
		{"another workspace", "other", "set", 1, "price", nil, "not_found"},
		{"unknown target", "ws", "set", 1, "cost", nil, "not_found"},
		{"an input the set does not declare", "ws", "set", 1, "price", in("rate", "1"), "invalid"},
		{"an input the set does not declare, after the lookup", "ws", "none", 1, "price", in("rate", "1"), "not_found"},
		{"an input twice before the lookup", "ws", "none", 1, "price", in("cost", "1", "cost", "1"), "invalid"},
		{"an input that is no number before the lookup", "ws", "none", 1, "price", in("cost", "x"), "invalid"},
		{"an input too long before the lookup", "ws", "none", 1, "price", in("cost", strings.Repeat("1", 65)), "invalid"},
		{"an input name outside its form before the lookup", "ws", "none", 1, "price", in("co st", "1"), "invalid"},
	}
	for _, c := range cases {
		result, err := r.EvaluateRevision(c.workspace, c.set, c.number, c.target, c.inputs)
		if kind(err) != c.want || result.Digest != "" || result.Set != "" {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestReadsAndPages(t *testing.T) {
	r, _ := open(t)
	for _, number := range []int{5, 2, 9, 7} {
		drafted(t, r, number, "2")
	}
	r.RunCheck("ws", "set", 7)
	list := func(after, limit int) string {
		listed, err := r.Revisions("ws", "set", after, limit)
		if err != nil {
			return kind(err)
		}
		parts := []string{}
		for _, row := range listed {
			parts = append(parts, strings.Join([]string{string(rune('0' + row.Number)), row.State}, ":"))
		}
		return strings.Join(parts, " ")
	}
	cases := []struct {
		after, limit int
		want         string
	}{
		{0, 3, "2:draft 5:draft 7:checked"}, {7, 3, "9:draft"}, {2, 2, "5:draft 7:checked"}, {9, 3, ""}, {999999, 1, ""}, {0, 1, "2:draft"},
		{0, 4, "invalid"}, {0, 0, "invalid"}, {-1, 1, "invalid"}, {1000000, 1, "invalid"},
	}
	for _, c := range cases {
		if got := list(c.after, c.limit); got != c.want {
			t.Errorf("after %d limit %d: %q", c.after, c.limit, got)
		}
	}
	if empty, err := r.Revisions("ws", "set", 9, 1); err != nil || empty == nil {
		t.Fatalf("an empty page: %v", err)
	}
	type call func(workspace, set string, number int) error
	calls := map[string]call{
		"revision": func(w, s string, n int) error { _, err := r.Revision(w, s, n); return err },
		"content":  func(w, s string, n int) error { _, err := r.Content(w, s, n); return err },
	}
	for name, read := range calls {
		for _, c := range []struct {
			workspace, set string
			number         int
			want           string
		}{{"ws", "set", 5, "ok"}, {"", "set", 5, "invalid"}, {"ws", "", 5, "invalid"}, {"ws", "set", 0, "invalid"}, {"ws", "set", 1000000, "invalid"},
			{"ws", "set", 6, "not_found"}, {"ws", "none", 5, "not_found"}, {"other", "set", 5, "not_found"}} {
			if err := read(c.workspace, c.set, c.number); kind(err) != c.want {
				t.Errorf("%s %s %s %d: %v", name, c.workspace, c.set, c.number, err)
			}
		}
	}
	for _, c := range [][3]string{{"", "set", "invalid"}, {"ws", "s et", "invalid"}, {"ws", "none", "not_found"}, {"other", "set", "not_found"}} {
		if _, err := r.Active(c[0], c[1]); kind(err) != c[2] {
			t.Errorf("active %s %s: %v", c[0], c[1], err)
		}
		if _, err := r.Revisions(c[0], c[1], 0, 1); kind(err) != c[2] {
			t.Errorf("revisions %s %s: %v", c[0], c[1], err)
		}
	}
}

func TestTheTenantBoundary(t *testing.T) {
	r, _ := open(t)
	subject := impacted(t, r, 1, "2")
	if _, err := r.Draft("other", "set", 1, "author", plain("2")); err != nil {
		t.Fatal(err)
	}
	r.RunCheck("other", "set", 1)
	view, err := r.RecordImpact("other", "set", 1, full(0))
	mine, _ := r.Revision("ws", "set", 1)
	if err != nil || view.Digest != mine.Digest || view.Impact.Subject == subject {
		t.Fatalf("equal content in another workspace: %v", err)
	}
	if _, err := r.RecordApproval("other", "set", 1, "approver", "ref-1", subject); kind(err) != "rejected" {
		t.Fatalf("a subject of another workspace: %v", err)
	}
	if _, err := r.Draft("ws", "second", 1, "author", plain("2")); err != nil {
		t.Fatal(err)
	}
	r.RunCheck("ws", "second", 1)
	sibling, _ := r.RecordImpact("ws", "second", 1, full(0))
	checked(t, r, 2, "2")
	twin, _ := r.RecordImpact("ws", "set", 2, full(0))
	subjects := map[string]bool{subject: true, view.Impact.Subject: true, sibling.Impact.Subject: true, twin.Impact.Subject: true}
	if len(subjects) != 4 || twin.Digest != mine.Digest {
		t.Fatalf("subjects of equal content: %d", len(subjects))
	}
	fields := map[string]func(*Impact){
		"manifest": func(i *Impact) { i.Manifest = "m-2" }, "manifest digest": func(i *Impact) { i.ManifestDigest = strings.Repeat("b", 64) },
		"coverage": func(i *Impact) { i.Coverage = CoverageSample }, "evaluated": func(i *Impact) { i.Evaluated = 11 },
		"changed": func(i *Impact) { i.Changed = 5 }, "without value": func(i *Impact) { i.WithoutValue = 1 }, "failed": func(i *Impact) { i.Failed = 1 },
		"two counts exchanged": func(i *Impact) { i.WithoutValue, i.Failed = 0, 0; i.Evaluated, i.Changed = 4, 4 },
	}
	for name, change := range fields {
		impact := full(0)
		change(&impact)
		if got := subjectDigest("ws", "set", 1, mine.Digest, impact); subjects[got] {
			t.Errorf("%s leaves the subject", name)
		} else {
			subjects[got] = true
		}
	}
	if got := subjectDigest("ws", "set", 1, mine.Digest, full(1)); subjects[got] {
		t.Fatal("the epoch leaves the subject")
	}
	if got := subjectDigest("ws", "set", 1, digestA, full(0)); subjects[got] {
		t.Fatal("the content leaves the subject")
	}
	if subjectDigest("ws", "set", 1, mine.Digest, full(0)) != subject {
		t.Fatal("the subject is not a function of its fields")
	}
}

func TestTimeIsTheHighestReadingInUTC(t *testing.T) {
	r, clock := open(t)
	seoul := time.FixedZone("KST", 9*3600)
	clock.set(start.Add(time.Hour).In(seoul))
	view, _ := r.Draft("ws", "set", 1, "author", plain("2"))
	if view.DraftedAt.Location() != time.UTC || !view.DraftedAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("drafted at: %v", view.DraftedAt)
	}
	clock.set(start)
	checked, _ := r.RunCheck("ws", "set", 1)
	if !checked.CheckedAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("a clock that went back: %v", checked.CheckedAt)
	}
	clock.set(start.Add(2 * time.Hour))
	if _, err := r.Revision("ws", "set", 1); err != nil {
		t.Fatal(err)
	}
	clock.set(start)
	impact, _ := r.RecordImpact("ws", "set", 1, full(0))
	if !impact.Impact.At.Equal(start.Add(2 * time.Hour)) {
		t.Fatalf("a reading taken by a read: %v", impact.Impact.At)
	}
}

func TestErrorsAreFixedValues(t *testing.T) {
	r, _ := open(t)
	secret := "secret-name"
	_, invalid := r.Draft("ws", "set", 1, "author", setOf(vars(decl(secret, Type{Kind: secret})), fm("f", usd, vr(secret))))
	_, missing := r.Revision(secret, secret, 1)
	_, _, compile := Compile(testProfile(t), Limits{}, plain("2"))
	_, evaluate := must(t, plain("2")).Evaluate(secret, in(secret, secret))
	for _, err := range []error{invalid, missing, compile, evaluate} {
		if err == nil || strings.Contains(err.Error(), secret) {
			t.Fatalf("an error repeats the request: %v", err)
		}
		if err != ErrInvalid && err != ErrNotFound {
			t.Fatalf("an error that is not a fixed value: %v", err)
		}
	}
	_, found, _ := Compile(testProfile(t), testLimits(), setOf(nil, fm("f", usd, vr(secret))))
	if codes(found) != "unknown_name@f" {
		t.Fatalf("a finding: %s", codes(found))
	}
}
