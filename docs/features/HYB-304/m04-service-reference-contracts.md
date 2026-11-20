---
type: ImplementationPlan
title: HYB-304 service reference and authority contracts
description: Verifies contract-party eligibility, controlled lifecycle drift, tenant-bound scope validation, bounded legacy enumeration and pickup authentication freshness.
status: draft
sources:
  - resource: ./m04-record-lifecycle.md
---

# Ownership and reviewed scope

HYB-304 remains In Progress, Seung Woo Park, machine mac_mini, after pushed/read-back 70d3781c7f206829ed882abbc3c7ee98e4eae3dd. Root is sole writer of focused server tests and this evidence. Separate read-only /root/hyb310_review approved the following test-only design. No production change, registry-map rewrite, new dependency seam, coverage exclusion or waiver is authorized by this plan.

Named-contract terms with wildcard customer/legal-entity selectors must still validate their actual contract customer and seller. Suspend each party with public organization APIs while keeping the declared owning organization independently active. A real previously successful create is a nonempty control; the refused create must preserve identity, declaration, credential/pickup and server revision counts. A public lifecycle Change that selects an absent organization revision must yield dependency refusal without service effects. This is controlled public-module integration drift, not a claim that ordinary HTTP contract commands create it.

A real configured sellerless contract is valid at a later representable reference date. Produced terms from an explicitly configured foreign-workspace scope must fail the service validator's tenant binding. An unrepresentable helper reference date may test CurrentPosition error propagation, but cannot be described as a real dependency outage. Withdrawing every lifecycle event instead yields NotStarted with nil error and exercises only inactive-state refusal; it does not establish the error arm.

Legacy P02 service enumeration with a configured one-row module page must traverse two actual created accounts, terminate after its final empty batch and refuse an actually expired caller without partial output. This does not restore the removed M04 population client. For pickup freshness, install a short authentication-age policy before issuance, then advance beyond its age while remaining inside actual idle/absolute/pickup deadlines. Identity Peek must still succeed, but pickupLive and a fresh observer must refuse to describe the old issuer's pickup as open. Do not stretch a pickup deadline or change policy afterward to simulate this case.

Use actual and staged-helper assertions with clear scope, followed by independent implementation review, scoped/race verification and semantic mutation where meaningful. Only a terminal fresh full server profile establishes a new package result. Earlier99.5%/59-uncovered coverage remains the baseline and does not meet required100%. Bounded M04 aggregate review passed; overall readiness, authorized merge/read-back and parent M03/M02 remain open.

# Observed scoped verification

All six new groups passed in1.208 seconds. The initial four reference groups also passed in1.090 seconds before adding legacy enumeration and pickup freshness. These tests verify existing correct behavior; no product bug or fail-before fix is claimed. Fourteen new and related declaration/eligibility/audience groups passed under the race detector in7.320 seconds. Server-package vet passed. Separate read-only /root/hyb310_review inspected the actual two test files and plan and reported PASS; the reviewer executed no tests or mutations.

Two semantic mutations failed the intended assertions. Skipping actual contract-customer eligibility admitted a create that the implicit-party test refused; skipping pickup authentication freshness left an actually stale issuer's pickup open. Each production file was restored byte-for-byte to its recorded SHA, and the private cache was removed before final checks. No production source change remains. The year10000 reference is explicitly a helper time-boundary input; revision99 is public-module drift; neither is described as an ordinary HTTP-produced dependency outage.

The terminal full server short suite passed in186.417 seconds with99.6% statement coverage:11,783 weighted statements and51 uncovered. This supersedes the latest99.5%/59 measurement; required100% remains unmet. No new Console build/browser/all-package/foreign-compilation run is claimed because production is unchanged. The earlier75-package and actual-app snapshots retain their original boundaries. The two documentation status tests passed; root separately checked changed metadata and relative links. [The service reference manifest](service-reference-evidence.json) binds current sources, terminal logs/profile and semantic mutation evidence. Bounded aggregate M04 review remains PASS; required coverage, authorized merge/read-back and parent M03/M02 remain open.
