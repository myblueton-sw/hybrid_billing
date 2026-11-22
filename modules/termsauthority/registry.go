package termsauthority

import (
	"github.com/myblueton-sw/hybrid_billing/modules/internal/grantauthority"
	"time"
)

func NewRegistry(limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	return grantauthority.NewRegistry(grantauthority.Domain{Version: "synthetic-issuance-term-authority-v1", GrantPrefix: "issuance-term-grant-", Capabilities: []string{RecordTerm, ApproveTerm, BindTerm, ApproveBinding}}, limits, authority, targets, effects, clock)
}
