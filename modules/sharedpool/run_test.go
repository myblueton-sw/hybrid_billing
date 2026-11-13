package sharedpool

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/allocation"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestTheCasesOfTheAcceptance(t *testing.T) {
	weights := []Weight{w("org-a", Organization, "60"), w("org-b", Organization, "25"), w("org-c", Organization, "10"), w("ops", MSP, "3"), w("lost", Unattributed, "1")}
	want := "lost=10101(unattributed/1) ops=30303(msp/3) org-a=606061(organization/60) org-b=252525(organization/25) org-c=101010(organization/10)"
	cuts := [][][]int{{{0, 1, 2, 3, 4}}, {{4, 3, 2, 1, 0}}, {{0}, {1}, {2}, {3, 4}}, {{4, 2}, {0}, {3, 1}}, {{1, 3}, {4}, {2, 0}}}
	for index, cut := range cuts {
		h := newHarness(t, func(l *Limits) { l.ChunkWeights = 5 })
		h.pool(capped(1, 10, "1000000", ""))
		h.run(1, "r", Simulation, len(cut))
		for position := len(cut); position >= 1; position-- {
			var part []Weight
			for _, at := range cut[position-1] {
				part = append(part, weights[at])
			}
			h.stage(1, "r", position, part...)
		}
		if state := h.seal(1, "r"); state != Sealed {
			t.Fatal(state)
		}
		h.publish(1, "r")
		if got := h.shares(1, "r"); got != want {
			t.Fatalf("cut %d:\n got %s\nwant %s", index, got, want)
		}
		if got := sums(h.view(1, "r")); got != "published amount=1000000 denominator=99 msp=30303 organization=959596 unattributed=10101" {
			t.Fatal(got)
		}
	}
	h := newHarness(t, nil)
	h.pool(capped(1, 10, "1000000", "100"))
	h.run(1, "below", Chargeback, 2)
	h.stage(1, "below", 1, w("org-a", Organization, "60.0"))
	before := h.r.dump()
	wantErr(t, second(h.r.Seal("ws1", "shared-1", 1, "below")), ErrRejected, "a run with a chunk missing")
	h.unchanged(before, "a refused seal")
	h.stage(1, "below", 2, w("org-b", Organization, "25"))
	if state := h.seal(1, "below"); state != Sealed {
		t.Fatal(state)
	}
	view := h.view(1, "below")
	if view.ByBearer != nil || view.Residual != "" || view.ResidualReason != "" || view.Denominator != "100" || view.WeightSum != "85" || view.Targets != 2 || view.Staged != 2 || view.Declared != 2 || view.WasPublished {
		t.Fatalf("a sealed run shows no sum of a class and no residual: %+v", view)
	}
	wantErr(t, second(h.r.Shares("ws1", "shared-1", 1, "below", "", 4)), ErrRejected, "the shares of a run that is not published")
	h.publish(1, "below")
	if got := h.shares(1, "below"); got != ":residual=150000(msp/15/residual) org-a=600000(organization/60.0) org-b=250000(organization/25)" {
		t.Fatalf("what lies below the capacity is a share of its own: %s", got)
	}
	if got := sums(h.view(1, "below")); got != "published amount=1000000 denominator=100 residual=150000(below_capacity) msp=150000 organization=850000" {
		t.Fatal(got)
	}
	z := newHarness(t, nil)
	z.pool(capped(1, 10, "1000000", ""))
	z.run(1, "zero", Chargeback, 1)
	z.stage(1, "zero", 1, w("org-a", Organization, "0"), w("org-b", Organization, "0.0"))
	if state := z.seal(1, "zero"); state != Held {
		t.Fatal(state)
	}
	if view := z.view(1, "zero"); view.Hold != ZeroDenominator || view.WeightSum != "0" || view.Denominator != "" || view.State != Held {
		t.Fatalf("%+v", view)
	}
	wantErr(t, second(z.r.Publish("ws1", "shared-1", 1, "zero", approved)), ErrRejected, "a held run")
	wantErr(t, second(z.r.Shares("ws1", "shared-1", 1, "zero", "", 4)), ErrRejected, "a held run has no share")
	for _, bad := range []Alternative{{Reference: "r", Bearer: MSP}, {Approver: "a", Bearer: MSP}, {Approver: "a", Reference: "r"}, {Approver: "a", Reference: "r", Bearer: Organization}, {Approver: "a b", Reference: "r", Bearer: MSP}} {
		wantErr(t, second(z.r.Settle("ws1", "shared-1", 1, "zero", bad)), ErrInvalid, "an alternative")
	}
	alternative := Alternative{Approver: "approver-2", Reference: "decision-7", Bearer: Unattributed}
	if state, err := z.r.Settle("ws1", "shared-1", 1, "zero", alternative); err != nil || state != Sealed {
		t.Fatal(state, err)
	}
	before = z.r.dump()
	if state, err := z.r.Settle("ws1", "shared-1", 1, "zero", alternative); err != nil || state != Sealed {
		t.Fatal(state, err)
	}
	wantErr(t, second(z.r.Settle("ws1", "shared-1", 1, "zero", alter(alternative, func(a *Alternative) { a.Bearer = MSP }))), ErrConflict, "another alternative")
	if state, err := z.r.Seal("ws1", "shared-1", 1, "zero"); err != nil || state != Sealed {
		t.Fatal(state, err)
	}
	z.unchanged(before, "repeats of a settlement")
	z.publish(1, "zero")
	if got := z.shares(1, "zero"); got != ":residual=1000000(unattributed/0/residual) org-a=0(organization/0) org-b=0(organization/0.0)" {
		t.Fatalf("a settled run: %s", got)
	}
	view = z.view(1, "zero")
	if got := sums(view); got != "published amount=1000000 denominator= residual=1000000(recorded_alternative) organization=0 unattributed=1000000" || *view.Alternative != alternative || view.Hold != ZeroDenominator {
		t.Fatal(got)
	}
	view.Alternative.Bearer, view.ByBearer[Organization] = MSP, "9"
	if again := z.view(1, "zero"); again.Alternative.Bearer != Unattributed || again.ByBearer[Organization] != "0" {
		t.Fatal("a view shares state")
	}
}

