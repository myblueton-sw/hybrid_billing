---
type: DesignSpecification
title: Audit reporting verification and product policy
description: Defines audit report support demo licensing and AI egress settings with evidence and authority boundaries.
status: draft
---

# Context

Inherits [shared rules](README.md). Sources: HB-58 OPS07–10/DOC02–03; v1.16 B0195, B0920, B1286, B1293, B1317–B1328; [commands](../../../contracts/commands/planning-actions.md), [screen binding register](../../../ui/screen-binding-register.md), [license modes](../../../licensing/usage-modes-requirements.md), [product policy](../../../licensing/product-policy.md) and [intelligence architecture](../../../architecture/intelligence-layer.md). Unless named below, action bindings are **unresolved**.

# R02 — cross-domain audit (OPS07)

Entry: Settings → Audit or a permitted entity's History link; scoped auditors. Fields/filters: actor type and permitted identity, action, target type/ID, UI/API/CLI/worker path, request/operation/job ID, time, decision/result, current disclosure classification, masked before/after values, evidence/approval references and policy revision. Include denied operations without revealing a forbidden object's existence. Event history is immutable; a later correction is linked, not an edit.

Flow: filter permitted events → inspect event/evidence → follow authorized domain history → request bounded export. Evidence links recheck current row/field rights; auditor role does not automatically expose cost or secrets. Walkthrough E01: a trust activation shows proposal, independent approval and propagation status while protected identity details remain masked. A revoked auditor cannot open a previously copied evidence URL. A missing audit partition shows incomplete coverage with time range, not 'no events'. Audit query/export bindings remain unresolved; existing source/PDF audit stays linked rather than replaced.

# R01/I04 — saved views and complete exports (OPS08)

Entry: Reports → Saved views, or a financial/history screen → Export. Fields: view ID/owner, query revision, columns, row/field scope, period, historical-attribution versus current-organization basis, snapshot/result revision, currency/conversion basis, sort, retention and intended format. Saving a view stores preferences, not grants. Own saved views are the committed planning scope. Sharing policy remains undecided in the original R01 specification; sharing controls stay unavailable until a separately reviewed scope/ownership/recipient policy and command binding exists. Any eventual sharing cannot expose previously inaccessible targets.

Export preview freezes authorized target/filter/column manifest and result revision, totals/counts by currency, expected partitions, completeness, disclosure class and expiry. Generation is a job; output manifest records actual parts/counts/totals and failures. Successful technical processing with missing partitions is incomplete, never a complete download. Limit crossings split with an explicit manifest; no silent truncation or cross-currency total. Financial historical reports preserve immutable source snapshots; current-organization presentation is separately labelled and cannot restate issued attribution.

Walkthrough E02: a user resumes a large export after losing cost-field access; the old artifact cannot be downloaded, refreshed or shared under stale grants. Rebuild a newly authorized projection if requested, preserving original snapshot provenance and audit. Complete/expired/revoked outputs remain distinct from job states; retry checks original operation and current rights, not a new financial calculation. Saved view CRUD/export generation/download/revocation bindings are unresolved. Existing report save/export functionality is extended, not asserted absent.

# P05/P06 — support evidence and isolated scenarios (OPS09, DOC03)

Entry: Settings → Supported profiles / Demo scenarios. P05 fields: provider/ERP/product version, connection mode, data/schema revision, workflow path, evidence run/version/date, environment class, expected/observed result, limitations, verification owner and status. Separate configured, connection-tested, synthetic-tested and actual supported evidence. Expired/missing evidence is unverified; a demo pass cannot promote a real support claim. Support-profile edit/publication bindings are unresolved.

P06 fields: isolated demo environment ID/revision; operating profile `msp`, `standalone_enterprise`, `corporate_group`; purpose `public_sample_view`, `synthetic_settlement`, `failure_reproduction`; scenario/fixture revisions, seed, virtual clock, expected-result bundle and fault set. `demo.scenario_run` requires an independent fixed expected bundle for synthetic/failure purposes; public sample viewing has no expected bundle and may display incomplete months/missing documents. Fault injection is restricted to failure reproduction. Reject real credentials, secret-store references, operational targets and unregistered destinations.

