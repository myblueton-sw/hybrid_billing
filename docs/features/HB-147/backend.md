---
type: ImplementationRecord
title: Protected retained connection registration reader
description: Records the bounded GET metadata projection, public pin boundaries and isolated backend verification for HB-147.
status: draft
---

# Implemented scope

[HB-147](https://linear.app/hybrid-billing/issue/HB-147/bind-s01-to-protected-retained-collection-connection-metadata) adds `connection_projection.py` and fixed GET `/api/focus-connections` to the existing trusted-local `OverviewServer`. The backend implementer owns the projection, dispatcher, new isolated tests and this report; root owns frontend, retained-environment inspection and integration. The [approved design](connections.md) remains the contract. Actual-source independent review and integrated/browser acceptance are separate root handoff steps, not claimed by this implementation record.

The reader executes `collection.access(actor,'query')` and the current authority query transaction before projecting and delivering. Operator-only contexts are denied. Existing Host/Origin/FetchSite/custom-header, no-body, method, concurrency and no-store boundaries remain. New-path error and method responses use safe request IDs; successful evidence has no transient request ID. Both the projected body and dispatched wire response are capped at2MiB. Older overview, commitment, Marketplace and receipt routes retain their dispatch and error shapes.

# Projection and public identity

Only explicitly registered, entirely owned and uniquely mapped domain sets enter records, exact counts, search or public pins. Foreign, mixed, unmapped, empty-domain, duplicate-domain and inconsistent connection-ID records are excluded by the existing ownership helper. The guard may inspect protected control/configuration metadata internally; no original bodies, warehouse query, registration/add/definition command, account/domain list, raw configuration, credential, object path or capability payload is requested or serialized by this projection. Legacy deliveries never imply a registration.

Every owned exported record validates source-version registry support, day/month grain, registered UTC interval through the existing pure calendar validator, semantic-profile enum and producer integer limits. The calendar checks midnight/month boundaries and its3660-window limit. Invalid owned exported metadata fails closed. The exact10 record keys match the design; numeric registration values are canonical decimal strings, including explicit zero allowed lateness. Policy revisions use the existing producer's positive integer range below2^63.

`connections` and `capabilities` return the bounded registration selection. `security` and `diagnostics` are unsupported: total=null, records=[], next_after=null; they do not fabricate authentication or diagnostic results. Confirmed empty registration selections return total=0 independently of any legacy delivery evidence. Returned semantics do not certify supplier capabilities, FOCUS conformance, collection health or financial authority.

Public IDs and pins use only the exact approved allowlists. Record IDs hash profile/source/connection. Authorization scope hashes profile/source/fixed actor/audience/sorted opaque authorized member IDs. Permission revision hashes that scope and sorted member/policy-revision pairs. Snapshot hashes profile/permission revision/view/query/capabilities/exact total and all filtered sorted whitelisted records. Observation time and page offset are excluded so pinned pages remain usable. Hidden registration/capability/policy fields and foreign activity do not affect public pins. Visible record changes affect snapshot; policy revisions and current membership changes invalidate the corresponding authority pins. The current guard still executes per request.

The query accepts only view/q/after and paired SHA256 pins. Positive offsets require pins, stale pins conflict, more than10000 matching records reject rather than truncate, and each page contains at most25. The projection does not change stored state.

# Actual verification

Actual execution used macOS arm64 and Python3.11.7 with temporary pinned dependencies prepared by root. Tests use synthetic in-memory registration/configuration state, temporary guards and ephemeral loopback HTTP servers. No retained `.env`, `.local`, customer data, real account, database initialization, ingestion, reset, migration, publication or financial operation was used by this backend implementation or its tests.

| Command/scope | Actual result |
| --- | --- |
| New pure connection projection suite | PASS62 tests in the final combined run; ownership/configuration/membership negatives, hidden/foreign pin invariance, visible/policy/membership invalidation, strict query/scalar/calendar/offset/count/size bounds, confirmed zero versus unsupported null, guard nesting and forbidden operation checks |
| New ephemeral connection HTTP suite | PASS35 tests; closed route/query, duplicate security headers, Host/Origin/FetchSite/body/method guards, missing/stale pins, two-slot busy state, wire2MiB, sanitized errors and old-route dispatch/receipt-bound preservation |
| Combined new suites plus existing receipt, overview HTTP/projection/ports, commitment and Marketplace suites | PASS284 tests, zero failures, in50.08s |
| Python compilation and whitespace | `compileall` for new source/tests and `git diff --check`: PASS |

The combined command was `python -S -m pytest -q --import-mode=importlib tests/focus_connections tests/focus_receipts tests/focus_overview/test_http.py tests/focus_overview/test_projection.py tests/focus_overview/test_ports.py tests/focus_commitments tests/focus_marketplace`, with repository and root-prepared temporary dependency paths in PYTHONPATH. Root must provide its verified execution environment when reproducing; the temporary path is not a product installation dependency.

An initial sandbox run produced185 passes and97 setup errors because loopback bind was prohibited. Permission-enabled execution then produced281 passes and one incorrect test expectation: Python's HTTP client used the absolute URL hostname for Host, so the existing Host guard correctly rejected403 before path404. The test now distinguishes that request from a raw absolute URI with canonical Host, which rejects404. No security behavior was weakened. Two additional owned non-string grain/profile negatives were added; final284-test execution passed. These failures remain distinct from the final result.

# Handoff limits

No source comment/docstring, generated report, private environment file or existing test file is part of the implementation. Maintained files remain below1500 physical lines. This record does not claim actual retained registration counts, frontend/browser success, independent change review, owner FIX, production authentication/authorization, CSP diagnostics, Windows/Linux operation, PR merge or deployment. Root and the independent reviewer must verify those assigned scopes separately.
