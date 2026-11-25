---
type: Verification
title: Owned ERP result replacement service integration
description: Records retained application chains, bounded admission, recovery and independent corrective review.
status: draft
---

# Scope and ownership

HYB-341, machine:mac_mini, continues the full original result-ingestion goal. Root owns implementation and integration; refusal_review independently reviewed financial/service boundaries without editing. The verified ticket remains In Progress, assigned to Seung Woo Park with machine:mac_mini. This checkpoint follows d5403ff on feat/HYB-341-erp-result. No merge, deployment, external ERP command or financial approval consumption occurred.

# Implemented behavior

Each result application retains its immutable actor, intent and original receipt plus a detached exact completed snapshot. Successors bind their predecessor and exact before snapshot. Both reverse indexes identify the reserved head, including an unknown successor; complete bounded traversal validates every record and rejects branches, cycles, missing parents, mismatched indexes and orphan history. Historical inspection validates retained evidence rather than comparing an ancestor with today's ledger. Current apply/cache replay still checks current rights, watermark, head identity and exact live snapshot.

Replacement admission verifies the completed predecessor and actual ledger, checks retained representation and invokes CheckResultReplacement before reserving. The module shares complete locked validation between that nonpublishing preflight and atomic ReplaceResult; invocation repeats validation. Unknown predecessors cannot admit another successor. Interrupted successors remain owned by the original actor. Recovery invokes only from the exact before state or recognizes the exact reserved complete after state. Foreign, unavailable and changed states do not settle ownership. Service timestamps bound recognition and advance monotonically at reservation/completion.

Retained representation is checked with complete prospective before/after snapshots, actual actor-bearing links, exact stale history, maximum timestamp widths and a receipt placeholder. Each record is limited to 128 KiB and the registry to 100 records. Post-effect representation validation remains defensive. Original public output limits remain enforced separately.

# Verification and corrective review

The first combined focused service/module run passed in 6.608 and 0.318 seconds respectively. After adding actual capacity scenarios, the targeted race suite passed: server 40.900 seconds, module 1.254 seconds. Those measurements precede the final exact-size correction and stricter error assertions. The final targeted race suite passed after both corrections: server 45.175 seconds and module 1.152 seconds. The complete server regression subsequently passed in 193.170 seconds. Root reran vet for server/module and diff whitespace checks successfully on the corrected source.

Actual governed fixtures cover three consecutive replacements, exact stale-prefix retention, initial and successor receipts, before-write and after-write interruption, original-actor recovery, unchanged preparation/external command state, old fresh application refusal and current replay. HTTP scenarios cover replacement readiness, interrupted apply/new-key recovery, old cached success refusal and preserved historical inspection. Changed apply rights, foreign actor, newer confirmation and third-state ledger changes preserve unresolved successor ownership and impact counts. Corruption cases cover missing or altered completed snapshots, missing parent, cycle, altered before state, reverse index mismatch and orphan heads.

The actual ledger capacity test completes 50 applications with two rows each, reaching its 100-link limit. Revision 51 refuses before changing application count, either head index or ledger. A module preflight test independently verifies no publication and capacity rejection.

Independent review found two blocking issues and root corrected both: module capacity was not initially preflighted, and the first retained-size estimate omitted variable per-link actor overhead. The reviewer rechecked both corrections and returned scoped service source PASS. The source reviewer inspected source/tests and the final module race line, but did not execute tests or observe final server completion. The independent evidence reviewer subsequently verified the final module/server targeted race log and requested clarification that serialized representation bounds exclude actual in-memory object/index overhead; root corrected that wording. Neither reviewer executed tests. This is not UI acceptance.

# Remaining original scope

At this service checkpoint, B09 explanation and rendered review remained pending. Subsequent implementation and actual evidence are recorded in result-replacement-screen-verification.md. Staged diff audit and PR update remain pending for this increment. StageERPFinal using the existing approvals registry, issuance, delivery, persistence and the complete owner goal remain unfinished. These tests use synthetic local in-memory evidence and do not certify production durability, external integrations or other operating systems.
