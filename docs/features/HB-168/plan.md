---
type: ImplementationPlan
title: Operating profiles and party model
description: Defines the first domain model of work item W-01, which separates installation, workspace, legal party, billing profile and contract transaction path, with pure rules for issuance path, billing eligibility and aggregation, held in an in-memory registry with expected revisions.
status: reviewed
---

# Scope and ownership

[HB-168](https://linear.app/hybrid-billing/issue/HB-168/w-01-operating-profiles-and-party-model-installation-workspace-party) is work item W-01 (requirement HB-01) with the HB-01 share of detailed contract AR-01. It is the second increment built under the owner's instruction of 2026-11-12. Base: main `451cd57`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, as tenant boundaries and billing responsibility require. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-12:

- Requirement HB-01: MSP, single enterprise and group use one core. Installation, secure workspace, legal entity, organization, account and the buying, paying and issuing parties are distinct; a single enterprise needs no virtual MSP. The proposed direction separates Installation, Workspace, Party, OrganizationNode, Contract and BillingProfile; an operating profile selects menus and default procedure only and never grants permission, tax or billing responsibility; external issuance and internal chargeback are separate business paths. Acceptance: the three profiles run on the same core, internal settlement needs no external invoice, and ineligible sums across legal entities or currencies are blocked.
- Detailed contract AR-01: a contract revision carries the direct or resale transaction path, per-component payment responsibility and the MSP's eligible billing scope. A directly contracted CSP usage charge is a basis for viewing and allocation and is distinct from the MSP's service fee; selling calculation and approval must block rebilling usage for which the MSP has no payment responsibility. Acceptance AUD-AR-01: the same CSP input run as direct contract shows only the service fee, and as resale only approved eligible selling components.
- The shared domain contract lists Installation, Workspace, Party and BillingProfile with immutable IDs, operating type, issuing and buying parties and state, and the invariant that workspace, legal entity, group and payer are distinct. References use canonical IDs and revisions, never names.
- Existing code has only installation, workspace and tenant UUID scoping in the FOCUS migrations and the Go `customeraccess` store (companies, tenants, subjects with versions and an in-memory registry). There is no party, billing profile or contract model.
- The business database is not selected (PostgreSQL is the preferred candidate) and `go.mod` has no dependency. The existing Go module keeps state in memory behind expected versions.
- Open decisions of HB-01: the pilot operating type, legal entity, buyer and payment responsibility, and the first selling or internal-settlement mode. Organization relationships and moves are W-02, permissions W-03, contract lifecycle and periods W-15, rate calculation W-16.

# Decision

One Go package, `modules/organizations`, pure domain logic over the standard library, with an in-memory registry. No storage engine, HTTP, command or permission check is introduced. The proposed repository layout places billing profiles and contract revisions under `modules/contracts`; they are kept beside the parties for now because every write validates its party references in the same atomic step and there is no storage layer to share between two packages. Moving them is a rename when W-15 builds the contract lifecycle.

This ticket does not complete W-01. The design task HB-01-D needs the owner's decisions (pilot operating type, legal entities and responsibilities, first selling or internal-settlement mode), and the verification task HB-01-V and the acceptance cases GH01, GH02, GH07 and AUD-AR-01 remain NOT_RUN.

## Entities

| Entity | Fields and rules |
| --- | --- |
| Installation | One registry serves exactly one installation, named at construction |
| Workspace | Immutable ID, operating profile (`msp`, `single_enterprise`, `group`), state (`active`, `suspended`). The profile is a label for menus and default procedure: no rule branches on it |
| Party | Immutable ID, workspace, kind (`legal_entity`, `internal_unit`), state. An internal unit names one legal entity of the same workspace as its parent; kind and parent never change. `internal_unit` is provisional: departments are organization nodes in the planning documents, and W-02 will reconcile this flat parent pointer with them |
| BillingProfile | Immutable ID, workspace, issuer, buyer, payer, optional document recipient, issuance path (`external_invoice`, `internal_chargeback`), one currency (three uppercase letters), state. Issuer and buyer differ. Everything except the state is fixed at creation; a different issuer, buyer, payer, recipient, path or currency is a new profile. For `external_invoice` issuer, buyer and payer are legal entities, the payer and a recipient, if named, are not the issuer, and a recipient is a legal entity; a paying agent or a recipient never changes the buyer. For `internal_chargeback` issuer and buyer resolve to the same legal entity, and there is no payer and no recipient, since an internal allocation carries no payment obligation and no external invoice or payment demand; an internal statement is a later document concern |
| ContractRevision | Contract ID, revision number, workspace, opaque provider reference, customer (a legal entity), optional seller (a legal entity other than the customer), transaction path (`direct`, `resale`), and components. A component has an ID, a kind (`provider_charge` or `seller_service`), for a provider charge the party responsible for paying the provider, and a `seller_billable` flag giving the contract's eligible scope. Revisions are append-only and numbered from one without gaps, and every revision of a contract keeps the first revision's customer and provider reference; a revision carries no effective or recorded time, state, approval or notion of "current", which W-15 adds |

The customer's obligation to pay the issuer lives only on the billing profile's payer. The contract's responsibility field means responsibility toward the provider and nothing else.

Registration rules for a contract revision: a provider charge names the customer or the seller as responsible; under `direct` every provider charge is the customer's responsibility and is not seller-billable; a seller-billable provider charge requires the seller to be the responsible party; `resale`, and any seller service, requires a seller. The holder of the provider contract is thereby the customer under `direct` and, for a billable charge, the seller under `resale`.

IDs use the alphabet of `modules/allocation` (letters, digits, `.`, `_`, `:`, `-`, at most 128 bytes) and are compared bytewise.

## Rules (pure functions)

| Rule | Behavior |
| --- | --- |
| Issuance requirement | `external_invoice` requires an external invoice; `internal_chargeback` requires none. The answer depends on the billing profile's path only |
| Selling eligibility (AR-01) | For a contract revision, a component and an issuer, for external selling only, checked in this order: an unknown path gives `invalid_contract`; an issuer that is not the contract's seller gives `not_seller`; an unknown component gives `unknown_component`; a component with an unknown kind, a provider charge without a responsible party or a service that names one gives `invalid_contract`; a provider charge under a direct contract, or one whose responsible party is not the issuer, gives `no_payment_responsibility`; a component outside the contract's eligible scope gives `outside_eligible_scope`; otherwise `eligible_pending_approval`. The function takes a caller-supplied value and therefore fails closed on anything the registry would not have stored; the registration rules are an additional defence. Callers must pass a revision read from the registry. Internal allocation or chargeback of a party's own provider charges is not evaluated by this function, is never rebilling, and belongs to allocation (W-14) |
| Document key | A billing profile resolves to (workspace, issuer legal entity, buyer legal entity, payer, currency, issuance path), with an internal unit resolved to its legal entity. Lines may be combined into one billing document only when their keys are equal; an empty request, a key without workspace, parties or currency, an unknown path, an external key without a payer or a chargeback key with one is rejected. This guard is for billing documents: a cross-entity group summary is a report, not a sum, and is outside it. Period, tax and ERP route will extend the key later. The function returns eligibility only and adds no amounts, so it needs no numeric profile |

## Registry

`Registry` holds workspaces, parties, billing profiles and contract revisions in memory behind one mutex. Workspaces, parties and billing profiles carry a version and are written with an expected version; a mismatch is a conflict. Contract revisions are append-only with the expected next revision number. Every lookup is keyed by workspace and ID, and a reference to a record of another workspace is indistinguishable from a reference to a record that does not exist. A write into a suspended workspace, or one that references a suspended party or an internal unit whose legal entity is suspended, is rejected; existing records stay readable. The one exception is suspending an existing party or billing profile without any other change, which is always possible, so that dependents of a suspended party can themselves be suspended; suspension does not propagate by itself. A required reference that is empty or malformed is an invalid record, not a missing one. Reads return copies. There is no delete. Each write, including its reference checks, is one atomic step, which is the unit a database transaction will later replace; persistence keys will be (installation, workspace, ID), as in the existing FOCUS authority tables.

The registry is not an authorization source. It accepts workspace IDs from its caller and checks only consistency. No API, command or screen may expose it without a server-established installation, workspace and principal context, which is W-03. The same-workspace check is not customer isolation: inside an MSP workspace nothing here stops one customer's profile from naming another customer as payer; that boundary is also W-03.

## What is not decided here

The pilot operating type, real legal entities, payment responsibilities, the first selling mode, tax treatment, the component catalogue, and the database. Provisional choices made so that code can exist, each to be confirmed or replaced: the `internal_unit` party kind; internal chargeback limited to one legal entity; one currency per billing profile; the two states; no delete; the document key's fields. Settlement between two legal entities without an invoice (for example between group companies) is undecided and is rejected with a reason of its own rather than allowed or classified as departmental chargeback. Tests use synthetic parties.

# Allowed files

`modules/organizations/*` (new); `docs/features/HB-168/*`.

# Verification

- Table tests with literal expectations:
  - billing profiles: external invoice between two legal entities, with a third-party payer, with a separate recipient; internal chargeback between units of one legal entity and from a legal entity to its own unit; rejected: invoice to oneself, an internal unit as issuer, buyer, payer or recipient, a missing payer, chargeback across legal entities (undecided), chargeback with a payer or recipient, a party of another workspace, an unknown party, invalid currency, path, ID or workspace;
  - the operating profile is only a label: identical party and profile data registered under each of the three labels give identical outcomes, and a single enterprise settles internally with no seller party. This shows independence from the label, not that three operating profiles have been run;
  - parties: kind and parent rules, immutability of kind and parent, stale versions, suspension, references to suspended parties and workspaces, cross-workspace reads;
  - contract revisions: the registration rules above with each violation, append-only numbering, an earlier revision unchanged by a later one, and stored and returned copies;
  - selling eligibility: AUD-AR-01 at rule level, with the same component list under a direct revision yielding only the seller's own service and under a resale revision the seller-paid, in-scope charge and the service; every reason code;
  - document key: units resolve to their legal entity; equal keys combine; a different currency, buyer, payer, issuer or path, an empty request and incomplete keys are rejected.
- A test with 64 parallel writers and one expected version under the race detector: exactly one succeeds. It shows the in-memory check-and-set only.
- Mutation check: temporary single-point changes to the eligibility conditions, the registration rules, the chargeback and payer checks, immutability, copying, revision order and key comparison must each fail the tests.
- `go vet ./...`, `gofmt -l`, `go test -race ./...`.
- Independent review of the actual change.

Not verified by this ticket: persistence, authorization, customer isolation inside a workspace, any real customer structure, the full AUD-AR-01 (which needs selling calculation and approval), GH01, GH02 and GH07, scale targets and projections.

# Deferred

Database selection and persistence of this model; permissions and delegation (W-03); organization nodes, relationship revisions and moves (W-02); contract lifecycle, periods and proration (W-15); selling calculation that consumes the eligibility decision (W-16); commands and API (W-33); the O01 to O03 screens.

# Design review resolution

The independent design review of 2026-11-12 (read-only, nothing executed) returned FAIL without a blocker: 3 HIGH, 6 MEDIUM, 4 LOW. Resolution:

| Finding | Resolution |
| --- | --- |
| F1 eligibility rule not evaluable and conflating responsibilities | Contract gains an optional seller and component kinds; the rule is driven by provider payment responsibility and the eligible scope, for external selling only; internal allocation is stated as outside it |
| F2 buying party missing | Billing profile has issuer, buyer, payer and optional recipient |
| F3 undirected payment responsibility | The contract field means responsibility toward the provider; customer-side payment lives on the billing profile |
| F4 inter-company settlement hard-coded | Rejected with its own reason and listed as undecided; no payer on chargeback; wording about external documents corrected |
| F5 weak aggregation key | Document key of issuer and buyer legal entities, payer, currency and path; scoped to billing documents; no amounts added |
| F6 placement and `internal_unit` | One package with the reason recorded; `internal_unit` labelled provisional |
| F7 tenant-security statements | Stated in the registry section; wrong-workspace and nonexistent references are indistinguishable |
| F8 ticket cannot close W-01 | Stated; provisional choices listed |
| F9 seed for W-15 | Append-only revisions with one numbering concept and no time, state or approval |
| F10 negative cases | Added |
| F11 ID alphabet | Named as the allocation alphabet |
| F12 registry shape | One atomic step per write; no storage interface yet |

The reviewer's hypotheses are open and recorded: a group company that recharges affiliates by invoice may need resale semantics without being an MSP, which the model allows because seller is a role and not an MSP; and the relation between the `customeraccess` company and tenant records and parties is unspecified.
