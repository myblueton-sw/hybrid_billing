---
type: ImplementationPlan
title: HYB-325 P04 storage results state AI source-body non-collection and the separate Assistant conversation data class
description: Adds one note to the P04 deletion storage results, next to the HYB-312 outside-scope note, that states only what this build does — AI usage records have no prompt or response body fields and AI usage intake is not connected; Assistant conversations are their own retention data class whose deletion runs delete only conversations; the Assistant is not connected.
status: stable
tags: [HYB-325, P04, retention, HB-27]
sources:
  - resource: ../HYB-312/verification.md
  - resource: ../HB-288/plan.md
  - resource: ../../ui/screens/P04/spec.md
  - resource: ../../ui/terminology/product-glossary.md
---

# Requirement

P04 spec (local planning input `docs/ui/screens/P04/spec.md`, authorized local workspace) line 41, storage results: "AI 원천 본문 비수집과 Assistant 대화 보존 분리". Lines 143 and 151 give the policy: billing AI source fields exclude prompt and response bodies by default; consumer identity is minimised to pseudonymous references; this is separate from Assistant work-conversation retention; Assistant conversations, attachments and tool-run references are a separate retention class, apart from billing work, approvals and audit records; deleting a conversation must not remove business evidence. HYB-312 recorded this as its residual.

# Truth in this build (main 4471cfd)

| Question | Finding | Evidence |
| --- | --- | --- |
| Does the AI usage model hold prompt or response bodies? | No. `Event` has identifiers, time, outcome, four token counts, usage/cache relation and an optional cost; `Input`/`Output`/`CacheRead`/`CacheWrite` are parsed as whole-number counts and rejected otherwise. No body field exists. | `modules/tokenmeter/types.go:158-176`, `modules/tokenmeter/registry.go:217-242` |
| Is AI usage intake connected to the server? | No. `apps/api` does not import `modules/tokenmeter` or `modules/assistant`. | `apps/api/server/server.go:15-35` (import list); repository grep for `modules/tokenmeter"` / `modules/assistant"` outside the modules finds none |
| Does anything else collect prompt or response text? | No field named prompt, response body or message text was found in `modules`, `apps/api` or console sources; `completion` matches are unrelated counters (`modules/connectors/types.go:287`, `modules/perfgate/types.go:49`, `modules/customerusage/export_package.go:24`). | grep over `*.go`, `*.py`, `*.ts`, `*.tsx` |
| Are consumer identities pseudonymous? | Not enforced. `Consumer` is any valid ID. | `modules/tokenmeter/registry.go:218` |
| Does `modules/assistant` store conversations? | In memory only, as `{ID, Owner, Expires}` with at most 365 days to expiry; owner or system-after-expiry may delete. It stores no message text, and the module is not wired to the server. | `modules/assistant/types.go:35`, `:146-150`; `modules/assistant/registry.go:326-382` |
| Is there a separate retention class for conversations? | Yes. `ClassConversation` is one of seven data classes, labelled "Assistant conversations" on P04; policies are per class. | `modules/retention/types.go:20-26`, `:91`; `apps/console/src/modules/administration/pages/retention/live-codes.ts:13`, `:20` |
| Does deleting a conversation remove evidence? | No. A deletion plan holds artifacts of one class only, and a `mentions` reference does not cascade; an item that another item `requires` is not deleted while that item exists. The module test deletes conversation `c1` while final record `e1` that mentions it stays accessible. | `modules/retention/changes.go:82`, `:115`, `:142`; `modules/retention/runs.go:225-227`; `modules/retention/records.go:109-126`; `modules/retention/policy_test.go:98-121` |
| Is a conversation deletion reachable in this build? | No. Deletion plans and workers are not connected (P04 already says so); the only conversation item is the sample `sample-conversation-1`. | `apps/api/server/retentionseed.go:314`; `live-copy.ts` deletion intro |
| Attachments and tool-run references as a retention class? | Not modelled. | absent from `modules/retention/types.go` classes |

# Statement

Key `aiNote` in `live-copy.ts`. It claims nothing beyond the table above. KO and AR are AI drafts pending native review.

- EN: "AI usage records have no field for prompt or response bodies. Assistant conversations are their own data class: deleting them does not delete the records that refer to them. AI usage intake and the Assistant are not connected in this build."
- KO: "AI 사용량 기록에는 프롬프트·응답 본문 필드가 없습니다. Assistant 대화는 별도 자료 등급이며, 대화를 삭제해도 대화를 참조하는 기록은 삭제되지 않습니다. 이 빌드에서는 AI 사용량 수집과 Assistant가 연결되어 있지 않습니다."
- AR: "لا تحتوي سجلات استخدام الذكاء الاصطناعي على حقل لنصوص الموجّهات أو الردود. محادثات المساعد فئة بيانات مستقلة، وحذفها لا يحذف السجلات التي تشير إليها. استلام استخدام الذكاء الاصطناعي والمساعد غير متصلين في هذا الإصدار."

Terms reuse existing P04 copy: data class / 자료 등급 / فئة البيانات, "Assistant conversations" / "Assistant 대화" / "محادثات المساعد" (`live-codes.ts:20`), deletion run / 삭제 작업 / عملية الحذف, "in this build" / 이 빌드에서는 / في هذا الإصدار.

# Placement

`LiveDeletion.tsx` `Stores`: a new `<p className="p04-small">` directly after the HYB-312 `outsideNote` paragraph, default text colour, shown for every run's storage results.

# Glossary

Add two rows to `docs/ui/terminology/product-glossary.md`: `ai-source-body` (prompt and response bodies) and `assistant-conversations` (data class `conversation`), with the forbidden implications above.

# Test

`live-model.test.ts`: assert the paragraph follows `outsideNote` in `LiveDeletion.tsx`, the EN/KO/AR strings carry the key phrases (prompt/response, not connected, data class), and the three differ.

# Residuals

- Pseudonymous consumer references are not enforced (`Consumer` is any ID); the note does not claim it.
- Rejection of source input that carries unexpected bodies (spec line 143) is not implemented: no AI usage intake exists to reject it.
- Attachments and tool-run references are not a modelled retention class.
- Parser staging, error logs and support bundles applying the same classification (spec line 143) are not verified.
- KO and AR wording are AI drafts pending native review.
