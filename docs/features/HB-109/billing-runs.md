---
type: ScreenDesign
title: Billing run input review and source comparison
description: Implements B08 input evidence and a volatile source-baseline comparison while financial run authorities remain unconnected.
status: draft
---

# Scope and evidence

HB-109 implements B08 `/billing-runs` with stable runs/inputs/trace/comparison/validation/recognition views from the authorized B08 spec and F01/F02/AR-01/06/07/ERPC planning. Root owns PM, frontend/model/tests/integration and QA; a read-only explorer verifies actual producer capabilities. Separate finance/contract/TW reviews and Claude CLI composition/final render comparison are required. This branch stacks on held Marketplace PR84 at `1a7912d9d2bcef56ef0251e7fbe743fba49df982`; main was separately read as `36394edf179d544d7ec2756f5329eba60bcc2d4e`. B07 is held for CSP-specific follow-up, not FIX, merge or universal support. Prior accepted/implemented leaf screens remain unchanged.

Current discovery finds protected collected FOCUS overview evidence but no authoritative financial billing-run/history/job/checkpoint, eligible component/claim, approved policy/FX/engine, approval or ERP recognition producer. Ingestion jobs, customer-export approvals and numeric probes are not billing capabilities. Do not invent runs, zero run counts, policies, progress totals, authors, manifests, results or hashes. Full original B08 workflow remains pending these authorities; this increment is actual read-only input inspection and volatile comparison, not completed billing execution.

# Actual interactions and data

Reuse `/api/focus-overview` and its protected client contract, with B08-only defensive validation; no new backend route, schema or import. Keep provider/charge-start month/canonical account scope; account q searches only account options. Full-selection Billed/Effective per-currency aggregates precede 25-row and account pages. Source row references are technical evidence, not customer, contract, legal entity or target manifest. Raw data is public/synthetic collected evidence; no demonstration run fixtures appear. Fresh protected source read confirms values before handoff. Source coverage declarations concern all authorized collected sources and are operator assertions, not certified selected-scope completeness.

The user can inspect real input pages, search/select accounts, open/close row details with focus return, retry/new query, and explicitly 'Keep source comparison baseline'. The baseline is one bounded copy of full-selection amounts, row count, provider/account/period, company/tenant/timezone/version and period basis identity plus source/query references and observed read time. Keep it only in component memory, not localStorage, server records, download, URL, or a financial draft. It is not an immutable input manifest, calculation lock, job or result hash. It survives view/page/account-search changes within this screen only; provider/period/account changes, any failed/denied/stale query and screen unmount discard it. New-query refresh within the exact same scope and exact protected source_snapshot can compare source observations. A changed source_snapshot discards the baseline before redisplay; the current API cannot reauthorize historical aggregates, so changed-source or previous-run comparison remains unavailable. Until a fresh authorized result arrives, no old source/baseline details are rendered. An incompatible company/tenant/version/basis/source-snapshot identity never displays baseline amounts or deltas.

Comparison is explicitly 'source input comparison', not previous billing-run comparison or customer charge variance. Pair identical currencies and identical Billed/Effective bases only. Compute exact signed integer coefficient differences at scale18 (current minus baseline) only when both sides contain rows and the selected measure is complete (missing0, present==rows). All missing, partial or missing currency counterpart yields unavailable, not zero. Never sum currencies, compare Billed minus Effective, infer savings, repricing, taxes, eligibility or adjustments. Browser searches/pages do not redefine full totals or target/pool dependencies. Source equality or zero difference cannot certify policy equality, no source-row changes, validation completion or eligibility. Rule/input/policy/adjustment cause attribution remains unavailable.

# Six service views

Runs: a clean operational workspace with actual source-input summary and an honest execution-service-unconnected empty state, never a fabricated zero-run result. Real actions inspect inputs or open source comparison. Do not render dead calculation/retry/approval buttons.

Inputs and policies: actual collected full-scope costs and paged input rows with canonical detail; separately unconnected contract/ownership, target manifest, eligible components/claim, approved allocation/sales/FX/engine versions. A browser source baseline is not any of these authorities.

Results and rule trace: source evidence remains inspectable, but customer charge lines, rule trace, separate adjustment lines and result hash are unconnected. Do not repurpose source line coefficients as selling amounts.

Comparison: baseline versus current source observations side by side, exact same-source, same-scope per-currency/per-basis input differences, explicit partial/unavailable semantics and memory-only lifetime. No prior-run result IDs or fake cause classification.

