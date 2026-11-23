---
type: VerificationRecord
title: M01 document evidence impact terminology and rendered verification
description: Actual pending document counts and scoped multilingual display evidence.
status: draft
---

# Scope and ownership

[HYB-340](https://linear.app/myblueton/issue/HYB-340), machine:mac_mini. Root owns implementation and integration. Independent refusal_review inspected design, source and the browser fixture; platform_refusals inspected terminology and evidence. Actual Claude CLI ui-designer supplied composition guidance and rendered re-review. This increment applies the installed Hybrid Billing design completion loop to the existing M01 bulk-removal preview.

M01 now labels document_evidence_proposals separately from document permission proposals in English, Korean and Arabic. Its note describes the original proposer's permission recheck at document confirmation, rather than financial approval. An unconfirmed recheck condition explicitly remains unknown. Existing full-width layout, neutral separators, typography, controls and CSS remain unchanged. The glossary records the separate meanings and excludes confirmed/revoked documents from the pending count.

The existing M01 impact list is the affected review scope; this change introduces no separate screen or owner acceptance. Root consulted the [Carbon data-table guidelines](https://www.carbondesignsystem.com/building-blocks/core/components/data-table/guidelines) for structured readable rows and available width. No third-party assets or styles were copied. Claude recommended preserving the existing composition and natural wrapping.

# Actual browser fixture

The opt-in TestDocumentEvidenceImpactRenderedBrowser uses authenticated real public commands and the ordinary sample runtime. It creates one pending, one confirmed and one revoked document for the selected proposer, plus two pending document permission proposals. The actual preview therefore reports one document proposal and two permission proposals. It prepares access-removal impact without executing removal. The runner blocks and counts bulk execution requests; the final count is zero. The fixture also checks unchanged pending-document count and zero ERP ownership after rendering.

Eighteen cases cover English, Korean and Arabic at 1440, 1024 and 390 pixels. Nine use the actual server rechecked=true response. Nine inject rechecked=false into the preview response solely to verify the unknown-condition display; these do not establish a different server confirmation policy. All cases assert distinct counts, translated labels, note text, direction, page overflow, local row clipping and note weight. Synthetic authentication stays in process memory/environment and is not written into evidence artifacts.

# Findings and verification

Console tests passed all 945 tests and the production build passed, with the existing large-chunk advisory. Logs: /private/tmp/hyb340-document-impact-console.log and /private/tmp/hyb340-document-impact-build.log.

The initial nine actual-response browser cases passed in 7.707s. The first eighteen-case run failed in 7.363s because its false-condition assertion incorrectly expected weight 700. Inspection showed that the unchanged existing live-strong style uses 600. Root corrected the runner to that existing style and added explicit local clipping assertions; no production CSS changed. The corrected eighteen-case run passed in 12.554s, with no errors, no overflow, zero local clipping and zero bulk execution requests. Final evidence: /private/tmp/hyb340-document-impact-ui-reviewed/evidence.json and its 36 context/row images. Final log: /private/tmp/hyb340-document-impact-browser-reviewed.log. The initial failure remains in /private/tmp/hyb340-document-impact-browser-final.log.

Source-rule tests passed in 0.296s and server vet passed without diagnostics. Logs: /private/tmp/hyb340-document-impact-source.log and /private/tmp/hyb340-document-impact-vet.log. Production Go code did not change in this increment; the full Go suite was not repeated. The preceding service checkpoint's final full-suite evidence remains in [its verification record](document-service-verification.md).

Independent source and terminology reviews passed. Platform_refusals directly inspected twelve images: true and false conditions in all three languages at 1440 and 390, plus the full eighteen-case evidence. Actual Claude ui-designer inspected all eighteen full-context images and four row crops, and returned scoped visual PASS with no blocker. Its optional English wording observation was nonblocking; technical-writing review accepted the explicit unknown wording. Reviewers did not execute tests or claim native-language sign-off. Claude evidence: /private/tmp/hyb340-document-impact-design.txt and /private/tmp/hyb340-document-impact-render.txt.

# Remaining work

Document proposal, independent confirmation, revocation and internal ERP preparation screens remain pending. Independent source investigation identified protected document-form context and request-scoped history discovery as prerequisites: existing financial review lacks form line data, candidate detail exposes broader supplier-cost information, and current inspections require already-known document/preparation identities. These are design findings, not approval of an unimplemented public contract. Next work must review that bounded contract before implementation, then use Claude-led composition and real authenticated browser journeys.

Only macOS execution is verified. Native-language sign-off, owner screen acceptance, remaining W-23 scope, merge and deployment are not claimed. The parent goal and ticket remain active.
