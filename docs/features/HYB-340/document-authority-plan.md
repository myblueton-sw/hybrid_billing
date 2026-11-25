---
type: ImplementationPlan
title: Dedicated synthetic document and ERP preparation authority
description: Establish explicit nondelegable user and console capabilities before service evidence governance and preparation.
status: draft
---

# Outcome and ownership

HYB-340, machine:mac_mini. Root owns implementation; refusal_review owns independent financial/security review and platform_refusals reviews evidence. The previous internal adapter checkpoint is d216d33. The runtime integration needs separate authority to govern explicit document evidence and prepare a record. Financial Approve, Issue and installation administration do not supply these rights. Preserve the complete W-23 objective; this prerequisite does not complete runtime or screen integration.

# Closed authority contract

Add direct-human nondelegable actions ProposeDocumentEvidence=propose_document_evidence, ConfirmDocumentEvidence=confirm_document_evidence, RevokeDocumentEvidence=revoke_document_evidence and PrepareERPRecord=prepare_erp_record. Ordinary user-plus-console intersection applies; do not change the financial humanOnly exception. Service principals, client-only execution, delegation and support sessions cannot exercise them. Grant expiry, membership and current-state recheck retain existing auth rules.

A fixed documentauthority wrapper over internal/grantauthority permits propose, confirm, revoke, prepare and inspect only, using version synthetic-document-authority-v1 and grant prefix document-grant-. ResourceKind is synthetic_document_evidence and EvidenceField is document_evidence. Future inspect uses ViewCost on that dedicated field with explicit actual-human/nondelegation checks. No implied read authority follows proposal, confirmation or preparation.

Generic grant additions must exclude each dedicated action, kind and field at catalog selection, current request validation and final application through the existing governedGrantTerms guard. Preserve explicit individual/bulk revocation. This checkpoint does not add the kind/field to the public server catalog, instantiate a provisioning domain, create default grants or expose commands. Runtime provisioning and protected Cost catalog registration follow together with screen-label validation.

# Owned implementation and verification

Root may edit modules/auth/types.go and scoped authority tests, modules/gate/types.go, new modules/documentauthority, a narrow server governed-grant guard and tests, and HYB-340 records. Reuse the existing tested grant engine, without changing its financial/input/pricing domains. Verify closed capability set, two-governor separation/current proposer checks, deterministic exact replay, distinct digests and effect IDs across domains, and no default effects. Exercise the actual auth evaluator for all four actions, including human/console isolation, revocation, expiry, scope, service/delegation/support refusal and unchanged financial approval behavior. Exercise actual generic request/current/final application refusal for dedicated actions plus kind/field classification, preserving revocation.

# Following runtime contract decisions

Document packets will reference owned charge/candidate IDs; server derives candidate facts and actual ERP snapshots. Explicit line product, quantity/unit and tax evidence remain mandatory. A different current authorized human confirms the exact immutable packet; confirmation is evidence acceptance, not approval of invoice gross or tax policy. Revocation stops new preparation while retaining prepared history. Preparation may be performed by the packet proposer if independently granted prepare authority; only proposer/confirmer separation is required at this stage. No financial permission is inferred.

Before that implementation, independently review bounded strict packet input under existing command field limits, allowlisted Cost inspection, current proposer checks, terminal confirmation/revocation replay, retained-history read semantics after revocation, hold/step-up/revision/idempotency behavior and output bounds. Actual runtime owns one named synthetic connection/mapping/survey fixture without vendor verification. The public action must authenticate and authorize before lookup and preserve current checks before owned preparation recovery. No send/post, issuance or approval consumption is part of internal preparation.

# Independent design review

refusal_review passed this prerequisite contract before implementation. Tests use an explicit test-only catalog with valid dedicated kind/field grants to prove generic-add refusal is enforced independently of unknown-identifier validation. Runtime governance and disclosure remain separate review gates.
