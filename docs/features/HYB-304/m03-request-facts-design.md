---
type: DesignPlan
title: M03 request rows without whole-population facts
description: How the bounded requests read builds a request row's eligibility without loading the whole roster, every user, every grant and every delegation, while the rule stays the one the commands use.
status: draft
---

# Scope and ownership

HYB-304 increment C4, machine:mac_mini. Root wrote this on 2026-11-26 from the code on main `f863feae`; nothing was built for it. It answers the condition of the C4b change review ([query design](m03-query-design.md)): before the M03 screen reads requests from `GET /identity/access/requests`, and before C5, a request row must not cost a read of every person and every grant. Permission decisions are not changed by this design; it changes how the facts behind a displayed row are fetched.

# Facts

- `accessFacts` (`apps/api/server/accessrequests.go:28-47`) loads the whole roster (`s.roster`, pages of 100 until the end), every user (`s.users`, the same way) and every grant (`s.admin.Grants()`).
- A request row (`changeRequestRow`, `:395-423`) uses them as follows.
  - `eligible(user)` (`:65-69`): the user's member record and user record, and `holdsAdmin(user)` (`:56-63`), a scan of every grant for an effective `AdministerInstallation` grant in the installation workspace.
  - `EligibleApprovers` (`:410-414`): `eligible` over every member of the roster.
  - `basis` (`:123-`): `person` of the proposer and the subject (member and user records), `ceilingFor` (`:84-97`, a scan of every grant for console-client grants), `equivalent` (`:106-114`, a scan of every grant for the subject's), `covered` (one `Explain`).
  - `mayEnd(holder)` (`:385-393`): a scan of every delegation (`s.admin.Delegations()`) for those the holder gives or receives.
  - `standing` (`:296-`): facts of the approver, and `protectedGrant` (`:300`), which reads a grant by identifier.
  - A grant by identifier: `s.grantView` (`access.go:290-297`) walks every grant on each call; a revoke row calls it up to three times, in `basis` (`:130`), in `standing` and directly (`:402`).
  - `covered` is one `Explain`, which inside the auth module reads that subject's grants and delegations (`modules/auth/decide.go:243`); it is not a server-side population read and is not charged to the read's budget. `can_approve` also reads the ledger block (`ledgerBlocksIncrease`, `:416`).
- The same `accessFacts` serves the request commands (`:199`), the whole read (`access.go:184`), the financial and grant authority facts (`financialauthority.go:27`, `financialpermission.go:35`, `grantauthorityfacts.go:20`, `:70`) and the trust evidence (`trustevidence.go:88`, which reads every member on purpose).
- The auth registry keeps grants and delegations in maps with an identifier order (`modules/auth/registry.go:27-33`); it has no lookup by holder, by client or by action. `identity.Workspace.People` answers up to 256 people by identifier (`modules/identity/people.go:5`, `:21`).

# Design

**One rule, two sources.** The row builder and the helpers it calls stop reading `f.members`, `f.users` and `f.grants` directly and ask a small source instead: a person's member and user records; a grant by identifier; the grants a subject holds; the console client's grants; the delegations a user gives or receives; the users who hold an effective `AdministerInstallation` grant in the installation workspace. `accessFacts` keeps its whole-population source, so the commands, the whole read and the other callers behave as they do today. The requests read uses a bounded source. Because both go through the same helpers, a row cannot be computed by a second copy of the rule.

**Bounded source.**

- *Auth lookups.* The auth registry gains secondary indexes kept in step with `AddGrant` and `AddDelegation`: grant identifiers by subject, by client and by installation action, and delegation identifiers by delegator and by delegate. Reads: the grants of a subject, the grants of a client, the holders of an installation action in a workspace, the delegations of a user; each takes a limit and says whether more remain. The gate passes them through, as it does the page reads.
- *People.* The people named by the rows of the page (subject, proposer, holder, the caller) and the administrator-grant holders are fetched with `People` in batches of at most 256.
- *Eligible approvers.* The eligible set is computed once per call: the holders of the administrator grant, kept when their member record is an effective administrator, their account active and human, the conditions of `eligible`. A row's count is that set less the proposer and, for an added grant, the subject, as today.
- *Bound.* Every lookup is charged to the read's scan budget. If a lookup reports more than its limit (more administrator holders than the budget, or one holder with more grants than the budget), the read answers 503 rather than a row computed from part of the facts. That is a limit of the read, recorded in the API description.

**The instant.** The whole read evaluates its facts at the current time: `Grants()` and `Delegations()` compute `Active` and `Effective` at the registry's clock, and `ceilingFor`, `holdsAdmin` and `mayEnd` use those. The bounded source does the same: every auth lookup (a grant by identifier through the existing `GrantAt`, a subject's grants, the client's grants, delegations, holders) is given the current time, and `facts.now` is the current time, not the cursor's reference time as C4b set it. Only the request's own state and lapse are taken at the reference time, as the API description already says. The equality test in R3 compares both reads at one fixed clock.

**Basis of the cursor.** Unchanged: the access generation, the change registry version, the directory version, the trust hold, the identity subjects version and the ledger block.

# Increments

1. **R1, auth lookups.** The indexes and the four reads in `modules/auth` with the gate pass-throughs. Tests: each read equals a filter of `Grants()` or `Delegations()` over generated registries, after additions and revocations; limits and the more flag.
2. **R2, the source.** First a test that records, over generated states, what `eligible`, `holdsAdmin`, `basis`, `standing` and the row builder answer today; then `accessFacts` and its helpers read through the source, with only the whole-population source, and the same test must still pass, with the whole server package.
3. **R3, the bounded source** and its use by the requests read. Tests: for generated states, every row of the requests read equals the row the whole read gives the same caller; the 503 limit; that the read no longer calls the roster, the user pages, `Grants()` or `Delegations()`.

Then C4c: the screen reads requests, revocations and subjects from the bounded reads.

# Review and risk

`accessFacts` decides who may approve, reject and withdraw. R2 touches that code without meaning to change it, and R3 adds a second source of its facts; each increment has an independent change review with a permission lens, and this design an independent design review. The main risk is a difference between the two sources; the equality test in R3 is the guard, and the commands keep the whole-population source, so a difference could mislead a displayed row but not admit a command.

# Reviewed

Independent design review (role-based AI reviewer separate from the writer, by reading, nothing built), 2026-11-26: approve with conditions. Two blocking findings, taken into this revision and resolved on a bounded re-review: the grant-by-identifier read was a whole-population read missing from the facts, and the instant at which the bounded source evaluates was not stated. One condition taken: R2 gets its own before-and-after test. The reviewer read that grants and delegations are only added and closed, never removed or re-keyed, and that the registry has no restore path (`AddGrant`, `AddDelegation`, `RevokeGrant` and `RevokeDelegation` in `modules/auth/registry.go`), so the indexes cannot go stale; that the eligible set computed from the administrator-grant holders is the set `eligible` gives over every member; and that nothing here needs an owner decision. Not verified by the reviewer: how `Explain` scans beyond the cited line; realistic counts of administrators and console-client grants.

# Unknown

- Whether any caller of `accessFacts` other than the trust evidence needs every member; to be read in R2.
