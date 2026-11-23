package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialDraftLiveBrowserJourney(t *testing.T) {
	runner, dist := os.Getenv("HB_DRAFT_BROWSER_RUNNER"), os.Getenv("HB_DRAFT_BROWSER_DIST")
	if runner == "" || dist == "" {
		t.Skip("explicit synthetic browser runner and built console required")
	}
	if _, err := os.Stat(filepath.Join(dist, "index.html")); err != nil {
		t.Fatal("built console unavailable")
	}
	f, member, reviewer, spec, _ := configuredDraftWorld(t)
	terms := gate.Terms{Action: gate.View, Kind: approvalKind, Fields: []string{approvalField}, Period: gate.AnyPeriod(), Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	validity := gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "browser-approval-view-human", SubjectID: SampleMember, Terms: terms, Validity: validity}))
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "browser-approval-view-client", ClientID: ConsoleClient, Terms: terms, Validity: validity}))
	files := http.FileServer(http.Dir(dist))
	live := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			f.h.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}))
	f.must(f.s.Bind(live.Listener.Addr()))
	origin := "http://" + live.Listener.Addr().String()
	f.s.origins = map[string]bool{origin: true}
	live.Start()
	defer live.Close()
	input, err := json.Marshal(map[string]any{
		"origin": origin, "request": spec.Application.FinalRequest,
		"submitter": map[string]string{"name": member.cookie.Name, "value": member.cookie.Value},
		"reviewer":  map[string]string{"name": reviewer.cookie.Name, "value": reviewer.cookie.Value},
	})
	f.must(err)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", runner)
	command.Stdin = bytes.NewReader(input)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated browser journey failed: %v\n%s", err, output)
	}
	if len(f.s.financialIssuance.drafts) != 1 {
		t.Fatal("browser did not create exactly one draft")
	}
	if len(f.s.financialIssuance.draftReviews) != 1 {
		t.Fatal("browser did not save exactly one review")
	}
	var review financialDraftReview
	for _, retained := range f.s.financialIssuance.draftReviews {
		review = retained
	}
	for _, draft := range f.s.financialIssuance.drafts {
		view, err := f.s.financialIssuance.documents.DraftOf(SampleWorkspace, draft.Spec.ID)
		f.must(err)
		if draft.Spec.Application == nil || draft.Spec.Application.FinalRequest != spec.Application.FinalRequest || draft.Spec.Application.Configuration != spec.Application.Configuration || draft.Draft.Total != "16" || draft.Draft.Currency != "USD" || draft.Draft.Scale != 2 {
			t.Fatal("browser did not preserve selected source configuration and exact amount")
		}
		if review.DraftID != draft.Spec.ID || review.Actor != "configured-draft-reviewer" || review.State != "approved" || view.Approver != review.Actor || view.Approver == draft.Submitter || view.Coverage == nil || *view.Coverage != review.Coverage {
			t.Fatal("browser receipt lacks independent module approval and exact coverage")
		}
	}
	t.Log(string(output))
}
