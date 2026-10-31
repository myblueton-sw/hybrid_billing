---
type: Runbook
title: Design implementation and preview build guide
description: Reproducible local build, asset checks, rendered QA and handoff steps for the multilingual scalable console.
status: draft
---

# Prerequisites and ownership

Follow the repository ticket, machine claim, branch, scoped review and PR workflow. Read [active foundations](design-system.md) and [design work rules](../.agents/rules/design-system.md). Root integrates work; assign one writer per file and independent review for important boundary changes. Existing accepted modules remain active until specifically replaced by owner acceptance.

This runbook builds the dependency-free HTML/ES-module design preview, not a production React application. Node/npm and Python 3 are required. Browser QA additionally needs Playwright and a compatible Chromium executable. The verified environment uses Playwright 1.56.1 with Chromium headless shell build 1243; no automated remote CI or production build is implied.

## Prepare and build

Run from repository root:

```sh
npm ci --ignore-scripts --no-audit --no-fund --prefix tools/design
npm run check --prefix tools/design
npm run build --prefix tools/design
python3 -m http.server 8763 --bind 127.0.0.1 --directory output/designs/HB-63-console
```

Dependency preparation may require registry access. The lockfile fixes the four IBM Plex package versions and integrity hashes; review dependency changes and their licenses before updating it. After preparation, building and serving require no external runtime request. Do not substitute a font CDN if an offline build fails.

The build refuses unknown output files or symlinks, then rebuilds its prior manifest-owned output directory cleanly. It checks immutable foundation hashes, rejects source color literals/rounded controls/shadows/placeholder branding, copies the preview runtime files, including the contribution screen and canonical assets, resolves Fontsource CSS URLs into local font files and carries each package's LICENSE. `build-manifest.json` records font versions and hashes of generated files without timestamps, so identical inputs can be compared byte-for-byte. This manifest is evidence for the built artifact, not proof that every visual/workflow requirement passed.

Only `output/designs/HB-63-console` is the runnable artifact. Serving the source directory directly no longer resolves the generated foundation/font paths. Stop an older local server before reusing its port. Open `http://127.0.0.1:8763/?lang=ko#operations`; substitute `en` or `ar`, or use `#contents` for all routes.

## Change a screen or foundation

1. Locate the stable route in catalog.js and its baseline/registry mapping. Update purpose, fields, columns, actions and related links together. Do not represent catalog coverage as complete business-workflow behavior.
2. Edit shared visual tokens/assets at packages/design-system; do not patch generated output. Source SVG metadata is preserved. A changed canonical source needs updated provenance and explicit review; never merely update a hash to silence an unexpected mismatch.
3. Use the shared shell/widgets for paging, listboxes, selection and jobs. Preserve scope invalidation, honest count/error states, bounded rendering and keyboard focus. Local synthetic behavior must remain labeled.
4. Keep new text localized in English/Arabic/Korean, record regional terminology evidence, and identify native-review gaps. Do not hard-code a translated brand name or use machine translation as approval.
5. Rebuild, run checks appropriate to the change, inspect screenshots, and update the screen README/verification and this guide if commands or boundaries changed. Stage source, lockfile and notices only; never node_modules, generated fonts/screenshots or customer data.

## Browser verification

With the built preview server running:

```sh
HB_PLAYWRIGHT_MODULE=/absolute/path/to/playwright \
HB_CHROMIUM=/absolute/path/to/chromium \
node docs/design-modules/00-scalable-console/r1/verify.cjs
```

`HB_PREVIEW_URL` and `HB_QA_OUTPUT` override the endpoint/output location. Inspect the generated report and representative desktop/mobile screenshots in all three languages. Verify the logo loads unchanged and stays LTR, IBM Plex faces actually load, clear space remains visible, tables/pickers stay bounded, and no external runtime request or missing local asset occurs. Keyboard/focus, error/empty/loading, zero/unknown counts and expired selection need actual runtime checks. Full screen-reader, native-language and production API acceptance are separate.

## Failure and handoff

- Unexpected output files: inspect the named files and preserve any human-authored material outside the generated directory before rebuilding. Do not bypass the ownership check.
- Hash/style guard failure: compare canonical files with provenance and the reviewed source; correct the implementation or document a justified new decision. Do not bypass the guard or edit output.
- Missing font/package/license: run the locked dependency preparation, confirm paths/licenses, then rebuild. Do not claim offline readiness from a fallback font.
- Browser launch/runtime failure: record the actual failure; use the approved installed executable/environment, rerun the failed check and distinguish failed attempts from successful verification.
- PR handoff: include changed behavior, source provenance, actual build/browser outcomes and remaining limits. Keep the ticket open until authorized merge and merge read-back. Building this artifact grants no deployment or merge permission.
