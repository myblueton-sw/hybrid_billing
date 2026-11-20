package server

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestCandidateReadEmptyInstallation(t *testing.T) {
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatal("bootstrap", got.status)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	root := actor{cookie, csrf}
	if got := f.candidateRead(root, ""); got.status != 403 {
		t.Fatal("unseeded read grant exists", got.raw)
	}
	f.s.domain.Lock()
	terms := candidateTerms(EmptyWorkspace)
	valid := gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
	for _, grant := range []gate.Grant{{ID: "empty-candidate-client", ClientID: ConsoleClient, Terms: terms, Validity: valid}, {ID: "empty-candidate-user", SubjectID: EmptyAdmin, Terms: terms, Validity: valid}} {
		if err := f.s.admin.AddGrant(grant); err != nil {
			t.Fatal(err)
		}
	}
	f.s.domain.Unlock()
	got := f.candidateRead(root, "")
	if got.status != 200 || fmt.Sprint(got.body["count"]) != "0" || len(got.body["records"].([]any)) != 0 || got.header.Get("Cache-Control") != "no-store" {
		t.Fatal("empty response", got.raw)
	}
	if len(f.s.chargeProducer.candidates.records) != 0 {
		t.Fatal("empty read captured data")
	}
}

func TestCandidateReadOrderingAndEndedSession(t *testing.T) {
	f := setup(t, SeedSample).ready()
	defer f.s.Close()
	root := f.as(AdminLogin)
	f.s.domain.Lock()
	for _, id := range []string{"z-last", "a-first"} {
		in := candidateFixture()
		in.Workspace = SampleWorkspace
		in.ID = id
		if _, err := f.s.chargeProducer.candidates.capture(in); err != nil {
			t.Fatal(err)
		}
	}
	f.s.domain.Unlock()
	before := f.s.counter
	got := f.candidateRead(root, "")
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	ids := []string{}
	for _, raw := range got.body["records"].([]any) {
		ids = append(ids, raw.(map[string]any)["id"].(string))
	}
	if !reflect.DeepEqual(ids, []string{"a-first", "sample-charge-01", "z-last"}) || fmt.Sprint(got.body["count"]) != "3" || f.s.counter != before {
		t.Fatal("incomplete order or read mutation", got.raw)
	}
	if got.header.Get("Cache-Control") != "no-store" || f.candidateRead(root, "/sample-charge-01").header.Get("Cache-Control") != "no-store" {
		t.Fatal("cacheable cost evidence")
	}
	f.s.domain.Lock()
	for _, verifier := range f.s.sessions.byLogin(AdminLogin) {
		if err := f.s.identity.Boundary().EndSession(verifier); err != nil {
			t.Fatal(err)
		}
	}
	f.s.domain.Unlock()
	for _, path := range []string{"", "/sample-charge-01"} {
		if got := f.candidateRead(root, path); got.status != 401 {
			t.Fatal("ended identity session admitted", got.raw)
		}
	}
}

func TestCandidateReadMalformedStoredJSON(t *testing.T) {
	f := setup(t, SeedSample).ready()
	defer f.s.Close()
	root := f.as(AdminLogin)
	f.s.domain.Lock()
	store := &f.s.chargeProducer.candidates
	key := [2]string{SampleWorkspace, "sample-charge-01"}
	store.mu.Lock()
	record := store.records[key]
	record.envelope = []byte("{")
	store.records[key] = record
	store.mu.Unlock()
	f.s.domain.Unlock()
	for _, path := range []string{"", "/sample-charge-01"} {
		if got := f.candidateRead(root, path); got.status != 503 {
			t.Fatal("malformed JSON not refused", got.raw)
		}
	}
}

func TestCandidateReadSeedReplayAndDocument(t *testing.T) {
	f := setup(t, SeedSample).ready()
	defer f.s.Close()
	root := f.as(AdminLogin)
	before := f.candidateRead(root, "")
	f.s.domain.Lock()
	err := f.s.seedChargeCandidates(f.s.clock())
	f.s.domain.Unlock()
	if !errors.Is(err, gate.ErrConflict) {
		t.Fatal("duplicate seed did not reach duplicate grant refusal", err)
	}
	after := f.candidateRead(root, "")
	if after.status != 200 || !reflect.DeepEqual(before.body, after.body) {
		t.Fatal("seed replay changed candidates", after.raw)
	}
	for _, detail := range []bool{false, true} {
		path := candidateReadPath(detail)["get"].(map[string]any)
		if path["x-action"] != string(gate.ViewCost) || path["x-field"] != candidateField || path["x-csrf"] != true {
			t.Fatal("read documentation weakens admission")
		}
	}
	if candidateReadPath(false)["get"].(map[string]any)["operationId"] == candidateReadPath(true)["get"].(map[string]any)["operationId"] {
		t.Fatal("duplicate operation ID")
	}
}

func TestCandidateReadGrantRequiresSeparateApproval(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	proposer, approver, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	asked := f.requestGrant(proposer, SampleMember, map[string]string{"action": "view_cost", "kind": candidateKind, "fields": candidateField})
	if asked.status != 200 || asked.body["business_state"] != "requested" {
		t.Fatal("request", asked.raw)
	}
	if got := f.candidateRead(member, ""); got.status != 403 {
		t.Fatal("pending request granted access", got.raw)
	}
	if got := f.approveRequest(proposer, asked); got.status != 422 {
		t.Fatal("self approval", got.raw)
	}
	if got := f.approveRequest(approver, asked); got.status != 200 || got.body["business_state"] != "granted" {
		t.Fatal("separate approval", got.raw)
	}
	member = f.as(MemberLogin)
	if got := f.candidateRead(member, ""); got.status != 200 {
		t.Fatal("approved cost access", got.raw)
	}
}
