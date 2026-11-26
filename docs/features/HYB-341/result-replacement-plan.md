---
type: ImplementationPlan
title: Owned ERP result revision replacement and retained application history
description: Replace one complete owned result atomically while preserving immutable decisions, historical links and current authority checks.
status: draft
---

# Ownership and original outcome

HYB-341, machine:mac_mini, continues d5403ff and the original result-ingestion plan. Root owns planning, code and integration. refusal_review provides independent read-only financial/security review; platform_refusals reviews evidence. Existing no-restoration authorization and synthetic in-memory boundary remain in force. This increment does not complete StageERPFinal, issuance, delivery, persistence or the complete owner goal. No external ERP command, approval consumption, merge or deployment is authorized.

# Observed producer constraints

Document accepts a higher revision but retains earlier links; only missing/overallocated rows become stale. Link cannot replace a live allocation with a different amount or actor. ResultOf provides coherent current evidence, including stale and foreign links. Therefore sequential Document/Link calls cannot implement safe replacement by simply relaxing the current service ownership check.

The service currently reserves one result per external document and preparation. Application-owner validation and impact enumeration assume one application per head index. Historical receipts are derived from exact coherent snapshots; cached successful application rechecks current watermark, authority and ledger facts. These assumptions must be updated together, not bypassed in the mutation path alone.

# Atomic module prerequisite under independent review

Add a dedicated ReplaceResult operation to the actual erpledger registry. It receives workspace, preparation key, the exact expected predecessor ResultView, the next Document, complete next links and the actual actor. It takes one existing registry lock and compares the full current predecessor snapshot before any write. This is a conditional mutation, not a new external submission or an approval assertion.

Require a human actor, a prepared/nondiverged submission with no external commands, one exclusively attributed document, strictly increasing revision of the same connection/external identity, and complete exact one-to-one row/line coverage. Preserve configured numeric/row/link limits. Reject shared documents, foreign/stale-unrecognized links, changed predecessor facts, unsupported submission states, malformed identities, duplicate rows/lines and incomplete/overallocated next amounts. The service additionally verifies product and preparation provenance and all current financial/result rights.

Construct the replacement in isolated candidate state. Retain previous links as stale history, append the new exact same-actor allocations, and validate complete reconciliation before publishing one replacement document pointer. Cumulative retained-link capacity is preflighted; no history is deleted to make room. ResultOf observers see either the complete predecessor or complete successor, never a document-only intermediate. A module error before publication leaves documents, links and submissions unchanged, apart from the normal monotonic read clock. The returned snapshot is defensive. Module replay does not adopt an arbitrary matching successor; service-owned recovery controls recognition.

# Service ownership and recovery

Retain every immutable application record and receipt. Introduce explicit predecessor binding and a bounded acyclic strictly increasing chain; the existing external/preparation indexes identify its unique current head. Both indexes must agree. Validate every ancestor and reverse association before using the head; cardinality equality between applications and heads no longer expresses integrity.

A new replacement requires a completed owned predecessor receipt whose exact current ledger snapshot still matches its retained evidence. Unknown predecessor operations remain owned and refuse replacement; a later report must not erase or silently settle an interrupted earlier operation. This preserves the existing refusal boundary rather than pretending uncertainty is absent. The new confirmed result must still be the monotonic watermark and pass current original-proposer/confirmer/applying-actor authority, document/charge/preparation provenance, hold and policy checks.

Before invoking ReplaceResult, reserve the immutable successor intent with predecessor identity, digest, actor and exact before-snapshot, and preflight bounded eventual output/receipt representation. Preserve reservation and uncertainty on any interrupted mutation/receipt path. Original-actor recovery revalidates current evidence and distinguishes the exact before-snapshot (eligible to invoke) from the exact intended complete after-snapshot (eligible to recognize). Any third state remains uncertain; no fallback overwrite or silent adoption is allowed. Publish the successor receipt only after coherent exact after-state verification. Old receipts remain historical and readable, while fresh old apply/recovery remains subject to the current watermark.

History, inspection, advice, cached receipts, M01 impact counts and B09 action review must understand the same chain. Do not claim replacement readiness using old one-head-per-application assumptions. The public result packet remains independently supplied original evidence; do not copy amounts from preparation to manufacture a match.

# Allowed files and acceptance

Root may add narrowly scoped replacement implementation/tests under modules/erpledger, factor a locked ResultOf helper, extend the financialERP ledger interface and result application/ownership/advice/output/impact code, and add relevant frontend review behavior only after its design review. Preserve other domains and legacy Document/Link semantics. Directly maintained files remain below 1,500 lines; no explanatory source comments.

