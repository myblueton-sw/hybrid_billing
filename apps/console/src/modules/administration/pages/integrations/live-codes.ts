import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export type Words = Record<string, [string, string, string]>;

export const unknownCode = (l: Locale, code: string) => t(l, `Unknown (${code})`, `알 수 없음(${code})`, `غير معروف (${code})`);

export function words(l: Locale, table: Words, key: string): string {
  const w = Object.prototype.hasOwnProperty.call(table, key) ? table[key] : undefined;
  return w ? t(l, w[0], w[1], w[2]) : unknownCode(l, key);
}

export const sectionWords: Words = {
  configured: ["Configured", "설정됨", "مُعدّ"],
  not_configured: ["Not configured", "설정되지 않음", "غير مُعدّ"],
};

export const connectionWords: Words = {
  not_connected: ["Not connected", "연결되지 않음", "غير متصل"],
  not_issued: ["Not issued", "발급되지 않음", "لم يُصدَر"],
};

export const clientKindWords: Words = {
  first_party_console: ["First-party console", "자사 콘솔", "وحدة التحكم الأساسية"],
};

export const clientStateWords: Words = {
  active: ["Active", "사용 중", "نشط"],
};

export const grantStateWords: Words = {
  active: ["Active", "유효", "سارية"],
  pending: ["Not yet valid", "유효 시작 전", "لم تبدأ بعد"],
  expired: ["Expired", "만료됨", "منتهية"],
  revoked: ["Revoked", "회수됨", "مُلغاة"],
};

export const scopeWords: Words = {
  all: ["All", "전체", "الكل"],
  any: ["Any period", "전체 기간", "أي فترة"],
};

export const serviceUserWords: Words = {
  active: ["Active", "사용 중", "نشط"],
  suspended: ["Suspended", "정지됨", "موقوف"],
  retired: ["Retired", "폐기됨", "متقاعد"],
};

export const installWords: Words = {
  installed: ["Installed (not enabled)", "설치됨(사용 안 함)", "مثبّتة (غير مفعّلة)"],
  enabled: ["Enabled", "사용 설정됨", "مفعّلة"],
  stopped: ["Stopped", "중지됨", "متوقفة"],
  removed: ["Removed", "제거됨", "مُزالة"],
};

export const installReasonWords: Words = {
  operator: ["Stopped by an administrator", "관리자가 중지함", "أوقفها مسؤول"],
  timeout: ["Stopped after a timeout", "시간 초과로 중지됨", "توقفت بعد انتهاء المهلة"],
};

export const capabilityWords: Words = {
  read_events: ["Read events", "이벤트 읽기", "قراءة الأحداث"],
  call_command: ["Call command", "명령 호출", "استدعاء أمر"],
  serve_screen: ["Serve screen", "화면 제공", "تقديم شاشة"],
};

export const componentWords: Words = {
  table: ["Table", "표", "جدول"],
  form: ["Form", "입력 양식", "نموذج"],
  detail: ["Detail", "상세", "تفاصيل"],
  chart: ["Chart", "차트", "مخطط"],
};

export const deletionWords: Words = {
  delete_on_removal: ["Deleted on removal", "제거 시 삭제", "تُحذف عند الإزالة"],
  retain_for_evidence: ["Kept for evidence", "증빙용 보관", "تُحفظ كدليل"],
};

export const migrationWords: Words = {
  none: ["No migration", "이관 없음", "بلا ترحيل"],
  additive: ["Additive changes only", "추가 변경만", "تغييرات إضافية فقط"],
  copy_forward: ["Copied forward", "새 버전으로 복사", "نسخ إلى الإصدار الجديد"],
};

export const proposalKindWords: Words = {
  publisher: ["Publisher", "게시자", "الناشر"],
  key: ["Publisher key", "게시자 키", "مفتاح الناشر"],
  version: ["Plugin version", "플러그인 버전", "إصدار الإضافة"],
  workspace: ["Plugin workspace", "플러그인 업무 공간", "مساحة عمل الإضافات"],
  runtime: ["Runtime identity", "실행 ID", "هوية التشغيل"],
  install: ["Install", "설치", "تثبيت"],
  upgrade: ["Upgrade", "업그레이드", "ترقية"],
  rollback: ["Rollback", "롤백", "إرجاع"],
  enable: ["Enable", "사용 설정", "تفعيل"],
  remove: ["Remove", "제거", "إزالة"],
  endpoint: ["Endpoint", "엔드포인트", "نقطة النهاية"],
};

