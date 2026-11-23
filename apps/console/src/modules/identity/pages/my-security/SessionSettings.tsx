import type { Locale } from "../../../../shared/types";
import type { AccessSetting } from "../../../../service/admin";
import { text as t } from "../../../../shared/copy";

const names: Record<string, (l: Locale) => string> = {
  session_query_page_size: l => t(l,"Installation default rows per session page","세션 페이지 설치 기본 행 수","عدد الصفوف الافتراضي للتثبيت في صفحة الجلسات"),
  session_query_scan: l => t(l,"Sessions scanned per request","요청당 조회하는 세션 수","الجلسات المفحوصة في كل طلب"),
  session_query_lifetime_seconds: l => t(l,"Query and preparation lifetime","조회·목록 준비 유효기간","مدة صلاحية الاستعلام والإعداد"),
  session_selection_pending: l => t(l,"Unsubmitted lists per current session","현재 세션당 미제출 목록 수","القوائم غير المرسلة لكل جلسة حالية"),
  session_selection_max_targets: l => t(l,"Targets per confirmed list","확정 목록당 대상 수","الأهداف في كل قائمة مؤكدة"),
  session_record_references: l => t(l,"Shared record capacity","공유 기록 용량","السعة المشتركة للسجلات"),
};
const units: Record<string, (l: Locale) => string> = {
  session_query_page_size: l => t(l,"rows","행","صفوف"),
  session_query_scan: l => t(l,"sessions","세션","جلسات"),
  session_query_lifetime_seconds: l => t(l,"seconds","초","ثوانٍ"),
  session_selection_pending: l => t(l,"lists","목록","قوائم"),
  session_selection_max_targets: l => t(l,"targets","대상","أهداف"),
  session_record_references: l => t(l,"capacity units","용량 단위","وحدات سعة"),
};

export const limitCopy = {
  title: (l: Locale) => t(l,"Effective query and preparation limits","조회·목록 준비 유효 한도","الحدود الفعلية للاستعلام والإعداد"),
  lead: (l: Locale) => t(l,"Read-only values returned by this server. Shared capacity includes records, targets, result rows (including not-included rows) and retry references; it is not the number of available sessions.","이 서버가 반환한 읽기 전용 값입니다. 공유 용량에는 기록·대상·결과 행(미포함 행 포함)·재시도 참조가 포함되며, 사용 가능한 세션 수를 뜻하지 않습니다.","قيم للقراءة فقط يعيدها هذا الخادم. تشمل السعة المشتركة السجلات والأهداف وصفوف النتائج (بما فيها الصفوف غير المشمولة) ومراجع إعادة المحاولة؛ ولا تمثّل عدد الجلسات المتاحة."),
  unavailable: (l: Locale) => t(l,"Effective limits are unavailable.","유효 한도를 확인할 수 없습니다.","الحدود الفعلية غير متاحة."),
  range: (l: Locale) => t(l,"Allowed range:","허용 범위:","النطاق المسموح:"),
  default: (l: Locale) => t(l,"Default:","기본값:","القيمة الافتراضية:"),
  source: (l: Locale) => t(l,"Source:","출처:","المصدر:"),
  sourceDefault: (l: Locale) => t(l,"Installation default","설치 기본값","القيمة الافتراضية للتثبيت"),
  configured: (l: Locale) => t(l,"Installation setting","설치 설정","إعداد التثبيت"),
  unknownSource: (l: Locale) => t(l,"Source unavailable","출처 확인 불가","المصدر غير متاح"),
  expiry: (l: Locale) => t(l,"Preparation expires at:","목록 준비 만료 시각:","تنتهي صلاحية الإعداد في:"),
  expiryNote: (l: Locale) => t(l,"This deadline limits preparation. Accepted results have a separate retention limit and may become unavailable. The server checks validity when you submit.","이 시각은 목록 준비의 유효기간입니다. 접수 완료 결과는 별도 보관 한도가 있어 확인할 수 없게 될 수 있습니다. 제출 시 서버가 유효성을 확인합니다.","يحدّد هذا الموعد صلاحية الإعداد. للنتائج المقبولة حدّ احتفاظ منفصل وقد تصبح غير متاحة. يتحقّق الخادم من الصلاحية عند الإرسال."),
};

export function PreparationExpiry({ l, at }: { l: Locale; at: string }) {
  return <div className="mysec-preparation-expiry"><p>{limitCopy.expiry(l)} <bdi dir="ltr">{at}</bdi></p><p className="live-meta">{limitCopy.expiryNote(l)}</p></div>;
}

export function SessionSettings({ l, settings, scope = "query" }: { l: Locale; settings: AccessSetting[]; scope?: "query" | "preparation" }) {
  const expected = Object.keys(names).slice(0,scope === "query" ? 3 : 6);
  return <details className="live-ended mysec-query-settings"><summary>{limitCopy.title(l)}</summary>
    <p className="live-meta">{limitCopy.lead(l)}</p><dl className="live-facts">{expected.map(name => {
      const row = settings.find(setting => setting.name === name);
      return <div key={name}><dt>{names[name](l)} ({units[name](l)})</dt><dd>{row ? <><strong><bdi dir="ltr">{row.value}</bdi></strong><span className="live-sub">{limitCopy.range(l)} <bdi dir="ltr">{row.min}–{row.max}</bdi> · {limitCopy.default(l)} <bdi dir="ltr">{row.default}</bdi></span><span className="live-sub">{limitCopy.source(l)} {row.source === "start_flag" ? limitCopy.configured(l) : row.source === "default" ? limitCopy.sourceDefault(l) : limitCopy.unknownSource(l)}</span></> : limitCopy.unavailable(l)}</dd></div>;
    })}</dl>
  </details>;
}
