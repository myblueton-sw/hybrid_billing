---
type: Verification
title: Result authority discovery and provisioning context checkpoint
description: Server-only evidence for exact governor-protected proposal and option reads before UI integration.
status: draft
---

# Scope

HYB-341, machine:mac_mini. Root added result_authority.history and result_authority.context, backed by the actual result-domain registry and current identity/grant facts. This server prerequisite does not complete the M03/P02 user flow. No external rights, deployment, merge or owner acceptance occurred.

# Checks completed

Actual HTTP tests verify active ordinary human recipients, proposer exclusion, matching owned console ceilings, cross-capability cursor refusal, complete ten-row paging, time-driven proposal expiry without service-revision change, invalid fields/cursors, unauthorized ordinary users, unavailable effect lookup and orphan ceiling refusal. A population of 1,001 isolated synthetic humans verifies complete recipient bounds. Unknown post-write effects remain unresolved after history/context inspection, and held-state reads remain available. Tests also verify fractional-second ceiling boundaries can be reused exactly by a human grant proposal and missing applied-console reverse associations fail closed.

Initial targeted HTTP run passed in 0.845 seconds. The final result authority/permission race run passed in 4.828 seconds. Final server vet passed. Full Go short regression passed all 83 packages, including server 147.650 seconds. Independent source reviewer required exact timestamp precision and stronger reverse-association checks; root corrected both and received scoped corrective source PASS. The reviewer inspected source/tests and did not execute them.

# Failures that changed the work

Initial tests used an inactive sample member and null request maps; the fixture now activates its synthetic recipient and sends the proper empty object. The new command definitions initially omitted EvidenceStale from declared refusals, causing generic internal errors instead of the intended stale response; the declaration is now explicit. The expiry test originally allowed the synthetic login to time out; it now renews authentication after advancing time, verifies the service revision is unchanged and then exercises snapshot expiry. No authorization or expiry check was removed.

# Open acceptance

Frontend strict parsing/session and attempt state, M03/P02/B09 navigation, actual multi-user provisioning-to-use browser journeys, localization and rendered independent review remain open. Existing B09 result review acceptance does not certify these new provisioning controls. Current decision checks are deliberately deferred to submission; historical rows are not ready/authorized flags. Read-only inspection never settles unknown grants.
