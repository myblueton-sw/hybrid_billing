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
import AmountSummary from "./AmountSummary";
import WorkSummary from "./WorkSummary";
import "./overview.css";
import { emptyClosingFilter } from "./closing-filter";
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
  const period =
    params.get("period") ?? params.get("overview-period") ?? months.at(-1)!;
  const currency: Currency =
    (params.get("ccy") ?? params.get("ov-currency")) === "KRW" ? "KRW" : "USD";
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
  const [compact, setCompact] = useState(false);
  useEffect(() => {
    const query = window.matchMedia("(max-width: 767px)");
    const update = () => setCompact(query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  const [listFilter, setListFilter] = useState(emptyClosingFilter);
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
    setListFilter(emptyClosingFilter);
  }, [key]);
  useEffect(() => {
    onState(
      JSON.stringify({
        listFilter,
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
        "U01:summary:amount-meanings",
        "U01:work:attention",
        "U01:state:receivables",
        "U01:scope:provenance",
        "U01:scope:example-data-set",
        "overview:field:reporting-month",
        "overview:field:source-selection",
        "overview:field:organization-selection",
        "overview:field:organization-search",
        "overview:field:fixture-profile",
        "overview:action:page",
        "overview:field:closing-search",
        "overview:field:closing-stage-filter",
        "overview:field:closing-status-filter",
        "overview:action:closing-list-reset",
        "overview:action:page-jump",
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
  }, [key, active?.id, active?.stage, listFilter, onState]);
  function amount(value: ReturnType<typeof total>, c: Currency) {
    return (
      <>
        <bdi dir="ltr">
          {value.value === null
            ? `${c} · ${money(value.value, c)}`
            : money(value.value, c)}
        </bdi>
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
        <div
          className="table-wrap"
          role="region"
          tabIndex={0}
          aria-label={t(
            l,
            "Source amounts, horizontally scrollable",
            "원천별 금액, 가로 스크롤",
            "مبالغ المصادر، تمرير أفقي",
          )}
        >
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
        selectedSources={selectedSources}
        selectedOrganizations={selectedOrganizations}
        change={change}
      />
      <div className="overview-basis" data-design-id="U01:scope:provenance">
        <p>
          {t(l, "Synthetic data", "합성 데이터", "بيانات تجريبية")}{" "}
          <bdi dir="ltr">{mode === "synthetic" ? snapshot : "—"}</bdi> ·{" "}
          {t(
            l,
            "No live connection · tax excluded · no FX",
            "실제 연동 없음 · 세금 제외 · 환산 없음",
            "دون اتصال مباشر · باستثناء الضريبة · دون تحويل",
          )}
        </p>
        <details>
          <summary>
            {t(
              l,
              "About these figures",
              "금액·상태 기준",
              "أساس المبالغ والحالات",
            )}
          </summary>
          <p>
            {t(
              l,
              "Source cost is not customer sales or organizational allocation. Partial amounts include known records only and are not final settlements. Work categories overlap; pending status grants no approval permission. Stages are independent and internal non-applicable states remain distinct. Owners, deadlines and authoritative as-of times are unavailable. Sales, allocation and receipt ledgers are not connected; no balances or overdue amounts are estimated.",
              "원천 비용은 고객 판매·조직 배분과 구분합니다. 부분 합계는 확인된 자료만 포함하며 최종 정산액이 아닙니다. 업무 항목은 중복될 수 있고 대기 상태가 승인 권한을 부여하지 않습니다. 각 처리 단계는 독립적이며 내부 정산의 해당 없음과 미확인은 구분합니다. 담당자·납기·실제 확인 시각은 미연결입니다. 판매·배분·수납 원장이 없어 미수·연체 금액을 추정하지 않습니다.",
              "تكلفة المصادر منفصلة عن المبيعات والتخصيص. تشمل المبالغ الجزئية السجلات المتاحة فقط وليست تسويات نهائية. قد تتداخل فئات العمل والانتظار لا يمنح صلاحية الموافقة. المراحل مستقلة والحالات غير المنطبقة منفصلة عن غير المؤكدة. المسؤولون والمواعيد ووقت التحقق الفعلي غير متاحين. دفاتر المبيعات والتخصيص والتحصيل غير متصلة ولا تقدر أرصدة أو متأخرات.",
            )}
          </p>
        </details>
      </div>
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
          {!compact && <AmountSummary rows={current} locale={l} />}
          <div className="overview-work-context">
            <WorkSummary
              rows={closing}
              missing={missing}
              scope={key}
              locale={l}
              onInspect={inspect}
            />
            {compact && <AmountSummary rows={current} locale={l} />}
            <CostTrend
              rows={all}
              currency={currency}
              locale={l}
              period={period}
              onMonth={(month) => change({ period: month })}
              onCurrency={(ccy) => change({ ccy })}
            />
          </div>
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
                          <bdi dir="ltr">
                            {r.minor === null
                              ? `${r.currency} · ${money(r.minor, r.currency)}`
                              : money(r.minor, r.currency)}
                          </bdi>
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
          <Lifecycle
            key={key}
            onFilterChange={(filter) => {
              setDetail(null);
              setListFilter(filter);
            }}
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
          <div className="overview-compositions">
            {breakdown("source", selectedSources)}
            <OrgBreakdown
              rows={current}
              enterprise={enterprise}
              scope={key}
              locale={l}
              onSelect={(id) => inspect("organization", id)}
            />
          </div>
        </>
      )}
      <details className="overview-example-config">
        <summary>
          {t(
            l,
            "Example configuration",
            "예제 데이터 설정",
            "إعدادات البيانات التجريبية",
          )}
        </summary>
        <div className="overview-profile">
          <label>
            {t(
              l,
              "Example data set",
              "예제 데이터 세트",
              "مجموعة البيانات التجريبية",
            )}
            <select
              id="overview-workspace"
              value={workspace}
              onChange={(e) => change({ scope: e.target.value })}
            >
              <option value="sample-a">
                {t(l, "Example A", "예제 A", "مثال A")}
              </option>
              <option value="sample-b">
                {t(l, "Example B", "예제 B", "مثال B")}
              </option>
            </select>
          </label>
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
      </details>
    </div>
  );
}
