import type { Locale } from "../../../../shared/types";
import type { PendingKind, RightId } from "../../../../service/integrations";
import { rightIds } from "../../../../service/integrations";
import { text as t } from "../../../../shared/copy";
import { iso } from "./live-copy";

const labels: Record<RightId, [string, string, string]> = {
  administer_api_clients: ["API client administration", "API client 관리", "إدارة عملاء API"],
  administer_modules: ["Module administration", "모듈 관리", "إدارة الوحدات"],
  approve_modules: ["Module approval", "모듈 승인", "اعتماد الوحدات"],
  stop_modules: ["Module stop", "모듈 중지", "إيقاف الوحدات"],
  administer_channels: ["Subscription and channel administration", "구독·채널 관리", "إدارة الاشتراكات والقنوات"],
  approve_channels: ["Subscription and channel approval", "구독·채널 승인", "اعتماد الاشتراكات والقنوات"],
  stop_channels: ["Subscription and channel stop", "구독·채널 중지", "إيقاف الاشتراكات والقنوات"],
  reprocess_events: ["Event reprocessing", "이벤트 재처리", "إعادة معالجة الأحداث"],
};

const pendingWords: Record<PendingKind, [string, string, string]> = {
  plugin_proposal: ["Plugin change proposal", "플러그인 변경 제안", "مقترح تغيير إضافة"],
  subscription_proposal: ["Subscription proposal", "구독 제안", "مقترح اشتراك"],
  resume_request: ["Resume request", "재개 요청", "طلب استئناف"],
  redeliver_request: ["Redelivery request", "재발송 요청", "طلب إعادة تسليم"],
  dismiss_request: ["Dismissal request", "정리 요청", "طلب إزالة"],
};

export function rightLabel(l: Locale, id: RightId): string {
  const w = labels[id];
  return t(l, w[0], w[1], w[2]);
}

export function inSentence(l: Locale, label: string): string {
  return l === "en" && !label.startsWith("API") ? label.charAt(0).toLowerCase() + label.slice(1) : label;
}

function rightInSentence(l: Locale, id: RightId): string {
  return inSentence(l, rightLabel(l, id));
}

export function rightList(l: Locale, held: RightId[]): string {
  const names = rightIds.filter((id) => held.includes(id)).map((id) => rightInSentence(l, id));
  return names.join(l === "en" ? "; " : l === "ar" ? "، " : ", ");
}

function area(l: Locale, view: string): string {
  return view === "modules" ? t(l, "Modules", "모듈", "الوحدات")
    : view === "channels" ? t(l, "Subscriptions and channels", "구독·채널", "الاشتراكات والقنوات")
      : view === "events" ? t(l, "Events and redelivery", "이벤트·재처리", "الأحداث وإعادة التسليم")
        : t(l, "API clients", "API client", "عملاء API");
}

