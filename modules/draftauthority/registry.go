package draftauthority

import (
	"github.com/myblueton-sw/hybrid_billing/modules/internal/grantauthority"
	"time"
)

func NewRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Version: "synthetic-issuance-draft-authority-v1", GrantPrefix: "issuance-draft-grant-", Capabilities: []string{Submit, Approve, Inspect}}, limits, authority, targets, effects, clock)
}
