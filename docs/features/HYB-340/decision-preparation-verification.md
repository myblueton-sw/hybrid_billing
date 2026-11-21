---
type: VerificationRecord
title: Authenticated financial preparation verification
description: Separate current action outcomes without financial mutation or permission inference.
status: draft
---

# Scope and independent review

HYB-340, machine:mac_mini, follows b91cbea. Root implemented financial_charge.prepare as a non-mutating View approval_review/approval_evidence command with actual-human and console read checks before binding lookup. It independently checks current human charge approval permission and requester exclusion, pending state, shared candidate/policy evidence, stranded effects, per-action hold/step-up and module CheckDecision. The existing mutating approval exception is unchanged. The fixed fifteen-field response contains exact recorded facts and closed action outcomes; existing field/value preflight refuses undersized capability before response construction.

Independent refusal_review passed the concrete pre-code design and source review without executing tests. Independent platform_refusals passed implementation/evidence review against logs and found an obsolete preparation-pending sentence in the contract, which was corrected. It requested a required-MFA regression, which was added. Preparation permits no current-producer probe after decision permission refusal. Reject needs decision permission/separation and fresh evidence but no amount coverage. The shared freshness helper is also used by fresh mutations. Preparation creates no decision, reservation or financial revision; existing command audit/rate state, role expiry sweeps and module clock bookkeeping remain possible. checked_at is a post-check wall-clock observation, not a module watermark or future validity guarantee.

# Verification and corrections

Initial existing financial transport tests passed 1.883s. The first new test build used the wrong private authority-record signature; the test was corrected to use the existing authenticated revoke command. The initial combined preparation/transport/hold/step-up tests passed 5.746s. A later policy replacement assertion expected evidence_stale, but actual response recorded superseded/not_pending; module confirmation intentionally supersedes requests, so only the test expectation was corrected. The first required-MFA test correctly received unauthenticated after policy replacement invalidated its old session. It now asserts that refusal and signs in freshly before checking both actions report prerequisite_failed.

The final focused race run, including MFA and all preparation/transport/mixed-projection cases, passed 10.075s (/private/tmp/hyb340-preparation-final-race.log). Earlier pre-MFA race passed 9.904s. go vet ./... passed (/private/tmp/hyb340-preparation-vet.log). The full go test -short ./... suite passed across 81 packages (server 136.461s); this run began before the later MFA-only test addition, which is covered by the final focused race run.

Actual HTTP tests cover exact USD15, original author/requester/policy/revision, repeated preparation without request/revision mutation, stale selected digest, terminal state, read/decision separation, no-read nondisclosure, original-author/requester exclusions, missing/foreign/corrupt bindings, revoked approval permission, revoked amount authority with rejection still eligible, modified producer bytes, unavailable producer, hold, aged session, MFA, stranded effects, policy supersession, output bounds and a later refused mutation after current conditions changed. Browser execution is not claimed for this API-only increment. B09 controls remain the next separately reviewed work.

No owner acceptance, main merge, deployment, production/customer data, payment or external message occurred. HYB-340 and complete W-23 remain active.
