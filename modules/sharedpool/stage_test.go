package sharedpool

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRunsAndChunks(t *testing.T) {
	h := newHarness(t, nil)
	h.pool(capped(1, 10, "1000", ""))
	start := func(spec RunSpec) error { return second(h.r.StartRun("ws1", "shared-1", 1, spec)) }
	for name, spec := range map[string]RunSpec{"identifier": {Purpose: Showback, Chunks: 1}, "purpose": {ID: "r", Purpose: "billing", Chunks: 1}, "no purpose": {ID: "r", Chunks: 1}, "no chunk": {ID: "r", Purpose: Showback}, "negative chunks": {ID: "r", Purpose: Showback, Chunks: -1}} {
		wantErr(t, start(spec), ErrInvalid, name)
		wantErr(t, second(h.r.StartRun("ws1", "shared-9", 1, spec)), ErrInvalid, name+" before the pool is looked up")
	}
	wantErr(t, start(RunSpec{ID: "r", Purpose: Showback, Chunks: 5}), ErrLimit, "more chunks than the limit")
	wantErr(t, second(h.r.StartRun("ws1", "shared-9", 1, RunSpec{ID: "r", Purpose: Showback, Chunks: 5})), ErrNotFound, "the pool is looked up before the number of chunks is judged")
	h.run(1, "r", Showback, 4)
	h.clock.advance(time.Hour)
	h.view(1, "r")
	h.clock.advance(-3 * time.Hour)
	h.run(1, "late", Simulation, 1)
	if view := h.view(1, "late"); !view.Started.Equal(start0.Add(time.Hour)) || !view.Changed.Equal(start0.Add(time.Hour)) {
		t.Fatalf("the registry keeps the latest reading of its clock: %+v", view)
	}
	h.clock.advance(3 * time.Hour)
	before := h.r.dump()
	h.run(1, "r", Showback, 4)
	wantErr(t, start(RunSpec{ID: "r", Purpose: Simulation, Chunks: 4}), ErrConflict, "another purpose")
	wantErr(t, start(RunSpec{ID: "r", Purpose: Showback, Chunks: 3}), ErrConflict, "another number of chunks")
	h.unchanged(before, "repeats and conflicts")
	stage := func(id string, position int, weights ...Weight) error {
		return second(h.r.StageChunk("ws1", "shared-1", 1, id, position, weights))
	}
	good := []Weight{w("b", Organization, "2"), w("a", Organization, "1.0"), w("c", MSP, "")}
	for name, c := range map[string]struct {
		position int
		weights  []Weight
	}{
		"an empty chunk": {1, nil}, "position zero": {0, good}, "a malformed target": {1, []Weight{w("a b", MSP, "1")}}, "no target": {1, []Weight{w("", MSP, "1")}}, "a target that starts with a colon": {1, []Weight{w(":x", MSP, "1")}},
		"the residual's identifier": {1, []Weight{w(residualID, MSP, "1")}}, "a target twice": {1, []Weight{w("a", MSP, "1"), w("b", MSP, "1"), w("a", MSP, "2")}}, "an unknown bearer": {1, []Weight{w("a", "customer", "1")}},
		"no bearer": {1, []Weight{w("a", "", "1")}}, "a negative weight": {1, []Weight{w("a", MSP, "-1")}}, "a weight of negative zero": {1, []Weight{w("a", MSP, "-0")}}, "a weight that is no number": {1, []Weight{w("a", MSP, "half")}},
		"a malformed weight after a good one": {1, []Weight{w("a", MSP, "1"), w("b", MSP, "1e3")}},
	} {
		wantErr(t, stage("r", c.position, c.weights...), ErrInvalid, name)
		wantErr(t, stage("r9", c.position, c.weights...), ErrInvalid, name+" before the run is looked up")
	}
	five := []Weight{w("a", MSP, "1"), w("b", MSP, "1"), w("c", MSP, "1"), w("d", MSP, "1"), w("e", MSP, "x")}
	wantErr(t, stage("r", 1, five...), ErrLimit, "more weights than a chunk may have, counted before they are read")
	wantErr(t, stage("r9", 1, five...), ErrLimit, "and before the run is looked up")
	wantErr(t, stage("r9", 1, good...), ErrNotFound, "a run that does not exist")
	wantErr(t, second(h.r.Chunk("ws1", "shared-1", 1, "r", 2, "", 4)), ErrNotFound, "a chunk that is not staged")
	h.stage(1, "r", 2, good...)
	h.clock.advance(time.Hour)
	before = h.r.dump()
	h.stage(1, "r", 2, good[2], good[0], good[1])
	for name, apply := range map[string]func([]Weight){
		"a weight": func(ws []Weight) { ws[0].Weight = "3" }, "the spelling of a weight": func(ws []Weight) { ws[1].Weight = "1" }, "a weight that became known": func(ws []Weight) { ws[2].Weight = "0" },
		"a bearer": func(ws []Weight) { ws[0].Bearer = MSP }, "a target": func(ws []Weight) { ws[0].Target = "bb" }, "weights exchanged": func(ws []Weight) { ws[0].Weight, ws[1].Weight = ws[1].Weight, ws[0].Weight },
	} {
		changed := append([]Weight(nil), good...)
		apply(changed)
		wantErr(t, stage("r", 2, changed...), ErrConflict, name)
	}
	wantErr(t, stage("r", 2, good[:2]...), ErrConflict, "a weight less")
	wantErr(t, stage("r", 5, w("z", MSP, "1")), ErrRejected, "a position beyond the declared number")
	wantErr(t, stage("r", 1, w("y", MSP, "1"), w("a", Organization, "1.0")), ErrRejected, "a target that another chunk holds")
	h.unchanged(before, "repeats, conflicts and rejections")
	page, err := h.r.Chunk("ws1", "shared-1", 1, "r", 2, "", 2)
	if err != nil || len(page.Items) != 2 || !page.More || page.Items[0] != good[1] || page.Items[1] != good[0] {
		t.Fatalf("%+v %v", page, err)
	}
	page, err = h.r.Chunk("ws1", "shared-1", 1, "r", 2, "b", 4)
	if err != nil || len(page.Items) != 1 || page.More || page.Items[0] != good[2] {
		t.Fatalf("%+v %v", page, err)
	}
	for _, c := range []struct {
		position int
		after    string
		limit    int
	}{{0, "", 1}, {2, "a b", 1}, {2, "", 0}, {2, "", 5}} {
		wantErr(t, second(h.r.Chunk("ws1", "shared-1", 1, "r", c.position, c.after, c.limit)), ErrInvalid, "a page of a chunk")
	}
	h.stage(1, "r", 1, w("d", MSP, "1"), w("e", MSP, "1"), w("f", MSP, "1"), w("g", MSP, "1"))
	before = h.r.dump()
	wantErr(t, stage("r", 3, w("h", MSP, "1"), w("i", MSP, "1")), ErrLimit, "more targets than a run may have")
	wantErr(t, stage("r", 3, w("h", MSP, "1"), w("a", MSP, "1")), ErrRejected, "rejected before no room")
	h.unchanged(before, "the limit of targets")
	h.stage(1, "r", 3, w("h", MSP, "1"))
	if view := h.view(1, "r"); view.Staged != 3 || view.Declared != 4 || view.Targets != 8 || view.NotKnown != 1 || view.State != Staging || !view.Started.Equal(start0) || !view.Changed.Equal(start0.Add(2*time.Hour)) || view.Amount != "1000" {
		t.Fatalf("%+v", view)
	}
	if err := h.r.Withdraw("ws1", "shared-1", 1, "r", ByOperator); err != nil {
		t.Fatal(err)
	}
	wantErr(t, stage("r", 4, w("z", MSP, "1")), ErrRejected, "a chunk for a run that is not staging")
	if state, err := h.r.StageChunk("ws1", "shared-1", 1, "r", 2, good); err != nil || state != Withdrawn {
		t.Fatal(state, err)
	}
	if state, err := h.r.StartRun("ws1", "shared-1", 1, RunSpec{ID: "r", Purpose: Showback, Chunks: 4}); err != nil || state != Withdrawn {
		t.Fatal(state, err)
	}
	wantErr(t, second(h.r.Seal("ws1", "shared-1", 1, "r")), ErrRejected, "a seal of a run withdrawn while staging")
	for _, id := range []string{"m", "c", "x", "a", "k", "b"} {
		h.run(1, id, Simulation, 1)
	}
	wantErr(t, start(RunSpec{ID: "one-more", Purpose: Showback, Chunks: 1}), ErrLimit, "a withdrawn run counts towards the limit")
	wantErr(t, start(RunSpec{ID: "m", Purpose: Showback, Chunks: 1}), ErrConflict, "a conflict before no room")
	rows, err := h.r.Runs("ws1", "shared-1", 1, "", 4)
	if err != nil || len(rows) != 4 || rows[0] != (RunRow{ID: "a", Purpose: Simulation, State: Staging}) || rows[3].ID != "k" {
		t.Fatalf("%+v %v", rows, err)
	}
	rows, err = h.r.Runs("ws1", "shared-1", 1, "m", 4)
	if err != nil || len(rows) != 2 || rows[0] != (RunRow{ID: "r", Purpose: Showback, State: Withdrawn}) || rows[1].ID != "x" {
		t.Fatalf("%+v %v", rows, err)
	}
	for _, c := range []struct {
		after string
		limit int
	}{{"a b", 1}, {"", 0}, {"", 5}} {
		wantErr(t, second(h.r.Runs("ws1", "shared-1", 1, c.after, c.limit)), ErrInvalid, "a page of runs")
	}
}

