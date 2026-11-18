---
type: Verification
title: HYB-322 legal entity member refusal verification
description: relationship.propose now refuses a legal entity as a member of any relationship and an internal unit as a member of any kind other than the organization hierarchy, narrowing the HB-274 member rule after the HYB-318 finding.
status: stable
tags: [HYB-322, O02, relationships, HB-274]
sources:
  - resource: ../HB-274/plan.md
  - resource: ../HYB-318/verification.md
---

# HYB-322 verification (root, 2026-11-18, main 026000d)

## Requirement and evidence

- HYB-318 (R8) found that `relationship.propose` accepted a legal entity as a hierarchy member: a probe with `sample-le-north` sealed with targets `sample-le-south` and `sample-unit-south-web`.
- HYB-318 decided (R8) that a legal entity is not a hierarchy member and removed the O01 action on legal-entity nodes; HB-170 (`plan.md:39`, `:46`) defines payer and pool members as accounts; the seed uses units as members only in the hierarchy.
- HB-274 plan line 209 recorded party-kind ordering for hierarchy as residual H1.
- The seed's relationships (`apps/api/server/orgseed.go:53-77`) use internal units as members only in the hierarchy (unit → legal entity) and source accounts as members of every kind.

## Change

- **Server.** `checkIDs` (`apps/api/server/relplans.go`) refuses `invalid_request [member]` when the member is a party that is a legal entity, is not active, or is used with any kind other than `hierarchy`. Source accounts are unchanged; an active internal unit stays a valid hierarchy member. Targets are unchanged. This narrows the HB-274 public rule "member must be a known party or a member of any kind" (`HB-274/plan.md:129`), recorded there as an amendment.
- **Console O02.** Member choices never offer a legal entity, offer active internal units only for the hierarchy (or before a kind is chosen), and accounts for every kind (`changes/live-model.ts` `memberChoices`, kind passed from `Live.tsx`).

## Tests

- `TestProposalRefusals`: legal entity in the hierarchy, legal entity as a payer member, unit as a payer member, unit in an allocation pool, suspended unit as a hierarchy member — each `invalid_request [member]`.
- `TestRejectedPlansAndViewGroups`: a legal entity under its own unit is refused; a cycle is still built through the server with two units (north-ops under north-data activated, then north-data under north-ops) and is rejected with a cycle violation.
- The timeline fault on the target lookup is reached with a counting fault (the first timelines call passes, the second fails); the member-lookup fault keeps its own case.
- Console: `memberChoices` and `allMemberChoices` tests per kind.

## Checks

`gofmt` clean, `go vet ./apps/api/server` ok, `go test -count=1 ./apps/api/server` ok with every function at 100.0% (per-function profile); `tsc` clean; `npm test` 747/747.

## Review

Independent change review round 1: FAIL (M1 O02 choices still offered parties; L1 false cycle claim; L2 target fault branch untested; L3 member state; L4 residual; nit). All folded in above.

## Residual

- A unit may still target another unit in the hierarchy (allowed by the organizations model); deeper hierarchy rules beyond party kind remain with `modules/relationships` validation.
- An existing invalid party membership could not be ended through O02, because the member check applies to every action; unreachable today (relationships are in memory and the seed has none). Revisit if relationships persist.
- Hierarchy targets keep the HB-274 rule (any active party); unit-to-unit chains are bounded by the relationships module's depth and cycle rules.

Independent change review round 2: PASS with L5 (console offered a suspended unit from member listings) and two nits, all fixed: party members that are not active are skipped in both loops (test with a suspended unit), the duplicate fault case name renamed, and the residual list formatted.

