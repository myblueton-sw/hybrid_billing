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

Authenticated candidate-backed requests, repeated prerequisite validation at decision/effect boundaries, RequestApproval provisioning, actual human financial decisions and later ERP/issue/delivery/recognition integration remain separate planned work under HYB-340/HYB-225. This increment creates no new governance form or owner screen acceptance. Existing local service state is not upgraded or deployed by this contract.

# Governed amount-authority extension

financial_authority.propose/confirm/inspect/revoke operate on a service-owned proposal ID. Propose requires explicit recipient, currency, canonical exact nonnegative limit, from and to. The derived authority ID is financial-limit-{proposal}; module validation bounds it to128 bytes. Stage is fixed to charge_approval. Both dates are required, validated in2000–2099, and include the complete UTC start/end days. The amount is a maximum absolute amount in the stated currency, without conversion or inferred production defaults.

Propose and confirm require current installation/module governors; proposer and confirmer must differ from each other and the recipient. Recipient must be a currently active human with effective workspace membership. Confirmation rechecks these facts even on a fresh-key historical retry. Inspect and revoke require a current governor. Revoke names the one service-bound module authority and preserves actor, reason and time; it is allowed under hold, unlike propose/confirm. All mutations retain the ordinary session, CSRF, expected-revision, idempotency and step-up envelope.

The immutable bounded binding preserves proposal, confirmation and revocation reasons. All output bounds are checked before effects:23 fields,128 bytes per field and capacity for configured rationale. Pure PrepareAuthority canonicalization creates no records and advances no clock. AuthorityAt returns the module's effective clamped time with its authority snapshot. Receipts derive module_amount_authority from that time and expose authority_as_of, so time rollback cannot falsely reactivate an expired limit. Revocation takes precedence over date status; historical confirmation cannot restore a revoked limit.

M01 financial_authority_proposals counts only pending proposals, separately from financial_policy_proposals and pricing/input permissions. Missing or corrupt service bindings make impact calculation unavailable. The category is Charge approval limit proposals / 청구 승인 한도 제안 / مقترحات حدود اعتماد الرسوم.

The amount-authority extension makes no AddGrant call. Module amount authority alone is not financial approval permission. The separately governed permission extension below supplies direct-human grants; narrowly authenticated decision transport and candidate/prerequisite integration remain required. Existing M03 cannot provision that human-only grant through a console ceiling; that restriction remains unchanged.

# Direct-human permission extension

financial_permission.propose/approve/reject/withdraw/inspect use a service-owned proposal ID. Propose accepts recipient and explicit RFC3339Nano not_before/not_after only. The fixed capability is approve, the resource kind is charge_approval, and the grant covers the complete installation workspace with no field restriction, no client and no console ceiling. Its maximum validity is24 hours; a pending proposal expires after1 hour. The bounded registry holds1000 proposals. No default grant is installed.

Every command, including same-key replay and inspection, requires current authenticated installation and module-governor membership. The closed command family is checked under the existing domain lock before command receipt replay. Unsupported unseeded mode retains the command registry's501 response. Proposal/approval separately recheck the stored proposer and current target; the canonical module-admin set participates in freshness. Proposer and independent approver cannot be the recipient. The recipient must be an active human with effective membership before a new effect. The engine binds actor, rationale and digest before attempting the effect.

Unknown outcomes retain the exact intended grant identity. Before/after-effect faults and lookup failures cannot cause blind recreation. An exact existing revoked or expired grant may resolve uncertainty as historical evidence; it does not restore authority. Replies explicitly mark receipt=historical_decision and recorded_grant_status is historical. Current gate permission, current module amount limits and current candidate prerequisites remain separate checks. Output preflight requires20 fields and1024 bytes per value before mutation.

M01 counts pending, applying and unknown financial_permission_proposals separately as Charge approval permission requests / 청구 승인 권한 요청 / طلبات صلاحية اعتماد الرسوم. Tally failure makes bulk impact unavailable. Hold refuses propose/approve but permits inspect/reject/withdraw; mutations retain step-up. Generic M03 additions refuse the dedicated kind at catalog, request, retained approval and final application. Independently authorized M03 revocation remains available for the actual subject grant. M03 labels the grant in all three languages and explicitly excludes charge-approval eligibility from its console-operation preview.

# Current pricing evidence prerequisite

The internal pricingpolicy.CheckEvidence read validates a retained approved policy's exact scope/digest, requested interval, current target and monotonic clock. It returns a defensive record without granting UsePricingPolicy or any financial permission. The existing Resolve path shares these checks but retains mandatory actor and UseAction authorization first; capture still uses Resolve. No HTTP evidence route is added. The future financial-subject adapter must authorize its actual caller independently. This prerequisite creates no financial subject, request or decision and does not establish external input/source freshness by itself.

# Candidate-backed subject preparation

The internal financialSubjectLocked helper now derives an aggregate StageCharge subject from one retained v7 candidate. It verifies raw envelope SHA256, the separately retained input bytes, requested candidate digest/workspace, original CaptureAuthor and complete governed provenance. It rechecks current approved pricing evidence and current input publication/reconciliation, rebuilds the full normalized input from current source/mapping/target/attribution/lifecycle/rows without storing a candidate, and rerates through the existing exact composition boundary. Any input/result mismatch, held/excluded line or unsupported module numeric precision refuses. No rounding to fit the approval profile is permitted.

The current financial policy must match an exact confirmed service-owned proposal and binding; demonstration policy alone is insufficient. Required digests are selected exactly: versioned hb_financial_evidence_v1 input, rule, mapping and attribution hashes; result is the verified full v7 candidate envelope digest. Required fx or erp_draft refuses until actual producers exist. Amount/currency come from the verified result, Author from immutable CaptureAuthor, and Entries is empty. Returned digest maps are defensive. Pure PrepareSubject validates module shape/precision without creating a request or reading its clock.

The hb_financial_target_v1 identity binds workspace/customer/entity/contract/billing profile/provider/account/period/from/to and excludes candidate/request IDs, currency and producer revisions. Replacement candidates within this scope share a target. A different billing profile, provider/account or period is a different target: no cross-scope obligation deduplication is claimed. This internal preparation neither authenticates nor authorizes callers. Future HTTP request/decision transport must authorize the actual session before lookup, recheck all external evidence under the domain lock and only then call the approval module. RequestApproval provisioning and actual financial commands remain unimplemented.
