---
type: DesignPlan
title: Per-screen frontend design and portable review preview
description: Defines the full screen-design implementation, item identity, local owner review evidence and OS-neutral installation boundary.
status: draft
---

# Owner scope and responsibilities

HB-63 continues on `feat/HB-63-scalable-screens`, machine mac_mini. The owner requested frontend design of the entire current screen inventory, per-item IDs and per-screen review/FIX, and clarified that OS independence applies to installation and operation. Root owns integration/review UI and documentation. Isolated screen_presentations and portable_preview implementers own domain screen modules and portable tools respectively. gap_review provides independent design and actual-change review.

The existing 112 route identities and registry/baseline links remain. Design screens are synthetic, with local frontend actions and explicit unavailable production data. No provider/model/ERP call, actual monetary calculation, authorization, approval or issuance is introduced. Owner review is not delegated; tests use isolated browser storage.

# Implementation

- Screen-specific profiles refine layout, data hierarchy, form validation, evidence, prerequisites and next actions. Dedicated onboarding, AI, reporting, approval and customer overview remain integrated; bounded list behavior remains reusable. Every route includes review controls and a current-state implementation checklist with stable semantic item IDs.
- Review tooling is visually separate from product actions. The owner sees exact screen/revision, language, layout and displayed state/items before confirming. Confirmation records only that declared scope; it cannot certify all locales/states or incomplete backend behavior.
- Local FIX records remain in browser storage, labelled local and unauthenticated, and can be exported as JSON for ticket/Git reconciliation. Read/corruption/quota failures remain visible and never report a successful save. Older revision/variant evidence is preserved. A changed revision indicates impact assessment pending; it does not erase the old acceptance or by itself prove the design changed.
- Build-generated per-screen revisions cover the screen catalog entry and actual module/shared asset dependencies. Normalize text line endings and paths. An open confirmation pins route/revision/layout/item scope; changes require reopening before save.
- Portable Node entry points provide installation/build/start and browser-QA commands without OS-specific shell syntax, Python, Homebrew, absolute user paths or system service dependencies. Built assets remain usable on a standards-compliant static host.

# Validation

Verify inventory/ID uniqueness and actual rendered checklist coverage; all routes in EN/AR/KO; storage failure and corrupt input; isolated local confirmation and reload; variant/revision mismatch; export evidence; modal focus and mobile overflow. Test the static server for request methods, path/symlink traversal, MIME and manifest ownership. Re-run affected existing UI suites and inspect representative screenshots.

OS-independent design is a requirement. Actual Windows/Linux host execution and production deployment compatibility remain UNVERIFIED until exercised. Only actual commands and browser evidence will appear in the completion record.


# Result and remaining boundary

Implementation and independent source re-review passed for the current frontend scope. Root verification passed 16 Node tests, 98 generic browser checks, 71 journey checks and 104 final screen-review checks. See [verification](verification.md) for commands, actual hosts, rendered inspection and limitations. The portable preview starts through npm and serves 895 manifest-owned files locally.

Owner screen acceptance remains pending. Individual profiles and synthetic local flows are review candidates; route coverage is not proof that every production field, business exception or backend contract is complete. No prior module is retired without explicit acceptance of its replacement scope. Local review export must be reconciled with the ticket/Git evidence before canonical acceptance. Windows/Linux and production installation/operation are design targets with no execution certification in this increment.
