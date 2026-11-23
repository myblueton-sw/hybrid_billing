---
type: VerificationRecord
title: Dedicated document authority primitives and generic-add isolation
description: Direct-human actions and a fixed grant domain required before actual document governance and ERP preparation commands.
status: draft
---

# Implemented boundary

HYB-340, machine:mac_mini. Four new auth actions represent proposing, confirming and revoking document evidence and preparing an internal ERP record. They reuse directHuman enforcement with ordinary user-plus-console intersection, not financial Approve's humanOnly exception. Financial approval remains unchanged. The fixed documentauthority wrapper permits propose, confirm, revoke, prepare and inspect with a distinct digest domain and effect prefix; it reuses the existing grant-governance engine.

The server's shared governedGrantTerms boundary excludes every new action, resource kind and evidence field from generic additions, including ceiling selection, retained request standing and final application. Explicit individual revocation remains available. No public kind/field registration, provisioning-domain initialization, default grant, runtime preparation command or screen is introduced. Existing internal preparation and complete W-23 scope remain distinct.

# Coverage and independent review

Existing pricing authorization tests were generalized to all eight pricing/document actions without dropping prior assertions. They test actual auth decisions and stamp rechecks for missing human/console rights, cross-capability isolation, service/client-only/support/delegated execution, revocation, suspension, membership removal, expiry and workspace/customer/entity/contract/period/kind restrictions. Financial Approve still rejects ordinary-console use.

Wrapper tests cover all five closed capabilities, independent governors, current proposer authority, exact applied replay and input/pricing cross-domain digest/effect substitution. Wrapper creation produces no effects. These use synthetic Authority/Targets/Effects ports and do not prove actual server provisioning, which remains unassembled.

Actual server generic-add tests install valid dedicated grants using an explicit test-only catalog. Each action, kind and field is independently refused through crafted HTTP requests, retained-request standing and final application; the test therefore does not pass merely because a kind is unknown. No dedicated default grants exist. Actual explicit revocation closes an injected dedicated human grant. Bulk revocation internals are unchanged; no new document-specific bulk flow is claimed.

Independent refusal_review passed the design before implementation and passed concrete source/tests afterward. It confirmed existing pricing coverage remains intact and the bypass test has valid grants. It did not run tests or independently verify result logs. Independent platform_refusals passed the document/source/log comparison, including the mock-port versus actual-server distinction and pending full regression. It did not rerun tests or perform UI verification.

# Root validation

Full auth race passed 2.402s, gate race 2.951s, documentauthority race 1.284s and the shared internal/grantauthority race suite passed from cache. Evidence: /private/tmp/hyb340-document-authority-race.log. Focused actual server document/pricing generic-access and source-rule race tests passed 2.483s (/private/tmp/hyb340-document-access-race.log). Vet for auth/gate/documentauthority/server passed with no diagnostics (/private/tmp/hyb340-document-authority-vet.log).

Root subsequently completed go test ./... -short: all 82 packages passed, including server 136.448s (/private/tmp/hyb340-document-authority-full.log). The new documentauthority package accounts for the increase from 81 packages. This checkpoint changes no visible product copy or UI layout, so rendered verification is not claimed. Only macOS execution is verified; no new dependency or OS-specific implementation is introduced.

# Next connection

The [authority plan](document-authority-plan.md) defines the next reviewed runtime boundary: separately provisioned rights, protected Cost inspection, immutable explicit document packets, distinct confirmation and authenticated preparation. Confirmation must not imply gross/tax approval, and preparation must not imply send/post/issuance. Current authority, evidence and owned-recovery checks remain mandatory. No deployment, merge, external effect or owner FIX occurred.
