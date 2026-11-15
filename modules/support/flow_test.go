package support

import (
	"strings"
	"testing"
	"time"
)

func TestProposals(t *testing.T) {
	f := setupWith(t, limits)
	c := Change{Kind: ChangeSite, Site: Site{ID: "s1", Tenant: "t1"}, Upload: Upload{Site: "x"}}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, agent)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{ID: "x", Kind: "robot"})), ErrInvalid)
	expect(t, "kind", second(f.r.Propose(Change{Kind: "x"}, ann)), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if p.Change.Upload.Site != "" {
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
	expect(t, "unknown site", second(f.r.Propose(Change{Kind: ChangeUpload, Upload: Upload{Site: "s9", Destination: "d"}}, ann)), ErrNotFound)
	expect(t, "site twice", f.try(Change{Kind: ChangeSite, Site: Site{ID: "s1", Tenant: "t"}}), ErrConflict)
	expect(t, "site form", f.try(Change{Kind: ChangeSite, Site: Site{ID: "s2"}}), ErrInvalid)
	f.approve(Change{Kind: ChangeSite, Site: Site{ID: "s2", Tenant: "t"}})
	f.approve(Change{Kind: ChangeSite, Site: Site{ID: "s3", Tenant: "t"}})
	expect(t, "site room", f.try(Change{Kind: ChangeSite, Site: Site{ID: "s4", Tenant: "t"}}), ErrLimit)
}

func TestProposalCaps(t *testing.T) {
	l := limits
	l.Proposals = 5
	f := setupWith(t, l)
	site := func(id string, actor Actor) error {
		return second(f.r.Propose(Change{Kind: ChangeSite, Site: Site{ID: id, Tenant: "t"}}, actor))
	}
	f.must(site("a", ann))
	f.must(site("b", ann))
	expect(t, "per proposer", site("c", ann), ErrLimit)
	f.must(site("c", bob))
	f.must(site("d", bob))
	f.must(site("e", cat))
	expect(t, "scope", site("f", Actor{ID: "dan", Kind: KindHuman}), ErrLimit)
	g := setupWith(t, Limits{Sites: 10, Destinations: 2, Bundles: 1, SiteBytes: 1, Bytes: 1, BundleLife: time.Hour, Proposals: 2})
	var numbers []uint64
	for _, id := range []string{"a", "b", "c", "d"} {
		numbers = append(numbers, g.approve(Change{Kind: ChangeSite, Site: Site{ID: id, Tenant: "t"}}).Number)
	}
	expect(t, "pruned", second(g.r.Proposal(numbers[0])), ErrNotFound)
	expect(t, "pruned next", second(g.r.Proposal(numbers[1])), ErrNotFound)
	g.must(second(g.r.Proposal(numbers[2])))
	h := setupWith(t, Limits{Sites: 10, Destinations: 2, Bundles: 1, SiteBytes: 1, Bytes: 1, BundleLife: time.Hour, Proposals: 1000})
	for index := 0; index < maxSchemaPending; index++ {
		actor := Actor{ID: "h" + names(20)[index], Kind: KindHuman}
		h.must(second(h.r.Propose(Change{Kind: ChangeSchema, Schema: Schema{Source: SourceMetric, Fields: []FieldSpec{{Key: names(20)[index], Class: ClassInteger}}}}, actor)))
	}
	expect(t, "schema pending", second(h.r.Propose(Change{Kind: ChangeSchema, Schema: Schema{Source: SourceMetric, Fields: []FieldSpec{{Key: "zz", Class: ClassInteger}}}}, ann)), ErrLimit)
	h.must(second(h.r.Propose(Change{Kind: ChangeSite, Site: Site{ID: "s", Tenant: "t"}}, ann)))
}

func TestSchemaAndDestinationChanges(t *testing.T) {
	f := standard(t)
	reason(t, "sensitive", f.try(Change{Kind: ChangeSchema, Schema: Schema{Source: SourceConfig, Fields: []FieldSpec{{Key: "db_password", Class: ClassEnum, Values: []string{"set"}}}}}), ReasonSensitiveKey)
	expect(t, "schema form", f.try(Change{Kind: ChangeSchema, Schema: Schema{Source: "x"}}), ErrInvalid)
	expect(t, "destination form", f.try(Change{Kind: ChangeDestination, Destination: Destination{ID: "d", Host: "A", Port: 443}}), ErrInvalid)
	expect(t, "port low", f.try(Change{Kind: ChangeDestination, Destination: Destination{ID: "d", Host: "a.example", Port: 0}}), ErrInvalid)
	expect(t, "port high", f.try(Change{Kind: ChangeDestination, Destination: Destination{ID: "d", Host: "a.example", Port: 65536}}), ErrInvalid)
	f.approve(Change{Kind: ChangeDestination, Destination: Destination{ID: "edge", Host: "a.example", Port: 65535}})
	expect(t, "destination twice", f.try(Change{Kind: ChangeDestination, Destination: Destination{ID: "vendor", Host: "b.example", Port: 1}}), ErrConflict)
	expect(t, "destination room", f.try(Change{Kind: ChangeDestination, Destination: Destination{ID: "third", Host: "c.example", Port: 1}}), ErrLimit)
	expect(t, "upload form", f.try(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1"}}), ErrInvalid)
	expect(t, "upload missing", f.try(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1", Destination: "nope"}}), ErrNotFound)
	f.must(f.r.DisableDestination("edge", ann))
	reason(t, "upload disabled destination", f.try(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1", Destination: "edge"}}), ReasonDestinationDisabled)
	expect(t, "re-approve", f.try(Change{Kind: ChangeDestination, Destination: Destination{ID: "edge", Host: "a.example", Port: 1}}), ErrConflict)
	expect(t, "disable form", f.r.DisableDestination("", ann), ErrInvalid)
	expect(t, "disable missing", f.r.DisableDestination("nope", ann), ErrNotFound)
	reason(t, "disable assistant", f.r.DisableDestination("vendor", bot), ReasonAINotAllowed)
	expect(t, "upload disable form", f.r.DisableUpload("", ann), ErrInvalid)
	expect(t, "upload disable missing", f.r.DisableUpload("s9", ann), ErrNotFound)
	reason(t, "upload disable assistant", f.r.DisableUpload("s1", bot), ReasonAINotAllowed)
}

func TestPreviewExportUpload(t *testing.T) {
	f := standard(t)
	b := f.bundle("s1")
	reason(t, "export unpreviewed", second(f.r.Export("s1", b.ID, b.Digest, ann)), ReasonNotPreviewed)
	p, err := f.r.Preview("s1", b.ID, ann)
	f.must(err)
	if p.State != StatePreviewed || p.Digest != b.Digest || sum(p.Bytes) != p.Digest {
		t.Fatalf("preview %+v", p)
	}
	p.Bytes[0]++
	p.Items[0].Fields[0].Value = "x"
	p.Summary.Pseudonyms[ClassHost] = 99
	if again, _ := f.r.Preview("s1", b.ID, ann); sum(again.Bytes) != b.Digest || again.Items[0].Fields[0].Value == "x" || again.Summary.Pseudonyms[ClassHost] == 99 {
		t.Error("preview shares storage")
	}
	reason(t, "other human", second(f.r.Export("s1", b.ID, b.Digest, bob)), ReasonNotPreviewed)
	expect(t, "digest form", second(f.r.Export("s1", b.ID, "x", ann)), ErrInvalid)
	reason(t, "digest mismatch", second(f.r.Export("s1", b.ID, sum([]byte("x")), ann)), ReasonDigestMismatch)
	data, err := f.r.Export("s1", b.ID, b.Digest, ann)
	if err != nil || sum(data) != b.Digest {
		t.Fatalf("export %v", err)
	}
	reason(t, "export assistant", second(f.r.Export("s1", b.ID, b.Digest, bot)), ReasonAINotAllowed)
	reason(t, "upload disabled by default", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)), ReasonUploadDisabled)
	f.approve(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1", Destination: "vendor"}})
	reason(t, "reset after enablement", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)), ReasonNotPreviewed)
	f.must(second(f.r.Preview("s1", b.ID, ann)))
	expect(t, "destination form", second(f.r.Upload("s1", b.ID, b.Digest, "", ann)), ErrInvalid)
	reason(t, "other destination", second(f.r.Upload("s1", b.ID, b.Digest, "edge", ann)), ReasonUploadDisabled)
	reason(t, "system upload", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", agent)), ReasonNotHuman)
	rec, err := f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)
	if err != nil || rec.Digest != b.Digest || rec.By != "ann" || rec.Destination != "vendor" || !rec.At.Equal(start) {
		t.Fatalf("upload %+v %v", rec, err)
	}
	f.advance(time.Minute)
	if again, _ := f.r.Upload("s1", b.ID, b.Digest, "vendor", ann); !again.At.Equal(start) {
		t.Error("repeat upload not idempotent")
	}
	reason(t, "digest mismatch upload", second(f.r.Upload("s1", b.ID, sum([]byte("x")), "vendor", ann)), ReasonDigestMismatch)
	other := f.bundle("s2")
	f.must(second(f.r.Preview("s2", other.ID, ann)))
	reason(t, "other site disabled", second(f.r.Upload("s2", other.ID, other.Digest, "vendor", ann)), ReasonUploadDisabled)
	f.must(f.r.DisableUpload("s1", ann))
	reason(t, "after disable", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)), ReasonNotPreviewed)
	f.must(second(f.r.Preview("s1", b.ID, ann)))
	reason(t, "disabled", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)), ReasonUploadDisabled)
	f.must(second(f.r.Export("s1", b.ID, b.Digest, ann)))
	f.approve(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1", Destination: "vendor"}})
	f.approve(Change{Kind: ChangeUpload, Upload: Upload{Site: "s2", Destination: "vendor"}})
	f.must(second(f.r.Preview("s1", b.ID, ann)))
	f.must(second(f.r.Preview("s2", other.ID, ann)))
	f.must(f.r.DisableDestination("vendor", ann))
	reason(t, "destination disabled s1", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)), ReasonNotPreviewed)
	reason(t, "destination disabled s2", second(f.r.Export("s2", other.ID, other.Digest, ann)), ReasonNotPreviewed)
	f.must(second(f.r.Preview("s1", b.ID, ann)))
	reason(t, "no destination", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)), ReasonUploadDisabled)
	f.r.sites["s1"].destination = "vendor"
	reason(t, "defensive disabled", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)), ReasonUploadDisabled)
	f.r.destinations["ghost"] = nil
	f.r.sites["s1"].destination = "ghost"
	reason(t, "defensive missing", second(f.r.Upload("s1", b.ID, b.Digest, "ghost", ann)), ReasonUploadDisabled)
}

