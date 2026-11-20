import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { Plural } from "../../../../shared/plural";
import { units } from "../access/request-copy";

export const stateLabel: Record<string, [string, string, string]> = {
  draft: ["Draft", "초안", "مسودة"],
  validated: ["Validated", "검증됨", "تم التحقق"],
  independently_approved: ["Independently approved", "독립 승인됨", "معتمد اعتمادًا مستقلًا"],
  activated: ["Activated", "활성화됨", "مُفعَّل"],
  rejected: ["Rejected", "반려됨", "مرفوض"],
  cancelled: ["Cancelled", "취소됨", "ملغى"],
  superseded: ["Superseded", "대체됨", "مُستبدَل"],
  activation_unknown: ["Activation not confirmed", "활성화 확인 안 됨", "لم يُؤكَّد التفعيل"],
  revalidated_approval_kept: ["Validated again, approval kept", "재검증, 승인 유지", "أُعيد التحقق، والاعتماد قائم"],
};

const hold = [
  "The server could not confirm that the activation record was saved. Everyone in this workspace was signed out. Until the server restarts, only changes that reduce access, stop a running operation such as a deletion run, withdraw a pending proposal or place a retention hold are possible, plus ending sessions, password changes and using an already issued password reset link. Invitations cannot be accepted and new password reset links cannot be issued. At the next start the server applies the activation only if its record is readable.",
  "활성화 기록이 저장되었는지 확인할 수 없습니다. 이 작업 공간의 모든 사용자가 로그아웃되었습니다. 서버를 다시 시작할 때까지는 권한 축소, 진행 중인 작업(예: 삭제 실행) 중지, 대기 중인 제안 철회, 보존 보류 등록만 할 수 있으며 세션 종료, 비밀번호 변경, 이미 발급된 비밀번호 재설정 링크 사용은 가능합니다. 초대를 수락하거나 새 비밀번호 재설정 링크를 발급할 수 없습니다. 다음 시작 시 기록을 읽을 수 있는 경우에만 활성화가 적용됩니다.",
  "تعذّر التأكد من حفظ سجل التفعيل. تم تسجيل خروج الجميع في مساحة العمل هذه. حتى إعادة تشغيل الخادم، لا يمكن إلا تقليل الوصول أو إيقاف عملية جارية مثل تشغيل الحذف أو سحب اقتراح معلّق أو وضع تعليق الحذف، إضافةً إلى إنهاء الجلسات وتغيير كلمة المرور واستخدام رابط إعادة تعيين صادر مسبقًا. لا يمكن قبول الدعوات ولا إصدار روابط جديدة لإعادة تعيين كلمة المرور. عند البدء التالي يطبّق الخادم التفعيل فقط إذا كان سجله قابلًا للقراءة.",
] as const;

export const stateSentence: Record<string, [string, string, string]> = {
  draft: ["Trust change is a draft; active trust is unchanged.", "신뢰 변경 초안입니다. 현재 적용 중인 신뢰 설정은 바뀌지 않았습니다.", "تغيير الثقة مسودة؛ إعدادات الثقة النشطة لم تتغير."],
  validated: ["Exact proposal evidence is validated; independent approval is required.", "제안 내용의 근거가 검증되었습니다. 다른 관리자의 독립 승인이 필요합니다.", "تم التحقق من أدلة الاقتراح بدقة؛ يلزم اعتماد مستقل."],
  independently_approved: [
    "Independent approval is recorded; activation still rechecks current authority and old epoch.",
    "독립 승인이 기록되었습니다. 활성화할 때 현재 권한과 이전 신뢰 버전을 다시 확인합니다.",
    "تم تسجيل الاعتماد المستقل؛ يعيد التفعيل التحقق من الصلاحية الحالية وإصدار الثقة السابق.",
  ],
  activated: [
    "The new epoch and invalidation intent are committed. Inspect remaining invalidation progress.",
    "새 신뢰 버전과 무효화 의도가 확정되었습니다. 남은 무효화 진행 상황을 확인하세요.",
    "تم تثبيت إصدار الثقة الجديد ونية إلغاء الصلاحية. راجع تقدّم إلغاء الصلاحية المتبقي.",
  ],
  rejected: ["The proposal is rejected; active trust is unchanged.", "제안이 반려되었습니다. 현재 신뢰 설정은 바뀌지 않았습니다.", "تم رفض الاقتراح؛ إعدادات الثقة النشطة لم تتغير."],
  cancelled: ["The proposal is cancelled; active trust is unchanged.", "제안이 취소되었습니다. 현재 신뢰 설정은 바뀌지 않았습니다.", "تم إلغاء الاقتراح؛ إعدادات الثقة النشطة لم تتغير."],
  superseded: [
    "Another change was activated on the same epoch. This proposal can no longer be activated; propose again if it is still needed.",
    "같은 신뢰 버전에서 다른 변경이 활성화되었습니다. 이 제안은 더 이상 활성화할 수 없으니 필요하면 다시 제안하세요.",
    "تم تفعيل تغيير آخر على إصدار الثقة نفسه. لا يمكن تفعيل هذا الاقتراح؛ اقترح مجددًا إذا لزم.",
  ],
  activation_unknown: [hold[0], hold[1], hold[2]],
  blocked: [
    "Validation found blocked checks, so this change stays a draft and cannot be approved or activated.",
    "검증에서 차단된 항목이 있어 이 변경은 초안으로 남으며 승인하거나 활성화할 수 없습니다.",
    "وجد التحقق فحوصًا تمنع المتابعة، لذا يبقى هذا التغيير مسودة ولا يمكن اعتماده أو تفعيله.",
  ],
  stale_validated: [
    "The evidence changed after validation, so this change cannot be approved. Validate it again.",
    "검증 후 근거가 바뀌어 이 변경을 승인할 수 없습니다. 다시 검증하세요.",
    "تغيّرت الأدلة بعد التحقق، لذا لا يمكن اعتماد هذا التغيير. تحقّق منه مجددًا.",
  ],
  stale_approved: [
    "The evidence changed after validation, so the recorded approval cannot be used to activate. Validate again, then get a new independent approval.",
    "검증 후 근거가 바뀌어 기록된 승인으로는 활성화할 수 없습니다. 다시 검증한 뒤 독립 승인을 새로 받으세요.",
    "تغيّرت الأدلة بعد التحقق، لذا لا يمكن استخدام الاعتماد المسجَّل للتفعيل. تحقّق مجددًا ثم احصل على اعتماد مستقل جديد.",
  ],
};

export const pick = (l: Locale, v: readonly [string, string, string]) => t(l, v[0], v[1], v[2]);