export const proposalStateWords: Words = {
  pending: ["Pending", "승인 대기", "بانتظار الاعتماد"],
  applied: ["Applied", "적용됨", "مطبّق"],
  withdrawn: ["Withdrawn", "철회됨", "مسحوب"],
  closed: ["Closed", "종료됨", "مُغلق"],
};

export const destinationWords: Words = {
  active: ["Active", "사용 가능", "نشطة"],
  inactive: ["Inactive", "사용 불가", "غير نشطة"],
  revoked: ["Revoked", "회수됨", "مُلغاة"],
};

export const sourceWords: Words = {
  sample_allowlist: ["Sample allowlist", "샘플 허용 목록", "قائمة السماح التجريبية"],
};

export const subscriptionWords: Words = {
  proposed: ["Proposed", "승인 대기", "مقترح"],
  active: ["Active", "활성", "نشط"],
  paused: ["Paused", "일시 중지됨", "متوقف مؤقتًا"],
  suspended: ["Suspended", "자동 정지됨", "معلّق"],
  revoked: ["Revoked", "회수됨", "مُلغى"],
};

export const subscriptionReasonWords: Words = {
  paused: ["Paused by an administrator", "관리자가 일시 중지함", "أوقفه مسؤول مؤقتًا"],
  gap: ["Events were skipped (expired cursor)", "이벤트 누락(만료된 커서)", "تخطّي أحداث (مؤشر منتهي)"],
  backlog: ["Too many undelivered items", "미전달 건 과다", "عناصر غير مسلّمة كثيرة"],
  destination_inactive: ["Destination not active", "목적지 사용 불가", "الوجهة غير نشطة"],
  destination_revoked: ["Destination revoked", "목적지 회수됨", "الوجهة مُلغاة"],
  permission_revoked: ["Permission revoked", "권한 회수됨", "الصلاحية مُلغاة"],
  subscription_revoked: ["Revoked by an administrator", "관리자가 회수함", "ألغاه مسؤول"],
  dependency_unavailable: ["A required check was unavailable", "필수 확인을 사용할 수 없었음", "تعذّر إجراء فحص مطلوب"],
  key_unpublished: ["No published signing key", "게시된 서명 키 없음", "لا يوجد مفتاح توقيع منشور"],
  key_mismatch: ["Signing key mismatch", "서명 키 불일치", "عدم تطابق مفتاح التوقيع"],
  withdrawn: ["Withdrawn by the proposer", "제안자가 철회함", "سحبه صاحب الاقتراح"],
  requester_lost_right: ["The person who started it no longer holds the right", "시작한 사람에게 더 이상 권한이 없음", "لم يعد من بدأه يملك الصلاحية"],
};

export const keyStateWords: Words = {
  proposed: ["Proposed", "승인 대기", "مقترح"],
  active: ["Active", "사용 중", "نشط"],
  retired: ["Retired", "폐기됨", "متقاعد"],
};

export const historyWords: Words = {
  approve: ["Approved", "승인", "اعتُمد"],
  pause: ["Paused", "일시 중지", "أُوقف مؤقتًا"],
  request_resume: ["Resume requested", "재개 요청", "طُلب الاستئناف"],
  approve_resume: ["Resume approved", "재개 승인", "اعتُمد الاستئناف"],
  revoke: ["Revoked", "회수", "أُلغي"],
  withdraw: ["Withdrawn", "철회", "سُحب"],
  withdraw_resume: ["Resume request withdrawn", "재개 요청 철회", "سُحب طلب الاستئناف"],
  proposal_closed: ["Closed", "종료", "أُغلق"],
  resume_closed: ["Resume request closed", "재개 요청 종료", "أُغلق طلب الاستئناف"],
  redeliver_request_closed: ["Redelivery request closed", "재발송 요청 종료", "أُغلق طلب إعادة التسليم"],
  dismiss_request_closed: ["Dismissal request closed", "정리 요청 종료", "أُغلق طلب الإزالة"],
  redelivery_closed: ["Redelivery closed", "재발송 종료", "أُغلقت إعادة التسليم"],
};

export const eventStateWords: Words = {
  active: ["Approved", "승인됨", "معتمد"],
};

export const deliveryKindWords: Words = {
  open: ["Delivery", "발송 건", "تسليم"],
  redelivery: ["Redelivery", "재발송 건", "إعادة تسليم"],
  dead: ["Dead letter", "실패 보관 건", "رسالة غير مُسلّمة"],
  cancelled: ["Cancelled", "취소됨", "مُلغى"],
};

