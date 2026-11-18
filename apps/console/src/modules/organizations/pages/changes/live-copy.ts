import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

const iso = (s: string) => (s ? "\u2068" + s + "\u2069" : s);

export const c = {
  revision: (l: Locale) => t(l, "Relationship revision", "관계 리비전", "مراجعة العلاقات"),
  active: (l: Locale, n: number) => t(l, `Active revision ${n}`, `활성 리비전 ${n}`, `المراجعة النشطة ${n}`),
  openCount: (l: Locale, n: number, max: number) => t(l, `${n} of ${max} open`, `열린 계획 ${n}/${max}개`, `${n} من ${max} مفتوحة`),
  openPlans: (l: Locale) => t(l, "Open plans", "열린 변경 계획", "الخطط المفتوحة"),
  from: (l: Locale, d: string) => t(l, `From ${d}`, `${d}부터`, `اعتبارًا من ${d}`),
  staleTag: (l: Locale) => t(l, "Base changed", "기준 변경됨", "تغيّر الأساس"),
  selectedTag: (l: Locale) => t(l, "Selected", "선택됨", "محددة"),
  noOpen: (l: Locale) => t(l, "No open changes.", "열린 변경 계획이 없습니다.", "لا توجد تغييرات مفتوحة."),
  proposeLink: (l: Locale) => t(l, "Propose a change", "변경 제안하기", "اقتراح تغيير"),
  choose: (l: Locale) => t(l, "Choose a plan from the list.", "목록에서 계획을 선택하세요.", "اختر خطة من القائمة."),
  back: (l: Locale) => t(l, "Back to open plans", "열린 계획으로 돌아가기", "العودة إلى الخطط المفتوحة"),
  backClosed: (l: Locale) => t(l, "Back to closed plans", "종료된 계획으로 돌아가기", "العودة إلى الخطط المغلقة"),
  limit: (l: Locale, max: number) =>
    t(l, `${max} plans are open, which is the limit. Discard one before proposing another.`, `열린 변경 계획이 한도인 ${max}개입니다. 새로 제안하려면 먼저 하나를 폐기하세요.`, `عدد الخطط المفتوحة ${max}، وهو الحد الأقصى. تجاهل خطة قبل اقتراح أخرى.`),
  reload: (l: Locale) => t(l, "Reload", "새로 고침", "إعادة التحميل"),
  targetsTitle: (l: Locale) => t(l, "Change target", "변경 대상", "هدف التغيير"),
  impactTitle: (l: Locale) => t(l, "Impact review", "영향 검토", "مراجعة الأثر"),
  approvalTitle: (l: Locale) => t(l, "Approval", "승인", "الاعتماد"),
  resultTitle: (l: Locale) => t(l, "Results", "처리 결과", "النتائج"),
  lead: (l: Locale) => t(l, "Proposing creates a plan whose relationship changes are locked.", "제안하면 관계 변경이 고정된 계획이 만들어집니다.", "يُنشئ الاقتراح خطة بتغييرات علاقات مثبّتة."),
  relationship: (l: Locale) => t(l, "Relationship", "관계 종류", "نوع العلاقة"),
  change: (l: Locale) => t(l, "Change", "변경 방식", "التغيير"),
  changeHint: (l: Locale) => t(l, "Changing the relationship clears this choice.", "관계 종류를 바꾸면 이 선택이 초기화됩니다.", "يؤدي تغيير نوع العلاقة إلى مسح هذا الاختيار."),
  action: (l: Locale, a: string) =>
    a === "set" ? t(l, "Set (replace the current target)", "지정(현재 대상 교체)", "تعيين (استبدال الهدف الحالي)")
      : a === "end" ? t(l, "End (remove from this date)", "종료(이 날짜부터 해제)", "إنهاء (الإزالة من هذا التاريخ)")
        : a === "join" ? t(l, "Join the group", "그룹에 추가", "الانضمام إلى المجموعة")
          : a === "leave" ? t(l, "Leave the group", "그룹에서 제외", "مغادرة المجموعة") : a,
  actionShort: (l: Locale, a: string) =>
    a === "set" ? t(l, "Set", "지정", "تعيين") : a === "end" ? t(l, "End", "종료", "إنهاء") : a === "join" ? t(l, "Join", "추가", "انضمام") : a === "leave" ? t(l, "Leave", "제외", "مغادرة") : a,
  member: (l: Locale) => t(l, "Member", "구성원", "العضو"),
  memberHint: (l: Locale) => t(l, "Internal units, legal entities and members already in a relationship.", "내부 조직, 법인, 관계에 이미 있는 구성원입니다.", "الوحدات الداخلية والكيانات القانونية والأعضاء الموجودون في علاقة."),
  enterId: (l: Locale) => t(l, "Enter an ID", "ID 직접 입력", "إدخال معرّف"),
  pick: (l: Locale) => t(l, "Choose from the list", "목록에서 선택", "الاختيار من القائمة"),
  idHint: (l: Locale) => t(l, "The server checks this ID when you propose.", "제안할 때 서버가 이 ID를 확인합니다.", "يتحقق الخادم من هذا المعرّف عند الاقتراح."),
  memberIdLabel: (l: Locale) => t(l, "Member ID", "구성원 ID", "معرّف العضو"),
  targetIdLabel: (l: Locale) => t(l, "Target ID", "연결 대상 ID", "معرّف الجهة المستهدفة"),
  choosePlaceholder: (l: Locale) => t(l, "Choose…", "선택하세요…", "اختر…"),
  target: (l: Locale) => t(l, "Target", "연결 대상", "الجهة المستهدفة"),
  hintHierarchy: (l: Locale) => t(l, "An active party other than the member.", "구성원이 아닌 활성 법인·조직입니다.", "طرف نشط غير العضو."),
  hintPayer: (l: Locale) => t(l, "An active legal entity other than the member.", "구성원이 아닌 활성 법인입니다.", "كيان قانوني نشط غير العضو."),
  hintPool: (l: Locale) => t(l, "Only existing pools and groups. New ones cannot be created here.", "이미 있는 풀·그룹만 고를 수 있으며 여기서 새로 만들 수 없습니다.", "المجمعات والمجموعات الموجودة فقط، ولا يمكن إنشاء جديد هنا."),
  fromLabel: (l: Locale) => t(l, "Effective from", "적용 시작일", "تاريخ السريان"),
  fromHint: (l: Locale) =>
    t(l, "UTC date, today or later. The change applies from the start of that day (00:00 UTC).", "UTC 기준 오늘 이후 날짜입니다. 해당 날짜 UTC 00:00부터 적용됩니다.", "تاريخ بالتوقيت العالمي المنسق، اليوم أو بعده. يسري التغيير من بداية ذلك اليوم (00:00 UTC)."),
  current: (l: Locale) => t(l, "Current, from the organizations listing", "현재 관계(조직 목록 기준)", "الوضع الحالي، من قائمة المؤسسات"),
  onFrom: (l: Locale, d: string) => t(l, `On ${d} (effective from)`, `${d}(적용 시작일)`, `في ${d} (تاريخ السريان)`),
  today: (l: Locale, d: string) => t(l, `Today, ${d}`, `오늘(${d})`, `اليوم، ${d}`),
  currentNote: (l: Locale) => t(l, "This is the listing as read, not a computed result.", "조회한 목록 그대로이며 계산 결과가 아닙니다.", "هذه القائمة كما قُرئت، وليست نتيجة محسوبة."),
  noneOfKind: (l: Locale) => t(l, "No relationship of this kind", "이 종류의 관계 없음", "لا توجد علاقة من هذا النوع"),
  notInListing: (l: Locale) => t(l, "Not in the loaded listing for this relationship", "불러온 이 관계 목록에 없음", "غير موجود في القائمة المحمّلة لهذه العلاقة"),
  pickFirst: (l: Locale) => t(l, "Choose a member to see its current relationship.", "구성원을 선택하면 현재 관계를 표시합니다.", "اختر عضوًا لعرض علاقته الحالية."),
  incomplete: (l: Locale) => t(l, "Some relationship listings did not load completely, so these lists may be incomplete. You can still enter an ID.", "일부 관계 목록을 모두 불러오지 못해 선택지가 빠졌을 수 있습니다. ID를 직접 입력할 수 있습니다.", "لم تُحمّل بعض قوائم العلاقات بالكامل، لذا قد تكون هذه القوائم ناقصة. يمكنك إدخال معرّف."),
  loadingChoices: (l: Locale) => t(l, "Loading the organizations listing…", "조직 목록을 불러오는 중…", "جارٍ تحميل قائمة المؤسسات…"),
  propose: (l: Locale) => t(l, "Propose change", "변경 제안", "اقتراح التغيير"),
  proposing: (l: Locale) => t(l, "Proposing…", "제안하는 중…", "جارٍ الاقتراح…"),
  eKind: (l: Locale) => t(l, "Choose a relationship.", "관계 종류를 선택하세요.", "اختر نوع العلاقة."),
  eChange: (l: Locale) => t(l, "Choose a change.", "변경 방식을 선택하세요.", "اختر التغيير."),
  eMember: (l: Locale) => t(l, "Choose a member or enter its ID.", "구성원을 선택하거나 ID를 입력하세요.", "اختر عضوًا أو أدخل معرّفه."),
  eTarget: (l: Locale) => t(l, "Choose a target or enter its ID.", "연결 대상을 선택하거나 ID를 입력하세요.", "اختر الجهة المستهدفة أو أدخل معرّفها."),
  eDate: (l: Locale) => t(l, "Enter a date as YYYY-MM-DD.", "날짜를 YYYY-MM-DD 형식으로 입력하세요.", "أدخل التاريخ بالصيغة YYYY-MM-DD."),
  ePast: (l: Locale) => t(l, "Enter today or a later date (UTC).", "오늘(UTC) 이후 날짜를 입력하세요.", "أدخل تاريخ اليوم أو تاريخًا لاحقًا (UTC)."),
  eMemberServer: (l: Locale) => t(l, "Not found in the active revision.", "활성 리비전에서 찾을 수 없습니다.", "غير موجود في المراجعة النشطة."),
  eTargetServer: (l: Locale) => t(l, "Not usable as a target for this relationship in the active revision.", "활성 리비전에서 이 관계의 연결 대상으로 쓸 수 없습니다.", "لا يمكن استخدامه هدفًا لهذه العلاقة في المراجعة النشطة."),
  eActionServer: (l: Locale) => t(l, "This change is not allowed for this relationship, or the member already has too many entries.", "이 관계에 허용되지 않는 변경이거나 구성원의 항목이 너무 많습니다.", "هذا التغيير غير مسموح لهذه العلاقة، أو أن لدى العضو إدخالات كثيرة جدًا."),
  eFromServer: (l: Locale) => t(l, "The server did not accept this date. Enter today or a later date (UTC).", "서버가 이 날짜를 받지 않았습니다. 오늘(UTC) 이후 날짜를 입력하세요.", "لم يقبل الخادم هذا التاريخ. أدخل تاريخ اليوم أو تاريخًا لاحقًا (UTC)."),
  eServer: (l: Locale) => t(l, "The server did not accept this value.", "서버가 이 값을 받지 않았습니다.", "لم يقبل الخادم هذه القيمة."),
  conflict: (l: Locale) =>
    t(l, "Relationships changed while you were working. The latest revision is loaded and your entries are kept. Check and submit again.", "작업하는 동안 관계가 바뀌었습니다. 최신 리비전을 불러왔고 입력 내용은 유지했습니다. 확인 후 다시 제출하세요.", "تغيّرت العلاقات أثناء العمل. حُمّلت أحدث مراجعة واحتُفظ بإدخالاتك. تحقّق ثم أرسل مرة أخرى."),
  partialPropose: (l: Locale, op: string) =>
    t(l, `The proposal stopped part way and no plan was kept. Propose the change again. Operation ${op}.`, `제안이 도중에 중단되어 계획이 남지 않았습니다. 변경을 다시 제안하세요. 작업 ${op}.`, `توقف الاقتراح في منتصفه ولم تُحفظ أي خطة. اقترح التغيير مرة أخرى. العملية ${op}.`),
  stopped: (l: Locale, op: string) =>
    t(l, `Stopped after a change. Operation ${op}. Check the results before trying again.`, `변경 도중 중단되었습니다. 작업 ${op}. 다시 시도하기 전에 결과를 확인하세요.`, `توقفت العملية بعد إجراء تغيير. العملية ${op}. تحقّق من النتائج قبل إعادة المحاولة.`),
  renew: (l: Locale) => t(l, "This request expired. Check the values and submit again.", "요청이 만료되었습니다. 값을 확인하고 다시 제출하세요.", "انتهت صلاحية هذا الطلب. تحقّق من القيم وأرسل مرة أخرى."),
  retry: (l: Locale) => t(l, "The service did not answer. Submit again to resend the same request.", "서비스가 응답하지 않았습니다. 다시 제출하면 같은 요청을 다시 보냅니다.", "لم تستجب الخدمة. أرسل مرة أخرى لإعادة إرسال الطلب نفسه."),
  failed: (l: Locale) => t(l, "The request failed.", "요청이 실패했습니다.", "فشل الطلب."),
  progress: (l: Locale) => t(l, "Plan progress", "계획 진행", "تقدّم الخطة"),
  step: (l: Locale, n: number) => t(l, `Step ${n}`, `${n}단계`, `الخطوة ${n}`),
  stateWord: (l: Locale, s: string) =>
    s === "sealed" ? t(l, "Changes locked", "변경 고정됨", "التغييرات مقفلة") : s === "rejected" ? t(l, "Rejected", "거부됨", "مرفوضة")
      : s === "activated" ? t(l, "Activated", "활성화됨", "مفعّلة") : s === "discarded" ? t(l, "Discarded", "폐기됨", "تم تجاهلها")
        : s === "draft" ? t(l, "Draft", "초안", "مسودة") : t(l, `Unknown state ${s}`, `알 수 없는 상태 ${s}`, `حالة غير معروفة ${s}`),
  stepNames: (l: Locale) => [t(l, "Target", "대상", "الهدف"), t(l, "Impact", "영향", "الأثر"), t(l, "Approval", "승인", "الاعتماد"), t(l, "Apply", "적용", "التطبيق")],
  stepState: (l: Locale, s: string) =>
    s === "done" ? t(l, "Done", "완료", "مكتملة") : s === "current" ? t(l, "Current", "현재", "الحالية") : s === "held" ? t(l, "Held", "보류", "معلَّقة")
      : s === "stopped" ? t(l, "Ends here", "여기서 종료", "تنتهي هنا") : s === "na" ? t(l, "Not reached", "진행 전", "لم تبدأ") : t(l, "Unknown", "알 수 없음", "غير معروفة"),
  approvalHeld: (l: Locale, b: string) =>
    b === "own" ? t(l, "Proposed by you", "직접 제안함", "اقترحتها أنت") : b === "own_draft" ? t(l, "Draft created or edited by you", "직접 만들거나 고친 초안", "مسودة أنشأتَها أو عدّلتها") : b === "stale" ? t(l, "Base changed", "기준 변경됨", "تغيّر الأساس")
      : b === "passed" ? t(l, "Effective date passed", "적용 시작일 지남", "انقضى تاريخ السريان") : t(l, "Not available", "사용할 수 없음", "غير متاح"),
  heldLead: (l: Locale) => t(l, "Activation is held: the impact of this change on amounts, publication and ERP has not been calculated.", "활성화 보류: 이 변경의 금액·공개·ERP 영향이 계산되지 않았습니다.", "التفعيل معلَّق: لم يُحتسب أثر هذا التغيير على المبالغ والنشر وERP."),
  heldNext: (l: Locale) => t(l, "Impact calculation and the approval request are planned with the billing modules. Until then this plan can only be discarded.", "영향 계산과 승인 요청은 과금 모듈과 함께 제공될 예정입니다. 그전까지 이 계획은 폐기만 할 수 있습니다.", "سيُتاح احتساب الأثر وطلب الاعتماد مع وحدات الفوترة. حتى ذلك الحين لا يمكن إلا تجاهل هذه الخطة."),
  heldUnknownLead: (l: Locale) => t(l, "Activation is held: the impact of this change could not be read.", "활성화 보류: 이 변경의 영향을 확인하지 못했습니다.", "التفعيل معلَّق: تعذّرت قراءة أثر هذا التغيير."),
  heldUnknownNext: (l: Locale) => t(l, "Reload to try again. The plan can still be discarded.", "새로 고침으로 다시 시도하세요. 계획은 계속 폐기할 수 있습니다.", "أعد التحميل للمحاولة مجددًا. لا يزال بالإمكان تجاهل الخطة."),
  facts: {
    member: (l: Locale) => t(l, "Member", "구성원", "العضو"),
    kind: (l: Locale) => t(l, "Relationship", "관계 종류", "نوع العلاقة"),
    change: (l: Locale) => t(l, "Change", "변경 방식", "التغيير"),
    target: (l: Locale) => t(l, "Target", "연결 대상", "الجهة المستهدفة"),
    from: (l: Locale) => t(l, "Effective from", "적용 시작일", "تاريخ السريان"),
    base: (l: Locale) => t(l, "Base revision", "기준 리비전", "المراجعة الأساسية"),
    proposer: (l: Locale) => t(l, "Proposer", "제안자", "مقدّم الاقتراح"),
    at: (l: Locale) => t(l, "Proposed at", "제안 시각", "وقت الاقتراح"),
    reason: (l: Locale) => t(l, "Reason", "사유", "السبب"),
    state: (l: Locale) => t(l, "State", "상태", "الحالة"),
  },
  diff: (l: Locale) => t(l, "Changed members", "변경된 구성원", "الأعضاء المتغيّرون"),
  colMember: (l: Locale) => t(l, "Member", "구성원", "العضو"),
  colChange: (l: Locale) => t(l, "Dated change", "날짜별 변경", "التغيير المؤرخ"),
  colRanges: (l: Locale) => t(l, "Dates that differ", "달라지는 기간", "الفترات المختلفة"),
  colBefore: (l: Locale) => t(l, "Before (any date)", "변경 전 (전체 기간)", "قبل (أي تاريخ)"),
  colAfter: (l: Locale) => t(l, "After (any date)", "변경 후 (전체 기간)", "بعد (أي تاريخ)"),
  closedHere: (l: Locale) => t(l, "The selected plan is closed and is not in this list. See", "선택한 계획은 종료되어 이 목록에 없습니다. 다음에서 확인하세요:", "الخطة المحددة مغلقة وليست في هذه القائمة. راجع"),
  timelineNote: (l: Locale) =>
    t(l, "The before and after lists cover every date in the member's timeline, so a move shows both targets after. The dated change and the ranges say what applies when.", "변경 전후 목록은 구성원 타임라인의 모든 날짜를 포함하므로 이동하면 변경 후에 두 대상이 함께 보입니다. 언제 무엇이 적용되는지는 날짜별 변경과 기간을 보세요.", "تشمل قائمتا قبل وبعد كل تواريخ الجدول الزمني للعضو، لذا يظهر الهدفان معًا بعد النقل. يوضح التغيير المؤرخ والفترات ما يسري ومتى."),
  leadTo: (l: Locale, d: string, target: string) => t(l, `From ${d}: ${iso(target)}`, `${d}부터: ${iso(target)}`, `اعتبارًا من ${d}: ${iso(target)}`),
  leadNone: (l: Locale, d: string) => t(l, `From ${d}: no target`, `${d}부터: 대상 없음`, `اعتبارًا من ${d}: بلا هدف`),
  none: (l: Locale) => t(l, "None", "없음", "لا يوجد"),
  rangeOpen: (l: Locale, d: string) => t(l, `From ${d}`, `${d}부터`, `اعتبارًا من ${d}`),
  rangeClosed: (l: Locale, d: string, e: string) => t(l, `From ${d} until ${e} (not included)`, `${d}부터 ${e}까지(${e} 미포함)`, `من ${d} حتى ${e} (غير مشمول)`),
  notKept: (l: Locale) => t(l, "Changed members are not kept for rejected or discarded plans.", "거부되거나 폐기된 계획은 변경된 구성원을 보관하지 않습니다.", "لا يُحتفظ بالأعضاء المتغيّرين للخطط المرفوضة أو التي تم تجاهلها."),
  truncated: (l: Locale) => t(l, "Not all members shown. More members changed than are listed here.", "일부만 표시합니다. 여기 표시된 것보다 많은 구성원이 바뀌었습니다.", "لم يُعرض كل الأعضاء. تغيّر أعضاء أكثر مما هو مدرج هنا."),
  kinds: (l: Locale) => t(l, "Changes by relationship", "관계 종류별 변경", "التغييرات حسب نوع العلاقة"),
  kindCols: (l: Locale) => [t(l, "Relationship", "관계 종류", "نوع العلاقة"), t(l, "Changed members", "변경된 구성원", "الأعضاء المتغيّرون"), t(l, "Entries before", "변경 전 항목", "الإدخالات قبل"), t(l, "Entries after", "변경 후 항목", "الإدخالات بعد"), t(l, "Earliest change", "가장 이른 변경일", "أقرب تاريخ تغيير")],
  violations: (l: Locale) => t(l, "Why this plan was rejected", "거부 사유", "سبب رفض الخطة"),
  rule: (l: Locale, code: string): [string, string] =>
    code === "no_change" ? [t(l, "No change", "변경 없음", "لا تغيير"), t(l, "The change leaves the relationship as it is.", "이 변경으로 관계가 달라지지 않습니다.", "يترك التغيير العلاقة كما هي.")]
      : code === "cycle" ? [t(l, "Cycle", "순환", "حلقة"), t(l, "The organization would end up above itself.", "조직이 자기 자신의 상위가 됩니다.", "ستصبح المؤسسة فوق نفسها.")]
        : code === "depth" ? [t(l, "Too deep", "단계 초과", "عمق زائد"), t(l, "The organization tree would have more levels than allowed.", "조직 트리 단계가 허용 한도를 넘습니다.", "ستتجاوز شجرة المؤسسات عدد المستويات المسموح به.")]
          : code === "payer_chain" ? [t(l, "Payer chain", "지급 주체 연쇄", "سلسلة جهات دافعة"), t(l, "The payer would itself have a payer at the same time.", "지급 주체에게 같은 기간에 또 다른 지급 주체가 생깁니다.", "سيكون للجهة الدافعة جهة دافعة أخرى في الوقت نفسه.")]
            : code === "member_and_target" ? [t(l, "Member and target", "구성원이자 대상", "عضو وهدف"), t(l, "The same ID would be both a member and a pool in this relationship.", "같은 ID가 이 관계에서 구성원이자 풀이 됩니다.", "سيكون المعرّف نفسه عضوًا ومجمعًا في هذه العلاقة.")]
              : code === "plan_limit" ? [t(l, "Too many problems", "문제 과다", "مشكلات كثيرة"), t(l, "More problems were found than can be listed.", "표시할 수 있는 것보다 많은 문제가 발견되었습니다.", "وُجدت مشكلات أكثر مما يمكن عرضه.")]
                : [t(l, `Rule ${code}`, `규칙 ${code}`, `القاعدة ${code}`), ""],
  impact: (l: Locale) => t(l, "Impact on amounts, publication and ERP", "금액·공개·ERP 영향", "الأثر على المبالغ والنشر وERP"),
  impactRows: (l: Locale) => [t(l, "Amounts", "금액", "المبالغ"), t(l, "Publication", "공개", "النشر"), "ERP"],
  notComputed: (l: Locale) => t(l, "Not calculated", "계산되지 않음", "لم يُحتسب"),
  computedImpact: (l: Locale) => t(l, "Calculated", "계산됨", "محتسَب"),
  unknownImpact: (l: Locale) => t(l, "Could not be read", "확인하지 못함", "تعذّرت القراءة"),
  notZero: (l: Locale) => t(l, "This does not mean no impact.", "영향이 없다는 뜻이 아닙니다.", "لا يعني ذلك عدم وجود أثر."),
  impactNote: (l: Locale) => t(l, "This pre-launch build does not calculate these impacts. Nothing here means zero.", "출시 전 빌드에서는 이 영향을 계산하지 않습니다. 여기 표시된 어떤 항목도 0을 뜻하지 않습니다.", "لا يحتسب هذا الإصدار السابق للإطلاق هذه الآثار، ولا شيء هنا يعني صفرًا."),
  stale: (l: Locale, n: number, b: number) =>
    t(l, `Base changed. The active revision moved to ${n} after this plan was proposed on revision ${b}. Propose the change again.`, `기준 변경됨. 이 계획을 리비전 ${b}에서 제안한 뒤 활성 리비전이 ${n}(으)로 바뀌었습니다. 변경을 다시 제안하세요.`, `تغيّر الأساس. انتقلت المراجعة النشطة إلى ${n} بعد اقتراح هذه الخطة على المراجعة ${b}. اقترح التغيير مرة أخرى.`),
  notFound: (l: Locale) => t(l, "This plan is not among the latest plans, or it is no longer kept.", "이 계획은 최근 계획 목록에 없거나 더 이상 보관되지 않습니다.", "هذه الخطة ليست ضمن أحدث الخطط، أو لم تعد محفوظة."),
  showOpen: (l: Locale) => t(l, "Show open plans", "열린 계획 보기", "عرض الخطط المفتوحة"),
  toApproval: (l: Locale) => t(l, "Continue to approval", "승인으로 이동", "المتابعة إلى الاعتماد"),
  toDiscard: (l: Locale) => t(l, "Go to discard", "폐기로 이동", "الانتقال إلى التجاهل"),
  another: (l: Locale) => t(l, "Propose another change", "다른 변경 제안", "اقتراح تغيير آخر"),
  sentence: (l: Locale, a: string, m: string, tg: string, kind: string) => c.sentenceRaw(l, a, iso(m), iso(tg), kind),
  sentenceRaw: (l: Locale, a: string, member: string, target: string, kind: string) =>
    a === "set" ? t(l, `${member}: set to ${target}`, `${member}을(를) ${target}(으)로 지정`, `تعيين ${target} لـ${member}`)
      : a === "end" ? t(l, `${member}: ${kind} ends`, `${member}의 ${kind} 관계 종료`, `إنهاء علاقة ${kind} لـ${member}`)
        : a === "join" ? t(l, `${member} joins ${target}`, `${member}을(를) ${target}에 추가`, `انضمام ${member} إلى ${target}`)
          : a === "leave" ? t(l, `${member} leaves ${target}`, `${member}을(를) ${target}에서 제외`, `مغادرة ${member} لـ${target}`) : `${member} ${a} ${target}`,
  activate: (l: Locale) => t(l, "Activation", "활성화", "التفعيل"),
  restate: (l: Locale, d: string, sentence: string, kind: string) =>
    t(l, `If this plan is activated, the change applies from ${d} (00:00 UTC): ${sentence} in ${kind}, and a new relationship revision is created.`, `이 계획이 활성화되면 ${d} 00:00(UTC)부터 ${kind}에 다음 변경이 적용되고 새 관계 리비전이 만들어집니다: ${sentence}`, `إذا فُعّلت هذه الخطة، يسري التغيير اعتبارًا من ${d} (00:00 UTC): ${sentence} في ${kind}، وتُنشأ مراجعة علاقات جديدة.`),
  reviewActivate: (l: Locale) => t(l, "Review and activate", "검토 후 활성화", "المراجعة والتفعيل"),
  confirmActivate: (l: Locale) => t(l, "Confirm activation", "활성화 확인", "تأكيد التفعيل"),
  activateBtn: (l: Locale, d: string) => t(l, `Activate, effective ${d}`, `활성화(${d}부터 적용)`, `تفعيل (يسري من ${d})`),
  activating: (l: Locale) => t(l, "Activating…", "활성화하는 중…", "جارٍ التفعيل…"),
  cancel: (l: Locale) => t(l, "Cancel", "취소", "إلغاء"),
  blockedTitle: (l: Locale) => t(l, "Activation is not available", "지금은 활성화할 수 없습니다", "التفعيل غير متاح"),
  own: (l: Locale) => t(l, "You proposed this change. Another administrator must activate it.", "직접 제안한 변경입니다. 다른 관리자가 활성화해야 합니다.", "أنت من اقترح هذا التغيير، ويجب أن يفعّله مسؤول آخر."),
  ownDraft: (l: Locale) =>
    t(l, "You created or edited the draft for this plan. Another administrator must activate it.", "이 계획의 초안을 직접 만들거나 고쳤습니다. 다른 관리자가 활성화해야 합니다.", "أنشأتَ مسودة هذه الخطة أو عدّلتها، ويجب أن يفعّلها مسؤول آخر."),
  staleBlock: (l: Locale) => t(l, "The active revision changed after this was proposed.", "제안 이후 활성 리비전이 바뀌었습니다.", "تغيّرت المراجعة النشطة بعد الاقتراح."),
  passed: (l: Locale) => t(l, "The effective date has passed.", "적용 시작일이 지났습니다.", "انقضى تاريخ السريان."),
  otherBlock: (l: Locale) => t(l, "Activation is not available for this plan.", "이 계획은 활성화할 수 없습니다.", "التفعيل غير متاح لهذه الخطة."),
  refused: (l: Locale, r: string) => (r === "impact" ? c.heldLead(l) : r === "impact_unknown" ? c.heldUnknownLead(l) : r === "own" ? c.own(l) : r === "own_draft" ? c.ownDraft(l) : c.otherBlock(l)),
  next: (l: Locale) => t(l, "Discard this plan and propose it again.", "이 계획을 폐기하고 다시 제안하세요.", "تجاهل هذه الخطة واقترحها مرة أخرى."),
  discard: (l: Locale) => t(l, "Discard", "폐기", "التجاهل"),
  reviewDiscard: (l: Locale) => t(l, "Review and discard", "검토 후 폐기", "المراجعة والتجاهل"),
  confirmDiscard: (l: Locale) => t(l, "Confirm discard", "폐기 확인", "تأكيد التجاهل"),
  permanent: (l: Locale) =>
    t(l, "Discarding is permanent. This plan can no longer be activated. To make this change later, propose it again.", "폐기는 되돌릴 수 없습니다. 이 계획은 더 이상 활성화할 수 없으며, 나중에 변경하려면 다시 제안해야 합니다.", "التجاهل نهائي. لن يمكن تفعيل هذه الخطة بعد ذلك، ولإجراء التغيير لاحقًا اقترحه مرة أخرى."),
  discardBtn: (l: Locale) => t(l, "Discard plan", "계획 폐기", "تجاهل الخطة"),
  discarding: (l: Locale) => t(l, "Discarding…", "폐기하는 중…", "جارٍ التجاهل…"),
  keep: (l: Locale) => t(l, "Keep plan", "계획 유지", "الإبقاء على الخطة"),
  onlySealed: (l: Locale, state: string) =>
    t(l, `State of this plan: ${state}. Only open plans with locked changes can be discarded or activated.`, `이 계획의 상태: ${state}. 변경이 고정된 열린 계획만 폐기하거나 활성화할 수 있습니다.`, `حالة هذه الخطة: ${state}. لا يمكن تجاهل أو تفعيل إلا الخطط المفتوحة ذات التغييرات المقفلة.`),
  stageChanged: (l: Locale) => t(l, "This plan can no longer take this step. The list has been reloaded; check its current state.", "이 계획은 더 이상 이 단계를 진행할 수 없습니다. 목록을 다시 불러왔으니 현재 상태를 확인하세요.", "لم تعد هذه الخطة قادرة على هذه الخطوة. أُعيد تحميل القائمة، فتحقّق من حالتها الحالية."),
  closed: (l: Locale) => t(l, "Closed plans", "종료된 변경 계획", "الخطط المغلقة"),
  resultCols: (l: Locale) => [t(l, "Member", "구성원", "العضو"), t(l, "Change", "변경", "التغيير"), t(l, "Effective from", "적용 시작일", "تاريخ السريان"), t(l, "Outcome", "결과", "النتيجة"), t(l, "Proposed", "제안", "الاقتراح")],
  closing: (l: Locale, k: string, n: number) =>
    k === "activated" ? t(l, `Activated as revision ${n}`, `리비전 ${n}(으)로 활성화`, `فُعّلت كمراجعة ${n}`)
      : k === "rejected" ? t(l, "Rejected when proposed", "제안 시 거부됨", "رُفضت عند الاقتراح")
        : k === "discarded" ? t(l, "Discarded by an administrator", "관리자가 폐기함", "تجاهلها مسؤول")
          : k === "discarded_stale" ? t(l, "Discarded because the base changed", "기준 변경으로 폐기됨", "تم تجاهلها لتغيّر الأساس")
            : k === "discarded_expired" ? t(l, "Discarded because the effective date passed", "적용 시작일 경과로 폐기됨", "تم تجاهلها لانقضاء تاريخ السريان") : t(l, "Closing reason not stated", "종료 사유 없음", "سبب الإغلاق غير مذكور"),
  noClosed: (l: Locale) => t(l, "No closed changes yet.", "아직 종료된 변경이 없습니다.", "لا توجد تغييرات مغلقة بعد."),
  readOnly: (l: Locale) => t(l, "Read-only. Closed plans cannot be changed.", "읽기 전용입니다. 종료된 계획은 바꿀 수 없습니다.", "للقراءة فقط، ولا يمكن تعديل الخطط المغلقة."),
  o01Link: (l: Locale, d: string, n: number) => t(l, `Open organizations on ${d}, revision ${n}`, `${d}·리비전 ${n} 기준 조직 보기`, `عرض المؤسسات في ${d}، المراجعة ${n}`),
  aSealed: (l: Locale) => t(l, "Plan proposed. Its relationship changes are locked.", "계획을 제안했습니다. 관계 변경이 고정되었습니다.", "اقتُرحت الخطة. تغييرات العلاقات مقفلة."),
  aRejected: (l: Locale) => t(l, "Plan rejected. See the reasons.", "계획이 거부되었습니다. 사유를 확인하세요.", "رُفضت الخطة. راجع الأسباب."),
  aActivated: (l: Locale, n: string) => t(l, `Activated. Relationship revision ${n} is now active.`, `활성화했습니다. 관계 리비전 ${n}이(가) 활성 상태입니다.`, `تم التفعيل. مراجعة العلاقات ${n} نشطة الآن.`),
  aDiscarded: (l: Locale) => t(l, "Plan discarded.", "계획을 폐기했습니다.", "تم تجاهل الخطة."),
  aReloaded: (l: Locale) => t(l, "Plans reloaded.", "계획 목록을 다시 불러왔습니다.", "أُعيد تحميل الخطط."),
  loading: (l: Locale) => t(l, "Loading relationship changes…", "관계 변경 계획을 불러오는 중…", "جارٍ تحميل تغييرات العلاقات…"),
  loadFailed: (l: Locale) => t(l, "Relationship changes could not be loaded.", "관계 변경 계획을 불러오지 못했습니다.", "تعذّر تحميل تغييرات العلاقات."),
  prefilled: (l: Locale, r: string) =>
    t(l, `Prefilled from Organizations and relationships, based on relationship revision ${r}. Choose the change and the target.`, `조직·관계 탐색에서 가져온 값입니다. 관계 리비전 ${r} 기준입니다. 변경 방식과 연결 대상을 고르세요.`, `قيم مأخوذة من شاشة «المؤسسات والعلاقات»، استنادًا إلى مراجعة العلاقات ${r}. اختر التغيير والجهة المستهدفة.`),
  prefilledNoBase: (l: Locale) =>
    t(l, "Prefilled from Organizations and relationships. Choose the change and the target.", "조직·관계 탐색에서 가져온 값입니다. 변경 방식과 연결 대상을 고르세요.", "قيم مأخوذة من شاشة «المؤسسات والعلاقات». اختر التغيير والجهة المستهدفة."),
  baseChanged: (l: Locale, r: string, a: string) =>
    t(l, `This change is based on relationship revision ${r}, but revision ${a} is now active. Check the current relationships, then base it on revision ${a}.`,
      `이 변경은 관계 리비전 ${r} 기준이지만 지금 활성 리비전은 ${a}입니다. 현재 관계를 확인한 뒤 리비전 ${a} 기준으로 바꾸세요.`,
      `يستند هذا التغيير إلى مراجعة العلاقات ${r}، لكن المراجعة النشطة الآن ${a}. راجِع العلاقات الحالية ثم اتخذ المراجعة ${a} أساسًا.`),
  baseRefused: (l: Locale, r: string, a: string) =>
    t(l, `The change was not proposed. The relationship revision changed from ${r} to ${a}. Your entries are kept. Check the current relationships, then base the change on revision ${a}.`,
      `변경이 제안되지 않았습니다. 관계 리비전이 ${r}에서 ${a}(으)로 바뀌었습니다. 입력 내용은 유지했습니다. 현재 관계를 확인한 뒤 리비전 ${a} 기준으로 바꾸세요.`,
      `لم يُقترح التغيير. تغيّرت مراجعة العلاقات من ${r} إلى ${a}. احتُفظ بإدخالاتك. راجِع العلاقات الحالية ثم اتخذ المراجعة ${a} أساسًا للتغيير.`),
  useBase: (l: Locale, a: string) => t(l, `Base on revision ${a}`, `리비전 ${a} 기준으로 하기`, `اتخاذ المراجعة ${a} أساسًا`),
  baseAdopted: (l: Locale, a: number) => t(l, `The change is now based on relationship revision ${a}.`, `이제 관계 리비전 ${a}이(가) 기준입니다.`, `يستند التغيير الآن إلى مراجعة العلاقات ${a}.`),
  baseFact: (l: Locale) => t(l, "Based on relationship revision", "기준 관계 리비전", "مراجعة العلاقات المتخذة أساسًا"),
  baseActive: (l: Locale) => t(l, "active", "활성", "نشطة"),
  baseNotActive: (l: Locale, a: number) => t(l, `not active; revision ${a} is active`, `활성 아님(활성 리비전 ${a})`, `غير نشطة؛ المراجعة النشطة ${a}`),
  eBase: (l: Locale, r: number, a: number) =>
    t(l, `This change is based on revision\u00a0${iso(String(r))}, but revision\u00a0${iso(String(a))} is active. Choose "Base on revision\u00a0${iso(String(a))}" first.`,
      `이 변경의 기준은 리비전\u00a0${iso(String(r))}\u2060이지만 활성 리비전은\u00a0${iso(String(a))}\u2060입니다. 먼저 "리비전\u00a0${iso(String(a))} 기준으로 하기"를 고르세요.`,
      `يستند هذا التغيير إلى المراجعة\u00a0${iso(String(r))} بينما المراجعة النشطة\u00a0${iso(String(a))}. اختر أولًا «اتخاذ المراجعة\u00a0${iso(String(a))} أساسًا».`),
  fromPassed: (l: Locale, d: string) =>
    t(l, `The effective date in the link, ${d}, has passed, so it was not filled in. Choose today or a later date (UTC).`, `링크의 적용 시작일(${d})이 지나 입력하지 않았습니다. 오늘(UTC) 이후 날짜를 고르세요.`, `تاريخ السريان الوارد في الرابط (${d}) قد مضى، لذا لم يُملأ. اختر تاريخ اليوم أو تاريخًا لاحقًا (UTC).`),
  linkDropped: (l: Locale) => t(l, "A value in the link was not valid and was removed.", "링크에서 올바르지 않은 값을 제거했습니다.", "أُزيلت من الرابط قيمة غير صالحة."),
  groupTitle: (l: Locale) => t(l, "Change plan for a view group draft", "조회 그룹 초안의 변경 계획", "خطة تغيير لمسودة مجموعة عرض"),
  draftEffect: (l: Locale, n: number) =>
    t(l, `Proposing creates one change plan that, if activated, adds all ${n} members to this group from the effective date. If the plan passes its checks, activation is held: the impact of this change on amounts, publication and ERP has not been calculated.`,
      `제안하면 구성원 ${n}개 전체를 적용 시작일부터 이 그룹에 추가하는 변경 계획 하나가 만들어지며, 구성원은 계획이 활성화될 때 추가됩니다. 검사를 통과해도 활성화 보류: 이 변경의 금액·공개·ERP 영향이 계산되지 않았습니다.`,
      `يُنشئ الاقتراح خطة تغيير واحدة تضيف، إذا فُعّلت، كل الأعضاء (${n}) إلى هذه المجموعة اعتبارًا من تاريخ السريان. وإذا اجتازت الخطة الفحص، يبقى التفعيل معلَّقًا: لم يُحتسب أثر هذا التغيير على المبالغ والنشر وERP.`),
  membersFirst: (l: Locale, m: number, n: number) => t(l, `Members (first ${m} of ${n})`, `구성원(${n}개 중 처음 ${m}개)`, `الأعضاء (أول ${m} من ${n})`),
  draftNameFact: (l: Locale) => t(l, "Draft name", "초안 이름", "اسم المسودة"),
  draftName: (l: Locale) => t(l, "Draft name:", "초안 이름:", "اسم المسودة:"),
  groupFact: (l: Locale) => t(l, "Group", "그룹", "المجموعة"),
  newGroup: (l: Locale) => t(l, "New group", "새 그룹", "مجموعة جديدة"),
  membersFact: (l: Locale) => t(l, "Members", "구성원", "الأعضاء"),
  groupLead: (l: Locale, group: string, n: number | string, d: string) =>
    t(l, `Join view group ${iso(group)}: ${n} members from ${d}`, `조회 그룹 ${iso(group)}에 추가: ${d}부터 구성원 ${n}개`, `الانضمام إلى مجموعة العرض ${iso(group)}: عدد الأعضاء ${n}، اعتبارًا من ${d}`),
  planLimit: (l: Locale) =>
    t(l, "This change is too large for one plan. Remove members from the draft and propose again.", "한 계획에 담기에는 변경이 너무 큽니다. 초안에서 구성원을 줄인 뒤 다시 제안하세요.", "هذا التغيير أكبر من أن تتسع له خطة واحدة. أزِل أعضاء من المسودة ثم اقترح مجددًا."),
  draftLoading: (l: Locale) => t(l, "Loading the view group draft…", "조회 그룹 초안을 불러오는 중입니다…", "جارٍ تحميل مسودة مجموعة العرض…"),
  draftMissing: (l: Locale) => t(l, "This view group draft no longer exists, so no change plan can be created from it.", "이 조회 그룹 초안은 더 이상 없어 변경 계획을 만들 수 없습니다.", "لم تعد مسودة مجموعة العرض هذه موجودة، لذا لا يمكن إنشاء خطة تغيير منها."),
  draftNotOpen: (l: Locale) => t(l, "This view group draft is not open, so no change plan can be created from it. Open it in Organizations and relationships to see its state.", "이 조회 그룹 초안은 열린 상태가 아니어서 변경 계획을 만들 수 없습니다. 조직·관계 탐색에서 상태를 확인하세요.", "مسودة مجموعة العرض هذه ليست مفتوحة، لذا لا يمكن إنشاء خطة تغيير منها. افتحها في شاشة «المؤسسات والعلاقات» لمعرفة حالتها."),
  draftFailed: (l: Locale) => t(l, "The view group draft could not be loaded.", "조회 그룹 초안을 불러오지 못했습니다.", "تعذر تحميل مسودة مجموعة العرض."),
  draftRefused: (l: Locale) =>
    t(l, "The change plan was not created: this draft was not accepted as it is. A member or the group may have changed in the active relationships. Check the draft in Organizations and relationships.",
      "변경 계획을 만들지 않았습니다. 이 초안을 지금 상태로는 받을 수 없습니다. 활성 관계에서 구성원이나 그룹이 바뀌었을 수 있습니다. 조직·관계 탐색에서 초안을 확인하세요.",
      "لم تُنشأ خطة التغيير: لم تُقبل هذه المسودة كما هي. ربما تغيّر عضو أو المجموعة في العلاقات النشطة. راجع المسودة في شاشة «المؤسسات والعلاقات»."),
  notOwnership: (l: Locale) =>
    t(l, "A view group is not ownership. Creating or editing a draft grants no access and changes no relationship. Members join only when a change plan is activated.",
      "조회 그룹은 소유권이 아닙니다. 초안을 만들거나 고쳐도 접근 권한이 부여되지 않고 관계도 바뀌지 않습니다. 변경 계획이 활성화되어야 구성원이 추가됩니다.",
      "مجموعة العرض ليست ملكية. إنشاء المسودة أو تعديلها لا يمنح أي صلاحية وصول ولا يغيّر أي علاقة. ينضم الأعضاء فقط عند تفعيل خطة تغيير."),
  deniedTitle: (l: Locale) => t(l, "Relationship changes are not available to you", "관계 변경을 사용할 수 없습니다", "تغييرات العلاقات غير متاحة لك"),
};
