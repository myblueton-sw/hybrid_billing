package licensemode

import (
	"testing"
	"time"
)

func status(t testing.TB, f world, id string) StatusView {
	t.Helper()
	v, err := f.r.Status(id)
	f.must(err)
	return v
}

func TestLM01Noncommercial(t *testing.T) {
	f := standard(t)
	if v := status(t, f, "i1"); v.State != StateUnselected || v.Warning {
		t.Errorf("unselected %+v", v)
	}
	f.must(f.r.SelectMode("i1", ModeNoncommercial, ann))
	f.advance(15 * 24 * time.Hour)
	f.auth.set(time.Time{}, false)
	if v := status(t, f, "i1"); v.State != StateNoncommercial || v.Warning || v.Remaining != 0 || len(v.Actions) != 2 {
		t.Errorf("noncommercial %+v", v)
	}
}

func TestLM02EvaluationNeedsGrant(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeEvaluation, ann))
	if v := status(t, f, "i1"); v.State != StateEvaluationPending || v.Activation != nil {
		t.Errorf("pending %+v", v)
	}
	reason(t, "terms", second(f.r.Activate("i1", f.sign(evaluation("g1")), Acceptance{Actor: "ann", Terms: "other-terms"}, ann)), ReasonTermsMismatch)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	reason(t, "mode", second(f.r.Activate("i1", f.sign(evaluation("g1")), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)), ReasonModeMismatch)
	if v := status(t, f, "i1"); v.State != StateCommercialPending || len(f.r.grants) != 0 {
		t.Errorf("no trial %+v", v)
	}
}

func TestLM03EvaluationEdges(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeEvaluation, ann))
	a, err := f.r.Activate("i1", f.sign(evaluation("g1")), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)
	if err != nil || !a.Start.Equal(start) || !a.End.Equal(start.Add(336*time.Hour)) || a.AcceptedBy != "ann" || a.Grant != "g1" {
		t.Fatalf("activate %+v %v", a, err)
	}
	f.advance(336*time.Hour - time.Second)
	if v := status(t, f, "i1"); v.State != StateEvaluationActive || v.Remaining != time.Second || v.Warning {
		t.Errorf("before end %+v", v)
	}
	f.advance(time.Second)
	if v := status(t, f, "i1"); v.State != StateEvaluationExpired || !v.Warning || v.Actions[0] != ActionRegister || v.Actions[1] != ActionContact {
		t.Errorf("at end %+v", v)
	}
}

func TestLM04NoReset(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeEvaluation, ann))
	grant := f.sign(evaluation("g1"))
	f.must(second(f.r.Activate("i1", grant, Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)))
	f.advance(400 * time.Hour)
	again, err := f.r.Activate("i1", grant, Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)
	if err != nil || !again.Start.Equal(start) {
		t.Errorf("repeat %+v %v", again, err)
	}
	other := evaluation("g2")
	reason(t, "second grant", second(f.r.Activate("i1", f.sign(other), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)), ReasonAlreadyEvaluated)
	f.must(f.r.SelectMode("i1", ModeNoncommercial, ann))
	f.must(f.r.SelectMode("i1", ModeEvaluation, ann))
	if v := status(t, f, "i1"); v.State != StateEvaluationExpired {
		t.Errorf("mode switch reset %+v", v)
	}
	f.auth.set(start.Add(time.Hour), true)
	if v := status(t, f, "i1"); v.State != StateUnverified || v.Reason != OutcomeClockAnomaly || v.LastVerified != StateEvaluationExpired {
		t.Errorf("rollback %+v", v)
	}
	f.approve(Change{Kind: ChangeReset, Reset: Reset{At: start.Add(time.Hour).Unix(), Reason: "wrong source"}})
	if v := status(t, f, "i1"); v.State != StateEvaluationExpired {
		t.Errorf("latched after reset %+v", v)
	}
	reason(t, "reset too early", f.try(Change{Kind: ChangeReset, Reset: Reset{At: start.Add(-time.Hour).Unix(), Reason: "x"}}), ReasonResetTooEarly)
}

func TestLM05Purchased(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeEvaluation, ann))
	f.must(second(f.r.Activate("i1", f.sign(evaluation("g1")), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)))
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	if v := status(t, f, "i1"); v.State != StateCommercialPending {
		t.Errorf("pending %+v", v)
	}
	out, err := f.r.Register("i1", f.sign(commercial("e1", "agr-1", start.Add(-time.Hour), start.Add(365*24*time.Hour))), ann)
	if err != nil || out != OutcomeAccepted {
		t.Fatalf("register %s %v", out, err)
	}
	v := status(t, f, "i1")
	if v.State != StateCommercialActive || v.Entitlement == nil || v.Entitlement.Subject != "Acme Ltd" || v.Entitlement.Agreement != "agr-1" || v.Warning {
		t.Errorf("active %+v", v)
	}
	modes, subs, err := f.r.History("i1")
	if err != nil || len(modes) != 2 || len(subs) != 1 || subs[0].Outcome != OutcomeAccepted || subs[0].Digest == "" {
		t.Errorf("history %+v %+v", modes, subs)
	}
}

