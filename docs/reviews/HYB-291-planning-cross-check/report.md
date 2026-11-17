---
type: Review Report
title: HYB-291 planning cross-check of pre-launch phases 1 to 5
description: Read-only cross-check of the pre-launch phase 1 to 5 tickets (HB-258 to HB-289) against the HB-257 phase plan, the screen specs and the HYB-290 plan, with evidence, the reviewer's limits, and the disposition of each finding.
status: stable
tags: [HYB-291, pre-launch, planning, cross-check, review]
---

# Scope

[HYB-291](https://linear.app/hybrid-billing/issue/HYB-291) checks pre-launch phases 1 to 5 against the planning documents:
- the phase plan, [HB-257](../../features/HB-257/plan.md);
- the screen specs for L01–L03, M01–M05, O01–O03 and P01–P04 under `docs/ui/screens/`, with their trace sections;
- the planning baseline in `docs/planning/v1.16/` (`requirements.json`, `acceptance-trace.json`, `release-gates.json`, `retention-access.md`);
- the feature records `docs/features/HB-258` to `docs/features/HB-289`;
- the draft HYB-290 plan for P05 (`docs/features/HYB-290/plan.md`), read from its worktree.

Linear tickets moved from team HB to team HYB with the same numbers (HYB-n is HB-n). Records written before the move keep the HB prefix.

Most screen specs other than M02 are in the authorized local workspace and not yet tracked in Git; spec line numbers below refer to those files.

# Reviewer and limits

- **Who did what.** A read-only auditor, an AI agent (Opus 5.5, `claude-opus-5-5`) separate from the writers of HB-258 to HB-289, produced the findings and edited nothing. The HYB-291 writer agent, delegated by root, wrote this report, the trace classifications, the residual notes and the copy changes. The HYB-291 change review is the first independent check of that writing. None of this is owner acceptance.
- **No tests or builds were run** by the audit. Everything comes from reading files. Coverage claims below are inferred from the behaviour that was built.
- **Not checked:** Linear state; the full M04, P02, P03 and P04 specs, the `design.json` files, `planning-decisions.md` and `delivery-plan.md`; which Python paths the nine FOCUS screens use; whether every command the screens offer matches the specs beyond the items cited. Code was spot-checked only.
- **Overall result: FAIL.** One user-facing contradiction, three required behaviours missing without a record, several unrecorded deviations, and no requirement IDs in the feature records.

# Ticket map

| Phase | Tickets |
| --- | --- |
| 1 | HB-258 (service skeleton) |
| 2 | HB-259 (sign-in API), HB-260 (L01–L03), HB-261 (session gate in front of the Python readers) |
| 3 | HB-262 (installation administration action), M01 HB-263/264, M02 HB-265, M03 HB-266/267, M04 HB-268/269, M05 HB-270/271 |
| 4 | O01 HB-272/273, O02 HB-274/275, O03 HB-280/281 |
| 5 | P01 HB-282/283, P02 HB-284/285, P03 HB-286/287, P04 HB-288/289; P05 is HYB-290 (draft) |

The live screens are registered in `apps/console/src/app/screen-groups/identity.tsx:10-14`, `work.tsx:12-14` and `operations.tsx:12-15`.

# Findings per screen

Requirement and acceptance IDs come from each spec's trace section: `L01/spec.md:18`, `L02:18`, `L03:18`, `M01:118-120`, `M02:24-26`, `M03:133-159`, `M04:118-128`, `M05:18`, `O01:116`, `O02:117`, `O03:121-155`, `P01:118-136`, `P02:120-146`, `P03:120-130`, `P04:137-155`.

Before this ticket no record from HB-258 to HB-289 cited an HB-, IAM- or AUD- ID. The only citations were RET-01–08 as NOT_RUN (`HB-288/plan.md:301`), QA-09 (`HB-286/plan.md:227`, `HB-287/plan.md:656`) and QA-06 (`HB-284/plan.md:385`, `HB-285/plan.md:688`).

- **L01** (HB-03, HB-42; IAM-01, 02, 04, 07). Built: an identical 401 for every failure, the token kept in the URL fragment, paste allowed, a double-submit guard. Recorded deviations (`HB-260/plan.md:132-146`): no method field, focus on the error summary, no MFA or SSO. SAML, OIDC and LDAP are residuals.
- **L02** (HB-01, 36, 42; IAM-01, 02, 03, 05). Built: activation and set and reset links, one-time use, constant-time checks (HB-259). IAM-03 does not arise while only local accounts exist; nothing recorded that.
- **L03** (HB-03, 42; IAM-04, 06, 07). **Contradiction:** `apps/console/src/service/copy.ts:154-156` said administrator-issued reset links were not available, while HB-263/264 shipped `identity.issue_reset_link` (`apps/console/src/modules/identity/pages/users/live-copy.ts:253`) and M02's recovery view points to it (`HB-265/plan.md:178`).
- **M01** (HB-03, 34, 42; IAM-01–08). Built: invitation, set and reset links, suspend, depart, role change, last-administrator protection. Bulk revocation recorded as not built (`HB-263/plan.md:224`, `HB-264/plan.md:61`). **Missing without a record:** server search and the workspace, status, auth method, role and expiry filters (`M01/spec.md:61-63`; IAM-07, UI-04), as the API only pages (`HB-263/plan.md:105`); reauthentication before suspending a user (`M01/spec.md:73`; IAM-08), as step-up applies only to proposing and approving authentication policy changes and `apps/api/server/commands.go:208-213` otherwise checks only the policy's maximum age; role validity periods (`M01/spec.md:52`).
- **M02** (HB-03, 36, 42; IAM-01–08). Built: two-person proposal and approval of the policy, maximum age enforced on every request. Recorded as not available: providers, test login, keys, sync, recovery (`HB-265/plan.md:177-178, 200-208`). **Not recorded:** the draft, validated, independently approved, activated trust flow (`M02/spec.md:96-131`) is collapsed into propose, then approve, which applies.
- **M03** (HB-03, 04, 23, 33, 36, 42; AUD-AR-09, AUD-DB-08, AUD-QA-04). Built: grants, delegation revoke, effect preview, self-grant refused. Recorded: roles and support views not available, support sessions, narrower scopes (`HB-266/plan.md:153-161`, `HB-267/plan.md:41`). **Not recorded:** grant changes apply directly instead of through an approval request (`M03/spec.md:83`); the revocation states received, deny confirmed and cleanup running (`M03/spec.md:84, 109-113`) do not exist. Recorded risk: after a partial failure a second submit can create a second grant (`HB-267/verification.md`, Residuals).
- **M04** (HB-03, 05, 24, 33, 42; AUD-QA-04; IAM-01–08). Ownership transfer, the owner-departure policy (`docs/planning/v1.16/retention-access.md:24`) and credential authentication are recorded residuals (`HB-268/plan.md:159-168`).
- **M05** (HB-03, 31, 42, 43; IAM-01, 04, 06, 07, 08; AUD-QA-08). Built: own sessions, password change. Recorded as unavailable: notifications, recovery, MFA (`HB-271/plan.md:109-111, 154-160`). Self-service changes are not written to the command journal; recorded (`HB-270/plan.md:158`).
- **O01** (HB-01, 02, 03, 14). Read-only; admin-only and the flat, client-built tree are recorded (`HB-272/plan.md:156-161, 208-215`; `HB-273/plan.md:190-199`). The client-built tree conflicts with "no full load into the browser" (`O01/spec.md:102`; UI-04), though it is recorded. **Not recorded:** the view group draft and create change plan actions (`O01/spec.md:72-73`) and the legal-entity, relationship-kind and provider filters (`O01/spec.md:61`).
- **O02** (HB-02, 14, 15, 21, 23). Built: plans drafted, sealed and activated by two people. Recorded (`HB-274/plan.md:253-265`, `HB-275/plan.md:145-155`): no money, publication or ERP impact, no B09 approval, no scheduling, no resume. Activation still happens although the spec makes amount impact required (`O02/spec.md:54, 72`); owner acceptance is needed. The register finding on allowance preservation (`docs/ui/screen-binding-register.md:70`) is not addressed.
- **O03** (16 HB IDs; AUD-AR-01, AR-10, DB-03, QA-01, QA-02). Well recorded (`HB-280/plan.md:125-133`): `final_amount_decision: not_computed`, and the move from two-person to single-administrator review is an owner residual.
- **P01** (HB-03, 04, 36, 39, 40, 42; AUD-AR-09, DB-08, DB-13). Built: plan, proposals, a stale-observation block (part of AUD-DB-08), diagnostics reports. Recorded as not connected: diagnostic request, support bundle, support access (`HB-282/plan.md:87-94`).
- **P02** (14 HB IDs; AUD-AR-02, QA-04, QA-05, QA-06). Network, signing, marks, separate permissions and QA-06 origins are recorded (`HB-284/plan.md:363-386`).
- **P03** (HB-08, 21, 22, 34, 37, 42, 43; AUD-QA-09; RET-01–08). Restore, migration and non-human paths are recorded (`HB-286/plan.md:222-232`). RET-01–08 were not cited.
- **P04** (HB-06, 08, 11, 21, 27, 32, 37, 38, 43; AUD-AR-05, AUD-QA-10; RET-01–08). Plans, workers, archive and the backup catalog are recorded as not connected (`HB-288/plan.md:280-301`). AUD-AR-05 and AUD-QA-10 are cited only in the module ticket (`HB-256/plan.md:234`).

# Phase-level checks against HB-257

| Check | Result | Evidence |
| --- | --- | --- |
| Sign-in | PASS (read) | HB-259/260 match `HB-257/plan.md:121-147`. |
| Session gate in front of the Python endpoints | PARTIAL | Six reader paths are gated (HB-261). The onboarding service on 8793 is excluded though HB-257 listed `/api/local-*` (recorded, `HB-261/plan.md:24, 92`). The gate holds only with both opt-in switches; the start script is phase 10 (`HB-261/plan.md:89, 93`). Whether the nine FOCUS screens use only those paths is UNVERIFIED. The design review came after the code (recorded, `HB-261/plan.md:122`). |
| Audit journal sizing | FAIL, not recorded | HB-257 asks for an eight-hour session at 1 command per second (at least 28,800 records) and a test that drives the journal to that bound. The code keeps 2 hours (`apps/api/server/server.go:121`, `Retention: 2 * time.Hour`) and the test drives 90×60 = 5,400 commands (`apps/api/server/server_test.go:237-242`). Inferred effect: audit records older than 2 hours drop out during a long review session. |
| Seed rules | recorded deviations | The sample member acts in the seed (`HB-288/plan.md:295`); HYB-290 adds two module-only reviewers; HB-280 uses a `seed` actor (`HB-280/plan.md:57`). Names are marked as sample. |
| Pre-launch notice | present, incomplete | `apps/console/src/service/copy.ts:4-12` did not mention the audit journal resetting. |
| One mutation path | recorded exception | M05 self-service bypasses the command journal (`HB-270/plan.md:158`). |
| Platform independence | PASS as recorded | 386 and Windows cross builds; macOS arm64 is the tested combination; Windows file modes are a residual (`HB-258/verification.md:32, 70`). |
| Screen QA | not recorded | No record mentions `npm run qa:*`; every screen ticket from HB-260 used its own browser QA instead (`HB-257/plan.md:178`). |
| Security-focused review for phases 1–3 | not stated | HB-260, HB-264 and HB-267 do not state one (`HB-257/plan.md:191`). |
| Screen binding register | not extended | No binding ticket extended `docs/ui/screen-binding-register.md` (`HB-257/plan.md:176`). Only HB-288 records this, citing "as in HB-286", which HB-286 does not record. |
| Phase→ticket table | stale | `HB-257/plan.md:162-173` named HB-260 for M01–M05 and HB-261 for O01–O03. |

# Orphan requirement IDs

These IDs are in the specs of screens built in phases 1–5 but appeared in no record from HB-258 to HB-289 before this ticket:
- **HB IDs:** HB-01–06, 08, 10–12, 14–16, 18, 21–25, 27, 28, 31–34, 36–40, 42, 43. This includes every mandatory non-functional gate in `release-gates.json:5-14` (HB-03, 08, 21, 34, 37, 39, 42, 43).
- **IAM-01–08**, all of them.
- **AUD IDs:** AUD-AR-01, AR-02, AR-05, AR-09, AR-10; AUD-DB-03, DB-08, DB-13; AUD-QA-01, QA-02, QA-04, QA-05, QA-08, QA-10.
- **Partly cited:** RET-01–08 (HB-288 only, not P03); QA-06 and QA-09 without the AUD- prefix.
- **Release gates:** all 43 still read NOT_IMPLEMENTED. That is consistent with pre-launch, but nothing linked them to the built work.

This ticket adds a requirement trace to each verification record (see Changes in this ticket). The UI acceptance IDs in `acceptance-trace.json` (added_tests) are not in any screen spec's trace section, so they are not in the per-ticket traces. Their mapping is left to the owner:

| ID | Criterion (summary) | Screens it plausibly touches | Status |
| --- | --- | --- | --- |
| UI-04 | Large lists: filters, selection, paging and totals agree; no full load into the browser | M01, O01, P02 | unclassified – owner review |
| UI-07 | Authentication and revocation: local, SAML, OIDC, LDAP, MFA, recovery, 100,000 revocations | L01–L03, M01–M05 | unclassified – owner review |
| UI-08 | Retention and deletion: shortening, impact, hold, partial deletion, backup wait, restore | P03, P04 | unclassified – owner review |
| UI-09 | Accessibility: keyboard-only flows, status reading, contrast, zoom, colour vision | all phase 2–5 screens | unclassified – owner review |
| UI-10 | Errors and trust: stale, null, zero, unsupported and unknown kept distinct; no fake success | all phase 2–5 screens | unclassified – owner review |

# HYB-290 (P05) plan

Read from the HYB-290 worktree, `docs/features/HYB-290/plan.md` (status draft).

**Holds:** its spec citations (`P05/spec.md:25, 33, 59, 63, 74, 97, 103, 137-139`) check out; "43 gates, all NOT_IMPLEMENTED" is correct; `scale-validation.md:54` is quoted correctly; activation, proof, support decisions, the `revoked` state and paging are listed as not connected or residuals (`:281-309`); the seed-actor deviation is recorded.

**Gaps:**
- It does not cite P05's IDs: HB-04, 19, 24, 25, 26, 38, 39, 40, 41; AUD-DB-01; AUD-DB-09. AUD-DB-01 is modelled only implicitly through `common` and `common_needs_replacement`.
- AUD-DB-09 (an event test matrix per commitment profile, unsupported events recorded rather than passed) is neither covered nor deferred.
- Server search and the provider, contract or role, version, issuance path and evidence-state filters (`P05/spec.md:61-63`) are not mentioned.
- The evidence drawer's "reason not run" (`P05/spec.md:37`) is missing; tests without results get `latest_outcome: null` and no reason.
- The concurrent collect, close and delete load (`P05/spec.md:41`, SCALE-11) has no profile and is not mentioned.
- No reviewer verdict table is recorded yet, only root decisions.

# Findings by priority

**High**
- H1. L03 copy contradicts the built administrator resets (`copy.ts:154-156` against HB-263/264 and `HB-265/plan.md:178`).
- H2. No reauthentication or step-up on sensitive identity commands (`M01/spec.md:73`; IAM-08; `commands.go:208-213`).
- H3. M01 has no server search or filters (`M01/spec.md:61-63`; IAM-07, UI-04; `HB-263/plan.md:105`).

**Medium**
- M1. Audit journal sizing and its test do not match `HB-257/plan.md:109` (`server.go:121`, `server_test.go:237-242`).
- M2. M02 trust flow simplified without a record (`M02/spec.md:96-131`).
- M3. M03 grants have no approval request and no revocation states (`M03/spec.md:83-84, 109-113`).
- M4. O01 group-draft and change-plan actions and filters missing without a record (`O01/spec.md:61, 72-73`).
- M5. O02 activates although the required money impact is not computed (`O02/spec.md:54, 72`; `HB-274/plan.md:257`).
- M6. Screen binding register not extended by any binding ticket (`HB-257/plan.md:176`).
- M7. HYB-290 has no coverage or deferral of AUD-DB-09 or the P05 filters.
- M8. Security-focused review not stated for HB-260, HB-264 and HB-267 (`HB-257/plan.md:191`).
- M9. Session gate opt-in and the excluded 8793 path (`HB-261/plan.md:24, 92, 93`); the nine-screen path mapping is unverified.
- M10. M01 role validity periods (`M01/spec.md:52`) not built and not recorded; bulk revocation not built (recorded).

**Low**
- L1. No requirement or acceptance IDs in any record from HB-258 to HB-289.
- L2. HB-257's phase→ticket table is stale (`HB-257/plan.md:162-173`).
- L3. `npm run qa:*` replaced by per-ticket browser QA without a record.
- L4. The pre-launch notice omits the audit journal reset.
- L5. `HB-288` says "as in HB-286" for the register, but HB-286 does not record it.
- L6. HYB-290: no "reason not run", no concurrency profile, no recorded reviewer verdicts.

# Disposition

| Finding | Action | Where |
| --- | --- | --- |
| H1 L03 reset copy | fixed in this ticket | `apps/console/src/service/copy.ts` `forgotBody` (EN, KO, AR); residual note in `HB-260/verification.md` |
| H2 step-up reauthentication | fix ticket HYB-293 | residual notes in `HB-263`, `HB-264` verification |
| H3 M01 search and filters | fix ticket HYB-294 | residual notes in `HB-263`, `HB-264` verification |
| M1 audit journal sizing | fix ticket HYB-292 | residual in `HB-257/plan.md`; trace in `HB-258/verification.md` |
| M2 M02 trust flow | owner decision / residual | `HB-265/verification.md` |
| M3 M03 approval request and revocation states | owner decision / residual | `HB-266`, `HB-267` verification |
| M4 O01 actions and filters | owner decision / residual | `HB-272`, `HB-273` verification |
| M5 O02 activation without money impact | owner decision (acceptance needed) / residual | `HB-274`, `HB-275` verification |
| M6 screen binding register | fix ticket HYB-295 | `HB-288` citation corrected |
| M7 HYB-290 AUD-DB-09 and filters | HYB-290 plan amendment | HYB-290 plan |
| M8 security-focused review not stated | owner decision / residual (no review is claimed) | `HB-260`, `HB-264`, `HB-267` verification |
| M9 session gate opt-in and 8793 | owner decision / residual (phase 10 start script) | already recorded in `HB-261/plan.md` |
| M10 M01 role validity periods and bulk revocation | owner decision / residual | `HB-263`, `HB-264` verification |
| L1 no requirement IDs | fixed in this ticket (trace sections); HYB-290 plan amendment for P05 | every verification record from HB-258 to HB-289 |
| L2 stale phase→ticket table | fixed in this ticket | `HB-257/plan.md` |
| L3 bespoke browser QA instead of `npm run qa:*` | owner decision / residual | `HB-257/plan.md` Residuals and this report |
| L4 notice omits audit reset | fixed in this ticket | `copy.ts` `notice` (EN, KO, AR) |
| L5 HB-288 citation | fixed in this ticket | `HB-288/verification.md`, `HB-288/plan.md` |
| L6 HYB-290 reason not run, concurrency profile, reviewer verdicts | HYB-290 plan amendment | HYB-290 plan |

# Status meanings

Each verification record from HB-258 to HB-289 now ends with a "Planning cross-check (HYB-291)" section. Its requirement trace uses these statuses:
- **covered:** the pre-launch build addresses the whole criterion within pre-launch scope. This is inferred from the built behaviour; the acceptance test stays NOT_RUN.
- **deferred:** part or all of the ID is not built and is a recorded residual, a later ticket or an owner decision.
- **not connected:** the ID depends on something pre-launch does not connect (external systems, workers, money computation, durable storage).
- **unclassified – owner review:** the spec links the ID, but the cross-check could not tie it to built or recorded behaviour.

HB-03 is a mandatory gate (`requirements.json` HB-03: author and approver separation, one permission registry across API, workers, caches, files and search, no leakage under ID tampering, delegation expiry and revocation). It is `deferred` in every record: each ticket addresses it within pre-launch scope for its own path, but registry-wide and leakage acceptance is not built. Rows that cover only part of a criterion are `deferred` and say which part is covered.

HB-258, HB-261 and HB-262 have no screen spec; their IDs are derived from the HB-257 phase scope and say so.

# Changes in this ticket

- `apps/console/src/service/copy.ts`: L03 `forgotBody` now says the administrator issues a password reset link in Users and access ('사용자·접근', «المستخدمون والوصول»), that the link works once, and that reset links cannot be issued for administrator accounts, so administrators contact another administrator, following the M01 wording (`adminResetReason`, `issue_reset_link`). The pre-launch `notice` now names the audit history of changes among what resets on restart. EN, KO and AR are AI drafts and need the technical-writing and native-language review in `.agents/rules/product-terminology.md`.
- `docs/features/HB-258` to `HB-289` `verification.md`: a "Planning cross-check (HYB-291)" section with a requirement trace, and "Planning cross-check residuals (HYB-291)" where findings apply. Status fields are unchanged.
- `docs/features/HB-288`: the "as in HB-286" register citation corrected in `verification.md` and `plan.md`.
- `docs/features/HB-257/plan.md`: the phase→ticket table corrected, the HB→HYB move noted, the per-ticket requirement citation and phase-end audit rule added, and the QA and audit-journal residuals added.

# Unclassified IDs left for the owner

| Screen | IDs |
| --- | --- |
| FOCUS readers (HB-261) | HB-07 |
| M02 | IAM-01, IAM-03, IAM-05 |
| M03 | HB-04, AUD-DB-08 |
| M05 | HB-43, IAM-01, IAM-07 |
| O03 | AUD-QA-01 |
| P01 | HB-04, HB-39, HB-40 |
| P02 | HB-31, HB-37, HB-40 |
| P04 | HB-21, HB-27, HB-38 |
| all | UI-04, UI-07, UI-08, UI-09, UI-10 |
