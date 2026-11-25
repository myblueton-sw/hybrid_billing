---
type: VerificationRecord
title: Strict caller packet and server-derived document evidence
description: Evidence for the packet admission prerequisite to public document commands.
status: draft
---

# Scope and ownership

HYB-340, machine:mac_mini, follows 4c55cbf. Root owns implementation and validation; refusal_review independently reviewed design and source. The accepted design restricts caller input to explicit mode/profile/evidence, owned charge request/digest, connection selector, expected net/tax and line ID/product/quantity/unit/tax code/tax/evidence. Workspace, packet ID and actor remain validated server arguments. Required values are checked separately because the reused JSON shape validator does not require every key.

# Implemented boundary

The exact-field parser rejects unknown or duplicate keys, case aliases, nulls, missing mandatory values, invalid Unicode, trailing values and duplicate line identities. Input is limited to 4096 bytes and 100 lines; 100 is an additional upper bound, not a guaranteed payload size. Tax zero must be explicit. Decimal strings remain unchanged for the existing exact composition validator.

Under the caller-owned domain lock, the builder checks actual proposal rights before lookup, then current financial approval and its owned candidate binding. Organization attribution, candidate digest, charge rule/kind, connection, mappings and survey come from current server records. Mapping lookup order is deterministic. Complete existing document validation applies before return. No document, preparation ownership, approval execution or financial revision is created by building a packet. Errors preserve permission, unavailable-dependency and stale-evidence distinctions.

# Root execution and independent review

Focused packet tests passed 0.638s. Financial document/ERP/source-rule race tests passed 14.389s. After supplemental exact refusal-code, missing connection/mapping and revoked financial-approver cases, packet race tests passed three repetitions in 2.793s. Server vet passed without diagnostics. Logs are local temporary diagnostics at /private/tmp/hyb340-document-packet-race.log, /private/tmp/hyb340-document-packet-regression.log and /private/tmp/hyb340-document-packet-vet.log. The broader race run preceded the supplemental test-only additions; production code was unchanged. No full suite or browser run is claimed for this increment.

Independent refusal_review passed design and source by inspection, without executing tests. It required permission-before-lookup, mandatory-field checks, exact decimals, deterministic mapping lookup, no state insertion and a separate future output-capacity check. Its suggested missing-dependency and revoked-approver cases were added. Tests also exercise authoritative snapshot injection, denied existing/missing requests, wrong request/digest, incomplete rows, mixed units, invalid numeric representation, wrong totals, returned-copy isolation and unchanged stores.

# Remaining service integration

This is the actual server packet builder, not a published HTTP command. Public proposal admission must preflight the expanded inspection output and future event reasons before retention. Named sample ERP startup assembly, command registration, current permission checks before cached replay, hold/step-up/revision/idempotency, historical preparation inspection and pending-impact integration remain next. No UI change, external submission, deployment, merge, owner acceptance or goal completion occurred. Only macOS execution is verified.
