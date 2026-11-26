package server

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (f world) assertServiceQueryIDs(a actor, path string, expected []string) {
	f.t.Helper()
	want := slices.Clone(expected)
	slices.Sort(want)
	got := []string{}
	seen := map[string]bool{}
	after := ""
	reference := ""
	for pages := 0; pages < 100; pages++ {
		uri, err := url.Parse(path)
		f.must(err)
		query := uri.Query()
		query.Set("limit", "2")
		if after != "" {
			query.Set("after", after)
		}
		uri.RawQuery = query.Encode()
		reply := f.serviceQuery(a, uri.String())
		if reply.status != 200 {
			f.t.Fatal("query oracle refused", path, reply.status, reply.body["code"])
		}
		rows := reply.body["rows"].([]any)
		if len(rows) > 2 {
			f.t.Fatal("oracle exceeded bounded output")
		}
		current := reply.body["reference_at"].(string)
		if reference == "" {
			reference = current
		} else if reference != current {
			f.t.Fatal("cursor changed reference time")
		}
		for _, raw := range rows {
			id := raw.(map[string]any)["id"].(string)
			if seen[id] {
				f.t.Fatal("paged query duplicated canonical ID", path, id)
			}
			seen[id] = true
			got = append(got, id)
		}
		next := reply.body["next"].(string)
		count := reply.body["count"].(map[string]any)
		if count["value"] != strconv.Itoa(len(got)) {
			f.t.Fatal("cumulative matching count differs from independently traversed set", path)
		}
		if next == "" {
			if count["certainty"] != "exact" {
				f.t.Fatal("terminal count is not exact")
			}
			slices.Sort(got)
			if !slices.Equal(got, want) {
				f.t.Fatalf("independent filter oracle %s: got%v want%v", path, got, want)
			}
			return
		}
		if count["certainty"] != "lower_bound" || next == after {
			f.t.Fatal("unfinished source falsely declared complete or stalled")
		}
		after = next
	}
	f.t.Fatal("bounded oracle did not terminate", path)
}

func TestServiceQueryIndependentExpiryAudienceAndCredentialOracles(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	verifier := root.verifier(f)
	names := []string{"none", "expired", "ending", "soon", "long", "future", "revoked", "threshold", "mixed"}
	services := map[string]string{}
	credentials := map[string]string{}
	serial := 100
	for _, name := range names {
		services[name] = f.newService(root, "oracle-"+name)
	}
	add := func(name string, from, until time.Time) string {
		serial++
		id := fmt.Sprintf("cred-%016x", serial)
		f.must(f.s.identity.Workspace().AddCredential(verifier, SampleWorkspace, services[name], id, identity.Verifier{byte(serial)}, identity.Validity{NotBefore: from, NotAfter: until}))
		credentials[name] = id
		return id
	}
	add("expired", start, start.Add(30*time.Minute))
	add("ending", start, start.Add(time.Hour))
	add("soon", start, start.Add(48*time.Hour))
	add("long", start, start.Add(20*24*time.Hour))
	add("future", start.Add(48*time.Hour), start.Add(20*24*time.Hour))
	revoked := add("revoked", start, start.Add(20*24*time.Hour))
	f.must(f.s.identity.Workspace().RevokeCredential(verifier, SampleWorkspace, revoked))
	add("threshold", start, start.Add(14*24*time.Hour+time.Hour))
	mixedExpired := add("mixed", start, start.Add(30*time.Minute))
	mixedLive := add("mixed", start, start.Add(20*24*time.Hour))
	legacy := "legacy-oracle"
	f.must(f.s.identity.Workspace().CreateServiceUser(verifier, SampleWorkspace, legacy, SampleAdmin))
	f.advance(time.Hour)
	root = f.as(AdminLogin)
	recorded, _, _ := f.issued(root, services["long"], "1")
	all := []string{legacy}
	for _, name := range names {
		all = append(all, services[name])
	}
	declared := []string{}
	for _, name := range names {
		declared = append(declared, services[name])
	}
	cases := []struct {
		query string
		ids   []string
	}{
		{"", all}, {"expiry=none", []string{services["none"], legacy}},
		{"expiry=expired", []string{services["expired"], services["ending"], services["mixed"]}},
		{"expiry=expiring", []string{services["soon"], services["threshold"], services["long"]}},
		{"expiry=valid", []string{services["soon"], services["threshold"], services["long"], services["future"], services["mixed"]}},
		{"audience=test-service-api", declared}, {"audience=test-service-api&scope=read", declared},
		{"audience=test-service-api&scope=absent", nil}, {"audience=absent", nil},
		{"organization=sample-le-seller", declared}, {"declaration=missing", []string{legacy}},
		{"declaration=recorded", declared}, {"owner=" + SampleAdmin, all}, {"owner=unknown", nil},
		{"q=ORACLE", all}, {"q=no-matching-name", nil}, {"state=suspended", nil},
	}
	before := f.s.Revision()
	for _, c := range cases {
		f.assertServiceQueryIDs(root, "/api/v1/identity/services?"+c.query, c.ids)
	}
	credentialCases := []struct {
		query string
		ids   []string
	}{
		{"state=expired", []string{credentials["expired"], credentials["ending"], mixedExpired}},
		{"state=revoked", []string{revoked}},
		{"state=active", []string{credentials["soon"], credentials["long"], credentials["future"], credentials["threshold"], mixedLive, recorded}},
		{"collection=uncollected", []string{recorded}}, {"service=unknown", nil}, {"q=not-a-credential", nil},
		{"service=" + services["none"], nil}, {"service=" + services["mixed"], []string{mixedExpired, mixedLive}},
	}
	for _, c := range credentialCases {
		f.assertServiceQueryIDs(root, "/api/v1/identity/service-credentials?"+c.query, c.ids)
	}
	if f.s.Revision() != before {
		t.Fatal("oracle reads changed command revision")
	}
}
