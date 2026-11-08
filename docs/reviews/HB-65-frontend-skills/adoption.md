---
type: SkillAdoptionReport
title: Project-local frontend implementation skills
description: Adds two scoped project skills informed by HB-64 research while preserving existing workflow and frontend acceptance boundaries.
status: draft
---

# Scope and result

[HB-65](https://linear.app/hybrid-billing/issue/HB-65/adapt-researched-frontend-skills-to-project-implementation-rules) follows the owner's instruction to proceed after the external skills survey. Root owns the two skill files and this report; gap_review performs independent read-only scenario evaluation and actual-text review. Accountable owner: Seung Woo Park. Machine: mac_mini. The branch starts at main b48b23c.

- [hb-react-screens](../../../.agents/skills/hb-react-screens/SKILL.md) guides React screen/component implementation, request-scoped state, selective performance work, design traceability and portable tooling.
- [hb-screen-regression](../../../.agents/skills/hb-screen-regression/SKILL.md) guides rendered scenario checks, local fonts/RTL/keyboard behavior, asynchronous failures and isolated screen-review records.

These are concise original project instructions with pinned external references, not copies of upstream SKILL files or executable helpers. Existing skills, global configuration and dependency lockfiles are unchanged. Current project-authored material follows [LICENSING.md](../../../LICENSING.md); linked third-party material retains its own rights. A future import must preserve exact-file notices.

# Adoption choices

The [HB-64 research](https://github.com/myblueton-sw/hybrid_billing/blob/69950741876ef3f2c9e6bfc1adcfa7275c3b1978/docs/reviews/HB-64-external-skills/research.md) supports selective React composition/performance and interface/browser references. The skills directly pin the interface checklist instead of following its mutable wrapper. They do not activate Superpowers orchestration, design-system generation, cloud deployment or alpha/beta dependencies. Vite/Vitest remain conditional on actual toolchain selection.

The skills resolve current screen/installation contracts from the authorized workspace. Those inputs are in [HB-63 revision 4d5176](https://github.com/myblueton-sw/hybrid_billing/tree/4d51763458b314fd766998c05fb32a1c40c99e72), a separate unmerged PR. A fresh main checkout lacks some companions; these skills explicitly require obtaining verified inputs before dependent work. This PR neither vendors those pending changes nor certifies a standalone clone.

Use the implementation skill for a concrete React screen/component task and the regression skill for changed rendered behavior. Normal skill metadata remains available for discovery; no global hook or explicit-only setting is introduced. This session reads skill files directly. Automatic discovery in a restarted client has not been verified.

# Verification

- PASS: both files pass the bundled skill-creator `quick_validate.py` with isolated validation dependency PyYAML 6.0.2 through `uv run --no-project --with PyYAML==6.0.2 python`. The first system-Python attempt failed because PyYAML was absent; the isolated rerun passed. This dependency is validation tooling, not a product or skill runtime requirement.
- PASS: independent actual-text review found no blocking conflict with local ownership, design, financial/tenant boundaries, scoped acceptance or OS rules.
- PASS: independent read-only forward planning covered a slow old customer response arriving after a new scope, a shared component affecting an accepted screen, KO/AR review with failed browser storage, and a fresh checkout lacking design inputs with an alpha CLI suggestion. The plans retained request identity, invalidated stale selection/preview, assessed affected acceptance scopes, kept failed saves visible, isolated synthetic review records, limited macOS evidence and obtained missing inputs before dependent work.
- PASS: root checked frontmatter, nonempty descriptions, local Markdown links, immutable GitHub source references against the HB-64 manifest, scoped file inventory and staged whitespace/diff checks.
- UNVERIFIED: actual React product implementation, runtime behavior produced using these skills, browser/assistive-technology execution, automatic client discovery and Windows/Linux installation/operation. Forward testing was a read-only action-plan evaluation, not execution of the scenarios.

The larger frontend implementation request remains active. This change completes the researched skill-adoption prerequisite; it does not declare any product screen implemented or owner-fixed. Product work must retain its own screen/item trace, actual tests and independent review. No merge, deployment, real-account action or owner acceptance occurred.
