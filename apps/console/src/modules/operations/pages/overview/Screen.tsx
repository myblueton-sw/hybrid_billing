import { useEffect, useRef, useState } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { useLocation } from "../../../../shared/navigation";
import {
  loadOverview,
  ServiceError,
  type Basis,
  type OverviewData,
} from "./focus-api";
import { decimal } from "./focus-values";
import FocusChart from "./FocusChart";
import "./overview.css";
import "./focus.css";
const keys = [
  "period",
  "provider",
  "account",
  "basis",
  "q",
  "after",
  "account_after",
  "snapshot",
];
export default function Overview({
  locale: l,
  onState,
  view = "settlement",
}: ScreenProps & { view?: string }) {
  const detailHeading = useRef<HTMLHeadingElement>(null);
  const hash = useLocation(),
    params = new URLSearchParams(hash.split("?")[1]),
    query = new URLSearchParams();
  for (const key of keys)
    if (params.has(`focus-${key}`)) query.set(key, params.get(`focus-${key}`)!);
  if (query.has("q")) query.set("q", query.get("q")!.trim());
  const key = query.toString(),
    generation = useRef(0);
  const [state, setState] = useState<{
    key: string;
    data?: OverviewData;
    error?: string;
  }>({ key });
  const [retry, setRetry] = useState(0),
    [search, setSearch] = useState(query.get("q") ?? "");
  const data = state.key === key ? state.data : undefined,
    error = state.key === key ? state.error : undefined;
  useEffect(() => {
    const controller = new AbortController(),
      current = ++generation.current;
    setState({ key });
    loadOverview(new URLSearchParams(key), controller.signal)
      .then((data) => {
        if (!controller.signal.aborted && current === generation.current)
          setState({ key, data });
      })
      .catch((e) => {
        if (!controller.signal.aborted && current === generation.current)
          setState({
            key,
            error: e instanceof ServiceError ? e.code : "service_unavailable",
          });
      });
    return () => controller.abort();
  }, [key, retry]);
  useEffect(() => setSearch(query.get("q") ?? ""), [key]);
  const chartCurrency = data?.options.currencies.includes(
    params.get("focus-currency") ?? "",
  )
    ? params.get("focus-currency")!
    : data?.options.currencies.includes("USD")
      ? "USD"
      : (data?.options.currencies[0] ?? "USD");
  useEffect(() => {
    onState(
      JSON.stringify({
        selection: data?.selection ?? null,
        after: data?.after ?? null,
        account_after: data?.account_after ?? null,
        chartCurrency,
        snapshot: data?.snapshot ?? null,
        company: data?.company ?? null,
        row_count: data?.row_count ?? null,
        source: "collected-focus",
        status: error ?? (data ? "loaded" : "loading"),
        view,
      }),
      [
        "overview:region:layout",
        "U01:scope:collected-focus",
        "U01:scope:provenance",
        "U01:summary:amount-meanings",
        "overview:field:reporting-month",
        "overview:field:source-selection",
        "overview:field:organization-selection",
        "overview:field:organization-search",
        "overview:field:cost-basis",
        "overview:action:page",
        "overview:field:chart-currency",
        "overview:state:unconnected",
        ...(data
          ? [
              "overview:data:monthly-cost",
              "overview:data:source-breakdown",
              "overview:data:organization-breakdown",
              "overview:section:detail",
            ]
          : []),
      ],
    );
  }, [data, error, view, chartCurrency, onState]);
  function change(values: Record<string, string>, continuation = false) {
    const next = new URLSearchParams(location.hash.split("?")[1]);
    if (!continuation)
      for (const name of ["after", "account_after", "snapshot"])
        next.delete(`focus-${name}`);
    for (const [name, value] of Object.entries(values))
      value ? next.set(`focus-${name}`, value) : next.delete(`focus-${name}`);
    location.hash = `${location.hash.split("?")[0]}?${next}`;
  }
  function page(name: "after" | "account_after", value: number) {
    if (data) change({ [name]: String(value), snapshot: data.snapshot }, true);
  }
  function refresh() {
    if (
      query.has("snapshot") ||
      query.has("after") ||
      query.has("account_after")
    )
      change({});
    else {
      setState({ key });
      setRetry((n) => n + 1);
    }
  }
  useEffect(() => {
    if (data?.selection.account) detailHeading.current?.focus();
  }, [data?.snapshot]);
  const basis: Basis =
    data?.selection.basis ??
    (query.get("basis") === "effective_cost"
      ? "effective_cost"
      : "billed_cost");
  const label =
    basis === "billed_cost"
      ? t(l, "Billed cost", "청구 비용", "التكلفة المفوترة")
      : t(l, "Effective cost", "실효 비용", "التكلفة الفعلية");
  const amount = (coefficient: string | null, currency: string) => (
    <bdi dir="ltr">
      <span>{currency}</span>{" "}
      <span className="focus-exact">{decimal(coefficient, l)}</span>
    </bdi>
  );
  const table = (label: string, children: React.ReactNode) => (
    <div
      className="table-wrap focus-table"
      role="region"
      aria-label={label}
      tabIndex={0}
    >
      {children}
    </div>
  );
  const pager = (
    name: "after" | "account_after",
    total: number,
    offset: number,
    next: number | null,
  ) => (
    <nav
      className="overview-paging"
      aria-label={
        name === "after"
          ? t(l, "Source rows pages", "원천 행 페이지", "صفحات الصفوف")
          : t(l, "Account pages", "계정 페이지", "صفحات الحسابات")
      }
    >
      <span>
        <bdi>
          {total ? offset + 1 : 0}–{Math.min(offset + 25, total)} / {total}
        </bdi>
      </span>
      <button
        disabled={offset === 0}
        onClick={() => page(name, Math.max(0, offset - 25))}
      >
        {t(l, "Previous", "이전", "السابق")}
      </button>
      <button
        disabled={next === null}
        onClick={() => next !== null && page(name, next)}
      >
        {t(l, "Next", "다음", "التالي")}
      </button>
    </nav>
  );
  const unconnected = (
    <section className="unified-panel">
      <h2>
        {t(
          l,
          "Operational ledgers not connected",
          "업무 원장 미연결",
          "السجلات التشغيلية غير متصلة",
        )}
      </h2>
      <p>
        {t(
          l,
          "Sales, allocation execution, closing, approval, ERP and receivables require their own operational records. Collected FOCUS costs do not establish completion or outstanding balances.",
          "판매·배분 실행·마감·승인·ERP·미수는 각 업무 원장 연결이 필요합니다. 수집된 FOCUS 비용으로 완료 상태나 미수 잔액을 생성하지 않습니다.",
          "تتطلب المبيعات والتخصيص والإقفال والاعتماد وERP والمستحقات سجلات مستقلة. لا تثبت تكاليف FOCUS اكتمال الأعمال أو الأرصدة المستحقة.",
        )}
      </p>
    </section>
  );
  return (
    <div
      className="overview-dashboard unified-overview focus-overview"
      data-focus-state={error ? "error" : data ? "loaded" : "loading"}
      data-focus-query={key}
    >
      <div className="unified-filters">
        <label>
          {t(l, "Charge period", "과금 기간", "فترة الرسوم")}
          <select
            id="focus-period"
            disabled={!data}
            value={query.get("period") ?? "all"}
            onChange={(e) => change({ period: e.target.value })}
          >
            <option value="all">
              {t(
                l,
                "All collected periods",
                "전체 수집 기간",
                "كل الفترات المجمعة",
              )}
            </option>
            {(data?.options.periods ?? []).map((p) => (
              <option key={p}>{p}</option>
            ))}
          </select>
        </label>
        <label>
          {t(l, "Provider", "공급사", "المزود")}
          <select
            id="focus-provider"
            disabled={!data}
            value={query.get("provider") ?? ""}
            onChange={(e) => change({ provider: e.target.value, account: "" })}
          >
            <option value="">
              {t(l, "All providers", "전체 공급사", "كل المزودين")}
            </option>
            {(data?.options.providers ?? []).map((p) => (
              <option key={p}>{p}</option>
            ))}
          </select>
        </label>
        <fieldset className="focus-basis">
          <legend>{t(l, "Cost basis", "비용 기준", "أساس التكلفة")}</legend>
          {(["billed_cost", "effective_cost"] as const).map((b) => (
            <label key={b}>
              <input
                type="radio"
                name="cost-basis"
                value={b}
                checked={basis === b}
                onChange={() => change({ basis: b })}
              />
              {b === "billed_cost" ? "BilledCost" : "EffectiveCost"}
            </label>
          ))}
        </fieldset>
        <button
          onClick={() => {
            const next = new URLSearchParams(params);
            for (const k of keys) next.delete(`focus-${k}`);
            location.hash = `${location.hash.split("?")[0]}?${next}`;
          }}
        >
          {t(l, "Reset scope", "전체 통합 보기", "إعادة النطاق")}
        </button>
        <button onClick={refresh} disabled={!data && !error}>
          {t(l, "Refresh query", "새로 조회", "تحديث الاستعلام")}
        </button>
      </div>
      {!data && !error && (
        <section role="status" className="unified-panel">
          <h2>
            {t(
              l,
              "Reading collected FOCUS data",
              "수집된 FOCUS 데이터 조회 중",
              "جارٍ قراءة بيانات FOCUS المجمعة",
            )}
          </h2>
          <p>
            {t(
              l,
              "Verifying authorized accounts and sealed source revisions.",
              "허용된 계정과 확정된 수집 리비전을 확인하고 있습니다.",
              "التحقق من الحسابات المصرح بها وإصدارات المصدر.",
            )}
          </p>
        </section>
      )}
      {error && (
        <section role="alert" className="unified-panel">
          <h2>
            {t(
              l,
              "Collected data unavailable",
              "수집 데이터 조회 불가",
              "البيانات المجمعة غير متاحة",
            )}
          </h2>
          <p>
            {t(
              l,
              "The source, permission or selection could not be verified. Refresh to start a new query; previous amounts are cleared.",
              "원천·권한·조회 범위를 확인하지 못했습니다. 새로 조회하면 현재 범위에서 다시 시작합니다. 이전 금액은 제거했습니다.",
              "تعذر التحقق من المصدر أو الصلاحية أو النطاق. أعد الاستعلام؛ تم مسح المبالغ السابقة.",
            )}
          </p>
          <p>
            <bdi>{error}</bdi>
          </p>
          <button onClick={refresh}>
            {t(l, "Refresh query", "다시 조회", "إعادة الاستعلام")}
          </button>
        </section>
      )}
      {data && (
        <>
          <div className="overview-basis">
            <p>
              {t(
                l,
                "Collected FOCUS samples",
                "수집된 FOCUS 샘플",
                "عينات FOCUS مجمعة",
              )}{" "}
              <bdi>{data.versions.join(", ")}</bdi> ·{" "}
              {t(l, "Public", "공개", "عام")}{" "}
              {Object.entries(data.provenance)
                .filter(([k]) => k.startsWith("official-public"))
                .reduce((n, [, v]) => n + v, 0)}{" "}
              · {t(l, "Synthetic", "합성", "تجريبي")}{" "}
              {data.provenance["synthetic-contract-fixture"] ?? 0} ·{" "}
              {t(
                l,
                "No FX conversion · query only",
                "환산 없음 · 조회 전용",
                "دون تحويل عملة · للقراءة فقط",
              )}
            </p>
            <details>
              <summary>
                {t(
                  l,
                  "Source and organization basis",
                  "수집·조직 기준",
                  "أساس المصدر والمؤسسة",
                )}
              </summary>
              <p>
                {t(
                  l,
                  "Amounts use charge-start month in each tenant timezone without proration. Periods differ by provider. BilledCost and EffectiveCost are separate source costs, not customer sales or approved allocations.",
                  "테넌트 시간대의 과금 시작 월을 사용하며 기간 배분하지 않습니다. 공급사별 수집 기간이 다릅니다. BilledCost와 EffectiveCost는 원천 비용이며 고객 판매·승인된 배분 금액과 다릅니다.",
                  "تُنسب القيم لشهر بدء الرسوم حسب المنطقة الزمنية للمستأجر دون توزيع زمني. تختلف الفترات بين المزودين. التكاليف مستقلة عن المبيعات والتخصيصات المعتمدة.",
                )}
              </p>
              <p>
                {t(
                  l,
                  "Internal company membership is synthetic sample authorization. BillingAccountId and SubAccountId are FOCUS-observed scopes; legal entities, departments and native ancestry are unverified.",
                  "내부 회사 구성은 합성 샘플 권한 매핑입니다. BillingAccountId·SubAccountId는 FOCUS에서 확인된 계정 범위이며, 법인·부서·원천 조직 계층은 미확인입니다.",
                  "عضوية الشركة الداخلية تخويل تجريبي. الحسابات مرصودة في FOCUS؛ الكيانات القانونية والأقسام والتسلسل المؤسسي غير متحقق منها.",
                )}
              </p>
              <p>
                <bdi>{data.company}</bdi> ·{" "}
                {data.tenants.map((v) => (
                  <bdi key={v.key}>
                    {v.key} ({v.timezone}){" "}
                  </bdi>
                ))}
              </p>
              <p>
                {t(
                  l,
                  "Finalized slices · authorized source scope",
                  "확정 수집 슬라이스 · 전체 권한 원천 범위",
                  "شرائح نهائية · نطاق المصدر المصرح به",
                )}
                : {data.coverage.finalized_slices}/{data.coverage.slices}.{" "}
                {t(
                  l,
                  "Operator coverage assertions are unverified; source finalization is not financial closing.",
                  "운영자 수집 범위 선언은 미검증이며, 수집 확정은 재무 마감이 아닙니다.",
                  "إقرارات تغطية المشغل غير متحققة؛ نهائية المصدر ليست إقفالاً مالياً.",
                )}
              </p>
              <p>
                {t(l, "Query observed at", "조회 시각", "وقت الاستعلام")}:{" "}
                <bdi>{data.observed_at}</bdi>
              </p>
              <p>
                {t(l, "Selection snapshot", "조회 스냅샷", "لقطة الاستعلام")}:{" "}
                <bdi>{data.snapshot}</bdi>
              </p>
            </details>
          </div>
          {view !== "settlement" ? (
            unconnected
          ) : (
            <>
              <section
                className="focus-amounts"
                aria-label={t(l, "Cost summary", "비용 요약", "ملخص التكلفة")}
              >
                {data.amounts.map((a) => (
                  <div key={a.currency}>
                    <h2>
                      {a.currency} · {label}
                    </h2>
                    <strong className="focus-money">
                      {amount(a[basis].coefficient, a.currency)}
                    </strong>
                    <small>
                      {a.rows} {t(l, "rows", "행", "صفوف")} ·{" "}
                      {a[basis].missing
                        ? t(
                            l,
                            `${a[basis].missing} values missing`,
                            `${a[basis].missing}개 값 누락`,
                            `${a[basis].missing} قيم مفقودة`,
                          )
                        : t(
                            l,
                            "No values missing",
                            "값 누락 없음",
                            "دون قيم مفقودة",
                          )}
                    </small>
                  </div>
                ))}
                {!data.amounts.length && (
                  <div>
                    <h2>{label}</h2>
                    <strong>
                      {t(
                        l,
                        "No matching rows",
                        "해당 범위 행 없음",
                        "لا صفوف مطابقة",
                      )}
                    </strong>
                  </div>
                )}
                <div>
                  <h2>
                    {t(l, "Collected scope", "수집 범위", "نطاق التجميع")}
                  </h2>
                  <p>
                    <strong data-focus-rows>{data.row_count}</strong> /{" "}
                    {data.authorized_rows}{" "}
                    {t(l, "authorized rows", "권한 범위 행", "صفوف مصرح بها")}
                  </p>
                  <small>
                    {data.account_count}{" "}
                    {t(
                      l,
                      "account combinations",
                      "계정 조합",
                      "مجموعات حسابات",
                    )}{" "}
                    · {data.providers.length}{" "}
                    {t(l, "providers", "공급사", "مزودين")}
                  </small>
                </div>
                <div>
                  <h2>
                    {t(
                      l,
                      "Operational connection",
                      "업무 연결",
                      "الاتصال التشغيلي",
                    )}
                  </h2>
                  <p>
                    {t(
                      l,
                      "Sales · allocation · AR · closing",
                      "판매·배분·미수·마감",
                      "المبيعات · التخصيص · المستحقات · الإقفال",
                    )}
                  </p>
                  <small>{t(l, "Not connected", "미연결", "غير متصل")}</small>
                </div>
              </section>
              <div className="focus-analysis">
                <FocusChart
                  data={data}
                  currency={chartCurrency}
                  locale={l}
                  onCurrency={(v) => change({ currency: v })}
                  onMonth={(v) => change({ period: v })}
                />
                <section className="unified-panel focus-providers">
                  <header>
                    <h2>
                      {t(
                        l,
                        "Provider composition",
                        "공급사별 구성",
                        "توزيع المزودين",
                      )}
                    </h2>
                    <span>{label}</span>
                  </header>
                  {table(
                    t(l, "Provider amounts", "공급사별 금액", "مبالغ المزودين"),
                    <table>
                      <thead>
                        <tr>
                          <th>{t(l, "Provider", "공급사", "المزود")}</th>
                          <th>{label}</th>
                          <th>{t(l, "Rows", "행", "صفوف")}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {data.amounts.map((a) => (
                          <tr className="focus-subtotal" key={a.currency}>
                            <th>
                              {a.currency} {t(l, "total", "합계", "المجموع")}
                            </th>
                            <td>{amount(a[basis].coefficient, a.currency)}</td>
                            <td>{a.rows}</td>
                          </tr>
                        ))}
                        {data.providers.flatMap((p) =>
                          p.amounts.map((a) => (
                            <tr key={p.provider + a.currency}>
                              <th>
                                <button
                                  onClick={() =>
                                    change({
                                      provider: p.provider,
                                      account: "",
                                    })
                                  }
                                >
                                  {p.provider}
                                </button>
                              </th>
                              <td>
                                {amount(a[basis].coefficient, a.currency)}
                                {a[basis].missing > 0 && (
                                  <small>
                                    {t(l, "Partial", "부분", "جزئي")}:{" "}
                                    {a[basis].missing}
                                  </small>
                                )}
                              </td>
                              <td>{a.rows}</td>
                            </tr>
                          )),
                        )}
                      </tbody>
                    </table>,
                  )}
                </section>
              </div>
              <section className="unified-panel focus-accounts">
                <header>
                  <div>
                    <h2>
                      {t(
                        l,
                        "Organization and account mapping",
                        "조직·계정 매핑",
                        "ربط المؤسسة والحسابات",
                      )}
                    </h2>
                    <p>
                      {t(
                        l,
                        "Observed billing and subaccounts · internal sample membership",
                        "수집된 결제·하위 계정 · 내부 샘플 소속",
                        "حسابات فوترة وحسابات فرعية مرصودة · عضوية تجريبية داخلية",
                      )}
                    </p>
                  </div>
                  {data.selection.account && (
                    <button onClick={() => change({ account: "" })}>
                      {t(
                        l,
                        "Clear account scope",
                        "계정 범위 해제",
                        "مسح نطاق الحساب",
                      )}
                    </button>
                  )}
                </header>
                <form
                  className="focus-search"
                  onSubmit={(e) => {
                    e.preventDefault();
                    change({ q: search.trim() });
                  }}
                >
                  <label htmlFor="focus-account-search">
                    {t(
                      l,
                      "Search account mapping only",
                      "계정 매핑 목록 내 검색",
                      "البحث في ربط الحسابات فقط",
                    )}
                  </label>
                  <input
                    id="focus-account-search"
                    type="search"
                    maxLength={128}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                  />
                  <button>{t(l, "Search", "검색", "بحث")}</button>
                  <span>
                    {data.account_matches} / {data.account_count}
                  </span>
                </form>
                {table(
                  t(l, "Account mapping", "계정 매핑", "ربط الحسابات"),
                  <table>
                    <thead>
                      <tr>
                        <th>
                          {t(
                            l,
                            "Internal sample tenant",
                            "내부 샘플 테넌트",
                            "مستأجر تجريبي داخلي",
                          )}
                        </th>
                        <th>{t(l, "Provider", "공급사", "المزود")}</th>
                        <th>BillingAccountId</th>
                        <th>SubAccountId</th>
                        <th>BilledCost / EffectiveCost</th>
                        <th>{t(l, "Rows", "행", "صفوف")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.accounts.map((a) => (
                        <tr key={a.id}>
                          <td>
                            <bdi>{a.tenant}</bdi>
                          </td>
                          <td>{a.provider}</td>
                          <th>
                            <button
                              onClick={() => change({ account: a.id, q: "" })}
                            >
                              <bdi>{a.billing_account}</bdi>
                            </button>
                          </th>
                          <td>
                            <bdi>
                              {a.sub_account ??
                                t(l, "Not provided", "제공 없음", "غير مقدم")}
                            </bdi>
                          </td>
                          <td>
                            {a.amounts.map((v) => (
                              <div
                                className="focus-account-money"
                                key={v.currency}
                              >
                                {amount(v.billed_cost.coefficient, v.currency)}
                                <small>
                                  {amount(
                                    v.effective_cost.coefficient,
                                    v.currency,
                                  )}
                                </small>
                              </div>
                            ))}
                          </td>
                          <td>{a.rows}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>,
                )}
                {!data.account_matches && (
                  <p role="status">
                    {t(
                      l,
                      "No matching account mappings. Cost totals are unchanged by list search.",
                      "검색된 계정 매핑이 없습니다. 목록 검색은 비용 합계를 변경하지 않습니다.",
                      "لا روابط حسابات مطابقة. بحث القائمة لا يغير إجماليات التكلفة.",
                    )}
                  </p>
                )}
                {pager(
                  "account_after",
                  data.account_matches,
                  data.account_after,
                  data.next_account_after,
                )}
              </section>
              <section className="unified-panel focus-source">
                <header>
                  <h2 ref={detailHeading} tabIndex={-1}>
                    {t(
                      l,
                      "Collected source details",
                      "수집 원천 상세",
                      "تفاصيل المصدر المجمع",
                    )}
                  </h2>
                  <span>
                    {data.row_count}{" "}
                    {t(
                      l,
                      "rows in selected scope",
                      "행 · 선택 범위",
                      "صفوف ضمن النطاق",
                    )}
                  </span>
                </header>
                {table(
                  t(
                    l,
                    "Collected FOCUS rows",
                    "수집 FOCUS 행",
                    "صفوف FOCUS المجمعة",
                  ),
                  <table>
                    <thead>
                      <tr>
                        <th>
                          {t(l, "Charge month", "과금 시작 월", "شهر الرسوم")}
                        </th>
                        <th>
                          {t(
                            l,
                            "Provider / service",
                            "공급사 / 서비스",
                            "المزود / الخدمة",
                          )}
                        </th>
                        <th>BillingAccountId / SubAccountId</th>
                        <th>BilledCost</th>
                        <th>EffectiveCost</th>
                        <th>{t(l, "Source", "출처", "المصدر")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.rows.map((r) => (
                        <tr key={r.id}>
                          <td>
                            <bdi>{r.charge_month}</bdi>
                          </td>
                          <td>
                            {r.provider}
                            <small>{r.service ?? "—"}</small>
                          </td>
                          <td>
                            <bdi>{r.billing_account}</bdi>
                            <small>
                              <bdi>{r.sub_account ?? "—"}</bdi>
                            </small>
                          </td>
                          <td>{amount(r.billed_cost, r.currency)}</td>
                          <td>{amount(r.effective_cost, r.currency)}</td>
                          <td>
                            <details>
                              <summary>{r.evidence_class}</summary>
                              <dl>
                                <dt>
                                  {t(
                                    l,
                                    "Charge start",
                                    "과금 시작",
                                    "بدء الرسوم",
                                  )}
                                </dt>
                                <dd>
                                  <bdi>{r.charge_start}</bdi>
                                </dd>
                                <dt>
                                  {t(
                                    l,
                                    "Charge end",
                                    "과금 종료",
                                    "نهاية الرسوم",
                                  )}
                                </dt>
                                <dd>
                                  <bdi>{r.charge_end}</bdi>
                                </dd>
                                <dt>
                                  {t(
                                    l,
                                    "Batch / ordinal",
                                    "수집 배치 / 순번",
                                    "دفعة / ترتيب",
                                  )}
                                </dt>
                                <dd>
                                  <bdi>
                                    {r.batch} / {r.ordinal}
                                  </bdi>
                                </dd>
                              </dl>
                            </details>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>,
                )}
                {!data.row_count && (
                  <p>
                    {t(
                      l,
                      "No rows in this selection",
                      "해당 범위에 수집 행이 없습니다",
                      "لا صفوف ضمن هذا النطاق",
                    )}
                  </p>
                )}
                {pager("after", data.row_count, data.after, data.next_after)}
              </section>
            </>
          )}
        </>
      )}
    </div>
  );
}
