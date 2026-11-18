---
type: Verification
title: HYB-295 verification of the screen binding register catch-up
description: How each pre-launch service binding entry for L01-L03, M01-M05, O01-O03 and P01-P07 in the screen binding register was checked against Git history, the feature records, the code at main 82c9272 and Linear, with the inconsistencies found and what remains unverified.
status: stable
tags: [HYB-295, pre-launch, screen-binding, register]
sources:
  - resource: ../../ui/screen-binding-register.md
  - resource: ../HB-257/plan.md
  - resource: ../../reviews/HYB-291-planning-cross-check/report.md
---

# Scope

[HYB-295](https://linear.app/hybrid-billing/issue/HYB-295) brings the [screen binding register](../../ui/screen-binding-register.md) up to date for the pre-launch service bindings, as `HB-257/plan.md:177-178` requires ("records which screen fields come from which module read or command, extending `docs/ui/screen-binding-register.md`") and [HYB-291](../../reviews/HYB-291-planning-cross-check/report.md) finding M6 recorded as missing. Documentation only; no code was changed.

- **Base.** Branch `docs/HYB-295-binding-register` on main `82c9272`, checked on 2026-11-18 on mac_mini.
- **Writer.** An AI agent (Opus 5.5) delegated by root wrote the register section and this record.
- **Review.** Round 1 independent review (root's reviewer): FAIL with docs-only findings F1–F8; it found all code citations it checked correct. This record and the register were revised for F1–F8 (see Review round 1). Re-review is pending. Nothing here is owner acceptance.

# Placement

The new section is appended after the HB-23 sections. Other records cite this file by line, for example `screen-binding-register.md:70` in `HB-274/verification.md:103`, `HB-275/verification.md:94`, `HYB-300/plan.md:439, 671` and `docs/reviews/HYB-291-planning-cross-check/report.md:56`. Changes inside lines 1–81 were made in place:
- the frontmatter `title` and `description` lines;
- the `sources` list, rewritten as a YAML flow sequence on the same six lines (6–11), adding `HB-257/plan.md` and the HYB-291 report (parsed with PyYAML);
- intro lines 16, 18, 20 and 22, now limited to the HB-15 and HB-22 sections;
- the heading "Residuals", renamed "HB-23 residuals" on the same line.

Lines 1–81 keep their numbering, and line 70 is still the O02 row. No other file links to an anchor in the register (`grep 'screen-binding-register.md#'` found none). The register has 292 lines. The per-field maps are split by screen group into `docs/ui/screen-binding-fields/` (`sign-in.md`, `identity.md`, `organizations.md`, `operations.md`), each OKF with status draft, and each screen entry links to its section.

# How each column was checked

| Column | Method |
| --- | --- |
| Binding tickets and PR | `git log --merges --oneline` from `96fd245` (HB-257) to `82c9272` maps each ticket branch to its PR number. `git log --no-merges` lists the commit subjects. `git show --name-only <commit>` gives the console page directories each follow-up changed. The table below lists the follow-up commits checked. |
| Reads | Routes are taken from `apps/api/server/boundary.go:21-59`. Client methods are taken from `apps/console/src/service/admin.ts:818-1103` and `api.ts`. Call sites come from a grep of `api.<method>(` in each page directory, excluding tests. A script checked that every cited `boundary.go` line contains the route it names: no mismatch. |
| Commands | Command IDs come from a grep of quoted command IDs in each page directory. Where a page builds the ID by concatenation (`users/Live.tsx:51`, `service-accounts/Live.tsx:234`), the IDs come from the action lists (`users/live-model.ts:34-47`, `service-accounts/Live.tsx:191`). Server definitions come from a grep of `Definitions() []commands.Definition` in `apps/api/server`. The `CommandId` union is at `admin.ts:621`. |
| States | Taken from the binding commit subjects and the Scope, Results and Residuals sections of each verification record. They were not re-run. |
| Not connected and residuals | Taken from the Residuals and "Planning cross-check residuals (HYB-291)" sections of each record and from the HYB-291 report. |
| Open tickets | Linear GraphQL was queried read-only through `scripts/linear_workspace.py` on 2026-11-18, for team HYB issues from number 286 up (title and state) and for the descriptions of HYB-304 and HYB-305. Title searches for P06, P07, L01, L02, L03 and "binding register" were also run. The Linear MCP server answered 401, so the script was used. |
| PR numbers | A script parsed `git log --merges --oneline 96fd245..82c9272`, which lists 46 ticket merges. It matched every ticket-and-PR pair in the register against that list: 120 pairs, no mismatch. |
| Field maps | The `sign-in.md` and `organizations.md` maps were written by the HYB-295 writer, `identity.md` and `operations.md` by two delegated writer agents, all from the parsers in `apps/console/src/service/` and the render lines of the live pages. Server JSON tags were compared with the parser keys for the organizations, plans and session reads. Cited lines were re-opened with `sed -n`: about 90 lines in the sign-in and organizations maps, with 14 corrected after the first check. The identity agent re-opened 40 of its cited lines (25 console, 15 server) and the operations agent 62 (25 console, 37 server). Both reported every re-opened line correct after corrections. Their remaining citations were not re-opened one by one. All four files parse as YAML, every relative link resolves, and every register anchor matches a heading. |
| Owner acceptance | No record claims owner FIX acceptance. Every record that mentions it lists it as a residual, so the register marks every screen pending. |
| P06, P07 | `apps/console/src/app/screen-groups/operations.tsx:21` limits the live switch to P01–P05, and `:92-95` select the preview components. A grep for `fetch(` and `api.` in `modules/administration/pages/demo` and `licenses` found none. `boundary.go` has no demo, simulation or licence route. |

# Follow-up commits checked

| Commit | Ticket | Console pages changed (from `git show --name-only`) |
| --- | --- | --- |
| `b709bfc` | HYB-293 | installation, integrations, recovery, retention, access, authentication, my-security, service-accounts, users, changes, contracts, organizations, and `service/` |
| `d36246c` | HYB-298 | retention (plural helper moved to `shared/plural`), access, users (copy) |
| `a08bc8e` | HYB-297 | access, authentication |
| `6caa506` | HYB-299 | users, `service/identity-bulk.ts` |
| `1b7d3ae` | HYB-296 | changes |
| `4fee31d` | HYB-300 | organizations |
| `bfba901` | HYB-318 | changes, organizations, screen groups |
| `ef92914` | HYB-319 | changes, organizations, screen groups |
| `4a675b7` | HYB-322 | changes |
| `e093859` | HYB-312 | retention |
| `649a906` | HYB-325 | retention |
| `d7084c7` | HYB-315 | release-evidence, screen groups |
| `3ea012a` | HYB-291 | `service/copy.ts` (L03 `forgotBody`, notice) |
| `de8c82b`, `f5503a3` | HYB-292, HYB-302 | no console page; listed under the common entry (HYB-292) or not at all (HYB-302 concerns the FOCUS reader proxy) |

HYB-321 changed tests only. HB-258, HB-261 and HB-262 bind no screen in this set and appear only in the common entry.

# Results per screen

| Screen | Binding found | Checked against |
| --- | --- | --- |
| L01–L03 | yes (HB-259, HB-260; L03 copy HYB-291) | `Service.tsx:155, 211, 235`, `api.ts:71, 107-134`, `boundary.go:23, 55-59`, HB-260 verification |
| M01 | yes | `users/Live.tsx`, `live-model.ts`, `RolePeriod.tsx`, `BulkPrepare.tsx`, `RevocationJobs.tsx`, `identity-bulk.ts`, `admin.go:471`, `bulkstage.go:98` |
| M02 | yes | `authentication/Live.tsx:96`, `TrustConfirm.tsx:64`, `trust.go:331` |
| M03 | yes | `access/Live.tsx`, `Requests.tsx:77`, `access.go:340`, `accessrequests.go:426` |
| M04 | yes | `service-accounts/Live.tsx`, `services.go:379` |
| M05 | yes | `my-security/Live.tsx:89, 172, 268`, `boundary.go:35, 48-49` |
| O01 | yes | `organizations/Live.tsx`, `drafts.tsx`, `groupdraftcmds.go:353` |
| O02 | yes | `changes/Live.tsx`, `relplans.go:453` |
| O03 | yes | `contracts/Live.tsx`, `LiveActions.tsx:170`, `contracts.go:753` |
| P01–P04 | yes | each page's `Live.tsx` and `LiveCommand.tsx` or `LiveProposals.tsx`; `installation.go:547`, `integrationcmds.go:389`, `recoverycmds.go:178`, `retentioncmds.go:382` |
| P05 | yes (HYB-290, HYB-315) | `release-evidence/Live.tsx:109`, `LiveCommand.tsx:164`, `evidencecmds.go:216` |
| P06 | **no** | `operations.tsx:21, 92-93`; no route; HB-257 phase table |
| P07 | **no** | `operations.tsx:21, 94-95`; no route; HB-257 phase table |

# Review round 1

| Finding | Change |
| --- | --- |
| F1 per-field map | Added under `docs/ui/screen-binding-fields/` and linked from each bound screen entry (L01–L03, M01–M05, O01–O03, P01–P05). |
| F2 plan citation | Now `HB-257/plan.md:177-178`. |
| F3 O03 server citation | "action hints at `contracts.go:521-531`, command handler `lifecycleCommand` (`contracts.go:757`)". |
| F4 modules | M03: gate (auth), accesschange, identity, decisionledger. L02: passwords, identity (`signin.go:18-21` imports). |
| F5 HYB-293 on L screens | L01, L02 and L03 all list HYB-293 (#210) for the shared `api.ts` and `Service.tsx` change, with no step-up on the sign-in routes (`stepup.go:30-39`). |
| F6 intro scope | Lines 16, 18, 20 and 22 are limited to the HB-15 and HB-22 sections, on the same lines. |
| F7 line-70 citers | The HYB-291 report `:56` is added to the list above. |
| F8 wording and sources | "M screens" for `identity.tsx`; the bulk commands are listed in line order (execute `:139`, prepare `:224`, discard `:244`); the KO and AR statement names the tickets whose residuals record it; `HB-257/plan.md` and the HYB-291 report are added to the register sources. |

# Inconsistencies found (not fixed here)

1. **P05 field map.** `HYB-315/plan.md:1176` says its `verification.md` records the P05 field-to-read and action-to-command map as input for HYB-295. `HYB-315/verification.md` has no such section. The register entry for P05 was written from the code.
2. **P05 search and filters have no owning ticket.** `HYB-290/verification.md:56` and `HYB-315/plan.md:76` say P05 search, filters and paging are tracked in HYB-304 and HYB-305. In Linear, HYB-304 covers M02–M05. HYB-305 covers O02, O03 and P01–P04, and its description says "P05 later with its screen". The P05 screen has merged, so no ticket owns P05 search and filters.
3. **HB-257 phase table.** `HB-257/plan.md:168` names only HYB-290 for P05. The P05 screen ticket HYB-315 and the phase 2–5 follow-ups (HYB-293, HYB-296 to HYB-300, HYB-312, HYB-318, HYB-319, HYB-322, HYB-325) are not in the table.
4. **Status values.** The OKF rule (`.agents/rules/knowledge-format.md`) allows `draft`, `stable` or `deprecated` for `status`. 37 of the 38 verification records cited by the register use `status: final`; HB-289 uses `stable`.
5. **Superseded residual lines left in place.** Later tickets superseded several residual lines, but those lines were not marked:
   - `HB-260/verification.md`, "Administrator-issued resets": superseded by HB-263/264; HYB-291 adds a note.
   - The HYB-291 residual in `HB-264/verification.md`, "the screen shows bulk revocation as not available": superseded by HYB-299.
   - `HB-265/verification.md`, "Trust flow simplified": HYB-297 closes it, as `HYB-297/verification.md` records under Planning cross-check.
   - `HB-267/verification.md` and `HB-266`, "grants apply directly": superseded by HYB-298.
   The register states the current behaviour. These lines are records of their time; whether to annotate them is for root.
6. **Copy against behaviour (recorded elsewhere).** HYB-293 records that the M05 preview copy still says "reauthentication". The register repeats this; it is not new.
7. **HB-23 rows not re-checked.** The HB-23 rows for M05 (FA-13), P02 (FA-13), O03 (EI-01, SP-07, SP-08) and P06 (SR-03) were not checked against the bound screens. Only the O02 allowance-preservation row has evidence: it is not addressed (`HYB-300/plan.md:671`).

8. **Found while mapping fields (O screens).**
   - O01: the organizations read casts `parties` and `profiles` to typed arrays without checking each field (`admin.ts:988-989`). Other reads in the same client check each field.
   - O02: the proposal form reads the member, kind and target choices from `GET /api/v1/organizations`, page by page for every kind in the browser, up to 10 pages of 99 members per kind (`changes/Live.tsx:22-23`, `:40-60`). This is a client-side full load. UI-04 forbids it, and HYB-305 (O02 server search) is open.
   - O03: the list filters run in the browser over the loaded contracts (`contracts/Live.tsx:146`, `:233`). HB-281 records that server-side search beyond 200 contracts is out of scope.

9. **Server fields the console does not read (identity and operations maps).** Each field map ends with a list of these. In summary:
   - M02 ignores `trust.stricter_fields`.
   - M01 ignores bulk-job fields: `selection`, `impact.api_keys` and `impact.pending_jobs`. The preview shows fixed "not applicable" copy instead.
   - The access, services and me parsers cast row arrays without checking each field.
   - P01 drops `site.config_overrides` and reduces history and backups to counts.
   - P02's clients view never checks `clients.state`.
   - P05 shows fixed "not connected" copy instead of the gate-profile `failures` it parses.

   These are recorded for the code owners; no code was changed.

No mismatch was found between the cited code lines and the binding commit subjects.

# Not verified

- No build, test or browser run. The states come from the binding tickets' own records.
- The states and residuals of each screen were not compared with the full screen specs. Most specs are local companions outside Git.
- Linear titles and states are data read on 2026-11-18 and may change. "Open ticket against a screen" is matched by title or description only.
- Independent review: round 1 FAIL (F1–F8 fixed), round 2 PASS with five Low/Nit citation and wording items, fixed by root.
