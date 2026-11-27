---
type: ImplementationPlan
title: HYB-304 exact lookup and object parsing contracts
description: Simplifies two independently proven lookup/parser invariants and verifies malformed admitted-component paging and strict authenticated session objects.
status: stable
sources:
  - resource: ./m04-cleanup-boundary-contracts.md
---

# Final status

Checkpoint record. The work it names as remaining or not yet built was completed later in HYB-304, which closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Ownership and independent design

HYB-304 remains In Progress, Seung Woo Park, machine mac_mini, after pushed/read-back 95da857654bd9bb499424206d40d79588bd471fa. Root owns only servicequery_audience.go, sessionjson.go, focused tests and evidence. Separate read-only /root/hyb310_review approved two narrow production simplifications and four malformed-component propagation cases. Design review is not actual diff review or a coverage waiver.

Exact audience lookup uses the immutable successfully constructed catalog. Its remaining lookup errors may uniformly use the existing403 PermissionDenied policy, preserving admission, boot-workspace, ID and source checks. The catalog's module-level invalid/unready behavior is unchanged. In sessionObject, after a successful object opening and successful key-position Token, the installed Go1.25.7 encoding/json state machine guarantees a string token. More does not consume or change that state. Replace the contradictory non-string arm with a direct string assertion. Keep token errors, size bounds, allowed and duplicate keys, value decoding, closing delimiter and trailing-input validation. These are proven-invariant simplifications, not product fail-before bug fixes or permission to discard other guards.

For each choice helper, establish a healthy genuine admission and real producer control. Change only cursor.After in a fresh admitted value copy to a non-ID string. Require that current module's ErrInvalid, nil rows, more false, unchanged cursor/matched progress, stable source/state and a usable actor. Owners, organizations and audiences make no scan progress; scopes reserves eight units before ScopePage and must retain that honest cost. The mapped component failure must be one JSON object without page metadata. A resealed malformed position must still be rejected at actual HTTP admission with409. These are malformed admitted-component inputs, not attacker-signable continuations or dependency outages; unready modules are not fabricated.

Actual authenticated preparation must reject an unknown-only object; session end inputs must continue rejecting unknown/duplicate keys, nonobjects and trailing values without session/operation/accounting effects. A valid single-target preparation is the healthy control. Run existing exact-audience and strict-session suites before and after changes, meaningful semantic mutations, scoped race, server regression/vet and Linux/Windows compilation. No new UI copy or frontend source changes are planned. Required100% remains unmet at native99.6%/45 uncovered; bounded aggregate M04 review passed, while authorized merge/read-back and parent M03/M02 remain open.

# Initial mutation scope correction

Before and after the two approved simplifications, five focused groups passed in1.993 and1.399 seconds. Swallowing owner/scope module errors failed the intended component assertions. The first unknown-key mutation survived when selected only against end inputs: decodeEnd independently requires the expected field count, so that entry still refuses the added unknown key. This is a surviving mutation in that test scope, not a compile failure or a product defect. The regression assertion now also sends an unknown-only object to actual selection preparation, where the strict parser is the responsible boundary. Keep the original survivor artifact and separately record the strengthened test's result.

# Final verification

The strengthened five focused groups passed in1.437 seconds. Fourteen affected groups passed under the race detector in3.578 seconds. The original unknown-key survivor and its exit0 remain recorded; the same mutant then failed the new actual preparation assertion. Together with owner/scope error propagation this establishes three assertion-killed behaviors across four experiments, including one initial survivor. Every file was restored byte-for-byte and both private caches were removed before final checks. Independent actual-source review passed the two production files, new test groups and strengthened preparation assertion; reviewers executed no tests or mutations.

The full Go short suite passed75 packages; the server passed in199.482 seconds with99.7% native statement coverage:11,779 weighted statements and40 uncovered. This does not meet required100%. The server-only profile is an exact subset of the native all-package profile. Prior isolated collision child/combined profiles remain historical and were not merged into the changed-source result. All-package vet and Linux/Windows amd64 server-test compilation passed; foreign execution is unverified. The unchanged frontend and earlier browser/Console evidence were not rerun. The Go1.25.7 toolchain and standard-library object-key source hash are bound with107 frozen production files. Two documentation checks cover status only; root separately checks current metadata/links. [The lookup/parser manifest](lookup-parser-evidence.json) preserves scopes and artifacts. Bounded M04 aggregate reconciliation remains PASS; required100%, authorized merge/read-back and parent M03/M02 remain open.
