---
type: ImplementationRecord
title: Truthful administration refusal and failure references
description: Records how P01–P05 preserve response and client request identity through error, refusal and remount lifecycles.
status: stable
sources:
  - resource: ./plan.md
  - resource: ./verification.md
---

# Behavior

P01–P05 now distinguish confirmed server error codes/request IDs from the browser's original client request ID and canonical target. Missing, malformed or absent response metadata remains unavailable. Transport failures do not manufacture a server code; cancelling local password confirmation associates retained response details with the preceding challenge.

Each open form retains its own diagnostic details, including server field errors. Each kept row refusal carries command/target metadata in the same existing scoped marker as its refusal state; references survive reload and tab remount and disappear with that marker on success, pruning or scope reset. Several failed actions on one row are separately labelled by Action ID. The page-level “Action request references” region supplements these individual references.

The client preserves metadata through rate limiting and cancelled step-up. Mounted session/command/target generation guards check result/failure continuations, reload continuations and finally updates. Layout-effect cleanup disposes tokens synchronously at unmount; saved form details are bound to their generation. No rationale, submitted values, idempotency key, stack or diagnostic history store is introduced.

P01/P03/P04 data-load failures now name loading and service availability rather than sign-in failure. P01–P04 command fallbacks describe an unconfirmed result and preserve the existing action-specific next steps and field links.

# Consolidated lifecycle

The shared `Refusals<T>` store preserves original primitive refusal values and deep-copies diagnostic entries on write and read. All five modules use it; no duplicate renderer or unused parallel store remains. A fresh attempt clears only the exact previous command/target marker. P02 grouped rows select the canonical resume or redelivery request target for withdrawal actions.

Monotonic model scope generations invalidate old reads even across workspace A → B → A. Read cleanup and command guards dispose synchronously. Terminal session/access feedback and ordinary notices keep their associated response references; a new attempt clears earlier notice references. Actual response codes and request tokens are bounded to 128 characters. Local cancellation retains the preceding challenge reference with an explicit cancellation label.

HYB-327 strict response validation and field bindings remain present through a history-preserving merge. Both published HYB-330 histories are retained. Current checks and reviewer provenance are in [integration verification](integration.md); the two earlier verification records are distinct candidate evidence.

# Contracts and impact

The Go error producer, authentication, authorization, command bodies, retry eligibility, idempotency and financial semantics are unchanged. The client adds an optional response request reference to AdminError. The field binding map, register and terminology glossary are updated in this ticket. Vite's existing source fingerprint includes the changed source dependencies; no owner FIX or previous acceptance is carried forward.

The browser QA runner is portable through explicit URL, activation, Playwright module, Chromium and output arguments. Its only filesystem input is a synthetic activation file; no real accounts, external dispatch or deployment is used. Raw logs and screenshots remain local, outside Git.