Module acceptance covers exact replacement, retained predecessor snapshots/links, repeated increasing revisions, changed row identities, exact amounts, same-total wrong-product evidence, wrong tenant/actor/state, foreign/shared attribution, malformed/stale expected snapshots, cumulative limits, no writes on rejection, defensive copies and concurrent ResultOf observations. Service acceptance covers actual governed predecessor and successor proposals, independent confirmation, current rights/watermark rechecks, owned historical receipts, head-chain corruption, interrupted atomic write and original-actor recovery, revoked/newer evidence refusal, unchanged financial approval/command state, and actual HTTP/read/UI coherence. Run focused and race checks before broader regression; independent corrective review is required. StageERPFinal follows this evidence boundary and must use the existing approvals registry.

# Review state

The initial proposal below was followed by independent module and service protocol review. Their conditions and the current implementation boundary are recorded in the subsequent sections; those reviews are not final public/UI integration acceptance.

# Independent design conditions resolved for the module prerequisite

The independent reviewer found the direction feasible and required complete CAS, detached deep copies, exact stale-history preservation, explicit prepared state, cumulative bounds and coordinated service-chain updates. Root adopts these conditions. Module comparison uses the complete ResultView, not selected fields or Reconciliation.Matched. Reject every foreign-preparation link, including stale history. Copy each retained link object before setting its stale flag; existing stale entries remain byte-for-byte equivalent and formerly active entries change only that flag. Append exactly one new active allocation for every preparation line and every next row. No live link object is mutated before publication.

For product identity, validate the current mapping set against the preparation's pinned mapping references, then resolve each preparation line's product through the actual connection/product mapping and require the corresponding next row's product to match. Equal totals never authorize a product substitution. Existing module numeric representation and exact scaled arithmetic remain authoritative; the service's synthetic packet profile remains stricter where documented. No inferred tax/accounting rule is added.

Service recovery must not predict the module write timestamp. Reserve the service observation time before invocation. An intended successor snapshot must have the exact next document, original replacement actor, exact stale predecessor prefix and exact appended active links, unchanged preparation and complete exact reconciliation. Its nonzero document timestamp must be no earlier than both the reservation observation and predecessor timestamp and no later than the current monotonic service observation. This is a recognition predicate for already reserved owned intent, never permission to adopt a matching foreign write. Retain the actual after-snapshot and its receipt once verified.

The module publishes only a detached candidate document after validation under the existing mutex; all prior failures preserve registry state. Its operation has no automatic same-input replay. Service recovery decides whether the actual state is still the exact before-snapshot or already the reserved after-state. Current policy/authority and head/watermark checks remain mandatory in both cases.

The initial module implementation is a prerequisite to the full coordinated service protocol. It cannot by itself enable public replacement or certify the overall requirement. Unknown predecessor/unknown successor ownership remains retained and cannot be skipped to create another chain head.

# Module implementation checkpoint

After the resolution section, independent module design review passed. Root implemented the atomic operation and added the rejection, product-bound, retained-history, alias and concurrency scenarios. Full module race tests passed in 2.006 seconds; vet passed. Independent corrective module source/test review passed with no remaining blocker. See result-replacement-module-verification.md. Service-chain, recovery, advice/public and UI integration remain the next required work; the module result does not certify those stages.

# Service protocol design and corrective implementation

Independent service design review required one retained application per result, with both reverse indexes identifying the reserved chain head, including an unknown successor. Root adopted explicit predecessor, detached before/after snapshots and reservation time. Completed ancestors retain their original receipt and actor. Missing retained evidence refuses replacement; it is never reconstructed from arbitrary current ledger facts.

The shared bounded validator checks every record, receipt, predecessor, strictly increasing revision and both reverse associations. Complete traversal coverage rejects cycles, branches and orphan records. At most 100 retained records, each at most 128 KiB, bound the aggregate serialized record representation to 12.5 MiB; in-memory object and index overhead is additional; the service additionally preflights the complete serialized successor before/after record with every link author, receipt and maximum-width timestamp before reserving. Original result and public field limits remain separate constraints.

Root implemented this protocol across application, inspection/history ownership, advice, impact and cached current receipt validation. Current cached receipt validation requires exact current snapshot and head ownership; historical reads use retained evidence. Unknown successor recovery recognizes only the exact retained before state or the reserved intended complete after state, with the original applying actor and current authority/watermark checks. Reservation and completion advance the service observation watermark. No timestamps are predicted.

Independent implementation review found a blocking admission gap: service JSON bounds did not preflight actual module link capacity or fresh module constraints. At the actual 100-link limit, a deterministic rejection could otherwise reserve an unnecessary unknown successor. Root added CheckResultReplacement using the exact same locked candidate validation as ReplaceResult without publishing it. Advice/admission invokes that check before reservation; ReplaceResult still repeats the check atomically at invocation. Actual runtime changes between those two observations remain conservatively uncertain after invocation.

The correction includes a real-ledger 50-result/100-link boundary scenario: revision 51 must refuse before changing either index, adding an application or changing the ledger. Additional service scenarios cover three successive replacements, before/after-write interruption, wrong actor, revoked apply rights, newer watermark, third-state divergence, historical receipts and old cached HTTP success refusal. Final corrective review and public UI integration remain required; no merge or full-goal completion is claimed here.
