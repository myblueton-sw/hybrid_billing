package server

import (
	"errors"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/servicecatalog"
)

func TestServiceChoiceComponentsRejectMalformedPositionsWithoutPartialProgress(t *testing.T) {
	for _, area := range []string{"owners", "organizations", "audiences", "scopes"} {
		t.Run(area, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			path, target := "/api/v1/identity/service-"+area+"?limit=1", ""
			if area == "scopes" {
				path, target = "/api/v1/identity/service-audiences/test-service-api/scopes?limit=1", "test-service-api"
			}
			audience, err := f.s.serviceCatalog.Audience(SampleWorkspace, "test-service-api")
			f.must(err)
			invoke := func(walk *serviceWalk) (int, bool, bool, error) {
				switch area {
				case "owners":
					rows, more, err := walk.owners()
					return len(rows), rows == nil, more, err
				case "organizations":
					rows, more, err := walk.organizations()
					return len(rows), rows == nil, more, err
				case "audiences":
					rows, more, err := walk.audiences()
					return len(rows), rows == nil, more, err
				default:
					rows, more, err := walk.scopes(audience)
					return len(rows), rows == nil, more, err
				}
			}
			r := serviceWalkRequest(root, path)
			control, ok := f.s.beginServiceQuery(httptest.NewRecorder(), r, area, target)
			if !ok {
				t.Fatal("actual healthy choice admission premise")
			}
			if count, nilRows, _, err := invoke(control); err != nil || nilRows || count != 1 || control.scanned == 0 || control.cursor.Matched != 1 {
				t.Fatal("healthy choice component did not consume an actual row")
			}
			admitted, ok := f.s.beginServiceQuery(httptest.NewRecorder(), r, area, target)
			if !ok {
				t.Fatal("fresh genuine admitted choice premise")
			}
			bad := *admitted
			bad.cursor.After = "invalid-position!"
			before := bad.cursor
			basis, err := f.s.serviceBasis(bad.actor, area)
			f.must(err)
			sealed, valid := f.s.sealServiceCursor(bad.cursor)
			if !valid {
				t.Fatal("server-signer malformed position fixture premise")
			}
			assertServiceQueryFailurePayload(t, f.serviceQuery(root, path+"&after="+url.QueryEscape(sealed)), 409, commands.RevisionConflict)
			count, nilRows, more, err := invoke(&bad)
			expected, scanned := identity.ErrInvalid, 0
			if area == "organizations" {
				expected = organizations.ErrInvalid
			} else if area == "audiences" || area == "scopes" {
				expected = servicecatalog.ErrInvalid
				if area == "scopes" {
					scanned = 8
				}
			}
			if !errors.Is(err, expected) || count != 0 || !nilRows || more || !reflect.DeepEqual(bad.cursor, before) || bad.scanned != scanned || bad.total != admitted.total || bad.stopped != admitted.stopped || bad.availability != admitted.availability {
				t.Fatal("malformed component position was not fail-closed without partial progress")
			}
			w := httptest.NewRecorder()
			serviceQueryFailure(w, r, err)
			assertServiceQueryFailurePayload(t, serviceQueryRecordedFailure(t, w), 503, commands.DependencyUnavailable)
			after, err := f.s.serviceBasis(bad.actor, area)
			f.must(err)
			if after != basis || f.me(root).status != 200 {
				t.Fatal("malformed component input changed source or actor authority")
			}
		})
	}
}
