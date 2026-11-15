package migration

import (
	"sync"
	"testing"
	"time"
)

func TestBatchLifecycle(t *testing.T) {
	f := standard(t)
	d1, d2 := doc("a", 1), doc("b", 2)
	id, v := f.batch("src-1", docs(d1, d2), "t1")
	if v.State != BatchVerified || v.Classes[ClassNew] != 2 || v.Effect == "" || len(v.Series) != 1 || v.Series[0].Number != 2 || v.Printed != 2 || v.Keys != 0 {
		t.Fatalf("verified %+v", v)
	}
	expect(t, "add after seal", f.r.Add("src-1", id, docs(doc("c", 3))[0], imp), ErrRejected)
	reason(t, "second batch while unfinished", f.try(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "b9", Count: 1, Digest: sum("x")}}), ReasonBatchOpen)
	reason(t, "wrong effect", f.try(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: sum("x")}}), ReasonEffectChanged)
	expect(t, "unknown batch", f.try(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: "zz", Effect: v.Effect}}), ErrNotFound)
	f.approve(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: v.Effect}})
	got, err := f.r.Document("t1", "src-1", "a")
	if err != nil || got.Number != 1 || !got.IssuedAt.Equal(d1.IssuedAt) {
		t.Fatal(got, err)
	}
	accepted, _ := f.r.Batch("t1", "src-1", id)
	if accepted.State != BatchAccepted || accepted.Classes[ClassNew] != 2 || accepted.Effect != v.Effect {
		t.Errorf("accepted %+v", accepted)
	}
	reason(t, "accept twice", f.try(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: v.Effect}}), ReasonNotVerified)
	reason(t, "reject accepted", f.r.Reject("src-1", id, "late", ann), ReasonNotOpen)
	expect(t, "reuse id", f.try(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: id, Count: 1, Digest: sum("x")}}), ErrConflict)
	if hw, _ := f.r.HighWater("t1", "ws1", "inv"); hw != 2 {
		t.Errorf("high water %d", hw)
	}
	if taken, _ := f.r.PrintedTaken("t1", "ws1", "INV-2"); !taken {
		t.Error("printed number not fenced")
	}
}

func TestAddRules(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "b1", Count: 2, Digest: sum("x")}})
	d := doc("a", 1)
	expect(t, "form", f.r.Add("src-1", "b1", Item{}, imp), ErrInvalid)
	expect(t, "both", f.r.Add("src-1", "b1", Item{Document: &d, Key: &KeyReference{}}, imp), ErrInvalid)
	bad := doc("a", 0)
	expect(t, "document form", f.r.Add("src-1", "b1", Item{Document: &bad}, imp), ErrInvalid)
	secretDoc := doc("a", 1)
	secretDoc.Display = "-----BEGIN"
	reason(t, "secret", f.r.Add("src-1", "b1", Item{Document: &secretDoc}, imp), ReasonSecretMaterial)
	pub, _, _ := publicDER(t)
	reason(t, "secret key id", f.r.Add("src-1", "b1", Item{Key: &KeyReference{SourceID: "private-1", Kind: KeyPublic, DER: pub}}, imp), ReasonSecretMaterial)
	sneaky := doc("a", 1)
	sneaky.SourceSeries = "begin-key"
	reason(t, "secret series", f.r.Add("src-1", "b1", Item{Document: &sneaky}, imp), ReasonSecretMaterial)
	expect(t, "key form", f.r.Add("src-1", "b1", Item{Key: &KeyReference{SourceID: "k", Kind: KeyPublic}}, imp), ErrInvalid)
	expect(t, "actor", f.r.Add("src-1", "b1", Item{Document: &d}, Actor{ID: "x"}), ErrInvalid)
	expect(t, "source", f.r.Add("zz", "b1", Item{Document: &d}, imp), ErrNotFound)
	expect(t, "batch", f.r.Add("src-1", "zz", Item{Document: &d}, imp), ErrNotFound)
	reason(t, "other importer", f.r.Add("src-1", "b1", Item{Document: &d}, imp2), ReasonNotImporter)
	outside := doc("o", 1)
	outside.Workspace = "ws9"
	reason(t, "workspace", f.r.Add("src-1", "b1", Item{Document: &outside}, imp), ReasonWorkspaceNotAllowed)
	f.must(f.r.Add("src-1", "b1", Item{Document: &d}, imp))
	f.must(f.r.Add("src-1", "b1", Item{Document: &d}, imp))
	changed := doc("a", 5)
	expect(t, "same id", f.r.Add("src-1", "b1", Item{Document: &changed}, imp), ErrConflict)
	f.must(f.r.Add("src-1", "b1", docs(doc("b", 2))[0], imp))
	expect(t, "count", f.r.Add("src-1", "b1", docs(doc("c", 3))[0], imp), ErrLimit)
	v, err := f.r.Seal("src-1", "b1", imp)
	if err != nil || v.State != BatchMismatch || v.Computed == v.Digest {
		t.Fatalf("mismatch %+v %v", v, err)
	}
	expect(t, "seal again", second(f.r.Seal("src-1", "b1", imp)), ErrRejected)
	f.approve(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "b2", Count: 3, Digest: sum("x")}})
	f.must(f.r.Add("src-1", "b2", Item{Document: &d}, imp))
	v, _ = f.r.Seal("src-1", "b2", imp)
	if v.State != BatchMismatch {
		t.Error("short batch not mismatch")
	}
	expect(t, "seal form", second(f.r.Seal("", "b2", imp)), ErrInvalid)
}

