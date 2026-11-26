---
type: ImplementationPlan
title: HYB-304 canonical session admission consistency
description: Centralizes the existing canonical caller-owner and bound-handle guarantee at successful authentication while preserving tenant, target and cursor checks.
status: draft
sources:
  - resource: ./m04-completion.md
  - resource: ./m04-invariant-cleanup.md
---

# Scope and independent design review

HYB-304 remains In Progress, accountable owner Seung Woo Park, machine mac_mini, following pushed/read-back c7c559bdc973e8cb89a96bd241cb9918bcfea4ba. Root owns implementation and integration. Separate read-only /root/hyb310_review traced the authentication producers and consumers and approved a narrow admission-consistency design; this is not approval of an unseen implementation, a coverage waiver or feature readiness.

The substantive purpose is one consistent authentication guarantee for all current access consumers. After existing denial, trust epoch, identity authentication, freshness, access-hook and presence checks, successful access must require the captured session's workspace and canonical owner triple to match the actual authenticated answer and require a nonempty bound handle. Mismatched component inputs return Unauthenticated without newly terminating either actual session. Place this before successful idle touch/return. Remove only the duplicated caller owner comparisons in self/session query/detail/preview/preparation/operation reads and the duplicated caller-owner/handle clauses in M04 query admission.

OpenSession resolves the real user and epoch and binds owner/workspace/handle before returning a cookie. Session binding cannot subsequently change owner. Invitation acceptance and its direct identity access occur after binding; pending membership does not prohibit binding. No production pre-bind Server.access caller was found in independent review. Its guarantee reaches self/admitted, current-session reads, commands, integration admission, reader proxy, password commit and reauthentication commit.

Keep the separate M04 boot-workspace restriction with its existing status. Keep target/selection/operation record ownership, original-caller rules, cursor bindings, other-session Peek/epoch/owner projection, source, expiry, revocation, query and capacity checks. Healthy-session validation precedence is unchanged. Inconsistent component inputs may be refused before downstream malformed-query validation; do not claim identical status precedence for impossible published session states.

# Validation before and after implementation

Authenticate two real actors through the actual password/sign-in paths and capture legitimate session values. Compose invalid local value copies for caller owner, owner workspace/epoch and missing handle; do not edit registry maps or session-store records. The current code must fail the new refusal assertion before the change. These are component-input invariants, not fabricated HTTP exploit reproductions. Both original actors must remain usable through actual self reads afterward. Also verify separate same-owner callers, actual invitation acceptance after binding, ended captured sessions and malformed queries from an ordinary valid actor. Preserve applicable existing invitation, revocation, step-up and session-query suites.

Root will run scoped race/semantic guard mutation, the full short server suite/coverage, vet and portable compilation checks, then obtain actual-source and evidence re-review. Current M04 behavior closure remains recorded separately. Required package100% is still unmet at the preceding99.1%/101-uncovered profile; centralizing repeated defenses does not authorize discarding other guards or claiming completion.


# Additional bounded contracts

The same verification increment exercises already-reviewed module defenses after real actor expiry: admitted account-credential aggregation and next-service components must refuse before publishing progress, and supported service/credential module faults must publish only dependency failure. Healthy controls and subsequent actual expired-session401 distinguish these staged component tests from fabricated HTTP scheduling. Public installer descriptors must be independent copies, named selection setters must update their actual documented fields, and individually valid selection settings whose record budget is insufficient must fail the real constructor before start; its exact combined bound remains configurable and accepted. These tests introduce no production dependency interface or registry-map mutation.

The first baseline correctly exposed all four new component admission refusals. An additional invitation setup expectation was corrected before the implementation run: password redemption can accept pending membership through an identity probe. The final fixture revokes the invitation during password setup, then reinvites before actual sign-in, following the supported producer sequence. Its self result asserts the actual user_id field. The corrected pre-change baseline fails only the intended inconsistent-component cases; ordinary same-owner, ended-capture and invitation cases pass.

# Observed verification

The corrected pre-change run failed only the four intended inconsistent captured-value assertions. After the guard change, all seven new groups passed in1.450 seconds. Nineteen new and existing authentication, invitation and session groups passed under the race detector in4.390 seconds. Two semantic mutations were assertion-killed: disabling the central guard admitted inconsistent captures, and dropping the actual session on component refusal ended an unchanged actor. Each mutation restored exact source SHA2568091f911637d308b4e8c9c5bc98db2d55a6dc618fc5a545e8b0f9e480d7b08fd; its private build cache was removed before final validation.

The fresh full Go short suite passed75 packages. Server coverage was99.3% in199.068 seconds:11,783 weighted statements,79 uncovered. This supersedes the latest package measurement of99.1%/101 uncovered; required100% remains unmet. All-package vet and Linux/Windows amd64 test-binary compilation passed; no foreign-platform runtime execution is claimed.

The restored current backend also passed the existing actual action/form browser group: three scenarios,29 captures, zero page errors. The group covers distinct declaration/revoke field refusals in all EN/KO/AR and1440/1280/1024/390 variants, actual password confirmation retaining an unsent revoke rationale, and actual Suspend/Reinstate/Retire states. It does not replace the separately source-bound navigation/declaration evidence. The unchanged Console product still binds683 current inputs and439 artifacts to f2df55941f08768cedec2aec304ef8bb4837e2482c49e2fc68ff28d9ebdcd6ff. The owned loopback runtime was stopped before evidence hashing. The previous1,016-case Console checkpoint was not rerun for this backend-only increment.

Independent actual-source security review by /root/hyb310_review passed the nine-file implementation and all three new test files. The reviewer did not execute tests or services. Only eight duplicated caller-owner blocks and M04's duplicated caller clauses were removed; record/target/cursor/other-session and boot-workspace defenses remain. Evidence and source bindings are recorded in [the admission manifest](admission-consistency-evidence.json). Final aggregate M04 review, required package100%, PR readiness and authorized merge/read-back remain open. Parent M03/M02 and the full planning goal remain open; no deployment, customer-data, native/AT/BFCache, Claude visual or owner FIX acceptance is claimed.
