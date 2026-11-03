import { useEffect, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text } from "../../../../shared/copy";
import {
  source,
  validProfile,
  validCandidates,
  page,
  uuid,
  digest,
  ordinal,
  rowOrdinal,
  ServiceError,
  type Profile,
  type Candidates,
  type Page,
  type Row,
} from "./api";
import "./style.css";
export default function SourceData({
  locale: l,
  objectId,
  onState,
}: {
  locale: Locale;
  objectId?: string;
  onState: (s: string, items: string[]) => void;
}) {
  const c = (en: string, ko: string, ar: string) => text(l, en, ko, ar);
  const query = new URLSearchParams(location.hash.split("?")[1]);
  const snapshot = query.get("snapshot"),
    after = query.get("after") ?? "0",
    rowId = query.get("row"),
    focus = query.get("focus");
  const [profile, setProfile] = useState<Profile | null>(null),
    [list, setList] = useState<Candidates | null>(null),
    [data, setData] = useState<Page | null>(null);
  const [error, setError] = useState<ServiceError | null>(null),
    [loading, setLoading] = useState(true),
    [retry, setRetry] = useState(0);
  const href = (release?: string, changes: Record<string, string> = {}) => {
    const q = new URLSearchParams({ lang: l, service: "local" });
    if (release && snapshot) {
      q.set("snapshot", snapshot);
      q.set("after", after);
    }
    for (const [k, v] of Object.entries(changes))
      if (v) q.set(k, v);
      else q.delete(k);
    return `#/source-data${release ? "/" + release : ""}?${q}`;
  };
  useEffect(() => {
    const controller = new AbortController();
    let active = true;
    setProfile(null);
    setList(null);
    setData(null);
    setError(null);
    setLoading(true);
    void (async () => {
      try {
        if (
          (objectId && !uuid.test(objectId)) ||
          !ordinal.test(after) ||
          (snapshot !== null && !digest.test(snapshot)) ||
          (rowId !== null && !rowOrdinal.test(rowId))
        )
          throw new ServiceError("request_invalid", "");
        const p = await source<Profile>("/profile", controller.signal);
        if (!validProfile(p)) throw new ServiceError("response_invalid", "");
        const candidates = await source<Candidates>(
          "/candidates" + (objectId ? "/" + objectId : ""),
          controller.signal,
        );
        if (
          !validCandidates(candidates) ||
          candidates.candidates.length > 25 ||
          candidates.candidates.some(
            (r) => !uuid.test(r.release_id) || !digest.test(r.manifest_digest),
          ) ||
          (objectId &&
            (candidates.candidates.length !== 1 ||
              candidates.candidates[0].release_id !== objectId))
        )
          throw new ServiceError("response_invalid", "");
        let result: Page | null = null;
        if (objectId) {
          result = await page(
            objectId,
            snapshot ?? candidates.candidates[0].manifest_digest,
            after,
            controller.signal,
          );
          if (rowId && !result.rows.some((r) => r.ordinal === rowId))
            throw new ServiceError("request_invalid", "");
        }
        if (active) {
          setProfile(p);
          setList(candidates);
          setData(result);
        }
      } catch (e) {
        if (active) {
          setProfile(null);
          setList(null);
          setData(null);
          setError(
            e instanceof ServiceError
              ? e
              : new ServiceError("service_unavailable", ""),
          );
        }
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
      controller.abort();
    };
  }, [objectId, snapshot, after, rowId, retry]);
  useEffect(() => {
    if (data)
      document
        .getElementById(
          rowId
            ? "source-detail"
            : focus
              ? `source-${window.innerWidth < 640 ? "mobile" : "row"}-${focus}`
              : "source-page-title",
        )
        ?.focus();
  }, [data, rowId, focus]);
  useEffect(
    () =>
      onState(
        JSON.stringify({ profile, data, loading, error: error?.code, rowId }),
        [
          "S04:scope:local-inspection",
          "S04:list:releases",
          "S04:table:normalized",
          "S04:page:snapshot",
          "S04:detail:row",
          "S04:state:unavailable",
        ],
      ),
    [profile, data, loading, error, rowId, onState],
  );
  const missing = c("Missing", "누락", "مفقود"),
    unsupported = c(
      "Not provided by this contract",
      "이 계약에서 제공하지 않음",
      "غير متاح في هذا العقد",
    );
  const value = (r: Row, key: string) => (
    <bdi dir="ltr">
      {Object.hasOwn(r.values, key) ? (r.values[key] ?? missing) : unsupported}
    </bdi>
  );
  const amount = (r: Row, key: string) => (
    <bdi dir="ltr" className="source-amount">
      {value(r, key)} <bdi>{r.values.billing_currency ?? missing}</bdi>
    </bdi>
  );
  const selected = data?.rows.find((r) => r.ordinal === rowId);
  const detailHref = (r: Row) =>
    href(objectId, { snapshot: data!.snapshot, row: r.ordinal });
  function rememberRow(r: Row) {
    history.replaceState(
      null,
      "",
      href(objectId, { snapshot: data!.snapshot, focus: r.ordinal }),
    );
  }
  useEffect(() => {
    if (!selected || !data) return;
    const close = (event: KeyboardEvent) => {
      if (
        event.key === "Escape" &&
        !event.defaultPrevented &&
        !document.querySelector("dialog[open]")
      ) {
        event.preventDefault();
        location.hash = href(objectId, {
          snapshot: data.snapshot,
          focus: selected.ordinal,
        });
      }
    };
    window.addEventListener("keydown", close);
    return () => window.removeEventListener("keydown", close);
  }, [selected, data]);
  const cols = [
    c("Row", "행", "الصف"),
    c("Account / service", "계정 / 서비스", "الحساب / الخدمة"),
    c("Usage interval (UTC)", "사용 구간 (UTC)", "فترة الاستخدام (UTC)"),
    c("Quantity / unit", "수량 / 단위", "الكمية / الوحدة"),
    "Billed",
    "Effective",
  ];
  return (
    <div className="source-data">
      <section className="source-intro">
        <h2>
          {c(
            "Received cost and usage",
            "수신된 비용·사용량",
            "التكلفة والاستخدام المستلمان",
          )}
        </h2>
        <p>
          {c(
            "Local synthetic inspection. These are received inputs, not confirmed billing amounts. Production cost-access permissions are not connected.",
            "로컬 합성 자료 조회입니다. 수신된 입력이며 청구 확정 금액이 아닙니다. 운영 환경의 원가 열람 권한은 아직 연결되지 않았습니다.",
            "فحص محلي لبيانات تجريبية. هذه مدخلات مستلمة وليست مبالغ فوترة مؤكدة. صلاحيات عرض التكلفة للإنتاج غير متصلة.",
          )}
        </p>
      </section>
      <div role="status">
        {loading
          ? c(
              "Loading source rows…",
              "명세를 조회하고 있습니다…",
              "جارٍ تحميل صفوف المصدر…",
            )
          : ""}
      </div>
      {error && (
        <section role="alert">
          <h2>
            {c(
              "Source data unavailable",
              "명세를 조회할 수 없습니다",
              "بيانات المصدر غير متاحة",
            )}
          </h2>
          <p>
            {error.code === "scope_denied"
              ? c(
                  "Access is unavailable. Previous data has been cleared.",
                  "열람 권한을 확인할 수 없어 이전 데이터를 지웠습니다.",
                  "الوصول غير متاح. تم مسح البيانات السابقة.",
                )
              : c(
                  "The service, snapshot or input could not be verified. No earlier rows are retained.",
                  "서비스·조회 개정·입력 상태를 확인하지 못했습니다. 이전 행은 유지하지 않습니다.",
                  "تعذر التحقق من الخدمة أو المراجعة أو المدخل. لم يتم الاحتفاظ بالصفوف السابقة.",
                )}
          </p>
          <bdi>{error.code}</bdi>
          <p>
            <button onClick={() => setRetry((v) => v + 1)}>
              {c("Retry", "다시 조회", "إعادة المحاولة")}
            </button>{" "}
            <a href={href()}>
              {c("Release list", "수신 목록", "قائمة المدخلات")}
            </a>
          </p>
        </section>
      )}
      {profile && (
        <dl className="source-scope">
          <div>
            <dt>{c("Account", "계정", "الحساب")}</dt>
            <dd>
              <bdi>{profile.account}</bdi>
            </dd>
          </div>
          <div>
            <dt>
              {c(
                "Billing period · start inclusive, end exclusive",
                "청구 기간 · 시작 포함, 끝 제외",
                "فترة الفوترة · البداية مشمولة والنهاية مستثناة",
              )}
            </dt>
            <dd>
              <bdi dir="ltr" className="source-date">
                {profile.period_start}
              </bdi>
              <bdi dir="ltr" className="source-date">
                {profile.period_end}
              </bdi>
            </dd>
          </div>
          <div>
            <dt>{c("Workspace", "업무 공간", "مساحة العمل")}</dt>
            <dd>
              <bdi>{profile.workspace}</bdi>
            </dd>
          </div>
        </dl>
      )}
      {list && !objectId && (
        <section>
          <h2>
            {c(
              "Select received input",
              "조회할 수신 입력 선택",
              "اختيار المدخل المستلم",
            )}
          </h2>
          <p>
            {c(
              "Latest 25 at most. This list is not a total count.",
              "최근 최대 25개입니다. 전체 건수를 뜻하지 않습니다.",
              "آخر 25 مدخلاً كحد أقصى. ليست عدداً إجمالياً.",
            )}
          </p>
          <div className="source-release-list">
            {list.candidates.map((r) => (
              <a
                key={r.release_id}
                href={href(r.release_id, {
                  snapshot: r.manifest_digest,
                  after: "0",
                })}
              >
                <strong>
                  <bdi>{r.release_id}</bdi>
                </strong>
                <span>
                  {c("Source revision", "원본 리비전", "مراجعة المصدر")}{" "}
                  <bdi>{r.source_revision}</bdi> · {r.row_count}{" "}
                  {c("declared rows", "선언 행", "صفوف معلنة")} ·{" "}
                  <bdi>{r.state}</bdi>
                </span>
              </a>
            ))}
          </div>
          {!list.candidates.length && (
            <p>
              {c(
                "No received input.",
                "수신 입력이 없습니다.",
                "لا توجد مدخلات مستلمة.",
              )}
            </p>
          )}
          {list.more && (
            <p>
              {c(
                "Older inputs exist. Open an exact release link to inspect one.",
                "이전 입력이 더 있습니다. 정확한 입력 링크로 조회할 수 있습니다.",
                "توجد مدخلات أقدم. استخدم رابط المدخل المحدد لفحصه.",
              )}
            </p>
          )}
        </section>
      )}
      {data && (
        <>
          <section>
            <a href={href()}>
              {c(
                "All received inputs",
                "수신 입력 목록",
                "قائمة المدخلات المستلمة",
              )}
            </a>
            <h2 id="source-page-title" tabIndex={-1}>
              {c("Normalized rows", "정규화 명세", "الصفوف المعيارية")}
            </h2>
            <p>
              <bdi>{data.candidate.release_id}</bdi> ·{" "}
              <bdi dir="ltr">FOCUS {data.candidate.version}</bdi> ·{" "}
              {c("Source revision", "원본 리비전", "مراجعة المصدر")}{" "}
              <bdi>{data.candidate.source_revision}</bdi>
            </p>
            <p>
              {c("Rows shown", "표시 행", "الصفوف المعروضة")}:{" "}
              {Number(after) + 1}–{Number(after) + data.rows.length} /{" "}
              {data.candidate.row_count}{" "}
              {c("declared rows", "선언 행", "صفوف معلنة")}
            </p>
            <p>
              {c(
                "Billed and Effective remain separate; currencies and units are never summed. Metadata is pinned; warehouse contents have not been revalidated by this view.",
                "Billed와 Effective를 분리하며 통화·단위를 합산하지 않습니다. 근거 명세의 개정은 고정되지만 이 조회에서 저장 내용의 무결성을 재검증하지는 않습니다.",
                "تبقى Billed وEffective منفصلتين ولا تُجمع العملات أو الوحدات. مراجعة البيانات الوصفية مثبتة؛ لم يُعَد التحقق من محتوى المستودع هنا.",
              )}
            </p>
            <details>
              <summary>
                {c(
                  "Snapshot and eligibility",
                  "조회 개정과 적격성",
                  "المراجعة والأهلية",
                )}
              </summary>
              <p>
                <bdi>{data.snapshot}</bdi>
              </p>
              <p>
                {c("Observed", "조회 시각", "وقت الفحص")}:{" "}
                <bdi>{data.observed_at}</bdi>
              </p>
              <p>
                {c("Approval blockers", "확정 차단 사유", "عوائق التأكيد")}:{" "}
                {data.candidate.blockers.map((b) => (
                  <span key={b}>
                    <bdi>{b}</bdi>{" "}
                  </span>
                ))}
              </p>
              <p>
                {c(
                  "Attribution, conversion history and AI-specific fields are not connected.",
                  "귀속·변환 이력·AI 전용 필드는 아직 연결되지 않았습니다.",
                  "الإسناد وسجل التحويل وحقول الذكاء الاصطناعي غير متصلة.",
                )}
              </p>
            </details>
          </section>
          <div className="source-mobile">
            {data.rows.map((r) => (
              <article key={r.ordinal}>
                <h3>
                  <a
                    id={`source-mobile-${r.ordinal}`}
                    href={detailHref(r)}
                    onClick={() => rememberRow(r)}
                  >
                    {cols[0]} <bdi>{r.ordinal}</bdi>
                  </a>
                </h3>
                <p>{value(r, "service_name")}</p>
                <dl>
                  <dt>Billed</dt>
                  <dd>{amount(r, "billed_cost")}</dd>
                  <dt>Effective</dt>
                  <dd>{amount(r, "effective_cost")}</dd>
                  <dt>{cols[3]}</dt>
                  <dd>
                    <bdi dir="ltr">
                      {value(r, "consumed_quantity")}{" "}
                      {value(r, "consumed_unit")}
                    </bdi>
                  </dd>
                </dl>
              </article>
            ))}
          </div>
          <details
            className="source-full-table"
            open={window.innerWidth >= 640}
          >
            <summary>
              {c("Full row table", "전체 명세 표", "جدول الصفوف الكامل")}
            </summary>
            <div
              className="planning-table"
              tabIndex={0}
              role="region"
              aria-label={c(
                "Source rows, horizontally scrollable",
                "명세 표, 가로 스크롤 가능",
                "صفوف المصدر، قابلة للتمرير أفقياً",
              )}
            >
              <table>
                <thead>
                  <tr>
                    {cols.map((h) => (
                      <th key={h} scope="col">
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {data.rows.map((r) => (
                    <tr key={r.ordinal}>
                      <th scope="row">
                        <a
                          id={`source-row-${r.ordinal}`}
                          href={detailHref(r)}
                          onClick={() => rememberRow(r)}
                        >
                          {r.ordinal}
                        </a>
                      </th>
                      <td>
                        {value(r, "billing_account_id")}
                        <small>{value(r, "service_name")}</small>
                      </td>
                      <td>
                        {value(r, "charge_period_start")}
                        <small>{value(r, "charge_period_end")}</small>
                      </td>
                      <td>
                        {value(r, "consumed_quantity")}
                        <small>{value(r, "consumed_unit")}</small>
                      </td>
                      <td>{amount(r, "billed_cost")}</td>
                      <td>{amount(r, "effective_cost")}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </details>
          <nav
            className="source-pagination"
            aria-label={c("Source row pages", "명세 페이지", "صفحات المصدر")}
          >
            <a
              aria-disabled={after === "0"}
              href={
                after === "0"
                  ? undefined
                  : href(objectId, {
                      snapshot: data.snapshot,
                      after: String(Math.max(0, Number(after) - 50)),
                    })
              }
            >
              {c("Previous", "이전", "السابق")}
            </a>
            <span>
              {c("50 rows per page", "페이지당 50행", "50 صفاً لكل صفحة")}
            </span>
            <a
              aria-disabled={data.next_after === null}
              href={
                data.next_after === null
                  ? undefined
                  : href(objectId, {
                      snapshot: data.snapshot,
                      after: data.next_after,
                    })
              }
            >
              {c("Next", "다음", "التالي")}
            </a>
          </nav>
          {selected && (
            <section id="source-detail" tabIndex={-1} className="source-detail">
              <h2>
                {c("Row detail", "행 상세", "تفاصيل الصف")} ·{" "}
                <bdi>{selected.ordinal}</bdi>
              </h2>
              <a
                href={href(objectId, {
                  snapshot: data.snapshot,
                  focus: selected.ordinal,
                })}
              >
                {c(
                  "Close detail and return to row",
                  "상세 닫고 행으로 복귀",
                  "إغلاق التفاصيل والعودة للصف",
                )}
              </a>
              <dl>
                {data.fields.map((f) => (
                  <div key={f}>
                    <dt>
                      <bdi>{f}</bdi>
                    </dt>
                    <dd>{value(selected, f)}</dd>
                  </div>
                ))}
              </dl>
              <p>
                {c(
                  "Evidence viewing is not connected yet (S05). No original is downloaded.",
                  "증빙 열람은 아직 연결되지 않았습니다(S05). 원본을 다운로드하지 않습니다.",
                  "عرض الأدلة غير متصل بعد (S05). لا يتم تنزيل الأصل.",
                )}
              </p>
            </section>
          )}
        </>
      )}
    </div>
  );
}
