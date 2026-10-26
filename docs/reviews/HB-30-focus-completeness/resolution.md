---
type: RemediationReport
title: HB-30 installation-scoped common FOCUS remediation
description: Connects independent source coverage, pinned consumed semantics and guarded receipt resolution to the installed common ingestion path.
---

# Scope and compatibility

The owner requested implementation of all seven [audit findings](findings.md). Root integrates the existing HB-30 branch/PR26; an isolated implementer owns semantic contracts and strict parser fixes, with separate DBA/security design/change review and QA. README/licensing continues to describe a customer-operated installation under MIT. No SaaS hosting dependency, cloud credentials, private customer input or MSP settlement is introduced.

Retained exploratory snapshots keep their original fact shape, mapping, control digests and graph definitions. New explicit connections add source identity and a pinned consumed semantic profile. Normalized semantic contexts are derived from verified retained source_json using that full pinned profile; changing today's registry does not reinterpret history. The separate synthetic strict parser is corrected too, but its passing tests are not substituted for common-path integration acceptance.

# Finding resolution map

| Finding | Implementation | Verification responsibility |
| --- | --- | --- |
| F01 independent source identity | `connections.py` registers exact connection/dataset/economic coverage and per-connection calendars; journal window/current/revision keys include that identity. Disjoint domains may use different grains; shared-domain/legacy overlaps reject | Actual independent AWS day/Microsoft month publication and correction30→27 without changing the other source; explicit overlap negatives |
| F02 omission detection | Expected domains must be observed or exactly covered by non-overlapping zero attestations. Queries expose reader-scoped expected calendars and missing/late/intake/publication/zero states, with expectation pins in paging snapshots | Authorized missing-account quarantine, approved same-byte zero-evidence replacement, foreign/unclassifiable metadata-only rejection, tenant/company isolation and cursor invalidation |
| F03 consumed semantics | New1.2/1.3 profiles validate30 fields with version/capability predicates, exact quantities/prices/costs, units, SKU and provider/host/issuer distinctions; contexts are returned by common queries | Pure semantic regressions plus actual CSV/typed-Parquet ingestion, independent controls and authorized query contexts |
| F04 scheduler fairness | Bound all directory entries at10000; scan qualifying files beyond the former first100; skip oversized/unreadable unrelated candidates | Pure reproduction now collects registered suffix after100 candidates; explicit bound over irrelevant directories |
| F05 receipt resolution | Admission precedes raw retention. Unsafe source gets bounded safe metadata only. Admitted unusable source gets installation-qualified immutable quarantine; digest-bound independent replacement approval, lineage budget, backup/restore, holds and approved retention cleanup | Actual original bytes/restore, no-plan bypass denial, invalid replacement budget, lost-S3-ack pending-intent evidence and cleanup/re-intake denial |
| F06 conditional/strict parsing | Explicit NULL-domain authorization and observed account-only graph; capability-dependent headers; terminalZ lexical time; unique scalar Tags preserved exactly; pinned enum/format checks | Actual NULL graph/query/archive/restore, exact Tags decimal lexical value and pure conditional-capability/source-format negatives |
| F07 finality reopening | Regressing as-of or unfinalizing requires separate expiring predecessor/new-descriptor-bound approval, consumed with registration | Denial without approval, changed body denial, approved reopen and retry idempotence |

# Review-driven corrections

Independent semantic review initially rejected offset timestamps, missing SKU applicability and unconditional conditional headers. Validator revision2 fixes these; retained revision1 profiles preserve their pinned historical interpretation. Typed Parquet instants become UTCZ representations while arbitrary source strings are not rewritten into valid lexical input.

Independent security review initially rejected post-effect approval consumption and resettable child attempt budgets. The pending effect now durably contains approval consumption and root lineage counter before quarantine PUT; descendants share terminal escalation. Raw receipt cleanup retains a closed-intake control, preventing expired/deleted quarantine from silently restoring retry authority. Replacement copies retain the lineage's receipt-time origin; window-end policies continue to use the source window.

The initial localhost test attempt was sandbox-denied; a subsequent attempt found healthy-container/internal ClickHouse but a broken host forwarding response. Restarting only the local verification ClickHouse container restored the host port while preserving its volume. No unrelated containers were restarted. Test-fixture resets affect disposable hb_focus only, followed by restoration of the independently retained sample backup.

# Verification

Full regression run `20261025T122007Z-6a0b3a`: **337 PASS**, zero failures/errors/skips. The final admission/completeness/origin changes then pass14 targeted cases in `20261025T141836Z-628bbc`; parser-error normalization passes2 additional real-stack cases in `20261025T142128Z-57afc2`. These counts overlap: they are a full regression plus scoped checks after the final changes, not353 unique tests. [Evidence](remediation-evidence.json) pins JUnit and final source hashes.

Intermediate unsuccessful attempts are retained honestly in evidence: initial sandbox/host-port failures,420-second timeout, one escalation test-oracle mismatch, one internalUTC serialization expectation mismatch and one test variable shadowing error. Assertions were corrected where behavior already met the contract; product admission/origin/parser defects received actual fixes and independent re-review.

Default runtime run `20261025T142220Z-16485f` executes six successful CLI commands restoring the existing graph-inclusive legacy CSV backup and verifying company1121, tenantA1112, tenantB9 and foreign2 (1123 overall). Journal sequence58. The sample remains queryable under its original exploratory profile; it was not promoted to consumed/provider conformance. No private customer inputs or live CSP credentials were used. Final independent evidence QA and23-file staged index review report PASS.

See the [runbook](../../ops/installation/common-focus-connections.md) for commands, bounds, required operator declarations, zero-evidence meaning and independent approval roles.

# Remaining boundaries

The verified changes address the seven audited implementation gaps; it is not a full FOCUS provider-conformance certificate, supplier invoice reconciliation, live CSP connector or production installer/authentication/capacity/HA certification. The CLI retains trusted local scenario actors and10000-row query bounds. Connection calendars require explicit initial authorized account domains; onboarding new accounts or adopting overlapping legacy windows needs a separately reviewed configuration/adoption path. Unknown external effects remain closed without automatic reconciliation/takeover. Backup/retention reports managed-copy residuals, not physical all-copy erasure. Caller-owned inbox/download copies are outside managed cleanup.