export const jobWords: Words = {
  queued: ["Queued", "대기열 등록", "في قائمة الانتظار"],
  running: ["Running", "실행 중", "قيد التشغيل"],
  waiting: ["Waiting", "대기 중", "في الانتظار"],
  succeeded: ["Succeeded", "성공", "نجح"],
  failed: ["Failed", "실패", "فشل"],
  cancelled: ["Cancelled", "취소됨", "أُلغي"],
};

export const deliveryReasonWords: Words = {
  rejected: ["Rejected by the receiver", "수신 측이 거부함", "رفضها المستلم"],
  attempts_exhausted: ["Attempts exhausted", "재시도 횟수 소진", "استُنفدت المحاولات"],
  max_age: ["Too old to deliver", "전달 가능 기간 초과", "قديمة جدًا للتسليم"],
  delivered: ["Delivered", "전달됨", "سُلّمت"],
  subscription_revoked: subscriptionReasonWords.subscription_revoked,
  permission_revoked: subscriptionReasonWords.permission_revoked,
  destination_inactive: subscriptionReasonWords.destination_inactive,
  destination_revoked: subscriptionReasonWords.destination_revoked,
  dependency_unavailable: subscriptionReasonWords.dependency_unavailable,
  requester_lost_right: ["Requester no longer holds the right", "요청자에게 더 이상 권한이 없음", "لم يعد مقدّم الطلب يملك الصلاحية"],
};

export const resultWords: Words = {
  requested: ["Requested", "요청됨", "مطلوب"],
  dismissed: ["Dismissed", "정리됨", "أُزيلت"],
  ...proposalStateWords,
  ...subscriptionWords,
  ...installWords,
  ...jobWords,
};

export const changeWords: Words = {
  install: ["Install {v}", "{v} 설치", "تثبيت {v}"],
  upgrade: ["Upgrade to {v}", "{v}(으)로 업그레이드", "الترقية إلى {v}"],
  rollback: ["Roll back to {v}", "{v}(으)로 롤백", "الإرجاع إلى {v}"],
  enable: ["Enable", "사용 설정", "تفعيل"],
  remove: ["Remove", "제거", "إزالة"],
};

export const summaryWords: Words = {
  publisher: ["Register publisher {name} ({id})", "게시자 {name}({id}) 등록", "تسجيل الناشر {name} ({id})"],
  key: ["Register publisher key {key} of {publisher}, valid {from} to {to}", "{publisher} 게시자 키 {key} 등록, {from}~{to} 유효", "تسجيل مفتاح الناشر {key} لـ {publisher}، صالح من {from} إلى {to}"],
  version: ["Register version {version} of {plugin}", "{plugin} 버전 {version} 등록", "تسجيل الإصدار {version} من {plugin}"],
  workspace: ["Register plugin workspace {workspace} (runtime {runtime})", "플러그인 업무 공간 {workspace} 등록(실행 ID {runtime})", "تسجيل مساحة عمل الإضافات {workspace} (التشغيل {runtime})"],
  runtime: ["Change the runtime of {workspace} to {runtime}", "{workspace} 실행 ID를 {runtime}(으)로 변경", "تغيير بيئة تشغيل {workspace} إلى {runtime}"],
  install: ["Install {plugin} {version}", "{plugin} {version} 설치", "تثبيت {plugin} {version}"],
  upgrade: ["Upgrade {plugin} to {version}", "{plugin}을(를) {version}(으)로 업그레이드", "ترقية {plugin} إلى {version}"],
  rollback: ["Roll back {plugin} to {version}", "{plugin}을(를) {version}(으)로 롤백", "إرجاع {plugin} إلى {version}"],
  enable: ["Enable {plugin}", "{plugin} 사용 설정", "تفعيل {plugin}"],
  remove: ["Remove {plugin}", "{plugin} 제거", "إزالة {plugin}"],
  endpoint: ["Approve endpoint {endpoint} of {plugin}, certificate {fingerprint}", "{plugin} 엔드포인트 {endpoint} 승인, 인증서 {fingerprint}", "اعتماد نقطة النهاية {endpoint} لـ {plugin}، الشهادة {fingerprint}"],
};

