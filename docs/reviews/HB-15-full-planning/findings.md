---
type: Planning Review
title: Full planning review after reconciliation
description: Records a newly identified fetch-policy conflict and the remaining canonical integration and product-contract work across the complete planning scope.
status: draft
---

# Full planning review

Tracking: [HB-15](https://linear.app/hybrid-billing/issue/HB-15). Reviewed Git baseline: `76102ddd2653e88c383fe8dcc6b9d6c19da71e90`, supplemented by the authorized local v1.16 source and screen specifications. Accountable owner: Seung Woo Park. Root integrated separate architecture/security, DBA and coverage/UI/QA reviews. See the [brief](brief.md) and [verification](verification.md).

**Overall planning-completeness verdict: FAIL; planning is not ready to be declared complete.** The supplied boundary and financial contracts show substantial progress. The main unfinished work is concrete product-contract definition and reconciliation into authoritative work, screen, command and acceptance records. This is not a finding that provider scope must be reduced or that every future customer's configuration must be known now.

There is one newly identified contract conflict within existing ED-03. Other findings below are known integration/decision work, including a newly localized U02 ambiguity. Their identifiers are report references, not new product requirements, new PC/ED findings or Linear tickets. Severity describes documentary risk if implemented prematurely, not observed production damage.

## Established scope and controls

- All 43 requirement groups remain in scope, including AWS, Azure, GCP, OCI, Alibaba, VMware, HawkEye and TokenMeter. Conditional activation and later capabilities retain their original design obligations.
- Source preservation, normalization, missing/duplicate checks, reconciliation and correction revisions are already requirements. Accuracy first, mandatory independent cross-validation and operator confirmation are owner-confirmed directions; exact confirmation interaction and authority remain proposed.
- Provider cost, customer sales, internal allocation, recognized amounts and cash/receivable evidence are distinct. An accepted request is not final ERP posting; input confirmation is not amount approval or issuance.
- The default calculation order, FX precedence, common XLSX/manual-result path, payment terms and monthly recognition independent of customer invoicing already exist. They are not new questions for the owner.
- Current authorization applies across API, workers, queries, exports and derived views. Immutable issued history, linked corrections, independent recovery controls and conditional training restrictions remain required.

## Findings and required closure

### FR-01 — MEDIUM: fetch exception wording weakens an original deny rule

**New conflict within ED-03.** [Access/trust contract](../../contracts/security/access-and-trust.md), line 42, permits metadata/loopback/link-local access through a separately reviewed capability. Original source B0189, B0725 and B1428 instead block metadata access and, in B0725, loopback and unapproved management networks while permitting approved private on-prem destinations. Local source evidence: `.local/planning/source.txt` lines 266, 999 and 1911; [AR-02](../../planning/v1.16/detailed-requirements.md#ar-02), lines 39–41, preserves that deny requirement.

A future connector designer could read capability review as sufficient authority to allow a metadata endpoint for a follow-on fetch. This conflicts with the original prohibition. The supplement explicitly creates no current network exception at line 44; no exploit or deployed exposure was tested or inferred.

**Closure:** retain explicit metadata/loopback denial and separately define approved private destination support. Do not equate every link-local destination with cloud metadata. Changing an original prohibition requires an explicit owner decision and independent design review; ordinary profile review is insufficient. Add source-linked negative cases for initial and redirected/reference fetches.

### FR-02 — MEDIUM: input confirmation is not integrated into the canonical flow

**Known INT-03.** [Collection design](../../architecture/data-collection.md), lines 65–75, and [financial contract](../../contracts/billing/inputs-pricing-and-erp.md), line 16, require evidence-bound confirmation before eligible publication. [Domain contracts](../../planning/v1.16/domain-contracts.md), line 48, still abbreviates collection as completeness validation followed by usable input; detailed S03/S06/B09 specifications and work/acceptance records do not yet carry the complete new gate.

An implementer following only those canonical artifacts could omit confirmation or preserve it after a mapping/input revision changes. The supplement already prohibits both, so this is integration work rather than a new approval requirement.

**Closure:** synchronize candidate/blocked/awaiting-confirmation/eligible states; comparison evidence and actor authority; revision binding and invalidation; races, retries and correction behavior across requirements, commands, screens, work and acceptance. Never allow a click to waive absent coverage or grant amount/issuance authority. Keep the interrupted optional-capability phrase unresolved.

### FR-03 — MEDIUM: trust-root and operating refinements remain outside detailed screens

**Known INT-04/05.** [Access/trust contract](../../contracts/security/access-and-trust.md), lines 24–30, requires independent trust-root activation and authority epoch handling. [M02 specification](../../ui/screens/M02/spec.md), lines 85–87, provides generic activation/rotation without the corresponding replacement-authority flow. [Decision register](../../planning/v1.16/planning-decisions.md), lines 48–50, already records M02, S02 metrics and U01 favorites integration as pending.

A generic activation action is insufficient to express the independent actor, old/new trust digest, proof and session/epoch invalidation. Likewise, connection success cannot substitute for first verified-bill metrics, and stored favorites cannot bypass current access.

**Closure:** reconcile M02 trust replacement, S02 cohort/milestone/window/wait semantics and U01 retention/regrant/current-access behavior into their fields, actions, command schemas and tests. B11/I04 independent hold commands also remain within INT-05: inquiry resolution, hold release, delivery and financial effects must stay distinct. Re-review the concrete changed artifacts. Do not count supplied supplemental prose as final PC/ED closure.

### FR-04 — MEDIUM: financial product mechanisms remain underspecified

**Known D-03 / PC-01 / ED-04.** [Billing model](../../architecture/data-ownership/billing-data-model.md), line 55, and [requirements](../../planning/v1.16/requirements.md), line 385, leave exact economic Claim identity, partial consumption, expiry and correction behavior unresolved. [Financial contract](../../contracts/billing/inputs-pricing-and-erp.md), lines 35 and 47–54, leaves numeric bounds/scales, supported expressions and exact analytical processing profile for selection.

For a 100-unit service with 60 units consumed, a correction/retry must neither reopen all 100 nor block the remaining 40. Existing invariants prohibit these outcomes, but the fields, overlap representation and command/state transitions remain to be defined. Similarly, division/truncation or chunk-dependent residual assignment needs explicit deterministic behavior and independent expected answers.

**Closure:** specify supported numeric/DSL mechanisms and rejection boundaries; Claim key/partial representation and reservation/release/correction transitions, including unknown ERP outcomes; independent order/rounding/overlap cases. Keep actual customer rates, tax, FX and approved accounting mapping in versioned profiles. Missing product mechanisms are not merely missing customer values, and this review does not invent universal financial defaults.

### FR-05 — MEDIUM: U02 deadline changes lack an explicit stage target

**Newly localized integration ambiguity within HB-31 / D-07 / INT-06.** Original B1376/B1377 distinguishes collection, calculation, approval, issuance, delivery and payment schedules. [U02 specification](../../ui/screens/U02/spec.md), lines 43–51 and 66, lists cycle-level original/current deadlines and a schedule-change action without an explicit stage/schedule identity. Its stage filter and checklist do not specify which deadline the change command modifies.

The product-level requirement remains correct; the detailed interaction is ambiguous. A delivery postponement must not silently alter approval deadlines or an issued invoice's effective payment due date.

**Closure:** key each deadline and change request by cycle, stage/schedule identity and expected revision, retain per-stage history and authority, and link payment-date changes to the separate approved extension contract. Add a case proving a delivery deadline change leaves approval and payment dates unchanged.

### FR-06 — MEDIUM: structural coverage is not complete semantic or portable trace

**Known INT-01/05/06 and PP-08.** Mechanical checks retain 43 groups, 43 work items, 129 D/I/V task IDs, 35 detail groups, 210 existing acceptance IDs and 49 screens. This proves reference integrity within the checked local inputs, not complete integration of subsequent UD/H11/H13/KO additions or every clause in the original source.

At the reviewed baseline, only 4 of 31 files under `docs/planning/v1.16` and none of 147 files under `docs/ui/screens` are Git-tracked. Local review is possible; a fresh clone cannot reproduce the whole planning baseline. This limitation is already acknowledged, not a discovery of lost source. [HB-14 reconciliation](../../planning/v1.16/planning-reconciliation.md) explicitly calls itself an index rather than a full detailed replacement.

QA also found that B08/B09/B11/C01/C02/I01/I02/I03/O03/R01/U01/U02/U03 specifications omit some payment/recognition-added HB identifiers present in their JSON summaries, although the substantive appended requirements are present. This is trace cleanup, not proof of missing functionality.

**Closure:** produce the complete English requirement/action/command/screen/work/acceptance trace with source provenance and preserved historical records. Reconcile supplementary case families and local screen identifiers; publish the authorized baseline through scoped reviewed tickets. Verify the eventual fresh-clone package before claiming portability. Do not change historic counts to imply those past reviews checked later additions.

## Decisions and evidence, not additional defects

| Class | Remaining work | Correct completion boundary |
| --- | --- | --- |
| Product-global choices | Exact command/permission registry, supported numeric/DSL/Claim contracts, disclosure schema, retention/restore mechanism, installation alternatives and component comparison | Root prepares concrete alternatives and domain review; consequential choices receive owner acceptance |
| Customer configuration | Roles/ceilings, rate/tax/FX/payment profiles, calendars, retention durations, IdP/MFA, allowed destinations, workload and recovery targets | Define required fields, validation, authority and unknown/unsupported behavior now; actual approved values gate the relevant installation/path |
| External evidence | Exact provider/ERP/IdP/runtime versions, source identity/coverage, posting/lookup capability, rights and licenses | Record dated evidence and unsupported conditions; actual runtime certification follows authorized development |
| Owner clarification | Interrupted optional capability; named financial/security/data/operations authorities; VM-first validation wording versus conditional existing-Kubernetes preference | Recover available evidence and present concrete alternatives before asking; do not ask the owner to reselect documented providers |

The [D-01–07 register](../../planning/v1.16/planning-decisions.md) remains the current classification. Legacy `delivery-plan.md` D01–D11 are a different reference set; qualify the source document when referencing them and reconcile them during PP-08. No cross-register equivalence is assumed. Historical review verdicts remain baseline-specific.

## Recommended remaining work sequence

This is root's recommendation derived from review evidence, not JEV product advice or a reduction of scope.

1. Resolve FR-01 source/supplement wording and integrate the mandatory confirmation and trust-root contracts (FR-02/03) with concrete negative cases.
2. Specify the remaining Claim/numeric mechanisms (FR-04); integrate operations, payment/recognition and the stage-specific deadline contract (FR-05). Work can proceed across the entire provider matrix without selecting a reduced first target.
3. Complete detailed English canonical trace and publication (FR-06), preserving all original and owner-added obligations and explicit conditional activation.
4. Prepare decision packets with named responsibility and evidence; obtain only genuine owner choices. Perform final independent closure review against changed artifacts and explicit remaining conditions, then request the owner's separate planning-complete declaration.

These are follow-up work packages, not tasks executed by this review. No implementation, live provider/ERP tests, benchmark, legal validation or final PP-09 acceptance occurred. No existing PC/ED finding is closed by this report.

**JEV recommendation:** use `review`, with uncertain `standard` depth. Root instead determined review breadth from the full planning request and monetary/security risks, using independent domain design review and technical-writing integration. JEV granted no execution or acceptance authority.
