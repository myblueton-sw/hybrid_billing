package customeraccess_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

var scopeA = ca.Scope{CompanyID: "company-a", TenantID: "tenant-a"}
var scopeB = ca.Scope{CompanyID: "company-b", TenantID: "tenant-b"}

type identityKey struct{}

type fixture struct {
	store     *ca.Store
	now       atomic.Int64
	base      time.Time
	principal ca.Principal
	subject   ca.Subject
	request   ca.Selection
}

func requireWrite(t *testing.T, version uint64, err error) uint64 {
	t.Helper()
	if err != nil || version == 0 {
		t.Fatalf("write: version=%d error=%v", version, err)
	}
	return version
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{base: time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)}
	f.now.Store(f.base.UnixNano())
	var err error
	f.store, err = ca.NewStore("installation-a", []string{"amount", "currency", "quantity"}, func() time.Time {
		return time.Unix(0, f.now.Load())
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range []ca.Scope{scopeA, scopeB} {
		v, err := f.store.PutCompany(ca.Company{ID: scope.CompanyID, Active: true}, 0)
		requireWrite(t, v, err)
		v, err = f.store.PutTenant(ca.Tenant{ID: scope.TenantID, CompanyID: scope.CompanyID, Active: true}, 0)
		requireWrite(t, v, err)
	}
	f.subject = ca.Subject{ID: "subject-a", Active: true, Epoch: 1, Grants: []ca.Grant{{Scope: scopeA, Action: ca.UsageRead, Fields: []string{"amount", "currency"}, ExpiresAt: f.base.Add(time.Hour)}}}
	v, err := f.store.PutSubject(f.subject, 0)
	requireWrite(t, v, err)
	f.principal = ca.Principal{Installation: "installation-a", SubjectID: "subject-a", Epoch: 1, ValidUntil: f.base.Add(time.Hour)}
	f.request = ca.Selection{Scopes: []ca.Scope{scopeA}, Fields: []string{"amount", "currency"}, Limit: 10}
	return f
}

func verify(ctx context.Context) (ca.Principal, error) {
	p, ok := ctx.Value(identityKey{}).(ca.Principal)
	if !ok {
		return ca.Principal{}, errors.New("private verifier diagnostics")
	}
	return p, nil
}

func (f *fixture) ctx() context.Context {
	return context.WithValue(context.Background(), identityKey{}, f.principal)
}

func (f *fixture) service(t *testing.T, reader ca.ReadFunc) *ca.Service {
	t.Helper()
	s, err := ca.NewService(verify, f.store, reader, ca.Limits{MaxRows: 10, MaxScopes: 2, MaxFields: 3})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func rowA() ca.Row {
	return ca.Row{Scope: scopeA, ID: "row-1", Values: map[string]string{"amount": "12345678901234567890.000000001", "currency": "KRW"}}
}

func TestQueryReturnsExactValuesAndOwnsResults(t *testing.T) {
	f := newFixture(t)
	input := []ca.Row{rowA()}
	s := f.service(t, func(ctx context.Context, selection ca.Selection) ([]ca.Row, error) {
		if !reflect.DeepEqual(selection, f.request) || ctx.Value(identityKey{}) == nil {
			t.Fatal("reader did not receive authorized selection/context")
		}
		selection.Scopes[0] = scopeB
		selection.Fields[0] = "quantity"
		return input, nil
	})
	rows, err := s.Query(f.ctx(), f.request)
	if err != nil || !reflect.DeepEqual(rows, []ca.Row{rowA()}) {
		t.Fatalf("result=%v error=%v", rows, err)
	}
	input[0].Values["amount"] = "source changed"
	if rows[0].Values["amount"] != "12345678901234567890.000000001" || f.request.Scopes[0] != scopeA || f.request.Fields[0] != "amount" {
		t.Fatal("ownership leaked")
	}
	rows[0].Values["currency"] = "result changed"
	if input[0].Values["currency"] != "KRW" {
		t.Fatal("result aliases source")
	}
}

func TestDeniedRequestsNeverRead(t *testing.T) {
	cases := map[string]func(*testing.T, *fixture){
		"unknown subject":          func(t *testing.T, f *fixture) { f.principal.SubjectID = "foreign" },
		"wrong installation":       func(t *testing.T, f *fixture) { f.principal.Installation = "installation-b" },
		"stale epoch":              func(t *testing.T, f *fixture) { f.principal.Epoch = 2 },
		"identity expiry equality": func(t *testing.T, f *fixture) { f.principal.ValidUntil = f.base },
		"zero identity expiry":     func(t *testing.T, f *fixture) { f.principal.ValidUntil = time.Time{} },
		"foreign scope":            func(t *testing.T, f *fixture) { f.request.Scopes = []ca.Scope{scopeB} },
		"mixed scopes":             func(t *testing.T, f *fixture) { f.request.Scopes = []ca.Scope{scopeA, scopeB} },
		"wrong company":            func(t *testing.T, f *fixture) { f.request.Scopes[0].CompanyID = scopeB.CompanyID },
		"unknown tenant":           func(t *testing.T, f *fixture) { f.request.Scopes[0].TenantID = "missing" },
		"unknown field":            func(t *testing.T, f *fixture) { f.request.Fields = []string{"raw_source"} },
		"ungranted field":          func(t *testing.T, f *fixture) { f.request.Fields = []string{"quantity"} },
		"duplicate field":          func(t *testing.T, f *fixture) { f.request.Fields = []string{"amount", "amount"} },
		"duplicate scope":          func(t *testing.T, f *fixture) { f.request.Scopes = []ca.Scope{scopeA, scopeA} },
		"no scopes":                func(t *testing.T, f *fixture) { f.request.Scopes = nil },
		"no fields":                func(t *testing.T, f *fixture) { f.request.Fields = nil },
		"zero limit":               func(t *testing.T, f *fixture) { f.request.Limit = 0 },
		"excess limit":             func(t *testing.T, f *fixture) { f.request.Limit = 11 },
		"wildcard":                 func(t *testing.T, f *fixture) { f.request.Scopes[0].TenantID = "*" },
		"inactive company": func(t *testing.T, f *fixture) {
			v, err := f.store.PutCompany(ca.Company{ID: scopeA.CompanyID}, 1)
			requireWrite(t, v, err)
		},
		"inactive tenant": func(t *testing.T, f *fixture) {
			v, err := f.store.PutTenant(ca.Tenant{ID: scopeA.TenantID, CompanyID: scopeA.CompanyID}, 1)
			requireWrite(t, v, err)
		},
		"tenant reassignment": func(t *testing.T, f *fixture) {
			v, err := f.store.PutTenant(ca.Tenant{ID: scopeA.TenantID, CompanyID: scopeB.CompanyID, Active: true}, 1)
			requireWrite(t, v, err)
		},
		"inactive subject": func(t *testing.T, f *fixture) {
			f.subject.Active = false
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
		},
		"removed grants": func(t *testing.T, f *fixture) {
			f.subject.Grants = nil
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
		},
		"expired grant": func(t *testing.T, f *fixture) {
			f.subject.Grants[0].ExpiresAt = f.base
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
		},
		"export does not grant read": func(t *testing.T, f *fixture) {
			f.subject.Grants[0].Action = ca.UsageExport
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			change(t, f)
			s := f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) {
				t.Error("denied request reached reader")
				return []ca.Row{rowA()}, nil
			})
			rows, err := s.Query(f.ctx(), f.request)
			if err != ca.ErrDenied || rows != nil {
				t.Fatalf("rows=%v error=%v", rows, err)
			}
		})
	}
}