export const blockerWords: Words = {
  own_proposal: ["You proposed this. Another administrator must approve it.", "본인이 제안했습니다. 다른 관리자가 승인해야 합니다.", "أنت من اقترح هذا. يجب أن يعتمده مسؤول آخر."],
  not_pending: ["This is no longer waiting for a decision.", "더 이상 결정을 기다리고 있지 않습니다.", "لم يعد هذا بانتظار قرار."],
  own_request: ["You requested this. Another administrator must approve it.", "본인이 요청했습니다. 다른 관리자가 승인해야 합니다.", "أنت من طلب هذا. يجب أن يعتمده مسؤول آخر."],
  already_requested: ["A request is already waiting for approval.", "이미 승인을 기다리는 요청이 있습니다.", "يوجد طلب بانتظار الاعتماد بالفعل."],
  not_requested: ["Nothing has been requested yet.", "아직 요청된 것이 없습니다.", "لم يُطلب شيء بعد."],
  not_active: ["Only an active subscription can be paused.", "활성 구독만 일시 중지할 수 있습니다.", "يمكن إيقاف الاشتراك النشط فقط مؤقتًا."],
  destination_inactive: ["The destination is not active.", "목적지를 사용할 수 없습니다.", "الوجهة غير نشطة."],
  subscribe_refused: ["The subscriber is not permitted to receive these event types.", "구독자에게 이 이벤트 유형을 받을 권한이 없습니다.", "المشترك غير مسموح له بتلقي أنواع الأحداث هذه."],
  dependency_unavailable: ["A required check is unavailable. Try again later.", "필수 확인을 지금 사용할 수 없습니다. 나중에 다시 시도하세요.", "فحص مطلوب غير متاح. حاول لاحقًا."],
  not_enabled: ["Only an enabled plugin can be stopped.", "사용 설정된 플러그인만 중지할 수 있습니다.", "يمكن إيقاف الإضافة المفعّلة فقط."],
  not_allowed_state: ["This is not allowed in the plugin's current state.", "플러그인의 현재 상태에서는 허용되지 않습니다.", "هذا غير مسموح في حالة الإضافة الحالية."],
  direction: ["The version is not newer (upgrade) or older (rollback) than the installed one.", "버전이 설치된 버전보다 새롭지(업그레이드) 않거나 오래되지(롤백) 않았습니다.", "الإصدار ليس أحدث (ترقية) أو أقدم (إرجاع) من المثبّت."],
  storage_changed: ["The version changes the storage contract, so it cannot replace the installed one.", "버전이 저장 계약을 바꾸므로 설치된 버전을 대체할 수 없습니다.", "يغيّر الإصدار عقد التخزين، لذا لا يمكنه استبدال المثبّت."],
  namespace_taken: ["Another installation already uses this storage namespace.", "다른 설치가 이미 이 저장 네임스페이스를 사용합니다.", "تستخدم عملية تثبيت أخرى مساحة أسماء التخزين هذه بالفعل."],
  runtime_taken: ["Another workspace already uses this runtime identity.", "다른 업무 공간이 이미 이 실행 ID를 사용합니다.", "تستخدم مساحة عمل أخرى هوية التشغيل هذه بالفعل."],
  not_declared: ["The installed version does not declare this endpoint.", "설치된 버전이 이 엔드포인트를 선언하지 않았습니다.", "لا يعلن الإصدار المثبّت عن نقطة النهاية هذه."],
  fingerprint_mismatch: ["The certificate fingerprint does not match.", "인증서 지문이 일치하지 않습니다.", "بصمة الشهادة غير مطابقة."],
  key_inactive: ["The signing key is not valid now.", "서명 키가 지금 유효하지 않습니다.", "مفتاح التوقيع غير صالح الآن."],
  key_life: ["The key's validity period is outside the allowed range.", "키 유효 기간이 허용 범위를 벗어났습니다.", "فترة صلاحية المفتاح خارج النطاق المسموح."],
  key_mismatch: ["The key does not match its proof.", "키가 증명과 일치하지 않습니다.", "المفتاح لا يطابق إثباته."],
  weak_key: ["The key is weak and cannot be used.", "키가 약해 사용할 수 없습니다.", "المفتاح ضعيف ولا يمكن استخدامه."],
  publisher_inactive: ["The publisher is not registered or not active.", "게시자가 등록되지 않았거나 활성 상태가 아닙니다.", "الناشر غير مسجّل أو غير نشط."],
  publisher_mismatch: ["The version names a different publisher than its key.", "버전에 키와 다른 게시자가 지정되어 있습니다.", "يذكر الإصدار ناشرًا مختلفًا عن ناشر مفتاحه."],
  signature_invalid: ["The version's signature is not valid.", "버전의 서명이 올바르지 않습니다.", "توقيع الإصدار غير صالح."],
  incompatible: ["The version does not support this core version.", "버전이 이 코어 버전을 지원하지 않습니다.", "لا يدعم الإصدار هذا الإصدار الأساسي."],
  vulnerable: ["The version is marked vulnerable and cannot run outside a verification workspace.", "버전에 취약 표시가 있어 검증용 업무 공간 밖에서는 실행할 수 없습니다.", "الإصدار موسوم بأنه معرّض للخطر ولا يمكن تشغيله خارج مساحة عمل للتحقق."],
  revoked: ["The publisher or its key is revoked or compromised.", "게시자 또는 키가 회수되었거나 손상되었습니다.", "الناشر أو مفتاحه مُلغى أو مخترق."],
  license_not_permitted: ["The version's license is not on this installation's list of permitted plugin licenses.", "버전의 라이선스가 이 설치의 허용 플러그인 라이선스 목록에 없습니다.", "ترخيص الإصدار ليس ضمن قائمة تراخيص الإضافات المسموح بها في هذا التثبيت."],
  isolated: ["The plugin is isolated.", "플러그인이 격리되어 있습니다.", "الإضافة معزولة."],
  expired: ["The proposal's evidence has expired.", "제안의 증빙이 만료되었습니다.", "انتهت صلاحية أدلة المقترح."],
  not_found: ["Something this proposal names is no longer registered.", "이 제안이 가리키는 항목이 더 이상 등록되어 있지 않습니다.", "شيء يذكره هذا المقترح لم يعد مسجّلًا."],
  conflict: ["Another record already uses this identifier.", "다른 기록이 이미 이 식별자를 사용합니다.", "يستخدم سجل آخر هذا المعرّف بالفعل."],
  limit: ["A registry limit is reached.", "레지스트리 한도에 도달했습니다.", "تم بلوغ حد في السجل."],
};

