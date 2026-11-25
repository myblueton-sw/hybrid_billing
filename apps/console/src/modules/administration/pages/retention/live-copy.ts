import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  accessWords, blockerWords, classWords, connectionWords, counted, decisionWords, holdOutcomeWords, holdStateWords, keptWords, ledgerWords, narrowWords, originWords, placerWords,
  plurals, populationWords, proposalKindWords, proposalStateWords, resultWords, runOutcomeWords, runStateNotes, runStateWords, standingWords, stateSentenceWords, storeResultWords,
  unknownCode, words,
} from "./live-codes";

export const iso = (s: string) => (s ? "⁨" + s + "⁩" : s);
export const mono = (s: string) => (s ? "" + s + "" : s);
export const plain = (s: string) => s.replace(/[]/g, "");
const has = (table: object, key: string) => Object.prototype.hasOwnProperty.call(table, key);

export const k = {
  deniedTitle: (l: Locale) => t(l, "Retention is not available to you", "데이터 보관·삭제 화면을 볼 수 없습니다", "الاحتفاظ غير متاح لك"),
  denied: (l: Locale) =>
    t(l, "You do not have permission to manage retention in this installation.", "이 설치에서 데이터 보관을 관리할 권한이 없습니다.", "ليست لديك صلاحية إدارة الاحتفاظ في هذا التثبيت."),
  sessionEnded: (l: Locale) => t(l, "Your session has ended.", "세션이 끝났습니다.", "انتهت جلستك."),
  signInAgain: (l: Locale) => t(l, "Sign in again", "다시 로그인", "تسجيل الدخول مجددًا"),
  loading: (l: Locale) =>
    t(l, "Loading retention data… Uncounted values are never shown as 0.", "보관 데이터를 불러오는 중… 집계되지 않은 값은 0으로 표시하지 않습니다.", "جارٍ تحميل بيانات الاحتفاظ… لا تُعرض القيم غير المحسوبة على أنها 0."),
  reload: (l: Locale) => t(l, "Reload", "새로 고침", "إعادة التحميل"),
  workspace: (l: Locale) => t(l, "Workspace ID", "업무 공간 ID", "معرّف مساحة العمل"),
  revision: (l: Locale) => t(l, "Data revision", "데이터 리비전", "مراجعة البيانات"),
  now: (l: Locale) => t(l, "Now (UTC)", "현재(UTC)", "الآن (UTC)"),
  viewTitle: (l: Locale, v: string) =>
    v === "expiry" ? t(l, "Usage and expiry", "사용량·만료", "الاستخدام والانتهاء")
      : v === "holds" ? t(l, "Holds and references", "보류·참조", "التعليقات والمراجع")
        : v === "deletion" ? t(l, "Deletion runs", "삭제 작업", "عمليات الحذف")
          : v === "restore" ? t(l, "Archive restore", "아카이브 복원", "استعادة الأرشيف")
            : v === "decisions" ? t(l, "Decision records and restore freshness", "결정 기록·복원 최신성", "سجلات القرار وحداثة الاستعادة")
              : t(l, "Retention policies", "보관 정책", "سياسات الاحتفاظ"),
  statementLead: (l: Locale) => t(l, "Records decisions only.", "결정만 기록합니다.", "تسجيل القرارات فقط."),
  statement: (l: Locale) =>
    t(
      l,
      "Nothing is deleted, archived, restored or exported from this screen in this build, and no deletion worker runs.",
      "이 빌드에서는 이 화면에서 아무것도 삭제·아카이브·복원·반출하지 않으며 삭제 작업자도 실행되지 않습니다.",
      "لا يُحذف أي شيء ولا يُؤرشف ولا يُستعاد ولا يُصدَّر من هذه الشاشة في هذا الإصدار، ولا تعمل أي جهة حذف.",
    ),
  valuesNote: (l: Locale) =>
    t(
      l,
      "No legal or contractual retention period has been decided. Periods here are values entered by administrators; in the sample data they are synthetic.",
      "법적·계약상 보관 기간은 아직 정해지지 않았습니다. 여기의 기간은 관리자가 입력한 값이며, 샘플 데이터의 값은 가상 값입니다.",
      "لم تُحدَّد بعد أي مدة احتفاظ قانونية أو تعاقدية. المدد هنا قيم يُدخلها المسؤولون، وهي في بيانات العينة قيم افتراضية.",
    ),
  permission: (l: Locale) =>
    t(
      l,
      "Every command here currently needs installation administration. Separate policy, hold, deletion and evidence-access permissions are not available yet, and auditors cannot open this screen yet.",
      "현재 이 화면의 모든 명령에는 설치 관리 권한이 필요합니다. 정책·보존 보류·삭제·증빙 접근 권한은 아직 별도로 제공되지 않으며, 감사 조회자는 아직 이 화면을 열 수 없습니다.",
      "تتطلب كل الأوامر هنا حاليًا صلاحية إدارة التثبيت. صلاحيات السياسات وتعليق الحذف والحذف والوصول إلى الأدلة غير متاحة بشكل منفصل بعد، ولا يستطيع المدققون فتح هذه الشاشة بعد.",
    ),
  notConfigured: (l: Locale) => t(l, "Retention is not configured in this workspace", "이 업무 공간에는 데이터 보관이 설정되어 있지 않습니다", "الاحتفاظ غير مُعدّ في مساحة العمل هذه"),
  stateUnknown: (l: Locale, code: string) =>
    t(l, `The retention read returned an unknown state (${mono(code)}).`, `보관 조회가 알 수 없는 상태(${mono(code)})를 반환했습니다.`, `أعادت قراءة الاحتفاظ حالة غير معروفة (${mono(code)}).`),
  readOnly: (l: Locale) =>
    t(l, "Commands are unavailable because the data revision is unknown. Reload.", "데이터 리비전을 알 수 없어 명령을 사용할 수 없습니다. 새로 고침하세요.", "الأوامر غير متاحة لأن مراجعة البيانات غير معروفة. أعد التحميل."),
  unknown: (l: Locale) => t(l, "Unknown", "알 수 없음", "غير معروف"),
  unknownCode,
  none: (l: Locale) => t(l, "None", "없음", "لا شيء"),
  noneDot: (l: Locale) => t(l, "None.", "없음.", "لا شيء."),
  maybeMore: (l: Locale) => t(l, "There may be more.", "더 있을 수 있습니다.", "قد يوجد المزيد."),
  vanished: (l: Locale) =>
    t(l, "This item is no longer listed. Your earlier request may have been applied.", "이 항목은 더 이상 목록에 없습니다. 이전 요청이 이미 반영되었을 수 있습니다.", "لم يعد هذا العنصر مدرجًا. ربما طُبّق طلبك السابق."),
  vanishedProposal: (l: Locale) =>
    t(l, "Your earlier request may have been applied. The proposal is no longer listed.", "이전 요청이 이미 반영되었을 수 있습니다. 제안이 더 이상 목록에 없습니다.", "ربما طُبّق طلبك السابق. لم يعد المقترح مدرجًا."),
  dismissNotice: (l: Locale) => t(l, "Dismiss", "닫기", "إغلاق"),
  mayApplied: (l: Locale) =>
    t(l, "Your earlier request may have been applied. Check this proposal's state.", "이전 요청이 이미 반영되었을 수 있습니다. 이 제안의 상태를 확인하세요.", "ربما طُبّق طلبك السابق. تحقّق من حالة هذا المقترح."),
  days: (l: Locale) => t(l, "days", "일", "يومًا"),

  policiesLead: (l: Locale) =>
    t(
      l,
      "A policy sets how long each data class is kept, counted from each item's start time. Approving a policy adds a revision; it deletes nothing. Policy scope by jurisdiction or contract and archive rules are not connected in this build.",
      "정책은 각 자료 등급을 항목별 기산 시각부터 얼마나 보관할지 정합니다. 정책을 승인하면 새 revision이 추가될 뿐 아무것도 삭제하지 않습니다. 이 빌드에서는 관할·계약별 정책 범위와 아카이브 기준이 연결되어 있지 않습니다.",
      "تحدد السياسة مدة الاحتفاظ بكل فئة بيانات بدءًا من وقت بدء كل عنصر. الموافقة على سياسة تضيف مراجعة جديدة ولا تحذف شيئًا. نطاق السياسة حسب الاختصاص أو العقد وقواعد الأرشفة غير متصلة في هذا الإصدار.",
    ),
  expiryLead: (l: Locale) =>
    t(
      l,
      "Shows which registered items are due for deletion and why the others are not. Usage, growth and storage size are not connected, and nothing is deleted from here.",
      "등록된 항목 중 삭제 대상과 그렇지 않은 이유를 보여줍니다. 사용량·증가량·저장 크기는 연결되어 있지 않으며 여기에서는 아무것도 삭제하지 않습니다.",
      "يعرض العناصر المسجّلة المستحقة للحذف وسبب عدم استحقاق غيرها. الاستخدام والنمو وحجم التخزين غير متصلة، ولا يُحذف أي شيء من هنا.",
    ),
  holdsLeadA: (l: Locale) => t(l, "A hold stops deletion of covered items that still exist.", "보존 보류는 대상 항목 중 아직 남아 있는 항목의 삭제를 막습니다.", "يمنع تعليق الحذف حذف العناصر المشمولة التي ما زالت موجودة."),
  holdsLeadB: (l: Locale) =>
    t(
      l,
      "It cannot bring back items that are already deleted, being deleted or partly deleted.",
      "이미 삭제되었거나 삭제 중이거나 일부만 삭제된 항목은 되살리지 못합니다.",
      "ولا يمكنه إعادة العناصر المحذوفة أو قيد الحذف أو المحذوفة جزئيًا.",
    ),
  holdsLeadC: (l: Locale) =>
    t(
      l,
      "Releasing a hold needs two administrators who did not place it: one proposes, the other approves.",
      "보류를 해제하려면 보류를 등록하지 않은 관리자 두 명이 필요합니다. 한 명이 제안하고 다른 한 명이 승인합니다.",
      "يتطلب رفع التعليق مسؤولَين لم يضعاه: أحدهما يقترح والآخر يوافق.",
    ),
  deletionLead: (l: Locale) =>
    t(
      l,
      "Deletion runs were recorded by the sample data. Proposing or approving a deletion plan, the deletion worker, retries and backup expiry are not connected in this build, so a running run does not progress. Stopping a run releases the items no worker has claimed; it restores nothing.",
      "삭제 작업은 샘플 데이터로 기록되었습니다. 이 빌드에서는 삭제 계획 제안·승인, 삭제 작업자, 재시도, 백업 만료가 연결되어 있지 않아 실행 중인 작업도 진행되지 않습니다. 작업을 중지하면 작업자가 아직 맡지 않은 항목이 작업에서 풀려날 뿐 아무것도 복원하지 않습니다.",
      "سُجّلت عمليات الحذف من بيانات العينة. اقتراح خطة حذف والموافقة عليها وجهة الحذف وإعادة المحاولة وانتهاء النسخ الاحتياطية غير متصلة في هذا الإصدار، لذلك لا تتقدم العملية الجارية. إيقاف العملية يحرر العناصر التي لم تتسلمها أي جهة بعد، ولا يستعيد شيئًا.",
    ),
  restoreNotConnected: (l: Locale) =>
    t(
      l,
      "Archive restore is not connected in this build. No archive exists, nothing is restored, and no temporary copy is made.",
      "이 빌드에서는 아카이브 복원이 연결되어 있지 않습니다. 아카이브가 없으며 아무것도 복원하지 않고 임시 사본도 만들지 않습니다.",
      "استعادة الأرشيف غير متصلة في هذا الإصدار. لا يوجد أرشيف، ولا تتم أي استعادة، ولا تُنشأ أي نسخة مؤقتة.",
    ),
  decisionsLead: (l: Locale) =>
    t(
      l,
      "Deletion records and the decision watermark come from the in-memory retention register, which resets with the process. An independent durable decision ledger, epoch, durable commit and restore freshness checks are not connected, so no deletion is shown as confirmed.",
      "삭제 기록과 결정 워터마크는 프로세스와 함께 초기화되는 메모리 내 보관 레지스트리에서 읽습니다. 독립 결정 원장, epoch, 내구 확정, 복원 최신성 검사가 연결되어 있지 않아 어떤 삭제도 확정으로 표시하지 않습니다.",
      "تأتي سجلات الحذف وعلامة القرار المائية من سجل الاحتفاظ داخل الذاكرة، الذي يُعاد تعيينه مع العملية. سجل القرارات الدائم المستقل والحقبة والتثبيت الدائم وفحوص حداثة الاستعادة غير متصلة، لذلك لا يُعرض أي حذف على أنه مؤكد.",
    ),

  policiesCaption: (l: Locale) => t(l, "Retention policies by data class", "자료 등급별 보관 정책", "سياسات الاحتفاظ حسب فئة البيانات"),
  colClass: (l: Locale) => t(l, "Data class", "자료 등급", "فئة البيانات"),
  colRevision: (l: Locale) => t(l, "Revision", "Revision", "المراجعة"),
  colPeriod: (l: Locale) => t(l, "Period (days)", "기간(일)", "المدة (أيام)"),
  colGrace: (l: Locale) => t(l, "Grace (days)", "유예(일)", "السماح (أيام)"),
  colTotal: (l: Locale) => t(l, "Total (days)", "합계(일)", "المجموع (أيام)"),
  colStarts: (l: Locale) => t(l, "Starts from", "기산 기준", "يبدأ من"),
  colScope: (l: Locale) => t(l, "Scope · archive rule", "범위·아카이브 기준", "النطاق · قاعدة الأرشفة"),
  colApproved: (l: Locale) => t(l, "Approved", "승인", "الاعتماد"),
  colAction: (l: Locale) => t(l, "Action", "작업", "الإجراء"),
  colPolicy: (l: Locale) => t(l, "Policy", "정책", "السياسة"),
  noPolicy: (l: Locale) => t(l, "No policy: items of this class never become due", "정책 없음: 이 등급의 항목은 삭제 대상이 되지 않습니다", "لا توجد سياسة: لا تصبح عناصر هذه الفئة مستحقة أبدًا"),
  scopeLine: (l: Locale, v: string) => t(l, `Scope: ${v}`, `범위: ${v}`, `النطاق: ${v}`),
  archiveLine: (l: Locale, v: string) => t(l, `Archive rule: ${v}`, `아카이브 기준: ${v}`, `قاعدة الأرشفة: ${v}`),
  proposedBy: (l: Locale, who: string) => t(l, `Proposed by ${mono(who)}`, `제안자 ${mono(who)}`, `اقترحه ${mono(who)}`),
  proposeChange: (l: Locale) => t(l, "Propose a change", "변경 제안", "اقتراح تغيير"),
  alreadyPolicy: (l: Locale, n: string, own: boolean) =>
    !n
      ? t(l, "A proposal is already waiting for approval.", "제안이 이미 승인을 기다리고 있습니다.", "يوجد مقترح بانتظار الاعتماد بالفعل.")
      : own
        ? t(l, `Your policy proposal ${mono("#" + n)} for this class is waiting for another administrator's approval.`, `이 등급에 대한 귀하의 정책 제안 ${mono("#" + n)}이(가) 다른 관리자의 승인을 기다리고 있습니다.`, `مقترح السياسة الخاص بك ${mono("#" + n)} لهذه الفئة بانتظار اعتماد مسؤول آخر.`)
        : t(l, `Policy proposal ${mono("#" + n)} for this class is already waiting for approval.`, `이 등급에 대한 정책 제안 ${mono("#" + n)}이(가) 이미 승인을 기다리고 있습니다.`, `مقترح السياسة ${mono("#" + n)} لهذه الفئة بانتظار الاعتماد بالفعل.`),
  alreadyRelease: (l: Locale, n: string, own: boolean) =>
    !n
      ? t(l, "A proposal is already waiting for approval.", "제안이 이미 승인을 기다리고 있습니다.", "يوجد مقترح بانتظار الاعتماد بالفعل.")
      : own
        ? t(l, `Your release proposal ${mono("#" + n)} for this hold is waiting for another administrator's approval.`, `이 보존 보류에 대한 귀하의 해제 제안 ${mono("#" + n)}이(가) 다른 관리자의 승인을 기다리고 있습니다.`, `مقترح الرفع الخاص بك ${mono("#" + n)} لهذا التعليق بانتظار اعتماد مسؤول آخر.`)
        : t(l, `Release proposal ${mono("#" + n)} is waiting for approval.`, `해제 제안 ${mono("#" + n)}이(가) 승인을 기다리고 있습니다.`, `مقترح الرفع ${mono("#" + n)} بانتظار الاعتماد.`),

  policyProposalsTitle: (l: Locale) => t(l, "Policy proposals", "정책 제안", "مقترحات السياسات"),
  releaseProposalsTitle: (l: Locale) => t(l, "Release proposals", "해제 제안", "مقترحات الرفع"),
  planProposalsTitle: (l: Locale) => t(l, "Deletion plan proposals", "삭제 계획 제안", "مقترحات خطط الحذف"),
  noProposals: (l: Locale, kind: string) =>
    kind === "policy" ? t(l, "No policy proposals.", "정책 제안이 없습니다.", "لا توجد مقترحات سياسات.")
      : kind === "release" ? t(l, "No release proposals.", "해제 제안이 없습니다.", "لا توجد مقترحات رفع.")
        : t(l, "No deletion plan proposals.", "삭제 계획 제안이 없습니다.", "لا توجد مقترحات خطط حذف."),
  planReadOnly: (l: Locale) =>
    t(
      l,
      "Deletion plans are not connected: these were recorded by the sample data. Approving or withdrawing them is not available here.",
      "삭제 계획은 연결되어 있지 않습니다. 아래 항목은 샘플 데이터로 기록되었습니다. 여기에서는 승인하거나 철회할 수 없습니다.",
      "خطط الحذف غير متصلة: سُجّلت هذه من بيانات العينة. اعتمادها أو سحبها غير متاح هنا.",
    ),
  waitingGroup: (l: Locale, n: number) => t(l, `Waiting for approval (${n})`, `승인 대기(${n})`, `بانتظار الاعتماد (${n})`),
  closedGroup: (l: Locale, n: number) => t(l, `Closed (${n})`, `종료(${n})`, `مغلقة (${n})`),
  proposalN: (l: Locale, n: string) => t(l, `Proposal ${n}`, `제안 ${n}`, `المقترح ${n}`),
  created: (l: Locale, at: string) => t(l, `Created ${iso(at)}`, `생성 ${iso(at)}`, `أُنشئ ${iso(at)}`),
  reasonLabel: (l: Locale) => t(l, "Reason", "사유", "السبب"),
  noReason: (l: Locale) => t(l, "No reason recorded", "기록된 사유 없음", "لا يوجد سبب مسجّل"),
  approvedBy: (l: Locale, who: string) => t(l, `Approved by ${mono(who)}`, `승인자 ${mono(who)}`, `اعتمده ${mono(who)}`),
  withdrawnBy: (l: Locale, who: string) => t(l, `Withdrawn by ${mono(who)}`, `철회자 ${mono(who)}`, `سحبه ${mono(who)}`),
  closingReason: (l: Locale) => t(l, "Closing reason", "종료 사유", "سبب الإغلاق"),
  retiredRow: (l: Locale) =>
    t(l, "Retired: the retention register no longer holds this proposal", "폐기됨: 보관 레지스트리에 이 제안이 더 이상 없습니다", "مُسقَط: لم يعد سجل الاحتفاظ يحتفظ بهذا المقترح"),
  appliedNote: (l: Locale) =>
    t(l, "Applied means recorded in the retention register; it deletes nothing.", "적용됨은 보관 레지스트리에 기록되었다는 뜻이며 아무것도 삭제하지 않습니다.", "«مطبّق» تعني أنه سُجّل في سجل الاحتفاظ، ولا يحذف شيئًا."),
  impactDigest: (l: Locale) => t(l, "Impact digest", "영향 요약값", "ملخص الأثر"),
  approve: (l: Locale) => t(l, "Approve", "승인", "اعتماد"),
  withdraw: (l: Locale) => t(l, "Withdraw", "철회", "سحب"),
  approveHeading: (l: Locale, n: string) => t(l, `Approve proposal ${n}`, `제안 ${n} 승인`, `اعتماد المقترح ${n}`),
  withdrawHeading: (l: Locale, n: string) => t(l, `Withdraw proposal ${n}`, `제안 ${n} 철회`, `سحب المقترح ${n}`),
  confirmApprove: (l: Locale) => t(l, "Check before you approve", "승인하기 전에 확인하세요", "تحقّق قبل الاعتماد"),
  confirmWithdraw: (l: Locale) => t(l, "Check before you withdraw", "철회하기 전에 확인하세요", "تحقّق قبل السحب"),

  populationNote: (l: Locale, code: string) =>
    code === "registered_sample_artifacts"
      ? t(
          l,
          "Counts cover only the sample items registered in the retention register for this workspace, not all data in the installation.",
          "건수는 이 업무 공간에 대해 보관 레지스트리에 등록된 샘플 항목만 포함하며 설치의 전체 데이터가 아닙니다.",
          "تشمل الأعداد فقط عناصر العينة المسجّلة في سجل الاحتفاظ لمساحة العمل هذه، وليس كل بيانات التثبيت.",
        )
      : !code
        ? t(l, "Counted population: Unknown", "집계 대상: 알 수 없음", "المجموعة المحسوبة: غير معروف")
        : t(l, `Counted population: ${unknownCode(l, mono(code))}`, `집계 대상: ${unknownCode(l, mono(code))}`, `المجموعة المحسوبة: ${unknownCode(l, mono(code))}`),
  decisionNote: (l: Locale) =>
    t(
      l,
      "No durable decision record is connected. A deletion or access block shown here is recorded only in the in-memory register and is not confirmed.",
      "내구성 있는 결정 기록이 연결되어 있지 않습니다. 여기에 표시된 삭제나 접근 차단은 메모리 내 레지스트리에만 기록되었으며 확정되지 않았습니다.",
      "لا يوجد سجل قرارات دائم متصل. أي حذف أو منع وصول معروض هنا مسجّل فقط في السجل داخل الذاكرة وغير مؤكد.",
    ),
  jumpsLabel: (l: Locale) => t(l, "Data classes", "자료 등급", "فئات البيانات"),
  usage: (l: Locale) => t(l, "Usage", "사용량", "الاستخدام"),
  growth: (l: Locale) => t(l, "Growth", "증가량", "النمو"),
  bytes: (l: Locale) => t(l, "Storage size", "저장 크기", "حجم التخزين"),
  noItems: (l: Locale) => t(l, "No registered items in this class.", "이 등급에 등록된 항목이 없습니다.", "لا توجد عناصر مسجّلة في هذه الفئة."),
  itemsCaption: (l: Locale, cls: string) => t(l, `Items in ${cls}`, `${cls}의 항목`, `العناصر في ${cls}`),
  colItem: (l: Locale) => t(l, "Item", "항목", "العنصر"),
  colDue: (l: Locale) => t(l, "Due / Start", "만료 / 기산", "الاستحقاق / البدء"),
  colStores: (l: Locale) => t(l, "Stores", "저장소", "المخازن"),
  colAccess: (l: Locale) => t(l, "Access", "접근", "الوصول"),
  colStanding: (l: Locale) => t(l, "Standing", "삭제 판정", "الوضع"),
  colRun: (l: Locale) => t(l, "Run", "작업", "العملية"),
  shared: (l: Locale) => t(l, "Shared with another workspace", "다른 업무 공간과 공유됨", "مشترك مع مساحة عمل أخرى"),
  never: (l: Locale) => t(l, "Never (no policy)", "없음(정책 없음)", "أبدًا (لا توجد سياسة)"),
  start: (l: Locale) => t(l, "Start", "기산", "البدء"),
  runOutside: (l: Locale) => t(l, "Owned by a run that is not shown in this workspace", "이 업무 공간에 표시되지 않는 작업에 속함", "تملكه عملية غير معروضة في مساحة العمل هذه"),
  placeHold: (l: Locale) => t(l, "Place hold", "보존 보류 등록", "وضع تعليق الحذف"),
  tagDamaged: (l: Locale) => t(l, "Partly deleted (damaged)", "일부 삭제됨(손상)", "محذوف جزئيًا (تالف)"),
  decisionUnknown: (l: Locale) => t(l, "Decision record: Unknown", "결정 기록: 알 수 없음", "سجل القرار: غير معروف"),
  decisionCode: (l: Locale, code: string) => t(l, `Decision record: ${unknownCode(l, mono(code))}`, `결정 기록: ${unknownCode(l, mono(code))}`, `سجل القرار: ${unknownCode(l, mono(code))}`),

  placeWorkspaceHold: (l: Locale) => t(l, "Place a workspace hold", "업무 공간 보존 보류 등록", "وضع تعليق حذف لمساحة العمل"),
  tenantNote: (l: Locale) =>
    t(l, "Covers every registered item of this workspace, including items registered later.", "이 업무 공간에 등록된 모든 항목과 이후 등록되는 항목까지 포함합니다.", "يشمل كل عنصر مسجّل في مساحة العمل هذه، بما في ذلك العناصر المسجّلة لاحقًا."),
  heldShown: (l: Locale, n: number, m: number) => t(l, `Showing ${n} of ${m} released holds, newest first.`, `해제된 보존 보류 ${m}건 중 ${n}건을 최신순으로 표시합니다.`, `تُعرض تعليقات الحذف المرفوعة: ${n} من أصل ${m}، الأحدث أولًا.`),
  heldAll: (l: Locale, m: number) => t(l, `Showing all ${m} released holds, newest first.`, `해제된 보존 보류 ${m}건을 모두 최신순으로 표시합니다.`, `تُعرض جميع تعليقات الحذف المرفوعة (${m})، الأحدث أولًا.`),
  heldInForce: (l: Locale, n: number) => t(l, `In force (${n})`, `유지 중(${n})`, `سارية (${n})`),
  heldReleased: (l: Locale, n: number) => t(l, `Released (${n})`, `해제됨(${n})`, `مرفوعة (${n})`),
  heldNoneInForce: (l: Locale) => t(l, "No holds are in force.", "유지 중인 보존 보류가 없습니다.", "لا توجد تعليقات حذف سارية."),
  heldNoneReleased: (l: Locale) => t(l, "No released holds.", "해제된 보존 보류가 없습니다.", "لا توجد تعليقات حذف مرفوعة."),
  heldMore: (l: Locale) => t(l, "Load older released holds", "이전 해제된 보존 보류 불러오기", "تحميل تعليقات حذف مرفوعة أقدم"),
  heldLoading: (l: Locale) => t(l, "Loading…", "불러오는 중…", "جارٍ التحميل…"),
  heldFailed: (l: Locale) => t(l, "Older released holds could not be loaded. The holds already shown are unchanged.", "이전 해제된 보존 보류를 불러오지 못했습니다. 이미 표시된 보존 보류는 그대로입니다.", "تعذّر تحميل تعليقات الحذف المرفوعة الأقدم. لم تتغيّر تعليقات الحذف المعروضة."),
  heldTryAgain: (l: Locale) => t(l, "Try again", "다시 시도", "إعادة المحاولة"),
  heldReset: (l: Locale) => t(l, "The list was refreshed. The older released holds you loaded are no longer shown.", "목록을 새로 고쳤습니다. 불러온 이전 해제된 보존 보류는 더 이상 표시되지 않습니다.", "حُدِّثت القائمة. لم تعد تعليقات الحذف المرفوعة الأقدم التي حمّلتها معروضة."),
  noHolds: (l: Locale) => t(l, "No holds in this workspace.", "이 업무 공간에 보존 보류가 없습니다.", "لا توجد تعليقات حذف في مساحة العمل هذه."),
  factOwner: (l: Locale) => t(l, "Owner", "책임자", "المالك"),
  factPlacedBy: (l: Locale) => t(l, "Placed by", "등록자", "وضعه"),
  factPlaced: (l: Locale) => t(l, "Placed", "등록 시각", "وقت الوضع"),
  factHoldReason: (l: Locale) => t(l, "Hold reason", "보존 보류 사유", "سبب التعليق"),
  factPlacingReason: (l: Locale) => t(l, "Reason for placing", "등록 처리 사유", "سبب الوضع"),
  covers: (l: Locale) => t(l, "Covers", "대상", "يشمل"),
  thisWorkspace: (l: Locale) => t(l, "This workspace", "이 업무 공간", "مساحة العمل هذه"),
  allClasses: (l: Locale) => t(l, "all classes", "모든 등급", "كل الفئات"),
  nothingCovered: (l: Locale) => t(l, "Nothing in this workspace", "이 업무 공간에 해당 항목 없음", "لا شيء في مساحة العمل هذه"),
  cannotPreserve: (l: Locale) => t(l, "What this hold cannot preserve", "이 보존 보류가 보존하지 못하는 항목", "ما لا يستطيع هذا التعليق حفظه"),
  couldNotPreserve: (l: Locale) =>
    t(l, "While in force, this hold could not preserve…", "유지되는 동안 이 보존 보류가 보존하지 못한 항목…", "أثناء سريانه، لم يستطع هذا التعليق حفظ…"),
  otherScopeCounts: (l: Locale) =>
    t(l, "Not shown: this hold also covers data outside this workspace.", "표시 안 함: 이 보존 보류는 이 업무 공간 밖의 자료도 포함합니다.", "غير معروضة: يشمل هذا التعليق أيضًا بيانات خارج مساحة العمل هذه."),
  beingDeleted: (l: Locale) => t(l, "Being deleted (outcome pending)", "삭제 중(결과 대기)", "قيد الحذف (النتيجة معلّقة)"),
  colOutcome: (l: Locale) => t(l, "Outcome", "결과", "النتيجة"),
  colStoreResults: (l: Locale) => t(l, "Store results", "저장소별 결과", "نتائج المخازن"),
  onlyListed: (l: Locale) =>
    t(
      l,
      "Only items this hold cannot preserve, or whose deletion outcome is pending, are listed. Items it does preserve are not listed.",
      "이 보존 보류가 보존하지 못하거나 삭제 결과가 대기 중인 항목만 표시합니다. 보존되는 항목은 표시하지 않습니다.",
      "تُدرج فقط العناصر التي لا يستطيع هذا التعليق حفظها أو التي نتيجة حذفها معلّقة. لا تُدرج العناصر التي يحفظها.",
    ),
  moreOutcomes: (l: Locale) => t(l, "More items are not shown.", "더 많은 항목은 표시하지 않습니다.", "لا تُعرض عناصر أخرى."),
  noneListed: (l: Locale) => t(l, "None listed.", "표시할 항목 없음.", "لا شيء مدرج."),
  requiredTitle: (l: Locale) => t(l, "Required items", "참조 중인 항목", "العناصر المطلوبة"),
  requiredLead: (l: Locale) =>
    t(
      l,
      "Another item requires this one, so it is not deleted while that item exists. The requiring item is not named.",
      "다른 항목이 이 항목을 참조하므로 그 항목이 있는 동안에는 삭제하지 않습니다. 참조하는 항목은 표시하지 않습니다.",
      "يتطلب عنصر آخر هذا العنصر، لذا لا يُحذف ما دام ذلك العنصر موجودًا. لا يُذكر العنصر الطالب.",
    ),
  requiredCaption: (l: Locale) => t(l, "Items required by another item", "다른 항목이 참조하는 항목", "عناصر يتطلبها عنصر آخر"),
  outcomesCaption: (l: Locale) => t(l, "Items this hold cannot preserve", "보존하지 못하는 항목", "العناصر التي لا يستطيع التعليق حفظها"),
  proposeRelease: (l: Locale) => t(l, "Propose release", "해제 제안", "اقتراح الرفع"),

  unconnectedTitle: (l: Locale) => t(l, "Not connected in this build", "이 빌드에서 연결되지 않음", "غير متصل في هذا الإصدار"),
  worker: (l: Locale) => t(l, "Deletion worker", "삭제 작업자", "جهة الحذف"),
  retry: (l: Locale) => t(l, "Retry", "재시도", "إعادة المحاولة"),
  backupExpiry: (l: Locale) => t(l, "Backup expiry", "백업 만료", "انتهاء النسخ الاحتياطية"),
  planProposals: (l: Locale) => t(l, "Deletion plan proposals", "삭제 계획 제안", "مقترحات خطط الحذف"),
  noRuns: (l: Locale) => t(l, "No deletion runs in this workspace.", "이 업무 공간에 삭제 작업이 없습니다.", "لا توجد عمليات حذف في مساحة العمل هذه."),
  factClass: (l: Locale) => t(l, "Data class", "자료 등급", "فئة البيانات"),
  factPolicyRevision: (l: Locale) => t(l, "Policy revision", "정책 revision", "مراجعة السياسة"),
  factDecision: (l: Locale) => t(l, "Decision record", "결정 기록", "سجل القرار"),
  decisionNotConnected: (l: Locale) => t(l, "Not connected (not confirmed)", "미연결(미확정)", "غير متصل (غير مؤكد)"),
  factChunks: (l: Locale) => t(l, "Chunks", "묶음", "الدفعات"),
  factChunkSize: (l: Locale) => t(l, "Chunk size", "묶음 크기", "حجم الدفعة"),
  factLease: (l: Locale) => t(l, "Lease", "점유 시간", "مدة الحجز"),
  seconds: (l: Locale, n: number) => t(l, `${n} s`, `${n}초`, `${n} ث`),
  factDamaged: (l: Locale) => t(l, "Damaged items", "손상 항목", "العناصر التالفة"),
  factProposedBy: (l: Locale) => t(l, "Proposed by", "제안자", "اقترحه"),
  factApprovedBy: (l: Locale) => t(l, "Approved by", "승인자", "اعتمده"),
  factStoppedBy: (l: Locale) => t(l, "Stopped by", "중지자", "أوقفه"),
  notStopped: (l: Locale) => t(l, "Not stopped", "중지되지 않음", "لم تُوقف"),
  factPlanReason: (l: Locale) => t(l, "Plan reason", "계획 사유", "سبب الخطة"),
  stoppedBy: (l: Locale, who: string) =>
    t(l, `Stopped by ${mono(who)}: unclaimed items were released, not deleted.`, `${mono(who)}이(가) 중지함: 맡지 않은 항목은 삭제되지 않고 풀려났습니다.`, `أوقفها ${mono(who)}: حُرّرت العناصر غير المستلمة ولم تُحذف.`),
  outcomeCounts: (l: Locale) => t(l, "Outcome counts", "결과별 건수", "أعداد النتائج"),
  perStore: (l: Locale) => t(l, "Per store", "저장소별", "حسب المخزن"),
  storesCaption: (l: Locale) => t(l, "Counts per store", "저장소별 건수", "الأعداد حسب المخزن"),
  sumWarn: (l: Locale) => t(l, "These counts do not add up; treat them as unknown.", "이 건수는 합계가 맞지 않습니다. 알 수 없는 값으로 보세요.", "هذه الأعداد لا تتطابق في مجموعها؛ اعتبرها غير معروفة."),
  sumNote: (l: Locale) =>
    t(
      l,
      "Target = deleted + failed + excluded + pending. Excluded items are held, required, not due or stopped.",
      "대상 = 삭제 + 실패 + 제외 + 대기. 제외 항목은 보존 보류·참조·기한 전·중지 항목입니다.",
      "الهدف = محذوف + فشل + مستبعد + معلّق. العناصر المستبعدة معلّقة الحذف أو مطلوبة أو لم يحن موعدها أو موقوفة.",
    ),
  outsideNote: (l: Locale) =>
    t(
      l,
      "Copies already sent to ERP or downloaded are outside the controlled scope: deleting here does not delete them.",
      "ERP로 이미 보낸 사본과 다운로드한 사본은 통제 범위 밖입니다. 여기서 삭제해도 그 사본은 삭제되지 않습니다.",
      "النسخ المرسلة مسبقًا إلى ERP أو التي نُزِّلت خارج النطاق المتحكَّم فيه: الحذف هنا لا يحذفها.",
    ),
  aiNote: (l: Locale) =>
    t(
      l,
      "AI usage records have no field for prompt or response bodies. Assistant conversations are their own data class: deleting them does not delete the records that refer to them. AI usage intake and the Assistant are not connected in this build.",
      "AI 사용량 기록에는 프롬프트·응답 본문 필드가 없습니다. Assistant 대화는 별도 자료 등급이며, 대화를 삭제해도 대화를 참조하는 기록은 삭제되지 않습니다. 이 빌드에서는 AI 사용량 수집과 Assistant가 연결되어 있지 않습니다.",
      "لا تحتوي سجلات استخدام الذكاء الاصطناعي على حقل لنصوص الموجّهات أو الردود. محادثات المساعد فئة بيانات مستقلة، وحذفها لا يحذف السجلات التي تشير إليها. استلام استخدام الذكاء الاصطناعي والمساعد غير متصلين في هذا الإصدار.",
    ),
  backupNote: (l: Locale) => t(l, "Backup copies are not tracked (backup expiry not connected).", "백업 사본은 추적하지 않습니다(백업 만료 미연결).", "لا تُتتبَّع النسخ الاحتياطية (انتهاء النسخ الاحتياطية غير متصل)."),
  colStore: (l: Locale) => t(l, "Store", "저장소", "المخزن"),
  colTarget: (l: Locale) => t(l, "Target", "대상", "الهدف"),
  colDeleted: (l: Locale) => t(l, "Deleted", "삭제", "محذوف"),
  colFailed: (l: Locale) => t(l, "Failed", "실패", "فشل"),
  colExcluded: (l: Locale) => t(l, "Excluded", "제외", "مستبعد"),
  colPending: (l: Locale) => t(l, "Pending", "대기", "معلّق"),
  itemsTitle: (l: Locale) => t(l, "Items", "항목", "العناصر"),
  runItemsCaption: (l: Locale) => t(l, "Items in this run", "이 작업의 항목", "عناصر هذه العملية"),
  stopRun: (l: Locale) => t(l, "Stop run", "작업 중지", "إيقاف العملية"),

  factWatermark: (l: Locale) => t(l, "Decision watermark (installation-wide)", "결정 워터마크(설치 전체)", "علامة القرار المائية (على مستوى التثبيت)"),
  factLedger: (l: Locale) => t(l, "Ledger", "원장", "السجل"),
  factEpoch: (l: Locale) => t(l, "Epoch", "Epoch", "الحقبة"),
  factDurable: (l: Locale) => t(l, "Durable commit", "내구 확정", "التثبيت الدائم"),
  factRestoreCheck: (l: Locale) => t(l, "Restore freshness check", "복원 최신성 검사", "فحص حداثة الاستعادة"),
  deletionRecords: (l: Locale) => t(l, "Deletion records", "삭제 기록", "سجلات الحذف"),
  skipNote: (l: Locale) => t(l, "Sequence numbers are installation-wide, so numbers can be skipped.", "순번은 설치 전체에서 매기므로 번호가 건너뛸 수 있습니다.", "أرقام التسلسل على مستوى التثبيت، لذا قد تُتخطى بعض الأرقام."),
  noRecords: (l: Locale) => t(l, "No deletion records in this workspace.", "이 업무 공간에 삭제 기록이 없습니다.", "لا توجد سجلات حذف في مساحة العمل هذه."),
  newest50: (l: Locale) => t(l, "Showing the newest 50.", "최신 50건을 표시합니다.", "عرض أحدث 50."),
  colSequence: (l: Locale) => t(l, "Sequence", "순번", "التسلسل"),
  colRecorded: (l: Locale) => t(l, "Recorded", "기록 시각", "وقت التسجيل"),
  colDecision: (l: Locale) => t(l, "Decision record", "결정 기록", "سجل القرار"),
  recordsCaption: (l: Locale) => t(l, "Deletion records in this workspace", "이 업무 공간의 삭제 기록", "سجلات الحذف في مساحة العمل هذه"),

  holdHeading: (l: Locale, id: string) => t(l, `Place a hold on ${mono(id)}`, `${mono(id)}에 보존 보류 등록`, `وضع تعليق حذف على ${mono(id)}`),
  workspaceHoldHeading: (l: Locale) => t(l, "Place a workspace hold", "업무 공간 보존 보류 등록", "وضع تعليق حذف لمساحة العمل"),
  proposeHeading: (l: Locale, cls: string) => t(l, `Propose a change to ${cls}`, `${cls} 변경 제안`, `اقتراح تغيير على ${cls}`),
  releaseHeading: (l: Locale, id: string) => t(l, `Propose release of ${mono(id)}`, `${mono(id)} 해제 제안`, `اقتراح رفع ${mono(id)}`),
  stopHeading: (l: Locale, run: string) => t(l, `Stop ${mono(run)}`, `${mono(run)} 중지`, `إيقاف ${mono(run)}`),
  confirmPropose: (l: Locale) => t(l, "Check before you propose this change", "변경을 제안하기 전에 확인하세요", "تحقّق قبل اقتراح هذا التغيير"),
  confirmHold: (l: Locale) => t(l, "Check before you place this hold", "보존 보류를 등록하기 전에 확인하세요", "تحقّق قبل وضع هذا التعليق"),
  confirmRelease: (l: Locale) => t(l, "Check before you propose this release", "해제를 제안하기 전에 확인하세요", "تحقّق قبل اقتراح هذا الرفع"),
  confirmStop: (l: Locale) => t(l, "Check before you stop this run", "작업을 중지하기 전에 확인하세요", "تحقّق قبل إيقاف هذه العملية"),
  proposeVerb: (l: Locale) => t(l, "Propose this change", "이 변경 제안", "اقتراح هذا التغيير"),
  holdVerb: (l: Locale) => t(l, "Place hold", "보존 보류 등록", "وضع تعليق الحذف"),
  releaseVerb: (l: Locale) => t(l, "Propose release", "해제 제안", "اقتراح الرفع"),
  stopVerb: (l: Locale, run: string) => t(l, `Stop ${mono(run)}`, `${mono(run)} 중지`, `إيقاف ${mono(run)}`),
  approveVerb: (l: Locale, n: string) => t(l, `Approve proposal ${n}`, `제안 ${n} 승인`, `اعتماد المقترح ${n}`),
  withdrawVerb: (l: Locale, n: string) => t(l, `Withdraw proposal ${n}`, `제안 ${n} 철회`, `سحب المقترح ${n}`),

  factProposal: (l: Locale) => t(l, "Proposal", "제안", "المقترح"),
  factSummary: (l: Locale) => t(l, "Summary", "요약", "الملخص"),
  factProposerReason: (l: Locale) => t(l, "Proposer's reason", "제안 사유", "سبب المقترِح"),
  factCurrentPolicy: (l: Locale) => t(l, "Current policy", "현재 정책", "السياسة الحالية"),
  factCurrent: (l: Locale) => t(l, "Current", "현재", "الحالية"),
  factProposed: (l: Locale) => t(l, "Proposed", "제안", "المقترحة"),
  factChange: (l: Locale) => t(l, "Change", "변경", "التغيير"),
  factBecomeDue: (l: Locale) => t(l, "Become due earlier (installation)", "더 일찍 삭제 대상이 됨(설치 전체)", "تصبح مستحقة مبكرًا (التثبيت)"),
  factInWorkspace: (l: Locale) => t(l, "In this workspace", "이 업무 공간", "في مساحة العمل هذه"),
  factComputed: (l: Locale) => t(l, "Computed", "계산 시각", "وقت الحساب"),
  factHold: (l: Locale) => t(l, "Hold", "보존 보류", "تعليق الحذف"),
  factTarget: (l: Locale) => t(l, "Target", "대상", "الهدف"),
  factReleaseReason: (l: Locale) => t(l, "Release reason", "해제 사유", "سبب الرفع"),
  factRun: (l: Locale) => t(l, "Run", "작업", "العملية"),
  factState: (l: Locale) => t(l, "State", "상태", "الحالة"),
  factNotYetClaimed: (l: Locale) => t(l, "Not yet claimed", "아직 맡지 않음", "لم يُتسلَّم بعد"),
  noPolicyShort: (l: Locale) => t(l, "No policy", "정책 없음", "لا توجد سياسة"),
  itemTarget: (l: Locale, id: string) => t(l, `Item ${mono(id)}`, `항목 ${mono(id)}`, `العنصر ${mono(id)}`),
  shorter: (l: Locale, d: number) => t(l, `${d} days shorter`, `${d}일 짧아짐`, `أقصر بـ ${d} يومًا`),
  longer: (l: Locale, d: number) => t(l, `${d} days longer`, `${d}일 길어짐`, `أطول بـ ${d} يومًا`),
  sameTotal: (l: Locale) => t(l, "Same total", "합계 동일", "المجموع نفسه"),
  newPolicy: (l: Locale) => t(l, "First policy for this class", "이 등급의 첫 정책", "أول سياسة لهذه الفئة"),

  proposeImpact: (l: Locale, next: number | null, cls: string) =>
    next === null
      ? t(
          l,
          "Another administrator must approve it. Approval adds a new revision; it deletes nothing, because deletion plans are not connected.",
          "다른 관리자가 승인해야 합니다. 승인하면 새 revision이 추가되며, 삭제 계획이 연결되어 있지 않아 아무것도 삭제하지 않습니다.",
          "يجب أن يعتمده مسؤول آخر. تضيف الموافقة مراجعة جديدة، ولا تحذف شيئًا لأن خطط الحذف غير متصلة.",
        )
      : t(
          l,
          `Another administrator must approve it. Approval adds revision ${next} for ${cls}; it deletes nothing, because deletion plans are not connected.`,
          `다른 관리자가 승인해야 합니다. 승인하면 ${cls}에 revision ${next}이(가) 추가되며, 삭제 계획이 연결되어 있지 않아 아무것도 삭제하지 않습니다.`,
          `يجب أن يعتمده مسؤول آخر. تضيف الموافقة المراجعة ${next} لـ ${cls}، ولا تحذف شيئًا لأن خطط الحذف غير متصلة.`,
        ),
  shortenSentence: (l: Locale) =>
    t(
      l,
      "Because this shortens the period, the approving administrator must review how many items become due earlier. That count is computed after you propose; it cannot be previewed here.",
      "기간을 줄이므로 승인하는 관리자는 더 일찍 삭제 대상이 되는 항목 수를 검토해야 합니다. 이 건수는 제안한 뒤에 계산되며 여기에서 미리 볼 수 없습니다.",
      "لأن هذا يقصّر المدة، يجب على المسؤول المعتمِد مراجعة عدد العناصر التي تصبح مستحقة مبكرًا. يُحسب هذا العدد بعد الاقتراح؛ ولا يمكن معاينته هنا.",
    ),
  shortenUnknown: (l: Locale) =>
    t(
      l,
      "Whether this shortens the period is unknown; if it does, the approving administrator must review the impact.",
      "기간이 줄어드는지 알 수 없습니다. 줄어든다면 승인하는 관리자가 영향을 검토해야 합니다.",
      "لا يُعرف ما إذا كان هذا يقصّر المدة؛ وإن كان كذلك، فيجب على المسؤول المعتمِد مراجعة الأثر.",
    ),
  notShortening: (l: Locale) =>
    t(
      l,
      "Approval adds this revision now. The total does not get shorter, so no impact review is required. Nothing is deleted.",
      "승인하면 지금 이 revision이 추가됩니다. 합계가 짧아지지 않으므로 영향 검토가 필요하지 않습니다. 아무것도 삭제하지 않습니다.",
      "تضيف الموافقة هذه المراجعة الآن. لا يصبح المجموع أقصر، لذا لا تلزم مراجعة الأثر. لا يُحذف أي شيء.",
    ),
  reviewStrong: (l: Locale) => t(l, "Review the impact.", "영향을 검토하세요.", "راجع الأثر."),
  impactChanged: (l: Locale, change: string) =>
    t(l, `The impact changed since you reviewed it: ${iso(change)} items.`, `검토 이후 영향이 바뀌었습니다: ${iso(change)}건.`, `تغيّر الأثر منذ مراجعتك له: ${iso(change)} عنصر.`),
  releaseApproveImpact: (l: Locale, hold: string) =>
    t(
      l,
      `Approval releases hold ${mono(hold)} now. Items it covered are no longer protected by it and may be deleted by a later deletion run. Nothing is deleted now, because deletion plans and workers are not connected.`,
      `승인하면 보존 보류 ${mono(hold)}이(가) 지금 해제됩니다. 대상 항목은 더 이상 이 보류로 보호되지 않으며 이후 삭제 작업에서 삭제될 수 있습니다. 삭제 계획과 작업자가 연결되어 있지 않아 지금은 아무것도 삭제되지 않습니다.`,
      `تؤدي الموافقة إلى رفع تعليق الحذف ${mono(hold)} الآن. لم تعد العناصر المشمولة محمية به وقد تُحذف في عملية حذف لاحقة. لا يُحذف أي شيء الآن لأن خطط الحذف وجهاته غير متصلة.`,
    ),
  holdStrong: (l: Locale) =>
    t(l, "Takes effect now; no second approval: you alone place this hold.", "지금 바로 적용되며 두 번째 승인이 없습니다. 이 보존 보류는 귀하 혼자 등록합니다.", "يسري الآن دون موافقة ثانية: أنت وحدك تضع هذا التعليق."),
  holdBody: (l: Locale) =>
    t(
      l,
      "It stops deletion of covered items that still exist. It cannot bring back items already deleted, being deleted or partly deleted; the result lists them.",
      "대상 항목 중 아직 남아 있는 항목의 삭제를 막습니다. 이미 삭제되었거나 삭제 중이거나 일부만 삭제된 항목은 되살리지 못하며, 결과에 그 항목이 표시됩니다.",
      "يمنع حذف العناصر المشمولة التي ما زالت موجودة. ولا يمكنه إعادة العناصر المحذوفة أو قيد الحذف أو المحذوفة جزئيًا، وتُدرج في النتيجة.",
    ),
  itemDeleted: (l: Locale) => t(l, "This item is already recorded as deleted. The hold cannot preserve it.", "이 항목은 이미 삭제로 기록되었습니다. 보존 보류로 보존할 수 없습니다.", "هذا العنصر مسجّل كمحذوف بالفعل. لا يستطيع التعليق حفظه."),
  itemDamaged: (l: Locale) => t(l, "This item is partly deleted. The hold cannot restore what is gone.", "이 항목은 일부 삭제되었습니다. 보존 보류로 사라진 부분을 복원할 수 없습니다.", "هذا العنصر محذوف جزئيًا. لا يستطيع التعليق استعادة ما فُقد."),
  itemInRun: (l: Locale, run: string) =>
    t(l, `This item is in deletion run ${mono(run)}. The hold protects it only if the run has not started deleting it.`, `이 항목은 삭제 작업 ${mono(run)}에 포함되어 있습니다. 작업이 이 항목의 삭제를 시작하지 않은 경우에만 보존 보류로 보호됩니다.`, `هذا العنصر ضمن عملية الحذف ${mono(run)}. يحميه التعليق فقط إذا لم تبدأ العملية بحذفه.`),
  itemInOtherRun: (l: Locale) =>
    t(
      l,
      "This item is in a deletion run that is not shown in this workspace. The hold protects it only if the run has not started deleting it.",
      "이 항목은 이 업무 공간에 표시되지 않는 삭제 작업에 포함되어 있습니다. 작업이 이 항목의 삭제를 시작하지 않은 경우에만 보존 보류로 보호됩니다.",
      "هذا العنصر ضمن عملية حذف غير معروضة في مساحة العمل هذه. يحميه التعليق فقط إذا لم تبدأ العملية بحذفه.",
    ),
  workspaceCovers: (l: Locale, cls: string) =>
    cls
      ? t(l, `It covers every registered item of this workspace of class ${cls}, including items registered later.`, `이 업무 공간에서 ${cls} 등급으로 등록된 모든 항목과 이후 등록되는 항목까지 포함합니다.`, `يشمل كل عنصر مسجّل في مساحة العمل هذه من فئة ${cls}، بما في ذلك العناصر المسجّلة لاحقًا.`)
      : t(l, "It covers every registered item of this workspace, including items registered later.", "이 업무 공간에 등록된 모든 항목과 이후 등록되는 항목까지 포함합니다.", "يشمل كل عنصر مسجّل في مساحة العمل هذه، بما في ذلك العناصر المسجّلة لاحقًا."),
  releaseLater: (l: Locale) =>
    t(
      l,
      "Releasing it later needs two other administrators: one to propose and one to approve. If this workspace has fewer than two administrators besides you, this hold cannot be released.",
      "나중에 해제하려면 다른 관리자 두 명이 필요합니다. 한 명이 제안하고 한 명이 승인합니다. 이 업무 공간에 귀하 외의 관리자가 두 명보다 적으면 이 보존 보류는 해제할 수 없습니다.",
      "يتطلب رفعه لاحقًا مسؤولَين آخرين: أحدهما يقترح والآخر يعتمد. إذا كان لدى مساحة العمل هذه أقل من مسؤولَين عداك، فلا يمكن رفع هذا التعليق.",
    ),
  releaseProposeImpact: (l: Locale) =>
    t(
      l,
      "Another administrator who did not place this hold must approve it. Approval releases the hold; nothing is deleted now.",
      "이 보존 보류를 등록하지 않은 다른 관리자가 승인해야 합니다. 승인하면 보류가 해제되며 지금은 아무것도 삭제되지 않습니다.",
      "يجب أن يعتمده مسؤول آخر لم يضع هذا التعليق. تؤدي الموافقة إلى رفع التعليق؛ ولا يُحذف أي شيء الآن.",
    ),
  stopStrong: (l: Locale) =>
    t(l, "Takes effect now; no second approval: you alone stop this run.", "지금 바로 적용되며 두 번째 승인이 없습니다. 이 작업은 귀하 혼자 중지합니다.", "يسري الآن دون موافقة ثانية: أنت وحدك توقف هذه العملية."),
  stopBody: (l: Locale) =>
    t(
      l,
      "Items in chunks no worker has claimed leave the run with the outcome Stopped and are no longer blocked by it. Items already deleted stay deleted; stopping is not a restore. Chunks already claimed are not recalled.",
      "작업자가 맡지 않은 묶음의 항목은 '중지됨'으로 작업에서 빠지고 더 이상 이 작업 때문에 차단되지 않습니다. 이미 삭제된 항목은 그대로이며 중지는 복원이 아닙니다. 이미 맡은 묶음은 회수하지 않습니다.",
      "تخرج عناصر الدفعات التي لم تتسلمها أي جهة من العملية بنتيجة «موقوف» ولا تبقى محجوبة بسببها. تبقى العناصر المحذوفة محذوفة، والإيقاف ليس استعادة. لا تُسترد الدفعات التي تم تسلمها.",
    ),
  withdrawImpact: (l: Locale) => t(l, "Withdrawing closes this proposal. Nothing else changes.", "철회하면 이 제안이 종료됩니다. 그 밖에는 바뀌는 것이 없습니다.", "يغلق السحب هذا المقترح. لا يتغيّر شيء آخر."),
  repeat: (l: Locale) => t(l, "This repeats your earlier request exactly as it was sent.", "이전에 보낸 요청을 그대로 다시 보냅니다.", "يكرر هذا طلبك السابق تمامًا كما أُرسل."),

  policyApplied: (l: Locale, n: string) =>
    t(l, `Policy proposal ${mono("#" + n)} applied. Nothing was deleted.`, `정책 제안 ${mono("#" + n)}이(가) 적용되었습니다. 아무것도 삭제되지 않았습니다.`, `طُبّق مقترح السياسة ${mono("#" + n)}. لم يُحذف أي شيء.`),
  releaseApplied: (l: Locale, n: string, hold: string) =>
    hold
      ? t(l, `Release proposal ${mono("#" + n)} applied. Hold ${mono(hold)} is released. Nothing was deleted.`, `해제 제안 ${mono("#" + n)}이(가) 적용되었습니다. 보존 보류 ${mono(hold)}이(가) 해제되었습니다. 아무것도 삭제되지 않았습니다.`, `طُبّق مقترح الرفع ${mono("#" + n)}. رُفع تعليق الحذف ${mono(hold)}. لم يُحذف أي شيء.`)
      : t(l, `Release proposal ${mono("#" + n)} applied. Nothing was deleted.`, `해제 제안 ${mono("#" + n)}이(가) 적용되었습니다. 아무것도 삭제되지 않았습니다.`, `طُبّق مقترح الرفع ${mono("#" + n)}. لم يُحذف أي شيء.`),
  withdrawnDone: (l: Locale, n: string) => t(l, `Proposal ${mono("#" + n)} withdrawn.`, `제안 ${mono("#" + n)}을(를) 철회했습니다.`, `سُحب المقترح ${mono("#" + n)}.`),
  policyProposed: (l: Locale, n: string) =>
    t(l, `Policy proposal ${mono("#" + n)} is waiting for another administrator's approval.`, `정책 제안 ${mono("#" + n)}이(가) 다른 관리자의 승인을 기다리고 있습니다.`, `مقترح السياسة ${mono("#" + n)} بانتظار اعتماد مسؤول آخر.`),
  holdInForce: (l: Locale, hold: string) => t(l, `Hold ${mono(hold)} is in force.`, `보존 보류 ${mono(hold)}이(가) 유지 중입니다.`, `تعليق الحذف ${mono(hold)} سارٍ.`),
  holdPartial: (l: Locale, hold: string, a: number | null, b: number | null, c: number | null) => {
    const u = t(l, "Unknown", "알 수 없음", "غير معروف");
    const parts = [a !== null && a > 0 ? counted(l, a, plurals.notPreservedPart) : "", c !== null && c > 0 ? counted(l, c, plurals.partlyPart) : ""].filter(Boolean);
    const unknownLost = a === null || c === null
      ? " " + t(l, `Items already deleted or partly deleted: ${u}.`, `이미 삭제되었거나 일부 삭제된 항목: ${u}.`, `العناصر المحذوفة بالفعل أو المحذوفة جزئيًا: ${u}.`)
      : "";
    const or = t(l, " or ", " 또는 ", " أو ");
    const lead = parts.length
      ? t(l, `Hold ${mono(hold)} is in force, but it cannot preserve ${parts.join(or)}.`, `보존 보류 ${mono(hold)}이(가) 유지 중이지만 ${parts.join(or)}은(는) 보존할 수 없습니다.`, `تعليق الحذف ${mono(hold)} سارٍ، لكنه لا يستطيع حفظ ${parts.join(or)}.`)
      : t(l, `Hold ${mono(hold)} is in force.`, `보존 보류 ${mono(hold)}이(가) 유지 중입니다.`, `تعليق الحذف ${mono(hold)} سارٍ.`);
    const tail = b === null ? " " + t(l, "Items with a deletion outcome pending: Unknown.", "삭제 결과 대기 항목: 알 수 없음.", "العناصر التي نتيجة حذفها معلّقة: غير معروف.") : b > 0 ? " " + counted(l, b, plurals.pendingPart) : "";
    return lead + unknownLost + tail;
  },
  releaseProposed: (l: Locale, n: string, hold: string) =>
    t(
      l,
      `Release proposal ${mono("#" + n)} for hold ${mono(hold)} is waiting for approval by another administrator who did not place the hold.`,
      `보존 보류 ${mono(hold)}에 대한 해제 제안 ${mono("#" + n)}이(가) 보류를 등록하지 않은 다른 관리자의 승인을 기다리고 있습니다.`,
      `مقترح الرفع ${mono("#" + n)} لتعليق الحذف ${mono(hold)} بانتظار اعتماد مسؤول آخر لم يضع التعليق.`,
    ),
  stopDone: (l: Locale, run: string, n: number | null, state: string) =>
    (n === null
      ? t(l, `${mono(run)} stopped: the number of released items is unknown. Nothing was restored.`, `${mono(run)}을(를) 중지했습니다. 풀려난 항목 수는 알 수 없습니다. 아무것도 복원되지 않았습니다.`, `أُوقفت ${mono(run)}: عدد العناصر المحررة غير معروف. لم يُستعد شيء.`)
      : counted(l, n, plurals.stopDone, { run: mono(run) })) + (state ? " " + state : ""),
  duplicate: (l: Locale, n: string) => t(l, `This was already proposed as ${mono("#" + n)}.`, `이미 ${mono("#" + n)}(으)로 제안되었습니다.`, `اقتُرح هذا بالفعل بوصفه ${mono("#" + n)}.`),
  resultUnknown: (l: Locale, state: string) => {
    const code = state ? mono(state) : t(l, "Unknown", "알 수 없음", "غير معروف");
    return t(l, `The request was recorded with an unknown result (${code}). Reload to check.`, `요청이 알 수 없는 결과(${code})로 기록되었습니다. 새로 고침해 확인하세요.`, `سُجّل الطلب بنتيجة غير معروفة (${code}). أعد التحميل للتحقق.`);
  },

  blocker: (l: Locale, code: string) => {
    if (code === "missing") return t(l, "This is not available now.", "지금은 사용할 수 없습니다.", "هذا غير متاح الآن.");
    if (has(blockerWords, code)) return words(l, blockerWords, code);
    return t(l, `This is not available now (${mono(code)}).`, `지금은 사용할 수 없습니다(${mono(code)}).`, `هذا غير متاح الآن (${mono(code)}).`);
  },
  placer: (l: Locale, action: "approve" | "propose_release") => words(l, placerWords, action),
  narrow: (l: Locale, code: string) => words(l, narrowWords, code),
  kept: (l: Locale, key: string) => words(l, keptWords, key),
  stateSentence: (l: Locale, code: string) => words(l, stateSentenceWords, code),
  gone: (l: Locale, kind: string) =>
    kind === "proposal" ? t(l, "This proposal is no longer available.", "이 제안은 더 이상 사용할 수 없습니다.", "لم يعد هذا المقترح متاحًا.")
      : kind === "hold" ? t(l, "This hold is no longer available.", "이 보존 보류는 더 이상 사용할 수 없습니다.", "لم يعد تعليق الحذف هذا متاحًا.")
        : kind === "run" ? t(l, "This run is no longer available.", "이 작업은 더 이상 사용할 수 없습니다.", "لم تعد هذه العملية متاحة.")
          : kind === "class" ? t(l, "This class is no longer available.", "이 등급은 더 이상 사용할 수 없습니다.", "لم تعد هذه الفئة متاحة.")
            : kind === "workspace" ? t(l, "This workspace is no longer available.", "이 업무 공간은 더 이상 사용할 수 없습니다.", "لم تعد مساحة العمل هذه متاحة.")
              : t(l, "This item is no longer available.", "이 항목은 더 이상 사용할 수 없습니다.", "لم يعد هذا العنصر متاحًا."),
  pendingAttempt: (l: Locale) => t(l, "The last request may have been applied. Check the list, or send it again.", "직전 요청이 이미 반영되었을 수 있습니다. 목록을 확인하거나 다시 보내세요.", "ربما طُبّق الطلب الأخير. راجع القائمة أو أرسله مجددًا."),
  holdPendingNote: (l: Locale) =>
    t(
      l,
      "Check the holds list before sending again. Signing out forgets this unfinished request.",
      "다시 보내기 전에 보존 보류 목록을 확인하세요. 로그아웃하면 이 미완료 요청은 잊혀집니다.",
      "راجع قائمة التعليقات قبل الإرسال مجددًا. يؤدي تسجيل الخروج إلى نسيان هذا الطلب غير المكتمل.",
    ),
  editWarning: (l: Locale) =>
    t(l, "If you change anything, a new hold is placed in addition to any earlier one.", "무엇이든 바꾸면 이전 보존 보류와 별도로 새 보존 보류가 등록됩니다.", "إذا غيّرت أي شيء، يُوضع تعليق جديد إضافة إلى أي تعليق سابق."),
  staleMatch: (l: Locale, id: string) =>
    t(
      l,
      `Hold ${mono(id)}, placed by you with the same target, owner and reason, is already listed. It may be the result of your earlier request.`,
      `같은 대상·책임자·사유로 귀하가 등록한 보존 보류 ${mono(id)}이(가) 이미 목록에 있습니다. 이전 요청의 결과일 수 있습니다.`,
      `تعليق الحذف ${mono(id)}، الذي وضعته بالهدف والمالك والسبب نفسها، مدرج بالفعل. ربما يكون نتيجة طلبك السابق.`,
    ),
  staleNoMatch: (l: Locale) =>
    t(l, "No hold placed by you with the same target, owner and reason is listed.", "같은 대상·책임자·사유로 귀하가 등록한 보존 보류가 목록에 없습니다.", "لا يوجد تعليق وضعته بالهدف والمالك والسبب نفسها في القائمة."),
  goToHold: (l: Locale, id: string) => t(l, `Go to hold ${mono(id)}`, `보존 보류 ${mono(id)}(으)로 이동`, `الانتقال إلى التعليق ${mono(id)}`),
  placeAnother: (l: Locale) => t(l, "Place another hold", "보존 보류 추가 등록", "وضع تعليق آخر"),
  impactNotSent: (l: Locale) =>
    t(l, "The impact could not be sent as read. Reload and review the impact again.", "영향을 읽은 그대로 보낼 수 없습니다. 새로 고침한 뒤 영향을 다시 검토하세요.", "تعذّر إرسال الأثر كما قُرئ. أعد التحميل وراجع الأثر مجددًا."),
  builtBug: (l: Locale) => t(l, "The request could not be sent as built. Reload and try again.", "요청을 구성된 그대로 보낼 수 없습니다. 새로 고침한 뒤 다시 시도하세요.", "تعذّر إرسال الطلب كما بُني. أعد التحميل وحاول مجددًا."),
  conflictAgain: (l: Locale) => t(l, "The retention data changed. Check the values and confirm again.", "보관 데이터가 바뀌었습니다. 값을 확인하고 다시 확인하세요.", "تغيّرت بيانات الاحتفاظ. تحقّق من القيم وأكّد مرة أخرى."),
  renew: (l: Locale) => t(l, "That request expired. Start again.", "요청이 만료되었습니다. 처음부터 다시 하세요.", "انتهت صلاحية الطلب. ابدأ من جديد."),
  partial: (l: Locale, op: string) => t(l, `Stopped after a change. Operation ${mono(op)}. Check the list before trying again.`, `변경 후 중단되었습니다. 작업 ${mono(op)}. 다시 시도하기 전에 목록을 확인하세요.`, `توقف بعد تغيير. العملية ${mono(op)}. راجع القائمة قبل المحاولة مجددًا.`),

  className: (l: Locale, s: string) => words(l, classWords, s),
  standing: (l: Locale, s: string) => words(l, standingWords, s),
  access: (l: Locale, s: string) => words(l, accessWords, s),
  holdOutcome: (l: Locale, s: string) => words(l, holdOutcomeWords, s),
  storeResult: (l: Locale, s: string) => words(l, storeResultWords, s),
  runState: (l: Locale, s: string) => words(l, runStateWords, s),
  runStateNote: (l: Locale, s: string) => (has(runStateNotes, s) ? words(l, runStateNotes, s) : ""),
  runOutcome: (l: Locale, s: string) => words(l, runOutcomeWords, s),
  holdState: (l: Locale, s: string) => words(l, holdStateWords, s),
  origin: (l: Locale, s: string) => words(l, originWords, s),
  connection: (l: Locale, s: string) => words(l, connectionWords, s),
  ledger: (l: Locale, s: string) => words(l, ledgerWords, s),
  decision: (l: Locale, s: string) => words(l, decisionWords, s),
  population: (l: Locale, s: string) => words(l, populationWords, s),
  proposalKind: (l: Locale, s: string) => words(l, proposalKindWords, s),
  proposalState: (l: Locale, s: string) => words(l, proposalStateWords, s),
  result: (l: Locale, s: string) => words(l, resultWords, s),

  form: {
    rationale: (l: Locale) => t(l, "Reason for this action", "이 작업의 사유", "سبب هذا الإجراء"),
    rationaleHint: (l: Locale, n: number) => t(l, `${n} of 256 bytes. Required. Do not enter customer data.`, `256바이트 중 ${n}바이트. 필수. 고객 정보를 입력하지 마세요.`, `${n} من 256 بايت. مطلوب. لا تُدخل بيانات العملاء.`),
    rationaleMissing: (l: Locale) => t(l, "Enter a reason.", "사유를 입력하세요.", "أدخل سببًا."),
    checkReason: (l: Locale) => t(l, "Check the reason and try again.", "사유를 확인하고 다시 시도하세요.", "تحقّق من السبب وحاول مجددًا."),
    rationaleLong: (l: Locale) => t(l, "The reason is too long. Use at most 256 bytes.", "사유가 너무 깁니다. 256바이트 이하로 입력하세요.", "السبب طويل جدًا. استخدم 256 بايت على الأكثر."),
    review: (l: Locale) => t(l, "Review", "검토", "مراجعة"),
    cancel: (l: Locale) => t(l, "Cancel", "취소", "إلغاء"),
    backToEdit: (l: Locale) => t(l, "Back to edit", "수정으로 돌아가기", "العودة إلى التعديل"),
    sending: (l: Locale) => t(l, "Sending…", "보내는 중…", "جارٍ الإرسال…"),
    period: (l: Locale) => t(l, "Period (days)", "기간(일)", "المدة (أيام)"),
    grace: (l: Locale) => t(l, "Grace (days)", "유예(일)", "السماح (أيام)"),
    periodHint: (l: Locale) => t(l, "1 to 36,500", "1~36,500", "من 1 إلى 36,500"),
    graceHint: (l: Locale) => t(l, "0 to 365", "0~365", "من 0 إلى 365"),
    periodRange: (l: Locale) => t(l, "Enter whole days from 1 to 36,500.", "1~36,500 사이의 정수 일수를 입력하세요.", "أدخل عدد أيام صحيحًا من 1 إلى 36,500."),
    graceRange: (l: Locale) => t(l, "Enter whole days from 0 to 365.", "0~365 사이의 정수 일수를 입력하세요.", "أدخل عدد أيام صحيحًا من 0 إلى 365."),
    sameValues: (l: Locale) => t(l, "These are the current values. Change the period or the grace.", "현재 값과 같습니다. 기간이나 유예를 바꾸세요.", "هذه هي القيم الحالية. غيّر المدة أو فترة السماح."),
    owner: (l: Locale) => t(l, "Owner", "책임자", "المالك"),
    ownerPlaceholder: (l: Locale) => t(l, "Choose an owner", "책임자 선택", "اختر مالكًا"),
    ownerHint: (l: Locale) => t(l, "The person accountable for this hold.", "이 보존 보류에 책임을 지는 사람입니다.", "الشخص المسؤول عن هذا التعليق."),
    ownerMissing: (l: Locale) => t(l, "Choose an owner.", "책임자를 선택하세요.", "اختر مالكًا."),
    ownerGone: (l: Locale) => t(l, "This person can no longer own a hold. Choose another.", "이 사람은 더 이상 보존 보류의 책임자가 될 수 없습니다. 다른 사람을 선택하세요.", "لم يعد بإمكان هذا الشخص امتلاك تعليق. اختر شخصًا آخر."),
    ownersFailed: (l: Locale) => t(l, "The list of possible owners could not be read.", "책임자 후보 목록을 읽을 수 없습니다.", "تعذّرت قراءة قائمة المالكين المحتملين."),
    ownersEmpty: (l: Locale) => t(l, "No active member can own a hold.", "보존 보류의 책임자가 될 수 있는 활성 구성원이 없습니다.", "لا يوجد عضو نشط يمكنه امتلاك تعليق."),
    ownersLoading: (l: Locale) => t(l, "Reading the list of possible owners…", "책임자 후보 목록을 읽는 중…", "جارٍ قراءة قائمة المالكين المحتملين…"),
    tryAgain: (l: Locale) => t(l, "Try again", "다시 시도", "حاول مجددًا"),
    holdReason: (l: Locale) => t(l, "Hold reason (stored with the hold)", "보존 보류 사유(보류와 함께 저장)", "سبب التعليق (يُحفظ مع التعليق)"),
    releaseReason: (l: Locale) => t(l, "Release reason", "해제 사유", "سبب الرفع"),
    textHint: (l: Locale, n: number) => t(l, `${n} of 200 bytes. One line.`, `200바이트 중 ${n}바이트. 한 줄.`, `${n} من 200 بايت. سطر واحد.`),
    holdReasonMissing: (l: Locale) => t(l, "Enter a hold reason.", "보존 보류 사유를 입력하세요.", "أدخل سبب التعليق."),
    holdReasonLong: (l: Locale) => t(l, "The hold reason is too long (at most 200 bytes).", "보존 보류 사유가 너무 깁니다(200바이트 이하).", "سبب التعليق طويل جدًا (200 بايت على الأكثر)."),
    releaseReasonMissing: (l: Locale) => t(l, "Enter a release reason.", "해제 사유를 입력하세요.", "أدخل سبب الرفع."),
    releaseReasonLong: (l: Locale) => t(l, "The release reason is too long (at most 200 bytes).", "해제 사유가 너무 깁니다(200바이트 이하).", "سبب الرفع طويل جدًا (200 بايت على الأكثر)."),
    textControl: (l: Locale) => t(l, "Remove line breaks and control characters.", "줄바꿈과 제어 문자를 지우세요.", "احذف فواصل الأسطر وأحرف التحكم."),
    textInvalid: (l: Locale) => t(l, "The retention register did not accept this text. Check it and try again.", "보관 레지스트리가 이 문구를 받지 않았습니다. 확인한 뒤 다시 시도하세요.", "لم يقبل سجل الاحتفاظ هذا النص. تحقّق منه وحاول مجددًا."),
    classLabel: (l: Locale) => t(l, "Data class", "자료 등급", "فئة البيانات"),
    allClasses: (l: Locale) => t(l, "All classes", "모든 등급", "كل الفئات"),
  },
};
