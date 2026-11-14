---
type: ImplementationPlan
title: Disclosure policy versions with semantic allowlists and inference guards, one projection for every channel, bound delivery packages and expiring links
description: Defines the first increment of work item W-27, a register that classifies fields, keeps dual-controlled disclosure policy versions with per-role allowlists that always exclude unclassified fields and refuse forbidden inference combinations, projects rows to a recipient within its customer scope with source locators replaced by public references, produces the same projection for every delivery channel, binds delivery packages to input, policy, recipient, projection and file digests, and serves links that expire and stop on recipient revocation or policy supersession, with an escaped HTML preview.
status: draft
---

# Scope and ownership

[HB-229](https://linear.app/hybrid-billing/issue/HB-229) starts work item W-27 (requirement HB-27; detail contracts DB-05, DB-06, DB-07; HB-27-V) after HB-228 (`modules/exports`). Base: main `ef83738`. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviews the plan and the change, because the register decides what customers and partners can see. Machine: mac_mini.

Out of scope, recorded as residuals: PDF rendering, mail and portal delivery themselves, ERP attachment transport, binding the HB-227 ERP delivery gate and the `modules/exports` customer artifact to disclosure packages, HTTP cache and content security headers, recipients combining files kept from older revisions to complete a forbidden combination, key rotation, upstream authentication of system channels, a staff register through `modules/auth` (until then `recipient_actor` only catches identifiers already proposed as recipients and does not undo earlier staff actions), streaming and chunk manifests at scale, the payment-term views (`modules/receivables`), persistence and screens.

# Observed baseline

- HB-27-D: MSP source files, internal settlement files and customer usage and billing files are different products; customer disclosure builds a new file from semantically allowed fields and excludes new unclassified fields; public fields, recipient rights, inference-combination conflicts and direct-link scope must be defined.
- HB-27-I: ExportPolicyVersion, ExportJob, ExportArtifact, DeliveryPackage; input, policy, recipient and file hashes bound together; public references instead of source locators; the same projection for portal, API, AI, PDF, mail and ERP attachments.
- HB-27-V: byte originals preserved; zero leaks of customer rows, fields, file names or caches; CSV-to-document amount reconciliation; policy change, permission revocation and link expiry; summary plus paged detail instead of a full preview, while full amount and leak checks are recorded.
- DB-06: HTML previews defended against injection; renderers make no external requests.
- `modules/exports` produces settlement and customer artifacts with byte digests; `modules/erpadapters` gates automatic ERP delivery. No module holds disclosure policies, projections, packages or links.

# Decision

A new package `modules/disclosure` follows the sibling conventions: injected clock, one mutex, workspace isolation, `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`), `ErrLimit`, check order form, lookup, repeat, rule, room; an identical repeat by the same human returns the view; assistant actors are refused on every call. Only the standard library is used. `NewRegistry(Config{Keys, Random}, Limits, clock)` takes a key provider that returns a per-workspace secret (never stored in code, documents or logs) and a randomness source (`crypto/rand` in production, a fixed reader in tests).

Staff operations (`Classify`, combinations, policies, grants, revocations, `Project`, `Package`, `Link`, `Resolve`, `ProjectionOf`, internal previews) refuse any actor identifier that has ever been proposed as a recipient in any workspace of the registry (`recipient_actor`); `Project`, `Package` and `Link` also accept system channels, and a grant cannot name its proposer or confirmer; a proper staff register through `modules/auth` is a residual.

Nothing is cached: every call computes from the register's current state. HTTP `Cache-Control: no-store` for portal and API delivery is a residual.

## Field catalogue

`Classify(workspace, Field{Name, Class, Locator}, actor)` records a field once: an identifier name, a class (`public`, `customer`, `internal`, `secret`) and whether it is a source locator. A locator field must be `public` or `customer`. A classification never changes. The field `customer` is classified like any other field and is projected only when allowlisted. Fields that are not classified are never projected.

## Forbidden combinations

`ProposeCombination` and `ConfirmCombination` (a human other than the proposer) maintain a workspace register of forbidden combinations: sets of 2 to 8 classified fields that together allow inferring something hidden. Confirming a combination that an active policy violates for any role suspends that policy at once (`policy_suspended` on projection and access) until a new revision is confirmed.

## Policy versions

