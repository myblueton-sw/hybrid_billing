package financialpermission

import (
	"github.com/myblueton-sw/hybrid_billing/modules/internal/grantauthority"
	"time"
)

func NewRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Mode: grantauthority.DirectHumanMode, Version: "synthetic-financial-permission-v1", GrantPrefix: "financial-permission-", Capabilities: []string{Approve}}, limits, authority, targets, effects, clock)
}

func NewERPFinalRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Mode: grantauthority.DirectHumanMode, Version: "synthetic-erp-final-permission-v1", GrantPrefix: "erp-final-permission-", Capabilities: []string{Approve}}, limits, authority, targets, effects, clock)
}
