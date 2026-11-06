---
type: ScreenDesign
title: Billing application history and protected coverage inspection
description: Designs B10 read-only Claim evidence without inventing a ledger producer or rebilling authority.
status: draft
---

# Scope and ownership

HB-111 B10 `/claims` follows B09 on open PR86 at f444c5a791c5489242340abd1ff7b4bd67ea8cfe. Root owns PM/UI integration/QA; independent finance/TW and security reviewers inspect design and actual changes; Claude CLI ui-designer leads composition and actual render comparison. One contract implementer may own only the new B10 API/types/tests/fixtures. The branch is stacked, not claimed fresh-main; no previous leaf, shared design token/style or backend is edited. Marketplace remains held.

The B10 spec and claim-and-numeric contract distinguish canonical coverage identity, reserved/applied/reversed/available coverage, money, original-linked corrections and external ERP outcomes. Existing discovery found no Claim/ERP producer. Normal service therefore shows unconnected with unknown counts, no fixture rows/amounts or fabricated ledger. No source FOCUS row becomes a Claim. No release/reverse/replace/reserve/submit/correct/post or ledger edit. Live payloads fail closed until a separately reviewed producer binding. Isolated QA intercepts alone provide always-labelled synthetic records; no URL/window/storage fixture mode.

# Proposed read-only consumer

GET `/api/claim-history` is an unbound consumer proposal, not an implemented backend/API authority. Same-origin GET with trusted local-client header, no-store, redirect rejection, abort/monotonic publication gating, streamed2MiB cap and strict nested validation. Actual404/501→unconnected; denied/stale/malformed/server failures discard old rows/details. No endpoint switch, writes, downloads, persistent cache or command drafts. Query view ledger/partial/reversals/references, state filter, bounded normalized q, optional cursor with snapshot+permission_revision pins; opaque server auth_scope/snapshot/permission hashes and exact response query identity. At most25 rows and exact/unknown total. All source_mode live rejected; synthetic always labelled.

Draft records: canonical Claim ID/revision; target label, nullable legal entity/customer/contract/component/period; coverage identity/unit/version plus source-description and nullable finite nonnegative decimal-string observations reserved/applied/reversed/available. No client geometry or subtraction; scalar counters cannot prove interval availability or replacement eligibility. Coverage protection enum available/reserved/outcome_unknown/applied/release_pending/released/replacement_pending/cancelled/superseded/unknown is a source declaration, never executable eligibility; protected unknown cannot present known available coverage. Bounded monetary observations per currency+basis (net/tax/gross/expense) with nullable applied/remaining/reversed finite signed strings, preserved exactly without rounding/padding/cross-unit totals. Remaining money is not available coverage or receivable balance. No universal numeric scale is activated; transport max80characters/76digits/38fraction matches B09 read bounds only.

Each row includes partial=true/false/null; source updated time; nullable source/allocation/sales/charge_line/submission references; correction original Claim ID, bounded linked correction IDs and declared disposition none/money_only/quantity_reversal/replacement_pending/cancelled/superseded/unknown. Money-only credit never reopens quantity; quantity reversal never authorizes rebilling. External references bounded16 with kind erp_order/prepayment/invoice/submission/other, canonical ID, nullable revision/observed_at and source outcome pending/unknown/applied/not_applied/cancelled. They do not prove matching/legal reconciliation. Plain evidence/blockers are source text, bounded and direction-isolated. Unknown remainder is not zero or available. No progress percentage, aggregate total or named actor is inferred.

Partial view requires partial=true; reversals requires original/link/disposition correction evidence; references requires at least one external reference. Ledger is all supplied source statuses. User can filter, page, inspect, close/Escape/focus return, reread and navigate generically to B08/B11/I02 without passing unverified IDs; no fake correction creation. Details only for current-page canonical IDs; view/filter/cursor/error/revocation removes stale selection. Browser history rereads, never restores old detail authority. Full read identity and selected source evidence enter review digest; owner-only review control remains separate.

# Composition and stable items

