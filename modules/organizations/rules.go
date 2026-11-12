package organizations

func RequiresExternalInvoice(profile BillingProfile) (bool, error) {
	switch profile.Path {
	case ExternalInvoice:
		return true, nil
	case InternalChargeback:
		return false, nil
	}
	return false, ErrInvalid
}

func SellingEligibility(revision ContractRevision, componentID, issuerID string) Decision {
	if (revision.Path != Direct && revision.Path != Resale) || componentID == "" ||
		(revision.SellerID != "" && revision.SellerID == revision.CustomerID) {
		return Decision{Reason: InvalidContract}
	}
	matches := 0
	for _, component := range revision.Components {
		if component.ID == componentID {
			matches++
		}
	}
	if matches > 1 {
		return Decision{Reason: InvalidContract}
	}
	if revision.SellerID == "" || issuerID != revision.SellerID {
		return Decision{Reason: NotSeller}
	}
	for _, component := range revision.Components {
		if component.ID != componentID {
			continue
		}
		switch component.Kind {
		case ProviderCharge:
			if component.ProviderPaymentResponsibleID == "" {
				return Decision{Reason: InvalidContract}
			}
			if revision.Path == Direct || component.ProviderPaymentResponsibleID != issuerID {
				return Decision{Reason: NoPaymentResponsibility}
			}
		case SellerService:
			if component.ProviderPaymentResponsibleID != "" {
				return Decision{Reason: InvalidContract}
			}
		default:
			return Decision{Reason: InvalidContract}
		}
		if !component.SellerBillable {
			return Decision{Reason: OutsideEligibleScope}
		}
		return Decision{Eligible: true, Reason: EligiblePendingApproval}
	}
	return Decision{Reason: UnknownComponent}
}

func completeKey(key DocumentKey) bool {
	if key.WorkspaceID == "" || key.IssuerLegalEntityID == "" || key.BuyerLegalEntityID == "" || key.Currency == "" {
		return false
	}
	switch key.Path {
	case ExternalInvoice:
		return key.PayerID != ""
	case InternalChargeback:
		return key.PayerID == ""
	}
	return false
}

func SameDocument(keys ...DocumentKey) error {
	if len(keys) == 0 {
		return ErrMixedDocument
	}
	for _, key := range keys {
		if key != keys[0] || !completeKey(key) {
			return ErrMixedDocument
		}
	}
	return nil
}
