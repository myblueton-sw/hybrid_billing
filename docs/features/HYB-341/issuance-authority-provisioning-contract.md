---
type: ContractDesign
title: Public provisioning of issuance setup authority
description: Public governed commands that grant the protected, payment-term and draft capabilities which only test helpers can grant today.
status: draft
---

# Outcome and ownership

HYB-341, machine:mac_mini. This is the provisioning part of item 5 of [the handoff](handoff.md). Three authority domains already exist and are enforced by every issuance setup command: protected setup, payment terms and issuance drafts. Each has a registry, a version, a grant prefix, closed capabilities and a two-governor grant protocol. None has a public command, so every capability used since the transfer was granted by a package-internal test helper. This contract exposes the same six provisioning commands that the document and result authority domains already have, for these three domains, through the existing shared handler. It adds no capability, no grant, no default and no new protocol.

Root owns the design and integration. An independent reviewer covers the permission boundary, because these commands hand out the rights that the protected, payment-term and draft commands rely on. The [result authority plan](result-authority-plan.md) is the reviewed precedent for the same shared handler and is not repeated.

# Verified starting facts

Read from main at b064071.

- `newServerGrantAuthority` builds `protectedAuthority`, `termAuthority` and `draftAuthority` in the sample installation with a one-hour proposal lifetime, a 24-hour maximum grant validity and a capacity of 1000, and `grantAuthority` returns each by domain.
- Their closed capabilities are: protected setup `register_channel`, `admit_channel`, `request_change`, `verify_change`, `approve_change`, `inspect`; payment terms `record`, `approve`, `bind`, `approve_binding`, `inspect`; drafts `submit`, `approve`, `inspect`. Each maps to one gate action on the domain's resource kind, and each `inspect` to ViewCost with the domain's evidence field.
- `grantAuthorityDefinitions` builds `propose_console`, `propose_human`, `approve`, `reject`, `withdraw` and `inspect` for a domain from one template, under the administer-installation permission, with `inputAuthorityCommand` as the handler. It has a prefix for six domains and returns nothing for these three.
- For the document, result and two financial request domains the handler, in this order, refuses a missing authority, requires a transport budget of 20 fields and 1024 bytes, calls the authority's governor caller check and its current-governor check, performs the effect and marks the outcome with a `historical_decision` receipt. The current-governor check requires a human, active, effective administrator; without this branch `inspect` runs only the caller check. `financialPermissionBoundary` runs the same two checks at the boundary for those domains' commands.
- The shared registry enforces the decision protocol: the proposer cannot approve; a recipient can neither propose, approve nor reject their own grant; an uncertain grant write is retained as unknown and recovered only by the same actor repeating the same rationale and digest. The server adapter, not the registry, enforces the grant shape, at proposal time and again at apply time: the boot workspace, a console ceiling whose recipient is the console client, a human recipient who is an effective active human member, and a ceiling from the same domain's own records that is active, has the same capability and contains the grant's validity. The existing `protectedShape`, `termShape` and `draftShape` checks already close the final grant shape, so no new shape check is needed.
- The generic grant paths refuse the dedicated actions and resource kinds of all three domains.
- Hold rules for the existing provisioning commands refuse the two proposals and the approval, and allow rejection and withdrawal as reductions of pending work.
- Bulk revocation impact enumerates pending provisioning proposals per domain through `grantAuthorityTally`; a test requires every proposing namespace to be enumerated.
- Each domain has its own fault prefix for the grant write and impact fault points.
- The result domain also has `history` and `context` reads that list its proposals. They are specific to that domain. The other domains with public provisioning have only `inspect` by proposal identifier.

# Commands

Three namespaces, each with the six commands of the template: `issuance_protected_authority`, `issuance_term_authority` and `issuance_draft_authority`.

| Command | Target | Fields | Mutating |
| --- | --- | --- | --- |
| `propose_console` | new proposal identifier | `capability`, `not_before`, `not_after`; rationale | yes |
| `propose_human` | new proposal identifier | `capability`, `not_before`, `not_after`, `recipient`, `ceiling`; rationale | yes |
| `approve`, `reject`, `withdraw` | proposal identifier | `digest`; rationale | yes |
| `inspect` | proposal identifier | none | no |

- The static permission is administer installation, as for every existing provisioning command.
- The three namespaces join the document and result domains in the handler and in the boundary: the governor caller check and the current-governor check run at the boundary, and the handler runs the transport budget and both checks again before any effect, and the outcome carries the `historical_decision` receipt. The boundary is keyed by literal command names, so all eighteen names, six per namespace, are added to `financialPermissionBoundary`; the boundary runs before command execution and therefore before a cached replay.
- `capability` must be one of the domain's closed capabilities; the registry refuses anything else. A capability of one domain cannot be granted through another namespace, because each namespace reaches only its own registry.
- Everything else is the shared handler's existing behavior and is unchanged: instants in RFC 3339, validity bounded by the registry limits, the proposal identifier as the command target, the digest that approval, rejection and withdrawal must name, the refusal mapping, and the retained unknown state of an uncertain grant write.

## Hold, step-up and impact

- Hold: for each namespace the two proposals and the approval are refused and rejection and withdrawal are allowed, exactly as for the existing provisioning commands. There is no general read rule: reads are allowed during hold only by a literal list of command names, and an unlisted command is refused. `issuance_protected_authority.inspect`, `issuance_term_authority.inspect` and `issuance_draft_authority.inspect` are added to that list, as the inspection commands of the existing provisioning namespaces are, so that a governor can read the digest that the allowed rejection and withdrawal must name. The exhaustive hold test gains fifteen mutation rows, from 180 to 195.
- Step-up applies as to every mutating command.
- Bulk revocation impact gains three pending stores, `issuance_protected_authority_proposals`, `issuance_term_authority_proposals` and `issuance_draft_authority_proposals`, each counted through the existing tally with the rechecked flag the other provisioning stores carry. Each needs an M01 label and a glossary row in English, Korean and Arabic, delivered with the console change of this item together with their terminology review; rendered verification remains with the owner. A proposal left in the unknown state is counted for its proposer although only its approver can recover it; this is the existing behavior of every provisioning store and fails closed.

