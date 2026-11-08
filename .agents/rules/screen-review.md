# Screen design, review and fixed revisions

The owner established this workflow on 2026-11-01 under HB-63. It applies to individual product-screen designs and their interactive previews. AGENTS.md, the ticket workflow and the owner's instructions remain authoritative.

## Design unit and implementation IDs

- Design each screen's actual layout, information hierarchy, fields, controls, tables, interactions and relevant states. A whole-story wireframe, route inventory or generic renderer is not a substitute for that screen design. Related-screen links provide context without making approval an all-product decision.
- Retain the existing canonical screen ID and registry/baseline mappings. Give every planned implementation item a stable ID scoped to that screen, such as `setup-connectors:field:provider` or `setup-connectors:action:review-plan`. Cover displayed data, input validation, actions, permission/error/empty/loading states and navigation behavior where applicable. Do not renumber items when their display order changes or reuse retired IDs for a different requirement.
- Map each item to its purpose, screen location, expected behavior, relevant states and implementation/verification status. Show the ID with its screen-design annotation or implementation checklist; runtime product users do not need design bookkeeping mixed into normal tasks.
- Keep design review status separate from implementation status. A pending backend integration may remain explicitly deferred when its screen behavior and boundary are settled. Unresolved design blockers must be resolved or explicitly excluded from the owner's acceptance scope before FIX.

## Per-screen review and FIX

- Every reviewable screen must expose a consistently located, keyboard-accessible `검수 확인` control, localized as `Confirm screen review` in English. Display the screen ID, exact design revision and current review status beside it. Include it on detail, dialog and state variants when they are separately identified review units; a reusable dialog may instead be an item of its owning screen with that scope made explicit.
- Use `Draft → Ready for review → FIX` for an accepted revision. Only the owner's explicit confirmation of that screen and stated scope can produce FIX. The assistant, a delegated reviewer, passing tests, a build or JEV must never confirm on the owner's behalf. Automated tests may exercise isolated test records but must not create real owner acceptance.
- Before confirmation, expose the revision, item checklist, included locale/layout/state variants and any deferred items. Confirmation applies only to that declared scope. Do not infer that reviewing one language, viewport or state accepts every variant or every related screen.
- Record the screen ID, immutable revision/source reference, accepted item IDs and variants, owner confirmation evidence, actual confirmation time and deferred/excluded items. Keep a ticket-linked, Git-reviewable acceptance record. Never invent reviewer identity or confirmation evidence.
- A browser-only mark is a local review record until its evidence is reconciled into the canonical acceptance record. A failed save must remain visible and must not report persisted FIX. The UI must state its actual storage boundary; localStorage alone is not authenticated owner identity or durable project acceptance.
- Surface per-screen review status in the screen index so the owner can distinguish unreviewed, fixed and change-pending screens. Do not convert route coverage or all passing tests into an acceptance count.

## Changes after FIX

- Preserve the accepted revision as a baseline. Do not modify a fixed screen's layout, copy, interactions, item meaning or approved states without an owner-authorized change. An explicit owner change request is already authorization for that scoped revision; do not request duplicate permission.
- Record the affected screen/item IDs, reason and impact before changing them. Mark the new candidate `Change pending`, retain the old acceptance evidence, then return the candidate to `Ready for review`. Only renewed owner confirmation fixes the new revision; never carry FIX forward merely because the screen ID is unchanged.
- Shared tokens, components, text or contracts can affect multiple fixed screens. Assess the actual design impact, identify affected screens and invalidate acceptance of the changed candidate scopes. A dependency-only edit with no impact on accepted design does not by itself require owner reapproval; record that impact assessment. Do not silently rewrite approved screens or erase their prior acceptance records. Use a source revision or fingerprint that covers actual screen dependencies, not just its route name.
- Apply final-document cleanup only to the specifically accepted replacement scope. Update links and generators, remove truly superseded maintained copies in that finalization change and preserve distinct evidence/history under the existing cleanup rule. Fixing one screen does not retire an unreplaced module or the entire prior baseline.
- FIX means screen-design acceptance only. It grants no production readiness, financial approval, permission enforcement, merge, deployment, message or payment authorization. The normal implementation, independent review, QA and authorized integration workflow still applies.
