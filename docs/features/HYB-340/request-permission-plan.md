---
type: ImplementationPlan
title: Governed permission to request charge approval
description: Separate console ceiling and human RequestApproval grants without financial decision authority.
status: draft
---

# Scope and ownership

Continue HYB-340 after a8960f8 on machine:mac_mini. Root owns implementation and integration; independent refusal_review accepted the concrete domain/command design before code. Owned files are modules/financialrequestauthority and tests, narrow server grant-domain/factory/command/boundary/assembly/hold/impact changes, scoped M01/M03/P02 EN/KO/AR labels and verification. Independent reviewers remain read-only; actual Claude CLI ui-designer reviews changed renders.

# Fixed permission boundary

New closed domain4 uses synthetic-financial-request-authority-v1, financial-request- grant/fault namespace and only request capability. It maps to gate.RequestApproval on charge_approval, complete workspace scope, AnyPeriod and no fields. Reuse the ceiling-mode recoverable engine. Domain3 remains exclusively direct-human Approve; no console Approve is permitted and no default grant is seeded.

The six financial_request_permission commands propose_console/propose_human/approve/reject/withdraw/inspect require current installation and module governors, including before same-key replay. Capability is fixed server-side and omitted from the wire. Console proposals name ConsoleClient only; human proposals name an active human plus an exactly matching approved domain4 ceiling and contained validity. Reject cross-domain ceiling provenance. Final grant construction independently checks these shapes; historical reconciliation does not pretend a revoked ceiling is current.

Keep distinct instances, bindings and ceiling provenance. Preserve exact intents, current proposer/recipient checks before fresh effects, before/after-effect recovery, historical revoked-effect reconciliation and no blind reapplication. Preflight 20 fields and 1024 bytes per value; mark receipts historical_decision. Hold permits inspection/rejection/withdrawal and refuses proposals/approval; mutations retain step-up. Legacy input/pricing contracts and digest identities remain unchanged.

Generic charge_approval additions stay excluded. Independently authorized M03 human-grant revocation remains available; protected console grants are not made revocable through M03. M01 separately counts pending/applying/unknown permission proposals and refuses impact preparation when the tally is unavailable. Copy must distinguish permission to submit a charge approval request from permission to decide it, amount limits and actual financial requests.

# Acceptance and next boundary

Verify fixed domain/capability/kind, independent governors and recipient exclusions, no defaults or console Approve, current session/module governance before replay, approved/active ceiling containment and cross-domain refusal, exact recovery/revocation, generic bypass refusal, SeedNone, hold/step-up/output capacity and legacy regression. Run actual isolated M01/M03/P02 locale/width journeys with scoped independent terminology and Claude visual reviews. Request/decision HTTP transport, bounded financial review and immutable request-to-candidate binding remain the next separately reviewed increment; this permission grant alone creates no financial request or decision.
