package identity

import (
	"testing"
	"time"
)

func lastErr[A, B any](_ A, _ B, err error) error { return err }

func TestPeekAdmitsLikeAccessWithoutTouchingTheIdleEnd(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	session := f.login("ann")
	f.at(start.Add(20 * time.Minute))
	answer, idle, err := f.b.Peek(session, "w1")
	f.must(err)
	if !idle.Equal(start.Add(30 * time.Minute)) {
		t.Errorf("idle end %v", idle)
	}
	f.must(f.b.Recheck(answer))
	if answer.User() != "ann" || answer.Workspace() != "w1" || !answer.NotAfter().Equal(start.Add(8*time.Hour)) {
		t.Errorf("answer %+v", answer)
	}
	f.at(start.Add(29 * time.Minute))
	if _, again, err := f.b.Peek(session, "w1"); err != nil || !again.Equal(idle) {
		t.Errorf("a second peek moved the idle end: %v %v", again, err)
	}
	f.at(start.Add(30 * time.Minute))
	f.refusal("peek at the idle end", lastErr(f.b.Peek(session, "w1")), ReasonSession)
	f.refusal("access at the idle end", second(f.b.Access(session, "w1")), ReasonSession)

	touched := f.login("ann")
	f.at(start.Add(50 * time.Minute))
	f.must(second(f.b.Access(touched, "w1")))
	if _, extended, err := f.b.Peek(touched, "w1"); err != nil || !extended.Equal(start.Add(80*time.Minute)) {
		t.Errorf("peek after access: %v %v", extended, err)
	}

	f.want("an invalid workspace", lastErr(f.b.Peek(touched, "")), ErrInvalid)
	f.refusal("no membership", lastErr(f.b.Peek(touched, "w2")), ReasonMembership)
	until := start.Add(60 * time.Minute)
	f.bounded("bob", Member, until)
	bob := f.login("bob")
	f.at(until)
	f.refusal("after the role period", lastErr(f.b.Peek(bob, "w1")), ReasonExpired)
	var missing *Boundary
	f.want("no registry", lastErr(missing.Peek(bob, "w1")), ErrUnavailable)
}

func TestPeekCapsTheIdleEndAtTheAbsoluteEnd(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	session := v("short")
	f.must(f.b.OpenSession(triple("ann"), session, 30*time.Minute, 40*time.Minute))
	f.at(start.Add(20 * time.Minute))
	f.must(second(f.b.Access(session, "w1")))
	_, idle, err := f.b.Peek(session, "w1")
	if err != nil || !idle.Equal(start.Add(40*time.Minute)) {
		t.Errorf("idle end %v %v", idle, err)
	}
}
