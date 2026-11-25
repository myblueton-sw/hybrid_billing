package documentauthority

import (
	"github.com/myblueton-sw/hybrid_billing/modules/internal/grantauthority"
	"time"
)

func NewRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Version: "synthetic-document-authority-v1", GrantPrefix: "document-grant-", Capabilities: []string{Propose, Confirm, Revoke, Prepare, Inspect}}, limits, authority, targets, effects, clock)
}
