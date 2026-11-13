package evidence

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestHeldAtTheEdgesOfAVerification(t *testing.T) {
	h := newHarness(t, nil)
	retain := start.Add(100 * day)
	h.register(art("far", Document, remote("far", retain)))
	h.confirm(bundleSpec2("b1", retain, append(h.set("b1"), Item{Document, "far"})))
	h.clock.advance(100*day - time.Nanosecond)
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
	h.clock.advance(time.Nanosecond)
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=false" {
		t.Fatalf("a verification that ends exactly now still holds, and protection ends at the retention time: %s", got)
	}
	if out, err := h.r.Download("ws1", "b1", Document, "far"); err != nil || len(out.Instances) != 1 {
		t.Fatal(out, err)
	}
	h.clock.advance(time.Nanosecond)
	if got := brief(h.inspect("b1")); got != "broken[far:verification_ended:]/1 risk[]/0 operational protected=false" {
		t.Fatal(got)
	}
	_, err := h.r.Download("ws1", "b1", Document, "far")
	wantErr(t, err, ErrRejected, "a download through a verification that ended")
	h.clock.advance(-50 * day)
	if got := brief(h.inspect("b1")); !strings.HasPrefix(got, "broken") {
		t.Fatalf("the registry's time does not go back: %s", got)
	}
	later := newHarness(t, nil)
	later.register(art("far", Document, remote("far", start.Add(10*day))))
	later.clock.advance(20 * day)
	spec := bundleSpec("b1", append(later.set("b1"), Item{Document, "far"}))
	spec.RetainUntil = start.Add(5 * day)
	later.refused(spec, "retention_not_ahead not_held:document:far")
	unverified := newHarness(t, nil)
	unverified.register(art("far", Document, remote("far", start.Add(-day))))
	unverified.refused(bundleSpec("b1", append(unverified.set("b1"), Item{Document, "far"})), "not_held:document:far")
}

func bundleSpec2(id string, retain time.Time, items []Item) BundleSpec {
	spec := bundleSpec(id, items)
	spec.RetainUntil = retain
	return spec
}

