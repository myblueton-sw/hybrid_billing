---
type: DesignOptions
title: How an issuance step holder reaches the O03 setup screens
description: Verified facts, options and a recommendation for letting a person who holds issuance step permissions but not installation administration use the O03 documents and payment terms views.
status: draft
---

# Outcome and ownership

HYB-348, machine:mac_mini. This document puts a decision before the owner. It decides nothing and changes no product code; it adds one server test that pins the facts below. Any option that changes who may read contract or organization data is a tenant and permission change and needs independent design review before implementation.

# The problem

Issuance setup steps are designed for separated people: a protected settings change needs five distinct people (the channel's registrar and admitter, and a requester, verifier and approver who are none of them, `modules/protected/changes.go`). The O03 contract screen is the only place that sends these steps, and today only installation administrators can open it. The sample installation has three. A change therefore cannot be completed through the screen without creating more administrators, and whether administrators should hold step permissions at all is an open owner residual of HYB-341.

# Verified facts

Read from main at 7c815f5 on 2026-11-24. The first three rows are pinned by `TestIssuanceStepHolderReadsSetupButNotContracts`, which passes.

| Fact | Evidence |
| --- | --- |
| A member who holds the protected settings read permission and the payment terms read permission, and no installation administration, is refused `GET /api/v1/contracts`, `GET /api/v1/contracts/{id}` and `GET /api/v1/organizations` with 403 | test; `Server.admitted` in `apps/api/server/users.go:96` requires the installation administration action |
| The same member can read `issuance_setup.context` and `issuance_setup.configurations` for a contract identifier | test; permission in `financialissuancesetupcontext.go:151` is the read action on the protected settings kind |
| The same member can read `protected`, `protected_channels`, `protected_changes`, `attestations`, `references` and `terms` for a confirmed configuration | test |
| The workspace Administrator role does not carry installation administration; in the sample installation it is granted by the seed to three people | real-server run recorded in the HYB-341 handoff; `seed.go` `grantInstallation` |
| The contract screen shows "Contracts are not available to you" when the contract list read is refused, before any setup view is rendered | `apps/console/src/modules/organizations/pages/contracts/Live.tsx:119` |
| The documents view takes the contract identifier, the State-on date and the party list; the payment terms view takes the contract identifier and the date only | `LiveDetail.tsx:57`, `:60` |
| Channel, change, statement and payment terms step commands need only their step permission plus the read permissions. Proposing and confirming a configuration and the protected rules require installation administration by design | `financialissuancesetupmutation.go:147`, `financialissuanceprotectedwrites.go:28`; HYB-341 journey test, where administrators perform those four and a member and four further people perform the rest |
| No setup read needs more than the two read permissions, including the option list and action reads | `financialissuancesetupoptions.go:10`, `financialissuancesetupaction.go:22`, `trusthold.go:255` |
| A protected settings reader can already read, for any contract identifier in the workspace, the contract name, the billing profile identifier and name, the buyer, payer and issuer identifiers, the currency and the revisions. The permission is workspace-wide, not per contract. An unknown identifier answers `permission_denied` and an inactive one `prerequisite_failed` | `financialissuancesetupcontext.go:180` to `:220`, `financialissuanceprotectedaccess.go:106` |
| The two views call only the session read and `issuance_setup` commands; a refused organization read is tolerated and names fall back to identifiers; the revision that step writes send comes from setup reads | `setup/client.ts:55`, `Live.tsx:107`, `live-format.tsx:20`, `Documents.tsx:47`, `PaymentTerms.tsx:42` |

So the service already lets a step holder perform every step of the two views that is not reserved to installation administrators. The obstacle is that the console reaches those views only through the contract list and detail.

# Options

1. Setup entry by contract identifier, client only. When the contract reads are refused and the address names a contract, the console renders the documents or payment terms view for that identifier from the setup reads alone, without the contract list, the contract header facts or party names. No server change, and no value is shown that the person cannot already read; what changes is that the setup context of any contract in the workspace, described in the facts above, becomes reachable by a link instead of only by calling the command. The contract name comes from the setup context read. Costs: the person needs a link or the identifier from someone else, since nothing lists contracts for them; the party names in the settings section fall back to identifiers; the settings and rules write forms remain administrator-only and must be hidden or shown as unavailable; the default State-on date comes from the contract list today, so the entry needs another source for it.
2. A narrow list for setup readers. Add a read, under the protected settings read permission, that returns the contracts of the workspace that have a confirmed issuance configuration, with identifier and display name only. Adds discoverability to option 1. Costs: it discloses contract names and the existence of configurations to every setup reader; it is a new public read and a permission decision; it needs paging and the usual snapshot rules.
3. A contract read permission. Introduce a grantable permission to read contracts and organizations without installation administration, and require it of step holders. Solves the general problem that only administrators can read contracts. Costs: the largest change; it widens what non-administrators see across O01 to O03; needs its own provisioning area, labels and review.
4. Step holders are administrators. Keep the screens as they are and state that step permissions are granted only to installation administrators. No code change. Costs: a change needs five installation administrators, which concentrates administration in more people in order to separate duties; it decides the open residual about governors holding step permissions by default rather than on its merits; the sample installation cannot complete a change through the screen.

Two further options were not developed: an entry from "My work" or a list of pending steps limited to contracts where the person is a named actor, and a read grant per contract. Both would add discoverability with less disclosure than option 2 and both are new server reads or permissions.

# Recommendation

Option 1 first, as the smallest change that removes the dead end without altering any permission, and option 2 as a separate decision once the owner has seen option 1 in use. Option 4 should be rejected explicitly if it is not chosen, because it is what the product does today by accident. This is root's reading and not a decision.

# What option 1 would need

- A design of the entry state: what the page shows in place of the contract list and header, how the identifier is presented, the wording that tells the person they are seeing setup views only, and the behaviour of the other tabs of O03, which stay unavailable.
- Rendered English, Korean and Arabic checks, the ui-designer role for the composition, terminology review, and the owner's screen review.
- An extension of the real-server run so that the steps are performed by people without installation administration, which would replace the seed helper the run uses today.

# Not checked

- Every use the documents view makes of the party list and of the contract detail beyond its props; the feasibility of option 1 is established for the reads, not for each rendered field.
- Whether a contract identifier is itself considered confidential to a setup reader.
- Navigation: how the main menu behaves for a person who can open only this entry.

# Review

Independent design review (role-based AI reviewer separate from the writer, by reading, nothing executed), 2026-11-24: FAIL with three findings — the document said step commands need only their step permission although proposing and confirming a configuration and the rules are reserved to installation administrators; it said the contract would be shown by identifier although the setup context read returns its name; and "no new disclosure" omitted what a setup reader can already read for any contract identifier in the workspace. Corrected with the reviewer's wording and references. Judged to hold: the other facts read from code (the HYB-341 handoff record and the internals of the payment terms write checks were not checked), the count of five people for one change with one approval, the test as not vacuous, the description of options 2 to 4 and the recommendation. Corrective re-review of the three findings: PASS.
