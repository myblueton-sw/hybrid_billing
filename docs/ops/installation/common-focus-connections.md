---
type: Runbook
title: Installation-scoped FOCUS connections and guarded intake
description: Configures independent source periods, expected-domain coverage, pinned consumed semantics and admitted quarantine inside a customer-operated installation.
---

# Operating boundary

HB-30 extends the trusted single-host [collection/lifecycle CLI](focus-collection-lifecycle.md). Commands use `tests/engine_probes/.venv/bin/python tools/focus_lifecycle.py`. Scenario actor names are a local verification boundary, not production identity authentication. Configure real installation authentication, a supported installer/upgrader, service supervision and measured capacity separately. No live provider API fetch, hosted SaaS dependency or MSP settlement is introduced.

# Independent source connections

After existing tenant configuration and `activate`, use `connection-register --actor operator --file CONNECTION.json`. Required connection fields:

| Field | Contract |
| --- | --- |
| id, dataset, coverage_key | Stable lowercase identifiers; provider display name alone is not economic identity |
| version | Declared source version selected through the mapping registry;1.2/1.3 consumed profiles supplied |
| grain | day or month, independently for each connection |
| start, end | Explicit bounded UTC calendar; at most3660 windows per connection |
| lag_seconds | Explicit expected delivery lag,0–604800 |
| domains | Exact provider/account/subaccount triples already authorized by active tenant assignments; JSON null subaccount requires an explicit null assignment |
| semantic_profile | `focus-consumed-v1`, or explicitly `exploratory-query-only` for projection evidence |
| capabilities | Optional complete capability definition for the consumed validator; see the pinned connection response/registry. No capability is inferred from provider/service display names |
| max_replacements | Explicit1–10 attempt limit shared across an admitted receipt's replacement lineage |

Connections are immutable and pin their full semantic definition. Overlapping schedules sharing an economic domain reject, even across different grains. New connections cannot overlap legacy unscoped windows; unscoped deliveries cannot overlap registered connection calendars. Existing historical snapshots remain valid and are not silently adopted/reinterpreted. Use disjoint evidenced source calendars or a separately reviewed adoption migration. A new account is not authorized merely by appearing in a source file.

# Describe, register and publish

Use `describe --file FILE --version 1.3 --connection ID --start UTC --end UTC --grain day --revision 1 --complete-through UTC --finalized`. Save the returned descriptor. `--zero-domains FILE` supplies exactly the missing expected domains as entries with `domain`, safe `reason` code and SHA-256 `evidence_digest`; observed/duplicate/foreign attestations reject. This is pinned operator evidence, not independent upstream verification. For an explicitly exploratory1.2/1.3 connection, `describe --exploratory` retains legacy projection token/header behavior.

Run `register --actor operator --file FILE --version VERSION --descriptor DESCRIPTOR`. A successful delivery has `delivery_id`; an invalid admitted source has `state=quarantined` and `receipt_id`; unsafe/unmapped/unclassifiable input has safe `state=rejected` metadata with no retained raw original. Do not send a quarantined/rejected receipt to `collect`.

Then use the existing `collect` or `run-due`, independent `confirm`, and `publish` commands. Corrections replace only that connection/dataset/coverage/window. Physical transport revisions remain separate from source economic revisions. Scheduler scans at most10000 directory entries and skips unreadable/oversized unregistered candidates; it does not permanently exclude a registered suffix after100 files. Exceeding the bound is an explicit error; partition the local inbox. Publication remains separate from file registration and does not certify financial billability.

# Consumed fields and expected coverage

Consumed profiles validate30 normalized semantic fields with version/capability predicates, exact bounded decimals, unit/quantity/price checks, source formats and provider/host/issuer distinctions. Source lexical timestamps require terminalZ; typed Parquet timestamps are normalized to UTCZ. Tags retain validated original JSON text, with unique keys and scalar values, so numeric precision is not lost. Conditional headers are enabled by explicit pinned provider capabilities. Query rows expose `semantic_context`; physical facts/archives retain their existing projection and exact original evidence. The full definition/digest is pinned, so current registry changes do not reinterpret retained releases. This is consumed-field validation, not full provider FOCUS conformance or invoice reconciliation.

Query responses add `expected_coverage` for only the reader's authorized tenant domains. Expected but unseen windows are missing/late; present intake can be registered/verified/sealed, with separate current publication/finality/as-of and known-zero evidence. Admitted unresolved receipts expose quarantined/replacement-requested/escalated state without foreign receipt content. Coverage includes connection/account/calendar expectations, rather than a service-specific zero assertion. Snapshot pins include expected coverage so affected paging does not silently span a coverage change. Legacy expectations remain explicitly operator-asserted/unverified. Enumeration and fact selections retain10000-item bounds.

A null subaccount stays null in source/facts. Its hierarchy uses the actual observed billing-account node for account-only context, marks `account-only-subaccount-absent` and adds no invented CSP ID or self-edge. Shared account ancestry still grants no access.

# Reopening and replacement approvals

To reduce complete-through or unfinalize a finalized current delivery, prepare a JSON plan using `reopen-plan --actor operator --file PLAN`. The body has `connection_id`, the exact new `descriptor` without plan references, safe reason code, explicit `expires_at` (within7 days), and `receipt_id:null`. A distinct `approve --actor lifecycle-approver --manifest-digest DIGEST` binds the new file, predecessor and connection definition. Add `reopen_plan:DIGEST` to registration. Changed input, predecessor, expiry or reused approval rejects.

A fully authorized file with missing expected account coverage or invalid numeric values is admitted quarantine, not an unsafe-source rejection. For admitted unusable source data, use `replacement-plan` with the same plan fields and the original `receipt_id`. It binds exact connection/window/version, predecessor bytes and both descriptors; the same bytes may be corrected only through an approved changed descriptor (for example new zero evidence); approval does not send messages. Add `replacement_plan` to the new descriptor. Changed bytes for a scope with unresolved quarantine cannot bypass this flow. Failed replacements share the root receipt budget and propagate terminal escalation; a child receipt cannot reset it. Approval consumption and attempt reservation enter the durable pending intent before an external quarantine PUT. An uncertain effect remains closed until a separately reviewed reconciliation path is used; there is no automatic takeover.

# Quarantine protection and recovery

`status --actor operator` exposes trusted operator receipt/connection metadata. Admitted originals use an installation-qualified restricted quarantine prefix; rejected metadata never contains raw input. `receipt-retention --actor operator --receipt ID [--hold KEY] [--release]` reports eligibility or changes a hold. Retention honors every affected tenant's declared received-time/window-end basis and backup horizon. Shared originals use the last applicable horizon; no legal duration is invented.

Eligible removal uses `receipt-delete-preview --actor operator --receipt ID`, separate lifecycle approval, then `receipt-delete --actor operator --manifest-digest DIGEST`. New holds or relevant changes invalidate approval. Raw cleanup closes that receipt connection/window to new intake and preserves lineage denial/budget metadata; it does not reset retry authority. Managed backups may retain copies; no physical all-copy erasure is claimed. Backup/restore includes admitted quarantine bytes under original horizons and latest independent restrictions. Missing current dependencies, expired horizons or unknown operations block restore. Retirement refuses retained quarantine dependencies. Caller inbox/download copies remain outside managed cleanup.
