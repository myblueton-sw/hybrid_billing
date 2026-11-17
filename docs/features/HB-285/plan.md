---
type: ImplementationPlan
title: Live P02 integrations screen in the console service mode, bound to the HB-284 integrations API
description: Phase 5c of the pre-launch plan (HB-257 scope). In service mode, P02 shows the console API client with its grants and the service users managed on M04, the plugin registry with publishers, versions, the installation and plugin proposals, webhook destinations and subscriptions with their keys and history, and event types, the outbox and deliveries with dead letters. It offers the fifteen HB-284 commands only where the read allows them. Nothing is dispatched, sent, signed or issued, and the screen says so.
status: final
---

# Scope and ownership

[HB-285](https://linear.app/hybrid-billing/issue/HB-285) binds P02 to the [HB-284](../HB-284/plan.md) API ([verification](../HB-284/verification.md)). Base: main `e47fd6f`. It follows the merged [HB-283 plan](../HB-283/plan.md), its [verification record](../HB-283/verification.md) and the code that plan produced (`apps/console/src/modules/administration/pages/installation/`, `apps/console/src/service/installation.ts`, the P01 branch in `apps/console/src/app/screen-groups/operations.tsx`). The P02 design is `docs/ui/screens/P02/spec.md`.

**Ownership**
- Root (Claude Code) owns the plan, integration, browser QA and verification. Implementation may be delegated to one writer in this worktree.
- The `ui-designer` role (AI, read-only) writes the visual specification and the external references, and compares the final renders.
- An independent read-only reviewer reviews the design and the change. Focus:
  - nothing implies a dispatch, a send, a signature, a token or a running plugin;
  - two-person rules shown truthfully;
  - unknown values never shown as zero;
  - commands only where the read allows them, and fail-closed parsing.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/integrations.ts` (types and hardened parsers) and `integrations.test.ts` (including the command envelope and resend identity).
- `apps/console/src/service/admin.ts`: `integrations()` and the fifteen command IDs.
- `apps/console/src/modules/administration/pages/integrations/`:
  - new `Live.tsx` (host), `LiveClients.tsx`, `LiveModules.tsx`, `LiveChannels.tsx`, `LiveEvents.tsx` and `LiveCommand.tsx` (the shared command form);
  - `live-model.ts`, `live-model.test.ts`, `live-copy.ts`, `live-codes.ts` (vocabularies), `live-format.tsx` and `live.css`.
  - Any file nearing 1,500 lines is split by responsibility, for example `live-codes.ts` into a plugin part and a webhook part.
  - The existing preview `Screen.tsx` stays for the unconnected mode.
- `apps/console/src/app/screen-groups/operations.tsx`: the live branch for P02.
- `apps/console/package.json`: the test list.
- `docs/features/HB-285/*`.
- No API, module, P01, M03 or M04 file changes. P02 may import read-only helpers from other screens, for example `actionLabel` and `kindLabel` from `identity/pages/access/live-copy.ts` and `identity/pages/users/live.css`. It keeps its own attempt store and command form, so P01 and P02 attempts never mix.

# What the API offers (merged HB-284 code)

## Read: `GET /api/v1/integrations`

Route: `boundary.go:37`. The read is admin only, through `admitted` (`integrations.go:705`).

**Whole-page states**
- **Another workspace:** `{revision, now, state: "not_configured"}` and nothing else (`integrations.go:710-712`).
- **Any module read error:** `dependency_unavailable` for the whole read (`integrations.go:714-717`). The console maps it to `busy`.
- **Otherwise:** `{revision, now, state: "configured", clients, modules, channels, events}` (`integrations.go:307-315`, `689-699`).
  - Every section has its own `state`, `configured` or `not_configured` (`integrations.go:317-322`).
  - `clients` is always `configured` (`integrations.go:340`).
  - `modules` is `configured` when a publisher, a plugin or the plugin workspace exists (`integrations.go:480`).
  - `channels` is `configured` when a subscription is stored; `events` when an event type is approved (`integrations.go:662-664`).

**An action's availability** is `{allowed, blocked}`, where `allowed` is true exactly when `blocked` is empty (`integrations.go:29-36`).

**`clients`** (`integrations.go:38-69`, `331-358`)
- `clients`: one row, `{id: "console", kind: "first_party_console", state: "active", grants}` (`integrations.go:332`).
  - Grants: `{id, action, kind, scope: {customer, legal_entity, contract, period}, not_before, not_after, state}`.
  - `AdministerInstallation` grants are hidden (`integrations.go:334`).
  - Scope selectors read `all`, or an ID (`access.go:88-93`). `period` is `any` or `<from>-<to>` (`access.go:95-101`).
  - Grant `state` is `active`, `pending`, `expired` or `revoked` (`access.go:103-113`).
- `service_users`: `{id, state, owner, owner_login, usable_credentials}`.
  - `state` is `active`, `suspended` or `retired` (`services.go:55-63`).
  - `usable_credentials` is a count (`integrations.go:348-355`).
- `manage_on: "M04"` and `oauth_clients: "not_connected"` (`integrations.go:340`).

**`modules`** (`integrations.go:71-197`, `360-511`)
- `workspace`: `{id, runtime, verification}` or `null` when no plugin workspace exists (`integrations.go:483-487`).
- `publishers`: `{id, name, revocation, keys}`.
  - Each key is `{version, not_before, not_after, retired, compromise}`.
  - Marks are `null` or `{by, by_login, reason, confirmed_by, confirmed_by_login}` (`installation.go:123-129`). There is no key material.
- `plugins`: one row per registered version (`integrations.go:113-130`, `377-397`):
  - `{plugin, version, publisher, key_version, digest, core_min, core_max, capabilities, egress, endpoints, timeout_seconds, memory_mib, concurrency, storage, screens, vulnerable}`;
  - each capability is one string `kind:target` (`integrations.go:386-388`). The kinds are `read_events`, `call_command` and `serve_screen` (`modules/plugins/types.go:48-50`);
  - `storage` is `{namespace, owner, retention_days, deletion, migration}`. `deletion` is `delete_on_removal` or `retain_for_evidence`; `migration` is `none`, `additive` or `copy_forward` (`types.go:57-62`);
  - `screens` are `{id, component, label, fields: [{name, label, source}]}`. `component` is `table`, `form`, `detail` or `chart` (`types.go:52-55`).
- `installations`: `{plugin, version, state, reason, revision, endpoints, disposal, can_stop, stop_blocked, can_disable, disable_blocked, tokens, runtime}` (`integrations.go:139-153`, `405-423`):
  - `state` is `installed`, `enabled`, `stopped` or `removed` (`types.go:43-46`);
  - endpoints are `{id, fingerprint, approver, approver_login}`;
  - `disposal` is a storage row only when `removed` (`integrations.go:411-414`);
  - `stop_blocked` is `not_enabled` unless `enabled`. `disable_blocked` is `not_allowed_state` unless `enabled` or `stopped` (`integrations.go:415-421`);
  - `tokens: "not_issued"` and `runtime: "not_connected"` (`integrations.go:407`).
- `proposals`: newest first, at most 50 (`integrations.go:506`, `installseed.go:19`). Each row (`integrations.go:169-188`, `467-476`) has:
  - `{number, target: "plugin-proposal-<n>", kind, state, created_at, proposer, proposer_login, approver, approver_login, rationale, closed_by, closed_by_login, closing_rationale, summary, can_approve, approve_blocked, can_withdraw, withdraw_blocked}`;
  - `kind` is one of `publisher`, `key`, `version`, `workspace`, `runtime`, `install`, `upgrade`, `rollback`, `enable`, `remove` and `endpoint` (`types.go:23-33`). `state` is `pending`, `applied` or `withdrawn` (`types.go:39-41`);
  - `summary`: `{publisher, publisher_name, key_version, key_not_before, key_not_after, plugin, version, workspace, runtime, endpoint, fingerprint}`. Key validity is filled only for `key` (`integrations.go:155-167`, `457-465`).
- **Approve blockers** (`integrations.go:442-455`, `425-440`):
  - `not_pending` for closed rows; `own_proposal` for the reader's own;
  - otherwise the registry's dry run (`Recheck`): any plugins refusal reason (`types.go:64-96`), or `not_found`, `conflict` or `limit`.
- **Withdraw blockers:** `not_pending` or `not_proposer`.

**`channels`** (`integrations.go:199-258`, `513-601`, `660-687`)
- `destinations`: `{id, state, source: "sample_allowlist"}` (`integrations.go:665-667`).
  - In this build: `sample-erp-webhook` active, `sample-legacy-webhook` inactive, `sample-payments-webhook` active (`integrationdeps.go:34-38`).
  - Destination states are `active`, `inactive` and `revoked` (`modules/webhooks/types.go:48-50`).
- `subscribers`: `["sample-erp-connector"]` (`integrations.go:662`).
- `subscriptions`:
  - `{id, target: "webhook-subscription:<id>", subscriber, destination, types, state, reason, cursor, lag, gaps: [{after, to}], created_at, rationale, proposer(+_login), approver(+_login), resumer(+_login), history: [{action, actor, actor_login, rationale}], keys, signer: "not_connected", actions}` (`integrations.go:228-251`, `576-601`).
  - `state` is `proposed`, `active`, `paused`, `suspended` or `revoked` (`webhooks/types.go:52-56`).
  - `lag` is `head − cursor`, or 0 when the cursor is at or past the head (`integrations.go:582-584`).
  - `keys` lists active keys only: `{version, not_before, not_after, state, proposer(+_login), approver(+_login)}` (`integrations.go:591-599`).
- Subscription actions (`integrations.go:536-568`); the keys are `approve`, `pause`, `request_resume`, `approve_resume` and `revoke`:
  - `approve`: `not_pending`, `own_proposal`, or the admission check `destination_inactive`, `subscribe_refused` or `dependency_unavailable` (`integrations.go:513-530`);
  - `pause`: `not_active`;
  - `request_resume`: `not_pending` (not paused or suspended) or `already_requested`;
  - `approve_resume`: `not_pending` (no requester), `own_request` or the admission check;
  - `revoke`: `not_pending` when already revoked.

**`events`** (`integrations.go:260-305`, `603-687`)
- `types`: `{name, schema_version, fields, state: "active", proposer(+_login), approver(+_login)}` (`integrations.go:668-671`).
- `outbox`: `{floor, head}` (`integrations.go:672-673`).
- `dispatcher: "not_connected"` (`integrations.go:664`).
- `deliveries`: up to 50 per stored subscription, merged and sorted by sequence (`integrations.go:26`, `641`, `685`). Each row (`integrations.go:277-292`) has:
  - `{id, target: "webhook-letter:<id>", subscription, sequence, source, event_id, kind, state, reason, attempts, outcome_unknown, redeliver_request, dismiss_request, actions}`;
  - `kind`: `open`, `redelivery`, `dead` or `cancelled`. `state` (job): `queued`, `running`, `waiting`, `succeeded`, `failed` or `cancelled` (`webhooks/types.go:64-74`, `deliveries.go:403-419`);
  - `reason` is empty, or one of `rejected`, `attempts_exhausted`, `max_age`, `subscription_revoked`, `permission_revoked`, `destination_inactive`, `destination_revoked`, `dependency_unavailable` (`deliveries.go:62-108`, `255-292`, `334-360`);
  - `outcome_unknown` is true for a cancelled delivery that had been confirmed (`deliveries.go:423`);
  - requests are `null` or `{by, by_login, rationale}` (`integrations.go:271-275`).
- Delivery actions (`integrations.go:610-636`):
  - rows that are not dead letters: every action `not_dead`;
  - `request_redeliver` and `request_dismiss`: `already_requested`;
  - `approve_redeliver`: `not_requested`, `own_request` or `revoked` (revoked subscription);
  - `approve_dismiss`: `not_requested` or `own_request`.

No row carries a public key, proof, signature, token, body or header (`HB-284 plan.md:275`).

## Commands

All fifteen are `integrations.<kind>` (`integrationcmds.go:14-38`, `389-417`):
- Each takes a reason. The `expected_revision` is the read's `revision`.
- Each needs `AdministerInstallation` (`integrationcmds.go:394-396`). This is a recorded deviation from P02's separate permissions (`HB-284 plan.md:308-310`).
- The handler refuses a reason over 256 bytes with a fieldless `invalid_request` (`integrationcmds.go:360-361`, `directory.go:13`). The command shape allows up to 2,000 bytes (`server.go:114`), so the 256-byte limit is enforced only there and the screen must enforce it itself.

| Command | Target | Fields | Result `business_state` / `fields` | Command-specific refusals |
| --- | --- | --- | --- | --- |
| `plugin_approve` | `plugin-proposal-<n>` | — | proposal state (`applied`); `proposal`, `target` (`:140-154`) | non-pending → `invalid_stage` (`:147-148`); self-approval → `prerequisite_failed`; other dry-run refusals → `invalid_stage`; limit → `prerequisite_failed` |
| `plugin_withdraw` | `plugin-proposal-<n>` | — | `withdrawn`; `proposal`, `target` | not proposer → `prerequisite_failed`; not pending → `invalid_stage` |
| `plugin_propose` | `plugin:<plugin>` | `kind` (required enum `install`, `upgrade`, `rollback`, `enable`, `remove`), `version` (optional text) | `pending`; `proposal`, `target`. Same proposer and same change → the existing proposal (`HB-284 plan.md:359`) | `version` not registered for install/upgrade/rollback, or not empty for enable/remove → `invalid_request` [`version`] (`:173-174`); no plugin workspace → `invalid_stage` (`:175-176`); module refusal (`direction`, `not_allowed_state`, …) → `invalid_stage`; `limit` → `prerequisite_failed` |
| `plugin_stop` | `plugin:<plugin>` | — | `stopped`; `target` (`:206`) | not `enabled` → `invalid_stage` (`:191-192`) |
| `plugin_disable` | `plugin:<plugin>` | — | `installed`; `target` | `installed` → `invalid_stage` (`:191-192`); `removed` → `invalid_stage` (module refusal) |
| `subscription_propose` | `webhooks` | `destination` (text, required), `types` (text, required, comma-separated) | `proposed`; `subscription`, `target` (`:209-212`). The same proposer with the same content, still proposed → the existing subscription (`:274-278`) | destination not in the allowlist → `invalid_request` [`destination`] (`:263-265`); types not 1–8 distinct approved names → `invalid_request` [`types`] (`:214-224`, `266-268`); identical proposed or active content from another proposer, or already active → `invalid_stage` (`:274-275`); subscription limit → `prerequisite_failed` |
| `subscription_approve` | `webhook-subscription:<id>` | — | `active` | not proposed → `invalid_stage` (`:302`, `309-310`); self → `prerequisite_failed`; `destination_inactive`, `subscribe_refused` → `prerequisite_failed`; `dependency_unavailable` → `dependency_unavailable` (`:61-64`) |
| `subscription_pause` | same | — | `paused` | not active → `invalid_stage` |
| `subscription_request_resume` | same | — | the subscription state | requester already set → `invalid_stage` (`:304`); not paused or suspended → `invalid_stage` |
| `subscription_approve_resume` | same | — | `active` | self → `prerequisite_failed`; no requester → `invalid_stage`; admission as for approve |
| `subscription_revoke` | same | — | `revoked` | already revoked → `invalid_stage` (`:306`) |
| `letter_request_redeliver` | `webhook-letter:<delivery-id>` | — | `requested`; `delivery`, `target` (`:326-328`) | pending request of that kind → `invalid_stage` (`:331-333`); a letter already dropped from the store → `permission_denied` through `Resolve` (`:101-106`, `125`); `invalid_stage` for a letter that is no longer dead only in a race the store has not seen |
| `letter_approve_redeliver` | same | — | the new delivery's state (`queued`); `delivery`, `target`, `redelivery` (`:341-344`) | self → `prerequisite_failed`; no request or revoked subscription → `invalid_stage`; once approved, the letter is dropped (`:352-354`), so any later call → `permission_denied` |
| `letter_request_dismiss` | same | — | `requested` | as `letter_request_redeliver` |
| `letter_approve_dismiss` | same | — | `dismissed` (`:347`) | self → `prerequisite_failed`; no request → `invalid_stage`; once approved, dropped as for redelivery → `permission_denied` |

Line references in the table are to `integrationcmds.go`.

**Error mapping and fields**
- **Mapping** (`integrationcmds.go:41-69`):
  - `self_approval`, `not_proposer`, `destination_inactive`, `subscribe_refused` and `ErrLimit` → `prerequisite_failed`;
  - `dependency_unavailable` → `dependency_unavailable`;
  - not found, conflict, stale and every other refusal → `invalid_stage`;
  - `ai_not_allowed`, `not_human` and anything else → `internal_error`.
- **Unknown targets:** `Resolve` admits only targets in the server store; an unknown target is `permission_denied` (`integrationcmds.go:113-133`, `modules/commands/execute.go:330-331`). `webhooks` resolves only once an event type is approved (`integrationcmds.go:120-121`).
- **Codes that carry fields:**
  - Only `invalid_request` carries fields.
  - From a handler, only declared names: `kind` or `version` for `plugin_propose`, `destination` or `types` for `subscription_propose` (`execute.go:399-413`). In this build the handler sends only `version`, `destination` and `types`.
  - Shape validation may also name `rationale`, `target_id`, `expected_revision`, `idempotency_key`, `request_id` or `kind` (an enum outside the values) (`execute.go:53-107`, `282-284`).
  - Every other code is fieldless (`HB-284 plan.md:357`), so the read's blockers explain refusals.
- **Revision:** a successful command moves the revision (`integrationcmds.go:386`).
- **Console failure classes:** `invalid` (invalid_request), `refused` (prerequisite_failed), `stage` (invalid_stage), `denied` (permission_denied), `conflict` (revision_conflict), `busy` (dependency_unavailable), `limited`, `stale` (idempotency), `unauthenticated`. `internal_error` becomes `partial` only with an operation ID, otherwise `unavailable` (`admin.ts:289-314`).
- **Replay:** a resend with the same idempotency key and body returns the stored result (command journal).
- **Replay exception: targets that disappear.** `Resolve` and the permission decision run before the journal lookup (`execute.go:307-331` before `342-351`), and an unknown target is `permission_denied`.
  - A successful `letter_approve_redeliver` or `letter_approve_dismiss` removes the letter from the server store (`integrationcmds.go:352-354`), and `Resolve` then no longer finds it (`integrationcmds.go:101-106`, `125`). A resend of either after an unknown outcome is therefore answered `permission_denied`, not with the stored result.
  - The only other non-replayable case: a closed plugin proposal pruned from the store once more than `installKept` (200) entries are kept (`integrationseed.go:97-108`, `installseed.go:18`). Its approve or withdraw resend is also `permission_denied`.
- **Repeats are refused, not replayed.** Unlike P01, a fresh approval of a proposal or subscription that is no longer pending is refused (`integrationcmds.go:147-148`, `302`).

## Sample seed (who did what)

`--seed sample`; `integrationseed.go:298-432`. First administrator `sample-admin` = `admin@example.invalid`; second `sample-admin-2` = `admin2@example.invalid` (`seed.go:22-25`).

**Plugins**
- Applied proposals #1–#7 were proposed by the first administrator and approved by the second: publisher, key, versions 1.0.0 and 1.1.0, workspace, install 1.0.0, enable.
- Pending:
  - **#8** upgrade to 1.1.0, **proposed by the second administrator**. The first can approve it.
  - **#9** endpoint `api`, **proposed by the first administrator**. The second can approve it.
  - Source: `integrationseed.go:331-341`, `integrations_api_test.go:133-136`, `194-197`.
- Installation: `sample-ledger-export` 1.0.0, `enabled`, revision 2.

**Webhooks**
- `sample-erp-invoices`: proposed by the first administrator and approved by the second, active, cursor 3, lag 0, key `sample-webhook-key-1` (`integrationseed.go:393-411`).
- `sample-payments-feed`: proposed by the second administrator and still `proposed`. The first can approve it (`integrationseed.go:426-430`).
- Dead letter `delivery-3` (rejected, sequence 2): the **first administrator requested redelivery**. The second can approve it. Dismissal can be requested by anyone (`integrationseed.go:418-425`, `integrations_api_test.go:180-186`).
- `delivery-5`: open, `waiting`, 1 attempt.

**Event types:** `invoice.issued` and `payment.received`, proposed by the first administrator and approved by the second.

# Design

The design follows the P01, O01–O03 and M01 live patterns:
- full viewport width; neutral rules and tags, no coloured edge strips;
- EN, KO and AR copy;
- the shared failure box and the reason field (UTF-8 bytes, 256), with an error summary;
- `useWide` at 1280px;
- dates through `Intl` in UTC with first-strong isolation;
- IDs, digests and fingerprints in `<bdi dir=ltr>` monospace.

Classes use the `p02-` prefix.

## Host, remount and URL state

- **Live branch.** `operations.tsx` sets `live = !!service && (screenId === "P01" || screenId === "P02")`. For P02 it returns, before the preview wrapper:
  - `<div className="work-domain remaining-security" data-screen-id="P02" data-source-state="live" dir lang key={`P02:${locale}`}>` around `LiveIntegrations`.
  - The key is locale only. Hooks stay unconditional, and the live branch never reports `source: unconnected` (the existing `if (live) return;` covers it).
- **Remount per view (rule).** The parent `RemainingScreens` keys `RevisionScope` by screen, locale and view (`RemainingScreens.tsx:81-92`), so every tab change remounts `LiveIntegrations` and reloads the read, whatever the host key.
  - Component state (the page, notices, open forms) does not survive a tab change. Only the module-level store (attempts, drafts, kept refusals) survives.
  - Announcements are shown within one view. A cross-view link carries its target in the URL (`subscription=`, `delivery=`), never in component state.
- **URL state:**
  - `view=` is one of `clients`, `modules`, `channels` or `events`; anything else falls back to `clients`.
  - Optional selections, each validated against the read and otherwise ignored:
    - `proposal=<n>` (modules; canonical decimal, present in `modules.proposals`);
    - `subscription=<id>` (channels and events; present in `channels.subscriptions`);
    - `delivery=<id>` (events; present in `events.deliveries`).
  - A selected row is scrolled into view once and marked `aria-current`.
  - On events, `subscription=` filters the deliveries to that subscription, with a "Show all deliveries" link.
  - Subscription rows link to their deliveries (`view=events&subscription=<id>`), and delivery rows link to their subscription. Only the canonical ID is passed.
- **Focus and live region:**
  - After a successful command, a status message (`#p02-notice`, `role=status`) is focused.
  - A refusal focuses the row note.
  - Cancel returns focus to the opener.
  - Error-summary links move focus to the field.
  - The reload sets `aria-busy` on the screen.
- **Stale responses.** A request sequence (`seq` ref) drops stale read responses. A command form ignores a second send while busy.
- **Store.** The attempt and draft store (module level, keyed by command and target) is scoped by workspace, user and session (CSRF handle), as `setScope` in P01. It is cleared by `forget()` on sign-out and on a scope change.

## Shared frame (every view)

- **Scope bar:** workspace, data revision, now (UTC), Reload.
- **Statement:**
  - "**Records decisions only.** Nothing is dispatched, sent, signed or issued from this screen in this build."
  - KO: "**결정만 기록합니다.** 이 빌드에서는 이 화면에서 아무것도 발송·전송·서명·발급하지 않습니다."
  - AR: "**تسجيل القرارات فقط.** لا يُرسَل أي شيء ولا يُوزَّع ولا يُوقَّع ولا يُصدَر من هذه الشاشة في هذا الإصدار."
- **Permission note** (small, under the statement):
  - "Every command here currently needs installation administration. Separate API, module and channel permissions are not available yet."
  - KO: "현재 이 화면의 모든 명령에는 설치 관리 권한이 필요합니다. API·모듈·채널별 권한은 아직 제공되지 않습니다."
  - AR: "تتطلب كل الأوامر هنا حاليًا صلاحية إدارة التثبيت. الصلاحيات المنفصلة لواجهات API والوحدات والقنوات غير متاحة بعد."
- **Whole-page states:**
  - first load: "Loading integrations…" with no numbers;
  - reload: `aria-busy`;
  - `not_configured`: "Integrations are not configured in this workspace";
  - an unknown `state`: "The integrations read returned an unknown state (code)";
  - denied: "Integrations are not available to you" / "You do not have permission to manage integrations in this installation.";
  - unauthenticated: "Your session has ended." with "Sign in again" (calls `forget()`);
  - failure (`busy`, `unavailable`, `limited` with the wait) with Try again.
- **Section states.** A section whose `state` is `not_configured` shows its sentence instead of its tables:
  - modules: "No plugins are registered in this workspace.";
  - channels: "No webhook subscriptions are registered.";
  - events: "No event types are approved.";
  - an unknown section state: "Unknown (code)".

## Views (the registered P02 tab keys)

### 1. API clients (`clients`)

- **Lead:**
  - "OAuth and API client registration is not connected in this build. No client credential or token is issued here."
  - KO: "이 빌드에서는 OAuth·API client 등록이 연결되어 있지 않습니다. 여기에서는 client 자격 증명이나 토큰을 발급하지 않습니다."
  - AR: "تسجيل عملاء OAuth وواجهات API غير متصل في هذا الإصدار. لا تُصدَر هنا أي بيانات اعتماد أو رموز للعملاء."
- **Clients table** (spec column order: name → kind → … → state): client ID, kind in words ("First-party console"), state in words, grant count.
- **Console grants:**
  - Below the client, a table: grant ID, action (M03 `actionLabel`, fallback "Unknown (code)"), data kind (M03 `kindLabel`, fallback), scope, valid from, valid until, state.
  - Scope reads "Customer: All · Legal entity: All · Contract: All · Period: Any period". IDs are isolated; a `<from>-<to>` period is one ltr isolate.
  - Empty: "This client has no grants shown here. Installation administration grants are not listed."
- **Service users:**
  - Table: ID, state, owner (login, falling back to ID), usable credentials.
  - The count is a number; `0` reads "None usable"; a missing count reads "Unknown".
  - "Service users and their credentials are managed on M04 (Service accounts)", with a link to `#/service-accounts` when `manage_on` is `M04`. Another value shows the code as "Unknown (code)" without a link.
  - Empty: "No service users."
- **OAuth clients:** `not_connected` reads "Not connected"; another value reads "Unknown (code)".

### 2. Modules (`modules`)

- **Lead:**
  - "Approving records the decision in the plugin registry. No plugin runs and no plugin token is issued: the plugin runtime is not connected in this build."
  - KO: "승인하면 플러그인 레지스트리에 결정이 기록됩니다. 이 빌드에서는 플러그인 실행 환경이 연결되어 있지 않아 플러그인이 실행되지 않고 플러그인 토큰도 발급되지 않습니다."
  - AR: "يسجّل الاعتماد القرار في سجل الإضافات. لا تعمل أي إضافة ولا يُصدَر رمز لها، لأن بيئة تشغيل الإضافات غير متصلة في هذا الإصدار."
- **Plugin workspace:**
  - ID, runtime identity, verification workspace (Yes / No / Unknown).
  - `null` reads "No plugin workspace is registered. Plugin changes cannot be proposed."
- **Publishers:**
  - ID and name, revocation ("Not revoked", or "Revoked by {login}: {reason}" and "confirmed by {login}").
  - Keys: version, valid from, valid until, "Retired" or "Current", and a compromise mark in words.
  - "Registering publishers, keys and versions is not connected in this build."
- **Installation** (one block per row):
  - Plugin, version, state in words, reason in words ("None" when empty), registry revision.
  - Tokens: "Not issued". Runtime: "Not connected".
  - Approved endpoints (ID, certificate fingerprint, approver), or "No approved endpoints".
  - When removed, the disposal contract in words (see storage).
  - Stop and Disable are shown only when `can_stop` / `can_disable` are true, with blocker sentences otherwise (see Blockers).
  - Stop confirm: "Takes effect now; no second approval. Stop records the plugin as stopped. It does not recall anything already sent; in this build no plugin has run."
  - Disable confirm: "Takes effect now; no second approval. Disable returns the plugin to installed. A new enable proposal and another administrator's approval are needed to enable it again."
- **Versions table:**
  - Spec column order: name → kind → version/digest → capability → scope → verification → error.
  - Columns:
    - plugin;
    - "Plugin version";
    - version, with the digest (64 hex, wrapping) below;
    - capabilities in words with the target isolated ("Read events: `invoice.issued`", "Serve screen: `overview`");
    - egress destinations and endpoint IDs;
    - verification (signed with key `{key_version}` of `{publisher}`; core range `1.0.0 – 1.9.9` as one ltr isolate);
    - the vulnerable mark ("None", or "Marked vulnerable by {login}: {reason}").
  - A details disclosure per version shows:
    - limits: "30 s", "128 MiB", "2 concurrent"; unknown numbers read "Unknown";
    - the storage contract: namespace and owner; "Kept for evidence for 30 days after removal" or "Deleted on removal" (0 days is a real zero); migration in words;
    - declared screens: label, component in words, fields as `label (source)`. Labels are plain text from the manifest, shown in `<bdi>`.
- **Plugin proposals:**
  - Pending first, then closed, newest first. Each row:
    - number, kind in words, a summary in words, state;
    - proposer and created time, and the rationale on pending rows;
    - for closed rows, the approver or withdrawer with their closing reason.
  - "Showing the newest 50." when 50 rows return. "No plugin proposals." when empty.
  - Summaries per kind (fallback "Unknown (code)"):

| Kind | EN | KO | AR |
| --- | --- | --- | --- |
| publisher | Register publisher {name} ({id}) | 게시자 {name}({id}) 등록 | تسجيل الناشر {name} ({id}) |
| key | Register publisher key {key} of {publisher}, valid {from} to {to} | {publisher} 게시자 키 {key} 등록, {from}~{to} 유효 | تسجيل مفتاح الناشر {key} لـ {publisher}، صالح من {from} إلى {to} |
| version | Register version {version} of {plugin} | {plugin} 버전 {version} 등록 | تسجيل الإصدار {version} من {plugin} |
| workspace | Register plugin workspace {workspace} (runtime {runtime}) | 플러그인 업무 공간 {workspace} 등록(실행 ID {runtime}) | تسجيل مساحة عمل الإضافات {workspace} (التشغيل {runtime}) |
| runtime | Change the runtime of {workspace} to {runtime} | {workspace} 실행 ID를 {runtime}(으)로 변경 | تغيير بيئة تشغيل {workspace} إلى {runtime} |
| install | Install {plugin} {version} | {plugin} {version} 설치 | تثبيت {plugin} {version} |
| upgrade | Upgrade {plugin} to {version} | {plugin}을(를) {version}(으)로 업그레이드 | ترقية {plugin} إلى {version} |
| rollback | Roll back {plugin} to {version} | {plugin}을(를) {version}(으)로 롤백 | إرجاع {plugin} إلى {version} |
| enable | Enable {plugin} | {plugin} 사용 설정 | تفعيل {plugin} |
| remove | Remove {plugin} | {plugin} 제거 | إزالة {plugin} |
| endpoint | Approve endpoint {endpoint} of {plugin}, certificate {fingerprint} | {plugin} 엔드포인트 {endpoint} 승인, 인증서 {fingerprint} | اعتماد نقطة النهاية {endpoint} لـ {plugin}، الشهادة {fingerprint} |

  - Every placeholder is isolated: IDs and versions in ltr monospace, names in `<bdi>`, dates with FSI. An empty value reads "Unknown".
  - Approve is shown only when `can_approve`; Withdraw only when `can_withdraw`.
  - Approve confirm: "Approval records the change in the plugin registry. No plugin code runs and no token is issued in this build."
  - Withdraw confirm: "Withdrawing closes this proposal. Nothing else changes."
- **Propose a plugin change** (per plugin; derived preconditions, see Command availability):
  - A radio group of the offered changes: "Install {v}", "Upgrade to {v}", "Roll back to {v}", "Enable", "Remove".
  - The confirm restates the plugin, the change, the version, the installed version and state, and for Remove the storage contract.
  - Impact: "Another administrator must approve it. Approval records the installation in the plugin registry; no plugin code is deployed or run in this build."
  - When nothing is offered, one sentence says why: "No plugin workspace is registered", "No other version is registered", "Proposal #{n} already asks for this", or "Your proposal #{n} already asks for this".

### 3. Subscriptions and channels (`channels`)

- **Lead:**
  - "Approving or resuming a subscription records it. No webhook is signed or sent: the signer and the dispatcher are not connected in this build."
  - KO: "구독을 승인하거나 재개하면 기록만 됩니다. 이 빌드에서는 서명기와 발송기가 연결되어 있지 않아 웹훅을 서명하거나 보내지 않습니다."
  - AR: "يسجّل اعتماد الاشتراك أو استئنافه القرار فقط. لا يُوقَّع أي webhook ولا يُرسَل، لأن أداة التوقيع والمُرسِل غير متصلين في هذا الإصدار."
- **Destinations table:** ID, state in words, source ("Sample allowlist"; unknown → "Unknown (code)").
  - "Destination addresses and redirects are not checked in this build; the list is a fixed sample allowlist."
- **Subscriptions** (spec column order):
  - ID, subscriber, destination, event types, state with reason.
  - Cursor and lag: "Cursor 3 · 0 behind". A real 0 reads "0"; an unknown number reads "Unknown".
  - Gaps: "Skipped events after {after} through {to}", or "No gaps".
  - Proposer, approver, resume requester; created; the proposer's rationale.
- Subscription details disclosure:
  - **History:** action in words, actor, rationale.
  - **Signing keys:** version, valid from and until, state, proposer, approver, or "No active signing key".
  - "Signer: Not connected. Key proposals and rotation are not connected in this build."
- **Actions per row**, each shown only when its availability is allowed:
  - **Approve.** "Approval records the subscription. Nothing is sent: the dispatcher is not connected in this build."
  - **Pause.** "Takes effect now; no second approval. Pausing records the subscription as paused; another administrator must approve a later resume."
  - **Request resume.** "Another administrator must approve the resume."
  - **Approve resume.** "The subscription becomes active again. Nothing is sent in this build."
  - **Revoke.** "Takes effect now; no second approval. Revoking ends the subscription permanently and cancels its open delivery. A delivery that may already have reached the receiver is marked outcome unknown. Queued redeliveries of this subscription will not be sent. This cannot be undone."
- **Propose a subscription** (derived preconditions, see Command availability):
  - The subscriber is shown as a fact (the single value of `subscribers`).
  - Destination: a radio list. Only `active` destinations are selectable; inactive and revoked ones are listed as "Inactive – cannot be used".
  - Event types: checkboxes of the approved types, 1 to 8.
  - The confirm restates the subscriber, the destination and the types.
  - Impact: "Another administrator must approve it. Nothing is sent in this build."
  - The types are sent joined with "," in ascending order.

### 4. Events and redelivery (`events`)

- **Lead:**
  - "The dispatcher is not connected. Queued and waiting deliveries stay as they are, and approving a redelivery only queues it. Nothing is sent from this build."
  - KO: "발송기가 연결되어 있지 않습니다. 대기 중인 발송 건은 그대로 남아 있고, 재발송을 승인해도 대기열에 넣기만 합니다. 이 빌드에서는 아무것도 보내지 않습니다."
  - AR: "المُرسِل غير متصل. تبقى عمليات التسليم المنتظرة كما هي، واعتماد إعادة التسليم يضعها في قائمة الانتظار فقط. لا يُرسَل أي شيء من هذا الإصدار."
- **Event types table:** name, schema version, fields, state, proposer, approver.
  - "Event publication and type proposals are not connected here."
- **Outbox:** "Oldest kept event {floor} · newest event {head}". Unknown numbers read "Unknown".
- **Deliveries table:**
  - Columns: delivery ID, subscription, sequence, event (source and event ID, each isolated), kind in words, job state in words, reason in words ("None" when empty), attempts (a real 0 reads "0").
  - When `outcome_unknown` is true: "Outcome unknown: the receiver may have processed it before it was cancelled." `false` reads nothing; a missing flag reads "Outcome: Unknown".
  - Empty: "No open, queued, dead or cancelled deliveries. Delivered ones are not kept."
  - "Showing up to 50 deliveries per subscription."
  - Rows of a revoked subscription add: "The subscription is revoked; this delivery will not be sent."
- **Dead-letter rows** add the pending requests ("Redelivery requested by {login}: {reason}", "Dismissal requested by …") and the four actions, each shown only when allowed:
  - **Request redelivery.** "Another administrator must approve it."
  - **Approve redelivery.** "A new delivery is queued and this dead letter is removed. Nothing is sent: the dispatcher is not connected in this build."
  - **Request dismissal.** "Another administrator must approve it."
  - **Approve dismissal.** "The dead letter is removed. Nothing is sent."
  - After approving a redelivery, the notice names the new delivery ID from `fields.redelivery`, or "Unknown" when it is absent.

## Command availability (fail closed)

- **Parsing.** An availability counts only when `allowed === true` and `blocked === ""`.
  - Otherwise the blocker is the string, or `missing` when absent or not a string; an empty blocker with `allowed` not true is also `missing`.
  - A missing actions map or action key is `missing`.
  - The flat pairs (`can_approve`/`approve_blocked`, `can_withdraw`/`withdraw_blocked`, `can_stop`/`stop_blocked`, `can_disable`/`disable_blocked`) use the same rule as P01's `hint` (`installation.ts:172-176`): `can_*` true with a non-empty blocker is not allowed.
- **Targets.** Commands use the row's own `target` only when it matches its pattern and its ID. Otherwise no command is offered and the row reads "No command is available for this row."
  - plugin proposals: `^plugin-proposal-[1-9][0-9]*$` and equal to `plugin-proposal-` + `number`;
  - subscriptions: equal to `webhook-subscription:` + `id`;
  - letters: equal to `webhook-letter:` + `id`;
  - plugins (built by the screen): `plugin:` + the installation or version row's `plugin`.
  - Every target must be 1–128 bytes of `[A-Za-z0-9._:-]` (`modules/commands/types.go:297-307`).
- **Read blockers** gate approve and withdraw (plugin proposals), stop and disable (installations), the five subscription actions and the four letter actions.
- **Blocker display policy.** A blocked action is never shown as a button. Its blocker is either hidden or explained:
  - **Hidden only** (the row's state already says why): `not_pending` on a closed or non-pending row; `not_active`; `not_dead`; `not_proposer`; `not_enabled` and `not_allowed_state` on stop and disable, when the installation's state column shows the state.
  - **Explained with a sentence:** `own_proposal`, `own_request`, `already_requested`, `not_requested`, the admission codes (`destination_inactive`, `subscribe_refused`, `dependency_unavailable`), every dry-run code, `revoked`, `missing` and any unknown code.
  - `not_pending` on a pending plugin proposal is unexpected and therefore explained. `not_pending` on the resume actions and revoke is hidden, because the subscription state says it.
- **Derived preconditions** for the two propose commands. The read has no availability for them, so the screen derives when to offer them. These rules only narrow what is offered: the server remains the authority, and any refusal shows the kept sentence.
  - **`plugin_propose`:**
    - Offered only when `modules.workspace` is present and the plugin has at least one well-formed version row.
    - Install `{v}`: no installation row for the plugin, or the row is `removed`.
    - Upgrade or roll back to `{v}`: the installation is not `removed` and `{v}` is a registered semver newer or older than the installed version.
    - Enable: the state is `installed` or `stopped`, and the installed version passes the two checks below. Remove: the state is not `removed`.
    - Versions with a vulnerable mark are not offered for install, upgrade, rollback or enable unless the workspace is a verification workspace.
    - While the publisher is revoked or its signing key is compromised, install, upgrade, rollback and enable are not offered.
    - Remove is exempt from both rules, so a vulnerable or revoked plugin can always be proposed for removal.
    - A change already pending with the same kind, plugin and version is not offered again, whether the reader or another administrator proposed it. It reads "Proposal #{n} already asks for this", or "Your proposal #{n} already asks for this" for the reader's own.
  - **`subscription_propose`:**
    - Offered only when `events.types` has at least one approved type, at least one destination is `active`, and `subscribers` has exactly one well-formed value. These are its only preconditions: it is shown even when the channels section is `not_configured` (no subscription yet).
    - A chosen combination that equals a `proposed` or `active` subscription (same destination and the same set of types) is not sent. It reads "Subscription {id} already covers this destination and these event types", or "You already proposed this as {id}" for the reader's own proposed one.
- **Resend.** A pending attempt (unknown outcome) keeps its command usable even if the reloaded row no longer allows it. The resend reuses the frozen target, values, reason and `expected_revision`, and the journal returns the original result, except for targets that disappear (see Replay exception).

## Blocker sentences (every code, unknown fallback)

Every sentence has EN, KO and AR copy in `live-codes.ts`. Any other code reads "This is not available now (code)" with the code in an ltr isolate. `missing` reads "This is not available now."

| Code | Where | EN |
| --- | --- | --- |
| `own_proposal` | plugin and subscription approve | You proposed this. Another administrator must approve it. |
| `not_pending` | plugin approve and withdraw; subscription actions | This is no longer waiting for a decision. / (request resume) The subscription is not paused or suspended. / (approve resume) No resume has been requested. / (revoke) The subscription is already revoked. |
| `not_proposer` | plugin withdraw | (no sentence; Withdraw is hidden) |
| `own_request` | approve resume, approve redelivery, approve dismissal | You requested this. Another administrator must approve it. |
| `already_requested` | request resume, redelivery, dismissal | A request is already waiting for approval. |
| `not_requested` | approve redelivery or dismissal | Nothing has been requested yet. |
| `not_dead` | letter actions | (no sentence; letter actions are shown only on dead letters) |
| `revoked` | approve redelivery | The subscription is revoked, so this dead letter cannot be redelivered. It can still be dismissed. |
| `not_active` | pause | Only an active subscription can be paused. |
| `destination_inactive` | subscription approve and resume; plugin dry run | The destination is not active. |
| `subscribe_refused` | subscription approve and resume | The subscriber is not permitted to receive these event types. |
| `dependency_unavailable` | subscription approve and resume; dry run | A required check is unavailable. Try again later. |
| `not_enabled` | stop | Only an enabled plugin can be stopped. |
| `not_allowed_state` | disable; dry run | This is not allowed in the plugin's current state. |
| `direction` | dry run | The version is not newer (upgrade) or older (rollback) than the installed one. |
| `storage_changed` | dry run | The version changes the storage contract, so it cannot replace the installed one. |
| `namespace_taken` | dry run | Another installation already uses this storage namespace. |
| `runtime_taken` | dry run | Another workspace already uses this runtime identity. |
| `not_declared` | dry run | The installed version does not declare this endpoint. |
| `fingerprint_mismatch` | dry run | The certificate fingerprint does not match. |
| `key_inactive` | dry run | The signing key is not valid now. |
| `key_life` | dry run | The key's validity period is outside the allowed range. |
| `key_mismatch` | dry run | The key does not match its proof. |
| `weak_key` | dry run | The key is weak and cannot be used. |
| `publisher_inactive` | dry run | The publisher is not registered or not active. |
| `publisher_mismatch` | dry run | The version names a different publisher than its key. |
| `signature_invalid` | dry run | The version's signature is not valid. |
| `incompatible` | dry run | The version does not support this core version. |
| `vulnerable` | dry run | The version is marked vulnerable and cannot run outside a verification workspace. |
| `revoked` | dry run (plugin) | The publisher or its key is revoked or compromised. |
| `isolated` | dry run | The plugin is isolated. |
| `expired` | dry run | The proposal's evidence has expired. |
| `not_found` | dry run | Something this proposal names is no longer registered. |
| `conflict` | dry run | Another record already uses this identifier. |
| `limit` | dry run | A registry limit is reached. |

- `revoked` has two sentences, chosen by where it appears: the letter action or the plugin dry run.
- A dry-run blocker that the spec calls "blocked" (`signature_invalid`, `vulnerable`, `revoked`, `key_inactive`, `publisher_inactive`, `incompatible`) adds the spec's sentence: "Unverified version, destination or permission. It cannot be activated." (KO "검증되지 않은 버전·목적지·권한입니다. 활성화할 수 없습니다.").

## Attempts, refusals and reactions

- **Attempts.** A request ID and an idempotency key are kept per command and target in the scoped store.
  - They are reused only to resend identical values (fields and reason) after an unknown outcome (network failure or `busy`), even if a reload moved the revision.
  - Any edit or any definite answer discards them.
  - A pending attempt shows "The last request may have been applied. Check the list, or send it again."
- **Reason.** The reason is trimmed before sending; an empty or whitespace-only reason is rejected on the client ("Enter a reason"), as is one over 256 UTF-8 bytes.
- **Drafts.** The reason, the propose choices (kind and version; destination and types) are kept per command and target while the session lasts.
- **Reactions:**
  - **`invalid` with fields.**
    - `version` (plugin propose) → "The plugin's versions or state changed. Review the current choices." and a reload.
    - `kind` (plugin propose) → "The screen sent a change the service does not accept. Reload and try again." The screen only sends listed kinds, so this indicates a client bug.
    - `destination` or `types` → an error on that field in the summary and a reload, because the allowlist or types may have changed.
    - `rationale` → the reason field.
    - Any other field (`target_id`, `expected_revision`, `idempotency_key`, `request_id` or an unknown name) → "The request could not be sent as built. Reload and try again." It is never mapped to the reason field.
  - **`invalid` without fields** → the reason field ("The reason is too long" when it is over 256 bytes, otherwise "Check the reason").
  - **`stage` or `refused`:**
    - Reload, then the row's current blocker sentence.
    - When the reloaded row shows no blocker (or there is no row), a kept per-command sentence:
      - plugin approve: "The plugin registry refused this approval. Nothing changed. Review the proposal, or ask the proposer to withdraw it.";
      - plugin withdraw: "The plugin registry refused this withdrawal. Nothing changed.";
      - plugin propose: "The plugin registry refused this proposal. Nothing changed. Check the installed version and state; you may also have too many pending proposals.";
      - stop or disable: "The plugin registry refused this change. Nothing changed.";
      - subscription propose: "The subscription was refused. Nothing changed. Another administrator may already have proposed the same destination and event types, or the subscription limit is reached.";
      - subscription actions: "This subscription action was refused. Nothing changed. Reload and check the subscription's state.";
      - letter actions: "This dead-letter action was refused. Nothing changed. It may already have been redelivered, dismissed or requested."
    - The kept sentence clears on the next definite answer for that command and target, or when the reloaded row shows a blocker or closes. Kept sentences for rows that left the read are dropped (P01 `pruneRefusals`).
  - **`conflict`** → reload, keep the reason and the choices, and ask to confirm again.
  - **`denied` on a command:**
    - Reload. If the read succeeds: "This proposal is no longer available", "This plugin is no longer available", "This subscription is no longer available" or "This dead letter is no longer available".
    - For the two propose commands: "Plugin changes can no longer be proposed here" or "Subscriptions can no longer be proposed here".
    - If the read is denied, the screen-level denied block.
    - **Exception for `letter_approve_redeliver` and `letter_approve_dismiss` with a pending attempt.** When `denied` answers a resend (or a first send that follows an unknown outcome) and the letter is missing from the reloaded read, the screen says "Your earlier approval may have been applied. The dead letter is gone." For redelivery it adds "Check the deliveries for a new queued redelivery of event {event_id}." The attempt is then discarded. The same sentence covers a pruned plugin proposal, without the deliveries hint.
- **Pending attempts whose row vanished.** After every reload, a pending attempt whose target is no longer in the read is shown in a view-level notice above the lists (the row that would hold it is gone), with the sentence above for letters and "This item is no longer listed. Your earlier request may have been applied." otherwise. The notice clears when the attempt is discarded.
  - **Other failures:**
    - `unauthenticated` → sign in again;
    - `limited` → the wait;
    - `busy` or `unavailable` → retry with the same attempt;
    - `stale` → "Start again" (renew the attempt);
    - `partial` → the operation text with its ID;
    - anything else → "Failed".
- **Duplicate results.** A propose whose returned `proposal` or `subscription` was already listed announces "This was already proposed as #{n}" or "…as {id}".
  - "Already listed" means listed in the read the attempt was built from: the attempt stores a snapshot of the listed proposal numbers and subscription IDs taken before its first send. A resend compares against that snapshot, so a replayed stored success is announced as the new proposal it was, not as a duplicate.
  - Through the real API this message is not expected. The preconditions hide a change the reader already proposed, and a stale tab gets `revision_conflict`: every success moves the revision (`integrationcmds.go:386`), and an outdated `expected_revision` is refused before `Apply` (`execute.go:356-358`). The message is kept as a defence and is covered only by model and service tests with a stubbed success.
- **Successes** name the result in words, for example "Proposal #9 applied", "Subscription sample-payments-feed is active", "Redelivery requested", or "Redelivery queued as {redelivery}. Nothing is sent." Unknown result states read "Unknown (code)".

## Unknown versus zero or empty

- The page `revision` is an opaque string and is shown and sent as is; an empty one reads "Unknown" and disables commands.
- Numbers are finite safe integers or `null`; `null` reads "Unknown". This covers `cursor`, `lag`, `after`, `to`, `floor`, `head`, `sequence`, `attempts`, `revision`, `timeout_seconds`, `memory_mib`, `concurrency`, `retention_days`, `usable_credentials` and `number`. A real 0 reads "0".
- Missing arrays are `null` and read "Unknown"; empty arrays read "None" or the view's empty sentence.
- Booleans (`verification`, `retired`, `outcome_unknown`, `can_*`) are `null` when not a boolean. A null `can_*` fails closed.
- Mark and request objects: `null` reads "None"; a non-object reads "Unknown".
- Times that are empty, unparseable or year 1 read "Unknown".
- Empty strings read "Unknown", except `reason` (empty reads "None") and an empty approver on a pending row (not shown).
- An unknown enum value reads "Unknown (code)", never blank.

## Mobile, RTL and wrapping

- **Mobile.** Below 768px, tables become stacked labelled rows; grids and forms become one column; radio and checkbox lists go full width.
  - Digests and fingerprints wrap with `overflow-wrap:anywhere` inside `<bdi dir=ltr>`.
  - No horizontal scroll at 390px.
- **RTL.** A compound identifier is one ltr isolate, never split into pieces in RTL:
  - `kind:target` capabilities when unknown;
  - `plugin:` and `webhook-…:` targets;
  - `<from>-<to>` periods; the core range `1.0.0 – 1.9.9`;
  - `{after}–{to}` gaps; `source` with `event_id`.
  - Logins are isolated as ltr. Dates use FSI…PDI inside sentences. Manifest labels and reasons use `<bdi>` (first strong).

## Vocabularies (EN / KO / AR)

All are in `live-codes.ts` with an "Unknown (code)" fallback; tests check every key in every language. Korean and Arabic are AI drafts; the open terminology choices are listed under Out of scope.

| Group | Code | EN | KO | AR |
| --- | --- | --- | --- | --- |
| Section | configured | Configured | 설정됨 | مُعدّ |
| Section | not_configured | Not configured | 설정되지 않음 | غير مُعدّ |
| Connection | not_connected | Not connected | 연결되지 않음 | غير متصل |
| Connection | not_issued | Not issued | 발급되지 않음 | لم يُصدَر |
| Client kind | first_party_console | First-party console | 자사 콘솔 | وحدة التحكم الأساسية |
| Client state | active | Active | 사용 중 | نشط |
| Grant state | active | Active | 유효 | سارية |
| Grant state | pending | Not yet valid | 유효 시작 전 | لم تبدأ بعد |
| Grant state | expired | Expired | 만료됨 | منتهية |
| Grant state | revoked | Revoked | 회수됨 | مُلغاة |
| Scope | all | All | 전체 | الكل |
| Scope | any | Any period | 전체 기간 | أي فترة |
| Service user | active | Active | 사용 중 | نشط |
| Service user | suspended | Suspended | 정지됨 | موقوف |
| Service user | retired | Retired | 폐기됨 | متقاعد |
| Installation | installed | Installed (not enabled) | 설치됨(사용 안 함) | مثبّتة (غير مفعّلة) |
| Installation | enabled | Enabled | 사용 설정됨 | مفعّلة |
| Installation | stopped | Stopped | 중지됨 | متوقفة |
| Installation | removed | Removed | 제거됨 | مُزالة |
| Install reason | operator | Stopped by an administrator | 관리자가 중지함 | أوقفها مسؤول |
| Install reason | timeout | Stopped after a timeout | 시간 초과로 중지됨 | توقفت بعد انتهاء المهلة |
| Capability | read_events | Read events | 이벤트 읽기 | قراءة الأحداث |
| Capability | call_command | Call command | 명령 호출 | استدعاء أمر |
| Capability | serve_screen | Serve screen | 화면 제공 | تقديم شاشة |
| Component | table | Table | 표 | جدول |
| Component | form | Form | 입력 양식 | نموذج |
| Component | detail | Detail | 상세 | تفاصيل |
| Component | chart | Chart | 차트 | مخطط |
| Deletion | delete_on_removal | Deleted on removal | 제거 시 삭제 | تُحذف عند الإزالة |
| Deletion | retain_for_evidence | Kept for evidence | 증빙용 보관 | تُحفظ كدليل |
| Migration | none | No migration | 이관 없음 | بلا ترحيل |
| Migration | additive | Additive changes only | 추가 변경만 | تغييرات إضافية فقط |
| Migration | copy_forward | Copied forward | 새 버전으로 복사 | نسخ إلى الإصدار الجديد |
| Proposal kind | publisher | Publisher | 게시자 | الناشر |
| Proposal kind | key | Publisher key | 게시자 키 | مفتاح الناشر |
| Proposal kind | version | Plugin version | 플러그인 버전 | إصدار الإضافة |
| Proposal kind | workspace | Plugin workspace | 플러그인 업무 공간 | مساحة عمل الإضافات |
| Proposal kind | runtime | Runtime identity | 실행 ID | هوية التشغيل |
| Proposal kind | install | Install | 설치 | تثبيت |
| Proposal kind | upgrade | Upgrade | 업그레이드 | ترقية |
| Proposal kind | rollback | Rollback | 롤백 | إرجاع |
| Proposal kind | enable | Enable | 사용 설정 | تفعيل |
| Proposal kind | remove | Remove | 제거 | إزالة |
| Proposal kind | endpoint | Endpoint | 엔드포인트 | نقطة النهاية |
| Proposal state | pending | Pending | 승인 대기 | بانتظار الاعتماد |
| Proposal state | applied | Applied | 적용됨 | مطبّق |
| Proposal state | withdrawn | Withdrawn | 철회됨 | مسحوب |
| Destination | active | Active | 사용 가능 | نشطة |
| Destination | inactive | Inactive | 사용 불가 | غير نشطة |
| Destination | revoked | Revoked | 회수됨 | مُلغاة |
| Destination source | sample_allowlist | Sample allowlist | 샘플 허용 목록 | قائمة السماح التجريبية |
| Subscription | proposed | Proposed | 승인 대기 | مقترح |
| Subscription | active | Active | 활성 | نشط |
| Subscription | paused | Paused | 일시 중지됨 | متوقف مؤقتًا |
| Subscription | suspended | Suspended | 자동 정지됨 | معلّق |
| Subscription | revoked | Revoked | 회수됨 | مُلغى |
| Subscription reason | paused | Paused by an administrator | 관리자가 일시 중지함 | أوقفه مسؤول مؤقتًا |
| Subscription reason | gap | Events were skipped (expired cursor) | 이벤트 누락(만료된 커서) | تخطّي أحداث (مؤشر منتهي) |
| Subscription reason | backlog | Too many undelivered items | 미전달 건 과다 | عناصر غير مسلّمة كثيرة |
| Subscription reason | destination_inactive | Destination not active | 목적지 사용 불가 | الوجهة غير نشطة |
| Subscription reason | destination_revoked | Destination revoked | 목적지 회수됨 | الوجهة مُلغاة |
| Subscription reason | permission_revoked | Permission revoked | 권한 회수됨 | الصلاحية مُلغاة |
| Subscription reason | subscription_revoked | Revoked by an administrator | 관리자가 회수함 | ألغاه مسؤول |
| Subscription reason | dependency_unavailable | A required check was unavailable | 필수 확인을 사용할 수 없었음 | تعذّر إجراء فحص مطلوب |
| Subscription reason | key_unpublished | No published signing key | 게시된 서명 키 없음 | لا يوجد مفتاح توقيع منشور |
| Subscription reason | key_mismatch | Signing key mismatch | 서명 키 불일치 | عدم تطابق مفتاح التوقيع |
| Key state | proposed | Proposed | 승인 대기 | مقترح |
| Key state | active | Active | 사용 중 | نشط |
| Key state | retired | Retired | 폐기됨 | متقاعد |
| History action | approve | Approved | 승인 | اعتُمد |
| History action | pause | Paused | 일시 중지 | أُوقف مؤقتًا |
| History action | request_resume | Resume requested | 재개 요청 | طُلب الاستئناف |
| History action | approve_resume | Resume approved | 재개 승인 | اعتُمد الاستئناف |
| History action | revoke | Revoked | 회수 | أُلغي |
| Event type | active | Approved | 승인됨 | معتمد |
| Delivery kind | open | Delivery | 발송 건 | تسليم |
| Delivery kind | redelivery | Redelivery | 재발송 건 | إعادة تسليم |
| Delivery kind | dead | Dead letter | 실패 보관 건 | رسالة غير مُسلّمة |
| Delivery kind | cancelled | Cancelled | 취소됨 | مُلغى |
| Job state | queued | Queued | 대기열 등록 | في قائمة الانتظار |
| Job state | running | Running | 실행 중 | قيد التشغيل |
| Job state | waiting | Waiting (or, when the row's subscription is paused or suspended, "Waiting for the subscription to resume") | 대기 중 (구독이 일시 중지·자동 정지된 경우 "구독 재개 대기") | في الانتظار (أو "بانتظار استئناف الاشتراك" عند إيقاف الاشتراك مؤقتًا أو تعليقه) |
| Job state | succeeded | Succeeded | 성공 | نجح |
| Job state | failed | Failed | 실패 | فشل |
| Job state | cancelled | Cancelled | 취소됨 | أُلغي |
| Delivery reason | rejected | Rejected by the receiver | 수신 측이 거부함 | رفضها المستلم |
| Delivery reason | attempts_exhausted | Attempts exhausted | 재시도 횟수 소진 | استُنفدت المحاولات |
| Delivery reason | max_age | Too old to deliver | 전달 가능 기간 초과 | قديمة جدًا للتسليم |
| Delivery reason | delivered | Delivered | 전달됨 | سُلّمت |
| Delivery reason | subscription_revoked, permission_revoked, destination_inactive, destination_revoked, dependency_unavailable | (as the subscription reasons) | | |
| Letter request | requested | Requested | 요청됨 | مطلوب |
| Result state | dismissed | Dismissed | 정리됨 | أُزيلت |

`partial` from the spec is not a P02 job state in this API. A `waiting` row reads "Waiting" or "Waiting for the subscription to resume", never "Partly done".

## Tabs

The shell labels the tabs from `screens.json`:
- KO: API client / 모듈 / 구독·채널 / 사건·재처리.
- In-screen view titles: EN "API clients", "Modules", "Subscriptions and channels", "Events and redelivery"; KO "API client", "모듈", "구독·채널", "이벤트·재처리"; AR "عملاء API", "الوحدات", "الاشتراكات والقنوات", "الأحداث وإعادة التسليم".

## External references

The ui-designer (AI, read-only) recalled these from memory; nobody opened them, so they are unverified:
- **Stripe webhooks** (https://docs.stripe.com/webhooks). Adopted: a subscription per destination with chosen event types; delivery attempts with state and count; a per-row resend. Rejected: test events, revealing secrets, coloured pills.
- **GitHub redelivering webhooks** (https://docs.github.com/en/webhooks/testing-and-troubleshooting-webhooks/redelivering-webhooks). Adopted: the delivery ID and event on each row; a redelivery creates a new delivery. Rejected: the body and header viewer, green and red icons.
- **Amazon SQS dead-letter queues** (https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-dead-letter-queues.html). Adopted: a dead letter as an explicit state with its reason and attempt count; redrive as a deliberate action. Rejected: bulk redrive.
- **Atlassian UPM, viewing installed apps** (https://confluence.atlassian.com/upm/viewing-installed-apps-273875714.html). Adopted: an installed-plugin block with version and state; explicit enable and disable; a details disclosure. Rejected: marketplace browsing, upload, one-click install without a second approver.
- **Svix app portal** (https://docs.svix.com/app-portal). Adopted: event types as checkboxes; attempts linked to their endpoints. Rejected: a secret rotation UI, URL entry.

The visual specification (ui-designer, AI) defines the `p02-` classes and the layout; root keeps it with the ticket's working notes.

## Rules carried from the HB-283 reviews

All are part of this design from the start:
- **Host and remount:** live branch before the preview wrapper, locale-only key, hooks unconditional.
- **URL state:** validated selections. **Focus and live region:** status focused after success, row note after refusal, opener after cancel, summary links to fields.
- **Stale responses:** the request sequence.
- **Store:** scoped by workspace, user and session, and cleared on sign-out.
- **Mobile:** stacking at 390 with no horizontal scroll.
- **RTL:** one ltr isolate per compound ID; FSI for dates.
- **Unknown versus zero and empty.** A year-1 time reads "Unknown".
- **Blockers:** a sentence for every blocker code, with an unknown fallback. Rows without a well-formed target offer no command; a missing number fails closed. The proposer's rationale shows on pending rows and in the approve confirm. Logins fall back to IDs. "Showing the newest 50."
- **Refusals:** the kept sentence when the reloaded blocker is empty, and its clearing. On a command, `denied` → reload → "no longer available".
- **Truthful statements:** what each decision does and does not do.

## Out of scope (owner residuals)

- **Not connected in this build:**
  - dispatch, signing, plugin tokens and the runtime;
  - webhook key proposal, approval and rotation;
  - publisher, key, version and endpoint registration;
  - marks (vulnerable, compromise, revocation, confirm, clear);
  - runtime changes and verification workspaces;
  - event publication and type proposals;
  - OAuth and API client registration;
  - `modules/trust` destinations;
  - persistence.
- **Separate permissions:** API, module and channel permissions, plus stop and reprocess permissions (the HB-284 deviation).
- **Not built from the P02 spec:**
  - server search, filters and stable sort;
  - the selected-row drawer, before and after comparison, and bulk selection with snapshots and cursors (the lists are small and bounded at 50);
  - an expired-cursor resync command (only the `gap` reason and gaps are shown);
  - external application origins (QA-06).
- **Navigation:** the only cross-screen link is the M04 link on the clients view. Links to M03, U03, R02, P04 and P05, and M04 actions, are residuals.
- **Reviews:** owner FIX of the screen.
- **Terminology** (owner KO and AR review and technical-writer review; the proposed terms are used until then):
  - webhook delivery: KO "발송 건", because the glossary's "수집본" means a collection delivery;
  - dead letter: KO "실패 보관 건", AR "رسالة غير مُسلّمة";
  - plugin publisher: KO "플러그인 게시자", because the glossary's "상품 제공사" is the FOCUS publisher;
  - the registered tab label "사건·재처리" against "이벤트·재처리" in the screen text;
  - every other KO and AR string in this plan, which are AI drafts.

# Verification plan

- **Model tests** (`live-model.test.ts`):
  - availability parsing fails closed (missing map, missing key, `allowed` true with a blocker, non-string blocker), and the flat `can_*` pairs with `can_*` true plus a blocker;
  - the blocker display policy (each hidden-only code hidden, each explained code with its sentence);
  - Remove offered for a vulnerable version or a revoked publisher while install, upgrade, rollback and enable are not; Enable hidden when the installed version is vulnerable;
  - the subscription propose offered while the channels section is `not_configured`;
  - `letter_approve_*` with a pending attempt answered `denied` and the letter missing after reload → the "may have been applied" sentence (with the event hint for redelivery) and the attempt discarded; the view-level notice for a pending attempt whose row vanished;
  - reason trimming and whitespace-only rejection; the `kind` and non-reason field mappings;
  - target patterns per row kind (malformed, mismatched number or ID, over 128 bytes, characters outside the ID set);
  - blocker sentences for every code in every language, the `missing` sentence and the unknown fallback;
  - proposal summaries per kind with the unknown fallback; proposals ordered pending first, newest first;
  - the plugin-propose preconditions (no workspace; installed, enabled, stopped and removed states; semver direction; vulnerable versions with and without a verification workspace; revoked publisher or compromised key; an existing pending same change, own and other);
  - the subscription-propose preconditions (no types, no active destination, subscriber count not one, an existing proposed or active same set, own and other; type order normalised);
  - the storage contract wording, including a real 0 days;
  - unknown versus zero for every numeric field, null lists, null flags and a year-1 time;
  - every vocabulary key in EN, KO and AR, with no copy claiming a dispatch, a send, a signature or a token;
  - failure reactions; the attempt store (identical values only, scoped by session, cleared by `forget`);
  - the kept refusal sentence for every command when the reloaded blocker is empty, and its clearing;
  - command `denied` → reload → the per-target "no longer available" sentence; the duplicate announcement against the attempt's snapshot with a stubbed success, and no announcement when a resend replays a stored success; UTF-8 byte counting; the URL selection validation.
- **Service tests** (`integrations.test.ts`):
  - hardened parsers for every section (missing arrays, wrong types, unknown enums, unsafe integers);
  - the whole-page and per-section `not_configured`;
  - the command envelope for a command with fields (`plugin_propose`, `subscription_propose`) and without;
  - resend identity, and the result fields (`proposal`, `subscription`, `delivery`, `redelivery`);
  - a stubbed `invalid_stage` for `subscription_propose` with identical content from another proposer (`integrationcmds.go:274-275`), mapped to `stage` and leading to the kept subscription-propose sentence.
- **Checks:** `npx tsc --noEmit`, `npm test`, `npm run build`.
- **Playwright QA** against the real API (`hb-api --seed sample`). Each flow group below starts on a fresh API (restarted with an empty data directory), so no group's starting state depends on another group or on order:
  - **All four views** (Group A's fresh API, before its commands) in EN, KO and AR at 1440 and 390: no horizontal scroll; `dir` per locale; neutral tag borders; the statement and the per-view "not connected" sentences present.
  - **Group A, as the second administrator** (`admin2@example.invalid`):
    - plugin proposal #8 reads "You proposed this" with Withdraw;
    - #9 (endpoint) is approved through the confirm step; the installation then lists endpoint `api` with its fingerprint and approver;
    - `sample-payments-feed` reads "You proposed this";
    - dead letter `delivery-3` shows Approve redelivery; after approval the letter is gone, and a new queued redelivery row of the same event appears (its ID is read from the notice, not hard-coded) with "Nothing is sent".
  - **Group B, as the first administrator** (`admin@example.invalid`), with the second administrator only where stated:
    - #9 reads "You proposed this";
    - #8 (upgrade to 1.1.0) is approved, and the installation then reads 1.1.0;
    - Stop, then Disable (each confirm says "Takes effect now; no second approval"), then propose Enable: the own proposal reads "You proposed this" and Propose no longer offers Enable;
    - propose a rollback to 1.0.0 and withdraw it;
    - approve `sample-payments-feed`;
    - pause `sample-erp-invoices` and request its resume, which then reads "You requested this"; as the second administrator, approve the resume;
    - on `delivery-3`, Approve redelivery is absent (own request) and Request dismissal works, after which a second request is blocked with "already waiting";
    - propose a subscription to `sample-erp-webhook` with `payment.received`; `sample-legacy-webhook` is not selectable;
    - as the second administrator, choosing `sample-erp-webhook` with `payment.received` again shows "Subscription {id} already covers this destination and these event types" and offers no Confirm.
  - **Group C, revoke:** as the first administrator, revoke `sample-erp-invoices` (the confirm says queued redeliveries will not be sent). Then:
    - `delivery-5` reads cancelled with "Outcome unknown". This is deterministic: the seed confirmed it before reporting a failure (`deliveries.go:296`, `423`).
    - As the second administrator (the first sees `own_request`, which `letterActions` returns first, `integrations.go:630-633`), `delivery-3` reads the `revoked` approve-redelivery sentence, and Approve dismissal stays unavailable only for lack of a request: after the first administrator requests dismissal, the second administrator's Approve dismissal is allowed.
  - **Group D, failure paths:**
    - a revision conflict from two tabs (reason kept, confirm again);
    - a network failure on send by route abort for a command whose target stays resolvable, `subscription_pause` on `sample-erp-invoices` (or `plugin_approve` on #8 (as the first administrator; #8 is the second administrator's proposal)): the pending-attempt sentence shows, and the resend returns the original result from the journal;
    - as the second administrator, Approve redelivery on `delivery-3` with the response aborted after the server applied it: the resend is `denied`, the reload no longer lists the letter, and the screen shows "Your earlier approval may have been applied. The dead letter is gone." with the deliveries hint; the attempt is discarded;
    - a reason over 256 bytes is caught on the client;
    - the member login (`member@example.invalid`) sees the denied block;
    - an invalid `view`, `proposal` or `delivery` in the URL is ignored.
  - **Regressions:** P01 (qa283full), O03 (qa281), O02 (qa275), O01 (qa273) and M01 (qa264), each on a fresh API.
  - Console errors during QA: none expected.
- **Reviews and record:** ui-designer render comparison; independent design and change review; `verification.md`.
