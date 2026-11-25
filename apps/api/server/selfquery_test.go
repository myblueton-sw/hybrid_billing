package server

import "testing"

func TestSelfAccountResponseStaysBoundedAcrossSessionAndResultPopulations(t *testing.T) {
	f, a := selectionPopulation(t, 150)
	assertBounded := func(expectedSessions, expectedOperations int) {
		t.Helper()
		page := f.me(a)
		if page.status != 200 {
			t.Fatal(page.status)
		}
		if _, expanded := page.body["sessions"]; expanded {
			t.Fatal("account read still downloads the session population")
		}
		if _, expanded := page.body["session_ends"]; expanded {
			t.Fatal("account read still expands retained operation results")
		}
		if countOf(page.body["recorded_session_count"]) != expectedSessions || countOf(page.body["accepted_operation_count"]) != expectedOperations {
			t.Fatal("account recorded counts are not scoped totals")
		}
		row, ok := page.body["current_session"].(map[string]any)
		if !ok || row["handle"] != handleOfActor(a) || row["current"] != true || row["can_end"] != false {
			t.Fatal("current session missing independently of the query page")
		}
		if len(page.raw) > 8000 {
			t.Fatal("account response exceeds its bounded metadata shape")
		}
	}
	assertBounded(151, 0)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	for n := 0; prepared.status == 200 && prepared.body["state"] == "building" && n < 5; n++ {
		prepared = f.selfPost(a, "sessions/selections", selectionResumeBody(prepared))
	}
	if prepared.status != 200 || prepared.body["state"] != "sealed" {
		t.Fatal("selection preparation failed")
	}
	if got := f.selfPost(a, "sessions/end", selectionEndBody(prepared)); got.status != 202 {
		t.Fatal(got.status)
	}
	assertBounded(1, 1)
	for i := 0; i < 120; i++ {
		if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
			t.Fatal(err)
		}
	}
	assertBounded(121, 1)
}
