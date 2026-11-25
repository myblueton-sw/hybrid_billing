---
type: Plan
title: Server search, filters and cursor paging for O02, O03 and P01 to P05
description: What the seven screens read today, which of their lists can grow, the common shape proposed for paged reads and the order of the increments.
status: draft
---

# Outcome and ownership

HYB-305, machine:mac_mini, 2026-11-25. This document records the present state of seven screens and splits the ticket into increments. It changes no product code. Each increment gets its own design section here, its own reviews and its own merge.

Requirements: UI-04 (`docs/planning/v1.16/ui-design.md:547`) and the bulk table contract (`:410-430`); in each screen's specification the sentences on server search and stable sort and on reading lists at a fixed revision by cursor without loading the whole data set into the browser (`docs/ui/screens/O02/spec.md:61-63`, `:101-103`; `O03/spec.md:63-65`, `:105-107`; `P01/spec.md:62-64`, `:102-104`; `P02/spec.md:62-64`, `:104-106`; `P03/spec.md:62-64`, `:104-106`; `P04/spec.md:73-75`, `:121-123`; `P05/spec.md:61-63`, `:101-103`). The trace of 2026-11-17 lists them as required and missing (`docs/reviews/HYB-301-trace-classification/report.md:317`); P05 was added to this ticket on 2026-11-18.

# Facts read on main 3a387a8

Common to all seven: the main read of the screen takes no search, filter, sort, cursor or limit and returns one document; caps are constants in the code, mostly 50; no screen filters, sorts or searches its main lists on the server. None of the seven screens has a selection of a page or of a whole filter for a bulk action today (the only checkboxes are inside one contract and inside one form), and the proposal of a deletion plan on P04 is not connected.

| Screen | Read | Lists and their bounds | What is said when a list is cut |
| --- | --- | --- | --- |
| O02 | `GET /api/v1/organizations/plans` (`apps/api/server/relplans.go:617-661`) | Closed plans: 200 kept across all workspaces together, so one workspace's closures can push out another's history (`:20`, `:104-118`); at most 50 plans shown, open ones first (`:644-655`). Changed members of a plan: 50 per kind (`:21`, `:568`) | Closed plans: nothing, on the service or the screen. Members: the true count per kind and a truncated mark |
| O03 | `GET /api/v1/contracts` (`contracts.go:148-176`) | Contracts: at most 200 in a workspace, a limit of the module at creation (`modules/contracts/registry.go:216-217`). No command creates a contract; the list is the five seeded ones. The four filters run in the browser (`contracts/live-model.ts:9-13`) | Nothing is cut |
| P01 | `GET /api/v1/installation` (`installation.go:433-450`) | Proposals: closed ones are dropped beyond 200 kept, newest 50 shown (`installseed.go:18-19`, `:59-70`). Releases 32, destinations 16, reports 200 are limits of the module | The service says nothing; the screen says "Showing the newest 50" whenever 50 are shown, also when exactly 50 exist |
| P02 | `GET /api/v1/integrations` (`integrations.go:836-855`) | Deliveries: every delivery of every subscription is read on each request (`:745-758`), then every dead or held one and 50 others per subscription are returned; the module bounds a subscription's deliveries, about 3,200 in all with 16 subscriptions (`modules/webhooks/deliveries.go:62`, `:84-86`). The module pages deliveries by position, and positions shift as deliveries change state. Service accounts: all read, at most 200. Plugin proposals: newest 50. The action history of each subscription: never trimmed, returned whole (`:656-659`) | Deliveries: the subscriptions that were cut are named. Proposals: the screen's "newest 50" note only |
| P03 | `GET /api/v1/recovery` (`recovery.go:441`) | Open calls (module limit 10,000), unknown targets, pending events, each cut at 50 with a total; history per set cut at 50 with a total; lost events and locks returned whole and never capped (`:316-318`, `:354`); proposals, newest 50. The module copies every open call, unknown target and lost event on each read (`modules/recovery/recovery.go:485`) | Totals for the cut lists; proposals: the screen's note only |
| P04 | `GET /api/v1/retention` (`retention.go:860-891`) | Artifacts per class (module limit 10,000,000), holds (10,000), required references, artifacts of a run (100,000), deletion records within a window of 1,000, proposals; each cut at 50 with `total` and `more`. Hold outcomes have a "more" mark and no total. The artifacts in the scope of each hold are returned whole, up to 10,000 for one hold (`:537-541`). Every visible artifact is walked on each request with one lookup each, and again for the references; each run walks every artifact of its plan (`:426-482`, `:588-624`, `:667-675`) | As stated per list |
| P05 | `GET /api/v1/release-evidence` (`evidence.go:703-739`) | Requirements (module limit 10,000; the oldest 50 are shown, `:542`), paths (1,000), builds (10,000), results (1,000,000; all walked and sorted on each request, `:577-603`), profiles (1,000), harnesses (100, 50 shown), proposals. For each requirement shown, the replacements are found by checking every stored requirement (`:460-476`); each path shown evaluates a full readiness check (`:410`) | `total` and `more` on every list |