func TestVerdictsExposureAndWhatALossReturns(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Listed = 2 })
	retain := start.Add(365 * day)
	shared := art("shared", SourceOriginal, remote("shared", retain.Add(day)))
	h.register(shared)
	if _, err := h.r.AddCustody("ws1", "shared", product("shared")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"b1", "b2", "b3", "b4"} {
		items := h.set(id)
		items[0] = Item{SourceOriginal, "shared"}
		h.confirm(bundleSpec(id, items))
	}
	if exposed, _ := h.r.Exposed("ws1"); exposed.Total != 0 || len(exposed.Items) != 0 {
		t.Fatalf("%+v", exposed)
	}
	if exposed, err := h.r.Exposed("ws9"); err != nil || exposed.Total != 0 {
		t.Fatal(exposed, err)
	}
	_, err := h.r.Exposed("")
	wantErr(t, err, ErrInvalid, "workspace")
	out, err := h.r.ReportLoss("ws1", "shared", Product, "store:shared", Corrupted)
	if err != nil || out.Total != 0 {
		t.Fatalf("still held until the retention time through the remote instance: %+v %v", out, err)
	}
	if _, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b3"}}); err != nil {
		t.Fatal(err)
	}
	if got := brief(h.inspect("b3")); got != "held[]/0 risk[shared]/1 operational protected=true" {
		t.Fatal(got)
	}
	if again, _ := h.r.ReportLoss("ws1", "shared", Product, "store:shared", Corrupted); !reflect.DeepEqual(again, Listed{Total: 1, Items: []string{"b3"}}) {
		t.Fatalf("%+v", again)
	}
	out, err = h.r.ReportLoss("ws1", "b2-pol", Product, "store:b2-pol", LostDeleted)
	if err != nil || !reflect.DeepEqual(out, Listed{Total: 1, Items: []string{"b2"}}) {
		t.Fatalf("%+v %v", out, err)
	}
	if got := brief(h.inspect("b2")); got != "broken[b2-pol:lost:deleted]/1 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
	out, err = h.r.ReportLoss("ws1", "shared", Remote, "bucket:shared", AccessRevoked)
	if err != nil || !reflect.DeepEqual(out, Listed{Total: 4, Items: []string{"b1", "b2"}}) {
		t.Fatalf("an emergency withdrawal is not refused and says what it broke: %+v %v", out, err)
	}
	if got := brief(h.inspect("b1")); got != "broken[shared:lost:access_revoked]/1 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
	if got := brief(h.inspect("b2")); got != "broken[shared:lost:access_revoked b2-pol:lost:deleted]/2 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
	if exposed, _ := h.r.Exposed("ws1"); !reflect.DeepEqual(exposed, Listed{Total: 4, Items: []string{"b1", "b2"}}) {
		t.Fatalf("%+v", exposed)
	}
	h.r.ReportLoss("ws1", "b2-res", Product, "store:b2-res", Expired)
	inspected := h.inspect("b2")
	if inspected.BrokenTotal != 3 || len(inspected.Broken) != 2 || inspected.Broken[1].Artifact != "b2-pol" {
		t.Fatalf("%+v", inspected)
	}
	if _, err := h.r.AddCustody("ws1", "shared", product("shared-restored")); err != nil {
		t.Fatal(err)
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatalf("held again through a new instance: %s", got)
	}
	if exposed, _ := h.r.Exposed("ws1"); !reflect.DeepEqual(exposed, Listed{Total: 1, Items: []string{"b2"}}) {
		t.Fatalf("%+v", exposed)
	}
	h.register(art("mixed", Document, remote("mixed", start.Add(day))))
	h.r.AddCustody("ws1", "mixed", remote("mixed-mirror", retain))
	h.confirm(bundleSpec("b5", append(h.set("b5"), Item{Document, "mixed"})))
	h.r.ReportLoss("ws1", "mixed", Remote, "bucket:mixed-mirror", Expired)
	if got := brief(h.inspect("b5")); got != "held[]/0 risk[mixed]/1 operational protected=true" {
		t.Fatal(got)
	}
	h.clock.advance(2 * day)
	if got := brief(h.inspect("b5")); got != "broken[mixed:verification_ended:]/1 risk[]/0 operational protected=true" {
		t.Fatalf("lost only when every instance is lost: %s", got)
	}
	h.r.ReportLoss("ws1", "mixed", Remote, "bucket:mixed", VerificationWithdrawn)
	if got := brief(h.inspect("b5")); got != "broken[mixed:lost:verification_withdrawn]/1 risk[]/0 operational protected=true" {
		t.Fatalf("the reason of the instance lost last: %s", got)
	}
	h.register(art("pair", Document, product("pair")))
	h.r.AddCustody("ws1", "pair", product("pair-b"))
	h.confirm(bundleSpec("b6", append(h.set("b6"), Item{Document, "pair"})))
	h.r.ReportLoss("ws1", "pair", Product, "store:pair", Corrupted)
	h.r.ReportLoss("ws1", "pair", Product, "store:pair-b", AccessRevoked)
	if got := brief(h.inspect("b6")); got != "broken[pair:lost:access_revoked]/1 risk[]/0 operational protected=true" {
		t.Fatalf("the reason of the instance lost last, not of the first: %s", got)
	}
	h.register(art("r1", Document, remote("r1", retain)))
	h.register(art("r2", Document, remote("r2", retain)))
	h.register(art("r3", Document, remote("r3", retain)))
	h.confirm(bundleSpec("b7", append(h.set("b7"), Item{Document, "r1"}, Item{Document, "r2"}, Item{Document, "r3"})))
	h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h7", Reason: "case:7", Bundles: []string{"b7"}})
	if out := h.inspect("b7"); out.AtRiskTotal != 3 || len(out.AtRisk) != 2 || out.AtRisk[1].Artifact != "r2" {
		t.Fatalf("items at risk are listed up to the bound: %+v", out)
	}
	h.r.ReleaseHold("ws1", "h7", MatterClosed)
	h.clock.advance(400 * day)
	if got := brief(h.inspect("b5")); got != "broken[mixed:lost:verification_withdrawn]/1 risk[]/0 operational protected=false" {
		t.Fatal(got)
	}
	if exposed, _ := h.r.Exposed("ws1"); !reflect.DeepEqual(exposed, Listed{Total: 4, Items: []string{"b2", "b5"}}) {
		t.Fatalf("a broken bundle is exposed whether or not it is protected: %+v", exposed)
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "b1", ErrNotFound}, {"ws1", "b9", ErrNotFound}, {"", "b1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.Inspect(c.ws, c.id)
		wantErr(t, err, c.want, "inspect")
	}
}

