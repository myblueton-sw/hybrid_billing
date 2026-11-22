package server

import (
	"errors"
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

var errFinancialDraftUncertain = errors.New("draft operation may have changed retained or module state")

type financialDraftEffect struct {
	retainedChanged, moduleAttempted, observedPublicationChanged bool
}

func (e *financialDraftEffect) changed() bool {
	return e != nil && (e.retainedChanged || e.moduleAttempted || e.observedPublicationChanged)
}

func (e *financialDraftEffect) attempt() {
	if e != nil {
		e.moduleAttempted = true
	}
}

func (a *financialIssuance) storeDraft(p financialDraftRecord, e *financialDraftEffect, publication bool) {
	old, exists := a.drafts[p.Spec.ID]
	if exists && reflect.DeepEqual(old, p) {
		return
	}
	a.drafts[p.Spec.ID] = copyFinancialDraft(p)
	if e != nil {
		e.retainedChanged = true
		e.observedPublicationChanged = e.observedPublicationChanged || publication
	}
}

func (a *financialIssuance) storeDraftReview(r financialDraftReview, e *financialDraftEffect, publication bool) {
	old, exists := a.draftReviews[r.ID]
	if exists && reflect.DeepEqual(old, r) {
		return
	}
	a.draftReviews[r.ID] = copyDraftReview(r)
	if e != nil {
		e.retainedChanged = true
		e.observedPublicationChanged = e.observedPublicationChanged || publication
	}
}

func draftPublicError(err error, effect *financialDraftEffect) error {
	if effect.changed() {
		return errFinancialDraftUncertain
	}
	var failure *commands.Failure
	if errors.As(err, &failure) {
		return err
	}

	switch {
	case errors.Is(err, issuance.ErrInvalid):
		return commands.Fail(commands.InvalidRequest)
	case errors.Is(err, issuance.ErrConflict), errors.Is(err, issuance.ErrNotFound):
		return commands.Fail(commands.EvidenceStale)
	case errors.Is(err, issuance.ErrLimit), errors.Is(err, protected.ErrNotFound), errors.Is(err, protected.ErrRejected), errors.Is(err, receivables.ErrRejected):
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return commands.Fail(commands.DependencyUnavailable)
}
