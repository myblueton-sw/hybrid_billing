---
type: DesignPlan
title: M04 rotation follow-up from the HYB-304 integration reviews
description: Presentation and checking corrections to the credential rotation of the rebuilt M04 screen, taken from the non-blocking findings of the two reviews of PR 251.
status: draft
---

# Scope and ownership

HYB-368, machine:mac_mini. Console presentation, one console check and one module test. No read, command or field of the service changes.

# Built

- The result sentence of a rotation shows its three identifiers in mono, left to right and unbroken.
- The 24px above the rotate form's title now applies; the earlier rule lost to a more specific one and had no effect, on the form as on the result panel.
- The identifier in "Replaced by" and "Replaces" of a credential's detail is a button styled as an underlined mono link, 44px tall, that selects that credential. In the list it stays text.
- The credential and account views show validity, observation, reference, pickup and declaration times in the screen's date format with UTC, as the review step does, instead of raw timestamps. The format shows minutes and no seconds; where a time stands alone (observation, reference, pickup, declaration) the exact value is kept in the element's `datetime` and `title`. A time that cannot be parsed is shown as received.
- The rotate form checks its request against the served limits with the fields it sends, including the overlap (`admitsRotateDraft`).
- Module test: replacing, revoking and pruning over more rounds than ended credentials are kept leaves the live count at zero, the bounded pages equal to the whole list without repeats, and room for exactly the limit of new credentials.

# Not done

- A larger gap before the issuance target group: the stack already separates its parts by 40px, measured between the credential detail and what follows; an added spacer doubled it and was removed.
- "Cancel rotation" as a separated link-style button: it already uses the link-button class of the service; how that class renders in this screen is the rebuilt screen's baseline.
- A refusal of a plain issue for the same account can still appear in the rotate form; the store keeps one refusal per account and does not say which kind of attempt it answered.
- The rotate form still does not keep an unsent draft across navigation.
- The other `m04-*.mjs` runners were not rerun, and rotation under a trust hold or with step-up was not exercised in the browser.

# Verification

`modules/identity` tests pass with the new test. Console TypeScript clean; 1477 tests, 1472 passed, 5 skipped. `tools/m04-rotation-qa.mjs` against a local service with a service catalog: 55 checks passed, in Korean and Arabic at 390 and English at 1440, each rotating a real credential.

# Reviews

- Independent review (role-based AI reviewer separate from the writer, by reading), 2026-11-26: approve; its note on seconds is taken up above.
- ui-designer rendered review, 2026-11-26: the first captures failed three of five items (the link drawn as a bordered button, one raw time left under the list, the added spacer producing about 120px); corrected as described. Not owner acceptance.