func TestSealEvidence(t *testing.T) {
	f := standard(t)
	d := doc("a", 1)
	f.ev.missing[d.Content] = true
	_, v := f.batch("src-1", docs(d), "t1")
	if v.State != BatchMissing {
		t.Fatalf("missing %+v", v)
	}
	g := standard(t)
	g.ev.err = errFake
	g.approve(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "b1", Count: 1, Digest: digestOf(g, "src-1", "b1", "t1", docs(d))}})
	g.must(g.r.Add("src-1", "b1", docs(d)[0], imp))
	reason(t, "dependency", second(g.r.Seal("src-1", "b1", imp)), ReasonDependency)
	g.ev.err = nil
	var once sync.Once
	g.ev.during = func() { once.Do(func() { g.must(g.r.Reject("src-1", "b1", "stop", ann)) }) }
	reason(t, "stale", second(g.r.Seal("src-1", "b1", imp)), ReasonStale)
}

var errFake = errorString("synthetic outage")

type errorString string

func (e errorString) Error() string { return string(e) }

func digestOf(f world, source, id, tenant string, items []Item) string {
	b := &batch{spec: BatchSpec{Source: source, ID: id, Count: len(items)}, codes: map[string]string{}}
	for _, it := range items {
		d := *it.Document
		b.codes[d.SourceID] = d.code()
	}
	return batchDigest(tenant, source, b)
}

func TestReject(t *testing.T) {
	f := standard(t)
	id, v := f.batch("src-1", docs(doc("a", 1)), "t1")
	p, err := f.r.Propose(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: v.Effect}}, ann)
	f.must(err)
	reason(t, "assistant", f.r.Reject("src-1", id, "stop", bot), ReasonAINotAllowed)
	expect(t, "reason", f.r.Reject("src-1", id, "", ann), ErrInvalid)
	expect(t, "missing", f.r.Reject("src-1", "zz", "stop", ann), ErrNotFound)
	f.must(f.r.Reject("src-1", id, "Wrong export", ann))
	f.must(f.r.Reject("src-1", id, "again", ann))
	expect(t, "accept retired", second(f.r.Approve(p.Number, bob)), ErrNotFound)
	if b, _ := f.r.Batch("t1", "src-1", id); b.State != BatchRejected {
		t.Error("rejected state")
	}
	f.batch("src-1", docs(doc("a", 1)), "t1")
	g := standard(t)
	s := g.r.sources["src-1"]
	for index := 0; index < maxBatches; index++ {
		s.order = append(s.order, "x"+itoa(index))
	}
	expect(t, "batch records", g.try(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "new", Count: 1, Digest: sum("x")}}), ErrLimit)
}