func TestLM06DistinctOutcomes(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	good := f.sign(commercial("e1", "agr-1", start.Add(-time.Hour), start.Add(24*time.Hour)))
	f.must(second(f.r.Register("i1", good, ann)))
	altered := f.sign(commercial("e2", "agr-2", start.Add(-time.Hour), start.Add(24*time.Hour)))
	altered.Body[len(altered.Body)-20] ^= 1
	c := commercial("e3", "agr-3", start.Add(-time.Hour), start.Add(24*time.Hour))
	c.KeyID = "k9"
	unknown := Signed{Body: Encode(c), Signature: make([]byte, 64)}
	wrong := commercial("e4", "agr-4", start.Add(-time.Hour), start.Add(24*time.Hour))
	wrong.Installation = "i2"
	cases := map[string]Signed{
		OutcomeInvalid:       altered,
		OutcomeUnverifiable:  unknown,
		OutcomeScopeMismatch: f.sign(wrong),
		OutcomeWrongKind:     f.sign(evaluation("g1")),
	}
	for want, doc := range cases {
		if out, err := f.r.Register("i1", doc, ann); err != nil || out != want {
			t.Errorf("%s got %s %v", want, out, err)
		}
	}
	if v := status(t, f, "i1"); v.State != StateCommercialActive || v.Entitlement.ID != "e1" || cases[v.LastOutcome].Body == nil {
		t.Errorf("valid kept %+v", v)
	}
	_, subs, _ := f.r.History("i1")
	if len(subs) != 5 {
		t.Errorf("submissions %d", len(subs))
	}
}

func TestLM07ExpiredVersusUnavailable(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	f.must(second(f.r.Register("i1", f.sign(commercial("e1", "agr-1", start.Add(-time.Hour), start.Add(time.Hour))), ann)))
	status(t, f, "i1")
	f.auth.set(time.Time{}, false)
	if v := status(t, f, "i1"); v.State != StateUnverified || v.Reason != OutcomeTimeUnavailable || v.LastVerified != StateCommercialActive || !v.LastVerifiedAt.Equal(start) {
		t.Errorf("unavailable %+v", v)
	}
	f.advance(time.Hour)
	if v := status(t, f, "i1"); v.State != StateCommercialExpired || !v.Warning {
		t.Errorf("expired %+v", v)
	}
	if out, _ := f.r.Register("i1", f.sign(commercial("e0", "agr-0", start.Add(-2*time.Hour), start.Add(time.Hour))), ann); out != OutcomeExpired {
		t.Errorf("expired registration %s", out)
	}
	f.auth.set(time.Time{}, false)
	if out, _ := f.r.Register("i1", f.sign(commercial("e2", "agr-2", start, start.Add(48*time.Hour))), ann); out != OutcomeTimeUnavailable {
		t.Errorf("unavailable registration %s", out)
	}
}

func TestLM08HistoryAfterExpiry(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeEvaluation, ann))
	f.must(second(f.r.Activate("i1", f.sign(evaluation("g1")), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)))
	f.advance(400 * time.Hour)
	v := status(t, f, "i1")
	modes, _, err := f.r.History("i1")
	if v.State != StateEvaluationExpired || err != nil || len(modes) != 1 || v.Activation == nil {
		t.Errorf("history %+v %v", v, err)
	}
}

func TestLM09NoInference(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	if v := status(t, f, "i1"); v.State != StateCommercialPending || v.Entitlement != nil {
		t.Errorf("commercial mode grants nothing %+v", v)
	}
	expect(t, "unknown mode", f.r.SelectMode("i1", "enterprise", ann), ErrInvalid)
}

func TestLM10Offline(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	f.advance(400 * 24 * time.Hour)
	c := commercial("e1", "agr-1", start, start.Add(800*24*time.Hour))
	if out, _ := f.r.Register("i1", f.sign(c), ann); out != OutcomeAccepted {
		t.Errorf("offline import after signing window %s", out)
	}
	f.advance(400 * 24 * time.Hour)
	c2 := commercial("e2", "agr-2", start, start.Add(900*24*time.Hour))
	if out, _ := f.r.Register("i1", f.sign(c2), ann); out != OutcomeUnverifiable {
		t.Errorf("after acceptance window %s", out)
	}
}

func TestLM11NoSideEffects(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeEvaluation, ann))
	f.auth.set(time.Time{}, false)
	reason(t, "activate unavailable", second(f.r.Activate("i1", f.sign(evaluation("g1")), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)), OutcomeTimeUnavailable)
	if len(f.r.grants) != 0 {
		t.Error("no start invented")
	}
	if v := status(t, f, "i1"); v.State != StateUnverified || v.Reason != OutcomeTimeUnavailable || !v.Warning {
		t.Errorf("unverified %+v", v)
	}
}