Further reads that are whole: the parties and billing profiles that every answer of the organizations read repeats, refused above 500 (`organizations.go:16`, `:188-189`), used by O01, O02 and O03; all active members for the choice of a hold's owner on P04 (`apps/api/server/access.go:155-168`); and the member roster read for the approvers on every main read of P04 and P05 (`retention.go:846-858`, `evidence.go:707`).

Existing paged reads, as they are. The users list (`apps/api/server/userfilter.go`, `users.go:126-258`, `userwalk.go`): strict parsing of one value per parameter, a search term with case folding, enumerated filters echoed back, an opaque cursor over a stated order, a page of 1 to 100, a scan budget with a "stopped" mark; no total; no pin, its list version is only echoed; a conflict only when the cursor's person has since departed, any other unknown cursor is an invalid request. The organizations lists (`organizations.go:105-184`): a page of 1 to 99 and a pin of the relationship revision, which is immutable; the parties and profiles in the same answer are read as they are now. Reusable as they are: the parameter parsing, the search rules, the failure answers of a read, and the total-and-more envelope of the release evidence read (`evidence.go:31-53`). The walk itself is specific to each store.

# What the ticket means for each list

- A list bounded by a small limit of its module (O02 open plans 16; P01 releases and destinations; P02 publishers, plugins, installations, subscriptions, types; P03 externals; P04 the seven classes) stays in the main read. Paging it adds nothing.
- A list that can grow gets a read of its own, with the common shape below, and the main read keeps a first page of it so that the screen opens with one request as today.
- A list that is cut without saying so is corrected first, because the screen then states something untrue.
- Every one of the seven specifications asks for server search. Search is therefore part of each list's own read, not an optional extra.

# Common shape of a paged read

One route per list under the screen's own path. Parameters: `after` (an opaque cursor), `limit` (1 to 100, default 50), `q`, and the filters the specification names for that screen that the service can evaluate. Unknown or repeated parameters are refused. The answer carries the rows, `next`, the count of matching rows in the caller's scope where it is known without walking everything, `scan_stopped` with the budget when a filtered walk ended early, the filters as understood, the order in words, and the time the answer was computed.

Order and cursor, by kind of list. None of the seven stores keeps snapshots, and the service's revision is one counter for everything, so a later page cannot be pinned to the state of the first page and must not be refused because something unrelated changed.

- A list that only grows at its newest end and is shown newest first (plans, proposals, holds, runs, deletion records): the cursor is the row's own increasing number. New rows fall above the cursor and do not disturb later pages; they are seen on a reload. When the cursor points below what is still kept, the answer says that older entries are not kept, and that is not an error.
- A list ordered by a value that can change (results by time, artifacts by standing, deliveries by position): the answer says that rows can move between pages. A fixed snapshot of such a list needs work in its module; where an increment does not do that work, it is recorded as a residual of that increment.
- The first page inside the main read and a later page from the list's own read are separate answers and can differ; each carries its own count and time.

The screen side, from the bulk table contract (`ui-design.md:416-419`): the order is stated and no sort is offered that the service does not perform; filters are shown as removable chips with "clear all"; the person is told when a list has to be read again from the start; "Show more" is the paging control on these screens, and the page size is not a choice (recorded as a difference from `:416`, which names a page size choice).

