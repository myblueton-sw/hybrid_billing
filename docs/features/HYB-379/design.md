---
type: DesignPlan
title: HYB-379 — list filters, page size and page position in the address for M02, M03 requests and revocations, and M05 operations
description: Which list state each remaining M02, M03 and M05 list keeps in the address, under which names, and how Back, Forward and reload restore it.
status: draft
---

# Scope and ownership

[HYB-379](https://linear.app/myblueton/issue/HYB-379), machine:mac_mini, root (Claude). Written 2026-11-27 from main `e3c4eef9`. Rule: HYB-304 plan, common contract item 6 — browser query state follows the canonical route; apply, clear, page size, next and previous, and history restoration preserve normalised filters.

# Facts

- Already in the address: M05 sessions (`session`, `sq`, `ssince`, `suntil`, `slimit`, `safter`), M04 lists (`svc_a_*`, `svc_c_*`), M03 permissions (`acc_*`).
- In component state only: the M02 filter, page sizes and both cursors (`authentication/TrustLists.tsx`, `access/paged.tsx`); M03 waiting and ended requests and revocations (`access/paged.tsx`); the M03 person picker; M05 operations and each operation's result rows (`my-security/SessionOperations.tsx`).
- All identity screens write the address through one helper, `toParams` in `app/screen-groups/identity.tsx` (replace, or push for history entries). M02 is not given it today.
- `usePagedList` (`access/paged.tsx`) keeps `after` in state, keyed by the list key and a generation, with a cursor trail for Previous.

# Decisions

1. **Main lists keep filters, page size and cursor in the address.** Pushing a history entry on Apply, chip removal, Reset, a page size change and First/Previous/Next; Back and Forward restore the list exactly; reload asks for the same page.
2. **Names** (short, screen-prefixed, as the existing lists):
   - M02: `tc_q`, `tc_state`, `tc_ol`/`tc_cl` (open and closed page size), `tc_oa`/`tc_ca` (open and closed cursor).
   - M03 requests and revocations: `req_ea`/`req_el` (ended requests cursor and size), `req_wa` (waiting cursor), `rev_a` (revocations cursor).
   - M05 operations: `op_a` (cursor).
3. **Normalisation.** A value that does not parse (search longer than allowed, unknown state, size outside 1 to 100, cursor longer than 4096) is dropped when read, and the list falls back to its default; nothing invalid is sent. A changed filter or size always clears that list's cursor in the same write.
4. **Previous after Back, Forward or reload.** The cursor trail lives in memory and is cleared whenever the address cursor changes by anything other than the list's own paging; Previous then goes to the first page (the same rule as M04 and M05).
5. **A reload of the screen's data resets position.** When M02 rereads its policy (after a command or Reload), both cursors are removed from the address (replace, not a new history entry), because a continuation taken before the change may no longer fit. A default page size is written as no parameter. This includes returning to the policy view from another M02 view, which rereads the policy: that history entry comes back on the first page.
6. **Hidden lists.** In the narrow step view the lists stay mounted and hidden; Back can change a list the user does not see, and the open detail stays open. Accepted: the address describes the list, not the panel.
7. **Stay in component state, recorded:** the M03 person picker (a field inside the request form; the form itself is not in the address) and the result rows nested inside an M05 operation card (opened on demand). Their position is not part of the screen's canonical route.
8. **Mechanism.** `usePagedList` gains an optional address binding `{ after, go }`; without it the hook behaves as today. M02 receives `params` and `onParams` like M03, M04 and M05.

# Increments

- U1: M02 (filter, both sizes, both cursors).
- U2: M03 requests and revocations.
- U3: M05 operations.

Each: unit tests for parse and normalisation, rendered checks including Back and Forward, independent change review, all M02–M05 checks rerun.

# Progress

- U1 (2026-11-27): M02 done. Independent review approve with two conditions (trail after Back/Forward; cursor after a reread), both taken and re-reviewed. m02-lists 206 with Back/Forward/reload/invalid-address checks.
