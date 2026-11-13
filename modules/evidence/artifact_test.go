package evidence

import (
	"reflect"
	"testing"
	"time"
)

func TestConstruction(t *testing.T) {
	clock := &testClock{at: start}
	changes := []func(*Limits){
		func(l *Limits) { l.Artifacts = 0 }, func(l *Limits) { l.Bundles = 0 }, func(l *Limits) { l.Holds = 0 }, func(l *Limits) { l.Gaps = 0 }, func(l *Limits) { l.Instances = 0 },
		func(l *Limits) { l.Items = 0 }, func(l *Limits) { l.HoldBundles = 0 }, func(l *Limits) { l.Restricted = 0 }, func(l *Limits) { l.Page = 0 }, func(l *Limits) { l.Listed = 0 },
	}
	for index, change := range changes {
		limits := testLimits()
		change(&limits)
		if _, err := NewRegistry(limits, clock.read); err != ErrInvalid {
			t.Fatalf("limit %d: %v", index, err)
		}
	}
	if _, err := NewRegistry(testLimits(), nil); err != ErrInvalid {
		t.Fatal(err)
	}
	var none *Registry
	for _, r := range []*Registry{none, {}} {
		checks := []error{
			second(r.Register(art("a", Policy, product("a")))), second(r.AddCustody("ws1", "a", product("b"))), second(r.ReportLoss("ws1", "a", Product, "store:a", Expired)),
			second(r.DeleteArtifact("ws1", "a", DeletedBy)), r.Restrict(Engine, "v1", "cve"), second(r.Artifact("ws1", "a")), second(r.Confirm(bundleSpec("b", []Item{{Policy, "a"}}))),
			second(r.Extend("ws1", "b", start)), r.Purge("ws1", "b", PurgedBy), second(r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h", Reason: "case", Bundles: []string{"b"}})),
			r.ReleaseHold("ws1", "h", MatterClosed), second(r.RecordGap(GapSpec{WorkspaceID: "ws1", ID: "g", Customer: "c", Period: "p", Role: Policy, Reason: CaptureFailed, Owner: "o", Path: "p"})),
			r.CloseGap("ws1", "g", "a"), second(r.Gaps("ws1", "c", "p")), second(r.Inspect("ws1", "b")), second(r.Exposed("ws1")), second(r.Download("ws1", "b", Policy, "a")),
			second(r.Items("ws1", "b", 0, 1)), second(r.Bundle("ws1", "b")),
		}
		for index, err := range checks {
			if err != ErrInvalid {
				t.Fatalf("operation %d on a registry that was not built: %v", index, err)
			}
		}
	}
}

func second[T any](_ T, err error) error { return err }

func TestRulesForStrings(t *testing.T) {
	long := ""
	for len(long) < 128 {
		long += "a"
	}
	for text, want := range map[string]bool{"a": true, "A.z_0:9-": true, long: true, long + "a": false, "": false, "a b": false, "a/b": false, "https://x": false, "a?sig=1": false, "a@b": false, "a[b": false, "a`b": false, "a{b": false} {
		if validID(text) != want {
			t.Errorf("identifier %q", text)
		}
	}
	for text, want := range map[string]bool{"0123abcd": true, "0123abc": false, long: true, long + "a": false, "0123abcg": false, "0123ABCD": false, "0123abc/": false, "0123abc:": false, "0123abc`": false} {
		if validDigest(text) != want {
			t.Errorf("digest %q", text)
		}
	}
}

func TestRegisterRefusalsRepeatsAndConflicts(t *testing.T) {
	later := start.Add(30 * day)
	cases := []struct {
		name   string
		change func(*ArtifactSpec)
	}{
		{"workspace", func(s *ArtifactSpec) { s.WorkspaceID = "" }},
		{"id", func(s *ArtifactSpec) { s.ID = "a b" }},
		{"kind", func(s *ArtifactSpec) { s.Kind = "invoice" }},
		{"version", func(s *ArtifactSpec) { s.Version = "" }},
		{"digest", func(s *ArtifactSpec) { s.Digest = "XYZ" }},
		{"size zero", func(s *ArtifactSpec) { s.Size = 0 }},
		{"size negative", func(s *ArtifactSpec) { s.Size = -1 }},
		{"no custody", func(s *ArtifactSpec) { s.Custody = Custody{} }},
		{"custody kind", func(s *ArtifactSpec) { s.Custody.Kind = "tape" }},
		{"a location that is a URL", func(s *ArtifactSpec) { s.Custody.Location = "https://bucket/x?sig=abc" }},
		{"product without a receipt", func(s *ArtifactSpec) { s.Custody.Receipt = "" }},
		{"product with a verification", func(s *ArtifactSpec) { s.Custody.VerifiedUntil = later }},
		{"product with evidence", func(s *ArtifactSpec) { s.Custody.Evidence = "check:1" }},
		{"remote with a receipt", func(s *ArtifactSpec) { s.Custody = remote("a1", later); s.Custody.Receipt = "r" }},
		{"remote verified without evidence", func(s *ArtifactSpec) { s.Custody = remote("a1", later); s.Custody.Evidence = "" }},
		{"remote with evidence and no time", func(s *ArtifactSpec) { s.Custody = remote("a1", time.Time{}); s.Custody.Evidence = "check:1" }},
		{"remote with malformed evidence", func(s *ArtifactSpec) { s.Custody = remote("a1", later); s.Custody.Evidence = "see mail" }},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		spec := art("a1", Policy, product("a1"))
		c.change(&spec)
		before := h.r.dump()
		_, err := h.r.Register(spec)
		wantErr(t, err, ErrInvalid, c.name)
		h.unchanged(before, c.name)
	}
	h := newHarness(t, func(l *Limits) { l.Artifacts = 3 })
	for index, kind := range kinds {
		other := newHarness(t, nil)
		spec := art("a1", kind, remote("a1", time.Time{}))
		spec.Size = int64(index + 1)
		other.register(spec)
	}
	spec := art("a1", Policy, remote("a1", later))
	h.register(spec)
	h.clock.advance(time.Hour)
	before := h.r.dump()
	h.register(spec)
	for name, change := range map[string]func(*ArtifactSpec){
		"kind":       func(s *ArtifactSpec) { s.Kind = Mapping },
		"version":    func(s *ArtifactSpec) { s.Version = "v2" },
		"digest":     func(s *ArtifactSpec) { s.Digest = digestOf("other") },
		"size":       func(s *ArtifactSpec) { s.Size = 11 },
		"custody":    func(s *ArtifactSpec) { s.Custody = product("a1") },
		"location":   func(s *ArtifactSpec) { s.Custody.Location = "bucket:other" },
		"until":      func(s *ArtifactSpec) { s.Custody.VerifiedUntil = later.Add(time.Second) },
		"evidence":   func(s *ArtifactSpec) { s.Custody.Evidence = "check:other" },
		"unverified": func(s *ArtifactSpec) { s.Custody = remote("a1", time.Time{}) },
	} {
		other := spec
		change(&other)
		_, err := h.r.Register(other)
		wantErr(t, err, ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	if until, err := h.r.AddCustody("ws1", "a1", remote("a1", later.Add(day))); err != nil || !until.Equal(later.Add(day)) {
		t.Fatal(until, err)
	}
	if state, err := h.r.Register(spec); err != nil || state != Registered {
		t.Fatalf("a repeat is compared with what was registered: %v %v", state, err)
	}
	elsewhere := spec
	elsewhere.WorkspaceID, elsewhere.Kind = "ws2", Mapping
	h.register(elsewhere)
	h.register(art("a2", Policy, product("a2")))
	h.register(art("a3", Policy, product("a3")))
	before = h.r.dump()
	_, err := h.r.Register(art("a4", Policy, product("a4")))
	wantErr(t, err, ErrLimit, "artifact limit")
	h.register(art("a3", Policy, product("a3")))
	h.unchanged(before, "limit")
	if _, err := h.r.DeleteArtifact("ws1", "a3", WrongRegistration); err != nil {
		t.Fatal(err)
	}
	if state, err := h.r.Register(art("a3", Policy, product("a3"))); err != nil || state != Deleted {
		t.Fatal(state, err)
	}
	_, err = h.r.Register(art("a4", Policy, product("a4")))
	wantErr(t, err, ErrLimit, "deleted artifacts still count")
	view, err := h.r.Artifact("ws1", "a1")
	if err != nil || !view.Registered.Equal(start) || !view.Changed.Equal(start.Add(time.Hour)) || len(view.Instances) != 1 || !view.Instances[0].Custody.VerifiedUntil.Equal(later.Add(day)) || view.Spec.Custody.VerifiedUntil != later {
		t.Fatalf("%+v %v", view, err)
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws3", "a1", ErrNotFound}, {"ws1", "zz", ErrNotFound}, {"", "a1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.Artifact(c.ws, c.id)
		wantErr(t, err, c.want, "artifact view")
	}
}

func TestTimesAreKeptInUTC(t *testing.T) {
	h := newHarness(t, nil)
	h.clock.at = start.In(time.FixedZone("kst", 9*3600))
	h.register(art("a1", Policy, remote("a1", start.Add(day).In(time.FixedZone("kst", 9*3600)))))
	view, _ := h.r.Artifact("ws1", "a1")
	if view.Registered.Location() != time.UTC || !view.Registered.Equal(start) || view.Instances[0].Custody.VerifiedUntil.Location() != time.UTC || view.Changed.Location() != time.UTC {
		t.Fatalf("%+v", view)
	}
}

func TestProductLocationBelongsToOneArtifact(t *testing.T) {
	h := newHarness(t, nil)
	h.register(art("a1", Engine, product("shared")))
	same := art("a2", Engine, product("shared"))
	_, err := h.r.Register(same)
	wantErr(t, err, ErrConflict, "the same location in one workspace")
	same.WorkspaceID = "ws2"
	_, err = h.r.Register(same)
	wantErr(t, err, ErrConflict, "the same location in another workspace")
	namesake := art("a1", Engine, product("shared"))
	namesake.WorkspaceID = "ws2"
	_, err = h.r.Register(namesake)
	wantErr(t, err, ErrConflict, "the same location under the same identifier in another workspace")
	h.register(art("a3", Engine, product("a3")))
	before := h.r.dump()
	_, err = h.r.AddCustody("ws1", "a3", product("shared"))
	wantErr(t, err, ErrConflict, "the same location added later")
	h.unchanged(before, "taken location")
	if _, err := h.r.ReportLoss("ws1", "a1", Product, "store:shared", Corrupted); err != nil {
		t.Fatal(err)
	}
	_, err = h.r.AddCustody("ws1", "a3", product("shared"))
	wantErr(t, err, ErrConflict, "a lost location stays taken")
	if _, err := h.r.DeleteArtifact("ws1", "a1", DeletedBy); err != nil {
		t.Fatal(err)
	}
	_, err = h.r.Register(same)
	wantErr(t, err, ErrConflict, "the location of a deleted artifact stays taken")
	far := Custody{Kind: Remote, Location: "store:shared"}
	if _, err := h.r.AddCustody("ws1", "a3", far); err != nil {
		t.Fatalf("a remote instance with the same text is another instance: %v", err)
	}
	other := art("a4", Engine, Custody{Kind: Remote, Location: "bucket:a9"})
	h.register(other)
	other.ID = "a5"
	h.register(other)
	h.clock.advance(3 * time.Hour)
	if _, err := h.r.AddCustody("ws1", "a4", product("a4")); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.AddCustody("ws1", "a4", product("a4")); err != nil {
		t.Fatal(err)
	}
	if view, _ := h.r.Artifact("ws1", "a4"); !view.Changed.Equal(start.Add(3*time.Hour)) || !view.Registered.Equal(start) {
		t.Fatalf("%+v", view)
	}
}

func TestCustodiesAreOnlyAdded(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Instances = 4 })
	t1, t2 := start.Add(10*day), start.Add(20*day)
	h.register(art("a1", Policy, remote("a1", t1)))
	h.clock.advance(time.Hour)
	steps := []struct {
		name    string
		custody Custody
		want    time.Time
		err     error
		writes  bool
	}{
		{"identical", remote("a1", t1), t1, nil, false},
		{"earlier", remote("a1", t1.Add(-time.Second)), t1, nil, false},
		{"unverified", remote("a1", time.Time{}), t1, nil, false},
		{"the same time with other evidence", Custody{Kind: Remote, Location: "bucket:a1", VerifiedUntil: t1, Evidence: "check:other"}, t1, nil, false},
		{"later", Custody{Kind: Remote, Location: "bucket:a1", VerifiedUntil: t2, Evidence: "check:second"}, t2, nil, true},
		{"the first again", remote("a1", t1), t2, nil, false},
		{"malformed", Custody{Kind: Remote}, time.Time{}, ErrInvalid, false},
		{"a product copy", product("a1"), time.Time{}, nil, true},
		{"the copy again", product("a1"), time.Time{}, nil, false},
		{"the copy with another receipt", Custody{Kind: Product, Location: "store:a1", Receipt: "receipt:other"}, time.Time{}, ErrConflict, false},
		{"another remote location", remote("mirror", time.Time{}), time.Time{}, nil, true},
		{"a second copy", product("a1-second"), time.Time{}, nil, true},
		{"a fifth instance", product("a1-third"), time.Time{}, ErrLimit, false},
	}
	for _, step := range steps {
		before := h.r.dump()
		until, err := h.r.AddCustody("ws1", "a1", step.custody)
		if err != step.err || !until.Equal(step.want) {
			t.Fatalf("%s: %v %v", step.name, until, err)
		}
		if (h.r.dump() != before) != step.writes {
			t.Fatalf("%s: writes %v", step.name, !step.writes)
		}
	}
	view, _ := h.r.Artifact("ws1", "a1")
	if len(view.Instances) != 4 || view.Instances[0].Custody.Evidence != "check:second" || !view.Changed.Equal(start.Add(time.Hour)) {
		t.Fatalf("%+v", view)
	}
	_, err := h.r.AddCustody("ws1", "zz", product("zz"))
	wantErr(t, err, ErrNotFound, "missing artifact")
	_, err = h.r.AddCustody("ws2", "a1", product("zz"))
	wantErr(t, err, ErrNotFound, "another workspace")
	_, err = h.r.AddCustody("", "a1", product("zz"))
	wantErr(t, err, ErrInvalid, "workspace")
	_, err = h.r.AddCustody("ws1", "", product("zz"))
	wantErr(t, err, ErrInvalid, "id")

	h.clock.advance(time.Hour)
	losses := []struct {
		name     string
		kind     CustodyKind
		location string
		reason   LossReason
		err      error
		writes   bool
	}{
		{"reason", Product, "store:a1", "stolen", ErrInvalid, false},
		{"kind", "tape", "store:a1", Expired, ErrInvalid, false},
		{"location form", Product, "", Expired, ErrInvalid, false},
		{"an absent instance", Product, "store:zz", Expired, ErrNotFound, false},
		{"the location under the other kind", Remote, "store:a1", Expired, ErrNotFound, false},
		{"a product verification withdrawn", Product, "store:a1", VerificationWithdrawn, ErrRejected, false},
		{"corrupted", Product, "store:a1", Corrupted, nil, true},
		{"corrupted again", Product, "store:a1", Corrupted, nil, false},
		{"another reason", Product, "store:a1", LostDeleted, ErrConflict, false},
		{"withdrawn", Remote, "bucket:a1", VerificationWithdrawn, nil, true},
		{"expired", Remote, "bucket:mirror", Expired, nil, true},
		{"revoked", Product, "store:a1-second", AccessRevoked, nil, true},
	}
	for _, loss := range losses {
		before := h.r.dump()
		_, err := h.r.ReportLoss("ws1", "a1", loss.kind, loss.location, loss.reason)
		if err != loss.err || (h.r.dump() != before) != loss.writes {
			t.Fatalf("%s: %v", loss.name, err)
		}
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "a1", ErrNotFound}, {"ws1", "zz", ErrNotFound}, {"", "a1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.ReportLoss(c.ws, c.id, Product, "store:a1", Corrupted)
		wantErr(t, err, c.want, "loss")
	}
	before := h.r.dump()
	for _, again := range []Custody{product("a1"), remote("a1", t2), remote("a1", t2.Add(day)), remote("mirror", time.Time{})} {
		_, err := h.r.AddCustody("ws1", "a1", again)
		wantErr(t, err, ErrRejected, "a lost instance does not come back")
	}
	_, err = h.r.AddCustody("ws1", "a1", product("a1-restored"))
	wantErr(t, err, ErrLimit, "lost instances count towards the limit")
	h.unchanged(before, "after losses")
	view, _ = h.r.Artifact("ws1", "a1")
	wantReasons := []LossReason{VerificationWithdrawn, Corrupted, Expired, AccessRevoked}
	for index, held := range view.Instances {
		if !held.Lost || held.Reason != wantReasons[index] {
			t.Fatalf("%+v", view.Instances)
		}
	}
	if !view.Changed.Equal(start.Add(2 * time.Hour)) {
		t.Fatal(view.Changed)
	}

	wide := newHarness(t, nil)
	wide.register(art("a1", Policy, product("a1")))
	if _, err := wide.r.ReportLoss("ws1", "a1", Product, "store:a1", Corrupted); err != nil {
		t.Fatal(err)
	}
	if _, err := wide.r.AddCustody("ws1", "a1", product("a1-restored")); err != nil {
		t.Fatalf("the same artifact is held again at a new location: %v", err)
	}
	items := wide.set("b1")
	items[4] = Item{Policy, "a1"}
	wide.confirm(bundleSpec("b1", items))
	if got := brief(wide.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
	if _, err := wide.r.DeleteArtifact("ws1", "b1-pol", DeletedBy); err != nil {
		t.Fatal(err)
	}
	before = wide.r.dump()
	_, err = wide.r.AddCustody("ws1", "b1-pol", product("again"))
	wantErr(t, err, ErrRejected, "custody on a deleted artifact")
	_, err = wide.r.AddCustody("ws1", "b1-pol", product("b1-pol"))
	wantErr(t, err, ErrRejected, "the same custody again on a deleted artifact")
	_, err = wide.r.ReportLoss("ws1", "b1-pol", Product, "store:b1-pol", Corrupted)
	wantErr(t, err, ErrRejected, "loss on a deleted artifact")
	_, err = wide.r.ReportLoss("ws1", "b1-pol", Remote, "bucket:unknown", Expired)
	wantErr(t, err, ErrRejected, "loss of an unknown instance of a deleted artifact")
	wide.unchanged(before, "a deleted artifact is final")
	if view, _ := wide.r.Artifact("ws1", "b1-pol"); view.State != Deleted || view.DeleteReason != DeletedBy || view.Instances[0].Lost {
		t.Fatalf("%+v", view)
	}
	if !reflect.DeepEqual(wide.inspect("b1").Broken, []BrokenItem(nil)) {
		t.Fatal("another artifact's deletion broke the bundle")
	}
}
