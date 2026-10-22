---
type: Verification Record
title: HB-27 demo-first scope integration verification
description: Records ticket consolidation, document review and bounded checks for the owner-requested FOCUS demo dataset baseline.
status: draft
---

# Demo-first scope integration verification

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Owner: Seung Woo Park. Machine: `mac_mini`. Date: 2026-10-22. Base: `3f6e6bf` on the existing `feature/HB-27-physical-data-verification` branch and [PR 22](https://github.com/myblueton-sw/hybrid_billing/pull/22).

## Consolidation and ownership

The owner requested updating Git and consolidating common ticket work. The authenticated API returned the Hybrid Billing team and project and the assigned owner. The complete current HB ticket title inventory contained no separate demo-review ticket. HB-26 is Done and is the build-design input; HB-27 is In Progress and already covers FOCUS physical design, fixtures and local verification. The clarification was added to HB-27's description, preserving its existing scope, assignee and machine claim. No unrelated ticket was closed or marked duplicate.

Root is sole writer of the new [demo baseline](HB-27-focus-demo-baseline.md), its entry links in the build design and engine plan, and this record. The isolated worktree uses the existing ticket branch. Existing engine code, Docker configuration and run reports are unchanged. This document integration does not finish the broader HB-27 ticket or release the ongoing work's machine claim.

The portable companion rules, skills and binding file remain absent in this workspace. This bounded documentation integration uses the owner's existing authorization to proceed with available evidence; missing policy contents are not reconstructed. HB-24 retains the separate portable-baseline work. No product implementation is introduced.

## Review and checks

Independent read-only final review of the three substantive Markdown changes: PASS. The reviewer confirmed demo-first evidence, preserved HB-26 decisions, explicit handling of the candidate 1.0 sample, separate observed/synthetic/live evidence, Datadog presence UNKNOWN and unchanged engine results. No blocking finding. Root owns this administrative verification record.

Root document QA: PASS for the three substantive documents, including frontmatter type, balanced fences, local link targets, common credential patterns and whitespace. The four-file staged scope, this record and staged diff are checked before commit. These checks establish document consistency only.

Demo selection/download/row inspection, license verification, sample-backed runtime tests and live connector tests: NOT_RUN in this integration. Earlier engine outcomes are preserved, not rerun or newly certified. Physical-design completion and the existing PR's full acceptance remain open. Commit and publication read-back are recorded on the ticket after publication; no merge is claimed by this record.
