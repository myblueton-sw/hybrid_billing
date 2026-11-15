package recovery

import (
	"strings"
	"testing"
	"time"
)

func TestConstantsAndPrimitives(t *testing.T) {
	got := []int64{maxInstallations, maxWriters, maxExternals, maxLive, maxOpenPerWriter, maxOpen, maxClosed, maxChunks, maxItems, maxPages, maxSeries, maxKeys, maxKeysTotal,
		maxEvents, maxEventsTotal, maxHistory, maxProposals, maxBegins, int64(maxSkew), int64(minMaxCall), int64(maxMaxCall), maxMismatch}
	want := []int64{100000, 16, 16, 1000, 1000, 10000, 10000, 10000, 10000, 1000, 1000, 1000000, 10000000, 1000000, 10000000, 100, 1000, 1000, int64(time.Hour),
		int64(time.Second), int64(time.Hour), 100}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("constant %d = %d", index, got[index])
		}
	}
	if strings.Join(domains, ",") != "database,evidence,keys,policies,modules,numbering,submissions,outbox,deletions" {
		t.Error("domains")
	}
	for _, id := range []string{"a", "Z", "z", "9", "A.z_0:9-", strings.Repeat("x", maxIDBytes)} {
		if !validID(id) {
			t.Errorf("%q refused", id)
		}
	}
	for _, id := range []string{"", strings.Repeat("x", maxIDBytes+1), "a b", "[", "`", "{", "@", "/"} {
		if validID(id) {
			t.Errorf("%q accepted", id)
		}
	}
	for _, text := range []string{"a", strings.Repeat("x", maxTextBytes), " "} {
		if !validText(text) {
			t.Errorf("%q refused", text)
		}
	}
	for _, text := range []string{"", strings.Repeat("x", maxTextBytes+1), "a\nb", "\x7f", "\u0080", "\u009f", "\xff"} {
		if validText(text) {
			t.Errorf("%q accepted", text)
		}
	}
	if !validHex(strings.Repeat("0a", 32)) || !validHex(strings.Repeat("f9", 32)) {
		t.Error("hex")
	}
	for _, text := range []string{strings.Repeat("0", 63), strings.Repeat("0", 65), strings.Repeat("A", 64), "g" + strings.Repeat("0", 63), strings.Repeat("/", 64), strings.Repeat(":", 64), strings.Repeat("`", 64)} {
		if validHex(text) {
			t.Errorf("%q accepted", text)
		}
	}
	expect(t, "clock", second(NewRegistry(nil)), ErrInvalid)
	err := refuse(ReasonFenced)
	if err.Error() != ErrRejected.Error()+": fenced" {
		t.Error(err)
	}
	expect(t, "unwrap", err, ErrRejected)
}

func TestManifestRules(t *testing.T) {
	ok := []Inventory{{Items: 0}, {Items: 3, Chunks: []Chunk{{Start: 0, End: 3, Checksum: sum("a")}}}, {Items: 4, Chunks: []Chunk{{Start: 0, End: 1, Checksum: sum("a")}, {Start: 1, End: 4, Checksum: sum("b")}}}}
	for index, inv := range ok {
		if !inv.valid() {
			t.Errorf("valid %d refused", index)
		}
	}
	bad := map[string]Inventory{
		"gap":      {Items: 4, Chunks: []Chunk{{Start: 0, End: 1, Checksum: sum("a")}, {Start: 2, End: 4, Checksum: sum("b")}}},
		"overlap":  {Items: 4, Chunks: []Chunk{{Start: 0, End: 2, Checksum: sum("a")}, {Start: 1, End: 4, Checksum: sum("b")}}},
		"empty":    {Items: 4, Chunks: []Chunk{{Start: 0, End: 0, Checksum: sum("a")}, {Start: 0, End: 4, Checksum: sum("b")}}},
		"short":    {Items: 4, Chunks: []Chunk{{Start: 0, End: 3, Checksum: sum("a")}}},
		"late":     {Items: 4, Chunks: []Chunk{{Start: 1, End: 4, Checksum: sum("a")}}},
		"checksum": {Items: 1, Chunks: []Chunk{{Start: 0, End: 1, Checksum: "x"}}},
		"items":    {Items: 1},
	}
	for name, inv := range bad {
		if inv.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	big := Inventory{Items: maxChunks + 1}
	for index := 0; index <= maxChunks; index++ {
		big.Chunks = append(big.Chunks, Chunk{Start: uint64(index), End: uint64(index + 1), Checksum: sum("c")})
	}
	if big.valid() {
		t.Error("too many chunks accepted")
	}
	big.Chunks, big.Items = big.Chunks[:maxChunks], maxChunks
	if !big.valid() {
		t.Error("full chunks refused")
	}
	m := manifest()
	base := manifestDigest(m)
	if base != manifestAnswer {
		t.Errorf("manifest digest %s", base)
	}
	changes := map[string]func([]Inventory){
		"domain":   func(m []Inventory) { m[0].Domain = "x" },
		"items":    func(m []Inventory) { m[0].Items = 11 },
		"start":    func(m []Inventory) { m[0].Chunks[1].Start = 5 },
		"end":      func(m []Inventory) { m[0].Chunks[0].End = 5 },
		"checksum": func(m []Inventory) { m[0].Chunks[0].Checksum = sum("x") },
		"chunks":   func(m []Inventory) { m[0].Chunks = m[0].Chunks[:1] },
		"count":    func(m []Inventory) { m[8] = Inventory{} },
	}
	for name, change := range changes {
		c := copyManifest(m)
		change(c)
		if manifestDigest(c) == base {
			t.Errorf("%s does not change the digest", name)
		}
	}
	c := copyManifest(m)
	c[0].Chunks[0].Checksum = "x"
	if m[0].Chunks[0].Checksum == "x" {
		t.Error("copy shares chunks")
	}
}

const manifestAnswer = "a6c9ebb2b05e63ac402826b7de861da470b14d49f95b0a35571981a5f1f8493b"

func TestProposalFlow(t *testing.T) {
	f := setup(t)
	c := Change{Kind: ChangeInstallation, Installation: Installation{ID: "i1", Tenant: "t"}, Writer: Writer{ID: "ignored"}}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, erp)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{ID: "x"})), ErrInvalid)
	expect(t, "kind", second(f.r.Propose(Change{Kind: "x"}, ann)), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if p.Change.Writer.ID != "" {
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
	expect(t, "unknown scope", second(f.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "i9", ID: "x", Kind: WriterERP}}, ann)), ErrNotFound)
}

