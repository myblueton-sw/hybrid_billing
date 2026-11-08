---
type: DesignSpecification
title: HB-59 textual coverage and verification
description: Maps every selected audit finding to a detailed draft scenario and records actual review limits and adoption gates.
status: draft
---

# Evidence and scope

[HB-59](https://linear.app/hybrid-billing/issue/HB-59) follows [HB-58 immutable audit](https://github.com/myblueton-sw/hybrid_billing/blob/b87e723c79650e9e558caa82633239d6252c2a8a/docs/reviews/HB-58-planning-coverage/report.md). Branch base is current fetched origin/main `b48b23c`; the HB-58 PR is not a merged dependency. Original v1.16 source SHA-256 recorded in the audit is `e7772927bc521ee035d50a7762b2f75611b80b2162e385878dc4c36c6c0db84d`; B locators are zero-based DOCX body children and R denotes table rows. This work uses that audit and the linked current contracts; it is not a new exhaustive extraction or rendered-book validation.

Root owns PM/planning, writing and integration. Separate read-only administration/security, financial and operations reviewers each returned PASS on the design plan before drafting; their constraints were incorporated. Actual-document review and root QA results are recorded below when completed. No named role grants product authority.

# Finding-to-scenario trace

Each referenced section includes entry/capability, fields/validation, action sequence, normal and blocked/recovery behavior. The shared contract supplies current rights, scope, errors, concurrency, job state and accessibility requirements. The following are textual planning cases, all runtime NOT_RUN.

| Finding | Draft location | Scenario / acceptance assertion |
| --- | --- | --- |
| ADM01 | [Access](../../02-access/r2/administration.md) M01/M03 | A01: scoped invite/grants; expired/revoked token refused; stale/bulk partial result handled |
| ADM02 | [Access](../../02-access/r2/administration.md) M02/L03 | A02: independent trust activation; current epoch; unavailable provider/unknown commit fails closed |
| ADM03 | [Access](../../02-access/r2/administration.md) M04 | A03: owned service scope/version; partial rotation and urgent revoke |
| ADM04 | [Customers](../../04-work/r2/administration.md) O01/O02 | W01: typed relationship move preserves allowances/history and complete atomic units |
| ADM05 | [Customers](../../04-work/r2/administration.md) O03/I01 | W02: existing-channel verification and independent approval; issued snapshot unchanged |
| ADM06 | [Customers](../../04-work/r2/administration.md) lifecycle | W03: ended/pending_final distinct; late credit lineage; security revoke never waits |
| ADM07 | [Customers](../../04-work/r2/administration.md) U01/O01 | W04: profile-aware amounts, freshness and current rights; no mixed-currency total |
| ADM08 | [Access](../../02-access/r2/administration.md) M05 | A04: proved link/last-method protection; personal notification scope only |
| OPS01 | [Work](../../04-work/r2/work-management.md) U02 | T01: stage/calendar revision and explicit DST; payment due date separate |
| OPS02 | [Work](../../04-work/r2/work-management.md) rules | T02: approved rule; recipient recheck; read/ack/accept/resolution separate |
| OPS03 | [Work](../../04-work/r2/work-management.md) U03/U04 | T03/T04: fixed targets, partial results, unknown external effects and evidence-based anomaly review |
| OPS04 | [Operations](operations.md) P01 | O01: existing-data preflight, migration-aware recovery and bounded support |
| OPS05 | [Operations](operations.md) P03 | O02: isolated restore, latest controls, fencing, external reconciliation and cutover |
| OPS06 | [Operations](operations.md) P04 | O03: hold race, shared evidence, denial/cleanup/backup separation and authorized recall |
| OPS07 | [Evidence](evidence-and-policy.md) R02 | E01: cross-domain audit with masked evidence and current disclosure rights |
| OPS08 | [Evidence](evidence-and-policy.md) R01/I04 | E02: fixed complete export, saved-view scope and revoked artifact denial |
| OPS09 | [Evidence](evidence-and-policy.md) P05/P06 | E03: support evidence distinct from mock-only scenarios |
| OPS10 | [Evidence](evidence-and-policy.md) P01/A01 | E05: explicit egress approval; core flows usable without AI |
| DOC01 | [FX supplement](../../06-contracts/r2/fx-settings.md) B05 | F01: positive raw rate/unit; zero/signed components according to approved field policy |
| DOC02 | [Evidence](evidence-and-policy.md) P07 | E04: established modes, nonblocking expiry, verification uncertainty and user contact |
| DOC03 | [Evidence](evidence-and-policy.md) P06 | E03: required profile/purpose, independent expected bundle, sample compare disabled |

# Completion and remaining gates

Textual coverage means each of the 21 findings has a proposed detailed path, not that the corresponding runtime finding is closed. Canonical commands are reused only where registered. Unresolved bindings require typed request/result, permission mapping, atomicity/idempotency, error/status and read-back design before implementation. Future implementation QA must execute the named scenarios, including cross-tenant negative cases, revoked-authority races, changed revisions, unknown external outcomes and partial retries.

Adoption order: owner reviews proposed module scope → separate tickets resolve bindings and policy choices → independently reviewed implementation → actual UI/accessibility and runtime QA → explicit module acceptance/replacement manifest → update assembled edition and remove only accepted superseded scope. HB-24 local companion publication/adoption and unrelated prior findings are not closed here. Identity vendors, actual values, support combinations and retention/recovery bounds remain decisions, not implicit defaults.

Scope exclusions retained: conditional optional plugins; deferred authentication selection and onboarding visual redesign; HB-54 payment/deposit/master-child, receivables, monthly ERP/quarterly aggregation, VMware detail, disclosure and full AI workflow remain outside this focused revision. Existing v10 financial and navigation content remains active until explicitly replaced. No code, generated UI, invitations, credentials, customer data, deployment, business operation or merge is included.

# Actual verification

Plan review: PASS from three independent read-only reviewers before writing. Actual-document review: PASS from administration/security, finance and operations reviewers after scoped re-review. Corrections retained permitted overlapping reporting groups with aggregate deduplication, conditional saved-view sharing, the separate ED-06 training gate, a complete U04 anomaly review path and explicit calculation/reconciliation before final approval. No reviewer edited files.

Root QA: PASS for all 8 new documents having required nonempty metadata and draft status, every local Markdown link resolving, exactly one trace row for each of 21 selected findings, and all 21 explicitly named command IDs existing in the canonical registry. A local read-only Python check performed those assertions; `git diff --check` passed. Root read the complete draft and staged scope for synthetic examples, authority/financial boundaries, secrets and unrelated files. These checks validate document consistency, not executable contracts. Rendered UI, runtime, security enforcement, provider/ERP connectivity and performance: UNVERIFIED / NOT_RUN. No product checklist item is marked executed by textual review.
