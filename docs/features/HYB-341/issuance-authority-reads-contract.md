---
type: ContractDesign
title: Provisioning reads for issuance setup authority
description: History and context reads that let a governor find pending proposals, eligible recipients and active ceilings for the protected, payment-term and draft authority domains.
status: draft
---

# Outcome and ownership

HYB-341, machine:mac_mini. This is the next increment of item 5 of [the handoff](handoff.md), named as out of scope by [the provisioning contract](issuance-authority-provisioning-contract.md): an approver must today be told a proposal identifier, and a proposer must already know a ceiling identifier and a recipient's user identifier. The result authority domain has two governed reads, `history` and `context`, that answer those three questions. This contract gives the same two reads to `issuance_protected_authority`, `issuance_term_authority` and `issuance_draft_authority` by making the existing handler domain-aware. It adds no write, no grant and no new kind of disclosure.

Root owns the design and integration. An independent reviewer covers the disclosure and the domain separation.

# Verified starting facts

Read from main at a05040a.

- `resultAuthorityRead` serves `result_authority.history` and `result_authority.context` under the administer-installation permission. It requires, in order, an existing authority, the governor caller check, the current-governor check, the target equal to the workspace, a transport budget of 19 fields and 1024 bytes, and whole-second positive limits.
- `history` lists every proposal record of the domain with exactly ten fields: `proposal`, `digest`, `kind`, `capability`, `recipient`, `state`, `proposer`, `proposed_at`, `expires_at` and `decision_actor`, the last empty when there is no decision. It carries no decision instant, grant identifier, grant status, rationale or validity; those are returned only by `inspect`. `context` takes a capability and a collection: `recipients` lists effective active human members other than the caller with their login, and `ceilings` lists active console ceilings of that capability with their validity. Only `context` carries a `policy` header, with the closed capabilities in a fixed order and the two limits in whole seconds; the policy enters `history` only through the snapshot digest.
- Both are paged ten rows at a time in identifier order with a snapshot digest over the domain version, the command, the headers, the policy and all rows; a cursor with a stale or missing snapshot, or a cursor that is not a row, is refused as stale evidence. A response that exceeds the transport budget is refused.
- Before listing, `resultRecords` cross-checks the registry against the adapter's own maps: at most 1000 records, bindings and ceilings; one binding per record with the same spec; the domain's shape check on every spec; one ceiling per applied console record with the expected grant identifier. Any mismatch is refused as stale evidence. `resultCeilings` additionally looks each ceiling up in the gate and lists only active ones.
- The code is bound to the result domain in seven places: the authority field the handler reads, the shape function in the cross-checks, the literal capability list in the policy, the terms function that validates the `context` capability, the literal version in the snapshot, the command identifier prefix, and the single registration. Nothing else in it is specific to result evidence; recipients, the `mode` and `current_eligibility` headers and the grant prefix are already domain-neutral.
- The shared registry keeps its capability set as an unexported unordered map and its version unexported, and the domain packages pass both as inline literals. Neither is readable by the server today, and the capability order feeds the policy and the snapshot.
- The boundary and the hold read list name `result_authority.history` and `result_authority.context` literally.
- The three issuance domains have their own registry, `bindings` and `ceilings` maps, shape checks (`protectedShape`, `termShape`, `draftShape`), versions and capability lists.

# Reads

Six commands: `history` and `context` in each of the three namespaces. Target, fields, permission, order of checks, paging, snapshot, bounds, refusals and row shapes are those of the result reads, with the domain substituted:

| Element | Result domain today | Issuance domain |
| --- | --- | --- |
| Authority instance and maps | `resultAuthority` | the namespace's own instance |
| Shape check in the cross-checks | `resultShape` | the domain's own shape check |
| Capability accepted by `context` | the result terms function | the domain's own terms function, and membership in the list below |
| Capabilities listed in `policy`, in this order | `propose`, `confirm`, `revoke`, `apply`, `inspect` | protected: `register_channel`, `admit_channel`, `request_change`, `verify_change`, `approve_change`, `inspect`; term: `record`, `approve`, `bind`, `approve_binding`, `inspect`; draft: `submit`, `approve`, `inspect` |
| Version in the snapshot | `synthetic-erp-result-authority-v1` | `synthetic-issuance-protected-authority-v1`, `synthetic-issuance-term-authority-v1`, `synthetic-issuance-draft-authority-v1` |
| Command prefix and registration | `result_authority`, registered once | the namespace, registered per domain |

