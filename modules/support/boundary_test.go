package support

import (
	"testing"
	"time"
)

func TestBoundaries(t *testing.T) {
	if normal("Z") != "z" || normal("A") != "a" || !validHost("a9") {
		t.Error("letters or digits")
	}
	for _, values := range [][]string{{"host-0"}, {"user-09"}} {
		if (Schema{Source: SourceLog, Fields: []FieldSpec{{Key: "a", Class: ClassEnum, Values: values}}}).valid() {
			t.Errorf("%v accepted", values)
		}
	}
	if !(Schema{Source: SourceLog, Fields: []FieldSpec{{Key: "a", Class: ClassEnum, Values: []string{"a"}}}}).valid() {
		t.Error("one-letter enum refused")
	}
	f := standard(t)
	b, err := f.r.Create("s1", []Item{{Source: SourceLog, Fields: []Field{{Key: "subject", Int: 10}}}, {Source: SourceLog, Fields: []Field{{Key: "subject", Text: "10"}}}}, ann)
	f.must(err)
	if b.Items[0].Fields[0].Value != "identifier-1" || b.Items[1].Fields[0].Value != "identifier-1" {
		t.Errorf("integer identifier %+v", b.Items)
	}
	for index := 0; index < 9; index++ {
		next := f.bundle("s2")
		f.must(f.r.Discard("s2", next.ID, ann))
	}
	if last := f.bundle("s2"); last.ID != "bundle-10" {
		t.Errorf("id %s", last.ID)
	}
	f.approve(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1", Destination: "vendor"}})
	if f.r.sites["s1"].bundles[b.ID].view.State != StateDraft {
		t.Error("reset state")
	}
	g := setupWith(t, Limits{Sites: 10, Destinations: 2, Bundles: 1, SiteBytes: 1, Bytes: 1, BundleLife: time.Hour, Proposals: 4})
	g.must(second(g.r.Propose(Change{Kind: ChangeSite, Site: Site{ID: "a", Tenant: "t"}}, ann)))
	expect(t, "one per proposer of four", second(g.r.Propose(Change{Kind: ChangeSite, Site: Site{ID: "b", Tenant: "t"}}, ann)), ErrLimit)
}

func TestByteBoundaries(t *testing.T) {
	probe := standard(t)
	size := int64(len(probe.bundle("s1").Bytes))
	l := limits
	l.SiteBytes, l.Bytes = size*2, size*3
	g := standardWith(t, l)
	g.bundle("s1")
	g.bundle("s1")
	g.bundle("s2")
	if g.r.bytes != l.Bytes || g.r.sites["s1"].bytes != l.SiteBytes {
		t.Errorf("accounting %d %d", g.r.bytes, g.r.sites["s1"].bytes)
	}
}

func TestRegisterSweep(t *testing.T) {
	probe := standard(t)
	size := int64(len(probe.bundle("s1").Bytes))
	l := limits
	l.SiteBytes, l.Bytes = size*2, size*2
	g := standardWith(t, l)
	g.bundle("s1")
	g.advance(30 * time.Minute)
	g.bundle("s2")
	g.advance(30 * time.Minute)
	g.must(second(g.r.Create("s2", sample(), ann)))
	if len(g.r.sites["s1"].bundles) != 0 || len(g.r.sites["s2"].bundles) != 2 {
		t.Error("sweep")
	}
}

func TestIsolationAndSwitch(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeDestination, Destination: Destination{ID: "other", Host: "other.example", Port: 443}})
	b := f.bundle("s1")
	expect(t, "foreign site", second(f.r.Preview("s2", b.ID, ann)), ErrNotFound)
	f.approve(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1", Destination: "vendor"}})
	f.approve(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1", Destination: "other"}})
	f.must(second(f.r.Preview("s1", b.ID, ann)))
	reason(t, "replaced", second(f.r.Upload("s1", b.ID, b.Digest, "vendor", ann)), ReasonUploadDisabled)
	f.must(second(f.r.Upload("s1", b.ID, b.Digest, "other", ann)))
}

func TestConcurrentUse(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeUpload, Upload: Upload{Site: "s1", Destination: "vendor"}})
	done := make(chan struct{})
	for worker := 0; worker < 4; worker++ {
		go func(worker int) {
			defer func() { done <- struct{}{} }()
			for step := 0; step < 50; step++ {
				site := []string{"s1", "s2"}[step%2]
				b, err := f.r.Create(site, sample(), ann)
				if err == nil {
					_, _ = f.r.Preview(site, b.ID, ann)
					_, _ = f.r.Export(site, b.ID, b.Digest, ann)
					_, _ = f.r.Upload(site, b.ID, b.Digest, "vendor", ann)
					_ = f.r.Discard(site, b.ID, ann)
				}
				if worker == 0 && step%10 == 0 {
					_ = f.r.DisableUpload("s1", ann)
				}
			}
		}(worker)
	}
	for worker := 0; worker < 4; worker++ {
		<-done
	}
	if f.r.bytes < 0 {
		t.Error("accounting")
	}
}

func TestHandleNumbering(t *testing.T) {
	f := setupWith(t, Limits{Sites: 20, Destinations: 1, Bundles: 1, SiteBytes: 1, Bytes: 1, BundleLife: time.Hour, Proposals: 8})
	for _, id := range names(10) {
		f.approve(Change{Kind: ChangeSite, Site: Site{ID: id, Tenant: "t"}})
	}
	if f.r.sites[names(10)[9]].handle != "site-10" {
		t.Errorf("handle %s", f.r.sites[names(10)[9]].handle)
	}
}