# What this does not do

- It does not list proposals. An approver needs the proposal identifier and digest, which `inspect` returns for a known identifier. A history read like the result domain's is needed before an M03 screen can show pending proposals for these domains; it is the next increment of this item and has its own contract.
- It does not seed, default or renew a grant. A grant lasts at most 24 hours in this boundary and a proposal one hour; a long-running operator journey must re-provision, as the mature-channel sequence already shows.
- It does not change who may govern. Installation governance decides, and a governor cannot grant themselves a capability.
- It does not revoke. A human grant keeps its existing explicit revocation path. A console ceiling has no public revocation; it ends by expiry within 24 hours.
- It does not label these grants on existing access screens. See the residuals.
- It does not weaken the generic grant exclusions or let a step capability, Inspect or administrator standing substitute for a provisioned grant.

# Owner-visible residuals

- A step capability granted here lasts at most 24 hours. Channel maturity alone takes 24 hours, so the people who registered and admitted a channel must be re-provisioned before a change can be requested through it, and the later writes recheck them.
- Provisioning is all-or-nothing per capability for the workspace. The grant terms cover every customer, legal entity and contract; no narrower scope exists in these domains.
- Without a list read, an approver must be told the proposal identifier.
- Once real grants of these domains exist, the existing M03 human-grant rows and P02 console-ceiling rows have no labels for the eleven issuance actions, three resource kinds and three evidence fields, and fall back to raw identifiers. That the grants appear in those rows is inferred from the result domain and not yet verified. The labels, their terminology review and rendered verification are HYB-346 and are not part of this change.
- A governor can receive a step capability when two other governors act. Nothing in provisioning prevents one person from holding both sides of a separated flow, such as requester and verifier or recorder and approver; the separation that matters is enforced per object by the modules, which refuse the same person acting twice on one channel, change, term, binding or draft.
- Proposal records are never pruned in this boundary. Re-provisioning every 24 hours consumes the capacity of 1000 per domain until restart, and pending proposals of one governor can exhaust it.

Decisions for the owner, with the current behavior as the default until answered: whether governors may receive step capabilities; whether one person may hold both sides of a separated flow; whether the 24-hour maximum validity stays; whether an expiry-only console ceiling is acceptable.

# Alternatives not chosen

- One namespace with a domain field. The template builds per-domain commands, the boundary and hold tables are keyed by command name, and a domain field would let one command name carry three different authorities.
- Reusing an existing namespace such as the document authority for these capabilities. The domains are separate registries with separate versions and grant prefixes by an earlier reviewed decision.
- Adding the history read in the same change. It is a new disclosure with its own bounds and belongs to its own review.

# Required verification

Command HTTP tests, with every grant in them created through these public commands and no package-internal grant helper.

- for each of the three namespaces and each of its capabilities: a console ceiling proposed and approved by two governors, a human grant proposed and approved, and the gate then allowing the granted person that capability's action where it refused before, with no grant appearing in another domain;
- real commands succeeding only after the public grant, at least a terms read, a protected read, a registration attestation and `record_term`, with a capability that was not granted still refused. The capabilities that need a matured channel (`request_change`, `verify_change`, `approve_change`) and `approve_binding` are exercised by real commands in the unseeded sequence instead;
- protocol refusals through the public commands: the proposer approving, the recipient proposing, approving or rejecting their own grant, forged wire fields on `propose_console` (`recipient`, `ceiling`, `action`, `kind`, `workspace`), the closed final grant shape, a human grant without an approved ceiling, a ceiling of another capability or another domain, validity beyond the ceiling or the registry maximum, an unknown capability, a capability of another domain, a stale digest, approval of a rejected or withdrawn proposal;
- authority: a non-governor, a governor whose standing was revoked, an ended session, the reduced transport budget, a changed governor before a cached replay, and an installation without the sample seed returning unsupported capability for all six commands of each namespace with no grant created;
- hold: proposals and approval refused, rejection, withdrawal and `inspect` allowed, for each namespace;
- uncertain grant write: the domain's fault point reached through `approve`, the unknown state retained, recovery by the same governor, refusal for another;
- impact: a pending proposal in each namespace counted for its proposer in bulk revocation impact, failing closed on the impact fault;
- generic paths: the generic grant command still refusing each dedicated action and resource kind;
- compatibility: existing private-helper tests unchanged and passing; the proposal digest produced through the public command equals the one the helper produces for the same input in a second identical deterministic installation, since the digest covers the clock instant, the governance generation and the ceiling version;
- the unseeded sequence: starting from a sample installation with no issuance grants, provisioning through these commands and then running the public rules, channel, attestation, change and terms commands for one field and one binding, with the clock advance and re-provisioning stated. The sequence needs three governors and more distinct holders than the sample's four humans, so members are created by the existing helper, which grants no issuance capability; governors who propose rules are provisioned protected `inspect`; after the 24-hour advance every actor signs in again, because the session lasts 12 hours and step-up 15 minutes, and new ceilings are approved before new human grants;
- hold-table and proposing-namespace completeness.

# Limits

Synthetic in-memory boundary only. The M03 screen for these namespaces, the history read and the HYB-346 labels are not part of this change. No implementation, test run or review result is claimed by this draft.
