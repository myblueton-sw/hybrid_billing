---
type: ImplementationPlan
title: Governed payment terms for source-bound issuance
description: Connect actual independently approved calendar terms and contract coverage to the retained issuance workflow.
status: draft
---

# Outcome and ownership

HYB-341, machine:mac_mini. Root owns implementation and integration; refusal_review is the independent read-only financial and authorization reviewer. This extends the real empty receivables registry assembled at308c489. Deliver actual owned RecordTerm, ApproveTerm, Bind and ApproveBinding operations, then use their current coverage in source-derived draft approval. No default payment days, copied approval privilege, external issue, payment or delivery is authorized.

Owned files are a narrowly scoped term-permission module, four closed action additions in modules/auth and modules/gate, necessary server grant-authority/catalog/initialization adapters, new financialissuanceterm*.go files and focused tests, plus this plan. Existing generic grant boundaries remain closed. Public commands and screens follow the independently reviewed service contract; this plan alone does not approve them.

# Independently reviewed authority contract

Independent architecture review rejected reusing financial RequestApproval/Approve and financial DirectHumanMode for payment-term configuration. Implement four new closed, nondelegable direct-human actions: record_issuance_term, approve_issuance_term, bind_issuance_term and approve_issuance_term_binding, all on synthetic_issuance_terms. Capabilities record, approve, bind and approve_binding map one-to-one to those actions. All four require an independently governed ConsoleClient ceiling and actual human grant; use actual-session gate.Decide with ConsoleClient and preserve the console intersection. The generic grant registry remains in CeilingMode. No grant is seeded. The authority domain has its own version and grant prefix. Monetary inspection remains a separate future Cost-field capability, absent from this initial four-action domain.

Reuse the existing two-governor grant protocol and exact binding/read-back. Installation administration provisions permissions but cannot authorize payment-term writes. Reject generic grant creation for every dedicated action or the dedicated resource kind. Current human session/admission and exact owned grant must precede module operations. Recheck recorded authors and deciders using current subject authority inspection; do not fabricate their login sessions. Independent business decisions enforce distinct actor IDs in addition to their distinct rights. Prior ERP-final, charge, pricing or result rights cannot substitute.

The independent reviewer approved this mapping as the next authority boundary with the actual four-operation receivables composition still required. Protected channels/changes and draft operations will use their own closed direct-human actions and console ceilings. Protected rule proposal/confirmation may use the separately reviewed two-governor configuration boundary, without granting channel, verification or value-change rights.

# Explicit term facts and scope

Require caller-supplied term ID, calendar mode, integer net days, positive revision and rationale. Net days remain within the actual registry limit; unsupported working-day modes refuse. Store only synthetic configuration facts. A deterministic service-owned term ID prevents adoption of an unowned matching record. A retained binding identifies the exact module digest, author, intended decision actor and state.

A contract binding additionally names an owned confirmed issuance profile, exact profile digest, explicit from/to dates, timezone, reason and an owned approved term revision. Derive workspace and contract identifier from the profile's current approved source; do not accept a caller-selected contract or billing amount. Preserve exact source contract and billing revisions in the service binding. Compare the named zone with the explicit profile zone; refuse mismatches rather than infer jurisdiction or currency from language. Calendar coverage can be reused across economic periods only through the same owned approved contract binding whose contract and source-party revisions remain applicable. Do not create a separate overlapping term binding for every invoice.

Before approving a binding, recheck current profile/source, owned term approval and current standing of its author/approver, then read the exact module binding and use the actual ApproveBinding overlap/zone checks. Pending overlapping bindings remain a conflict; no automatic withdrawal or shortening is part of this slice. Initial changes are append-only; effective-date modification/withdrawal requires a subsequent reviewed protocol preserving issued history.

# Recovery and consumption

For each module mutation reserve exact immutable intent and ownership before the first write, with bounded capacity. A missing publication after a successful module write stays unknown. An unknown operation keeps its original ID, contract/date interval and actor; a competing proposal cannot adopt or supersede it. Retry reconciles an exact owned module record before deciding whether another mutation is needed. Independently approving an unknown author operation is forbidden until the author reconciles it.