Rights and scope. Each list's read repeats the admission of the section it belongs to; on P02 the sections are admitted separately (`integrations.go:821-831`). Every count covers only rows the caller may see. Every increment adds reads to the public contract and has an independent change review; none adds a command.

A filter named by a specification that the service has no data for is not offered and is recorded as not connected in the binding register. Known today: on P05 the provider and the contract or role; on P01 the profile, check area, severity and validity of a diagnosis, of which almost none applies to the proposal history; on P03 the restore point and the check status; on O02 the dependent pool, while the impact is not computed. Each increment states how the specification's filters map to data; P04's "standing" in particular is this service's word for the specification's deletion stage and hold reason together.

Selection of a whole filter with a manifest on the server (`ui-design.md:421`, and the matching sentence of each specification) is not applicable on these seven screens today, because none has a bulk action; it is recorded as such, and returns with the first bulk action.

# Increments, in order

1. **Histories.** O02 closed plans are cut with nothing said: that is corrected first, with a count and a read for the rest, and the pruning is made per workspace. The proposal histories of P01, P02 and P03 already say that the newest 50 are shown, also when nothing was cut: they get a count, a read for the rest, and the note only when something is cut. Every history says where it ends because older entries are not kept. Search over the plan or proposal identifier and its people.
2. **P05.** Requirements (search, the evidence-state filter, newest first instead of the oldest 50, and the replacements found without checking every requirement for each row), results (paged, without walking all of them on each main read), builds, paths, profiles, harnesses.
3. **P04.** Artifacts of a class (search by identifier, filters by store and by deletion stage and hold reason), holds with the artifacts in their scope, references, artifacts of a run, deletion records, hold outcomes with a total; the main read stops walking every artifact and every run's plan.
4. **P03.** Open calls, unknown targets, pending events, lost events, locks, history; needs a paged interface in the recovery module instead of a copy of everything.
5. **P02.** The action history of a subscription; deliveries of a subscription with filters by kind and state, which needs a stable order in the webhooks module (they are bounded at about 3,200 today, so this comes late); service accounts stay whole at 200.
6. **O02 members of a plan** beyond 50 per kind, and **O03**: the contracts list is complete, five rows today and at most 200, and filtered in the browser; it moves to the server together with a higher limit or a way to create contracts, which is not in this ticket. The parties and profiles read whole (cap 500) is taken here, with the index by kind that the O01 tree needs when the cap is lifted (`docs/features/HYB-320/design.md`, limits).
7. **Member reads on P04 and P05**: the hold owner choice becomes a search; the approver roster is not read on every main read.

Each increment: a design section in this document with the facts of that list, independent design review, any module interface change reviewed on its own, server with tests (pages equal the whole list, the cursor while rows are added and while old rows are dropped, the budget, another workspace, counts in scope, and one run with data at the limit of the module), console with unit and rendered checks in three languages, ui-designer review of the controls and copy, independent change review, binding documents.

# Open for the owner

- Whether the 200 contracts limit of O03 stays and when contracts can be created; until then O03's list is small and complete and nothing is built for it.
- The order of the increments above; another order is equally possible.
- Whether a fixed snapshot is required for the lists whose order can change, which means work in three modules, or whether "rows can move between pages" stated on the screen is acceptable before launch.

# Review

Independent review of this plan (AI, Claude Opus 5.5, by reading the code; nothing executed), 2026-11-25: approve with conditions, all incorporated above. It verified the table cell by cell and corrected it: only O02's closed plans are cut silently (the other three histories carry a note on the screen); deliveries on P02 are bounded; service accounts are capped at 200; P04 does not give a total on every list and returns the artifacts of each hold whole; P03 also copies everything on each read and returns lost events and locks whole; P05's largest walk is the search for replacements; O03 has five contracts and no way to create one. Its blocking finding: the first version pinned later pages to the service revision and treated a vanished cursor row as a conflict, which cannot work on stores without snapshots whose histories drop old rows; the shape now distinguishes lists that only grow from lists whose order can change. Not verified by the reviewer: the size of the installer's installed history, the bound on unknown recovery targets, and the cost of any walk.

# Not verified

No measurement exists of any of the full walks, and nothing here was run against large data.
