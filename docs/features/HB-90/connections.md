---
type: ImplementationDesign
title: HB-90 local connection scope inspection
description: Presents actual connection metadata and installed CSV contracts without implying external provider authentication or period completeness.
status: draft
---

# Scope and ownership

Root owns implementation and QA. Independent authority and planning/UX reviewers gave conditional design PASS. Pending HB-94 PR73 is the explicit dependency. This increment implements S01 read-only local connection detail; S02 onboarding and configuration mutation remain separate work. The fixed process identity does not establish production OP/BL authorization.

# Authority and data

A separate metadata-inspection transaction maps loader to the existing ingest right and approver to confirm. Existing inspection signatures and confirm requirements remain unchanged. Each request uses server-derived RLS scope, advisory lock, read-only transaction, active capability and final current-right/capability recheck. Readers and exact foreign connection IDs are denied. No request can choose an actor or required right.

The response projects connection/account/subaccount, configured period, policy revision, installed local CSV registry versions/revisions/digests and admission limits. The latest registered candidate is selected by creation time and UUID, separately from quarantine and unresolved-operation flags. Its receipt timestamp is not the latest successful collection or latest quarantine attempt. No monetary controls, rows, secrets, endpoints or original paths are returned.

Configured period is not proven data coverage. Completeness is unknown; external authentication, provider reads and invoice retrieval are unconnected. Refresh is a metadata read, not a provider diagnostic. Fixed route enums derive from current rights; destinations independently reauthorize. Loader links to S03; approver links to S04 and S06, carrying the latest candidate ID when available.

# Screen contract

Keep the canonical full-width shell, neutral square borders, local fonts, EN/KO/AR and owner-only review. Use connection identity and scope first, installed contracts next, then latest registered candidate and separate holds, followed by real next actions. The previously consulted Cloudscape [details-page pattern](https://cloudscape.design/patterns/resource-management/details/details-page/) informs hierarchy; no external code or visual theme is imported.

| Item ID | Behavior |
| --- | --- |
| S01:scope:connection | Exact server connection, period, policy revision and observation time |
| S01:data:input-support | Installed CSV contracts and local admission limits, not provider support |
| S01:data:last-reception | Latest registered candidate only; separate quarantine stream |
| S01:state:holds | Quarantine and unresolved operations retain their actual meaning |
| S01:action:next | Current-role source workflow links; destination reauthorization |
| S01:action:refresh | Read current metadata, clear old data during failure/denial |
| S01:state:unavailable | Missing/foreign/invalid response fails closed |

No configuration, credential replacement, revocation or diagnostic buttons are faked. Drafts, provider matrices, live external tests, retention and go-live decisions remain unconnected. New implementation/review scope does not inherit owner FIX.

# Validation plan

Actual isolated database reads for loader/approver, reader/foreign-ID denial, rights revocation, inactive capability, final policy recheck, read-only write rejection and transaction scope reset; metadata projection and route partitions; S04/S05/S06 authority regression; malformed/stale/error/late UI responses, exact navigation and EN/KO/AR desktop/mobile rendering. Runtime and independent change/visual review outcomes will be recorded separately. No main merge or cross-OS execution is claimed.

# Implementation and review evidence

Implemented the scoped read model and canonical S01 live screen in the onboarding module. The latest-candidate projection excludes row count as well as monetary/content fields following independent boundary review. Existing confirm-only inspection remains unchanged. A new read-only metadata path checks current ingest/confirm rights without giving loaders cost access.

Independent backend review initially rejected the additional row-count projection. It was removed from SQL, response and UI; targeted re-review PASS. Independent frontend source review PASS for exact identity, clearing/refresh, role routes, canonical ownership and meaning boundaries. No owner acceptance was created.

Actual verification on macOS arm64, Python 3.11.7, Node 26.7.0 and Chromium headless shell 1243:

- Connection tests: 7 PASS, including actual database metadata, loader/approver partition, reader and foreign-ID denial, missing rights/inactive capability, current-right/policy recheck, read-only write rejection, transaction-local scope reset, unchanged release count and HTTP origin/method controls.
- Existing S04/S05/S06 backend regressions: 24 PASS. Loader still cannot use confirm-only inspection.
- Console tests: 69 PASS, including malformed metadata, action sets and exact candidate handoffs.
- Typecheck/build: PASS; existing bundle-size warning remains.
- Browser QA: six EN/KO/AR desktop/mobile renders PASS. Actual metadata, exact IDs, keyboard refresh, live S03/S04/S06 navigation, empty latest-candidate state, denied/policy/malformed clearing and retry, late response discard and foreign/invalid IDs checked. No owner review confirmation was executed.
- Claude CLI visual review: initial FAIL for incomplete loading captures and RTL limit layout; fixed refresh capture synchronization, localized separate limit rows and UTC-second display. Four populated desktop/mobile captures received visual re-review PASS. Keyboard behavior and runtime font size were checked by browser QA, not inferred from images.
- Independent final source re-review caught malformed timestamp rendering; explicit-zone API validation, defensive display and regression coverage resolved it. Targeted re-review PASS with three API tests independently executed.

Evidence is generated under `output/validation/HB-90` and excluded from the commit. Full S01 mutation/diagnostic functionality, production OP/BL permissions, provider capability tests, native-language/screen-reader acceptance, large-scale performance and Windows/Linux operation remain unverified. No main merge or deployment is implied.
