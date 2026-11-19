---
type: ImplementationPlan
title: HYB-330 truthful command refusal details
description: Preserve server error codes and request identities in inline and retained P01 through P05 feedback.
status: draft
---

# Scope and ownership

Ticket: https://linear.app/myblueton/issue/HYB-330. Accountable owner: Seung Woo Park. Existing machine claim: mac_mini, verified through Linear. Base: 653adfe. Branch: fix/HYB-330-refusal-details. The owner authorized continuation without restoring missing companion files. The ticket quotes P02 specification clause 102; the full local P02 specification is unavailable.

Root owns shared service parsing, P02 implementation, terminology, integration and verification. The platform implementer owns only P01/P03/P04/P05 files in an isolated worktree. The independent reviewer is read-only. No backend permission or financial contract changes are planned.

# Behavior

Preserve bounded string code and request_id from the actual error response. Boundary failures use the ID generated in boundary.go; executed command results can echo body.RequestID through commands.go and errors.go. Therefore display the response identity, which may legitimately equal the browser attempt ID, rather than substituting the attempt when response metadata is missing. Preserve metadata through Retry-After handling and use the final response after step-up. Network failures, local validation and cancelled step-up must not invent server details.

Keep retained details in the same entry and lifecycle as the matching command/target refusal. Grouped rows select the same command for their message and details. P02 permission-loss notes have their own retained store and require the same treatment. Success, pruning and scope changes remove both. Inline validation and command feedback retain the response details while shown. Existing next-step text and retry semantics remain.

# Presentation and terminology

Use existing full-width feedback regions, neutral separators and no decorative borders. Add definition-list details below the message; isolate machine values LTR and wrap long IDs at mobile widths. This is a small feedback addition, not a new page composition. English: Error code / Request ID; Korean: 오류 코드 / 요청 ID; Arabic: رمز الخطأ / معرّف الطلب. Existing P05 terminology informs the ID label. Native-language acceptance remains pending.

Professional references inspected on 2026-11-20:
- https://learn.microsoft.com/ko-kr/entra/identity/devices/troubleshoot-device-dsregcmd: separate error code and server request ID for diagnosis.
- https://learn.microsoft.com/ar-sa/azure/active-directory/reports-monitoring/concept-sign-in-log-activity-details: error reason plus request identity for investigation.

Adopt their separation of human explanation and diagnostic identity. Do not copy their layouts or replace supported next steps with raw server messages.

# Review and validation

Independent design review by refusal_review: PASS with constraints on server identity, throttling, step-up, grouped actions, permission loss and cleanup. The reviewer ran no tests. Validate parser metadata, final-response identity, malformed or missing values, network/cancelled failures, retained lifecycle and grouped-action selection. Run console typecheck, relevant tests and build, then inspect EN/KO/AR desktop/mobile feedback. Independent actual-change review follows implementation. No owner FIX or deployment is implied.
