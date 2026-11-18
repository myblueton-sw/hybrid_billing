import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { counted, type Plural } from "../../../../shared/plural";

export const units = {
  days: { en: ["1 day", "{n} days"], ko: "{n}일", ar: ["{n} يوم", "يوم واحد", "يومين", "{n} أيام", "{n} يومًا", "{n} يوم"] },
  hours: { en: ["1 hour", "{n} hours"], ko: "{n}시간", ar: ["{n} ساعة", "ساعة واحدة", "ساعتان", "{n} ساعات", "{n} ساعة", "{n} ساعة"] },
  requests: { en: ["1 request", "{n} requests"], ko: "{n}건", ar: ["{n} طلب", "طلب واحد", "طلبان", "{n} طلبات", "{n} طلبًا", "{n} طلب"] },
  revocations: { en: ["1 revocation", "{n} revocations"], ko: "{n}건", ar: ["{n} عملية إبطال", "عملية إبطال واحدة", "عمليتا إبطال", "{n} عمليات إبطال", "{n} عملية إبطال", "{n} عملية إبطال"] },
  records: { en: ["1 record", "{n} records"], ko: "{n}건", ar: ["{n} قيد", "قيد واحد", "قيدان", "{n} قيود", "{n} قيدًا", "{n} قيد"] },
  approvers: {
    en: ["1 eligible approver", "{n} eligible approvers"],
    ko: "승인 가능한 관리자 {n}명",
    ar: ["{n} معتمِد مؤهَّل", "معتمِد مؤهَّل واحد", "معتمِدان مؤهَّلان", "{n} معتمِدين مؤهَّلين", "{n} معتمِدًا مؤهَّلًا", "{n} معتمِد مؤهَّل"],
  },
  waiting: {
    en: ["1 waiting", "{n} waiting"],
    ko: "대기 {n}건",
    ar: ["{n} طلب بانتظار الاعتماد", "طلب واحد بانتظار الاعتماد", "طلبان بانتظار الاعتماد", "{n} طلبات بانتظار الاعتماد", "{n} طلبًا بانتظار الاعتماد", "{n} طلب بانتظار الاعتماد"],
  },
} satisfies Record<string, Plural>;

