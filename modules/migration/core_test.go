package migration

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestConstantsAndForms(t *testing.T) {
	got := []int64{maxSources, maxWorkspaces, maxItems, maxBatches, maxHistory, maxHistoryTotal, maxQuarantine, maxSeries, maxSeriesTotal, maxPrinted, maxPrintedTotal,
		maxDelivery, maxDeliveryTotal, maxKeyRefs, maxProposals, maxNumber, maxDER, maxIDBytes, maxTextBytes}
	want := []int64{10000, 64, 100000, 1000, 1000000, 10000000, 100000, 10000, 100000, 1000000, 10000000, 2000000, 20000000, 10000, 1000, 1000000000000000, 16384, 128, 200}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("constant %d = %d", index, got[index])
		}
	}
	expect(t, "evidence", second(NewRegistry(Config{}, func() time.Time { return start })), ErrInvalid)
	expect(t, "clock", second(NewRegistry(Config{Evidence: &evidence{}}, nil)), ErrInvalid)
	d := doc("a", 1)
	if !d.valid(start) {
		t.Fatal("fixture refused")
	}
	bad := map[string]func(*Document){
		"id":        func(d *Document) { d.SourceID = "" },
		"kind":      func(d *Document) { d.Kind = "memo" },
		"workspace": func(d *Document) { d.Workspace = "" },
		"series":    func(d *Document) { d.SourceSeries = "" },
		"zero":      func(d *Document) { d.Number = 0 },
		"big":       func(d *Document) { d.Number = maxNumber + 1 },
		"display":   func(d *Document) { d.Display = "" },
		"long":      func(d *Document) { d.Display = strings.Repeat("x", 65) },
		"control":   func(d *Document) { d.Display = "a\nb" },
		"high":      func(d *Document) { d.Display = "a\x7f" },
		"before":    func(d *Document) { d.IssuedAt = time.Unix(-1, 0) },
		"future":    func(d *Document) { d.IssuedAt = start.Add(time.Nanosecond) },
		"currency":  func(d *Document) { d.Currency = "krw" },
		"cur len":   func(d *Document) { d.Currency = "KRWX" },
		"cur low":   func(d *Document) { d.Currency = "K@W" },
		"total":     func(d *Document) { d.Total = -1 },
		"total big": func(d *Document) { d.Total = maxNumber + 1 },
		"content":   func(d *Document) { d.Content = "x" },
		"external":  func(d *Document) { d.ExternalKey = "a b" },
		"submitted": func(d *Document) { d.Submitted = true },
	}
	for name, change := range bad {
		c := doc("a", 1)
		change(&c)
		if c.valid(start) {
			t.Errorf("%s accepted", name)
		}
	}
	good := map[string]func(*Document){
		"edge number": func(d *Document) { d.Number = maxNumber },
		"edge total":  func(d *Document) { d.Total = maxNumber },
		"now":         func(d *Document) { d.IssuedAt = start },
		"epoch":       func(d *Document) { d.IssuedAt = time.Unix(0, 0) },
		"display":     func(d *Document) { d.Display = "~ " + strings.Repeat("x", 62) },
		"submitted":   func(d *Document) { d.Submitted, d.ExternalKey = true, "erp-1" },
		"kinds":       func(d *Document) { d.Kind = DocumentReceipt },
		"credit":      func(d *Document) { d.Kind = DocumentCredit },
		"debit":       func(d *Document) { d.Kind = DocumentDebit },
		"zero total":  func(d *Document) { d.Total = 0 },
		"cur edge":    func(d *Document) { d.Currency = "AZZ" },
	}
	for name, change := range good {
		c := doc("a", 1)
		change(&c)
		if !c.valid(start) {
			t.Errorf("%s refused", name)
		}
	}
	for _, v := range []string{"PRIVATE", "private", "-----begin", "Begin"} {
		if !secret("x", v) {
			t.Errorf("%s not secret", v)
		}
	}
	if secret("invoice", "INV-1") {
		t.Error("plain text treated as secret")
	}
	if boolText(true) != "1" || boolText(false) != "0" {
		t.Error("bool text")
	}
}

