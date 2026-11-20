---
type: VerificationRecord
title: License installation API and consolidated local preview
description: Authenticated durable installation binding, signed outcomes and recovery of collected S04 navigation from preview links.
status: draft
---

# API behavior

The optional `--license-installation` setting binds product `hybrid-billing` to a stable host-selected installation ID. Its state is under the private API data directory at `license-state`. Startup opens an existing matching store at the final startup step; absent storage remains uninitialized. A malformed, locked or wrong-bound store fails startup. Server Close releases ownership under the domain lock. No configured time authority or commercial rights are invented by the CLI.

`GET /api/v1/licenses` requires a live session, CSRF and installation-administration permission. Foreign installation workspace reads disclose no installation or status. The response distinguishes not_configured, uninitialized and configured. A configured authority is not proof of successful verification; actual mode/status/reason and last verified time come from a committed durable read. Returned entitlement/evaluation fields are explicitly selected; verification keys, raw signed input and filesystem paths are excluded. Read watermarks do not change command revisions; explicit commands do.

The existing command envelope now exposes `licenses.initialize`, `licenses.select_mode`, `licenses.register` and `licenses.activate` with target `license-installation`. It preserves permission, revision, idempotency, rationale and step-up checks. `humanActor` confirms the authenticated actor before constructing a domain human actor. Mode selection is only a declaration. Initialization of an already open matching store returns already_initialized. Signed input uses canonical base64 body/body_tail chunks (4096/1368 encoded characters, 4096 decoded bytes total), a 64-byte signature and explicit evaluation terms. This preserves the command framework's 4096-character per-value limit.

Registration returns its actual outcome, including invalid, unverifiable, scope_mismatch or accepted/repeat. Activation returns activation_recorded for a successful/repeated domain call, since an existing record may already be expired. Callers must read current status rather than infer active rights from command completion. Domain refusals retain their stable reason. An uncertain storage failure returns the framework's unknown-operation response with inspect_operation; it is not a settled refusal or an automatic safe-to-retry claim. No second status read obscures an already committed mutation. If initialization itself fails, the server latches license failure: subsequent reads return 503 rather than falsely reporting uninitialized, and mutations remain outcome-unknown until restart.

# Verification

- Nine focused license API tests with race detection: PASS, final root run 2.716 seconds. Coverage includes anonymous/CSRF/member denial; exact binding; initialization, declaration without grant and idempotent initialization; actual synthetic signed activation/registration with injected authority; terms mismatch, unknown key and scope mismatch; unavailable authority and last verified evidence; storage outcome unknown and poisoned reads; strict base64 bounds; command replay/stale revision/reauthentication; foreign workspace/target and OpenAPI.
- Full server lifecycle regression creates state through HTTP, closes the server, constructs/binds/starts a new server against the same data directory and reads the retained mode through HTTP. A wrong installation fails startup and leaves no license ownership marker.
- Nonhuman guard regression uses a nonexistent worker identity and verifies no state is created. Actual service-account rejection relies on the existing humanActor lookup contract; a new service-account HTTP fixture is not claimed.
- An initialization-failure regression creates a conflicting state directory after startup, verifies unknown-operation failure, then verifies that the read remains unavailable rather than reporting uninitialized. Independent race re-review passed in 1.520 seconds. This test covers the shared failure latch, not a simulated post-replacement initialization failure; uncertain commit phases are covered separately by durable-layer tests.
- CLI invalid binding test: PASS, 0.326 seconds. Independent reviewer ran the CLI test suite successfully in 3.029 seconds.
- Independent final change review: PASS. Reviewer reran focused license API race tests successfully in 2.519 seconds.
- Initial member test was invalid because it omitted password setup and received 401 before reaching authorization. The fixture was corrected; the final test authenticates the member and verifies 403. No failed run is counted as PASS.
- Broader initial `go test ./apps/api/server ./apps/api/cmd/hb-api`: server package INCOMPLETE/FAIL from the default 10-minute timeout in `TestScaleFilterListsOverHTTP` (9m52s in that test); stack was in identity membership reinstatement. The CLI package passed in 5.213 seconds. This unchanged scale test was not diagnosed as a regression or rerun with a larger timeout; no full-server-suite PASS is claimed. Final scoped API race checks above passed after fixes.
- `go vet ./apps/api/server ./apps/api/cmd/hb-api`: PASS.
- Final independent evidence review: PASS; reviewer viewed the integrated S04 screenshot with 1121 rows and separate currencies, and checked recorded 401/403 boundaries. Redirect/sign-in journeys remain root-executed evidence.
- Go 1.26.4 on macOS arm64. No production grants, deployment, issuer provisioning or real external time service was exercised.

# S04 local-preview correction

The user reached S04 through the separate P07 preview on port 8832 (revision 8a3e41ebfc55). That frontend used ungated reader proxies; the reader correctly refused requests without the Go gate's secret. The fix preserves the origin/secret boundary rather than widening the reader's access policy.

A local-only detached integration checkout combines HYB-335's sample sign-in fill with the reviewed P07 notice increment. Its source commit is 02bbc68 and it is not a main merge or published integration branch. The authenticated frontend on port 8830 now uses this checkout with HB_GATED=1. Old preview ports 8832 and 8833 redirect GET/HEAD to the authenticated console, preserving the requested route and selecting API mode. The Go API, storage and readers were not restarted or weakened.

Actual browser verification starts from the old 8832 link, confirms redirect to 8830, signs in using the local fill button, then loads overview and S04. Both return 1121 authorized rows across six providers, no origin_denied and no page errors. Anonymous access remains 401 and direct-reader access remains 403. The original React LICENSE is also served successfully on 8830. Non-sensitive results are in [integrated-preview-evidence.json](integrated-preview-evidence.json); screenshots remain local. The combined frontend passes typecheck and all 876 tests. These are local runtime facts, not deployed hosting changes.

# Remaining boundary

The CLI currently supplies no real time authority; nil remains unavailable rather than using server time. Issuer trust/key administration is not exposed and no key or grant is auto-seeded. Synthetic key provisioning exists only inside tests. Production authority/provider selection, issuer management and live P07 remain pending. Generic command receipts and identity sessions retain the existing API lifecycle; domain grant/document replay is durable, but generic receipt persistence across restart is not claimed. Existing local state cannot defend against an operator replacing the entire store with an older copy.

Owner Seung Woo Park; root writer; independent read-only review; machine mac_mini. Continue PR #242; no merge, deployment or ticket completion is implied.
