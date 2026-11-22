package financialrequestauthority

import (
	"github.com/myblueton-sw/hybrid_billing/modules/internal/grantauthority"
	"time"
)

func NewRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Version: "synthetic-financial-request-authority-v1", GrantPrefix: "financial-request-", Capabilities: []string{Request}}, limits, authority, targets, effects, clock)
}

func NewERPFinalRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Version: "synthetic-erp-final-request-authority-v1", GrantPrefix: "erp-final-request-", Capabilities: []string{Request}}, limits, authority, targets, effects, clock)
}
