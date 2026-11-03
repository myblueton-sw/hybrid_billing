import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text } from "../../../../shared/copy";
import { connection, ServiceError, type Connection } from "../connections/api";
import {
  stages,
  stageIndex,
  stageUrl,
  destination,
  validContext,
} from "./model";
import "./style.css";

export default function Onboarding({
  locale,
  stage,
  expected,
  onState,
}: {
  locale: Locale;
  stage: string;
  expected?: string;
  onState: (state: string, items: string[]) => void;
}) {
  const c = (en: string, ko: string, ar: string) => text(locale, en, ko, ar);
  const [data, setData] = useState<Connection | null>(null),
    [error, setError] = useState<ServiceError | null>(null),
    [loading, setLoading] = useState(true),
    [refresh, setRefresh] = useState(0);
  const title = useRef<HTMLHeadingElement>(null),
    index = stageIndex(stage);
  const names = [
    c("Path selection", "경로 선택", "اختيار المسار"),
    c("Setup and permissions", "설정·권한", "الإعداد والأذونات"),
    c("First data", "첫 데이터", "البيانات الأولى"),
    c("First reconciliation", "첫 대사", "المطابقة الأولى"),
    c("Activation review", "개통 검토", "مراجعة التشغيل"),
  ];
  useEffect(() => {
    const controller = new AbortController();
    let active = true;
    setData(null);
    setError(null);
    setLoading(true);
    if (!validContext(expected)) {
      setError(new ServiceError("request_invalid", ""));
      setLoading(false);
      return () => controller.abort();
    }
    void connection(expected, controller.signal)
      .then((value) => {
        if (active) setData(value);
      })
      .catch((e) => {
        if (active)
          setError(
            e instanceof ServiceError
              ? e
              : new ServiceError("service_unavailable", ""),
          );
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
      controller.abort();
    };
  }, [expected, refresh]);
  useEffect(() => {
    title.current?.focus();
  }, [stage, data]);
  useEffect(
    () =>
      onState(JSON.stringify({ data, error: error?.code, loading, stage }), [
        "S02:scope:connection",
        "S02:navigation:stage",
        "S02:data:path",
        "S02:data:permissions",
        "S02:data:first-input",
        "S02:data:reconciliation",
        "S02:state:activation",
        "S02:action:refresh",
      ]),
    [data, error, loading, stage, onState],
  );
  const unknown = c("Not verified", "미확인", "لم يتم التحقق");
  const time = (v: string) =>
    Number.isFinite(Date.parse(v))
      ? new Date(v)
          .toISOString()
          .replace("T", " ")
          .replace(/\.\d{3}Z$/, " UTC")
      : "—";
  const scopeUrl = data
    ? "#/connections/" +
      data.connection +
      "?" +
      new URLSearchParams({ lang: locale, service: "local" })
    : "";
  const link = (action: Connection["actions"][number], label: string) => {
    const href = data && destination(data, action, locale);
    return href ? (
      <a data-action={action} href={href}>
        {label}
      </a>
    ) : null;
  };
  const facts = (rows: string[][]) => (
    <dl>
      {rows.map(([k, v]) => (
        <div key={k}>
          <dt>{k}</dt>
          <dd>
            <bdi
              className={
                /^[0-9]|^FOCUS/.test(v) ? "onboarding-code" : undefined
              }
            >
              {v}
            </bdi>
          </dd>
        </div>
      ))}
    </dl>
  );
  return (
    <div className="onboarding-live">
      <div className="onboarding-intro">
        <p>
          {c(
            "Inspect the local input path and continue to its evidence. Viewing a stage does not complete or approve it.",
            "로컬 입력 경로를 확인하고 실제 근거 화면으로 이어갑니다. 단계를 열어도 완료·승인 처리되지 않습니다.",
            "افحص مسار الإدخال المحلي وانتقل إلى أدلته. عرض المرحلة لا يعني إكمالها أو اعتمادها.",
          )}
        </p>
        <button
          id="onboarding-refresh"
          disabled={loading}
          onClick={() => setRefresh((v) => v + 1)}
        >
          {c("Refresh evidence", "근거 새로고침", "تحديث الأدلة")}
        </button>
      </div>
      <p role="status">
        {loading
          ? c(
              "Checking current access and connection…",
              "현재 권한과 연결을 확인하고 있습니다…",
              "جارٍ التحقق من الوصول والاتصال…",
            )
          : ""}
      </p>
      {error && (
        <section role="alert">
          <h2>
            {c(
              "Evidence unavailable",
              "근거를 확인할 수 없습니다",
              "الأدلة غير متاحة",
            )}
          </h2>
          <p>
            {c(
              "Previous information and links were cleared. Retry using Refresh evidence.",
              "이전 정보와 이동 링크를 지웠습니다. 근거 새로고침으로 다시 조회하세요.",
              "تم مسح المعلومات والروابط السابقة. أعد المحاولة بتحديث الأدلة.",
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
      {data && (
        <>
          <div className="onboarding-scope">
            {facts([
              [c("Connection", "연결", "الاتصال"), data.connection],
              [
                c(
                  "Workspace / account",
                  "작업 공간 / 계정",
                  "مساحة العمل / الحساب",
                ),
                data.workspace + " / " + data.account,
              ],
              [
                c(
                  "Allowed period · start included, end excluded",
                  "입력 허용 기간 · 시작 포함, 끝 제외",
                  "الفترة المسموحة · البداية مشمولة والنهاية مستثناة",
                ),
                time(data.period_start) + " → " + time(data.period_end),
              ],
              [
                c(
                  "Policy revision / observed at",
                  "정책 개정 / 조회 시각",
                  "مراجعة السياسة / وقت الفحص",
                ),
                data.policy_revision + " / " + time(data.observed_at),
              ],
            ])}
          </div>
          <div className="onboarding-layout">
            <nav
              aria-label={c(
                "Onboarding stages",
                "온보딩 단계",
                "مراحل التهيئة",
              )}
              className="onboarding-stages"
            >
              <ol>
                {stages.map((key, i) => (
                  <li key={key}>
                    <a
                      href={stageUrl(key, locale, expected)}
                      aria-current={index === i ? "step" : undefined}
                    >
                      <span>
                        {i + 1}. {names[i]}
                      </span>
                      <small>
                        {i === 0
                          ? c(
                              "Configured local path",
                              "설정된 로컬 경로",
                              "مسار محلي مهيأ",
                            )
                          : i === 2 && data.latest_candidate
                            ? c(
                                "Candidate recorded",
                                "후보 등록됨",
                                "مرشح مسجل",
                              )
                            : unknown}
                      </small>
                    </a>
                  </li>
                ))}
              </ol>
            </nav>
            <section
              className="onboarding-stage"
              aria-labelledby="onboarding-stage-title"
            >
              <p className="onboarding-eyebrow">
                <bdi dir="ltr">{index + 1} / 5</bdi>
              </p>
              <h2 id="onboarding-stage-title" ref={title} tabIndex={-1}>
                {names[index]}
              </h2>
              {index === 0 && (
                <>
                  <h3>
                    {c(
                      "Manual synthetic CSV",
                      "수동 합성 CSV",
                      "CSV تجريبي يدوي",
                    )}
                  </h3>
                  <p>
                    {c(
                      "This is the configured path, not a newly saved onboarding record. Provider API and continuous collection are not connected.",
                      "현재 설정된 경로이며 새로 저장한 온보딩 기록이 아닙니다. Provider API와 연속 수집은 미연결입니다.",
                      "هذا هو المسار المهيأ وليس سجل تهيئة جديداً. واجهة المزود والجمع المستمر غير متصلين.",
                    )}
                  </p>
                  {facts([
                    [
                      c(
                        "Installed input contracts",
                        "설치된 입력 계약",
                        "عقود الإدخال المثبتة",
                      ),
                      data.contracts
                        .map((v) => "FOCUS " + v.version + " / " + v.revision)
                        .join(", "),
                    ],
                    [
                      c(
                        "Estimated query / storage cost",
                        "예상 조회 / 저장 비용",
                        "تكلفة الاستعلام / التخزين المتوقعة",
                      ),
                      c(
                        "Unknown; estimate not connected",
                        "알 수 없음 · 추정 근거 미연결",
                        "غير معروفة؛ لا يوجد تقدير متصل",
                      ),
                    ],
                  ])}
                  <a href={scopeUrl}>
                    {c(
                      "Inspect connection scope",
                      "연결 범위 확인",
                      "فحص نطاق الاتصال",
                    )}
                  </a>
                </>
              )}
              {index === 1 && (
                <>
                  <p>
                    {c(
                      "Metadata access does not verify setup or operational access. A role name is not evidence of a provider read.",
                      "메타데이터 열람 권한으로 설정·운영 접근을 검증할 수 없습니다. 역할 이름은 Provider 실제 읽기의 근거가 아닙니다.",
                      "الوصول إلى البيانات الوصفية لا يثبت صلاحيات الإعداد أو التشغيل. اسم الدور ليس دليلاً على قراءة من المزود.",
                    )}
                  </p>
                  {facts(
                    [
                      c("Setup owner", "설정 담당자", "مسؤول الإعداد"),
                      c("Export writer", "내보내기 작성자", "كاتب التصدير"),
                      c("Collector reader", "수집 읽기 주체", "قارئ التجميع"),
                    ].map((role) => [role, unknown]),
                  )}
                  <p>
                    {c(
                      "Principal assignment and read-verification evidence are not connected. Task-limited handoff is unavailable: assignment, expiry and revocation contracts are not implemented.",
                      "주체 지정과 실제 접근 검증 근거가 미연결입니다. 담당 지정·만료·회수 계약이 없어 작업 한정 인계를 사용할 수 없습니다.",
                      "تعيين الهوية وأدلة التحقق من القراءة غير متصلين. التسليم المحدود بالمهمة غير متاح لغياب عقود التعيين والانتهاء والسحب.",
                    )}
                  </p>
                </>
              )}
              {index === 2 && (
                <>
                  <p>
                    {c(
                      "Latest registered candidate; not proof of first-ever data, completeness or zero usage.",
                      "최근 등록 후보입니다. 최초 수신·자료 완전성·사용량 0의 근거가 아닙니다.",
                      "أحدث مرشح مسجل؛ ليس دليلاً على أول بيانات أو اكتمالها أو انعدام الاستخدام.",
                    )}
                  </p>
                  {data.latest_candidate ? (
                    facts([
                      [
                        c("Candidate ID", "후보 ID", "معرّف المرشح"),
                        data.latest_candidate.release_id,
                      ],
                      [
                        c("Received at", "수신 시각", "وقت الاستلام"),
                        time(data.latest_candidate.created_at),
                      ],
                      [
                        c(
                          "Source revision / input state",
                          "원본 개정 / 입력 상태",
                          "مراجعة المصدر / حالة الإدخال",
                        ),
                        data.latest_candidate.source_revision +
                          " / " +
                          data.latest_candidate.state,
                      ],
                      [
                        c("Input contract", "입력 계약", "عقد الإدخال"),
                        "FOCUS " + data.latest_candidate.version,
                      ],
                    ])
                  ) : (
                    <p>
                      {c(
                        "No registered candidate. The cause of missing data is unknown.",
                        "등록된 후보가 없습니다. 자료가 없는 원인은 미확인입니다.",
                        "لا يوجد مرشح مسجل. سبب غياب البيانات غير معروف.",
                      )}
                    </p>
                  )}
                  <p>
                    {data.holds.quarantine
                      ? c(
                          "Quarantined input exists.",
                          "격리된 입력이 있습니다.",
                          "توجد مدخلات معزولة.",
                        )
                      : c(
                          "No quarantine record at observation time.",
                          "조회 시점에 격리 기록이 없습니다.",
                          "لا يوجد سجل عزل وقت الفحص.",
                        )}
                  </p>
                  <p>
                    {data.holds.unresolved
                      ? c(
                          "An operation outcome is unresolved.",
                          "결과 미확정 작업이 있습니다.",
                          "توجد عملية غير محسومة.",
                        )
                      : c(
                          "No unresolved operation at observation time.",
                          "조회 시점에 결과 미확정 작업이 없습니다.",
                          "لا توجد عملية غير محسومة وقت الفحص.",
                        )}
                  </p>
                  <div className="onboarding-links">
                    {link(
                      "imports",
                      c("Inspect intake", "수집 결과 확인", "فحص الاستيراد"),
                    )}
                    {link(
                      "source-data",
                      c(
                        "Inspect normalized rows",
                        "정규화 명세 조회",
                        "فحص الصفوف المعيارية",
                      ),
                    )}
                  </div>
                </>
              )}
              {index === 3 && (
                <>
                  <h3>
                    {c(
                      "Reconciliation evidence not checked",
                      "대사 근거 미확인",
                      "أدلة المطابقة غير مفحوصة",
                    )}
                  </h3>
                  <p>
                    {c(
                      "This metadata does not return reconciliation results. S06 compares source and storage; a match does not prove independently verified billing.",
                      "이 메타데이터는 대사 결과를 제공하지 않습니다. S06의 원본·저장 비교가 일치해도 독립적인 청구 검증을 뜻하지 않습니다.",
                      "هذه البيانات الوصفية لا تقدم نتائج المطابقة. يقارن S06 المصدر والتخزين؛ التطابق لا يثبت تحققاً مستقلاً من الفوترة.",
                    )}
                  </p>
                  {destination(data, "reconciliation", locale) ? (
                    link(
                      "reconciliation",
                      c(
                        "Open source and storage reconciliation",
                        "원본·저장 대사 열기",
                        "فتح مطابقة المصدر والتخزين",
                      ),
                    )
                  ) : (
                    <p>
                      {c(
                        "The current service identity cannot inspect reconciliation. Navigation does not grant access.",
                        "현재 실행 신원에는 대사 열람 경로가 없습니다. 화면 이동으로 권한을 부여하지 않습니다.",
                        "هوية الخدمة الحالية لا تستطيع فحص المطابقة. الانتقال لا يمنح صلاحية الوصول.",
                      )}
                    </p>
                  )}
                </>
              )}
              {index === 4 && (
                <>
                  <p>
                    {c(
                      "Receiving data does not authorize activation. No activation request is submitted from this screen.",
                      "자료 수신만으로 개통할 수 없습니다. 이 화면에서는 개통 요청을 제출하지 않습니다.",
                      "استلام البيانات لا يجيز التشغيل. لا تُرسل طلبات تشغيل من هذه الشاشة.",
                    )}
                  </p>
                  {facts(
                    [
                      c("Installation acceptance", "설치 검수", "قبول التثبيت"),
                      c(
                        "Provider operational read",
                        "Provider 운영 읽기",
                        "القراءة التشغيلية من المزود",
                      ),
                      c(
                        "Independent billing verification",
                        "독립 청구 검증",
                        "التحقق المستقل من الفوترة",
                      ),
                      c(
                        "ERP issuance and tax integration",
                        "ERP 발행·세금 연동",
                        "إصدار ERP والتكامل الضريبي",
                      ),
                      c(
                        "Recovery and two-period cutover",
                        "복구·2개 기간 병행 대사",
                        "الاستعادة والمطابقة لفترتين",
                      ),
                      c(
                        "Operational go-live approval",
                        "운영 개통 승인",
                        "اعتماد التشغيل",
                      ),
                    ].map((gate) => [gate, unknown]),
                  )}
                  <p>
                    {c(
                      "Approval, issuance ownership and recovery evidence are not connected. Activation and owner handoff are unavailable.",
                      "승인·발행 주체·복구 근거가 미연결입니다. 개통 요청과 담당자 인계는 사용할 수 없습니다.",
                      "أدلة الاعتماد وملكية الإصدار والاستعادة غير متصلة. التشغيل وتسليم المسؤولية غير متاحين.",
                    )}
                  </p>
                </>
              )}
              <footer className="onboarding-footer">
                {index > 0 && (
                  <a href={stageUrl(stages[index - 1], locale, expected)}>
                    {c("Previous stage", "이전 단계", "المرحلة السابقة")}
                  </a>
                )}
                {index < 4 && (
                  <a
                    className="onboarding-next"
                    href={stageUrl(stages[index + 1], locale, expected)}
                  >
                    {c(
                      "View next stage",
                      "다음 단계 보기",
                      "عرض المرحلة التالية",
                    )}
                  </a>
                )}
              </footer>
            </section>
          </div>
        </>
      )}
    </div>
  );
}