Full viewport, existing IA/logo/local Plex,16px body/control/table,20px h2,14px meta,Arabic1.8; neutral square separators/no colored edges/shadows. Mobile all four tabs visible and source text auto-isolated; canonical IDs/dates/money LTR. Actual default unconnected state; labelled QA loaded list/table/detail; no KPI zeros. Read-only detail groups coverage/protection before money, then lineage/correction/external evidence. All columns retained in scrollable disclosure, amounts never combined. Final composition follows actual Claude CLI design.

| Item | Purpose/location | States and verification |
| --- | --- | --- |
| B10:scope:history-filter | Source state and bounded Claim/target search | loaded/loading/error/query invalidation |
| B10:data:history | Bounded application records, exact/unknown count | empty/partial/unconnected |
| B10:action:inspect-claim | Current-page canonical card; close/Escape/focus | selection/navigation/unmount |
| B10:data:coverage | Canonical coverage references/counters/protection, no geometry inference | reserved/unknown/missing/zero |
| B10:data:amounts | Per-currency/basis applied/remaining/reversed strings | zero/null/signed/unsupported |
| B10:data:lineage | Supplied source→allocation→sales→line→submission references | missing/partial |
| B10:data:correction | Original/linked references, money-only vs quantity/replacement boundaries | partial/reversed/protected |
| B10:data:external | ERP order/prepayment/source outcome observations | pending/unknown/no-effect |
| B10:state:async | Clear data on denied/stale/failure and superseded responses | read/error/retry/race |

Validation plan: independent design/change finance/security/TW; strict unit checks; actual404 default; isolated synthetic states;48 main combinations (4views×3locales×2widths×2modes), keyboard/focus/filters/25row cursor/denial/stale/late success+error and no writes/storage/ownerFIX; existing B09 regression. Other OS/engines, native/human finance, live authority and complete original financial workflow remain pending.

# External evidence

[Microsoft Customer transactions](https://learn.microsoft.com/en-us/dynamics365/finance/accounts-receivable/customer-transactions-list-page) informs selecting a canonical transaction before inspecting related documents/history and restricting cross-entity views to permitted entities. Adopt source-linked detail hierarchy, not its settlement/undo/export/payment controls or receivables semantics. [Stripe credit-note Dashboard](https://docs.stripe.com/invoicing/dashboard/credit-notes) informs retaining an original invoice reference and separating amount corrections from payments. Do not adopt tax/refund/credit formulas, commands, assets or Stripe integration. The B10 economic coverage contract remains authoritative; external products cannot prove Claim geometry or replacement eligibility.

# Independent design conditions

Security/contract conditional PASS requires fixed read URL, strict allowlists/byte/page/query guards, live rejection and unconditional synthetic labels; no claims of current server authorization from digest shape. Finance requires explicit supplied application_id, obligation_id, purpose and coverage universe/unit/version. The protection enum describes only that row's supplied application/coverage subset, never the entire obligation: different intervals may remain reserved/consumed/available simultaneously. No browser coordinate union, scalar subtraction, duplicate certification or eligibility action. Protected unknown observations carry available=null; unknown is not a zero or reusable scope. Full read identity participates in revision-bound screen review. All original financial enforcement tests remain deferred.

# Design reconciliation

Actual Claude CLI composition read the current generic B10 capture, B09 precedents, foundation guides and the installed official frontend-design skill. Adopted full-width5/12 list7/12 source card, protection before money, compact identifier-first rows, explicit synthetic tag, no row money total,2×2 mobile tabs, source lineage and closed full-column/evidence disclosures. Inline mobile detail follows the selected row so the canonical origin remains visible. Finance/TW preferred Billing application history / Partial billing / Reversals and corrections / External references overrides the CLI's older generic labels; canonical keys stay unchanged. Unsupported schemas fail validation rather than showing unverified numeric values. Denied/stale errors require a new query, not a retry against the old scope. The draft has one external unknown enum, not unknown_state.

Independent actual-source review found that same-application external pending/unknown plus known available coverage was an ambiguous projection even when protection was not outcome_unknown. Finance agreed on conservative rejection: any non-null available scalar, including0, is invalid with same-row pending/unknown external evidence. Missing available remains null; no state/value is rewritten. Root regression failed before the guard and passed after it, with available/applied states, pending/unknown outcomes and zero/nonzero values. Both independent reviewers reran17 API tests PASS after the amendment. A future mixed-scope external producer needs explicit per-reference scopes and separate design review.