func TestKeyReferences(t *testing.T) {
	pub, priv, cert := publicDER(t)
	if k, err := keyView(KeyReference{SourceID: "k1", Kind: KeyPublic, DER: pub}); err != nil || k.Fingerprint != sumBytes(pub) {
		t.Errorf("public %v", err)
	}
	if _, err := keyView(KeyReference{SourceID: "k2", Kind: KeyCertificate, DER: cert}); err != nil {
		t.Error(err)
	}
	reason(t, "pkcs8", second(keyView(KeyReference{SourceID: "k3", Kind: KeyPublic, DER: priv})), ReasonSecretMaterial)
	expect(t, "garbage", second(keyView(KeyReference{SourceID: "k4", Kind: KeyPublic, DER: []byte(strings.Repeat("a", 64))})), ErrInvalid)
	expect(t, "wrong kind", second(keyView(KeyReference{SourceID: "k5", Kind: KeyCertificate, DER: pub})), ErrInvalid)
	expect(t, "kind", second(keyView(KeyReference{SourceID: "k6", Kind: "secret", DER: pub})), ErrInvalid)
	expect(t, "empty", second(keyView(KeyReference{SourceID: "k7", Kind: KeyPublic})), ErrInvalid)
	expect(t, "large", second(keyView(KeyReference{SourceID: "k8", Kind: KeyPublic, DER: make([]byte, maxDER+1)})), ErrInvalid)
	expect(t, "id", second(keyView(KeyReference{Kind: KeyPublic, DER: pub})), ErrInvalid)
	rsaDER, ecDER := otherPrivate(t)
	reason(t, "pkcs1", second(keyView(KeyReference{SourceID: "k9", Kind: KeyPublic, DER: rsaDER})), ReasonSecretMaterial)
	reason(t, "ec", second(keyView(KeyReference{SourceID: "k10", Kind: KeyPublic, DER: ecDER})), ReasonSecretMaterial)
	f := standard(t)
	v := f.accept("src-1", "t1", []Item{{Key: &KeyReference{SourceID: "k1", Kind: KeyPublic, DER: pub}}})
	if v.Classes[ClassNew] != 1 {
		t.Fatalf("key batch %+v", v)
	}
	k, err := f.r.Key("t1", "src-1", "k1")
	if err != nil || k.Fingerprint != sumBytes(pub) {
		t.Fatal(k, err)
	}
	k.DER[0]++
	if again, _ := f.r.Key("t1", "src-1", "k1"); again.DER[0] == k.DER[0] {
		t.Error("key view shares bytes")
	}
	expect(t, "not a key", second(f.r.Key("t1", "src-1", "nope")), ErrNotFound)
	expect(t, "key tenant", second(f.r.Key("t2", "src-1", "k1")), ErrNotFound)
}

func TestProposalFlow(t *testing.T) {
	f := setup(t)
	c := Change{Kind: ChangeSource, Source: Source{ID: "s", Tenant: "t", Importer: "i", Workspaces: []string{"w"}}, Batch: BatchSpec{ID: "ignored"}}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, imp)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{ID: "x"})), ErrInvalid)
	expect(t, "kind", second(f.r.Propose(Change{Kind: "x"}, ann)), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if p.Change.Batch.ID != "" {
		t.Error("unrelated payload kept")
	}
	again, _ := f.r.Propose(c, ann)
	other, _ := f.r.Propose(c, cat)
	if again.Number != p.Number || other.Number == p.Number {
		t.Error("numbering")
	}
	reason(t, "self", second(f.r.Approve(p.Number, ann)), ReasonSelfApproval)
	reason(t, "approve assistant", second(f.r.Approve(p.Number, bot)), ReasonAINotAllowed)
	expect(t, "missing", second(f.r.Approve(99, bob)), ErrNotFound)
	f.must(second(f.r.Approve(p.Number, bob)))
	if v, err := f.r.Approve(p.Number, bob); err != nil || v.State != ProposalApplied {
		t.Error("repeat")
	}
	reason(t, "third", second(f.r.Approve(p.Number, cat)), ReasonNotPending)
	expect(t, "stale", second(f.r.Approve(other.Number, bob)), ErrConflict)
	reason(t, "withdraw other", second(f.r.Withdraw(other.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw applied", second(f.r.Withdraw(p.Number, ann)), ReasonNotPending)
	reason(t, "withdraw assistant", second(f.r.Withdraw(other.Number, bot)), ReasonAINotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, ann)), ErrNotFound)
	if w, err := f.r.Withdraw(other.Number, cat); err != nil || w.State != ProposalWithdrawn {
		t.Error("withdraw")
	}
	expect(t, "view withdrawn", second(f.r.Proposal(other.Number)), ErrNotFound)
	f.must(second(f.r.Proposal(p.Number)))
	expect(t, "unknown scope", second(f.r.Propose(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "zz", SourceSeries: "a", Workspace: "w", Series: "s"}}, ann)), ErrNotFound)
	g := setup(t)
	limit := (maxProposals + 3) / 4
	for index := 0; index < limit; index++ {
		g.must(second(g.r.Propose(Change{Kind: ChangeSource, Source: Source{ID: "s" + itoa(index), Tenant: "t", Importer: "i" + itoa(index), Workspaces: []string{"w"}}}, ann)))
	}
	expect(t, "per proposer", second(g.r.Propose(Change{Kind: ChangeSource, Source: Source{ID: "over", Tenant: "t", Importer: "io", Workspaces: []string{"w"}}}, ann)), ErrLimit)
	for _, actor := range []Actor{bob, cat, {ID: "dan", Kind: KindHuman}} {
		for index := 0; index < limit; index++ {
			g.must(second(g.r.Propose(Change{Kind: ChangeSource, Source: Source{ID: actor.ID + itoa(index), Tenant: "t", Importer: "i" + actor.ID + itoa(index), Workspaces: []string{"w"}}}, actor)))
		}
	}
	expect(t, "scope", second(g.r.Propose(Change{Kind: ChangeSource, Source: Source{ID: "eve", Tenant: "t", Importer: "ie", Workspaces: []string{"w"}}}, Actor{ID: "eve", Kind: KindHuman})), ErrLimit)
	h := setup(t)
	var numbers []uint64
	for index := 0; index < maxProposals+2; index++ {
		numbers = append(numbers, h.approve(Change{Kind: ChangeSource, Source: Source{ID: "s" + itoa(index), Tenant: "t", Importer: "i" + itoa(index), Workspaces: []string{"w"}}}).Number)
	}
	expect(t, "pruned", second(h.r.Proposal(numbers[1])), ErrNotFound)
	h.must(second(h.r.Proposal(numbers[2])))
}