func TestDownloadsStayTheSame(t *testing.T) {
	h := newHarness(t, nil)
	retain := start.Add(365 * day)
	h.register(art("doc", Document, remote("doc", retain)))
	h.r.AddCustody("ws1", "doc", remote("doc-unverified", time.Time{}))
	h.r.AddCustody("ws1", "doc", product("doc"))
	h.r.AddCustody("ws1", "doc", product("doc-b"))
	h.r.AddCustody("ws1", "doc", remote("alpha", retain.Add(day)))
	h.confirm(bundleSpec("b1", append(h.set("b1"), Item{Document, "doc"})))
	first, err := h.r.Download("ws1", "b1", Document, "doc")
	want := Download{ID: "doc", Version: "v1", Digest: digestOf("bytes", "doc"), Size: 10, Instances: []Custody{product("doc"), product("doc-b"), remote("alpha", retain.Add(day)), remote("doc", retain)}}
	if err != nil || !reflect.DeepEqual(first, want) {
		t.Fatalf("%+v %v", first, err)
	}
	first.Instances[0].Location = "changed"
	h.r.ReportLoss("ws1", "doc", Remote, "bucket:doc", Expired)
	h.r.ReportLoss("ws1", "doc", Remote, "bucket:alpha", LostDeleted)
	h.r.ReportLoss("ws1", "doc", Product, "store:doc", AccessRevoked)
	h.r.ReportLoss("ws1", "b1-eng", Product, "store:b1-eng", Corrupted)
	if err := h.r.Restrict(Engine, "v1", "cve:1"); err != nil {
		t.Fatal(err)
	}
	h.clock.advance(300 * day)
	second, err := h.r.Download("ws1", "b1", Document, "doc")
	if err != nil || second.ID != want.ID || second.Version != want.Version || second.Digest != want.Digest || second.Size != want.Size || !reflect.DeepEqual(second.Instances, []Custody{product("doc-b")}) {
		t.Fatalf("a document is read whatever became of the engine: %+v %v", second, err)
	}
	if out, err := h.r.Download("ws1", "b1", Result, "b1-res"); err != nil || out.Digest != digestOf("bytes", "b1-res") {
		t.Fatal(out, err)
	}
	_, err = h.r.Download("ws1", "b1", Engine, "b1-eng")
	wantErr(t, err, ErrRejected, "an item that is not held")
	for _, c := range []struct {
		ws, id string
		role   Kind
		art    string
		want   error
	}{
		{"ws1", "b1", Policy, "doc", ErrNotFound}, {"ws1", "b1", Document, "b1-res", ErrNotFound}, {"ws1", "b1", Document, "zz", ErrNotFound}, {"ws2", "b1", Document, "doc", ErrNotFound},
		{"ws1", "b9", Document, "doc", ErrNotFound}, {"ws1", "b1", "invoice", "doc", ErrInvalid}, {"ws1", "b1", Document, "", ErrInvalid}, {"", "b1", Document, "doc", ErrInvalid}, {"ws1", "", Document, "doc", ErrInvalid},
	} {
		_, err := h.r.Download(c.ws, c.id, c.role, c.art)
		wantErr(t, err, c.want, fmt.Sprint(c))
	}
	h.r.ReportLoss("ws1", "doc", Product, "store:doc-b", Corrupted)
	_, err = h.r.Download("ws1", "b1", Document, "doc")
	wantErr(t, err, ErrRejected, "nothing is offered in place of what is not held, an unverified remote copy included")
}