export const notPendingWords: Words = {
  request_resume: ["The subscription is not paused or suspended.", "구독이 일시 중지·자동 정지 상태가 아닙니다.", "الاشتراك ليس متوقفًا مؤقتًا أو معلّقًا."],
  approve_resume: ["No resume has been requested.", "재개 요청이 없습니다.", "لم يُطلب الاستئناف."],
  revoke: ["The subscription is already revoked.", "구독이 이미 회수되었습니다.", "الاشتراك مُلغى بالفعل."],
};

export const letterRevoked: [string, string, string] = [
  "The subscription is revoked, so this dead letter cannot be redelivered. It can still be dismissed.",
  "구독이 회수되어 이 실패 보관 건은 재발송할 수 없습니다. 정리는 할 수 있습니다.",
  "الاشتراك مُلغى، لذا لا يمكن إعادة تسليم هذه الرسالة غير المُسلّمة. لا يزال بالإمكان إزالتها.",
];

export const unverifiedSentence: [string, string, string] = [
  "Unverified version, destination or permission. It cannot be activated.",
  "검증되지 않은 버전·목적지·권한입니다. 활성화할 수 없습니다.",
  "إصدار أو وجهة أو صلاحية غير موثّقة. لا يمكن تفعيلها.",
];

export const licenseStateWords: Words = {
  permitted: ["Permitted on this installation", "이 설치에서 허용됨", "مسموح به في هذا التثبيت"],
  not_permitted: ["Not permitted on this installation.", "이 설치에서 허용되지 않음.", "غير مسموح به في هذا التثبيت."],
};

export const licenseSourceWords: Words = {
  default: ["installation default", "설치 기본값", "الإعداد الافتراضي للتثبيت"],
  start_flag: ["set when the server started", "서버 시작 시 지정", "حُدّد عند بدء الخادم"],
};

export const allWordTables: Record<string, Words> = {
  sectionWords, connectionWords, clientKindWords, clientStateWords, grantStateWords, scopeWords, serviceUserWords, installWords, installReasonWords, capabilityWords,
  componentWords, deletionWords, migrationWords, proposalKindWords, proposalStateWords, destinationWords, sourceWords, subscriptionWords, subscriptionReasonWords,
  keyStateWords, historyWords, eventStateWords, deliveryKindWords, jobWords, deliveryReasonWords, resultWords, changeWords, summaryWords, blockerWords, notPendingWords, licenseStateWords, licenseSourceWords,
};
