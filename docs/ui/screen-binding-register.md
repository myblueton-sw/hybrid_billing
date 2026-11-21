---
type: Screen Binding Register
title: Screen binding register for local companion screen specifications and pre-launch service bindings
description: Records the field action and state bindings that HB-15 FA-04 FA-05 FA-13 SR-03 and the HB-22 contract clauses require in local companion screen specifications, and, per HB-257, which reads and commands the pre-launch service mode binds for L01-L03, M01-M05, O01-O03 and P01-P07, with tickets, states, residuals and verification records.
status: stable
sources: [{resource: "../../통합_빌링_플랫폼_구현_기획안_v1.16.docx"},
  {resource: "../reviews/HB-15-full-planning/re-audit.md"}, {resource: "../reviews/HB-15-full-planning/source-review.md"},
  {resource: "../planning/v1.16/source-omission-trace.json"},
  {resource: "../contracts/commands/planning-actions.md"},
  {resource: "../features/HB-257/plan.md"},
  {resource: "../reviews/HYB-291-planning-cross-check/report.md"}]
---

# Screen binding register

Tracking: Linear ticket [HB-23](https://linear.app/hybrid-billing/issue/HB-23). The HB-15 and HB-22 sections of this register list, for each screen whose specification is an authorized local Korean companion outside Git, the binding that a finding or contract clause requires and the local line that currently conflicts with it or lacks it.

**An entry in the HB-15 and HB-22 sections does not change the local specification.** The cited local text remains in force as written until it is adopted. Adoption of each entry into its specification, and publication of those specifications, belong to [HB-24](https://linear.app/hybrid-billing/issue/HB-24) under HB-15 FA-14. Every finding that depends on a local specification is therefore partially remediated, with the residual named here, not remediated.

Inputs of the HB-15 and HB-22 sections: [HB-15 re-audit](../reviews/HB-15-full-planning/re-audit.md) FA-04, FA-05 and FA-13; [HB-15 source review](../reviews/HB-15-full-planning/source-review.md) SR-03; the `screen_followup` field of the [HB-22 source omission trace](../planning/v1.16/source-omission-trace.json) with contracts [source ERP and issuance obligations](../contracts/billing/source-erp-and-issuance-obligations.md) (EI), [source pricing and cost obligations](../contracts/billing/source-pricing-and-cost-obligations.md) (SP) and [source policy, onboarding and verification obligations](../contracts/operations/source-policy-onboarding-and-verification-obligations.md) (OP). Commands are the proposed IDs of the [planning action registry](../contracts/commands/planning-actions.md); a dash means the binding uses an existing screen action with no new command. Source locators are DOCX body-child indexes (`B`) and table rows (`.R`); they were checked against the extracted original text. Local line references are `docs/ui/screens/<ID>/spec.md:<line>` in the authorized local workspace at the time of writing. Acceptance IDs are NOT_RUN.

The HB-15 and HB-22 sections are a planning register under the planning gate. They select no customer value, threshold or timing; source examples such as D-7 stay examples, and conditional scope stays conditional.

## Screens edited in Git by HB-23

Tracked specifications adopt their bindings directly: [B09](screens/B09/spec.md) (adjustment decision, final amount approval, stale dependencies, FX components), [U02](screens/U02/spec.md) (operational notification rules, stale dependencies), [B11](screens/B11/spec.md) (OP-08 consent and distinct holds, adjustment drafts), [I04](screens/I04/spec.md) (EI-05 customer files), [S02](screens/S02/spec.md) (EI-07, OP-05, OP-06, SP-02 readiness and cutover) and [S06](screens/S06/spec.md) (EI-07 parallel reconciliation, adjustment drafts). They are not repeated below.

## HB-15 findings

| Screen | Finding / clause | Source locator | Required field, action and state binding | Command | Local line that conflicts or lacks it |
| --- | --- | --- | --- | --- | --- |
| B04 | FA-04 adjustment lifecycle | B0486.R007, B0495 | Target total and quantity correction field carries kind (`quantity`, `fixed_amount`, `target_total`), declared net, tax or gross basis, currency and expected base result revision and digest. Difference preview runs on a saved draft revision. Adjustment approval is a B09 adjustment decision, not a policy approval. States draft, computing, validated, blocked, stale | `adjustment.draft`, `adjustment.compute`, `adjustment.cancel`, `adjustment.approve` | `B04/spec.md:51` lacks basis, currency and revision; `:69` preview is not bound to a draft revision; `:70` sends the result as `정책 승인 요청` (policy approval), which conflicts |
| B08 | FA-04 application and month-to-quarter invalidation | B0495; MS-05, MS-07, MS-09 | Show base and applied adjustment lines separately with result revision; apply an approved adjustment; publish a replacement month; recompute the complete aggregate; show stale approval and dependency status distinct from generic blocked; reject apply on issued or unknown-dispatch targets with the correction path | `adjustment.apply`, `close.publish_replacement`, `settlement.recompute_aggregate`, `settlement.dependency_status` | `B08/spec.md:65–68` offer calculate, compare, retry and approval request only; `:79` blocked says a new run is needed without a superseded-month or stale-approval state; `:48` shows differences without adjustment revision |
| B05 | FA-05 raw rate versus markup and fee | B0463, B0464, B0467, B0474, B0477.R002 | Split raw published rate (greater than zero, positive quotation unit, correct pair and direction, approved and effective) from selling markup and conversion fee (zero valid; sign per component policy). Blocked state for an invalid raw rate; finding when a raw rate contains markup | FX component validation in the registry; existing approval request | `B05/spec.md:50` applies `0/음수 거절` (reject zero or negative) to raw rate, markup and fee together, which conflicts |
| P02 | FA-13 channels for operational rules | B1387, B1392 | Channel bindings are referenceable by operational notification rules; channel connection and notification receipt grant no billing permission; channel failure stays visible in the app inbox | `notification_rule.draft` consumes channel refs | `P02/spec.md:52` covers channel destinations and `:72–76` are extension-module actions; neither binds operational notification rules |
| M05 | FA-13 separation from personal preferences | B1391 | Personal quiet hours, digest, snooze and severity re-alert cannot edit an operational rule or remove an escalation step; the personal alternate is labelled as a personal preference distinct from the operational alternate owner | — | `M05/spec.md:54` personal `대체 담당` (alternate owner) is not distinguished from the operational alternate owner; `:74` save action does not state that operational rules are unaffected |
| U03 | FA-13 entry point | B1388.R001–R005, B1391 | Authorized users reach U02 operational rules; read, acknowledgement, owner acceptance and resolution remain separate | — | `U03/spec.md:68–70` route notification settings only to personal M05 preferences |
| P06 | SR-03 demo entry profile and purpose | B1286, B1293, B1317, B1318.R001–R008, B1328.R001–R010, B1396 | Fields operating profile (MSP, standalone enterprise, corporate group), purpose (public sample view, synthetic settlement, failure reproduction), scenario ID and revision, expected-result bundle revision required for synthetic and failure purposes. Public sample view labelled as viewing with missing documents and partial months, never an invoice answer. Trace SIM01–SIM10 and OPSUI06 | `demo.scenario_run`, `demo.compare_expected`, `demo.reset` | `P06/spec.md:17` narrows the purpose to synthetic scenarios; `:49–55` have no profile or purpose field; `:71–74` actions are generic; `:115–133` trace DB-09 and QA-10 only |

## HB-22 contract clauses

| Screen | Finding / clause | Source locator | Required field, action and state binding | Command | Local line that conflicts or lacks it |
| --- | --- | --- | --- | --- | --- |
| A01 | P2-06 OP-09 | B0624, B0653.R001–R009, B0662 | Bind AIV-01 to AIV-11. Context header shows installation, issuing entity, customer, account and period; answers carry source, refresh time and provisional or final state; results show job number and target, success, held and failed counts | — | `A01/spec.md:32` context shows workspace, target, period and currency without issuing entity or account; `:110–114` acceptance has no AIV IDs |
| B01 | P1-04, P1-05, P1-14 SP-04 | B0060, B0309, B0310, B0312, B0314, B0319.R001–R004 | Asset acquisition cost, residual value, useful life and effective period from customer input or approved asset system; shared-server split ratio with revision and approver summing exactly to the server cost with explicit idle and unattributed lines; allocation denominator selection; margin versus internal subsidy; internal allocation never summed with external sale | — | `B01/spec.md:48` lacks the asset inputs and their source; `:46–51` have no split-ratio or denominator field; `:79` blocked covers only missing evidence and duplication |
| B01 | P1-06 OP-01 | B0301, B0302 | Setup order register pool, link assets and organizations, select basis, preview, customer approval, apply; templates stay drafts; scope without approved policy shows calculation-not-ready | — | `B01/spec.md:65–68` actions lack the customer-approval step and calculation-not-ready state |
| B02 | P1-03 SP-05 | B0065 | Missing usage never zero; an estimate exists only when the contract permits it and links a separate provisional statement | — | `B02/spec.md:50` marks estimates but has no estimate permission or provisional-statement link |
| B02 | P1-04 SP-04 metering | B0319.R001–R004, B0324, B0325, B0327 | Power cumulative versus interval with corrected-value evidence; VM identity from source plus stable identifier distinguishing rename, recreation and move; host and VM never both charged; vCenter counter check with no settings change; period without history or retention is unsupported for exact usage billing | — | `B02/spec.md:46–51` lack VM identity and host/VM exclusivity; `:65` diagnosis does not state the read-only vCenter rule; `:78` blocked lacks the unsupported-period reason |
| B02 | P1-12 OP-03 | B0279 | HawkEye statement fields and KST calendar month; explicit approved mapping to a different contract period or timezone | — | `B02/spec.md:47` has intervals only, no statement-month mapping |
| B03 | P1-11 OP-02 | B0141 | Closed organization's cost stays unallocated with reason until an approved reassignment revision names target, interval and approver; allocated plus unallocated equals target per currency | — | `B03/spec.md:50` idle and unattributed handling has no organization-closure state; `:79` |
| B04 | P1-06 OP-01 | B0301, B0302 | Templates and recommended values never activate automatically; without approved policy show calculation-not-ready; retroactive change shows affected periods and amounts; platform update never changes an approved revision | — | `B04/spec.md:66` and `:79` lack calculation-not-ready and retroactive impact |
| B04 | P2-01 SP-01 | B0545.R001–R004, B0548, B0549, B0569, B0571, B0576, B0581 | One selling method per component with activation precondition; savings basis evidence, held when absent; usage-only versus net-of-unused savings; separate unused-burden fee setting; loss raises warning and approval, never an automatic rate increase | — | `B04/spec.md:47–48` offer tiers, discounts, markup and target margin without selling method, savings basis or unused-burden fee setting |
| B04 | P3-02 SP-07 | B1218, B1219.R001–R005, B1222 | Contract pool revision: pool ID, participants, target SKUs, unit, currency, period, shared or per-entity allowance and minimum, consumption order, carryover, payer, paying agent, recipient, overage policy; simulation before and after join or leave | — | `B04/spec.md:47` has per-rule allowance only, no pool membership or consumption order |
| B05 | P1-07, P2-02 SP-03 | B0460.R004, B0469, B0551, B0667.R009 | FX snapshot purpose and version: prepay purchase date versus monthly billing; cancellation and credit default to the original invoice rate with override reason; realized payment difference recorded separately; FX change after approval requires recalculation and reapproval | — | `B05/spec.md:48–49` select rate type and date only; `:56` columns have no snapshot purpose |
| B06 | P2-01 SP-01 | B0576 | Prepay already recovered is not billed again monthly; no amortization recalculation | — | `B06/spec.md:49` has a pass-through policy only |
| B06 | P2-03 SP-02 | B0523.R001–R006, B0526, B0527 | Provider-specific retained evidence and double-count guards; FOCUS source and target versions recorded and never changed implicitly; MSP extension values separate; unsupported values null with reason | — | `B06/spec.md:47` separates Billed and Effective (consistent); `:48–50` lack provider guards and null-with-reason display |
| B07 | P2-09 SP-10 | B0686.R001–R006, B0697, B0699, B0700 | Required transaction type; FOCUS columns versus separate namespace; explicit discount inheritance and Marketplace exclusion; CSP private offer outside MACC consumption; no automatic purchase; late seller settlement never changes a customer invoice | — | `B07/spec.md:46` has no transaction type; `:49` lacks inheritance and exclusion flags; `:65` mapping action does not reject a missing type |
| B08 | P3-07 OP-08 | B1373 | Comparison cause separates same-source attribution, policy or FX change from new source correction; history views and exports keep row and field policies and audit | — | `B08/spec.md:30` and `:66` distinguish input, policy and adjustment changes but not FX change versus new source correction |
| C02 | P1-03 SP-05 | B0065 | Provisional statement labelled with estimation basis; final difference via a linked adjustment or correction | — | `C02/spec.md:44` document kinds have no provisional statement; `:46` |
| C02 | P1-08, P1-09 SP-06 | B0034, B0129, B0130, B1219.R003, B1224 | Document kind distinguishes provider original, derived per-customer attribution statement, on-premises internal cost statement and MSP sales invoice; consolidated view is a non-billing report | — | `C02/spec.md:44` lists sales invoice, internal allocation, credit and correction without derived-statement or provider-original distinction |
| I01 | P1-08, P1-09 SP-06 | B0034, B0129, B0130, B1219.R003, B1224 | Issuance checks combine conditions (selling entity, customer, currency, period, tax; for groups buying entity and ERP route) and splits on mismatch; different issuers never merge; document kind and title per SP-06 | — | `I01/spec.md:49` kinds are ERP, self and internal only; `:73` issuance preconditions lack the combine-or-split check |
| I02 | P1-01 EI-01 | B0155, B0873.R009 | ERP outage never switches to self-issuance; connection failures and mode changes resume after recovery | — | `I02/spec.md:92–101` states have no outage or mode-change state |
| I02 | P1-02 EI-02 | B0219, B0850 | Automatic posting off at initial go-live; enabling is a separate approved profile change; protection mechanism is a precondition | — | `I02/spec.md:75` posting request has no automatic-posting default or protection state |
| I02 | P2-07 EI-03 | B0850, B0873.R003, B0873.R005 | Post-posting read-back compared with the pre-approval snapshot (amount, tax, lines, currency, customer, product); difference after posting opens correction review; manual draft edit requires reapproval | — | `I02/spec.md:54` external revision only; no read-back result field |
| I02 | P2-08 EI-04 | B0846.R003, B0870, B0871, B0873.R006 | Master designated at onboarding with no bidirectional overwrite; locked period or inactive master returns error and hold without substitute date or product; history after ERP replacement through the original connection | — | `I02/spec.md:49` lacks master designation; no period-lock hold state |
| I02 | P2-06 OP-09 | B0873.R001–R009 | Bind ERV-01 to ERV-09 | — | `I02/spec.md:119–169` trace has no ERV IDs |
| I02 | P3-04 EI-06 | B1044, B1047, B1048 | Route lock, manual assignment request and designated-owner approval, recovery synchronization and unlock; route and per-submission states; held rows are not dropped | `erp_route.lock_dispatch`, `erp_route.assign_manual`, `erp_route.approve_manual_assignment`, `erp_route.sync_after_recovery`, `erp_route.unlock_dispatch` | `I02/spec.md:72–76` have no route actions; `:51` shared ledger is consistent |
| I03 | P3-03 EI-05 | B1032.R001–R007, B1035, B1040 | Seven-sheet standard workbook; template version metadata (ERP product, version, format, columns, codes, encoding, tax responsibility, rounding, test history); unmapped codes block upload-file generation; ERP-determined values marked not final; boundary splits with file list, counts and totals | — | `I03/spec.md:50` template version lacks metadata; `:73` has no unmapped-code block |
| I03 | P3-04 EI-06 | B1044, B1047 | Review-only file for unknown outcomes; manual result registration recorded operator-confirmed, distinct from ERP-confirmed | `erp_route.register_manual_result`, `erp_route.approve_manual_assignment` | `I03/spec.md:74` registration has no operator versus ERP confirmation distinction; `:75` re-handoff is not bound to designated-owner approval |
| O02 | P3-02 SP-07 | B1219.R001–R005, B1222 | Moving an entity between groups never resets its allowance; separating an entity in a lookup view keeps the contract pool; mid-term exit follows contract rules | — | `O02/spec.md:49` and `:53` have no allowance preservation or lookup-versus-pool distinction |
| O03 | P1-01 EI-01 | B0155, B0873.R009 | Issuer or issuance-mode change applies only to unissued scope from its effective date; self-issuance only where capabilities are verified | — | `O03/spec.md:50` issuer field has no effective-date scope rule |
| O03 | P3-01 SP-08 | B1469.R001–R006, B1472 | Lifecycle event policies and proration declaration (method, timezone, bounds, amount or quantity, allowance, minimum and discount ratio); without policy, proration is not finalized | — | `O03/spec.md:52–53` reference period and proration policy only; `:99` blocked is consistent |
| O03 | P3-02 SP-07 | B1219.R001–R005 | Paying agent is never interpreted as a change of buying entity | — | `O03/spec.md:50` lists buyer, payer and issuer without paying agent |
| R01 | P1-10 SP-09 | B0078 | AI view shows remaining included quantity with estimated and final cost | — | `R01/spec.md:48` dimensions lack remaining included quantity |
| R01 | P2-02 SP-03 | B0551, B0667.R009 | Margin report shows FX difference as a separate component when cost and selling bases differ | — | `R01/spec.md:46` amount meanings have no FX-difference component |
| S01 | P3-05 OP-06 | B1159, B1163, B1183 | Connection fixes FOCUS version and dataset, extension columns, collection time, account scope and period, target schema and conversion version; acquisition complete separate from billing go-live possible | — | `S01/spec.md:46–52` lack FOCUS version and conversion fields; `:24–26` readiness lacks the split |
| S05 | P1-09 SP-06, P2-12 EI-08 | B0034, B0130, B0977 | Derived statement never labelled provider original; full multi-customer original never disclosed automatically; preserved CSV only to a full-scope principal after recheck; converted CSV and partial extracts labelled | — | `S05/spec.md:50` lacks provenance labels; `:71` original download lacks the full-scope check |

## HB-23 residuals

All entries above are open until HB-24 adopts them. Named residuals: FA-04 in local B04 and B08; FA-05 in local B05; FA-13 in local P02, M05 and U03; SR-03 in local P06; HB-22 follow-up 3 in local A01, B01, B02, B03, B04, B05, B06, B07, B08, C02, I01, I02, I03, O02, O03, R01, S01 and S05. Runtime, UI, accessibility, ERP and provider behavior remain NOT_RUN.

## Pre-launch service bindings

Tracking: [HYB-295](https://linear.app/hybrid-billing/issue/HYB-295). [HB-257](../features/HB-257/plan.md) (Screen binding) requires each binding ticket to record which screen fields come from which module read or command in this register. None of the binding tickets did; the [HYB-291 audit](../reviews/HYB-291-planning-cross-check/report.md) recorded this as finding M6. This section is the catch-up, written from Git history, the feature records and the code at main `82c9272` (2026-11-18). It was added after the HB-23 sections so that the existing line citations of this file (for example `screen-binding-register.md:70`) stay valid.

**Owner acceptance is pending for every screen below.** No ticket, check or review cited here is owner acceptance (FIX). How each entry was checked is in the [HYB-295 verification record](../features/HYB-295/verification.md).

### Conventions

- Short paths: `admin.ts` is `apps/console/src/service/admin.ts`, `api.ts` is `apps/console/src/service/api.ts`, `boundary.go` is `apps/api/server/boundary.go`; page paths start at `apps/console/src/modules/`.
- **Reads** name the HTTP route, its registration line in `boundary.go` and the console client method in `admin.ts` or `api.ts`. **Commands** name the command IDs the screen sends through `POST /api/v1/commands/{command}` (`boundary.go:24`, client `admin.ts:871`) and the server definition function.
- **Tickets** list every merged ticket that bound or changed the screen in service mode, with its pull request. Linear moved from team HB to HYB with the same numbers; records written before the move keep the HB prefix.
- **States** summarise what the binding ticket says it built; the verification record holds the evidence. A state that is only unit-tested or model-tested is marked so where the record says so.
- **Open tickets** were read from Linear on 2026-11-18 (title and state only). A ticket is listed against a screen when its title or description names that screen.
- **Field maps.** Each screen field, column and state, with its read or command, JSON field, parser line and render line, is in [sign-in](screen-binding-fields/sign-in.md), [identity](screen-binding-fields/identity.md), [organizations](screen-binding-fields/organizations.md) and [operations](screen-binding-fields/operations.md), linked from each screen entry.
- Every screen shares the platform-wide residuals in [Common to every bound screen](#common-to-every-bound-screen); they are not repeated per screen.

### Summary

| Screen | Bound in service mode | Binding tickets (PR) | Later changes (PR) | Owner acceptance |
| --- | --- | --- | --- | --- |
| L01 | yes | HB-259 (#179), HB-260 (#180) | HYB-293 (#210, shared `api.ts` and `Service.tsx`) | pending |
| L02 | yes | HB-259 (#179), HB-260 (#180) | HYB-293 (#210, shared `api.ts` and `Service.tsx`) | pending |
| L03 | yes, guidance only | HB-260 (#180) | HYB-291 (#206, copy), HYB-293 (#210, shared `Service.tsx`) | pending |
| M01 | yes | HB-263 (#183), HB-264 (#184) | HYB-293 (#210), HYB-298 (#212, copy), HYB-299 (#218), HYB-294 (#227), HYB-323 (this change) | pending |
| M02 | yes | HB-265 (#185) | HYB-293 (#210), HYB-297 (#215) | pending |
| M03 | yes | HB-266 (#186), HB-267 (#187) | HYB-293 (#210), HYB-297 (#215), HYB-298 (#212) | pending |
| M04 | yes | HB-268 (#188), HB-269 (#189) | HYB-293 (#210) | pending |
| M05 | yes | HB-270 (#190), HB-271 (#191) | HYB-293 (#210) | pending |
| O01 | yes | HB-272 (#192), HB-273 (#193) | HYB-293 (#210), HYB-300 (#216), HYB-318 (#217), HYB-319 (#219) | pending |
| O02 | yes | HB-274 (#194), HB-275 (#195) | HYB-293 (#210), HYB-296 (#214), HYB-318 (#217), HYB-319 (#219), HYB-322 (#221) | pending |
| O03 | yes | HB-280 (#196), HB-281 (#197) | HYB-293 (#210) | pending |
| P01 | yes | HB-282 (#198), HB-283 (#199) | HYB-293 (#210) | pending |
| P02 | yes | HB-284 (#200), HB-285 (#201) | HYB-293 (#210) | pending |
| P03 | yes | HB-286 (#202), HB-287 (#203) | HYB-293 (#210) | pending |
| P04 | yes | HB-288 (#204), HB-289 (#205) | HYB-293 (#210), HYB-298 (#212, refactor only), HYB-312 (#222), HYB-325 (#223) | pending |
| P05 | yes | HYB-290 (#209), HYB-315 (#211) | none found | pending |
| P06 | **no** | none | none | pending |
| P07 | **no** | none | none | pending |

### Common to every bound screen

- **Service mode and session.** Service mode is entered with `?service=api` or `/activate` (HB-260). The console checks `GET /api/v1/session` (`boundary.go:23`, `api.ts:107`) and shows L01 without a session. The service skeleton, cookie session, CSRF token and command endpoint come from HB-258 (#178).
- **Live switch.** The M screens switch to their live components in `apps/console/src/app/screen-groups/identity.tsx:41-166`, the O screens in `work.tsx:44-61`, the P screens in `operations.tsx:21` and `:51-79`.
- **Commands.** Every command carries the expected revision, a reason and an idempotency attempt (`admin.ts:871`). Every registered mutating command and `POST /api/v1/me/sessions/end` require a recent password confirmation (HYB-293; `apps/api/server/stepup.go:30-41`); the console asks for it in `service/Reauthenticate.tsx` and posts `POST /api/v1/me/reauthenticate` (`boundary.go:50`, `admin.ts:795`).
- **Permission.** The installation administration action (HB-262, #182) gates the administrator screens; exceptions are named per screen.
- **Journal.** The command journal keeps eight hours (HYB-292, #213; `apps/api/server/server.go:158`).
- **Pre-launch limits.** Data, registers and the journal reset on restart, as the notice says (`service/copy.ts:4`).
- **Open tickets on every phase 2–5 screen.** Accessibility verification HYB-313 (UI-09) and error and trust state check HYB-314 (UI-10), both Todo. The residuals of HB-260, HB-267, HB-269, HB-271, HB-273, HB-275, HB-281, HB-283, HB-285, HB-287, HB-289, HYB-296, HYB-297, HYB-298, HYB-299, HYB-300, HYB-315 and HYB-318 record the KO and AR copy as an AI draft or as pending terminology and native-language review.
- **Process gaps (HYB-291).** The per-ticket rule of running `npm run qa:*` was replaced by per-ticket browser QA (HYB-291 L3), and a security-focused review is not stated for HB-260, HB-264 and HB-267 (HYB-291 M8).

### Sign-in screens (L01–L03)

**L01 Sign in**
- **Tickets.** HB-259 sign-in API (#179), HB-260 screens (#180). HYB-293 (#210) changed the shared `api.ts` and `Service.tsx`, which all three sign-in screens use; it adds no step-up to them.
- **Field map.** [L01 fields](screen-binding-fields/sign-in.md#l01-sign-in).
- **Reads.** `GET /api/v1/auth/csrf` pre-session token (`boundary.go:55`, `api.ts:71`); `GET /api/v1/session` (`boundary.go:23`, `api.ts:107`).
- **Commands.** `POST /api/v1/auth/login` (`boundary.go:56`, `api.ts:117`), `POST /api/v1/auth/logout` (`boundary.go:57`, `api.ts:121`). Not on the command path. Component: `SignIn` in `apps/console/src/service/Service.tsx:155`. Modules: passwords, identity, authpolicy.
- **States.** Empty, failed sign-in with one identical message for every failure, limited (429 with wait), unavailable with Try again, signed out, successful sign-in that keeps the requested route ([HB-260 verification](../features/HB-260/verification.md), Scope and Checks run).
- **Not connected / residual.** No sign-in method field, SAML, OIDC, LDAP, MFA or SSO; focus goes to the error summary rather than the first error (recorded deviation, `HB-260/plan.md:134-135`).
- **Verification.** [HB-259](../features/HB-259/verification.md), [HB-260](../features/HB-260/verification.md).

**L02 Set password or activate from a one-time link**
- **Tickets.** HB-259 (#179), HB-260 (#180); HYB-293 (#210) as for L01.
- **Field map.** [L02 fields](screen-binding-fields/sign-in.md#l02-set-password-or-activate-from-a-one-time-link).
- **Reads.** `GET /api/v1/auth/csrf` (`boundary.go:55`).
- **Commands.** `POST /api/v1/auth/link` peek (`boundary.go:58`, `api.ts:127`), `POST /api/v1/auth/redeem` (`boundary.go:59`, `api.ts:134`). Not on the command path. Component: `Activate` in `Service.tsx:235`; the token is read only from the URL fragment (`Service.tsx:380`). Modules: passwords, identity.
- **States.** Set, reset and first-administrator activation; invalid or used link; password rule errors; limited; unavailable (HB-260 verification).
- **Not connected / residual.** IAM-03 does not arise while only local accounts exist (HYB-291). Open: HYB-303 (In Progress), concurrent first-administrator creation and acceptance from another workspace (IAM-01); not merged.
- **Verification.** [HB-259](../features/HB-259/verification.md), [HB-260](../features/HB-260/verification.md).

**L03 Recovery guidance**
- **Tickets.** HB-260 (#180); HYB-291 (#206) corrected `forgotBody` (`service/copy.ts:195`) to say the administrator issues a reset link in Users and access (HYB-291 finding H1); HYB-293 (#210) as for L01.
- **Field map.** [L03 fields](screen-binding-fields/sign-in.md#l03-recovery-guidance).
- **Reads and commands.** None; static guidance. Component: `Help` in `Service.tsx:211`.
- **States.** Guidance only, reached from L01 or `view=help`.
- **Not connected / residual.** Self-service recovery and emergency access are not built. The HB-260 residual "Administrator-issued resets" is superseded by HB-263/264.
- **Verification.** [HB-260](../features/HB-260/verification.md) (HYB-291 cross-check section).

### Identity administration (M01–M05)

**M01 Users and access**
- **Tickets.** HB-263 API (#183), HB-264 screen (#184), HYB-299 role validity periods and bulk revocation (#218), HYB-293 step-up (#210), HYB-294 server search, filters and paging (#227), HYB-323 select everyone matching a filter (this change). HYB-298 (#212) changed only M01 copy (`users/live-copy.ts`).
- **Field map.** [M01 fields](screen-binding-fields/identity.md#m01-users-and-access).
- **Reads.** `GET /api/v1/identity/users` (`boundary.go:25`, `admin.ts:854`, called at `identity/pages/users/Live.tsx:509, 529`); `GET /api/v1/identity/links/{ref}` one-time link pickup (`boundary.go:30`, `admin.ts:1092`, `users/Live.tsx:255`); `GET /api/v1/identity/users/count` bounded count before fixing a filter list (`boundary.go:26`, `service/identity-bulk.ts:174`, called at `users/Matching.tsx`); `GET /api/v1/identity/bulk-revocations`, `/{id}`, `/{id}/items`, `/{id}/targets` (`boundary.go:27-30`, `service/identity-bulk.ts`).
- **Commands.** `identity.create_user` (`users/Live.tsx:210`); `identity.` + the action offered by `users/live-model.ts:34-47` (`suspend_user`, `reinstate_user`, `depart_user`, `invite`, `change_role`, `set_role_period`, `suspend_membership`, `reinstate_membership`, `remove_membership`, `issue_set_link`, `issue_reset_link`, `revoke_invitation`) through `users/Live.tsx:143`; `identity.set_role_period` also at `users/RolePeriod.tsx:170`; `bulk_revocation.add_targets` (`identity-bulk.ts`), `bulk_revocation.add_matching` (`users/Matching.tsx:176`), `bulk_revocation.execute`, `prepare` and `discard` (`users/BulkPrepare.tsx:139, 224, 244`, `users/RevocationJobs.tsx:100`). Server: `userDefinitions` (`apps/api/server/admin.go:471`), `bulkDefinitions` (`bulkstage.go:98`). Modules: identity, bulkrevoke, decisionledger.
- **States.** Paged users, invitations and memberships; actions derived from state with self, departed, administrator-reset and promotion blocks; one-time link shown once; reload after settled failures; role periods with an end; bulk revocation staging, preview with exact counts, execute and replay ([HB-264](../features/HB-264/verification.md), [HYB-299](../features/HYB-299/verification.md)).
- **Not connected / residual.** Server search and filters: HYB-294 (#227). Select everyone matching a filter: HYB-323 (this change; residuals in `HYB-323/verification.md`). Removing chosen people from a filter list: HYB-324 (open). HYB-299 residuals R1–R15 and owner questions Q1–Q11 stand (`HYB-299/verification.md`, Residuals).
- **Verification.** [HB-263](../features/HB-263/verification.md), [HB-264](../features/HB-264/verification.md), [HYB-299](../features/HYB-299/verification.md), [HYB-293](../features/HYB-293/verification.md).

**M02 Authentication policy and trust changes**
- **Tickets.** HB-265 API and screen (#185), HYB-297 trust change workflow (#215), HYB-293 (#210).
- **Field map.** [M02 fields](screen-binding-fields/identity.md#m02-authentication-policy-and-trust-changes).
- **Reads.** `GET /api/v1/identity/policy` (`boundary.go:31`, `admin.ts:883`, `identity/pages/authentication/Live.tsx:96`); trust changes are parsed from it (`admin.ts:550, 581`).
- **Commands.** `trust.propose`, `trust.validate`, `trust.approve`, `trust.activate`, `trust.reject`, `trust.cancel` (`authentication/TrustConfirm.tsx:64`). Server: `trustDefinitions` (`apps/api/server/trust.go:331`). Permission: separate installation trust rights, not installation administration (HYB-297). Modules: authpolicy, trustchange, decisionledger.
- **States.** Draft, validated, independently approved and activated by three different people; stale sign-in handling; activation ends every session; an unconfirmed activation holds the installation to reductions until restart; providers, test login, keys, sync and recovery views marked unavailable ([HB-265](../features/HB-265/verification.md), [HYB-297](../features/HYB-297/verification.md)). The HB-265 residual "trust flow simplified" is closed by HYB-297.
- **Not connected / residual.** Providers, MFA, IAM-02 to IAM-04; owner questions Q1–Q7 of HYB-297; M02 setting for the step-up rule (HYB-293 owner question O-4). Open: HYB-304 search, filters and cursor paging (Todo).
- **Verification.** [HB-265](../features/HB-265/verification.md), [HYB-297](../features/HYB-297/verification.md).

**M03 Access grants and delegations**
- **Tickets.** HB-266 API (#186), HB-267 screen (#187), HYB-298 approval requests and revocation states (#212), HYB-293 (#210). HYB-297 (#215) changed files in `identity/pages/access`.
- **Field map.** [M03 fields](screen-binding-fields/identity.md#m03-access-grants-and-delegations).
- **Reads.** `GET /api/v1/identity/access` (`boundary.go:32`, `admin.ts:886`, `identity/pages/access/Live.tsx:383`); `GET /api/v1/identity/access/preview` per-field effects (`boundary.go:33`, `admin.ts:910`, `access/Live.tsx:255`).
- **Commands.** `access.request_grant`, `access.request_revoke`, `access.approve_request`, `access.reject_request`, `access.withdraw_request`, `access.revoke_delegation` (`access/Live.tsx:79, 160`, `access/Requests.tsx:77`). Server: `accessDefinitions` (`apps/api/server/access.go:340`), `requestDefinitions` (`accessrequests.go:426`). Modules: gate (auth), accesschange, identity, decisionledger.
- **States.** Grants bounded by the console ceiling; self-grant refused; approval by a second administrator; revocation states received, decision unconfirmed and complete; stale preview results dropped; roles and support views marked unavailable ([HB-267](../features/HB-267/verification.md), [HYB-298](../features/HYB-298/verification.md)).
- **Not connected / residual.** Delegation creation, roles, support sessions and narrower scopes; HYB-298 residuals m5, m6, n1, n2 and its "Not done or not verified" list (no independent security, render or technical-writing review within that task). Open: HYB-304 (Todo).
- **Verification.** [HB-266](../features/HB-266/verification.md), [HB-267](../features/HB-267/verification.md), [HYB-298](../features/HYB-298/verification.md).

**M04 Service accounts**
- **Tickets.** HB-268 API (#188), HB-269 screen (#189), HYB-293 (#210), HYB-316 retry persistence, and HYB-333 unknown-outcome and secret-pickup tracking (implementation candidate; combined verification pending).
- **Field map.** [M04 fields](screen-binding-fields/identity.md#m04-service-accounts).
- **Reads.** `GET /api/v1/identity/services` (`boundary.go:34`, `admin.ts:916`, `identity/pages/service-accounts/Live.tsx:551`); `GET /api/v1/identity/links/{ref}` one-time secret pickup (`boundary.go:30`, `admin.ts:931`, `service-accounts/Live.tsx:342`).
- **Commands.** `service.create`, `service.issue_credential`, `service.revoke_credential` (`service-accounts/Live.tsx:68`), `service.suspend`, `service.reinstate`, `service.retire` (`service-accounts/Live.tsx:191, 234`). Server: `serviceDefinitions` (`apps/api/server/services.go:379`). Module: identity.
- **States.** Accounts with reasons; credentials with metadata; one-time secret reveal with acknowledgement; held retry that replays a lost issue; unstored reminders; usage and transfer views marked unavailable. The "Revoke required" state after a partial retire is unit-tested, not rendered ([HB-269](../features/HB-269/verification.md), Residuals).
- **HYB-333 binding.** The services read adds per-active-credential `secret_pickup` and the estimated `secret_pickup_until`, plus `limits.replay_seconds`. `collected` means returned once, never confirmed stored. The credential-issue store retains the frozen request across form/view changes, offers same-request retry, and guards an accidental unload. The never-collected region derives from server metadata and survives browser reload; the request store stays in page memory only. See [implementation status](../features/HYB-333/implementation.md).
- **Not connected / residual.** Credentials cannot authenticate; no use history and no owner transfer. HYB-304 remains open. KO/AR and rendered behavior await combined verification; this entry is not owner acceptance.
- **Verification.** [HB-268](../features/HB-268/verification.md), [HB-269](../features/HB-269/verification.md).

**M05 My security**
- **Tickets.** HB-270 API (#190), HB-271 screen (#191), HYB-293 (#210), HYB-307 session-end manifest, and HYB-306 recent access history (implementation candidate; combined verification pending).
- **Field map.** [M05 fields](screen-binding-fields/identity.md#m05-my-security).
- **Reads.** `GET /api/v1/me` and `GET /api/v1/me/access-history`. History is scoped on the server to the authenticated user, incarnation and workspace; refused sign-ins carry no workspace. `service/access-history.ts` parses it, `AdminClient.accessHistory` requests it, and `my-security/AccessHistory.tsx` renders it after the sessions region.
- **Commands.** `POST /api/v1/me/password` (`boundary.go:48`, `admin.ts:957`, `my-security/Live.tsx:89`) and `POST /api/v1/me/sessions/end` (`boundary.go:49`, `admin.ts:960`, `my-security/Live.tsx:172`). Neither is on the command path; password changes write a log line, not a journal record (`HB-270/plan.md:158`). Module: identity, passwords.
- **States.** Read-only account view; password change with shared lockout and honest unconfirmed outcomes; own sessions with end one, end all others and sign out here; recovery and notification views marked unavailable ([HB-271](../features/HB-271/verification.md)).
- **Not connected / residual.** MFA, recovery, linking of other sign-in methods, profile changes and notification preferences. The M05 preview copy still says "reauthentication" (HYB-293 residual). HYB-306 adds persistent own-access history, period filters, bounded cursor pages, device families, and retention/recording/deletion notes; see [implementation status](../features/HYB-306/implementation.md). HYB-307 is in the current baseline. HYB-304 remains open. Combined verification and owner acceptance are pending.
- **Verification.** [HB-270](../features/HB-270/verification.md), [HB-271](../features/HB-271/verification.md), [HYB-293](../features/HYB-293/verification.md).

### Organizations and contracts (O01–O03)

**O01 Organizations and relationships**
- **Tickets.** HB-272 API (#192), HB-273 screen (#193), HYB-300 filters, search and paging (#216), HYB-318 create change plan handoff (#217), HYB-319 view group drafts (#219), HYB-293 (#210).
- **Field map.** [O01 fields](screen-binding-fields/organizations.md#o01-organizations-and-relationships).
- **Reads.** `GET /api/v1/organizations` with the as-of date, legal entity, relationship kind, unattributed, search, member and target filters (`boundary.go:36`, `admin.ts:964`, `organizations/pages/organizations/Live.tsx:160, 179`, `organizations/drafts.tsx:409`); `GET /api/v1/organizations/targets` (`boundary.go:38`, `admin.ts:1002`, `organizations/Live.tsx:185`); `GET /api/v1/organizations/view-group-drafts` and `/{id}/members` (`boundary.go:39-40`, `admin.ts:1032, 1048`, `organizations/drafts.tsx:57-78, 268, 324, 401`).
- **Commands.** `view_group.draft`, `view_group.edit`, `view_group.discard_draft` (`organizations/drafts.tsx:452, 455, 501, 565`). Server: `draftDefinitions` (`apps/api/server/groupdraftcmds.go:353`). "Create change plan" opens O02 with the inspected base revision (HYB-318); `view_group.propose` is sent from O02. Modules: organizations, relationships.
- **States.** Pinned relationship revision and as-of date in the URL; tree that places every loaded party once and names unattributed, conflicting, not-loaded and cyclic cases; filtered and unfiltered empty; next-page failure keeps rows; scan stop; denied; drafts that grant nothing, with batched saves and exact progress ([HB-273](../features/HB-273/verification.md), [HYB-300](../features/HYB-300/verification.md), [HYB-319](../features/HYB-319/verification.md)).
- **Not connected / residual.** Provider filter not connected (`provider_filter: "not_connected"`); admin-only, as the spec's group management permission does not exist in the gate; the tree is still built in the browser: HYB-320 (Todo), owner acceptance of the deferral needed; drafts reset on restart; groups created by drafts cannot become effective while O02 activation is held.
- **Verification.** [HB-272](../features/HB-272/verification.md), [HB-273](../features/HB-273/verification.md), [HYB-300](../features/HYB-300/verification.md), [HYB-318](../features/HYB-318/verification.md), [HYB-319](../features/HYB-319/verification.md).

**O02 Relationship change plans**
- **Tickets.** HB-274 API (#194), HB-275 screen (#195), HYB-296 activation held (#214), HYB-318 base revision (#217), HYB-319 view group plans (#219), HYB-322 member validation (#221), HYB-293 (#210).
- **Field map.** [O02 fields](screen-binding-fields/organizations.md#o02-relationship-change-plans).
- **Reads.** `GET /api/v1/organizations/plans` with per-plan impact (`boundary.go:37`, `admin.ts:1021`, `organizations/pages/changes/Live.tsx:448`); `GET /api/v1/organizations` for member choices (`changes/Live.tsx:42, 52`); view group drafts and members (`changes/Live.tsx:531, 536`).
- **Commands.** `relationship.propose`, `relationship.activate`, `relationship.discard`, `view_group.propose` (`changes/Live.tsx:585`). Server: `relationshipDefinitions` (`apps/api/server/relplans.go:453`), `draftDefinitions` (`groupdraftcmds.go:353`). Module: relationships.
- **States.** Four-step progress (target, impact, approval, apply); held block shown first with only discard available; impact rows "Not calculated"; stale base blocked until "Base on revision N"; only valid members for the chosen kind; closed plans with an O01 link ([HB-275](../features/HB-275/verification.md), [HYB-296](../features/HYB-296/verification.md), [HYB-318](../features/HYB-318/verification.md), [HYB-322](../features/HYB-322/verification.md)). Activation is refused while impact is not computed (`relationship.activate` returns `prerequisite_failed`), so the activated state is reachable only through test injection.
- **Not connected / residual.** Money, publication and ERP impact, the B09 approval request and scheduled apply: HYB-317 (Todo). Allowance preservation (HB-23 row at line 70 of this file) is not addressed (`HYB-300/plan.md:671`). Admin-only. Open: HYB-305 search, filters and cursor paging (Todo).
- **Verification.** [HB-274](../features/HB-274/verification.md), [HB-275](../features/HB-275/verification.md), [HYB-296](../features/HYB-296/verification.md), [HYB-318](../features/HYB-318/verification.md), [HYB-319](../features/HYB-319/verification.md), [HYB-322](../features/HYB-322/verification.md).

**O03 Contract lifecycle**
- **Tickets.** HB-280 API (#196), HB-281 screen (#197), HYB-293 (#210).
- **Field map.** [O03 fields](screen-binding-fields/organizations.md#o03-contract-lifecycle).
- **Reads.** `GET /api/v1/contracts` (`boundary.go:41`, `admin.ts:1061`, `organizations/pages/contracts/Live.tsx:66`); `GET /api/v1/contracts/{id}` (`boundary.go:42`, `admin.ts:1065`, `contracts/Live.tsx:92`); `GET /api/v1/organizations` for party names (`contracts/Live.tsx:107`).
- **Commands.** `contract.suspend`, `contract.resume`, `contract.renew`, `contract.end`, `contract.withdraw`, `contract.evidence`, `contract.settle`, `contract.reopen` (`contracts/LiveActions.tsx:170`). Server: `contractDefinitions` (`apps/api/server/contracts.go:753`), action hints at `contracts.go:521-531`, command handler `lifecycleCommand` (`contracts.go:757`). Module: contracts.
- **States.** Contracts with state on a day and current final settlement; parties and path; lifecycle schedule with segments; components by terms revision; termination with evidence; write log; actions offered only where the API hints allow; money, proration and the final amount decision stated as not computed ([HB-281](../features/HB-281/verification.md)).
- **Not connected / residual.** Money, proration, publication, ERP, documents, payment terms and recognition; single-administrator lifecycle instead of two-person review (owner residual, `HB-280/plan.md:125-133`); contract creation, terms changes and search beyond 200 contracts. Open: HYB-308 payment and recipient change control (Todo), HYB-305 (Todo). Whether the HB-23 rows for O03 (lines 71–73 of this file) are met was not checked by HYB-295.
- **Verification.** [HB-280](../features/HB-280/verification.md), [HB-281](../features/HB-281/verification.md).

### Installation and operations (P01–P07)

**P01 Installation**
- **Tickets.** HB-282 API (#198), HB-283 screen (#199), HYB-293 (#210).
- **Field map.** [P01 fields](screen-binding-fields/operations.md#p01-installation).
- **Reads.** `GET /api/v1/installation` (`boundary.go:43`, `admin.ts:1077`, `administration/pages/installation/Live.tsx:52`).
- **Commands.** `installation.approve`, `installation.withdraw`, `installation.propose_plan` (`installation/LiveProposals.tsx:78`). Server: `installationDefinitions` (`apps/api/server/installation.go:547`). Module: installer.
- **States.** Site, observation, installed state, install plan with conflicts and effects, upgrade and rollback, releases, proposals with dry-run blockers, allowlisted destinations with guided reports; every decision states that nothing is installed ([HB-283](../features/HB-283/verification.md)).
- **Not connected / residual.** Runs, install execution, key and release management, diagnostic requests, support bundles and support access; the screen derives some checks from the newest 50 proposals. Open: HYB-305 (Todo).
- **Verification.** [HB-282](../features/HB-282/verification.md), [HB-283](../features/HB-283/verification.md).

**P02 Integrations**
- **Tickets.** HB-284 API (#200), HB-285 screen (#201), HYB-293 (#210).
- **Field map.** [P02 fields](screen-binding-fields/operations.md#p02-integrations).
- **Reads.** `GET /api/v1/integrations` (`boundary.go:44`, `admin.ts:1080`, `administration/pages/integrations/Live.tsx:59`).
- **Commands.** `integrations.plugin_propose`, `plugin_approve`, `plugin_withdraw`, `plugin_stop`, `plugin_disable`, `subscription_propose`, `subscription_approve`, `subscription_pause`, `subscription_request_resume`, `subscription_approve_resume`, `subscription_revoke`, `letter_request_redeliver`, `letter_approve_redeliver`, `letter_request_dismiss`, `letter_approve_dismiss`, and since HYB-310 the author-only withdrawals `subscription_withdraw`, `subscription_withdraw_resume`, `letter_withdraw_redeliver`, `letter_withdraw_dismiss`, plus `redelivery_reapprove` (`integrations/LiveCommand.tsx`). Server: `integrationDefinitions` (`apps/api/server/integrationcmds.go`), rights table `integrationRight` (`apps/api/server/integrationrights.go`). Modules: plugins, webhooks.
- **States.** Clients, modules, channels and events views; plugin proposals and operations; subscription lifecycle; dead-letter requests; safe resends after lost responses; unknown values fail closed; a constant statement that nothing is dispatched, sent, signed or issued ([HB-285](../features/HB-285/verification.md)).
- **Not connected / residual.** Dispatch, sending, signing, token issuance, egress checks, key management, marks, filters, search, row drawers and cursor resync; each command and area needs its own integration right since [HYB-310](../features/HYB-310/plan.md) (eight installation rights; `rights`, `own_pending`, `not_permitted` areas, `no_right` and `requester_lost_right` reasons); no way to grant integration rights after setup (HYB-310 R2). Plugin license check before activation: [HYB-309](../features/HYB-309/plan.md) (`modules.licenses`, `plugins[].license`, `license_state`; the `-plugin-licenses` installation setting). Open: HYB-305 (Todo). Whether the HB-23 FA-13 row for P02 (line 35 of this file) is met was not checked by HYB-295.
- **Verification.** [HB-284](../features/HB-284/verification.md), [HB-285](../features/HB-285/verification.md).

**P03 Recovery**
- **Tickets.** HB-286 API (#202), HB-287 screen (#203), HYB-293 (#210).
- **Field map.** [P03 fields](screen-binding-fields/operations.md#p03-recovery).
- **Reads.** `GET /api/v1/recovery` (`boundary.go:45`, `admin.ts:1083`, `administration/pages/recovery/Live.tsx:60`).
- **Commands.** `recovery.approve` (unlock proposals only), `recovery.withdraw`, `recovery.lock`, `recovery.propose_unlock` (`recovery/LiveCommand.tsx:95`). Server: `recoveryDefinitions` (`apps/api/server/recoverycmds.go:178`). Module: recovery.
- **States.** Restore sets, isolation, reconciliation, migration (not connected) and resume views; actions only where the read allows them; safe resends with frozen snapshots; wording that nothing is restored, pruned or deleted ([HB-287](../features/HB-287/verification.md)).
- **Not connected / residual.** Begin, abandon, prune, writer removal, migration and AU/BL read access; all commands use installation administration; a global `bdi` rule forces reasons left-to-right in Arabic. Open: HYB-305 (Todo).
- **Verification.** [HB-286](../features/HB-286/verification.md), [HB-287](../features/HB-287/verification.md).

**P04 Retention**
- **Tickets.** HB-288 API (#204), HB-289 screen (#205), HYB-312 copies outside the controlled scope (#222), HYB-325 AI source text and Assistant conversation statement (#223), HYB-293 (#210). HYB-298 (#212) only moved the plural helper to `shared/plural`.
- **Field map.** [P04 fields](screen-binding-fields/operations.md#p04-retention).
- **Reads.** `GET /api/v1/retention` (`boundary.go:46`, `admin.ts:1086`, `administration/pages/retention/Live.tsx:69`); `GET /api/v1/identity/access` for the hold owner choice (`boundary.go:32`, `retention/LiveHoldForm.tsx:20`).
- **Commands.** `retention.approve`, `retention.withdraw`, `retention.propose_policy`, `retention.place_hold`, `retention.propose_release`, `retention.stop_run` (`retention/LiveCommand.tsx:150`). Server: `retentionDefinitions` (`apps/api/server/retentioncmds.go:382`). Module: retention.
- **States.** Policies, expiry, holds, deletion runs, restore (not connected) and decision records; frozen impact; resends that never duplicate a hold; other-scope and unknown values shown as such (model tests only, as the seed has one workspace); deletion storage results that state copies sent to ERP or downloaded are outside the controlled scope and that AI usage and Assistant data are not connected ([HB-289](../features/HB-289/verification.md), [HYB-312](../features/HYB-312/verification.md), [HYB-325](../features/HYB-325/verification.md)).
- **Not connected / residual.** Deletion plan proposals and approval, the deletion worker, retry, backup expiry, archive restore, usage and size figures and a durable decision record; all commands use installation administration. Open: HYB-311 minimal claim references after source deletion (RET-06, Todo), HYB-305 (Todo).
- **Verification.** [HB-288](../features/HB-288/verification.md), [HB-289](../features/HB-289/verification.md), [HYB-312](../features/HYB-312/verification.md), [HYB-325](../features/HYB-325/verification.md).

**P05 Support combinations and release evidence**
- **Tickets.** HYB-290 API (#209), HYB-315 screen (#211). HYB-315 merged after HYB-293, so step-up applies from the start.
- **Field map.** [P05 fields](screen-binding-fields/operations.md#p05-support-combinations-and-release-evidence).
- **Reads.** `GET /api/v1/release-evidence` (`boundary.go:47`, `admin.ts:1089`, `administration/pages/release-evidence/Live.tsx:109`).
- **Commands.** `release_evidence.approve`, `release_evidence.withdraw`, `release_evidence.retire_requirement`, `release_evidence.retire_profile` (`release-evidence/LiveCommand.tsx:164`). Server: `evidenceDefinitions` (`apps/api/server/evidencecmds.go:216`). Modules: releasetrace, perfgate.
- **States.** Separate design, implementation, test and customer proof states per combination and requirement; unexpected or positive values fail closed with a field locator; nothing reads ready, active, passed, approved or customer-proven; "Not run" and "Not measured" distinct from 0 ([HYB-315](../features/HYB-315/verification.md), Planning cross-check).
- **Not connected / residual.** Release, activation, support decisions, customer proof intake, budget approval, artifact files, the commitment event matrix (AUD-DB-09) and the concurrent load profile (SCALE-11); every command uses installation administration. Search, filters, stable sort and cursors are not built; HYB-290 and HYB-315 cite HYB-304 and HYB-305, but neither ticket's scope covers P05 (see the HYB-295 verification record).
- **Verification.** [HYB-290](../features/HYB-290/verification.md), [HYB-315](../features/HYB-315/verification.md).

**P06 Demo and scenarios: not bound**
- **Service mode.** `operations.tsx:21` sets the live switch only for P01–P05, so in service mode P06 renders the preview component `administration/pages/demo/Screen.tsx` (`operations.tsx:92-93`), which makes no API call. `boundary.go` registers no demo, scenario or simulation route.
- **Tickets.** No binding ticket; `HB-257/plan.md:168` says P06 is not yet ticketed. Linear HYB-129 "[P06] Implement 데모·시나리오 (/demo)" (In Review) is a preview implementation ticket, not a service binding.
- **Residual.** The HB-23 SR-03 row for P06 (line 38 of this file) stays open.

**P07 Licences and distribution notices: not bound**
- **Service mode.** As P06; P07 renders the preview component `administration/pages/licenses/Screen.tsx` (`operations.tsx:94-95`), which makes no API call. `boundary.go` registers no licence route.
- **Tickets.** No binding ticket; `HB-257/plan.md:168` says P07 is not yet ticketed. Linear HYB-130 "[P07] Implement 라이선스·배포 고지 (/licenses)" (In Review) is a preview implementation ticket, not a service binding.
- **Residual.** Distribution-notice evidence (HB-40) is deferred to P07 by HYB-315.


## HYB-330 diagnostic binding

P01–P05 `error:diagnostics` now displays confirmed error code and server request ID, separately labelled client request/target identity, field-location and existing next-step guidance. Refused row references live with their exact scoped marker; each open form keeps its own failure references. Unknown values remain unavailable. The API contract, rights and retry rules are unchanged. Grouped request actions use their canonical request target; terminal notices retain their own references. See [field map](screen-binding-fields/operations.md#administration-error-diagnostics-hyb-330) and [integration verification](../features/HYB-330/integration.md). No owner FIX is inferred.

## HYB-327 field-map follow-up

Identity/organization response validation and M01/M02/P02/P05 field bindings are reconciled in [HYB-327 verification](../features/HYB-327/verification.md). P01 retains a validated count summary and explicitly documented detail/override limitations. This follow-up does not certify all original screen specifications or carry forward owner acceptance.

## Phase 6 B09 read binding (HYB-337, in review)

[HYB-337 plan](../features/HYB-337/plan.md) and [verification](../features/HYB-337/verification.md) bind the existing read-only synthetic contract at `GET /api/approval-review` to the Go service. This is a read increment; it does not complete B09 approval workflows or phase 6.

| Field or behavior | Actual producer and boundary |
| --- | --- |
| Request ID, stage, target, author, digests and amount | `modules/approvals.RequestOf`; service index registers workspace, request/target IDs and explicit synthetic gross basis. Unsupported or inconsistent records fail unavailable before filtering. |
| Approval kind | `charge_approval` → `billing_result`; `erp_final_approval` → `final_erp_amount`. No inferred adjustment, policy or recognition records. |
| Candidate amount | Exact module decimal string and currency, explicitly synthetic gross basis. Before, difference, ERP values, entity, customer and period remain null when not supplied. |
| Recorded decision and actor | One final module approve/reject decision only. Current execution authority is not inferred from historical approval; deadlines remain null. Withdrawn or multi/partial-decision requests cannot be represented and fail unavailable. |
| Validation, dependencies, completeness, authority and ERP applicability | Unknown; producer prerequisites are unconnected. Pending module requests are blocked in the service projection. No unknown is replaced with zero or not-applicable. |
| Read actor and access | Current authenticated session and workspace-wide gate View grant for approval_review/approval_evidence. Synthetic grants cover three sample administrators and the console client; no member/default seed-none read grant is inferred. |
| Filters and paging | Pending, own decisions and history; request kind/search; 25 rows sorted by ID. HMAC cursor binds session, workspace, permissions, selection, installation revision and projected snapshot. |
| Recognition and delegation | Explicit unconnected state; an empty request index never asserts completed reconciliation. |
| Commands | `read_only=true`, `commands=unconnected`; no approve/reject/issue/deliver/post endpoint added. |

The reviewed EN/KO/AR layout is retained. Synthetic/read-only notices remain visible. HYB-225 still owns cross-stage and producer integration prerequisites. B09 financial acceptance references AUD-AR-01/06/07, AUD-QA-01 and ERPC-01–08 remain unestablished at system level; the read binding is not their acceptance evidence.


## B09 candidate-backed historical records (HYB-340, in review)

[Projection plan](../features/HYB-340/approval-projection-plan.md) and [verification](../features/HYB-340/approval-projection-verification.md) extend the HYB-337 baseline with actual service-owned synthetic StageCharge requests. This supersedes the baseline's unconnected-producer/pending-blocked description only for those governed records; legacy demonstration records retain that boundary.

The read projects the bounded union of the legacy index and validated immutable financial bindings directly, with a combined 1000-entry cap and no duplicate retained index. Candidate amounts use rated_candidate_total; sample gross observations remain separate. Requester and historical policy revision accompany the original author and request digest. Service Pending remains pending; approved/rejected/superseded are historical module states. Current prerequisites, authority, expiry and ERP applicability are not inferred. Snapshot/paging/session permissions remain enforced.

B09 details expose the separate actor facts and recorded policy revision in existing facts rows. Synthetic environment records are labeled as nonproduction; the current-check warning is distinct from failed-validation blockers. The historical GET remains read-only. The explicit financial_charge_v1 capability now binds selected service records to current preparation and explicit approval/rejection confirmation, with session-scoped unresolved attempts and identical retry. See [screen verification](../features/HYB-340/decision-screen-verification.md). Capability and historical facts never grant decision permission. No owner FIX or full W-23 acceptance is asserted.
