import { useCallback, useEffect, useRef, useState } from "react";
import SlowReadHint from "../../../../shared/SlowReadHint";
import type { Props } from "../../../../shared/domain-screen-ui/work/contracts";
import {
  query,
  queryString,
  identifier,
  recordKeys,
  type Query,
  type Data,
} from "./contract";
import { load, ReadError, type ReadResult } from "./api";
import { t, words, timestamp, shortTimestamp } from "./copy";
import ReceiptList from "./ReceiptList";
import CollectionSummary from "./CollectionSummary";
import ReceiptDetail from "./ReceiptDetail";
import "./screen.css";
type State = {
  key: string;
  result?: ReadResult;
  error?: string;
  requestId?: string;
};
function href(
  params: URLSearchParams,
  q: Query,
  changes: {
    q?: string;
    after?: number;
    snapshot?: string;
    permission_revision?: string;
  },
) {
  const p = new URLSearchParams(params);
  for (const k of [...p.keys()]) if (k.startsWith("receipt-")) p.delete(k);
  p.set("view", q.view);
  const text = changes.q ?? q.q;
  if (text) p.set("receipt-q", text);
  const after = changes.after ?? 0;
  if (after) p.set("receipt-after", String(after));
  if (changes.snapshot) p.set("receipt-snapshot", changes.snapshot);
  if (changes.permission_revision)
    p.set("receipt-permission", changes.permission_revision);
  return "#/imports?" + p.toString();
}
export default function Imports(p: Props) {
  const l = p.locale;
  let q: Query | undefined;
  try {
    q = query(p.params, p.view);
  } catch {}
  const qText = q ? queryString(q) : "invalid";
  const [attempt, setAttempt] = useState(0),
    [state, setState] = useState<State>({ key: "" }),
    [search, setSearch] = useState(q?.q ?? ""),
    [invalid, setInvalid] = useState(false),
    [selection, setSelection] = useState<{
      key: string;
      snapshot: string;
      id: string;
    } | null>(null);
  const key = JSON.stringify([l, qText, attempt]);
  const current = state.key === key ? state : undefined;
  const data: Data | undefined =
    current?.result?.kind === "loaded" ? current.result.data : undefined;
  const phase = !q
    ? "error"
    : current?.error
      ? "error"
      : data
        ? "loaded"
        : current?.result?.kind === "unconnected"
          ? "unconnected"
          : "loading";
  const selected =
    data && selection?.key === key && selection.snapshot === data.snapshot
      ? data.records.find((r) => r.id === selection.id)
      : undefined;
  const [summary, setSummary] = useState<{ key: string; value: string } | null>(
    null,
  );
  const summaryState = useCallback(
    (value: string) => setSummary({ key, value }),
    [key],
  );
  const authorityError = useCallback(
    (error: ReadError) => {
      setSelection(null);
      setSummary(null);
      setState({ key, error: error.code, requestId: error.requestId });
    },
    [key],
  );
  const seq = useRef(0),
    status = useRef<HTMLParagraphElement>(null),
    input = useRef<HTMLInputElement>(null),
    trigger = useRef<HTMLButtonElement | null>(null),
    focusStatus = useRef(false);
  useEffect(() => {
    p.onDirty(false);
    return () => p.onDirty(false);
  }, [p.onDirty]);
  useEffect(() => {
    const c = new AbortController(),
      n = ++seq.current;
    setState({ key });
    setSelection(null);
    setSearch(q?.q ?? "");
    setInvalid(false);
    if (q)
      load(q, c.signal)
        .then((result) => {
          if (!c.signal.aborted && n === seq.current) setState({ key, result });
        })
        .catch((e) => {
          if (!c.signal.aborted && n === seq.current)
            setState({
              key,
              error: e instanceof ReadError ? e.code : "service_unavailable",
              requestId: e instanceof ReadError ? e.requestId : undefined,
            });
        });
    return () => c.abort();
  }, [key]);
  useEffect(() => {
    p.onState(
      JSON.stringify({
        phase,
        view: p.view,
        profile: "retained-focus-receipts-v1",
        commands: "unconnected",
        financial_authority: false,
        requested: q ?? null,
        request_authority: "unverified",
        source: data
          ? {
              auth_scope: data.auth_scope,
              permission_revision: data.permission_revision,
              snapshot: data.snapshot,
              total: data.total,
              capabilities: data.capabilities,
              observed_at: data.observed_at,
            }
          : null,
        error: phase === "error" ? (current?.error ?? "invalid_query") : null,
        error_request_id:
          phase === "error" ? (current?.requestId ?? null) : null,
        selected: selected?.id ?? null,
        stored_status: summary?.key === key ? JSON.parse(summary.value) : null,
      }),
      [
        `S03:data:${p.view}`,
        "S03:state:async",
        "S03:action:filter",
        "S03:action:refresh",
        "S03:action:inspect",
        "S03:data:stored-rows",
        "S03:data:source-provenance",
        "S03:data:source-windows",
        "S03:state:summary-async",
        "S03:action:source-rows",
        ...recordKeys.map((k) => `S03:field:${k}`),
      ],
    );
  }, [
    phase,
    p.view,
    qText,
    data,
    current?.error,
    current?.requestId,
    selected?.id,
    summary,
    key,
    p.onState,
  ]);
  useEffect(() => {
    if (phase !== "loading" && focusStatus.current) {
      focusStatus.current = false;
      status.current?.focus({ preventScroll: true });
    }
  }, [phase, data]);
  const close = () => {
    setSelection(null);
    requestAnimationFrame(() => {
      if (trigger.current?.isConnected) trigger.current.focus();
      else status.current?.focus();
    });
  };
  const reload = () => {
    focusStatus.current = true;
    setSelection(null);
    setState({ key: "" });
    if (q && (q.after !== 0 || q.snapshot || q.permission_revision))
      location.hash = href(p.params, q, { after: 0 });
    else if (q) setAttempt((n) => n + 1);
    else {
      const next = new URLSearchParams(p.params);
      for (const k of [...next.keys()])
        if (k.startsWith("receipt-")) next.delete(k);
      location.hash = "#/imports?" + next.toString();
    }
  };
  const unsupported =
    data &&
    (data.view === "extraction" ||
      (data.view === "completeness" &&
        data.capabilities.completeness === "unavailable"));
  const errorCode = q ? current?.error : "invalid_query";
  const stale =
    errorCode === "query_snapshot_changed" ||
    errorCode === "snapshot_required" ||
    errorCode === "snapshot_stale" ||
    errorCode === "receipt_snapshot_stale" ||
    errorCode === "permission_changed";
  return (
    <section
      className="receipt-screen"
      data-read-phase={phase}
      aria-busy={phase === "loading"}
    >
      <header className="receipt-context">
        <div className="receipt-scope-strip">
          <h2>{t(l, words.scope)}</h2>
          {data && (
            <p>
              {t(
                l,
                data.audience === "company"
                  ? ["회사 범위", "Company scope", "نطاق الشركة"]
                  : ["테넌트 범위", "Tenant scope", "نطاق المستأجر"],
              )}{" "}
              ·{" "}
              {t(l, ["허용 구성원", "Authorized members", "الأعضاء المسموحون"])}{" "}
              <bdi dir="ltr">{data.member_count}</bdi>
            </p>
          )}
          <button type="button" onClick={reload} disabled={phase === "loading"}>
            {t(l, words.refresh)}
          </button>
        </div>
        <details className="receipt-pins">
          <summary>
            {t(l, [
              "조회 범위·개정 근거",
              "Read scope and revision evidence",
              "أدلة نطاق القراءة والمراجعة",
            ])}
          </summary>
          <p>{t(l, words.local)}</p>
          <p>
            {t(
              l,
              p.view === "completeness" ? words.completeness : words.metadata,
            )}
          </p>
          {data && (
            <dl>
              {(["auth_scope", "permission_revision", "snapshot"] as const).map(
                (field) => (
                  <div key={field}>
                    <dt>
                      {t(
                        l,
                        field === "auth_scope"
                          ? [
                              "읽기 범위 해시",
                              "Read scope digest",
                              "بصمة نطاق القراءة",
                            ]
                          : field === "permission_revision"
                            ? [
                                "현재 권한 개정",
                                "Current permission revision",
                                "مراجعة الصلاحية الحالية",
                              ]
                            : [
                                "접수 조회 스냅샷",
                                "Receipt snapshot",
                                "لقطة الإيصالات",
                              ],
                      )}
                    </dt>
                    <dd>
                      <bdi dir="ltr" className="receipt-id">
                        {data[field]}
                      </bdi>
                    </dd>
                  </div>
                ),
              )}
            </dl>
          )}
        </details>
      </header>
      {data && (
        <CollectionSummary
          key={key}
          locale={l}
          scope={data}
          identity={key}
          onAuthorityError={authorityError}
          onState={summaryState}
        />
      )}
      <div className="receipt-section-heading">
        <h2>
          {t(l, [
            "접수 메타데이터",
            "Receipt metadata",
            "بيانات الإيصالات الوصفية",
          ])}
        </h2>
        {data && (
          <p className="receipt-meta">
            {t(l, ["접수 조회 시각", "Receipt read at", "وقت قراءة الإيصالات"])}
            :{" "}
            <time dateTime={data.observed_at}>
              <bdi dir="ltr">{shortTimestamp(data.observed_at)}</bdi>
            </time>
          </p>
        )}
      </div>
      <p>
        {t(l, [
          "파일 전체가 허용 범위에 속하는 접수만 표시합니다. 접수가 표시되지 않는 파일에도 조회 가능한 행이 있을 수 있습니다.",
          "Only whole-file receipts within your permitted scope are shown. Queryable rows may also come from files whose receipts are not shown.",
          "تُعرض إيصالات الملفات الواقعة بالكامل ضمن نطاقك المسموح فقط. قد تأتي الصفوف القابلة للاستعلام أيضًا من ملفات لا تُعرض إيصالاتها.",
        ])}
      </p>
      {q && p.view !== "extraction" && !unsupported && (
        <form
          role="search"
          className="receipt-search"
          onSubmit={(e) => {
            e.preventDefault();
            const next = search.trim().toLowerCase();
            if (!identifier(next)) {
              setInvalid(true);
              input.current?.focus();
              return;
            }
            setInvalid(false);
            focusStatus.current = true;
            setSelection(null);
            if (next === q.q && q.after === 0) reload();
            else location.hash = href(p.params, q, { q: next, after: 0 });
          }}
        >
          <label htmlFor="receipt-search">
            {t(l, words.query)}
            <input
              id="receipt-search"
              type="search"
              ref={input}
              value={search}
              maxLength={128}
              autoComplete="off"
              spellCheck={false}
              aria-invalid={invalid || undefined}
              aria-describedby={invalid ? "receipt-search-error" : undefined}
              onChange={(e) => {
                setSearch(e.target.value);
                setInvalid(false);
              }}
            />
          </label>
          <div className="receipt-actions">
            <button type="submit">{t(l, words.apply)}</button>
            {q.q && (
              <button
                type="button"
                onClick={() => {
                  setSearch("");
                  focusStatus.current = true;
                  location.hash = href(p.params, q, { q: "", after: 0 });
                }}
              >
                {t(l, words.clear)}
              </button>
            )}
          </div>
          <p className="receipt-filter-note">
            {t(l, [
              "이 검색은 접수 목록에만 적용됩니다.",
              "This search filters receipts only.",
              "يُطبّق هذا البحث على الإيصالات فقط.",
            ])}
          </p>
          {invalid && (
            <p id="receipt-search-error" role="alert">
              {t(l, words.invalid)}
            </p>
          )}
        </form>
      )}
      <p
        className="receipt-result-status"
        role="status"
        aria-live="polite"
        tabIndex={-1}
        ref={status}
      >
        {phase === "loading" ? (
          <>
            {t(l, words.loading)}
            <SlowReadHint locale={l} />
          </>
        ) : phase === "unconnected" ? (
          t(l, words.unconnected)
        ) : phase === "error" ? (
          t(l, words.error)
        ) : unsupported ? (
          t(
            l,
            data.view === "extraction" ? words.extraction : words.unsupported,
          )
        ) : data?.total === 0 ? (
          t(l, words.empty)
        ) : data ? (
          <>
            {t(l, ["접수 검색 결과 ", "Receipt results ", "نتائج الإيصالات "])}
            <bdi dir="ltr">
              {`${data.selection.after + 1}–${data.selection.after + data.records.length}`}
            </bdi>
            {t(l, [
              ` / ${data.total}건`,
              ` of ${data.total}`,
              ` من ${data.total}`,
            ])}
          </>
        ) : (
          t(l, words.unknown)
        )}
      </p>
      {(phase === "error" || phase === "unconnected") && (
        <div className="receipt-error" role="alert">
          {stale && (
            <p>
              {t(l, [
                "근거나 권한이 변경됐습니다. 기존 페이지를 새 범위에 자동으로 합치지 않습니다.",
                "Evidence or permissions changed. The old page will not be silently rebased.",
                "تغيّرت الأدلة أو الصلاحيات. لن تُعاد مواءمة الصفحة السابقة تلقائيًا.",
              ])}
            </p>
          )}
          {errorCode && (
            <p>
              {t(l, ["오류 코드", "Error code", "رمز الخطأ"])}:{" "}
              <bdi dir="ltr">{errorCode}</bdi>
            </p>
          )}
          {current?.requestId && (
            <p>
              {t(l, ["요청 ID", "Request ID", "معرّف الطلب"])}:{" "}
              <bdi dir="ltr">{current.requestId}</bdi>
            </p>
          )}
          <button onClick={reload}>{t(l, words.retry)}</button>
        </div>
      )}
      {phase === "loaded" && data && !unsupported && (
        <>
          <div
            className={`receipt-workspace ${selected ? "receipt-has-detail" : ""}`}
          >
            <div>
              <h3>
                {t(
                  l,
                  p.view === "quarantine"
                    ? [
                        "허용된 격리 접수",
                        "Permitted quarantine receipts",
                        "إيصالات العزل المسموحة",
                      ]
                    : p.view === "completeness"
                      ? [
                          "등록된 기대 수집 범위",
                          "Registered collection expectations",
                          "توقعات التجميع المسجلة",
                        ]
                      : [
                          "수집 접수와 처리 상태",
                          "Collection receipts and processing state",
                          "إيصالات التجميع وحالة المعالجة",
                        ],
                )}
              </h3>
              {data.records.length > 0 ? (
                <ReceiptList
                  locale={l}
                  view={data.view}
                  records={data.records}
                  selected={selected?.id}
                  onSelect={(r, button) => {
                    trigger.current = button;
                    setSelection({ key, snapshot: data.snapshot, id: r.id });
                    requestAnimationFrame(() =>
                      document
                        .getElementById("receipt-detail-title")
                        ?.focus({ preventScroll: false }),
                    );
                  }}
                />
              ) : null}
              {data.view === "quarantine" && (
                <p className="receipt-meta">
                  {t(l, [
                    "현재 범위를 입증한 접수의 비민감 사유만 표시합니다. 원문·객체 경로·비소유 자료의 건수는 제공하지 않습니다.",
                    "Only attributable receipts and nonsensitive reasons are shown. No source bodies, object paths or unowned record counts are provided.",
                    "تُعرض الإيصالات المنسوبة للنطاق والأسباب غير الحساسة فقط. لا تُقدّم نصوص المصادر أو مسارات الكائنات أو أعداد السجلات غير المملوكة.",
                  ])}
                </p>
              )}
              {q && data.total !== 0 && (
                <nav
                  className="receipt-pagination"
                  aria-label={t(l, [
                    "접수 페이지",
                    "Receipt pages",
                    "صفحات الإيصالات",
                  ])}
                >
                  <button
                    disabled={q.after === 0}
                    onClick={() => {
                      focusStatus.current = true;
                      location.hash = href(p.params, q, {
                        after: Math.max(0, q.after - 25),
                        snapshot: data.snapshot,
                        permission_revision: data.permission_revision,
                      });
                    }}
                  >
                    {t(l, words.back)}
                  </button>
                  <button
                    disabled={data.next_after === null}
                    onClick={() => {
                      if (data.next_after === null) return;
                      focusStatus.current = true;
                      location.hash = href(p.params, q, {
                        after: data.next_after,
                        snapshot: data.snapshot,
                        permission_revision: data.permission_revision,
                      });
                    }}
                  >
                    {t(l, words.next)}
                  </button>
                  {data.total !== null && data.total > 0 && (
                    <span>
                      {t(l, [
                        `${Math.floor(data.selection.after / 25) + 1} / ${Math.ceil(data.total / 25)} 페이지`,
                        `Page ${Math.floor(data.selection.after / 25) + 1} of ${Math.ceil(data.total / 25)}`,
                        `الصفحة ${Math.floor(data.selection.after / 25) + 1} من ${Math.ceil(data.total / 25)}`,
                      ])}
                    </span>
                  )}
                  <span className="receipt-meta">
                    {t(l, [
                      "현재 필터·허용 범위의 결과만 조회합니다.",
                      "Only results in the current filter and permitted scope are read.",
                      "تُقرأ النتائج ضمن المرشح والنطاق المسموح الحاليين فقط.",
                    ])}
                  </span>
                </nav>
              )}
            </div>
            {selected && (
              <ReceiptDetail locale={l} record={selected} onClose={close} />
            )}
          </div>
        </>
      )}
      <p className="receipt-meta receipt-deferred">{t(l, words.deferred)}</p>
    </section>
  );
}