func TestUploadRecordBound(t *testing.T) {
	f := standard(t)
	b := f.bundle("s1")
	f.must(second(f.r.Preview("s1", b.ID, ann)))
	f.r.sites["s1"].destination = "vendor"
	for index := 0; index < maxUploads; index++ {
		f.r.sites["s1"].bundles[b.ID].uploads = append(f.r.sites["s1"].bundles[b.ID].uploads, UploadRecord{Destination: "d" + names(16)[index]})
	}
	expect(t, "records", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)), ErrLimit)
}

func TestSchemaChangePurges(t *testing.T) {
	f := standard(t)
	logs := f.bundle("s1")
	versions, err := f.r.Create("s2", []Item{{Source: SourceVersion, Fields: []Field{{Key: "product", Text: "2.4.0"}}}}, ann)
	f.must(err)
	before := f.r.bytes
	f.approve(Change{Kind: ChangeSchema, Schema: Schema{Source: SourceLog, Fields: []FieldSpec{{Key: "retries", Class: ClassInteger}}}})
	expect(t, "purged", second(f.r.Preview("s1", logs.ID, ann)), ErrNotFound)
	f.must(second(f.r.Preview("s2", versions.ID, ann)))
	if f.r.bytes != before-int64(len(logs.Bytes)) || f.r.sites["s1"].bytes != 0 {
		t.Error("bytes not released")
	}
	b, err := f.r.Create("s1", sample(), ann)
	f.must(err)
	if b.Summary.Kept != 2 || fields(b.Items[0]) != "log{retries=integer:3}" {
		t.Errorf("new schema not applied %+v", b.Items)
	}
}

