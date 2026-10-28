package customeraccess_test

import (
	"context"
	"testing"
	"time"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

func exportService(t *testing.T, f *fixture, read ca.ReadFunc) *ca.Service {
	t.Helper()
	s, err := ca.NewServiceForAction(verify, f.store, read, ca.Limits{MaxRows: 10, MaxScopes: 2, MaxFields: 3}, ca.UsageExport)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestExportActionAndCheckpointBinding(t *testing.T) {
	f := newFixture(t)
	calls := 0
	reader := func(context.Context, ca.Selection) ([]ca.Row, error) { calls++; return []ca.Row{rowA()}, nil }
	s := exportService(t, f, reader)
	if _, err := s.Capture(f.ctx(), f.request); err != ca.ErrDenied {
		t.Fatal("read grant authorized export")
	}
	f.subject.Grants[0].Action = ca.UsageExport
	v, err := f.store.PutSubject(f.subject, 1)
	requireWrite(t, v, err)
	if _, err := f.service(t, reader).Query(f.ctx(), f.request); err != ca.ErrDenied {
		t.Fatal("export grant authorized read")
	}
	checkpoint, err := s.Capture(f.ctx(), f.request)
	if err != nil {
		t.Fatal(err)
	}
	for _, pin := range []*ca.Checkpoint{nil, {}} {
		if rows, err := s.QueryPinned(f.ctx(), f.request, pin); rows != nil || err != ca.ErrDenied {
			t.Fatal("forged checkpoint allowed")
		}
	}
	altered := f.request
	altered.Limit = 9
	if _, err := s.QueryPinned(f.ctx(), altered, checkpoint); err != ca.ErrDenied {
		t.Fatal("changed target allowed")
	}
	f.principal.ValidUntil = f.base.Add(2 * time.Hour)
	if rows, err := s.QueryPinned(f.ctx(), f.request, checkpoint); err != nil || len(rows) != 1 {
		t.Fatal("same epoch refreshed credential denied")
	}
	if calls != 1 {
		t.Fatalf("unexpected reader calls %d", calls)
	}
	other := newFixture(t)
	other.subject.Grants[0].Action = ca.UsageExport
	v, err = other.store.PutSubject(other.subject, 1)
	requireWrite(t, v, err)
	if _, err := exportService(t, other, reader).QueryPinned(other.ctx(), other.request, checkpoint); err != ca.ErrDenied {
		t.Fatal("another store accepted checkpoint")
	}
	if _, err := f.service(t, reader).QueryPinned(f.ctx(), f.request, checkpoint); err != ca.ErrDenied {
		t.Fatal("another action accepted checkpoint")
	}
}

func TestCheckpointRejectsRegrantAndAvailabilityABA(t *testing.T) {
	for _, mode := range []string{"regrant", "availability", "epoch", "unrelated"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			f.subject.Grants[0].Action = ca.UsageExport
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
			s := exportService(t, f, func(context.Context, ca.Selection) ([]ca.Row, error) { return []ca.Row{rowA()}, nil })
			pin, err := s.Capture(f.ctx(), f.request)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "regrant":
				removed := f.subject
				removed.Grants = nil
				v, err = f.store.PutSubject(removed, v)
				requireWrite(t, v, err)
				v, err = f.store.PutSubject(f.subject, v)
				requireWrite(t, v, err)
			case "availability":
				if err := f.store.SetAvailable(false); err != nil {
					t.Fatal(err)
				}
				if err := f.store.SetAvailable(true); err != nil {
					t.Fatal(err)
				}
			case "epoch":
				f.subject.Epoch = 2
				v, err = f.store.PutSubject(f.subject, v)
				requireWrite(t, v, err)
				f.principal.Epoch = 2
			case "unrelated":
				v, err = f.store.PutSubject(ca.Subject{ID: "other", Epoch: 1, Active: true}, 0)
				requireWrite(t, v, err)
			}
			rows, err := s.QueryPinned(f.ctx(), f.request, pin)
			if mode == "unrelated" {
				if err != nil || len(rows) != 1 {
					t.Fatalf("unrelated: %v", err)
				}
			} else if err != ca.ErrDenied || rows != nil {
				t.Fatal("stale checkpoint accepted")
			}
		})
	}
}
