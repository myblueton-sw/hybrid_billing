---
type: Decision Register
title: Planning decisions and integration actions
description: Identifies concrete unresolved product choices customer configurations evidence needs and source-backed integration work without reopening established requirements.
status: draft
---

# Decisions and actions

Tracking: [HB-14](https://linear.app/hybrid-billing/issue/HB-14). Applies to the [full reconciliation](planning-reconciliation.md), including all providers and original conditional/later requirements. The HB-11 D-01–07 identifiers are retained. Their earlier first-profile wording now means evidence/activation sequencing, never permission to remove design targets.

Seung Woo Park is the accountable owner for obtaining decisions. Finance, security, data and operations below are requested domain responsibilities, not named or completed assignments. Root prepares evidence and proposals; independent reviewers assess contracts. No response, AI review, PR merge or expired waiting period is approval.

## Decision classes and completion evidence

A packet can contain several classes; classify each selected field when it is resolved.

| Class | Meaning / next action | Completion evidence |
| --- | --- | --- |
| Product contract | Root specifies supported behavior/mechanism or alternatives, domain reviewer checks, owner accepts consequential choice | Selected alternative, rationale, scope, effective revision and review |
| Customer configuration | A required parameter varies by installation/customer/contract; the product validates and versions it, never invents a universal default | Named authorized customer/domain decision, value/unit/bounds, approval and effective dates |
| External evidence | A source/ERP/IdP/runtime capability must be proven for the exact version/account/environment | Source-dated official evidence, profile manifest and later actual integration evidence; documents are not runtime proof |
| Owner clarification | The instruction or authority is genuinely missing/ambiguous | Explicit owner answer with affected scope; do not repeatedly ask about facts already in planning |
| Integration | Known semantics must be reconciled across source/work/API/screens/acceptance | Exact links/IDs, coherent changed artifacts, source check and independent re-review |

Customer values and external evidence need not all exist for every future customer before product design can progress. Planning must specify required fields, validation, unsupported/unknown behavior, authority and activation gates. Customer onboarding supplies actual values before its path becomes eligible. Final planning acceptance must explicitly state any remaining conditional decision; no universal support is inferred.

## Current packets

| Packet / status | Already established; do not reopen | Actual unresolved fields and next work | Blocks |
| --- | --- | --- | --- |
| D-01 OPEN: operation, ownership and product choices | MSP/enterprise/group share a core; all eight collection families; self-hosted delivery; source-available commercial direction | Product: compare stack/storage choices and installation alternatives from workload evidence; define responsibility matrix. Customer: actual entity/payment/operating relationships, operators and budgets. Owner: name decision owners; resolve VM-first source wording versus conditional existing-Kubernetes preference. Rights-holder, final commercial/SDK terms and pinned third-party rights need separate rights evidence | Final installation/ownership/rights profile; not documentation of other providers |
| D-02 OPEN: source/detail/ERP evidence | Immutable permitted evidence, full-scope completeness, monetary basis distinctions, authority-specific reconciliation, complete authorized detail and common XLSX/manual results | Product: exact source identity/mode/precedence/mapping and disclosure schema. Customer: account/contract/region/source permissions and field disclosure. External: versioned endpoints, limits, supplier originals, complete exports and ERP draft/post/lookup/upload capabilities across the full matrix | Eligible input and certified adapter activation |
| D-03 OPEN: financial policies | Default calculation order and AR-06 FX precedence already exist; zero unexplained difference; Claim uniqueness; explicit financial approvals; monthly recognition differs from billing | Product: exact numeric/DSL/Claim/rounding/residual contracts and independent expected cases. Customer/finance: rates, tiers, pools, proration, actual FX/tax, warning predicates/ceilings, cash/receivable authority, payment terms and ERP recognition mapping. External: guarded posting/recognition/dedup evidence. Preserve named authorization and provenance, not invented tax values | Policy activation and final financial approval |
| D-04 OPEN: retention | Class/scope/dependency retention; legal holds; original/effective clocks; active deletion differs from backup expiry; no resurrection | Product: policy schema, dependency/hold races, archive/deletion evidence and key lifecycle. Customer/domain: durations/basis dates/holds/privacy justification. External: object/backup/replica/key capabilities and contractual obligations | Claims of guaranteed retention/deletion and eligible destructive operations |
| D-05 OPEN: capacity and recovery | More than 1M incoming records/day; at least 30M/month; minimum 100k-row bulk cases; 100k managed-user axis is separate from concurrent sessions; coherent restore/fencing | Product: bounded processing and measured test plan. Customer: row sizes/peaks/skew/retention/concurrency/closing windows, hardware, RPO/RTO/SLO. External: actual deployment and fault/recovery evidence after development authorization | Sizing/performance/recovery certification; not fixed scale premises |
| D-06 OPEN: identity, commands and intelligence | Current scoped authorization, existing financial segregation, governed trust-root change, common command boundary; LLM cannot calculate/approve money; no implied tenant training | Product: command/permission IDs, schemas/versioning, role ceilings, auth/session/revocation/trust policies, plugin signing/compatibility, exact input-confirmation interaction. Customer: IdP/MFA/LDAP/egress and approved data scope. External: compatibility and bounded enforcement evidence. Owner: clarify interrupted optional capability; training inclusion requires separate policy, not inference | Sensitive command/model activation and final permission contract |
| D-07 OPEN: operations and screens | Separate dispute/collection/delivery holds, deadlines, statuses, personal permission-safe favorites, onboarding milestones, payment/overdue semantics | Product: supported cohorts/denominators/timing, retention/regrant favorites behavior, saved filters/search, notification/hold boundaries and UI/API trace. Customer: calendars/channels/escalation, hold/extension authority, receipt freshness and refresh targets. External: actual ERP hold/result capabilities | Final operating profile and corresponding screen/action acceptance |

## D-08 OPEN: SaaS subscription extension

Tracking: [HB-18](https://linear.app/hybrid-billing/issue/HB-18/plan-saas-subscription-and-hybrid-billing-management); [RA-15 supplement](../saas-subscription-billing.md). Established direction: include future SaaS subscription and billing management in planning. SaaS vendors billing customers is the working interpretation; hosted Hybrid Billing delivery and purchased-SaaS procurement remain separate decisions. Existing scope, human financial approval and self-hosted delivery remain intact.

| Class | Required decision or evidence | Activation/completion gate |
| --- | --- | --- |
| Product contract | Supported lifecycle transitions, price migration, recurring/seat/meter/hybrid semantics, proration/cadence rules, cancellation races, entitlement synchronization, one issuer/cash authority and versioned common commands | Independent billing/security review; reconcile D-03/06/07 and canonical source before final contract acceptance |
| Customer configuration | Plan/rate/currency, seat definition, included allowance/reset/pooling, billing anchor, advance/arrears, trial/grace, refund/credit terms, current authority, payment consent/mandate and tax/finance mapping | Versioned approved profile with required values; absent values block the affected charge/action, not documentation of the full capability |
| External evidence | Meter completeness and seat history, payment-provider authentication/idempotency/lookup/refund behavior, service entitlement acknowledgements, ERP support | Exact account/version/environment evidence; unknown outcome reconciles before retry; unsupported automatic paths stay disabled |
| Integration | Map SB-01–10 and SB-A01–16 to existing source requirements/work, screens, command/permission IDs and test plans; define MRR/ARR/churn separately from revenue recognition | Published supplement and Markdown overlap supplied by HB-18; detailed local source/API/screen trace still pending; source group/screen counts unchanged |
| Owner clarification | Accept the concrete capability interpretation and consequential product policies, name finance/security/service owners and decide release timing | Explicit decision; neither silence nor a PR merge constitutes policy approval or planning completion |

Monthly ERP cost recognition stays under the existing contract; SaaS revenue/deferred-revenue design is an additional finance mapping, not inferred from annual subscriptions. No processor, pricing default, automated debit or legal/tax regime is selected by this packet. Root coordinates; Seung Woo Park is accountable for decisions; domain labels are not completed assignments.

## Integration queue and responsibility

The following are work packages, not invented Linear tickets. HB-14 supplies the crosswalk and English financial addition. Further independently deliverable edits need matching tickets before execution. Root remains the single writer of shared canonical files.

| Action | Current evidence / concrete next artifact | Owner and review | Status |
| --- | --- | --- | --- |
| INT-01 | 43 groups, existing work/detail/screen/acceptance references and source hashes in planning-reconciliation.json; retain all detailed clauses in subsequent English baseline | Root; architecture/DBA coverage review | Index supplied; full detailed English baseline pending |
| INT-02 | Payment Terms and monthly recognition English contract with all PT/ERPC scenarios; preserve local original and exact source IDs | Root; DBA plus source/trace QA | Supplied in HB-14; runtime NOT_RUN |
| INT-03 | Integrate H13 confirmation into S03/S06/B09 fields/actions/stale/blocked/current-authority cases, linked work and planned acceptance | Root using UI skill; architecture/DBA review | Contract exists; full canonical screen/work/API integration pending |
| INT-04 | Integrate ED-02 trust epoch/independent authority into M02; PC-06 metrics into S02; PC-07 favorites into U01 | Root; independent security/QA review | Supplemental contracts exist; canonical integration pending |
| INT-05 | Reconcile PC-01–07 and ED-01–06 across requirement detail/work/API/screens/test semantics; retain historical review verdicts | Root; relevant domain review | Final finding closure remains OPEN |
| INT-06 | Prepare complete requirement-to-screen/action/API/test matrix, source provenance, English detailed baseline and portability check | Root technical writer; independent coverage QA | Partial index only; local required inputs still unpublished |
| INT-07 | Independently inspect all resolved decisions, explicit remaining conditions and full source coverage; present owner acceptance package | Separate architect/DBA/QA; root integration | PP-09 pending after evidence and integration |

## Questions reserved for real owner decisions

Do not send this table as a repeated questionnaire. Root first prepares concrete alternatives or recovers available evidence. Confirm only missing authority/preferences that affect the current dependent work.

- Clarify the interrupted optional-capability statement; keep dependent exceptions unresolved meanwhile.
- Obtain named finance/security/data/operations decision owners; role labels are not assignments.
- Resolve the installation ordering conflict with a documented comparison; no silent Kubernetes or VM adoption.
- Accept product-global behavior choices after review; obtain customer-specific values from the proper profile owner instead of forcing global rates, retention or tax defaults.

JEV recommendation for this turn was no selected skill and uncertain depth. This register's classifications and next actions come from source/contract analysis and independent domain review, not a JEV product decision. All runtime tests remain NOT_RUN; planning completion is still the owner's separate decision.
