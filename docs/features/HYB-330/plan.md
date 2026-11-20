---
type: ImplementationPlan
title: HYB-330 truthful refusal diagnostics for administration screens
description: Preserve confirmed error codes and request identities through command failures and closed refusal forms without changing authorization or retry rules.
status: stable
---

# Scope and evidence

Root owns PM, implementation, documentation and integration on machine mac_mini. Independent read-only reviewer checks diagnostic truthfulness, async scope and actual diff; separate QA/TW checks rendered EN/KO/AR. Ticket: https://linear.app/myblueton/issue/HYB-330. Baseline: 653adfe8bb8e1804aaeeea10accc4920fbf14e62.

P02 spec.md:102 says: “오류는 안정 code·request ID·필드 위치·다음 조치를 표시하고 비밀/고객 원문 stack trace를 노출하지 않는다.” Lines 98–100 require the next step for failed/revoked states. Existing P02 CommandForm drops error metadata when closing to RowNote, and AdminError drops request_id. P01/P03/P04 share this defect. P05 has ErrorDetail for some in-form errors but kept refusals close the form and lose the detail. Inspecting those consumers is therefore in this ticket.

# Planned behavior

- Extend AdminError with an optional confirmed response request ID, preserving existing constructor callers. Parse bounded code and request tokens from actual JSON only. Do not synthesize stable codes from HTTP status, thrown objects or transport failures. Retain diagnostic metadata when applying Retry-After or step-up cancellation; mark cancellation as local, not a second server refusal.
- A shared administration diagnostic region, labelled “Action request references”, records the latest command attempt on P01–P05. Show response code and server request ID (or explicit unavailable), separately label the client request ID when known, and canonical target with no rationale or command values. This persistent page-level region supplements existing row/form explanations and next steps when forms close. Preserve current field links, retry/reauth/idempotency, permission and blocker semantics.
- The supplementary latest region clears when a new command starts or succeeds. Retained row/form diagnostics follow their own actual refusal/error state. Keep through reload and same-scope remount of retained notes. Do not let a late older request replace the latest command or cross workspace, user, session, screen, view or unmount. The region is scoped in memory; no persisted diagnostics or owner FIX.
- Surface read error metadata in the existing load-failure region and denied/session-ended variants, with unknown values explicit; no invented correlation. Existing flow controls remain authoritative.
- Keep neutral current feedback styles, full width, local fonts, zero radius and bidi isolation. No material screen recomposition; reuse existing definition-list/notice pattern. Canonical item scope P01–P05:error:diagnostics. Current owner FIX is not claimed.

# Ownership and allowed files

Root is sole writer: service/admin.ts and its tests; a shared diagnostic model/component and tests; five administration Live.tsx entry points; their existing CommandForm/useOpen/RowNote and P01 manual close/kept-note consumers; P05 existing error-detail renderer; five scoped refusal-marker models and P02 lost-right markers; package test list; HYB-330 records and binding register/field maps. No Go/product contract change or permission change. The read-only reviewer may inspect all related producer/consumer code but edits nothing.

# Follow-up acceptance corrections

The first implementation passed190 browser assertions, but root requirement audit and independent re-review found missing cases. The current design enriches each existing scoped refusal marker with sanitized command/target references. Metadata is retained, pruned and forgotten with that exact marker, including tab remounts. P02 lost-right markers carry their corresponding reference too. Each row shows separately labelled Action ID and Target ID entries so multiple actions cannot be confused. There is no separate diagnostic history cache.

Each open form also retains its own diagnostic state, including server field errors, independent of the supplementary latest-action region. A mounted session/command/target generation guard precedes result/failure side effects, follows asynchronous reloads and guards finally updates; stale completions cannot write markers or references in a newer scope.

Independent TW found P01/P03/P04 read failures reused sign-in instructions while already authenticated. These now use loading/service-specific copy. Command fallback copy on P01–P04 describes an unconfirmed outcome and directs the user to check current state, preserving existing retry and field links.

Observed fail-first checks: original closed-refusal diagnostic check failed on unchanged main;12 new checks failed before per-row/copy correction; three remount checks failed with component-local diagnostics. Two-open-form baseline checks separately exercise independent rate-limit and server field-error messages. These findings are not dismissed by earlier green checks.

# Verification

Add diagnostic parser/transport and latest-attempt/scope tests before implementation and observe failure on unchanged source. Then component render tests and real entry-point browser injections cover kept refusal, reload, rate limit, denied/session end, missing/malformed metadata, lost response, step-up cancellation and delayed reply. Browser assertions must fail on baseline before the fix where feasible; new test modules importing new code cannot execute on baseline, so report import failure distinctly. Inspect EN/KO/AR at1440/1280/1024/390 including long tokens and focus. Console tests, type/build, targeted style/asset checks, docs format, whitespace and line counts. No new Go behavior requires a redundant whole server/race rerun. Independent actual-change review and scoped rendered/TW review before PR; merge only with CLEAN/read-back, then release machine.

# Design review

Round 1 /root/hyb310_review: FAIL M1. P05 already substitutes local failure names for server codes and labels the client attempt as request ID. Scope now explicitly replaces that metadata presentation while preserving its next steps and field links. Null/array/scalar JSON, bounded metadata, latest-scope-before-effect cleanup and local-cancel challenge provenance are mandatory checks. Other plan points accepted conditionally. Round 2: PASS on revised design, including scoped wrapper/generation approach; source/runtime review remains pending.

# Boundaries

Native-language and assistive-technology approval, designated Claude CLI success, actual Linux/Windows execution and owner FIX are not inferred. Existing remote CI is unconfigured. External visual evidence for this existing diagnostic pattern is inherited from HYB-310/333; no new screen composition or external asset is added. This document does not pre-certify tests or reviews.

# Integration scope

The published PR238 and PR243 candidates are consolidated without rewriting either history, together with merged HYB-327. Preserve original primitive refusal semantics through shared `Refusals<T>`, deep-copy nested diagnostics, retain terminal/notice references, clear exact previous markers on new attempts, invalidate reads with monotonic scope generations and bind grouped P02 actions to canonical request targets. Final implementation and verification are described in [implementation](implementation.md) and [integration verification](integration.md).
