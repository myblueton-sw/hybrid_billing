---
type: ImplementationPlan
title: Live M04 service-account screen in the console service mode, bound to the HB-268 API
description: Phase 3h of the pre-launch plan (HB-257). In service mode, M04 renders:
  - an accounts view (list, create, suspend, reinstate and retire with reasons);
  - a credentials view (per-account metadata, issue with a one-time secret reveal, and revoke);
  - usage and transfer views marked as not available in pre-launch.
status: stable
---

# Scope and ownership

[HB-269](https://linear.app/hybrid-billing/issue/HB-269) binds M04 to the [HB-268](../HB-268/plan.md) API. Base: main `f54e6b5`.

**Ownership**
- Root (Claude Code) owns the plan, the implementation, the tests and the browser QA.
- The `ui-designer` role (AI, read-only) wrote the visual specification and compares the final renders.
- A separate read-only reviewer reviews the design and the change, with a security focus because of the secret reveal.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/admin.ts` and its test.
- `apps/console/src/modules/identity/pages/service-accounts/{Live.tsx,live-copy.ts,live-model.ts,live-model.test.ts}`: new.
- `apps/console/src/modules/identity/pages/users/live.css`: M04-scoped rules only.
- `apps/console/src/app/screen-groups/identity.tsx`.
- `apps/console/package.json`: the test list.
- `docs/features/HB-269/*`.

# Design

The layout follows M01–M03:
- the full viewport width;
- a list with a side panel at 1280px and wider, and stepped below that with "Back";
- Reason, FailureBox and Summary;
- the unavailable views;
- no decorative coloured borders: only neutral row rules and the existing neutral panel divider.

## Views

**accounts**
- The table of non-retired service accounts shows the ID (mono), the owner (with "Created by"), the status (with a credential summary) and a Manage button.
- Retired accounts are in a disclosure.
- The panel defaults to the create form:
  - the name has a fixed `svc-` prefix and a live ID preview;
  - the owner is chosen from the `owners` select;
  - Reason.
- Manage opens a detail panel with the facts, plus Suspend and Retire (when active) or Reinstate and Retire (when suspended), and a "Manage credentials" link.
- Each state change is a step with an impact text and a Reason. Reinstate warns to revoke exposed credentials first.

**credentials**
- A service-account select. It is preselected from accounts; otherwise it is the first active account.
- A persistent pre-launch note: credentials cannot authenticate yet, and use is not recorded.
- The select lists every account, retired and suspended ones included, so that credentials left active by a partial retire stay revocable. On a retired account those rows read "Revoke required", not "On hold".
- States come from the API's `now`, not the browser clock.
- The table of active credentials shows the ID, the validity, the status and a Revoke button. Ended credentials are in a disclosure.
- Status labels:
  - `usable` false on an active credential is shown as "On hold", not "Active";
  - expired and revoked keep their own labels.
- The panel defaults to the issue form: days 1 to `limits.credential_days` (default 30), the live end date, and Reason. It is replaced by a blocking note when the account is suspended, retired or at the credential limit.

**usage and transfer:** unavailable views with "Go to Service accounts". No API call is made.

## The one-time secret

1. `service.issue_credential` returns `secret_ref`. The panel shows the Reveal step, which fetches `/identity/links/{ref}` exactly once.
2. A new client method `secret(ref)` validates the response:
   - the secret is `hbsk_` followed by 43 base64url characters;
   - the credential is `cred-` followed by 16 hex characters.
   Anything else is treated as malformed (`invalid`) and shown as "can no longer be shown", without a retry.
3. The secret is shown in a read-only, labelled, `dir=ltr` textarea next to the credential ID and the end date.
   - It has no select-on-focus.
   - It sets `autocomplete`, `autocorrect`, `autocapitalize` and `spellcheck` off, and `translate="no"`.
   - It is described by the "shown once" text.
   - The fetch is automatic on entry, guarded by an `asked` ref against a double fetch. The 10-minute pickup window therefore starts and ends while the administrator is on the step.
   - The returned credential must equal the issue outcome's `credential`; a malformed or mismatched response is treated as "can no longer be shown", not retried.
4. **No Copy button.**
   - The planning record `docs/ui/screens/M04/design.json` says the credential detail shows only the reference, version and expiry, and forbids exposing the secret through the clipboard or logs ("자격 상세"). The record has no one-time reveal step; HB-268 introduced one.
   - Showing the raw secret once is still needed for a credential to be usable at all. A manual copy by the administrator would still go through the clipboard.
   - The owner decides whether a one-time raw reveal is allowed, and whether a manual copy or a Copy button is acceptable. Until then this screen shows the value once, with no Copy button and no select-on-focus, and does not claim the secret never reaches the clipboard.
5. Leaving the Reveal requires either:
   - ticking "I have stored this secret", then selecting Done; or
   - "I didn't store it: revoke", which opens the revoke step for that credential with a prefilled, editable reason.
6. **Secret lifetime**
   - The secret lives only in the Reveal component's state.
   - It is dropped from state and the DOM on Done, on revoke, on unmount and on `pagehide`, so a back-forward cache restore cannot bring it back. JavaScript strings cannot be wiped; they are only released.
   - It never appears in the URL, in storage, in a notice, in a live region, in an `aria-label` or `title`, or in the console.
   - While it is unacknowledged:
     - a `beforeunload` prompt is registered;
     - each credential ID is marked "unstored" as soon as its issue succeeds (a set, so several can be pending), and a mark is cleared only by an acknowledged Done, a successful revoke, or a deliberate Dismiss; the marks are kept per signed-in user in page memory, so they survive switching views but not a reload; Done is offered only while the secret is shown, and the other states offer Close, which keeps the mark;
     - `pagehide` also moves the step to "can no longer be shown";
     - while the mark is set, the credentials view shows a reminder with Revoke and Dismiss.
7. **Retrieval failures**
   - If the pickup has already been consumed or has expired (`denied`), the Reveal shows "The secret can no longer be shown" with a Revoke button.
   - An `unavailable` failure offers one retry.

## Failures

Failure copy is keyed by command and failure:
- create:
  - `invalid` on `name` or `owner` (or `days` on issue) becomes a field error;
  - `refused` means the cap is reached.
- issue: `refused` is suspended or at the limit, chosen from the reloaded page; `stage` means retired.
- state steps: `stage` means the state changed.
- revoke: `stage` means already revoked.
- `denied` means the target no longer exists.
- `conflict` and `stale` mean the list changed.
- `partial` on retire means retired, but some credentials remain; revoke them on the credentials tab.
- `busy` and `unavailable` mean not responding. On issue, the message adds "check the list before retrying".
- **Issue retry keeps the attempt.** While an attempt is held, the issue form stays available with "Try again" even if the reloaded account shows the limit or a suspension, so the held key can still be sent. Any definitive failure (`conflict`, `stale`, `refused`, `stage`, `denied`, `invalid`) releases the hold; `limited` and `unauthenticated` during a held retry keep it. Submitting an issue pins the selected account so a reload cannot switch the form away. After `busy`, `unavailable` or `partial` on issue, the form keeps the same idempotency key and the same expected revision across the reload. A retry therefore replays the stored outcome, with the same `secret_ref`, instead of issuing a second credential. Editing an input starts a new attempt.
- `denied` covers both a target that no longer exists and a lost permission or ended session.
- `limited`, `unauthenticated` and an `invalid` failure without a field use the shared M01 failure text.

A settled failure reloads the page, the same as M01–M03.

## External references (checked on 2026-11-16)

- **Microsoft Entra, "Add and manage app credentials"** (learn.microsoft.com/en-us/entra/identity-platform/how-to-add-credentials).
  - **Observed:**
    - a "New client secret" step with a description and an expiry (a custom lifetime up to 24 months);
    - the value is recorded after Add, and the page says it is never displayed again after you leave the page;
    - the secret ID is separate from the value.
  - **Adopted:** an issue form in a side panel; the warning that the secret is not shown again after leaving; showing the credential ID next to the value.
  - **Not adopted:**
    - the description field (no API);
    - lifetimes over 30 days;
    - showing the value inline in the list table.
- **GitHub, "Managing your personal access tokens"** (docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens).
  - **Observed:**
    - expiration is chosen when the token is generated, and the pre-fill defaults to 30 days;
    - classic tokens offer an optional copy-to-clipboard button;
    - the fine-grained steps end at generation.
  - **Adopted:** a 30-day default and expiry chosen in days with the end date shown.
  - **Not adopted:**
    - the copy-to-clipboard button (the M04 design record forbids it);
    - regeneration;
    - scope pickers;
    - last-used columns (no data).

# Review history

- **Design round 3 (PASS):** N7 (keep the hold on `limited` and `unauthenticated`) and N8 (Dismiss listed) fixed; the account is pinned at submit.
- **Design round 2 (FAIL):** N1 (the held retry blocked by the limit) and N2 (Done clearing the mark without a reveal) fixed; N3 plan text, N4 renew on definitive failures, N5 a set of marks, N6 `pagehide` to gone fixed. Residual: a remount after an unknown outcome loses the held attempt.
- **Design round 1 (FAIL):**
  - M1, issue retry: fixed with a held attempt and revision.
  - M2, clipboard framing: rewritten as an owner decision; no select-on-focus.
  - M3, the recovery path after a partial retire: fixed; retired accounts are selectable, with "Revoke required".
  - L1–L8: addressed above.

# Tests and verification

- **Unit (model):**
  - name validation, mirroring the API;
  - day bounds;
  - the credential state labels, including `usable`;
  - the split into retired and ended;
  - the live count;
  - the secret format check in the client.
- **Browser QA against a live `hb-api`:**
  - create (valid, invalid and taken names), suspend, reinstate, retire;
  - issue with the Reveal, then Done refused without the acknowledgement, then Done;
  - "I didn't store it: revoke";
  - the unstored reminder after leaving;
  - the secret absent from the DOM after Done;
  - revoke;
  - the usage and transfer views;
  - the member denied view;
  - EN, KO and AR at 1440 and 390;
  - focus returns, no horizontal scroll, computed borders neutral.
- Reruns of the M01, M02 and M03 QA, because the CSS is shared.

# Residuals for the owner

- **One-time reveal and clipboard:** the planning record shows only the reference, version and expiry, and bans clipboard exposure. Whether a one-time raw reveal, a manual copy and a Copy button are allowed is the owner's decision. Until then the screen shows the value once, with no Copy button.
- The KO tab label stays "자격 메타데이터", while the body says "자격 증명". The terminology needs review.
- The KO and AR copy is an unreviewed AI draft.
- The unstored reminders are lost if the administrator leaves the M04 screen, and a held issue attempt is lost if the form remounts (account switch or view change) after an unknown outcome.
- Credentials cannot authenticate in pre-launch, and there is no use history and no owner transfer.
- Screen acceptance (FIX) is the owner's alone.