const itemNames: Record<string, [string, string, string]> = {
  test_login: ["Test login", "테스트 로그인", "تسجيل الدخول التجريبي"],
  recovery_path: ["Recovery path", "복구 경로", "مسار الاسترداد"],
  issuer_subject: ["Issuer and stable subject", "발급자와 고정 주체", "جهة الإصدار والمعرّف الثابت"],
  provider_diagnostics: ["Provider diagnostics", "공급자 진단", "تشخيص المزوّد"],
  external_status: ["External identity status", "외부 ID 상태", "حالة الهوية الخارجية"],
  mfa_weakening: ["MFA weakening", "MFA 약화", "إضعاف MFA"],
  affected_subjects: ["Affected members", "영향받는 구성원", "الأعضاء المتأثرون"],
  session_impact: ["Open sessions at validation", "검증 시점의 열린 세션", "الجلسات المفتوحة وقت التحقق"],
  record_size: ["Record size", "기록 크기", "حجم السجل"],
};

const reasons: Record<string, [string, string, string]> = {
  recovery_verified: [
    "Three different administrators who can sign in under the proposed rules hold the propose, approve and activate rights, and one holds the validate right.",
    "제안된 규칙으로 로그인할 수 있고 제안·승인·활성화 권한을 각각 가진 서로 다른 관리자 3명과 검증 권한을 가진 관리자 1명이 있습니다.",
    "يوجد ثلاثة مسؤولين مختلفين يمكنهم تسجيل الدخول وفق القواعد المقترحة ويملكون صلاحيات الاقتراح والاعتماد والتفعيل، ومسؤول يملك صلاحية التحقق.",
  ],
  test_login_failed: ["Your current sign-in would not pass the proposed rules.", "현재 로그인이 제안된 규칙을 통과하지 못합니다.", "لن يجتاز تسجيل دخولك الحالي القواعد المقترحة."],
  recovery_unverified: [
    "A later correction needs three different administrators who can sign in under the proposed rules and hold the propose, approve and activate rights, plus one who holds the validate right. This installation does not have them.",
    "이후 수정을 하려면 제안된 규칙으로 로그인할 수 있고 제안·승인·활성화 권한을 각각 가진 서로 다른 관리자 3명과 검증 권한을 가진 관리자 1명이 필요합니다. 이 설치에는 해당 관리자가 부족합니다.",
    "يتطلب أي تصحيح لاحق ثلاثة مسؤولين مختلفين يمكنهم تسجيل الدخول وفق القواعد المقترحة ويملكون صلاحيات الاقتراح والاعتماد والتفعيل، إضافةً إلى مسؤول يملك صلاحية التحقق. لا يتوفر ذلك في هذا التثبيت.",
  ],
  local_password: ["Local password sign-in has no external issuer.", "로컬 비밀번호 로그인에는 외부 발급자가 없습니다.", "تسجيل الدخول بكلمة مرور محلية ليس له جهة إصدار خارجية."],
  no_provider_connection: ["No identity provider is connected.", "연결된 ID 공급자가 없습니다.", "لا يوجد مزوّد هوية متصل."],
  no_external_identity: ["No external identity is used.", "외부 ID를 사용하지 않습니다.", "لا تُستخدم هوية خارجية."],
  record_limit: ["The activation record would exceed its size limit.", "활성화 기록이 크기 한도를 넘습니다.", "سيتجاوز سجل التفعيل حدّ حجمه."],
  epoch_exhausted: ["No newer trust version is available in this ledger generation.", "이 원장 세대에서는 더 새로운 신뢰 버전을 만들 수 없습니다.", "لا يتوفر إصدار ثقة أحدث في جيل الدفتر هذا."],
  weakens_mfa: ["Weakens MFA", "MFA를 약화함", "يُضعف MFA"],
  unchanged: ["Unchanged", "변경 없음", "دون تغيير"],
};

const roleWords: Record<string, [string, string, string]> = {
  trust_propose: ["propose", "제안", "الاقتراح"],
  trust_validate: ["validate", "검증", "التحقق"],
  trust_approve: ["approve", "승인", "الاعتماد"],
  trust_activate: ["activate", "활성화", "التفعيل"],
};

const progressNames: Record<string, [string, string, string]> = {
  console_sessions: ["Console sessions below the new version", "새 버전보다 낮은 콘솔 세션", "جلسات وحدة التحكم الأقدم من الإصدار الجديد"],
  password_confirmation: ["Password confirmation evidence", "비밀번호 재확인 근거", "دليل تأكيد كلمة المرور"],
  secret_pickups: ["One-time secret links issued by ended sessions", "종료된 세션이 발급한 일회용 비밀 링크", "روابط أسرار لمرة واحدة صادرة من جلسات منتهية"],
  admitted_reads: ["Requests admitted before activation and still running", "활성화 전에 허용되어 아직 실행 중인 요청", "طلبات قُبلت قبل التفعيل وما زالت قيد التنفيذ"],
  service_credentials: ["Service credentials", "서비스 자격 증명", "بيانات اعتماد الخدمات"],
  tokens_queues_caches: ["Refresh tokens, queued privileged commands, cached permissions", "갱신 토큰, 대기 중인 권한 명령, 캐시된 권한", "رموز التحديث والأوامر المميزة المنتظرة والصلاحيات المخزنة مؤقتًا"],
  browser_copies: ["Copies delivered to browsers", "브라우저에 전달된 사본", "النسخ المسلَّمة إلى المتصفحات"],
};

const results: Record<string, [string, string, string]> = {
  verified: ["Verified", "확인됨", "تم التحقق"],
  not_applicable: ["Not applicable", "해당 없음", "لا ينطبق"],
  blocked: ["Blocked", "차단됨", "يمنع المتابعة"],
  information: ["Information", "참고", "معلومة"],
  done: ["Done", "완료", "مكتمل"],
  in_progress: ["In progress", "진행 중", "قيد التنفيذ"],
  unknown: ["Unknown", "알 수 없음", "غير معروف"],
  outside_scope: ["Outside the controlled scope", "통제 범위 밖", "خارج النطاق المتحكَّم فيه"],
};

const lookup = (l: Locale, table: Record<string, [string, string, string]>, key: string) => (table[key] ? pick(l, table[key]) : key);

