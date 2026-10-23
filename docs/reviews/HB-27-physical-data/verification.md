---
type: Review Report
title: HB-27 exact-head review and physical-design verification
description: Records independent review of the engine-probe PR and the separately scoped physical-design follow-up without claiming unexecuted product acceptance.
status: draft
---

# HB-27 verification

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Accountable owner: Seung Woo Park. Machine: `mac_mini`. Root owns edits and integration; separate read-only reviewer and DBA own independent assessments. Date: 2026-10-23.

## PR 22 review

Independent reviewer `pr22_review` reviewed exact head `c8f82874ed705642f7c8f023c4c06cafd8c014cc` against base `a68c0756ada690087b717d6c6105e0904267128f`: **PASS**, no actionable merge blockers. Actual scope: 23 files. This reviews the resulting implementation, including the final minor changes; their individual historical diffs cannot be reconstructed from the single engine-probe commit.

Actual read-only checks: all 14 evidence-manifest file SHA-256 values match the head; retained raw outcome hash, timestamps, exit status and all 51 case outcomes match the committed report (47 CONFIRMS / 3 CONTRADICTS / 1 NOT_RUN). Python AST, XML parsing, Markdown metadata/local links, physical source sizes, common credential patterns and current local credential literal scan passed; no values were printed. `git diff --check a68c075 c8f8287` passed. Reviewer did not execute engine tests or mutate external state.

Root independently confirmed the four HB verification containers were healthy. Root reran `tests/engine_probes/.venv/bin/python -m pytest tests/engine_probes -q`: exit 0, 51 cases; local run `20261023T020837Z-c3ddc5` recorded 47 CONFIRMS / 3 CONTRADICTS / 1 NOT_RUN, identical to the historical classification; the historical [engine report](../../validation/reports/20261022T035254Z-bac19a/summary.md) remains unchanged. No demo, product, capacity or live-connector acceptance follows from this review.

## Scope and ownership

The owner instructed proceeding with PR 22 merge and the remaining physical design on 2026-10-23. Correct Linear workspace, team, project, accountable owner and sole existing `machine:mac_mini` claim were verified through the repository's bound connection. The generic Linear connector could not resolve this ticket and was not used for updates. No duplicate ticket was created. PR 22 merged as `9a5bbb5` after owner-authorized browser merge through the existing repository-owner session. Connector merge was denied (403); the CLI account had read-only repository permission and its merge attempt failed (404). Neither failed attempt is recorded as success. Browser confirmed merged/closed. Remote fetch/read-back confirmed the merge, an empty `git log --oneline feature/HB-27-physical-data-verification --not origin/main`, and identical Compose contents. No automated commit statuses/workflow runs were returned; the browser reported ready to merge. Physical-design verification follows below.


## Physical-design follow-up

Root wrote the first-path [physical schema](../../architecture/data-ownership/focus-physical-schema.md), [mapping](../../architecture/data-ownership/focus-consumed-columns.md), [lifecycle prerequisites](../../architecture/data-ownership/focus-lifecycle-protocols.md) and [official demo inventory](../../validation/reports/HB-27-focus-demo-inventory/summary.md), plus the existing build-design links/missing-value correction. No engine code, historical report or production data was changed.

Separate researcher `focus_inventory` profiled pinned official demo bytes (5,488,359 rows) and compared 20 independent Decimal cost groups: PASS for profiling, not product acceptance. The original 1.0 sample remains exploratory-only. Root preserves aggregate JSON and hashes, without source identifiers or rows; original download and analysis scripts have local-only retention. DBA design advice was incorporated before writing the proposal. Actual draft review: independent DBA `physical_design_review` PASS for design acceptance after re-review; source specialist `focus_inventory` PASS; architecture/security/change reviewer `pr22_review` PASS. DBA identified a HIGH risk where attribution revision could create a fresh publication pointer; root changed uniqueness to stable scope and made split/merge/remap retire/install pointers atomically across all affected scopes. Re-review confirmed resolution. Two MEDIUM receipt identity/digest clarifications and a LOW official source-path correction were applied and independently re-reviewed. No blocking findings remain.

HB-27 remains In Progress: exact numeric/provider profiles, executable DDL/loaders, supported-version fixture decision, demo ingestion/marts and adversarial lifecycle tests remain pending. No deployment, live-account access, financial action, customer-data use or external operational message was authorized or performed. Local verification stack is left running for this ticket.


## Document QA and remaining execution

Root and independent reviewer checked six Markdown documents for required nonempty type metadata, local links and balanced fences; five JSON artifacts parsed; all four inventory-manifest file hashes/sizes matched. Root reconciled provider/service row counts, Datadog count and all 20 Decimal groups/missing counts against the separate source pass. Credential-pattern and current-local-credential literal scans found no matches in the eleven scoped files; no values were printed. Whitespace validation passed. Only the two existing documents plus three new physical-design documents and six inventory files are in scope; unrelated untracked workspace assets remain untouched.

These are source/document checks and design acceptance. No executable DDL/migration, supported-version end-to-end ingestion, lifecycle concurrency/outage test or throughput certification is asserted. Follow-up publication remains pending at this revision; HB-27 is not Done. Next work specializes reviewed profiles and executable constraints, selects the supported demo/adapter path and runs the explicit acceptance cases before claiming completion.