func TestReimportAndConflicts(t *testing.T) {
	f := standard(t)
	a, b := doc("a", 1), doc("b", 2)
	f.accept("src-1", "t1", docs(a, b))
	v := f.accept("src-1", "t1", docs(a, b))
	if v.Classes[ClassDuplicate] != 2 {
		t.Fatalf("reimport %+v", v)
	}
	changed := a
	changed.Total = 99
	v = f.accept("src-1", "t1", docs(changed))
	if v.Classes[ClassConflict] != 1 {
		t.Fatalf("conflict %+v", v)
	}
	if got, _ := f.r.Document("t1", "src-1", "a"); got.Total != a.Total {
		t.Error("history overwritten")
	}
	q, _ := f.r.Quarantine("t1", "src-1")
	if len(q) != 1 || q[0].Class != ClassConflict || q[0].Stored == "" || q[0].Incoming == q[0].Stored {
		t.Fatalf("quarantine %+v", q)
	}
	f.accept("src-1", "t1", docs(changed))
	if q, _ := f.r.Quarantine("t1", "src-1"); len(q) != 1 {
		t.Errorf("quarantine not deduplicated %d", len(q))
	}
	rekeyed := doc("z", 2)
	ext := doc("y", 7)
	ext.Submitted, ext.ExternalKey = true, "erp-7"
	f.accept("src-1", "t1", docs(ext))
	clash := doc("x", 8)
	clash.ExternalKey = "erp-7"
	shown := doc("w", 9)
	shown.Display = "INV-1"
	v = f.accept("src-1", "t1", docs(rekeyed, clash, shown))
	if v.Classes[ClassNumberConflict] != 2 || v.Classes[ClassExternalConflict] != 1 {
		t.Fatalf("natural keys %+v", v)
	}
	inner1, inner2 := doc("m", 50), doc("n", 50)
	inner2.Display = "INV-50b"
	inner3 := doc("o", 51)
	inner3.Display = "INV-50"
	ex1, ex2 := doc("p", 60), doc("q", 61)
	ex1.ExternalKey, ex2.ExternalKey = "erp-60", "erp-60"
	v = f.accept("src-1", "t1", docs(inner1, inner2, inner3, ex1, ex2))
	if v.Classes[ClassNew] != 2 || v.Classes[ClassNumberConflict] != 2 || v.Classes[ClassExternalConflict] != 1 {
		t.Fatalf("inside batch %+v", v)
	}
	other := doc("a", 1)
	other.Workspace = "ws9"
	v = f.accept("src-2", "t2", docs(other))
	if v.Classes[ClassNew] != 1 {
		t.Errorf("other tenant %+v", v)
	}
}

func TestDeliveryFences(t *testing.T) {
	f := standard(t)
	a := doc("a", 1)
	f.accept("src-1", "t1", docs(a))
	if fenced, _ := f.r.DeliveryFenced("t1", "ws1", "src-1", "a"); fenced {
		t.Error("undelivered fenced")
	}
	delivered := a
	delivered.Delivered = true
	v := f.accept("src-1", "t1", docs(delivered))
	if v.Classes[ClassConflict] != 1 {
		t.Fatal(v)
	}
	if fenced, _ := f.r.DeliveryFenced("t1", "ws1", "src-1", "a"); !fenced {
		t.Error("conflict did not fence delivery")
	}
	b := doc("b", 2)
	b.Submitted, b.ExternalKey = true, "erp-2"
	f.accept("src-1", "t1", docs(b))
	plain := b
	plain.Submitted, plain.ExternalKey = false, "erp-2b"
	f.accept("src-1", "t1", docs(plain))
	for _, key := range []string{"erp-2", "erp-2b"} {
		if fenced, _ := f.r.ExternalFenced("t1", "ws1", key); !fenced {
			t.Errorf("%s not fenced", key)
		}
	}
	if fenced, _ := f.r.ExternalFenced("t2", "ws1", "erp-2"); fenced {
		t.Error("fence leaked across tenants")
	}
	big := doc("c", 900)
	f.accept("src-1", "t1", docs(big))
	lower := doc("d", 5)
	lower.Display = "INV-900"
	f.accept("src-1", "t1", docs(lower))
	if hw, _ := f.r.HighWater("t1", "ws1", "inv"); hw != 900 {
		t.Errorf("high water %d", hw)
	}
	e, err := f.r.Fences("t1", "ws1")
	if err != nil || len(e.HighWater) != 1 || e.HighWater[0].Number != 900 || len(e.Keys) != 4 || len(e.Printed) < 3 {
		t.Fatalf("export %+v %v", e, err)
	}
	for _, k := range append(e.Keys, e.Printed...) {
		if !recoveryValid(k) {
			t.Errorf("%s not recovery-valid", k)
		}
	}
}

