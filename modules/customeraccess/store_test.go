package customeraccess_test

import (
	"context"
	"testing"
	"time"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

func TestEpochRollbackAndReactivationCannotResurrectCredential(t *testing.T) {
	f := newFixture(t)
	f.subject.Epoch = 2
	v, err := f.store.PutSubject(f.subject, 1)
	requireWrite(t, v, err)
	f.subject.Epoch = 1
	if _, err := f.store.PutSubject(f.subject, v); err != ca.ErrInvalidPolicy {
		t.Fatalf("epoch rollback: %v", err)
	}
	s := f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) { return []ca.Row{rowA()}, nil })
	if rows, err := s.Query(f.ctx(), f.request); rows != nil || err != ca.ErrDenied {
		t.Fatalf("old credential restored: %v %v", rows, err)
	}
	f.subject.Epoch = 2
	f.subject.Active = false
	v, err = f.store.PutSubject(f.subject, v)
	requireWrite(t, v, err)
	f.subject.Active = true
	if _, err := f.store.PutSubject(f.subject, v); err != ca.ErrInvalidPolicy {
		t.Fatalf("same epoch reactivation: %v", err)
	}
	f.subject.Epoch = 3
	v, err = f.store.PutSubject(f.subject, v)
	requireWrite(t, v, err)
	f.principal.Epoch = 2
	if rows, err := s.Query(f.ctx(), f.request); rows != nil || err != ca.ErrDenied {
		t.Fatalf("disabled credential restored: %v %v", rows, err)
	}
	f.principal.Epoch = 3
	if rows, err := s.Query(f.ctx(), f.request); err != nil || len(rows) != 1 {
		t.Fatalf("fresh credential: %v %v", rows, err)
	}
}

func TestInvalidUpdatesPreserveDataAndVersion(t *testing.T) {
	cases := map[string]func(*ca.Subject){
		"zero epoch":      func(s *ca.Subject) { s.Epoch = 0 },
		"unknown field":   func(s *ca.Subject) { s.Grants[0].Fields = []string{"private"} },
		"duplicate field": func(s *ca.Subject) { s.Grants[0].Fields = []string{"amount", "amount"} },
		"no fields":       func(s *ca.Subject) { s.Grants[0].Fields = nil },
		"unknown action":  func(s *ca.Subject) { s.Grants[0].Action = "admin" },
		"no expiry":       func(s *ca.Subject) { s.Grants[0].ExpiresAt = time.Time{} },
		"wrong company":   func(s *ca.Subject) { s.Grants[0].Scope.CompanyID = scopeB.CompanyID },
		"unknown tenant":  func(s *ca.Subject) { s.Grants[0].Scope.TenantID = "missing" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			change(&f.subject)
			if _, err := f.store.PutSubject(f.subject, 1); err != ca.ErrInvalidPolicy {
				t.Fatalf("invalid update: %v", err)
			}
			s := f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) { return []ca.Row{rowA()}, nil })
			if rows, err := s.Query(f.ctx(), f.request); err != nil || len(rows) != 1 {
				t.Fatalf("state damaged: %v %v", rows, err)
			}
			v, err := f.store.PutSubject(ca.Subject{ID: "subject-a", Epoch: 1}, 1)
			if err != nil || v != 2 {
				t.Fatalf("version changed after rejection: %d %v", v, err)
			}
		})
	}
}

func TestStaleSnapshotsDoNotRestoreRevokedGrants(t *testing.T) {
	f := newFixture(t)
	removed := f.subject
	removed.Grants = nil
	v, err := f.store.PutSubject(removed, 1)
	requireWrite(t, v, err)
	if _, err := f.store.PutSubject(f.subject, 1); err != ca.ErrConflict {
		t.Fatalf("stale restore: %v", err)
	}
	s := f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) {
		t.Error("revoked grant reached reader")
		return nil, nil
	})
	if rows, err := s.Query(f.ctx(), f.request); rows != nil || err != ca.ErrDenied {
		t.Fatalf("revoked grant restored: %v %v", rows, err)
	}
	if _, err := f.store.PutCompany(ca.Company{ID: scopeA.CompanyID}, 0); err != ca.ErrConflict {
		t.Fatalf("company CAS: %v", err)
	}
	if _, err := f.store.PutTenant(ca.Tenant{ID: scopeA.TenantID, CompanyID: scopeA.CompanyID}, 0); err != ca.ErrConflict {
		t.Fatalf("tenant CAS: %v", err)
	}
}

