---
type: VerificationRecord
title: Atomic ERP result evidence snapshot
description: Coherent submission, reconciliation and complete linked document facts without financial eligibility claims.
status: draft
---

# Implemented prerequisite

[HYB-341](https://linear.app/myblueton/issue/HYB-341), machine:mac_mini. Root implemented the independently reviewed first prerequisite in [the plan](plan.md): erpledger.Registry.ResultOf returns a SubmissionView, Reconciliation and complete linked DocumentViews under one existing registry lock. Reconcile uses the same extracted private calculation. No service endpoint, result ingestion, authority or financial readiness was added.

Every document with any retained link to the target is included, including stale links. Its rows, products, revision and all links remain visible, including links to other submissions. Documents follow the existing sorted connection/external order. Existing copy routines isolate submission lines, mappings, commands, watch and resolution plus document rows and links. Reconciliation slices are freshly allocated. Missing, foreign and malformed identities return the existing ErrNotFound result. Output is bounded by configured retained records and is not truncated; a future transport needs its own reviewed bounds.

# Actual verification

The initial focused ResultOf tests passed in 0.429s. Full erpledger race tests, including source rules and existing reconciliation regressions, passed in 2.131s. Module go vet passed. Evidence: /private/tmp/hyb341-result-first.log, /private/tmp/hyb341-result-race.log and /private/tmp/hyb341-result-vet.log. Scoped consumer regression (Test.*(ERP|Document|SourceRules)) passed for server in 6.460s and erpadapters in 0.270s; /private/tmp/hyb341-result-consumers.log. These are filtered tests, not the full server suite. Independent refusal_review passed the actual source and tests with no blocker and inspected the race log without executing tests. Independent platform_refusals passed the source/log/evidence comparison; it did not rerun tests.

Actual module fixtures cover empty/unlinked preparation, partial and complete attribution, deterministic document ordering, malformed/foreign identity, mixed attribution, stale links and equal-total product changes. The latter deliberately leaves Reconcile.Matched true while exposing the new revision/product for later policy validation. Concurrent replacement checks that document rows, total, revision/product and reconciliation values belong to one snapshot, without requiring writer ordering. Mutating returned nested members does not alter the retained registry or an earlier snapshot.

The lifecycle-copy case uses existing in-memory module submit/observation/resolution APIs solely to populate command, watch and resolution members for alias checks. It has no network adapter or external execution and does not claim service-owned ERP result ingestion. The read itself leaves those histories unchanged.

# Remaining integration

This snapshot supplies coherent facts, not complete financial validation. Governed independently supplied synthetic result packets, provenance ownership, partial ingestion recovery, separate current recording/confirmation authority, complete service-level reconciliation and StageERPFinal remain required. No external submit/post, issuance, delivery, persistence, production tax policy, UI, native-language approval, owner FIX, merge or deployment is claimed. The full W-23 and owner goal remain active. Execution was on macOS only.