func recoveryValid(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func TestUnmappedAndEffectChanges(t *testing.T) {
	f := standard(t)
	other := doc("a", 1)
	other.SourceSeries = "NEW"
	id, v := f.batch("src-1", docs(other), "t1")
	if v.Effect != "" {
		t.Error("effect for unmapped batch")
	}
	reason(t, "unmapped", f.try(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: sum("x")}}), ReasonUnmappedSeries)
	f.approve(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-1", SourceSeries: "NEW", Workspace: "ws1", Series: "inv2"}})
	v, _ = f.r.Batch("t1", "src-1", id)
	p, err := f.r.Propose(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: v.Effect}}, ann)
	f.must(err)
	f.r.sources["src-1"].history["a"] = &entry{code: "other", document: &Document{}}
	reason(t, "stale effect", second(f.r.Approve(p.Number, bob)), ReasonEffectChanged)
}

func TestAtomicCaps(t *testing.T) {
	cases := map[string]func(f world){
		"history":        func(f world) { f.r.tenant("t1").history = maxHistory },
		"history total":  func(f world) { f.r.history = maxHistoryTotal },
		"keys":           func(f world) { f.r.tenant("t1").keys = maxKeyRefs },
		"series":         func(f world) { fill(f.r.tenant("t1").high, maxSeries) },
		"series total":   func(f world) { f.r.series = maxSeriesTotal },
		"printed":        func(f world) { fillBool(f.r.tenant("t1").printed, maxPrinted) },
		"printed total":  func(f world) { f.r.printed = maxPrintedTotal },
		"delivery":       func(f world) { fillBool(f.r.tenant("t1").delivery, maxDelivery) },
		"delivery total": func(f world) { f.r.delivery = maxDeliveryTotal },
		"quarantine":     func(f world) { f.r.sources["src-1"].quarantine = make([]Classified, maxQuarantine) },
	}
	pub, _, _ := publicDER(t)
	for name, prepare := range cases {
		f := standard(t)
		f.approve(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-1", SourceSeries: "LEGACY2", Workspace: "ws1", Series: "inv2"}})
		f.accept("src-1", "t1", docs(doc("a", 1)))
		changed := doc("a", 2)
		d := doc("b", 3)
		d.Delivered = true
		fresh := doc("e", 4)
		fresh.SourceSeries = "LEGACY2"
		items := append(docs(changed, d, fresh), Item{Key: &KeyReference{SourceID: "k", Kind: KeyPublic, DER: pub}})
		id, v := f.batch("src-1", items, "t1")
		before := len(f.r.sources["src-1"].history)
		prepare(f)
		expect(t, name, f.try(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: v.Effect}}), ErrLimit)
		if len(f.r.sources["src-1"].history) != before {
			t.Errorf("%s changed history", name)
		}
	}
	f := standard(t)
	f.r.tenant("t1").history = maxHistory - 1
	f.accept("src-1", "t1", docs(doc("a", 1)))
}

func fill(m map[[2]string]int64, n int) {
	for len(m) < n {
		m[[2]string{"z", itoa(len(m))}] = 1
	}
}