func TestSourcesAndMappings(t *testing.T) {
	f := standard(t)
	forms := map[string]Change{
		"source id":     {Kind: ChangeSource, Source: Source{Tenant: "t", Importer: "i", Workspaces: []string{"w"}}},
		"no workspaces": {Kind: ChangeSource, Source: Source{ID: "s", Tenant: "t", Importer: "i"}},
		"order":         {Kind: ChangeSource, Source: Source{ID: "s", Tenant: "t", Importer: "i", Workspaces: []string{"b", "a"}}},
		"importer":      {Kind: ChangeImporter, Source: Source{ID: "src-1"}},
		"mapping":       {Kind: ChangeMapping, Mapping: Mapping{Source: "src-1"}},
		"batch count":   {Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "b", Digest: sum("x")}},
		"batch big":     {Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "b", Count: maxItems + 1, Digest: sum("x")}},
		"batch digest":  {Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "b", Count: 1, Digest: "x"}},
		"accept":        {Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: "b"}},
	}
	for name, c := range forms {
		expect(t, name, f.try(c), ErrInvalid)
	}
	many := make([]string, maxWorkspaces+1)
	for index := range many {
		many[index] = "w" + itoa(100+index)
	}
	expect(t, "many workspaces", f.try(Change{Kind: ChangeSource, Source: Source{ID: "s", Tenant: "t", Importer: "i", Workspaces: many}}), ErrInvalid)
	f.approve(Change{Kind: ChangeSource, Source: Source{ID: "edge", Tenant: "t1", Importer: "i-edge", Workspaces: many[:maxWorkspaces]}})
	expect(t, "twice", f.try(Change{Kind: ChangeSource, Source: Source{ID: "src-1", Tenant: "t1", Importer: "i9", Workspaces: []string{"ws1"}}}), ErrConflict)
	reason(t, "importer taken", f.try(Change{Kind: ChangeSource, Source: Source{ID: "s3", Tenant: "t1", Importer: "imp-1", Workspaces: []string{"ws1"}}}), ReasonIdentityTaken)
	reason(t, "claimed", f.try(Change{Kind: ChangeSource, Source: Source{ID: "s3", Tenant: "t2", Importer: "i3", Workspaces: []string{"ws0", "ws1"}}}), ReasonWorkspaceClaimed)
	f.approve(Change{Kind: ChangeSource, Source: Source{ID: "s3", Tenant: "t1", Importer: "i3", Workspaces: []string{"ws1"}}})
	reason(t, "outside", f.try(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-1", SourceSeries: "X", Workspace: "ws9", Series: "inv"}}), ReasonWorkspaceNotAllowed)
	expect(t, "remap", f.try(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-1", SourceSeries: "LEGACY", Workspace: "ws1", Series: "other"}}), ErrConflict)
	reason(t, "importer reuse", f.try(Change{Kind: ChangeImporter, Source: Source{ID: "src-1", Importer: "imp-2"}}), ReasonIdentityTaken)
	f.approve(Change{Kind: ChangeImporter, Source: Source{ID: "src-1", Importer: "imp-new"}})
	d := doc("x", 1)
	reason(t, "old importer", f.r.Add("src-1", "b1", Item{Document: &d}, imp), ReasonNotImporter)
	for len(f.r.sources) < maxSources {
		f.r.sources["x"+itoa(len(f.r.sources))] = &source{}
	}
	expect(t, "sources", f.try(Change{Kind: ChangeSource, Source: Source{ID: "s4", Tenant: "t1", Importer: "i4", Workspaces: []string{"ws1"}}}), ErrLimit)
}

func itoa(n int) string { return strconv.Itoa(n) }
