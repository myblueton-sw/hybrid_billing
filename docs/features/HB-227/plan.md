---
type: ImplementationPlan
title: ERP product adapters with per-command evidence, simulator-proof certification, ledger capabilities by evidence level and an automatic delivery gate
description: Defines the first increment of work item W-25. It adds a register of ERP product and version profiles that never share one API across products, and adapters that map the platform commands to product methods with an evidence level for each command, where a simulated pass never certifies compatibility. Capabilities reach the ERP ledger only at a chosen evidence level. Automatic attachment, mail and portal delivery stays blocked unless recipient and attachment control are proven and no internal file is attached. The register also plans document chunks within proven row limits, and a test-only simulated ERP drives the full ledger round trip.
status: draft
---

# Scope and ownership

[HB-227](https://linear.app/hybrid-billing/issue/HB-227) starts work item W-25 (requirement HB-25; detail contracts DB-03, DB-04, DB-05, DB-11 and DB-12; acceptance AUD-DB-05 and HB-25-V) after HB-226 (`modules/erpledger`). It follows the owner's instructions of 2026-11-12, 2026-11-14 and 2026-11-15. Base: main `b7a7b6a`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the plan and the change, because adapters decide which ERP commands are trusted and the delivery gate decides what reaches customers. Machine: mac_mini.

The following are outside this increment and are recorded as residuals:

- calls to a real Odoo or SAP system, provider credentials and sandbox evidence;
- the HB-25-V measurements of call limits, maximum rows and draft latency, and call-limit enforcement;
- the choice of the first pilot product, which is the owner's decision;
- binding a ledger connection to an adapter, refusing unadmitted capabilities, checking transports and diagnosis targets against the ledger;
- the qualified certifier role (with `modules/auth`);
- approved manual delivery routes (HB-27, HB-29);
- persistence and screens I02, I04 and S02.

# Observed baseline

- **HB-25-D.** An Odoo-first vertical proof is a proposal only, and SAP products and versions are separate. S/4HANA, ECC and Business One are never treated as one API. The design must record the first ERP product, version, deployment, sandbox, required modules and whether automatic posting is possible.
- **HB-25-I.** In one pilot combination, the platform proves the full path: approved specification, draft, amount recovery, final approval, posting, final document and correction. The adapter owns the mapping from platform commands to product methods.
- **HB-25-V.** Tests cover a normal round trip, another entity or unit, a locked period, price re-application, timeout, manual edit and partial collection. A simulated ERP pass is not a compatibility certification. Call limits, maximum document rows and draft latency are measured in a real sandbox. Cost rows and ERP document rows are mapped by contract through aggregation and tracing.
- **DB-05 and AUD-DB-05.**
  - Automatic attachment, mail and portal publication are diagnosed as connection capabilities.
  - Unless recipient-level field and attachment control is proven, automatic delivery stays off, and only approved delivery routes are allowed.
  - Platform export checks alone never pass ERP delivery safety.
  - A template that adds an internal cost file blocks automatic delivery, regardless of issue or posting state.
- **Existing modules.** `modules/erpledger` checks capabilities per connection but does not know where they come from. No module holds ERP product profiles or adapters.

# Decision

A new package `modules/erpadapters` follows the conventions of the sibling registers:

- an injected clock and one mutex;
- workspace isolation;
- the errors `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`) and `ErrLimit`;
- the check order form, lookup, repeat or conflict, rule, room;
- writes by humans, except that a system test harness may record simulated evidence.

An identical repeat by the same human returns the view. Assistants are always refused.

## Profiles

`Profile(workspace, Profile{ID, Vendor, Product, Version, Deployment, Modules}, actor)` records one immutable product and version profile:

- the vendor and product pairs are exactly: `odoo` with `odoo`; `sap` with `s4hana`, `ecc` or `business_one`; `other` with `other` (`vendor_mismatch` otherwise);
- the version and module labels are identifiers;
- the deployment is `cloud`, `on_premise` or `private_cloud`.