Validation: distinguish observed source query/amount presence from unknown source completeness, contract eligibility, pool/claim checks, approved policy/FX/engine, result validation and separate calculation/issuance locks. 'Read successfully' never becomes 'ready to bill'. Present compact operational dependency groups, not a developer implementation checklist.

Recognition: monthly expense recognition and quarterly customer billing stay distinct. Legal entity/book/amount basis/cadence, monthly batches/actual ERP posting, quarterly linkage, adjustments and reconciliation are unconnected. Show no fake months, zero costs or balanced reconciliation. No invoice creation, posting, recognition approval or payment-term clock begins.

F02 ordering follows authoritative ui-design: scope → source readiness/reconciliation → allocation → full calculation → approval → authorized issuance/delivery → closing. D7 remains scoped to flow presentation; this increment creates no alternative finalization order. D4 correction ownership and D5 payment extensions belong to later tickets.

# Composition, stable items and review

Full available viewport and canonical IA/logo/local Plex. Body/control/table16px, h2 20px, metadata14px; neutral square separators, no shadows or decorative colored borders. KO/EN/AR with labelled counts and complete LTR isolates for IDs/dates/money/FOCUS version. Avoid mobile inherited-bold selected detail and bidi errors observed in B07. Desktop source table/detail, inline mobile details and separately disclosed full table. Money headers retain Korean word groups. Preserve common per-screen owner review controls using complete scoped state; no assistant/QA owner acknowledgment.

Stable items: B08:scope:source; B08:data:run-boundary; B08:data:input-evidence; B08:data:result-boundary; B08:action:source-baseline; B08:data:source-comparison; B08:data:validation-boundary; B08:data:recognition-boundary; B08:action:inspect-source; B08:state:async.

[Stripe preview invoice documentation](https://docs.stripe.com/api/invoices/create_preview) separates preview from an issued/listed payable invoice. Adopt the explicit authority distinction, reject its API/policies/rounding as our implementation and do not install its suggested tooling. [AWS Batch view-jobs guide](https://docs.aws.amazon.com/batch/latest/userguide/view-jobs.html) separates queue scope and actual job filters/status evidence; adopt bounded scope-first presentation, reject fake job statuses/counts where no producer exists. Carbon's current data-table guide from B07 informs bounded row/detail structure; existing project tokens and owner constraints take precedence. Stripe and AWS pages were actually read in this ticket; no external account was accessed.

# Validation and limits

Independent finance/contract design and change review before claiming completion. Pure tests cover scope compatibility, exact signed/large/zero/partial/null/currency-counterpart deltas, full totals unaffected by pages/search, guarded API response and volatile baseline lifecycle. Meaningful numeric invariants may use deterministic tables without new dependencies. Actual browser checks cover pin/refresh/discard/scope invalidation, failed/denied/stale/late responses, source paging/account search, detail/focus/Escape, no financial commands or storage writes and all 36 view/locale/layout combinations (six views × KO/EN/AR ×1440/390). Source read is not a production capacity claim. Financial run/policy/claim/approval/ERP integration, full original AR/ERPC acceptance, native-language/human finance, other OS/engines and owner FIX remain pending.

# Independent design conditions

Finance/TW design conditionally passed the temporary source-only baseline, complete-measure same-currency arithmetic and distinct monthly expense-recognition authority. Contract review requires exact source_snapshot equality, because current member keys alone cannot reauthorize historical aggregates after an opaque protected source reference changes. Changed source references discard the baseline before any render. The B08-only validator strengthens existing overview validation for unique sets/page identities, selected scope/membership, actual calendar/timezone timestamps and pinned offsets; it does not alter shared consumers. Old data stays hidden during every fresh request. Full original billing-run comparison remains unimplemented, not silently approximated by this source reference check.

# Composition and scoped implementation reconciliation

The actual Claude CLI ui-designer inspected the generic B08 render and selected B07 precedents. Root adopted its full-width scope/baseline strip, source totals, source table/detail, side-by-side comparison and compact dependency groups. The shared tabs remain in the existing shell position, not rewritten under this leaf ticket. The designer suggested publisher/issuer/SKU fields by analogy with B07; the actual overview contract does not expose them, so B08 shows only its real service/provider/account/interval/quantity/source-reference fields. The stricter independent contract condition discards changed source references instead of displaying a historical identity-mismatch amount table. Differences are displayed only after a newer authorized read, not immediately after pinning the same observation. This increment is not the full original B08 run workflow.
