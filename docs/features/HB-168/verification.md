---
type: VerificationRecord
title: Operating profiles and party model verification
description: Records what was executed and reviewed for the first domain model of work item W-01, with results, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-168](https://linear.app/hybrid-billing/issue/HB-168/w-01-operating-profiles-and-party-model-installation-workspace-party) adds the Go package `modules/organizations` from main `451cd57`: workspaces with an operating-profile label, legal entities and internal units, billing profiles with separate issuer, buyer, payer and recipient and an issuance path, append-only contract revisions with a transaction path and per-component provider payment responsibility, three pure rules (issuance requirement, selling eligibility, document key) and an in-memory registry with expected versions. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan before implementation and the implemented change before merge. Machine: mac_mini. See the [plan](plan.md).

Nothing calls the package yet. It stores nothing durably, authorizes nothing and activates no customer structure. It does not complete W-01: HB-01-D and HB-01-V, and the acceptance cases GH01, GH02, GH07 and AUD-AR-01, remain open and NOT_RUN.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| `go test -race ./...` | PASS: `modules/organizations` and the four existing Go packages |
| `go vet ./...`, `gofmt -l` | No finding, no unformatted file |
| Billing profile cases | PASS: 5 accepted shapes and 18 rejections, each with its expected error |
| Operating profile as a label | PASS: identical data under `msp`, `single_enterprise` and `group` gives identical outcomes; a single enterprise registers an internal chargeback with no seller party |
| Party cases | PASS: 10 cases on kind, parent, immutability, versions and suspension, plus cross-workspace reads |
| Contract revision cases | PASS: 4 accepted shapes and 16 rejections; append-only numbering; copies on write and read |
| Selling eligibility | PASS: 21 decisions on literal values covering every reason, including values the registry would not store (unknown path or kind, a direct contract naming the seller responsible), and two on a revision read back from the registry; a direct revision leaves only the seller's own service eligible and a resale revision the seller-paid in-scope charge and the service |
| Document key | PASS: equal keys combine; 5 differing dimensions, an empty request, 5 incomplete or inconsistent keys, and identical identifiers in two workspaces are rejected |
| Suspension and frozen fields | PASS: a billing profile's issuer, buyer, payer, currency and path cannot change; a profile or unit can be suspended while a party it names is suspended and reactivated only when that party is active again; writes naming a suspended issuer, buyer, payer, recipient, customer, seller, parent legal entity or workspace are rejected; a contract keeps its customer and provider reference |
| 64 parallel writers with one expected version | PASS under the race detector: one success, 63 conflicts |
| Mutation check | Nineteen temporary single-point changes each made the tests fail; all were reverted. One of them, ignoring a suspended parent legal entity, first survived and was then covered by an added case |

Host: macOS arm64, Go 1.25.7. Other operating systems were not exercised; the code is pure computation.

# Provisional choices and undecided matters

Provisional, to be confirmed or replaced: the `internal_unit` party kind and its flat parent pointer; internal chargeback limited to one legal entity; one currency per billing profile; the active and suspended states; no delete; the fields of the document key; billing profiles and contract revisions living in `modules/organizations`. Undecided and not encoded: the pilot operating type, real legal entities and responsibilities, the first selling mode, tax, the component catalogue, the database, and settlement between two legal entities without an invoice, which is rejected with its own reason.

# Limits

- The registry is not an authorization source and provides no customer isolation inside a workspace; both are W-03.
- No persistence: state is lost with the process, and the concurrency test shows only an in-memory check-and-set.
- AUD-AR-01 is covered at rule level only; selling calculation and approval, which must consume the decision, do not exist.
- The relation between these parties and the companies and tenants of `modules/customeraccess` is unspecified.
- Nothing links a billing profile's issuer and buyer to a contract's seller and customer.
- All internal chargebacks inside one legal entity and currency share one document key whatever units are involved, and the document recipient is not part of the key; whether per-unit statements or recipients need to separate documents is open.
- A resale revision whose provider charges are all the customer's responsibility is accepted; whether that is a coherent resale contract is a W-15 question.
- Suspension does not propagate from a party to the records that name it.

# Independent review

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI), 2026-11-12. It read files and executed nothing.

| Round | Result |
| --- | --- |
| Design review of the plan, before implementation | FAIL without blocker: 3 HIGH (eligibility rule, missing buyer, undirected payment responsibility), 6 MEDIUM, 4 LOW. The model was redesigned as in the plan's resolution table before code was written |
| Change review of the implemented code | FAIL without blocker: 1 HIGH, 5 MEDIUM, 8 LOW, corrected before merge. HIGH: the document key had no workspace, so lines of two workspaces with identical identifiers could combine; the workspace is now part of the key. MEDIUM: the eligibility function accepted values the registry would reject and now fails closed; a suspended party made its dependents unsuspendable and a suspended legal entity did not block chargeback between its units, both corrected; workspace guards and suspended references were untested, cases added; billing profile fields other than state are now frozen; a contract keeps its customer and provider reference. LOW corrected: empty references are invalid rather than missing, check order as stated, incomplete keys rejected, payer or recipient equal to the issuer rejected, the label test compares with a literal. No second review round ran on these corrections; they were checked by the added cases and nine further mutations |