func TestRegistrationRules(t *testing.T) {
	f := standard(t)
	forms := map[string]Change{
		"installation id":   {Kind: ChangeInstallation, Installation: Installation{Tenant: "t"}},
		"installation skew": {Kind: ChangeInstallation, Installation: Installation{ID: "i3", Tenant: "t", Skew: maxSkew + 1}},
		"negative skew":     {Kind: ChangeInstallation, Installation: Installation{ID: "i3", Tenant: "t", Skew: -1}},
		"writer kind":       {Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-x", Kind: "fax", MaxCall: time.Minute}},
		"writer call low":   {Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-x", Kind: WriterERP, MaxCall: minMaxCall - 1}},
		"writer call high":  {Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-x", Kind: WriterERP, MaxCall: maxMaxCall + 1}},
		"external kind":     {Kind: ChangeExternal, External: External{Installation: "i1", ID: "x", Kind: "fax"}},
		"verifier":          {Kind: ChangeVerifier, Verifier: Verifier{Installation: "i1"}},
		"unlock lost":       {Kind: ChangeUnlock, Unlock: Unlock{Installation: "i1", Generation: 1, Lost: "x"}},
		"unlock gen":        {Kind: ChangeUnlock, Unlock: Unlock{Installation: "i1", Lost: lostDigest(nil)}},
		"prune":             {Kind: ChangePrune, Prune: Prune{Installation: "i1"}},
		"abandon":           {Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1"}},
	}
	for name, c := range forms {
		expect(t, name, f.try(c), ErrInvalid)
	}
	edge := []Change{
		{Kind: ChangeInstallation, Installation: Installation{ID: "i3", Tenant: "t", Skew: maxSkew}},
		{Kind: ChangeWriter, Writer: Writer{Installation: "i3", ID: "w-low", Kind: WriterPayment, MaxCall: minMaxCall}},
		{Kind: ChangeWriter, Writer: Writer{Installation: "i3", ID: "w-high", Kind: WriterMessage, MaxCall: maxMaxCall}},
		{Kind: ChangeWriter, Writer: Writer{Installation: "i3", ID: "w-del", Kind: WriterDelivery, MaxCall: time.Minute}},
	}
	for _, c := range edge {
		f.approve(c)
	}
	expect(t, "installation twice", f.try(Change{Kind: ChangeInstallation, Installation: Installation{ID: "i1", Tenant: "t"}}), ErrConflict)
	reason(t, "writer identity", f.try(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i2", ID: "w-erp", Kind: WriterERP, MaxCall: time.Minute}}), ReasonIdentityTaken)
	reason(t, "verifier identity", f.try(Change{Kind: ChangeVerifier, Verifier: Verifier{Installation: "i2", ID: "w-erp"}}), ReasonIdentityTaken)
	expect(t, "external twice", f.try(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "erp-x", Kind: WriterERP}}), ErrConflict)
	for index := 0; index < maxWriters-2; index++ {
		f.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-n" + itoa(index), Kind: WriterERP, MaxCall: time.Minute}})
	}
	expect(t, "writers", f.try(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-over", Kind: WriterERP, MaxCall: time.Minute}}), ErrLimit)
	f.must(f.r.RemoveWriter("i1", "w-n0", ann))
	f.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-over", Kind: WriterERP, MaxCall: time.Minute}})
	reason(t, "reuse removed identity", f.try(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-n0", Kind: WriterERP, MaxCall: time.Minute}}), ReasonIdentityTaken)
	for index := 0; index < maxExternals-2; index++ {
		f.approve(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "x" + itoa(index), Kind: WriterERP}})
	}
	expect(t, "externals", f.try(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "x-over", Kind: WriterERP}}), ErrLimit)
	f.approve(Change{Kind: ChangeVerifier, Verifier: Verifier{Installation: "i1", ID: "v-2"}})
	reason(t, "old verifier", second(f.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k"}, ver)), ReasonNotWriter)
	expect(t, "remove form", f.r.RemoveWriter("i1", "", ann), ErrInvalid)
	expect(t, "remove missing", f.r.RemoveWriter("i1", "w-zz", ann), ErrNotFound)
	expect(t, "remove site", f.r.RemoveWriter("i9", "w-erp", ann), ErrNotFound)
	reason(t, "remove assistant", f.r.RemoveWriter("i1", "w-erp", bot), ReasonAINotAllowed)
}
