---
type: VerificationRecord
title: Direct-human permission proposal engine verification
description: Module-only HYB-340 extension preserving legacy ceiling permissions and uncertain-effect recovery.
status: draft
---

# Scope

After7c75d99, the internal grant-authority engine supports a closed direct-human mode exposed by modules/financialpermission. It rejects console/ceiling shapes and protects proposer/approver/recipient separation. Existing input/pricing modes and fixed input digests remain unchanged. This increment does not connect a server command, create an actual gate grant, add decision transport, render a new screen or complete HYB-340.

# Executed checks

The initial fresh race run passed all four affected packages at1.504–1.506s (/private/tmp/hyb340-permission-module-race.log). Supplemental tests added independent pending rejection/withdrawal and fresh-governor refusal. Review of the unknown-termination assertion found that rejection used the proposer and could pass for the wrong reason; it now uses the independent governor and requires ErrConflict, while withdrawal uses the proposer. The final race run passed all four packages: three unchanged packages were cached and financialpermission passed1.350s (/private/tmp/hyb340-permission-module-final.log).

The final unchanged source passed go test -short ./... for all80 packages, compared exactly with go list: server126.323s and financialpermission0.317s (/private/tmp/hyb340-permission-all-short.log). go vet ./... passed (/private/tmp/hyb340-permission-vet.log). This is not a full-suite race claim. macOS execution does not establish Windows/Linux operation. Console source did not change; no new browser or visual acceptance is claimed.

Coverage includes unknown/immutable modes, closed capability/kind/ceiling shapes, all three recipient exclusions, independent governor approval/rejection, proposer-only withdrawal, current-governor refusal, exact effects, distinct digest domains, existing input hash fixtures, uncertain before/after-effect handling, changed intent/target refusal, revoked/expired effect reconciliation without recreation, lookup failure, capacity and defensive records. Fixtures supply abstract authority/target/effect ports; they do not prove actual session authorization or gate.AddGrant integration.

# Independent review and next boundary

Independent refusal_review accepted the concrete mode design before implementation and passed the final module source/test review with no blockers. It inspected the final log, executed no tests and made no edits. The default mode remains unchanged; no engine recovery rule was relaxed. The subsequent server adapter design received a separate conditional acceptance documented in permission-plan.md.

Next implement the fixed direct-human server domain, exact grant reconciliation and current governance/recipient checks, then command/impact/hold/step-up and independently reviewed UI evidence. Preserve separate module amount limits and the prohibition on console Approve grants. Candidate-backed RequestApproval and actual financial decision transport remain outstanding, along with the original W-23 residual scope. No merge, deployment, payment, real-account/customer-data operation or owner FIX occurred.
