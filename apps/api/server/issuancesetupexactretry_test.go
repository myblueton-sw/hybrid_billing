package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func moveRevision(t *testing.T, f world) {
	t.Helper()
	before := f.s.Revision()
	f.must(f.s.Change("synthetic unrelated change", func() (bool, error) { return true, nil }))
	if f.s.Revision() == before {
		t.Fatal("the revision did not move")
	}
}

func TestIssuanceSetupExactRetryOfUnadmittedRequestIsRevisionConflict(t *testing.T) {
	for _, first := range []string{"never sent", "refused before any effect"} {
		for _, kind := range []string{"register_channel", "request_change"} {
			t.Run(first+"/"+kind, func(t *testing.T) {
				call := publicProtectedStage(t, kind)
				f, a := call.w.f, call.w.f.s.financialIssuance
				observe := func() protectedWriteObservation {
					var out protectedWriteObservation
					f.must(inputAuthorityAs(f, call.who, func() error {
						out = a.observeProtectedWrite(SampleWorkspace, call.family, call.id, call.claim)
						return nil
					}))
					return out
				}
				before := observe()
				body := protectedWriteBody(f, "lost-write", call.kind, call.w.c, call.fields)
				if first == "refused before any effect" {
					f.s.fault = func(at string) bool { return at == "issuance-channel-lookup" || at == "issuance-change-lookup" }
					got := f.call("issuance_setup."+call.kind, body, call.who.cookie, call.who.csrf)
					f.s.fault = nil
					f.code(t, "clean refusal", got, 503, string(commands.DependencyUnavailable))
					if got.body["operation_id"] != nil {
						t.Fatal("a clean refusal carried an operation", got.raw)
					}
				}
				moveRevision(t, f)
				moved := f.s.Revision()
				for attempt := 1; attempt <= 2; attempt++ {
					got := f.call("issuance_setup."+call.kind, body, call.who.cookie, call.who.csrf)
					f.code(t, "exact retry", got, 409, string(commands.RevisionConflict))
					if got.body["operation_id"] != nil || f.s.Revision() != moved || !reflect.DeepEqual(before, observe()) {
						t.Fatal("the refused retry carried an operation or changed state", attempt, got.raw)
					}
				}
				f.ok(t, "prepared again from the current revision", protectedWrite(f, call.who, call.kind, call.w.c, call.fields), call.recovered, "")
			})
		}
	}
}

func TestIssuanceSetupExactRetryOfStoredEffectIsReplayedAfterRevisionMoved(t *testing.T) {
	for _, tc := range []struct{ kind, fault string }{{"register_channel", "issuance-channel-after-register"}, {"request_change", "issuance-change-after-request"}} {
		t.Run(tc.kind, func(t *testing.T) {
			call := publicProtectedStage(t, tc.kind)
			f := call.w.f
			f.s.fault = func(at string) bool { return at == tc.fault }
			body := protectedWriteBody(f, "applied-write", call.kind, call.w.c, call.fields)
			got := f.call("issuance_setup."+call.kind, body, call.who.cookie, call.who.csrf)
			f.s.fault = nil
			if got.status != 500 || got.body["operation_id"] == nil {
				t.Fatal("the effect was not reported as uncertain", got.raw)
			}
			moveRevision(t, f)
			replay := f.call("issuance_setup."+call.kind, body, call.who.cookie, call.who.csrf)
			if replay.status != 500 || replay.body["operation_id"] != got.body["operation_id"] || replay.body["code"] == string(commands.RevisionConflict) {
				t.Fatal("the exact retry of an applied write was not replayed", replay.raw)
			}
		})
	}
}

func TestCommandRetentionCoversTheNotRecordedBound(t *testing.T) {
	const bound = 30 * time.Minute
	if l := CommandLimits(); l.Retention < 2*bound || l.Replay < bound {
		t.Fatalf("retention %s or replay %s no longer covers the %s bound of the console's not-recorded rule", l.Retention, l.Replay, bound)
	}
}
