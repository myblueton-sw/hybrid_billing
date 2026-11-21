---
type: ImplementationPlan
title: Synthetic ERP result evidence and complete reconciliation
description: Advance the planned financial stage chain through coherent result evidence before final ERP approval.
status: draft
---

# Ownership and original outcome

[HYB-341](https://linear.app/myblueton/issue/HYB-341) continues [HYB-225](../HYB-225/plan.md) after [HYB-340 document preparation](../HYB-340/document-screen-verification.md). Root owns planning, implementation and integration. Seung Woo Park is accountable; machine:mac_mini is the sole claim. Independent refusal_review owns read-only financial/security review; platform_refusals owns read-only evidence review. Branch feat/HYB-341-erp-result in /private/tmp/hyb341 is explicitly stacked on unmerged HYB-340 commit 3454c36, not created from current main. Existing absent companion inputs remain absent under the owner's no-restoration instruction.

The full outcome is governed explicitly supplied synthetic ERP-result evidence, complete exact reconciliation and subsequent StageERPFinal integration. Preparation alone proves neither an ERP draft nor a final gross amount. No production tax/accounting policy, external confirmation, submit/post operation, approval consumption, issuance, delivery or durable persistence is adopted. The HB-257 synthetic in-memory boundary remains in force. No merge or deployment is authorized.

# Producer discovery and first prerequisite

The actual erpledger registry supplies Document, Link, DocumentOf and Reconcile. Reconcile.Matched covers submission lines; its document attributed totals include other live submissions and do not prove exclusive complete ownership. It also omits document revisions, row product identity and individual stale links. Independent result evidence must retain these facts instead of converting a boolean into approval readiness.

Separate calls to SubmissionOf, DocumentOf and Reconcile acquire separate registry locks. A caller without exclusive ownership of every module writer cannot combine them into a coherent snapshot. The first implementation therefore adds a read-only ResultOf(workspace, key) snapshot on the actual registry. Under one existing mutex/clock entry it returns the defensive SubmissionView, Reconciliation and the full DocumentView for every document with any retained link to this submission, including stale links. Document ordering follows connection/external identity; line and row ordering preserve the owning records. Missing, malformed and foreign workspace/key identities return ErrNotFound, following find without adding ErrInvalid validation. Output is bounded by configured retained records, not a small HTTP limit, and is never truncated. No record, revision, link, command, watch or approval is changed; normal monotonic read clock advancement remains possible.

The result type is an internal module evidence product, not a public HTTP schema or final financial eligibility certificate. Reconcile remains backward compatible. Factor its locked calculation into a private helper so ResultOf computes the same result without nested locking. Return owned copies of all nested slices and pointers through existing copy routines. Do not synthesize a document for an unlinked preparation or hide incomplete/stale links. The snapshot includes complete document links, including other preparations, for the later validator to reject unsupported mixed attribution.

Root may edit modules/erpledger/types.go, documents.go, a small result read implementation/test file and this ticket's records. No service command, new permission, UI, external action or financial policy is introduced by this prerequisite. Independent contract review must pass before implementation.

# First acceptance

Use actual registry preparation, independently stated document rows and actual links. Verify missing/foreign/malformed identity, an unlinked preparation, complete and partial attribution, stale links after a changed document revision, another submission sharing a document, exact retained products/revisions, deterministic document order, defensive copies of submission/commands/watch/resolution/document rows/links/reconciliation slices, and unchanged module state apart from the read clock. Concurrent document replacement and ResultOf must return internally consistent document totals/revisions/rows and reconciliation values. Preserve existing Reconcile behavior. Run focused and module race tests, vet and source rules as appropriate; record failures and corrective review honestly.

# Subsequent integration remains required

Next independently review a bounded immutable result packet with independently supplied source bytes/digest, synthetic mode, connection, external identity/revision, rows and explicit allocation to one owned preparation. Do not generate actual-result amounts by copying preparation. Validate exact product/line identity, complete row and document coverage, no unexpected scopes or stale links, and current version provenance in addition to Reconcile.Matched.

Separate result recording/confirmation authority and independent confirmation must be designed before public ingestion; preparation/financial approval rights do not grant result entry. Document plus multiple Link mutations are non-atomic and need ownership-first admission and exact interrupted-operation reconciliation. Historical snapshots must not substitute for current charge, document, result or authority checks. StageERPFinal must use the existing approvals registry only after these producers exist. Completion of ResultOf does not complete HYB-341, W-23 or the owner goal.

# Independent prerequisite design review

refusal_review passed the scoped ResultOf design before implementation, requiring one registry lock, existing reconciliation ordering, all stale/mixed links, defensive copies, the actual ErrNotFound contract and explicit retained-record output bounds. It required an equal-total changed-product regression and a concurrent test asserting within-snapshot coherence rather than thread ordering. This review did not authorize result intake or final approval.