func TestPoolVersions(t *testing.T) {
	h := newHarness(t, nil)
	from, to := month(10)
	good := capped(1, 10, "1000000", "100")
	for name, apply := range map[string]func(*Spec){
		"workspace": func(s *Spec) { s.WorkspaceID = "" }, "pool": func(s *Spec) { s.Pool = "a b" }, "version zero": func(s *Spec) { s.Version = 0 }, "a version above the maximum": func(s *Spec) { s.Version = 1000000 },
		"currency": func(s *Spec) { s.Currency = "krw" }, "a period backwards": func(s *Spec) { s.PeriodStart, s.PeriodEnd = to, from }, "an empty period": func(s *Spec) { s.PeriodEnd = from },
		"a period with a fraction": func(s *Spec) { s.PeriodEnd = to.Add(time.Millisecond) }, "a period that is too long": func(s *Spec) { s.PeriodEnd = from.Add(40*day + time.Second) },
		"no amount": func(s *Spec) { s.Amount = "" }, "a negative amount": func(s *Spec) { s.Amount = "-1" }, "a negative zero": func(s *Spec) { s.Amount = "-0" }, "an amount that is no number": func(s *Spec) { s.Amount = "much" },
		"an amount finer than the quantum": func(s *Spec) { s.Amount = "1000000.5" }, "a quantum of two": func(s *Spec) { s.Quantum = "2" }, "a quantum of ten": func(s *Spec) { s.Quantum = "10" }, "a start with a fraction": func(s *Spec) { s.PeriodStart = from.Add(time.Millisecond) }, "a quantum written otherwise": func(s *Spec) { s.Quantum = "1.0" }, "no quantum": func(s *Spec) { s.Quantum = "" },
		"a capacity of zero": func(s *Spec) { s.Capacity = "0" }, "a negative capacity": func(s *Spec) { s.Capacity = "-100" }, "a capacity that is no number": func(s *Spec) { s.Capacity = "all" },
		"a capacity without a bearer": func(s *Spec) { s.ResidualBearer = "" }, "a residual borne by an organisation": func(s *Spec) { s.ResidualBearer = Organization }, "an unknown bearer": func(s *Spec) { s.ResidualBearer = "nobody" },
		"a bearer without a capacity": func(s *Spec) { s.Capacity = "" },
	} {
		wantErr(t, h.r.OpenPool(alter(good, apply)), ErrInvalid, name)
	}
	if h.r.dump() != "" {
		t.Fatal("a refused pool wrote")
	}
	fine, err := NewRegistry(testLimits(), profileOf(t, 12, 6, 40), h.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	wantErr(t, fine.OpenPool(alter(good, func(s *Spec) { s.Quantum, s.Amount = "0.0001", "0.00001" })), ErrInvalid, "five places under a profile that reads six")
	wantErr(t, fine.OpenPool(alter(good, func(s *Spec) { s.Quantum, s.Amount = "0.0001", "7.00000" })), ErrInvalid, "five places that are zeros")
	if err := fine.OpenPool(alter(good, func(s *Spec) { s.Quantum, s.Amount = "0.0001", "7.0000" })); err != nil {
		t.Fatalf("four places: %v", err)
	}
	coarse, err := NewRegistry(testLimits(), profileOf(t, 12, 2, 40), h.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	wantErr(t, coarse.OpenPool(alter(good, func(s *Spec) { s.Quantum = "0.001" })), ErrInvalid, "a quantum finer than the profile reads")
	for index, c := range [][2]string{{"1", "7"}, {"0.1", "7.5"}, {"0.01", "7.25"}, {"0.001", "7.125"}, {"0.0001", "7.0625"}, {"1", "0"}, {"0.01", "7.5"}} {
		spec := alter(good, func(s *Spec) { s.Version, s.Quantum, s.Amount = index+1, c[0], c[1] })
		spec.Pool = fmt.Sprintf("q%d", index/4)
		h.pool(spec)
	}
	h = newHarness(t, nil)
	h.pool(alter(good, func(s *Spec) {
		s.PeriodStart = from.In(time.FixedZone("kst", 9*3600))
		s.Version = 999999
		s.PeriodEnd = from.Add(40 * day)
	}))
	h.pool(good)
	before := h.r.dump()
	h.pool(good)
	for name, apply := range map[string]func(*Spec){
		"currency": func(s *Spec) { s.Currency = "USD" }, "start": func(s *Spec) { s.PeriodStart = from.Add(time.Second) }, "end": func(s *Spec) { s.PeriodEnd = to.Add(-time.Second) }, "amount": func(s *Spec) { s.Amount = "1000001" },
		"the spelling of the amount": func(s *Spec) { s.Amount = "1000000.0"; s.Quantum = "0.1" }, "quantum": func(s *Spec) { s.Quantum = "0.1" }, "capacity": func(s *Spec) { s.Capacity = "101" },
		"the spelling of the capacity": func(s *Spec) { s.Capacity = "100.0" }, "the bearer": func(s *Spec) { s.ResidualBearer = Unattributed }, "normalising": func(s *Spec) { s.Capacity, s.ResidualBearer = "", "" },
	} {
		wantErr(t, h.r.OpenPool(alter(good, apply)), ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	spec, err := h.r.Pool("ws1", "shared-1", 999999)
	if err != nil || spec.PeriodStart.Location() != time.UTC || !spec.PeriodEnd.Equal(from.Add(40*day)) || spec.Amount != "1000000" || spec.Capacity != "100" || spec.ResidualBearer != MSP {
		t.Fatalf("%+v %v", spec, err)
	}
	for _, c := range []struct {
		ws, pool string
		number   int
		want     error
	}{{"ws2", "shared-1", 1, ErrNotFound}, {"ws1", "shared-2", 1, ErrNotFound}, {"ws1", "shared-1", 2, ErrNotFound}, {"", "shared-1", 1, ErrInvalid}, {"ws1", "", 1, ErrInvalid}, {"ws1", "shared-1", 0, ErrInvalid}} {
		wantErr(t, second(h.r.Pool(c.ws, c.pool, c.number)), c.want, "pool")
		wantErr(t, second(h.r.StartRun(c.ws, c.pool, c.number, RunSpec{ID: "r", Purpose: Showback, Chunks: 1})), c.want, "run")
		wantErr(t, second(h.r.Runs(c.ws, c.pool, c.number, "", 1)), c.want, "runs")
		wantErr(t, second(h.r.Run(c.ws, c.pool, c.number, "r")), c.want, "a run")
		wantErr(t, second(h.r.Seal(c.ws, c.pool, c.number, "r")), c.want, "seal")
		wantErr(t, second(h.r.Publish(c.ws, c.pool, c.number, "r", approved)), c.want, "publish")
		wantErr(t, second(h.r.Settle(c.ws, c.pool, c.number, "r", Alternative{Approver: "a", Reference: "r", Bearer: MSP})), c.want, "settle")
		wantErr(t, h.r.Withdraw(c.ws, c.pool, c.number, "r", ByOperator), c.want, "withdraw")
		wantErr(t, second(h.r.Shares(c.ws, c.pool, c.number, "r", "", 1)), c.want, "shares")
		wantErr(t, second(h.r.Chunk(c.ws, c.pool, c.number, "r", 1, "", 1)), c.want, "chunk")
		wantErr(t, second(h.r.StageChunk(c.ws, c.pool, c.number, "r", 1, []Weight{w("a", MSP, "1")})), c.want, "stage")
	}
	for index := 2; index <= 5; index++ {
		h.pool(alter(good, func(s *Spec) { s.Version = index }))
	}
	wantErr(t, h.r.OpenPool(alter(good, func(s *Spec) { s.Version = 7 })), ErrLimit, "a seventh version")
	rows, err := h.r.Versions("ws1", "shared-1", 0, 4)
	if err != nil || len(rows) != 4 || rows[0] != (VersionRow{Version: 1, Currency: "KRW", PeriodStart: from, PeriodEnd: to}) || rows[3].Version != 4 {
		t.Fatalf("%+v %v", rows, err)
	}
	rows, err = h.r.Versions("ws1", "shared-1", 4, 4)
	if err != nil || len(rows) != 2 || rows[0].Version != 5 || rows[1].Version != 999999 {
		t.Fatalf("%+v %v", rows, err)
	}
	for _, c := range []struct{ after, limit int }{{-1, 1}, {0, 0}, {0, 5}} {
		wantErr(t, second(h.r.Versions("ws1", "shared-1", c.after, c.limit)), ErrInvalid, "a page of versions")
	}
	wantErr(t, second(h.r.Versions("ws1", "shared-9", 0, 1)), ErrNotFound, "versions of a pool that does not exist")
	wantErr(t, second(h.r.Versions("ws2", "shared-1", 0, 1)), ErrNotFound, "versions in another workspace")
	for _, name := range []string{"p2", "p3"} {
		h.pool(alter(good, func(s *Spec) { s.Pool = name }))
	}
	before = h.r.dump()
	wantErr(t, h.r.OpenPool(alter(good, func(s *Spec) { s.Pool = "p4" })), ErrLimit, "a fourth pool")
	wantErr(t, h.r.OpenPool(alter(good, func(s *Spec) { s.Amount = "1" })), ErrConflict, "a conflict before no room")
	h.unchanged(before, "limits")
	if err := h.r.OpenPool(alter(good, func(s *Spec) { s.WorkspaceID, s.Pool = "ws2", "p4" })); err != nil {
		t.Fatalf("limits are per workspace: %v", err)
	}
}

func TestConstructionIdentifiersAndTheScan(t *testing.T) {
	clock := func() time.Time { return start }
	good := profileOf(t, 12, 4, 40)
	for name, apply := range map[string]func(*Limits){
		"pools": func(l *Limits) { l.Pools = 0 }, "versions": func(l *Limits) { l.Versions = 0 }, "runs": func(l *Limits) { l.Runs = 0 }, "chunks": func(l *Limits) { l.Chunks = 0 }, "weights": func(l *Limits) { l.ChunkWeights = 0 },
		"targets": func(l *Limits) { l.Targets = 0 }, "page": func(l *Limits) { l.Page = 0 }, "period": func(l *Limits) { l.MaxPeriod = 0 }, "intermediate": func(l *Limits) { l.Intermediate = 0 },
	} {
		_, err := NewRegistry(alter(testLimits(), apply), good, clock)
		wantErr(t, err, ErrInvalid, name)
	}
	_, err := NewRegistry(testLimits(), decimal.Profile{}, clock)
	wantErr(t, err, ErrInvalid, "no profile")
	_, err = NewRegistry(testLimits(), good, nil)
	wantErr(t, err, ErrInvalid, "no clock")
	one, _ := decimal.Parse(good, "1")
	for id, want := range map[string]bool{"a": true, "A.z_0:9-": true, residualID: true, strings.Repeat("a", 128): true, strings.Repeat("a", 129): false, "": false, "a b": false, "a/b": false, "kim@example": false} {
		if validID(id) != want {
			t.Errorf("identifier %q", id)
		}
		if id == "" {
			continue
		}
		_, err := allocation.LargestRemainderV1(allocation.Request{Profile: good, Limits: allocation.Limits{MaxTargets: 2, MaxIntermediateDigits: 40}, Encoding: allocation.EncodingASCIIBytewiseV1, Total: one, Quantum: one, Mode: decimal.HalfEven,
			Targets: []allocation.Target{{ID: id, Weight: one}}})
		if (err == nil) != want {
			t.Errorf("the allocation and this package differ on %q: %v", id, err)
		}
	}
	for code, want := range map[string]bool{"KRW": true, "krw": false, "KR": false, "KRWW": false, "1RW": false, "K1W": false, "KR1": false, "kRW": false, "KrW": false, "KRw": false} {
		if validCurrency(code) != want {
			t.Errorf("currency %q", code)
		}
	}
	for _, kind := range []error{ErrInvalid, ErrNotFound, ErrConflict, ErrRejected, ErrLimit, ErrSelfCheck} {
		if strings.ContainsAny(kind.Error(), "%0123456789") {
			t.Fatalf("%q", kind)
		}
	}
}
