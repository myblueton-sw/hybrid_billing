---
type: Plan
title: HYB-296 O02 activation held until money, publication and ERP impact evidence exists
description: Step 1 of the owner's 2026-11-17 decision. The relationship.activate command refuses, and the O02 plans read reports a held state, while the plan has no amount, publication and ERP impact evidence; in pre-launch no producer exists, so every sealed plan is held with a truthful sentence and only discard stays available.
status: final
tags: [HYB-296, O02, HB-274, HB-275, HB-257, relationships]
sources:
  - resource: ../HB-274/plan.md
  - resource: ../HB-275/plan.md
  - resource: ../HB-257/plan.md
---

# HYB-296 O02 activation held until impact evidence exists

## Requirement and planning cross-check

- Owner decision 2026-11-17 (faithful to planning), recorded on HYB-296: O02 makes amount, publication and ERP impact evidence required and orders impact calculation, B09 approval request, then scheduled or applied execution. Today a sealed plan can be activated without impact (HB-274/HB-275). Step 1 (this ticket): hold activation with a truthful blocked state until impact evidence exists. Step 2 (phase 6): impact calculation and the B09 approval request once the billing modules are served.
- O02 spec (local planning input `docs/ui/screens/O02/spec.md`, untracked in the authorized workspace):
  - field contract line 54: "금액·공개·ERP 영향 | evidence | 예 (required) | 읽기 전용 (read-only)";
  - action contract line 72: "승인 요청 (approval request) … 전제 (precondition): 전체 의존 검증 통과 (all dependency checks pass)", preceded by "영향 계산 (impact calculation)";
  - linked screens line 85 include B09; line 80: a view with no data is hidden or states a not-applicable reason, never fake values. In step 1 the console shows no B09 link and no approval-request action; the held sentence is that reason;
  - progress display line 33: target → impact → approval → schedule or apply.
- HB-274 plan line 81-87 (`relationship.activate` preconditions) and line 107 (`impact` reported as `not_computed`). This ticket adds one precondition and one blocked reason; it changes nothing else in HB-274.

Base: main 9699fa1. Owner and writer: one implementer under root. Allowed files: `apps/api/server/relplans.go`, `apps/api/server/relplans_test.go` (and a new `relplans_hold_test.go` if needed), `apps/console/src/modules/organizations/pages/changes/{Live.tsx,live-model.ts,live-model.test.ts,live-copy.ts,live.css}` and their render tests, `apps/console/src/service/admin.ts` and `admin.test.ts`, `docs/ui/terminology/product-glossary.md` (two rows), this folder, the scratchpad QA runner `qa275body.mjs`.

## Server

1. Impact evidence is a per-plan value `impact` with three parts (money, publication, ERP), each one of `computed`, `not_computed` or `unknown` (`unknown` only when the accessor fails); the console maps any other or missing value to `unknown`. No pre-launch module produces it. The source is an unexported func field on `Server`, `planImpact func(*planMeta) (planImpact, error)`, never set by `New`, `Config`, a flag or a command; nil means all three `not_computed` (fail closed). Only `_test.go` files assign it, as with `s.fault`. One accessor is used by both `relationship.activate` and the read. An accessor error answers `dependency_unavailable` in the command and `impact` `unknown` with `blocked` `impact_unavailable` in the read (prepared for the step-2 producer).
2. `relationship.activate` gains a precondition after the stage checks and before the two-person check: if any part is not `computed`, refuse `prerequisite_failed`; an accessor error refuses `dependency_unavailable`. The order is: `dependency_unavailable` (active revision unreadable), then `invalid_stage` (not sealed, stale, effective date passed), then the impact check (`dependency_unavailable` on an accessor error, `prerequisite_failed` when a part is not computed), then the proposer check. Nothing is activated and the revision does not move.
3. The plans read: for a sealed plan that is not stale and whose effective date has not passed, `blocked` is `impact_not_computed` when evidence is missing, before `own_proposal`; `can_activate` is false. Each plan row gains `impact: {money, publication, erp}` with the per-plan values. The page-level `impact` map stays for compatibility as a legacy constant that describes no plan; the console stops reading it and reads only the row value.
4. Discard, propose, the open-plan limit and expiry are unchanged.
5. Public contract: `blocked` gains the values `impact_not_computed` and `impact_unavailable`; plan rows gain `impact`; `relationship.activate` can answer `prerequisite_failed` for a reason other than the proposer. The `next_action` of `prerequisite_failed` stays as defined. OpenAPI does not describe the read rows; the command catalogue is unchanged.