func TestExpiryAndDiscard(t *testing.T) {
	f := standard(t)
	b := f.bundle("s1")
	f.advance(time.Hour - time.Nanosecond)
	f.must(second(f.r.Preview("s1", b.ID, ann)))
	f.advance(time.Nanosecond)
	reason(t, "expired", second(f.r.Preview("s1", b.ID, ann)), ReasonExpired)
	expect(t, "purged", second(f.r.Preview("s1", b.ID, ann)), ErrNotFound)
	if f.r.bytes != 0 {
		t.Error("expired bytes kept")
	}
	c := f.bundle("s1")
	expect(t, "discard form", f.r.Discard("s1", "", ann), ErrInvalid)
	expect(t, "discard site", f.r.Discard("s9", c.ID, ann), ErrNotFound)
	reason(t, "discard assistant", f.r.Discard("s1", c.ID, bot), ReasonAINotAllowed)
	f.must(f.r.Discard("s1", c.ID, ann))
	expect(t, "discarded", f.r.Discard("s1", c.ID, ann), ErrNotFound)
	d := f.bundle("s1")
	e := f.bundle("s1")
	f.advance(time.Hour)
	expect(t, "purge others", second(f.r.Preview("s1", e.ID+"x", ann)), ErrNotFound)
	if len(f.r.sites["s1"].bundles) != 0 || d.ID == e.ID {
		t.Error("expired siblings kept")
	}
}