export const q = {
  requestTitle: (l: Locale) => t(l, "Request access", "권한 추가 요청", "طلب صلاحية"),
  requestLead: (l: Locale) =>
    t(
      l,
      "Asks for one person to get access within the console ceiling. Nothing changes until another administrator approves this exact request. You can't request access for yourself.",
      "콘솔 권한 상한 안에서 사용자 한 명에게 권한을 주도록 요청합니다. 다른 관리자가 이 요청을 그대로 승인하기 전에는 아무것도 바뀌지 않습니다. 자신에 대한 권한은 요청할 수 없습니다.",
      "يطلب منح شخص واحد صلاحية ضمن الحد الأقصى لوحدة التحكم. لا يتغيّر شيء حتى يعتمد مسؤول آخر هذا الطلب بعينه. لا يمكنك طلب صلاحية لنفسك.",
    ),
  requestImpact: (l: Locale, login: string, access: string, days: number) =>
    t(
      l,
      `If approved, ${login} will be able to ${access} for all customers for ${counted(l, days, units.days)} from the approval.`,
      `승인되면 ${login}님은 승인 시점부터 ${counted(l, days, units.days)} 동안 모든 고객에 대해 ${access} 권한을 갖습니다.`,
      `عند الاعتماد سيتمكن ${login} من ${access} لجميع العملاء مدة ${counted(l, days, units.days)} من وقت الاعتماد.`,
    ),
  requestSent: (l: Locale) => t(l, "Request sent. Another administrator must approve it.", "요청을 보냈습니다. 다른 관리자가 승인해야 합니다.", "أُرسل الطلب. يجب أن يعتمده مسؤول آخر."),
  outcomeUnknown: (l: Locale) =>
    t(l, "Outcome unknown. Check Access change requests below.", "결과를 확인하지 못했습니다. 아래 권한 변경 승인 요청을 확인하세요.", "النتيجة غير معروفة. راجع طلبات تغيير الصلاحيات أدناه."),
  additionsPaused: (l: Locale) =>
    t(
      l,
      "Grant additions are paused while the decision ledger can't record decisions. See the decision ledger line in the Revocations section.",
      "결정 원장이 결정을 기록할 수 없는 동안 권한 추가는 중지됩니다. '회수' 섹션의 결정 원장 상태를 확인하세요.",
      "تتوقف إضافة الصلاحيات ما دام دفتر القرارات غير قادر على تسجيل القرارات. راجع حالة دفتر القرارات في قسم «الإبطال».",
    ),
  requestRevocation: (l: Locale) => t(l, "Request revocation", "회수 요청", "طلب الإبطال"),
  revocationRequested: (l: Locale) =>
    t(l, "Revocation requested. Access stays active until it is approved.", "회수를 요청했습니다. 승인될 때까지 권한은 유지됩니다.", "طُلب الإبطال. تبقى الصلاحية سارية حتى يُعتمد."),
  revokeRequestLead: (l: Locale, login: string) =>
    t(
      l,
      `Asks to remove this access from ${login}. Another administrator must approve it; ${login} may approve it too. Access stays active until then.`,
      `${login}님의 이 권한을 회수하도록 요청합니다. 다른 관리자가 승인해야 하며 ${login}님도 승인할 수 있습니다. 그때까지 권한은 유지됩니다.`,
      `يطلب إزالة هذه الصلاحية من ${login}. يجب أن يعتمده مسؤول آخر، ويمكن أن يعتمده ${login} أيضًا. تبقى الصلاحية سارية حتى ذلك الحين.`,
    ),
  requestsTitle: (l: Locale) => t(l, "Access change requests", "권한 변경 승인 요청", "طلبات تغيير الصلاحيات"),
  requestsLead: (l: Locale) =>
    t(
      l,
      "Each request is fixed when it is made. If access changes before approval, the request goes out of date and can't be applied.",
      "요청은 만들 때의 내용으로 고정됩니다. 승인 전에 권한 상태가 바뀌면 요청이 최신이 아니게 되어 적용할 수 없습니다.",
      "يُثبَّت كل طلب لحظة إنشائه. إذا تغيّرت الصلاحيات قبل الاعتماد يصبح الطلب قديمًا ولا يمكن تطبيقه.",
    ),
  waiting: (l: Locale, n: number) => counted(l, n, units.waiting),
  noRequests: (l: Locale) => t(l, "No access change requests are waiting.", "대기 중인 권한 변경 승인 요청이 없습니다.", "لا توجد طلبات تغيير صلاحيات بانتظار الاعتماد."),
  notKept: (l: Locale, at: string) =>
    t(l, `Access change requests made before the server started at ${at} are not kept.`, `서버가 ${at}에 시작되기 전에 만든 권한 변경 승인 요청은 보관되지 않습니다.`, `لا تُحفظ طلبات تغيير الصلاحيات التي أُنشئت قبل بدء الخادم في ${at}.`),
  endedRequests: (l: Locale, n: number) => t(l, `Ended requests (${n})`, `종료된 요청 (${n})`, `الطلبات المنتهية (${n})`),
  colChange: (l: Locale) => t(l, "Change", "변경", "التغيير"),
  colRequest: (l: Locale) => t(l, "Request", "요청", "الطلب"),
  changeAdd: (l: Locale) => t(l, "Add access", "권한 추가", "إضافة صلاحية"),
  changeRevoke: (l: Locale) => t(l, "Remove access", "권한 회수", "إزالة صلاحية"),
  adds: (l: Locale, what: string) => t(l, `Adds: ${what}`, `추가: ${what}`, `يضيف: ${what}`),
  removes: (l: Locale, what: string) => t(l, `Removes: ${what}`, `회수: ${what}`, `يزيل: ${what}`),
  forDays: (l: Locale, n: number) =>
    t(l, `For ${counted(l, n, units.days)} from approval`, `승인 시점부터 ${counted(l, n, units.days)}`, `لمدة ${counted(l, n, units.days)} من الاعتماد`),
  proposedBy: (l: Locale, who: string, at: string) => t(l, `Requested by ${who} at ${at}`, `${who}님이 ${at}에 요청`, `طلبه ${who} في ${at}`),
  expires: (l: Locale, at: string) => t(l, `Expires ${at}`, `${at} 만료`, `ينتهي في ${at}`),
  decidedBy: (l: Locale, who: string, at: string) => t(l, `By ${who} at ${at}`, `${who}님, ${at}`, `بواسطة ${who} في ${at}`),
  revision: (l: Locale) => t(l, "Request revision", "요청 revision", "بصمة الطلب"),
  approvers: (l: Locale, n: number) => counted(l, n, units.approvers),
  st: {
    pending: (l: Locale) => t(l, "Waiting for approval", "승인 대기", "بانتظار الاعتماد"),
    applied: (l: Locale) => t(l, "Approved", "승인됨", "اعتُمد"),
    rejected: (l: Locale) => t(l, "Rejected", "반려됨", "رُفض"),
    withdrawn: (l: Locale) => t(l, "Withdrawn", "철회됨", "سُحب"),
    expired: (l: Locale) => t(l, "Expired", "만료됨", "انتهت مدته"),
    superseded: (l: Locale) => t(l, "Out of date", "최신 아님", "قديم"),
  },

  noApprover: (l: Locale) =>
    t(
      l,
      "No eligible approver: no other active administrator who holds installation administration can approve it.",
      "승인 가능한 관리자가 없습니다. 설치 관리 권한을 가진 다른 활성 관리자가 없습니다.",
      "لا يوجد معتمِد مؤهَّل: لا يوجد مسؤول نشط آخر يملك صلاحية إدارة التثبيت يمكنه اعتماده.",
    ),
  outOfDate: (l: Locale) =>
    t(
      l,
      "Access changed after this request was made. It is not applied as reviewed; reject or withdraw it and request again.",
      "권한 revision이 바뀌었습니다. 이전 미리보기로 적용하지 않고 다시 평가하세요. 반려하거나 철회한 뒤 다시 요청하세요.",
      "تغيّرت الصلاحيات بعد إنشاء هذا الطلب، فلا يُطبَّق كما رُوجع. ارفضه أو اسحبه ثم اطلب من جديد.",
    ),
  approve: (l: Locale) => t(l, "Approve", "승인", "اعتماد"),
  reject: (l: Locale) => t(l, "Reject", "반려", "رفض"),
  withdraw: (l: Locale) => t(l, "Withdraw", "철회", "سحب"),
  approveTitle: (l: Locale) => t(l, "Approve request", "요청 승인", "اعتماد الطلب"),
  rejectTitle: (l: Locale) => t(l, "Reject request", "요청 반려", "رفض الطلب"),
  withdrawTitle: (l: Locale) => t(l, "Withdraw request", "요청 철회", "سحب الطلب"),
  approveLead: (l: Locale) =>
    t(
      l,
      "You approve exactly the content and revision shown here. If anything changed since the request was made, it is not applied.",
      "여기 표시된 내용과 revision 그대로 승인합니다. 요청 이후 무엇이든 바뀌었다면 적용되지 않습니다.",
      "أنت تعتمد المحتوى وبصمة الطلب المعروضين هنا بالضبط. إذا تغيّر أي شيء منذ إنشاء الطلب فلن يُطبَّق.",
    ),
  approveRevokeLead: (l: Locale) =>
    t(
      l,
      "Approving blocks the access at once. The server then tries to record the revocation in the decision ledger; it is not confirmed until the ledger has recorded it. Its state appears in the Revocations section.",
      "승인하면 즉시 접근이 차단됩니다. 이어서 서버가 회수 결정을 결정 원장에 기록하려고 시도하며, 원장에 기록되기 전까지는 확정되지 않습니다. 상태는 '회수' 섹션에 표시됩니다.",
      "يحجب الاعتماد الوصول فورًا، ثم يحاول الخادم تسجيل الإبطال في دفتر القرارات، ولا يُعدّ مؤكدًا حتى يسجّله الدفتر. تظهر حالته في قسم «الإبطال».",
    ),
  rejectLead: (l: Locale) => t(l, "The request ends without any change. Give the reason the requester will see.", "변경 없이 요청이 종료됩니다. 요청자가 볼 사유를 입력하세요.", "ينتهي الطلب دون أي تغيير. اذكر السبب الذي سيراه مقدّم الطلب."),
  withdrawLead: (l: Locale) => t(l, "Your request ends without any change.", "내 요청이 변경 없이 종료됩니다.", "ينتهي طلبك دون أي تغيير."),
  mayEnd: (l: Locale) => t(l, "May end with this revocation", "이 회수로 함께 끝날 수 있는 위임", "تفويضات قد تنتهي مع هذا الإبطال"),
  mayEndNone: (l: Locale) => t(l, "No active delegation depends on this person.", "이 사용자와 관련된 유효한 위임이 없습니다.", "لا يوجد تفويض ساري يعتمد على هذا الشخص."),
  doneApprovedAdd: (l: Locale, login: string) => t(l, `Approved. ${login} now has this access.`, `승인했습니다. ${login}님에게 이 권한이 적용됩니다.`, `اعتُمد. أصبحت لدى ${login} هذه الصلاحية.`),
  doneApprovedRevoke: (l: Locale) =>
    t(l, "Approved. The revocation was received; its state is shown in the Revocations section.", "승인했습니다. 회수가 접수되었으며 상태는 '회수' 섹션에 표시됩니다.", "اعتُمد. استُلم الإبطال وتظهر حالته في قسم «الإبطال»."),
  doneRejected: (l: Locale) => t(l, "Request rejected.", "요청을 반려했습니다.", "رُفض الطلب."),
  doneWithdrawn: (l: Locale) => t(l, "Request withdrawn.", "요청을 철회했습니다.", "سُحب الطلب."),
  doneRevokedDeleg: (l: Locale) =>
    t(l, "Delegation revocation received; its state is shown in the Revocations section.", "위임 회수를 접수했습니다. 상태는 '회수' 섹션에 표시됩니다.", "استُلم إبطال التفويض وتظهر حالته في قسم «الإبطال»."),
  descendants: (l: Locale) => t(l, "Delegations that end with it", "함께 끝나는 하위 위임", "تفويضات تنتهي معه"),
  noDescendants: (l: Locale) => t(l, "No delegation was made from it.", "이 위임에서 이어진 위임이 없습니다.", "لم يصدر عنه أي تفويض."),
  refusedRequest: (l: Locale) =>
    t(
      l,
      "This step was refused: you may be the requester or the person concerned, you may not hold this access yourself, or a pending-request limit was reached. The list has been reloaded.",
      "이 단계가 거부되었습니다. 요청자이거나 대상자이거나, 이 권한을 직접 갖고 있지 않거나, 대기 요청 한도에 도달했을 수 있습니다. 목록을 다시 불러왔습니다.",
      "رُفضت هذه الخطوة: ربما كنت مقدّم الطلب أو الشخص المعني، أو لا تملك هذه الصلاحية بنفسك، أو بلغ عدد الطلبات بانتظار الاعتماد حدّه. أُعيد تحميل القائمة.",
    ),
  stageRequest: (l: Locale) =>
    t(
      l,
      "The request is no longer waiting, is out of date, or another request or an active grant already covers it. The list has been reloaded.",
      "요청이 더 이상 대기 중이 아니거나 최신이 아니거나, 다른 요청이나 유효한 권한이 이미 있습니다. 목록을 다시 불러왔습니다.",
      "لم يعد الطلب بانتظار الاعتماد أو أصبح قديمًا، أو يغطيه طلب آخر أو صلاحية سارية. أُعيد تحميل القائمة.",
    ),
  busyIncrease: (l: Locale) =>
    t(
      l,
      "Grant additions are paused because the decision ledger can't record decisions, or the access service is not responding. Revocations still apply.",
      "결정 원장이 결정을 기록할 수 없거나 권한 서비스가 응답하지 않아 권한 추가가 중지되었습니다. 회수는 계속 적용됩니다.",
      "توقفت إضافة الصلاحيات لأن دفتر القرارات لا يستطيع تسجيل القرارات أو لأن خدمة الصلاحيات لا تستجيب. يستمر تطبيق عمليات الإبطال.",
    ),
};

