---
type: VerificationRecord
title: HYB-331 M01 Membership tab role cell without a repeated No membership, verification
description: The change to the M01 Membership tab role cells, the HYB-294 visual-spec §10 amendment, the [TW] "No role" copy and the checks run with their observed results. Written by the implementer; no change, render or technical-writing review has seen this build.
status: stable
tags: [HYB-331, HYB-294, verification, M01, membership]
sources:
  - resource: ../HYB-294/visual-spec.md
  - resource: ../HYB-323/verification.md
---

# Scope

- Branch `fix/HYB-331-m01-membership-role-cell`, worktree `/private/tmp/hb331`, on main `e5f7751`. Nothing is committed. Observed on 2026-11-19 on the Mac mini (darwin/arm64).

# Change

- `Live.tsx`: on the Membership tab, the Workspace role cell (L) and the role line of the Role and period cell (M, stacked) render "—" (`aria-hidden`) plus a visually hidden "No role" when the row has no membership. The membership status cell keeps "No membership". Users and Invitations tabs and the row detail are unchanged. `Table` is exported for the render test.
- `live-copy.ts`: `noRole` — EN "No role", KO "역할 없음", AR "بلا دور" [TW].
- `docs/features/HYB-294/visual-spec.md` §10: dated HYB-331 amendment paragraph appended; earlier text kept.
- `docs/ui/terminology/product-glossary.md`: row `no-role-cell` [TW].
- New `table-render.test.tsx`, added to the console `npm test` list.

# Checks

| Check | Result |
| --- | --- |
| `table-render.test.tsx` before the fix (copy key and export present, cells unchanged) | 1 of 3 failing: Membership role cell actual "No membership", expected "—" + hidden "No role" |
| `table-render.test.tsx` after the fix | 3/3 |
| Console `npm test` | 796/796 |
| Console `npm run build` (tsc + vite) | passes |
| Browser QA: `qa323.mjs` phases S, F (ports 4571/8901, API built from this worktree) | S 22/22; F 696/696, 0 FAIL lines and 0 lines naming No membership / 소속 없음 / بلا عضوية (HYB-323 round 2 on main before this change: F 684/696 with 12 such failures) |
| Browser QA regression: `qa294.mjs` phases S, X, F, G, W (ports 4572/8902) | S 22/22; X 27/27; F 1497/1497; G 70/70 with 2 page errors `net::ERR_FAILED` (the same 2 recorded by HYB-294 r6 and HYB-323 round 2); W 33/33 |

# Residuals

- KO and AR "No role" are AI drafts pending native-language review.
- No change, render or technical-writing review has been run.

## Root final verification (2026-11-19)

- Render and copy review (ui-designer role): PASS; optional AR consistency change adopted: "بلا دور" (matches بلا عضوية and بلا تاريخ انتهاء) in copy, test and glossary.
- Root re-ran after rebasing on current main: tsc clean, `table-render.test.tsx` 3/3, npm test 796/796, build ok, OKF status test ok.