func TestBundleBounds(t *testing.T) {
	f := standard(t)
	for index := 0; index < limits.Bundles; index++ {
		f.bundle("s1")
	}
	expect(t, "count", second(f.r.Create("s1", sample(), ann)), ErrLimit)
	size := int64(len(f.bundle("s2").Bytes))
	l := limits
	l.SiteBytes, l.Bytes = size*2, size*3
	g := standardWith(t, l)
	g.bundle("s1")
	g.bundle("s1")
	expect(t, "site bytes", second(g.r.Create("s1", sample(), ann)), ErrLimit)
	g.bundle("s2")
	expect(t, "register bytes", second(g.r.Create("s2", sample(), ann)), ErrLimit)
	g.advance(time.Hour)
	g.must(second(g.r.Create("s2", sample(), ann)))
	big := []Item{}
	for index := 0; index < maxItems; index++ {
		item := Item{Source: SourceLog}
		for _, key := range []string{"client", "contact", "node", "operator", "subject"} {
			item.Fields = append(item.Fields, Field{Key: key, Text: strings.Repeat(string(rune('a'+index%26)), 200) + itoa(index%10)})
		}
		big = append(big, item)
	}
	h := standardWith(t, Limits{Sites: 3, Destinations: 2, Bundles: 3, SiteBytes: maxSiteBytes, Bytes: maxSiteBytes, BundleLife: time.Hour, Proposals: 8})
	h.must(second(h.r.Create("s1", big, ann)))
}

func standardWith(t *testing.T, l Limits) world {
	f := setupWith(t, l)
	f.approve(Change{Kind: ChangeSite, Site: Site{ID: "s1", Tenant: "t1"}})
	f.approve(Change{Kind: ChangeSite, Site: Site{ID: "s2", Tenant: "t2"}})
	f.approve(Change{Kind: ChangeSchema, Schema: logSchema()})
	f.approve(Change{Kind: ChangeSchema, Schema: Schema{Source: SourceVersion, Fields: []FieldSpec{{Key: "product", Class: ClassVersion}}}})
	return f
}

func TestOutputBound(t *testing.T) {
	f := standardWith(t, Limits{Sites: 3, Destinations: 2, Bundles: 3, SiteBytes: maxSiteBytes, Bytes: maxSiteBytes, BundleLife: time.Hour, Proposals: 8})
	value := "v" + strings.Repeat("x", maxEnumBytes-1)
	wide := Schema{Source: SourceMetric}
	for _, name := range names(maxSpecs) {
		wide.Fields = append(wide.Fields, FieldSpec{Key: name + strings.Repeat("q", maxKeyBytes-3), Class: ClassEnum, Values: []string{value}})
	}
	f.approve(Change{Kind: ChangeSchema, Schema: wide})
	items := []Item{}
	for index := 0; index < maxItems; index++ {
		item := Item{Source: SourceMetric}
		for _, spec := range wide.Fields {
			item.Fields = append(item.Fields, Field{Key: spec.Key, Text: value})
		}
		items = append(items, item)
	}
	expect(t, "output", second(f.r.Create("s1", items, ann)), ErrLimit)
}

func TestApprovalRecheck(t *testing.T) {
	f := standard(t)
	p, err := f.r.Propose(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1", Destination: "vendor"}}, ann)
	f.must(err)
	f.must(f.r.DisableDestination("vendor", cat))
	reason(t, "recheck", second(f.r.Approve(p.Number, bob)), ReasonDestinationDisabled)
}

func TestRefusalError(t *testing.T) {
	err := refuse(ReasonEmpty)
	if err.Error() != ErrRejected.Error()+": empty" {
		t.Error(err)
	}
	expect(t, "unwrap", err, ErrRejected)
}
