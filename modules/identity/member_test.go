package identity

import (
	"errors"
	"testing"
	"time"
)

func TestOperatorMemberRead(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.bounded("bob", Administrator, start.Add(time.Hour))
	for name, c := range map[string]struct {
		operator  Operator
		workspace string
		user      string
	}{"operator": {Operator{}, "w1", "ann"}, "workspace": {operator, "", "ann"}, "user": {operator, "w1", ""}} {
		if _, found, err := f.in.Member(c.operator, c.workspace, c.user); !errors.Is(err, ErrInvalid) || found {
			t.Errorf("an invalid %s: %v", name, err)
		}
	}
	got, found, err := f.in.Member(operator, "w1", "bob")
	if err != nil || !found || got.User != "bob" || got.Role != Administrator || !got.Effective || !got.Until.Equal(start.Add(time.Hour)) || got.UserState != Active {
		t.Errorf("bob %+v %v %v", got, found, err)
	}
	list, _ := f.in.Members(operator, "w1")
	for _, m := range list {
		one, ok, err := f.in.Member(operator, "w1", m.User)
		if err != nil || !ok || one != m {
			t.Errorf("%s differs: %+v %+v", m.User, one, m)
		}
	}
	if _, found, err := f.in.Member(operator, "w1", "nobody"); found || err != nil {
		t.Errorf("an unknown user %v %v", found, err)
	}
	if _, found, _ := f.in.Member(operator, "w2", "ann"); found {
		t.Error("another workspace")
	}
	var r Registry
	if _, found, err := r.Installation().Member(operator, "w1", "ann"); !errors.Is(err, ErrUnavailable) || found {
		t.Error("an unconstructed registry")
	}
}
