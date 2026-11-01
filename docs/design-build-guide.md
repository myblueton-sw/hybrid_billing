---
type: Runbook
title: Design implementation and preview build guide
description: Reproducible local build, asset checks, rendered QA and handoff steps for the multilingual scalable console.
status: draft
---

# Prerequisites and ownership

Follow the repository ticket, machine claim, branch, scoped review and PR workflow. Read [active foundations](design-system.md) and [design work rules](../.agents/rules/design-system.md). Root integrates work; assign one writer per file and independent review for important boundary changes. Existing accepted modules remain active until specifically replaced by owner acceptance.

This runbook builds the dependency-free HTML/ES-module design preview, not a production React application. Node.js 22+ and npm are required for building and local serving; the viewer needs only a compatible browser. Browser QA additionally needs Playwright and a compatible Chromium executable. The verified environment uses Playwright 1.56.1 with Chromium headless shell build 1243; no automated remote CI or production build is implied.

## Prepare and build

Run from repository root:

```sh
npm ci --ignore-scripts --no-audit --no-fund --prefix tools/design
npm run check --prefix tools/design
npm run build --prefix tools/design
npm start --prefix tools/design
```

Dependency preparation may require registry access. The lockfile fixes the four IBM Plex package versions and integrity hashes; review dependency changes and their licenses before updating it. After preparation, building and serving require no external runtime request. Do not substitute a font CDN if an offline build fails.

The build refuses unknown output files or symlinks, then rebuilds its prior manifest-owned output directory cleanly. It checks immutable foundation hashes, rejects source color literals/rounded controls/shadows/placeholder branding, copies the preview runtime files, including contribution, onboarding, analysis and scoped journey screens plus canonical assets, resolves Fontsource CSS URLs into local font files and carries each package's LICENSE. `build-manifest.json` records font versions and hashes of generated files without timestamps, so identical inputs can be compared byte-for-byte. This manifest is evidence for the built artifact, not proof that every visual/workflow requirement passed.

Only `output/designs/HB-63-console` is the runnable artifact. Serving the source directory directly no longer resolves the generated foundation/font paths. Stop an older local server before reusing its port. Open `http://127.0.0.1:8763/?lang=ko#operations`; substitute `en` or `ar`, or use `#contents` for all routes.

## Change a screen or foundation

1. Read the [screen-review rule](../.agents/rules/screen-review.md). Check the current screen/revision acceptance and any owner-authorized change scope before editing. Locate the stable route in catalog.js and its baseline/registry mapping. Update purpose, fields, columns, actions and related links together. Do not represent catalog coverage as complete business-workflow behavior.
2. Edit shared visual tokens/assets at packages/design-system; do not patch generated output. Source SVG metadata is preserved. A changed canonical source needs updated provenance and explicit review; never merely update a hash to silence an unexpected mismatch.
3. Use the shared shell/widgets for paging, listboxes, selection and jobs. Preserve scope invalidation, honest count/error states, bounded rendering and keyboard focus. Local synthetic behavior must remain labeled.
4. Keep new text localized in English/Arabic/Korean, record regional terminology evidence, and identify native-review gaps. Do not hard-code a translated brand name or use machine translation as approval.
5. Keep stable implementation-item IDs and mark a changed fixed-screen candidate for renewed owner review, including affected shared-component consumers. Do not assign owner FIX from a build or automated test.
6. Rebuild, run checks appropriate to the change, inspect screenshots, and update the screen README/verification and this guide if commands or boundaries changed. Stage source, lockfile and notices only; never node_modules, generated fonts/screenshots or customer data.

## Browser verification

With the built preview server running:

```sh
npm run qa --prefix tools/design -- --suite all --playwright-module "path to installed playwright" --chromium "path to chromium executable"
```

The portable runner accepts `--url` and `--suite generic|journeys|review|all`. Its optional `--playwright-module` and `--chromium` arguments avoid shell-specific environment assignments. Playwright/browser installation is a QA prerequisite only, not required on a viewer's device. The generic, journey and screen-review suites write separate evidence under `output/validation`; review tests use isolated storage and do not create owner acceptance. See [tooling](../tools/design/README.md) for local HTTP options and host-test limits.

Inspect the generated report and representative desktop/mobile screenshots in all three languages. Verify the logo loads unchanged and stays LTR, IBM Plex faces actually load, clear space remains visible, tables/pickers stay bounded, and no external runtime request or missing local asset occurs. Keyboard/focus, error/empty/loading, zero/unknown counts and expired selection need actual runtime checks. Full screen-reader, native-language and production API acceptance are separate.

## Failure and handoff

- Unexpected output files: inspect the named files and preserve any human-authored material outside the generated directory before rebuilding. Do not bypass the ownership check.
- Hash/style guard failure: compare canonical files with provenance and the reviewed source; correct the implementation or document a justified new decision. Do not bypass the guard or edit output.
- Missing font/package/license: run the locked dependency preparation, confirm paths/licenses, then rebuild. Do not claim offline readiness from a fallback font.
- Browser launch/runtime failure: record the actual failure; use the approved installed executable/environment, rerun the failed check and distinguish failed attempts from successful verification.
- PR handoff: include changed behavior, source provenance, actual build/browser outcomes and remaining limits. Keep the ticket open until authorized merge and merge read-back. Building this artifact grants no deployment or merge permission.


## OS neutrality and screen review

The [platform-independence rule](../.agents/rules/platform-independence.md) applies to installation and operation. Portable npm commands replace the required Python/OS-specific server path. The included server binds loopback and serves only manifest-owned files; a normal static host can serve the built artifact separately. Windows/Linux are design targets, not verified host executions. This frontend design does not certify production installers or backend runtime portability.

Every route exposes screen-design review controls and a current-state item-ID list. Local confirmation is scoped to the source-derived revision, locale, layout, exact width, visible state and item IDs. Records remain in the browser and can be exported for ticket/Git reconciliation. No local mark is authenticated project acceptance. Changed revision fingerprints preserve old evidence and request impact assessment; no-impact dependency changes need recorded assessment rather than automatic owner reapproval. Keep review tests isolated from the owner's browser profile. Build-generated revision metadata is derived from normalized source and shared dependency hashes; it is not manually edited.