- The ordered capability lists and versions are per-domain values in the server, in one new file, built from the domain packages' exported capability constants. The shared registry module is not changed. The version strings duplicate the registries' unexported literals; a mismatch would change only the snapshot digest, and that duplication is accepted. A test binds the lists: every listed capability resolves through the domain's terms function and is accepted by `propose_console`, the list length equals the domain's closed set, and a name outside the list is refused.
- The handler gains a domain; the result commands keep their identifiers and their responses byte for byte, including the snapshot. What must not change for the result domain: the two command identifiers; the snapshot structure and its `Command` value (the bare `history` or `context`); the version; the policy keys and capability order; the header set, with no `policy` header on `history`; the ten history fields and the recipients and ceilings field sets; identifier order, page size and `next`; the order of checks, the refusal codes, the 19-field and 1024-byte budget and the declared stale-evidence refusal.
- Each of the six new definitions declares the stale-evidence refusal and is registered. No row is added to the hold table, which holds mutating commands only.
- The six names are added to `financialPermissionBoundary` and to the hold read list, as the result names are.
- A capability name outside the domain's closed list is refused as an invalid request. `inspect` and `approve` are names shared between domains and are valid in each domain that lists them; they reach only that namespace's records.

# Disclosure

To a current governor only, exactly as in the result domain: proposal history of the namespace including proposer, recipient and decision actor identifiers; logins of effective active human members; identifiers and validity of active ceilings. No evidence text, no attestation, no reference, no customer data and no configuration content is reachable through these reads. A governor already administers the installation and can list members through M01.

The recipient of a protected step capability shows who holds each side of a separated flow; a governor can already read this through `inspect`.

Residuals. Each registry refuses a new proposal once it holds 1000 records, which the provisioning contract already records; the reads refuse only above 1000 records and so cannot be reached, so at capacity proposals stop while `history` and `context` keep working. What the reads add is a usability limit: `history` includes every state, is never pruned, has no filter and returns ten rows per call, so near capacity finding the pending proposals takes up to a hundred sequential calls, and any new proposal, decision or expiry changes the snapshot and makes the client restart paging after a clean stale-evidence refusal. Whether that is acceptable for the provisioning screen, or a pending-only filter, pruning of terminal records or another capacity is wanted, is an owner decision for the screen increment; this contract keeps the result domain's behavior.

# What this does not do

- No screen. The M03 provisioning screen for these namespaces follows and has its own design.
- No filter by state or capability on `history`; the client filters the bounded list.
- No change to who may read: a holder of a step capability or of `inspect` who is not a governor is refused.

# Alternatives not chosen

- Copying the result handler three times. The substitutions above are the whole difference.
- One read with a domain field. Command names key the boundary and the hold list, as the provisioning contract already decided.

# Required verification

- for each namespace: `history` empty, then listing a pending ceiling, an applied ceiling, a pending human proposal and a rejected one with the ten fields; `context` recipients excluding the caller; `context` ceilings listing only an active ceiling of the asked capability and omitting a pending one, another capability's and an expired one. A ceiling that has not started cannot exist, because approval is refused before the start instant, so that filter is defensive and has no test;
- `history` returning the list, without settling anything, while it holds an expired pending record and a record left in the unknown decision state;
- the `context` policy header listing the domain's capabilities in the stated order, bound to what `propose_console` accepts; an unknown collection refused;
- separation: a proposal of one namespace never appears in another namespace's history or ceilings or in the result domain's; a name outside the domain's closed list refused; a cursor and snapshot of one namespace refused in another, and identical empty lists of two namespaces having different snapshots;
- authority: a non-governor refused, including one who holds the domain's `inspect`; a governor who holds `inspect` allowed; an installation without the sample seed returning unsupported capability; target other than the workspace refused;
- paging: more than ten rows, cursor and snapshot round trip, a changed list refusing the old snapshot, a cursor that is not a row refused;
- bounds: the reduced transport budget refused; a corrupted binding or ceiling map refused as stale evidence;
- hold: both reads allowed for each namespace;
- compatibility: the result reads' existing tests unchanged and passing, and a test that is not skipped asserting the result `history` and `context` snapshots of a fixed deterministic sequence against values generated from the unchanged code at a05040a;
- the journey: an approver finding a pending proposal's identifier and digest through `history` and a proposer finding the ceiling and recipient through `context`, with no identifier passed out of band.

# Limits

Synthetic in-memory boundary only. No implementation, test run or review result is claimed by this draft.
