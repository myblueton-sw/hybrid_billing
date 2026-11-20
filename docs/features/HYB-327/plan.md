---
type: ImplementationPlan
title: Console response validation and field binding
description: Defines HYB-327 scope, ownership and independent verification.
status: draft
---

# Scope

[HYB-327](https://linear.app/myblueton/issue/HYB-327) is owned by Seung Woo Park and claimed by `machine:mac_mini`. The branch starts at `653adfe` independently of unmerged HYB-330. Root owns planning, integration, UI changes and QA; a separate implementer owns response parsing; an independent reviewer owns read-only design and change review. Missing companion rules remain under the owner's existing restoration waiver.

## Intended behavior

- Validate O01 party/profile rows and M03 access, M04 service, M05 identity row arrays against actual server contracts. Malformed permission or identity data must fail the read with the existing invalid-response path, never silently become an empty valid list or a usable permission.
- Reconcile M02 stricter fields; M01 selection and API-key/pending-job impact; P01 configuration overrides/history/backups; P02 clients section state; P05 gate-profile failure counts against current producers and field maps.
- Preserve existing server-confirmed counts, unknowns and explicit empty states. Do not infer zero, false or not-applicable from missing data.
- Keep existing composition, neutral borders, full-width layout and EN/KO/AR copy. No new visual composition is proposed.

## Ownership and boundaries

The parsing implementer may edit `service/admin.ts`, new response-parser modules and their tests. Root owns all other files, including screen bindings, `identity-bulk.ts`, installation models and evidence. Reviewer is read-only. Concurrent writers use isolated worktrees. No backend contract, billing calculation, deployment or merge changes.

## Verification

Run malformed and valid response tests, focused render tests including unknown/zero/empty cases, typecheck, full console tests and build. Independently review design before implementation and final changes before handoff. Render changed UI in EN/KO/AR at desktop/mobile widths, inspect state truthfulness, overflow and neutral borders. Record exact executed evidence and any unavailable acceptance. Open a PR and release the machine at review handoff; ticket completion follows an authorized merge and read-back.

## Independent design review

Separate reviewer `refusal_review` inspected producer contracts and returned PASS before implementation. Required constraints: validate nested actionable fields and exact booleans; retain omitted optional credential pickup metadata; reject malformed supplied metadata; preserve unfamiliar display-only strings without turning them into action authority. M01 impacts are string states. P05 failures count selected-build non-passed runs after budget approval. P01 summary-only scope and current empty override producer may be retained with an explicit limitation. Failed validation must not leave old actionable rows appearing freshly valid.
