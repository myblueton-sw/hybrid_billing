---
type: DesignWorkflow
title: Claude CLI visual design leadership
description: Defines the owner-requested Claude CLI design role, evidence handoff and actual execution record.
---

# Ownership and workflow

The owner rejected the prior entry composition and requested that future UI visual design be led through Claude CLI. Root remains accountable for scope, frontend implementation, integration and actual browser checks. The repository role is `.claude/agents/ui-designer.md`; mandatory style constraints remain in `.agents/rules/service-screen-style.md`.

The design lead receives current rendered desktop/mobile images, existing source, canonical assets, actual external service reference captures and the owner's constraints. It diagnoses visual hierarchy and proportion, proposes one coherent composition, critiques the proposal and supplies implementation-ready specifications. Root implements that specification, then returns new renders to the same CLI role for visual comparison. A CLI response, model name or visual PASS is not human-designer certification, owner acceptance or production readiness.

The official installed `frontend-design` skill provides visual-design process guidance. Owner rules override skill defaults, including full-viewport service layout, neutral borders, canonical logo and local IBM Plex, square geometry, no gradients/shadows/decorative accent strokes, preserved IA, EN/KO/AR and RTL. Full-viewport layout does not mean controls must be stretched to the viewport width. No invented authentication, customer claims or service capability is allowed.

# Invocation boundary

Use the locally installed Claude CLI with the inherited CLI model. Explicitly load the ui-designer agent definition and allow only Read/Glob/Grep. The initial invocation uses restricted mode, empty strict MCP configuration, no session persistence and an explicit task brief. No shell execution, writes, nested agents/CLI, form submission, deployment or external messages are granted to the design agent. Root owns repository writes.

CLI is development tooling; it is not a dependency of product installation or operation. The agent definition contains no OS-specific skill path. Each invocation supplies the verified local skill location and only the required additional read directory. Do not copy third-party skill source or reference images into maintained product code. If authentication, the CLI, images or skill are unavailable, report that boundary instead of claiming successful design work.

# Initial execution inputs

Claude Code 2.1.316 was found and authenticated. Authentication account metadata is deliberately omitted. The official local frontend-design skill was read; SHA-256: `d91970639e9f5c37682ac7ab60094d35f1c7c1f38d731bd56396563aee10c1d3`. Its source and license were not copied into this repository.

The design request supplied four current start/login KO/AR desktop/mobile captures, seven relevant source files and existing tokens. Previously captured official Atlassian/Microsoft references were supplied as local visual evidence; the CLI must not claim to have browsed them itself. GitLab was excluded as unavailable sign-in evidence. The owner's reported problems are disproportionate controls, excessive empty space and incoherent overall scale.

# Returned design assessment

The actual invocation completed successfully (`is_error=false`, subtype `success`, 19 turns, 236048 ms, no permission denials). Reported model: `claude-opus-5-5`, inherited from the CLI environment. The reviewer reported reading the assigned skill, constraints, source and supplied images. It did not browse, run commands, render pages or execute tests.

Observed defects: approximately 748px-wide desktop controls, disconnected title/form regions, large empty areas below the 40/60 panes, inconsistent alignment edges and a mobile header approximately 165–170px high. These are the reviewer's image estimates, not browser measurements.

The proposed replacement is a full-viewport white grid with aligned title/primary/alternatives/context regions; no centered page container. Desktop fields are capped at 400px, primary actions use content width, and secondary routes become neutral separated rows capped at 640px. Shared gutters scale from 24px to 64px. Below 900px the layout becomes a single column. Mobile controls use the available column width, and the language selector becomes compact while retaining its accessible name. Existing IA, four entry paths, unconnected notices, preview separation, logo and IBM Plex remain constraints.

The reviewer adopted narrow readable form columns and explicit alternate paths from the supplied Atlassian/Microsoft captures. It rejected floating cards, shadows, illustrations and copied provider options. Reference sources: https://id.atlassian.com/login and https://login.microsoftonline.com/. GitLab was blocked and supplied no usable sign-in evidence.

This is a proposal, not an implemented or accepted replacement. The reviewer explicitly identified remaining risks: a two-column design may still resemble the rejected split, wide-screen whitespace remains unproven, secondary rows may appear generic, and shared CSS/component internals require verification. Root must resolve these concerns during implementation and return actual EN/KO/AR desktop/mobile renders to Claude CLI. No prior owner FIX is fabricated or overwritten by this report.

# Validation and measured usage

PASS: actual read-only CLI invocation returned a substantive screenshot-based design assessment; role and mandatory ownership rules are recorded. PASS: scoped document/role content manually checked against the invocation and owner constraints. No application source changed in this workflow revision; no new browser/build tests were run for it. The proposed screen implementation and follow-up rendered review remain pending.

CLI-reported token counts for this invocation only: uncached input 12; cache creation input 40050; cache read input 91902; output 24966, including reported thinking 16210. These categories reflect cumulative CLI telemetry, not unique source size or root-turn usage. Restricted mode ignored settings/hooks; JEV use inside this invocation is not claimed. Total root-turn token/cache accounting and a no-skill control run are unavailable. No measured token savings or causal optimization comparison can be stated.
