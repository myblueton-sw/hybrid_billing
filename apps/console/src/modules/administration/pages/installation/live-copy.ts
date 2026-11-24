import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  actionWords, artifactWords, causeWords, checkWords, conflictWords, effectWords, failureWords, installedWords, observationWords, permissionWords, proposalKindWords,
  proposalStateWords, reasonWords, stepWords, summaryWords, unknownCode, words,
} from "./live-codes";

export const iso = (s: string) => (s ? "⁨" + s + "⁩" : s);

const reasonPhrase = (l: Locale, code: string) => words(l, reasonWords, code);

export const k = {
  deniedTitle: (l: Locale) => t(l, "Installation is not available to you", "설치 화면을 볼 수 없습니다", "التثبيت غير متاح لك"),
  denied: (l: Locale) =>
    t(l, "This screen needs installation administration. Ask an administrator if you need access.", "이 화면은 설치 관리 권한이 필요합니다. 필요하면 관리자에게 요청하세요.", "تحتاج هذه الشاشة إلى صلاحية إدارة التثبيت. اطلب الوصول من مسؤول إذا احتجت إليه."),
  sessionEnded: (l: Locale) => t(l, "Your session has ended.", "세션이 끝났습니다.", "انتهت جلستك."),
  signInAgain: (l: Locale) => t(l, "Sign in again", "다시 로그인", "تسجيل الدخول مجددًا"),
  loading: (l: Locale) => t(l, "Loading installation…", "설치 정보를 불러오는 중…", "جارٍ تحميل التثبيت…"),
  reload: (l: Locale) => t(l, "Reload", "새로 고침", "إعادة التحميل"),
  site: (l: Locale) => t(l, "Site", "설치 사이트", "الموقع"),
  revision: (l: Locale) => t(l, "Data revision", "데이터 리비전", "مراجعة البيانات"),
  now: (l: Locale) => t(l, "Now (UTC)", "현재(UTC)", "الآن (UTC)"),
  viewTitle: (l: Locale, v: string) =>
    v === "diagnostics" ? t(l, "Diagnostics", "진단", "التشخيص")
      : v === "support" ? t(l, "Support bundle", "지원 번들", "حزمة الدعم")
        : v === "access" ? t(l, "Limited support access", "제한된 지원 접근", "وصول دعم محدود")
          : t(l, "Install plan and proposals", "설치 계획과 제안", "خطة التثبيت والمقترحات"),
  statementLead: (l: Locale) => t(l, "Records only.", "기록만 합니다.", "تسجيل فقط."),
  statement: (l: Locale) =>
    t(l, "Approving here records a decision. Nothing is installed, downloaded or changed on a host in this build.", "여기서 승인하면 결정만 기록됩니다. 이번 빌드에서는 어떤 호스트에도 설치, 다운로드, 변경을 하지 않습니다.", "الاعتماد هنا يسجّل قرارًا فقط. لا يُثبَّت أو يُنزَّل أو يُغيَّر أي شيء على أي مضيف في هذا الإصدار."),
  notConfigured: (l: Locale) => t(l, "No installation site is configured in this workspace", "이 업무 공간에는 설치 사이트가 설정되어 있지 않습니다", "لا يوجد موقع تثبيت مُعدّ في مساحة العمل هذه"),
  stateUnknown: (l: Locale, code: string) => t(l, `The installation read returned an unknown state (${code}).`, `설치 조회가 알 수 없는 상태(${code})를 반환했습니다.`, `أعادت قراءة التثبيت حالة غير معروفة (${code}).`),
  unknown: (l: Locale) => t(l, "Unknown", "알 수 없음", "غير معروف"),
  unknownCode,
  none: (l: Locale) => t(l, "None", "없음", "لا شيء"),
  observation: (l: Locale) => t(l, "Observation", "사이트 관측", "المراقبة"),
  obsTag: (l: Locale, s: string) => words(l, observationWords, s),
  observed: (l: Locale, a: string, b: string) => t(l, `Observed ${iso(a)}; valid until ${iso(b)}`, `${iso(a)} 관측, ${iso(b)}까지 유효`, `رُصد في ${iso(a)}؛ صالح حتى ${iso(b)}`),
  obsBlocks: (l: Locale) => t(l, "This blocks plan approvals and new plan proposals.", "이 상태에서는 계획 승인과 새 계획 제안을 할 수 없습니다.", "يمنع هذا اعتماد الخطط واقتراح خطط جديدة."),
  obsRestart: (l: Locale) =>
    t(l, "A stale observation cannot recover until the service restarts in this build.", "이번 빌드에서는 서비스를 다시 시작하기 전까지 오래된 관측이 갱신되지 않습니다.", "لا يمكن تحديث المراقبة القديمة في هذا الإصدار حتى تُعاد تشغيل الخدمة."),
  obsMissing: (l: Locale) => t(l, "No current observation is available.", "현재 사용할 수 있는 관측이 없습니다.", "لا تتوفر مراقبة حالية."),
  obsUnknown: (l: Locale) => t(l, "The observation state could not be determined.", "관측 상태를 확인할 수 없습니다.", "تعذّر تحديد حالة المراقبة."),
  installed: (l: Locale) => t(l, "Installed", "설치 상태", "المثبّت"),
  installedTag: (l: Locale, s: string) => words(l, installedWords, s),
  notInstalled: (l: Locale) => t(l, "Not installed.", "설치되지 않았습니다.", "غير مثبّت."),
  ownedServices: (l: Locale) => t(l, "Owned services", "관리 중인 서비스", "الخدمات المملوكة"),
  ownedVolumes: (l: Locale) => t(l, "Owned volumes", "관리 중인 볼륨", "وحدات التخزين المملوكة"),
  history: (l: Locale) => t(l, "Install history entries", "설치 이력 항목", "إدخالات سجل التثبيت"),
  backups: (l: Locale) => t(l, "Backups", "백업", "النسخ الاحتياطية"),
  containsData: (l: Locale) => t(l, "contains data", "데이터 있음", "تحتوي بيانات"),
  noData: (l: Locale) => t(l, "no data", "데이터 없음", "بلا بيانات"),
  dataUnknown: (l: Locale) => t(l, "data unknown", "데이터 여부 알 수 없음", "البيانات غير معروفة"),
  detached: (l: Locale) => t(l, "detached", "분리됨", "مفصولة"),
  installPlan: (l: Locale) => t(l, "Install plan", "설치 계획", "خطة التثبيت"),
  version: (l: Locale) => t(l, "Version", "버전", "الإصدار"),
  hash: (l: Locale) => t(l, "Hash", "해시", "التجزئة"),
  availability: (l: Locale) => t(l, "Availability", "사용 가능 여부", "التوفر"),
  available: (l: Locale) => t(l, "Available", "사용 가능", "متاح"),
  notAvailable: (l: Locale, code: string) => t(l, `Not available: ${reasonPhrase("en", code)}`, `사용 불가: ${reasonPhrase("ko", code)}`, `غير متاح: ${reasonPhrase("ar", code)}`),
  notApplicable: (l: Locale, why: string) =>
    why === "no_release" ? t(l, "Not applicable: no release to preview", "해당 없음: 미리 볼 릴리스가 없습니다", "لا ينطبق: لا يوجد إصدار للمعاينة")
      : why === "no_newer" ? t(l, "Not applicable: no other release to upgrade to", "해당 없음: 업그레이드할 다른 릴리스가 없습니다", "لا ينطبق: لا يوجد إصدار آخر للترقية إليه")
        : why === "no_earlier" ? t(l, "Not applicable: no earlier release to roll back to", "해당 없음: 롤백할 이전 릴리스가 없습니다", "لا ينطبق: لا يوجد إصدار سابق للتراجع إليه")
          : t(l, "Not applicable: nothing is installed", "해당 없음: 설치된 것이 없습니다", "لا ينطبق: لا شيء مثبّت"),
  conflicts: (l: Locale) => t(l, "Conflicts", "충돌", "التعارضات"),
  conflictLead: (l: Locale) => t(l, "A conflict blocks approval of this plan.", "충돌이 있으면 이 계획을 승인할 수 없습니다.", "يمنع التعارض اعتماد هذه الخطة."),
  services: (l: Locale) => t(l, "Services", "서비스", "الخدمات"),
  ports: (l: Locale) => t(l, "Ports", "포트", "المنافذ"),
  volumes: (l: Locale) => t(l, "Volumes", "볼륨", "وحدات التخزين"),
  dataEffects: (l: Locale) => t(l, "Data effects", "데이터 영향", "آثار البيانات"),
  permissions: (l: Locale) => t(l, "Permissions to grant", "부여할 권한", "الصلاحيات المطلوب منحها"),
  leftovers: (l: Locale) => t(l, "Leftovers", "남는 항목", "البقايا"),
  action: (l: Locale, s: string) => words(l, actionWords, s),
  conflict: (l: Locale, s: string) => words(l, conflictWords, s),
  effect: (l: Locale, s: string) => words(l, effectWords, s),
  permission: (l: Locale, s: string) => words(l, permissionWords, s),
  artifact: (l: Locale, s: string) => words(l, artifactWords, s),
  basisPending: (l: Locale) => t(l, "Previewed because it is the newest pending plan proposal.", "승인 대기 중인 가장 최근 계획 제안이라서 미리 봅니다.", "تُعرض لأنها أحدث مقترح خطة بانتظار الاعتماد."),
  basisRelease: (l: Locale) => t(l, "Previewed because it is the newest release not installed.", "설치되지 않은 가장 최근 릴리스라서 미리 봅니다.", "تُعرض لأنها أحدث إصدار غير مثبّت."),
  accepted: (l: Locale, who: string) =>
    [t(l, "Plan", "계획", "الخطة"), t(l, `accepted by ${iso(who)}. Not installed: runs are not connected in this build.`, `을(를) ${iso(who)}이(가) 수락했습니다. 이번 빌드에서는 실행이 연결되지 않아 설치하지 않았습니다.`, `قبلها ${iso(who)}. لم تُثبَّت: عمليات التشغيل غير متصلة في هذا الإصدار.`)] as const,
  upgrade: (l: Locale) => t(l, "Upgrade", "업그레이드", "الترقية"),
  rollback: (l: Locale) => t(l, "Rollback", "롤백", "التراجع"),
  showEffects: (l: Locale) => t(l, "Show effects", "영향 보기", "عرض الآثار"),
  propose: (l: Locale) => t(l, "Propose this plan", "이 계획 제안", "اقتراح هذه الخطة"),
  proposeBlocked: (l: Locale, blocker: string, v: string) => {
    switch (blocker) {
      case "unavailable":
        return t(l, "The plan cannot be proposed while it is not available.", "계획을 사용할 수 없는 동안에는 제안할 수 없습니다.", "لا يمكن اقتراح الخطة ما دامت غير متاحة.");
      case "not_applicable":
        return t(l, "There is no plan to propose.", "제안할 계획이 없습니다.", "لا توجد خطة لاقتراحها.");
      case "conflicts":
        return t(l, "Remove the conflicts before proposing this plan.", "이 계획을 제안하기 전에 충돌을 해결하세요.", "أزِل التعارضات قبل اقتراح هذه الخطة.");
      case "conflicts_unknown":
        return t(l, "The plan's conflicts are unknown, so it cannot be proposed.", "계획의 충돌 여부를 알 수 없어 제안할 수 없습니다.", "تعارضات الخطة غير معروفة، لذا لا يمكن اقتراحها.");
      case "hash":
        return t(l, "The plan has no valid hash, so it cannot be proposed.", "계획에 올바른 해시가 없어 제안할 수 없습니다.", "لا تملك الخطة تجزئة صالحة، لذا لا يمكن اقتراحها.");
      case "accepted":
        return t(l, `Plan ${iso(v)} is already accepted.`, `계획 ${iso(v)}은(는) 이미 수락되었습니다.`, `الخطة ${iso(v)} مقبولة بالفعل.`);
      case "waiting":
        return t(l, `A plan proposal for ${iso(v)} is already waiting for approval.`, `${iso(v)} 계획 제안이 이미 승인을 기다리고 있습니다.`, `يوجد مقترح خطة للإصدار ${iso(v)} بانتظار الاعتماد بالفعل.`);
      default:
        return t(l, "This plan cannot be proposed now.", "지금은 이 계획을 제안할 수 없습니다.", "لا يمكن اقتراح هذه الخطة الآن.");
    }
  },
  staleOwn: (l: Locale) =>
    t(l, "The install plan changed after you proposed this. Withdraw it and propose the current plan.", "제안한 뒤 설치 계획이 바뀌었습니다. 기존 제안을 철회하고 현재 계획을 제안하세요.", "تغيّرت خطة التثبيت بعد أن اقترحتها. اسحبها واقترح الخطة الحالية."),
  releases: (l: Locale) => t(l, "Releases", "릴리스", "الإصدارات"),
  keyVersion: (l: Locale) => t(l, "Key version", "키 버전", "إصدار المفتاح"),
  upgradeFrom: (l: Locale) => t(l, "Upgrade from", "업그레이드 가능 버전", "الترقية من"),
  platforms: (l: Locale) => t(l, "Platforms", "플랫폼", "المنصات"),
  mark: (l: Locale) => t(l, "Mark", "표시", "العلامة"),
  markedBy: (l: Locale, who: string) => t(l, `Marked by ${iso(who)}:`, `${iso(who)} 표시:`, `وضع العلامة ${iso(who)}:`),
  confirmedBy: (l: Locale, who: string) => t(l, `confirmed by ${iso(who)}`, `${iso(who)} 확인`, `أكّدها ${iso(who)}`),
  noReleases: (l: Locale) => t(l, "No releases are registered.", "등록된 릴리스가 없습니다.", "لا توجد إصدارات مسجّلة."),
  proposals: (l: Locale) => t(l, "Installer proposals", "설치 관리자 제안", "مقترحات المثبّت"),
  noProposals: (l: Locale) => t(l, "No installer proposals", "설치 관리자 제안이 없습니다", "لا توجد مقترحات للمثبّت"),
  waitingGroup: (l: Locale, n: number) => t(l, `Waiting for approval (${n})`, `승인 대기(${n})`, `بانتظار الاعتماد (${n})`),
  closedGroup: (l: Locale, n: number) => t(l, `Closed (${n})`, `종료(${n})`, `مغلقة (${n})`),
  newest: (l: Locale, n: number) => t(l, `Showing the newest ${n}.`, `최근 ${n}건만 표시합니다.`, `تُعرض أحدث ${n} فقط.`),
  proposalN: (l: Locale, n: string) => t(l, `Proposal ${n}`, `제안 ${n}`, `المقترح ${n}`),
  kind: (l: Locale, s: string) => words(l, proposalKindWords, s),
  state: (l: Locale, s: string) => words(l, proposalStateWords, s),
  summaryLead: (l: Locale, s: string) => words(l, summaryWords, s),
  proposedBy: (l: Locale, who: string) => t(l, `Proposed by ${iso(who)}`, `제안자 ${iso(who)}`, `اقترحه ${iso(who)}`),
  created: (l: Locale, d: string) => t(l, `Created ${iso(d)}`, `${iso(d)} 생성`, `أُنشئ ${iso(d)}`),
  reason: (l: Locale) => t(l, "Reason", "사유", "السبب"),
  noReason: (l: Locale) => t(l, "No reason recorded", "기록된 사유 없음", "لا سبب مسجّل"),
  approvedBy: (l: Locale, who: string) => t(l, `Approved by ${iso(who)}`, `승인자 ${iso(who)}`, `اعتمده ${iso(who)}`),
  withdrawnBy: (l: Locale, who: string) => t(l, `Withdrawn by ${iso(who)}`, `철회자 ${iso(who)}`, `سحبه ${iso(who)}`),
  closedBy: (l: Locale, who: string) => t(l, `Closed by ${iso(who)}`, `종료자 ${iso(who)}`, `أغلقه ${iso(who)}`),
  closingReason: (l: Locale) => t(l, "Closing reason", "종료 사유", "سبب الإغلاق"),
  proposedHash: (l: Locale) => t(l, "Proposed hash", "제안한 해시", "التجزئة المقترحة"),
  currentHash: (l: Locale) => t(l, "Current hash", "현재 해시", "التجزئة الحالية"),
  hashChanged: (l: Locale) => t(l, "The install plan changed after this was proposed.", "제안 이후 설치 계획이 바뀌었습니다.", "تغيّرت خطة التثبيت بعد اقتراح هذا."),
  approve: (l: Locale) => t(l, "Approve", "승인", "اعتماد"),
  withdraw: (l: Locale) => t(l, "Withdraw", "철회", "سحب"),
  approveBlocked: (l: Locale, code: string) => {
    switch (code) {
      case "own_proposal":
        return t(l, "You proposed this. Another administrator must approve it.", "본인이 제안했습니다. 다른 관리자가 이 제안을 승인해야 합니다.", "أنت من اقترح هذا. يجب أن يعتمده مسؤول آخر.");
      case "not_pending":
        return t(l, "This proposal is closed.", "이 제안은 종료되었습니다.", "هذا المقترح مغلق.");
      case "plan_changed":
        return t(l, "The install plan changed after this was proposed. It cannot be approved.", "제안 이후 설치 계획이 바뀌어 승인할 수 없습니다.", "تغيّرت خطة التثبيت بعد اقتراح هذا. لا يمكن اعتماده.");
      case "observation_stale":
        return t(l, "The site observation is stale, so this plan cannot be approved.", "사이트 관측이 오래되어 이 계획을 승인할 수 없습니다.", "مراقبة الموقع قديمة، لذا لا يمكن اعتماد هذه الخطة.");
      case "not_observed":
        return t(l, "No current site observation is available, so this plan cannot be approved.", "현재 사이트 관측이 없어 이 계획을 승인할 수 없습니다.", "لا تتوفر مراقبة حالية للموقع، لذا لا يمكن اعتماد هذه الخطة.");
      case "conflicts":
        return t(l, "The install plan has conflicts. A conflict blocks approval.", "설치 계획에 충돌이 있습니다. 충돌이 있으면 승인할 수 없습니다.", "خطة التثبيت فيها تعارضات. يمنع التعارض الاعتماد.");
    }
    if (Object.prototype.hasOwnProperty.call(reasonWords, code)) return t(l, `Approval is not available: ${reasonPhrase("en", code)}.`, `승인 불가: ${reasonPhrase("ko", code)}.`, `الاعتماد غير متاح: ${reasonPhrase("ar", code)}.`);
    return t(l, "Approval is not available for this proposal now.", "지금은 이 제안을 승인할 수 없습니다.", "الاعتماد غير متاح لهذا المقترح الآن.");
  },
  withdrawBlocked: (l: Locale) => t(l, "Withdrawal is not available for this proposal now.", "지금은 이 제안을 철회할 수 없습니다.", "السحب غير متاح لهذا المقترح الآن."),
  noCommand: (l: Locale) => t(l, "This proposal cannot be acted on from this screen.", "이 화면에서는 이 제안을 처리할 수 없습니다.", "لا يمكن التعامل مع هذا المقترح من هذه الشاشة."),
  keptApprove: (l: Locale) =>
    t(l, "The installer refused this approval. Nothing changed. Check the destinations, or ask the proposer to withdraw it.", "설치 관리자가 이 승인을 거부했습니다. 바뀐 것은 없습니다. 목적지를 확인하거나 제안자에게 철회를 요청하세요.", "رفض المثبّت هذا الاعتماد. لم يتغيّر شيء. تحقّق من الوجهات، أو اطلب من صاحب المقترح سحبه."),
  keptWithdraw: (l: Locale) => t(l, "The installer refused this withdrawal. Nothing changed.", "설치 관리자가 이 철회를 거부했습니다. 바뀐 것은 없습니다.", "رفض المثبّت هذا السحب. لم يتغيّر شيء."),
  keptPropose: (l: Locale) =>
    t(l, "The installer refused this plan proposal. Review the current plan; you may also have too many pending proposals.", "설치 관리자가 이 계획 제안을 거부했습니다. 현재 계획을 검토하세요. 승인 대기 중인 제안이 너무 많을 수도 있습니다.", "رفض المثبّت مقترح الخطة هذا. راجع الخطة الحالية؛ وقد يكون لديك عدد كبير من المقترحات المعلّقة."),
  gone: (l: Locale) => t(l, "This proposal is no longer available.", "이 제안은 더 이상 사용할 수 없습니다.", "لم يعد هذا المقترح متاحًا."),
  goneProposal: (l: Locale) => t(l, "The plan can no longer be proposed here.", "이 계획은 더 이상 여기서 제안할 수 없습니다.", "لم يعد بالإمكان اقتراح الخطة هنا."),
  pendingAttempt: (l: Locale) => t(l, "The last request may have been applied. Check the proposals, or send it again.", "직전 요청이 이미 반영되었을 수 있습니다. 제안 목록을 확인하거나 다시 보내세요.", "ربما طُبّق الطلب الأخير. راجع المقترحات أو أرسله مجددًا."),
  planChanged: (l: Locale) => t(l, "The plan changed; review the current plan.", "계획이 바뀌었습니다. 현재 계획을 검토하세요.", "تغيّرت الخطة؛ راجع الخطة الحالية."),
  conflictAgain: (l: Locale) => t(l, "The installation data changed. Check the values and confirm again.", "설치 데이터가 바뀌었습니다. 값을 확인하고 다시 확인하세요.", "تغيّرت بيانات التثبيت. تحقّق من القيم وأكّد مرة أخرى."),
  renew: (l: Locale) => t(l, "That request expired. Review and confirm again.", "요청이 만료되었습니다. 다시 검토하고 확인하세요.", "انتهت صلاحية الطلب. راجع وأكّد مجددًا."),
  partial: (l: Locale, op: string) => t(l, `Stopped after a change. Operation ${iso(op)}. Check the proposals before trying again.`, `변경 후 중단되었습니다. 작업 ${iso(op)}. 다시 시도하기 전에 제안 목록을 확인하세요.`, `توقف بعد تغيير. العملية ${iso(op)}. راجع المقترحات قبل المحاولة مجددًا.`),
  approvedDone: (l: Locale, n: string, s: string) => t(l, `Proposal ${n} approved. State: ${s}. Nothing was installed.`, `제안 ${n}을(를) 승인했습니다. 상태: ${s}. 설치한 것은 없습니다.`, `اعتُمد المقترح ${n}. الحالة: ${s}. لم يُثبَّت شيء.`),
  withdrawnDone: (l: Locale, n: string) => t(l, `Proposal ${n} withdrawn.`, `제안 ${n}을(를) 철회했습니다.`, `سُحب المقترح ${n}.`),
  proposedDone: (l: Locale, n: string, v: string) =>
    t(l, `Plan ${iso(v)} proposed as proposal ${n}. Another administrator must approve it.`, `계획 ${iso(v)}을(를) 제안 ${n}(으)로 제출했습니다. 다른 관리자가 승인해야 합니다.`, `اقتُرحت الخطة ${iso(v)} بوصفها المقترح ${n}. يجب أن يعتمدها مسؤول آخر.`),
  duplicate: (l: Locale, n: string) => t(l, `This plan was already proposed as #${n}. No new proposal was created, and that proposal keeps its original reason.`, `이 계획은 이미 #${n}(으)로 제안되었습니다. 새 제안은 만들어지지 않았으며, 해당 제안의 기존 사유가 유지됩니다.`, `اقتُرحت هذه الخطة بالفعل برقم #${n}. لم يُنشأ مقترح جديد، ويبقى سبب ذلك المقترح الأصلي.`),
  form: {
    rationale: (l: Locale) => t(l, "Reason", "사유", "السبب"),
    rationaleHint: (l: Locale, n: number) => t(l, `${n} of 256 bytes. Required. Do not enter customer data.`, `256바이트 중 ${n}바이트. 필수. 고객 정보를 입력하지 마세요.`, `${n} من 256 بايت. مطلوب. لا تُدخل بيانات العملاء.`),
    rationaleMissing: (l: Locale) => t(l, "Enter a reason.", "사유를 입력하세요.", "أدخل سببًا."),
    checkReason: (l: Locale) => t(l, "Check the reason and try again.", "사유를 확인하고 다시 시도하세요.", "تحقّق من السبب وحاول مجددًا."),
    rationaleLong: (l: Locale) => t(l, "The reason is longer than 256 bytes.", "사유가 256바이트를 넘습니다.", "السبب أطول من 256 بايت."),
    review: (l: Locale) => t(l, "Review", "검토", "مراجعة"),
    cancel: (l: Locale) => t(l, "Cancel", "취소", "إلغاء"),
    backToEdit: (l: Locale) => t(l, "Back to edit", "수정으로 돌아가기", "العودة إلى التعديل"),
    sending: (l: Locale) => t(l, "Sending…", "보내는 중…", "جارٍ الإرسال…"),
    confirmPropose: (l: Locale) => t(l, "Confirm plan proposal", "계획 제안 확인", "تأكيد مقترح الخطة"),
    confirmApprove: (l: Locale) => t(l, "Confirm approval", "승인 확인", "تأكيد الاعتماد"),
    confirmWithdraw: (l: Locale) => t(l, "Confirm withdrawal", "철회 확인", "تأكيد السحب"),
    proposeImpact: (l: Locale) =>
      t(l, "This records a proposal. Another administrator must approve it. Nothing is installed.", "제안만 기록합니다. 다른 관리자가 승인해야 합니다. 설치하는 것은 없습니다.", "يسجّل هذا مقترحًا. يجب أن يعتمده مسؤول آخر. لا يُثبَّت شيء."),
    approveImpact: (l: Locale) =>
      t(l, "Approval records the decision and applies it to the installer's registry. Nothing is installed.", "승인하면 결정을 기록하고 설치 관리자 레지스트리에 반영합니다. 설치하는 것은 없습니다.", "يسجّل الاعتماد القرار ويطبّقه على سجل المثبّت. لا يُثبَّت شيء."),
    withdrawImpact: (l: Locale) => t(l, "Withdrawing closes this proposal. Nothing is installed or removed.", "철회하면 이 제안이 종료됩니다. 설치하거나 제거하는 것은 없습니다.", "يُغلق السحب هذا المقترح. لا يُثبَّت أو يُزال شيء."),
    proposeVerb: (l: Locale, v: string) => t(l, `Propose plan ${v}`, `계획 ${v} 제안`, `اقتراح الخطة ${v}`),
    approveVerb: (l: Locale, n: string) => t(l, `Approve proposal ${n}`, `제안 ${n} 승인`, `اعتماد المقترح ${n}`),
    withdrawVerb: (l: Locale, n: string) => t(l, `Withdraw proposal ${n}`, `제안 ${n} 철회`, `سحب المقترح ${n}`),
    proposal: (l: Locale) => t(l, "Proposal", "제안", "المقترح"),
    kind: (l: Locale) => t(l, "Kind", "종류", "النوع"),
    summary: (l: Locale) => t(l, "Summary", "요약", "الملخص"),
    proposerReason: (l: Locale) => t(l, "Proposer's reason", "제안 사유", "سبب صاحب المقترح"),
  },
  destinations: (l: Locale) => t(l, "Allowed destinations", "허용 목적지", "الوجهات المسموح بها"),
  noDestinations: (l: Locale) => t(l, "No destinations are allowed", "허용된 목적지가 없습니다", "لا توجد وجهات مسموح بها"),
  colId: (l: Locale) => t(l, "ID", "ID", "المعرّف"),
  colHost: (l: Locale) => t(l, "Host", "호스트", "المضيف"),
  colPort: (l: Locale) => t(l, "Port", "포트", "المنفذ"),
  colRanges: (l: Locale) => t(l, "Allowed ranges", "허용 주소 범위", "النطاقات المسموح بها"),
  reports: (l: Locale) => t(l, "Diagnostic reports", "진단 보고", "تقارير التشخيص"),
  noReports: (l: Locale) => t(l, "No diagnostic reports", "진단 보고가 없습니다", "لا توجد تقارير تشخيص"),
  colCheck: (l: Locale) => t(l, "Check", "검사", "الفحص"),
  colTarget: (l: Locale) => t(l, "Target", "대상", "الهدف"),
  colStatus: (l: Locale) => t(l, "Status", "상태", "الحالة"),
  colRequired: (l: Locale) => t(l, "Required or warning", "필수 또는 경고", "مطلوب أو تحذير"),
  colRoute: (l: Locale) => t(l, "Route", "경로", "المسار"),
  colNext: (l: Locale) => t(l, "Next action", "다음 조치", "الإجراء التالي"),
  required: (l: Locale) => t(l, "Required", "필수", "مطلوب"),
  notResolved: (l: Locale) => t(l, "Not resolved", "확인되지 않음", "لم يُحلّ"),
  direct: (l: Locale) => t(l, "Direct", "직접 연결", "مباشر"),
  via: (l: Locale) => t(l, "Through", "경유", "عبر"),
  check: (l: Locale, s: string) => words(l, checkWords, s),
  failure: (l: Locale, s: string) => words(l, failureWords, s),
  cause: (l: Locale, s: string) => words(l, causeWords, s),
  step: (l: Locale, s: string) => words(l, stepWords, s),
  requests: (l: Locale) => t(l, "Diagnostic requests", "진단 요청", "طلبات التشخيص"),
  requestsBody: (l: Locale) =>
    t(l, "Diagnostic requests need the site's probe and are not connected in this build.", "진단 요청에는 사이트 프로브가 필요하며 이번 빌드에서는 연결되지 않았습니다.", "تحتاج طلبات التشخيص إلى مسبار الموقع وهي غير متصلة في هذا الإصدار."),
  unconnected: (l: Locale) => t(l, "Not connected in this build.", "이번 빌드에서는 연결되지 않았습니다.", "غير متصل في هذا الإصدار."),
  willHold: (l: Locale) => t(l, "When connected, this will hold:", "연결되면 다음을 담습니다:", "عند الاتصال سيحتوي على:"),
  supportItems: (l: Locale) => [
    t(l, "A preview of the material the bundle includes", "번들에 포함되는 자료 미리 보기", "معاينة للمواد التي تتضمنها الحزمة"),
    t(l, "Exclusions, with secrets and customer content redacted", "제외 항목과 비밀 정보·고객 내용 가림 처리", "الاستثناءات، مع حجب الأسرار ومحتوى العملاء"),
    t(l, "Local export only, with no automatic upload", "자동 업로드 없이 로컬 내보내기만", "تصدير محلي فقط، دون رفع تلقائي"),
  ],
  accessItems: (l: Locale) => [
    t(l, "The actual support actor", "실제 지원 담당자", "جهة الدعم الفعلية"),
    t(l, "The scope of permitted work", "허용된 작업 범위", "نطاق العمل المسموح به"),
    t(l, "Expiry and revocation", "만료와 회수", "الانتهاء والإلغاء"),
    t(l, "No impersonation of approvers", "승인자 대리 수행 금지", "لا انتحال لصفة المعتمِدين"),
  ],
};
