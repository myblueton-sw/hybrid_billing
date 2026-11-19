package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

func TestHistoryConfiguredPeriodAndSettingCatalogue(t *testing.T) {
	list := HistorySettingList()
	if len(list) != 3 || list[0].Default != 90 || list[0].Min != 1 || list[0].Max != 1095 {
		t.Fatalf("history installation catalogue: %+v", list)
	}
	flag := list[0].Flag
	list[0].Flag = "changed-by-caller"
	if HistorySettingList()[0].Flag != flag {
		t.Fatal("caller modified installation catalogue")
	}
	f := setup(t, SeedSample)
	f.s.cfg.Access.HistoryDays = 30
	if f.s.cfg.Access.Given == nil {
		f.s.cfg.Access.Given = map[string]bool{}
	}
	f.s.cfg.Access.Given["access_history_days"] = true
	f = f.ready()
	f.setPassword(MemberLogin, pw)
	defer f.s.Close()
	got := f.history(f.as(MemberLogin), "")
	if got.status != 200 || fmt.Sprint(got.body["retention_days"]) != "30" || got.body["retention_source"] != SourceStartFlag {
		t.Fatalf("configured period not exposed: %+v", got)
	}
}

func TestHistoryUnsafeStorageAndDuplicateOpenRefuseConfiguration(t *testing.T) {
	t.Run("unsafe history path", func(t *testing.T) {
		f := setup(t, SeedSample)
		defer f.s.Close()
		if err := os.MkdirAll(f.s.cfg.Data, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.s.cfg.Data, accesshistory.DirName), []byte("not a directory"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := f.s.Start(); !errors.Is(err, ErrConfig) || f.s.history != nil {
			t.Fatalf("unsafe history storage not refused: %v", err)
		}
	})
	t.Run("duplicate writer", func(t *testing.T) {
		f := setup(t, SeedSample).ready()
		defer f.s.Close()
		previous := f.s.history
		if _, err := f.s.openHistory(); !errors.Is(err, ErrConfig) || f.s.history != previous {
			t.Fatalf("duplicate history writer not refused: %v", err)
		}
	})
}

func TestHistoryUnavailableAdapterAndRemovedIdentityDoNotInventEvents(t *testing.T) {
	f := historyWorld(t)
	defer f.s.Close()
	a := f.signedIn(MemberLogin, pw)
	previous := f.s.history
	f.s.history = nil
	if err := f.s.incarnation("not-started"); err != nil {
		t.Fatalf("unstarted history marker: %v", err)
	}
	got := f.do("GET", "/api/v1/me/access-history", "", func(r *http.Request) { r.AddCookie(a.cookie); r.Header.Set(csrfHeader, a.csrf) })
	f.s.history = previous
	if got.status != 503 || got.body["code"] != "dependency_unavailable" {
		t.Fatalf("missing producer shown as successful history: %+v", got)
	}
	captured, ok := f.s.sessions.get(verifierOfActor(a))
	if !ok {
		t.Fatal("no member session")
	}
	root := f.as(AdminLogin)
	if ended := f.run(root, "identity.depart_user", SampleMember, nil); ended.status != 200 {
		t.Fatalf("depart member: %+v", ended)
	}
	previous.Flush()
	before := len(previous.Read(SampleMember, accesshistory.Query{Workspace: SampleWorkspace}).Events)
	f.s.recordSession(accesshistory.KindSessionEnded, accesshistory.CauseAdministrator, captured)
	previous.Flush()
	if after := len(previous.Read(SampleMember, accesshistory.Query{Workspace: SampleWorkspace}).Events); after != before {
		t.Fatalf("stale removed identity created event: before=%d after=%d", before, after)
	}
}

func TestHistoryStoppedRecordingMapsNotesWithoutBreakingSignIn(t *testing.T) {
	f := historyWorld(t)
	defer f.s.Close()
	tomorrow := start.Add(24 * time.Hour)
	if err := os.Mkdir(filepath.Join(f.s.cfg.Data, accesshistory.DirName, tomorrow.Format("2006-01-02")+".log"), 0o700); err != nil {
		t.Fatal(err)
	}
	f.advance(24 * time.Hour)
	refused := f.signinAgent(MemberLogin, "wrong password here", chromeMac)
	if refused.status != 401 {
		t.Fatalf("recording failure changed sign-in refusal: %+v", refused)
	}
	f.s.history.Flush()
	a := f.signedIn(MemberLogin, pw)
	got := f.history(a, "")
	if got.status != 200 {
		t.Fatalf("failed recorder blocked own history status: %+v", got)
	}
	recording := got.body["recording"].(map[string]any)
	if recording["state"] != "stopped" || recording["reason"] != "write_failed" || recording["since"] == nil {
		t.Fatalf("stopped recorder not mapped: %v", recording)
	}
	own := false
	for _, value := range got.body["notes"].([]any) {
		n := value.(map[string]any)
		if n["kind"] == "dropped" {
			own = true
			if fmt.Sprint(n["count"]) != "2" {
				t.Errorf("own missing events: %v", n)
			}
		}
	}
	if !own {
		t.Fatal("recording gaps not disclosed")
	}
	other := f.history(f.as(AdminLogin), "")
	for _, value := range other.body["notes"].([]any) {
		n := value.(map[string]any)
		if n["kind"] == "dropped" && n["count"] != nil {
			t.Fatalf("another person's missing-event count exposed: %v", n)
		}
	}
}

func TestHistoryHoldSnapshotUsesRetentionCoverage(t *testing.T) {
	if got := (&Server{}).historyHolds(); len(got) != 0 {
		t.Fatal("unstarted retention invented a hold")
	}
	f := setup(t, SeedSample).ready()
	defer f.s.Close()
	holds := []retention.Hold{
		{ID: "history-identity", Owner: SampleAdmin, Reason: "identity preservation", Tenants: []string{SampleWorkspace}, Classes: []string{retention.ClassIdentity}},
		{ID: "history-all", Owner: SampleAdmin, Reason: "whole tenant preservation", Tenants: []string{SampleWorkspace}},
		{ID: "history-source", Owner: SampleAdmin, Reason: "source only", Tenants: []string{SampleWorkspace}, Classes: []string{retention.ClassSource}},
	}
	for _, hold := range holds {
		if _, err := f.s.retention.PlaceHold(hold, retA1); err != nil {
			t.Fatal(err)
		}
	}
	f.s.holdPlaced.Store("history-identity", start)
	snapshot := f.s.historyHolds()
	if len(snapshot) != 2 {
		t.Fatalf("nonidentity hold included or covering hold absent: %+v", snapshot)
	}
	for _, hold := range snapshot {
		if !hold.Covers(SampleWorkspace) || hold.Covers("other-workspace") {
			t.Fatalf("hold scope broadened: %+v", hold)
		}
		if hold.ID == "history-identity" && !hold.PlacedAt.Equal(start) {
			t.Fatal("known hold placement lost")
		}
		if hold.ID == "history-all" && !hold.PlacedAt.IsZero() {
			t.Fatal("unknown hold placement fabricated")
		}
	}
}

func TestHistoryNonSampleInvitationCreatesAnIncarnation(t *testing.T) {
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatalf("bootstrap: %+v", got)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	root := actor{cookie, csrf}
	_, link := f.created(root, "second@example.invalid", "administrator")
	if got := f.redeem(link, pw); got.status != 204 {
		t.Fatalf("non-sample invitation: %+v", got)
	}
	signed := f.auth("login", noneLogin("second@example.invalid"), f.pre())
	if signed.status != 200 {
		t.Fatalf("non-sample sign-in: %+v", signed)
	}
	var member actor
	for _, c := range readCookies(signed.header) {
		if c.Name == CookieName {
			member = actor{c, signed.body["csrf_token"].(string)}
		}
	}
	if got := f.history(member, ""); kindList(historyEvents(t, got)) != "signed_in" {
		t.Fatalf("new incarnation history: %+v", got)
	}
}

func TestHistoryMalformedDecodedCursorsAreRefused(t *testing.T) {
	f := historyWorld(t)
	defer f.s.Close()
	a := f.signedIn(MemberLogin, pw)
	for _, raw := range []string{"1", "-0", "1a-0", strings.Repeat("1", 21) + "-0", "18446744073709551616-0"} {
		cursor := base64.RawURLEncoding.EncodeToString([]byte(raw))
		got := f.history(a, "before="+cursor)
		if got.status != 422 || got.body["code"] != "invalid_request" || !strings.Contains(fmt.Sprint(got.body["fields"]), "before") {
			t.Errorf("malformed cursor %q: %+v", raw, got)
		}
	}
}

func TestHistorySeveralSessionRetriesRemainIdempotent(t *testing.T) {
	f := historyWorld(t)
	defer f.s.Close()
	a, b, c := f.as(MemberLogin), f.as(MemberLogin), f.as(MemberLogin)
	f.failModuleEnds("self-end-unavailable")
	got := f.selfPost(a, "sessions/end", f.othersBody(a))
	if got.status != 202 || len(f.s.retries) != 2 {
		t.Fatalf("two failed session ends not retained: %+v", got)
	}
	f.s.history.Flush()
	before := len(f.s.history.Read(SampleMember, accesshistory.Query{Workspace: SampleWorkspace}).Events)
	v := verifierOfActor(b)
	if !f.s.sessions.deny(v, MemberLogin, start.Add(time.Hour)) {
		t.Fatal("duplicate denial lost")
	}
	f.s.sessions.mu.Lock()
	same := f.s.sessions.denied[v].at.Equal(start)
	removedAgain := f.s.sessions.remove(v)
	f.s.sessions.mu.Unlock()
	if !same || removedAgain {
		t.Fatal("repeated session cleanup changed denial or removed an absent session")
	}
	f.s.fault = nil
	f.advance(selfEndBackoff[0])
	if worked := f.drainSelf(); worked != 2 {
		t.Fatalf("retried %d sessions", worked)
	}
	if f.s.sessions.isDenied(v) || f.s.sessions.isDenied(verifierOfActor(c)) || len(f.s.retries) != 0 {
		t.Fatal("successful cleanup retained retry or deny state")
	}
	f.s.history.Flush()
	after := len(f.s.history.Read(SampleMember, accesshistory.Query{Workspace: SampleWorkspace}).Events)
	if after != before {
		t.Fatalf("cleanup retries duplicated access events: before=%d after=%d", before, after)
	}
	var absent identity.Verifier
	f.s.sessions.mu.Lock()
	absentRemoved := f.s.sessions.remove(absent)
	f.s.sessions.mu.Unlock()
	if absentRemoved {
		t.Fatal("nonexistent session removed")
	}
}