func TestStoreOwnsGrantAndCatalogCopies(t *testing.T) {
	f := newFixture(t)
	f.subject.Grants[0].Scope = scopeB
	f.subject.Grants[0].Fields[0] = "quantity"
	f.subject.Grants = nil
	s := f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) { return []ca.Row{rowA()}, nil })
	if rows, err := s.Query(f.ctx(), f.request); err != nil || len(rows) != 1 {
		t.Fatalf("grant alias: %v %v", rows, err)
	}
	fields := []string{"amount"}
	store, err := ca.NewStore("installation", fields, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	fields[0] = "private"
	v, err := store.PutCompany(ca.Company{ID: "c", Active: true}, 0)
	requireWrite(t, v, err)
	v, err = store.PutTenant(ca.Tenant{ID: "t", CompanyID: "c", Active: true}, 0)
	requireWrite(t, v, err)
	subject := ca.Subject{ID: "s", Active: true, Epoch: 1, Grants: []ca.Grant{{Scope: ca.Scope{CompanyID: "c", TenantID: "t"}, Action: ca.UsageRead, Fields: []string{"amount"}, ExpiresAt: time.Now().Add(time.Hour)}}}
	v, err = store.PutSubject(subject, 0)
	requireWrite(t, v, err)
	subject.Grants[0].Fields[0] = "private"
	if _, err := store.PutSubject(subject, v); err != ca.ErrInvalidPolicy {
		t.Fatalf("catalog alias: %v", err)
	}
}

func TestConstructionFailsClosed(t *testing.T) {
	for _, fields := range [][]string{nil, {""}, {"*"}, {"amount", "amount"}} {
		if _, err := ca.NewStore("installation", fields, time.Now); err != ca.ErrInvalidPolicy {
			t.Fatalf("invalid catalog: %v", err)
		}
	}
	if _, err := ca.NewStore("installation", []string{"amount"}, nil); err != ca.ErrInvalidPolicy {
		t.Fatalf("nil clock: %v", err)
	}
	f := newFixture(t)
	read := func(context.Context, ca.Selection) ([]ca.Row, error) { return nil, nil }
	limits := ca.Limits{MaxRows: 1, MaxScopes: 1, MaxFields: 1}
	if _, err := ca.NewService(nil, f.store, read, limits); err != ca.ErrInvalidPolicy {
		t.Fatalf("nil verifier: %v", err)
	}
	if _, err := ca.NewService(verify, nil, read, limits); err != ca.ErrInvalidPolicy {
		t.Fatalf("nil store: %v", err)
	}
	if _, err := ca.NewService(verify, &ca.Store{}, read, limits); err != ca.ErrInvalidPolicy {
		t.Fatalf("zero store: %v", err)
	}
	if _, err := ca.NewService(verify, f.store, nil, limits); err != ca.ErrInvalidPolicy {
		t.Fatalf("nil reader: %v", err)
	}
	if _, err := ca.NewService(verify, f.store, read, ca.Limits{}); err != ca.ErrInvalidPolicy {
		t.Fatalf("empty limits: %v", err)
	}
	fresh, err := ca.NewStore("installation-a", []string{"amount", "currency"}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ca.NewService(verify, fresh, func(context.Context, ca.Selection) ([]ca.Row, error) {
		t.Error("empty store reached reader")
		return nil, nil
	}, limits)
	if err != nil {
		t.Fatal(err)
	}
	request := ca.Selection{Scopes: []ca.Scope{scopeA}, Fields: []string{"amount"}, Limit: 1}
	if rows, err := s.Query(f.ctx(), request); rows != nil || err != ca.ErrDenied {
		t.Fatalf("empty store: %v %v", rows, err)
	}
}
