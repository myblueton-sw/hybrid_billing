import { CreateDraft } from "./drafts/Create";
import { DraftOutcomes } from "./drafts/Outcomes";
import { useDrafts } from "./drafts/use-drafts";
import { decisionFamily } from "./decision-api";
import DraftScreen from "./drafts/Screen";
import ResultScreen from "./results/Screen";
import { useDocuments } from "./documents/use-documents";
import { Failure } from "./documents/Fields";
import { Attempts } from "./documents/Review";
import { useDecision } from "./use-decision";
import { DecisionResult } from "./DecisionPanel";
import { dc } from "./decision-copy";
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
  "B09:draft:create-entry",
  "B09:scope:request-filter",
  "B09:data:inbox",
  "B09:action:inspect-request",
  "B09:data:result-card",
  "B09:data:evidence-boundary",
  "B09:data:decision-history",
  "B09:data:delegation-boundary",
  "B09:data:recognition-boundary",
  "B09:state:async-unconnected",
  "B09:documents:scope",
  "B09:documents:history",
  "B09:documents:inspection",
  "B09:documents:permission-readiness",
  "B09:documents:explicit-fields",
  "B09:documents:validation-review",
  "B09:documents:governed-actions",
  "B09:documents:propose",
  "B09:documents:confirm",
  "B09:documents:revoke",
  "B09:documents:prepare",
  "B09:documents:attempt-results",
  "B09:documents:identical-retry",
  "B09:documents:owned-recovery",
  "B09:documents:reauthentication",
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
function ApprovalScreen({
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
          : "loading";
  const decision = useDecision(data, selected, JSON.stringify([key, data?.snapshot, data?.permission_revision]), phase === "loading" || phase === "error", () => reload(false));
  const documents = useDocuments(data, selected, JSON.stringify([key, data?.snapshot, data?.permission_revision]), phase === "loading" || phase === "error");
  const drafts=useDrafts(),[draftEditing,setDraftEditing]=useState(false);
  const finalSelected=selected?.status==="approved"&&decisionFamily(selected)==="erp_final"?selected:undefined;
  useEffect(()=>{if(finalSelected?.evidence.revision)void drafts.store?.final(finalSelected.id,finalSelected.evidence.revision);else drafts.store?.clearSelection();},[drafts.store,finalSelected?.id,finalSelected?.evidence.revision,key,data?.snapshot,data?.permission_revision]);
  function openDraft(id:string){const next=new URLSearchParams(params);next.set("view","drafts");next.set("draft-id",id);location.hash=`#/approvals?${next}`;}
  const documentBusy = !!documents.state.draft || !!documents.state.review || documents.unresolved;
  const decisionBusy = !!decision.state.action || Object.values(decision.state.entries).some(e => e.phase === "pending" || e.phase === "unknown");
  useEffect(() => onDirty(decisionBusy || documentBusy || draftEditing || drafts.unresolved), [decisionBusy, documentBusy, draftEditing, drafts.unresolved, onDirty]);
  const review = JSON.stringify({
      phase: "approval-read",
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
      drafts: {selected:finalSelected?.id,context: drafts.state.context.loading?"loading":drafts.state.context.error??drafts.state.context.value?.submit_action??"idle",entries:drafts.state.entries.map(e=>[e.intent.target,e.phase,e.error])},
      documents: { enabled: documents.enabled, context: documents.state.context.phase, history: [documents.state.documents.phase,documents.state.preparations.phase], detail: [documents.state.document.phase,documents.state.preparation.phase], review: documents.state.review?.action, error: documents.state.error, entries: documents.state.entries.map(e => [e.intent.target,e.phase,e.error]) },
      decision: { checking: decision.state.checking, action: decision.state.action, error: decision.state.error, prepared: !!decision.state.preparation, entries: Object.values(decision.state.entries).map(e => [e.submission.target, e.phase, e.error]) },
    });
  useEffect(() => onState(review, items), [review, onState]);
  useEffect(() => {
    if (focusStatus.current && phase !== "loading") {
      focusStatus.current = false;
      status.current?.focus();
    }
  }, [phase, data]);
  function go(next: URLSearchParams, focus = true) {
    if(draftEditing&&!window.confirm(t(l,"Discard the unsaved draft form?","저장하지 않은 초안 입력을 버릴까요?","هل تريد تجاهل بيانات المسودة غير المحفوظة؟")))return;
    setState({ key: "pending" });
    setSelection(null);
    setAttempts((n) => n + 1);
    focusStatus.current = focus;
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
  function refresh() { reload(true); }
  function reload(focus: boolean) {
    decision.store?.invalidatePreparation();
    documents.store?.invalidate();
    const next = new URLSearchParams(params);
    for (const k of [
      "approval-cursor",
      "approval-snapshot",
      "approval-permission_revision",
    ])
      next.delete(k);
    go(next, focus);
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
          } else if (selected && decision.state.entries[selected.id]?.phase !== "pending") {
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
          {dc(l,"help")}
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
              : dc(l,"evidence")}
        {data && (
          <>
            {" "}
            <bdi className="b09-count" dir="ltr">
              {data.observed_at}
            </bdi>
          </>
        )}
      </p>
      <Failure locale={l} code={documents.error} />
      <Attempts locale={l} binding={documents} />
      {Object.values(decision.state.entries).filter(e => e.submission.target !== selected?.id && e.phase !== "refused").map(entry => <DecisionResult key={entry.submission.attempt.key} locale={l} entry={entry} retry={() => void decision.store?.retry(entry.submission.target)} refresh={refresh} reauthenticate={decision.reauthenticate} refreshFailed={phase === "error"} />)}
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
          decision={decision} documents={documents}
          refresh={refresh}
          data={data}
          selected={selected}
          select={(id) => {if(!draftEditing||window.confirm(t(l,"Discard the unsaved draft form?","저장하지 않은 초안 입력을 버릴까요?","هل تريد تجاهل بيانات المسودة غير المحفوظة؟")))setSelection(id ? { key, id } : null);}}
          page={page}
        />
      ) : null}
      {finalSelected&&<CreateDraft key={finalSelected.id+finalSelected.evidence.revision} locale={l} binding={drafts} request={finalSelected.id} digest={finalSelected.evidence.revision!} onDirty={setDraftEditing} onOpen={openDraft}/>}
      <DraftOutcomes locale={l} binding={drafts} onInspect={openDraft}/>
    </div>
  );
}

export default function Screen(props: Parameters<typeof ApprovalScreen>[0]) {
 return props.view === "drafts" ? <DraftScreen locale={props.locale} onState={props.onState} onDirty={props.onDirty} /> : props.view === "results" ? <ResultScreen locale={props.locale} onState={props.onState} onDirty={props.onDirty} /> : <ApprovalScreen {...props} />;
}
