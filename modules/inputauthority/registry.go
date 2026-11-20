package inputauthority

import (
	"github.com/myblueton-sw/hybrid_billing/modules/internal/grantauthority"
	"time"
)

func NewRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Version: "synthetic-input-authority-v1", GrantPrefix: "input-grant-", Capabilities: []string{Confirm, Inspect}}, limits, authority, targets, effects, clock)
}
