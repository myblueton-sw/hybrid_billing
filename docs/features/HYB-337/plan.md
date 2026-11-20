---
type: ImplementationPlan
title: B09 module-backed approval reads
description: First phase 6 binding increment, preserving the existing read-only contract and explicit missing producer prerequisites.
status: draft
---

# Scope and ownership

[HYB-337](https://linear.app/myblueton/issue/HYB-337) implements the read portion of B09 under the [pre-launch plan](../HB-257/plan.md). Root owns PM, planning, implementation and integration; an independent read-only reviewer owns billing/security design and change review. Claude CLI ui-designer reviews actual EN/KO/AR renders. Machine: mac_mini. Base: origin/main 0f9d9a2. No merge, deployment, real-account or external financial action is authorized.

# Design for independent review

The current client contract explicitly requires read_only=true, commands=unconnected and source_mode=synthetic. Preserve that boundary. Register GET /api/approval-review in the Go service and bind rows to modules/approvals RequestOf. Keep a bounded service index of registered request IDs and explicit amount basis/target metadata; never duplicate module state or fabricate approval decisions. Add an approvals registry during service initialization and synthetic requests via existing seed mutation steps and real module governance. No production or caller-supplied subjects enter this increment.

Authorize every read using the authenticated session and current gate View decision for a new approval_review kind, workspace-wide scope. Initial synthetic audience is workspace administrators. Ordinary members receive no seed read grant. A workspace-wide decision must not be treated as proof of a narrower customer/entity grant. Actor ID comes from the gate Answer; user input cannot select actor/workspace. Responses are no-store. Unknown prerequisites remain unknown with blockers; no displayed record asserts execution eligibility.

Map charge_approval to billing_result and erp_final_approval to final_erp_amount. Subject amount remains an exact decimal string in an explicitly seeded gross basis; other bases and before/difference/ERP values remain null. Map input/result digests only when actually present. Module pending requests with missing integration evidence appear blocked. Module decisions may be shown as historical decisions, never as proof that downstream prerequisites are current. Preserve separate author and actual decision actor. Unsupported recognition integration returns an explicit unavailable response, not a false empty completed reconciliation. Delegation remains unconnected. HYB-225 retains cross-stage, protected, adjustment and recognition integration responsibilities.

Support pending, decisions and history views, kind/search filters and 25-row pages. Reject unknown/duplicate query keys, invalid UTF-8/control characters and malformed hashes/cursors. Bind opaque cursor to session identity, workspace, current permission context, query and installation revision; reject stale or cross-scope pages. Snapshot hashing covers the filtered module-derived rows and permission context. Read revision and rows under the existing domain lock. Do not derive pending deadlines or expired state: RequestOf supplies neither. Historical approved/rejected status stays historical, with null deadline and unknown current authority. Observation time is excluded from snapshot hashing. No reusable bearer cursor grants access. Bound index and query work; unknown or unavailable module state fails closed rather than disappearing.

# Design review correction

Independent reviewer refusal_review found three blocking ambiguities before code: withdrawn state is absent from the client enum; requests can have multiple/partial decisions while the client has one approver; expiry is an execution check rather than a stored pending deadline. This increment therefore accepts only service-registered synthetic single-approval policies and requests with zero decisions (pending/superseded) or exactly one final approve/reject decision matching the stored state. Withdrawn, partial/multiple decisions, unsupported stage/state, policy-revision mismatch or inconsistent indexed metadata fail the whole read as unavailable, including outside the selected page. They are never silently omitted or relabelled. Fixtures exercise these fail-closed cases.

Approved rows describe the recorded historical decision only. No expiry/deadline or current limit is inferred; authority, dependencies and validation remain unknown. Superseded requests retain their exact stored state. Sort by request ID ascending. Snapshot excludes observed_at and incorporates projected rows, selection, session scope, permission generation and installation revision. Scope fingerprint derives from a one-way hash of the session verifier plus workspace and actual actor; no raw verifier is returned. Cursor is a page offset with a keyed SHA-256 MAC over offset and snapshot using a per-process random key. This remains under 128 characters and no token is an authority substitute.

# Owned files and validation

Root may edit apps/api/server approval-specific files, server assembly/catalog, seed and route registration; B09 API/client integration only where required by actual service evidence; portable tools for B09 QA; docs/features/HYB-337 and the screen binding register. No approvals module rules change without separate reviewed scope. Preserve other branches and the user's running services.

Before code, independently review authorization, exact amounts, state mapping, pagination and missing evidence. Verify authenticated/anonymous/member/cross-workspace reads, all filters, paging, stale/replayed cursors, invalid inputs, module failures, exact amounts and seed=none. Run Go formatting/vet/race tests and decision mutation checks, relevant client tests/build and real browser journeys with isolated synthetic sessions. Review loading/empty/error/refusal, old responses/session changes, EN/KO/AR desktop/mobile/tablet layout, keyboard focus and neutral borders. Record actual results and unverified limits, then re-review corrections and create a draft PR.

# Planning trace and residuals

B09 requirements HB-03, HB-23, HB-24, HB-28, HB-32 and HB-15 are only partially covered by read evidence. AUD-AR-01/06/07, AUD-QA-01 and ERPC-01–08 are not established by this read binding. Financial decisions, live prerequisites, monthly recognition, delegation and external ERP remain unconnected; this increment does not complete phase 6 or the overall goal. The next implementation must address HYB-225 prerequisites before enabling decisions.

HYB-313 remains open with PR #244 and its recorded accessibility residuals. Required companion files absent from the authorized checkout remain absent under the owner's instruction to continue without restoration; no substitute policies or baseline acceptance are invented.

# Reviewed composition adjustment

The independent reviewer approved an exact import exception for packages/primitives/decimal in approvalseed.go only: approvals.NewRegistry requires decimal.Profile and no module factory exists. The service constructs the profile; parsing, arithmetic and monetary decisions remain in modules. Synthetic precision is provisional: 14 input integer digits / 8 fractional digits, 16 result integer digits / 8 fractional digits and 80 intermediate digits. This is not an adopted production money policy. Other primitive imports and binary fractions remain forbidden by the source rule. No wrapper is introduced solely to evade the boundary.

Initial implementation review also required UTF-16 query length parity with the browser and non-binary-fraction count assertions; both are corrected. Initial API tests passed after correcting the stale snapshot response to an explicit HTTP 409 (the shared failCode helper has no revision-conflict entry). Full regression and rendered verification are pending.

# Composed integration with the charge candidate dependency

The independently reviewed next increment stacks this branch on HYB-225, which includes HYB-338/HYB-339 scale corrections. Preserve both registries/seed steps and exact separate grants: candidate Cost/ViewCost and approval General/View. Recalculate exact combined counters/catalog/grant expectations; no implicit candidate-to-approval binding or financial command is introduced. Earlier isolated rendered checks remain evidence, while this new composition needs its own checks.

Extend allowed files to M03 access/live-copy.ts and its existing model tests, P02's shared label consumption, focused approval/candidate grant isolation tests and this ticket's glossary/evidence. All three capabilities (FOCUS, candidate cost and approval review) must be understandable and selectable. Label approval_review as Approval review records / 승인 검토 기록 / سجلات مراجعة الاعتمادات, and approval_evidence as Approval request and decision evidence / 승인 요청·결정 근거 / أدلة طلبات الاعتماد والقرارات. View means reading recorded requests/decisions, not approving, confirming current authority or satisfying unknown producer prerequisites. These are independently reviewed AI terminology candidates, not native acceptance.

Verify exact capability payloads and grant separation on both user/client sides, seed-none explicit-grant reads, combined catalog/counters, unchanged candidate and B09 values/states/cursors/refusals, EN/KO/AR M03/P02/B09 renders and actual Claude/TW re-review. Run fresh full regression on this composed head; the parent HYB-225 run cannot certify it. The eventual draft PR base is HYB-225. No main merge or deployment is authorized.

## Authenticated input and pricing dependency composition

Independent refusal_review approved the local feature-branch integration of HYB-225 a7261d3 before conflict resolution. Root remains sole writer in the existing HYB-337 worktree. Preserve both histories with a local feature-branch merge; no main or PR merge is authorized. The combined service retains one financial approvals registry/index and its existing demonstration rows, separate input/pricing registries and the authenticated v7 charge-candidate producer. B09 remains read-only with unknown current financial prerequisites; this dependency update does not create candidate-backed financial requests.

Resolve shared assembly by retaining both registry initializers, both seed steps in dependency order, distinct approval General/View and candidate/input/pricing Cost capabilities, and all exact source-effect exceptions. Shared M03/P02 copy and glossary keep both previously reviewed capability families. No new layout or wording is introduced. Root owns these resolutions, necessary exact composed expectation corrections, combined regression and this evidence; independent reviewer remains read-only.

Verify approval/candidate grant isolation, v5/v6 byte compatibility, v7 authorship, current input/pricing capture, seed replay/none, hold/step-up and explicit effect inventories. Run focused server race, full server short, console tests/build and actual B09/M03/P02 plus inherited capture journeys on the combined service. Render reviews distinguish unchanged layouts from changed composition and do not fabricate native-language or owner acceptance. Subsequent financial governance and subject adapters must reuse this assembly under HYB-225 scope.