`Propose(workspace, Policy{ID, Roles}, actor)` proposes a revision with, for each recipient role (`customer`, `partner`, `auditor`), an allowlist of classified fields. Refusals at `Propose` and again at `Confirm`: `unclassified_field`; `class_not_allowed` (`internal` or `secret` for `customer` or `partner`, `secret` for anyone); `inference_conflict` when a role's allowlist contains every field of a confirmed forbidden combination. `Confirm(workspace, policy, revision, actor)` activates it by a human other than the proposer (`self_approval`, `stale_proposal`). A new active revision supersedes the previous one.

## Recipients

`Grant(workspace, Recipient{ID, Role, Customers, Policy}, actor)` proposes a recipient grant: a role, a non-empty customer scope and the one policy its projections use. `ConfirmGrant` activates it by a human other than the proposer. A later grant for the same identifier is a new grant revision; packages issued under an earlier revision stop (`recipient_changed`). `RevokeRecipient` ends a recipient for good; the identifier can never be granted again. Projection, packaging and access take only the recipient identifier and read role and scope from the active grant; unknown or ungranted identifiers are `ErrNotFound`, revoked ones `recipient_revoked`.

## Projection

`Project(workspace, Request{Policy, Recipient, Rows}, actor)` projects input rows (the policy must be the recipient's granted policy, `policy_not_granted` otherwise) (maps from field name to text with a non-empty `customer` value; anything else is `ErrInvalid`) for one recipient under the active revision of the policy (`policy_superseded` is impossible here, `policy_suspended` refuses):

- a role with no allowlisted fields is refused (`no_fields`); rows of customers outside the recipient's scope are dropped; only allowlisted fields of the recipient's role are kept; unknown fields are dropped;
- locator fields (only when allowlisted) are replaced by a public reference `ref:` plus 24 hex digits of HMAC-SHA256 under the workspace secret over the recipient and the value, so references are stable across policy revisions, unlinkable between recipients and not reversible without the secret; two values that map to the same reference in one projection are refused (`reference_collision`); an internal map from reference to recipient and locator is kept per workspace for `Resolve(workspace, reference, actor)`, and a reference that would map to a different recipient or value anywhere in the workspace is refused; workspace keys must not be rotated in this increment (rotation is a residual);
- output rows are sorted by their canonical encoding.

The register records the projection: workspace, recipient and grant revision, policy and revision, the input digest it computes over the canonical input rows, the projected rows and the projection digest (label `disclosure_projection_v1`) over all of these. The caller receives the recipient view (projected rows, field list, projection digest); dropped row and field counts go only to the internal audit view `ProjectionOf`. Projection takes no channel, so every channel that delivers through a package presents the identical projection.

## Delivery packages and links

`Package(workspace, projection digest, actor)` accepts only a recorded projection of this workspace and creates a package with a generated opaque identifier. `Package` re-checks that the policy revision is still active and the grant unchanged. Its single file is the safe viewing copy rendered by this register from the stored projection as CSV (UTF-8 with a byte order mark, header of field names, rows in projection order, risky header names and values neutralised with the HB-228 rule: one leading apostrophe unless the value is a canonical decimal) and named `<package>.csv`; exact machine values are not delivered to recipients; caller-supplied files, names and extensions are not accepted. The package binds the projection, input and file digests, the policy revision and the recipient grant revision.

`Link(workspace, package, lifetime, actor)` issues a new link with a token of 128 random bits on every call; only its digest is stored, and the token is returned once. Lifetimes are 1 hour to 30 days; access at or after the expiry is refused (`link_expired`).

`Access(token, actor)` resolves the workspace from the link and returns only the recipient view of the package (identifier, file name, file digest and bytes) when the actor is the package's recipient (`recipient_mismatch` otherwise; a system channel acting for the recipient uses the recipient's identifier; authenticating that the channel serves that recipient happens upstream and is a residual) and the link is live. The recipient is checked before the link state. It is refused when the link is expired or revoked (`link_revoked`), the policy revision was superseded (`policy_superseded`) or suspended (`policy_suspended`), or the recipient grant changed (`recipient_changed`) or was revoked (`recipient_revoked`). An unknown token is `ErrNotFound`. AI consumers receive only `Access` output through the system channel.

Delivery on every channel must present a package accepted by `Access`; binding the HB-227 ERP delivery gate and the `modules/exports` customer artifact to disclosure packages is a residual.

## Preview

