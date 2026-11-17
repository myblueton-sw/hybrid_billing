---
type: ImplementationPlan
title: Live P01 installation screen in the console service mode, bound to the HB-282 installation API
description: Phase 5b of the pre-launch plan (HB-257, HB-262 scope). In service mode, P01 shows the site, observation, install, upgrade and rollback previews and releases on the plan view, installer proposals with dry-run blockers and the approve, withdraw and propose-plan commands, and allowlisted destinations with guided probe reports on the diagnostics view. Diagnostic requests, support bundles and support access are stated as not connected. Approval installs nothing, and the screen says so.
status: final
---

# Scope and ownership

[HB-283](https://linear.app/hybrid-billing/issue/HB-283) binds P01 to the [HB-282](../HB-282/plan.md) API. Base: main `748d0c2`.

**Ownership**
- Root (Claude Code) owns the plan, integration, browser QA and verification. Implementation may be delegated to one writer in this worktree.
- The `ui-designer` role (AI, read-only) writes the visual specification and compares the final renders.
- An independent read-only reviewer reviews the design and the change. Focus: approval and separation of duties shown truthfully, nothing implies a host change, unknown values never shown as zero, commands only where the read allows them.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/installation.ts` (types and hardened parsers) and its test `installation.test.ts` (including the command envelope); `apps/console/src/service/admin.ts`: `installation()` and the three command IDs.
- `apps/console/src/modules/administration/pages/installation/`: new `Live.tsx`, `LivePlan.tsx`, `LiveProposals.tsx`, `live-copy.ts` (split into `live-copy.ts` and `live-codes.ts` if it nears 1,500 lines), `live-model.ts`, `live-model.test.ts`, `live-format.tsx`, `live.css`. The existing preview `Screen.tsx` stays for the unconnected mode.
- `apps/console/src/app/screen-groups/operations.tsx`: the live branch for P01 when the service context is present.
- `apps/console/package.json`: the test list.
- `docs/features/HB-283/*`.

# What the API offers (merged HB-282 code)

`GET /api/v1/installation` returns either `{revision, now, state: "not_configured"}` or `state: "configured"` with:
- `site` (id, platform, probe, config_overrides — always empty in this build) and `observation` (state `fresh`, `stale`, `missing` or `unknown`; observed and expiry times);
- `installed` (state `not_installed` or the installed version, state, owned services and volumes, history, backups);
- `releases` (version, key version, upgrade-from, platforms, services with ports and volumes, volumes with the data flag, artifacts with kind, and an optional mark);
- `preview`, `upgrade`, `rollback`: each `{kind, version, state, reason, hash, services, ports, volumes, data_effects, permissions, leftovers, conflicts}` where `state` is `available`, `unavailable` (with `reason` such as `not_observed`, `observation_stale` or a module refusal) or `not_applicable` (empty reason); item rows are `{kind, target, action, conflict}`;
- `proposals` (newest first, at most 50): number, target, kind (`key`, `release`, `site`, `destination`, `plan`, …), state (`pending`, `applied`, `withdrawn`), created, proposer and approver (ID and login), rationale, closed by and closing rationale, a summary (key version, release version, site, destination host, port and ranges, plan version, hash and admin), `can_approve` with `approve_blocked` (`own_proposal`, `not_pending`, `plan_changed`, `observation_stale`, `not_observed`, `conflicts` or another refusal), `can_withdraw` with `withdraw_blocked` (`not_proposer`, `not_pending`), and `current_hash` for plan acceptances;
- `destinations` (id, host, port, ranges) and `reports` (destination, kind, failure, address, via, cause, steps);
- `runs`: `none`.

Commands (reason required, `expected_revision` = the read's `revision`): `installation.approve` and `installation.withdraw` with target `installation-proposal-<n>` and no fields; `installation.propose_plan` with target `installation` and fields `version`, `hash`. Success returns the proposal number and state. Refusals: `prerequisite_failed` (self-approval, not the proposer, limit), `invalid_stage` (plan changed, stale or missing observation, conflicts, not pending), `invalid_request` with `version` or `hash`, `permission_denied` (also for an unknown proposal target), `internal_error`, `dependency_unavailable`, `revision_conflict`.

# Design

Follows the O01–O03 live patterns: full viewport width, neutral rules and tags, EN, KO and AR, shared failure box, reason field (UTF-8 bytes, 256), error summary, `useWide` at 1280px, dates through `Intl` in UTC with first-strong isolation, IDs and hashes in `<bdi dir=ltr>` monospace.

## Views (the registered P01 tab keys)

1. **Plan** (`plan`):
   - A scope bar: site ID and platform, data revision, now (UTC), Reload.
   - A statement under it: "Approving here records a decision. Nothing is installed, downloaded or changed on a host in this build."
   - Observation: the state in words with the observed time and the expiry ("Observed 16 Nov 2026, 09:00 UTC; valid until 17 Nov 2026, 09:00 UTC"). Stale and missing say what that blocks (plan approvals and new plan proposals).
   - Installed: "Not installed" or the installed version and state with owned services and volumes.
   - Install plan (the `preview`): version and hash, then sections for services, ports, volumes (with "contains data" in words), data effects, permissions to grant, leftovers, and conflicts (a conflict blocks approval and says so). Unavailable previews show the reason in words. Not-applicable copy is per kind: install "No release to preview", upgrade and rollback "Nothing is installed". The view states which version is previewed and why ("Newest pending plan proposal" or "Newest release not installed") and shows the latest applied plan acceptance ("Plan 1.0.0 accepted by X on D. Not installed: runs are not connected in this build.").
   - "Propose this plan": offered only when the preview `state` is `available`, it has no conflicts, its hash is 64 lowercase hex characters, no plan acceptance for that version is already applied, and the reader has no pending plan proposal with the same version and hash. If the reader's own pending plan for that version has a different hash: "The install plan changed after you proposed this. Withdraw it and propose the current plan." It sends `version` and the preview `hash`, with a confirm step restating both and "Another administrator must approve it. Approval installs nothing." A duplicate (the returned number was already listed) announces "This plan was already proposed as #N".
   - Releases: a table (version, key version, upgrade from, platforms, services, volumes with data flag, mark in words or "None").
2. **Diagnostics** (`diagnostics`):
   - Destinations table (ID, host, port, allowed ranges).
   - Reports table, following the spec's column order: check (kind), target (destination, and the address or "Not resolved"), status (failure in words; `ok` reads "Passed – no action"), required or warning ("Required" for failures, "None" for `ok`), route (`via` destination or "Direct"), next action (cause and steps as a numbered list). Every failure, cause and step code from the module's diagnostics table has EN, KO and AR copy with an "Unknown (code)" fallback, and no copy suggests turning off TLS verification.
   - "Diagnostic requests need the site's probe and are not connected in this build."
3. **Support** (`support`) and **Access** (`access`): "Not connected in this build" with what each will hold (support bundle preview with exclusions and no automatic upload; limited support access with actor, scope and expiry).

## Proposals (shown on the plan view, below the install plan)

- A list of proposals, pending first then closed, newest first: number, kind in words, summary in words (for example "Allow destination export.example.invalid:443 (198.51.100.0/24)" or "Accept install plan 1.0.0, hash …"), state, proposer, created, and for closed ones the approver or withdrawer with their reason.
- Approve is shown only when `can_approve` is true; Withdraw only when `can_withdraw` is true. Otherwise the blocker reads as a sentence: own proposal ("You proposed this. Another administrator must approve it."), not pending, plan changed ("The install plan changed after this was proposed. Current hash …"), observation stale or missing, conflicts, or an unknown reason ("Approval is not available for this proposal now"). A plan acceptance shows the proposed hash next to `current_hash`.
- Confirm steps restate the proposal and say what approval does: "Approval records the decision and applies it to the installer's registry. Nothing is installed."
- The proposal list is hidden when empty with "No installer proposals".

## Attempts, refusals and states

- Attempts (request ID and idempotency key) are kept per command and target in a module-level store scoped by workspace, user and session, and reused only to resend identical values after an unknown outcome (network failure or `busy`) even if a reload moved the revision; any edit or definite answer discards them; a pending attempt shows "The last request may have been applied. Check the proposals, or send it again."
- Reactions: `invalid` with `version` or `hash` → "The plan changed; review the current plan." and reload; `invalid` without a field → the reason field; `stage`/`refused` → reload, then the row's current blocker sentence, or, when the reloaded blocker is empty (destination, key and release approvals have no dry run; `propose_plan` has no row), a kept per-target sentence ("The installer refused this approval. Nothing changed. Check the destinations, or ask the proposer to withdraw it."; for propose: "The installer refused this plan proposal. Review the current plan; you may also have too many pending proposals."); `conflict` → reload, keep the reason, confirm again; `denied` on a command → reload, and if the read succeeds "This proposal is no longer available", otherwise the unavailable block; `unauthenticated` → sign in; `limited` → the wait; `busy`/`unavailable` → retry with the same attempt; `stale` → renew; `partial` → the operation text.
- A resend reuses the frozen `expected_revision`, target and values and is offered even if the reloaded row no longer allows the action. A repeated approval by the same approver succeeds and moves the revision.
- States: first load (no numbers), reload (`aria-busy`), `not_configured` ("No installation site is configured in this workspace"), denied, unauthenticated, failure with Try again. Missing numbers and unknown enum values read "Unknown (code)", never 0 or blank.

## External references

The ui-designer recalled these from memory; nobody opened them, so they are unverified:
- **Terraform plan then apply** (https://developer.hashicorp.com/terraform/cli/commands/plan). Adopted: full preview before change, a content-identified plan, review separate from execution. Rejected: coloured +/~/− glyphs.
- **GitHub deployment reviews** (https://docs.github.com/en/actions/managing-workflow-runs-and-deployments/managing-deployments/reviewing-deployments). Adopted: approval by another person, the self-approval block stated, a decision comment. Rejected: a modal multi-select.
- **AWS Systems Manager Change Manager** (https://docs.aws.amazon.com/systems-manager/latest/userguide/change-manager.html). Adopted: state, requester and approver; closed requests keep who and why. Rejected: calendars, templates and runbook execution.
- **Google Cloud Connectivity Tests** (https://cloud.google.com/network-intelligence-center/docs/connectivity-tests/concepts/overview). Adopted: one result per destination, status in words, route, cause with next steps. Rejected: the trace diagram and re-running from the UI.
- **GOV.UK check answers** (https://design-system.service.gov.uk/patterns/check-answers/). Adopted: the confirm step restates every value; error summary links. Rejected: per-row Change links.

The visual specification (ui-designer, AI) defines the `p01-` classes, the wrapper, the status band, the conflicts band and effects grid, the proposal rows and the tables; root keeps it with the ticket's working notes.

## Out of scope (owner residuals)

- Runs, install execution, bootstrap codes, key and release management, probe observations and diagnostic requests, support bundles and support access.
- Owner FIX of the screen; KO and AR terminology review.

# Design review round 1 rules (applied)

- **Host and remount (M6).** The live branch returns from `operations.tsx` before the preview wrapper, with `work-domain remaining-security`, `data-source-state="live"` and a locale-only key, like `work.tsx`; hooks stay unconditional and the live branch does not report `source: unconnected`. URL state is `view=` and an optional `proposal=<n>` (validated against the read). Results use a status message (focused after success) and a polite live region; cancel returns focus to the opener; error summary links move focus to fields. A request sequence drops stale responses. The attempt and draft store is scoped by workspace, user and session and cleared on sign-out.
- **Mobile (M6).** Below 768px tables become stacked labelled rows and grids one column; hashes wrap with `overflow-wrap:anywhere` inside `<bdi dir=ltr>`; no horizontal scroll at 390px.
- **Observation (L2).** A year-1 or empty time reads "Unknown"; `unknown` state has its own sentence; `missing` reads "No current observation is available"; comparisons use the response's `now`; the screen says a stale observation cannot recover until the service restarts in this build.
- **Proposal rows (L3).** Summaries for every kind (`key`, `release`, `site`, `probe`, `config`, `destination`, `plan`, `upgrade`, `rollback`) with an "Unknown (code)" fallback; the port only for destinations; logins fall back to IDs; the proposer's rationale on pending rows and in the approve confirm; `current_hash` only when non-empty; `not_proposer` hides Withdraw without a sentence; commands use the row's `target` when it matches `^installation-proposal-[1-9][0-9]*$`, otherwise no command is offered; a missing number fails closed. "Showing the newest 50" when 50 rows return.
- **Vocabularies (L6).** EN, KO and AR copy with fallbacks for permissions, item actions, conflicts, data effects, installed states, artifact kinds and approve-blocker reasons.
- **Marks (L1).** A release mark reads "Marked by {login}: {reason}" and "confirmed by {login}" when present; mark commands are not exposed.



# Verification plan

- Model tests: blocker sentences and unknown fallbacks, proposal summaries per kind, preview availability and "propose" gating, observation wording, attempt store, failure reactions, UTF-8 bytes, null and unknown display.
- Model tests for the round-1 rules: the kept refusal sentence for approve and propose when the reloaded blocker is empty, and its clearing (on the next definite answer for that target, or when the reloaded row shows a blocker or a closed state); command `denied` → reload → "This proposal is no longer available" ("The plan can no longer be proposed here" for `propose_plan`); the own stale-plan sentence and Propose staying offered when the hash differs; Propose hidden when that version's acceptance is applied; the duplicate announcement; report `ok`, "Not resolved" and "Direct"; a year-1 observation time; a malformed target offering no command.
- Service tests: hardened parsers (missing arrays, wrong types, unknown enums), `not_configured`, the command envelope and resend identity.
- `npx tsc --noEmit`, `npm test`, `npm run build`.
- Playwright QA against the real API (`--seed sample`): plan and diagnostics views in EN, KO and AR at 1440 and 390; as the first administrator the destination row reads "You proposed this" and plan 1.0.0 can be approved; as the second administrator the destination can be approved and the plan row reads "You proposed this"; after approving plan 1.0.0 the accepted-plan line appears and the previewed version and reason are stated; withdraw by the proposer; propose plan and the duplicate case; no horizontal scroll; O03, O02 and M01 regressions.
- ui-designer render comparison; independent change review; `verification.md`.