export const v = {
  title: (l: Locale) => t(l, "Revocations", "회수", "الإبطال"),
  lead: (l: Locale) =>
    t(
      l,
      "A revocation blocks access first. It counts as confirmed only after the decision ledger has durably recorded it; cleanup follows.",
      "회수는 먼저 접근을 차단합니다. 결정 원장에 내구 기록된 뒤에만 확정으로 표시하며 그 다음에 후속 정리를 합니다.",
      "يحجب الإبطال الوصول أولًا، ولا يُعدّ مؤكدًا إلا بعد أن يسجّله دفتر القرارات تسجيلًا دائمًا، ثم يتبعه التنظيف.",
    ),
  none: (l: Locale) => t(l, "No revocations yet.", "아직 회수가 없습니다.", "لا توجد عمليات إبطال بعد."),
  name: {
    received: (l: Locale) => t(l, "Received", "회수 접수", "مُستلَم"),
    deny_confirmed: (l: Locale) => t(l, "Block confirmed", "보안 차단 확정", "تأكّد الحجب"),
    cleanup_running: (l: Locale) => t(l, "Cleanup running", "후속 정리 중", "التنظيف جارٍ"),
    revocation_complete: (l: Locale) => t(l, "Revocation complete", "회수 완료", "اكتمل الإبطال"),
    decision_unconfirmed: (l: Locale) => t(l, "Decision not confirmed", "결정 기록 미확정", "القرار غير مؤكَّد"),
  },
  copy: {
    received: (l: Locale) =>
      t(l, "The revocation request was received. The security block is not confirmed yet.", "회수 요청을 접수했습니다. 보안 차단 확정은 아직 확인되지 않았습니다.", "استُلم طلب الإبطال. لم يتأكد الحجب الأمني بعد."),
    deny_confirmed: (l: Locale) =>
      t(l, "The revocation decision and the access block are confirmed. Check the cleanup state.", "회수 결정과 접근 차단이 확정되었습니다. 후속 자격 정리 상태를 확인하세요.", "تأكّد قرار الإبطال وحجب الوصول. راجع حالة التنظيف."),
    cleanup_running: (l: Locale) =>
      t(l, "The access block stays in effect while sessions, keys, caches and jobs are cleaned up.", "접근 차단 효력은 유지되며 세션·키·캐시·작업을 정리 중입니다.", "يبقى حجب الوصول ساريًا أثناء تنظيف الجلسات والمفاتيح وذاكرات التخزين المؤقت والمهام."),
    revocation_complete: (l: Locale) =>
      t(l, "The block and the cleanup of the designated scope are complete.", "회수 차단과 지정 범위의 후속 정리가 완료되었습니다.", "اكتمل الحجب وتنظيف النطاق المحدد."),
    decision_unconfirmed: (l: Locale) =>
      t(
        l,
        "The independent decision record could not be confirmed. This is not shown as a successful revocation.",
        "독립 결정 기록 확정을 확인하지 못했습니다. 회수 성공으로 표시하지 않습니다.",
        "تعذّر تأكيد قيد القرار المستقل. لا يُعرض هذا على أنه إبطال ناجح.",
      ),
  },
  thisProcess: (l: Locale) =>
    t(
      l,
      "Access is blocked in this server process only. The decision is not confirmed as recorded, so it may not be re-applied after a restart.",
      "접근은 이 서버 프로세스 안에서만 차단되었습니다. 결정이 기록되었는지 확인되지 않아 재시작 후 다시 적용되지 않을 수 있습니다.",
      "حُجب الوصول في عملية الخادم هذه فقط. لم يتأكد تسجيل القرار، لذا قد لا يُعاد تطبيقه بعد إعادة التشغيل.",
    ),
  job: {
    running: (l: Locale) => t(l, "Job: running", "작업: 진행 중", "المهمة: قيد التشغيل"),
    waiting: (l: Locale) => t(l, "Job: waiting (decision not confirmed)", "작업: 대기 (결정 기록 미확정)", "المهمة: في الانتظار (القرار غير مؤكَّد)"),
    succeeded: (l: Locale) => t(l, "Job: succeeded", "작업: 성공", "المهمة: نجحت"),
  },
  earlierRun: (l: Locale) => t(l, "From an earlier server run", "이전 서버 실행에서 기록됨", "من تشغيل سابق للخادم"),
  replay: {
    reapplied: (l: Locale) => t(l, "Re-applied at start", "시작 시 다시 적용됨", "أُعيد تطبيقه عند البدء"),
    absent: (l: Locale) => t(l, "Target not present after restart", "재시작 후 대상이 없음", "الهدف غير موجود بعد إعادة التشغيل"),
    protected: (l: Locale) => t(l, "Not re-applied: protected grant", "다시 적용하지 않음: 보호된 권한", "لم يُعَد تطبيقه: صلاحية محمية"),
  },
  grantOf: (l: Locale, who: string, access: string) => t(l, `Grant: ${access} for ${who}`, `권한: ${who}님의 ${access}`, `صلاحية ${access} الممنوحة إلى ${who}`),
  grantGone: (l: Locale, id: string) => t(l, `Grant ${id}`, `권한 ${id}`, `الصلاحية ${id}`),
  delegationOf: (l: Locale, id: string) => t(l, `Delegation ${id}`, `위임 ${id}`, `التفويض ${id}`),
  recordedAt: (l: Locale, at: string) => t(l, `Recorded ${at}`, `${at} 기록`, `سُجّل في ${at}`),
  receivedAt: (l: Locale, at: string) => t(l, `Received ${at}`, `${at} 접수`, `استُلم في ${at}`),
  endedDelegs: (l: Locale) => t(l, "Delegations ended with it", "함께 끝난 위임", "التفويضات التي انتهت معه"),
  group: {
    done: (l: Locale) => t(l, "Done", "완료", "تم"),
    not_applicable: (l: Locale) => t(l, "Not applicable", "해당 없음", "لا ينطبق"),
    outside_scope: (l: Locale) => t(l, "Outside the controlled scope (not counted as cleaned)", "통제 범위 밖 (정리 완료로 세지 않음)", "خارج النطاق المتحكَّم فيه (لا يُعدّ منظَّفًا)"),
    not_started: (l: Locale) => t(l, "Not started: waits for a confirmed decision", "시작 안 함: 확정된 결정을 기다림", "لم يبدأ: بانتظار قرار مؤكَّد"),
  },
  progress: (l: Locale, done: number, total: number) => t(l, `${done} of ${total} cleanup items settled`, `정리 항목 ${total}개 중 ${done}개 처리`, `سُوّي ${done} من أصل ${total} من بنود التنظيف`),
  details: (l: Locale) => t(l, "Technical details", "전문가 상세", "التفاصيل التقنية"),
  sequence: (l: Locale) => t(l, "Ledger sequence", "원장 순번", "تسلسل الدفتر"),
  watermark: (l: Locale) => t(l, "Confirmed watermark", "결정 워터마크", "علامة التأكيد"),
  denyRevision: (l: Locale) => t(l, "Deny revision", "deny revision", "رقم إصدار الحجب"),
  failure: (l: Locale) => t(l, "Ledger result", "원장 결과", "نتيجة الدفتر"),
  request: (l: Locale) => t(l, "Access change request", "권한 변경 승인 요청", "طلب تغيير الصلاحيات"),
  revocationId: (l: Locale) => t(l, "Revocation ID", "회수 ID", "معرّف الإبطال"),
  reason: {
    commit_failed: (l: Locale) => t(l, "The write to the ledger failed", "원장 쓰기 실패", "فشلت الكتابة في الدفتر"),
    commit_unknown: (l: Locale) => t(l, "Unknown whether the record reached the disk", "기록이 디스크에 남았는지 알 수 없음", "غير معروف إن كان القيد قد وصل إلى القرص"),
    ledger_isolated: (l: Locale) => t(l, "The ledger is isolated", "원장이 격리됨", "الدفتر معزول"),
    ledger_stopped: (l: Locale) => t(l, "The ledger stopped after a failure", "실패 후 원장이 멈춤", "توقف الدفتر بعد إخفاق"),
    ledger_full: (l: Locale) => t(l, "The ledger reached its size bound", "원장이 크기 상한에 도달함", "بلغ الدفتر حدّ حجمه"),
  },
  item: {
    C1: (l: Locale) => t(l, "Decision stamps", "결정 stamp", "أختام القرارات"),
    C2: (l: Locale) => t(l, "Console sessions", "콘솔 세션", "جلسات وحدة التحكم"),
    C3: (l: Locale) => t(l, "Service credentials and API keys", "서비스 자격 증명·API 키", "بيانات اعتماد الخدمات ومفاتيح API"),
    C4: (l: Locale) => t(l, "HTTP caches of reader responses", "조회 응답 HTTP 캐시", "ذاكرات HTTP المؤقتة لاستجابات القراءة"),
    C5: (l: Locale) => t(l, "Copies already shown or downloaded", "이미 표시되거나 내려받은 사본", "نسخ عُرضت أو نُزّلت مسبقًا"),
    C6: (l: Locale) => t(l, "Reads already in progress", "이미 진행 중인 조회", "عمليات قراءة جارية"),
    C7: (l: Locale) => t(l, "Jobs started under this access", "이 권한으로 시작된 작업", "مهام بدأت بهذه الصلاحية"),
    C8: (l: Locale) => t(l, "Search indexes, projections and exports", "검색 색인·프로젝션·내보내기", "فهارس البحث والإسقاطات والتصديرات"),
    C9: (l: Locale) => t(l, "Delegations that depend on it", "이 권한에 의존하는 위임", "التفويضات المعتمدة عليه"),
  },
  why: {
    deny_generation: (l: Locale) => t(l, "Earlier decisions no longer pass a recheck", "이전 결정은 재확인을 통과하지 못함", "لم تعد القرارات السابقة تجتاز إعادة التحقق"),
    no_permission_in_sessions: (l: Locale) => t(l, "Sessions hold no permission; every read decides again", "세션에 권한이 없고 매 조회 시 다시 판단함", "لا تحمل الجلسات صلاحيات؛ تُقرَّر كل قراءة من جديد"),
    human_grants_only: (l: Locale) => t(l, "Grants here go to people only", "여기서는 사람에게만 권한을 부여함", "تُمنح الصلاحيات هنا للأشخاص فقط"),
    no_store: (l: Locale) => t(l, "Reader responses are never cached", "조회 응답은 캐시되지 않음", "لا تُخزَّن استجابات القراءة مؤقتًا أبدًا"),
    cannot_recall: (l: Locale) => t(l, "Cannot be recalled", "회수할 수 없음", "لا يمكن استرجاعها"),
    not_cancelled: (l: Locale) => t(l, "A read admitted before the block finishes", "차단 전에 허용된 조회는 끝까지 진행됨", "تكتمل القراءة المقبولة قبل الحجب"),
    no_job_runner: (l: Locale) => t(l, "No job runner is connected", "작업 실행기가 연결되지 않음", "لا يوجد مشغّل مهام متصل"),
    nothing_keyed_to_grants: (l: Locale) => t(l, "Nothing is kept per grant", "권한별로 보관하는 것이 없음", "لا يُحفظ شيء لكل صلاحية"),
    chain_rules: (l: Locale) => t(l, "Ended by the delegation chain rules", "위임 사슬 규칙으로 종료됨", "أنهتها قواعد سلسلة التفويض"),
    no_dependent_delegation: (l: Locale) => t(l, "No delegation depended on it", "의존하는 위임이 없음", "لا يوجد تفويض معتمد عليه"),
    server_restarted: (l: Locale) => t(l, "Ended by the server restart", "서버 재시작으로 종료됨", "أنهتها إعادة تشغيل الخادم"),
  },
  ledgerOk: (l: Locale, n: number, seq: string) =>
    t(
      l,
      `Decision ledger: ${counted(l, n, units.records)}, last record number ${seq}.`,
      `결정 원장: 기록 ${counted(l, n, units.records)}, 마지막 기록 번호 ${seq}.`,
      `دفتر القرارات: ${counted(l, n, units.records)}، رقم آخر قيد ${seq}.`,
    ),
  ledgerChain: (l: Locale) =>
    t(
      l,
      "Each record is chained to the one before it, which detects damage to the file. It does not stop someone with access to the server from replacing the file.",
      "각 기록은 앞 기록과 연결되어 파일 손상을 감지합니다. 서버에 접근할 수 있는 사람이 파일을 바꾸는 것을 막지는 못합니다.",
      "يرتبط كل قيد بالقيد الذي قبله، ما يكشف تلف الملف. لكنه لا يمنع من يملك الوصول إلى الخادم من استبدال الملف.",
    ),
  ledgerRecovered: (l: Locale, at: string, n: number) =>
    t(
      l,
      `At start (${at}) an unfinished last line of ${n} bytes was removed. It had never been confirmed.`,
      `시작 시(${at}) 끝나지 않은 마지막 줄 ${n}바이트를 제거했습니다. 확정된 적 없는 기록입니다.`,
      `عند البدء (${at}) أُزيل سطر أخير غير مكتمل طوله ${n} بايت. لم يُؤكَّد قط.`,
    ),
  ledgerIsolated: {
    missing: (l: Locale) =>
      t(
        l,
        "Grant additions are paused: the decision ledger is missing. Revocations made now are applied in this server process only and will not be re-applied after a restart. An operator must restore the ledger or replace it with a new one.",
        "권한 추가가 중지되었습니다. 결정 원장이 없습니다. 지금 하는 회수는 이 서버 프로세스에서만 적용되며 재시작 후 다시 적용되지 않습니다. 운영자가 원장을 복구하거나 새 원장으로 교체해야 합니다.",
        "توقفت إضافة الصلاحيات: دفتر القرارات مفقود. تُطبَّق عمليات الإبطال التي تُجرى الآن في عملية الخادم هذه فقط ولن يُعاد تطبيقها بعد إعادة التشغيل. يجب على المشغّل استعادة الدفتر أو استبداله بدفتر جديد.",
      ),
    integrity_failed: (l: Locale, line: number) =>
      t(
        l,
        `Grant additions are paused: the decision ledger failed its integrity check at line ${line}. Revocations recorded before line ${line} were re-applied at start; those from line ${line} on were not. Revocations made now are applied in this server process only and will not be re-applied after a restart. An operator must restore the ledger or replace it with a new one.`,
        `권한 추가가 중지되었습니다. 결정 원장이 ${line}번째 줄에서 무결성 검사에 실패했습니다. ${line}번째 줄 이전에 기록된 회수는 시작 시 다시 적용했고, ${line}번째 줄부터의 회수는 다시 적용하지 않았습니다. 지금 하는 회수는 이 서버 프로세스에서만 적용되며 재시작 후 다시 적용되지 않습니다. 운영자가 원장을 복구하거나 새 원장으로 교체해야 합니다.`,
        `توقفت إضافة الصلاحيات: فشل دفتر القرارات في فحص السلامة عند السطر ${line}. أُعيد عند البدء تطبيق عمليات الإبطال المسجلة قبل السطر ${line}، ولم تُطبَّق تلك المسجلة من السطر ${line} فصاعدًا. تُطبَّق عمليات الإبطال التي تُجرى الآن في عملية الخادم هذه فقط ولن يُعاد تطبيقها بعد إعادة التشغيل. يجب على المشغّل استعادة الدفتر أو استبداله بدفتر جديد.`,
      ),
    id_mismatch: (l: Locale) =>
      t(
        l,
        "Grant additions are paused: the decision ledger is not the one this installation created. Nothing from it was re-applied. Revocations made now are applied in this server process only and will not be re-applied after a restart. An operator must restore the ledger or replace it with a new one.",
        "권한 추가가 중지되었습니다. 결정 원장이 이 설치에서 만든 원장이 아닙니다. 이 원장의 결정은 다시 적용하지 않았습니다. 지금 하는 회수는 이 서버 프로세스에서만 적용되며 재시작 후 다시 적용되지 않습니다. 운영자가 원장을 복구하거나 새 원장으로 교체해야 합니다.",
        "توقفت إضافة الصلاحيات: دفتر القرارات ليس الذي أنشأه هذا التثبيت. لم يُعَد تطبيق أي شيء منه. تُطبَّق عمليات الإبطال التي تُجرى الآن في عملية الخادم هذه فقط ولن يُعاد تطبيقها بعد إعادة التشغيل. يجب على المشغّل استعادة الدفتر أو استبداله بدفتر جديد.",
      ),
  },
  ledgerStopped: (l: Locale) =>
    t(
      l,
      "Grant additions are paused: a write to the decision ledger failed in this server process. Revocations made now are applied in this server process only and will not be re-applied after a restart.",
      "권한 추가가 중지되었습니다. 이 서버 프로세스에서 결정 원장 쓰기가 실패했습니다. 지금 하는 회수는 이 서버 프로세스에서만 적용되며 재시작 후 다시 적용되지 않습니다.",
      "توقفت إضافة الصلاحيات: فشلت كتابة في دفتر القرارات في عملية الخادم هذه. تُطبَّق عمليات الإبطال التي تُجرى الآن في عملية الخادم هذه فقط ولن يُعاد تطبيقها بعد إعادة التشغيل.",
    ),
  ledgerFull: (l: Locale) =>
    t(
      l,
      "Grant additions are paused: the decision ledger reached its size bound. Revocations made now are applied in this server process only and will not be re-applied after a restart. An operator can raise the bound when restarting the server.",
      "권한 추가가 중지되었습니다. 결정 원장이 크기 상한에 도달했습니다. 지금 하는 회수는 이 서버 프로세스에서만 적용되며 재시작 후 다시 적용되지 않습니다. 운영자가 서버를 재시작할 때 상한을 올릴 수 있습니다.",
      "توقفت إضافة الصلاحيات: بلغ دفتر القرارات حدّ حجمه. تُطبَّق عمليات الإبطال التي تُجرى الآن في عملية الخادم هذه فقط ولن يُعاد تطبيقها بعد إعادة التشغيل. يمكن للمشغّل رفع الحد عند إعادة تشغيل الخادم.",
    ),
  ledgerUnknown: (l: Locale) =>
    t(l, "The decision ledger state was not reported. Grant additions may be refused.", "결정 원장 상태가 보고되지 않았습니다. 권한 추가가 거부될 수 있습니다.", "لم تُبلَّغ حالة دفتر القرارات. قد تُرفض إضافة الصلاحيات."),
  stillAvailable: (l: Locale) =>
    t(
      l,
      "Still available: requesting and approving revocations, revoking delegations, rejecting and withdrawing requests, and suspending a person in Users and access.",
      "계속 사용할 수 있는 작업: 회수 요청·승인, 위임 회수, 요청 반려·철회, '사용자·접근'에서 사용자 정지.",
      "ما زال متاحًا: طلب عمليات الإبطال واعتمادها، وإبطال التفويضات، ورفض الطلبات وسحبها، وتعليق شخص من «المستخدمون والوصول».",
    ),
  ledgerStoppedStartSync: (l: Locale) =>
    t(
      l,
      "Grant additions are paused: at start the server could not confirm that the decision ledger file is fully written to disk. Nothing new was written to it. Revocations made now are applied in this server process only and will not be re-applied after a restart.",
      "권한 추가가 중지되었습니다. 시작 시 서버가 결정 원장 파일이 디스크에 완전히 기록되었는지 확인하지 못했습니다. 파일에 새로 쓴 내용은 없습니다. 지금 하는 회수는 이 서버 프로세스에서만 적용되며 재시작 후 다시 적용되지 않습니다.",
      "توقفت إضافة الصلاحيات: لم يتمكن الخادم عند البدء من التأكد من أن ملف دفتر القرارات مكتوب بالكامل على القرص. لم يُكتب فيه أي شيء جديد. تُطبَّق عمليات الإبطال التي تُجرى الآن في عملية الخادم هذه فقط ولن يُعاد تطبيقها بعد إعادة التشغيل.",
    ),
  ledgerNotDurable: {
    stopped: (l: Locale) =>
      t(
        l,
        "Any revocations replayed at start came from records not confirmed durable. They stay applied. A restart checks the file again.",
        "시작 시 다시 적용한 회수가 있다면 확정 내구 기록으로 확인되지 않은 기록에서 적용한 것입니다. 회수는 계속 적용됩니다. 재시작하면 파일을 다시 확인합니다.",
        "أي عمليات إبطال أُعيد تطبيقها عند البدء جاءت من قيود لم يتأكد أنها دائمة، وتبقى مطبَّقة. تتحقق إعادة التشغيل من الملف مجددًا.",
      ),
    isolated: (l: Locale) =>
      t(
        l,
        "Any revocations replayed at start, those recorded before the damaged line, came from records not confirmed durable. They stay applied. An operator must restore the ledger or replace it with a new one.",
        "시작 시 다시 적용한 회수(손상된 줄 이전에 기록된 회수)가 있다면 확정 내구 기록으로 확인되지 않은 기록에서 적용한 것입니다. 회수는 계속 적용됩니다. 운영자가 원장을 복구하거나 새 원장으로 교체해야 합니다.",
        "أي عمليات إبطال أُعيد تطبيقها عند البدء، وهي المسجلة قبل السطر التالف، جاءت من قيود لم يتأكد أنها دائمة، وتبقى مطبَّقة. يجب على المشغّل استعادة الدفتر أو استبداله بدفتر جديد.",
      ),
  },
  ledgerReplaced: (l: Locale, from: string) =>
    t(
      l,
      `This ledger replaced an earlier one (${from}). Decisions recorded in the replaced ledger are no longer re-applied, so any sample access revoked under it is active again. The old file is kept for investigation.`,
      `이 원장은 이전 원장(${from})을 대체했습니다. 대체된 원장에 기록된 결정은 더 이상 다시 적용되지 않으므로, 그 원장에서 회수한 샘플 권한이 있다면 다시 유효합니다. 이전 파일은 조사용으로 보관됩니다.`,
      `حلّ هذا الدفتر محل دفتر سابق (${from}). لم تعد القرارات المسجلة في الدفتر المستبدَل يُعاد تطبيقها، لذا عادت سارية أي صلاحيات عيّنة أُبطلت بموجبه. يُحتفظ بالملف القديم للتحقيق.`,
    ),
  ledgerId: (l: Locale) => t(l, "Ledger ID", "원장 ID", "معرّف الدفتر"),
  createdAt: (l: Locale) => t(l, "Created", "생성", "أُنشئ"),
  checkedAt: (l: Locale) => t(l, "Checked", "검사", "فُحص"),
  file: (l: Locale) => t(l, "File", "파일", "الملف"),
  unknownLedger: (l: Locale) => t(l, "unknown", "알 수 없음", "غير معروف"),
  previewUnconfirmed: (l: Locale) =>
    t(
      l,
      "A revocation for this person is not confirmed. This result reflects this server process only.",
      "이 사용자에 대한 회수가 확정되지 않았습니다. 이 결과는 이 서버 프로세스만 반영합니다.",
      "إبطال يخص هذا الشخص غير مؤكَّد. تعكس هذه النتيجة عملية الخادم هذه فقط.",
    ),
  endedByChain: (l: Locale) => t(l, "Ended: the access behind it no longer applies", "종료됨: 이 위임을 뒷받침하던 권한이 더 이상 적용되지 않음", "انتهى: لم تعد الصلاحية التي يستند إليها سارية"),
};

