---
type: VerificationReport
title: HB-37 license modes requirements verification
description: Records documentation scope, source checks and independent review for future license mode behavior.
status: draft
---

# Scope and ownership

[HB-37](https://linear.app/hybrid-billing/issue/HB-37/define-noncommercial-commercial-trial-and-purchased-license); accountable owner Seung Woo Park; machine mac_mini. Branch `docs/HB-37-license-modes` starts from main `4f990f168790ad15b10bf6359927fadeefb8e6ef`. Assigned writer owns only [usage requirements](../../licensing/usage-modes-requirements.md) and this report. Root owns independent review and integration.

The tech-writer skill and required companion rules were read in the authorized workspace. Root approved the task's design boundaries before writing. HB-36 public-license adoption subsequently merged into main at `4b4d22fe1f31bf9a013c26f4fb842e7e32a900da`, as confirmed by the root integrator; this change does not modify LICENSE, configuration, source code, contracts or runtime behavior.

# Evidence and checks

Official PolyForm Noncommercial 1.0.0 and Keygen offline-license primary documentation were opened on 2026-10-28. The requirements distinguish the owner's product decision from public-license permissions and technical examples; no vendor integration is claimed.

Writer checks completed on 2026-10-28:

- PASS: both files have nonempty OKF `type` frontmatter and all local Markdown links resolve.
- PASS: scope contains exactly the two assigned new documentation files; no existing license or runtime file changed.
- PASS: eleven numbered acceptance scenarios cover mode selection, grant acceptance, boundary expiry, reinstall/clock rollback, purchased registration, invalid/unknown/mismatched keys, network uncertainty, historical access/export, offline limits and nondestructive behavior.
- PASS: implementation/test status is explicitly NOT_IMPLEMENTED / NOT_RUN; primary sources are identified without implying integration.
- PASS: direct whitespace check of both new files; `git diff --check` also completed, but does not itself inspect untracked file contents.

PASS: root independently reviewed both actual documents for grant/mode separation, evaluation permission, noncommercial continuity, signed-key custody, time/trust limitations and explicit runtime status. The 336-hour inclusive-start/exclusive-end requirement was accepted subject to matching evaluation terms. This scoped artifact review is not legal certification. Runtime acceptance scenarios are NOT_IMPLEMENTED / NOT_RUN, not passing software tests. No commit, PR or merge is claimed here.

# Remaining work

Root owns subsequent commit and integration. Later implementation requires reviewed time/activation persistence, online/offline behavior, signed-key handling and actual evaluation/commercial permissions. No deployment, automatic notification, payment action or data lock is authorized by this document.
