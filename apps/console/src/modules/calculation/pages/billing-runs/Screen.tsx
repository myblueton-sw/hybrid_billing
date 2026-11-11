import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { load, query, ServiceError, type OverviewData } from "./api";
import { makeBaseline, compatible, type Baseline } from "./model";
import { t } from "./copy";
import AccountScope from "./AccountScope";
import SourceInputs, { Totals } from "./SourceInputs";
import {
  Runs,
  Authorities,
  Trace,
  Comparison,
  Validation,
  Recognition,
} from "./ViewPanels";
import "./screen.css";
type State = {
  key: string;
  data?: OverviewData;
  error?: string;
  requestId?: string;
};
const items = [
  "B08:scope:source",
  "B08:data:run-boundary",
  "B08:data:input-evidence",
  "B08:data:result-boundary",
  "B08:action:source-baseline",
  "B08:data:source-comparison",
  "B08:data:validation-boundary",
  "B08:data:recognition-boundary",
  "B08:action:inspect-source",
  "B08:state:async",
  "B08:data:source-field-origin",
  "B08:state:value-missing-zero",
];
const requestKey = (p: URLSearchParams) =>
  JSON.stringify([
    query(p).toString(),
    [...p]
      .filter(([k]) => !k.startsWith("focus-") && !["view", "lang"].includes(k))
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
    [search, setSearch] = useState(q.get("q") ?? ""),
    [pin, setPin] = useState<Baseline | null>(null),
    [reason, setReason] = useState(""),
    [selection, setSelection] = useState<{ key: string; id: string } | null>(
      null,
    );
  const sourceScope = JSON.stringify([
    q.get("provider") ?? "",
    q.get("account") ?? "",
    q.get("period") ?? "all",
  ]);
  const lastScope = useRef(sourceScope);
  const sequence = useRef(0),
    status = useRef<HTMLParagraphElement>(null),
    focusStatus = useRef(false),
    current = state.key === key ? state : undefined,
    data = current?.data,
    baseline = data && pin && compatible(pin, data) ? pin : null,
    selected =
      data && selection?.key === key
        ? data.rows.find((r) => r.id === selection.id)
        : undefined;
  useEffect(() => onDirty(false), [onDirty]);
  useEffect(() => {
    const c = new AbortController(),
      n = ++sequence.current;
    if (lastScope.current !== sourceScope) {
      lastScope.current = sourceScope;
      setPin(null);
      setReason("scope_changed");
    }
    setState({ key });
    setSearch(q.get("q") ?? "");
    setSelection(null);
    load(q, c.signal)
      .then((data) => {
        if (!c.signal.aborted && n === sequence.current)
          setState({ key, data });
      })
      .catch((e) => {
        if (!c.signal.aborted && n === sequence.current) {
          setPin(null);
          setReason("failed");
          setState({
            key,
            error: e instanceof ServiceError ? e.code : "service_unavailable",
            requestId: e instanceof ServiceError ? e.requestId : undefined,
          });
        }
      });
    return () => c.abort();
  }, [key, retry]);
  useEffect(() => {
    if (data && pin && !compatible(pin, data)) {
      setPin(null);
      setReason("source_changed");
    }
  }, [data, pin]);
  useEffect(() => {
    if (data && focusStatus.current) {
      focusStatus.current = false;
      status.current?.focus();
    }
  }, [data]);
  const phase = current?.error ? "error" : data ? "loaded" : "loading",
    review = JSON.stringify({
      phase: "billing-input-evidence",
      variant: `${view}:${phase}:${baseline ? "baseline" : "no-baseline"}`,
      requestKey: key,
      scope: data?.snapshot ?? "none",
      status: phase,
      view,
      data: data ?? null,
      baseline: baseline ?? null,
      baselineState: reason,
      selected: selected ? [selected.id] : [],
      detail: !!selected,
      page: data?.after ?? 0,
      error: current?.error ?? null,
      requestId: current?.requestId ?? null,
      attempts,
    });
  useEffect(() => onState(review, items), [review, onState]);
  function go(next: URLSearchParams) {
    setState({ key: "pending" });
    setSelection(null);
    setAttempts((n) => n + 1);
    if (requestKey(next) === key) {
      setState({ key });
      setRetry((n) => n + 1);
    }
    if (next.toString() !== params.toString())
      location.hash = `#/billing-runs?${next}`;
  }
  function fresh(next: URLSearchParams) {
    for (const k of ["focus-after", "focus-account_after", "focus-snapshot"])
      next.delete(k);
    focusStatus.current = true;
    go(next);
  }
  function scope(k: string, v: string) {
    const next = new URLSearchParams(params);
    next.set("focus-" + k, v);
    if (k !== "q") {
      setPin(null);
      setReason("scope_changed");
    }
    fresh(next);
  }
  function page(n: number, account = false) {
    if (!data) return;
    const next = new URLSearchParams(params);
    next.set("focus-snapshot", data.snapshot);
    next.set(account ? "focus-account_after" : "focus-after", String(n));
    if (account) next.delete("focus-after");
    focusStatus.current = true;
    go(next);
  }
  function goView(v: string) {
    const next = new URLSearchParams(params);
    next.set("view", v);
    location.hash = `#/billing-runs?${next}`;
  }
  const denied =
      !!current?.error &&
      /denied|revoked|permission|forbidden/.test(current.error),
    stale =
      !!current?.error &&
      /snapshot|selection_unavailable|stale|cursor/.test(current.error);
  return (
    <div
      className="b08-screen"
      data-b08-state={phase}
      data-b08-view={view}
      data-b08-baseline={baseline ? "kept" : "none"}
      data-b08-selected={selected?.id ?? ""}
      aria-busy={phase === "loading"}
      onKeyDown={(e) => {
        if (e.key === "Escape" && !e.defaultPrevented) {
          const d = (e.target as HTMLElement).closest<HTMLDetailsElement>(
            "details[open]",
          );
          if (
            d?.classList.contains("b08-full-table") &&
            window.innerWidth >= 768
          ) {
            if (selected) setSelection(null);
            e.preventDefault();
          } else if (d) {
            d.open = false;
            d.querySelector("summary")?.focus();
            e.preventDefault();
          }
        }
      }}
    >
      <form
        className="b08-toolbar"
        onSubmit={(e) => {
          e.preventDefault();
          scope("q", search);
        }}
      >
        <label htmlFor="b08-provider">
          {t(l, "Collected source", "수집 원천", "المصدر المجمّع")}
          <select
            id="b08-provider"
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
        <label htmlFor="b08-period">
          {t(l, "Charge-start month", "과금 시작 월", "شهر بدء الرسوم")}
          <select
            id="b08-period"
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
        <label htmlFor="b08-search">
          {t(l, "Account search", "계정 검색", "بحث الحسابات")}
          <input
            id="b08-search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            aria-describedby="b08-search-help"
          />
          <small id="b08-search-help">
            {t(
              l,
              "Searches account options only",
              "계정 목록만 검색합니다",
              "يبحث في خيارات الحسابات فقط",
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
        <section role="alert" className="b08-error">
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
                  "Source read unavailable",
                  "원천 조회 불가",
                  "قراءة المصدر غير متاحة",
                )}
          </h2>
          <p>
            {stale
              ? t(
                  l,
                  "The source reference is no longer current. Start a new query.",
                  "현재 원천 기준과 일치하지 않습니다. 새로 조회하세요.",
                  "مرجع المصدر لم يعد حالياً. ابدأ استعلاماً جديداً.",
                )
              : t(
                  l,
                  "Source details and the temporary baseline were removed. No financial run was created.",
                  "원천 상세와 임시 비교 기준을 제거했습니다. 청구 계산 실행은 생성되지 않았습니다.",
                  "أُزيلت تفاصيل المصدر والمرجع المؤقت. لم تُنشأ عملية احتساب مالي.",
                )}
          </p>
          <p className="b08-meta">
            <bdi dir="ltr">
              {current.error} {current.requestId}
            </bdi>
          </p>
          {!denied && !stale && (
            <button
              onClick={() => {
                setState({ key });
                setRetry((n) => n + 1);
                setAttempts((n) => n + 1);
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
            "Reading authorized source evidence…",
            "허용된 원천 근거를 조회하는 중입니다…",
            "جارٍ قراءة أدلة المصدر المصرّح بها…",
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
          <p className="b08-meta" role="status" tabIndex={-1} ref={status}>
            {t(
              l,
              "Observed source rows",
              "관측 원천 행",
              "صفوف المصدر المرصودة",
            )}{" "}
            <bdi className="b08-count" dir="ltr">
              {data.row_count}
            </bdi>{" "}
            <bdi dir="ltr">FOCUS {data.versions.join(", ")}</bdi>
          </p>
          <section className="b08-baseline-strip">
            <div>
              <p role="status">
                {baseline
                  ? t(
                      l,
                      "Temporary source baseline kept",
                      "원천 비교 기준을 잠시 보관 중입니다",
                      "مرجع المصدر محفوظ مؤقتاً",
                    )
                  : reason === "source_changed"
                    ? t(
                        l,
                        "Source reference changed; keep a new comparison baseline.",
                        "원천 기준이 바뀌었습니다. 비교 기준을 다시 선택하세요.",
                        "تغيّر مرجع المصدر؛ احتفظ بمرجع مقارنة جديد.",
                      )
                    : reason === "scope_changed"
                      ? t(
                          l,
                          "Previous baseline discarded: scope changed.",
                          "조회 범위가 바뀌어 이전 비교 기준을 버렸습니다.",
                          "أُلغي المرجع السابق لتغيّر النطاق.",
                        )
                      : reason === "failed"
                        ? t(
                            l,
                            "Previous baseline discarded after an unsuccessful read.",
                            "조회 실패로 이전 비교 기준을 버렸습니다.",
                            "أُلغي المرجع السابق بعد تعذر القراءة.",
                          )
                        : t(
                            l,
                            "No source comparison baseline kept",
                            "보관 중인 원천 비교 기준이 없습니다",
                            "لا مرجع مقارنة محفوظ للمصدر",
                          )}
              </p>
              <small>
                {t(
                  l,
                  "Memory only on this screen. Not a frozen manifest, calculation lock, job or billing result. Leaving the screen discards it.",
                  "이 화면의 메모리에만 보관합니다. 확정 대상 명세·계산 잠금·작업·청구 결과가 아니며 화면을 떠나면 사라집니다.",
                  "يُحفظ في ذاكرة هذه الشاشة فقط. ليس بياناً ثابتاً أو قفل احتساب أو مهمة أو نتيجة فوترة. يُلغى عند مغادرة الشاشة.",
                )}
              </small>
            </div>
            <div className="b08-actions">
              {baseline ? (
                <>
                  <button
                    id="b08-open-comparison"
                    onClick={() => goView("comparison")}
                  >
                    {t(
                      l,
                      "View source comparison",
                      "원천 비교 보기",
                      "عرض مقارنة المصدر",
                    )}
                  </button>
                  <button
                    id="b08-discard"
                    onClick={() => {
                      setPin(null);
                      setReason("discarded");
                    }}
                  >
                    {t(
                      l,
                      "Discard baseline",
                      "비교 기준 버리기",
                      "إلغاء مرجع المقارنة",
                    )}
                  </button>
                </>
              ) : (
                <button
                  id="b08-keep"
                  disabled={!data.row_count}
                  onClick={() => {
                    setPin(makeBaseline(data));
                    setReason("kept");
                  }}
                >
                  {t(
                    l,
                    "Temporarily keep source baseline",
                    "원천 비교 기준 잠시 보관",
                    "الاحتفاظ المؤقت بمرجع المصدر",
                  )}
                </button>
              )}
            </div>
          </section>
          {view === "runs" ? (
            <Runs locale={l} data={data} go={goView} />
          ) : view === "inputs" ? (
            <>
              <Totals locale={l} data={data} />
              <SourceInputs
                locale={l}
                data={data}
                selected={selected}
                select={(id) => setSelection(id ? { key, id } : null)}
                page={(n) => page(n)}
              />
              <Authorities locale={l} />
            </>
          ) : view === "trace" ? (
            <Trace locale={l} row={selected} go={() => goView("inputs")} />
          ) : view === "comparison" ? (
            <Comparison locale={l} data={data} baseline={baseline} />
          ) : view === "validation" ? (
            <Validation locale={l} data={data} />
          ) : (
            <Recognition locale={l} />
          )}
          <details className="b08-provenance">
            <summary>
              {t(
                l,
                "Source query and coverage evidence",
                "원천 조회·수집 범위 근거",
                "أدلة استعلام المصدر والتغطية",
              )}
            </summary>
            <p>
              {t(
                l,
                "These coverage declarations concern all authorized collected sources, not completeness of this provider/month. They are operator assertions, not financial approval.",
                "수집 범위 선언은 전체 허용 수집 원천 기준이며 현재 공급사·월의 완전성을 뜻하지 않습니다. 운영자 선언이며 재무 승인이 아닙니다.",
                "إقرارات التغطية تخص كل المصادر المجمّعة المصرّح بها، لا اكتمال هذا المزوّد أو الشهر. هي إقرارات مشغّل وليست اعتماداً مالياً.",
              )}
            </p>
            <dl className="b08-facts">
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
                    "Collected source reference",
                    "수집 원천 기준 ID",
                    "معرّف مرجع المصدر المجمّع",
                  ),
                  data.source_snapshot,
                ],
                [
                  t(
                    l,
                    "Query reference",
                    "조회 기준 ID",
                    "معرّف مرجع الاستعلام",
                  ),
                  data.snapshot,
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