var start0 = start

func TestSealHoldsResidualsAndBounds(t *testing.T) {
	one := func(spec Spec, weights ...Weight) (*harness, State, error) {
		h := newHarness(t, nil)
		h.pool(spec)
		h.run(spec.Version, "r", Simulation, 1)
		h.stage(spec.Version, "r", 1, weights...)
		state, err := h.r.Seal("ws1", "shared-1", spec.Version, "r")
		return h, state, err
	}
	h, state, err := one(capped(1, 10, "100", "10"), w("a", Organization, ""), w("b", Organization, "99"), w("c", MSP, ""))
	if view := h.view(1, "r"); err != nil || state != Held || view.Hold != WeightNotKnown || view.NotKnown != 2 || view.WeightSum != "" || view.Denominator != "" {
		t.Fatalf("a weight that is not known is judged before the capacity: %+v %v", view, err)
	}
	h, state, err = one(capped(1, 10, "100", "10"), w("a", Organization, "6"), w("b", Organization, "4.0001"))
	if view := h.view(1, "r"); err != nil || state != Held || view.Hold != OverCapacity || view.WeightSum != "10.0001" || view.Denominator != "" {
		t.Fatalf("%+v %v", view, err)
	}
	before := h.r.dump()
	if state, err := h.r.Seal("ws1", "shared-1", 1, "r"); err != nil || state != Held {
		t.Fatal(state, err)
	}
	wantErr(t, second(h.r.Settle("ws1", "shared-1", 1, "r", Alternative{Approver: "a", Reference: "r", Bearer: MSP})), ErrRejected, "an alternative for a run over its capacity")
	wantErr(t, second(h.r.Publish("ws1", "shared-1", 1, "r", Approval{})), ErrRejected, "a held run published")
	h.unchanged(before, "a held run")
	if err := h.r.Withdraw("ws1", "shared-1", 1, "r", EnteredInError); err != nil {
		t.Fatal(err)
	}
	if state, err := h.r.Seal("ws1", "shared-1", 1, "r"); err != nil || state != Withdrawn {
		t.Fatalf("a run that a seal has judged answers with its state: %v %v", state, err)
	}
	h, state, err = one(capped(1, 10, "100", "10"), w("a", Organization, "0"), w("b", MSP, "0"))
	h.publish(1, "r")
	if got := h.shares(1, "r"); err != nil || state != Sealed || got != ":residual=100(msp/10/residual) a=0(organization/0) b=0(msp/0)" {
		t.Fatalf("a stated capacity with weights of zero is not held: %s %v", got, err)
	}
	h, _, _ = one(capped(1, 10, "100", "100"), w("a", Organization, "60.0"), w("b", Organization, "40"))
	h.publish(1, "r")
	if view := h.view(1, "r"); h.shares(1, "r") != "a=60(organization/60.0) b=40(organization/40)" || view.Residual != "" || view.ResidualReason != "" || sums(view) != "published amount=100 denominator=100 organization=100" {
		t.Fatalf("weights that use the capacity up leave no residual: %s", sums(view))
	}
	h, _, _ = one(capped(1, 10, "0", "100"), w("a", Organization, "60"))
	h.publish(1, "r")
	if view := h.view(1, "r"); h.shares(1, "r") != ":residual=0(msp/40/residual) a=0(organization/60)" || view.Residual != "0" || view.ResidualReason != BelowCapacity {
		t.Fatalf("an amount of zero has a residual of zero, which is not no residual: %+v", view)
	}
	for _, c := range []struct{ first, second, want string }{
		{"1a", "b", "1a=1(organization/1) :residual=0(msp/1/residual) b=0(organization/1)"},
		{"a", "b", ":residual=1(msp/1/residual) a=0(organization/1) b=0(organization/1)"},
		{"-x", "b", "-x=1(organization/1) :residual=0(msp/1/residual) b=0(organization/1)"},
		{".x", "b", ".x=1(organization/1) :residual=0(msp/1/residual) b=0(organization/1)"},
		{"_x", "b", ":residual=1(msp/1/residual) _x=0(organization/1) b=0(organization/1)"},
		{"Z", "b", ":residual=1(msp/1/residual) Z=0(organization/1) b=0(organization/1)"},
	} {
		h, _, _ = one(capped(1, 10, "1", "3"), w(c.second, Organization, "1"), w(c.first, Organization, "1"))
		h.publish(1, "r")
		if got := h.shares(1, "r"); got != c.want {
			t.Fatalf("a unit left over between equal weights:\n got %s\nwant %s", got, c.want)
		}
	}
	for _, c := range []struct{ quantum, amount, want string }{
		{"0.1", "10", ":residual=2.5(msp/1/residual) a=5(organization/2) b=2.5(organization/1)"},
		{"0.01", "0.03", ":residual=0.01(msp/1/residual) a=0.01(organization/2) b=0.01(organization/1)"},
		{"0.001", "0.001", ":residual=0(msp/1/residual) a=0.001(organization/2) b=0(organization/1)"},
		{"0.0001", "1.0001", ":residual=0.25(msp/1/residual) a=0.5001(organization/2) b=0.25(organization/1)"},
		{"1", "7", ":residual=2(msp/1/residual) a=3(organization/2) b=2(organization/1)"},
	} {
		spec := capped(1, 10, c.amount, "4")
		spec.Quantum = c.quantum
		h, _, _ = one(spec, w("a", Organization, "2"), w("b", Organization, "1"))
		h.publish(1, "r")
		if got := h.shares(1, "r"); got != c.want {
			t.Fatalf("%s at %s:\n got %s\nwant %s", c.amount, c.quantum, got, c.want)
		}
	}
	full := newHarness(t, nil)
	full.pool(capped(1, 10, "900", "9"))
	full.run(1, "r", Simulation, 4)
	full.stage(1, "r", 3, w("a", Organization, "1"), w("h", Organization, "1"))
	full.stage(1, "r", 1, w("e", Organization, "1"), w("b", Organization, "1"))
	full.stage(1, "r", 4, w("c", Organization, "1"), w("f", Organization, "1"))
	full.stage(1, "r", 2, w("g", Organization, "1"), w("d", Organization, "1"))
	if state := full.seal(1, "r"); state != Sealed {
		t.Fatalf("a run with as many targets as the limit and a residual: %s", state)
	}
	full.publish(1, "r")
	if got := full.shares(1, "r"); !strings.HasPrefix(got, ":residual=100(msp/1/residual) a=100(organization/1)") || strings.Count(got, "=100(") != 9 {
		t.Fatal(got)
	}
	zoned, err := NewRegistry(testLimits(), profileOf(t, 12, 4, 40), func() time.Time { return start.In(time.FixedZone("kst", 9*3600)) })
	if err != nil {
		t.Fatal(err)
	}
	z := &harness{t: t, r: zoned}
	z.pool(capped(1, 10, "100", ""))
	z.run(1, "r", Simulation, 1)
	if view := z.view(1, "r"); view.Started.Location() != time.UTC || view.Changed.Location() != time.UTC || !view.Started.Equal(start) {
		t.Fatalf("the times of a run are kept in UTC: %+v", view)
	}
	clock := &testClock{at: start}
	small, err := NewRegistry(testLimits(), profileOf(t, 3, 4, 40), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	s := &harness{t: t, r: small, clock: clock}
	s.pool(capped(1, 10, "100", ""))
	s.run(1, "r", Simulation, 1)
	s.stage(1, "r", 1, w("a", Organization, "999"), w("b", Organization, "1"))
	before = s.r.dump()
	for attempt := 0; attempt < 2; attempt++ {
		wantErr(t, second(s.r.Seal("ws1", "shared-1", 1, "r")), ErrLimit, "weights whose sum the profile cannot hold")
	}
	s.unchanged(before, "a seal that ends on a bound")
	if view := s.view(1, "r"); view.State != Staging {
		t.Fatal(view.State)
	}
	if err := s.r.Withdraw("ws1", "shared-1", 1, "r", ByOperator); err != nil {
		t.Fatalf("the exit of a run that can never be sealed: %v", err)
	}
	tight, err := NewRegistry(testLimits(), profileOf(t, 3, 4, 7), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	tt := &harness{t: t, r: tight, clock: clock}
	tt.pool(alter(capped(1, 10, "100.0001", ""), func(s *Spec) { s.Quantum = "0.0001" }))
	tt.run(1, "r", Simulation, 1)
	tt.stage(1, "r", 1, w("a", Organization, "1"), w("b", Organization, "1"))
	before = tt.r.dump()
	wantErr(t, second(tt.r.Seal("ws1", "shared-1", 1, "r")), ErrLimit, "the primitive's bound of intermediates")
	tt.unchanged(before, "a bound of the primitive")
	narrow := newHarness(t, func(l *Limits) { l.Intermediate = 6 })
	narrow.pool(capped(1, 10, "1200000", ""))
	narrow.run(1, "r", Simulation, 1)
	narrow.stage(1, "r", 1, w("a", Organization, "60"), w("b", Organization, "40"))
	before = narrow.r.dump()
	wantErr(t, second(narrow.r.Seal("ws1", "shared-1", 1, "r")), ErrLimit, "the allocation's bound of intermediates")
	narrow.unchanged(before, "a bound of the allocation")
}

func TestPublishingWithdrawingAndChargeback(t *testing.T) {
	h := newHarness(t, nil)
	from, to := month(10)
	if billed, err := h.r.Chargeback("ws1", "shared-1", from, to); err != ErrNotFound || billed != (PublishedChargeback{}) {
		t.Fatal(billed, err)
	}
	h.pool(capped(1, 10, "100", "4"))
	ready := func(version int, id string, purpose Purpose) {
		t.Helper()
		h.run(version, id, purpose, 1)
		h.stage(version, id, 1, w("a", Organization, "2"), w("b", Organization, "1"))
		if state := h.seal(version, id); state != Sealed {
			t.Fatal(state)
		}
	}
	ready(1, "c1", Chargeback)
	ready(1, "s1", Showback)
	for _, bad := range []Approval{{Approver: "a b", Reference: "r"}, {Approver: "a", Reference: "see mail"}} {
		wantErr(t, second(h.r.Publish("ws1", "shared-1", 1, "c1", bad)), ErrInvalid, "an approval")
	}
	before := h.r.dump()
	for name, c := range map[string]struct {
		id       string
		approval Approval
	}{"a chargeback without an approver": {"c1", Approval{Reference: "r"}}, "a chargeback without a reference": {"c1", Approval{Approver: "a"}}, "a chargeback without an approval": {"c1", Approval{}},
		"a showback with an approver": {"s1", Approval{Approver: "a"}}, "a showback with a reference": {"s1", Approval{Reference: "r"}}, "a showback with an approval": {"s1", approved}} {
		wantErr(t, second(h.r.Publish("ws1", "shared-1", 1, c.id, c.approval)), ErrRejected, name)
	}
	h.unchanged(before, "a refused publication")
	if billed, err := h.r.Chargeback("ws1", "shared-1", from, to); err != ErrNotFound || billed != (PublishedChargeback{Sequence: 7}) {
		t.Fatal(billed, err)
	}
	h.clock.advance(time.Hour)
	h.publish(1, "c1")
	h.publish(1, "s1")
	before = h.r.dump()
	h.publish(1, "c1")
	h.publish(1, "s1")
	wantErr(t, second(h.r.Publish("ws1", "shared-1", 1, "c1", Approval{Approver: "approver-2", Reference: approved.Reference})), ErrConflict, "another approver")
	wantErr(t, second(h.r.Publish("ws1", "shared-1", 1, "c1", Approval{Approver: approved.Approver, Reference: "doc-2"})), ErrConflict, "another reference")
	wantErr(t, second(h.r.Publish("ws1", "shared-1", 1, "s1", approved)), ErrConflict, "an approval for a published showback")
	if state, err := h.r.Seal("ws1", "shared-1", 1, "c1"); err != nil || state != Published {
		t.Fatal(state, err)
	}
	wantErr(t, second(h.r.Settle("ws1", "shared-1", 1, "c1", Alternative{Approver: "a", Reference: "r", Bearer: MSP})), ErrRejected, "an alternative for a published run")
	h.unchanged(before, "repeats after publication")
	billed, err := h.r.Chargeback("ws1", "shared-1", from.In(time.FixedZone("kst", 9*3600)), to)
	if err != nil || billed != (PublishedChargeback{Version: 1, Run: "c1", Sequence: 9}) {
		t.Fatalf("%+v %v", billed, err)
	}
	if view := h.view(1, "c1"); view.Approval != approved || !view.WasPublished || !view.Changed.Equal(start.Add(time.Hour)) || view.ByBearer[Organization] != "75" || view.Residual != "25" {
		t.Fatalf("%+v", view)
	}
	page, err := h.r.Shares("ws1", "shared-1", 1, "c1", "", 2)
	if err != nil || len(page.Items) != 2 || !page.More || page.State != Published || page.Purpose != Chargeback || page.Sequence != 9 || page.Items[0] != (Share{Target: residualID, Bearer: MSP, Weight: "1", Amount: "25", Residual: true}) {
		t.Fatalf("%+v %v", page, err)
	}
	page.Items[0].Amount = "9"
	page, err = h.r.Shares("ws1", "shared-1", 1, "c1", residualID, 4)
	if err != nil || len(page.Items) != 2 || page.More || page.Items[0] != (Share{Target: "a", Bearer: Organization, Weight: "2", Amount: "50"}) || h.shares(1, "c1") != ":residual=25(msp/1/residual) a=50(organization/2) b=25(organization/1)" {
		t.Fatalf("%+v %v", page, err)
	}
	if page, err := h.r.Shares("ws1", "shared-1", 1, "s1", "b", 4); err != nil || len(page.Items) != 0 || page.More || page.Purpose != Showback {
		t.Fatalf("%+v %v", page, err)
	}
	for _, c := range []struct {
		after string
		limit int
	}{{"a b", 1}, {"", 0}, {"", 5}} {
		wantErr(t, second(h.r.Shares("ws1", "shared-1", 1, "c1", c.after, c.limit)), ErrInvalid, "a page of shares")
	}
	ready(1, "c2", Chargeback)
	h.pool(capped(2, 10, "200", ""))
	ready(2, "c3", Chargeback)
	ready(2, "sim", Simulation)
	h.pool(capped(3, 11, "300", ""))
	ready(3, "nov", Chargeback)
	h.pool(alter(capped(4, 10, "400", ""), func(s *Spec) { s.PeriodStart, s.PeriodEnd, s.Currency = to.Add(-day), to.Add(day), "USD" }))
	ready(4, "across", Chargeback)
	before = h.r.dump()
	for name, c := range map[string]struct {
		version int
		id      string
	}{"a second chargeback of the version": {1, "c2"}, "a chargeback of another version of the period": {2, "c3"}, "a chargeback of a period that overlaps, in another currency": {4, "across"}} {
		wantErr(t, second(h.r.Publish("ws1", "shared-1", c.version, c.id, approved)), ErrRejected, name)
	}
	h.unchanged(before, "a refused chargeback")
	h.publish(2, "sim")
	h.publish(3, "nov")
	wantErr(t, second(h.r.Publish("ws1", "shared-1", 4, "across", approved)), ErrRejected, "a chargeback across two published ones")
	for name, period := range map[string][2]time.Time{"a day": {from, from.Add(day)}, "its last day": {to.Add(-day), to}, "the month and a second": {from, to.Add(time.Second)}} {
		got, err := h.r.Chargeback("ws1", "shared-1", period[0], period[1])
		if err != ErrNotFound || got.Version != 0 || got.Run != "" || got.Sequence == 0 {
			t.Fatalf("%s: %+v %v", name, got, err)
		}
	}
	for name, period := range map[string][2]time.Time{"backwards": {to, from}, "empty": {from, from}, "a fraction": {from, to.Add(time.Millisecond)}, "too long": {from, from.Add(40*day + time.Second)}, "none": {}} {
		_, err := h.r.Chargeback("ws1", "shared-1", period[0], period[1])
		wantErr(t, err, ErrInvalid, name)
	}
	for _, c := range []struct {
		ws, pool string
		want     error
	}{{"ws2", "shared-1", ErrNotFound}, {"ws1", "shared-9", ErrNotFound}, {"", "shared-1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.Chargeback(c.ws, c.pool, from, to)
		wantErr(t, err, c.want, "chargeback")
	}
	for _, reason := range []WithdrawReason{"", "superseded", "replaced"} {
		wantErr(t, h.r.Withdraw("ws1", "shared-1", 1, "c1", reason), ErrInvalid, "a reason")
	}
	wantErr(t, h.r.Withdraw("ws1", "shared-1", 1, "c9", ByOperator), ErrNotFound, "a run that does not exist")
	_, sequence := h.billed(10)
	h.clock.advance(time.Hour)
	if err := h.r.Withdraw("ws1", "shared-1", 1, "c1", EnteredInError); err != nil {
		t.Fatal(err)
	}
	if err := h.r.Withdraw("ws1", "shared-1", 1, "c1", EnteredInError); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.Withdraw("ws1", "shared-1", 1, "c1", ByOperator), ErrConflict, "a second reason")
	if run, after := h.billed(10); run != "" || after != sequence+1 {
		t.Fatal(run, after)
	}
	page, err = h.r.Shares("ws1", "shared-1", 1, "c1", "", 4)
	if err != nil || len(page.Items) != 3 || page.State != Withdrawn || page.Sequence != sequence+1 {
		t.Fatalf("a published run stays readable after its withdrawal and says so: %+v %v", page, err)
	}
	if view := h.view(1, "c1"); view.State != Withdrawn || view.WithdrawReason != EnteredInError || !view.WasPublished || view.ByBearer == nil || !view.Changed.Equal(start.Add(2*time.Hour)) {
		t.Fatalf("%+v", view)
	}
	if state, err := h.r.Publish("ws1", "shared-1", 1, "c1", approved); err != nil || state != Withdrawn {
		t.Fatal(state, err)
	}
	if err := h.r.Withdraw("ws1", "shared-1", 1, "c2", ByOperator); err != nil {
		t.Fatal(err)
	}
	wantErr(t, second(h.r.Shares("ws1", "shared-1", 1, "c2", "", 4)), ErrRejected, "a run withdrawn before publication stays unreadable")
	wantErr(t, second(h.r.Publish("ws1", "shared-1", 1, "c2", approved)), ErrRejected, "a withdrawn run published")
	h.publish(2, "c3")
	if run, _ := h.billed(10); run != "c3" {
		t.Fatal("after the withdrawal the period is free for another chargeback")
	}
	if billed, _ := h.r.Chargeback("ws1", "shared-1", from, to); billed.Version != 2 {
		t.Fatal(billed)
	}
	if run, _ := h.billed(11); run != "nov" {
		t.Fatal(run)
	}
	far := capped(1, 10, "100", "")
	far.WorkspaceID = "ws2"
	alone := h.r.dump()
	if err := h.r.OpenPool(far); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.StartRun("ws2", "shared-1", 1, RunSpec{ID: "c3", Purpose: Chargeback, Chunks: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.StageChunk("ws2", "shared-1", 1, "c3", 1, []Weight{w("a", Organization, "1")}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.Seal("ws2", "shared-1", 1, "c3"); err != nil {
		t.Fatal(err)
	}
	if state, err := h.r.Publish("ws2", "shared-1", 1, "c3", approved); err != nil || state != Published {
		t.Fatalf("a chargeback of the same pool name and period in another workspace: %v %v", state, err)
	}
	for _, line := range strings.Split(alone, "\n") {
		if !strings.Contains(h.r.dump(), line) {
			t.Fatalf("another workspace changed: %s", line)
		}
	}
	var unbuilt Registry
	var missing *Registry
	for _, r := range []*Registry{&unbuilt, missing} {
		wantErr(t, r.OpenPool(capped(1, 10, "1", "")), ErrInvalid, "pool")
		wantErr(t, second(r.StartRun("ws1", "shared-1", 1, RunSpec{ID: "r", Purpose: Showback, Chunks: 1})), ErrInvalid, "run")
		wantErr(t, second(r.StageChunk("ws1", "shared-1", 1, "r", 1, []Weight{w("a", MSP, "1")})), ErrInvalid, "stage")
		wantErr(t, second(r.Seal("ws1", "shared-1", 1, "r")), ErrInvalid, "seal")
		wantErr(t, second(r.Settle("ws1", "shared-1", 1, "r", Alternative{Approver: "a", Reference: "r", Bearer: MSP})), ErrInvalid, "settle")
		wantErr(t, second(r.Publish("ws1", "shared-1", 1, "r", approved)), ErrInvalid, "publish")
		wantErr(t, r.Withdraw("ws1", "shared-1", 1, "r", ByOperator), ErrInvalid, "withdraw")
		wantErr(t, second(r.Pool("ws1", "shared-1", 1)), ErrInvalid, "read pool")
		wantErr(t, second(r.Versions("ws1", "shared-1", 0, 1)), ErrInvalid, "versions")
		wantErr(t, second(r.Run("ws1", "shared-1", 1, "r")), ErrInvalid, "read run")
		wantErr(t, second(r.Runs("ws1", "shared-1", 1, "", 1)), ErrInvalid, "runs")
		wantErr(t, second(r.Shares("ws1", "shared-1", 1, "r", "", 1)), ErrInvalid, "shares")
		wantErr(t, second(r.Chunk("ws1", "shared-1", 1, "r", 1, "", 1)), ErrInvalid, "chunk")
		wantErr(t, second(r.Chargeback("ws1", "shared-1", from, to)), ErrInvalid, "chargeback")
	}
}

func (h *harness) billed(monthNumber int) (string, uint64) {
	h.t.Helper()
	from, to := month(monthNumber)
	out, err := h.r.Chargeback("ws1", "shared-1", from, to)
	if err != nil && err != ErrNotFound {
		h.t.Fatal(err)
	}
	return out.Run, out.Sequence
}

func TestTheSequenceNumberAndTheSourceScan(t *testing.T) {
	h := newHarness(t, nil)
	number := func() uint64 {
		_, sequence := h.billed(10)
		return sequence
	}
	h.pool(capped(1, 10, "100", ""))
	steps := []struct {
		name string
		call func() error
		want uint64
	}{
		{"a repeated pool", func() error { return h.r.OpenPool(capped(1, 10, "100", "")) }, 1},
		{"a run", func() error {
			return second(h.r.StartRun("ws1", "shared-1", 1, RunSpec{ID: "r", Purpose: Chargeback, Chunks: 1}))
		}, 2},
		{"the run repeated", func() error {
			return second(h.r.StartRun("ws1", "shared-1", 1, RunSpec{ID: "r", Purpose: Chargeback, Chunks: 1}))
		}, 2},
		{"a chunk", func() error { return second(h.r.StageChunk("ws1", "shared-1", 1, "r", 1, []Weight{w("a", MSP, "0")})) }, 3},
		{"the chunk repeated", func() error { return second(h.r.StageChunk("ws1", "shared-1", 1, "r", 1, []Weight{w("a", MSP, "0")})) }, 3},
		{"a seal that holds", func() error { return second(h.r.Seal("ws1", "shared-1", 1, "r")) }, 4},
		{"the seal repeated", func() error { return second(h.r.Seal("ws1", "shared-1", 1, "r")) }, 4},
		{"a settlement", func() error {
			return second(h.r.Settle("ws1", "shared-1", 1, "r", Alternative{Approver: "a", Reference: "d", Bearer: MSP}))
		}, 5},
		{"the settlement repeated", func() error {
			return second(h.r.Settle("ws1", "shared-1", 1, "r", Alternative{Approver: "a", Reference: "d", Bearer: MSP}))
		}, 5},
		{"a publication", func() error { return second(h.r.Publish("ws1", "shared-1", 1, "r", approved)) }, 6},
		{"the publication repeated", func() error { return second(h.r.Publish("ws1", "shared-1", 1, "r", approved)) }, 6},
		{"a withdrawal", func() error { return h.r.Withdraw("ws1", "shared-1", 1, "r", ByOperator) }, 7},
		{"the withdrawal repeated", func() error { return h.r.Withdraw("ws1", "shared-1", 1, "r", ByOperator) }, 7},
		{"another version", func() error { return h.r.OpenPool(capped(2, 11, "100", "")) }, 8},
	}
	for _, step := range steps {
		if err := step.call(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		if got := number(); got != step.want {
			t.Fatalf("after %s: %d, want %d", step.name, got, step.want)
		}
	}
	for name, call := range map[string]func() error{
		"a refused pool": func() error { return h.r.OpenPool(capped(1, 10, "101", "")) },
		"a refused run": func() error {
			return second(h.r.StartRun("ws1", "shared-1", 1, RunSpec{ID: "r", Purpose: Showback, Chunks: 1}))
		},
		"a refused chunk": func() error { return second(h.r.StageChunk("ws1", "shared-1", 1, "r", 1, []Weight{w("a", MSP, "1")})) },
		"a refused settlement": func() error {
			return second(h.r.Settle("ws1", "shared-1", 1, "r", Alternative{Approver: "b", Reference: "d", Bearer: MSP}))
		},
		"a refused withdrawal": func() error { return h.r.Withdraw("ws1", "shared-1", 1, "r", EnteredInError) },
	} {
		if call() == nil || number() != 8 {
			t.Fatalf("%s moved the number or succeeded", name)
		}
	}
	other := capped(1, 10, "100", "")
	other.Pool = "shared-2"
	h.pool(other)
	if number() != 8 {
		t.Fatal("another pool has its own number")
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	banned := regexp.MustCompile(`float32|float64|math\.|math/|ParseFloat|big\.|\.Quo\(|\.Div\(|\.Mul\(|time\.Now|rand\.`)
	primitive := regexp.MustCompile(`decimal\.[A-Za-z]+`)
	allowed := map[string]bool{"decimal.Parse": true, "decimal.Add": true, "decimal.Sub": true, "decimal.Cmp": true, "decimal.Decimal": true, "decimal.Profile": true, "decimal.HalfEven": true,
		"decimal.ErrResultBound": true, "decimal.ErrIntermediateBound": true}
	methods := regexp.MustCompile(`\.(Coefficient|Quanta|RoundToQuantum|DivideRound)\(`)
	scanned := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		if strings.Contains(string(content), "primitives/decimal") != strings.Contains(string(content), "\n\t\"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal\"\n") {
			t.Errorf("%s imports the primitive under another name", name)
		}
		for index, line := range strings.Split(string(content), "\n") {
			if found := banned.FindString(line); found != "" {
				t.Errorf("%s:%d uses %s", name, index+1, found)
			}
			if found := methods.FindString(line); found != "" {
				t.Errorf("%s:%d calls %s", name, index+1, found)
			}
			for _, use := range primitive.FindAllString(line, -1) {
				if !allowed[use] {
					t.Errorf("%s:%d uses %s", name, index+1, use)
				}
			}
			if strings.Contains(line, "//") || strings.Contains(line, "/*") {
				t.Errorf("%s:%d has a comment", name, index+1)
			}
		}
	}
	if scanned != 4 {
		t.Fatalf("scanned %d source files", scanned)
	}
}
