import { useEffect, useRef, useState } from "react";
import SlowReadHint from "../../../../shared/SlowReadHint";
import type { Locale } from "../../../../shared/types";
import { digest } from "../../../../shared/review-store";
import { load, query, ServiceError, type Data, type FieldState } from "./api";
import { t, stateLabel, costLabel, errorLabel } from "./copy";
import { range } from "./values";
import GroupTable, { MeasureValue } from "./GroupTable";
import GroupDetail from "./GroupDetail";
import AccountScope from "./AccountScope";
import "./screen.css";
type State = {
  key: string;
  queryHash?: string;
  data?: Data;
  error?: string;
  requestId?: string;
};
const items = [
  "B06:scope:authorized-source",
  "B06:data:commitment-groups",
  "B06:data:source-status",
  "B06:data:benefit-boundary",
  "B06:data:reconciliation-evidence",
  "B06:action:inspect-detail",
  "B06:state:async-unavailable",
  "B06:state:value-missing-zero",
];
function requestKey(params: URLSearchParams) {
  return JSON.stringify([
    query(params).toString(),
    [...params]
      .filter(
        ([k]) =>
          !k.startsWith("focus-") &&
          ![
            "view",
            "lang",
            "commitment-group",
            "commitment-row-after",
          ].includes(k),
      )
      .sort(),
  ]);
}
function Unidentified({ locale: l, data: d }: { locale: Locale; data: Data }) {
  return (
    <details className="b06-unidentified">
      <summary>
        {t(
          l,
          "Unidentified commitment-ID evidence",
          "약정 ID 미식별 근거",
          "أدلة معرّف الالتزام غير المحدد",
        )}{" "}
        <bdi dir="ltr">({d.unidentified_rows})</bdi>
      </summary>
      <p className="b06-meta">
        {t(
          l,
          "These rows remain source evidence, not one fabricated commitment group. Missing values are not zero.",
          "이 행은 원천 근거로 유지하며 임의 약정 그룹으로 만들지 않습니다. 누락 값은 0이 아닙니다.",
          "تظل هذه الصفوف أدلة مصدر ولا تُنشأ منها مجموعة التزام مفترضة. القيم المفقودة ليست صفراً.",
        )}
      </p>
      <dl className="b06-reasons">
        {Object.entries(d.unidentified_reasons).map(([k, n]) => (
          <div key={k}>
            <dt>{stateLabel(l, k as FieldState)}</dt>
            <dd>
              <bdi dir="ltr">{n}</bdi>
            </dd>
          </div>
        ))}
      </dl>
    </details>
  );
}
function EvidenceScope({ locale: l, data: d }: { locale: Locale; data: Data }) {
  return (
    <section className="b06-reconciliation">
      <h2>
        {t(l, "Source evidence review", "원천 근거 검토", "مراجعة أدلة المصدر")}
      </h2>
      <p>
        {t(
          l,
          "Evidence review is not financial reconciliation complete. A monthly or filtered selection cannot verify conservation over the whole commitment term.",
          "근거 검토는 재무 대사 완료가 아닙니다. 월별·필터 조회로 약정 전체 기간의 보존을 확인할 수 없습니다.",
          "مراجعة الأدلة ليست إكمالاً للتسوية المالية. لا يتحقق الاختيار الشهري أو المرشّح من الحفظ طوال مدة الالتزام.",
        )}
      </p>
      <dl className="b06-evidence">
        <div>
          <dt>
            {t(
              l,
              "Declared finalized / observed slices — all collected sources",
              "전체 수집 원천의 확정 선언 / 관측 구간",
              "الشرائح المعلنة نهائية / المرصودة — كل المصادر المجمّعة",
            )}
          </dt>
          <dd>
            <bdi dir="ltr">
              {d.coverage.finalized_slices} / {d.coverage.slices}
            </bdi>
          </dd>
        </div>
        <div>
          <dt>
            {t(
              l,
              "Earliest declared complete-through — all collected sources",
              "전체 수집 원천의 최소 완료 선언 시각",
              "أقدم تاريخ اكتمال معلن — كل المصادر المجمّعة",
            )}
          </dt>
          <dd>
            {d.coverage.complete_through ? (
              <bdi dir="ltr" className="b06-date">
                {d.coverage.complete_through}
              </bdi>
            ) : (
              t(
                l,
                "No complete-through declaration",
                "완료 시각 선언 없음",
                "لا توجد قيمة نهاية مؤكدة",
              )
            )}
          </dd>
        </div>
        <div>
          <dt>{t(l, "Coverage declaration", "범위 선언", "إقرار التغطية")}</dt>
          <dd>
            {d.coverage.expectations === "operator-asserted-unverified"
              ? t(
                  l,
                  "Operator declaration; completeness unverified",
                  "운영자 선언 · 완전성 미검증",
                  "إقرار المشغّل؛ لم يُتحقق من الاكتمال",
                )
              : t(
                  l,
                  "Declaration meaning unverified",
                  "선언 의미 미확인",
                  "معنى الإقرار غير متحقّق منه",
                )}
            <small>
              {t(
                l,
                "These declarations cover all collected sources, not the selected provider or period. They do not prove this selection is complete or establish commitment maturity.",
                "수집 원천 전체의 선언이며 선택한 공급사·기간의 완전성이나 약정 만기를 증명하지 않습니다.",
                "تشمل هذه الإقرارات كل المصادر المجمّعة، لا المزوّد أو الفترة المختارة. وهي لا تثبت اكتمال الاختيار أو استحقاق الالتزام.",
              )}
            </small>
            <details>
              <summary>
                {t(
                  l,
                  "Source declaration value",
                  "원천 선언 원문 값",
                  "قيمة الإقرار من المصدر",
                )}
              </summary>
              <bdi dir="ltr">{d.coverage.expectations}</bdi>
            </details>
          </dd>
        </div>
        <div>
          <dt>{t(l, "Source versions", "원천 버전", "إصدارات المصدر")}</dt>
          <dd className="b06-values">
            {d.versions.map((v) => (
              <bdi dir="ltr" key={v}>
                {v}
              </bdi>
            ))}
          </dd>
        </div>
        <div>
          <dt>{t(l, "Period basis", "기간 기준", "أساس الفترة")}</dt>
          <dd>
            {t(
              l,
              "Charge-start month in tenant timezone; no proration",
              "테넌트 시간대의 과금 시작 월 · 비례 배분 없음",
              "شهر بدء الرسوم في منطقة المستأجر الزمنية؛ دون توزيع نسبي",
            )}
          </dd>
        </div>
        <div>
          <dt>
            {t(
              l,
              "Pinned version mapping reference",
              "고정 버전 매핑 참조",
              "مرجع ربط الإصدارات المثبّت",
            )}
          </dt>
          <dd>
            <bdi dir="ltr">{d.mapping_digest}</bdi>
          </dd>
        </div>
        <div>
          <dt>
            {t(
              l,
              "Collected-data reference ID / Query reference ID",
              "수집 데이터 기준 ID / 조회 기준 ID",
              "معرّف مرجع البيانات المجمّعة / معرّف مرجع الاستعلام",
            )}
          </dt>
          <dd>
            <bdi dir="ltr">{d.source_snapshot}</bdi>
            <br />
            <bdi dir="ltr">{d.snapshot}</bdi>
          </dd>
        </div>
      </dl>
      <Unidentified locale={l} data={d} />
    </section>
  );
}
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
  canDiscard?: () => boolean;
}) {
  const q = query(params),
    key = requestKey(params),
    [state, setState] = useState<State>({ key: "" }),
    [retry, setRetry] = useState(0),
    [attempts, setAttempts] = useState(0),
    [search, setSearch] = useState(q.get("q") ?? "");
  const sequence = useRef(0),
    returnFocus = useRef<string | null>(null),
    status = useRef<HTMLParagraphElement>(null),
    focusStatus = useRef(false),
    current = state.key === key ? state : undefined,
    data = current?.data;
  useEffect(() => onDirty(false), [onDirty]);
  useEffect(() => {
    const controller = new AbortController(),
      g = ++sequence.current;
    setState({ key });
    setSearch(q.get("q") ?? "");
    Promise.all([load(q, controller.signal), digest(key)])
      .then(([data, queryHash]) => {
        if (controller.signal.aborted || sequence.current !== g) return;
        setState({ key, data, queryHash });
      })
      .catch((error) => {
        if (!controller.signal.aborted && sequence.current === g)
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
  const statusName = current?.error ? "error" : data ? "loaded" : "loading";
  const review = JSON.stringify({
    phase: "commitment-evidence",
    status: statusName,
    view,
    requestKey: key,
    query: current?.queryHash ?? null,
    scope: data?.snapshot ?? "none",
    filter: current?.queryHash ?? "none",
    page: data?.after ?? 0,
    selected: data?.selected ? [data.selected.group] : [],
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
      requestAnimationFrame(() => {
        const button = [
          ...document.querySelectorAll<HTMLButtonElement>(
            "button[data-b06-group]",
          ),
        ].find((b) => b.dataset.b06Group === id);
        button?.focus();
      });
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
      location.hash = `#/commitments?${next}`;
  }
  function scope(field: string, value: string) {
    const next = new URLSearchParams(params);
    next.set("focus-" + field, value);
    for (const k of [
      "commitment-group",
      "commitment-row-after",
      "focus-after",
      "focus-account_after",
      "focus-snapshot",
    ])
      next.delete(k);
    returnFocus.current = null;
    focusStatus.current = true;
    go(next);
  }
  function refresh() {
    const next = new URLSearchParams(params);
    for (const k of [
      "commitment-group",
      "commitment-row-after",
      "focus-after",
      "focus-account_after",
      "focus-snapshot",
    ])
      next.delete(k);
    returnFocus.current = null;
    focusStatus.current = true;
    go(next);
  }
  function repeat() {
    setState({ key });
    setAttempts((n) => n + 1);
    setRetry((n) => n + 1);
  }
  function select(id: string) {
    if (
      !data ||
      !data.groups.some((g) => g.id === id) ||
      data.selected?.group === id
    )
      return;
    const next = new URLSearchParams(params);
    next.set("commitment-group", id);
    next.delete("commitment-row-after");
    next.set("focus-snapshot", data.snapshot);
    go(next);
  }
  function close() {
    if (!data?.selected) return;
    returnFocus.current = data.selected.group;
    const next = new URLSearchParams(params);
    next.delete("commitment-group");
    next.delete("commitment-row-after");
    next.set("focus-snapshot", data.snapshot);
    go(next);
  }
  function page(after: number, kind: "group" | "account" | "row") {
    if (!data) return;
    const next = new URLSearchParams(params);
    next.set("focus-snapshot", data.snapshot);
    if (kind === "row") {
      if (
        !data.selected ||
        !data.groups.some((g) => g.id === data.selected!.group)
      )
        return;
      next.set("commitment-row-after", String(after));
    } else {
      next.delete("commitment-group");
      next.delete("commitment-row-after");
      next.set(
        kind === "group" ? "focus-after" : "focus-account_after",
        String(after),
      );
      if (kind === "account") next.delete("focus-after");
      returnFocus.current = null;
      focusStatus.current = true;
    }
    go(next);
  }
  const denied =
      !!current?.error &&
      [
        "permission_denied",
        "actor_revoked",
        "scope_denied",
        "revoked",
        "forbidden",
      ].includes(current.error),
    stale = !!current?.error && /snapshot|stale|cursor/.test(current.error),
    selected = data?.selected
      ? data.groups.find((g) => g.id === data.selected!.group)
      : undefined;
  return (
    <div
      className="b06-screen"
      data-b06-state={statusName}
      data-b06-view={view}
      data-b06-query={current?.queryHash ?? data?.snapshot ?? ""}
      data-b06-selected-group={data?.selected?.group ?? ""}
      onKeyDown={(e) => {
        if (
          e.key !== "Escape" ||
          e.defaultPrevented ||
          !(e.target instanceof HTMLElement)
        )
          return;
        const detail = e.target.closest<HTMLDetailsElement>("details[open]");
        if (detail) {
          detail.open = false;
          detail.querySelector("summary")?.focus();
          e.preventDefault();
        }
      }}
    >
      <form
        className="b06-toolbar"
        onSubmit={(e) => {
          e.preventDefault();
          scope("q", search);
        }}
      >
        <label htmlFor="b06-source-provider">
          {t(l, "Collected source", "수집 원천", "المصدر المجمّع")}
          <select
            id="b06-source-provider"
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
            {Array.from(
              new Set([
                q.get("provider") ?? "",
                ...(data?.options.providers ?? []),
              ]),
            )
              .filter(Boolean)
              .map((p) => (
                <option key={p}>{p}</option>
              ))}
          </select>
        </label>
        <label htmlFor="b06-period">
          {t(l, "Charge-start month", "과금 시작 월", "شهر بدء الرسوم")}
          <select
            id="b06-period"
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
            {Array.from(
              new Set([
                q.get("period") ?? "",
                ...(data?.options.periods ?? []),
              ]),
            )
              .filter((p) => p && p !== "all")
              .map((p) => (
                <option key={p}>{p}</option>
              ))}
          </select>
        </label>
        <label htmlFor="b06-search">
          {t(l, "Account search", "계정 검색", "بحث الحسابات")}
          <input
            id="b06-search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            aria-describedby="b06-search-help"
          />
          <small id="b06-search-help">
            {t(
              l,
              "Searches the account list only",
              "계정 목록만 검색합니다",
              "يبحث في قائمة الحسابات فقط",
            )}
          </small>
        </label>
        <button type="submit">{t(l, "Search", "검색", "بحث")}</button>
        <button type="button" onClick={refresh}>
          {t(l, "New query", "새 조회", "استعلام جديد")}
        </button>
      </form>
      {current?.error ? (
        <section role="alert" className="b06-error">
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
          <p>{errorLabel(l, current.error)}</p>
          <p className="b06-meta">
            <bdi dir="ltr">
              {current.error} {current.requestId}
            </bdi>
          </p>
          {!denied && (
            <button onClick={stale ? refresh : repeat}>
              {stale
                ? t(l, "New query", "새 조회", "استعلام جديد")
                : t(
                    l,
                    "Retry same query",
                    "같은 조회 다시 시도",
                    "إعادة الاستعلام نفسه",
                  )}
            </button>
          )}
        </section>
      ) : !data ? (
        <p role="status">
          {t(
            l,
            "Checking the selected scope and authorization; loading commitment cost and benefit evidence…",
            "선택한 범위와 권한을 확인하고 약정 비용·혜택 근거를 불러오는 중입니다…",
            "جارٍ التحقق من النطاق والتفويض وتحميل أدلة تكاليف الالتزامات ومنافعها…",
          )}
          <SlowReadHint locale={l} />
        </p>
      ) : (
        <>
          <AccountScope
            locale={l}
            data={data}
            select={(id) => scope("account", id)}
            page={(n) => page(n, "account")}
          />
          <div className="b06-boundary">
            <p className="b06-meta">
              {t(
                l,
                "Collected source-cost evidence, not selling amounts, savings or financial reconciliation. Observed ID groups are not a count of legal contracts.",
                "수집 원천 비용 근거이며 판매 금액·절감액·재무 대사 결과가 아닙니다. 관측 ID 그룹은 법적 계약 수가 아닙니다.",
                "أدلة تكلفة مصدر مجمّعة وليست مبالغ بيع أو وفورات أو تسوية مالية. مجموعات المعرّفات المرصودة ليست عدداً للعقود القانونية.",
              )}
            </p>
            <p className="b06-meta">
              {t(l, "Observed at", "원천 조회 시각", "وقت الرصد")}:{" "}
              <bdi dir="ltr" className="b06-date">
                {data.observed_at}
              </bdi>
            </p>
            {data.coverage.finalized_slices < data.coverage.slices && (
              <p className="b06-meta">
                {t(
                  l,
                  "Only part of the observed source slices is finalized. Completeness remains unverified.",
                  "관측 구간 중 일부만 확정되었습니다. 전체 완전성은 미검증입니다.",
                  "أُكد جزء فقط من شرائح المصدر المرصودة. لم يُتحقق من الاكتمال الكلي.",
                )}
              </p>
            )}
          </div>
          <div className="b06-counts">
            {[
              [t(l, "Source rows", "원천 행", "صفوف المصدر"), data.row_count],
              [
                t(
                  l,
                  "Identified commitment-ID rows",
                  "약정 ID 식별 행",
                  "صفوف ذات معرّف التزام",
                ),
                data.identified_rows,
              ],
              [
                t(
                  l,
                  "Unidentified commitment-ID rows",
                  "약정 ID 미식별 행",
                  "صفوف دون معرّف التزام محدد",
                ),
                data.unidentified_rows,
              ],
              [
                t(
                  l,
                  "Observed commitment ID groups",
                  "관측된 약정 ID 그룹",
                  "مجموعات معرّفات الالتزام المرصودة",
                ),
                data.group_count,
              ],
            ].map(([label, n]) => (
              <dl key={label}>
                <dt>{label}</dt>
                <dd>
                  <bdi dir="ltr">{n}</bdi>
                </dd>
              </dl>
            ))}
          </div>
          <section className="b06-cost-summary">
            <h2>
              {t(
                l,
                "Source costs across the full selection",
                "전체 조회 범위의 원천 비용",
                "تكاليف المصدر في الاختيار الكامل",
              )}
            </h2>
            <p className="b06-meta">
              {t(
                l,
                "Server totals before paging, separated by currency and ID evidence. Billed and effective costs are different bases; no difference or savings is inferred.",
                "페이지 분할 전 서버 합계이며 통화·ID 근거별로 구분합니다. 청구·실효 비용은 서로 다른 기준이며 차액·절감액을 추정하지 않습니다.",
                "إجماليات الخادم قبل التقسيم، منفصلة حسب العملة وأدلة المعرّف. التكلفة المفوترة والفعلية أساسان مختلفان؛ لا يُستنتج فرق أو توفير.",
              )}
            </p>
            <div className="b06-summary-table">
              <table>
                <thead>
                  <tr>
                    <th>
                      {t(
                        l,
                        "Source ID evidence",
                        "원천 ID 근거",
                        "أدلة معرّف المصدر",
                      )}
                    </th>
                    <th>{t(l, "Currency", "통화", "العملة")}</th>
                    <th>{t(l, "Source rows", "원천 행", "صفوف المصدر")}</th>
                    <th>{costLabel(l, "billed_cost")}</th>
                    <th>{costLabel(l, "effective_cost")}</th>
                  </tr>
                </thead>
                <tbody>
                  {(
                    [
                      [
                        data.identified_amounts,
                        t(
                          l,
                          "Identified ID rows",
                          "ID 식별 행",
                          "صفوف بمعرّف محدد",
                        ),
                      ],
                      [
                        data.unidentified_amounts,
                        t(
                          l,
                          "Unidentified ID rows",
                          "ID 미식별 행",
                          "صفوف بمعرّف غير محدد",
                        ),
                      ],
                    ] as const
                  ).flatMap(([amounts, label]) =>
                    amounts.map((a) => (
                      <tr key={`${label}-${a.currency}`}>
                        <th scope="row">{label}</th>
                        <td data-label={t(l, "Currency", "통화", "العملة")}>
                          <bdi dir="ltr">{a.currency}</bdi>
                        </td>
                        <td
                          data-label={t(
                            l,
                            "Source rows",
                            "원천 행",
                            "صفوف المصدر",
                          )}
                        >
                          <bdi dir="ltr">{a.rows}</bdi>
                        </td>
                        {(["billed_cost", "effective_cost"] as const).map(
                          (k) => (
                            <td key={k} data-label={costLabel(l, k)}>
                              <MeasureValue
                                locale={l}
                                value={a[k]}
                                currency={a.currency}
                                rows={a.rows}
                              />
                            </td>
                          ),
                        )}
                      </tr>
                    )),
                  )}
                </tbody>
              </table>
            </div>
          </section>
          {view === "reconciliation" ? (
            <EvidenceScope locale={l} data={data} />
          ) : (
            <Unidentified locale={l} data={data} />
          )}
          <p
            role="status"
            aria-live="polite"
            className="b06-meta"
            ref={status}
            tabIndex={-1}
          >
            {t(l, "Groups displayed", "표시 그룹", "المجموعات المعروضة")}:{" "}
            <bdi dir="ltr">
              {range(data.after, data.groups.length, data.group_count)}
            </bdi>
            {data.selected && (
              <>
                {" "}
                · {t(l, "Detail rows", "상세 행", "صفوف التفاصيل")}:{" "}
                <bdi dir="ltr">
                  {range(
                    data.selected.after,
                    data.selected.rows.length,
                    data.selected.total_rows,
                  )}
                </bdi>
              </>
            )}
          </p>
          <div className={`b06-workspace${selected ? " has-detail" : ""}`}>
            <GroupTable
              locale={l}
              data={data}
              view={view}
              select={select}
              page={(n) => page(n, "group")}
            />
            {selected && data.selected ? (
              <GroupDetail
                locale={l}
                group={selected}
                detail={data.selected}
                view={view}
                close={close}
                page={(n) => page(n, "row")}
              />
            ) : (
              <section className="b06-detail-placeholder">
                <h2>
                  {t(
                    l,
                    "Inspect source evidence",
                    "원천 근거 상세 보기",
                    "فحص أدلة المصدر",
                  )}
                </h2>
                <p>
                  {t(
                    l,
                    "Choose an observed group to inspect its source fields, status classifications and charge records.",
                    "관측 그룹을 선택하여 원천 필드·상태 분류·청구 행을 확인하세요.",
                    "اختر مجموعة مرصودة لفحص حقول المصدر وتصنيفات الحالة وسجلات الرسوم.",
                  )}
                </p>
                {view === "benefits" && (
                  <p className="b06-meta">
                    {t(
                      l,
                      "Provider evidence can be inspected. Internal allocation, customer discounts and unused-cost ownership are unconnected.",
                      "공급사 근거를 확인할 수 있습니다. 내부 배분·고객 할인·미사용 비용 소유 관계는 미연결입니다.",
                      "يمكن فحص أدلة المزوّد. التوزيع الداخلي وخصومات العملاء وملكية التكلفة غير المستخدمة غير متصلة.",
                    )}
                  </p>
                )}
              </section>
            )}
          </div>
        </>
      )}
    </div>
  );
}
