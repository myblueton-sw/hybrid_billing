---
type: VerificationRecord
title: Owner-authorized candidate integration and branch cleanup
description: Records independent conflict reviews, actual integrated checks and the boundaries of the November 8 merge handoff.
status: draft
---

# Scope and authorization

The owner explicitly requested merging the open work and cleaning merged branches on 2026-11-08. Root owns integration, tracking and cleanup. A separate read-only reviewer checked PR evidence, ancestry, the Marketplace financial boundary and the actual conflict resolutions. Existing per-ticket design, source and browser reviews remain scoped historical evidence, not a new whole-product certification.

The verified Linear workspace is `hybrid-billing`, team HB and project `a19e1f53-0244-4e2b-8bf6-0d7176e842c3`. All36 existing tickets have the actual accountable assignee and were checked for foreign machine claims before applying `machine:mac_mini`. HB-146 separately tracks the owner's later README cleanup request.

The initial main was `36394edf179d544d7ec2756f5329eba60bcc2d4e`. History-preserving merges completed for PR27, PR52-62 and PR67-89, in dependency order. PR91 independently removed only the README's first-person motivation, career, credentials and LinkedIn sections after scoped editorial review. Its product, installation, limitations, license and commercial-contact content remains. Before the final HB-145 handoff, remote main was `7738402` and integration commit `d73c676` included it. PR90 remains pending at this record's creation; its merge and remote ancestry read-back must be confirmed before cleanup or Git-work completion is recorded.

# Conflict resolutions and independent review

- HB-58 `c03ae3e` and HB-59 `4146cf1` preserve independent planning-review index entries rather than choosing one side. The HB-58 resolution received separate actual-diff review; root checked the equivalent additive-only HB-59 index merge.
- HB-63 `abdc52d` combines portability and service-screen rules, documents both static and React foundation consumers, and retains the documented16px body/table override. Separate actual-diff review passed. Canonical provenance assets remain unchanged.
- HB-74 `bd04b85` preserves the union of planning-routing and closing tests. Separate actual-diff review confirmed no test removal and that all8 registered paths exist.
- The final integration retains the HB-86 closing catalog's descriptive labels. Its dedicated18-variant closing journey was rerun after the final build.

# Actual integrated verification

Host: macOS arm64, Node26.7.0, Python3.11.7, Playwright runtime with Chromium153.0.8010.12. Frontend source fingerprint: `147ca3fa6cccc764261aa21ea1774395ecc349a25c0108d0f6ecd02eeaab2d33`.

| Check | Actual result |
| --- | --- |
| Final `npm test --prefix apps/console` | PASS254 tests, zero failures/skips |
| Final `npm run build --prefix apps/console` and `node apps/console/tools/verify-build.mjs` | PASS TypeScript/Vite and438 artifact hashes; existing large-chunk advisory remains |
| Isolated Python receipt/lock/overview/commitment/Marketplace suites | PASS193 tests; temporary dependencies use the repository's pinned versions and pytest8.4.2; no retained-service mutation suites |
| Static design build and `npm test --prefix tools/design` | PASS895 built files/eight immutable foundations and16 tests |
| Static design browser generic/journeys/review suites | PASS98/71/104 checks |
| Final remaining-candidate browser matrix | PASS852/852:31 screens excluding the subsequently connected S03,142 views ×KO/EN/AR ×1440/390; zero findings/page errors/external or non-GET requests |
| Final closing browser suite | PASS18 locale/viewport list/calendar/detail renders plus filters, preserved history, keyboard/dialog/scope/error cases |
| Prior canonical navigation smoke | PASS103 checks:17 previous candidates ×three locales ×two widths and the explicit S03 local file-input route without upload; not domain-workflow recertification |
| Root image inspection | KO desktop O01 and AR mobile M01 actual captures checked for legibility, layout and review revision; not native-language acceptance |

The matrix used a temporary copy of the existing runner with absolute evidence paths and S03 excluded. Its first run completed the views but failed while locating the relocated runner's build manifest; it is not a complete pass. The corrected runner repeated the matrix against the final build and wrote a complete852-check report. An earlier sandbox run could not open the tsx IPC or Vite cache and was rerun with execution permission. Python initially lacked pinned dependencies; a temporary target installation was used. The ambient global `tests` package then prevented a spawned lock-test worker from importing the repository namespace; a `-S` run with only repository and temporary dependency paths passed all193. No product/test code was changed to hide these environment failures.

Generated logs, captures and temporary dependencies stay outside Git. Exact local reports are under `/private/tmp/hb-merge-*`; they are execution evidence, not promised durable hosted artifacts. Backend tests use isolated temporary state. Browser smoke issues read-only GET requests through the existing local proxies; it does not establish that connected services read no control metadata. No reset, ingestion, financial command or customer-data operation was requested, and this integration run does not claim a new checksum audit of retained control files.

# Completion boundary

Remote main has no branch protection or repository rulesets, and the PRs report no GitHub check runs. Local verification above is actual evidence; GitHub CI is UNVERIFIED, not passed.

These merges integrate reviewed candidates. They do not grant owner screen FIX or certify commercial billing, production authentication, tenant enforcement, live CSP/ERP connectors, native-language review, scale, Windows/Linux or other browser engines. The Marketplace CSP-specific hold remains a product follow-up; source evidence still declares no financial authority. The owner's current merge instruction authorizes candidate-code integration without closing that hold.

After PR90 merges, fetch and verify every candidate head is an ancestor of remote main, read back the README and final code, and reconcile ticket Git-work status and machine release. Delete remote/local branches only when ancestry proves incorporation. Preserve unmerged branches, user changes and evidence-bearing worktree directories; stale metadata may be pruned. Checked-out incorporated branches may be detached at the same commit before their redundant branch references are deleted. No force deletion, reset, deployment, financial action or customer message is authorized by this handoff.
