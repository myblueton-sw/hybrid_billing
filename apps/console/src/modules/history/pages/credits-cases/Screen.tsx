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
import { t, environment } from "./copy";
import Inbox from "./Inbox";
import "./screen.css";
type State = {
  key: string;
  result?: ReadResult;
  error?: string;
  requestId?: string;
};
const items = [
  "B11:scope:read-query",
  "B11:data:credits",
  "B11:data:reservations",
  "B11:data:cases",
  "B11:data:consent",
  "B11:data:adjustments",
  "B11:data:payment-terms",
  "B11:data:holds",
  "B11:action:inspect",
  "B11:state:async",
];
const requestKey = (p: URLSearchParams, v: View) =>
  JSON.stringify([
    query(p, v).toString(),
    [...p]
      .filter(
        ([k]) => !k.startsWith("credit-") && !["view", "lang"].includes(k),
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
    key = requestKey(params, view);
  const [state, setState] = useState<State>({ key: "" }),
    [retry, setRetry] = useState(0),
    [attempts, setAttempts] = useState(0),
    [search, setSearch] = useState(q.get("q") ?? ""),
    [selection, setSelection] = useState<{ key: string; id: string } | null>(
      null,
    );
  const seq = useRef(0),
    status = useRef<HTMLParagraphElement>(null),
    focusStatus = useRef(false),
    current = state.key === key ? state : undefined;
  const data: Data | undefined =
      current?.result?.kind === "loaded" ? current.result.data : undefined,
    selected =
      data && selection?.key === key
        ? data.records.find((r) => r.id === selection.id)
        : undefined;
  const phase = current?.error
    ? "error"
    : data
      ? "loaded"
      : current?.result?.kind === "unconnected"
        ? "unconnected"
        : "loading";
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
  const review = JSON.stringify({
    phase: "credit-read-unbound",
    variant: `${view}:${phase}:${data?.source_mode ?? "unbound"}`,
    requestKey: key,
    view,
    status: phase,
    scope: data?.auth_scope ?? "producer-unbound",
    snapshot: data?.snapshot ?? null,
    permission: data?.permission_revision ?? null,
    data: data ?? null,
    selected: selected ? [selected.id] : [],
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
      location.hash = `#/credits-cases?${next}`;
  }
  function clearPins(p: URLSearchParams) {
    for (const k of ["cursor", "snapshot", "permission_revision"])
      p.delete("credit-" + k);
    return p;
  }
  function scope(k: string, v: string) {
    const next = clearPins(new URLSearchParams(params));
    next.set("credit-" + k, v);
    go(next);
  }
  function refresh() {
    go(clearPins(new URLSearchParams(params)));
  }
  function page(cursor: string | null) {
    if (!data) return;
    const next = new URLSearchParams(params);
    if (cursor === null) next.delete("credit-cursor");
    else next.set("credit-cursor", cursor);
    next.set("credit-snapshot", data.snapshot);
    next.set("credit-permission_revision", data.permission_revision);
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
          "This producer has not been bound or reviewed. Live billing or inquiry records were not displayed.",
          "연결 검토가 완료되지 않은 서비스입니다. 실제 청구 크레딧·문의 기록을 표시하지 않았습니다.",
          "لم تُربط هذه الخدمة أو تُراجع. لم تُعرض سجلات أرصدة واستفسارات حقيقية.",
        )
      : denied
        ? t(
            l,
            "Read access is not confirmed. Previous records and details were removed. Check access, then start a new query.",
            "열람 권한이 확인되지 않았습니다. 이전 기록과 상세를 제거했습니다. 현재 권한을 확인한 뒤 새로 조회하세요.",
            "صلاحية العرض غير مؤكدة. أُزيلت السجلات والتفاصيل السابقة. تحقق من صلاحيتك ثم ابدأ استعلاماً جديداً.",
          )
        : stale
          ? t(
              l,
              "The read or permission reference changed. Start a new query before inspecting a record.",
              "조회·권한 기준이 바뀌었습니다. 새로 조회한 뒤 크레딧·문의 기록을 확인하세요.",
              "تغيّر مرجع القراءة أو الصلاحيات. ابدأ استعلاماً جديداً قبل فحص السجل.",
            )
          : t(
              l,
              "Credit and inquiry evidence could not be read safely. Retry or start a new query.",
              "크레딧·문의 근거을 안전하게 읽지 못했습니다. 다시 시도하거나 새로 조회하세요.",
              "تعذرت قراءة سجل الأرصدة والاستفسارات بأمان. أعد المحاولة أو ابدأ استعلاماً جديداً.",
            );
  return (
    <section
      className="b11-screen"
      data-b11-state={phase}
      data-b11-view={view}
      data-b11-selected={selected?.id ?? ""}
      data-b11-mode={data?.source_mode ?? "unbound"}
      aria-busy={phase === "loading"}
      onKeyDown={(e) => {
        if (e.key !== "Escape" || e.defaultPrevented) return;
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
      }}
    >
      <details className="b11-help">
        <summary>
          {t(l, "About this screen", "이 화면 안내", "حول هذه الشاشة")}
        </summary>
        <p>
          {t(
            l,
            "Read credit, inquiry and payment evidence separately. This screen cannot create cases, send replies, apply credits, change due dates, record receipts or release holds. An inquiry or hold is not a payment exemption.",
            "크레딧·문의·지급 근거를 구분해 조회합니다. 문의 등록·답변 전송·크레딧 적용·지급 기한 변경·수납 확정·보류 해제를 실행하지 않습니다. 문의나 보류는 납부 의무 면제가 아닙니다.",
            "اقرأ أدلة الأرصدة والاستفسارات والسداد بشكل منفصل. لا تنشئ الشاشة استفسارات ولا ترسل ردوداً ولا تطبق أرصدة ولا تغيّر المواعيد أو تسجل إيصالات أو ترفع إجراءات تعليق. الاستفسار أو التعليق ليس إعفاءً من السداد.",
          )}
        </p>
      </details>
      {phase !== "unconnected" && (
        <form
          className="b11-toolbar"
          onSubmit={(e) => {
            e.preventDefault();
            scope("q", search);
          }}
        >
          <label htmlFor="b11-search">
            {t(
              l,
              "Reference and label search",
              "청구 크레딧·문의 기록·대상 검색",
              "بحث سجل الأرصدة والاستفسارات والهدف",
            )}
            <input
              id="b11-search"
              value={search}
              maxLength={128}
              onChange={(e) => setSearch(e.target.value)}
              aria-describedby="b11-search-help"
            />
            <small id="b11-search-help">
              {t(
                l,
                "Record reference and label",
                "크레딧·문의 기록 참조·대상 이름",
                "مرجع سجل الأرصدة والاستفسارات واسم الهدف",
              )}
            </small>
          </label>
          <button type="submit">{t(l, "Search", "검색", "بحث")}</button>
          <button type="button" onClick={refresh}>
            {t(l, "New query", "새 조회", "استعلام جديد")}
          </button>
        </form>
      )}
      <p ref={status} tabIndex={-1} role="status" className="b11-meta">
        {phase === "loading"
          ? t(
              l,
              "Reading credit and inquiry records…",
              "크레딧·문의 기록을 불러오는 중입니다…",
              "جارٍ قراءة سجلات الأرصدة والاستفسارات…",
            )
          : phase === "unconnected"
            ? t(
                l,
                "Credit and inquiry record count is unconfirmed",
                "크레딧·문의 기록 건수 미확인",
                "عدد سجلات الأرصدة والاستفسارات غير مؤكد",
              )
            : phase === "error"
              ? t(
                  l,
                  "Source evidence unavailable",
                  "크레딧·문의 근거 조회 불가",
                  "أدلة الأرصدة والاستفسارات غير متاحة",
                )
              : t(
                  l,
                  "Read-only source evidence",
                  "조회 전용 크레딧·문의 근거",
                  "أدلة أرصدة واستفسارات للقراءة فقط",
                )}
        {data && (
          <>
            {" "}
            <bdi dir="ltr">{data.observed_at}</bdi>
          </>
        )}
      </p>
      {phase === "loading" ? (
        <p>
          {t(
            l,
            "Checking this view and its read response.",
            "현재 보기의 조회 응답을 확인하고 있습니다.",
            "جارٍ التحقق من هذا العرض واستجابة القراءة.",
          )}
        </p>
      ) : phase === "unconnected" ? (
        <section className="b11-unconnected">
          <h2>{environment(l)}</h2>
          <p>
            {t(
              l,
              "The service that provides credit and inquiry records is not connected. This does not mean there are zero records.",
              "크레딧·문의 기록을 제공하는 서비스가 연결되지 않아 목록을 표시할 수 없습니다. 크레딧·문의 기록이 0건이라는 뜻은 아닙니다.",
              "الخدمة التي توفر سجلات الأرصدة والاستفسارات غير متصلة. هذا لا يعني أن عدد السجلات صفر.",
            )}
          </p>
          <div className="b11-actions">
            <button onClick={refresh}>
              {t(l, "Retry", "다시 조회", "إعادة المحاولة")}
            </button>
            <a href={`#/approvals?lang=${l}`}>
              {t(l, "Approval inbox", "승인함으로 이동", "صندوق الاعتمادات")}
            </a>
            <a href={`#/claims?lang=${l}`}>
              {t(
                l,
                "Billing application history",
                "청구 적용 이력으로 이동",
                "سجل تطبيق الفوترة",
              )}
            </a>
          </div>
        </section>
      ) : phase === "error" ? (
        <section className="b11-error">
          <h2>
            {t(
              l,
              "Credit and inquiry evidence unavailable",
              "크레딧·문의 근거 조회 불가",
              "سجل الأرصدة والاستفسارات غير متاح",
            )}
          </h2>
          <p>{errorCopy}</p>
          <div className="b11-actions">
            {!denied && !stale && (
              <button
                onClick={() => {
                  setState({ key });
                  setSelection(null);
                  setAttempts((n) => n + 1);
                  setRetry((n) => n + 1);
                }}
              >
                {t(l, "Retry", "다시 시도", "إعادة المحاولة")}
              </button>
            )}
            <button onClick={refresh}>
              {t(l, "New query", "새 조회", "استعلام جديد")}
            </button>
          </div>
          <details>
            <summary>
              {t(l, "Technical reference", "기술 참조", "مرجع تقني")}
            </summary>
            <p>
              <bdi dir="ltr">{current?.error}</bdi>{" "}
              <bdi dir="ltr">{current?.requestId}</bdi>
            </p>
          </details>
        </section>
      ) : (
        data && (
          <Inbox
            locale={l}
            data={data}
            selected={selected}
            select={(id) => setSelection(id ? { key, id } : null)}
            page={page}
          />
        )
      )}
    </section>
  );
}
