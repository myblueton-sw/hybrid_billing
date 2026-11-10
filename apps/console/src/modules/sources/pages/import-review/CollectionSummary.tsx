import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { t, timestamp, shortTimestamp, type Words } from "./copy";
import { ReadError } from "./api";
import {
  loadStatus,
  provenanceKeys,
  type CollectionStatus,
  type Scope,
} from "./status-api";
const text = {
  title: [
    "조회 가능한 수집 자료",
    "Queryable source data",
    "بيانات المصدر القابلة للاستعلام",
  ],
  rows: [
    "허용 범위의 조회 가능한 행",
    "Queryable rows in your permitted scope",
    "الصفوف القابلة للاستعلام ضمن نطاقك المسموح",
  ],
  deliveries: [
    "조회 가능한 수집본",
    "Queryable deliveries",
    "عمليات التسليم القابلة للاستعلام",
  ],
  completeness: [
    "원천 전체 수집 완전성",
    "Upstream completeness",
    "اكتمال التجميع من المصدر",
  ],
  unverified: ["미확인", "Unverified", "غير متحقق"],
  note: [
    "현재 권한으로 저장값을 읽어 확인한 범위입니다. 아래 접수 검색과 별개이며, 청구 확정 자료가 아닙니다.",
    "Stored values read within your current permissions, independent of the receipt search below. This is not approved billing input.",
    "قيم مخزنة قُرئت ضمن صلاحياتك الحالية، بصورة مستقلة عن بحث الإيصالات أدناه. ليست مدخلات فوترة معتمدة.",
  ],
  open: [
    "수집 자료 전체 보기",
    "Open all source data",
    "عرض جميع بيانات المصدر",
  ],
  loading: [
    "조회 가능한 저장 자료를 확인하고 있습니다.",
    "Reading queryable stored data…",
    "جارٍ قراءة البيانات المخزنة القابلة للاستعلام…",
  ],
  error: [
    "저장 자료를 조회하지 못했습니다. 접수 메타데이터는 별도 조회 결과입니다.",
    "Stored data could not be read. Receipt metadata is a separate observation.",
    "تعذرت قراءة البيانات المخزنة. بيانات الإيصالات الوصفية رصد مستقل.",
  ],
  retry: [
    "저장 자료 다시 조회",
    "Retry stored-data read",
    "إعادة محاولة قراءة البيانات المخزنة",
  ],
  empty: [
    "현재 허용 범위에서 조회 가능한 행이 없습니다. 원천 전체에 자료가 없다는 뜻은 아닙니다.",
    "No rows are queryable in this scope. This does not establish that the upstream source has no data.",
    "لا توجد صفوف قابلة للاستعلام ضمن هذا النطاق. لا يثبت ذلك عدم وجود بيانات في المصدر.",
  ],
  provider: ["공급자별 행", "Rows by provider", "الصفوف حسب المزوّد"],
  periods: [
    "원천 수집 구간별 행",
    "Rows by source delivery window",
    "الصفوف حسب نطاق تسليم المصدر",
  ],
  origin: ["자료 출처별 행", "Rows by data origin", "الصفوف حسب منشأ البيانات"],
  interval: [
    "UTC · 시작 포함, 종료 제외",
    "UTC · start inclusive, end exclusive",
    "UTC · البداية مشمولة والنهاية مستثناة",
  ],
  details: [
    "저장 조회 근거",
    "Stored-data read evidence",
    "أدلة قراءة البيانات المخزنة",
  ],
  observed: [
    "저장 조회 시각",
    "Stored-data read at",
    "وقت قراءة البيانات المخزنة",
  ],
  versions: [
    "원천 FOCUS 버전",
    "Source FOCUS versions",
    "إصدارات FOCUS للمصدر",
  ],
  original: [
    "원본 파일 가용성 · 미확인",
    "Original-file availability · Not checked",
    "توفر الملف الأصلي · لم يُتحقّق منه",
  ],
} satisfies Record<string, Words>;
const origins: Record<(typeof provenanceKeys)[number], Words> = {
  "official-public": ["공개 샘플", "Public samples", "عينات عامة"],
  "official-public-subset": [
    "공개 샘플 일부",
    "Public sample subsets",
    "أجزاء من عينات عامة",
  ],
  "synthetic-contract-fixture": [
    "합성 시험 자료",
    "Synthetic test data",
    "بيانات اختبار اصطناعية",
  ],
  unclassified: ["미분류 자료", "Unclassified data", "بيانات غير مصنفة"],
};
export default function CollectionSummary({
  locale: l,
  scope,
  identity,
  onAuthorityError,
  onState,
}: {
  locale: Locale;
  scope: Scope;
  identity: string;
  onAuthorityError: (error: ReadError) => void;
  onState: (state: string) => void;
}) {
  const [retry, setRetry] = useState(0),
    [state, setState] = useState<{
      key: string;
      data?: CollectionStatus;
      error?: ReadError;
    }>({ key: "" });
  const key = JSON.stringify([
    identity,
    scope.auth_scope,
    scope.permission_revision,
    scope.audience,
    scope.member_count,
    retry,
  ]);
  const generation = useRef(0),
    status = useRef<HTMLParagraphElement>(null),
    focus = useRef(false);
  const current = state.key === key ? state : undefined,
    data = current?.data,
    error = current?.error;
  const phase = data ? "loaded" : error ? "error" : "loading";
  useEffect(() => {
    const controller = new AbortController(),
      n = ++generation.current;
    setState({ key });
    loadStatus(scope, controller.signal)
      .then((data) => {
        if (!controller.signal.aborted && n === generation.current)
          setState({ key, data });
      })
      .catch((e) => {
        if (controller.signal.aborted || n !== generation.current) return;
        const error =
          e instanceof ReadError ? e : new ReadError("service_unavailable");
        if (
          error.httpStatus === 403 ||
          ["permission_changed", "reader_revoked", "scope_denied"].includes(
            error.code,
          )
        )
          onAuthorityError(error);
        else setState({ key, error });
      });
    return () => controller.abort();
  }, [key, onAuthorityError]);
  useEffect(() => {
    onState(
      JSON.stringify({
        phase,
        snapshot: data?.snapshot ?? null,
        observed_at: data?.observed_at ?? null,
        row_count: data?.row_count ?? null,
        delivery_count: data?.delivery_count ?? null,
        error: error?.code ?? null,
      }),
    );
  }, [phase, data, error, onState]);
  useEffect(() => {
    if (phase !== "loading" && focus.current) {
      focus.current = false;
      status.current?.focus();
    }
  }, [phase]);
  const count = (n: number) =>
    new Intl.NumberFormat(l, { numberingSystem: "latn" }).format(n);
  const windowLabel = (value: string) =>
    /T00:00:00(?:\.0+)?(?:Z|\+00:00)$/.test(value)
      ? value.slice(0, 10)
      : timestamp(value);
  const sourceLink =
    "#/source-data?" + new URLSearchParams({ lang: l }).toString();
  const table = (
    title: string,
    rows: { label: ReactNode; count: number; key: string }[],
    note?: string,
  ) => {
    const column =
      title === t(l, text.provider)
        ? t(l, ["공급자", "Provider", "المزوّد"])
        : title === t(l, text.periods)
          ? t(l, ["수집 구간", "Delivery window", "نطاق التسليم"])
          : t(l, ["출처", "Origin", "المنشأ"]);
    const render = (list: typeof rows) => (
      <table>
        <thead>
          <tr>
            <th scope="col">{column}</th>
            <th scope="col">{t(l, ["행 수", "Rows", "الصفوف"])}</th>
          </tr>
        </thead>
        <tbody>
          {list.map((r) => (
            <tr key={r.key}>
              <th scope="row">{r.label}</th>
              <td>
                <bdi dir="ltr">{count(r.count)}</bdi>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    );
    return (
      <section>
        <h3>{title}</h3>
        {render(rows.slice(0, 6))}
        {note && <p className="receipt-meta">{note}</p>}
        {rows.length > 6 && (
          <details>
            <summary>
              {t(l, [
                `나머지 ${count(rows.length - 6)}개 보기`,
                `Show ${count(rows.length - 6)} more`,
                `عرض ${count(rows.length - 6)} إضافية`,
              ])}
            </summary>
            {render(rows.slice(6))}
          </details>
        )}
      </section>
    );
  };
  return (
    <section
      className="collection-status"
      data-summary-phase={phase}
      aria-busy={phase === "loading"}
    >
      <header>
        <h2>{t(l, text.title)}</h2>
        <a href={sourceLink}>{t(l, text.open)}</a>
      </header>
      <p>{t(l, text.note)}</p>
      <p
        className="collection-read-status"
        role="status"
        aria-live="polite"
        tabIndex={-1}
        ref={status}
      >
        {phase === "loading"
          ? t(l, text.loading)
          : error
            ? t(l, text.error)
            : data && (
                <>
                  {t(l, text.observed)}:{" "}
                  <time dateTime={data.observed_at}>
                    <bdi dir="ltr">{shortTimestamp(data.observed_at)}</bdi>
                  </time>
                </>
              )}
      </p>
      {error && (
        <div className="collection-failure" role="alert">
          <p>
            <bdi dir="ltr">{error.code}</bdi>
            {error.requestId && (
              <>
                {" "}
                · <bdi dir="ltr">{error.requestId}</bdi>
              </>
            )}
          </p>
          <button
            type="button"
            onClick={() => {
              focus.current = true;
              setRetry((n) => n + 1);
            }}
          >
            {t(l, text.retry)}
          </button>
        </div>
      )}
      {data && (
        <>
          <dl className="collection-facts">
            {[
              [text.rows, count(data.row_count)],
              [text.deliveries, count(data.delivery_count)],
              [text.completeness, t(l, text.unverified)],
            ].map(([label, value], i) => (
              <div key={i}>
                <dt>{t(l, label as Words)}</dt>
                <dd>
                  <bdi dir={i < 2 ? "ltr" : undefined}>{value as string}</bdi>
                </dd>
              </div>
            ))}
          </dl>
          {data.row_count === 0 ? (
            <p>{t(l, text.empty)}</p>
          ) : (
            <div className="collection-breakdowns">
              {table(
                t(l, text.provider),
                [...data.providers]
                  .sort(
                    (a, b) =>
                      b.row_count - a.row_count ||
                      a.provider.localeCompare(b.provider),
                  )
                  .map((r) => ({
                    key: r.provider,
                    label: <bdi dir="auto">{r.provider}</bdi>,
                    count: r.row_count,
                  })),
              )}
              {table(
                t(l, text.periods),
                [...data.periods].reverse().map((r) => ({
                  key: r.window_start + r.window_end,
                  label: (
                    <>
                      <span className="receipt-meta">
                        {t(l, [
                          "시작 · 포함 (UTC)",
                          "Start · inclusive (UTC)",
                          "البداية · مشمولة (UTC)",
                        ])}
                      </span>
                      <br />
                      <bdi dir="ltr">{windowLabel(r.window_start)}</bdi>
                      <span className="collection-window-end">
                        <span className="receipt-meta">
                          {t(l, [
                            "종료 · 제외 (UTC)",
                            "End · exclusive (UTC)",
                            "النهاية · مستثناة (UTC)",
                          ])}
                        </span>
                        <br />
                        <bdi dir="ltr">{windowLabel(r.window_end)}</bdi>
                      </span>
                    </>
                  ),
                  count: r.row_count,
                })),
              )}
              {table(
                t(l, text.origin),
                provenanceKeys.map((k) => ({
                  key: k,
                  label: t(l, origins[k]),
                  count: data.provenance[k],
                })),
              )}
            </div>
          )}
          <details className="collection-evidence">
            <summary>{t(l, text.details)}</summary>
            <p>
              {t(l, text.versions)}:{" "}
              <bdi dir="ltr">{data.versions.join(", ") || "—"}</bdi>
            </p>
            <p>{t(l, text.original)}</p>
            <p>
              {t(l, text.observed)}:{" "}
              <time dateTime={data.observed_at}>
                <bdi dir="ltr">{timestamp(data.observed_at)}</bdi>
              </time>
            </p>
            <dl>
              <div>
                <dt>
                  {t(l, [
                    "저장 조회 스냅샷",
                    "Stored-data snapshot",
                    "لقطة البيانات المخزنة",
                  ])}
                </dt>
                <dd>
                  <bdi dir="ltr" className="receipt-id">
                    {data.snapshot}
                  </bdi>
                </dd>
              </div>
            </dl>
          </details>
        </>
      )}
    </section>
  );
}
