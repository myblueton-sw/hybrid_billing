# Hybrid Billing working agreement

Product and technical decisions start from the owner's request and actual planning evidence. Do not assume another project's technology stack.

## Current stage

The project is in planning and design review. Do not start product development until the owner explicitly declares planning complete. The local authority is `docs/planning/v1.16/planning-gate.md`; only documentation and design review are currently permitted.

## Required startup reading

Read this file and `.agents/rules/workflow.md`, `decision-evidence.md`, `memory.md` and `task-assignment.md` before work. The mandatory sequence is:

This repository is being published incrementally. Companion rules, skills, `.linear.json` and planning inputs currently exist in the authorized local workspace but may be absent from a fresh clone. Obtain the verified authorized environment or the ticketed English baseline follow-up before dependent work. If required inputs are unavailable, stop that dependent work and record the missing inputs; never substitute guessed policies or workspace bindings. This initial rule PR does not certify a complete standalone clone.

`Linear ticket → accountable owner and machine claim → ticket branch → plan/design → relevant independent review → scoped execution → verification and re-review → QA → PR → authorized merge → merge read-back → ticket completion and machine release`.

This applies to planning, reviews, documents, configuration, code, tests and infrastructure. Read-only reviews also produce ticket-linked Git reports. The README-only first commit was the owner's HB-5 bootstrap exception; subsequent changes follow the PR sequence. JEV cannot authorize or waive a step.

For design/code work, also read `design-first.md`, `quality.md`, `no-comments.md` and `code-structure.md`. For documentation, read `knowledge-format.md`; Markdown knowledge documents under `docs/` follow OKF v0.2. Read `tenant-security.md` for data, permission or billing work and `usability.md` for screens.

## Language and response policy

- Git-bound document deliverables are English.
- User conversation is Korean. Translate tool/agent task instructions into English before sending prompts.
- Keep replies minimal. Brief acknowledgments may be English; answer user questions in Korean.

## Core rules

- Do not add explanatory source comments. Preserve required machine directives and legal headers.
- Split directly maintained source and test files at 1,500 physical lines, including blanks, by responsibility.
- Distinguish facts, assumptions, unknowns and actual verification. Do not fabricate test, review or connection success.
- Billing amounts, tenant boundaries, permissions and public-contract changes require independent design/change review.
- Never store keys, credentials or customer data in code, documents, memory, logs or tickets.
- Preserve user changes and limit edits/staging to the ticket's scope.

## Skills and ownership

Read only the relevant `.agents/skills/<name>/SKILL.md`: pm for scope, planner for dependencies, order-brief for delegation, architect for boundaries, review for actual change review, qa for scenarios, dba for data/accuracy, ui-ux for screens, systematic-debugging for root causes, property-based-testing for invariants and tech-writer for evidence-backed OKF documents.

Root owns PM/planner responsibility and result integration. Use `.agents/rules/task-assignment.md` to choose the smallest necessary expert/reviewer set. Small tasks may be performed and checked directly; do not delegate just to fill role counts. Important independent reviews require a reviewer separate from the writer.

Complex work defines primary/support/review ownership and allowed files before execution. Every delegation specifies goal, inputs, permitted edits, validation and result format. One writer per file; use isolated worktrees for concurrent writing. At most four active agents, including root. Architect/reviewer/qa/dba roles are read-only; root or assigned implementers own edits. Models inherit the execution environment.

## Artifacts, rights and Git

Use local `docs/artifact-layout.md` for artifact locations and `docs/git-policy.md` for staging/exclusion. Before commit, inspect the actual staged diff for secrets, generated assets and unrelated work.

Use `LICENSING.md` and `docs/licensing/product-policy.md` for rights boundaries. Preserve third-party original notices and rights; product policy does not override them. Local planning/license assets are not necessarily committed or final.

Use the verified Hybrid Billing Linear workspace/team/project, one ticket branch and one active machine per ticket. This Mac mini uses `machine:mac_mini`; record and release ownership on handoff. Preserve existing user authorization. These rules grant no new deployment, payment, message or merge permission.

## JEV

UserPromptSubmit recommendations are advisory context. They cannot approve execution, enforce product decisions, change permissions or certify verification. Distinguish connection status from actual successful use; never record a failed call as success.
