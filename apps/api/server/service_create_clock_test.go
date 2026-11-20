package server

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func verifyCreateClockOutcome(t *testing.T, d createBoundaryFixture, outcome commands.Outcome, err error, jumped bool) {
	t.Helper()
	if jumped {
		var failure *commands.Failure
		if !errors.As(err, &failure) || (failure.Code != commands.PermissionDenied && failure.Code != commands.DependencyUnavailable) || !reflect.DeepEqual(outcome, commands.Outcome{}) {
			t.Fatal("observed create expiry did not refuse without an outcome")
		}
		d.observe(false)
		return
	}
	if err != nil || outcome.BusinessState != "service_created" || outcome.ResultRevision != d.proposed || !reflect.DeepEqual(outcome.Fields, map[string]string{"declaration_version": "1"}) {
		t.Fatal("healthy direct create lost its exact outcome and proposed revision")
	}
	d.observe(true)
}

func TestServiceCreateRefusesExpiryAtQuiescedClockBoundaries(t *testing.T) {
	control := newCreateBoundaryFixture(t)
	outcome, err, horizon, jumped := control.apply(0)
	if jumped || horizon == 0 {
		t.Fatal("healthy create observation horizon premise")
	}
	verifyCreateClockOutcome(t, control, outcome, err, false)
	refused, healthy := 0, 0
	for cutoff := 1; cutoff <= horizon+1; cutoff++ {
		t.Run(fmt.Sprintf("boundary-%d", cutoff), func(t *testing.T) {
			d := newCreateBoundaryFixture(t)
			outcome, err, _, jumped := d.apply(cutoff)
			verifyCreateClockOutcome(t, d, outcome, err, jumped)
			if jumped {
				refused++
			} else {
				healthy++
			}
		})
	}
	if refused == 0 || healthy == 0 {
		t.Fatal("create clock sweep did not exercise observed expiry and no-jump success")
	}
}
