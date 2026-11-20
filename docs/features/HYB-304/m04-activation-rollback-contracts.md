---
type: VerificationReport
title: HYB-304 first administrator activation with incomplete integration rollback
description: Defines actual HTTP activation verification for retained grants, withheld integration rights and bootstrap completion after a rollback failure.
status: draft
sources:
  - resource: ./m04-command-dependency-contracts.md
---

# Reviewed scope and ownership

HYB-304 remains In Progress, accountable Seung Woo Park, machine mac_mini, after pushed/read-back a564e41ee701365f6278389e2283de118d87da4c. Root owns integration_activation_rollback_test.go and scoped evidence/documents. Separate read-only /root/hyb310_review approved the concrete first-admin activation oracle. No product edits or new failure seams are planned.

Use SeedNone and its genuine activation token. During actual HTTP redemption install the existing integration-grant-3 and integration-rollback faults. The fourth subject grant is inserted before failure, and every attempted rollback is refused. Require eight open Console client grants, exactly four open subject grants (administer_api_clients, administer_modules, approve_modules, stop_modules), no later subject grants and no closed integration grants. Do not demand atomic rollback when rollback itself fails. Require exact eight-key integration rights and truthful available clients/modules versus withheld channels/events.

Activation must return204 and complete bootstrap despite grant failure. The token is consumed and reuse cannot create another user. Actual password sign-in and /me must still work; the firstAdmin helper alone is insufficient because it opens its returned session directly. Compare public identity users before/after token reuse. Existing installation-administrator authority must not supply the withheld integration rights. Fault callbacks are restored exactly before post-activation observations.

Validate behavior with meaningful semantic mutations of retained grants/rollback/bootstrap completion and exact restoration/private-cache cleanup. The signin errRollback branch changes logging only; this business-state oracle does not claim to kill a log-only mutation. Actual review and verification are recorded below. Required100%, authorized merge/read-back and parent M03/M02 remain open.


# Actual verification and limits

The first scoped native run passed1.024s; a separate focused native profile passed0.566s. Independent read-only /root/hyb310_review passed the actual source and its HTTP/rights/token/cleanup boundaries. Two semantic mutants were assertion-killed: revoking grants despite the rollback fault violates the exact retained grant set; keeping bootstrap live violates completed activation. Both restored the exact source SHA and removed the private cache. These mutations prove business-state assertions, not necessity of the errRollback logging distinction.

Four named race groups passed4.873s. Server vet and Linux/Windows amd64 test compilation passed; foreign execution remains unverified. Full server short passed193.884s at native99.8%,11,779 weighted statements/29 uncovered. Prior/full/focused profiles match production blocks and weights; the single new unit is hit by the separate focused profile. Profiles are not combined.

The [source/artifact manifest](activation-rollback-evidence.json) records this checkpoint. The existing rollback contract intentionally leaves four subject grants when all attempted revocations fail; successful activation does not certify complete integration authority. Prior75-package, frontend/browser and bounded aggregate evidence retain their source/run boundaries and were not rerun. Two documentation status checks and manual metadata/local-link checks are limited checks. Required100%, authorized merge/main read-back and parent M03/M02 remain open.
