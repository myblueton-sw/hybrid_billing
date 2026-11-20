---
type: ReviewRecord
title: Administration read-recovery accessibility
description: P01–P05 retry focus, localized read-completion status and stale-response checks.
status: draft
---

# Reproduction and implemented scope

HYB-313 remains in progress. Actual discovery across P01 installation, P02 integrations, P03 recovery, P04 retention and P05 release evidence showed Retry staying focused during a held read, then disappearing on success and leaving BODY active. No read-completion message was present; P01's unrelated proposal status is not evidence of refresh completion. Discovery is `/private/tmp/hyb313-admin-read-before/evidence.json`.

Following independent design and terminology review, Retry now synchronously focuses each screen's existing reload control. Explicit reloads set a boolean read-completion state only after the existing latest-request and current-session-scope checks. Every new read clears it. The separate, mounted status region uses current locale at render time; callback dependencies and background/command reload return contracts are unchanged. Terminal permission/authentication branches do not retain the region. Completion never explicitly moves focus. Command notices, refusal diagnostics and permission/data decisions remain separate and unchanged by the product diff.

The shared wording is “Data reloaded.” / “데이터를 다시 불러왔습니다.” / “أُعيد تحميل البيانات.” The [glossary](../../ui/terminology/product-glossary.md) defines it as a successful authorized read, not mutation, guaranteed freshness or operation completion. Existing product reload terminology and the official regional references in the plan informed the choice. Independent AI terminology review passed; human native-language and assistive-technology approval are not claimed. No new visual composition, stylesheet, asset or dependency was added.

# Actual behavioral verification

The final primary run `/private/tmp/hyb313-admin-read-final/evidence.json` passed 30 cases: five screens × three locales × desktop 1440/mobile 390. Every case covers initial unavailable, keyboard Retry, held loading, repeated 503, a settled failure, Retry success, toolbar reload and later keyboard movement while loading. It verifies the exact localized completion text and an actual empty→message transition for repeated reload, stable focus through Retry recovery, preservation of later focus movement, zero document overflow and no page errors or submitted account/domain commands.

Independent source review caught a harness race: the error button stays mounted during loading, so waiting for it after releasing 503 did not prove that failure had settled. The final runner waits for the injected response to finish and for the active screen's busy state to end before replacing the gate or asserting the next step. Earlier primary results are not final proof of repeated-failure settlement. The 60 rendered error/recovered captures reviewed below come from the earlier run; product code, layout and capture method are unchanged, while the final strengthened run separately establishes behavior.

The five-screen ordering run `/private/tmp/hyb313-admin-order/evidence.json` passed initial background silence, latest success followed by stale failure, latest failure followed by stale success, denied access after an authorized read, initial denied access and unauthenticated read. P05 specifically showed its revoked state after prior access and its denied state on an initial refusal. No terminal branch retained the success region. Status mutation counts were observed from before old-response release through 150ms, covering delayed publication checks. These injected 403/401 responses test client behavior; they do not claim a backend permission-policy change.

The final scope run `/private/tmp/hyb313-admin-scope-final/evidence.json` captured an actually authorized old read, held its response, signed out and signed in as the other seeded administrator, loaded the same screen for the new session, then released the old response. All five screens passed exact actor identity, unchanged displayed new-session scope facts and absence of old read-completion status. The source reviewer rejected an earlier overlapping regular-expression identity check and an overly broad “entire response ignored” label; the final evidence uses exact identity and specifically records announcement suppression plus stable displayed scope facts. It does not establish every hidden state field.

# Rendered and source review

Actual Claude CLI ui-designer reviews opened 20 error/recovered viewport images per locale in `/private/tmp/hyb313-admin-read-after/`: P01–P05, 1440/390, EN/KO/AR. All three returned scoped visual PASS without blockers. Focus rings were complete and readable around Retry in errors and Reload after recovery, and borders remained neutral. Header and lower-content regions outside the viewport were not re-certified. The new status text is visually hidden; browser assertions establish its actual text transitions, not the PNGs. The reports are `/private/tmp/hyb313-admin-{en,ko,ar}-visual.txt`.

Root inspected computed evidence across all 30 initial error captures: 60 buttons have neutral borders RGB 141,148,156, 30 error regions have a neutral top separator RGB 220,225,229 and no side/bottom border; 30 Retry controls show a 3px transient focus outline RGB 35,91,148. Existing P05 full-width mobile Reload and different diagnostic placement are recorded nonblocking composition observations. No new layout regression was found within the reviewed viewport scope.

Full TypeScript/build and 179 existing administration model tests passed. The tests cover the existing model behavior; the actual browser journeys provide evidence of focus and announcements. macOS arm64, Node 25.9.0 and Chromium 159.0.8154.6 were executed. Other browsers/platforms, native browser zoom, human screen-reader output and owner acceptance remain unverified. No whole-ticket completion, merge, deployment or user-runtime replacement is claimed.

# Reproduction

`node apps/console/tools/admin-read-recovery-qa.mjs` runs the primary matrix. `node apps/console/tools/admin-read-order-qa.mjs` runs ordering/terminal checks; add `--scope 1` for actual synthetic two-actor sign-out/sign-in checks. Supply explicit `--url`, isolated `--api`, disposable `--activation`, `--playwright-module`, `--chromium` and `--output`. Use a fresh disposable API for consumed activation links; scope mode activates both seeded administrators. Passwords and activation values stay in process memory. Generated JSON/images remain outside Git. Successful reads come from the disposable API; injected errors are browser-local.

Final independent source and technical-writing re-review returned scoped PASS after inspecting the corrected failure-settlement and exact-actor assertions, documentation and stored evidence. The reviewer did not execute checks or independently inspect screenshots. Root separately inspected the P05 Arabic mobile recovered image.
