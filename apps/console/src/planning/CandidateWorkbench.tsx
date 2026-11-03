import { useEffect, useMemo, useState } from "react";
import type { Locale, Mode } from "../shared/types";
import type { PlanningScreen } from "./registry";
import { candidateModels } from "./models";
import { customerTasks } from "../modules/customer-portal/shared-candidates";
import { local, tabLabel } from "./copy";
import { field, valueText } from "./fields";
import CandidateTable, { type CandidateRow } from "./CandidateTable";
import CandidateDetail from "./CandidateDetail";
import DraftForm from "./DraftForm";
import CommandBoundary from "./CommandBoundary";
import Inbox from "./Inbox";
import {
  OrganizationTree,
  PermissionMatrix,
  DocumentPreview,
  DisclosurePreview,
  ReportChart,
  LifecyclePanel,
  Conversation,
} from "./SpecialPanels";
export type CandidateProps = {
  screen: PlanningScreen;
  locale: Locale;
  mode: Mode;
  tab: string;
  scope: string;
  objectId?: string;
  canDiscard: () => boolean;
  period: string;
  currency: string;
  organization: string;
  scenario: string;
  customer: boolean;
  onDirty: (dirty: boolean) => void;
  onState: (state: string, items: string[]) => void;
};
export default function CandidateWorkbench({
  screen,
  locale: l,
  mode,
  tab,
  scope,
  objectId,
  canDiscard,
  period,
  currency,
  organization,
  scenario,
  customer,
  onDirty,
  onState,
}: CandidateProps) {
  const model =
      customer && screen.id === "U03"
        ? customerTasks
        : candidateModels[screen.id],
    [query, setQuery] = useState(""),
    [selected, setSelected] = useState<CandidateRow | null>(null),
    [tree, setTree] = useState(""),
    [draft, setDraft] = useState(false),
    [inboxId, setInboxId] = useState("");
  const ready = mode === "synthetic" && scenario === "normal";
  const rows = useMemo<CandidateRow[]>(
    () =>
      !ready || !model
        ? []
        : Array.from({ length: screen.id === "M05" ? 1 : 3 }, (_, i) => ({
            id: `${customer ? "PUBLIC" : "DEMO"}-${screen.id}-${i + 1}`,
            values: {
              ...model.values,
              ...(model.values.document
                ? { document: `${model.values.document}-${i + 1}` }
                : {}),
              id: `${screen.id}-${i + 1}`,
              revision: "r1",
              evidence: `${customer ? "PUBLIC" : "DEMO"}-${screen.id}-EV-${i + 1}`,
              scope,
              ...(i ? { state: i === 1 ? "pending" : "blocked" } : {}),
              ...(customer
                ? {
                    owner: "DEMO Self",
                    external: "notapplicable",
                    locator: "notapplicable",
                  }
                : {}),
            },
          })),
    [ready, model, screen.id, scope, customer],
  );
  const filtered = rows.filter(
    (r) =>
      organization === "DEMO-ORG-01" &&
      (!objectId || r.id === objectId) &&
      (!r.values.period || r.values.period === period) &&
      (!Object.values(r.values).some((v) => /^(USD|EUR|KRW) /.test(v)) ||
        Object.values(r.values).some((v) => v.startsWith(currency + " "))) &&
      (!tree || r.id.endsWith("-" + tree)) &&
      JSON.stringify(r).toLowerCase().includes(query.toLowerCase()),
  );
  const inboxCurrent = filtered.find((r) => r.id === inboxId) ?? filtered[0];
  const columns = model?.views[tab] ?? model?.columns ?? [];
  useEffect(() => {
    setSelected(null);
  }, [query, tab, tree, period, currency, organization, objectId]);
  useEffect(() => {
    onState(
      JSON.stringify({
        phase: tab,
        scope,
        period,
        currency,
        organization,
        objectId,
        filter: { query, tree },
        scenario,
        detail:
          !!selected || (screen.archetype.includes("inbox") && !!inboxCurrent),
        selected: selected
          ? [selected.id]
          : screen.archetype.includes("inbox") && inboxCurrent
            ? [inboxCurrent.id]
            : [],
      }),
      [
        `${screen.id}:region:${tab}`,
        `${screen.id}:field:search`,
        `${screen.id}:state:service`,
        ...columns.map((k) => `${screen.id}:data:${k}`),
        ...(selected ? [`${screen.id}:action:inspect`] : []),
        ...(draft ? [`${screen.id}:form:draft`] : []),
      ],
    );
  }, [
    tab,
    scope,
    period,
    currency,
    organization,
    objectId,
    query,
    tree,
    scenario,
    selected,
    inboxId,
    draft,
    screen.id,
    onState,
  ]);
  if (!ready)
    return (
      <section className="planning-panel planning-state" role="status">
        <h2>
          {local(
            l,
            mode === "unconnected"
              ? ["서비스 미연결", "Service not connected", "الخدمة غير متصلة"]
              : scenario === "empty"
                ? ["조회 대상 없음", "No results", "لا توجد نتائج"]
                : scenario === "loading"
                  ? ["조회 중", "Loading", "جار التحميل"]
                  : scenario === "revoked"
                    ? ["접근 철회", "Access revoked", "تم إلغاء الوصول"]
                    : scenario === "partial"
                      ? [
                          "일부 데이터 확인 대기",
                          "Partial data awaiting verification",
                          "بيانات جزئية بانتظار التحقق",
                        ]
                      : ["조회 실패", "Unable to load", "تعذر التحميل"],
          )}
        </h2>
        <p>
          {local(l, [
            "현재 상태에서 대상 정보를 표시하지 않습니다. 실제 서비스 상태는 미연결입니다.",
            "No target data is shown in this state. Actual service status is not connected.",
            "لا تعرض بيانات الأهداف في هذه الحالة. حالة الخدمة الفعلية غير متصلة.",
          ])}
        </p>
      </section>
    );
  if (objectId && !filtered.length)
    return (
      <section className="planning-panel" role="status">
        <h2>
          {local(l, [
            "대상을 표시할 수 없습니다",
            "Object unavailable",
            "العنصر غير متاح",
          ])}
        </h2>
        <p>
          {local(l, [
            "현재 범위의 목록으로 돌아가세요.",
            "Return to the list in the current scope.",
            "عد إلى القائمة ضمن النطاق الحالي.",
          ])}
        </p>
      </section>
    );
  if (screen.id === "A01") return <Conversation locale={l} onDirty={onDirty} />;
  if (!model) throw Error(`Missing candidate body ${screen.id}`);
  const current = filtered[0];
  const wizardFields: Record<string, Record<string, string[]>> = {
    O02: { targets: ["organization", "parent", "effective", "reason"] },
    S02: {
      path: ["path", "source", "contract"],
      permissions: ["owner", "secret"],
    },
    P01: {
      plan: ["environment", "service", "port", "volume"],
      support: ["purpose"],
    },
  };
  const editableStep = wizardFields[screen.id]?.[tab] ?? [];
  const documentTab =
    ["I01", "C02"].includes(screen.id) && ["body", "detail"].includes(tab);
  return (
    <div
      className={screen.id === "O01" ? "planning-tree-layout" : "planning-body"}
    >
      {screen.id === "O01" && (
        <OrganizationTree locale={l} onSelect={setTree} />
      )}
      <div className="planning-body">
        {current && screen.id === "C01" && tab === "summary" && (
          <section className="planning-facts">
            {["contract", "amount", "period", "due", "delivery"].map((k) => (
              <div key={k}>
                <span>{field(l, k)}</span>
                <strong>{valueText(l, current.values[k] ?? "unknown")}</strong>
              </div>
            ))}
          </section>
        )}
        {screen.id === "P06" && tab === "scenarios" && filtered.length > 0 && (
          <section className="planning-scenario-cards">
            {filtered.map((row) => (
              <button key={row.id} onClick={() => setSelected(row)}>
                <small>
                  <bdi>{row.id}</bdi> · DEMO
                </small>
                <strong>{row.values.scenario}</strong>
                <span>{valueText(l, row.values.purpose)}</span>
                <span>
                  {field(l, "bundle")}: {valueText(l, row.values.bundle)}
                </span>
              </button>
            ))}
          </section>
        )}

        {editableStep.length > 0 && (
          <DraftForm
            key={tab}
            locale={l}
            fields={editableStep}
            onDirty={onDirty}
          />
        )}
        {screen.id === "M03" &&
          current &&
          ["roles", "effects"].includes(tab) && <PermissionMatrix locale={l} />}
        {screen.id === "R01" && tab === "summary" && filtered.length > 0 && (
          <ReportChart locale={l} rows={filtered} />
        )}
        {["P03", "P04"].includes(screen.id) && (
          <LifecyclePanel locale={l} kind={screen.id} />
        )}
        {screen.id === "I04" && tab === "preview" && current && (
          <DisclosurePreview locale={l} row={current} />
        )}
        {screen.id === "S05" && tab === "documents" && current && (
          <section className="planning-panel">
            <h2>
              {local(l, [
                "안전한 문서 미리보기",
                "Safe document preview",
                "معاينة آمنة للمستند",
              ])}
            </h2>
            <p>
              <bdi>
                {current.values.document} · {current.values.hash}
              </bdi>
            </p>
            <p>
              {local(l, [
                "실행 가능한 원본을 불러오지 않는 텍스트 예제입니다. 원본 다운로드 권한은 별도입니다.",
                "Text-only example without active original content. Original download requires separate permission.",
                "مثال نصي دون محتوى أصلي نشط. تنزيل الأصل يتطلب صلاحية مستقلة.",
              ])}
            </p>
          </section>
        )}
        {documentTab && current ? (
          <DocumentPreview locale={l} row={current} />
        ) : (
          <section className="planning-panel">
            <header className="planning-section-header">
              <div>
                <h2>
                  {tabLabel(
                    l,
                    tab,
                    screen.tabs.find((t) => t.key === tab)?.labelKo ??
                      screen.titleKo,
                  )}
                </h2>
                <p>
                  {local(l, [
                    "합성 예제 · 조회 기준 2026-11-03 · 금액은 표시된 의미·통화별로 구분",
                    "Synthetic examples · as of 2026-11-03 · amounts retain their stated meaning and currency",
                    "أمثلة تجريبية · حتى 2026-11-03 · المبالغ بحسب المعنى والعملة",
                  ])}
                </p>
              </div>
              {model.draftFields.length > 0 &&
                !screen.archetype.includes("wizard") && (
                  <button
                    onClick={() => {
                      if (!draft || canDiscard()) {
                        setDraft(!draft);
                        onDirty(false);
                      }
                    }}
                    aria-expanded={draft}
                  >
                    {local(l, [
                      "검토용 초안",
                      "Review draft",
                      "مسودة للمراجعة",
                    ])}
                  </button>
                )}
            </header>
            <div className="planning-toolbar">
              <label>
                {local(l, ["목록 검색", "Search records", "البحث في السجلات"])}
                <input
                  type="search"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
              </label>
              <span aria-live="polite">
                {filtered.length} / {rows.length}{" "}
                {local(l, ["예제 대상", "example targets", "أهداف تجريبية"])}
              </span>
            </div>
            {filtered.length ? (
              screen.archetype.includes("inbox") ? (
                <Inbox
                  rows={filtered}
                  current={inboxCurrent}
                  keys={[...columns, ...model.details]}
                  screen={screen.id}
                  locale={l}
                  customer={customer}
                  onChoose={setInboxId}
                />
              ) : (
                <CandidateTable
                  columns={columns}
                  rows={filtered}
                  locale={l}
                  onOpen={setSelected}
                />
              )
            ) : (
              <p role="status">
                {local(l, [
                  "조건에 맞는 대상이 없습니다.",
                  "No matching records.",
                  "لا توجد سجلات مطابقة.",
                ])}
              </p>
            )}
          </section>
        )}
        {draft && !screen.archetype.includes("wizard") && (
          <DraftForm
            key={tab}
            locale={l}
            fields={
              screen.archetype.includes("wizard")
                ? columns.filter(
                    (k) => !["id", "state", "approver", "digest"].includes(k),
                  )
                : model.draftFields
            }
            onDirty={onDirty}
          />
        )}
        {!customer && current && ["B08", "U03", "I03"].includes(screen.id) && (
          <section className="planning-facts">
            {["targets", "succeeded", "held", "failed", "remaining"].map(
              (k) => (
                <div key={k}>
                  <span>{field(l, k)}</span>
                  <strong>
                    {valueText(l, current.values[k] ?? "unknown")}
                  </strong>
                </div>
              ),
            )}
          </section>
        )}
        {!screen.archetype.includes("inbox") && (
          <CommandBoundary screen={screen.id} locale={l} customer={customer} />
        )}
        <section className="planning-context">
          <h3>
            {local(l, [
              "연결·실행 범위",
              "Connection and execution boundary",
              "حدود الاتصال والتنفيذ",
            ])}
          </h3>
          <p>
            {local(l, [
              "이 화면은 합성 데이터로 구성한 검토 시안입니다. 승인·발행·회수·삭제·전달 등 실제 명령은 정책·권한·서비스 연동 전까지 실행하지 않습니다.",
              "This is a review candidate using synthetic data. Approval, issuance, revocation, deletion and delivery commands remain unavailable until policies, permissions and services are integrated.",
              "هذه معاينة للمراجعة ببيانات تجريبية. الموافقة والإصدار والإلغاء والحذف والتسليم غير متاحة حتى تكامل السياسات والصلاحيات والخدمات.",
            ])}
          </p>
        </section>
      </div>
      {selected && (
        <CandidateDetail
          locale={l}
          row={selected}
          keys={[...columns, ...model.details]}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  );
}
