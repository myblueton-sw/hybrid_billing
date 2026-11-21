package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/documentauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func documentGrantTerms(action gate.Action, kind string, fields []string) bool {
	switch action {
	case gate.ProposeDocumentEvidence, gate.ConfirmDocumentEvidence, gate.RevokeDocumentEvidence, gate.PrepareERPRecord:
		return true
	}
	if kind == documentauthority.ResourceKind {
		return true
	}
	for _, field := range fields {
		if field == documentauthority.EvidenceField {
			return true
		}
	}
	return false
}
