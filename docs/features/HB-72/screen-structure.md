---
type: Design
title: Planning-based screen structure for the Hybrid Billing console
description: Defines the 11-menu, 49-screen product structure, common frame, layout archetypes, routing, migration from the HB-63/HB-67 console and the phased Codex implementation plan.
status: draft
tags: [ui, information-architecture, routing, HB-72]
sources:
  - resource: ../../planning/v1.16/ui-design.md
  - resource: ../../ui/index.md
  - resource: ../../ui/navigation/roles-and-navigation.md
  - resource: ../../ui/flows/end-to-end.md
  - resource: ../../ui/screen-binding-register.md
  - resource: ../HB-67/claude-ui-design.md
  - resource: screen-registry.json
---

# Decisions and scope

Tracking: [HB-72](https://linear.app/hybrid-billing/issue/HB-72). Accountable owner: Seung Woo Park. Root and design author: Claude Code on machine mac_mini. Implementation: Codex, through separate tickets. This document changes no product code.

On 2026-11-01 the owner made three decisions:

1. The current HB-67 console mockups are **conditionally accepted**. This covers the visual system (14px work density, neutral borders, square geometry, full-width layout, canonical logo, IBM Plex, EN/KO/AR with RTL), the list-container pattern and the onboarding wizard presentation. It is not a per-screen FIX and does not accept any screen's content.
2. The product structure follows the planning baseline: the 11 menus and 49 screens of [ui-design.md](../../planning/v1.16/ui-design.md), its proposed paths, the page scope bar and the object tabs. HB-63's 112 fine-grained routes are absorbed into those 49 screens as tabs, steps or sub-views.
3. Claude designs and opens tickets. Codex writes the code.

Machine-readable companion: [screen-registry.json](screen-registry.json). It holds one record per screen (ID, Korean title, path, menu, audience, owning module, archetype, scope fields, tabs, flows, spec path, absorbed HB-63 routes and current React bodies). Implementation reads that file rather than re-deriving this document.

## Input availability

Per-screen field, action and state contracts are in `docs/ui/screens/<ID>/spec.md`. Most of those specifications are authorized local Korean companions outside Git. Only part of `docs/ui` is tracked. Pending bindings are listed in the [screen binding register](../../ui/screen-binding-register.md) and are owned by HB-24. Codex must work in the authorized local workspace where the specifications exist, or wait for their publication. A fresh clone is not a sufficient input. Where a specification and the register conflict, the register's binding applies when its screen ticket adopts it; a ticket must not silently choose either.

# Information architecture

## Shells

Three shells replace the current top-domain tabs.

| Shell | Used by | Structure |
| --- | --- | --- |
| Admin console | U, O, S, B, I, R, A, M01–M04, P, and M05 for admins | Top bar → left grouped menu → page header → scope bar → object tabs → content |
| Customer portal | C01–C03, U03 portal view, M05 for customers | Top bar with portal menu (내 비용, 청구서와 사용량, 문의, 알림) → page header → scope bar (own organizations only) → content; mobile bottom navigation for the four portal menus |
| Pre-login | L01–L03 | Existing entry shell (logo, language, aligned form column, context column, footer) |

**Top bar** (admin and portal): canonical logo at inline-start, installation name with operating/DEMO badge, workspace switcher, job status (the current background-task control), help, language, and a user menu that opens M05 and sign-out. The synthetic-preview strip remains above it while the console is a preview.

The workspace switcher selects a workspace (tenant boundary), not a provider source account. The current synthetic account picker (`scope=sample-a|sample-b`) is a preview stand-in and must be relabelled as such until a workspace contract exists. Switching installation is a larger boundary than an organization filter: open jobs and drafts show which installation and workspace they belong to, and the switch confirms that boundary (ui-design.md line 28).

**Left menu** (admin): nine groups in planning order, plus My account (M05) in the user menu. Each group lists its screens by planning ID order. A group is shown when at least one of its screens is permitted for the current role; the per-screen `roleCodes` in the registry decide this (`conditionalGrants` name grants that need a separate permission and no role code; the preview shows them as conditional). A01 additionally requires the separate permission named in its `requiresPermission` field (`assistant.use`) before its menu entry appears, and the table below only summarises the menu's main users. The current rail search remains at the top and searches only permitted screen titles and IDs.

This follows [roles-and-navigation.md](../../ui/navigation/roles-and-navigation.md), which places M05 under 내 계정; ui-design.md line 41 lists M05 under 사용자와 접근 (decision D11). The registry's `menus` list also contains the portal and pre-login keys, so it has twelve keys in total.

| Menu (KO) | Screens | Primary roles |
| --- | --- | --- |
| 업무 홈 | U01 U02 U03 U04 | BL, AP, CP, AU; U03 for all business roles |
| 조직과 계약 | O01 O02 O03 | AM, CP |
| 원천과 증빙 | S01 S02 S03 S04 S05 S06 | BL, OP |
| 원가와 요율 | B01 B02 B03 B04 B05 B06 B07 | PR, CP |
| 정산과 청구 | B08 B09 B10 B11 I01 I02 I03 I04 | BL, AP |
| 리포트와 이력 | R01 R02 | BL, AU |
| Assistant | A01 | Separately permitted roles |
| 사용자와 접근 | M01 M02 M03 M04 | IA, AM |
| 설치와 운영 | P01 P02 P03 P04 P05 P06 P07 | OP and separately granted business roles |
| 내 계정 | M05 (user menu) | Self |

The design-review directory (`overview`, `contents`) is review tooling, not product navigation. It moves to `#/design/screens` and `#/design/overview`, reachable from the review-tools disclosure and the preview strip. The contribution page moves under help.

## Routing

The console keeps hash routing so the static preview still runs without a server. Each route is `#<planning path>[/<object id>[/<tab key>]][?query]`:

- List or home: `#/billing-runs`. Object: `#/billing-runs/RUN-0001`. Object tab: `#/billing-runs/RUN-0001/compare`. Wizard step: `#/onboarding/<step key>`.
- Scope uses query keys `le`, `customer`, `contract`, `period`, `ccy` and `asof`, plus the existing `lang`, `mode`, `profile` and `scope` (synthetic account). The URL is never trusted as authorization; the server rechecks scope.
- Tab and step keys are stable English slugs defined per screen in the implementation registry. Visible labels come from copy.
- Every current hash ID redirects once using the registry's single-valued `legacyRedirects` table, which covers all 112 HB-63 IDs including `overview`, `contents` and `community`, keeping lang/mode/scope. Routes that mapped to several screens use the recorded primary target. The eight `setup-*` IDs go to interim `#/onboarding/<step>` routes that preserve the current wizard until decision D1. `legacyRoutes` per screen remains an absorption list, not a redirect source.
- An unknown or unpermitted route never reveals whether the object exists. The fallback depends on the authenticated audience: admins go to U01, customers to C01, unauthenticated users to L01. A customer opening an admin path or legacy admin ID receives no admin shell, menu, count or data. An admin opening a portal path or a legacy customer ID falls back to U01; admins see customer views only through the preview's audience selector.

Paths are information-architecture proposals from the planning document, not public API or deployment addresses. Tab slugs and machine scope keys are not yet in the registry; Foundation ticket 0a defines them and adds them to the registry before screen tickets start.

## Common frame components

These are built once and used by every screen. Each states its contract here; screen tickets only configure it.

| Component | Contract |
| --- | --- |
| Page header | Breadcrumb list → object → sub-view; title; object status tag; primary actions at inline-end. Back restores allowed filters, snapshot and scroll. When the object was deleted, unpublished or became unpermitted, content is not shown and only a route back to the permitted list is offered. |
| Scope bar | Ordered fields from the screen record. Shows as-of time and freshness. Changing scope discards selection, previews, approval previews and temporary links, and asks before discarding unsaved input. Shows only scopes the user may select; never shows counts for hidden scopes. |
| Object tabs | Only applicable tabs render; an inapplicable tab is omitted with its reason in the screen spec, never shown empty. Default set 요약/명세/검증/승인/문서/이력, renamed per screen as in the registry. Payment-terms (납기·연체) and recognition-cadence (인식 주기·대사) tabs are added only on the screens listed in the registry. |
| State view | Common vocabulary: loading/computing, verified zero, none, not received/waiting, unsupported, insufficient permission, stale, blocked, outcome unknown, partial, service failure. Jobs use queued/running/waiting/succeeded/failed/cancelled; partial is a result classification, never a job state. 0, none, not aggregated, unmeasurable, unsupported and unknown stay distinct. Revocation or session expiry clears protected content from the screen and local cache and requests re-authentication. |
| Amount meaning | Every amount carries its meaning (provider original provisional/final, internal cost, sales planned/final, allocation, Billed/Effective) and currency. Mixed currencies show display rate and date. Display rounding is distinguished from preserved values. |
| Bulk table | Normative contract: ui-design.md "대량 테이블·검색·선택·작업 계약" (lines 410–430), implemented in full. It includes: unsupported sorts are not offered; stable sort with ID tie-breaker; snapshot or as-of display; cursor expiry with regeneration; current-page versus all-matching selection bound to a server manifest; explicit keep or discard of selection on filter, workspace or period change; revision and permission recheck immediately before execution; page and selection totals per currency and revision, waiting instead of 0; sample preview never presented as full validation; dependency units in bulk changes; progress without invented percentages; cancel request distinct from completion; export scope, purpose, recipient, schema and manifest; long queries converted to jobs. Uses the HB-67 list container. |
| Job progress | Fixed denominator; succeeded/held/failed/unprocessed counts; resume of failed units only; cancel request distinct from cancellation complete; no percentage or ETA without evidence. |
| Approval card | Target, action, period, amount by currency, input/result digest, author, differences, warnings, evidence. Approve, reject and hand off are separate. The server rechecks revision, authority, limits and separation of duties. |
| Evidence link | Opens a safe viewer for evidence or source rows within current permission; original download is a separate permission. |
| Compare | Before/after or side-by-side with fixed revisions on both sides. |
| Timeline | Events with actual time, received time, actor type (person/AI/automation), revision and reason. |
| Wizard | Vertical step list from the HB-67 onboarding wizard (Carbon/Cloudscape evidence), horizontal row below 1280px, one primary action per state. |
| Secret reference | Shows reference, owner and expiry only; rotate or replace, never reveal. |
| Review panel | The existing per-screen design review control stays below the page header for every screen, keyed by planning screen ID, with item IDs handled as described under Migration. |

## Layout archetypes

Every screen uses one archetype. Archetypes are compositions of the common components; they do not add screen-specific behavior.

| Archetype | Composition | Screens |
| --- | --- | --- |
| dashboard | Scope bar → summary cards (amount meaning, fixed denominators) → work lists (blockers, my work) → links to detail. Mobile shows deadlines, blockers and next actions before charts. | U01, R01, P06 |
| list-workbench | Scope bar → list container (title, filters, bulk bar, table, pager) → row detail in a drawer or split pane → object tabs in the detail. | U02, U04, S01, S03–S06, B02–B08, B10, B11, I02–I04, R02, M01, M03, M04, P02–P05, P07 |
| list+detail-tabs | List page, then an object page with header, status and object tabs. | O03, B01, I01, M02 |
| tree-list-detail | Lazy tree (no hidden-child counts) → list → relation history panel. | O01 |
| inbox | Queue list with type separation → selected item detail with actions. | U03, B09 |
| wizard | Vertical step list → step content → action row; steps gated by evidence. | O02, S02, P01 |
| conversation | Context header (scope, as-of) → conversation → plan and result cards → links back to the originating screen. | A01 |
| detail-tabs | Self-scoped object with tabs. | M05 |
| prelogin-form / prelogin-wizard | Entry shell. | L01 / L02, L03 |
| portal-dashboard / portal-list-detail / portal-inbox | Portal shell variants of dashboard, list with document detail, and inbox. | C01 / C02 / C03 |

## Roles and visibility

Role codes follow the planning document: OP, AM, PR, BL, AP, CP, AU, CU, IA. Menu visibility follows the per-screen `roleCodes` in the registry. **Menu visibility is not authorization**: every read, write and download is rechecked by the server, and hidden objects never appear in counts, badges, search or errors. Customers never receive admin menus, internal cost, margin, other customers or internal notes; hiding columns in the browser is prohibited for this.

The preview adds a synthetic role selector inside review tools so reviewers can see per-role menus and disabled actions. It is labelled as preview tooling and grants nothing.

## Audience and shell selection

The shell and the data projection are chosen by the authenticated audience (admin, customer or unauthenticated), never by the URL path. U03 (`/tasks`) and M05 (`/my-security`) are shared paths: their customer views are separate projections with their own fixtures, not the admin component with columns hidden. Customers see only public document, download and inquiry events in U03 (U03 spec), and customer start pages are C01 rather than U01. The preview's synthetic audience selector switches the projection for review only.

Required negative cases for the foundation and portal tickets: a customer opening any admin path or legacy admin ID gets no admin shell, menu, counts or data; customer search, badges and errors contain no admin or other-customer counts; an admin role without permission for a screen sees neither its menu entry nor its existence in search.

# Screen index

All 49 screens. Tabs and steps use the specification's Korean terms. "Current React body" lists HB-67 implementations that become the screen's starting point. "Absorbed HB-63 routes" lists the remaining routes (the registry's `legacyRoutes` minus the current bodies) that become tabs, steps or variants; their content is re-checked against the spec, not copied as-is. Redirect targets are defined separately in `legacyRedirects`.

| ID | Screen | Path | Menu | Module | Archetype | Object tabs / steps | Current React body | Absorbed HB-63 routes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| L01 | 로그인 | `/login` | 로그인 전 | identity | prelogin-form | 로그인 방식 / 추가 인증 | entry-start, entry-login, entry-scope | — |
| L02 | 최초 등록·초대 수락 | `/activate` | 로그인 전 | identity | prelogin-wizard | 초대 확인 / 신원 확인 / 등록 결과 | entry-bootstrap, entry-invite | — |
| L03 | 계정 복구·긴급 접근 | `/recover` | 로그인 전 | identity | prelogin-wizard | 복구 경로 / 소유 증명 / 새 수단 설정 / 복구 결과 | entry-challenge | — |
| U01 | 전체 정산 현황 | `/home` | 업무 홈 | operations | dashboard | 현재 정산 / 내 작업 / 최근 변경 / 납기·연체 | operations | — |
| U02 | 마감 목록·달력 | `/closing` | 업무 홈 | operations | list-workbench | 목록 / 달력 / 선택 마감 상세 / 기한 변경 이력 / 인식 주기·대사 | core-index-close | — |
| U03 | 내 작업·알림함 | `/tasks` | 업무 홈 | operations | inbox | 내 작업 / 업무 알림 / 선택 상세 / 전달 상태 / 납기·연체 | my-tasks | operations |
| U04 | 청구 특이사항 | `/billing-findings` | 업무 홈 | operations | list-workbench | 특이사항 목록 / 검토 상세 / 관련 근거·조정 | — | operations, auto-exceptions |
| O01 | 조직·관계 탐색 | `/organizations` | 조직과 계약 | organizations | tree-list-detail | 조직 트리 / 원천 계정 / 조회 그룹 / 귀속 이력 | — | setup-attribution, organizations |
| O02 | 관계 변경·영향 검토 | `/organization-changes` | 조직과 계약 | organizations | wizard | 변경 대상 / 영향 검토 / 승인·예약 / 적용 결과 | — | index-mapping, organization-impact |
| O03 | 계약·발행 프로파일 | `/contracts` | 조직과 계약 | organizations | list+detail-tabs | 당사자·경로 / 청구 일정 / 상품·요율 / 문서·지급 / 종료·최종정산 / 납부 조건 / 인식 주기·대사 | customer-detail | setup-profile, supplier-register, customer-contract |
| S01 | 원천 연결·지원 범위 | `/connections` | 원천과 증빙 | onboarding | list-workbench | 연결 목록 / 지원 범위 / 인증·보안 / 진단 이력 | — | setup-connectors, setup-readtest |
| S02 | 온보딩·담당자 인계 | `/onboarding` | 원천과 증빙 | onboarding | wizard | 경로 선택 / 설정·권한 / 첫 데이터 / 첫 대사 / 개통 검토 | setup-progress, setup-profile, setup-attribution, setup-connectors, setup-readtest, setup-schedule, setup-first, setup-readiness | — |
| S03 | 수입·충돌·완전성 | `/imports` | 원천과 증빙 | sources | list-workbench | 수입 작업 / 격리·충돌 / 완전성 / 추출 검증 | index-intake | setup-first |
| S04 | 비용·사용량 명세 | `/source-data` | 원천과 증빙 | sources | list-workbench | 정규화 명세 / 원천 필드 / 사용량 / 변환 이력 | — | core-index-aws, core-index-variants, raw-explorer, ai-cost-waterfall |
| S05 | 증빙·문서 연결 | `/evidence` | 원천과 증빙 | sources | list-workbench | 증빙 목록 / 문서 연결 / 검증 / 보존·복원 | — | v4-inbox, v4-extract |
| S06 | 대사·차이 작업함 | `/reconciliation` | 원천과 증빙 | sources | list-workbench | 원천 대사 / 판매 대사 / 배분 대사 / ERP 대사 / 병행 운영 대사 | — | v4-differences, core-index-sources, cost-differences, core-reconciliation-case |
| B01 | 온프렘 원가 풀 | `/cost-pools` | 원가와 요율 | pricing | list+detail-tabs | 풀 요약 / 원가 구성 / 상품 포함 관계 / 승인 이력 | — | onprem-pools |
| B02 | 계측·자산 이력 | `/meters` | 원가와 요율 | sources | list-workbench | 자산 이력 / 계측 구간 / 누락·reset / 모델·미터 | — | asset-measurements |
| B03 | 배분·공유 풀 | `/allocations` | 원가와 요율 | pricing | list-workbench | 정책 / 전체 풀 / 시뮬레이션 / 비교·검증 | — | core-index-allocation |
| B04 | 요율·산식·영향 | `/rates` | 원가와 요율 | pricing | list-workbench | 요율 / 산식 단계 / 상속·예외 / 수동 조정 / 영향 비교 | — | auto-portfolio, supplier-aws, supplier-azure, supplier-gcp, supplier-oci, supplier-alibaba, auto-rulebook, auto-editor, auto-exclusions, auto-mapping, auto-bulk, sales-policy, sales-impact, core-index-pricing, index-pricing, index-seats, supplier-naver, supplier-kt |
| B05 | 환율·반올림 | `/exchange-rates` | 원가와 요율 | pricing | list-workbench | 정책 / 고시·수동 반입 / 환산 비교 / 승인 이력 | — | index-fx |
| B06 | 약정 원가·혜택 | `/commitments` | 원가와 요율 | pricing | list-workbench | 약정 명세 / 사용·미사용 / 혜택 전달 / 대사 | — | marketplace-commitment, index-commitments |
| B07 | Marketplace 거래 | `/marketplace` | 원가와 요율 | pricing | list-workbench | 구매자 청구 / 판매자 정산 / 혜택·크레딧 / 매핑 | — | marketplace-register, marketplace-commitment |
| B08 | 청구 실행·전후 비교 | `/billing-runs` | 정산과 청구 | calculation | list-workbench | 실행 목록 / 입력·정책 / 결과·규칙 추적 / 전후 비교 / 검증 / 인식 주기·대사 | — | auto-job, core-reconciliation-recalc, index-runs |
| B09 | 승인함·결과 카드 | `/approvals` | 정산과 청구 | issuance | inbox | 대기 / 내 결정 / 위임·대리 / 승인 이력 / 인식 주기·대사 | — | auto-activation, core-reconciliation-approval, index-approval |
| B10 | 청구 적용 이력 | `/claims` | 정산과 청구 | calculation | list-workbench | 적용 원장 / 부분 청구 / 역거래·정정 / 외부 참조 | — | auto-trace, auto-history |
| B11 | 크레딧·청구 문의 | `/credits-cases` | 정산과 청구 | history | list-workbench | 잔액 / 예약·적용 / 문의·이의 / 조정·역거래 / 납기·연체 | — | core-reconciliation-correction, index-correction |
| I01 | 인보이스·내부 명세 | `/invoices` | 정산과 청구 | issuance | list+detail-tabs | 문서 목록 / 문서 본문 / 원문·정정 / 승인·전달 이력 / 납기·연체 / 인식 주기·대사 | review-issue | assignment-policy, document-history, core-reconciliation-issued |
| I02 | ERP 제출·동기화 | `/submissions` | 정산과 청구 | delivery | list-workbench | 제출 목록 / 초안 비교 / 문서·행 연결 / 외부 사건 / 연결 capability / 납기·연체 / 인식 주기·대사 | — | erp |
| I03 | 파일 인계·반영 결과 | `/handoffs` | 정산과 청구 | delivery | list-workbench | 인계 목록 / 파일 구성 / 반영 결과 / 행별 대사 / 인식 주기·대사 | — | file-handoff |
| I04 | 공개 정책·전달 패키지 | `/exports` | 정산과 청구 | delivery | list-workbench | 정책 개정 / 공개 미리보기 / 패키지 / 전달 이력 | — | publication-packages |
| R01 | 정형 리포트 | `/reports` | 리포트와 이력 | analytics | dashboard | 요약 / 비교 / 상세 행 / 저장한 보기 / 내보내기 상태 / 납기·연체 / 인식 주기·대사 | — | cost-pivot, v4-variance, reports |
| R02 | 공통 이력·감사 | `/history` | 리포트와 이력 | history | list-workbench | 사건 목록 / 행위 상세 / 개정 차이 / 관련 증거 | — | v4-audit, auto-history, v4-monthly, v4-month, audit-events |
| A01 | AI 업무 Assistant | `/assistant` | Assistant | assistant | conversation | 대화 / 실행 계획 / 작업·결과 / 대화 보관 | — | assistant |
| M01 | 사용자·초대·대량 회수 | `/users` | 사용자와 접근 | identity | list-workbench | 사용자 / 초대 / 소속·역할 / 회수 작업 | — | users |
| M02 | 인증 제공자·로그인 정책 | `/authentication` | 사용자와 접근 | identity | list+detail-tabs | 제공자 연결 / 조직 로그인 정책 / 시험 로그인 / 키·인증서 / 복구 경로 / 외부 상태 동기화 | — | identity-policy |
| M03 | 역할·위임·효력 미리보기 | `/access` | 사용자와 접근 | identity | list-workbench | 역할 정의 / 부여·위임 / 효력 비교 / 지원 접근 | — | entry-roles |
| M04 | 서비스 계정·자격 | `/service-accounts` | 사용자와 접근 | identity | list-workbench | 서비스 계정 / 자격 메타데이터 / 최근 사용 / 책임자 이관 | — | service-accounts |
| M05 | 내 보안·로그인 연결·알림 | `/my-security` | 내 계정 | identity | detail-tabs | 내 계정 / 로그인 수단 / 추가 인증·복구 / 세션·최근 접근 / 알림 설정 | — | entry-sessions, personal-security |
| P01 | 설치·진단·지원 | `/installation` | 설치와 운영 | administration | wizard | 설치 계획 / 환경 진단 / 지원 자료 / 지원 접근 | — | core-reconciliation-setup, installation-diagnostics |
| P02 | API·플러그인·채널 | `/integrations` | 설치와 운영 | administration | list-workbench | API client / 모듈 / 구독·채널 / 사건·재처리 | — | integrations |
| P03 | 복구·이관·실행 상태 | `/recovery` | 설치와 운영 | administration | list-workbench | 복원 세트 / 격리 검사 / 외부 대사 / 고객 이관 / 재개 판정 | — | recovery-migration |
| P04 | 데이터 보관·삭제 | `/retention` | 설치와 운영 | administration | list-workbench | 보관 정책 / 사용량·만료 / 보류·참조 / 삭제 작업 / 아카이브 복원 / 결정 기록·복원 최신성 | — | retention-policy |
| P05 | 지원 조합·출시 증거 | `/release-evidence` | 설치와 운영 | administration | list-workbench | 지원 행렬 / 요구 추적 / 시험 artifact / 부하 프로파일 / 판정 이력 | — | support-matrix |
| P06 | 데모·시나리오 | `/demo` | 설치와 운영 | administration | dashboard | 시나리오 / 합성 입력 / 실행 결과 / 실패 주입 | — | demo-scenarios |
| P07 | 라이선스·배포 고지 | `/licenses` | 설치와 운영 | administration | list-workbench | 설치 권리 / 오프라인 파일 / 모듈 고지 / 계약·지원 | — | license-modes, license-status |
| C01 | 내 정산 홈 | `/portal/overview` | 고객 포털 | customer-portal | portal-dashboard | 요약 / 내 사용량 / 알림 요약 / 납기·연체 | — | core-customer-overview, v4-customer-months |
| C02 | 내 문서·사용량 | `/portal/documents` | 고객 포털 | customer-portal | portal-list-detail | 문서 목록 / 문서 상세 / 판매/배분 명세 / 허용 사용량 / 다운로드 상태 / 납기·연체 | — | auto-customer, ai-invoice-detail, v4-customer-months, customer-history, core-customer-invoice, core-customer-usage |
| C03 | 내 문의·답변 | `/portal/cases` | 고객 포털 | customer-portal | portal-inbox | 문의 목록 / 문의 작성 / 대화·자료 / 관련 문서·해결 | — | customer-inquiry, customer-response |

# Migration of the current console

## Implemented bodies

| Current route(s) | Becomes | Treatment |
| --- | --- | --- |
| entry-start, entry-login, entry-scope | L01 (`/login`) | Start and login merge into L01's sign-in flow; workspace selection becomes L01's post-authentication workspace step. Visual composition is kept. |
| entry-bootstrap, entry-invite | L02 (`/activate`) | Bootstrap and invitation become L02 variants fixed by token type. |
| entry-challenge | L03 (`/recover`) | Recovery paths and emergency access per L03 steps. |
| operations | U01 (`/home`) | The blocker queue becomes U01's blocker list region; U01 adds amount-meaning cards, closing summary and my-work region. |
| core-index-close | U02 (`/closing`) | List tab of U02; calendar and closing detail are added. |
| my-tasks | U03 (`/tasks`) | My-work tab of U03; notification and delivery tabs are added. |
| customer-detail | O03 (`/contracts/<id>`) | Becomes the contract object page; tabs are re-keyed to O03 tabs. |
| review-issue | I01 (`/invoices/<id>`) | Becomes I01's issuance preparation within the document object; approval decisions stay in B09. |
| index-intake | S03 (`/imports`) | Becomes S03's completeness and extraction-verification tabs. |
| setup-progress … setup-readiness | S02 (`/onboarding/<step>`) | Interim: the eight-step wizard moves intact to `/onboarding/start`, `/profile`, `/attribution`, `/connection`, `/diagnostics`, `/schedule`, `/first-receipt` and `/readiness`. It is restructured only after decision D1. |
| overview, contents | `#/design/…` | Review tooling outside product navigation. |

Implementation item IDs are **not renumbered**, following the screen-review rule ("retain the existing canonical screen ID and registry/baseline mappings… do not renumber items"). The 22 implemented bodies keep their existing item IDs (for example `operations:field:status-filter`) inside the planning screen that now hosts them. New items introduced by planning screens use the planning screen ID as prefix (for example `U01:region:amount-cards`). Foundation ticket 0a adds a persisted legacy-to-planning screen map so the review panel shows both the planning screen ID and the legacy screen ID of hosted items; when two hosted bodies define the same item suffix, both keep their distinct legacy prefixes, so no collision arises.

No canonical Git acceptance record exists for any current screen. Local browser acknowledgements, if a reviewer made any, cannot be checked from Git; they stay as recorded and are not carried to new fingerprints. Moving bodies changes every fingerprint, so all screens return to Ready for review.

## Onboarding decision required

The HB-67 wizard has eight steps (start, business profile, attribution, connection, diagnostics, schedule, first receipt, readiness). Planning S02 defines five steps: path selection, setup and permissions, first data, first reconciliation, activation review. Business profile and attribution belong to O03 and O01, connection to S01, and first receipt to S03.

**Recommended:** adopt the five planning steps. Step content links to the owning objects: setup and permissions uses S01 connection data; first data uses S03 receipt; first reconciliation uses S06. Business profile and attribution move to O03 and O01 forms opened from path selection. The wizard presentation (conditionally accepted) is reused. Alternative: keep eight steps as an S02-specific deviation recorded against the spec. The owner decides before the S02 ticket starts.

# Open decisions and contradictions

Each must be resolved in the named ticket, not assumed.

| # | Topic | Detail | Where decided |
| --- | --- | --- | --- |
| D1 | Onboarding steps | Five S02 spec steps vs eight HB-67 steps; wireframe W02 (ui-design.md line 253) also shows seven stages. | Owner, before S02 |
| D2 | CP role meaning | Specs define CP both as customer policy administrator (U01, U02, U04, R01, M05, C01–C03) and as internal cost owner (O/I/M/P glossaries, navigation doc). | Owner/domain review, before role-dependent screens |
| D3 | S05 customer access | S05 lists CU although customers must not be routed to internal evidence screens. Recommendation: CU reaches permitted evidence only through C02. | S05 ticket |
| D4 | Correction entry owner | I01, I02, B10 and B11 each offer correction entry; F04 starts from I01/B11. | B11/I01 tickets |
| D5 | Payment-term extension | U02 forbids it as a schedule change; I01 and O03 each have their own request. | O03/I01 tickets |
| D6 | `adjustment.draft` origin | B09 says B04 originates it; B04's spec does not define it. | B04 ticket with B09 |
| D7 | Flow order | ui-design F02 and end-to-end "월 정산" order differ; F01 and end-to-end "최초 설치" order differ as well. | U02/B08 and P01/L02 tickets |
| D8 | Spec vs design.json states | Ten screens differ (e.g. U02, B09, S02, S06, U03, I01–I03, O03). Spec text governs. | Each screen ticket |
| D9 | Region order with appended tabs | Payment/recognition regions listed first in English specs conflict with U01's stated order. Recommendation: appended tabs come after the screen's primary tabs. | U01/U02/B09/B11 tickets |
| D10 | A01 in the portal | A01 is not in the portal menu until a decision allows customer use. | Out of scope |
| D11 | M05 menu placement | Navigation doc: 내 계정 (user menu); ui-design.md line 41: 사용자와 접근. This design follows the navigation doc. | Owner, at foundation review |
| D12 | Archetype deviations from design.json | R01 and A01 (workbench → dashboard / conversation), U03 and B09 (table-detail / workbench → inbox), B01, O03 and M02 (form → list with detail tabs). Specs govern; the screen ticket confirms or reverts. | Each screen ticket |
| D13 | Type size guide | The owner approved 14px work density on 2026-11-01 (HB-67), but the untracked docs/design-system.md still records the 16px override. | Guide update when it is tracked |

# Implementation plan for Codex

Each ticket follows the mandatory sequence in AGENTS.md (machine claim, branch from current main, plan, independent review, scoped execution, verification, PR, authorized merge, read-back, release). The HB-66 ownership map (PR 62) is still open; foundation 0a records the module folders it creates and does not wait for that PR.

## Foundation (sequential, one writer for shared files)

| Ticket | Scope | Required tests |
| --- | --- | --- |
| 0a Registry and routing | Per-module screen registry files generated from screen-registry.json (tab slugs and machine scope keys added); hash routing with object, tab and step segments; the 112-entry legacy redirect table; legacy-to-planning screen map for review records; per-module copy files. | All 112 legacy IDs redirect to the recorded target with lang/mode/scope kept; unknown IDs fall back by audience; registry has 49 screens and unique paths. |
| 0b Shells and navigation | Admin, portal and pre-login shells; top bar (workspace boundary, installation badge, jobs, user menu); left menu with per-screen role filtering; synthetic role and audience preview in review tools; review tooling relocation. | Role preview grants nothing; per-role menus; customer negative cases (no admin shell, menu, counts or data from admin or legacy paths); keyboard navigation of menu and user menu; KO/EN/AR at 1440/390. |
| 0c Page frame | Page header with breadcrumb and cannot-restore fallback; scope bar with invalidation and unsaved-input prompt; object tabs; state view including revoked/session-expired; amount meaning; placeholder page for unimplemented screens (ID, purpose, roles, scope, tabs, labelled not implemented). | Scope change clears selection and previews; inapplicable tabs omitted; each state renders its text; placeholder never shows fixture data. |
| 0d Data components | Bulk table with the full normative contract; job progress; approval card; compare; timeline; evidence link; secret reference field. | Unit and browser tests for every bulk-table clause listed above, including all-matching manifest, cursor expiry, pre-execution recheck and export manifest; job counts without invented progress. |
| 0e Body migration | Move the 22 implemented bodies to their planning routes without behavior changes (S02 interim steps); keep item IDs; reset fingerprints with disclosure. | Existing suites (unit, qa, qa:work, qa:entry, qa:setup) adapted only for routes; each moved body renders at its new route; review panel shows planning and legacy screen IDs. |

## Screen phases

Order prioritises reaching a synthetic first billing cycle. F01's installation and authentication steps (P01, M02) are walked with synthetic preview data until their phases land.

| Phase | Screens | Depends on |
| --- | --- | --- |
| 1 | L01–L03 | 0a–0e |
| 2 | U01–U04 | 0a–0e, D2 for CP-dependent views |
| 3 | S01–S06 | 0a–0e, D1 for S02 |
| 4 | O01–O03 | 0a–0e, D2 |
| 5 | B01–B07 | 0a–0e |
| 6 | B08–B11, I01–I04 | 0a–0e |
| 7 | M01–M05 | 0a–0e, D2, D11 |
| 8 | P01–P07 | 0a–0e |
| 9 | R01, R02 | 0a–0e, D2 |
| 10 | C01–C03 | 0a–0e, D2 |
| 11 | A01 | 0a–0e |

Each phase is a parent ticket with one child ticket per screen. Only one screen ticket per owning module is active at a time, because screens of a module share its registry and copy files. Every screen ticket:

- **Inputs:** this document, screen-registry.json, the screen's spec.md and design.json, its screen-binding-register entries, the HB-67 visual system, and the rules screen-review.md, service-screen-style.md and design-system.md. Several of these exist only in the authorized local workspace (see Input availability).
- **Design process:** for a new visual composition, record external reference evidence (source, observation, adopted, rejected) and obtain the restricted Claude CLI ui-designer composition and render comparison, as service-screen-style.md requires.
- **Allowed files:** the owning module's folder (screen code, its per-module registry and copy files, fixtures, tests) and the screen's QA suite. Shared frame components change only through a separate shared-component ticket with an impact assessment.
- **Data:** synthetic fixtures only, visibly labelled; explicit unconnected mode shows no fixture data. No production API, credential, payment, deployment or message.
- **Validation:** unit tests for screen state logic; a browser suite covering the screen's states, role-preview differences, audience negative cases where relevant, scope-change invalidation, keyboard and focus, KO/EN/AR at 1440 and 390 px, computed neutral borders and no horizontal overflow; all existing suites.
- **Acceptance:** stable item IDs listed; independent read-only review; owner confirms the screen through the review control (FIX). No assistant, test or build creates FIX.

# Verification of this document

Root-executed on 2026-11-01:

- The registry has exactly 49 screen records matching the planning catalog IDs and paths in ui-design.md lines 32–44, plus per-screen purpose and roles parsed from the catalog tables (lines 90–163).
- Absorbed routes were computed from the HB-63 catalog (`apps/console/src/shared/catalog.json` on main 3e5f754): every planning screen maps to at least one existing route; three routes have no registry ID (`overview`, `contents`, `community`); four routes map to two or more screens (`operations`, `marketplace-commitment`, `auto-history`, `v4-customer-months`). That catalog is on main 3e5f754 and derives from HB-63 commit 4d51763. `legacyRedirects` covers all 112 IDs with exactly one target each; the four multi-mapped routes use recorded primary targets (U01, B06, B10, C01).
- An independent read-only reviewer reviewed the first draft (PASS WITH CHANGES, six blocking items) and sampled tabs and audience for 17 further screens against their specs. A re-review confirmed all six resolved and raised one new item, free-text roles, which this revision fixes by storing role codes and conditional grants.
- Tabs, scope fields and archetypes were extracted from all 49 specs by a separate read-only agent. Root compared the tab lists of U01, U02, S02, B08, B09, I01 and C02 with their specs: all match, except that the registry places the appended payment-terms and recognition-cadence tabs after the primary tabs (recommendation D9). Other screens' tabs rely on the extraction. Paths, the default object tab set and flows F01–F08 come from ui-design.md, not from the specs.
- Module owners follow HB-66 `screen-ownership.json` (PR 62, open). Where it lists several modules, the owner was chosen by planning menu; P06 is assigned to administration rather than design-review because it is a product screen.

Not verified: rendering, accessibility, performance, server authorization and every spec detail. All planning acceptance cases remain NOT_RUN.