## Console (O02 service mode)

1. `admin.ts` parses the row `impact` (`money`, `publication`, `erp`); a missing or unrecognised value is `unknown`. Live.tsx reads the row value instead of `page.impact`.
2. `blocked()` maps `impact_not_computed` to `"impact"` and `impact_unavailable` to `"impact_unknown"`. For a held plan the approval view shows a held block in place of `blockedTitle` (not stacked under it) and no activate button (absent, not disabled); discard stays:
   - lead (600): EN "Activation is held: the amount, publication and ERP impact of this change has not been calculated." / KO "활성화 보류: 이 변경의 금액·공개·ERP 영향이 계산되지 않았습니다." / AR "التفعيل معلَّق: لم يُحتسب أثر هذا التغيير على المبالغ والنشر وERP.";
   - second sentence: EN "Impact calculation and the approval request are planned with the billing modules. Until then this plan can only be discarded." / KO "영향 계산과 승인 요청은 과금 모듈과 함께 제공될 예정입니다. 그전까지 이 계획은 폐기만 할 수 있습니다." / AR "من المقرر أن يأتي احتساب الأثر وطلب الاعتماد مع وحدات الفوترة. حتى ذلك الحين لا يمكن إلا تجاهل هذه الخطة.";
   - `impact_unknown`: lead EN "Activation is held: the impact of this change could not be read." / KO "활성화 보류: 이 변경의 영향을 읽지 못했습니다." / AR "التفعيل معلَّق: تعذّرت قراءة أثر هذا التغيير."; second sentence EN "Reload to try again. The plan can still be discarded." / KO "다시 불러와 보세요. 계획은 폐기할 수 있습니다." / AR "أعد التحميل للمحاولة مجددًا. لا يزال بالإمكان تجاهل الخطة."; discard stays.
3. The impact view shows the same held lead above the impact rows; its primary button is relabelled EN "Activation and discard" / KO "활성화·폐기" / AR "التفعيل والتجاهل" (it only navigates; discard lives in the approval view).
4. Refused activation: after a `refused` reload, the notice comes from the reloaded row's `blocked` (`impact` → held lead, `impact_unknown` → its lead, `own` → the own-proposal sentence, anything else → `otherBlock`), not always the own-proposal sentence (today live-model.ts:149-150, Live.tsx:402).
5. Progress follows spec line 33 with four steps: Target, Impact, Approval, Apply. Step states: `done`, `current`, `held` (new; EN "Held" / KO "보류" / AR "معلَّق"), `stopped`, `na` ("Not reached", as today). The Impact step is derived from the row `impact`, never from `state` alone: it is `done` only when all three parts are `computed`. In step 1 the second administrator's activation is both the approval and the apply, so Approval is `current` when the plan can be activated and Apply becomes `done` with it.

   | Plan | Target | Impact | Approval | Apply |
   | --- | --- | --- | --- | --- |
   | draft | current | na | na | na |
   | sealed, impact not computed | done | held (label "Not calculated") | na | na |
   | sealed, impact unknown | done | held (label "Could not be read") | na | na |
   | sealed, stale or effective date passed, impact not computed or unknown | done | held | na | na |
   | sealed, all impact computed, can activate | done | done | current | na |
   | sealed, all impact computed, stale, passed or own proposal | done | done | held (label from `blocked`) | na |
   | activated | done | done if all computed, else held | done | done |
   | rejected (violations at sealing) | stopped | na | na | na |
   | discarded, discarded_stale, discarded_expired | stops at the first step that is not done (for a held plan, Impact `stopped`); later steps na | | | |

   Step words EN/KO/AR are drafts for the technical-writing review.
6. Copy that would contradict the held state is reworded: live-copy.ts:26 lead becomes EN "Proposing creates a plan with its relationship changes fixed." / KO "제안하면 관계 변경이 고정된 계획이 만들어집니다." / AR "يُنشئ الاقتراح خطة بتغييرات علاقات مثبّتة." (the "another administrator activates it" clause is dropped in step 1); the sealed state word becomes EN "Changes fixed" / KO "변경 고정됨" / AR "مثبّتة التغييرات"; line 189 "Its impact is fixed." becomes EN "Its relationship changes are fixed." / KO "관계 변경이 고정되었습니다." / AR "تغييرات العلاقات مثبّتة.". Glossary rows: "Activation held", "Changes fixed".
7. KO and AR copy are drafts; a separate technical-writing review checks the copy before merge.

## Tests