func TestPerScopeFieldGrantComposition(t *testing.T) {
	for _, splitAcrossScopes := range []bool{false, true} {
		t.Run(map[bool]string{false: "same scope union", true: "cross scope denied"}[splitAcrossScopes], func(t *testing.T) {
			f := newFixture(t)
			grant := f.subject.Grants[0]
			grant.Fields = []string{"currency"}
			f.subject.Grants[0].Fields = []string{"amount"}
			if splitAcrossScopes {
				grant.Scope = scopeB
				f.request.Scopes = append(f.request.Scopes, scopeB)
			}
			f.subject.Grants = append(f.subject.Grants, grant)
			v, err := f.store.PutSubject(f.subject, 1)
			requireWrite(t, v, err)
			calls := 0
			s := f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) { calls++; return []ca.Row{rowA()}, nil })
			rows, err := s.Query(f.ctx(), f.request)
			if splitAcrossScopes {
				if err != ca.ErrDenied || rows != nil || calls != 0 {
					t.Fatalf("rows=%v error=%v calls=%d", rows, err, calls)
				}
			} else if err != nil || len(rows) != 1 || calls != 1 {
				t.Fatalf("rows=%v error=%v calls=%d", rows, err, calls)
			}
		})
	}
}

func TestMalformedReaderResultsAreAtomicFailures(t *testing.T) {
	cases := map[string]func() []ca.Row{
		"foreign row":   func() []ca.Row { r := rowA(); r.Scope = scopeB; return []ca.Row{rowA(), r} },
		"extra field":   func() []ca.Row { r := rowA(); r.Values["raw_source"] = "private"; return []ca.Row{r} },
		"missing field": func() []ca.Row { r := rowA(); delete(r.Values, "currency"); return []ca.Row{r} },
		"substituted field": func() []ca.Row {
			r := rowA()
			delete(r.Values, "currency")
			r.Values["raw_source"] = "private"
			return []ca.Row{r}
		},
		"duplicate identity": func() []ca.Row { return []ca.Row{rowA(), rowA()} },
		"empty identity":     func() []ca.Row { r := rowA(); r.ID = ""; return []ca.Row{r} },
		"excess rows":        func() []ca.Row { return make([]ca.Row, 11) },
	}
	for name, result := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			s := f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) { return result(), nil })
			rows, err := s.Query(f.ctx(), f.request)
			if err != ca.ErrUnavailable || rows != nil {
				t.Fatalf("rows=%v error=%v", rows, err)
			}
		})
	}
}