func TestRestrictedVersionsBelongToTheInstallation(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Restricted = 2 })
	items := append(h.set("b1"), h.set("b1", "mod")...)
	spec := bundleSpec("b1", items)
	spec.Modules = true
	h.confirm(spec)
	other := art("eng", Engine, product("eng-ws2"))
	other.WorkspaceID = "ws2"
	h.register(other)
	for name, c := range map[string]struct {
		kind            Kind
		version, reason string
	}{"a policy": {Policy, "v1", "cve:1"}, "version": {Engine, "", "cve:1"}, "reason": {Engine, "v1", "a long story"}, "kind": {"", "v1", "cve:1"}} {
		wantErr(t, h.r.Restrict(c.kind, c.version, c.reason), ErrInvalid, name)
	}
	if err := h.r.Restrict(Engine, "v2", "cve:0"); err != nil {
		t.Fatal(err)
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
	if err := h.r.Restrict(Module, "v1", "cve:1"); err != nil {
		t.Fatal(err)
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 isolated_only protected=true" {
		t.Fatal(got)
	}
	before := h.r.dump()
	if err := h.r.Restrict(Module, "v1", "cve:1"); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.Restrict(Module, "v1", "cve:2"), ErrConflict, "another reason")
	wantErr(t, h.r.Restrict(Engine, "v1", "cve:1"), ErrLimit, "restricted versions")
	h.unchanged(before, "restriction repeats")
	if view, _ := h.r.Artifact("ws1", "b1-mod"); !view.Restricted {
		t.Fatal("module not restricted")
	}
	if view, _ := h.r.Artifact("ws1", "b1-eng"); view.Restricted {
		t.Fatal("an engine of the same version text as a module is another component")
	}
	wide := newHarness(t, nil)
	if err := wide.r.Restrict(Engine, "v1", "cve:1"); err != nil {
		t.Fatal(err)
	}
	wide.confirm(bundleSpec("b1", wide.set("b1")))
	if got := brief(wide.inspect("b1")); got != "held[]/0 risk[]/0 isolated_only protected=true" {
		t.Fatalf("a version restricted before registration, and a bundle confirmed with it: %s", got)
	}
	far := art("eng", Engine, product("eng-far"))
	far.WorkspaceID = "ws2"
	wide.register(far)
	if view, _ := wide.r.Artifact("ws2", "eng"); !view.Restricted {
		t.Fatal("a restriction holds in every workspace")
	}
	if out, err := wide.r.Download("ws1", "b1", Result, "b1-res"); err != nil || out.ID != "b1-res" {
		t.Fatal(out, err)
	}
	mislabelled := art("pol-as-engine", Policy, product("pae"))
	wide.register(mislabelled)
	if view, _ := wide.r.Artifact("ws1", "pol-as-engine"); view.Restricted {
		t.Fatal("a policy of the same version text is not restricted")
	}
}