- Go: activate refused `prerequisite_failed` for a sealed, current, other-person plan with default evidence, revision unchanged, plan still sealed; each single missing part refuses; an accessor error refuses `dependency_unavailable` and the read reports `impact_unavailable`; a server from a plain `New(cfg)` has the field unset and refuses; with all three injected as computed, the existing activation tests pass (moved onto the injection); relplans_test.go:63 and :344 (proposer's own sealed plan) now expect `impact_not_computed` without injection and `own_proposal` with it; order tests (stale and passed still `invalid_stage`); read `blocked`, `can_activate` and row `impact` per state; discard of a held plan works. Swapping the impact and proposer checks is an equivalent mutant (same code); it is recorded, not left as a survivor.
- admin.ts: row `impact` parsing including missing values.
- Console: model tests for `impact` and `impact_unknown`, the four-step model, the refused-reload notice choice; render tests in EN/KO/AR for the held block in the approval and impact views (no activate button, discard present), renamed words.
- Browser: `qa275body.mjs` expects the held state at EN/KO/AR × 1440/1280/390 (390 approval view: no activate button, discard kept). HB-275 browser checks that need activation become unreachable in the product: "Admin 2 activates, and the O01 link shows the new attribution" (HB-275 plan line 254), "a stale plan after another activation" (line 258), and the closed "activated as revision N" row with the O01 link (lines 114-116). They stay covered by Go tests with injected evidence and console render tests with `can_activate: true` fixtures; the browser gap is a residual. P01–P05 regressions re-run.
- Coverage stays 100% for `apps/api/server`; mutation on the changed Go lines.

## Residuals

- Every sealed plan is held in pre-launch; activation is unreachable from the product until step 2, and the browser checks for activation listed above are covered only by Go and render tests.
- Scheduling, B09 approval and resume of failed units stay out (phase 6).

## Review rounds

- Design review round 1: FAIL (F1 refused notice, F3 admin.ts out of scope, F5 contradicting copy, F6 progress premise, F8 lost browser checks; low F2 seam form, F4 page map, F7 impact view button, spec citations, B09, TW review, accessor errors). All folded in above.
- Design review round 2: PASS on condition M1 (explicit four-step table), plus L1 enum, L2 impact_unknown path and order list, L3 glossary rows, L4 lead clause. Folded in above; root checked the fold against the finding text.
- Change review PASS (5 low), render review PASS (4 low), technical-writing review FAIL (copy); all folded in, see the amendments below.

## Root amendments after the reviews (2026-11-18)

These supersede the matching text above.

1. Console 2: the EN held lead is "Activation is held: the impact of this change on amounts, publication and ERP has not been calculated." AR second sentence "سيُتاح احتساب الأثر وطلب الاعتماد مع وحدات الفوترة. حتى ذلك الحين لا يمكن إلا تجاهل هذه الخطة."; KO `impact_unknown` lead "활성화 보류: 이 변경의 영향을 확인하지 못했습니다."; KO second sentence "새로 고침으로 다시 시도하세요. 계획은 계속 폐기할 수 있습니다." The approval section heading is "Activation"; for a held plan the held block comes first under it, followed by a conditional restate ("If this plan is activated, the change applies from … and a new relationship revision is created."). A plan that is not held keeps the old order.
2. Console 3: the impact-view button and the result view's sealed-plan button read "Go to discard" / "폐기로 이동" / "الانتقال إلى التجاهل" for a held plan and "Continue to approval" otherwise ("Activation and discard" is dropped).
3. Console 5: AR held step state "معلَّقة"; impact step labels "Not calculated" and "Could not be read" (KO "확인하지 못함"), the same words as the impact row tags.
4. Console 6: "Changes fixed" becomes "Changes locked" / "변경 고정됨" / "التغييرات مقفلة"; EN lead "Proposing creates a plan whose relationship changes are locked."; `aSealed` ends "…are locked."; `onlySealed` "State of this plan: {state}. Only open plans with locked changes can be discarded or activated."; the open-plan limit sentence asks to discard one. Impact copy: "Amounts", "Impact on amounts, publication and ERP", per-row "This does not mean no impact.", pre-launch note "does not calculate these impacts", shown only while impact is not all calculated. Glossary: four rows (activation held, changes locked, not calculated, could not be read).
5. Console defence in depth: `canActivate` also requires all three impact parts calculated; such a row is held even if the server sends `can_activate: true`.
6. Deviations accepted: the impact source is `s.plans.impact` on the plan store (created only in `New`, set only by `_test.go`), not a `Server` field; "Calculated" appears only for injected values.