func TestVerifierReaderAndCancellationFailures(t *testing.T) {
	f := newFixture(t)
	calls := 0
	s := f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) {
		calls++
		return []ca.Row{rowA()}, errors.New("private source address")
	})
	rows, err := s.Query(context.Background(), f.request)
	if rows != nil || err != ca.ErrDenied || calls != 0 {
		t.Fatalf("unverified: %v %v %d", rows, err, calls)
	}
	rows, err = s.Query(f.ctx(), f.request)
	if rows != nil || err != ca.ErrUnavailable || calls != 1 {
		t.Fatalf("reader error: %v %v %d", rows, err, calls)
	}
	ctx, cancel := context.WithCancel(f.ctx())
	cancel()
	rows, err = s.Query(ctx, f.request)
	if rows != nil || err != context.Canceled || calls != 1 {
		t.Fatalf("cancelled before read: %v %v %d", rows, err, calls)
	}
	ctx, cancel = context.WithCancel(f.ctx())
	s = f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) { cancel(); return []ca.Row{rowA()}, nil })
	rows, err = s.Query(ctx, f.request)
	if rows != nil || err != context.Canceled {
		t.Fatalf("cancelled during read: %v %v", rows, err)
	}
	if err := f.store.SetAvailable(false); err != nil {
		t.Fatal(err)
	}
	s = f.service(t, func(context.Context, ca.Selection) ([]ca.Row, error) {
		t.Error("unavailable store reached reader")
		return nil, nil
	})
	rows, err = s.Query(f.ctx(), f.request)
	if rows != nil || err != ca.ErrUnavailable {
		t.Fatalf("outage: %v %v", rows, err)
	}
}

func TestExplicitMultiTenantGrantAndConcurrentIsolation(t *testing.T) {
	f := newFixture(t)
	subjectB := ca.Subject{ID: "subject-b", Active: true, Epoch: 1, Grants: []ca.Grant{{Scope: scopeB, Action: ca.UsageRead, Fields: []string{"amount", "currency"}, ExpiresAt: f.base.Add(time.Hour)}}}
	v, err := f.store.PutSubject(subjectB, 0)
	requireWrite(t, v, err)
	msp := ca.Subject{ID: "msp", Active: true, Epoch: 1, Grants: append([]ca.Grant{f.subject.Grants[0]}, subjectB.Grants...)}
	v, err = f.store.PutSubject(msp, 0)
	requireWrite(t, v, err)
	s := f.service(t, func(ctx context.Context, selection ca.Selection) ([]ca.Row, error) {
		rows := make([]ca.Row, 0, len(selection.Scopes))
		for _, scope := range selection.Scopes {
			r := rowA()
			r.Scope = scope
			if scope == scopeB {
				r.Values["amount"] = "2.00"
			}
			rows = append(rows, r)
		}
		return rows, nil
	})
	p := f.principal
	p.SubjectID = "msp"
	rows, err := s.Query(context.WithValue(context.Background(), identityKey{}, p), ca.Selection{Scopes: []ca.Scope{scopeA, scopeB}, Fields: []string{"amount", "currency"}, Limit: 2})
	if err != nil || len(rows) != 2 || rows[0].Values["amount"] != "12345678901234567890.000000001" || rows[1].Values["amount"] != "2.00" {
		t.Fatalf("MSP result: %v %v", rows, err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p := f.principal
			scope := scopeA
			want := "12345678901234567890.000000001"
			if i%2 == 1 {
				p.SubjectID = "subject-b"
				scope = scopeB
				want = "2.00"
			}
			request := ca.Selection{Scopes: []ca.Scope{scope}, Fields: []string{"amount", "currency"}, Limit: 1}
			rows, err := s.Query(context.WithValue(context.Background(), identityKey{}, p), request)
			if err != nil || len(rows) != 1 || rows[0].Scope != scope || rows[0].Values["amount"] != want {
				t.Errorf("isolated result: %v %v", rows, err)
			}
		}(i)
	}
	wg.Wait()
}
