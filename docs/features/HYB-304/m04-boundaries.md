---
type: ValidationRecord
title: HYB-304 service query boundary and reachability verification
description: Records additional cursor, numeric, catalog and authentication lifetime tests without changing the package coverage target or fabricating reachable production failures.
status: draft
sources:
  - resource: ./m04-consumer.md
  - resource: ./interaction-evidence.json
  - resource: ./m04-queries.md
  - resource: ./boundary-evidence.json
---

# Scope and ownership

This follows pushed/read-back checkpoint1e5ecffc191d0e97e377946b2bff251f8a49a72d on feat/HYB-304-identity-list-paging, HYB-304, machine mac_mini. Root owns four new server test files and this evidence; /root/hyb310_review independently inspected their source and reachability classification. No production Go or Console implementation, operational policy or customer data was changed. Synthetic fixtures use actual policy and lifecycle operations rather than corrupting registry maps. Existing full-feature PR/merge and the package100% requirement remain pending.

The prior full server short run measured98.1%, with229 uncovered weighted statement units across the package;99 were in servicequery files. These are statements reported by Go coverage, not missing user scenarios or branch-completeness counts. This work first adds meaningful public/component/helper tests, then reruns the server suite. A targeted subset's23.3% coverage is not a fresh full-package measurement.

# Tests and actual boundaries

| Test group | Input and independently checked result | Boundary |
| --- | --- | --- |
| Cursor decoder | Missing/extra segments, invalid base64, mismatched signature, authenticated malformed JSON, unknown fields, trailing objects/text, invalid position IDs, impossible aggregate fields and invalid reference interval are rejected; valid round trip and trailing whitespace retain exact fields | Authenticated malformed bodies are deliberately minted by the test using the actual signer. They test helper robustness; an HTTP caller is not claimed able to forge them. No key or signed cursor is printed or committed |
| Cursor encoder | Out-of-range years in reference, expiry or aggregate timestamp produce no continuation | time.Time serialization is a genuine fallible operation. These helper inputs do not claim normal HTTP clocks reach those years |
| Numeric saturation | Last representable active/expired/revoked aggregate and matching count remain exact; the next operation refuses and leaves every field unchanged | Pure scalar boundary values, without allocating an impossible number of rows or modifying registry maps |
| Catalog projection | Actual list/detail preserve named customer/legal-entity/contract IDs and exact2026-11 through excluded2027-01 period; nonmatching organization/scope searches return exact zero | Declaration metadata, not effective grants or request authentication |
| Ineligible/unavailable choices | A configured missing-customer scope remains known/ineligible but is omitted from eligible choices; supported contract-read dependency loss refuses the list without partial rows and marks known metadata unavailable | Configured existence, business eligibility and dependency evidence remain separate |
| Canonical organization/history | Suspended organization remains known/ineligible; missing scope/audience/history and malformed history targets share refusal without metadata | No target-existence or global permission-loss inference |
| Legacy account | Public identity creation without a declaration permits canonical declaration repair, reports missing metadata and refuses issuance without adding credentials | No invented empty purpose/scopes, ownership change or effective grant |
| MFA | Actual password sign-in after publishing the MFA policy keeps safe account/credential reads available but requires every applicable sensitive-action flag present and false, with prerequisite_failed separate from execution readiness | Password confirmation is not an MFA substitute. A pre-policy federated fixture was correctly refused by the password-only policy and was replaced with the actual permitted sign-in method |
| Session lifetime | Obtain actual query/detail admission, perform actual HTTP logout, then attempt publication: exactly one401 unauthenticated JSON response, no trailing response or projected metadata; the ended verifier cannot obtain source/admission/step-up readiness | Controlled component-boundary interleaving around a real logout, not a claim of concurrent complete HTTP requests or internal-record corruption |

All eleven new top-level tests passed unchanged production source. A helper call-signature error and the policy/login fixture mismatch were test setup errors, not product fail-before evidence. Independent review initially failed the new MFA assertion for accepting omitted flags and the logout assertion for checking only401. Corrections require all applicable flags and exactly one error object without row/rows/count/admission/eligibility/catalog_version/limits. Narrow independent test-source re-review passed; it did not execute tests.

# Mutation evidence and restoration

Removing strict unknown-field rejection made the authenticated malformed-cursor assertion fail. Disabling the aggregate saturation guard made the no-wrap/no-partial-change assertion fail. Both actual semantic mutants were killed; original source bytes/SHA were checked after restoration and each batch's disposable private Go cache was deleted.

The first aggregate trial removed the only math use and failed compilation. It is excluded from the semantic kill count. The corrected comparison mutation compiled and failed the intended aggregate assertion. The successful decoder mutant did not need to be rerun merely because that separate trial was invalid. No mutation remained live when the restored tests or broader checks started.

# Reachability classification and unresolved gate

Independent read-only analysis identified these limits at the98.1% profile. They neither narrow nor waive the package100% target:

- serviceBasis marshals only concrete boolean/string/unsigned-integer fields, including the two unsigned ServiceQueryVersion fields. Its current JSON returned-error branch has no reachable input of those types; allocation failure panics rather than returning that error. The cursor's time.Time fields are different and are exercised above. Do not invent corrupt maps or remove a security guard merely to color a coverage line.
- Successfully admitted session owner/handle invariants follow actual openSession/bind and identity admission. Ordinary identity/epoch/session loss fails earlier. A foreign-workspace test needs a genuinely admitted foreign administrator with real grant/policy evidence; changing a query workspace is not that test.
- Authentic continuations bind source generations. Public lifecycle operations retain recorded service keys. Completed-greater-than-total and missing continuation service guards are not ordinary external cursor attacks. Legitimate module authority loss between calls can still exercise propagation/refusal.
- Catalog-unavailable constructor guards assume an immutable validated catalog. Nil-pointer replacement is not a meaningful HTTP scenario. MemberPositions.More with zero scanned positions contradicts its positive-limit indexed-slot contract.

The fresh full server short suite passed at98.4% statement coverage. The profile retains192 uncovered weighted statement units, including62 in servicequery files; the package100% target remains unmet. All eleven new tests passed under the race detector, and go vet passed for apps/api/server. Linux amd64 and Windows amd64 server test binaries compiled successfully; their ELF/PE formats were inspected. These are compile checks, not Linux/Windows runtime success. Console source is unchanged from the preceding998-test/sixty-render checkpoint and those checks were not rerun for these Go-only additions. The [boundary evidence manifest](boundary-evidence.json) binds sources and local artifacts without replacing the earlier interaction evidence. M04's remaining actual journey/form/auth-notice verification, final reviews, PR, authorized merge and remote/main read-back still follow; HYB-304 M03/M02 remain afterward.
