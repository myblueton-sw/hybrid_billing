import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { load, query, ServiceError, type Data } from "./api";
import { t } from "./copy";
import AccountScope from "./AccountScope";
import { Summary, Totals, CandidateList, Seller } from "./SourceViews";
import "./screen.css";
type State = { key: string; data?: Data; error?: string; requestId?: string };
const items = [
  "B07:scope:source",
  "B07:data:buyer-candidates",
  "B07:data:seller-boundary",
  "B07:data:benefit-credit-evidence",
  "B07:data:mapping-evidence",
  "B07:action:inspect-source",
  "B07:state:async",
  "B07:state:value-missing-zero",
];
const requestKey = (p: URLSearchParams) =>
  JSON.stringify([
    query(p).toString(),
    [...p]
      .filter(
        ([k]) =>
          !k.startsWith("focus-") &&
          !["marketplace-row", "view", "lang"].includes(k),
      )
      .sort(),
  ]);
export default function Screen({
  locale: l,
  params,
  view,
  onState,
  onDirty,
}: {
  locale: Locale;
  params: URLSearchParams;
  view: string;
  onState: (s: string, i: string[]) => void;
  onDirty: (d: boolean) => void;
}) {
  const q = query(params),
    key = requestKey(params),
    [state, setState] = useState<State>({ key: "" }),
    [retry, setRetry] = useState(0),
    [attempts, setAttempts] = useState(0),
    [search, setSearch] = useState(q.get("q") ?? "");
  const seq = useRef(0),
    returnFocus = useRef<string | null>(null),
    status = useRef<HTMLParagraphElement>(null),
    focusStatus = useRef(false),
    current = state.key === key ? state : undefined,
    data = current?.data;
  useEffect(() => onDirty(false), [onDirty]);
  useEffect(() => {
    const controller = new AbortController(),
      n = ++seq.current;
    setState({ key });
    setSearch(q.get("q") ?? "");
    load(q, controller.signal)
      .then((data) => {
        if (!controller.signal.aborted && n === seq.current)
          setState({ key, data });
      })
      .catch((error) => {
        if (!controller.signal.aborted && n === seq.current)
          setState({
            key,
            error:
              error instanceof ServiceError
                ? error.code
                : "service_unavailable",
            requestId:
              error instanceof ServiceError ? error.requestId : undefined,
          });
      });
    return () => controller.abort();
  }, [key, retry]);
  const phase = current?.error ? "error" : data ? "loaded" : "loading";
  const review = JSON.stringify({
    phase: "marketplace-evidence",
    status: phase,
    view,
    requestKey: key,
    scope: data?.snapshot ?? "none",
    page: data?.after ?? 0,
    accountPage: data?.account_after ?? 0,
    selected: data?.selected ? [data.selected.id] : [],
    detail: !!data?.selected,
    data: data ?? null,
    error: current?.error ?? null,
    requestId: current?.requestId ?? null,
    attempts,
  });
  useEffect(() => onState(review, items), [review, onState]);
  useEffect(() => {
    if (!data) return;
    if (returnFocus.current && !data.selected) {
      const id = returnFocus.current;
      returnFocus.current = null;
      requestAnimationFrame(() =>
        [
          ...document.querySelectorAll<HTMLButtonElement>(
            "button[data-b07-row]",
          ),
        ]
          .find((b) => b.dataset.b07Row === id && b.getClientRects().length)
          ?.focus(),
      );
    } else if (focusStatus.current) {
      focusStatus.current = false;
      status.current?.focus();
    }
  }, [data]);
  function go(next: URLSearchParams) {
    setState({ key: "pending" });
    setAttempts((n) => n + 1);
    if (requestKey(next) === key) {
      setState({ key });
      setRetry((n) => n + 1);
    }
    if (next.toString() !== params.toString())
      location.hash = `#/marketplace?${next}`;
  }
  function fresh(next: URLSearchParams) {
    for (const k of [
      "marketplace-row",
      "focus-after",
      "focus-account_after",
      "focus-snapshot",
    ])
      next.delete(k);
    returnFocus.current = null;
    focusStatus.current = true;
    go(next);
  }
  function scope(k: string, v: string) {
    const next = new URLSearchParams(params);
    next.set("focus-" + k, v);
    fresh(next);
  }
  function select(id: string) {
    if (
      !data ||
      !data.rows.some((r) => r.id === id) ||
      data.selected?.id === id
    )
      return;
    const next = new URLSearchParams(params);
    next.set("marketplace-row", id);
    next.set("focus-snapshot", data.snapshot);
    go(next);
  }
  function close() {
    if (!data?.selected) return;
    returnFocus.current = data.selected.id;
    const next = new URLSearchParams(params);
    next.delete("marketplace-row");
    next.set("focus-snapshot", data.snapshot);
    go(next);
  }
  function page(n: number, account = false) {
    if (!data) return;
    const next = new URLSearchParams(params);
    next.delete("marketplace-row");
    next.set("focus-snapshot", data.snapshot);
    next.set(account ? "focus-account_after" : "focus-after", String(n));
    if (account) next.delete("focus-after");
    returnFocus.current = null;
    focusStatus.current = true;
    go(next);
  }
  const denied =
      !!current?.error &&
      /permission|revoked|denied|forbidden/.test(current.error),
    stale =
      !!current?.error &&
      /snapshot|stale|cursor|selection_unavailable/.test(current.error);
  return (
    <div
      className="b07-screen"
      data-b07-state={phase}
      data-b07-view={view}
      data-b07-selected={data?.selected?.id ?? ""}
      aria-busy={phase === "loading"}
      onKeyDown={(e) => {
        if (e.key === "Escape" && !e.defaultPrevented) {
          const d = (e.target as HTMLElement).closest<HTMLDetailsElement>(
            "details[open]",
          );
          if (d) {
            d.open = false;
            d.querySelector("summary")?.focus();
            e.preventDefault();
          }
        }
      }}
    >
      <form
        className="b07-toolbar"
        onSubmit={(e) => {
          e.preventDefault();
          scope("q", search);
        }}
      >
        <label htmlFor="b07-provider">
          {t(l, "Collected source", "수집 원천", "المصدر المجمّع")}
          <select
            id="b07-provider"
            value={q.get("provider") ?? ""}
            onChange={(e) => scope("provider", e.target.value)}
          >
            <option value="">
              {t(
                l,
                "All authorized providers",
                "전체 허용 공급사",
                "كل المزوّدين المصرّح بهم",
              )}
            </option>
            {[
              ...new Set([
                q.get("provider") ?? "",
                ...(data?.options.providers ?? []),
              ]),
            ]
              .filter(Boolean)
              .map((p) => (
                <option key={p}>{p}</option>
              ))}
          </select>
        </label>
        <label htmlFor="b07-period">
          {t(l, "Charge-start month", "과금 시작 월", "شهر بدء الرسوم")}
          <select
            id="b07-period"
            value={q.get("period") ?? "all"}
            onChange={(e) => scope("period", e.target.value)}
          >
            <option value="all">
              {t(
                l,
                "All charge-start months",
                "전체 과금 시작 월",
                "كل أشهر بدء الرسوم",
              )}
            </option>
            {[
              ...new Set([
                q.get("period") ?? "",
                ...(data?.options.periods ?? []),
              ]),
            ]
              .filter((p) => p && p !== "all")
              .map((p) => (
                <option key={p}>{p}</option>
              ))}
          </select>
        </label>
        <label htmlFor="b07-search">
          {t(l, "Account search", "계정 검색", "بحث الحسابات")}
          <input
            id="b07-search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            aria-describedby="b07-search-help"
          />
          <small id="b07-search-help">
            {t(
              l,
              "Searches the account list only",
              "계정 목록만 검색합니다",
              "يبحث في قائمة الحسابات فقط",
            )}
          </small>
        </label>
        <button type="submit">{t(l, "Search", "검색", "بحث")}</button>
        <button
          type="button"
          onClick={() => fresh(new URLSearchParams(params))}
        >
          {t(l, "New query", "새 조회", "استعلام جديد")}
        </button>
      </form>
      {current?.error ? (
        <section className="b07-error" role="alert">
          <h2>
            {denied
              ? t(
                  l,
                  "Source access unavailable",
                  "원천 접근 불가",
                  "الوصول للمصدر غير متاح",
                )
              : t(
                  l,
                  "Source query unavailable",
                  "원천 조회 불가",
                  "استعلام المصدر غير متاح",
                )}
          </h2>
          <p>
            {denied
              ? t(
                  l,
                  "Source details have been removed. Start a new authorized query.",
                  "원천 상세를 제거했습니다. 허용된 범위에서 새로 조회하세요.",
                  "أُزيلت تفاصيل المصدر. ابدأ استعلاماً جديداً ضمن النطاق المصرّح به.",
                )
              : stale
                ? t(
                    l,
                    "The snapshot or row selection is no longer current. Start a new query and select again.",
                    "조회 기준이나 선택 행이 현재 범위와 맞지 않습니다. 새로 조회한 뒤 다시 선택하세요.",
                    "مرجع الاستعلام أو الصف المحدد لم يعد حالياً. ابدأ استعلاماً جديداً ثم أعد الاختيار.",
                  )
                : t(
                    l,
                    "The source response could not be read safely. Retry this query or start a new one.",
                    "원천 응답을 안전하게 읽지 못했습니다. 다시 시도하거나 새로 조회하세요.",
                    "تعذرت قراءة استجابة المصدر بأمان. أعد المحاولة أو ابدأ استعلاماً جديداً.",
                  )}
          </p>
          <p className="b07-meta">
            <bdi dir="ltr">
              {current.error} {current.requestId}
            </bdi>
          </p>
          {!denied && !stale && (
            <button
              onClick={() => {
                setState({ key });
                setAttempts((n) => n + 1);
                setRetry((n) => n + 1);
              }}
            >
              {t(l, "Retry", "다시 시도", "إعادة المحاولة")}
            </button>
          )}
        </section>
      ) : !data ? (
        <p role="status">
          {t(
            l,
            "Loading authorized source evidence…",
            "허용된 원천 근거를 불러오는 중입니다…",
            "جارٍ تحميل أدلة المصدر المصرّح بها…",
          )}
        </p>
      ) : (
        <>
          <AccountScope
            locale={l}
            data={data}
            select={(id) => scope("account", id)}
            page={(n) => page(n, true)}
          />
          <p role="status" tabIndex={-1} ref={status} className="b07-meta">
            <span>
              {t(
                l,
                "Observed source rows",
                "관측 원천 행",
                "صفوف المصدر المرصودة",
              )}{" "}
              <bdi dir="ltr">{data.row_count}</bdi>
            </span>{" "}
            ·{" "}
            <span>
              {t(
                l,
                "Candidate source rows",
                "후보 원천 행",
                "صفوف المصدر المرشّحة",
              )}{" "}
              <bdi dir="ltr">{data.counts.candidate}</bdi>
            </span>{" "}
            · <bdi dir="ltr">FOCUS {data.versions.join(", ")}</bdi>
          </p>
          {view === "seller" ? (
            <Seller locale={l} count={data.counts.candidate} />
          ) : (
            <>
              <p className="b07-lead">
                {view === "buyer"
                  ? t(
                      l,
                      "Source rows with different invoice issuer and publisher names are Marketplace candidates, not confirmed purchases or customer invoices.",
                      "청구서 발행사·상품 제공사 이름이 다른 원천 행을 Marketplace 후보로 봅니다. 확정 구매·고객 청구가 아닙니다.",
                      "صفوف المصدر التي تختلف فيها أسماء جهة إصدار الفاتورة وناشر المنتج مرشّحة للسوق، وليست مشتريات أو فواتير عملاء مؤكدة.",
                    )
                  : view === "credits"
                    ? t(
                        l,
                        "Charge categories and commitment references are source context. Benefit eligibility and credit matching need contractual evidence.",
                        "청구 분류·약정 참조는 원천 맥락입니다. 혜택 적격성과 크레딧 연결에는 계약 근거가 필요합니다.",
                        "فئات الرسوم ومراجع الالتزام سياق من المصدر. أهلية المنافع وربط الأرصدة يتطلبان أدلة تعاقدية.",
                      )
                    : t(
                        l,
                        "Inspect observed source references alongside contract mappings that are not connected.",
                        "관측된 원천 참조와 아직 연결되지 않은 계약 근거를 함께 확인합니다.",
                        "افحص مراجع المصدر المرصودة بجانب الروابط التعاقدية غير المتصلة.",
                      )}
              </p>
              {view === "buyer" && (
                <>
                  <Summary locale={l} data={data} />
                  <Totals locale={l} data={data} />
                </>
              )}
              <CandidateList
                locale={l}
                data={data}
                view={view}
                select={select}
                close={close}
                page={(n) => page(n)}
              />
            </>
          )}
          <details className="b07-provenance">
            <summary>
              {t(
                l,
                "Query and coverage evidence",
                "조회·수집 범위 근거",
                "أدلة الاستعلام والتغطية",
              )}
            </summary>
            <p>
              {t(
                l,
                "Coverage declarations cover all authorized collected sources, not the selected provider/month. They are operator assertions, not verified completeness or financial approval.",
                "수집 범위 선언은 전체 허용 수집 원천 기준이며 선택한 공급사·월의 완전성을 뜻하지 않습니다. 운영자 선언은 검증된 완전성·재무 승인이 아닙니다.",
                "إقرارات التغطية تخص كل المصادر المجمّعة المصرّح بها، لا المزوّد أو الشهر المحدد. هي إقرارات مشغّل وليست اكتمالاً متحققاً أو اعتماداً مالياً.",
              )}
            </p>
            <dl className="b07-facts">
              {[
                [
                  t(
                    l,
                    "Declared finalized / observed slices",
                    "확정 선언 / 관측 구간",
                    "الشرائح المعلنة نهائية / المرصودة",
                  ),
                  `${data.coverage.finalized_slices} / ${data.coverage.slices}`,
                ],
                [
                  t(
                    l,
                    "Earliest declared complete-through",
                    "최소 완료 선언 시각",
                    "أقدم نهاية اكتمال معلنة",
                  ),
                  data.coverage.complete_through ??
                    t(l, "No declaration", "선언 없음", "لا إقرار"),
                ],
                [
                  t(
                    l,
                    "Source read time",
                    "원천 조회 시각",
                    "وقت قراءة المصدر",
                  ),
                  data.observed_at,
                ],
                [
                  t(
                    l,
                    "Collected-data reference ID",
                    "수집 데이터 기준 ID",
                    "معرّف مرجع البيانات المجمّعة",
                  ),
                  data.source_snapshot,
                ],
                [
                  t(
                    l,
                    "Query reference ID",
                    "조회 기준 ID",
                    "معرّف مرجع الاستعلام",
                  ),
                  data.snapshot,
                ],
                [
                  t(
                    l,
                    "Version mapping reference",
                    "버전 매핑 참조",
                    "مرجع ربط الإصدار",
                  ),
                  data.mapping_digest,
                ],
              ].map(([label, value]) => (
                <div key={label}>
                  <dt>{label}</dt>
                  <dd>
                    <bdi dir="ltr">{value}</bdi>
                  </dd>
                </div>
              ))}
            </dl>
          </details>
        </>
      )}
    </div>
  );
}
