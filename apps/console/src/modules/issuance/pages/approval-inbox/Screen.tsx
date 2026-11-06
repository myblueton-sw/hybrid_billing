import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import {
  query,
  load,
  ServiceError,
  type Data,
  type View,
  type ReadResult,
} from "./api";
import { t, kinds, kindLabel, tabLabel } from "./copy";
import Inbox from "./Inbox";
import "./screen.css";
type State = {
  key: string;
  result?: ReadResult;
  error?: string;
  requestId?: string;
};
const items = [
  "B09:scope:request-filter",
  "B09:data:inbox",
  "B09:action:inspect-request",
  "B09:data:result-card",
  "B09:data:evidence-boundary",
  "B09:data:decision-history",
  "B09:data:delegation-boundary",
  "B09:data:recognition-boundary",
  "B09:state:async-unconnected",
];
const requestKey = (p: URLSearchParams, view: View) =>
  JSON.stringify([
    query(p, view).toString(),
    [...p]
      .filter(
        ([k]) => !k.startsWith("approval-") && !["view", "lang"].includes(k),
      )
      .sort(),
  ]);
export default function Screen({
  locale: l,
  params,
  view: rawView,
  onState,
  onDirty,
}: {
  locale: Locale;
  params: URLSearchParams;
  view: string;
  onState: (s: string, i: string[]) => void;
  onDirty: (d: boolean) => void;
}) {
  const view = rawView as View,
    q = query(params, view),
    key = requestKey(params, view),
    [state, setState] = useState<State>({ key: "" }),
    [retry, setRetry] = useState(0),
    [attempts, setAttempts] = useState(0),
    [search, setSearch] = useState(q.get("q") ?? ""),
    [selection, setSelection] = useState<{ key: string; id: string } | null>(
      null,
    ),
    seq = useRef(0),
    status = useRef<HTMLParagraphElement>(null),
    focusStatus = useRef(false),
    current = state.key === key ? state : undefined,
    data: Data | undefined =
      current?.result?.kind === "loaded" ? current.result.data : undefined,
    selected =
      data && selection?.key === key
        ? data.records.find((r) => r.id === selection.id)
        : undefined;
  useEffect(() => onDirty(false), [onDirty]);
  useEffect(() => {
    const c = new AbortController(),
      n = ++seq.current;
    setState({ key });
    setSelection(null);
    setSearch(q.get("q") ?? "");
    load(q, c.signal)
      .then((result) => {
        if (!c.signal.aborted && n === seq.current) setState({ key, result });
      })
      .catch((e) => {
        if (!c.signal.aborted && n === seq.current)
          setState({
            key,
            error: e instanceof ServiceError ? e.code : "service_unavailable",
            requestId: e instanceof ServiceError ? e.requestId : undefined,
          });
      });
    return () => c.abort();
  }, [key, retry]);
  const phase = current?.error
      ? "error"
      : data
        ? "loaded"
        : current?.result?.kind === "unconnected"
          ? "unconnected"
          : "loading",
    review = JSON.stringify({
      phase: "approval-read-unbound",
      variant: `${view}:${phase}:${data?.source_mode ?? "unbound"}`,
      requestKey: key,
      view,
      status: phase,
      scope: data?.auth_scope ?? "producer-unbound",
      snapshot: data?.snapshot ?? null,
      permission: data?.permission_revision ?? null,
      data: data ?? null,
      selected: selected ? [selected.id] : [],
      detail: !!selected,
      error: current?.error ?? null,
      requestId: current?.requestId ?? null,
      attempts,
    });
  useEffect(() => onState(review, items), [review, onState]);
  useEffect(() => {
    if (focusStatus.current && phase !== "loading") {
      focusStatus.current = false;
      status.current?.focus();
    }
  }, [phase, data]);
  function go(next: URLSearchParams) {
    setState({ key: "pending" });
    setSelection(null);
    setAttempts((n) => n + 1);
    focusStatus.current = true;
    if (requestKey(next, view) === key) {
      setState({ key });
      setRetry((n) => n + 1);
    }
    if (next.toString() !== params.toString())
      location.hash = `#/approvals?${next}`;
  }
  function scope(k: string, v: string) {
    const next = new URLSearchParams(params);
    next.set("approval-" + k, v);
    for (const k of [
      "approval-cursor",
      "approval-snapshot",
      "approval-permission_revision",
    ])
      next.delete(k);
    go(next);
  }
  function refresh() {
    const next = new URLSearchParams(params);
    for (const k of [
      "approval-cursor",
      "approval-snapshot",
      "approval-permission_revision",
    ])
      next.delete(k);
    go(next);
  }
  function page(cursor: string | null) {
    if (!data) return;
    const next = new URLSearchParams(params);
    if (cursor === null) next.delete("approval-cursor");
    else next.set("approval-cursor", cursor);
    next.set("approval-snapshot", data.snapshot);
    next.set("approval-permission_revision", data.permission_revision);
    go(next);
  }
  const denied =
      !!current?.error &&
      /permission|denied|revoked|forbidden/.test(current.error),
    stale =
      !!current?.error &&
      /stale|snapshot|permission_revision|cursor/.test(current.error);
  const errorCopy =
    current?.error === "producer_unbound"
      ? t(
          l,
          "This producer has not been bound or reviewed. Live request data was not displayed.",
          "연결 검토가 완료되지 않은 서비스입니다. 실제 요청 데이터를 표시하지 않았습니다.",
          "لم تُربط هذه الخدمة أو تُراجع. لم تُعرض بيانات طلبات حقيقية.",
        )
      : denied
        ? t(
            l,
            "Request access is not confirmed. Previous details were removed. Check your current access, then start a new query.",
            "요청 열람 권한이 확인되지 않았습니다. 이전 상세를 제거했습니다. 현재 권한을 확인한 뒤 새로 조회하세요.",
            "صلاحية عرض الطلبات غير مؤكدة. أُزيلت التفاصيل السابقة. تحقق من صلاحيتك الحالية ثم ابدأ استعلاماً جديداً.",
          )
        : stale
          ? t(
              l,
              "The read or permission reference changed. Start a new query before selecting a request.",
              "조회·권한 기준이 바뀌었습니다. 새로 조회한 뒤 요청을 선택하세요.",
              "تغيّر مرجع القراءة أو الصلاحيات. ابدأ استعلاماً جديداً قبل اختيار الطلب.",
            )
          : t(
              l,
              "The request list could not be read safely. Retry or start a new query.",
              "요청 목록을 안전하게 읽지 못했습니다. 다시 시도하거나 새로 조회하세요.",
              "تعذرت قراءة قائمة الطلبات بأمان. أعد المحاولة أو ابدأ استعلاماً جديداً.",
            );
  return (
    <div
      className="b09-screen"
      data-b09-state={phase}
      data-b09-view={view}
      data-b09-selected={selected?.id ?? ""}
      data-b09-mode={data?.source_mode ?? "unbound"}
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
          } else if (selected) {
            setSelection(null);
            e.preventDefault();
          }
        }
      }}
    >
      <details className="b09-help">
        <summary>
          {t(l, "About this screen", "이 화면 안내", "حول هذه الشاشة")}
        </summary>
        <p>
          {t(
            l,
            "Read request targets, amounts and source evidence separately from executing decisions. This screen cannot approve, reject, hand off, delegate or post. Isolated synthetic examples are always labelled.",
            "요청 대상·금액·원천 근거의 조회와 업무 결정 실행을 구분합니다. 이 화면은 승인·반려·인계·위임·전기를 실행하지 않습니다. 합성 검토 예시는 항상 별도로 표시합니다.",
            "تُفصل قراءة أهداف الطلبات ومبالغها وأدلة المصدر عن تنفيذ القرارات. لا تنفّذ هذه الشاشة الاعتماد أو الرفض أو الإحالة أو التفويض أو الترحيل. تُميّز الأمثلة التجريبية دائماً.",
          )}
        </p>
      </details>
      {phase !== "unconnected" && view !== "delegation" && (
        <form
          className="b09-toolbar"
          onSubmit={(e) => {
            e.preventDefault();
            scope("q", search);
          }}
        >
          {view === "recognition" ? (
            <div className="b09-fixed-kind">
              <span>{t(l, "Approval type", "승인 유형", "نوع الاعتماد")}</span>
              <p>{kindLabel(l, "monthly_recognition")}</p>
            </div>
          ) : (
            <label htmlFor="b09-kind">
              {t(l, "Approval type", "승인 유형", "نوع الاعتماد")}
              <select
                id="b09-kind"
                value={q.get("kind") ?? ""}
                onChange={(e) => scope("kind", e.target.value)}
              >
                <option value="">
                  {t(
                    l,
                    "All request types",
                    "전체 요청 유형",
                    "جميع أنواع الطلبات",
                  )}
                </option>
                {kinds.map((k) => (
                  <option key={k} value={k}>
                    {kindLabel(l, k)}
                  </option>
                ))}
              </select>
            </label>
          )}
          <label htmlFor="b09-search">
            {t(l, "Request search", "요청 검색", "بحث الطلبات")}
            <input
              id="b09-search"
              value={search}
              maxLength={128}
              onChange={(e) => setSearch(e.target.value)}
              aria-describedby="b09-search-help"
            />
            <small id="b09-search-help">
              {t(
                l,
                "Request reference and target label",
                "요청 참조·대상 이름",
                "مرجع الطلب واسم الهدف",
              )}
            </small>
          </label>
          <button type="submit">{t(l, "Search", "검색", "بحث")}</button>
          <button type="button" onClick={refresh}>
            {t(l, "New query", "새 조회", "استعلام جديد")}
          </button>
        </form>
      )}
      <p ref={status} tabIndex={-1} role="status" className="b09-meta">
        {phase === "loading"
          ? t(
              l,
              "Reading requests…",
              "요청을 불러오는 중입니다…",
              "جارٍ قراءة الطلبات…",
            )
          : phase === "unconnected"
            ? t(
                l,
                "Request count is unconfirmed",
                "요청 건수 미확인",
                "عدد الطلبات غير مؤكد",
              )
            : phase === "error"
              ? t(
                  l,
                  "Request evidence unavailable",
                  "요청 근거 조회 불가",
                  "أدلة الطلبات غير متاحة",
                )
              : t(
                  l,
                  "Read-only request evidence",
                  "조회 전용 요청 근거",
                  "أدلة طلبات للقراءة فقط",
                )}
        {data && (
          <>
            {" "}
            <bdi className="b09-count" dir="ltr">
              {data.observed_at}
            </bdi>
          </>
        )}
      </p>
      {phase === "loading" ? (
        <p>
          {t(
            l,
            "Checking this view and its read response.",
            "현재 보기의 조회 응답을 확인하고 있습니다.",
            "جارٍ التحقق من استجابة القراءة لهذا العرض.",
          )}
        </p>
      ) : current?.error ? (
        <section role="alert" className="b09-error">
          <h2>
            {t(
              l,
              "Request list unavailable",
              "요청 목록 조회 불가",
              "قائمة الطلبات غير متاحة",
            )}
          </h2>
          <p>{errorCopy}</p>
          <div className="b09-actions">
            <button id="b09-new-query" onClick={refresh}>
              {t(l, "New query", "새 조회", "استعلام جديد")}
            </button>
            {!denied && !stale && current.error !== "producer_unbound" && (
              <button
                id="b09-retry"
                onClick={() => {
                  setState({ key });
                  setSelection(null);
                  setRetry((n) => n + 1);
                  setAttempts((n) => n + 1);
                }}
              >
                {t(l, "Retry", "다시 시도", "إعادة المحاولة")}
              </button>
            )}
          </div>
          <details>
            <summary>
              {t(l, "Technical reference", "기술 참조", "المرجع التقني")}
            </summary>
            <p className="b09-meta">
              <bdi dir="ltr">
                {current.error} {current.requestId}
              </bdi>
            </p>
          </details>
        </section>
      ) : phase === "unconnected" ? (
        <section className="b09-unconnected">
          <h2>
            {view === "delegation"
              ? t(
                  l,
                  "Delegation service is not connected",
                  "위임·대리 조회 서비스 미연결",
                  "خدمة التفويض والإنابة غير متصلة",
                )
              : t(
                  l,
                  "Approval request service is not connected",
                  "승인 요청 서비스 미연결",
                  "خدمة طلبات الاعتماد غير متصلة",
                )}
          </h2>
          <p>
            {view === "delegation"
              ? t(
                  l,
                  "An approval request or AP role does not establish delegation authority. No grant, substitute scope, ceiling or expiry has been verified.",
                  "승인 요청이나 AP 역할은 위임 권한이 아닙니다. 위임·대리 범위·한도·만료를 확인한 근거가 없습니다.",
                  "طلب الاعتماد أو دور AP لا يثبت صلاحية التفويض. لم تُتحقق منحة أو نطاق إنابة أو حد أو انتهاء.",
                )
              : t(
                  l,
                  "Requests cannot be shown because the service that provides them is not connected. This does not mean there are zero pending requests.",
                  "요청을 제공하는 서비스가 연결되지 않아 목록을 표시할 수 없습니다. 승인 대기 요청이 0건이라는 뜻은 아닙니다.",
                  "لا يمكن عرض الطلبات لأن خدمتها غير متصلة. لا يعني ذلك عدم وجود طلبات معلّقة.",
                )}
          </p>
          {view === "recognition" && (
            <p>
              {t(
                l,
                "Monthly expense recognition requests are separate from customer invoices, receivables, posting and collection. No monthly batches or reconciled amount are available.",
                "월별 비용 인식 요청은 고객 청구서·채권·전기·수납과 별개입니다. 월별 배치·대사 금액은 연결되지 않았습니다.",
                "طلبات الإثبات الشهري للمصروفات منفصلة عن فواتير العملاء والذمم والترحيل والتحصيل. لا تتوفر دفعات شهرية أو مبالغ مطابقة.",
              )}
            </p>
          )}
          <div className="b09-actions">
            {view !== "delegation" && (
              <button id="b09-connect-retry" onClick={refresh}>
                {t(l, "Try again", "다시 조회", "المحاولة مجدداً")}
              </button>
            )}
            <a href={`#/billing-runs?lang=${l}`}>
              {t(
                l,
                "Open billing input review",
                "청구 입력 검토로 이동",
                "فتح مراجعة مدخلات الفوترة",
              )}
            </a>
          </div>
        </section>
      ) : data ? (
        <Inbox
          locale={l}
          data={data}
          selected={selected}
          select={(id) => setSelection(id ? { key, id } : null)}
          page={page}
        />
      ) : null}
    </div>
  );
}