func TestGaps(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Gaps = 4; l.Listed = 2 })
	base := GapSpec{WorkspaceID: "ws1", ID: "g1", Customer: "cust-1", Period: "2026-10", Role: SourceOriginal, Reason: EmergencyRevocation, Owner: "team:ops", Path: "ticket:9"}
	for name, change := range map[string]func(*GapSpec){
		"workspace": func(g *GapSpec) { g.WorkspaceID = "" }, "id": func(g *GapSpec) { g.ID = "" }, "customer": func(g *GapSpec) { g.Customer = "" }, "period": func(g *GapSpec) { g.Period = "" },
		"role": func(g *GapSpec) { g.Role = "invoice" }, "reason": func(g *GapSpec) { g.Reason = "lost" }, "owner": func(g *GapSpec) { g.Owner = "ask Kim" }, "path": func(g *GapSpec) { g.Path = "" },
	} {
		spec := base
		change(&spec)
		_, err := h.r.RecordGap(spec)
		wantErr(t, err, ErrInvalid, name)
	}
	for index, reason := range []GapReason{EmergencyRevocation, CaptureFailed, ReceivedAfterEnd} {
		spec := base
		spec.ID, spec.Reason = fmt.Sprintf("g%d", index+1), reason
		if state, err := h.r.RecordGap(spec); err != nil || state != GapOpen {
			t.Fatal(state, err)
		}
	}
	before := h.r.dump()
	if state, err := h.r.RecordGap(base); err != nil || state != GapOpen {
		t.Fatal(state, err)
	}
	for name, change := range map[string]func(*GapSpec){
		"customer": func(g *GapSpec) { g.Customer = "cust-2" }, "period": func(g *GapSpec) { g.Period = "2026-11" }, "role": func(g *GapSpec) { g.Role = Policy },
		"reason": func(g *GapSpec) { g.Reason = CaptureFailed }, "owner": func(g *GapSpec) { g.Owner = "team:b" }, "path": func(g *GapSpec) { g.Path = "ticket:1" },
	} {
		spec := base
		change(&spec)
		_, err := h.r.RecordGap(spec)
		wantErr(t, err, ErrConflict, name)
	}
	h.unchanged(before, "gap repeats")
	if open, err := h.r.Gaps("ws1", "cust-1", "2026-10"); err != nil || !reflect.DeepEqual(open, Listed{Total: 3, Items: []string{"g1", "g2"}}) {
		t.Fatal(open, err)
	}
	if open, _ := h.r.Gaps("ws1", "cust-1", "2026-11"); open.Total != 0 {
		t.Fatal(open)
	}
	if open, err := h.r.Gaps("ws2", "cust-1", "2026-10"); err != nil || open.Total != 0 {
		t.Fatal(open, err)
	}
	for _, c := range [][3]string{{"", "c", "p"}, {"ws1", "", "p"}, {"ws1", "c", ""}} {
		_, err := h.r.Gaps(c[0], c[1], c[2])
		wantErr(t, err, ErrInvalid, "gaps")
	}
	items := h.set("b1")
	out := h.confirm(bundleSpec("b1", items))
	if !reflect.DeepEqual(out.Gaps, Listed{Total: 3, Items: []string{"g1", "g2"}}) {
		t.Fatalf("%+v", out)
	}
	h.register(art("late", SourceOriginal, product("late")))
	h.register(art("unheld", SourceOriginal, remote("unheld", time.Time{})))
	h.register(art("erased", SourceOriginal, product("erased")))
	h.r.DeleteArtifact("ws1", "erased", WrongRegistration)
	for _, c := range []struct {
		gap, artifact string
		want          error
	}{{"g1", "erased", ErrRejected}, {"g9", "late", ErrNotFound}, {"g1", "nowhere", ErrNotFound}, {"g1", "b1-pol", ErrRejected}, {"g1", "unheld", ErrRejected}, {"", "late", ErrInvalid}, {"g1", "", ErrInvalid}} {
		before := h.r.dump()
		wantErr(t, h.r.CloseGap("ws1", c.gap, c.artifact), c.want, c.gap+" "+c.artifact)
		h.unchanged(before, "a refused closing")
	}
	wantErr(t, h.r.CloseGap("ws2", "g1", "late"), ErrNotFound, "another workspace")
	wantErr(t, h.r.CloseGap("", "g1", "late"), ErrInvalid, "workspace")
	if err := h.r.CloseGap("ws1", "g2", "late"); err != nil {
		t.Fatal(err)
	}
	before = h.r.dump()
	if err := h.r.CloseGap("ws1", "g2", "late"); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.CloseGap("ws1", "g2", "b1-src"), ErrConflict, "closed with another artifact")
	second := base
	second.ID, second.Reason = "g2", CaptureFailed
	if state, err := h.r.RecordGap(second); err != nil || state != GapClosed {
		t.Fatal(state, err)
	}
	h.unchanged(before, "after closing")
	fourth := base
	fourth.ID = "g0"
	if _, err := h.r.RecordGap(fourth); err != nil {
		t.Fatal(err)
	}
	fifth := base
	fifth.ID = "g5"
	_, err := h.r.RecordGap(fifth)
	wantErr(t, err, ErrLimit, "gap limit")
	inspected := h.inspect("b1")
	if inspected.BoundTotal != 3 || !reflect.DeepEqual(inspected.Bound, []GapView{{ID: "g1", State: GapOpen}, {ID: "g2", State: GapClosed, Artifact: "late"}}) || !reflect.DeepEqual(inspected.Open, Listed{Total: 3, Items: []string{"g0", "g1"}}) {
		t.Fatalf("%+v", inspected)
	}
	if view, _ := h.r.Bundle("ws1", "b1"); !reflect.DeepEqual(view.Gaps, []string{"g1", "g2", "g3"}) {
		t.Fatalf("what was bound at confirmation stays: %+v", view.Gaps)
	}
	later := h.confirm(bundleSpec("b2", h.set("b2")))
	if !reflect.DeepEqual(later.Gaps, Listed{Total: 3, Items: []string{"g0", "g1"}}) {
		t.Fatalf("%+v", later)
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatalf("a gap changes no verdict: %s", got)
	}
}