Two profiles never merge.

## Adapters

`Adapter(workspace, Adapter{ID, Profile, Methods, MaxRows, CallsPerMinute}, actor)` records an immutable adapter bound to exactly one profile; it can never be rebound. Methods map platform commands (`connect`, `validate_mapping`, `prepare`, `submit_draft`, `read_draft`, `post`, `get_status`, `correct`, `get_balance`) to a product method identifier and a transport (`api` or `file`; `database` is refused as `direct_database_forbidden`). `MaxRows` and `CallsPerMinute` are declared limits from 1 to 1,000,000. Every method and each of the two limits starts at the evidence level `declared`.

Evidence rules:

- Levels in order: `declared`, `simulated`, `sandbox`, `production`; a raise may skip levels.
- `Evidence(workspace, adapter, command, Evidence{Level, Reference}, actor)` raises one command; `LimitEvidence(workspace, adapter, limit, Evidence, actor)` raises `max_rows` or `calls_per_minute` under the same rules.
- A system actor may record `simulated` only; a human may record any level. `sandbox` and `production` need an evidence reference and a human other than the adapter's author (`self_approval`); the same human may raise both. References are opaque identifiers; the register does not check their content, so a `test-fixture-` reference is accepted like any other and only the verification record distinguishes fixtures. Who is qualified to certify is a role for the integration with `modules/auth` (residual).
- An identical repeat by the same actor returns the view; any other raise to the current level or below is `level_down`; an unmapped command is `not_mapped`.
- `Revoke(workspace, adapter, item, reason, actor)` marks a command or limit `revoked` by a human other than the author, for example after an ERP upgrade or a wrong result. Revocation keeps who raised the evidence and when, and records who revoked it and when. Revocation is terminal: the item never counts again and cannot be raised (`revoked`). A new adapter is needed to restore it.

`Certification(workspace, adapter)` returns `production` when every lifecycle command (`submit_draft`, `read_draft`, `post`, `get_status`, `correct`, `get_balance`) is at `production`, `sandbox` when every one is at least `sandbox`, and otherwise `uncertified` with the blocking commands. A simulated or revoked command never certifies.

`Capabilities(workspace, adapter, minimum)` returns the commands at or above `minimum` (`sandbox` or `production` only). `Admits(workspace, adapter, minimum, capabilities)` returns the capabilities of a ledger connection that the adapter does not support at that level. The merged `modules/erpledger` still accepts self-declared capabilities at `Connect`; binding a connection to an adapter, refusing unadmitted capabilities and checking that the connection's transport matches the adapter's methods belong to the integration layer (residual). A test records this gap.

## Delivery gate

DB-05 diagnoses delivery per connection. `Diagnose(workspace, adapter, target, Delivery{AutoAttach, AutoMail, Portal, Attachments, RecipientControl, AttachmentControl}, actor)` records a numbered diagnosis for one target (the ledger connection or template identifier) under one adapter; diagnoses never transfer between targets. `Attachments` lists what the ERP template adds (`invoice_pdf`, `invoice_xml`, `statement`, `internal_cost`); the controls are evidence references or empty. The result is:

- `blocked`, when any automatic channel is on and `internal_cost` is attached (`internal_attachment`), recipient control is unproven (`recipient_unproven`) or attachment control is unproven (`attachment_unproven`);
- `allowed`, when automatic channels are on and none of those holds;
- `manual_only`, when every channel is off; it still reports `internal_attachment` when the template attaches an internal file, so a manual sender is warned.

`EnableDelivery(workspace, adapter, target, diagnosis, actor)` enables automatic delivery for the target by a human other than the diagnoser. It is refused unless the named diagnosis is the target's latest (`stale_diagnosis`) and is `allowed` (`delivery_unsafe`). Enablement records the diagnosis it relied on and its delivery settings, and stays tied to it while later diagnoses of the target are `allowed` with the same settings; a later diagnosis with different channels, attachments or controls disables delivery until a second human enables it again. Target identifiers are not checked against the ledger; that check is part of the binding residual. Any later diagnosis of the target that is not `allowed` disables delivery at once. Delivery state is independent of issue or posting state. Approved manual routes are owned by HB-27 and HB-29 (residual).

