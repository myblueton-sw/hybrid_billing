import { t, viewFields, type Props } from "./contracts";
import {
  Criteria,
  EmptyTable,
  Evidence,
  Intro,
  Notice,
  Panel,
  useScreenState,
  type InputDef,
} from "./ui";
const inputs: readonly InputDef[] = [
  {
    key: "time",
    label: ["검색 시간 기준", "Search time basis", "أساس وقت البحث"],
    required: true,
    options: [
      { value: "occurred", label: ["발생시각", "Occurred at", "وقت الحدوث"] },
      { value: "received", label: ["수신시각", "Received at", "وقت الاستلام"] },
    ],
  },
  {
    key: "start",
    label: ["시작일 (포함)", "Start date (inclusive)", "تاريخ البداية (مشمول)"],
    type: "date",
    required: true,
  },
  {
    key: "end",
    label: [
      "종료일 (미포함)",
      "End date (exclusive)",
      "تاريخ النهاية (غير مشمول)",
    ],
    type: "date",
    required: true,
  },
  {
    key: "target",
    label: ["업무 대상 참조", "Business target reference", "مرجع هدف العمل"],
    idOnly: true,
  },
  {
    key: "actor",
    label: ["실행 주체 유형", "Principal kind", "نوع الهوية المنفذة"],
    options: [
      { value: "user", label: ["사용자", "User", "المستخدم"] },
      {
        value: "service",
        label: ["서비스 계정", "Service account", "حساب الخدمة"],
      },
      {
        value: "support",
        label: ["지원 대리", "Support delegation", "تفويض الدعم"],
      },
      { value: "ai", label: ["AI 경로", "AI path", "مسار الذكاء الاصطناعي"] },
    ],
  },
  {
    key: "request",
    label: [
      "요청·사건 참조",
      "Request or event reference",
      "مرجع الطلب أو الحدث",
    ],
    idOnly: true,
  },
];
export default function Audit(props: Props) {
  const { locale: l, view, onDirty } = props,
    [, onReview] = useScreenState(props, "R02");
  return (
    <div
      className="fin-screen fin-audit"
      dir={l === "ar" ? "rtl" : "ltr"}
      data-finance-screen="R02"
      data-view={view}
    >
      <Intro
        l={l}
        purpose={[
          "실제 행위자·실행 경로·승인과 외부 결과를 고정 사건과 개정으로 추적합니다.",
          "Trace actual actors, execution paths, approvals and external outcomes through fixed events and revisions.",
          "تتبع الفاعلين الحقيقيين ومسارات التنفيذ والموافقات والنتائج الخارجية بأحداث ومراجعات ثابتة.",
        ]}
      />
      <Criteria
        l={l}
        screenId="R02"
        inputs={inputs}
        onDirty={onDirty}
        onReview={onReview}
      />
      {view === "events" ? (
        <Panel
          title={t(l, [
            "발생과 수신을 구분한 사건",
            "Events with separate occurrence and receipt times",
            "أحداث تميز وقت الحدوث عن الاستلام",
          ])}
        >
          <EmptyTable
            l={l}
            screenId="R02"
            columns={viewFields.R02.events}
            caption={[
              "감사 사건 목록",
              "Audit event list",
              "قائمة أحداث التدقيق",
            ]}
          />
          <Notice
            l={l}
            text={[
              "수신 순서는 실제 업무 시간순을 보장하지 않습니다. 사건 연결은 명시한 correlation 참조를 사용하며 행위자의 현재 재직 여부로 과거 책임을 바꾸지 않습니다.",
              "Receipt order does not establish business chronology. Event linkage requires explicit correlation, and an actor's current employment does not alter historical responsibility.",
              "ترتيب الاستلام لا يثبت التسلسل الزمني للعمل. ربط الأحداث يتطلب مرجع ارتباط صريح وحالة توظيف الفاعل الحالية لا تغير مسؤوليته التاريخية.",
            ]}
          />
        </Panel>
      ) : view === "detail" ? (
        <div className="fin-split">
          <Panel
            title={t(l, [
              "실제 실행 주체",
              "Actual executing principal",
              "الهوية المنفذة الحقيقية",
            ])}
          >
            <Evidence
              l={l}
              screenId="R02"
              keys={["event", "actor", "delegation", "route"]}
            />
          </Panel>
          <Panel
            title={t(l, [
              "접수·승인·실제 결과",
              "Acceptance, approval and actual outcome",
              "القبول والموافقة والنتيجة الفعلية",
            ])}
          >
            <Evidence
              l={l}
              screenId="R02"
              keys={["action", "approval", "external"]}
            />
            <Notice
              l={l}
              text={[
                "AI·지원·자동화 경로는 승인자를 대신하지 않습니다. 요청 접수나 승인은 실제 실행과 외부 반영의 증거가 아닙니다.",
                "AI, support and automation paths do not impersonate approvers. Request acceptance or approval is not evidence of execution or external application.",
                "مسارات الذكاء الاصطناعي والدعم والأتمتة لا تنتحل المعتمدين. قبول الطلب أو الموافقة ليس دليلاً على التنفيذ أو المعالجة الخارجية.",
              ]}
            />
          </Panel>
        </div>
      ) : view === "diff" ? (
        <Panel
          title={t(l, [
            "허용된 개정 차이",
            "Permitted revision differences",
            "فروق المراجعات المسموح بها",
          ])}
        >
          <div className="fin-two">
            <section>
              <h3>{t(l, ["변경 전", "Before", "قبل"])}</h3>
              <Evidence l={l} screenId="R02" keys={["before"]} />
            </section>
            <section>
              <h3>{t(l, ["변경 후", "After", "بعد"])}</h3>
              <Evidence l={l} screenId="R02" keys={["after"]} />
            </section>
          </div>
          <Evidence l={l} screenId="R02" keys={["event", "hash", "approval"]} />
          <Notice
            l={l}
            text={[
              "같은 객체의 허용된 필드만 비교합니다. 현재 조회 권한이 확인되지 않아 원문·차이 값을 표시하지 않습니다. 마스킹된 비밀을 복원하지 않습니다.",
              "Compare only permitted fields of the same object. No source or diff values are shown without verified current read authority. Redacted secrets cannot be restored.",
              "تُقارن الحقول المسموحة للكائن نفسه فقط. لا تُعرض قيم الأصل أو الفروق دون صلاحية قراءة حالية متحققة. لا يمكن استعادة الأسرار المحجوبة.",
            ]}
          />
        </Panel>
      ) : (
        <Panel
          title={t(l, [
            "고정 근거와 보존 상태",
            "Fixed evidence and retention state",
            "الأدلة الثابتة وحالة الاحتفاظ",
          ])}
        >
          <Evidence l={l} screenId="R02" keys={viewFields.R02.evidence} />
          <Notice
            l={l}
            text={[
              "과거 접근이나 승인 해시는 현재 권한을 증명하지 않습니다. 대상 근거를 열거나 감사 묶음을 반출할 때 현재 필드·보존·보류 권한을 다시 검사해야 합니다.",
              "Past access or an approval hash does not establish current authority. Opening evidence or exporting an audit package requires current field, retention and hold checks.",
              "الوصول السابق أو بصمة الموافقة لا يثبتان الصلاحية الحالية. فتح الأدلة أو تصدير حزمة تدقيق يتطلب تحققاً حديثاً من الحقول والاحتفاظ والتعليق.",
            ]}
          />
        </Panel>
      )}
      <details className="fin-requirements">
        <summary>
          {t(l, [
            "감사 조회·근거 열기·감사 묶음 요건",
            "Audit query, evidence and export requirements",
            "متطلبات استعلام التدقيق والأدلة والتصدير",
          ])}
        </summary>
        <p>
          {t(l, [
            "감사 역할은 모든 자료의 전역 읽기 권한이 아닙니다. 조회·개정 비교·근거 열기·감사 묶음은 각 범위의 현재 권한이 필요합니다. 감사 서비스와 반출 작업이 미연결이므로 현재 실행하지 않습니다.",
            "An audit role grants no global access. Queries, revision comparisons, evidence reads and audit exports each require current scoped authority. The audit reader and export jobs are unconnected and are not executed.",
            "دور التدقيق لا يمنح وصولاً شاملاً. الاستعلامات ومقارنة المراجعات وقراءة الأدلة وتصدير التدقيق تتطلب صلاحية حالية محددة النطاق. خدمات القراءة والتصدير غير متصلة ولا تُنفذ.",
          ])}
        </p>
      </details>
    </div>
  );
}