Flow: choose profile/purpose → validate allowed fixture → run → inspect evidence → `demo.compare_expected` only for synthetic/failure purposes → optional `demo.reset` for that isolated environment at its expected revision. Compare amount, state, blocker and permission results against independent expectations, never against a copy of the generated result. Reset scope preview excludes operational data, volumes and connections.

Walkthrough E03: a public sample run labels output as illustrative and disables expected-invoice comparison. A failure-reproduction run without an expected bundle is blocked; supplying an approved fixed bundle permits comparison but does not certify CSP/ERP connectivity. A reset request containing a production connection is rejected. This adopts the required fields in a new draft; local P06 companion adoption under HB-24 remains open.

# P07 — license mode and entitlement (DOC02)

Entry: Settings → License; permitted license operators register entitlement, other users see safe mode/status information. Fields: declared mode, permission/grant reference, authorized company, scope, accepted terms, authoritative activation/start/end, signing-key identifier, verification result/time and current uncertainty. Raw license keys are not logged or exposed in history. Signed-key/file input is a protected transient registration path; verified claims, not user-entered company labels, govern display.

Modes: public noncommercial under applicable license; commercial evaluation only with separate express accepted permission; purchased commercial under agreement and verified entitlement. The evaluation interval follows the established 14-day/336-hour contract, not a new adjustable timer. Restoring/reinstalling/changing mode cannot reset it. Public noncommercial does not require registration or an evaluation countdown. Mode selection is not a rights grant.

Flow: choose declared mode → show applicable basis → register protected entitlement where applicable → verify → show active, expired, invalid, unverifiable or scope-mismatch distinctly. Never replace a valid entitlement with an invalid submission. Expired terms show persistent nonblocking Register license key and user-initiated Contact myblueton@gmail.com actions; preserve historical access/export and ordinary flows under current authorization. Technical availability does not extend commercial permission. No automatic contact, telemetry or remote lock is authorized.

Walkthrough E04: a failed verification network request preserves last verified evidence labelled stale/unknown, not expired. A modified key is invalid and leaves the prior valid registration intact. Actual expiry produces the nonblocking warning and contact link without deleting data. Verification/time/revocation and registration command bindings remain unresolved; this corrects draft policy presentation, not the existing license text or the original local P07 companion.

# P01/A01 — AI placement and egress policy (OPS10)

Entry: Settings → Installation policy → AI assistance, with A01 showing the applicable policy state. Authorized security/data policy owners review changes. Fields: enabled/off/unavailable, purpose, deployment boundary, allowed data classes/fields and tenant scope, approved endpoint/model reference if selected, retention policy and separate training-eligibility reference (if applicable), owner, approval, effective revision, budget and destination validation evidence. Egress approval is not training permission. Sensitive-data training, artifact eligibility, approval and withdrawal/restore controls require the separate independently reviewed ED-06 gate in the [access and trust contract](../../../contracts/security/access-and-trust.md); this screen cannot activate training through a general egress approval. No actual model/vendor is selected. Missing egress approval or unknown destination blocks external transmission.

Flow: draft purpose/scope → preview data and destination implications → independent policy approval → activate exact revision → show policy status in A01. Execution rechecks current permission and policy; AI cannot approve its own actions, bypass normal command gates or add a new disclosure grant. Queries/responses expose source, freshness and provisional/final status within current scope. Full assistant workflow redesign is out of scope.

Walkthrough E05: a closed-network installation turns AI off; onboarding, calculation, approval, issuance and evidence navigation remain usable through ordinary screens. An unavailable approved endpoint offers the manual path and explicit unavailable status, not automatic external fallback. A request containing forbidden fields is blocked/redacted according to the approved policy before egress; absence of such policy blocks transmission. Policy edit/approval/activation bindings remain unresolved.