## Chunk planning

`Plan(workspace, adapter, rows)` splits 1 to 100,000,000 ERP document rows (anything else is `ErrInvalid`) into documents of at most `MaxRows` rows and returns each document's row range; a plan that would need more than 100,000 documents is `ErrLimit`. It refuses with `limit_unproven` while `max_rows` is below `sandbox` or revoked. Cost rows are aggregated into document rows before planning, by the integration layer.

# Tests

- AUD-DB-05: an internal cost attachment blocks automatic delivery even when the ERP document is posted, and the controls must be proven. With all channels off, the result is manual-only, and enabling requires a second human.
- The products are kept apart: evidence on an S/4HANA adapter does not certify an ECC or Business One adapter with an identical method map, adapters cannot be rebound, and vendor and product pairs are checked.
- Revocation removes a command from capabilities and certification; an enabled delivery is disabled by a later blocked diagnosis; enabling with a stale diagnosis is refused; a manual-only template with an internal file reports it.
- A simulated pass leaves the adapter `uncertified`, and `Capabilities` at `sandbox` excludes simulated commands. Certification follows the evidence of each command.
- AUD-DB-11: the `database` transport is refused in method maps.
- A test-only simulated ERP drives `modules/erpledger` through prepare, submit draft, read draft (with amounts recovered and compared), post, final document, linking, reconciliation and correction. It also drives a timeout resolved by polling and a manual edit leading to divergence. The test connects the ledger directly with the capabilities it needs, and separately checks `Admits` against an adapter; evidence raised in tests uses references with the reserved prefix `test-fixture-`. The verification record states that no sandbox exists and that this is simulated evidence, not certification.
- Forms, limits, repeats, isolation, a property test of `Plan` against an independent model, race, a 386 build, gofmt, vet, coverage and mutation testing.

# Provisional choices

- The lists of vendors, products, deployments, levels, lifecycle commands and attachment kinds.
- System actors may record simulated evidence only.
- `Capabilities` accepts only `sandbox` or `production`.
- `Plan` splits by row count alone.
- `get_balance` is a lifecycle command because partial collection depends on it.
- The first pilot product is not chosen here; the owner decides it.

# Review history

- Design review 1 (independent reviewer, read-only): CONDITIONAL PASS with 3 high, 5 medium and 5 low findings. Folded in: the ledger binding stated as a residual with `Admits` and a test recording the gap; `Revoke`; enablement bound to the latest numbered diagnosis and disabled by any later unsafe one; diagnosis per target; internal files reported in manual-only mode; approved routes as a residual; test fixtures marked and the round trip run without fabricated certification; explicit evidence rules; per-limit evidence; transport check as a residual; `get_balance` in the lifecycle; explicit pairs and ranges; meaningful separation tests.
- Design review 2 (same reviewer): PASS; all 13 findings resolved. Three low wording gaps folded in: identical repeats win over `level_down`, fixture references are accepted and only the record distinguishes them, enablement stays tied to its diagnosis and targets are unchecked; residuals merged into one list.

# Change review resolution

Change review 1 (a separate independent reviewer, read-only): PASS with 5 low findings and one note. All were applied: a revoked item is refused before any repeat check; revocation keeps the raiser and adds `RevokedBy` and `RevokedAt`; the 100,000-document plan bound is in the plan; a later manual-only diagnosis is tested to disable delivery; the ledger gap test calls `Connect` directly with unadmitted capabilities and names the residual in its failure message; the round trip reads the final document. The note was adopted: an enablement is tied to its delivery settings, so widening channels needs a new enablement.

Change review 2 (same reviewer): PASS; all findings resolved. Recorded observation: `Basis` describes the approved settings only while `Enabled` is set.