func fillBool(m map[[2]string]bool, n int) {
	for len(m) < n {
		m[[2]string{"z", itoa(len(m))}] = true
	}
}

func TestReadsAcrossTenants(t *testing.T) {
	f := standard(t)
	f.accept("src-1", "t1", docs(doc("a", 1)))
	expect(t, "tenant", second(f.r.Document("t2", "src-1", "a")), ErrNotFound)
	expect(t, "form", second(f.r.Document("", "src-1", "a")), ErrInvalid)
	expect(t, "missing", second(f.r.Document("t1", "src-1", "zz")), ErrNotFound)
	expect(t, "batch tenant", second(f.r.Batch("t2", "src-1", "b1")), ErrNotFound)
	expect(t, "batch missing", second(f.r.Batch("t1", "src-1", "zz")), ErrNotFound)
	expect(t, "quarantine tenant", second(f.r.Quarantine("t2", "src-1")), ErrNotFound)
	expect(t, "fences tenant", second(f.r.Fences("t9", "ws1")), ErrNotFound)
	expect(t, "fences form", second(f.r.Fences("t1", "")), ErrInvalid)
	expect(t, "high form", second(f.r.HighWater("t1", "ws1", "")), ErrInvalid)
	expect(t, "high tenant", second(f.r.HighWater("t9", "ws1", "inv")), ErrNotFound)
	expect(t, "printed form", second(f.r.PrintedTaken("t1", "ws1", "")), ErrInvalid)
	expect(t, "printed tenant", second(f.r.PrintedTaken("t9", "ws1", "x")), ErrNotFound)
	expect(t, "delivery form", second(f.r.DeliveryFenced("t1", "ws1", "", "a")), ErrInvalid)
	expect(t, "delivery tenant", second(f.r.DeliveryFenced("t9", "ws1", "src-1", "a")), ErrNotFound)
	expect(t, "external form", second(f.r.ExternalFenced("t1", "ws1", "")), ErrInvalid)
	expect(t, "external tenant", second(f.r.ExternalFenced("t9", "ws1", "k")), ErrNotFound)
	got, _ := f.r.Document("t1", "src-1", "a")
	got.Total = 1
	if again, _ := f.r.Document("t1", "src-1", "a"); again.Total == 1 {
		t.Error("document read shares storage")
	}
	_ = time.Second
}

func TestRemainingBranches(t *testing.T) {
	if !validText("ok") || validText("") || validText("a\nb") || validText(string(make([]byte, maxTextBytes+1))) || !validHex(sum("x")) || validHex("A"+sum("x")[1:]) || validHex("g"+sum("x")[1:]) || validHex("x") {
		t.Error("primitives")
	}
	if refuse(ReasonStale).Error() != ErrRejected.Error()+": stale" {
		t.Error("refusal text")
	}
	ch := seriesChanges(map[[2]string]int64{{"b", "x"}: 1, {"a", "y"}: 2, {"a", "x"}: 3})
	if ch[0].Workspace != "a" || ch[0].Series != "x" || ch[1].Series != "y" || ch[2].Workspace != "b" {
		t.Errorf("series order %+v", ch)
	}
	f := standard(t)
	f.r.retireAccepts("nowhere", "b1")
	f.approve(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-1", SourceSeries: "LEGACY", Workspace: "ws2", Series: "inv"}})
	d := doc("w2", 3)
	d.Workspace = "ws2"
	d.Delivered = true
	f.approve(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-1", SourceSeries: "LEGACY2", Workspace: "ws1", Series: "inv0"}})
	second := doc("s2", 4)
	second.SourceSeries = "LEGACY2"
	f.accept("src-1", "t1", docs(doc("a", 1), d, second))
	e, _ := f.r.Fences("t1", "ws1")
	if len(e.HighWater) != 2 || e.HighWater[0].Series != "inv" || len(e.Printed) != 2 || len(e.Keys) != 0 {
		t.Errorf("workspace filter %+v", e)
	}
}