export const r = {
  rightsLead: (l: Locale) => t(l, "Your integration rights:", "내 연동 권한:", "صلاحيات التكامل لديك:"),
  rightsNoneLead: (l: Locale) => t(l, "You hold no integration rights.", "연동 권한이 없습니다.", "لا تملك أي صلاحيات تكامل."),
  rightsNoneRest: (l: Locale) =>
    t(
      l,
      "Integration rights are given only when the installation is set up; they can't be granted here yet.",
      "연동 권한은 설치를 설정할 때만 부여되며 아직 여기에서 부여할 수 없습니다.",
      "تُمنح صلاحيات التكامل عند إعداد التثبيت فقط، ولا يمكن منحها هنا بعد.",
    ),
  rightsUnknown: (l: Locale) =>
    t(
      l,
      "Your integration rights could not be read, so no integration action is offered here. Reload to try again.",
      "연동 권한을 읽지 못해 이 화면에서는 연동 작업을 제공하지 않습니다. 다시 시도하려면 새로 고침하세요.",
      "تعذّرت قراءة صلاحيات التكامل لديك، لذا لا يُعرض هنا أي إجراء تكامل. أعد التحميل للمحاولة مجددًا.",
    ),
  withheld: (l: Locale, view: string) =>
    t(
      l,
      `You don't hold a right for ${area(l, view)}. Nothing from this area is shown.`,
      `${area(l, view)}에 대한 권한이 없습니다. 이 영역의 내용은 표시하지 않습니다.`,
      `لا تملك صلاحية لـ${area(l, view)}. لا يُعرض شيء من هذه المنطقة.`,
    ),
  noRight: (l: Locale, id: RightId) =>
    t(l, `Needs the ${rightInSentence(l, id)} right. You don't hold it.`, `${rightLabel(l, id)} 권한이 필요합니다. 현재 이 권한이 없습니다.`, `تتطلب صلاحية ${rightLabel(l, id)}، وأنت لا تملكها.`),
  starterLost: (l: Locale) =>
    t(
      l,
      "The person who started this no longer holds the right, so it can't be approved.",
      "시작한 사람에게 더 이상 권한이 없어 승인할 수 없습니다.",
      "لم يعد من بدأ هذا يملك الصلاحية، لذا لا يمكن اعتماده.",
    ),
  noLongerHeld: (l: Locale, id: RightId) =>
    t(
      l,
      `You no longer hold the ${rightInSentence(l, id)} right. Nothing was changed.`,
      `더 이상 ${rightLabel(l, id)} 권한이 없습니다. 아무것도 변경되지 않았습니다.`,
      `لم تعد تملك صلاحية ${rightLabel(l, id)}. لم يتغيّر شيء.`,
    ),
  closedLost: (l: Locale, object: string) =>
    t(
      l,
      `${object}: Not approved. It was closed because the person who started it no longer holds the right.`,
      `${object}: 승인되지 않았습니다. 시작한 사람에게 더 이상 권한이 없어 종료되었습니다.`,
      `${object}: لم يُعتمد. أُغلق لأن من بدأه لم يعد يملك الصلاحية.`,
    ),
  closedFound: (l: Locale, approver: string, request: string) =>
    t(
      l,
      `The person who started it no longer holds the right. Found when ${iso(approver)} tried to approve (request ${iso(request)}).`,
      `시작한 사람에게 더 이상 권한이 없습니다. ${iso(approver)}이(가) 승인하려 할 때 확인되었습니다(요청 ${iso(request)}).`,
      `لم يعد من بدأه يملك الصلاحية. اكتُشف ذلك عندما حاول ${iso(approver)} الاعتماد (الطلب ${iso(request)}).`,
    ),
  closedState: (l: Locale) => t(l, "Closed", "종료됨", "مُغلق"),
  withdrawnState: (l: Locale) => t(l, "Withdrawn", "철회됨", "مسحوب"),
  lastActionLost: (l: Locale) =>
    t(l, "Your last action was not carried out. Nothing was changed.", "마지막 작업은 수행되지 않았습니다. 아무것도 변경되지 않았습니다.", "لم يُنفّذ إجراؤك الأخير. لم يتغيّر شيء."),
  ownTitle: (l: Locale) => t(l, "Your pending items", "내 대기 항목", "عناصرك المعلّقة"),
  ownLead: (l: Locale) =>
    t(
      l,
      "You can withdraw work you started, even without a right for this area.",
      "이 영역의 권한이 없어도 직접 시작한 작업은 철회할 수 있습니다.",
      "يمكنك سحب عمل بدأته، حتى دون صلاحية لهذه المنطقة.",
    ),
  pendingKind: (l: Locale, kind: PendingKind) => {
    const w = pendingWords[kind];
    return t(l, w[0], w[1], w[2]);
  },
  pendingTag: (l: Locale) => t(l, "Pending", "승인 대기", "معلّق"),
  forTarget: (l: Locale) => t(l, "For", "대상", "الهدف"),
  withdrawName: (l: Locale, kind: string, id: string) => t(l, `Withdraw ${inSentence(l, kind)} ${id}`, `${kind} ${id} 철회`, `سحب ${kind} ${id}`),
  confirmWithdraw: (l: Locale) => t(l, "Confirm withdrawal", "철회 확인", "تأكيد السحب"),
  withdrawImpact: (l: Locale) =>
    t(l, "Withdrawing closes this request. Nothing else changes.", "철회하면 이 요청이 종료됩니다. 다른 것은 바뀌지 않습니다.", "يُغلق السحب هذا الطلب. لا يتغيّر شيء آخر."),
  withdrawnDone: (l: Locale, kind: string, id: string) => t(l, `${kind} ${id} withdrawn.`, `${kind} ${id}을(를) 철회했습니다.`, `سُحب ${kind} ${id}.`),
  held: (l: Locale) => t(l, "Held", "보류됨", "محتجَز"),
  heldReason: (l: Locale) => t(l, "Approver no longer holds the right", "승인자에게 더 이상 권한이 없음", "لم يعد المعتمِد يملك الصلاحية"),
  closedReason: (l: Locale) => t(l, "Requester no longer holds the right", "요청자에게 더 이상 권한이 없음", "لم يعد مقدّم الطلب يملك الصلاحية"),
  heldTitle: (l: Locale) => t(l, "Redelivery decision", "재발송 결정", "قرار إعادة التسليم"),
  heldStatus: (l: Locale) =>
    t(
      l,
      "Held: the approver no longer holds the event reprocessing right. Another holder must approve it again.",
      "보류됨: 승인자에게 더 이상 이벤트 재처리 권한이 없습니다. 다른 권한 보유자가 다시 승인해야 합니다.",
      "محتجَز: لم يعد المعتمِد يملك صلاحية إعادة معالجة الأحداث. يجب أن يعتمده صاحب صلاحية آخر مجددًا.",
    ),
  requestedBy: (l: Locale, who: string) => t(l, `Requested by ${iso(who)}`, `요청자 ${iso(who)}`, `طلبه ${iso(who)}`),
  previouslyApprovedBy: (l: Locale) => t(l, "Previously approved by", "이전 승인자", "اعتمده سابقًا"),
  againButton: (l: Locale) => t(l, "Approve again", "다시 승인", "اعتماد مجددًا"),
  againOwnRequest: (l: Locale) =>
    t(l, "You requested this redelivery. Another holder must approve it.", "본인이 이 재발송을 요청했습니다. 다른 권한 보유자가 승인해야 합니다.", "أنت من طلب إعادة التسليم هذه. يجب أن يعتمدها صاحب صلاحية آخر."),
  againOwnApproval: (l: Locale) =>
    t(l, "You approved it before. Another holder must approve it.", "이전에 본인이 승인했습니다. 다른 권한 보유자가 승인해야 합니다.", "اعتمدتها سابقًا. يجب أن يعتمدها صاحب صلاحية آخر."),
  confirmAgain: (l: Locale) => t(l, "Confirm re-approval", "재승인 확인", "تأكيد إعادة الاعتماد"),
  againImpact: (l: Locale) =>
    t(
      l,
      "The redelivery is released to the queue. Nothing is sent: the dispatcher is not connected in this build.",
      "재발송 건이 대기열로 돌아갑니다. 아무것도 보내지 않습니다. 이 빌드에서는 발송기가 연결되어 있지 않습니다.",
      "تُعاد إعادة التسليم إلى قائمة الانتظار. لا يُرسَل شيء: المُرسِل غير متصل في هذا الإصدار.",
    ),
  againVerb: (l: Locale, id: string) => t(l, `Approve redelivery ${iso(id)} again`, `재발송 ${iso(id)} 다시 승인`, `اعتماد إعادة التسليم ${iso(id)} مجددًا`),
  againDone: (l: Locale, id: string) =>
    t(l, `Redelivery ${iso(id)} approved again and queued. Nothing is sent.`, `재발송 ${iso(id)}을(를) 다시 승인해 대기열에 넣었습니다. 아무것도 보내지 않습니다.`, `اعتُمدت إعادة التسليم ${iso(id)} مجددًا وأُضيفت إلى قائمة الانتظار. لا يُرسَل شيء.`),
  proposalObject: (l: Locale, n: string) => t(l, `Proposal ${n}`, `제안 ${n}`, `المقترح ${n}`),
  subscriptionObject: (l: Locale, id: string) => t(l, `Subscription ${iso(id)}`, `구독 ${iso(id)}`, `الاشتراك ${iso(id)}`),
  letterObject: (l: Locale, id: string) => t(l, `Dead letter ${iso(id)}`, `실패 보관 건 ${iso(id)}`, `الرسالة غير المُسلّمة ${iso(id)}`),
  redeliveryObject: (l: Locale, id: string) => t(l, `Redelivery ${iso(id)}`, `재발송 ${iso(id)}`, `إعادة التسليم ${iso(id)}`),
  closedAutomatically: (l: Locale) => t(l, "Closed automatically", "자동 종료", "أُغلق تلقائيًا"),
  notResponding: (l: Locale) =>
    t(l, "The integrations service is not responding. Try again shortly.", "연동 서비스가 응답하지 않습니다. 잠시 후 다시 시도하세요.", "خدمة التكاملات لا تستجيب. أعد المحاولة بعد قليل."),
  loadFailed: (l: Locale) => t(l, "The integrations could not be loaded.", "연동 정보를 불러오지 못했습니다.", "تعذّر تحميل التكاملات."),
  readOnlyAfterFailure: (l: Locale) =>
    t(
      l,
      "The last reload failed, so this screen is read-only until a reload succeeds.",
      "마지막 새로 고침에 실패해, 새로 고침이 성공할 때까지 이 화면은 읽기 전용입니다.",
      "فشلت آخر إعادة تحميل، لذا هذه الشاشة للقراءة فقط حتى تنجح إعادة التحميل.",
    ),
  proposeChange: (l: Locale) => t(l, "Propose a change", "변경 제안", "اقتراح تغيير"),
  withdrawSubscription: (l: Locale) => t(l, "Withdraw", "철회", "سحب"),
  withdrawResume: (l: Locale) => t(l, "Withdraw resume request", "재개 요청 철회", "سحب طلب الاستئناف"),
  withdrawRedeliver: (l: Locale) => t(l, "Withdraw redelivery request", "재발송 요청 철회", "سحب طلب إعادة التسليم"),
  withdrawDismiss: (l: Locale) => t(l, "Withdraw dismissal request", "정리 요청 철회", "سحب طلب الإزالة"),
};
