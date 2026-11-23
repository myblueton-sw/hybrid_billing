---
type: Verification
title: Atomic owned-result replacement module prerequisite
description: Records module-only replacement evidence before service ownership-chain and public integration.
status: draft
---

# Scope

HYB-341, machine:mac_mini. Root implemented ReplaceResult in the actual erpledger registry following independent module design PASS. The operation compares complete predecessor evidence under the existing mutex, constructs a detached document with preserved stale-link history and exact new allocations, and publishes only after complete validation. ResultOf now shares its existing locked projection helper; Document and Link semantics remain unchanged.

At this module checkpoint, the service did not yet call this method. Subsequent coordinated service implementation and corrective review are recorded in result-replacement-service-verification.md and result-replacement-plan.md; the module-only result below does not certify that integration. StageERPFinal and the complete owner goal remain unfinished.

# Executed verification

The unchanged erpledger baseline passed in 0.531 seconds. Initial focused replacement tests passed in 0.315 seconds. The final full module race suite passed in 2.006 seconds. Module vet and diff whitespace validation passed. No full service integration, browser replacement or other-platform acceptance is claimed.

Tests use the actual registry and cover complete replacement, defensive input/output copies, retained predecessor history, repeated increasing revisions, cumulative link-capacity refusal, stale full-snapshot CAS, changed submission metadata, wrong products, incomplete/duplicate allocations, wrong revision/scope, nonhuman actors, changed pinned mappings, external-command state, and no document/link/submission mutation on rejection. Concurrent writers admit one CAS and one conflict; coherent read snapshots never expose mixed document/link state.

Additional cases reject both active and stale foreign-preparation links without changing either preparation snapshot. Zero and signed module-supported values remain exact. Equal-total compute/storage row swaps refuse despite numerical reconciliation; correct product-bound pairing succeeds. Amount equality uses scaled exact arithmetic, not allocation-fit predicates or binary fractions.

# Independent review

refusal_review inspected the design, implementation and tests independently. The reviewer required shared/stale-foreign, signed/zero, equal-total cross-product and changed-submission CAS cases; root added them. Final scoped module PASS found no remaining blocker and inspected the 2.006-second race log. The reviewer did not execute tests; vet/diff-check results are root-executed evidence.

The module can preserve same-preparation stale links but cannot independently certify their service-owned provenance. Exact historical chain validation remains mandatory in service integration. No permission, financial approval, external ERP command, owner FIX, merge or deployment is introduced by this checkpoint.
