---
type: Plan
title: HYB-309 P02 plugin license check before activation
description: Adds a license identifier to the signed plugin manifest and an installation list of permitted plugin licenses, refuses install, upgrade, rollback and enable while the version's license is not on that list, and shows each version's license and the list on P02 in EN, KO and AR.
status: stable
tags: [HYB-309, P02, HB-40, HB-284, HB-285, plugins, licensing]
sources:
  - resource: ../../reviews/HYB-301-trace-classification/report.md
  - resource: ../HB-284/plan.md
  - resource: ../HB-285/plan.md
  - resource: ../HB-253/plan.md
  - resource: ../HB-257/plan.md
  - resource: ../../../LICENSING.md
  - resource: ../../licensing/product-policy.md
---

# HYB-309 P02 plugin license check before activation

## Ticket, base and ownership

- **Ticket.** [HYB-309](https://linear.app/hybrid-billing/issue/HYB-309): "HB-40 / P02: no licence field or check exists in modules/plugins. Add the licence check before activation; review." Source: item 6 of the "Required and missing" table in [the HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) (`report.md:321`). Owner direction of 2026-11-17: follow the planning documents.
- **Base.** main `8d673f1`. Branch `feat/HYB-309-p02-plugin-licence`. Worktree `/private/tmp/hb309`. Machine `mac_mini`.
- **Ownership.**
  - Root (Claude Code) owns this plan, the integration and the PR.
  - One implementer writes the code in this worktree.
  - An independent read-only reviewer reviews the design before any code is written. Reason: the change alters a public read contract and the signed manifest format, and it adds a refusal to an approval path. A second review covers the actual change.
  - A separate read-only technical-writing review checks the new terms and the EN, KO and AR copy (`product-terminology.md`).
- **Allowed files.**
  - `modules/plugins/`: `types.go`, `manifest.go`, `changes.go`, `registry.go`, `operations.go`, and tests (`form_test.go`, `lifecycle_test.go`, `property_test.go`, `world_test.go`, `signing_test.go`, plus a new `license_test.go` if needed).
  - `apps/api/server/`:
    - `integrations.go`, `integrationseed.go`, `server.go` and `settings.go`, or a new `pluginsettings.go`;
    - their tests: `integrations_api_test.go`, `integration_commands_test.go`, `integration_edges_test.go`, `review_survivors_test.go`, and the settings tests.
  - `apps/api/cmd/hb-api/main.go`: the start flag.
  - `apps/console/src/service/integrations.ts` and its test.
  - `apps/console/src/modules/administration/pages/integrations/`: `live-model.ts`, `live-codes.ts`, `live-copy.ts`, `LiveModules.tsx`, `LiveProposals.tsx` (review F7), `live.css` (only if the layout needs it), `live-model.test.ts`.
  - `docs/ui/terminology/product-glossary.md`: the two new terms.
  - `docs/ui/screen-binding-register.md` and `docs/ui/screen-binding-fields/operations.md`: the P02 rows.
  - `docs/features/HYB-309/*`.
  - No other module and no other screen. No change to `modules/licensemode`, `modules/thirdparty` or `modules/marketplace`.

## 1. What the planning requires

The planning files below are in the authorized local workspace and are not tracked in Git. Line numbers refer to those files on 2026-11-18. Quotes are verbatim.

### Verbatim requirements

| ID | Source and line | Text (verbatim) |
| --- | --- | --- |
| HB-40 requirement | `docs/planning/v1.16/requirements.md:717` | 작은 자체 코어와 후보 OSS를 정확성·격리·운영·권리 기준으로 비교한다. 제품의 소스 공개형 상용 정책과 제3자 라이선스를 혼합하지 않는다. |
| HB-40 implementation direction | `requirements.md:719` | 후보마다 pinned version/commit, LICENSE, dependency/SBOM, 오프라인/telemetry, 수정·배포 방식, 시험 결과를 기록한다. … entitlement는 오프라인 검증과 과거 증빙 접근 유지를 설계한다. |
| HB-40 acceptance | `requirements.md:723` | 배포 고지·라이선스 경계·만료 후 이력 접근·업그레이드/자료 이관을 검증한다. 법률적 적합성은 본 분석으로 확정하지 않는다. |
| HB-40 open decisions | `requirements.md:725` | 코어 언어·상용 계약·SDK 라이선스·OSS 고정 릴리스. |
| HB-40 detail | `requirements.md:729`; `detailed-requirements.md:341-351` (DB-14) | DB-14 is about product sales through a Marketplace, which stays in later scope ("초기 출시 필수로 승격하지 않는다", `:347`). It is not the P02 check. |
| P02 manifest field | `docs/ui/screens/P02/spec.md:50` | 모듈 manifest·digest \| reference+digest \| 모듈 시 \| 등록 초안 \| 서명/호환/권한/라이선스 검사 \| 내부 |
| P02 activation action | `P02/spec.md:74` | 활성화 요청 \| 적격 승인 권한 \| 실증·라이선스·범위 검증 \| 고정 버전 활성화 계획 \| 권한·외부 영향 확인 |
| P02 blocked state | `P02/spec.md:98` | blocked \| 검증되지 않은 버전·목적지·권한입니다. 활성화할 수 없습니다. |
| P02 error state | `P02/spec.md:102` | 오류는 안정 code·request ID·필드 위치·다음 조치를 표시하고 비밀/고객 원문 stack trace를 노출하지 않는다. |
| P02 trace | `P02/spec.md:120` | HB: … HB-35, HB-37, HB-40, HB-42。 |
| P02 screen scope | `docs/planning/v1.16/ui-design.md:156` | P02 API·플러그인·채널 \| OP·IA·범위 내 BL \| client·scope·구독·모듈manifest·버전/digest·capability·송신대상·라이선스·작업제한, 등록/시험/승인/격리/중지/재처리 \| … HB-05·24·31·33·34·35·40 |
| P07 module licenses | `ui-design.md:161` | P07 라이선스·배포 고지 \| … 설치권리·기간·운영범위·서명상태·모듈라이선스·고지·지원수준 … |
| P07 notices | `ui-design.md:163` | 문서와 모듈별 원래 라이선스·고지·지원 주체는 다운로드 가능하게 제공한다. |
| P07 module rights | `docs/ui/screens/P07/spec.md:41`, `:53` | 제3자 원래 라이선스 보존; Marketplace 통합은 후속 범위 / 모듈 라이선스·고지 \| reference[] \| 예 \| 읽기 전용 \| 기존 제3자 권리/고지 보존 \| 공개 |
| Module execution (QA-06) | `P02/spec.md:150-152` | 과거 재현에 필요한 취약 구버전은 격리 검증 환경에서만 실행하며 … 인수: … 취약모듈 운영실행은 거절되며 … |

### Rights boundaries in tracked documents

- `LICENSING.md:18`: third-party libraries and tools keep "Their own licenses and notices. The root license does not relicense dependencies".
- `docs/licensing/product-policy.md:23`: "This change introduces no activation check, automatic external reporting, remote lock, fee collection or other enforcement runtime." This sentence is about the product's own source-available license.
- `docs/features/HB-253/plan.md:67, 88-105`: the third-party component register (`modules/thirdparty/changes.go:14-20`) has a closed table of SPDX identifiers. The table shows where each license may ship (`bundled`, `separate`, `tool_only`). Its status: "an engineering boundary for this register, an owner decision not confirmed by the owner, and not legal advice."

### What "license" means here

The planning text strongly supports this reading. It is not a guess, but the owner has not confirmed it (Q1).

- **The plugin's own license.** It is the license under which the plugin publisher offers the module (P07 "모듈라이선스" and "모듈별 원래 라이선스", `ui-design.md:161, 163`; "제3자 원래 라이선스 보존", `P07/spec.md:41`).
- **Not the product's own license or a commercial entitlement.** HB-40 forbids mixing the product's source-available policy with third-party licenses (`requirements.md:717`). `product-policy.md:23` states that the product-license policy added no activation check. Installation entitlements (P07 "설치권리", `modules/licensemode`) are therefore not consulted. Marketplace rights (DB-14) are later scope.
- **Not a legal finding.** HB-40 does not settle legal suitability (`requirements.md:723`). The check compares the declared identifier with a list the installation keeps. It does not decide whether a use is lawful.

### Requirements the planning leaves open

| Question | Planning | Plan decision |
| --- | --- | --- |
| What is checked | "라이선스 검사" on the manifest (`P02/spec.md:50`) and "라이선스 … 검증" before activation (`:74`). Content not defined. | The license identifier declared in the signed manifest is checked against the installation's list of permitted plugin licenses. |
| When | At manifest validation in the registration draft (`:50`), and as a precondition of the activation request (`:74`) | At registration: the identifier must be present and well formed. Before activation: it must be on the permitted list. "Before activation" covers install, upgrade, rollback and enable, and their approval (see 3.2). |
| Outcome states | Blocked: "검증되지 않은 버전·…·권한입니다. 활성화할 수 없습니다." (`:98`) | Permitted or not permitted. No warning-only state. The blocked sentence is shown with the refusal. |
| Who may override | Not stated. Activation needs "적격 승인 권한" (`:74`). | Nobody. There is no per-plugin exception. The only way to change the outcome is to change the installation's permitted list (section 4). Activation still needs a second administrator. |
| How it is shown | P02 data includes "라이선스" (`ui-design.md:156`). Errors show a stable code and next step (`P02/spec.md:102`). | Each version shows its license and whether it is permitted. The list and its source are shown. A blocked approval shows a reason in EN, KO and AR. |
| Which licenses are permitted | Not stated. "SDK 라이선스" is an open owner decision (`requirements.md:725`). | A changeable installation default (section 4, owner question Q2). |

## 2. Current behaviour

- **Manifest.** `Manifest` has no license field (`modules/plugins/types.go:188-201`). `valid()` checks IDs, digest, versions, capabilities, egress, endpoints, limits, storage and screens (`manifest.go:146-183`). The signed bytes are `encode()` under tag `hybrid-billing/plugin/v1` (`manifest.go:28, 238-275`). The known-answer hash is `form_test.go:60`, and the per-field distinct-encoding table is `form_test.go:262-305`.
- **Registration.** `checkVersion` refuses an unknown key, a publisher mismatch, an inactive publisher or key, a bad signature (`signature_invalid`) and an incompatible core (`incompatible`) (`changes.go:170-193`). Nothing about licenses is checked.
- **Activation path.** `runnable(w, v)` returns `revoked` for a revoked publisher or compromised key. It returns `vulnerable` for a vulnerable version outside a verification workspace (`changes.go:91-100`). It is called by:
  - install (`changes.go:259`), upgrade and rollback (`:293`) and enable (`:311`);
  - token issue and check (`tokens.go:62, 142, 199`);
  - the sweep that stops enabled installations after a mark (`operations.go:182-194`).
- **Approval.** `Propose` and `Approve` both run `check` (`registry.go:199, 236`). `Recheck` is the dry run that the read uses for `approve_blocked` (`signing.go:7-18`; `apps/api/server/integrations.go:442-455`).
- **Search.** `modules/plugins` and `apps/api/server/integration*.go` contain no license field or rule. This agrees with HYB-301 (`report.md:211-216`).
- **Server.**
  - The integrations read returns versions without a license (`pluginVersionRow`, `integrations.go:113-130`; `versionOf`, `:377-397`).
  - Commands map module refusals to command codes in `mapIntegration` (`integrationcmds.go:41-69`). An unlisted refusal reason becomes `invalid_stage`, as `vulnerable` and `revoked` do.
  - Every plugin command uses `AdministerInstallation` (`integrationcmds.go:394`). This is the deviation that HYB-310 tracks.
  - The registry is built with `plugins.Config{Core, Destinations, Random}` (`server.go:269`). Its state is in memory and is lost on restart.
- **Seed.** `sampleManifest(version)` (`integrationseed.go:288-296`) and `seedPlugins` (`:298-341`) sign versions 1.0.0 and 1.1.0 with a key generated at start. The seed then installs and enables 1.0.0, leaves an upgrade to 1.1.0 pending (proposal 8) and leaves an endpoint pending (proposal 9).
- **Installation settings.** Start flags with defaults and validation exist for access and organization settings (`apps/api/cmd/hb-api/main.go:92-139`; `server/orgsettings.go`; `settings.go:115`). The glossary defines "Installation setting" (`product-glossary.md:85`).
- **OpenAPI.** The document lists commands only (`server/openapi.go:13-37`, `modules/commands/openapi.go`). Reads are not described, and a search finds no `x-reads` key in the repository.
- **Console.**
  - The service parser `versionOf` has no license field (`apps/console/src/service/integrations.ts:236-260`).
  - The model decides which changes to offer with `runnable` (trusted publisher and key; not vulnerable unless in a verification workspace; `live-model.ts:189-193`) and `proposeOffer` (`:202-237`).
  - `dryRunBlocked` lists the reasons that add the spec's blocked sentence (`live-model.ts:144`; `live-copy.ts:458`; sentence at `live-codes.ts:260-264`).
  - The Versions table and its detail fold are in `LiveModules.tsx:286-375`.

## 3. Design

### 3.1 Data model (`modules/plugins`)

- **`Manifest.License string`.** One license identifier, required. The grammar has exactly two forms (review F1):
  - **Reference form.** The exact prefix `LicenseRef-` followed by 1–53 bytes from `[A-Za-z0-9.-]`, so 12–64 bytes in total.
  - **Plain form.** 1–64 bytes from `[A-Za-z0-9.-]` that start with a letter or digit. A plain identifier must not start with `licenseref-` in any letter case, so the two forms never overlap.
  - Anything else is refused, including `LicenseRef-` on its own, `licenseref-x`, `-` and `.`. `MIT-` is accepted in the plain form: it is well formed, and it activates only if it is on the permitted list.
  - It is not a license expression. Spaces, parentheses, `+`, `OR`, `AND` and `WITH` are refused because the characters or the grammar do not allow them. Expressions are owner question Q4.
  - The same grammar applies to each entry of the permitted list.
- **Signed encoding.**
  - `encode()` writes `License` immediately after `Digest`, and the tag becomes `hybrid-billing/plugin/v2`. A v1 signature therefore cannot verify against a v2 manifest that adds a license.
  - No v1 signature is kept anywhere: the seed signs at every start, and plugin state is not persisted.
  - The new known-answer hash `manifestAnswer` is built in the test independently of `encode()` (review F8). The test writes the length-prefixed fields by hand in the documented order, as the challenge known answer already does (`form_test.go:255`), so a wrong `encode()` cannot confirm itself.
  - `"license"` is added to the distinct-encoding table.
  - `docs/features/HB-242/plan.md:128` still describes the signed message under tag `hybrid-billing/plugin/v1`. That merged plan is not edited. This plan and the verification record state that v2 replaces it from HYB-309 on.
- **`Config.Licenses []string`.** The installation's permitted plugin licenses:
  - 0–64 entries, each in the manifest form, strictly ascending in byte order, with no duplicates;
  - `NewRegistry` returns `ErrInvalid` otherwise;
  - an empty list is valid and means that no plugin can be activated;
  - the registry keeps a copy, fixed for the life of the registry;
  - matching is exact and case-sensitive (owner question Q7).
- **Reason.** `ReasonLicenseNotPermitted = "license_not_permitted"`.
- **Reads.**
  - `VersionView.Permitted bool` is computed by the same function the rule uses.
  - `(*Registry).Licenses() []string` returns a copy of the list.

### 3.2 The check before activation

`runnable(w, v)` gains one case, in this order:
1. revoked publisher or compromised key → `revoked`;
2. `v.manifest.License` not in `Config.Licenses` → `license_not_permitted`;
3. vulnerable outside a verification workspace → `vulnerable`.

Effects:
- **Primary check.** `Propose` refuses install, upgrade, rollback and enable of a version that is not permitted. This is the only licence refusal an API caller or the console can reach in this build.
- **Backup checks (review F2).** The list is fixed for the life of the registry and plugin state is not persisted. The following checks therefore cannot be reached through the service in this build:
  - `Approve` and `Recheck`, and therefore `approve_blocked = license_not_permitted` in the read;
  - the `stopUnrunnable` sweep;
  - token issue and check (`tokens.go:62, 142`);
  - the enabled-state check at `tokens.go:199`, used by the egress path.

  They are kept as backups for later persistence or a list change at run time. They are tested white-box in package `plugins`: a test changes `r.config.Licenses` between `Propose` and `Approve`, and between enable and the token, sweep and egress calls. The read field `approve_blocked = license_not_permitted` and its console rendering are covered by console model fixtures only, not by browser QA. Mutation survivors are handled as follows:
  - a survivor in a backup check that the white-box tests reach is killed by a test;
  - a survivor that only the service path could kill is recorded as unreachable in this build, with the same reason for every backup check, never as dead code.
- **No exemption for verification workspaces.** A vulnerable version may run there, but the plugin's license terms apply wherever its code runs (owner question Q3).
- **Registration is not refused** for a license that is well formed but not permitted. The version is recorded, P02 shows it as not permitted, and it cannot be activated. Signature and compatibility are refused at registration because they say whether the package is genuine and fits. Permission is an installation decision that can change, so it is checked when activating (owner question Q6).
- **Unknown licenses.** An identifier that is not on the list is refused exactly like any other non-permitted license, whether it is a real SPDX identifier outside the list, a `LicenseRef-`, or a misspelling. There is no "unrecognised but allowed" state. A missing or malformed identifier makes the manifest invalid, so the version cannot be registered.
- **Override.** None. No command, role or flag permits one plugin. The installation list (section 4) applies to every plugin.

### 3.3 Server (`apps/api/server`, `apps/api/cmd/hb-api`)

- **Installation setting `plugin_licenses` (review F5).**
  - The start flag `-plugin-licenses` takes a comma-separated list that replaces the default.
  - `-plugin-licenses=` (an empty value) means the empty list: no license is permitted.
  - Every element must match the license grammar (3.1). Start is refused with `ErrConfig: plugin_licenses` for:
    - whitespace anywhere in the value;
    - an empty element (`a,,b`, a leading or a trailing comma);
    - a malformed or duplicate entry;
    - more than 64 entries.
    Nothing is trimmed or repaired: the check fails closed.
  - Giving the flag twice is refused with the same error.
  - Wiring (review L1, root decision):
    - `main.go` does not validate. Its `flag.Func` handler keeps the raw string, and counts how often the flag was used, in `server.Config` (`PluginLicenses{Raw, Given, Uses}`).
    - `Given` is set through the `fs.Visit` pattern of the existing settings (`apps/api/cmd/hb-api/main.go:117-123`).
    - `server.New` validates every case, including a repeated flag (`Uses > 1`), and returns an `ErrConfig` whose message ends in `plugin_licenses`.
    - The cases are tested in the server settings tests. `apps/api/cmd/hb-api/main_test.go` is added only if the flag wiring needs its own test.
  - The read reports `start_flag` exactly when the flag was given, including `-plugin-licenses=`.
  - The server sorts the list before passing it to `plugins.Config.Licenses`.
- **Sample seed.**
  - `sampleManifest(version, license)`. Versions 1.0.0 and 1.1.0 declare `Apache-2.0`. A new sample version 1.2.0 declares `LicenseRef-sample-proprietary`. Its storage contract is the same, so only the license blocks it.
  - 1.2.0 is registered by a propose and approve appended after the pending endpoint, so it is proposal 10. Proposals 8 and 9, which the QA runner relies on, stay as they are.
  - Before seeding, `server.New` checks that the list contains `Apache-2.0`. If it does not, start is refused with an `ErrConfig` whose message names the cause and ends in `plugin_licenses` ("the sample seed needs Apache-2.0 in plugin_licenses"). No seed step runs, and the seed never adds to the list.
- **Read `GET /api/v1/integrations`.** Additive fields only:
  - `modules.plugins[].license`: the declared identifier;
  - `modules.plugins[].license_state`: `permitted` or `not_permitted`;
  - `modules.licenses`: `{ "permitted": ["Apache-2.0", …], "source": "default" | "start_flag" }` (root amendment after review round 2: the values of the existing settings read, `apps/api/server/settings.go:11-14`).
  - No `summary.license` (review F7). The console reads the target version's license from `modules.plugins[]`, so a field in the proposal summary would have no consumer.
- **Commands.** No new command, field or enum value.
  - A plugin proposal refused for the license returns `invalid_stage` from the existing default in `mapIntegration`. This matches `vulnerable` and `revoked` (Q8 default, kept after review F6).
  - The refusal happens before any state change, journal entry or revision bump. An idempotent resend returns the same refusal.
  - **What an API caller learns.** The command error does not name the license. A caller learns the reason only from `license_state` in the read. This is recorded as a known limit of the stable-code contract, not a hidden behaviour.
- **OpenAPI.** The command catalogue does not change. The read schema is not in the OpenAPI document today. The new read fields are recorded in the P02 field map (`screen-binding-fields/operations.md`) and the binding register. The read stays outside OpenAPI, as before. This is a residual, not something this ticket adds.

### 3.4 Console P02 (Modules view)

- **Parser** (`integrations.ts`).
  - `license: str(r.license)`.
  - `license_state` is kept only if it is `permitted` or `not_permitted`. Any other value, including a missing one, becomes `unknown`.
  - `licenses: { permitted: strs(...), source }`. An unknown source shows as unknown.
- **Model** (`live-model.ts`).
  - `runnable(page, v)` also requires `v.license_state === "permitted"`. This fails closed: unknown is not runnable.
  - `proposeOffer` therefore never offers install, upgrade, rollback or enable to a version that is not permitted. It returns those versions so that the form can say why.
  - A new blocker code `license` is used when every runnable candidate was excluded only by its license. Mixed reasons keep `untrusted`, whose sentence now includes the license case.
  - `dryRunBlocked` gains `license_not_permitted`, so the spec's blocked sentence (`P02/spec.md:98`) follows the reason.
- **Versions table** (`LiveModules.tsx`, Verification column).
  - A line after "Signed by …": "License: ⁨Apache-2.0⁩ · Permitted on this installation", or "… · Not permitted on this installation. It cannot be installed, upgraded to or enabled."
  - The status is shown in words with the existing tag style. Colour is never the only signal, and there is no coloured border.
  - The identifier is isolated as left-to-right text in AR (`ltr()`).
- **Permitted list.** A short block at the top of the Modules view:
  - "Permitted plugin licenses (installation default): …" or "(installation setting): …";
  - "Set when the server starts. This list is an installation policy, not a legal review." An empty list reads "No plugin license is permitted on this installation, so no plugin can be installed, upgraded to or enabled."
- **Propose form.** For each version not offered because of its license, a note: "Version ⁨1.2.0⁩ is not offered: its license ⁨LicenseRef-sample-proprietary⁩ is not permitted on this installation."
- **Proposals (review F7; `LiveProposals.tsx`).**
  - Install, upgrade and rollback proposals show the target version's license and its state, looked up in `modules.plugins[]` by plugin and version. If the version is not in the list, the license reads unknown.
  - A proposal whose approval is blocked shows `approve_blocked = license_not_permitted` with the reason sentence and the blocked sentence. No Approve button is shown, as for the other dry-run blockers. Model fixtures cover this; the service cannot produce it in this build (3.2).
- **Refusal fallback (review F6).** When `integrations.plugin_propose` or `plugin_approve` comes back refused, and the target version's `license_state` is not `permitted`, the kept-refusal message adds the license sentence. The check uses the read that is fetched again after the refusal, not the page as it was before sending (review L3). This can happen when the read changes between loading the page and sending. Otherwise the message is unchanged.

**Copy.**
- EN becomes final only after the TW review. KO and AR are AI drafts pending native-language review.
- The TW review decides the AR rendering of "installation default" and the term for "plugin" (review F9).
- "Installed or enabled" covers every refused change, so the copy reads "installed, upgraded to or enabled" (review F10). A rollback is an install of an older version, so it is covered by "installed".

| Key | EN | KO | AR |
| --- | --- | --- | --- |
| `license_not_permitted` (blocker) | The version's license is not on this installation's list of permitted plugin licenses. | 버전의 라이선스가 이 설치의 허용 플러그인 라이선스 목록에 없습니다. | ترخيص الإصدار ليس ضمن قائمة تراخيص الإضافات المسموح بها في هذا التثبيت. |
| license permitted | License: {id} · Permitted on this installation | 라이선스: {id} · 이 설치에서 허용됨 | الترخيص: {id} · مسموح به في هذا التثبيت |
| license not permitted | License: {id} · Not permitted on this installation. It cannot be installed, upgraded to or enabled. | 라이선스: {id} · 이 설치에서 허용되지 않음. 설치·업그레이드·사용 설정할 수 없습니다. | الترخيص: {id} · غير مسموح به في هذا التثبيت. لا يمكن تثبيته أو الترقية إليه أو تفعيله. |
| license unknown | License: unknown. Treated as not permitted. | 라이선스: 알 수 없음. 허용되지 않은 것으로 처리합니다. | الترخيص: غير معروف. يُعامل على أنه غير مسموح به. |
| list heading | Permitted plugin licenses ({source}) | 허용 플러그인 라이선스({source}) | تراخيص الإضافات المسموح بها ({source}) |
| source default / setting | installation default / installation setting | 설치 기본값 / 설치 설정 | الإعداد الافتراضي للتثبيت / إعداد التثبيت |
| list note | Set when the server starts. This list is an installation policy, not a legal review. | 서버 시작 시 설정합니다. 이 목록은 설치 정책이며 법률 검토가 아닙니다. | يُحدَّد عند بدء تشغيل الخادم. هذه القائمة سياسة تثبيت وليست مراجعة قانونية. |
| empty list | No plugin license is permitted on this installation, so no plugin can be installed, upgraded to or enabled. | 이 설치에서 허용된 플러그인 라이선스가 없어 어떤 플러그인도 설치·업그레이드·사용 설정할 수 없습니다. | لا يُسمح بأي ترخيص إضافة في هذا التثبيت، لذا لا يمكن تثبيت أي إضافة أو الترقية إليها أو تفعيلها. |
| not offered note | Version {v} is not offered: its license {id} is not permitted on this installation. | {v} 버전은 제안할 수 없습니다. 라이선스 {id}이(가) 이 설치에서 허용되지 않습니다. | لا يُعرض الإصدار {v}: ترخيصه {id} غير مسموح به في هذا التثبيت. |
| blocker `license` | No registered version can be proposed: the license of each is not permitted on this installation. | 제안할 수 있는 등록 버전이 없습니다. 모든 버전의 라이선스가 이 설치에서 허용되지 않습니다. | لا يمكن اقتراح أي إصدار مسجّل: ترخيص كل منها غير مسموح به في هذا التثبيت. |

**Glossary.** Three new rows in `product-glossary.md`:

| Concept | KO | EN | AR | Definition and forbidden implications |
| --- | --- | --- | --- | --- |
| plugin-license / manifest `License` | 플러그인 라이선스 | Plugin license | ترخيص الإضافة | The license under which a plugin publisher offers its module, as one identifier in the signed manifest. It is not the product's own license, an installation entitlement (P07) or a Marketplace right, and it is not a legal compliance finding. |
| permitted-plugin-licenses / `-plugin-licenses` | 허용 플러그인 라이선스 | Permitted plugin licenses | تراخيص الإضافات المسموح بها | An installation setting: the license identifiers whose plugins this installation may install, upgrade to or enable. It is not editable on screen and is not a per-plugin exception or a legal review. |
| plugin / `modules/plugins` | 플러그인 | Plugin | الإضافة (TW to confirm) | A separately published module that an installation registers, installs and enables under a signed manifest. It is not a product component covered by the product's own license, and not an external app opened in its own origin (QA-06). |

"Plugin" was used on P02 without a glossary entry; this ticket adds one (review F9). The TW review confirms the AR term and the AR rendering of "installation default".

### 3.5 Alternatives considered

- **Use the installation entitlement (`modules/licensemode`).** Rejected. It would mix product licensing into third-party module rights (HB-40), and `product-policy.md:23` states that the product-license policy added no activation check.
- **Refuse registration of non-permitted licenses.** Rejected for now. P02 and P07 must be able to show a module's license even when this installation does not permit it. Owner question Q6.
- **Reuse the `modules/thirdparty` table directly.** Not possible as an import: `modules/plugins` may import only the standard library (`source_test.go`). Its `separate` column is used for the default list instead (section 4).
- **Allow-with-warning for unknown licenses.** Rejected. It is not fail-closed, and the spec has no warning state, only blocked (`P02/spec.md:98`).

## 4. Open values as changeable installation defaults

| Value | Installation default | How to change | Basis |
| --- | --- | --- | --- |
| Permitted plugin licenses | `Apache-2.0`, `BSD-2-Clause`, `BSD-3-Clause`, `ISC`, `LGPL-2.1-only`, `LGPL-3.0-only`, `MIT`, `MPL-2.0`, `Zlib` (9) | `-plugin-licenses` at start | A provisional stand-in for owner question Q2, which is unconfirmed (review F3). Taken from the software licences in the `separate` column of the HB-253 table (`HB-253/plan.md:92-100`). See the note below this table. |
| Maximum list size | 64 | Constant in this build | A bound for validation and display, not a policy |
| Matching | Exact, case-sensitive | Not configurable in this build | Fail-closed; Q7 |

**Note on the default list (review F3, root decision).**
- The HB-253 table and this list answer different questions:
  - the HB-253 table is a **distribution rule**: whether the product may ship a third-party component under that licence, bundled or separately;
  - this list is a **permission to run**: whether this installation may install and enable a plugin offered under that licence.
- The `separate` column is used only because it is the closest existing engineering boundary. It does not show that the two questions have the same answer.
- `CC-BY-4.0`, `CC-BY-SA-4.0` and `OFL-1.1` are in that column but are dropped here, because they are licences for content and fonts, not for software.
- `GPL-*`, `AGPL-*` and `PolyForm-Noncommercial-1.0.0` are not in the column and stay excluded.
- The list is not a legal review.

The read reports whether the list is the installation default or an installation setting, so a changed value is never presented as the product's position.

## 5. Tests

**`modules/plugins`**
- Form, for the manifest and for each permitted-list entry (review F1):
  - accepted: `MIT`, `MIT-`, `0BSD`, `LicenseRef-acme`, a 64-byte plain identifier, a 64-byte reference;
  - refused as `ErrInvalid`:
    - empty, a 65-byte plain identifier, a 65-byte reference;
    - `LicenseRef-` alone, `licenseref-x`, `LICENSEREF-x`;
    - `-`, `.`, `-MIT`, `.MIT`;
    - `MIT OR Apache-2.0`, `(MIT)`, `GPL-2.0+`;
    - a non-ASCII character, a control character.
- Encoding:
  - a new known-answer hash and the v2 tag;
  - `license` encodes differently from every other field change;
  - a signature over a manifest with license A does not verify with license B (`signature_invalid`).
- Config: `NewRegistry` refuses unsorted, duplicate, malformed and 65-entry lists, and accepts an empty list.
- Lifecycle, for each of install, upgrade, rollback and enable:
  - refused with `license_not_permitted` at `Propose`;
  - white-box backup checks (review F2): after a permitted `Propose`, the test removes the licence from `r.config.Licenses`; `Recheck` and `Approve` must then refuse with `license_not_permitted`;
  - the same white-box change after enable: `stopUnrunnable` (driven by a mark) stops the installation with that reason, `IssueToken` and `CheckToken` refuse, and the egress path refuses;
  - no state change, no revision bump and no proposal recorded on refusal;
  - a permitted license behaves exactly as before.
- Rules:
  - registration of a non-permitted version succeeds and `VersionView.Permitted` is false;
  - a verification workspace is not exempt;
  - reason order: `revoked` before `license_not_permitted` before `vulnerable`.
- Property test: random manifests, permitted lists and workspaces. An independent model gives "activation allowed if and only if trusted, license on the list, and (not vulnerable or verification)". The registry must agree on `Propose`, `Approve` and `Recheck`.

**`apps/api/server`**
- Read:
  - three versions;
  - `license` and `license_state` for each, with 1.2.0 `not_permitted`;
  - `modules.licenses.permitted` equals the default and `source` is `default`;
  - `source` is `start_flag` when the flag is given.
- Commands:
  - a propose to upgrade to 1.2.0 returns `invalid_stage`, records nothing, and an idempotent resend returns the same result;
  - with `-plugin-licenses Apache-2.0,LicenseRef-sample-proprietary`, the same upgrade is proposed and then approved by the second administrator.
- Settings (review F5), one test each:
  - the flag is parsed and sorted;
  - `-plugin-licenses=` gives the empty list, with `source` `start_flag`;
  - whitespace (`MIT, ISC`, ` MIT`), an empty element (`MIT,,ISC`, `,MIT`, `MIT,`), a malformed entry, a duplicate and 65 entries each refuse start with `ErrConfig: plugin_licenses`;
  - a repeated flag refuses start;
  - with the seed, a list without `Apache-2.0` (including the empty list) refuses start with the named message before any seed step;
  - an empty list without the seed starts.
- OpenAPI: the command catalogue is unchanged. The existing catalogue test is updated only if it hashes the read.
- The seed world test still passes with the same step count.

**Console**
- Parser: unknown `license_state` becomes `unknown`; a missing `licenses` object gives an unknown source and an unknown list (`permitted: null`), which the screen shows as "The list of permitted plugin licenses is unknown", not as an empty list (review round 3, I1).
- Model:
  - `runnable` is false for `not_permitted` and `unknown`;
  - fixtures with `approve_blocked = license_not_permitted` render the reason and the blocked sentence and offer no Approve (review F2);
  - proposals show the target version's licence, or unknown when the version is missing (review F7);
  - the refusal fallback adds the licence sentence only when the target version is not permitted (review F6);
  - `proposeOffer` excludes those versions and returns the `license` blocker when they are the only candidates;
  - `dryRunBlocked` includes `license_not_permitted`.
- Copy: every new key exists in EN, KO and AR (`allWordTables`).

**Mutation.** Operator mutants on the changed lines of `manifest.go`, `changes.go`, `registry.go`, `operations.go`, `integrations.go`, `integrationseed.go` and the settings file. Each batch runs in its own copy with its own `GOCACHE`, which is cleaned between batches. A survivor is killed by a test, or recorded as equivalent with a reason. Backup-check survivors follow the single rule in section 3.2. The tests are not weakened to remove a survivor.

**Builds.** `go vet`, `go test -race ./modules/plugins ./apps/api/...`, the cross builds used by HB-284, `npm test`, `tsc` and the console build.

## 6. Browser QA and regression

The runner is a copy of `qa285fullon297.mjs` (scratchpad of session `ff7545a1-…`) named `qa309.mjs`, against a worktree build with the sample seed.

**New checks**
1. Modules view in EN, KO and AR at 1440 and 390:
   - three versions;
   - 1.0.0 and 1.1.0 read "Permitted on this installation" and 1.2.0 reads "Not permitted …";
   - the permitted list block shows "installation default";
   - no horizontal scroll;
   - the AR identifier renders left to right;
   - neutral tag borders.
2. Propose form on the installed plugin: no "Upgrade to 1.2.0" radio, and the "not offered" note names 1.2.0 and its license.
3. A server started with `-plugin-licenses Apache-2.0,LicenseRef-sample-proprietary`:
   - 1.2.0 reads permitted, the source reads "installation setting", and the upgrade is offered;
   - admin1 proposes, admin2 approves, and the installation reads 1.2.0.
4. From the command line: a server started with `-plugin-licenses=MIT` and the seed refuses to start with the named message; `-plugin-licenses="MIT, ISC"` and a repeated flag refuse with `plugin_licenses`.
5. Not in browser QA: `approve_blocked = license_not_permitted`. The service cannot produce it in this build (3.2), so it is covered by model fixtures only.
6. Keyboard: the license line and the note are reached in reading order; the not-offered note is read once.

**Regression.** Groups A–D of `qa285fullon297.mjs` must pass unchanged, including:
- proposals 8 and 9;
- the upgrade to 1.1.0;
- stop, disable, enable, and the rollback to 1.0.0;
- subscriptions, dead letters, safe resends and fail-closed unknowns.
Screenshots of the changed states go to `docs/features/HYB-309/` only if `docs/artifact-layout.md` allows them there; otherwise they stay in the scratchpad and are referenced.

## 7. Requirement cross-check (HB-257, v1.16)

| Requirement | Source | Covered by this plan | Not covered |
| --- | --- | --- | --- |
| HB-40 license boundary (P02 part) | `requirements.md:717, 723`; `P02/spec.md:50, 74` | Manifest license, registration form check, refusal before activation, display | Legal suitability (stated as out of scope) |
| HB-40 distribution notices | `requirements.md:723`; `ui-design.md:163` | none | P07 module notices and downloads (HYB-328) |
| HB-40 history after expiry, upgrade and migration | `requirements.md:723` | none (P01, P03 and P07 scope) | As HYB-301 records |
| DB-14 Marketplace rights | `detailed-requirements.md:341-351` | none (later scope) | Later scope |
| P02 blocked state | `P02/spec.md:98` | The blocked sentence is shown with `license_not_permitted` | none |
| P02 activation permission "적격 승인 권한" | `P02/spec.md:74` | Second-administrator approval kept | Separate permission (HYB-310) |
| HB-257 phase 5 P02 binding | `HB-257/plan.md:168` | Read fields and console binding extended | none |

## 8. Residuals

- **P07 notices: ticket [HYB-328](https://linear.app/hybrid-billing/issue/HYB-328) (review F11).** The manifest carries the license identifier only. P07 needs the module's license text and notice ("모듈 라이선스·고지 reference[]", `P07/spec.md:53`). Adding their digests to the manifest would change the signed format again (v3). Owner question Q9.
- **Expressions.** License expressions (`OR`, `AND`, `WITH`, `+`) are refused at registration (Q4).
- **Changing the list.** The list can be changed only at start. A two-person change on screen is not built (Q5).
- **Persistence.** Plugin state is in memory. When it is persisted, a load with a narrower list must stop installations that are no longer permitted, as `stopUnrunnable` does after a mark (Q10).
- **Registration from the console.** Version registration from the console is still not connected (HB-284 residual), so a non-permitted version reaches P02 only through the seed.
- **Permission split.** P02's separate extension, verification and approval permissions are HYB-310.
- **No legal review** of the default list or of the HB-253 table.
- **OpenAPI** does not describe reads.

## 9. Owner questions and defaults

Development proceeds on the default unless the owner decides otherwise.

| # | Question | Default |
| --- | --- | --- |
| Q1 | Is the P02 "license" the plugin's own license, or a commercial entitlement to run it? | The plugin's own license (section 1). Entitlements stay in P07 and `licensemode`. |
| Q2 | Which licenses are permitted by default? | The provisional stand-in in section 4: 9 software licences from the HB-253 `separate` column. Alternative: an empty list, so nothing activates until the operator lists licenses. |
| Q3 | Are verification workspaces exempt, as they are for vulnerable versions? | No. |
| Q4 | Should license expressions be supported? | No. A single identifier; expressions are refused at registration. |
| Q5 | Should the list be changeable on screen with two-person approval? | No. Start flag only in this build. |
| Q6 | Should registration refuse a non-permitted license? | No. Register it, show it, and refuse activation. |
| Q7 | Should matching ignore case? | No. Exact match. |
| Q8 | Should the command code be `prerequisite_failed` instead of `invalid_stage`? | `invalid_stage`, as for `vulnerable` and `revoked`. The reason comes from `license_state` in the read (review L2). |
| Q9 | Should the license text and notice digests be added to the manifest now, ahead of P07? | No. Identifier only; P07 is HYB-328. |
| Q10 | When persistence arrives, should installations that are no longer permitted be stopped at load? | Yes, with reason `license_not_permitted`. |
| Q11 | May the fictional sample plugin declare `Apache-2.0`? | Yes, labelled as sample data. It is not a license grant. |

## 10. Verification and completion

- `docs/features/HYB-309/verification.md` records the commands run and the results observed, with PASS, FAIL or UNVERIFIED for each.
- Nothing that was not run is marked as passed.
- Order of work:
  1. independent design review of this plan;
  2. implementation;
  3. change review and re-review of any fix;
  4. TW review of the rendered copy;
  5. QA;
  6. PR;
  7. merge only after the PR state reads `MERGED`;
  8. ticket closure and machine release.

## Review round 1 (2026-11-18)

Design review by an independent read-only reviewer: CHANGES REQUIRED, 3 Medium, no High. F3, F5 and F7 are root decisions. All findings are applied in this revision. The plan stays a draft until re-review.

| # | Finding | Change in this plan |
| --- | --- | --- |
| F1 | The licence grammar overlapped (`LicenseRef-` could also be read as a plain identifier) and had untested edges | One grammar with two disjoint forms (3.1). A plain identifier must start with a letter or digit and not with `licenseref-` in any case. Edge cases are added to the form tests (5). |
| F2 | The Approve, Recheck, sweep, token and egress checks cannot be reached through the service | Stated as backup checks unreachable in this build (3.2). White-box tests change `r.config.Licenses` between calls. `approve_blocked` is covered by model fixtures only, not browser QA (6). One rule for mutation survivors. |
| F3 | The default list reused a distribution rule as a permission to run | Marked as a provisional stand-in for Q2, with the difference explained (4). `CC-BY-4.0`, `CC-BY-SA-4.0` and `OFL-1.1` are dropped. |
| F4 | The interpretation and the product-policy citation were overstated | "Strongly supported" (1), and `product-policy.md:23` "states that the product-license policy added no activation check" (1, 3.5). |
| F5 | The setting's edge cases were unspecified | `-plugin-licenses=` is the empty list. Whitespace or empty elements and a repeated flag are refused with `ErrConfig`. `Given` uses the `fs.Visit` pattern. The seed's need for `Apache-2.0` is checked before seeding with a named message. A test for each (3.3, 5). |
| F6 | The refusal code hides the licence reason | `invalid_stage` kept (Q8). Recorded that an API caller learns the reason only from `license_state`. The console refusal fallback names the licence when the target version is not permitted (3.3, 3.4). |
| F7 | `summary.license` had no consumer, and proposals did not show the target licence | `summary.license` dropped. Install, upgrade and rollback proposals show the target version's licence. `LiveProposals.tsx` is added to the allowed files. |
| F8 | The known-answer hash would be computed by `encode()` itself; a merged plan describes v1 | The hash is built independently of `encode()`. `HB-242/plan.md:128` is mentioned, not edited (3.1). |
| F9, F10 | AR term choices; the copy omitted upgrade | The AR "installation default" and the plugin term are left to the TW review, and a "plugin" glossary row is added. The copy reads "installed, upgraded to or enabled" (3.4). |
| F11 | P07 notices now have a ticket | The residual and the cross-check reference HYB-328 (7, 8). |

## Review round 2 (2026-11-18)

Review round 2: APPROVE (L1–L3 folded in).

- L1: `main.go` keeps the raw flag value, `Given` and a use count; `server.New` validates (3.3).
- L2: the Q8 default names `license_state` in the read as the source of the reason.
- L3: the console fallback uses the read fetched again after the refusal (3.4).

## Root amendment after review round 2 (2026-11-18)

- `modules.licenses.source` uses `default` and `start_flag`, the values of the existing settings read (`apps/api/server/settings.go:11-14`), instead of `installation_default` and `installation_setting`. Root decided this so that one setting-source vocabulary is used across reads.
- The displayed words are unchanged: "installation default" and "installation setting".

## Review round 3 (2026-11-18): change review PASS (2 Low), render and copy review FAIL (2 Medium, 4 Low)

All findings were applied as root decided (`scratchpad/hb309-fixes.md`). These entries replace the copy table rows and the sections above where they differ.

| # | Finding | Change |
| --- | --- | --- |
| M1 | "installation setting" and "Set when the server starts" did not tell how the list changes | The `start_flag` source reads "set when the server started" / "서버 시작 시 지정" / "حُدّد عند بدء الخادم". The list note reads "An installation setting: it changes only when the server is restarted with a different list. This list is an installation policy, not a legal review." / "설치 설정입니다. 다른 목록으로 서버를 다시 시작해야 바뀝니다. 이 목록은 설치 정책이며 법률 검토가 아닙니다." / "هذا إعداد تثبيت، ولا يتغيّر إلا بإعادة تشغيل الخادم بقائمة مختلفة. هذه القائمة سياسة تثبيت وليست مراجعة قانونية." |
| M2 | The licence refusal followed a message that suggested a pending-proposal limit | When the refused target version's licence is not permitted, the message leads with "The plugin registry refused this change. Nothing changed." and then reads "Version {v} has the license {id}, which is not permitted on this installation. Choose another version; the permitted list changes only when the server is restarted with a different list." (KO and AR as in `live-copy.ts`). Browser QA triggers and captures this refusal in EN, KO and AR. |
| L1 | The not-permitted version line ran into one line | Two lines: "License: {id}" in regular weight with the identifier in mono; then the state. For a not-permitted version only "Not permitted on this installation." is at weight 600, followed by "It cannot be installed, upgraded to or enabled." in regular weight. `licenseStateWords` is split accordingly. |
| L2 | Identifiers in the note and the refusal were not mono | The version and licence identifiers in the not-offered note and the refusal render through `ltr()` (mono, left to right). |
| L3 | The note was read before the choices were known | With the propose form open, the note is the last item of `fieldset.p02-choices` and the fieldset names it with `aria-describedby`. With the form closed it keeps its place under the row note. This is a deviation from section 3.4 ("Propose form"). |
| L4 | Enable proposals did not show a licence | Enable proposals show the installed version's licence ("License of {installed version}"). |
| F1 | "Not permitted" wording was used for an unknown state | The not-permitted wording appears only when `license_state` is `not_permitted`. An unknown state, or a version missing from the read, uses unknown variants: "Version {v} is not offered: its license is unknown, so it is treated as not permitted.", "The license of version {v} is unknown, so it is treated as not permitted. …", and the blocker `license_unknown`. For the installed version the note reads "Enable is not offered: the installed version {v} has the license {id}, which is not permitted on this installation." (or its unknown variant). |
| F2 | The property test did not compare the dry run with the approval | The property steps for install, enable, upgrade, rollback and remove call `Recheck` before `Approve` and fail on any difference. |
| I1 | A missing `licenses` object was described as empty | Corrected above: it shows as unknown. |
| I2 | `Licenses()` bypassed the fault hook | The read calls it through `pluginCall("licenses", …)`. The read-fault test includes `plugins-licenses`. |
| Glossary | — | Adds an installation-default row (الإعداد الافتراضي للتثبيت). The AR plugin term is الإضافة, kept for consistency with P02. Native-language review is pending. |

## Review round 4 (2026-11-18): change PASS, render and copy PASS, with lows

| # | Finding | Change |
| --- | --- | --- |
| L5, N1 | The refusal's next step did not fit every change kind | The refusal is built from three parts:<br>1. "The plugin registry refused this change. Nothing changed.";<br>2. the licence sentence, or its unknown variant;<br>3. a next step that depends on the context. **Propose:** "Choose another version; the permitted list changes only when the server is restarted with a different list." **Enable:** "Propose an upgrade or rollback to a permitted version instead." **Approve:** "This proposal cannot be approved; the proposer can withdraw it. The permitted list changes only when the server is restarted with a different list."<br>KO and AR are as in `live-copy.ts` (`keptLicenseNext`). The final wording is subject to the TW review. |
| N2 | The note inside the fieldset was read twice | The note stays inside `fieldset.p02-choices` as its last item, and the fieldset has no `aria-describedby`. QA checks the accessibility tree. This replaces the `aria-describedby` part of L3. |
| L6 | Hyphen wrapping of long identifiers | No change: this is the existing pattern. |
| Deviation | `LiveCommand.tsx` was not in the allowed files | `RowNote.kept` was widened from `string` to `ReactNode`, so that the refusal can render identifiers in mono. Reviewed: the only callers are the integrations pages, which still pass strings except for the licence refusal. Other screens have their own `RowNote`. |
