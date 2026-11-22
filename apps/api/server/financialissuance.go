package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

const financialIssuanceLimit = 100

type financialIssuance struct {
	drafts           map[string]financialDraftRecord
	draftScopes      map[string]string
	changes          map[string]financialChangeRecord
	changeOrders     map[string][]string
	channels         map[string]financialChannelRecord
	kindReservations map[string]financialKindReservation
	protectedRules   *financialProtectedRules
	termIntents      map[string]financialTermRecord
	termScopes       map[string]string
	bindingIntents   map[string]financialTermBinding
	server           *Server
	fields           *protected.Registry
	terms            *receivables.Registry
	documents        *issuance.Registry
	profiles         map[string]financialIssuanceProfile
	scopes           map[string]string
}

type unavailableIssuanceContent struct{}

func (unavailableIssuanceContent) Detail(string, string) (issuance.DetailFacts, error) {
	return issuance.DetailFacts{}, commands.Fail(commands.UnsupportedCapability)
}
func (unavailableIssuanceContent) Render(issuance.Snapshot) ([]byte, error) {
	return nil, commands.Fail(commands.UnsupportedCapability)
}
