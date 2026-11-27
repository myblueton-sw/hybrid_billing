---
type: DesignPlan
title: HYB-397 — P04 deletion records without installation-wide sequence numbers
description: How the retention reads stop exposing the installation's deletion sequence and watermark to a workspace while keeping the newest-first list and its further reads.
status: draft
---

# Scope and ownership

[HYB-397](https://linear.app/myblueton/issue/HYB-397), machine:mac_mini, root (Claude). Written 2026-11-27 from main. Owner decision of 2026-11-27: hide the installation-wide deletion sequence from a workspace. This changes a public read, so it needs an independent tenant/permission review before it is built.

# Facts

- The retention module numbers deletion records (tombstones) in one sequence for the whole installation (`retention.Tombstone.Sequence`), and its watermark is the latest number.
- `GET /api/v1/retention` (`decisions.watermark`, `decisions.tombstones.rows[].sequence`, `decisions.tombstones.next`) and `GET /api/v1/retention/tombstones` (`watermark`, `tombstones[].sequence`, `next`) send these numbers (`apps/api/server/retention.go:224, 242, 718-729`, `retentiontombstones.go`). The rows a workspace receives are filtered by `visibleArtifact`, but their numbers and the watermark are the installation's, so the gaps show roughly how many deletions the workspace cannot see.
- `next` is `keptCursor(sequence)`: the sequence in plain form. The server's sealed tokens (`sealQueryToken`, `sealSessionCursor`) are signed, not encrypted, so a sealed sequence would still be readable.
- The P04 screen shows a sequence column and a "Latest watermark" fact (`retention/LiveDecisions.tsx:19, 25, 64-65`; `service/retention.ts:121-123, 361, 504`).
- The retention read and the tombstone read answer only the installation's workspace (`retention.go`, `retentiontombstones.go`: other workspaces receive an empty page). Visibility (`visibleArtifact`) depends on the workspace, not the user, so the leak runs from other workspaces' deletions to the installation workspace, not between users.
- The tombstone read parses `after` with the shared `parseKeptQuery`, which accepts any number; `keptWalk` drops rows at or above it, so a caller could recover sequences by bisection even with the fields removed.
- The module keeps every deletion record (no cap) and pages them by position (`modules/retention/reads.go:84-95`); one record per artifact (`runs.go` settles an artifact once; `records.go` refuses a deleted artifact).

# Design

Revised after the independent design review of 2026-11-27 (approve with conditions; three blocking) and the owner's choice of the same day: the window becomes the workspace's own.

1. **No sequence, no watermark in either read.** Rows keep `artifact`, `class`, `run`, `other_scope`, `at`, `decision`. `decisions.watermark`, the tombstone page's `watermark` and every row's `sequence` are removed.
2. **The window is the workspace's own, always.** The server keeps, for the installation's workspace (the only one these reads answer), the last 1,000 deletion records it may see, in installation order (`tombstoneIndex`, `retentiontombstones.go`). Each read first catches the index up from the last position it examined to the newest record, in pages of 1,000 (`retention.Registry.Tombstones`), adding the visible ones and dropping the oldest beyond 1,000; a new registry (restore) or a shorter one rebuilds it from the start. An artifact's tenants never change (`modules/retention/records.go`), so a record's visibility is fixed and the index is never rewritten. There is no scan budget, no edge that moves with other workspaces' deletions and no completeness flag (revised after the second design review, which found that a 10,000-record budget reintroduced the moving installation-wide edge). The catch-up work grows with the records added since the previous read, not with the history.
3. **The order stays newest first**, by the sequence on the server, which is never sent.
4. **The only cursor is a sealed artifact token.** `next` is `sealQueryToken("retention-tombstone", {artifact})`: its body holds the artifact identifier and nothing derived from the sequence, an index, an offset or the watermark (the token is signed, not encrypted, so its body is readable). A further read opens the token, checks the artifact is still visible to the workspace, finds it in the window by its sequence and continues with the records older than it. A raw number or any other value in `after` is refused with 422 `invalid_request` on both reads; the shared `parseKeptQuery` number cursor is no longer accepted by the tombstone read. A token whose artifact has left the window answers 409 `revision_conflict`.
5. **Counts** (`total`, `matching`) count visible records of the window only, as now.
6. **Screen** (`retention/LiveDecisions.tsx`): the sequence column, the watermark fact and the note about skipped sequence numbers are removed; the facts grid and the table's column group shrink by one; rows are keyed and identified by artifact. `useOlderRows` and `newestFirst` (`shared/older-rows.ts`) gain a string key, keeping the numeric key their other callers use. On `revision_conflict` the list resets to the first page and shows the existing reset line, instead of retrying the dead cursor. The window sentence says "up to the last 1,000 deletion records of this workspace".
7. **Known residuals, stated:** the time a read takes grows with the deletion records added since the previous read by any caller of the installation workspace, hidden ones included (microseconds each; at best an order of magnitude, never the exact count the old window gave); and a row with `other_scope: true` still shows that a run of another workspace deleted an artifact this workspace shares, at time `at`. That concerns only shared artifacts and is the existing behaviour.

# Contract

Removing `sequence` and `watermark` from two reads is a public contract change (fields removed). The console is the only consumer in the repository. The owner's decision of 2026-11-27 covers the removal.

# Progress

- Built 2026-11-27: server (`retentiontombstones.go` index and token cursor, `retention.go`), console (parser, `older-rows` string key, P04 decisions view, reset on conflict, window sentence), `tools/records-qa.mjs`, binding fields. `apps/console/src/modules/administration/candidates/P04.json` keeps its planning-stage `watermarkDecision` sample value; it is a design candidate, not the live contract.

# Verification plan

- Server tests: two workspaces' artifacts deleted alternately; the installation workspace's reads contain no `sequence` or `watermark`, their window is its own 1,000 and does not move when the other workspace deletes; paging returns every visible record once, newest first; `after=<number>` answers 422; the decoded token holds only the artifact; a token whose artifact left the window answers 409; the window stays the workspace's last 1,000 when far more hidden records follow; the index rebuilds after a registry change; restore keeps one record per artifact in order.
- Console: parser tests; `older-rows` string-key tests with the numeric callers unchanged; the 409 reset; `tools/records-qa.mjs` rewritten for opaque tokens; all P04 rendered checks rerun.
- Documents: `docs/ui/screen-binding-fields/operations.md`, `docs/ui/screen-binding-register.md`, `apps/console/src/modules/administration/candidates/P04.json`, and the HYB-305 plan's increment 4g and open item, marked superseded by this ticket.
- Independent tenant/permission review of the change; ui-designer check of the P04 layout change.
