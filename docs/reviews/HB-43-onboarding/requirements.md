---
type: DesignRequirements
title: Identity, onboarding and customer contract setup
description: Defines the proposed access and first collection journey and the owner-requested customer payment and agreement fields.
status: draft
---

# Scope and ownership

[HB-43](https://linear.app/hybrid-billing/issue/HB-43) connects registration/login, permissions, initial setup, scoped collection and first billing readiness to the complete v10 review. Root is the sole writer and integrates PM/planning/QA; separate security and workflow reviewers assess the actual artifacts. Machine: mac_mini. This is static design, not authentication, connector or payment implementation. Auth product, enrollment policy, method selection and timeout values remain unresolved. Existing deferred runtime authentication work is not declared complete.

The owner additionally requested business registration number, configurable payment conditions/methods and master/child agreements on the customer detail previously at v9 page 5. That retained screen is v10 page 21 after sixteen new entry/setup states. No old screen is removed.

# Entry and authorization

The proposed self-hosted entry separates installation bootstrap, invited MSP staff, invited customer and existing member. Public uninvited signup is not enabled by this design; a future enrollment decision is separate. Bootstrap consumes an operator-held purpose/installation-bound proof atomically, creates one bounded administrator, and rejects expired, revoked, reused or concurrent duplicates. No shared default password or implicit financial approval follows installation administration.

Invitations bind intended identity, installation, membership, delegated role ceiling, resource scope and expiry. Acceptance rechecks inviter authority and current policy, consumes once and rejects mismatches; reissue invalidates prior proof. Email/domain matching never links identities or grants access automatically. Do not reveal tenant detail before proof.

Login uses the configured identity policy and generic failures. Organization-required sign-in cannot silently fall back to local credentials during an outage. Additional authentication, retry, lost-factor recovery and emergency access are separate paths. Recovery requires current approved custodians, bounded scope/time and audit; protect the last administrator. Trust replacement requires independent proposer/approver, current proof and epoch invalidation of affected sessions, step-up evidence and queued privileged actions. Concrete values need later approval and implementation testing.

Every operation must intersect current membership, role, delegated scope, customer/resource and field permission on the server. UI hiding is not enforcement. Role combination cannot bypass preparer/reviewer separation. Service identities cannot act as human approvers. Optional named issuer/reviewer fields do not waive authorization or required approval; actual actors remain recorded. Downloaded data cannot be recalled by session revocation.

| Role | Intended responsibility | Boundary |
| --- | --- | --- |
| Installation operator | Prove bootstrap and infrastructure custody | No automatic billing approval |
| Access administrator | Invite and manage permitted membership | No self-elevation beyond delegation |
| Collection operator | Connect and diagnose scoped source reads | No financial source acceptance or issuance |
| Billing preparer | Review evidence and propose calculations | No self-approval of own changes |
| Reviewer | Independently approve permitted fixed evidence | Role combination cannot bypass separation |
| Issuer | Issue approved fixed documents within scope | No replacement of prior approval |
| Customer viewer/auditor | Authorized customer data and explanation | No MSP source secrets or operator costs |

# Onboarding and collection

Follow existing OP05 ordering: legal entity/currency/time/period → customer organization/account attribution → issuance and ERP → cost/selling policy → FX/tax → validation and approval. Users may save drafts and resume; completed steps depend on the latest valid diagnostic's configuration dependency digest, product version, observed evidence and expiry. Changed or expired dependencies invalidate only affected downstream readiness. A handoff is assigned, scope/action limited and expiring, with no secret copied into the link.

Temporary setup rights, provider export writer and collector reader are separate custody classes. Confirm setup-right removal without removing required export writing, then prove actual scoped reads. Credential storage, endpoint reachability, authentication, authorized read, source arrival, schema parsing, account/period completeness, reconciled totals and accepted billing evidence are distinct states. Green jobs and FOCUS conformity alone never establish billability.

Show absent export, denied permission, generation pending, unsupported period and evidence-backed zero usage separately. Partial/unknown input is never zero. Keep source/target format versions, extensions, conversion and functional limitations visible. Bounded backfill, immutable input revisions, duplicate detection, checkpoints and authoritative read-back precede retries with unknown outcomes. Schedule activation requires a valid source profile and scoped read evidence.

Connection destinations require approved validation against metadata/loopback and unsafe redirects; credential references are not credentials and logs redact secrets. Customer attribution changes do not authorize historical transfer. Transfer needs the separately governed recipient package, acknowledgment, source-access revocation and pending-work handoff; imported issued history is read-only and cannot be reissued as new work.

Initial setup never guesses legal entity, currency, tax, FX or unit pricing. Connection, collection, evidence acceptance, calculation and issuance have separate readiness gates. ERP setup is not successful external posting, and an entitlement cannot bypass financial/security gates.

# Customer identity, payment and agreements

The customer detail retains identity, billing contact, address, currency, contract, renewal, notice deadline and linked source scope. Add business registration number with jurisdiction and identifier type, separately from tax ID; missing and not applicable are distinct. Use synthetic identifiers or blank values in the review.

Payment conditions are composable dimensions, not one overloaded dropdown:

| Dimension | Required options and configuration |
| --- | --- |
| Pricing basis | Fixed, usage-based, hybrid; metered usage does not imply monthly or postpaid |
| Billing cadence | Monthly, quarterly, semiannual, other/custom; explicit period anchors and boundaries |
| Timing | Prepaid or postpaid, with effective contract version |
| Settlement schedule | Single or advance/interim/balance; stage trigger, amount or percentage, due-date basis and currency |
| Due date | Applicable issuance/milestone basis, agreed days, time zone and calendar; no universal Net 30 default |
| Deposit | Optional deduction combined with other terms, confirmed available balance, allocation priority and insufficient-balance handling |
| Payment method | Bank transfer, credit card, electronic promissory note, other; allowed methods and default, not execution |

Validate stage totals, exact currency/rounding and residual allocation before activation; unresolved terms block affected issuance. Planned deposits are not received funds. Deposit allocation is a separately authorized auditable settlement event, not a price discount, revenue recognition or a second cash authority. Unknown/stale balances do not become zero. Preserve reversal and idempotency evidence under the existing payment contract. An electronic note's receipt does not itself confirm final settlement; maturity, clearing and reversal depend on approved authoritative evidence. Do not store card/bank secrets in these screens.

Master agreement status is none/present/unverified. Present requires linked stable ID, signed/validity state, legal parties, effective version and dates. List child agreements by ID, scope, status and dates. Show which terms are inherited, from which master revision, and which are explicitly approved overrides. A parent link alone proves neither signature nor universal applicability. Reject cycles and cross-scope links; conflicts or missing required terms block affected readiness. Master edits affect approved future/draft versions, never rewrite issued document snapshots. Recognition cadence remains independent from customer billing and payment cadence.

These additions expand the design options beyond the older payment contract's deferred installment scope. They do not approve a particular customer's schedule, default percentages or finance/ERP accounting treatment.

# Customer-visible product and instance pricing

Owner feedback: customer viewers currently have to reverse-calculate percentages or ask sales which discounts apply to their most-used products. Required outcome: display the contractual percentage and actual application directly in a product overview, with instance-level search and an understandable charge breakdown. This is a delivery checklist requirement as well as a mockup requirement.

Extend contract pricing operations (v9 p15 → v10 p33), operator evidence (v10 p59) and customer explanation (v10 p78). Search by product, instance name/stable provider ID and billing month within current authorized customer/account scope; use server-side scoped filtering and pagination at production scale. Product/type/SKU, region, usage period, currency, units, pricing basis, contract effective version, selected rate/percentage, discount amount and resulting charge must be retrievable. A display name cannot serve as a unique resource identity. Preserve shared charges and lines lacking resource attribution separately; never invent an instance to make a total fit.

Show applied, no discount, excluded, pending/unknown and blocked states. Explain product-discount exclusions, nonstacking rules and component limits. Do not display unknown as 0% or advertise every line as calculated while some are blocked. Customer fields include only permitted selling evidence, never supplier purchase rates, margin, internal rule references or another customer's terms. Backend, cache, exports and search suggestions must enforce the same current scope; UI visibility alone is insufficient.

Contractual percentage is not an inferred effective percentage. When a comparable positive basis is known, effective discount = (basis minus comparable discounted selling amount) / basis, with named components and currency/period. Explicitly identify credit inclusion/exclusion and avoid counting benefits twice. A zero/unknown/incomparable basis gives not applicable/unknown, not 0%. Fixed, tiered, commitment allocation and credit-based outcomes expose their actual formula and allocated benefit instead of a fictitious uniform percentage. FX/tax changes must not masquerade as discounts.

The static example expands the existing USD 1,000 DB charge as 100 hours × USD 10/hour, minus 10% (USD 100), giving USD 9/hour and USD 900 pretax. It is illustrative, not verified provider pricing. Existing full-period total remains USD 1,535; the searched DB subset is USD 900. Filtering and paging must not change pricing pools, thresholds, credit allocation or frozen totals. Multi-line/resource summaries reconcile to underlying lines and final document totals with explicit common charges, tax and rounding differences.

Estimated, approved and issued amounts are distinguished. Issued explanations use frozen source/calculation/contract/document revisions; current contract edits cannot rewrite history. A resource spanning rate changes must show separate effective segments. Customer disclosure approval is separate from operator draft pricing. No runtime search, automatic computation or customer authorization is claimed by the static screens.

[Delivery checklist](checklist.md) tracks every owner-requested workstream separately from mockup status. UI presence is never a checked runtime acceptance result.

# AI costs, stacked discounts and invoice detail

The owner's operational requirement is to eliminate manual explanations of CSP commitment benefits, provider additional discounts and MSP selling concessions for sensitive AI resource costs. Two additional states, operator AI cost waterfall (p60) and customer AI invoice detail (p79), expose this chain. Five additional cost-exploration/Marketplace states and their detailed boundaries are specified in [feedback review](feedback-review.md). The full book now has 89 pages: 66 retained plus 23 added. The prior instance search stays at p78.

The independent AI-DEMO-01 illustration uses 100 GPU hours at USD10/hour = USD1,000 selling basis. A fictional approved CSP commitment pass-through of 10% removes USD100; a fictional provider/Google additional 5% of USD900 removes USD45; an MSP special 5% of USD855 removes USD42.75. Final selling subtotal is USD812.25, net unit rate USD8.1225/GPU-hour, total concession USD187.75 and effective reduction18.775%. Do not sum the percentages. Round each discount stage half-up to two USD decimals, preserving unit-rate precision; other contracts need their own pinned rounding rule. This is not a researched assertion that actual Google terms stack in this order.

Separately evidenced synthetic procurement cost is USD800. It is not inferred from the selling discount chain. The USD12.25 difference is selling subtotal minus allocated acquisition cost, not net profit or cash. Source-net benefit IDs, allocation, eligibility, rule version, order and pass-through policy must prevent duplicate deduction. Supplier credits, amortized commitments, upfront charges, reseller incentives and customer concessions are not interchangeable. Unknown provenance or unresolved overlap blocks the affected calculation.

Default customer access omits procurement and margin. Where an explicitly approved open-book policy allows it, show only authorized acquisition-cost fields bound to recipient, source, version and effective approval. The customer AI screen illustrates this approved case, not universal disclosure. Apply the same rule to search, suggestions, caches, PDF and tabular exports; supplier private agreements and other customer terms remain excluded. Keep an approved customer-scoped reference distinct from internal raw-source IDs.

The AI statement is an unissued draft; tax is unconfirmed and final payable/issuance is blocked. Its only charge totals USD812.25 pretax. Invoice PDF and CSV/XLSX details must reconcile to the fixed document revision, with per-resource/SKU quantity/unit, basis, every applied discount, excluded rules/reasons, tax, rounding and final charge. Export generation is an implementation requirement, not a working button in this book. GPU hours, input/output/cached tokens, requests, storage and support retain native metering units; unsupported or unattributed data remains visible. No what-if sensitivity tool or actual AI provider adapter is claimed by this illustration.

# Trace and acceptance

Pages 3–10: entry, bootstrap, invitation, login, challenge/recovery, authorized workspace, role administration, sessions/revocation. Pages 11–18: restartable progress, business profile, attribution, connector choice, read diagnosis, schedule/backfill, first collection, readiness. Page 21: expanded customer detail. Remaining billing, contract pricing, PDF/AI reconciliation, issuance, history, customer and license coverage is retained.

Artifact acceptance: both languages contain all 89 pages; preserved IDs match all 66 v9 IDs; navigation and separate customer review contain no broken links; print pages fit, mobile root does not overflow, and no remote network request or active form is introduced. Visual inspection supplements automated bounds checks.

Runtime acceptance remains NOT_RUN: duplicate invitation/bootstrap races; expired/revoked/wrong-recipient acceptance; cross-tenant access and role reduction during queued work; self-approval; IdP outage and recovery; stale diagnostic invalidation; redirect/secret exposure; export-reader custody; missing/partial coverage; duplicate/backfill/unknown outcome; milestone/deposit allocation/reversal; parent conflict/unauthorized override; immutable issued payment terms.

# Evidence

- [Identity/session/permission boundaries](../../architecture/security-boundaries/identity-session-permissions.md)
- [Access and trust](../../contracts/security/access-and-trust.md)
- [Source policy and onboarding obligations](../../contracts/operations/source-policy-onboarding-and-verification-obligations.md)
- [Source relationship bindings](../../architecture/data-ownership/source-relationship-bindings.md)
- [Payment and recognition](../../contracts/billing/payment-and-recognition.md)
- [Prior automatic pricing requirements](../HB-42-auto-pricing/requirements.md)
- OWASP [Authentication](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html) and [Authorization](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html) support generic failure responses, reauthentication and deny-by-default current authorization. They do not select this product's authentication vendor.
