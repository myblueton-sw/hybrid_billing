package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

type setupReferenceEntry struct{ Code, Use string }

var setupReferenceCatalog = []setupReferenceEntry{
	{"synthetic_recipient_a", protected.FieldDocumentRecipient}, {"synthetic_recipient_b", protected.FieldDocumentRecipient},
	{"synthetic_template_a", protected.FieldDocumentTemplate}, {"synthetic_template_b", protected.FieldDocumentTemplate},
	{"synthetic_account_a", protected.FieldPaymentAccount}, {"synthetic_account_b", protected.FieldPaymentAccount},
	{"synthetic_channel_customer_a", "channel_customer"}, {"synthetic_channel_customer_b", "channel_customer"},
	{"synthetic_channel_document_kind_a", "channel_document_kind"}, {"synthetic_channel_document_kind_b", "channel_document_kind"},
	{"synthetic_channel_entity_a", "channel_entity"}, {"synthetic_channel_entity_b", "channel_entity"},
	{"synthetic_unregistered_sender", "unregistered_sender"},
}

func setupReference(code, use string) (protected.Ref, bool) {
	for _, entry := range setupReferenceCatalog {
		if entry.Code == code && entry.Use == use {
			id := "synthetic-reference:" + code
			sum := sha256.Sum256([]byte(id))
			return protected.Ref{ID: id, Fingerprint: hex.EncodeToString(sum[:])}, true
		}
	}
	return protected.Ref{}, false
}

func (a *financialIssuance) setupReferenceFields(workspace, actor, contract string, input map[string]string) (map[string]string, error) {
	c, err := a.protectedSetupConfiguration(workspace, actor, contract, "references", input)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{"schema": "hb_issuance_setup_references_v1", "contract": contract, "configuration": c.Spec.ID, "digest": c.Digest, "current_eligibility": "unchecked"}
	for i, entry := range setupReferenceCatalog {
		raw, err := draftJSON(map[string]string{"code": entry.Code, "use": entry.Use, "applicability": "synthetic_only"})
		if err != nil {
			return nil, err
		}
		fields[fmt.Sprintf("reference_%02d", i)] = raw
	}
	if err := a.boundedDraftFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