Expose distinct internal author/approval states and test before-module, after-module and after-publication interruption for both term and binding operations. Current author/decider permission loss or changed profile/source refuses recovery without releasing retained ownership. A module's historical replay is insufficient evidence of current eligibility. Public response handling must distinguish no-effect refusal from uncertain effects before these operations are exposed.

The later issuance draft freezes exact protected references and term coverage before approval. On retry, an already approved draft is compared with frozen approval facts; do not call issuance.Approve again to replace references silently. At issuance, Cover must be checked for the actual chosen issuance instant; approval-time coverage does not establish issue-time validity. The external number and issuance time still require explicit independently validated ERP provenance.

# Required validation

Exercise actual authenticated governors, issued permissions and receivables registry, not a successful resolver stub. Prove no defaults; disjoint author/decision rights; current session/human requirements; self/delegated/foreign-scope refusal; unowned record refusal; exact calendar/zone/date/contract provenance; stale/revoked authority and source refusal; real overlap and pending-term conflicts; all mutation/publication interruption recoveries; stable IDs and retained scope ownership; and current coverage read-back. Assert configuration does not issue documents, consume financial approvals, reserve numbers or deliver data.

# Open review decisions

The four-action mapping has independent architecture approval. The complete service contract also needs a concrete limit and interval ownership representation for pending/unknown contract bindings, and exact author/decider live-right checks without requiring the author to retain an active login session. Protected-field verification requires separate current custodians, requester, verifier and approver; that work must not weaken separation to fit the existing three-admin fixture. This document does not claim those decisions resolved or a retained issuance draft implemented.

# Authority implementation checkpoint

The four closed actions and term authority domain are implemented privately with actual sample-server assembly. The shared auth direct-governance matrix now includes all four actions, checking direct-human/console intersection, service/client-only/delegated/support refusals and current permission/scope changes. Actual server tests provision permissions through two authenticated governors and verify no defaults, fixed mappings, independent grant approval, foreign pricing-ceiling refusal, cross-capability/resource refusal, generic grant bypass refusal, revoked human/console rights, closed forms, SeedNone and recovery after an actual grant write with a lost publication.

Independent source review found the prior unknown-domain regression sentinel9 now names the new registered domain. Root changed only that sentinel to unassigned10, retaining0/255 and exact rejection checks. Independent corrective source review passed; it did not certify test execution. Actual targeted server term/domain/source race passed3.331s; auth full race passed2.051s and gate full race passed2.474s. Shared internal grant-authority tests passed from cache. The new facade has no standalone test suite; its real construction and effects are exercised by the server tests. Vet passed. Full server regression passed200.585s. A final domain-isolation assertion uses the same proposal ID, spec, author and rationale across pricing and term domains and passed0.444s; only the domain separates their digests and grant namespaces.

No payment-term business operation, public term command, inspection permission or issuance draft has been exposed. The four-operation receivables composition remains required. In particular, the existing profile-governance helper requires its acting caller to be an installation governor. Business consumption needs a distinct current-state profile check that keeps actual term/draft caller authorization separate from historical profile-governor standing; it must not fabricate a governor session or grant the business operator installation rights.

# Reviewed business-profile consumption boundary

Before reading a profile for a term operation, authorize the real current human session for the exact dedicated action through ConsoleClient. After validating the owned profile ID/digest/state and immutable source binding, inspect only its retained proposer/confirmer. A private domain-locked governor-standing helper reads current active-human identity/effective administrator membership and uses ExplainInstallation for the recorded subject plus ConsoleClient. It never fabricates a session, returns a permission stamp or authorizes the business operation. Preserve unavailable versus denied failures and refuse inconsistent generations. Existing profile proposal/confirmation retains its actual-governor caller requirement.

Independent review recommends the narrow Installation.Member read precedent in integrationrights.go with a truthful service-purpose operator reference. Any source-rule exception must be exact, not a broadened administration read bypass. Required cases are an ordinary granted business operator without admin rights, revoked/demoted/suspended profile governors, suspended console and refusal of missing/foreign profile IDs before arbitrary standing inspection. This consumption helper is designed but not implemented at the authority checkpoint.
