package protectedauthority

import (
	"github.com/myblueton-sw/hybrid_billing/modules/internal/grantauthority"
	"time"
)

func NewRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Version: "synthetic-issuance-protected-authority-v1", GrantPrefix: "issuance-protected-grant-", Capabilities: []string{RegisterChannel, AdmitChannel, RequestChange, VerifyChange, ApproveChange, Inspect}}, limits, authority, targets, effects, clock)
}
