package pricingauthority

import (
	"github.com/myblueton-sw/hybrid_billing/modules/internal/grantauthority"
	"time"
)

func NewRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Version: "synthetic-pricing-authority-v1", GrantPrefix: "pricing-grant-", Capabilities: []string{Propose, Approve, Revoke, Use, Inspect}}, limits, authority, targets, effects, clock)
}
