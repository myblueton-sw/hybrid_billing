package server

import (
	"fmt"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func verifyDetailClockReply(t *testing.T, d detailBoundaryFixture, got reply, jumped bool) {
	t.Helper()
	if !jumped {
		id := d.service
		if d.kind == "credential" {
			id = d.credential
		}
		row, known := got.body["row"].(map[string]any)
		if got.status != 200 || !known || row["id"] != id || got.body["workspace"] != SampleWorkspace {
			t.Fatal("unobserved detail clock step lost its healthy actual row", got.status)
		}
		return
	}
	code := commands.Unauthenticated
	switch got.status {
	case 401:
	case 403:
		code = commands.PermissionDenied
	case 503:
		code = commands.DependencyUnavailable
	default:
		t.Fatal("observed detail expiry published data or another failure class", got.status)
	}
	assertServiceQueryFailurePayload(t, got, got.status, code)
}

func verifyDetailClockBoundaries(t *testing.T, kind string) {
	t.Helper()
	control := newDetailBoundaryFixture(t, kind)
	control.clock.arm(0)
	got, horizon, jumped := control.read()
	if jumped || horizon == 0 {
		t.Fatal("healthy quiesced observation horizon premise")
	}
	verifyDetailClockReply(t, control, got, false)
	control.unchanged()
	refused := 0
	for cutoff := 1; cutoff <= horizon+1; cutoff++ {
		t.Run(fmt.Sprintf("boundary-%d", cutoff), func(t *testing.T) {
			d := newDetailBoundaryFixture(t, kind)
			d.clock.arm(cutoff)
			got, _, jumped := d.read()
			verifyDetailClockReply(t, d, got, jumped)
			if jumped {
				refused++
			}
			d.unchanged()
		})
	}
	if refused == 0 {
		t.Fatal("detail clock sweep did not exercise an observed expiry")
	}
}

func TestServiceAccountDetailRefusesExpiryAtQuiescedClockBoundaries(t *testing.T) {
	verifyDetailClockBoundaries(t, "account")
}

func TestServiceCredentialDetailRefusesExpiryAtQuiescedClockBoundaries(t *testing.T) {
	verifyDetailClockBoundaries(t, "credential")
}