`Preview(workspace, projection, page, size, internal, actor)` renders an HTML fragment with no attributes of any kind and only the elements `section`, `p`, `table`, `tr`, `th` and `td`: a summary (row count and field list, plus dropped counts only when `internal` is requested by staff). A recipient preview needs the projection's recipient or staff (`recipient_mismatch` for another recipient), and for the recipient the same policy and grant re-checks as `Access` apply; the internal variant needs staff (`recipient_actor` for a recipient) and one page of rows (size 1 to 500). Field names and values are escaped with `html.EscapeString`, and C0 and C1 controls and bidirectional formatting characters are replaced by U+FFFD. The fragment makes no external requests because it has no attributes; a content security policy and sandboxing belong to the serving layer (residual).

## Limits

`Limits{Workspaces, Fields, Policies, Recipients, Projections, Packages, Links, Rows per projection, Combinations}`, each with a fixed maximum.

# Tests

- Leaks: rows outside the granted scope, internal and secret fields, unclassified and new fields, locators (only references appear), dropped counts in recipient views and previews, and file names never leak; partner and auditor scopes come only from grants.
- Inference: a policy allowing a forbidden combination is refused at proposal and confirmation; confirming a combination suspends a violating active policy and its packages.
- References: unlinkable between recipients, stable across revisions, resolvable internally, collisions refused.
- Access: wrong recipient, other workspace tokens, expiry at the boundary, link revocation, policy change, grant change and recipient revocation stop access; a package for an unrecorded projection is refused.
- Preview: script, attribute, URL, entity, bidirectional and control payloads; paging; recipient and internal summaries.
- Review 2 rules: `recipient_actor` on each staff operation and grants naming their proposer or confirmer; `policy_not_granted`; CSV neutralisation leaving canonical decimals unchanged; `Package` refusing a superseded or suspended policy or a changed or revoked grant after projection; reference clashes across recipients in the workspace; two links per package with their own tokens and expiries; one recipient previewing another's projection.
- Forms, limits, repeats, isolation, a property test of projection against an independent model, race, 386 build, gofmt, vet, coverage and mutation testing.

# Provisional choices

- The classes, roles, combination sizes, link expiry range, reference length and preview page size.
- Dual control for policies, combinations and grants; any human may revoke links and recipients.
- Links are bound to the recipient identifier, not bearer links.
- Projection works on text values; amount reconciliation between CSV and documents uses `modules/exports` digests and is an integration check.

# Review history

- Design review 1 (independent security reviewer, read-only): FAIL with 3 high, 6 medium and 3 low findings. Folded in: a dual-controlled recipient register with terminal revocation and grant revisions; recorded projections with a computed input digest, packages only from recorded projections and files rendered by the register; random link tokens stored as digests, recipient-bound access, workspace from the link and the expiry boundary; keyed references unlinkable between recipients with collision refusal and internal resolution; a separate dual-controlled combination register that suspends violating policies; dropped counts internal only; an attribute-free escaped preview with control and bidirectional characters replaced; generated package identifiers and fixed CSV files; parity as a gate with the ERP and exports bindings as residuals; no caching; AI through the system channel; row and locator rules.
- Design review 2 (same reviewer): CONDITIONAL PASS; all 12 findings resolved, one high and two medium new findings plus five low, folded in: recipients cannot act as staff (`recipient_actor`); grants bind one policy (`policy_not_granted`); the package CSV is a neutralised viewing copy; upstream channel authentication, key rotation and cross-revision combination as residuals; workspace-wide reference uniqueness; `Package` re-checks state; every `Link` call issues a new link.
- Design review 3 (same reviewer): CONDITIONAL PASS with one medium and three low findings, folded in: recipient previews limited to the recipient or staff; tests for the review 2 rules; the ordering limit of `recipient_actor` stated with its residual; all residuals in one list.

# Change review resolution

Change review 1 (a separate independent security reviewer, read-only): FAIL with 1 high, 2 medium and 6 low findings. All were addressed:

1. High: a recipient preview skipped the access re-checks; it now applies the policy and grant checks, tested after supersession, grant change, suspension and revocation.
2. `Project`, `Package` and `Link` are staff or system operations; recipients get `recipient_actor`, which closes the reference oracle and forged deliveries.
3. Recipient identifiers are tracked registry-wide, so a recipient of another workspace cannot act as staff.
4. `Access` returns only the recipient view.
5. `Access` checks the recipient before the link state.
6. A role without allowlisted fields is refused (`no_fields`).
7. CSV header names are neutralised.
8. Combinations have a limit.
9. Revocation records its actor and time without rewriting earlier revisions.

Change review 2 (same reviewer): PASS; all findings resolved and no new leak path. Kept as residual: system channels are accepted by identifier until a staff and channel register exists in `modules/auth`.
