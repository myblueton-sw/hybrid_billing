---
type: ServiceContract
title: Synthetic charge approval policy governance
description: Current authenticated two-person policy governance and historical approval read boundaries.
status: draft
---

# Commands and authority

HYB-340 adds four commands to the existing authenticated command API. All require current user and console AdministerInstallation authorization, active human installation-administrator membership and membership in the existing approvals registry's governor set. Only sample mode is supported. No production policy initialization, financial Approve grant, amount ceiling, charge decision or downstream execution is implied.

| Command | Target | Body | Effect |
| --- | --- | --- | --- |
| financial_policy.current | charge_approval | Empty | Current StageCharge policy, explicitly synthetic |
| financial_policy.propose | Unique proposal ID | policy_revision, approvals, validity_hours, required | Pending policy proposal with immutable proposer and rationale |
| financial_policy.inspect | Service-owned proposal ID | Empty | Historical proposal/decision receipt |
| financial_policy.confirm | Service-owned proposal ID | digest | A different current governor confirms the exact proposal |

Numeric fields use canonical positive decimal integers; approvals is 1–3 and validity_hours is 1–2160. policy_revision must be the next revision at proposal and confirmation. required is a sorted, unique comma-separated subset of attribution, erp_draft, fx, input, mapping, result and rule, including result. These configure prerequisites; they do not supply or certify that evidence. The caller cannot select another stage or impersonate an actor. The existing command envelope supplies rationale, expected revision, request/idempotency IDs, CSRF and actual session. Mutations require existing step-up and refuse under trust hold. Authorized inspect/current reads remain available under hold.

Confirmation rechecks the stored proposer’s current identity, membership, installation authorization and module-governor membership. A fresh-key historical replay cannot bypass those checks. An ordinary same-key command replay follows the command registry's authorized receipt semantics, not a new confirmation. Changed policy fields, actor, digest or rationale cannot rewrite an existing receipt. Superseded proposals do not restore older policies. Competing confirmations of the same next revision cannot both apply.

The bounded service binding stores digest, proposer and immutable reasons beside the existing bounded proposal registry. Unknown demonstration proposals cannot be adopted for service mutations. Supported output capacity is checked before effects: at least16 fields,128 bytes per field, and enough bytes for the configured rationale maximum. Registry changes and service binding assignment occur under the existing server domain lock. This is in-memory synthetic operation, not durable financial storage.

# Impact and historical reads

M01 separately counts pending service-owned financial_policy_proposals by their recorded proposer. Missing/corrupt bindings or an unavailable tally refuse impact preparation rather than reporting zero. Confirmed proposals are excluded. The count does not cancel proposals or represent pending charge requests, pricing policy grants or payments.

B09 uses the request's confirmed historical policy revision. Superseded previously-approved requests retain their single historical approver and decision time; updated time records supersession. This does not establish present approval authority. The reader still refuses unsupported multi-approval/partial/exclusion shapes, and CheckCurrent/Execute continue using current policy and current prerequisites. Current policy replacement therefore does not erase truthful old history or make old history executable.

# Remaining W-23 integration

Financial authority grants, original-author v7 candidate subjects, fresh governed prerequisite validation, actual human financial decisions and later ERP/issue/delivery/recognition integration remain separate planned work under HYB-340/HYB-225. This increment creates no new governance form or owner screen acceptance. Existing local service state is not upgraded or deployed by this contract.
