---
type: VerificationRecord
title: P07 original notice serving and expiry copy
description: Scoped foundation repair before full license service binding.
status: draft
---

# Delivered boundary

The first HYB-328 increment fixes original frontend dependency notices in local development and corrects expiry guidance. The server returns only seven exact allowlisted LICENSE files as plain text; builds emit the same original buffers. Unknown notice paths return 404, unsupported methods 405, and HEAD has no body. Missing original files fail startup/build rather than inventing a notice. Source revision hashing includes the helper.

The changed EN/KO/AR guidance says expiry warnings do not block workflows or existing permitted access, while continued technical access does not extend commercial rights. No layout or licensing permission changed. P07 remains a preview: no entitlement API, registration, activation, support contract or original plugin notice source was added. Full HYB-328 remains open.

# Actual verification

- `npm ci --ignore-scripts --no-audit --no-fund --prefix apps/console`: PASS with locked dependencies.
- `npm run test --prefix apps/console`: 874 passed, zero failed. Includes actual HTTP original-byte checks for seven files, HEAD, unsupported methods, unknown/traversal paths and absent source files.
- `npm run build --prefix apps/console`: PASS, including TypeScript. Existing chunk-size warning remains.
- All seven emitted files match installed originals byte for byte. Actual GETs from development port 8832 and Vite built preview port 8833 also match; SHA-256 values are in [notice-evidence.json](notice-evidence.json).
- Chromium 153.0.8010.12 on macOS: KO/EN/AR at 1440 and 390 pixels. Each journey opens an original notice in a new tab, verifies its text, searches for an absent package, clears the search, and confirms all seven links return. No page errors or document overflow. Six rendered screenshots retained locally at `/private/tmp/hyb328-qa`.
- Independent change/security/TW review: PASS; reviewer independently executed the two focused tests. Native-language and assistive-technology acceptance are not claimed.
- Independent rendered review: PASS after viewing all six screenshots; no clipping/overlap, notice links readable and RTL order intact. Browser execution remains root evidence.
- Initial built-serving attempt used a nonexistent npm preview script and failed; corrected to the installed `vite preview --port 8833 --strictPort`, then all actual built HTTP checks passed.

# Remaining work and handoff

See [plan.md](plan.md) for durable installation identity, status-read persistence, time authority, replay/restore and original notice artifact prerequisites. Domain registries existing in memory do not establish these runtime contracts. Missing original planning inputs are recorded, not reconstructed. No real entitlement or legal approval was created.

The separate preview is `http://127.0.0.1:8832/?lang=ko#/licenses?view=notices&lang=ko`. The existing authenticated FOCUS preview on port 8830 remains running and unchanged. Development and built-preview checks do not establish Windows/Linux execution. Root owns this change; owner Seung Woo Park; machine mac_mini. PR review does not authorize merge or ticket completion.
