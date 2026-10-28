package customeraccess_test

import (
	"context"
	"testing"
	"time"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

func TestChangesDuringPausedRead(t *testing.T) {
	cases := []struct {
		name   string
		change func(*testing.T, *fixture)
		want   error
	}{
		{"grant removal", func(t *testing.T, f *fixture) {
			f.subject.Grants = nil
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
		}, ca.ErrDenied},
		{"revoke and regrant", func(t *testing.T, f *fixture) {
			removed := f.subject
			removed.Grants = nil
			v, err := f.store.PutSubject(removed, 1)
			requireWrite(t, v, err)
			v, err = f.store.PutSubject(f.subject, v)
			requireWrite(t, v, err)
		}, ca.ErrDenied},
		{"epoch rotation", func(t *testing.T, f *fixture) {
			f.subject.Epoch = 2
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
		}, ca.ErrDenied},
		{"subject disabled", func(t *testing.T, f *fixture) {
			f.subject.Active = false
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
		}, ca.ErrDenied},
		{"company inactive and restored", func(t *testing.T, f *fixture) {
			v, err := f.store.PutCompany(ca.Company{ID: scopeA.CompanyID}, 1)
			requireWrite(t, v, err)
			v, err = f.store.PutCompany(ca.Company{ID: scopeA.CompanyID, Active: true}, v)
			requireWrite(t, v, err)
		}, ca.ErrDenied},
		{"tenant moved and restored", func(t *testing.T, f *fixture) {
			v, err := f.store.PutTenant(ca.Tenant{ID: scopeA.TenantID, CompanyID: scopeB.CompanyID, Active: true}, 1)
			requireWrite(t, v, err)
			v, err = f.store.PutTenant(ca.Tenant{ID: scopeA.TenantID, CompanyID: scopeA.CompanyID, Active: true}, v)
			requireWrite(t, v, err)
		}, ca.ErrDenied},
		{"equivalent relevant update", func(t *testing.T, f *fixture) {
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
		}, ca.ErrDenied},
		{"expiry equality during read", func(t *testing.T, f *fixture) { f.now.Store(f.base.Add(time.Hour).UnixNano()) }, ca.ErrDenied},
		{"grant expiry during read", func(t *testing.T, f *fixture) { f.now.Store(f.base.Add(time.Minute).UnixNano()) }, ca.ErrDenied},
		{"outage", func(t *testing.T, f *fixture) {
			if err := f.store.SetAvailable(false); err != nil {
				t.Fatal(err)
			}
		}, ca.ErrUnavailable},
		{"outage and recovery", func(t *testing.T, f *fixture) {
			if err := f.store.SetAvailable(false); err != nil {
				t.Fatal(err)
			}
			if err := f.store.SetAvailable(true); err != nil {
				t.Fatal(err)
			}
		}, ca.ErrDenied},
		{"unrelated company", func(t *testing.T, f *fixture) {
			v, err := f.store.PutCompany(ca.Company{ID: scopeB.CompanyID}, 1)
			requireWrite(t, v, err)
		}, nil},
		{"unrelated tenant", func(t *testing.T, f *fixture) {
			v, err := f.store.PutTenant(ca.Tenant{ID: scopeB.TenantID, CompanyID: scopeB.CompanyID}, 1)
			requireWrite(t, v, err)
		}, nil},
		{"unrelated subject", func(t *testing.T, f *fixture) {
			v, err := f.store.PutSubject(ca.Subject{ID: "unrelated", Active: true, Epoch: 1}, 0)
			requireWrite(t, v, err)
		}, nil},
		{"stale write rejected", func(t *testing.T, f *fixture) {
			f.subject.Grants = nil
			if _, err := f.store.PutSubject(f.subject, 0); err != ca.ErrConflict {
				t.Fatalf("stale write: %v", err)
			}
		}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.name == "grant expiry during read" {
				f.subject.Grants[0].ExpiresAt = f.base.Add(time.Minute)
				v, err := f.store.PutSubject(f.subject, 1)
				requireWrite(t, v, err)
			}
			started := make(chan struct{})
			release := make(chan struct{})
			s := f.service(t, func(ctx context.Context, selection ca.Selection) ([]ca.Row, error) {
				close(started)
				select {
				case <-release:
					return []ca.Row{rowA()}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			})
			type outcome struct {
				rows []ca.Row
				err  error
			}
			done := make(chan outcome, 1)
			ctx, cancel := context.WithTimeout(f.ctx(), 5*time.Second)
			defer cancel()
			go func() { rows, err := s.Query(ctx, f.request); done <- outcome{rows, err} }()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("reader did not start")
			}
			tc.change(t, f)
			close(release)
			select {
			case got := <-done:
				if got.err != tc.want || (tc.want != nil && got.rows != nil) || (tc.want == nil && len(got.rows) != 1) {
					t.Fatalf("rows=%v error=%v want=%v", got.rows, got.err, tc.want)
				}
			case <-ctx.Done():
				t.Fatal("query did not finish")
			}
		})
	}
}

func TestCallerSelectionCopiedBeforeVerifierAndRead(t *testing.T) {
	f := newFixture(t)
	request := f.request
	s, err := ca.NewService(func(context.Context) (ca.Principal, error) {
		request.Scopes[0] = scopeB
		request.Fields[0] = "quantity"
		return f.principal, nil
	}, f.store, func(ctx context.Context, selection ca.Selection) ([]ca.Row, error) {
		if selection.Scopes[0] != scopeA || selection.Fields[0] != "amount" {
			t.Fatal("request aliases reader")
		}
		return []ca.Row{rowA()}, nil
	}, ca.Limits{MaxRows: 10, MaxScopes: 2, MaxFields: 3})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Query(context.Background(), request)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v error=%v", rows, err)
	}
}