func TestTenantBoundaryAndRefusalsThatRepeatNothing(t *testing.T) {
	h := newHarness(t, nil)
	h.confirm(bundleSpec("b1", h.set("b1")))
	for _, name := range []string{"src", "man", "map", "att", "pol", "eng", "res"} {
		spec := art("b1-"+name, roles[name], Custody{Kind: Remote, Location: "bucket:" + name})
		spec.WorkspaceID, spec.Digest = "ws2", digestOf("other", name)
		h.register(spec)
	}
	far := bundleSpec("b1", h.set("b1"))
	far.WorkspaceID = "ws2"
	out, err := h.r.Confirm(far)
	if err != ErrRejected || len(out.Reasons) != 7 || out.Reasons[0].Code != NotHeld {
		t.Fatalf("the same identifiers in another workspace are other artifacts: %+v %v", out, err)
	}
	if view, _ := h.r.Artifact("ws2", "b1-pol"); view.Spec.Digest != digestOf("other", "pol") || view.Bundles != 0 {
		t.Fatalf("%+v", view)
	}
	if view, _ := h.r.Artifact("ws1", "b1-pol"); view.Spec.Digest != digestOf("bytes", "b1-pol") || view.Bundles != 1 {
		t.Fatalf("%+v", view)
	}
	secret := "customer-text"
	_, invalid := h.r.Register(art(secret+" x", Policy, product("p")))
	_, conflict := h.r.Register(art("b1-pol", Policy, product(secret)))
	_, missing := h.r.AddCustody("ws1", secret, product(secret))
	_, absent := h.r.Download("ws1", "b1", Policy, secret)
	_, named := h.r.DeleteArtifact("ws1", "b1-pol", DeletedBy)
	refusal, rejected := h.r.Confirm(bundleSpec(secret, []Item{{Policy, secret}}))
	for index, pair := range []struct{ got, want error }{{invalid, ErrInvalid}, {conflict, ErrConflict}, {missing, ErrNotFound}, {absent, ErrNotFound}, {named, ErrRejected}, {rejected, ErrRejected}} {
		if pair.got != pair.want {
			t.Fatalf("%d: %v", index, pair.got)
		}
	}
	if refusal.Reasons[0].Artifact != secret {
		t.Fatal("the reasons of a refused confirmation name the items of the request")
	}
}
