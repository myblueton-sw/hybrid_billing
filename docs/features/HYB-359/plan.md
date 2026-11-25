---
type: DesignPlan
title: P05 retire form finds a replacement beyond the first twenty candidates
description: A search over the replacement candidates of one requirement, and the retire form of P05 filling its replacement choice from that search when more candidates exist than the main read lists.
status: draft
---

# Scope and ownership

HYB-359, machine:mac_mini. Found in HYB-305 increment 2b. It adds one read and changes the replacement field of the retire form; the retire command and its checks are not changed.

# Facts

- A requirement row of the release evidence reads lists the first twenty replacement candidates in recording order with their total and `more` (`evidence.go` `replacements`, `evidenceReplacements`). The candidates are those the trace module accepts as at least as strong (`modules/releasetrace/reads.go` `Replacements`), without the ones that are already a pending replacement.
- The retire form offered those twenty in a `select` and said that only the first twenty are listed. A candidate beyond the twentieth could not be chosen on the screen, although the command accepts it.
- No command adds a requirement; a running service has the six of the sample, where no requirement has more than five candidates. The case arises only with more requirements.

# Built

- `GET /api/v1/release-evidence/replacements` with `requirement` (a requirement target, required) and either `q` (matched without regard to case against the candidate's label and target) with `limit` (1 to 20), or `id` alone (one candidate target). It answers `replacements` (`target`, `label`) in recording order, `matching` and `more`. Without a term it gives the same twenty as the main read. The candidates are built by the function the main read uses, so the read cannot offer a requirement the main read would not. Admitted as the release evidence read is, with its fault step; a requirement that does not exist, and any request from another workspace, is refused (422). It does not read the member roster.
- The form (`release-evidence/LiveRetireForm.tsx`): with twenty or fewer candidates the field is unchanged. When more exist, under the `select` a status line says how many are shown of how many, and a labelled search (input and button; Enter searches and does not submit the form) fills the `select` with the matches. The placeholder and "No replacement" stay; a candidate already chosen stays an option when it does not match; nobody is chosen for the person; an empty search restores the first twenty. A failed search leaves the options as they were, with a retry. The confirmation names the chosen candidate by the label the search returned. When the chosen candidate is not among the requirement rows of the page, choosing it reads that one requirement by its target (the existing requirements read), so the confirmation shows its state at the candidate build as it does for a listed one; while that read is under way the label stands alone, and when it fails the confirmation says that the details of the replacement could not be read. It no longer says "Unknown" for a candidate chosen in this form.
- The composition and the strings in three languages are the ui-designer's specification of 2026-11-25: a `select` filled by a search, not the list of buttons used for the owner of a hold, because a candidate is one short string and the field also needs "No replacement".

# Verification

Server test `evidence_replacements_test.go`: sixty requirements added, the main read's twenty against the read without a term, a search that reaches candidates beyond the twentieth, a limit, a candidate by target, targets that are not candidates, the command accepting a candidate found by search, a pending replacement no longer offered to another retirement, refusals, 403, 401, the fault step, a roster failure that does not affect the read, another workspace. Console tests 1425, 1422 passed, 3 skipped. Rendered check `tools/replacement-search-qa.mjs`, 54 passed: the real service for the unchanged field and the read; routed answers (45 candidates) for the search states and the confirmation (the state read answered in English and Arabic, failed in Korean), in three languages at 1440 and 390.

# Not verified

The search with a real service holding more than twenty candidates in the browser; the server test covers that data through the read, the rendered check through routed answers.
