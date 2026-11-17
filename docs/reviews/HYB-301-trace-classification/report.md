---
type: Review Report
title: HYB-301 classification of the requirement IDs HYB-291 left unclassified
description: Classifies the 24 requirement and acceptance IDs that the HYB-291 cross-check of pre-launch phases 1 to 5 left as unclassified, from the criterion text in the v1.16 planning documents, the screen specs and the built code, and lists behaviour that the planning requires within phases 1 to 5 that is neither built nor ticketed.
status: stable
tags: [HYB-301, pre-launch, planning, requirement-trace, review]
---

# Scope

[HYB-301](https://linear.app/hybrid-billing/issue/HYB-301) follows the owner's decision of 2026-11-17 to follow the planning documents. [HYB-291](../HYB-291-planning-cross-check/report.md) left these IDs as "unclassified – owner review" in the phase 1–5 traces:
- HB-07 (HB-261);
- M02: IAM-01, IAM-03, IAM-05 (HB-265);
- M03: HB-04, AUD-DB-08 (HB-266/267);
- M05: HB-43, IAM-01, IAM-07 (HB-270/271);
- O03: AUD-QA-01 (HB-280/281);
- P01: HB-04, HB-39, HB-40 (HB-282/283);
- P02: HB-31, HB-37, HB-40 (HB-284/285);
- P04: HB-21, HB-27, HB-38 (HB-288/289);
- UI-04, UI-07, UI-08, UI-09, UI-10, which appear only in `acceptance-trace.json`.

Sources read:
- `docs/planning/v1.16/`: `requirements.json`, `detailed-requirements.md` and `.json`, `acceptance-trace.json`, `release-gates.json`, `retention-access.md`, `ui-design.md`;
- the screen specs under `docs/ui/screens/`;
- the feature plans and verification records `docs/features/HB-257` to `HB-289`;
- the code in `apps/api/server`, `apps/console/src`, `apps/console/vite.config.ts` and `modules/retention`.

Linear tickets moved from team HB to team HYB with the same numbers (HYB-n is HB-n). The planning documents and most screen specs are in the authorized local workspace and are not tracked in Git; line numbers refer to those files.

# Who did what and limits

- **Writer.** An AI agent (Opus 5.5, `claude-opus-5-5`), delegated by root, wrote this report and the changed trace rows. It wrote none of HB-258 to HB-289.
- **Independent check.** A first independent review returned CHANGES REQUIRED (M1–M3, L1–L6, N1–N2). This revision applies those findings; a second independent review passed, and its two Lows were applied by root (AUD-DB-08 reclassified as deferred; the Required and missing scope stated). None of this is owner acceptance.
- **Nothing was run.** No test, build or browser was run. Every acceptance test cited stays NOT_RUN.
- **Facts and inferences.** "Built" statements cite a plan, a verification record or code that was read. Statements marked *inference* are not read directly. No ID is marked covered, and none was marked covered on inference.
- **Not checked.** Linear state; the full text of every spec section; code paths beyond those cited; whether each built behaviour passes its own tests today.

# Rules applied

These are the [HYB-291 status meanings](../HYB-291-planning-cross-check/report.md#status-meanings):
- **covered** only if the whole criterion is met within pre-launch scope;
- **deferred** with what is built and what is not;
- **not connected** when the ID depends on an unbuilt external system, workers, money computation or durable storage;
- **not applicable** only when the screen's own field and action contract (or, for a ticket without a spec, its ticket scope) gives it no part of the criterion, with that reason stated. HYB-301 also adds this status to the per-ticket rule in `HB-257/plan.md:181`.

An ID that is required but not built, and that has no ticket, goes into [Required and missing](#required-and-missing).

# Summary

| ID | Ticket(s) | Classification |
| --- | --- | --- |
| HB-07 | HB-261 | not applicable |
| IAM-01 | HB-265 (M02) | not applicable |
| IAM-03 | HB-265 (M02) | not connected |
| IAM-05 | HB-265 (M02) | deferred |
| HB-04 | HB-266/267 (M03) | not connected |
| AUD-DB-08 | HB-266/267 (M03) | deferred |
| HB-43 | HB-270/271 (M05) | deferred |
| IAM-01 | HB-270/271 (M05) | not applicable |
| IAM-07 | HB-270/271 (M05) | deferred |
| AUD-QA-01 | HB-280/281 (O03) | deferred |
| HB-04 | HB-282/283 (P01) | deferred |
| HB-39 | HB-282/283 (P01) | deferred |
| HB-40 | HB-282/283 (P01) | deferred |
| HB-31 | HB-284/285 (P02) | deferred |
| HB-37 | HB-284/285 (P02) | deferred |
| HB-40 | HB-284/285 (P02) | deferred |
| HB-21 | HB-288/289 (P04) | deferred |
| HB-27 | HB-288/289 (P04) | deferred |
| HB-38 | HB-288/289 (P04) | deferred |
| UI-04 | phase 3–5 screens | deferred |
| UI-07 | L01–L03, M01–M05 | deferred |
| UI-08 | P03, P04 | deferred |
| UI-09 | phase 2–5 screens | deferred |
| UI-10 | phase 2–5 screens | deferred |

Covered: none. Not applicable: 3. Not connected: 2. Deferred: 19.

# Classification per ID

## HB-07 (HB-261)

- **Criterion.** The received FOCUS version and the internal target version are separate; BilledCost, EffectiveCost, ListCost and ContractedCost are not mixed with sales; null, not provided and zero are distinct; decimals are kept without float. Partial support is not shown as full compliance (`requirements.json` HB-07).
- **Built.** HB-261 is a session gate. It forwards the reader's response through a reverse proxy that filters headers only (`apps/api/server/readers.go:73-106`; `HB-261/plan.md`, Forwarding). The console calls six `/api/focus-*` paths, all in the gate list (`readers.go:28-29`, `apps/console/vite.config.ts:41-48`), and five `/api/local-*` paths, served by the 8793 onboarding service, which is excluded (recorded, `HB-261/plan.md:24`).
- **Classification: not applicable.** HB-261 has no spec; its ticket scope is the gate, which adds no mapping, conversion or display, so it cannot affect data meaning.
- **Unassessed.** HB-07 for the nine FOCUS reader screens themselves sits in the Python FOCUS import and reader screens built before HB-257. Phases 1–5 do not change them, and neither HYB-291 nor this report assesses them.

## M02 (HB-265)

### IAM-01

- **Criterion.** Block concurrent first-administrator creation, invitation replay and acceptance in another workspace (`retention-access.md:110`).
- **Built.** Nothing in M02. Bootstrap and invitations are L02 and M01 (HB-259, HB-263).
- **Classification: not applicable.** The M02 field and action contract (`M02/spec.md:53-101`) has no creation, invitation or acceptance path.

### IAM-03

- **Criterion.** An account with the same email or name from another IdP, another organization or a recreated LDAP entry does not gain existing rights (`retention-access.md:112`; account model at `:20-24`, linking at `:51`).
- **Built.** Only local password identities exist. They are keyed by issuer and login ID (`apps/api/server/signin.go:93`), never by email. A departed login ID can never be created again (`HB-263/plan.md:139, 254`).
- **Not built.** M02's issuer and stable-subject contract, group mapping and role ceiling, and the IdP and LDAP connections (`M02/spec.md:58, 64`).
- **Classification: not connected.** The criterion needs external IdP or LDAP connections, which are out of pre-launch scope (`HB-257/plan.md`, Out of scope).

### IAM-05

- **Criterion.** Suspension, departure and reduced rights take effect on sessions, keys, jobs and files, and the delay in detecting external changes is measured (`retention-access.md:114`).
- **Built.** The applied policy's maximum sign-in age is checked on every request, so a shorter age ends older sessions at their next request (`HB-265/plan.md`, verification plan: "Maximum age in `access`, tested on each path").
- **Not built.**
  - session impact and invalidation status on a trust change (`M02/spec.md:73`), part of the trust workflow in HYB-297;
  - external-status checks and holding sensitive commands while status is unknown (`M02/spec.md:48-51`), which need an external IdP;
  - effects on keys, jobs and files.
- **Classification: deferred.**

## M03 (HB-266/267)

### HB-04

- **Criterion.** Provider onboarding keeps authentication success, operational read, first data, full coverage, FOCUS conformance, reconciliation and billing go-live as separate states (`requirements.json` HB-04 requirement). The setup role is kept apart from export write and collector read (HB-04 implementation_proposal), and operational read is tested after setup rights are revoked (HB-04 acceptance).
- **Built.** Grants with a validity window and revoke (`HB-266/plan.md:25, 119`). The only grantable pair is `view focus_data rows` (`HB-266/plan.md:106`).
- **Not built.** A setup permission of any kind.
- **Classification: not connected.** The setup permission exists only with provider onboarding (S02) and collectors. Pre-launch connects no real CSP (`HB-257/plan.md`, Out of scope). S02 is outside the Go-bound pre-launch scope and is served ungated by the 8793 onboarding service (HYB-291 finding M9).

### AUD-DB-08

- **Criterion.** No go-live from an old diagnosis or a changed configuration; expired or other-task handoff links are refused; after setup rights are revoked only operational read remains (`detailed-requirements.md:265`; `M03/spec.md:143-149`).
- **Built.** Grants expire and can be revoked (`HB-266/plan.md:25, 119`).
- **Not built.** A setup permission with a revocation condition, and task-scoped expiring handoff links.
- **Classification: deferred** (changed after the second review). The CSP-side setup permission is external and not connected (see HB-04); the task-scoped handoff links are product behaviour that waits on binding S02 provider onboarding, an owner decision.

## M05 (HB-270/271)

### HB-43

- **Criterion.** Retention, archive and deletion policies per data kind; retention period, legal hold, physical deletion and backup expiry kept distinct (`requirements.json` HB-43). M05's part is the recent access history, whose retention period is an open value (`M05/spec.md:34, 119`).
- **Built.** The session list keeps a session until the 12-hour absolute limit prunes it (`HB-270/plan.md:161`).
- **Not built.** The recent access history, recorded as not built (`HB-270/plan.md:125`), and any retention period for it.
- **Classification: deferred.**

### IAM-01

- **Criterion.** As for M02 (`retention-access.md:110`).
- **Classification: not applicable.** The M05 action contract (`M05/spec.md:64-73`) acts only on the caller's own account and has no creation, invitation or acceptance action.

### IAM-07

- **Criterion.** Lookup, invitation, sync and revocation for 100,000 users stay isolated, resumable and checked against the blocking revision (`retention-access.md:116`). M05's part: scope fixed to the person's own user ID; cursor and period filters when there are many sessions; "all other sessions" ends a fixed target manifest after a session-generation check; blocking kept apart from chunked cleanup (`M05/spec.md:72, 93-95`).
- **Built.** Reads and ends are scoped to the caller's own login (`apps/api/server/self.go:74-89`). "End all other sessions" ends them in one call under the installation lock (`self.go:121-130`).
- **Not built.** Cursor and period filters, a fixed manifest with a generation check, and blocking shown apart from cleanup.
- **Inference.** In memory, ending is synchronous, so blocking and cleanup happen together. A session created between the read and the click is also ended.
- **Classification: deferred.**

## O03 AUD-QA-01 (HB-280/281)

- **Criterion.** Payment account, document recipient, tax identifier and template changes need separate verification, approval and a validity period. An account change by new email alone, or an LLM-generated account, is refused. A change reaches documents only after confirmation on the existing channel and separate approval, and only from its effective date (`detailed-requirements.md:355-363`; `O03/spec.md:56, 75, 151-153`).
- **Built.** The Documents and payment tab says it is not connected and names what it will hold (`HB-281/plan.md:49`).
- **Not built.** All of the following:
  - the protected references;
  - the separate change draft, with verification, approval and validity;
  - the channel-confirmation evidence;
  - the refusals.
- **Classification: deferred.** The control itself is internal. Only the confirmation on the existing channel depends on messaging, which pre-launch does not connect.

## P01 (HB-282/283)

### HB-04

- **Criterion.** As above. P01's part: setup evidence bound to a configuration hash and a check validity period, and a preview of what will be created, including rights, resources and cost, without blanket administrator fixes (`P01/spec.md:51, 53`).
- **Built.** The install preview lists services, ports, volumes, data effects, permissions, conflicts and a hash. A 24-hour observation blocks plan approval once it is stale (`HB-282/plan.md`, Read and Out of scope).
- **Not built.**
  - resource and cost figures;
  - the diagnostic request, which needs an authenticated probe (recorded, `HB-282/plan.md`, Out of scope);
  - provider readiness states, which are S02's.
- **Classification: deferred.**

### HB-39

- **Criterion.** Sustained intake above 1 million rows a day and at least 30 million a month are P0 gates. Single jobs are tested at 100,000 rows or more. Untested is never shown as passed (`requirements.json` HB-39; `release-gates.json` mandatory). P01's part is its volume section (`P01/spec.md:104-106`).
- **Built.** Every P01 list is bounded by module limits (4 sites, 200 reports, at most 50 proposals in the read; `HB-282/plan.md`) and read whole.
- **Not built.**
  - snapshot and cursor paging;
  - server search and the filters (`P01/spec.md:62-64`);
  - jobs with fixed denominators;
  - any test at the release profile.
- **Open values.** Resource minimums and performance targets are open values in the spec (`P01/spec.md:147`), not decisions this build could follow.
- **Classification: deferred.**

### HB-40

- **Criterion.** Verify distribution notices, licence boundaries, history access after expiry, and upgrade and data migration (`requirements.json` HB-40).
- **Built.** Signed releases, and upgrade and rollback previews for an installed site. The seed site is not installed (`HB-282/plan.md`).
- **Not built.** Notices, licence boundaries, access after expiry, data migration on upgrade.
- **Classification: deferred.** Licences and notices are P07. P07 is in phase 5 but has no ticket (`HB-257/plan.md:168`).

## P02 (HB-284/285)

### HB-31

- **Criterion.** Collection, validation, approval, issue, delivery and payment deadlines differ. A due date is not an issue command. Notification read, accept and resolve are kept apart (`requirements.json` HB-31). P02's part: the subscriptions and channels tab, with API, module and notification-channel permissions not merged (`P02/spec.md:33, 80`).
- **Built.** Webhook subscriptions with approval, pause, resume and revoke, and delivery states (`HB-284/plan.md:80-95`).
- **Not built.**
  - The separate API, module and channel permissions. This is internal and buildable; HB-284 records it as a deviation, with every command under `AdministerInstallation` (`HB-284/plan.md:308-310, 376`).
  - Notification channels and notification events, and schedules. These need external messaging, which is out of pre-launch scope (`HB-257/plan.md`, Out of scope).
- **Classification: deferred.** Part is internal and not built; part depends on messaging.

### HB-37

- **Criterion.** Recover the database, evidence, keys, policies, modules, numbers, submissions and the outbox together. Do not resume external writes before external results are reconciled (`requirements.json` HB-37, a mandatory gate). P02's part: each module's storage, migration and deletion contract (`P02/spec.md:51`).
- **Built.** Each plugin version's storage contract (namespace, owner, delete on removal or keep for evidence for N days, migration kind) is shown. Install and upgrade are refused when it changes (`HB-284/plan.md:47-55`; `HB-285/plan.md:306`).
- **Not built.** Holding webhook dispatch after a restore. The P03 recovery register has a writer lock (`HB-286/plan.md:38, 49`), but no `apps/api` integration file refers to it.
- **Classification: deferred.**

### HB-40

- **Criterion.** As above. P02's part: signature, compatibility, permission and licence checks on the module manifest, and a licence check before activation (`P02/spec.md:50, 74`).
- **Built.** Versions are signed by a registered publisher key. A revoked publisher, a compromised key or a vulnerable version blocks install (`HB-284/plan.md:47`).
- **Not built.** A licence check. A search of `modules/plugins` and `modules/webhooks` finds no licence field.
- **Classification: deferred.**

## P04 (HB-288/289)

### HB-21

- **Criterion.** No double billing of the same contract, period, component and scope across keys, clients and recalculations (`requirements.json` HB-21, a mandatory gate). P04's part: deletion dependencies include the minimal claim references (`P04/spec.md:35-37`). A source received again after deletion must not be billed twice (HB-43 acceptance; RET-06).
- **Built.** The register has a `claim` class (`modules/retention/types.go:23`). The seed gives it no policy, and references protect required artifacts (`HB-288/plan.md:118, 123-124`). A claim policy can be approved through the screen; only the absence of a deletion-plan command and a worker keeps claims from being deleted.
- **Not built.**
  - No rule keeps minimal duplicate-prevention records after their sources are deleted. The module uses `ClassClaim` only in its class list.
  - The billing claim ledger, which is phase 6.
- **Classification: deferred.**

### HB-27

- **Criterion.** MSP originals, internal settlement and customer files are different artifacts, and customer disclosure uses allowed fields (`requirements.json` HB-27). P04's part: show ERP and downloaded copies as outside retention control (`P04/spec.md:41, 115`).
- **Built.** The screen says backup copies are not tracked (`apps/console/src/modules/administration/pages/retention/live-copy.ts:305`).
- **Not built.** The statement of copies outside control ([Required and missing](#required-and-missing), item 10). Keeping MSP originals, internal settlement and customer files apart belongs to phases 7–8 (exports and disclosure).
- **Classification: deferred.**

### HB-38

- **Criterion.** The simulator runs on the same core. Runs are reproducible from a seed, a clock and versions, with independent expected values and DEMO and synthetic labels. Real external sends are blocked (`requirements.json` HB-38; its detail requirements include QA-10). P04's part is AUD-QA-10: demo reset kept away from operational data, and assistant conversations retained separately.
- **Built.** The seed is synthetic and marked as sample. The screen says periods are synthetic. A `conversation` class exists without a policy (`HB-288/plan.md:120, 124`).
- **Not built.** Scenario manifests, reproducibility, independent expected values, and demo separation.
- **Classification: deferred.** P06 (demo and scenarios) is in phase 5 but has no ticket (`HB-257/plan.md:168`).

## UI-04

- **Criterion.** With 100,000 users, lists of 100,000 rows or more and monthly cumulative detail, the filter, selection, page and total bases agree, and nothing is loaded whole into the browser (`ui-design.md:547`).
- **Built.**
  - M01 API paging, limit 1–100 (`HB-263/plan.md:105`).
  - O01 reads pages of 99, up to 10 pages, and builds the tree in the browser (`HB-273/plan.md:46-47`).
  - O03 filters on the client over at most 200 rows (`HB-281/plan.md:40`).
  - P02 and P03 lists are bounded at 50 (`HB-285/plan.md:685-687`, `HB-287/plan.md:663-665`).
  - P04 has no filters, search or cursors (`HB-289/plan.md:58`).
- **Not built.**
  - Server search and the spec filters on every phase 3–5 screen (for example `M03/spec.md:72`, `P04/spec.md:73-75`).
  - Fixed selection manifests.
  - Totals stated with their scope.
  - Any test at 100,000.
- **Tickets.** M01 and O01 are in HYB-294 and HYB-300; the others have no ticket.
- **Classification: deferred.**

## UI-07

- **Criterion.** Local, SAML, OIDC and LDAP, in normal operation and failure, with MFA, recovery, the same email for a different principal, and 100,000 revocations: no permission bypass and no false progress (`ui-design.md:550`).
- **Built.**
  - Local password sign-in with an identical failure response and dummy-hash timing (`HB-257/plan.md`, Login; HB-259, HB-260).
  - Administrator-issued one-time reset links, suspension and departure (HB-263, HB-264).
  - Ending one's own sessions (HB-270).
- **Not built.**
  - SAML, OIDC, LDAP and MFA, which are out of pre-launch scope.
  - Self-service recovery, an owner decision.
  - Bulk revocation (recorded; HYB-299).
  - Step-up on sensitive commands (HYB-293).
- **Classification: deferred.**

## UI-08

- **Criterion.** Policy shortening, then impact, hold, partial deletion, the backup wait and restore: the screen states exactly the confirmed scope and the copies outside control (`ui-design.md:551`).
- **Built.** In P04 (HB-288/289):
  - a shortening proposal with its stored impact (`HB-288/plan.md:131`);
  - holds with a `not_preserved` count (`:129`);
  - seeded runs with deleted and failed results per store and a damaged artifact (`:127`);
  - `decision: not_connected` on runs and tombstones (`:161`);
  - the note that backups are not tracked (`live-copy.ts:305`).
- **Not built.**
  - Deletion plan proposal and approval, the worker and the backup wait: recorded as not connected (`HB-288/plan.md:283`).
  - Archive restore: not connected.
  - The statement of copies outside control (see HB-27, item 10).
- **Classification: deferred.**

## UI-09

- **Criterion.** Keyboard-only sign-in, filters, selection, approval, modals and return from errors, plus checks of status reading, contrast, zoom, wrapping and colour vision (`ui-design.md:552`).
- **Built.**
  - Keyboard reach and a visible focus outline on M01 (`HB-264/verification.md:85`).
  - Keyboard order and no horizontal scroll at 390 and 1440 on O03 (`HB-281/verification.md:31`).
  - Polite live regions and focus restoration as an interaction pattern (`HB-265/plan.md:180-185`).
- **Not run.** No verification record from HB-260 to HB-289 states a contrast, zoom or colour-vision result (searched for "contrast", "zoom", "colour" and "axe"). No record states a keyboard-only end-to-end flow.
- **Classification: deferred.**

## UI-10

- **Criterion.** Keep stale, null, zero, unsupported and unconfirmed apart, along with channel acceptance against user confirmation and inquiry against resolution. No fake success and no action that does nothing (`ui-design.md:553`).
- **Built.**
  - Null kept apart from zero in O03 (`HB-281/plan.md:127`).
  - Unconnected areas are stated, not filled (`HB-281/plan.md:48-51`; P04 `decision: not_connected`).
  - Actions are offered only where the read allows them, with a blocker (HB-282 read blockers; HB-289).
- **Not built.** Channel acceptance against user confirmation, and inquiry against resolution: messages and cases are not connected (phases 7–8).
- **Not checked.** No check across screens that no action is dead and no success is faked.
- **Classification: deferred.**

# Required and missing

Behaviour found by HYB-291 and HYB-301 that the planning requires within phases 1–5 and that is neither built nor covered by an existing ticket; phases 1–5 were not re-audited in full for this list (HYB-292 to HYB-300). The proposed titles are the writer's; root or the owner decides whether to open them.

| # | Behaviour | Source | Recorded before? | Proposed ticket title |
| --- | --- | --- | --- | --- |
| 1 | Server search, the spec filters, stable sort and cursor paging on the identity screens | UI-04; `M02/spec.md:81`, `M03/spec.md:72`, `M04/spec.md:62`, `M05/spec.md:60` | no | Server search, filters and cursor paging for M02–M05 |
| 2 | The same on the operations and platform screens | UI-04, HB-39; `O02/spec.md:61`, `O03/spec.md:63`, `P01/spec.md:62-64`, `P02/spec.md:62-64`, `P03/spec.md:62`, `P04/spec.md:73-75` | O02, O03 (client side), P02, P03, P04 yes; P01 no | Server search, filters and cursor paging for O02, O03 and P01–P04 |
| 3 | M05 recent access history with a provisional retention period | HB-43; `M05/spec.md:34, 119` | yes (`HB-270/plan.md:125`) | M05 recent access history with a provisional retention period |
| 4 | M05 "end all other sessions" against a fixed session manifest and generation, with blocking shown apart from cleanup | IAM-07; `M05/spec.md:72, 95` | no | M05 end other sessions against a fixed session manifest |
| 5 | O03 protected payment, recipient, tax identifier and template changes with separate verification, approval and validity, and refusal of email-only or LLM-generated changes | AUD-QA-01; `O03/spec.md:56, 75, 151-153` | tab marked not connected (`HB-281/plan.md:49`) | O03 payment and recipient change control with separate verification and approval |
| 6 | Licence check in the plugin manifest and before activation | HB-40; `P02/spec.md:50, 74` | no | P02 plugin licence check before activation |
| 7 | P06 (demo and scenarios) bound to the service | HB-38, AUD-QA-10; `HB-257/plan.md:168` | yes, as "not yet ticketed" | Bind P06 demo and scenarios to the service |
| 8 | P07 (licences and distribution notices) bound to the service | HB-40; `HB-257/plan.md:168` | yes, as "not yet ticketed" | Bind P07 licences and distribution notices to the service |
| 9 | A retention rule that keeps minimal duplicate-prevention (claim) records after their sources are deleted | HB-21, RET-06; `P04/spec.md:35-37` | no | Retention rule keeping minimal claim references after source deletion |
| 10 | P04 statement that ERP and downloaded copies are outside retention control | HB-27, UI-08; `P04/spec.md:39-41` | no | P04 statement of copies outside retention control |
| 11 | Accessibility verification of the phase 2–5 screens: keyboard-only flows, status reading, contrast, zoom and colour vision | UI-09; `ui-design.md:552` | no | Accessibility verification of the phase 2–5 screens |
| 12 | A check across screens for unknown, stale, zero and not-connected states, fake success and dead actions | UI-10; `ui-design.md:553` | no | Error and trust state check of the phase 2–5 screens |
| 13 | Three Python paths that still bypass the gate | `HB-257/plan.md:153`; `apps/console/vite.config.ts:63-71` | no | Remove or gate the remaining direct Python proxy paths |
| 14 | Separate API, module and channel permissions, plus stop and reprocess permissions, on P02 commands | HB-31, HB-42; `P02/spec.md:33, 70-76` | yes, as a deviation (`HB-284/plan.md:308-310, 376`) | Split the P02 integration permissions |
| 15 | Concurrent first-administrator creation blocked, and acceptance of an activation or invitation link from another workspace refused, with tests | IAM-01; `retention-access.md:110` | yes, as not built or tested (`HB-259/verification.md:109`, `HB-263/verification.md:204`) | IAM-01 concurrent bootstrap and cross-workspace acceptance for L02 and M01 |

Notes:
- **Items 1 and 2.** They extend HYB-294 (M01) and HYB-300 (O01) to the other screens.
- **Item 4.** Low priority. *Inference:* in memory, ending is synchronous under the installation lock.
- **Item 5.** Confirmation on the existing channel needs messaging. The rest can be built, and could be scheduled with phase 7 issuance.
- **Items 7 and 8.** HB-257 names P06 and P07 in phase 5.
- **Item 9.** May depend on the phase 6 claims module.
- **Item 10.** Low priority. It matters once exports exist (phase 7); the spec's `policy_complete` state already says copies outside control are shown separately (`P04/spec.md:115`).
- **Items 11 and 12.** They could be part of the phase 10 QA ticket, which does not exist yet.
- **Item 13.** HB-257 says `apps/api` forwards "the other existing Python paths" and that "the Vite proxy then no longer reaches Python directly". `vite.config.ts:63-71` still proxies `/api/credit-case-review`, `/api/claim-history` and `/api/approval-review` straight to port 8806 in both modes; the gated mode replaces only the six reader paths (`:41-51, 84`). `modules/focus_import/overview_http.py` serves none of the three. *Inference:* the requests fail at the Python server, so no data is exposed. There is a second effect. In gated mode these three paths are same-origin, so the browser sends the HttpOnly session cookie (`Path=/api`) to the Python server on 8806. `HB-261/plan.md:96` lists only 8793 and the ungated readers as receiving it. This is outside the IDs assigned to HYB-301 and was found while checking HB-07.

## HYB-291 owner-decision findings

HYB-291 left these findings as owner decisions or residuals without a ticket (`HYB-291/report.md:147-155`). Each now has a ticket:

| HYB-291 finding | Ticket |
| --- | --- |
| M2 M02 trust flow | HYB-297 |
| M3 M03 approval request and revocation states | HYB-298 |
| M4 O01 actions and filters | HYB-300 |
| M5 O02 activation without money impact | HYB-296 |
| M10 M01 role validity periods and bulk revocation | HYB-299 |

None of these five remains without a ticket. HYB-291's M8 (security-focused review not stated) and M9 (session gate opt-in and the 8793 service) are still owner decisions and residuals without a ticket. They are process and phase 10 items, not product behaviour, so they are not in the table above.

Not in the list:
- Behaviour that depends on an external system or durable storage, such as federated sign-in, MFA, messaging, ERP and real connectors. These are out of pre-launch scope or not connected.
- Gaps already ticketed: HYB-293 step-up, HYB-297 M02 trust workflow (which includes M02 session impact), and HYB-299 bulk revocation.

# Changes in this ticket

- **Verification records.** `docs/features/HB-261`, `HB-265`, `HB-266`, `HB-267`, `HB-270`, `HB-271`, `HB-280`, `HB-281`, `HB-282`, `HB-283`, `HB-284`, `HB-285`, `HB-288` and `HB-289` `verification.md`:
  - each "unclassified – owner review" row is replaced by its classification, marked "(classified by HYB-301)" and linked to this report;
  - the HYB-291 framing paragraph is kept, with one added sentence;
  - no other result changes.
- **Phase plan.** `docs/features/HB-257/plan.md`: the per-ticket trace rule now lists "not applicable with a concrete reason".
- **HYB-291 report.** `docs/reviews/HYB-291-planning-cross-check/report.md`:
  - the UI table and the "Unclassified IDs left for the owner" table point to these classifications;
  - "not applicable" is added to the status meanings, including tickets without a spec.
