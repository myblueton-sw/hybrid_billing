package server

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func (f world) candidateRead(a actor, path string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/charge-candidates"+path, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func candidateTerms(workspace string) gate.Terms {
	return gate.Terms{Action: gate.ViewCost, Kind: candidateKind, Fields: []string{candidateField}, Period: gate.AnyPeriod(), Scope: gate.Scope{WorkspaceID: workspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
}

func TestCandidateReadActualSeed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	defer f.s.Close()
	root := f.as(AdminLogin)
	if f.s.chargeProducer.orgs != f.s.orgs || f.s.chargeProducer.lifecycles != f.s.lifecycles || f.s.chargeProducer.domain != &f.s.domain {
		t.Fatal("shadow organizations or lock")
	}
	before := len(f.s.chargeProducer.candidates.records)
	got := f.candidateRead(root, "")
	if got.status != 200 || fmt.Sprint(got.body["count"]) != "1" || got.body["source_mode"] != "synthetic" || got.body["approval_readiness"] != "unverified" || got.body["commands"] != "unconnected" {
		t.Fatalf("list: %d %s", got.status, got.raw)
	}
	row := got.body["records"].([]any)[0].(map[string]any)
	if row["id"] != "sample-charge-01" || row["rating_total"] != "15" || row["currency"] != "USD" || row["cost_basis"] != "billed" || row["has_held_rule"] != false {
		t.Fatalf("literal seed result: %v", row)
	}
	detail := f.candidateRead(root, "/sample-charge-01")
	if detail.status != 200 {
		t.Fatal(detail.raw)
	}
	record := detail.body["record"].(map[string]any)
	provenance := record["provenance"].(map[string]any)
	source := provenance["source_manifest"].(map[string]any)
	if source["workspace"] != SampleWorkspace || source["scope"] != "sample-acct-01" || source["period"] != "2026-10" || source["source_version"] != "synthetic-source-v1" || source["source_cost_field"] != "billed" || source["source_currency_field"] != "currency" || fmt.Sprint(source["manifest"]) != "1" || source["manifest"] != provenance["manifest_reference"] || source["provider"] != provenance["provider"] {
		t.Fatal("source HTTP binding", source)
	}
	for _, field := range []string{"content_digest", "attribution_digest"} {
		if value, ok := source[field].(string); !ok || len(value) != 64 {
			t.Fatal("missing source digest", field, source)
		}
	}
	admission := source["recorded_admission"].(map[string]any)
	if admission["difference"] != "same" || fmt.Sprint(admission["latest_manifest"]) != "1" {
		t.Fatal("source admission", admission)
	}
	for _, field := range []string{"staged_artifacts", "behind_revisions", "same_bytes", "blockers"} {
		if fmt.Sprint(admission[field]) != "0" {
			t.Fatal("source admission observation", field, admission)
		}
	}
	sourceRows := source["rows"].([]any)
	if len(sourceRows) != 2 {
		t.Fatal("source row coverage", sourceRows)
	}
	amounts := map[string]string{"sample-line-a": "10.25", "sample-line-b": "4.75"}
	for _, value := range sourceRows {
		row := value.(map[string]any)
		key, _ := row["key"].(string)
		amount, ok := amounts[key]
		if !ok || row["source_amount"] != amount || row["source_currency"] != "USD" || row["artifact"] != "sample-charge-source" || row["document"] != "" {
			t.Fatal("source row binding", row)
		}
		if digest, ok := row["digest"].(string); !ok || len(digest) != 64 {
			t.Fatal("source row digest", row)
		}
		delete(amounts, key)
	}
	lifecycle := record["provenance"].(map[string]any)["recorded_lifecycle"].(map[string]any)
	segments := lifecycle["segments"].([]any)
	if lifecycle["contract"] != "sample-contract-01" || fmt.Sprint(lifecycle["journal_sequence"]) != "1" || lifecycle["from"] != "2026-10-01" || lifecycle["to_exclusive"] != "2026-11-01" || len(segments) != 1 {
		t.Fatal("lifecycle HTTP shape", lifecycle)
	}
	segment := segments[0].(map[string]any)
	if segment["state"] != "active" || fmt.Sprint(segment["contract_revision"]) != "2" || segment["from"] != "2026-10-01" || segment["to_exclusive"] != "2026-11-01" {
		t.Fatal("lifecycle HTTP segment", segment)
	}
	if len(record["input_lines"].([]any)) != 2 || len(record["charges"].([]any)) != 2 || len(record["provenance"].(map[string]any)["rows"].([]any)) != 2 {
		t.Fatal("incomplete details", detail.raw)
	}
	for _, forbidden := range []string{"\"Policy\"", "\"Contract\"", "\"Cells\"", "sample-charge-policy"} {
		if strings.Contains(detail.raw, forbidden) {
			t.Fatal("internal envelope disclosed", forbidden)
		}
	}
	if len(f.s.chargeProducer.candidates.records) != before {
		t.Fatal("read captured a candidate")
	}
	f.s.domain.Lock()
	_, err := f.s.orgs.PutParty(organizations.Party{ID: "sample-le-seller", WorkspaceID: SampleWorkspace, Kind: organizations.LegalEntity, State: organizations.Suspended}, 1)
	f.s.domain.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	history := f.candidateRead(root, "/sample-charge-01")
	if history.status != 200 || !reflect.DeepEqual(record, history.body["record"]) {
		t.Fatal("historical record changed with live organization", history.raw)
	}
}

func TestCandidateReadAdmission(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	defer f.s.Close()
	root := f.as(AdminLogin)
	for _, path := range []string{"", "/sample-charge-01", "/foreign-id", "?workspace=foreign"} {
		if got := f.candidateRead(actor{}, path); got.status != 401 {
			t.Fatal("anonymous", got.raw)
		}
		if got := f.candidateRead(actor{cookie: root.cookie}, path); got.status != 403 {
			t.Fatal("csrf", got.raw)
		}
		if got := f.candidateRead(f.as(MemberLogin), path); got.status != 403 {
			t.Fatal("member", got.raw)
		}
	}
	for _, path := range []string{"?limit=1", "?workspace=foreign", "?q=%zz", "/sample-charge-01?x=1", "/bad%20id"} {
		if got := f.candidateRead(root, path); got.status != 422 {
			t.Fatal("invalid request", got.raw)
		}
	}
	if got := f.candidateRead(root, "/missing"); got.status != 404 {
		t.Fatal("unknown ID", got.raw)
	}
	f.s.domain.Lock()
	in := candidateFixture()
	in.Workspace = "foreign"
	in.ID = "foreign-id"
	_, err := f.s.chargeProducer.candidates.capture(in)
	f.s.domain.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if got := f.candidateRead(root, "/foreign-id"); got.status != 404 {
		t.Fatal("foreign ID disclosed", got.raw)
	}
	if got := f.candidateRead(root, ""); got.status != 200 || fmt.Sprint(got.body["count"]) != "1" {
		t.Fatal("foreign record listed", got.raw)
	}
}

func TestCandidateReadIndependentGrants(t *testing.T) {
	for _, kind := range []string{"user", "client", "kind", "field", "view only", "customer", "entity", "contract", "period"} {
		t.Run(kind, func(t *testing.T) {
			f := setupWithFields(t, SeedSample, map[string]gate.FieldClass{"state": gate.General, "other_cost": gate.Cost}).ready()
			defer f.s.Close()
			root := f.as(AdminLogin)
			if got := f.candidateRead(root, ""); got.status != 200 {
				t.Fatal("baseline", got.raw)
			}
			f.s.domain.Lock()
			grant := "candidate-" + SampleAdmin
			if kind == "client" {
				grant = "candidate-console"
			}
			if err := f.s.admin.RevokeGrant(grant, 1); err != nil {
				t.Fatal(err)
			}
			if kind == "view only" {
				retained := false
				for _, grant := range f.s.admin.Grants() {
					if grant.ID == "reader-"+SampleAdmin && grant.Terms.Action == gate.View && grant.Effective {
						retained = true
					}
				}
				if !retained {
					t.Fatal("View-only scenario lacks a valid retained View grant")
				}
			}
			if kind != "user" && kind != "client" && kind != "view only" {
				terms := candidateTerms(SampleWorkspace)
				switch kind {
				case "kind":
					terms.Kind = ReaderKind
				case "field":
					terms.Fields = []string{"other_cost"}
				case "customer":
					terms.Scope.Customer = gate.One("sample-le-north")
				case "entity":
					terms.Scope.LegalEntity = gate.One("sample-le-seller")
				case "contract":
					terms.Scope.Contract = gate.One("sample-contract-01")
				case "period":
					from, err := gate.NewMonth(2026, 10)
					if err != nil {
						t.Fatal(err)
					}
					to, err := gate.NewMonth(2026, 11)
					if err != nil {
						t.Fatal(err)
					}
					terms.Period = gate.Months(from, to)
				}
				if err := f.s.admin.AddGrant(gate.Grant{ID: "replacement", SubjectID: SampleAdmin, Terms: terms, Validity: gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}}); err != nil {
					t.Fatal(err)
				}
			}
			f.s.domain.Unlock()
			for _, path := range []string{"", "/sample-charge-01", "/missing"} {
				if got := f.candidateRead(root, path); got.status != 403 {
					t.Fatal("denied grant admitted", got.raw)
				}
			}
		})
	}
}
