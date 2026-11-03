import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text } from "../../../../shared/copy";
import {
  reconcile,
  ServiceError,
  type Candidate,
  type Candidates,
  type Profile,
  type Result,
} from "./api";
import "./style.css";
export default function Reconciliation({
  locale: l,
  onState,
}: {
  locale: Locale;
  onState: (state: string, items: string[]) => void;
}) {
  const copy = (en: string, ko: string, ar: string) => text(l, en, ko, ar);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [list, setList] = useState<Candidates | null>(null);
  const [selected, setSelected] = useState<Candidate | null>(null);
  const [result, setResult] = useState<Result | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ServiceError | null>(null);
  const request = useRef<AbortController | null>(null),
    epoch = useRef(0);
  const label = (key: string): string =>
    ({
      match: copy("Consistent", "일치", "متسق"),
      difference: copy("Difference found", "차이 있음", "يوجد اختلاف"),
      incomplete: copy("Missing values", "누락값 있음", "قيم مفقودة"),
      unavailable: copy("Not verified", "확인 불가", "لم يتم التحقق"),
      verified: copy("Validated input", "검증된 입력", "مدخل تم التحقق منه"),
      sealed: copy("Previously published", "이전 공개본", "منشور سابقاً"),
      pending: copy("Input pending", "입력 대기", "مدخل قيد الانتظار"),
      independent_cross_validation_missing: copy(
        "Independent billing validation evidence is missing.",
        "독립된 청구 검증 근거가 없습니다.",
        "أدلة التحقق المستقل للفوترة غير متوفرة.",
      ),
      canonical_confirm_publish_unavailable: copy(
        "Atomic confirmation and publication are not connected.",
        "원자적 확정·공개 기능이 연결되지 않았습니다.",
        "التأكيد والنشر الذريان غير متصلين.",
      ),
      quarantine_hold: copy(
        "Quarantined input exists in this connection.",
        "이 연결에 격리된 입력이 있습니다.",
        "توجد مدخلات معزولة في هذا الاتصال.",
      ),
      operation_unresolved: copy(
        "An operation outcome remains unresolved.",
        "결과가 미확정인 작업이 있습니다.",
        "توجد عملية نتيجتها غير محسومة.",
      ),
      policy_changed: copy(
        "The policy changed. Reload before inspecting.",
        "정책이 변경되었습니다. 새로 조회해 주세요.",
        "تغيرت السياسة. أعد التحميل قبل الفحص.",
      ),
      candidate_unverified: copy(
        "Input validation is incomplete.",
        "입력 검증이 완료되지 않았습니다.",
        "التحقق من المدخل غير مكتمل.",
      ),
      consistency_difference: copy(
        "Source and storage evidence differ.",
        "원본과 저장 근거에 차이가 있습니다.",
        "تختلف أدلة المصدر والتخزين.",
      ),
      measure_incomplete: copy(
        "Some measures contain missing values.",
        "일부 측정값에 누락이 있습니다.",
        "تحتوي بعض المقاييس على قيم مفقودة.",
      ),
      billed_cost: copy("Billed cost", "청구 비용", "التكلفة المفوترة"),
      effective_cost: copy("Effective cost", "실효 비용", "التكلفة الفعلية"),
      contracted_cost: copy(
        "Contracted cost",
        "계약 비용",
        "التكلفة التعاقدية",
      ),
      list_cost: copy("List cost", "정가 비용", "تكلفة القائمة"),
      consumed_quantity: copy(
        "Consumed quantity",
        "사용 수량",
        "الكمية المستهلكة",
      ),
      pricing_quantity: copy("Pricing quantity", "과금 수량", "كمية التسعير"),
      original_hash: copy("Original integrity", "원본 무결성", "سلامة الأصل"),
      source_manifest_count: copy(
        "Source row count",
        "원본 행 수",
        "عدد صفوف المصدر",
      ),
      source_manifest_hash: copy(
        "Source row hash",
        "원본 행 해시",
        "بصمة صفوف المصدر",
      ),
      source_manifest_controls: copy(
        "Source control totals",
        "원본 대사 합계",
        "مجاميع المصدر الرقابية",
      ),
      storage_manifest_count: copy(
        "Stored row count",
        "저장 행 수",
        "عدد الصفوف المخزنة",
      ),
      storage_manifest_hash: copy(
        "Stored row hash",
        "저장 행 해시",
        "بصمة الصفوف المخزنة",
      ),
      storage_aggregate_controls: copy(
        "Storage aggregate consistency",
        "저장 집계 일관성",
        "اتساق تجميع التخزين",
      ),
    })[key] ??
    copy("Evidence unavailable", "근거 확인 불가", "الأدلة غير متاحة");
  async function run(candidate?: Candidate) {
    const id = ++epoch.current;
    request.current?.abort();
    const c = new AbortController();
    request.current = c;
    setBusy(true);
    setError(null);
    setResult(null);
    if (!candidate) {
      setList(null);
      setSelected(null);
      setProfile(null);
    }
    try {
      if (candidate) {
        const r = await reconcile<Result>(
          "/candidates/" + candidate.release_id + "/revalidate",
          c.signal,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              expected_manifest: candidate.manifest_digest,
            }),
          },
        );
        if (id === epoch.current) {
          setResult(r);
          setSelected(r.candidate);
        }
      } else {
        const p = await reconcile<Profile>("/profile", c.signal);
        const rows = await reconcile<Candidates>("/candidates", c.signal);
        if (id === epoch.current) {
          setProfile(p);
          setList(rows);
        }
      }
    } catch (e) {
      if (id === epoch.current && !c.signal.aborted) {
        setProfile(null);
        setList(null);
        setSelected(null);
        setResult(null);
        setError(
          e instanceof ServiceError
            ? e
            : new ServiceError("service_unavailable", ""),
        );
      }
    } finally {
      if (id === epoch.current) setBusy(false);
    }
  }
  useEffect(() => {
    void run();
    return () => {
      epoch.current++;
      request.current?.abort();
    };
  }, []);
  useEffect(() => {
    onState(
      JSON.stringify({
        profile,
        list,
        selected,
        result,
        busy,
        error: error?.code ?? null,
      }),
      [
        "S06:region:source",
        "S06:local:scope",
        "S06:local:candidates",
        "S06:local:revalidate",
        "S06:local:comparisons",
        "S06:local:evidence",
        "S06:local:blockers",
        "S06:local:async",
      ],
    );
  }, [profile, list, selected, result, busy, error, onState]);
  function select(row: Candidate) {
    epoch.current++;
    request.current?.abort();
    setBusy(false);
    setSelected(row);
    setResult(null);
    setError(null);
  }
  const columns = [
    copy("Measure / unit", "항목 / 단위", "المقياس / الوحدة"),
    copy("Original replay", "원본 재계산", "إعادة حساب المصدر"),
    copy("Stored aggregate", "저장 집계", "التجميع المخزن"),
    copy("Difference", "차이", "الفرق"),
    copy("Result", "결과", "النتيجة"),
  ];
  const missing = copy("Missing", "누락", "مفقود");
  const amount = (value: string | null) =>
    value === null ? (
      <span>{missing}</span>
    ) : (
      <bdi dir="ltr" className="rec-money">
        {value}
      </bdi>
    );
  return (
    <div className="reconciliation" aria-busy={busy}>
      <section className="rec-intro" aria-labelledby="rec-title">
        <div className="rec-heading">
          <div>
            <h2 id="rec-title">
              {copy(
                "Source and storage reconciliation",
                "원본·저장 데이터 대사",
                "مطابقة المصدر والتخزين",
              )}
            </h2>
            <p>
              {copy(
                "Compare the original file with stored amounts, quantities and integrity evidence.",
                "원본 파일과 저장된 금액·수량·무결성 근거를 비교합니다.",
                "قارن الملف الأصلي بالمبالغ والكميات وأدلة السلامة المخزنة.",
              )}
            </p>
          </div>
          <button disabled={busy} onClick={() => void run()}>
            {copy("Refresh candidates", "후보 새로 조회", "تحديث المرشحين")}
          </button>
        </div>
        {profile && (
          <dl className="rec-scope">
            <div>
              <dt>{copy("Account", "계정", "الحساب")}</dt>
              <dd>
                <bdi>{profile.account}</bdi>
              </dd>
            </div>
            <div>
              <dt>
                {copy(
                  "Period (end exclusive)",
                  "기간 (종료시각 미포함)",
                  "الفترة (النهاية غير مشمولة)",
                )}
              </dt>
              <dd>
                <bdi>{profile.period_start}</bdi> –{" "}
                <bdi>{profile.period_end}</bdi>
              </dd>
            </div>
            <div>
              <dt>{copy("Workspace", "업무 공간", "مساحة العمل")}</dt>
              <dd>
                <bdi>{profile.workspace}</bdi>
              </dd>
            </div>
          </dl>
        )}
        <p className="rec-limit">
          {copy(
            "This checks source/storage consistency only. It does not confirm billing or publish data. Sales, allocation, ERP and parallel-run reconciliation are not connected.",
            "원본·저장 일관성만 확인합니다. 청구를 확정하거나 데이터를 공개하지 않습니다. 매출·배부·ERP·병행 대사는 미연결입니다.",
            "يفحص هذا اتساق المصدر والتخزين فقط، ولا يؤكد الفوترة أو ينشر البيانات. مطابقة المبيعات والتخصيص وERP والتشغيل الموازي غير متصلة.",
          )}
        </p>
      </section>
      <div role="status" className="rec-status">
        {busy
          ? copy("Reading evidence…", "근거 조회 중…", "جارٍ قراءة الأدلة…")
          : result
            ? copy(
                "Source/storage check: ",
                "원본·저장 대사: ",
                "فحص المصدر والتخزين: ",
              ) + label(result.state)
            : selected
              ? copy(
                  "Candidate selected. Run revalidation to compare evidence.",
                  "후보를 선택했습니다. 다시 대사하여 근거를 비교하세요.",
                  "تم اختيار المرشح. أعد الفحص لمقارنة الأدلة.",
                )
              : ""}
      </div>
      {error && (
        <section role="alert" className="rec-error">
          <h3>
            {copy(
              "Could not read current evidence",
              "현재 근거를 조회하지 못했습니다",
              "تعذرت قراءة الأدلة الحالية",
            )}
          </h3>
          <p>
            {error.code === "evidence_changed"
              ? copy(
                  "The manifest changed. Refresh candidates before retrying.",
                  "근거 명세가 변경되었습니다. 후보를 새로 조회해 주세요.",
                  "تغير بيان الأدلة. حدّث المرشحين قبل إعادة المحاولة.",
                )
              : copy(
                  "Check the local service and assigned scope, then refresh candidates. No successful comparison is available.",
                  "로컬 서비스와 지정 범위를 확인한 뒤 후보를 새로 조회해 주세요. 성공한 대사 결과가 없습니다.",
                  "تحقق من الخدمة المحلية والنطاق المحدد ثم حدّث المرشحين. لا تتوفر نتيجة مطابقة ناجحة.",
                )}
          </p>
          <bdi>{error.code}</bdi>
          {error.requestId && (
            <p>
              <bdi>{error.requestId}</bdi>
            </p>
          )}
        </section>
      )}
      {list && (
        <>
          <section
            className="rec-blockers"
            aria-labelledby="rec-blockers-title"
          >
            <h3 id="rec-blockers-title">
              {copy(
                "Confirmation remains unavailable",
                "확정 불가 사유",
                "أسباب عدم إتاحة التأكيد",
              )}
            </h3>
            <ul>
              {(result?.blockers ?? selected?.blockers ?? list.blockers).map(
                (b) => (
                  <li key={b}>
                    {label(b)}{" "}
                    <small>
                      <bdi>{b}</bdi>
                    </small>
                  </li>
                ),
              )}
            </ul>
          </section>
          <section aria-labelledby="rec-candidates">
            <h2 id="rec-candidates">
              {copy("Recent inputs", "최근 입력", "المدخلات الحديثة")}{" "}
              <span className="rec-count">{list.candidates.length}</span>
            </h2>
            <p>
              {list.more
                ? copy(
                    "25 most recent candidates shown; older entries are not listed.",
                    "최신 후보 25건을 표시합니다. 이전 후보는 목록에서 제외됩니다.",
                    "يظهر أحدث 25 مرشحاً؛ الإدخالات الأقدم غير مدرجة.",
                  )
                : copy(
                    "All candidates in this connection are shown (up to 25).",
                    "이 연결의 모든 후보를 표시합니다 (최대 25건).",
                    "تظهر جميع المرشحات في هذا الاتصال (حتى 25).",
                  )}
            </p>
            {list.candidates.length === 0 ? (
              <p>
                {copy(
                  "No input candidates in this scope.",
                  "이 범위에 입력 후보가 없습니다.",
                  "لا توجد مرشحات إدخال في هذا النطاق.",
                )}
              </p>
            ) : (
              <div className="planning-table rec-candidates">
                <table>
                  <thead>
                    <tr>
                      {[
                        copy("Candidate", "후보", "المرشح"),
                        copy("Source revision", "원본 리비전", "مراجعة المصدر"),
                        copy("Rows", "행 수", "الصفوف"),
                        copy("Received", "수신시각", "وقت الاستلام"),
                        copy("Input state", "입력 상태", "حالة المدخل"),
                      ].map((v) => (
                        <th key={v} scope="col">
                          {v}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {list.candidates.map((row) => (
                      <tr
                        key={row.release_id}
                        data-selected={selected?.release_id === row.release_id}
                      >
                        <td data-label={copy("Candidate", "후보", "المرشح")}>
                          <button
                            className="rec-select"
                            aria-pressed={
                              selected?.release_id === row.release_id
                            }
                            onClick={() => select(row)}
                          >
                            <bdi>{row.release_id}</bdi>
                          </button>
                          {selected?.release_id === row.release_id && (
                            <small>{copy("Selected", "선택됨", "محدد")}</small>
                          )}
                        </td>
                        <td data-label={copy("Revision", "리비전", "المراجعة")}>
                          <bdi>{row.source_revision}</bdi>
                        </td>
                        <td data-label={copy("Rows", "행 수", "الصفوف")}>
                          {row.row_count}
                        </td>
                        <td
                          data-label={copy(
                            "Received",
                            "수신시각",
                            "وقت الاستلام",
                          )}
                        >
                          <bdi>{row.created_at}</bdi>
                        </td>
                        <td data-label={copy("State", "상태", "الحالة")}>
                          {label(row.state)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {selected && (
              <a
                href="#rec-detail"
                onClick={(e) => {
                  e.preventDefault();
                  document.getElementById("rec-detail")?.focus();
                  document
                    .getElementById("rec-detail")
                    ?.scrollIntoView({ block: "start" });
                }}
              >
                {copy(
                  "Go to selected detail",
                  "선택한 상세로 이동",
                  "انتقل إلى التفاصيل المحددة",
                )}
              </a>
            )}
          </section>
        </>
      )}
      {selected && (
        <section
          id="rec-detail"
          tabIndex={-1}
          aria-labelledby="rec-detail-title"
        >
          <div className="rec-heading">
            <div>
              <h2 id="rec-detail-title">
                {copy("Selected input", "선택한 입력", "المدخل المحدد")}
              </h2>
              <p>
                <bdi>{selected.release_id}</bdi>
              </p>
            </div>
            <button
              className="primary"
              disabled={busy}
              onClick={() => void run(selected)}
            >
              {busy
                ? copy("Revalidating…", "다시 대사 중…", "جارٍ إعادة الفحص…")
                : copy(
                    "Revalidate source and storage",
                    "원본·저장 다시 대사",
                    "إعادة فحص المصدر والتخزين",
                  )}
            </button>
          </div>
          <p>
            {copy("Manifest", "근거 명세", "بيان الأدلة")}:{" "}
            <bdi>{selected.manifest_digest}</bdi>
          </p>
          {!result && !busy && !error && (
            <p>
              {copy(
                "No comparison has been run for this selection.",
                "선택한 후보의 대사를 아직 실행하지 않았습니다.",
                "لم يتم إجراء مقارنة لهذا المرشح بعد.",
              )}
            </p>
          )}
          {result && (
            <>
              <p>
                {copy("Observed", "확인시각", "وقت الفحص")}:{" "}
                <bdi>{result.observed_at}</bdi> ·{" "}
                {copy(
                  "Temporary result; not saved as approval evidence.",
                  "임시 조회 결과이며 승인 근거로 저장되지 않습니다.",
                  "نتيجة مؤقتة؛ لا تُحفظ كدليل موافقة.",
                )}
              </p>
              {result.state !== "unavailable" ? (
                <>
                  <dl className="rec-checks">
                    {result.checks.map((c) => (
                      <div key={c.id}>
                        <dt>{label(c.id)}</dt>
                        <dd>
                          {c.passed
                            ? copy("Pass", "통과", "اجتاز")
                            : copy("Fail", "실패", "فشل")}
                        </dd>
                      </div>
                    ))}
                  </dl>
                  <h3>
                    {copy(
                      "Amount and quantity comparison",
                      "금액·수량 비교",
                      "مقارنة المبالغ والكميات",
                    )}
                  </h3>
                  <p>
                    {copy(
                      "Measures compared",
                      "비교 항목",
                      "المقاييس المقارنة",
                    )}
                    : {result.comparison_count} ·{" "}
                    {copy("Differences", "차이", "الاختلافات")}:{" "}
                    {result.mismatch_count} ·{" "}
                    {copy("Incomplete", "누락 포함", "غير مكتمل")}:{" "}
                    {result.incomplete_count}
                  </p>
                  <div className="rec-mobile-values">
                    {result.comparisons.map((row, i) => (
                      <article key={i}>
                        <h4>
                          {label(row.measure)} · <bdi>{row.currency}</bdi>
                        </h4>
                        <p>
                          <bdi dir="ltr">
                            {row.consumed_unit ?? missing} /{" "}
                            {row.pricing_unit ?? missing}
                          </bdi>
                        </p>
                        <dl>
                          <dt>{columns[1]}</dt>
                          <dd>
                            {amount(row.expected)} · {missing}:{" "}
                            {row.expected_missing ?? "—"}
                          </dd>
                          <dt>{columns[2]}</dt>
                          <dd>
                            {amount(row.actual)} · {missing}:{" "}
                            {row.actual_missing ?? "—"}
                          </dd>
                          <dt>{columns[3]}</dt>
                          <dd>
                            {row.difference === null
                              ? copy(
                                  "Not computable",
                                  "계산 불가",
                                  "غير قابل للحساب",
                                )
                              : amount(row.difference)}
                          </dd>
                          <dt>{columns[4]}</dt>
                          <dd>{label(row.state)}</dd>
                        </dl>
                      </article>
                    ))}
                  </div>
                  <details
                    className="rec-full-table"
                    open={window.innerWidth >= 640}
                  >
                    <summary>
                      {copy(
                        "Full comparison table",
                        "전체 비교표",
                        "جدول المقارنة الكامل",
                      )}
                    </summary>
                    <p>
                      {copy(
                        "Scroll horizontally to read every column. Exact values are also shown above on narrow screens.",
                        "가로로 스크롤하여 모든 열을 확인하세요. 작은 화면에서는 위에 항목별 정확한 값도 표시합니다.",
                        "مرر أفقياً لقراءة جميع الأعمدة. تظهر القيم الدقيقة أيضاً أعلاه على الشاشات الضيقة.",
                      )}
                    </p>
                    <div
                      className="planning-table rec-comparison"
                      tabIndex={0}
                      role="region"
                      aria-label={copy(
                        "Comparison table, scroll horizontally",
                        "대사 비교표, 가로 스크롤 가능",
                        "جدول المقارنة، قابل للتمرير أفقياً",
                      )}
                    >
                      <table>
                        <caption>
                          {copy("Shown", "표시", "المعروض")}{" "}
                          {result.comparisons.length} /{" "}
                          {result.comparison_count}
                          {result.truncated
                            ? copy(
                                " · Limited to first 100",
                                " · 첫 100개로 제한",
                                " · مقتصر على أول 100",
                              )
                            : ""}
                        </caption>
                        <thead>
                          <tr>
                            {columns.map((c) => (
                              <th key={c} scope="col">
                                {c}
                              </th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {result.comparisons.map((row, i) => (
                            <tr key={i}>
                              <th scope="row">
                                {label(row.measure)}
                                <small>
                                  <bdi>{row.currency}</bdi> ·{" "}
                                  <bdi>{row.consumed_unit ?? missing}</bdi> /{" "}
                                  <bdi>{row.pricing_unit ?? missing}</bdi>
                                </small>
                              </th>
                              <td>
                                {amount(row.expected)}
                                <small>
                                  {copy("Missing", "누락", "مفقود")}:{" "}
                                  {row.expected_missing ??
                                    copy(
                                      "Group absent",
                                      "그룹 없음",
                                      "المجموعة غائبة",
                                    )}
                                </small>
                              </td>
                              <td>
                                {amount(row.actual)}
                                <small>
                                  {copy("Missing", "누락", "مفقود")}:{" "}
                                  {row.actual_missing ??
                                    copy(
                                      "Group absent",
                                      "그룹 없음",
                                      "المجموعة غائبة",
                                    )}
                                </small>
                              </td>
                              <td>
                                {row.difference === null
                                  ? copy(
                                      "Not computable",
                                      "계산 불가",
                                      "غير قابل للحساب",
                                    )
                                  : amount(row.difference)}
                              </td>
                              <td>{label(row.state)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </details>
                  <p>
                    {copy(
                      "Units: consumed / pricing. Each currency and unit group remains separate. Difference = stored − original. Missing values do not become zero; amounts are shown without rounding.",
                      "단위: 사용 / 과금. 통화·단위 그룹별로 구분합니다. 차이 = 저장 − 원본. 누락값은 0으로 바꾸지 않으며 금액을 반올림하지 않습니다.",
                      "الوحدات: الاستهلاك / التسعير. تبقى مجموعات العملات والوحدات منفصلة. الفرق = المخزن − المصدر. لا تتحول القيم المفقودة إلى صفر وتظهر المبالغ دون تقريب.",
                    )}
                  </p>
                </>
              ) : (
                <p>
                  {copy(
                    "Evidence verification failed. Comparison counts and amounts are unavailable.",
                    "근거 검증에 실패했습니다. 비교 건수와 금액을 확인할 수 없습니다.",
                    "فشل التحقق من الأدلة. أعداد المقارنات والمبالغ غير متاحة.",
                  )}
                </p>
              )}
              {result.evidence && (
                <details>
                  <summary>
                    {copy("Integrity evidence", "무결성 근거", "أدلة السلامة")}
                  </summary>
                  <dl>
                    {Object.entries(result.evidence).map(([k, v]) => (
                      <div key={k}>
                        <dt>
                          <bdi>{k}</bdi>
                        </dt>
                        <dd>
                          <bdi>{v}</bdi>
                        </dd>
                      </div>
                    ))}
                  </dl>
                </details>
              )}
            </>
          )}
        </section>
      )}
    </div>
  );
}
