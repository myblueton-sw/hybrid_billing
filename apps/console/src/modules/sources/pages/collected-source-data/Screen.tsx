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
import { provenanceClass, valueState } from "../../collected-focus/fields";
import FieldReference from "./FieldReference";
import FieldsTable from "./FieldsTable";
import RowDetail from "./RowDetail";
import {
  missingLabel,
  selectedLabel,
  provenanceLabel,
  unitMissingLabel,
  zeroLabel,
} from "./labels";
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
  }, [row?.id]);
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
          "S04:reference:field-mapping",
          "S04:detail:field-origin",
          "S04:state:value-missing-zero",
          "S04:state:evidence-unavailable",
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
  function inspect(r: FocusRow, element: HTMLElement) {
    trigger.current = element;
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
  const none = missingLabel(l),
    unitNone = unitMissingLabel(l);
  const zero = (v: string | null) =>
    valueState(v) === "zero" && <small>{zeroLabel(l)}</small>;
  const quantity = (v: string | null, u: string | null) => (
    <>
      {v === null ? none : <bdi dir="ltr">{decimal(v, l)}</bdi>}
      {zero(v)}
      <small>{u ?? unitNone}</small>
    </>
  );
  const value = (v: string | null, c: string) => (
    <>
      {v === null ? none : <bdi dir="ltr">{money(v, c, l)}</bdi>}
      {zero(v)}
    </>
  );
  const total = (v: string | null, c: string) =>
    v === null ? (
      <>
        <bdi dir="ltr">{c}</bdi>{" "}
        {t(l, "no values present", "값 없음", "لا قيم موجودة")}
      </>
    ) : (
      <bdi dir="ltr">{money(v, c, l)}</bdi>
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
          {t(l, "Charge-start month", "과금 시작 월", "شهر بدء الرسوم")}
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
            <option value="billed_cost">
              {t(l, "Billed cost", "청구 비용", "التكلفة المفوترة")}
            </option>
            <option value="effective_cost">
              {t(l, "Effective cost", "실효 비용", "التكلفة الفعلية")}
            </option>
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
              "The source, permission or pinned selection could not be verified. Previous rows are cleared. Refresh starts a new selection reference ID.",
              "원천·권한·고정된 조회 범위를 확인하지 못했습니다. 이전 행은 제거했습니다. 새로 조회하면 새 조회 기준 ID로 시작합니다.",
              "تعذر التحقق من المصدر أو الصلاحية أو اللقطة. مُسحت الصفوف السابقة؛ التحديث يبدأ معرّف مرجع استعلام جديد.",
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
              <p>
                {t(
                  l,
                  "Account scopes distinguish provider, billing account and subaccount within authorized tenant access; they are not legal entities or unique billing accounts. A selection reference identifies the chosen source context; a source data reference identifies the collected dataset. Neither signifies financial approval.",
                  "계정 범위는 접근 권한 내 공급사·결제 계정·하위 계정을 구분하며 법인 수나 고유 결제 계정 수가 아닙니다. 조회 기준 ID는 선택한 조회 범위, 수집 데이터 기준 ID는 수집 데이터 집합을 식별하며 재무 승인을 뜻하지 않습니다.",
                  "تميّز نطاقات الحسابات المزود وحساب الفوترة والحساب الفرعي ضمن وصول المستأجر المصرّح به؛ وليست كيانات قانونية أو حسابات فوترة فريدة. يحدّد مرجع الاستعلام نطاق البيانات المختار ومرجع بيانات المصدر مجموعة البيانات المجمعة؛ ولا يدل أيّ منهما على اعتماد مالي.",
                )}
              </p>
              <dl>
                <dt>
                  {t(
                    l,
                    "Selection reference ID",
                    "조회 기준 ID",
                    "معرّف مرجع الاستعلام",
                  )}
                </dt>
                <dd>
                  <bdi>{data.snapshot}</bdi>
                </dd>
                <dt>
                  {t(
                    l,
                    "Source data reference ID",
                    "수집 데이터 기준 ID",
                    "معرّف مرجع بيانات المصدر",
                  )}
                </dt>
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
                {t(l, "account scopes", "계정 범위 수", "نطاقات حسابات")}
              </span>
            </header>
            <div className="collected-totals">
              {data.amounts.map((a) => (
                <div key={a.currency}>
                  <span>
                    {data.selection.basis === "billed_cost"
                      ? t(l, "Billed cost", "청구 비용", "التكلفة المفوترة")
                      : t(l, "Effective cost", "실효 비용", "التكلفة الفعلية")}
                  </span>
                  <strong>
                    {total(a[data.selection.basis].coefficient, a.currency)}
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
                    <bdi>{a.sub_account ?? none}</bdi> · {a.rows}{" "}
                    {t(l, "rows", "행", "صفوف")}
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
                  "This collected path shows normalized fields and delivery lineage. A row-level transformation history and AI execution evidence are unavailable.",
                  "현재 수집 경로는 정규화 필드와 수집본 식별 정보를 제공합니다. 행별 변환 이력과 AI 실행 근거는 미연결입니다.",
                  "يعرض المسار حقولاً مطبّعة وبيانات تتبّع التسليم. سجل التحويل على مستوى الصف وأدلة تنفيذ الذكاء الاصطناعي غير متاحة.",
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
                          "Columns are grouped by where each value comes from. The Missing values column counts, per row, how many of the 12 nullable source fields have no value. Select a row to see every field.",
                          "열은 값의 출처별로 묶여 있습니다. 누락 값 열은 값이 없을 수 있는 원천 필드 12개 중 행마다 누락된 개수를 표시합니다. 행을 선택하면 모든 필드를 확인할 수 있습니다.",
                          "الأعمدة مجمّعة حسب مصدر كل قيمة. يعرض عمود القيم المفقودة لكل صف عدد الحقول المفقودة من بين 12 حقلاً مصدرياً يمكن أن تخلو من قيمة. اختر صفاً لعرض كل الحقول.",
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
              {view === "fields" && (
                <FieldReference l={l} versions={data.versions} />
              )}
              <p className="collected-scroll-hint">
                {t(
                  l,
                  "Scroll the table horizontally to see all columns.",
                  "표를 가로로 스크롤하면 모든 열을 확인할 수 있습니다.",
                  "مرّر الجدول أفقياً لعرض كل الأعمدة.",
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
                {view === "fields" ? (
                  <FieldsTable
                    l={l}
                    rows={data.rows}
                    versions={data.versions}
                    selectedId={row?.id}
                    rowLabel={rowLabel}
                    onInspect={inspect}
                  />
                ) : (
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
                          {t(
                            l,
                            "Charge-start month",
                            "과금 시작 월",
                            "شهر بدء الرسوم",
                          )}
                        </th>
                        {view === "usage" ? (
                          <>
                            <th scope="col">
                              {t(
                                l,
                                "Consumed quantity / unit",
                                "사용 수량 / 단위",
                                "كمية الاستهلاك / الوحدة",
                              )}
                            </th>
                            <th scope="col">
                              {t(
                                l,
                                "Pricing quantity / unit",
                                "요금 산정 수량 / 단위",
                                "كمية التسعير / الوحدة",
                              )}
                            </th>
                            <th scope="col">
                              {t(
                                l,
                                "List unit price / pricing unit",
                                "정가 단가 / 요금 산정 단위",
                                "سعر القائمة / وحدة التسعير",
                              )}
                            </th>
                            <th scope="col">
                              {t(
                                l,
                                "Contracted unit price / pricing unit",
                                "계약 단가 / 요금 산정 단위",
                                "سعر العقد / وحدة التسعير",
                              )}
                            </th>
                          </>
                        ) : (
                          <>
                            <th scope="col">
                              {t(
                                l,
                                "Billed cost",
                                "청구 비용",
                                "التكلفة المفوترة",
                              )}
                            </th>
                            <th scope="col">
                              {t(
                                l,
                                "Effective cost",
                                "실효 비용",
                                "التكلفة الفعلية",
                              )}
                            </th>
                            <th scope="col">
                              {t(
                                l,
                                "Consumed quantity / unit",
                                "사용 수량 / 단위",
                                "كمية الاستهلاك / الوحدة",
                              )}
                            </th>
                          </>
                        )}
                        <th scope="col">
                          {t(
                            l,
                            "Sample provenance / row ID",
                            "샘플 출처 / 행 ID",
                            "منشأ العينة / معرّف الصف",
                          )}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.rows.map((r) => (
                        <tr
                          key={r.id}
                          data-row-id={r.id}
                          data-selected={row?.id === r.id || undefined}
                        >
                          <th scope="row">
                            <button
                              aria-label={rowLabel(r)}
                              aria-expanded={row?.id === r.id}
                              onClick={(event) =>
                                inspect(r, event.currentTarget)
                              }
                            >
                              <bdi>{r.billing_account}</bdi>
                            </button>
                            <small>
                              {r.sub_account === null ? (
                                none
                              ) : (
                                <bdi>{r.sub_account}</bdi>
                              )}
                            </small>
                            {row?.id === r.id && (
                              <small className="collected-selected">
                                {selectedLabel(l)}
                              </small>
                            )}
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
                            {provenanceLabel(
                              l,
                              provenanceClass(r.evidence_class),
                            )}
                            <small>
                              <bdi>{r.id.slice(0, 12)}</bdi>
                            </small>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
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
                <RowDetail
                  l={l}
                  row={row}
                  data={data}
                  detailRef={detail}
                  onClose={close}
                />
              )}
            </>
          )}
        </>
      )}
    </div>
  );
}