export const ty = {
  state: (l: Locale, s: string) => lookup(l, stateLabel, s),
  sentence: (l: Locale, s: string) => lookup(l, stateSentence, s),
  item: (l: Locale, s: string) => lookup(l, itemNames, s),
  reason: (l: Locale, s: string) => lookup(l, reasons, s),
  role: (l: Locale, s: string) => lookup(l, roleWords, s),
  progress: (l: Locale, s: string) => lookup(l, progressNames, s),
  result: (l: Locale, s: string) => lookup(l, results, s),
  holdLong: (l: Locale) => pick(l, hold),
  currentTitle: (l: Locale) => t(l, "Current sign-in rules", "현재 로그인 규칙", "قواعد تسجيل الدخول الحالية"),
  rules: (l: Locale) => t(l, "Rules", "규칙", "القواعد"),
  trust: (l: Locale) => t(l, "Trust version and approvals", "신뢰 버전과 승인 기록", "إصدار الثقة والاعتمادات"),
  version: (l: Locale) => t(l, "Trust version (epoch)", "신뢰 버전(epoch)", "إصدار الثقة (epoch)"),
  policyRevision: (l: Locale) => t(l, "Policy revision", "정책 개정 번호", "رقم تعديل السياسة"),
  origin: (l: Locale) => t(l, "Origin", "출처", "المصدر"),
  originWord: (l: Locale, o: string) =>
    o === "seed" ? t(l, "Installation default", "설치 기본값", "الإعداد الافتراضي للتثبيت")
      : o === "this_process" ? t(l, "Activated in this server run", "이 서버 실행 중 활성화됨", "مُفعَّل في تشغيل الخادم هذا")
        : o === "ledger_replay" ? t(l, "Restored from the decision ledger at start", "시작 시 결정 원장에서 복원됨", "استُعيد من دفتر القرارات عند البدء")
          : o === "carried" ? t(l, "Carried from a replaced ledger", "교체된 원장에서 이어받음", "منقول من دفتر مستبدَل")
            : t(l, "Not recorded", "기록 없음", "غير مسجَّل"),
  activatedAt: (l: Locale) => t(l, "Activated at", "활성화 시각", "وقت التفعيل"),
  proposedBy: (l: Locale) => t(l, "Proposed by", "제안자", "اقترحه"),
  validatedBy: (l: Locale) => t(l, "Validated by", "검증자", "تحقّق منه"),
  approvedBy: (l: Locale) => t(l, "Approved by", "승인자", "اعتمده"),
  activatedBy: (l: Locale) => t(l, "Activated by", "활성화한 사람", "فعّله"),
  attemptedBy: (l: Locale) => t(l, "Activation attempted by", "활성화 시도자", "مَن حاول التفعيل"),
  holdSeeNotice: (l: Locale) =>
    t(l, "Activation not confirmed. What is still possible until the restart is listed in the notice above.", "활성화가 확인되지 않았습니다. 재시작 전까지 가능한 작업은 위쪽 안내에 있습니다.", "لم يُؤكَّد التفعيل. الإجراءات التي ما زالت ممكنة حتى إعادة التشغيل مذكورة في الإشعار أعلاه."),
  toBeActivatedBy: (l: Locale) => t(l, "To be activated by", "활성화할 사람", "سيُفعّله"),
  notRecorded: (l: Locale) => t(l, "Not recorded", "기록 없음", "غير مسجَّل"),
  notYet: (l: Locale) => t(l, "Not yet", "아직 없음", "ليس بعد"),
  ledgerLine: (l: Locale, records: string, generation: number) =>
    t(l, `Decision ledger: ${records}, ledger generation ${generation}.`, `결정 원장: 기록 ${records}, 원장 세대 ${generation}.`, `دفتر القرارات: ${records}، جيل الدفتر ${generation}.`),
  records: (): Plural => units.records,
  noPolicy: (l: Locale) =>
    t(
      l,
      "No sign-in rules are active yet. Until a trust change is activated, administrators can sign in and members cannot.",
      "아직 적용된 로그인 규칙이 없습니다. 신뢰 변경이 활성화되기 전에는 관리자만 로그인할 수 있습니다.",
      "لا توجد قواعد تسجيل دخول نشطة بعد. إلى أن يُفعَّل تغيير ثقة، يستطيع المسؤولون فقط تسجيل الدخول.",
    ),
  notKept: (l: Locale, at: string) =>
    t(l, `Changes from before the server started at ${at} are not kept.`, `${at}에 서버가 시작되기 전의 변경은 보관되지 않습니다.`, `لا تُحفظ التغييرات السابقة لبدء الخادم في ${at}.`),
  changesTitle: (l: Locale) => t(l, "Trust changes", "신뢰 변경", "تغييرات الثقة"),
  proposeOpen: (l: Locale) => t(l, "Propose trust change", "신뢰 변경 제안", "اقتراح تغيير ثقة"),
  holders: (l: Locale, n: number) =>
    t(
      l,
      `Administrators who can use at least one trust right now: ${n}. A trust change needs three different administrators who hold the propose, approve and activate rights, plus one who holds the validate right.`,
      `현재 신뢰 권한을 하나 이상 사용할 수 있는 관리자: ${n}명. 신뢰 변경에는 제안·승인·활성화 권한을 각각 가진 서로 다른 관리자 3명과 검증 권한을 가진 관리자 1명이 필요합니다.`,
      `عدد المسؤولين الذين يمكنهم استخدام صلاحية ثقة واحدة على الأقل حاليًا: ${n}. تتطلب تغييرات الثقة ثلاثة مسؤولين مختلفين يملكون صلاحيات الاقتراح والاعتماد والتفعيل، إضافةً إلى مسؤول يملك صلاحية التحقق.`,
    ),
  enrolmentLead: (l: Locale) =>
    t(
      l,
      "Trust changes need three different administrators who hold the propose, approve and activate rights, plus one who holds the validate right.",
      "신뢰 변경에는 제안·승인·활성화 권한을 각각 가진 서로 다른 관리자 3명과 검증 권한을 가진 관리자 1명이 필요합니다.",
      "تتطلب تغييرات الثقة ثلاثة مسؤولين مختلفين يملكون صلاحيات الاقتراح والاعتماد والتفعيل، إضافةً إلى مسؤول يملك صلاحية التحقق.",
    ),
  enrolmentRest: (l: Locale, n: number) =>
    t(
      l,
      `Administrators who can use at least one trust right now: ${n}. How further administrators receive trust rights is not decided yet.`,
      `현재 신뢰 권한을 하나 이상 사용할 수 있는 관리자: ${n}명. 다른 관리자에게 신뢰 권한을 부여하는 방법은 아직 정해지지 않았습니다.`,
      `عدد المسؤولين الذين يمكنهم استخدام صلاحية ثقة واحدة على الأقل حاليًا: ${n}. لم يُحدَّد بعد كيف يحصل مسؤولون آخرون على صلاحيات الثقة.`,
    ),
  rightsLine: (l: Locale, list: string) => t(l, `Your trust rights: ${list}.`, `내 신뢰 권한: ${list}.`, `صلاحيات الثقة لديك: ${list}.`),
  noRights: (l: Locale) => t(l, "You hold no trust rights.", "신뢰 권한이 없습니다.", "لا تملك أي صلاحيات ثقة."),
  unavailableLine: (l: Locale) =>
    t(l, "Trust changes are unavailable right now. See the notice at the top.", "지금은 신뢰 변경을 사용할 수 없습니다. 위쪽 안내를 확인하세요.", "تغييرات الثقة غير متاحة الآن. راجع الإشعار في الأعلى."),
  open: (l: Locale) => t(l, "Open", "진행 중", "مفتوحة"),
  closed: (l: Locale) => t(l, "Closed", "종료됨", "مغلقة"),
  openCount: (): Plural => ({ en: ["Open ({n})", "Open ({n})"], ko: "진행 중 ({n})", ar: ["مفتوحة ({n})", "مفتوحة ({n})", "مفتوحة ({n})", "مفتوحة ({n})", "مفتوحة ({n})", "مفتوحة ({n})"] }),
  closedCount: (): Plural => ({ en: ["Closed ({n})", "Closed ({n})"], ko: "종료됨 ({n})", ar: ["مغلقة ({n})", "مغلقة ({n})", "مغلقة ({n})", "مغلقة ({n})", "مغلقة ({n})", "مغلقة ({n})"] }),
  noOpen: (l: Locale) => t(l, "No open changes.", "진행 중인 변경이 없습니다.", "لا توجد تغييرات مفتوحة."),
  noClosed: (l: Locale) => t(l, "No closed changes.", "종료된 변경이 없습니다.", "لا توجد تغييرات مغلقة."),
  colChange: (l: Locale) => t(l, "Change", "변경", "التغيير"),
  colState: (l: Locale) => t(l, "State", "상태", "الحالة"),
  colBased: (l: Locale) => t(l, "Based on", "기준", "مبني على"),
  colNext: (l: Locale) => t(l, "Next step", "다음 단계", "الخطوة التالية"),
  caption: (l: Locale, which: string) => t(l, `${which} trust changes`, `${which} 신뢰 변경`, `تغييرات الثقة: ${which}`),
  change: (l: Locale, n: number) => t(l, `Change ${n}`, `변경 ${n}`, `التغيير ${n}`),
  earlierRun: (l: Locale, at: string) => t(l, `From an earlier server run (started ${at})`, `이전 서버 실행에서 생성(${at} 시작)`, `من تشغيل سابق للخادم (بدأ في ${at})`),
  evidenceChanged: (l: Locale) => t(l, "Evidence changed", "근거 변경됨", "تغيّرت الأدلة"),
  policyNumber: (l: Locale, n: number) => t(l, `Policy ${n}`, `정책 ${n}`, `السياسة ${n}`),
  selected: (l: Locale) => t(l, "Selected", "선택됨", "محدد"),
  close: (l: Locale) => t(l, "Close", "닫기", "إغلاق"),
  back: (l: Locale) => t(l, "Back to trust changes", "신뢰 변경 목록으로", "العودة إلى تغييرات الثقة"),
  nextLabel: (l: Locale) => t(l, "Next step:", "다음 단계:", "الخطوة التالية:"),
  next: {
    draft: (l: Locale) => t(l, "Validate the change.", "변경을 검증하세요.", "تحقّق من التغيير."),
    blocked: (l: Locale) => t(l, "Resolve the blocked checks, then validate again.", "차단된 항목을 해결한 뒤 다시 검증하세요.", "عالج الفحوص التي تمنع المتابعة ثم تحقّق مجددًا."),
    validated: (l: Locale) => t(l, "Needs independent approval by an administrator who did not propose it.", "제안자가 아닌 관리자의 독립 승인이 필요합니다.", "يحتاج إلى اعتماد مستقل من مسؤول غير مقترحه."),
    approved: (l: Locale) => t(l, "Needs activation by a third administrator, not the proposer or approver.", "제안자·승인자가 아닌 세 번째 관리자의 활성화가 필요합니다.", "يحتاج إلى تفعيل من مسؤول ثالث غير المقترح والمعتمد."),
    stale: (l: Locale) => t(l, "Validate again.", "다시 검증하세요.", "تحقّق مجددًا."),
    unknown: (l: Locale) => t(l, "Wait for the server restart.", "서버 재시작을 기다리세요.", "انتظر إعادة تشغيل الخادم."),
    activated: (l: Locale) => t(l, "No further step.", "추가 단계 없음", "لا خطوات أخرى"),
    progress: (l: Locale) => t(l, "Check invalidation progress.", "무효화 진행 상황을 확인하세요.", "تحقّق من تقدّم إلغاء الصلاحية."),
    closed: (l: Locale) => t(l, "Propose a new change if still needed.", "필요하면 새 변경을 제안하세요.", "اقترح تغييرًا جديدًا إذا لزم."),
  },
  why: {
    self: (l: Locale) => t(l, "You proposed this change, so you cannot approve or activate it.", "이 변경을 제안했으므로 승인하거나 활성화할 수 없습니다.", "اقترحت هذا التغيير، فلا يمكنك اعتماده أو تفعيله."),
    approver: (l: Locale) => t(l, "You approved this change, so a third administrator must activate it.", "이 변경을 승인했으므로 세 번째 관리자가 활성화해야 합니다.", "اعتمدت هذا التغيير، فيجب أن يفعّله مسؤول ثالث."),
    no_right: (l: Locale) => t(l, "You do not hold the trust right for this step.", "이 단계에 필요한 신뢰 권한이 없습니다.", "لا تملك صلاحية الثقة لهذه الخطوة."),
    not_validated: (l: Locale) => t(l, "It needs a validation with no blocked checks first.", "먼저 차단된 항목 없이 검증되어야 합니다.", "يجب أولًا أن يجتاز التحقق دون فحوص تمنع المتابعة."),
    needs_approval: (l: Locale) => t(l, "It needs independent approval first.", "먼저 독립 승인이 필요합니다.", "يحتاج أولًا إلى اعتماد مستقل."),
    evidence_stale: (l: Locale) => t(l, "The evidence changed after validation.", "검증 후 근거가 바뀌었습니다.", "تغيّرت الأدلة بعد التحقق."),
    trust_unavailable: (l: Locale) => t(l, "Trust changes are unavailable right now.", "지금은 신뢰 변경을 사용할 수 없습니다.", "تغييرات الثقة غير متاحة الآن."),
  },
  revise: (l: Locale) => t(l, "Revise", "수정", "تعديل"),
  validate: (l: Locale) => t(l, "Validate trust change", "신뢰 변경 검증", "التحقق من تغيير الثقة"),
  approve: (l: Locale) => t(l, "Independently approve trust change", "신뢰 변경 독립 승인", "اعتماد تغيير الثقة اعتمادًا مستقلًا"),
  activate: (l: Locale) => t(l, "Activate approved trust change", "승인된 신뢰 변경 활성화", "تفعيل تغيير الثقة المعتمد"),
  reject: (l: Locale) => t(l, "Reject trust change", "신뢰 변경 반려", "رفض تغيير الثقة"),
  cancel: (l: Locale) => t(l, "Cancel trust proposal", "신뢰 제안 취소", "إلغاء اقتراح الثقة"),
  reviseSubmit: (l: Locale) => t(l, "Revise trust change", "신뢰 변경 수정", "تعديل تغيير الثقة"),
  settings: (l: Locale) => t(l, "Settings", "설정", "الإعدادات"),
  mayBeOlder: (l: Locale) => t(l, "Current values may be older than the last activated change.", "현재 값은 마지막으로 활성화된 변경보다 이전 것일 수 있습니다.", "قد تكون القيم الحالية أقدم من آخر تغيير مُفعَّل."),
  validationTitle: (l: Locale) => t(l, "Validation", "검증", "التحقق"),
  checks: (l: Locale) => t(l, "Checks", "확인 항목", "الفحوص"),
  colCheck: (l: Locale) => t(l, "Check", "항목", "الفحص"),
  colResult: (l: Locale) => t(l, "Result", "결과", "النتيجة"),
  colReason: (l: Locale) => t(l, "Reason", "사유", "السبب"),
  colDetail: (l: Locale) => t(l, "Detail", "세부 정보", "التفاصيل"),
  colItem: (l: Locale) => t(l, "Item", "항목", "البند"),
  validatedAt: (l: Locale, at: string, who: string) => t(l, `Validated at ${at} by ${who}`, `${at}에 ${who}이(가) 검증`, `تم التحقق في ${at} بواسطة ${who}`),
  blockedAt: (l: Locale, at: string, who: string, k: number) =>
    t(
      l,
      k === 1 ? `Blocked at ${at} by ${who}: 1 check blocked` : `Blocked at ${at} by ${who}: ${k} checks blocked`,
      `${at}에 ${who}의 검증에서 ${k}개 항목 차단됨`,
      `توقّف في ${at} بواسطة ${who}: عدد الفحوص التي تمنع المتابعة ${k}`,
    ),
  notValidated: (l: Locale) => t(l, "Not validated", "검증 안 됨", "لم يتم التحقق"),
  missingRoles: (l: Locale, list: string) =>
    t(l, `No administrator who can sign in under the proposed rules holds: ${list}.`, `제안된 규칙으로 로그인할 수 있는 관리자 중 다음 권한을 가진 사람이 없습니다: ${list}.`, `لا يوجد مسؤول يمكنه تسجيل الدخول وفق القواعد المقترحة ويملك: ${list}.`),
  impactTitle: (l: Locale) => t(l, "Impact if activated", "활성화 시 영향", "الأثر عند التفعيل"),
  signsOut: (l: Locale) => t(l, "Everyone in this workspace is signed out at activation.", "활성화하면 이 작업 공간의 모든 사용자가 로그아웃됩니다.", "يُسجَّل خروج الجميع في مساحة العمل هذه عند التفعيل."),
  members: (l: Locale) => t(l, "Members", "구성원", "الأعضاء"),
  sessionsNow: (l: Locale) => t(l, "Open sessions now", "현재 열린 세션", "الجلسات المفتوحة الآن"),
  affected: (l: Locale, n: number) => t(l, `Affected members (${n})`, `영향받는 구성원 (${n})`, `الأعضاء المتأثرون (${n})`),
  showing: (l: Locale, n: number) => t(l, `Showing 100 of ${n}.`, `${n}명 중 100명 표시.`, `عرض 100 من ${n}.`),
  people: (l: Locale) => t(l, "People", "담당자", "الأشخاص"),
  history: (l: Locale) => t(l, "History", "기록", "السجل"),
  invalidated: (l: Locale) => t(l, "Approval invalidated: evidence changed", "승인 무효: 근거 변경", "أُلغي الاعتماد: تغيّرت الأدلة"),
  technical: (l: Locale) => t(l, "Technical details", "기술 세부 정보", "التفاصيل التقنية"),
  changeId: (l: Locale) => t(l, "Change ID", "변경 ID", "معرّف التغيير"),
  changeRevision: (l: Locale) => t(l, "Change revision", "변경 개정 번호", "رقم تعديل التغيير"),
  baseVersion: (l: Locale) => t(l, "Based on version", "기준 버전", "مبني على الإصدار"),
  rawEpoch: (l: Locale) => t(l, "Epoch value", "epoch 값", "قيمة epoch"),
  basePolicy: (l: Locale) => t(l, "Based on policy revision", "기준 정책 개정 번호", "مبني على رقم تعديل السياسة"),
  oldDigest: (l: Locale) => t(l, "Current configuration digest", "현재 구성 다이제스트", "ملخص الإعداد الحالي"),
  newDigest: (l: Locale) => t(l, "Candidate digest", "후보 다이제스트", "ملخص المرشّح"),
  changeDigest: (l: Locale) => t(l, "Change digest", "변경 다이제스트", "ملخص التغيير"),
  evidenceDigest: (l: Locale) => t(l, "Evidence digest", "근거 다이제스트", "ملخص الأدلة"),
  ledgerSequence: (l: Locale) => t(l, "Ledger sequence", "원장 순번", "تسلسل الدفتر"),
  watermark: (l: Locale) => t(l, "Confirmed watermark", "결정 워터마크", "علامة التأكيد"),
  supersededBy: (l: Locale) => t(l, "Superseded by", "대체한 변경", "استُبدل بـ"),
  newVersion: (l: Locale) => t(l, "New version", "새 버전", "الإصدار الجديد"),
  invalidationTitle: (l: Locale) => t(l, "Invalidation progress", "무효화 진행 상황", "تقدّم إلغاء الصلاحية"),
  job: (l: Locale) => t(l, "Job", "작업", "المهمة"),
  jobState: (l: Locale) => t(l, "Job state", "작업 상태", "حالة المهمة"),
  waiting: (l: Locale) => t(l, "Waiting", "대기 중", "قيد الانتظار"),
  jobReason: (l: Locale, r: string) =>
    r === "publish_failed" ? t(l, "Recorded, but not applied in this run", "기록되었으나 이번 실행에 적용되지 않음", "سُجِّل لكن لم يُطبَّق في هذا التشغيل")
      : t(l, "Writing could not be confirmed", "기록 여부를 확인할 수 없음", "تعذّر تأكيد الكتابة"),
  notices: (l: Locale) => t(l, "Notices", "안내", "إشعارات"),
  since: (l: Locale, at: string) => t(l, `Since ${at}`, `${at}부터`, `منذ ${at}`),
  isolatedLead: (l: Locale) => t(l, "The decision ledger cannot be verified.", "결정 원장을 확인할 수 없습니다.", "لا يمكن التحقق من دفتر القرارات."),
  isolatedRest: (l: Locale) =>
    t(
      l,
      "The sign-in rules shown may be older than the last activated change. Trust changes are unavailable until an operator restores the ledger or accepts a new one.",
      "표시된 로그인 규칙이 마지막으로 활성화된 변경보다 이전 것일 수 있습니다. 운영자가 원장을 복구하거나 새 원장을 승인할 때까지 신뢰 변경을 사용할 수 없습니다.",
      "قد تكون قواعد تسجيل الدخول المعروضة أقدم من آخر تغيير مُفعَّل. تغييرات الثقة غير متاحة حتى يستعيد المشغّل الدفتر أو يقبل دفترًا جديدًا.",
    ),
  problemLine: (l: Locale, n: number) => t(l, `(problem at line ${n})`, `(${n}번째 줄에서 문제 발생)`, `(مشكلة في السطر ${n})`),
  durableLead: (l: Locale) =>
    t(l, "The sign-in rules were restored at start from ledger records not confirmed as saved to disk.", "로그인 규칙이 시작 시 디스크 저장이 확인되지 않은 원장 기록에서 복원되었습니다.", "استُعيدت قواعد تسجيل الدخول عند البدء من قيود لم يُؤكَّد حفظها على القرص."),
  durableRest: (l: Locale) =>
    t(l, "The decision ledger stopped accepting records. Trust changes are unavailable until the server restarts.", "결정 원장이 기록을 더 받지 않습니다. 서버를 다시 시작할 때까지 신뢰 변경을 사용할 수 없습니다.", "توقف دفتر القرارات عن قبول القيود. تغييرات الثقة غير متاحة حتى إعادة تشغيل الخادم."),
  stoppedLead: (l: Locale) => t(l, "The decision ledger stopped accepting records.", "결정 원장이 기록을 더 받지 않습니다.", "توقف دفتر القرارات عن قبول القيود."),
  resetLine: (l: Locale, at: string) =>
    t(
      l,
      `Sign-in rules were reset to the installation defaults when the decision ledger was replaced on ${at}.`,
      `${at}에 결정 원장을 교체하면서 로그인 규칙이 설치 기본값으로 초기화되었습니다.`,
      `أُعيد ضبط قواعد تسجيل الدخول إلى الإعدادات الافتراضية للتثبيت عند استبدال دفتر القرارات في ${at}.`,
    ),
  carriedLine: (l: Locale, n: number) =>
    t(
      l,
      `The carried configuration may be older than the last activation (problem at line ${n}).`,
      `이어받은 구성이 마지막 활성화보다 이전 것일 수 있습니다(${n}번째 줄에서 문제 발생).`,
      `قد يكون الإعداد المنقول أقدم من آخر تفعيل (مشكلة في السطر ${n}).`,
    ),
  stillPossible: (l: Locale) => t(l, "Still possible", "가능한 작업", "ما زال ممكنًا"),
  notPossible: (l: Locale) => t(l, "Not possible until restart", "재시작 전까지 불가능한 작업", "غير ممكن حتى إعادة التشغيل"),
  formTitle: (l: Locale) => t(l, "Propose trust change", "신뢰 변경 제안", "اقتراح تغيير ثقة"),
  reviseTitle: (l: Locale, n: number) => t(l, `Revise change ${n}`, `변경 ${n} 수정`, `تعديل التغيير ${n}`),
  formLead: (l: Locale) =>
    t(
      l,
      "A trust change becomes active only after validation, independent approval and activation by three different administrators. In this pre-launch version, sign-in is by password only and MFA is not required.",
      "신뢰 변경은 서로 다른 관리자 3명이 검증, 독립 승인, 활성화를 마쳐야 적용됩니다. 출시 전 서비스에서는 비밀번호 로그인만 사용하며 MFA는 요구하지 않습니다.",
      "لا يصبح تغيير الثقة نشطًا إلا بعد التحقق والاعتماد المستقل والتفعيل من ثلاثة مسؤولين مختلفين. في خدمة ما قبل الإطلاق يبقى تسجيل الدخول بكلمة المرور فقط ولا تُطلب MFA.",
    ),
  review: (l: Locale) => t(l, "Review proposal", "제안 검토", "مراجعة الاقتراح"),
  goBack: (l: Locale) => t(l, "Go back", "돌아가기", "رجوع"),
  reloadClose: (l: Locale) => t(l, "Reload and close", "다시 불러오고 닫기", "إعادة التحميل والإغلاق"),
  confirmTitle: (l: Locale, action: string, n: number) => (n > 0 ? `${action}: ${t(l, `Change ${n}`, `변경 ${n}`, `التغيير ${n}`)}` : action),
  newChange: (l: Locale) => t(l, "New change", "새 변경", "تغيير جديد"),
  workspace: (l: Locale) => t(l, "Workspace", "작업 공간", "مساحة العمل"),
  route: (l: Locale) => t(l, "Route", "경로", "المسار"),
  localPassword: (l: Locale) => t(l, "Local password", "로컬 비밀번호", "كلمة مرور محلية"),
  membersAffected: (l: Locale) => t(l, "Members affected", "영향받는 구성원 수", "عدد الأعضاء المتأثرين"),
  expected: (l: Locale) => t(l, "expected", "예상값", "متوقَّع"),
  serverAuthoritative: (l: Locale) =>
    t(l, "Computed in this browser for display; the server's value is authoritative.", "표시용으로 이 브라우저에서 계산했습니다. 서버 값이 기준입니다.", "حُسب في هذا المتصفح للعرض فقط؛ القيمة المعتمدة هي قيمة الخادم."),
  revisionStep: (l: Locale, from: number, to: number) =>
    t(l, `Change revision ${from} becomes ${to}`, `변경 개정 번호가 ${from}에서 ${to}(으)로 바뀝니다`, `يصبح رقم تعديل التغيير ${to} بدلًا من ${from}`),
  currentState: (l: Locale) => t(l, "Current state", "현재 상태", "الحالة الحالية"),
  leads: {
    propose: (l: Locale) => t(l, "Creates a draft trust change from these settings.", "이 설정으로 신뢰 변경 초안을 만듭니다.", "يُنشئ مسودة تغيير ثقة من هذه الإعدادات."),
    validate: (l: Locale) => t(l, "Records validation evidence for the exact change shown.", "표시된 변경에 대한 검증 근거를 기록합니다.", "يسجّل أدلة التحقق للتغيير المعروض بعينه."),
    approve: (l: Locale) => t(l, "Records your independent approval of the exact change and evidence shown.", "표시된 변경과 근거를 독립적으로 승인합니다.", "يسجّل اعتمادك المستقل للتغيير والأدلة المعروضة بعينها."),
    activate: (l: Locale) => t(l, "Makes this change the active sign-in rules under a newer trust version.", "더 새로운 신뢰 버전으로 이 변경을 현재 로그인 규칙으로 적용합니다.", "يجعل هذا التغيير قواعد تسجيل الدخول النشطة بإصدار ثقة أحدث."),
    reject: (l: Locale) => t(l, "Rejects this proposal.", "이 제안을 반려합니다.", "يرفض هذا الاقتراح."),
    cancel: (l: Locale) => t(l, "Cancels your proposal.", "내 제안을 취소합니다.", "يلغي اقتراحك."),
  },
  consequence: {
    propose: (l: Locale) => t(l, "If it is activated later, everyone in the workspace is signed out.", "나중에 활성화되면 작업 공간의 모든 사용자가 로그아웃됩니다.", "إذا فُعِّل لاحقًا، يُسجَّل خروج الجميع في مساحة العمل."),
    proposeRisk: (l: Locale) => t(l, "Lockout risk is checked at validation (test login and recovery path).", "잠김 위험은 검증 단계에서 확인합니다(테스트 로그인과 복구 경로).", "يُفحص خطر الإقفال عند التحقق (تسجيل الدخول التجريبي ومسار الاسترداد)."),
    validate: (l: Locale) => t(l, "Your current session is used as the test login under the proposed rules.", "현재 세션을 제안된 규칙의 테스트 로그인으로 사용합니다.", "تُستخدم جلستك الحالية كتسجيل دخول تجريبي وفق القواعد المقترحة."),
    validateBlocked: (l: Locale) => t(l, "A blocked result keeps the change a draft.", "차단된 결과가 나오면 변경은 초안으로 남습니다.", "تُبقي نتيجة تمنع المتابعة التغيير مسودة."),
    approve: (l: Locale) => t(l, "Everyone is signed out at activation.", "활성화하면 모든 사용자가 로그아웃됩니다.", "يُسجَّل خروج الجميع عند التفعيل."),
    approveRecheck: (l: Locale) => t(l, "Activation rechecks authority and the old version.", "활성화할 때 권한과 이전 버전을 다시 확인합니다.", "يعيد التفعيل التحقق من الصلاحية والإصدار السابق."),
    activate: (l: Locale) => t(l, "Everyone in this workspace is signed out, including you.", "나를 포함해 이 작업 공간의 모든 사용자가 로그아웃됩니다.", "يُسجَّل خروج الجميع في مساحة العمل هذه، وأنت منهم."),
    activateUndo: (l: Locale) => t(l, "Undoing this needs a new trust change with a newer version.", "되돌리려면 더 새로운 버전의 새 신뢰 변경이 필요합니다.", "يتطلب التراجع عن هذا تغيير ثقة جديدًا بإصدار أحدث."),
    reject: (l: Locale) => t(l, "A rejected change cannot be reopened; propose again if needed.", "반려된 변경은 다시 열 수 없습니다. 필요하면 다시 제안하세요.", "لا يمكن إعادة فتح تغيير مرفوض؛ اقترح مجددًا إذا لزم."),
    cancel: (l: Locale) => t(l, "A cancelled proposal cannot be reopened.", "취소된 제안은 다시 열 수 없습니다.", "لا يمكن إعادة فتح اقتراح ملغى."),
    unchanged: (l: Locale) => t(l, "Active sign-in rules do not change.", "현재 로그인 규칙은 바뀌지 않습니다.", "لا تتغير قواعد تسجيل الدخول النشطة."),
  },
  boundary: {
    propose: (l: Locale) =>
      t(l, "Creates a draft only; active sign-in rules do not change. You can revise or cancel until activation.", "초안만 만듭니다. 현재 로그인 규칙은 바뀌지 않으며 활성화 전까지 수정하거나 취소할 수 있습니다.", "يُنشئ مسودة فقط؛ لا تتغير قواعد تسجيل الدخول النشطة. يمكنك التعديل أو الإلغاء حتى التفعيل."),
    validate: (l: Locale) => t(l, "Validation records evidence only; it does not approve or activate.", "검증은 근거만 기록하며 승인하거나 활성화하지 않습니다.", "يسجّل التحقق الأدلة فقط؛ ولا يعتمد ولا يفعّل."),
    approve: (l: Locale) =>
      t(
        l,
        "Approval is cleared if the change is revised, and cannot be used if the evidence changes; reject and cancel stay possible until activation.",
        "변경이 수정되면 승인은 해제되고, 근거가 바뀌면 승인을 사용할 수 없습니다. 활성화 전까지 반려와 취소가 가능합니다.",
        "يُلغى الاعتماد إذا عُدِّل التغيير، ولا يمكن استخدامه إذا تغيّرت الأدلة؛ ويبقى الرفض والإلغاء ممكنين حتى التفعيل.",
      ),
    activate: (l: Locale) => t(l, "After you confirm, this change cannot be rejected or cancelled.", "확인하면 이 변경은 반려하거나 취소할 수 없습니다.", "بعد التأكيد لا يمكن رفض هذا التغيير أو إلغاؤه."),
    reject: (l: Locale) => t(l, "Rejection is final.", "반려는 되돌릴 수 없습니다.", "الرفض نهائي."),
    cancel: (l: Locale) => t(l, "Cancellation is final.", "취소는 되돌릴 수 없습니다.", "الإلغاء نهائي."),
  },
  authority: (l: Locale) => t(l, "You hold the approval right and did not propose this change.", "승인 권한이 있으며 이 변경을 제안하지 않았습니다.", "تملك صلاحية الاعتماد ولم تقترح هذا التغيير."),
  you: (l: Locale) => t(l, "You", "나", "أنت"),
  expectedVersion: (l: Locale) => t(l, "Expected new version", "예상 새 버전", "الإصدار الجديد المتوقَّع"),
  checked: (l: Locale) => t(l, "Checked", "확인 대상", "يُفحص"),
  lockout: (l: Locale) => t(l, "Lockout risk", "잠김 위험", "خطر الإقفال"),
  yourAuthority: (l: Locale) => t(l, "Your authority", "내 권한", "صلاحيتك"),
  activatedTitle: (l: Locale) => t(l, "Activated", "활성화됨", "مُفعَّل"),
  waitingTitle: (l: Locale) => t(l, "Activation not confirmed", "활성화 확인 안 됨", "لم يُؤكَّد التفعيل"),
  sessionsEnded: (l: Locale) => t(l, "Sessions ended", "종료된 세션", "الجلسات المنتهية"),
  sessionsRemaining: (l: Locale) => t(l, "Sessions remaining", "남은 세션", "الجلسات المتبقية"),
  afterActivated: (l: Locale) =>
    t(
      l,
      "The change is activated. Your session has ended. Sign in again to see the remaining invalidation progress in the change detail.",
      "변경이 활성화되었습니다. 내 세션이 종료되었습니다. 다시 로그인하면 변경 상세에서 남은 무효화 진행 상황을 확인할 수 있습니다.",
      "تم تفعيل التغيير. انتهت جلستك. سجّل الدخول مجددًا لعرض تقدّم إلغاء الصلاحية المتبقي في تفاصيل التغيير.",
    ),
  afterUnclear: (l: Locale) =>
    t(
      l,
      "The server's reply did not confirm whether the change was activated. Sign in again to see its current state.",
      "서버 응답으로는 변경이 활성화되었는지 확인할 수 없습니다. 다시 로그인해 현재 상태를 확인하세요.",
      "لم يؤكّد ردّ الخادم ما إذا كان التغيير قد فُعِّل. سجّل الدخول مجددًا لعرض حالته الحالية.",
    ),
  afterUnclearTitle: (l: Locale) => t(l, "Result not confirmed", "결과 확인 안 됨", "لم تُؤكَّد النتيجة"),
  reload: (l: Locale) => t(l, "Reload", "다시 불러오기", "إعادة التحميل"),
  stricterFields: (l: Locale) => t(l, "Rules tightened by the restored/default comparison", "복원값·기본값 비교로 강화된 규칙", "القواعد المشدّدة بمقارنة القيم المستعادة والافتراضية"),
  fieldsUnknown: (l: Locale) => t(l, "Unknown", "알 수 없음", "غير معروف"),
  noStricterFields: (l: Locale) => t(l, "None", "없음", "لا شيء"),
  olderIgnored: (l: Locale) =>
    t(
      l,
      "The decision ledger file is from an older ledger generation than this installation expects, so its sign-in rules were not used as they are. For each rule the stricter of the installation default and that file's value applies.",
      "결정 원장 파일이 이 설치가 기대하는 세대보다 이전 것이어서 그 로그인 규칙을 그대로 사용하지 않았습니다. 규칙마다 설치 기본값과 그 파일의 값 중 더 엄격한 쪽을 적용합니다.",
      "ملف دفتر القرارات من جيل أقدم مما يتوقعه هذا التثبيت، لذا لم تُستخدم قواعد تسجيل الدخول فيه كما هي. يُطبَّق لكل قاعدة الأشد بين الإعداد الافتراضي للتثبيت وقيمة ذلك الملف.",
    ),
  signInAgain: (l: Locale) => t(l, "Sign in again", "다시 로그인", "تسجيل الدخول مجددًا"),
  done: {
    propose: (l: Locale, n: number) => t(l, `Change ${n} proposed as a draft.`, `변경 ${n}을(를) 초안으로 제안했습니다.`, `اقتُرح التغيير ${n} كمسودة.`),
    revise: (l: Locale, n: number) => t(l, `Change ${n} revised; it is a draft again.`, `변경 ${n}을(를) 수정했습니다. 다시 초안 상태입니다.`, `عُدِّل التغيير ${n} وعاد مسودة.`),
    validate: (l: Locale, n: number) => t(l, `Change ${n} validation recorded.`, `변경 ${n}의 검증을 기록했습니다.`, `سُجّل التحقق من التغيير ${n}.`),
    approve: (l: Locale, n: number) => t(l, `Change ${n} independently approved.`, `변경 ${n}을(를) 독립 승인했습니다.`, `اعتُمد التغيير ${n} اعتمادًا مستقلًا.`),
    reject: (l: Locale, n: number) => t(l, `Change ${n} rejected.`, `변경 ${n}을(를) 반려했습니다.`, `رُفض التغيير ${n}.`),
    cancel: (l: Locale, n: number) => t(l, `Change ${n} cancelled.`, `변경 ${n}을(를) 취소했습니다.`, `أُلغي التغيير ${n}.`),
  },
  fail: {
    title: (l: Locale) => t(l, "The trust change was not saved", "신뢰 변경을 저장하지 못했습니다", "لم يُحفظ تغيير الثقة"),
    self: (l: Locale) =>
      t(l, "A trust change needs three different people: the proposer, the approver and the activator. Reload and check who acted.", "신뢰 변경에는 제안자, 승인자, 활성화한 사람이 서로 다른 세 사람이어야 합니다. 다시 불러와 누가 처리했는지 확인하세요.", "يحتاج تغيير الثقة إلى ثلاثة أشخاص مختلفين: المقترح والمعتمد والمفعِّل. أعد التحميل وتحقّق ممن قام بالإجراء."),
    epoch: (l: Locale) =>
      t(l, "The active trust version changed. This change is based on an older version and needs a new proposal.", "현재 신뢰 버전이 바뀌었습니다. 이 변경은 이전 버전을 기준으로 하므로 새로 제안해야 합니다.", "تغيّر إصدار الثقة النشط. هذا التغيير مبني على إصدار أقدم ويحتاج إلى اقتراح جديد."),
    approval_stale: (l: Locale) =>
      t(l, "The change was revised after you loaded it. Reload and review the current revision.", "불러온 뒤 변경이 수정되었습니다. 다시 불러와 현재 개정을 검토하세요.", "عُدِّل التغيير بعد تحميله. أعد التحميل وراجع التعديل الحالي."),
    evidence_stale: (l: Locale) =>
      t(
        l,
        "The evidence changed after validation, so the recorded approval cannot be used to activate. Validate again, then get a new independent approval.",
        "검증 후 근거가 바뀌어 기록된 승인으로는 활성화할 수 없습니다. 다시 검증한 뒤 독립 승인을 새로 받으세요.",
        "تغيّرت الأدلة بعد التحقق، لذا لا يمكن استخدام الاعتماد المسجَّل للتفعيل. تحقّق مجددًا ثم احصل على اعتماد مستقل جديد.",
      ),
    needs_approval: (l: Locale) => t(l, "This change needs independent approval before it can be activated.", "활성화하려면 먼저 독립 승인이 필요합니다.", "يحتاج هذا التغيير إلى اعتماد مستقل قبل تفعيله."),
    trust_unavailable: (l: Locale) =>
      t(
        l,
        "Nothing was activated. The decision ledger stopped accepting records. Restart the server. If the ledger cannot be verified at the next start it is isolated, trust changes stay unavailable, and an operator must restore it or accept a new one.",
        "아무것도 활성화되지 않았습니다. 결정 원장이 기록을 더 받지 않습니다. 서버를 다시 시작하세요. 다음 시작 때 원장을 확인할 수 없으면 격리되어 신뢰 변경을 계속 사용할 수 없으며, 운영자가 원장을 복구하거나 새 원장을 승인해야 합니다.",
        "لم يُفعَّل شيء. توقف دفتر القرارات عن قبول القيود. أعد تشغيل الخادم. إذا تعذّر التحقق من الدفتر عند البدء التالي فسيُعزَل، وتبقى تغييرات الثقة غير متاحة، ويجب على المشغّل استعادته أو قبول دفتر جديد.",
      ),
    held: (l: Locale) => t(l, "Trust changes are unavailable until the server restarts.", "서버를 다시 시작할 때까지 신뢰 변경을 사용할 수 없습니다.", "تغييرات الثقة غير متاحة حتى إعادة تشغيل الخادم."),
    stage: (l: Locale) => t(l, "This change is no longer in a state that allows this step. It has been reloaded.", "이 변경은 더 이상 이 단계를 진행할 수 있는 상태가 아닙니다. 다시 불러왔습니다.", "لم يعد هذا التغيير في حالة تسمح بهذه الخطوة. أُعيد تحميله."),
    denied: (l: Locale) => t(l, "You do not hold the trust right for this step, or the change no longer exists.", "이 단계에 필요한 신뢰 권한이 없거나 변경이 더 이상 존재하지 않습니다.", "لا تملك صلاحية الثقة لهذه الخطوة، أو لم يعد التغيير موجودًا."),
    unchanged: (l: Locale) => t(l, "These values match the active sign-in rules.", "현재 로그인 규칙과 같은 값입니다.", "هذه القيم مطابقة لقواعد تسجيل الدخول النشطة."),
  },
  loadFailedBody: (l: Locale) => t(l, "The trust changes could not be loaded.", "신뢰 변경을 불러오지 못했습니다.", "تعذّر تحميل تغييرات الثقة."),
  revoked: (l: Locale) =>
    t(l, "Your access to these trust changes ended. Details were cleared. Reload to check your access.", "이 신뢰 변경에 대한 접근 권한이 종료되어 세부 정보를 지웠습니다. 다시 불러와 접근 권한을 확인하세요.", "انتهى وصولك إلى تغييرات الثقة هذه، ومُسحت التفاصيل. أعد التحميل للتحقق من وصولك."),
  revokedTitle: (l: Locale) => t(l, "Access changed", "접근 권한 변경됨", "تغيّر الوصول"),
};
