import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { useLocation } from "../../../../shared/navigation";
import {
  loadOverview,
  ServiceError,
  type OverviewData,
  type FocusRow,
} from "../../collected-focus/api";
import { decimal, money } from "../../collected-focus/values";
import { focusHref, focusQuery } from "../../collected-focus/query";
import "./style.css";
export default function CollectedSource({
  locale: l,
  view,
  onState,
  objectId,
}: {
  locale: Locale;
  view: string;
  objectId?: string;
  onState: (s: string, items: string[]) => void;
}) {
  const hash = useLocation(),
    params = new URLSearchParams(hash.split("?")[1]),
    query = focusQuery(params),
    key = query.toString();
  const [state, setState] = useState<{
    key: string;
    data?: OverviewData;
    error?: string;
  }>({ key });
  const [retry, setRetry] = useState(0),
    [selected, setSelected] = useState<{ key: string; id: string } | null>(
      null,
    ),
    [search, setSearch] = useState(query.get("q") ?? "");
  const generation = useRef(0),
    detail = useRef<HTMLElement>(null),
    trigger = useRef<HTMLElement | null>(null);
  const data = state.key === key ? state.data : undefined,
    error = state.key === key ? state.error : undefined;
  const row =
    selected?.key === key
      ? data?.rows.find((r) => r.id === selected.id)
      : undefined;
  useEffect(() => {
    const c = new AbortController(),
      g = ++generation.current;
    setState({ key });
    setSelected(null);
    if (objectId) {
      setState({ key, error: "selection_unavailable" });
      return () => c.abort();
    }
    loadOverview(new URLSearchParams(key), c.signal)
      .then((data) => {
        if (!c.signal.aborted && generation.current === g)
          setState({ key, data });
      })
      .catch((e) => {
        if (!c.signal.aborted && generation.current === g)
          setState({
            key,
            error: e instanceof ServiceError ? e.code : "service_unavailable",
          });
      });
    return () => c.abort();
  }, [key, retry, objectId]);
  useEffect(() => setSearch(query.get("q") ?? ""), [key]);
  useEffect(() => {
    if (row) detail.current?.focus();
  }, [row?.id, view]);
  useEffect(
    () =>
      onState(
        JSON.stringify({
          source: "collected-focus",
          selection: data?.selection ?? null,
          snapshot: data?.snapshot ?? null,
          after: data?.after ?? null,
          account_after: data?.account_after ?? null,
          view,
          row: row?.id ?? null,
          status: error ?? (data ? "loaded" : "loading"),
        }),
        [
          "S04:scope:collected-focus",
          "S04:table:normalized",
          "S04:table:usage",
          "S04:scope:account-mapping",
          "S04:detail:row",
          "S04:page:snapshot",
          "S04:state:unavailable",
          "S04:state:history-unconnected",
        ],
      ),
    [data, error, row?.id, view, onState],
  );
  function change(values: Record<string, string>, continuation = false) {
    const p = new URLSearchParams(location.hash.split("?")[1]);
    if (!continuation)
      for (const k of ["after", "account_after", "snapshot"])
        p.delete(`focus-${k}`);
    for (const [k, v] of Object.entries(values))
      v ? p.set(`focus-${k}`, v) : p.delete(`focus-${k}`);
    setSelected(null);
    location.hash = `#/source-data?${p}`;
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
      setSelected(null);
      setRetry((n) => n + 1);
    }
  }
  function inspect(r: FocusRow) {
    trigger.current = document.activeElement as HTMLElement;
    setSelected({ key, id: r.id });
  }
  function close() {
    setSelected(null);
    if (trigger.current?.isConnected) trigger.current.focus();
  }
  useEffect(() => {
    if (!row) return;
    const escape = (event: KeyboardEvent) => {
      if (
        event.key === "Escape" &&
        !event.defaultPrevented &&
        !document.querySelector("dialog[open]")
      ) {
        event.preventDefault();
        close();
      }
    };
    window.addEventListener("keydown", escape);
    return () => window.removeEventListener("keydown", escape);
  }, [row?.id]);
  const none = t(l, "Unavailable", "미확인", "غير متاح"),
    unitNone = t(l, "Unit unavailable", "단위 미확인", "الوحدة غير متاحة");
  const quantity = (v: string | null, u: string | null) => (
    <>
      <bdi dir="ltr">{v === null ? none : decimal(v, l)}</bdi>
      <small>{u ?? unitNone}</small>
    </>
  );
  const value = (v: string | null, c: string) => (
    <bdi dir="ltr">{v === null ? `${c} · ${none}` : money(v, c, l)}</bdi>
  );
  const rowLabel = (r: FocusRow) =>
    `${t(l, "Inspect row", "행 상세", "تفاصيل الصف")} ${r.provider} ${r.billing_account} ${r.sub_account ?? ""} ${r.charge_month} ${r.id.slice(0, 8)}`;
  const pager = data && (
    <nav
      className="collected-pages"
      aria-label={t(l, "Source pages", "명세 페이지", "صفحات المصدر")}
    >
      <span aria-live="polite">
        <bdi>
          {data.row_count ? data.after + 1 : 0}–
          {Math.min(data.after + 25, data.row_count)} / {data.row_count}
        </bdi>{" "}
        {t(l, "rows", "행", "صفوف")}
      </span>
      <button
        disabled={!data.after}
        onClick={() =>
          change(
            {
              after: String(Math.max(0, data.after - 25)),
              snapshot: data.snapshot,
            },
            true,
          )
        }
      >
        {t(l, "Previous", "이전", "السابق")}
      </button>
      <button
        disabled={data.next_after === null}
        onClick={() =>
          data.next_after !== null &&
          change(
            { after: String(data.next_after), snapshot: data.snapshot },
            true,
          )
        }
      >
        {t(l, "Next", "다음", "التالي")}
      </button>
    </nav>
  );
  return (
    <div
      className="collected-source"
      data-source-state={error ? "error" : data ? "loaded" : "loading"}
      data-source-query={key}
      data-source-view={view}
      aria-busy={!data && !error}
    >
      <div className="collected-toolbar">
        <label>
          {t(l, "Charge period", "과금 기간", "فترة الرسوم")}
          <select
            id="source-period"
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
            {data?.options.periods.map((p) => (
              <option key={p}>{p}</option>
            ))}
          </select>
        </label>
        <label>
          {t(l, "Provider", "공급사", "المزود")}
          <select
            id="source-provider"
            disabled={!data}
            value={query.get("provider") ?? ""}
            onChange={(e) => change({ provider: e.target.value, account: "" })}
          >
            <option value="">
              {t(l, "All providers", "전체 공급사", "كل المزودين")}
            </option>
            {data?.options.providers.map((p) => (
              <option key={p}>{p}</option>
            ))}
          </select>
        </label>
        <label>
          {t(l, "Cost basis", "비용 기준", "أساس التكلفة")}
          <select
            id="source-basis"
            value={query.get("basis") ?? "billed_cost"}
            onChange={(e) => change({ basis: e.target.value })}
          >
            <option value="billed_cost">BilledCost</option>
            <option value="effective_cost">EffectiveCost</option>
          </select>
        </label>
        <button onClick={refresh} disabled={!data && !error}>
          {t(l, "Refresh query", "새로 조회", "تحديث الاستعلام")}
        </button>
        <a href={focusHref("/home", params, data?.snapshot)}>
          {t(l, "Back to overview", "통합 현황으로", "العودة للنظرة العامة")}
        </a>
        {data?.selection.account && (
          <button onClick={() => change({ account: "" })}>
            {t(l, "Clear account scope", "계정 범위 해제", "مسح نطاق الحساب")}
          </button>
        )}
      </div>
      {!data && !error && (
        <section role="status">
          <h2>
            {t(
              l,
              "Reading collected source rows",
              "수집 명세 조회 중",
              "جارٍ قراءة الصفوف المجمعة",
            )}
          </h2>
          <p>
            {t(
              l,
              "Verifying source revisions and authorized account scope.",
              "수집 리비전과 허용된 계정 범위를 확인하고 있습니다.",
              "جارٍ التحقق من إصدارات المصدر ونطاق الحسابات المصرح بها.",
            )}
          </p>
        </section>
      )}
      {error && (
        <section role="alert">
          <h2>
            {t(
              l,
              "Source query unavailable",
              "명세 조회 불가",
              "الاستعلام غير متاح",
            )}
          </h2>
          <p>
            {t(
              l,
              "The source, permission or pinned selection could not be verified. Previous rows are cleared. Refresh starts a new snapshot.",
              "원천·권한·고정된 조회 범위를 확인하지 못했습니다. 이전 행은 제거했습니다. 새로 조회하면 새 스냅샷으로 시작합니다.",
              "تعذر التحقق من المصدر أو الصلاحية أو اللقطة. مُسحت الصفوف السابقة؛ التحديث يبدأ لقطة جديدة.",
            )}
          </p>
          <p>
            <bdi>{error}</bdi>
          </p>
          <button onClick={refresh}>
            {t(l, "Restart query", "다시 조회", "إعادة الاستعلام")}
          </button>
        </section>
      )}
      {data && (
        <>
          <div className="collected-provenance">
            <p>
              {t(
                l,
                "Collected FOCUS samples",
                "수집된 FOCUS 샘플",
                "عينات FOCUS مجمعة",
              )}{" "}
              <bdi>{data.versions.join(", ")}</bdi> ·{" "}
              {t(
                l,
                "Query only · no FX conversion",
                "조회 전용 · 환산 없음",
                "للاستعلام فقط · دون تحويل عملة",
              )}
            </p>
            <details>
              <summary>
                {t(
                  l,
                  "Scope and source basis",
                  "조회·출처 기준",
                  "أساس النطاق والمصدر",
                )}
              </summary>
              <p>
                {t(
                  l,
                  "Public samples and explicitly synthetic rows. Internal company membership is sample authorization; billing and subaccount IDs are observed FOCUS scopes. Native organization ancestry is unverified.",
                  "공개 샘플과 명시된 합성 행입니다. 내부 회사 소속은 샘플 권한 매핑이며 결제·하위 계정 ID는 FOCUS 수집 범위입니다. 원천 조직 계층은 미확인입니다.",
                  "عينات عامة وصفوف تجريبية صريحة. عضوية الشركة تخويل تجريبي والحسابات نطاقات FOCUS مرصودة؛ التسلسل المؤسسي غير متحقق منه.",
                )}
              </p>
              <p>
                {t(
                  l,
                  "Month attribution uses charge start in tenant timezone without proration. Quantities and prices retain original units. No mixed-unit totals or inferred pricing formula.",
                  "테넌트 시간대의 과금 시작 월을 사용하며 기간 배분하지 않습니다. 수량·단가는 원래 단위를 유지합니다. 단위가 다른 합계나 요금 산식을 추정하지 않습니다.",
                  "يُنسب الشهر لبدء الرسوم حسب منطقة المستأجر دون توزيع زمني. تبقى الوحدات أصلية دون جمع مختلط أو صيغة سعر مفترضة.",
                )}
              </p>
              <dl>
                <dt>
                  {t(l, "Selection snapshot", "조회 스냅샷", "لقطة الاستعلام")}
                </dt>
                <dd>
                  <bdi>{data.snapshot}</bdi>
                </dd>
                <dt>{t(l, "Source snapshot", "원천 스냅샷", "لقطة المصدر")}</dt>
                <dd>
                  <bdi>{data.source_snapshot}</bdi>
                </dd>
                <dt>{t(l, "Observed at", "조회 시각", "وقت الاستعلام")}</dt>
                <dd>
                  <bdi>{data.observed_at}</bdi>
                </dd>
              </dl>
            </details>
          </div>
          <section className="collected-scope">
            <header>
              <h2>{t(l, "Selected scope", "선택 범위", "النطاق المحدد")}</h2>
              <span>
                <bdi data-source-total>{data.row_count}</bdi>{" "}
                {t(l, "rows", "행", "صفوف")} · {data.account_count}{" "}
                {t(l, "account combinations", "계정 조합", "مجموعات حسابات")}
              </span>
            </header>
            <div className="collected-totals">
              {data.amounts.map((a) => (
                <div key={a.currency}>
                  <span>
                    {data.selection.basis === "billed_cost"
                      ? "BilledCost"
                      : "EffectiveCost"}
                  </span>
                  <strong>
                    {value(a[data.selection.basis].coefficient, a.currency)}
                  </strong>
                  <small>
                    {a[data.selection.basis].missing
                      ? t(
                          l,
                          `${a[data.selection.basis].missing} missing values`,
                          `${a[data.selection.basis].missing}개 값 누락`,
                          `${a[data.selection.basis].missing} قيم مفقودة`,
                        )
                      : t(
                          l,
                          "No cost values missing",
                          "비용 값 누락 없음",
                          "لا قيم تكلفة مفقودة",
                        )}
                  </small>
                </div>
              ))}
            </div>
          </section>
          <details className="collected-account">
            <summary>
              {t(
                l,
                "Find and narrow account mapping",
                "계정 매핑 찾기·범위 지정",
                "بحث وتحديد ربط الحساب",
              )}
            </summary>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                change({ q: search.trim() });
              }}
            >
              <label htmlFor="source-account-search">
                {t(
                  l,
                  "Account-list search only",
                  "계정 목록 내 검색",
                  "البحث في قائمة الحسابات فقط",
                )}
              </label>
              <input
                id="source-account-search"
                type="search"
                maxLength={128}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
              <button>
                {t(l, "Search accounts", "계정 검색", "بحث الحسابات")}
              </button>
            </form>
            <p>
              {t(
                l,
                "List search leaves source row totals unchanged. Select an account to narrow all source rows.",
                "목록 검색은 명세 합계를 바꾸지 않습니다. 계정을 선택하면 전체 명세 범위가 좁혀집니다.",
                "بحث القائمة لا يغير الإجماليات. اختيار حساب يحدد نطاق كل الصفوف.",
              )}
            </p>
            <ul>
              {data.accounts.map((a) => (
                <li key={a.id}>
                  <button onClick={() => change({ account: a.id, q: "" })}>
                    {a.provider} · <bdi>{a.billing_account}</bdi> /{" "}
                    <bdi>
                      {a.sub_account ??
                        t(
                          l,
                          "Subaccount not provided",
                          "하위 계정 제공 없음",
                          "الحساب الفرعي غير مقدم",
                        )}
                    </bdi>{" "}
                    · {a.rows} {t(l, "rows", "행", "صفوف")}
                  </button>
                </li>
              ))}
            </ul>
            <div className="collected-pages">
              <bdi>
                {data.account_matches ? data.account_after + 1 : 0}–
                {Math.min(data.account_after + 25, data.account_matches)} /{" "}
                {data.account_matches}
              </bdi>
              <button
                disabled={!data.account_after}
                onClick={() =>
                  change(
                    {
                      account_after: String(
                        Math.max(0, data.account_after - 25),
                      ),
                      snapshot: data.snapshot,
                    },
                    true,
                  )
                }
              >
                {t(l, "Previous accounts", "이전 계정", "الحسابات السابقة")}
              </button>
              <button
                disabled={data.next_account_after === null}
                onClick={() =>
                  data.next_account_after !== null &&
                  change(
                    {
                      account_after: String(data.next_account_after),
                      snapshot: data.snapshot,
                    },
                    true,
                  )
                }
              >
                {t(l, "Next accounts", "다음 계정", "الحسابات التالية")}
              </button>
            </div>
          </details>
          {view === "history" ? (
            <section className="collected-unavailable">
              <h2>
                {t(
                  l,
                  "Transformation history not connected",
                  "변환 이력 미연결",
                  "سجل التحويل غير متصل",
                )}
              </h2>
              <p>
                {t(
                  l,
                  "This collected path exposes verified normalized fields and source lineage. A row-level transformation history and AI execution evidence are unavailable.",
                  "현재 수집 경로는 검증된 정규화 필드와 수집 근거를 제공합니다. 행별 변환 이력과 AI 실행 근거는 미연결입니다.",
                  "يعرض المسار حقولاً مطبّعة ونسب المصدر. سجل التحويل على مستوى الصف وأدلة تنفيذ الذكاء الاصطناعي غير متاحة.",
                )}
              </p>
            </section>
          ) : (
            <>
              <header className="collected-list-header">
                <div>
                  <h2>
                    {view === "usage"
                      ? t(
                          l,
                          "Native quantities and prices",
                          "원천 수량·단가",
                          "الكميات والأسعار الأصلية",
                        )
                      : view === "fields"
                        ? t(
                            l,
                            "Mapped source fields",
                            "매핑된 원천 필드",
                            "حقول المصدر المربوطة",
                          )
                        : t(
                            l,
                            "Normalized cost details",
                            "정규화 비용 명세",
                            "تفاصيل التكلفة المطبّعة",
                          )}
                  </h2>
                  <p>
                    {view === "fields"
                      ? t(
                          l,
                          "Allowlisted fields only · select a row for billing/charge intervals and source lineage.",
                          "허용된 필드만 표시합니다. 행을 선택하면 청구·과금 구간과 수집 근거를 확인할 수 있습니다.",
                          "حقول مسموح بها فقط؛ اختر صفاً للفترات ونسب المصدر.",
                        )
                      : t(
                          l,
                          "Complete selected-scope totals above · 25 rows per page.",
                          "상단 합계는 전체 선택 범위 기준 · 페이지당 25행입니다.",
                          "الإجماليات أعلاه للنطاق الكامل · 25 صفاً لكل صفحة.",
                        )}
                  </p>
                </div>
                {pager}
              </header>
              <p className="collected-scroll-hint">
                {t(
                  l,
                  "Scroll horizontally to see all source columns.",
                  "전체 명세 열은 표를 가로로 스크롤하여 확인할 수 있습니다.",
                  "مرّر الجدول أفقياً لعرض كل أعمدة المصدر.",
                )}
              </p>
              <div
                className="collected-table"
                role="region"
                tabIndex={0}
                aria-label={t(
                  l,
                  "Collected source table, horizontally scrollable",
                  "수집 명세 표, 가로 스크롤",
                  "جدول المصدر المجمع، قابل للتمرير أفقياً",
                )}
              >
                <table>
                  <thead>
                    <tr>
                      <th scope="col">
                        {t(
                          l,
                          "Account / detail",
                          "계정 / 상세",
                          "الحساب / التفاصيل",
                        )}
                      </th>
                      <th scope="col">
                        {t(
                          l,
                          "Provider / service",
                          "공급사 / 서비스",
                          "المزود / الخدمة",
                        )}
                      </th>
                      <th scope="col">
                        {t(l, "Charge month", "과금 시작 월", "شهر بدء الرسوم")}
                      </th>
                      {view === "usage" ? (
                        <>
                          <th scope="col">
                            {t(
                              l,
                              "Consumed quantity / unit",
                              "소비 수량 / 단위",
                              "كمية الاستهلاك / الوحدة",
                            )}
                          </th>
                          <th scope="col">
                            {t(
                              l,
                              "Pricing quantity / unit",
                              "가격 수량 / 단위",
                              "كمية التسعير / الوحدة",
                            )}
                          </th>
                          <th scope="col">
                            {t(
                              l,
                              "List unit price / pricing unit",
                              "정가 단가 / 가격 단위",
                              "سعر القائمة / وحدة التسعير",
                            )}
                          </th>
                          <th scope="col">
                            {t(
                              l,
                              "Contracted unit price / pricing unit",
                              "계약 단가 / 가격 단위",
                              "سعر العقد / وحدة التسعير",
                            )}
                          </th>
                        </>
                      ) : view === "fields" ? (
                        <>
                          <th scope="col">
                            {t(
                              l,
                              "Charge interval",
                              "과금 구간",
                              "فترة الرسوم",
                            )}
                          </th>
                          <th scope="col">
                            {t(
                              l,
                              "Billing interval",
                              "청구 구간",
                              "فترة الفوترة",
                            )}
                          </th>
                          <th scope="col">
                            {t(
                              l,
                              "Batch / ordinal",
                              "수집 배치 / 순번",
                              "دفعة / ترتيب",
                            )}
                          </th>
                        </>
                      ) : (
                        <>
                          <th scope="col">BilledCost</th>
                          <th scope="col">EffectiveCost</th>
                          <th scope="col">
                            {t(
                              l,
                              "Consumed quantity / unit",
                              "소비 수량 / 단위",
                              "كمية الاستهلاك / الوحدة",
                            )}
                          </th>
                        </>
                      )}
                      <th scope="col">
                        {t(l, "Evidence", "출처 근거", "دليل المصدر")}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.rows.map((r) => (
                      <tr key={r.id} data-row-id={r.id}>
                        <th scope="row">
                          <button
                            aria-label={rowLabel(r)}
                            aria-expanded={row?.id === r.id}
                            onClick={() => inspect(r)}
                          >
                            <bdi>{r.billing_account}</bdi>
                          </button>
                          <small>
                            <bdi>
                              {r.sub_account ??
                                t(
                                  l,
                                  "Subaccount not provided",
                                  "하위 계정 제공 없음",
                                  "الحساب الفرعي غير مقدم",
                                )}
                            </bdi>
                          </small>
                        </th>
                        <td>
                          {r.provider}
                          <small>{r.service ?? none}</small>
                        </td>
                        <td>
                          <bdi>{r.charge_month}</bdi>
                        </td>
                        {view === "usage" ? (
                          <>
                            <td className="number">
                              {quantity(r.consumed_quantity, r.consumed_unit)}
                            </td>
                            <td className="number">
                              {quantity(r.pricing_quantity, r.pricing_unit)}
                            </td>
                            <td className="number">
                              {value(r.list_unit_price, r.currency)}
                              <small>{r.pricing_unit ?? unitNone}</small>
                            </td>
                            <td className="number">
                              {value(r.contracted_unit_price, r.currency)}
                              <small>{r.pricing_unit ?? unitNone}</small>
                            </td>
                          </>
                        ) : view === "fields" ? (
                          <>
                            <td>
                              <bdi>{r.charge_start}</bdi>
                              <small>
                                <bdi>{r.charge_end}</bdi>
                              </small>
                            </td>
                            <td>
                              <bdi>{r.billing_start}</bdi>
                              <small>
                                <bdi>{r.billing_end}</bdi>
                              </small>
                            </td>
                            <td>
                              <bdi>{r.batch}</bdi>
                              <small>{r.ordinal}</small>
                            </td>
                          </>
                        ) : (
                          <>
                            <td className="number">
                              {value(r.billed_cost, r.currency)}
                            </td>
                            <td className="number">
                              {value(r.effective_cost, r.currency)}
                            </td>
                            <td className="number">
                              {quantity(r.consumed_quantity, r.consumed_unit)}
                            </td>
                          </>
                        )}
                        <td>
                          {r.evidence_class.startsWith("official-public")
                            ? t(l, "Public sample", "공개 샘플", "عينة عامة")
                            : r.evidence_class === "synthetic-contract-fixture"
                              ? t(
                                  l,
                                  "Synthetic sample",
                                  "합성 샘플",
                                  "عينة تجريبية",
                                )
                              : t(l, "Unclassified", "출처 미분류", "غير مصنف")}
                          <small>
                            <bdi>{r.id.slice(0, 12)}</bdi>
                          </small>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {!data.rows.length && (
                <p role="status">
                  {t(
                    l,
                    "No rows match the selected scope. Change period, provider or account.",
                    "선택 범위에 해당하는 행이 없습니다. 기간·공급사·계정을 변경하세요.",
                    "لا صفوف مطابقة؛ غيّر الفترة أو المزود أو الحساب.",
                  )}
                </p>
              )}
              {pager}
              {row && (
                <section
                  className="collected-detail"
                  ref={detail}
                  tabIndex={-1}
                  aria-label={t(
                    l,
                    "Selected source row",
                    "선택 명세 상세",
                    "تفاصيل الصف المحدد",
                  )}
                >
                  <header>
                    <h2>
                      {t(
                        l,
                        "Selected row details",
                        "선택 행 상세",
                        "تفاصيل الصف المحدد",
                      )}
                    </h2>
                    <button onClick={close}>
                      {t(l, "Back to row", "행으로 돌아가기", "العودة للصف")}
                    </button>
                  </header>
                  <div className="collected-detail-grid">
                    <dl>
                      <dt>
                        {t(
                          l,
                          "Provider / service",
                          "공급사 / 서비스",
                          "المزود / الخدمة",
                        )}
                      </dt>
                      <dd>
                        {row.provider} / {row.service ?? none}
                      </dd>
                      <dt>BillingAccountId</dt>
                      <dd>
                        <bdi>{row.billing_account}</bdi>
                      </dd>
                      <dt>SubAccountId</dt>
                      <dd>
                        <bdi>{row.sub_account ?? none}</bdi>
                      </dd>
                      <dt>
                        {t(l, "Charge interval", "과금 구간", "فترة الرسوم")}
                      </dt>
                      <dd>
                        <bdi>{row.charge_start}</bdi>
                        <br />
                        <bdi>{row.charge_end}</bdi>
                      </dd>
                    </dl>
                    <dl>
                      {(
                        [
                          "billed_cost",
                          "effective_cost",
                          "list_cost",
                          "contracted_cost",
                        ] as const
                      ).map((k) => (
                        <div key={k}>
                          <dt>
                            {
                              {
                                billed_cost: "BilledCost",
                                effective_cost: "EffectiveCost",
                                list_cost: "ListCost",
                                contracted_cost: "ContractedCost",
                              }[k]
                            }
                          </dt>
                          <dd>{value(row[k], row.currency)}</dd>
                        </div>
                      ))}
                    </dl>
                    <dl>
                      <dt>
                        {t(
                          l,
                          "Consumed quantity",
                          "소비 수량",
                          "كمية الاستهلاك",
                        )}
                      </dt>
                      <dd>
                        {quantity(row.consumed_quantity, row.consumed_unit)}
                      </dd>
                      <dt>
                        {t(l, "Pricing quantity", "가격 수량", "كمية التسعير")}
                      </dt>
                      <dd>
                        {quantity(row.pricing_quantity, row.pricing_unit)}
                      </dd>
                      <dt>
                        {t(l, "List unit price", "정가 단가", "سعر القائمة")}
                      </dt>
                      <dd>
                        {value(row.list_unit_price, row.currency)}
                        <small>{row.pricing_unit ?? unitNone}</small>
                      </dd>
                      <dt>
                        {t(
                          l,
                          "Contracted unit price",
                          "계약 단가",
                          "سعر العقد",
                        )}
                      </dt>
                      <dd>
                        {value(row.contracted_unit_price, row.currency)}
                        <small>{row.pricing_unit ?? unitNone}</small>
                      </dd>
                    </dl>
                    <dl>
                      <dt>
                        {t(
                          l,
                          "Canonical row identity",
                          "행 식별자",
                          "هوية الصف",
                        )}
                      </dt>
                      <dd>
                        <bdi>{row.id}</bdi>
                      </dd>
                      <dt>
                        {t(
                          l,
                          "Evidence / batch / ordinal",
                          "출처 / 배치 / 순번",
                          "الدليل / الدفعة / الترتيب",
                        )}
                      </dt>
                      <dd>
                        <bdi>{row.evidence_class}</bdi>
                        <br />
                        <bdi>
                          {row.batch} / {row.ordinal}
                        </bdi>
                      </dd>
                      <dt>
                        {t(l, "Billing interval", "청구 구간", "فترة الفوترة")}
                      </dt>
                      <dd>
                        <bdi>{row.billing_start}</bdi>
                        <br />
                        <bdi>{row.billing_end}</bdi>
                      </dd>
                      <dt>
                        {t(
                          l,
                          "FOCUS versions in authorized source scope",
                          "전체 권한 원천 범위의 FOCUS 버전",
                          "إصدارات FOCUS في نطاق المصدر المصرح به",
                        )}
                      </dt>
                      <dd>{data.versions.join(", ")}</dd>
                    </dl>
                  </div>
                </section>
              )}
            </>
          )}
        </>
      )}
    </div>
  );
}
