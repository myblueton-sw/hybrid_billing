---
type: ImplementationPlan
title: Current pricing evidence for financial subjects
description: Read-only evidence validation without granting pricing-use or financial decision permission.
status: draft
---

# Scope and independent design

Continue HYB-340 aftera7a6651 on the same ticket branch and machine:mac_mini. Root owns modules/pricingpolicy/evidence.go, its focused tests, the narrow registry.go extraction and this evidence. Independent refusal_review accepted this design before code. No server route, UI, grant or financial subject/request is added by this prerequisite.

A financial approver need not hold UsePricingPolicy. The existing Resolve method authorizes policy use and therefore cannot be called as an impersonated capture author or bypassed for financial evidence. Add CheckEvidence(scope,id,digest,from,to) as a trusted internal module read, analogous to Record but validating the current approved state. The future server subject adapter must separately authorize its actual caller before consuming this evidence.

Share the exact locked evidence validation with Resolve: full scope, expected digest, approved state, contained interval, current target and monotonic clock checks. Return defensive copies. Read-only permits existing clock bookkeeping but creates no policy or decision. Resolve retains actor validation and mandatory UseAction authorization before evidence access; every capture path continues using Resolve. Preserve dependency errors from the target port and existing clock-refusal semantics; never turn unavailable into a historical success.

# Acceptance

Verify equality with authorized Resolve; pending/revoked policy refusal; invalid, foreign or mismatched scope/digest/interval; changed and unavailable targets; clock rollback; defensive nested copies; and an actor denied Use for whom internal evidence can still be checked while Resolve refuses. Run the complete pricingpolicy race suite and affected server pricing/capture tests, then independent source/evidence re-review. No unchanged screen rerun is required for this backend-only boundary. Full original W-23 remains active.