export const s = {
  title: (l: Locale) => t(l, "Approval settings", "승인 설정", "إعدادات الاعتماد"),
  lead: (l: Locale) =>
    t(
      l,
      "These are installation settings. A value is changed by restarting the server with a different value; it can't be changed here.",
      "설치 설정입니다. 값은 다른 값으로 서버를 다시 시작해야 바뀌며 여기에서 바꿀 수 없습니다.",
      "هذه إعدادات التثبيت. تُغيَّر القيمة بإعادة تشغيل الخادم بقيمة مختلفة، ولا يمكن تغييرها هنا.",
    ),
  colSetting: (l: Locale) => t(l, "Setting", "설정", "الإعداد"),
  colValue: (l: Locale) => t(l, "Value", "값", "القيمة"),
  colRange: (l: Locale) => t(l, "Allowed range", "허용 범위", "النطاق المسموح"),
  colSource: (l: Locale) => t(l, "Source", "출처", "المصدر"),
  range: (l: Locale, min: number, max: number, def: number) =>
    t(l, `${min}–${max}, default ${def}`, `${min}~${max}, 기본값 ${def}`, `من ${min} إلى ${max}، الافتراضي ${def}`),
  source: {
    default: (l: Locale) => t(l, "Installation default", "설치 기본값", "القيمة الافتراضية للتثبيت"),
    start_flag: (l: Locale) => t(l, "Set when the server started", "서버 시작 시 지정", "حُدّد عند بدء الخادم"),
  },
  name: {
    access_request_hours: (l: Locale) => t(l, "Request lifetime", "요청 유효 시간", "مدة سريان الطلب"),
    access_pending_per_proposer: (l: Locale) => t(l, "Waiting requests per requester", "요청자별 대기 요청 수", "الطلبات بانتظار الاعتماد لكل مقدّم طلب"),
    access_pending_per_workspace: (l: Locale) => t(l, "Waiting requests per workspace", "업무 공간별 대기 요청 수", "الطلبات بانتظار الاعتماد لكل مساحة عمل"),
    access_ended_kept: (l: Locale) => t(l, "Ended requests kept", "보관하는 종료 요청 수", "الطلبات المنتهية المحفوظة"),
    access_revocations_kept: (l: Locale) => t(l, "Revocations kept and shown", "보관·표시하는 회수 수", "عمليات الإبطال المحفوظة والمعروضة"),
    access_ended_shown: (l: Locale) => t(l, "Ended requests shown", "표시하는 종료 요청 수", "الطلبات المنتهية المعروضة"),
    decision_ledger_max_mib: (l: Locale) => t(l, "Decision ledger size bound", "결정 원장 크기 상한", "حدّ حجم دفتر القرارات"),
  },
  unit: {
    hours: (l: Locale, n: number) => counted(l, n, units.hours),
    requests: (l: Locale, n: number) => counted(l, n, units.requests),
    revocations: (l: Locale, n: number) => counted(l, n, units.revocations),
    MiB: (l: Locale, n: number) => t(l, `${n} MiB`, `${n} MiB`, `${n} MiB`),
  },
};
