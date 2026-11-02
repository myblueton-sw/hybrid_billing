import { useEffect, useRef, useState, useMemo } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { routeUrl, useLocation } from "../../../../shared/navigation";
import {
  sources,
  organizations,
  months,
  snapshot,
  fixtureCosts,
  selectCosts,
  total,
  formatMoney,
  fixtureClosings,
  selectClosings,
  type Currency,
  type Stage,
  type Closing,
} from "./model";
import { organizationName, stageName, statusName } from "./copy";
import ScopeFilters from "./ScopeFilters";
import CostTrend from "./CostTrend";
import Lifecycle from "./Lifecycle";
import "./overview.css";
import { enterpriseLeaves, resolveOrganizations } from "./organization-scope";
import OrgBreakdown from "./OrgBreakdown";
import Paging, { usePage } from "./Paging";
type Detail = {
  key: string;
  kind: "source" | "organization" | "record" | "stage";
  id: string;
  stage?: Stage;
};
export default function Overview({ locale: l, mode, onState }: ScreenProps) {
  const hash = useLocation(),
    params = new URLSearchParams(hash.split("?")[1]);
  const workspace =
    params.get("scope") === "sample-b" ? "sample-b" : "sample-a";
  const period = params.get("overview-period") ?? months.at(-1)!;
  const currency: Currency =
    params.get("ov-currency") === "KRW" ? "KRW" : "USD";
  const selectedSources = params.has("ov-sources")
    ? sources.filter((s) => params.get("ov-sources")!.split("|").includes(s))
    : [...sources];
  const enterprise = params.get("ov-scale") === "enterprise";
  const selectedOrganizations = resolveOrganizations(
    enterprise,
    params.get("ov-organizations"),
  );
  const key = JSON.stringify([
    workspace,
    enterprise,
    period,
    currency,
    selectedSources,
    selectedOrganizations,
    mode,
  ]);
  const [detail, setDetail] = useState<Detail | null>(null);
  const detailRef = useRef<HTMLElement>(null);
  const trigger = useRef<HTMLElement | null>(null);
  const active = detail?.key === key ? detail : null;
  const costs = useMemo(
    () =>
      mode === "synthetic"
        ? fixtureCosts(workspace, enterprise ? enterpriseLeaves : organizations)
        : [],
    [workspace, enterprise, mode],
  );
  const all = selectCosts(costs, selectedSources, selectedOrganizations);
  const current = all.filter((r) => r.month === period);
  const closing =
    mode === "synthetic"
      ? selectClosings(
          fixtureClosings(
            workspace,
            period,
            enterprise ? enterpriseLeaves : organizations,
          ),
          selectedSources,
          selectedOrganizations,
        )
      : [];
  const known = current.filter((r) => r.minor !== null).length;
  const missing = current.filter((r) => r.minor === null);
  const activeClosing =
    active?.kind === "stage"
      ? closing.find((r) => r.id === active.id)
      : undefined;
  const detailRows = active
    ? current.filter((r) =>
        active.kind === "source"
          ? r.source === active.id
          : active.kind === "organization"
            ? r.organization === active.id ||
              (enterprise && r.organization.startsWith(active.id + "."))
            : active.kind === "record"
              ? r.id === active.id
              : r.organization === activeClosing?.organization &&
                activeClosing.sources.includes(r.source),
      )
    : [];
  const detailPage = usePage(
    detailRows,
    key + "|" + active?.kind + "|" + active?.id + "|" + active?.stage,
  );
  const exceptions = [
    ...missing.map((r) => ({
      id: r.id,
      kind: "record" as const,
      label: `${r.source} · ${organizationName(l, r.organization)} · ${t(l, "Cost record missing", "비용 자료 누락", "سجل تكلفة مفقود")}`,
    })),
    ...closing
      .filter((r) => r.status.accounting === "unknown")
      .map((r) => ({
        id: r.id,
        kind: "stage" as const,
        label: `${organizationName(l, r.organization)} · ${t(l, "ERP confirmation missing", "ERP 반영 미확인", "تأكيد ERP مفقود")}`,
      })),
  ];
  const exceptionPage = usePage(exceptions, key);
  const money = (value: bigint | null, c: Currency) => formatMoney(value, c, l);
  function change(values: Record<string, string>) {
    setDetail(null);
    location.hash = routeUrl("overview", values);
  }
  function inspect(kind: Detail["kind"], id: string, stage?: Stage) {
    trigger.current = document.activeElement as HTMLElement;
    setDetail({ key, kind, id, stage });
  }
  function closeDetail() {
    setDetail(null);
    if (trigger.current?.isConnected) trigger.current.focus();
  }
  useEffect(() => {
    if (active) detailRef.current?.focus();
  }, [detail]);
  useEffect(() => {
    setDetail(null);
  }, [key]);
  useEffect(() => {
    onState(
      JSON.stringify({
        enterprise,
        workspace,
        scope: workspace,
        period,
        currency,
        selectedSources,
        selectedOrganizations,
        mode,
        detail: active
          ? { kind: active.kind, id: active.id, stage: active.stage }
          : null,
      }),
      [
        "overview:region:layout",
        "overview:field:reporting-month",
        "overview:field:source-selection",
        "overview:field:organization-selection",
        "overview:field:organization-search",
        "overview:field:fixture-profile",
        "overview:action:page",
        "overview:field:chart-currency",
        ...(mode === "synthetic"
          ? [
              "overview:data:monthly-cost",
              "overview:data:source-breakdown",
              "overview:data:organization-breakdown",
              "overview:data:closing-matrix",
              "overview:action:inspect",
              ...(active ? ["overview:section:detail"] : []),
            ]
          : ["overview:state:unconnected"]),
      ],
    );
  }, [key, active?.id, active?.stage, onState]);
  function amount(value: ReturnType<typeof total>, c: Currency) {
    return (
      <>
        <bdi>{money(value.value, c)}</bdi>
        {value.missing > 0 && (
          <small>
            {t(l, "Partial · missing", "부분 · 누락", "جزئي · مفقود")}{" "}
            {value.missing}
          </small>
        )}
        {value.expected === 0 && (
          <small>
            {t(l, "No matching data", "해당 자료 없음", "لا بيانات مطابقة")}
          </small>
        )}
      </>
    );
  }
  function breakdown(kind: "source" | "organization", ids: readonly string[]) {
    return (
      <section className="unified-panel">
        <header>
          <h2>
            {kind === "source"
              ? t(l, "Source composition", "원천별 구성", "توزيع المصادر")
              : t(
                  l,
                  "Organization composition",
                  "조직별 구성",
                  "توزيع المؤسسات",
                )}
          </h2>
        </header>
        <div className="table-wrap">
          <table className="unified-breakdown">
            <thead>
              <tr>
                <th>
                  {kind === "source"
                    ? t(l, "Source", "원천", "المصدر")
                    : t(l, "Organization", "조직", "المؤسسة")}
                </th>
                <th>USD</th>
                <th>KRW</th>
              </tr>
            </thead>
            <tbody>
              {ids.map((id) => {
                const rows = current.filter((r) =>
                  kind === "source" ? r.source === id : r.organization === id,
                );
                return (
                  <tr key={id}>
                    <th>
                      <button
                        onClick={() => inspect(kind, id)}
                        aria-pressed={active?.kind === kind && active.id === id}
                      >
                        {kind === "source" ? id : organizationName(l, id)}
                      </button>
                    </th>
                    {(["USD", "KRW"] as const).map((c) => (
                      <td key={c}>{amount(total(rows, c), c)}</td>
                    ))}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>
    );
  }
  return (
    <div
      className="overview-dashboard unified-overview"
      data-design-id="overview:region:layout"
    >
      <ScopeFilters
        locale={l}
        enterprise={enterprise}
        contextKey={JSON.stringify([
          workspace,
          enterprise,
          period,
          selectedSources,
          mode,
        ])}
        period={period}
        currency={currency}
        selectedSources={selectedSources}
        selectedOrganizations={selectedOrganizations}
        change={change}
      />
      <div className="overview-profile">
        <label>
          {t(l, "Example size", "예제 규모", "حجم المثال")}
          <select
            id="overview-profile"
            value={enterprise ? "enterprise" : "small"}
            onChange={(e) =>
              change({
                "ov-scale": e.target.value,
                "ov-organizations": "all",
                "ov-sources": sources.join("|"),
              })
            }
          >
            <option value="small">
              {t(
                l,
                "Small · 3 organizations",
                "소규모 · 조직 3개",
                "صغير · 3 مؤسسات",
              )}
            </option>
            <option value="enterprise">
              {t(
                l,
                "Enterprise · 2,400 departments",
                "대규모 · 부서 2,400개",
                "كبير · 2400 قسم",
              )}
            </option>
          </select>
        </label>
        <p data-scope-count>
          {enterprise &&
            t(
              l,
              "Selected companies / divisions / departments",
              "선택 법인 / 사업부 / 부서",
              "الشركات / القطاعات / الأقسام المختارة",
            )}{" "}
          <bdi dir="ltr">
            {enterprise
              ? `${new Set(selectedOrganizations.map((id) => id.split(".")[0])).size} / ${new Set(selectedOrganizations.map((id) => id.split(".").slice(0, 2).join("."))).size} / ${selectedOrganizations.length}`
              : `${selectedOrganizations.length} / 3`}
          </bdi>
        </p>
      </div>
      <p className="unified-basis">
        {t(
          l,
          "Synthetic source cost · tax excluded · reporting month · no currency conversion",
          "합성 원천 비용 · 세금 제외 · 보고월 기준 · 통화 환산 없음",
          "تكلفة مصادر تجريبية · باستثناء الضريبة · شهر التقرير · دون تحويل عملة",
        )}
        <br />
        {t(l, "Fixture snapshot", "예제 스냅샷", "لقطة تجريبية")}:{" "}
        <bdi>{mode === "synthetic" ? snapshot : "—"}</bdi> ·{" "}
        {t(l, "No live connection", "실제 연동 없음", "لا اتصال مباشر")}
      </p>
      {mode === "unconnected" ? (
        <section className="state" role="status">
          <h2>
            {t(
              l,
              "Integrated overview not connected",
              "통합 현황 서비스 미연결",
              "النظرة الموحدة غير متصلة",
            )}
          </h2>
          <p>
            {t(
              l,
              "Current costs and closing status cannot be verified.",
              "현재 비용과 정산 상태를 확인할 수 없습니다.",
              "لا يمكن التحقق من التكاليف وحالة الإقفال الحالية.",
            )}
          </p>
        </section>
      ) : !current.length ? (
        <section className="state" role="status">
          <h2>
            {t(
              l,
              "No data in the selected scope",
              "선택 범위에 자료가 없습니다",
              "لا بيانات في النطاق المحدد",
            )}
          </h2>
          <p>
            {t(
              l,
              "Choose one or more sources and organizations, and a supported reporting month. Empty selection never means all.",
              "원천과 조직을 하나 이상 선택하고 예제 보고월을 확인하세요. 선택 없음은 전체로 바뀌지 않습니다.",
              "اختر مصدراً ومؤسسة على الأقل وشهر تقرير متاحاً. الاختيار الفارغ لا يعني الكل.",
            )}
          </p>
        </section>
      ) : (
        <>
          <section
            className="unified-summary"
            aria-label={t(
              l,
              "Integrated monthly summary",
              "통합 월별 요약",
              "ملخص شهري موحد",
            )}
          >
            <div className="unified-total">
              <h2>
                {t(
                  l,
                  "Known monthly costs",
                  "선택 월 확인된 비용",
                  "التكاليف الشهرية المعروفة",
                )}
              </h2>
              <div className="unified-money">
                {(["USD", "KRW"] as const).map((c) => (
                  <div key={c} data-total={c}>
                    {amount(total(current, c), c)}
                  </div>
                ))}
              </div>
              <p>
                {t(
                  l,
                  "Currencies are separate totals. Partial data is not the final amount.",
                  "통화별 개별 합계입니다. 부분 자료는 최종 금액이 아닙니다.",
                  "إجماليات منفصلة حسب العملة؛ البيانات الجزئية ليست مبلغاً نهائياً.",
                )}
              </p>
            </div>
            <div>
              <h2>
                {t(
                  l,
                  "Monthly data coverage",
                  "월별 자료 확인",
                  "تغطية البيانات الشهرية",
                )}
              </h2>
              <strong data-coverage>
                <bdi dir="ltr">
                  {known} / {current.length}
                </bdi>
              </strong>
              <p>
                {t(
                  l,
                  "Source × organization records",
                  "원천 × 조직 자료",
                  "سجلات المصدر × المؤسسة",
                )}{" "}
                ·{" "}
                {missing.length
                  ? `${t(l, "Missing", "누락", "مفقود")} ${missing.length}`
                  : t(
                      l,
                      "Complete fixture coverage",
                      "예제 자료 모두 확인",
                      "تغطية تجريبية كاملة",
                    )}
              </p>
            </div>
            <div>
              <h2>
                {t(
                  l,
                  "Closing units in scope",
                  "범위 내 정산 단위",
                  "وحدات الإقفال في النطاق",
                )}
              </h2>
              <strong data-closing-count>{closing.length}</strong>
              <p>
                {t(
                  l,
                  "Unique contract paths · not a completed count",
                  "중복 없는 계약 경로 · 완료 건수 아님",
                  "مسارات عقود فريدة · ليس عدد المكتمل",
                )}
              </p>
            </div>
          </section>
          <div className="unified-analysis">
            <CostTrend
              rows={all}
              currency={currency}
              locale={l}
              period={period}
              onMonth={(month) => change({ "overview-period": month })}
            />
            <div className="unified-composition">
              {breakdown("source", selectedSources)}
            </div>
          </div>
          <OrgBreakdown
            rows={current}
            enterprise={enterprise}
            scope={key}
            locale={l}
            onSelect={(id) => inspect("organization", id)}
          />
          <Lifecycle
            scope={key}
            rows={closing}
            locale={l}
            selected={
              active?.kind === "stage" ? active.id + ":" + active.stage : ""
            }
            onSelect={(row: Closing, stage: Stage) =>
              inspect("stage", row.id, stage)
            }
          />
          {active && (
            <section
              ref={detailRef}
              tabIndex={-1}
              className="unified-panel unified-detail"
              aria-label={t(
                l,
                "Selected detail",
                "선택 상세",
                "التفاصيل المحددة",
              )}
            >
              <header>
                <h2>
                  {t(l, "Selected detail", "선택 상세", "التفاصيل المحددة")} ·{" "}
                  {active.kind === "source"
                    ? active.id
                    : active.kind === "organization"
                      ? organizationName(l, active.id)
                      : active.kind === "record"
                        ? active.id
                        : `${organizationName(l, activeClosing!.organization)} · ${stageName(l, active.stage!)}`}
                </h2>
                <button onClick={closeDetail}>
                  {t(l, "Close detail", "상세 닫기", "إغلاق التفاصيل")}
                </button>
              </header>
              <p>
                <bdi>
                  {period} · {snapshot}
                </bdi>{" "}
                ·{" "}
                {t(
                  l,
                  "Reporting-month fixture · tax excluded",
                  "보고월 예제 · 세금 제외",
                  "بيانات شهر التقرير · باستثناء الضريبة",
                )}
              </p>
              {activeClosing && (
                <>
                  <p>
                    <bdi>{activeClosing.id}</bdi>
                  </p>
                  <p>
                    {stageName(l, active.stage!)}:{" "}
                    <strong>
                      {statusName(l, activeClosing.status[active.stage!])}
                    </strong>{" "}
                    ·{" "}
                    {t(
                      l,
                      "Whole-unit sources",
                      "정산 건 전체 원천",
                      "مصادر الوحدة كاملة",
                    )}
                    : <bdi>{activeClosing.sources.join(" · ")}</bdi>
                  </p>
                  <p>
                    {t(
                      l,
                      "Reporting and settlement months are explicitly equal in this fixture only. Real period mapping and evidence services are not connected.",
                      "이 예제만 보고월과 정산월을 동일하게 지정했습니다. 실제 기간 매핑·근거 서비스는 미연결입니다.",
                      "في هذا المثال فقط يتطابق شهر التقرير والتسوية صراحة. خدمات ربط الفترات والأدلة غير متصلة.",
                    )}
                  </p>
                </>
              )}
              <p>
                {t(
                  l,
                  "Amounts below retain the selected source and organization filters.",
                  "아래 금액은 선택한 원천·조직 범위를 유지합니다.",
                  "المبالغ أدناه تحافظ على تصفية المصادر والمؤسسات المختارة.",
                )}
              </p>
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>{t(l, "Organization", "조직", "المؤسسة")}</th>
                      <th>{t(l, "Source", "원천", "المصدر")}</th>
                      <th>
                        {t(l, "Known cost", "확인된 비용", "التكلفة المعروفة")}
                      </th>
                      <th>
                        {t(l, "Fixture reference", "예제 참조", "مرجع تجريبي")}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {detailPage.rows.map((r) => (
                      <tr key={r.id}>
                        <td>{organizationName(l, r.organization)}</td>
                        <td>{r.source}</td>
                        <td>
                          <bdi>{money(r.minor, r.currency)}</bdi>
                          {r.minor === null &&
                            t(l, " · Missing", " · 누락", " · مفقود")}
                        </td>
                        <td>
                          <bdi>{r.id}</bdi>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Paging value={detailPage} locale={l} />
            </section>
          )}
          <section className="unified-panel unified-exceptions">
            <header>
              <h2>
                {t(
                  l,
                  "Items needing attention",
                  "확인이 필요한 항목",
                  "عناصر تتطلب الانتباه",
                )}
              </h2>
            </header>
            <ul>
              {exceptionPage.rows.map((r) => (
                <li key={r.id}>
                  <button
                    onClick={() =>
                      inspect(
                        r.kind,
                        r.id,
                        r.kind === "stage" ? "accounting" : undefined,
                      )
                    }
                  >
                    {r.label}
                  </button>
                </li>
              ))}
              {!missing.length &&
                !closing.some((r) => r.status.accounting === "unknown") && (
                  <li>
                    {t(
                      l,
                      "No missing cost or unknown ERP outcomes in this fixture selection.",
                      "선택한 예제에 비용 누락·ERP 미확인 항목이 없습니다.",
                      "لا توجد تكلفة مفقودة أو نتائج ERP غير مؤكدة في الاختيار التجريبي.",
                    )}
                  </li>
                )}
            </ul>
            <Paging value={exceptionPage} locale={l} />
          </section>
          <p className="unified-note">
            {t(
              l,
              "This view demonstrates integrated source costs and independent closing states. Sales, allocations, forecasts and receipt balances require separate data contracts and are not calculated here.",
              "이 화면은 통합 원천 비용과 독립 정산 상태를 보여주는 예제입니다. 판매·배분 금액, 예측, 미수 잔액은 별도 데이터 계약이 필요하며 여기서 계산하지 않습니다.",
              "هذا مثال للتكاليف الموحدة وحالات الإقفال المستقلة. تتطلب المبيعات والتخصيصات والتوقعات وأرصدة المستحقات عقود بيانات منفصلة ولا تحسب هنا.",
            )}
          </p>
        </>
      )}
    </div>
  );
}
