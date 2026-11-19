---
type: IntegrationVerification
title: HYB-333 integration on the merged HYB-310 baseline
description: Records the owner-authorized integration conflict resolution and scoped re-verification before merging the M04 credential issue changes.
status: stable
sources:
  - resource: ./verification.md
  - resource: ./integration-evidence.json
---

# Integration scope

Owner authorized finalization of the three existing tickets. Root integrated main `69405050fdbf031a4ce4e4187a1bb2b4cfc5e6ba` (HYB-310, PR235) into the HYB-333 branch without rewriting its published history. Machine: mac_mini.

The only textual conflict was `apps/api/server/openapi.go`: both `/identity/services` and `/integrations` remain in `x-reads`, retaining their reviewed helper implementations. The shared client, binding register and glossary auto-merged. No new behavior or permission choice was introduced.

# Review and verification

- Independent read-only integration review: PASS. It confirmed both read contracts, configured `AuthorClient`, preserved command IDs/pickup parser, and absence of pickup metadata in the integrations read. Source review was separate from root test execution.
- `go test -short ./apps/api/server ./modules/identity ./modules/auth ./modules/gate ./modules/commands`: PASS; server63.306s, every listed package ok.
- `npm test`: 838 passed, zero failures.
- `npm run build` and API build: PASS.
- Fresh M04 complete issue scenarios on the combined binary:118 passed, zero failures.
- Fresh P02 S regression on the same combined candidate:125 passed, zero failures, including EN/KO/AR and the four target widths.
- No unresolved merge path or staged whitespace error remained. Source/artifact hashes are in `integration-evidence.json`.

# Remote checks and completion boundary

Read-back found zero GitHub Actions workflows, no repository rulesets and no classic main protection. Remote empty check lists are not a CI PASS; the local checks and independent reviews above are the actual evidence. No CI setup or new implementation ticket was added.

Existing pre-launch residuals, unavailable designated Claude CLI review, native-language/assistive-technology limits and owner-only FIX remain as recorded in `verification.md`. Remote merge and main ancestry/content read-back are performed afterward and recorded in the ticket; this pre-merge report does not certify them.
