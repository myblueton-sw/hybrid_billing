---
type: VerificationRecord
title: P07 public verification-key workflows
description: Known-key and proposal lookup, explicit dual-controlled changes and session-safe UI verification.
status: draft
---

# Scope and contract

The existing Offline view now contains a separate verification-key section with four initially unselected tasks. Only the selected panel renders. Users can read a known key ID or canonical positive uint64 proposal number; there is no invented inventory or completeness claim. Read responses must match the requested identity, kind-specific action, actor/state relationships and revision. Missing records are not proof of withdrawal: the domain removes withdrawn proposals.

New-key proposals accept at most 4000 bytes of JSON containing exactly key_id, issuer, public_key, proof, not_before, not_after and accept_until. Public bytes and possession proof are canonical lowercase hexadecimal of 32 and 64 bytes respectively. Time bounds are integer UTC Unix seconds within the producer range. The preview hashes decoded public bytes with SHA-256; format checking does not certify possession proof or issuer authority. No private-key generation or signing is implemented in the product. Replacing a file clears confirmation.

Retirement and compromise first require a successful key lookup. The selected key and read revision bind the mutation; changing the lookup removes the actionable form. Signing validity and acceptance deadline are distinct. Retirement requires the server's authoritative time; no browser-clock inference is made. Compromise is explicitly consequential, irreversible through this screen and distinct from record deletion. A pending proposal offers only the actor-appropriate approval or withdrawal action, with express confirmation and reason. Server human/permission/step-up/revision/hold/dual-control checks remain authoritative.

All nine license commands share the existing session controller. Pending and unknown results block other license mutations across navigation; unknown results hide mutation forms while keeping permitted read-only lookup available. Command success is validated per operation, including proposal number/target. Settled outcomes retain their request diagnostics through subsequent read failure. Parent refresh/access changes clear key facts, confirmation and rationale and invalidate late queries. Read epochs prevent using a pre-settlement revision before effect-driven refresh completes. Confirmation must be repeated after revision refusal or a fresh read. Full-document reload remains outside the in-memory uncertainty guarantee.

# Design and review

Applied the owner-requested hb-design-completion-loop and existing React/screen regression skills. Actual Claude CLI ui-designer guidance selected progressive task radios, neutral summary lists, full-width layout, bounded inputs and a separate public-key section below entitlement submission. Existing IBM Plex, canonical branding and tabs remain. Official GOV.UK summary-list/error-summary references remain the recorded external basis. A redundant separator above the entitlement form was removed as part of the scoped visual extension.

Root reconciled recommendations with the producer: proposed/decided timestamps and proposer rationale are not returned, so none were fabricated. Withdrawn read records are not invented. Inapplicable approval/withdrawal buttons are omitted with the dual-control explanation; the server enforces the rule. The fingerprint is computed from public bytes, never from raw proof text. Terminology candidates and regional source limits are in the product glossary.

Independent design review PASS. Initial code review found parent refresh retaining stale key facts and confirmation; corrected parent-phase invalidation and settlement epochs passed independent code/TW re-review. Independent targeted trust/parser/session tests: 21 PASS in 143.5 ms. Earlier merge review also ran 25 targeted checks including notice serving and local-login boundaries. Initial visual review found lookup-button alignment and ambiguous task labels. Scoped margin correction and explicit lookup/proposal labels were applied; requested-action nouns and a local unknown-result notice were also clarified. The revised matrix passes, including input/button bottom alignment within two pixels and enabled read-only lookup under uncertainty. Final actual Claude CLI visual re-review PASS on six final renders; both blockers were visibly corrected with no new blocker. The reviewer inspected screenshots and prior findings, not runtime checks or the final source; none of these reviews grants owner FIX or native-language acceptance.

# Executed verification

- Merged origin/main at 9ecc8a5 into the existing ticket branch, preserving both test inventories, exact original notice serving and main's local login boundary. Combined baseline: 927 tests and build PASS.
- Final frontend after the alignment/copy fixes: 937 tests PASS, 10439.940958 ms; TypeScript/build PASS. Existing Vite configuration and large-bundle warnings remain. No dependency change.
- Focused real API regression: `go test ./apps/api/server -run '^TestLicense' -count=1` PASS, 2.815 s. No backend contract was changed by this screen increment.
- Actual app with isolated Go API: public-key proposal with synthetic in-memory signing fixture, proposer-only withdrawal UI, a second administrator's approval, exact fingerprint read, missing-time refusal for retirement, compromise proposal and proposer withdrawal followed by not_found. Six recorded journey checks PASS, no page errors. No real issuer or customer data was used.
- Controlled responses through actual app: eight checks PASS. Global refresh with unchanged revision clears confirmation; parent 403/503 clears facts; target edit removes reviewed action; a late lookup cannot restore facts after denial; pending/unknown trust commands lock entitlement and key submissions across navigation; revision refusal requires fresh unchecked review; mismatched server record fails closed.
- Existing P07 async regression: 11 checks PASS after integration, including duplicate submission, late previous-session responses, uncertainty across view/locale changes, fresh-login isolation, retained completed outcome after read failure, and terms-consent reset.
- Render matrix: 36 task captures across EN/KO/AR at 1440/1024/390, plus nine error/focus and nine unavailable variants. Four EN/AR desktop/mobile unknown-result captures supplement the matrix. No horizontal overflow or page errors. Error summary links focus the confirmation control without changing the SPA URL; keyboard Space toggles it.

Browser: Chromium 159.0.8154.6 via Playwright, Node 25.9.0, macOS arm64. Authentication and successful two-person operations use the isolated Go service on 8840; failure and layout cases inject declared synthetic responses. Test-generated signing keys exist only in harness memory. Initial development-server renders were blank after synchronization; restarting with dependency re-optimization restored the app. A first real-flow harness run did not reach the expected compromise response and did not retain its refusal body, so the exact cause was not established. The reviewed settlement-generation guards were tightened, and the complete flow passed in the final run. Failed attempts are not counted as PASS.

# Remaining boundary

External time authority, legal issuer provenance, original plugin notice sources and support/Marketplace producers remain pending. Existing user API, FOCUS readers and preview on 8820/8806/8810/8830 were not replaced. The new UI is tested in the isolated stack, not asserted deployed to the user's current session. Prior general-server scale timeout remains separately unverified. Human native-language and assistive-technology certification remain unverified. No merge/deployment/owner acceptance or ticket completion is inferred.

HYB-328; owner Seung Woo Park; machine mac_mini; root writer/integrator; independent read-only code/TW reviewer and actual Claude CLI visual lead; PR #242.
