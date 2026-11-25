package issuance

import "github.com/myblueton-sw/hybrid_billing/modules/receivables"

type reviewedApproval struct {
	references []Reference
	coverage   *receivables.Coverage
}

func (r *Registry) ApproveReviewed(workspace, id, digest string, expectedRefs []Reference, expectedCoverage *receivables.Coverage, actor Actor) (DraftView, error) {
	reviewed := reviewedApproval{references: append([]Reference(nil), expectedRefs...)}
	if expectedCoverage != nil {
		c := *expectedCoverage
		reviewed.coverage = &c
	}
	return r.approve(workspace, id, digest, actor, &reviewed)
}

func (v reviewedApproval) matches(refs []Reference, coverage *receivables.Coverage) error {
	if !sameReferences(v.references, refs) {
		return refuse(ReasonReferenceChanged)
	}
	if v.coverage == nil || coverage == nil {
		if v.coverage != coverage {
			return refuse(ReasonCoverageMismatch)
		}
	} else if *v.coverage != *coverage {
		return refuse(ReasonCoverageMismatch)
	}
	return nil
}
