import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  blockerWords, callsSourceWords, connectionWords, domainStatusWords, domainWords, eventKindWords, externalStatusWords, fencedWords, installStateWords, keptWords, migrationWords,
  modeWords, narrowWords, outcomeWords, phaseWords, proposalKindWords, proposalStateWords, publicLockWords, recoveryWords, resultWords, shownWords, unknownCode, words, writerKindWords,
} from "./live-codes";

export const iso = (s: string) => (s ? "⁨" + s + "⁩" : s);
export const mono = (s: string) => (s ? "\uE000" + s + "\uE001" : s);
export const plain = (s: string) => s.replace(/[\uE000\uE001]/g, "");
const has = (table: object, key: string) => Object.prototype.hasOwnProperty.call(table, key);

export const k = {
  deniedTitle: (l: Locale) => t(l, "Recovery is not available to you", "복구 화면을 볼 수 없습니다", "الاسترداد غير متاح لك"),
  denied: (l: Locale) =>
    t(l, "You do not have permission to manage recovery in this installation.", "이 설치에서 복구를 관리할 권한이 없습니다.", "ليست لديك صلاحية إدارة الاسترداد في هذا التثبيت."),
  sessionEnded: (l: Locale) => t(l, "Your session has ended.", "세션이 끝났습니다.", "انتهت جلستك."),
  signInAgain: (l: Locale) => t(l, "Sign in again", "다시 로그인", "تسجيل الدخول مجددًا"),
  loading: (l: Locale) => t(l, "Loading recovery state…", "복구 상태를 불러오는 중…", "جارٍ تحميل حالة الاسترداد…"),
  reload: (l: Locale) => t(l, "Reload", "새로 고침", "إعادة التحميل"),
  workspace: (l: Locale) => t(l, "Workspace ID", "업무 공간 ID", "معرّف مساحة العمل"),
  revision: (l: Locale) => t(l, "Data revision", "데이터 리비전", "مراجعة البيانات"),
  now: (l: Locale) => t(l, "Now (UTC)", "현재(UTC)", "الآن (UTC)"),
  viewTitle: (l: Locale, v: string) =>
    v === "isolation" ? t(l, "Isolation check", "격리 검사", "فحص العزل")
      : v === "reconciliation" ? t(l, "External reconciliation", "외부 대사", "المطابقة الخارجية")
        : v === "migration" ? t(l, "Customer migration", "고객 이관", "ترحيل العملاء")
          : v === "resume" ? t(l, "Resume decision", "재개 판정", "قرار الاستئناف")
            : t(l, "Restore sets", "복원 세트", "مجموعات الاستعادة"),
  statementLead: (l: Locale) => t(l, "Records decisions only.", "결정만 기록합니다.", "تسجيل القرارات فقط."),
  statement: (l: Locale) =>
    t(
      l,
      "Nothing is restored, pruned or deleted, and no host, network or credential is changed from this screen in this build.",
      "이 빌드에서는 이 화면에서 아무것도 복원·정리·삭제하지 않으며 호스트·네트워크·자격 증명도 변경하지 않습니다.",
      "لا تتم من هذه الشاشة في هذا الإصدار أي استعادة أو تقليم أو حذف، ولا يُغيَّر أي مضيف أو شبكة أو بيانات اعتماد.",
    ),
  permission: (l: Locale) =>
    t(
      l,
      "Every command here currently needs installation administration. Separate recovery verification, recovery execution, customer export, ERP reconciliation and resume permissions are not available yet, and auditors and billing operators cannot open this screen yet.",
      "현재 이 화면의 모든 명령에는 설치 관리 권한이 필요합니다. 복구 검증·복구 실행·고객 범위 반출·ERP 대사·재개 권한은 아직 별도로 제공되지 않으며, 감사 조회자와 청구 운영자는 아직 이 화면을 열 수 없습니다.",
      "تتطلب كل الأوامر هنا حاليًا صلاحية إدارة التثبيت. صلاحيات التحقق من الاسترداد وتنفيذه وتصدير نطاق العملاء ومطابقة ERP والاستئناف غير متاحة بشكل منفصل بعد، ولا يستطيع المدققون ومشغلو الفوترة فتح هذه الشاشة بعد.",
    ),
  notConfigured: (l: Locale) => t(l, "Recovery is not configured in this workspace", "이 업무 공간에는 복구가 설정되어 있지 않습니다", "الاسترداد غير مُعدّ في مساحة العمل هذه"),
  stateUnknown: (l: Locale, code: string) =>
    t(l, `The recovery read returned an unknown state (${mono(code)}).`, `복구 조회가 알 수 없는 상태(${mono(code)})를 반환했습니다.`, `أعادت قراءة الاسترداد حالة غير معروفة (${mono(code)}).`),
  readOnly: (l: Locale) =>
    t(l, "Commands are unavailable because the data revision is unknown. Reload.", "데이터 리비전을 알 수 없어 명령을 사용할 수 없습니다. 새로 고침하세요.", "الأوامر غير متاحة لأن مراجعة البيانات غير معروفة. أعد التحميل."),
  unknown: (l: Locale) => t(l, "Unknown", "알 수 없음", "غير معروف"),
  unknownCode,
  none: (l: Locale) => t(l, "None", "없음", "لا شيء"),
  dash: () => "—",
  vanished: (l: Locale) =>
    t(l, "This item is no longer listed. Your earlier request may have been applied.", "이 항목은 더 이상 목록에 없습니다. 이전 요청이 이미 반영되었을 수 있습니다.", "لم يعد هذا العنصر مدرجًا. ربما طُبّق طلبك السابق."),
  vanishedProposal: (l: Locale) =>
    t(l, "Your earlier request may have been applied. The proposal is no longer listed.", "이전 요청이 이미 반영되었을 수 있습니다. 제안이 더 이상 목록에 없습니다.", "ربما طُبّق طلبك السابق. لم يعد المقترح مدرجًا."),
  dismissNotice: (l: Locale) => t(l, "Dismiss", "닫기", "إغلاق"),
  mayApplied: (l: Locale) =>
    t(l, "Your earlier request may have been applied. Check this proposal's state.", "이전 요청이 이미 반영되었을 수 있습니다. 이 제안의 상태를 확인하세요.", "ربما طُبّق طلبك السابق. تحقّق من حالة هذا المقترح."),

  setsLead: (l: Locale) =>
    t(
      l,
      "Nothing is restored here. Restore sets are read from the recovery registry. Registering a restore set, requesting an isolated restore, abandoning a recovery and pruning decision events are not connected in this build.",
      "여기에서는 아무것도 복원하지 않습니다. 복원 세트는 복구 레지스트리에서 읽습니다. 이 빌드에서는 복원 세트 등록, 격리 복원 요청, 복구 중단, 결정 기록 정리가 연결되어 있지 않습니다.",
      "لا تتم أي استعادة هنا. تُقرأ مجموعات الاستعادة من سجل الاسترداد. تسجيل مجموعة استعادة وطلب استعادة معزولة والتخلي عن الاسترداد وتقليم أحداث القرار غير متصلة في هذا الإصدار.",
    ),
  isolationLead: (l: Locale) =>
    t(
      l,
      "Locking records the installation as locked in the recovery registry and moves its generation, so the registry refuses write permits and calls under the old generation. It does not stop a host, cut a network or revoke a credential. No writer is connected to the registry in this build.",
      "잠그면 복구 레지스트리에 설치가 잠김으로 기록되고 generation이 올라가, 레지스트리가 이전 generation의 쓰기 허가와 호출을 거부합니다. 호스트를 중지하거나 네트워크를 끊거나 자격 증명을 회수하지는 않습니다. 이 빌드에서는 레지스트리에 연결된 쓰기 주체가 없습니다.",
      "يسجّل القفل التثبيت كمقفل في سجل الاسترداد ويرفع رقم الجيل، فيرفض السجل أذونات الكتابة والاستدعاءات تحت الجيل السابق. لا يوقف أي مضيف ولا يقطع أي شبكة ولا يلغي أي بيانات اعتماد. لا توجد جهة كتابة متصلة بالسجل في هذا الإصدار.",
    ),
  reconciliationLead: (l: Locale) =>
    t(
      l,
      "External results are not checked from this screen. The reconciler is not connected in this build, so unknown targets stay unknown and fenced, and nothing is resubmitted.",
      "이 화면에서는 외부 결과를 확인하지 않습니다. 이 빌드에서는 대사 기능이 연결되어 있지 않아 미확인 대상은 미확인·차단 상태로 남고 아무것도 다시 제출하지 않습니다.",
      "لا يتم التحقق من النتائج الخارجية من هذه الشاشة. أداة المطابقة غير متصلة في هذا الإصدار، لذلك تبقى الأهداف غير المعروفة غير معروفة ومحجوبة، ولا يُعاد إرسال أي شيء.",
    ),
  migrationLead: (l: Locale) =>
    t(
      l,
      "Customer migration is not connected in this build. No migration package is prepared, no legacy document is imported, no opening balance is recorded and no customer data leaves the platform.",
      "이 빌드에서는 고객 이관이 연결되어 있지 않습니다. 이관 패키지를 준비하지 않고, 이전 문서를 반입하지 않으며, 기초 잔액을 기록하지 않고, 고객 데이터가 플랫폼 밖으로 나가지 않습니다.",
      "ترحيل العملاء غير متصل في هذا الإصدار. لا تُجهَّز أي حزمة ترحيل، ولا يُستورد أي مستند سابق، ولا يُسجَّل أي رصيد افتتاحي، ولا تغادر أي بيانات عملاء المنصة.",
    ),
  resumeLead: (l: Locale) =>
    t(
      l,
      "Approving an unlock records the decision in the recovery registry and returns the installation to serving. Nothing is restored, pruned or deleted, and no host changes.",
      "잠금 해제를 승인하면 복구 레지스트리에 결정이 기록되고 설치가 운영 상태로 돌아갑니다. 아무것도 복원·정리·삭제하지 않으며 호스트도 변경하지 않습니다.",
      "تسجّل الموافقة على إلغاء القفل القرار في سجل الاسترداد وتعيد التثبيت إلى التشغيل. لا تتم أي استعادة أو تقليم أو حذف، ولا يتغير أي مضيف.",
    ),
  resumeLeadStrong: (l: Locale) =>
    t(
      l,
      "A lock release checks no decision watermark: it checks only the lock, the generation and open calls.",
      "잠금 해제는 결정 워터마크를 확인하지 않고 잠금·generation·열린 호출만 확인합니다.",
      "رفع القفل لا يتحقق من علامة القرار المائية: يتحقق فقط من القفل والجيل والاستدعاءات المفتوحة.",
    ),
  reconciliationEmpty: (l: Locale) =>
    t(l, "No recovery is open, so there is nothing to reconcile.", "진행 중인 복구가 없어 대사할 대상이 없습니다.", "لا يوجد استرداد مفتوح، لذا لا يوجد ما تتم مطابقته."),

  colInstallation: (l: Locale) => t(l, "Installation", "설치", "التثبيت"),
  colRestore: (l: Locale) => t(l, "Restore ID", "복원 ID", "معرّف الاستعادة"),
  colAsOf: (l: Locale) => t(l, "As of", "기준 시각", "بتاريخ"),
  colGeneration: (l: Locale) => t(l, "Generation", "Generation", "الجيل"),
  colEvidence: (l: Locale) => t(l, "Evidence check", "증빙 검사", "فحص الأدلة"),
  colWatermark: (l: Locale) => t(l, "Decision watermark", "결정 워터마크", "علامة القرار المائية"),
  colExternal: (l: Locale) => t(l, "External reconciliation", "외부 대사", "المطابقة الخارجية"),
  colLock: (l: Locale) => t(l, "Public lock", "공개 잠금", "القفل العام"),
  installationGen: (l: Locale) => t(l, "Installation", "설치", "التثبيت"),
  pointGen: (l: Locale) => t(l, "Restore point", "복원 지점", "نقطة الاستعادة"),
  noPoint: (l: Locale) => t(l, "No restore point", "복원 지점 없음", "لا توجد نقطة استعادة"),
  verifiedOf: (l: Locale, v: number, n: number) =>
    t(l, `${v} of ${n} domains verified`, `${n}개 영역 중 ${v}개 검증됨`, `تم التحقق من ${v} من ${n} مجالات`),
  mismatched: (l: Locale, n: number) => t(l, `${n} mismatched`, `${n}개 불일치`, `${n} غير متطابقة`),
  notChecked: (l: Locale, n: number) => t(l, `${n} not checked yet`, `${n}개 검증 전`, `${n} لم يُتحقق منها بعد`),
  eventsPending: (l: Locale, n: number) =>
    t(l, `${n} decision events not yet reapplied`, `재적용되지 않은 결정 기록 ${n}건`, `${n} من أحداث القرار لم يُعد تطبيقها بعد`),
  noEventsPending: (l: Locale) => t(l, "No decision events pending", "대기 중인 결정 기록 없음", "لا توجد أحداث قرار معلّقة"),
  declaredLost: (l: Locale, n: number) => t(l, `${n} declared lost`, `유실 선언 ${n}건`, `${n} مُعلنة مفقودة`),
  lostUnknown: (l: Locale) => t(l, "Lost sequences: Unknown", "유실 순번: 알 수 없음", "التسلسلات المفقودة: غير معروف"),
  outsideRecovery: (l: Locale) => t(l, "Not checked outside a recovery", "복구 중이 아니면 확인하지 않음", "لا يُتحقق منها خارج الاسترداد"),
  reconciledOf: (l: Locale, r: number, n: number) =>
    t(l, `${r} of ${n} external systems reconciled`, `외부 시스템 ${n}개 중 ${r}개 대사 완료`, `تمت مطابقة ${r} من ${n} أنظمة خارجية`),
  notApplicable: (l: Locale) => t(l, "Not applicable (no recovery open)", "해당 없음(진행 중인 복구 없음)", "لا ينطبق (لا يوجد استرداد مفتوح)"),
  jumpsLabel: (l: Locale, id: string) => t(l, `Views for ${iso(id)}`, `${iso(id)} 관련 보기`, `طرق العرض لـ ${iso(id)}`),
  notConnectedLine: (l: Locale) => t(l, "Isolated restore request", "격리 복원 요청", "طلب استعادة معزولة"),
  abandonLine: (l: Locale) => t(l, "Abandon recovery", "복구 중단", "التخلي عن الاسترداد"),
  pointFold: (l: Locale, id: string) =>
    t(l, `Restore point and domain checks for ${mono(id)}`, `${mono(id)}의 복원 지점과 영역 검사`, `نقطة الاستعادة وفحوص المجالات لـ ${mono(id)}`),
  phase: (l: Locale) => t(l, "Phase", "단계", "المرحلة"),
  backup: (l: Locale) => t(l, "Backup reference", "백업 참조", "مرجع النسخة الاحتياطية"),
  manifest: (l: Locale) => t(l, "Manifest digest", "manifest 요약값", "ملخص البيان"),
  noChecksums: (l: Locale) => t(l, "Chunk checksums and item lists are not shown.", "청크 체크섬과 항목 목록은 표시하지 않습니다.", "لا تُعرض المجاميع الاختبارية للأجزاء ولا قوائم العناصر."),
  domainChecks: (l: Locale) => t(l, "Domain checks", "영역 검사", "فحوص المجالات"),
  colDomain: (l: Locale) => t(l, "Domain", "영역", "المجال"),
  colStatus: (l: Locale) => t(l, "Status", "상태", "الحالة"),
  colChunks: (l: Locale) => t(l, "Mismatched chunks", "불일치 청크", "الأجزاء غير المتطابقة"),
  chunkOne: (l: Locale) => t(l, "Chunk index {i} does not match", "청크 인덱스 {i} 불일치", "فهرس الجزء {i} غير مطابق"),
  chunkMany: (l: Locale) => t(l, "Chunk indices {i} do not match", "청크 인덱스 {i} 불일치", "فهارس الأجزاء {i} غير مطابقة"),
  chunkFirst100: (l: Locale) => t(l, "The first 100 mismatched chunk indices: {i}", "불일치 청크 인덱스 처음 100개: {i}", "أول 100 من فهارس الأجزاء غير المتطابقة: {i}"),
  countDiffers: (l: Locale) =>
    t(l, "Mismatch; no chunk index was reported (the item count differs).", "불일치. 보고된 청크 인덱스가 없습니다(항목 수가 다름).", "عدم تطابق؛ لم يُبلَّغ عن أي فهرس جزء (عدد العناصر مختلف)."),
  historyTitle: (l: Locale) => t(l, "Recovery history", "복구 이력", "سجل الاسترداد"),
  colRecovery: (l: Locale) => t(l, "Recovery", "복구", "الاسترداد"),
  colOutcome: (l: Locale) => t(l, "Outcome", "결과", "النتيجة"),
  colLost: (l: Locale) => t(l, "Lost sequences", "유실 순번", "التسلسلات المفقودة"),
  noHistory: (l: Locale) => t(l, "No completed or abandoned recoveries are kept.", "보관된 완료·중단 복구가 없습니다.", "لا توجد عمليات استرداد مكتملة أو متروكة محفوظة."),
  historyMore: (l: Locale, total: number) =>
    t(l, `Showing the newest 50 of ${total} kept.`, `보관된 ${total}건 중 최근 50건을 표시합니다.`, `تُعرض أحدث 50 من ${total} محفوظة.`),

  colWriters: (l: Locale) => t(l, "Writers", "쓰기 주체", "جهات الكتابة"),
  colVerifier: (l: Locale) => t(l, "Verifier", "검증 주체", "جهة التحقق"),
  stoppedAt: (l: Locale) => t(l, "Stopped at", "중지 시각", "توقف في"),
  notStopped: (l: Locale) => t(l, "Not stopped", "중지되지 않음", "لم يتوقف"),
  noWriters: (l: Locale) => t(l, "No writers registered", "등록된 쓰기 주체 없음", "لا توجد جهات كتابة مسجّلة"),
  noWriterRemoval: (l: Locale) => t(l, "Removing a writer is not available here.", "여기에서는 쓰기 주체를 제거할 수 없습니다.", "إزالة جهة كتابة غير متاحة هنا."),
  noVerifier: (l: Locale) => t(l, "No verifier registered", "등록된 검증 주체 없음", "لا توجد جهة تحقق مسجّلة"),
  permits: (l: Locale) => t(l, "Write permits", "쓰기 허가", "أذونات الكتابة"),
  permitsNever: (l: Locale) => t(l, "Write permits are never shown on this screen.", "이 화면에는 쓰기 허가를 표시하지 않습니다.", "لا تُعرض أذونات الكتابة على هذه الشاشة أبدًا."),
  openCalls: (l: Locale) => t(l, "Open calls", "열린 호출", "الاستدعاءات المفتوحة"),
  colCall: (l: Locale) => t(l, "Call", "호출", "الاستدعاء"),
  colWriter: (l: Locale) => t(l, "Writer", "쓰기 주체", "جهة الكتابة"),
  colDeadline: (l: Locale) => t(l, "Deadline", "마감 시각", "الموعد النهائي"),
  colTarget: (l: Locale) => t(l, "Target", "대상", "الهدف"),
  noCalls: (l: Locale) => t(l, "No open calls in this workspace.", "이 업무 공간에 열린 호출이 없습니다.", "لا توجد استدعاءات مفتوحة في مساحة العمل هذه."),
  callsNotShown: (l: Locale) =>
    t(l, "Open calls are listed only while a recovery is open.", "열린 호출은 복구가 진행 중일 때만 표시합니다.", "تُدرج الاستدعاءات المفتوحة فقط أثناء وجود استرداد مفتوح."),
  showingOf: (l: Locale, total: number) => t(l, `Showing 50 of ${total}.`, `${total}건 중 50건을 표시합니다.`, `تُعرض 50 من ${total}.`),
  otherScope: (l: Locale) =>
    t(l, "Calls in other workspaces also exist. They are not shown.", "다른 업무 공간의 호출도 있습니다. 표시하지 않습니다.", "توجد استدعاءات في مساحات عمل أخرى أيضًا. لا تُعرض."),
  otherScopeUnknown: (l: Locale) => t(l, "Calls in other workspaces: Unknown", "다른 업무 공간의 호출: 알 수 없음", "الاستدعاءات في مساحات العمل الأخرى: غير معروف"),
  otherTargets: (l: Locale) =>
    t(l, "Unknown targets in other workspaces also exist. They are not shown.", "다른 업무 공간의 미확인 대상도 있습니다. 표시하지 않습니다.", "توجد أهداف غير معروفة في مساحات عمل أخرى أيضًا. لا تُعرض."),
  otherTargetsUnknown: (l: Locale) => t(l, "Unknown targets in other workspaces: Unknown", "다른 업무 공간의 미확인 대상: 알 수 없음", "الأهداف غير المعروفة في مساحات العمل الأخرى: غير معروف"),
  deadlineNote: (l: Locale) =>
    t(
      l,
      "A call past its deadline is closed as unknown the next time the registry is read. This does not change the data revision.",
      "마감 시각이 지난 호출은 다음에 레지스트리를 읽을 때 결과 미확인으로 닫힙니다. 이때 데이터 리비전은 바뀌지 않습니다.",
      "يُغلق الاستدعاء الذي تجاوز موعده النهائي كغير معروف عند قراءة السجل في المرة التالية. لا يغيّر ذلك مراجعة البيانات.",
    ),
  lockShown: (l: Locale, n: number, m: number) => t(l, `Showing the newest ${n} of ${m} locks.`, `잠금 ${m}건 중 최근 ${n}건을 표시합니다.`, `تُعرض أحدث ${n} من أصل ${m}.`),
  lockAll: (l: Locale, m: number) => t(l, `Showing all ${m} locks, newest first.`, `잠금 ${m}건을 모두 최신순으로 표시합니다.`, `تُعرض جميع الأقفال (${m})، الأحدث أولًا.`),
  lockMore: (l: Locale) => t(l, "Load older locks", "이전 잠금 불러오기", "تحميل أقفال أقدم"),
  lockLoading: (l: Locale) => t(l, "Loading…", "불러오는 중…", "جارٍ التحميل…"),
  lockFailed: (l: Locale) => t(l, "Older locks could not be loaded. The locks already shown are unchanged.", "이전 잠금을 불러오지 못했습니다. 이미 표시된 잠금은 그대로입니다.", "تعذّر تحميل الأقفال الأقدم. لم تتغيّر الأقفال المعروضة."),
  lockTryAgain: (l: Locale) => t(l, "Try again", "다시 시도", "إعادة المحاولة"),
  lockReset: (l: Locale) => t(l, "The log was refreshed and starts again from the newest lock.", "잠금 기록을 새로 고쳐 최신 잠금부터 다시 표시합니다.", "حُدِّث سجل القفل ويبدأ من جديد بأحدث قفل."),
  lockLog: (l: Locale) => t(l, "Lock log", "잠금 기록", "سجل القفل"),
  colTime: (l: Locale) => t(l, "Time", "시각", "الوقت"),
  colAdmin: (l: Locale) => t(l, "Administrator", "관리자", "المسؤول"),
  colReason: (l: Locale) => t(l, "Reason", "사유", "السبب"),
  noLocks: (l: Locale) => t(l, "No locks recorded.", "기록된 잠금이 없습니다.", "لا توجد أقفال مسجّلة."),

  externalsTitle: (l: Locale) => t(l, "External systems", "외부 시스템", "الأنظمة الخارجية"),
  colSystem: (l: Locale) => t(l, "System", "시스템", "النظام"),
  colKind: (l: Locale) => t(l, "Kind", "종류", "النوع"),
  noExternals: (l: Locale) => t(l, "No external systems registered.", "등록된 외부 시스템이 없습니다.", "لا توجد أنظمة خارجية مسجّلة."),
  targetsTitle: (l: Locale) => t(l, "Unknown targets", "미확인 대상", "الأهداف غير المعروفة"),
  colKey: (l: Locale) => t(l, "Key", "키", "المفتاح"),
  colSeries: (l: Locale) => t(l, "Series", "번호 계열", "السلسلة"),
  colNumber: (l: Locale) => t(l, "Number", "번호", "الرقم"),
  colFenced: (l: Locale) => t(l, "Fenced", "차단", "الحجب"),
  colHighWater: (l: Locale) => t(l, "High water", "최고 번호", "أعلى رقم"),
  noSeries: (l: Locale) => t(l, "No series", "번호 계열 없음", "لا سلسلة"),
  noTargets: (l: Locale) => t(l, "No unknown targets in this workspace.", "이 업무 공간에 미확인 대상이 없습니다.", "لا توجد أهداف غير معروفة في مساحة العمل هذه."),
  skipped: (l: Locale) => t(l, "Skipped", "건너뜀", "المتخطّاة"),
  reconciler: (l: Locale) => t(l, "Reconciler", "대사 기능", "أداة المطابقة"),
  reconcilerNote: (l: Locale) =>
    t(
      l,
      "An unknown target is resolved only by the reconciler from the original submission. This screen records no reconciliation.",
      "미확인 대상은 대사 기능이 원래 제출 기록을 기준으로만 해소합니다. 이 화면은 대사를 기록하지 않습니다.",
      "لا يحلّ الهدفَ غير المعروف إلا أداةُ المطابقة انطلاقًا من الإرسال الأصلي. لا تسجّل هذه الشاشة أي مطابقة.",
    ),

  laterTitle: (l: Locale) => t(l, "Modules to connect later", "나중에 연결할 모듈", "وحدات تُوصل لاحقًا"),

  mode: (l: Locale) => t(l, "Mode", "모드", "الوضع"),
  lostSequences: (l: Locale) => t(l, "Lost sequences", "유실 순번", "التسلسلات المفقودة"),
  andMore: (l: Locale, n: number) => t(l, ` and ${n} more`, ` 외 ${n}건`, ` و${n} أخرى`),
  modeNone: (l: Locale) =>
    t(l, "This installation is serving; there is nothing to resume or release.", "이 설치는 운영 중이므로 재개하거나 해제할 대상이 없습니다.", "هذا التثبيت قيد التشغيل؛ لا يوجد ما يُستأنف أو يُرفع."),
  modeRelease: (l: Locale) => t(l, "This installation is locked without a recovery.", "이 설치는 복구 없이 잠겨 있습니다.", "هذا التثبيت مقفل دون استرداد."),
  modeReleaseStrong: (l: Locale) =>
    t(
      l,
      "Releasing the lock does not verify any restore, decision watermark or external result.",
      "잠금을 해제해도 복원·결정 워터마크·외부 결과는 검증되지 않습니다.",
      "رفع القفل لا يتحقق من أي استعادة أو علامة قرار مائية أو نتيجة خارجية.",
    ),
  pendingEvents: (l: Locale) => t(l, "Pending decision events", "대기 중인 결정 기록", "أحداث القرار المعلّقة"),
  colSequence: (l: Locale) => t(l, "Sequence", "순번", "التسلسل"),
  colRecorded: (l: Locale) => t(l, "Recorded", "기록 시각", "وقت التسجيل"),
  noEventKeys: (l: Locale) => t(l, "Decision event keys are not shown.", "결정 기록 키는 표시하지 않습니다.", "لا تُعرض مفاتيح أحداث القرار."),
  noPendingEvents: (l: Locale) => t(l, "No pending decision events.", "대기 중인 결정 기록이 없습니다.", "لا توجد أحداث قرار معلّقة."),
  digest: (l: Locale) => t(l, "Confirmation digest", "확인 요약값", "ملخص التأكيد"),
  goToProposal: (l: Locale, n: string) => t(l, `Go to proposal #${n}`, `제안 #${n}(으)로 이동`, `الانتقال إلى المقترح #${n}`),

  proposalsTitle: (l: Locale) => t(l, "Recovery proposals", "복구 제안", "مقترحات الاسترداد"),
  noProposals: (l: Locale) => t(l, "No recovery proposals.", "복구 제안이 없습니다.", "لا توجد مقترحات استرداد."),
  waitingGroup: (l: Locale, n: number) => t(l, `Waiting for approval (${n})`, `승인 대기(${n})`, `بانتظار الاعتماد (${n})`),
  closedGroup: (l: Locale, n: number) => t(l, `Closed (${n})`, `종료됨(${n})`, `مغلقة (${n})`),
  appliedNote: (l: Locale) => t(l, "Applied means recorded in the recovery registry.", "적용됨은 복구 레지스트리에 기록되었다는 뜻입니다.", "«مطبّق» تعني أنه سُجّل في سجل الاسترداد."),
  proposalN: (l: Locale, n: string) => t(l, `Proposal ${n}`, `제안 ${n}`, `المقترح ${n}`),
  proposedBy: (l: Locale, who: string) => t(l, `Proposed by ${iso(who)}`, `제안자 ${iso(who)}`, `اقترحه ${iso(who)}`),
  created: (l: Locale, at: string) => t(l, `Created ${iso(at)}`, `생성 ${iso(at)}`, `أُنشئ ${iso(at)}`),
  reason: (l: Locale) => t(l, "Reason", "사유", "السبب"),
  noReason: (l: Locale) => t(l, "No reason given", "사유 없음", "لم يُذكر سبب"),
  installLink: (l: Locale) => t(l, "Installation in resume decision:", "재개 판정의 설치:", "التثبيت في قرار الاستئناف:"),
  approvedBy: (l: Locale, who: string) => t(l, `Approved by ${iso(who)}`, `승인자 ${iso(who)}`, `اعتمده ${iso(who)}`),
  withdrawnBy: (l: Locale, who: string) => t(l, `Withdrawn by ${iso(who)}`, `철회자 ${iso(who)}`, `سحبه ${iso(who)}`),
  retiredRow: (l: Locale) =>
    t(l, "Retired: the recovery registry no longer holds this proposal", "폐기됨: 복구 레지스트리에 이 제안이 더 이상 없습니다", "مُسقَط: لم يعد سجل الاسترداد يحتفظ بهذا المقترح"),
  closingReason: (l: Locale) => t(l, "Closing reason", "종료 사유", "سبب الإغلاق"),
  skew: (l: Locale, n: number) => t(l, `${n} s`, `${n}초`, `${n} ث`),

  approve: (l: Locale) => t(l, "Approve", "승인", "اعتماد"),
  withdraw: (l: Locale) => t(l, "Withdraw", "철회", "سحب"),
  lock: (l: Locale) => t(l, "Lock", "잠금", "قفل"),
  proposeUnlock: (l: Locale) => t(l, "Propose unlock", "잠금 해제 제안", "اقتراح إلغاء القفل"),
  lockVerb: (l: Locale, id: string) => t(l, `Lock ${mono(id)}`, `${mono(id)} 잠금`, `قفل ${mono(id)}`),
  proposeVerb: (l: Locale, id: string) => t(l, `Propose unlock of ${mono(id)}`, `${mono(id)} 잠금 해제 제안`, `اقتراح إلغاء قفل ${mono(id)}`),
  approveVerb: (l: Locale, n: string) => t(l, `Approve proposal ${n}`, `제안 ${n} 승인`, `اعتماد المقترح ${n}`),
  withdrawVerb: (l: Locale, n: string) => t(l, `Withdraw proposal ${n}`, `제안 ${n} 철회`, `سحب المقترح ${n}`),
  confirmLock: (l: Locale) => t(l, "Confirm the lock", "잠금 확인", "تأكيد القفل"),
  confirmPropose: (l: Locale) => t(l, "Confirm the unlock proposal", "잠금 해제 제안 확인", "تأكيد مقترح إلغاء القفل"),
  confirmApprove: (l: Locale) => t(l, "Confirm the approval", "승인 확인", "تأكيد الاعتماد"),
  confirmWithdraw: (l: Locale) => t(l, "Confirm the withdrawal", "철회 확인", "تأكيد السحب"),
  factInstallation: (l: Locale) => t(l, "Installation", "설치", "التثبيت"),
  factState: (l: Locale) => t(l, "State", "상태", "الحالة"),
  factProposal: (l: Locale) => t(l, "Proposal", "제안", "المقترح"),
  factKind: (l: Locale) => t(l, "Kind", "종류", "النوع"),
  factSummary: (l: Locale) => t(l, "Summary", "요약", "الملخص"),
  factProposedBy: (l: Locale) => t(l, "Proposed by", "제안자", "المقترِح"),
  factProposerReason: (l: Locale) => t(l, "Proposer's reason", "제안 사유", "سبب المقترِح"),
  factInstallState: (l: Locale) => t(l, "Installation now", "현재 설치 상태", "التثبيت الآن"),
  lockStrong: (l: Locale) =>
    t(l, "Takes effect now; no second approval: you alone lock this installation.", "지금 바로 적용되며 두 번째 승인이 없습니다. 이 설치는 귀하 혼자 잠급니다.", "يسري الآن دون موافقة ثانية: أنت وحدك تقفل هذا التثبيت."),
  lockImpact: (l: Locale, id: string, g: number | null) =>
    g === null
      ? t(
          l,
          `Locking records ${mono(id)} as locked and moves its generation, so the recovery registry refuses write permits and calls under the current generation. It does not stop a host, cut a network or revoke a credential. Releasing the lock later needs an unlock proposal and another administrator's approval.`,
          `잠그면 ${mono(id)}이(가) 잠김으로 기록되고 generation이 올라가, 복구 레지스트리가 현재 generation의 쓰기 허가와 호출을 거부합니다. 호스트를 중지하거나 네트워크를 끊거나 자격 증명을 회수하지는 않습니다. 나중에 잠금을 해제하려면 잠금 해제 제안과 다른 관리자의 승인이 필요합니다.`,
          `يسجّل القفل ${mono(id)} كمقفل ويرفع جيله، فيرفض سجل الاسترداد أذونات الكتابة والاستدعاءات تحت الجيل الحالي. لا يوقف أي مضيف ولا يقطع أي شبكة ولا يلغي أي بيانات اعتماد. يتطلب رفع القفل لاحقًا اقتراح إلغاء قفل وموافقة مسؤول آخر.`,
        )
      : t(
          l,
          `Locking records ${mono(id)} as locked and moves its generation from ${g} to ${g + 1}, so the recovery registry refuses write permits and calls under generation ${g}. It does not stop a host, cut a network or revoke a credential. Releasing the lock later needs an unlock proposal and another administrator's approval.`,
          `잠그면 ${mono(id)}이(가) 잠김으로 기록되고 generation이 ${g}에서 ${g + 1}(으)로 올라가, 복구 레지스트리가 generation ${g}의 쓰기 허가와 호출을 거부합니다. 호스트를 중지하거나 네트워크를 끊거나 자격 증명을 회수하지는 않습니다. 나중에 잠금을 해제하려면 잠금 해제 제안과 다른 관리자의 승인이 필요합니다.`,
          `يسجّل القفل ${mono(id)} كمقفل ويرفع جيله من ${g} إلى ${g + 1}، فيرفض سجل الاسترداد أذونات الكتابة والاستدعاءات تحت الجيل ${g}. لا يوقف أي مضيف ولا يقطع أي شبكة ولا يلغي أي بيانات اعتماد. يتطلب رفع القفل لاحقًا اقتراح إلغاء قفل وموافقة مسؤول آخر.`,
        ),
  beginRetired: (l: Locale, n: string) =>
    t(
      l,
      `Pending recovery-opening proposal #${n} will be retired by this lock and can no longer be approved or withdrawn.`,
      `승인 대기 중인 복구 개시 제안 #${n}은(는) 이 잠금으로 폐기되며 더 이상 승인하거나 철회할 수 없습니다.`,
      `سيُسقَط مقترح فتح الاسترداد المعلّق #${n} بهذا القفل ولن يمكن اعتماده أو سحبه بعد ذلك.`,
    ),
  proposeLead: (l: Locale) => t(l, "Another administrator must approve it.", "다른 관리자가 승인해야 합니다.", "يجب أن يعتمده مسؤول آخر."),
  releaseImpact: (l: Locale, id: string, g: number | null) =>
    (g === null
      ? t(l, `Approval releases the lock and returns ${mono(id)} to serving.`, `승인하면 잠금이 해제되고 ${mono(id)}이(가) 운영 상태로 돌아갑니다.`, `يرفع الاعتماد القفل ويعيد ${mono(id)} إلى التشغيل.`)
      : t(
          l,
          `Approval releases the lock and returns ${mono(id)} to serving at generation ${g}.`,
          `승인하면 잠금이 해제되고 ${mono(id)}이(가) generation ${g}에서 운영 상태로 돌아갑니다.`,
          `يرفع الاعتماد القفل ويعيد ${mono(id)} إلى التشغيل عند الجيل ${g}.`,
        )) +
    " " +
    t(
      l,
      "A lock release checks only the lock, the generation and open calls; it checks no decision watermark, domain verification or external reconciliation. Nothing is restored, pruned or deleted, and no host changes.",
      "잠금 해제는 잠금·generation·열린 호출만 확인하며 결정 워터마크·영역 검증·외부 대사는 확인하지 않습니다. 아무것도 복원·정리·삭제하지 않으며 호스트도 변경하지 않습니다.",
      "يتحقق رفع القفل فقط من القفل والجيل والاستدعاءات المفتوحة؛ ولا يتحقق من علامة القرار المائية أو التحقق من المجالات أو المطابقة الخارجية. لا تتم أي استعادة أو تقليم أو حذف، ولا يتغير أي مضيف.",
    ),
  resumeImpact: (l: Locale, id: string, recovery: string) =>
    t(
      l,
      `Approval confirms the listed lost sequences, records recovery ${mono(recovery)} as completed and returns ${mono(id)} to serving. Nothing is restored, pruned or deleted, and no host changes.`,
      `승인하면 표시된 유실 순번이 확정되고 복구 ${mono(recovery)}이(가) 완료로 기록되며 ${mono(id)}이(가) 운영 상태로 돌아갑니다. 아무것도 복원·정리·삭제하지 않으며 호스트도 변경하지 않습니다.`,
      `يؤكد الاعتماد التسلسلات المفقودة المدرجة، ويسجّل الاسترداد ${mono(recovery)} كمكتمل، ويعيد ${mono(id)} إلى التشغيل. لا تتم أي استعادة أو تقليم أو حذف، ولا يتغير أي مضيف.`,
    ),
  withdrawImpact: (l: Locale) => t(l, "Withdrawing closes this proposal. Nothing else changes.", "철회하면 이 제안이 종료됩니다. 그 밖에는 바뀌는 것이 없습니다.", "يغلق السحب هذا المقترح. لا يتغيّر شيء آخر."),
  repeat: (l: Locale) => t(l, "This repeats your earlier request exactly as it was sent.", "이전에 보낸 요청을 그대로 다시 보냅니다.", "يكرر هذا طلبك السابق تمامًا كما أُرسل."),

  approvedDone: (l: Locale, n: string, serving: string) =>
    t(l, `Unlock proposal #${n} applied.`, `잠금 해제 제안 #${n}이(가) 적용되었습니다.`, `طُبّق مقترح إلغاء القفل #${n}.`) +
    (serving ? " " + t(l, `${mono(serving)} is serving.`, `${mono(serving)}은(는) 운영 중입니다.`, `${mono(serving)} قيد التشغيل.`) : ""),
  withdrawnDone: (l: Locale, n: string) => t(l, `Proposal #${n} withdrawn.`, `제안 #${n}을(를) 철회했습니다.`, `سُحب المقترح #${n}.`),
  lockedDone: (l: Locale, id: string) => t(l, `${mono(id)} is locked. No host was changed.`, `${mono(id)}이(가) 잠겼습니다. 호스트는 변경되지 않았습니다.`, `${mono(id)} مقفل. لم يُغيَّر أي مضيف.`),
  proposedDone: (l: Locale, n: string) =>
    t(l, `Unlock proposal #${n} is waiting for another administrator's approval.`, `잠금 해제 제안 #${n}이(가) 다른 관리자의 승인을 기다리고 있습니다.`, `مقترح إلغاء القفل #${n} بانتظار اعتماد مسؤول آخر.`),
  duplicate: (l: Locale, n: string) => t(l, `This was already proposed as #${n}.`, `이미 #${n}(으)로 제안되었습니다.`, `اقتُرح هذا بالفعل بوصفه #${n}.`),
  resultUnknown: (l: Locale, state: string) => {
    const code = state ? mono(state) : t(l, "Unknown", "알 수 없음", "غير معروف");
    return t(l, `The request was recorded with an unknown result (${code}). Reload to check.`, `요청이 알 수 없는 결과(${code})로 기록되었습니다. 새로 고침해 확인하세요.`, `سُجّل الطلب بنتيجة غير معروفة (${code}). أعد التحميل للتحقق.`);
  },

  blocker: (l: Locale, code: string) => {
    if (code === "missing") return t(l, "This is not available now.", "지금은 사용할 수 없습니다.", "هذا غير متاح الآن.");
    if (has(blockerWords, code)) return words(l, blockerWords, code);
    return t(l, `This is not available now (${mono(code)}).`, `지금은 사용할 수 없습니다(${mono(code)}).`, `هذا غير متاح الآن (${mono(code)}).`);
  },
  already: (l: Locale, n: string, own: boolean) =>
    !n
      ? t(l, "An unlock proposal is already waiting for approval.", "잠금 해제 제안이 이미 승인을 기다리고 있습니다.", "يوجد مقترح إلغاء قفل بانتظار الاعتماد بالفعل.")
      : own
        ? t(l, `Your unlock proposal #${n} is waiting for another administrator's approval.`, `귀하의 잠금 해제 제안 #${n}이(가) 다른 관리자의 승인을 기다리고 있습니다.`, `مقترح إلغاء القفل الخاص بك #${n} بانتظار اعتماد مسؤول آخر.`)
        : t(l, `Unlock proposal #${n} is already waiting for another administrator's approval.`, `잠금 해제 제안 #${n}이(가) 이미 다른 관리자의 승인을 기다리고 있습니다.`, `مقترح إلغاء القفل #${n} بانتظار اعتماد مسؤول آخر بالفعل.`),
  narrow: (l: Locale, code: string) => words(l, narrowWords, code),
  kept: (l: Locale, command: string) => words(l, keptWords, command),
  gone: (l: Locale, proposal: boolean) =>
    proposal
      ? t(l, "This proposal is no longer available.", "이 제안은 더 이상 사용할 수 없습니다.", "لم يعد هذا المقترح متاحًا.")
      : t(l, "This installation is no longer available.", "이 설치는 더 이상 사용할 수 없습니다.", "لم يعد هذا التثبيت متاحًا."),
  pendingAttempt: (l: Locale) => t(l, "The last request may have been applied. Check the list, or send it again.", "직전 요청이 이미 반영되었을 수 있습니다. 목록을 확인하거나 다시 보내세요.", "ربما طُبّق الطلب الأخير. راجع القائمة أو أرسله مجددًا."),
  lostChanged: (l: Locale) =>
    t(l, "The installation's lost sequences or state changed. Review the current state.", "설치의 유실 순번이나 상태가 바뀌었습니다. 현재 상태를 검토하세요.", "تغيّرت التسلسلات المفقودة للتثبيت أو حالته. راجع الحالة الحالية."),
  builtBug: (l: Locale) => t(l, "The request could not be sent as built. Reload and try again.", "요청을 구성된 그대로 보낼 수 없습니다. 새로 고침한 뒤 다시 시도하세요.", "تعذّر إرسال الطلب كما بُني. أعد التحميل وحاول مجددًا."),
  conflictAgain: (l: Locale) => t(l, "The recovery data changed. Check the values and confirm again.", "복구 데이터가 바뀌었습니다. 값을 확인하고 다시 확인하세요.", "تغيّرت بيانات الاسترداد. تحقّق من القيم وأكّد مرة أخرى."),
  renew: (l: Locale) => t(l, "That request expired. Start again.", "요청이 만료되었습니다. 처음부터 다시 하세요.", "انتهت صلاحية الطلب. ابدأ من جديد."),
  partial: (l: Locale, op: string) => t(l, `Stopped after a change. Operation ${mono(op)}. Check the list before trying again.`, `변경 후 중단되었습니다. 작업 ${mono(op)}. 다시 시도하기 전에 목록을 확인하세요.`, `توقف بعد تغيير. العملية ${mono(op)}. راجع القائمة قبل المحاولة مجددًا.`),

  installState: (l: Locale, s: string) => words(l, installStateWords, s),
  publicLock: (l: Locale, s: string) => words(l, publicLockWords, s),
  recoveryState: (l: Locale, s: string) => words(l, recoveryWords, s),
  phaseWord: (l: Locale, s: string) => words(l, phaseWords, s),
  modeWord: (l: Locale, s: string) => words(l, modeWords, s),
  domain: (l: Locale, s: string) => words(l, domainWords, s),
  domainStatus: (l: Locale, s: string) => words(l, domainStatusWords, s),
  writerKind: (l: Locale, s: string) => words(l, writerKindWords, s),
  externalStatus: (l: Locale, s: string) => words(l, externalStatusWords, s),
  fenced: (l: Locale, s: string) => words(l, fencedWords, s),
  callsSource: (l: Locale, s: string) => words(l, callsSourceWords, s),
  eventKind: (l: Locale, s: string) => words(l, eventKindWords, s),
  outcome: (l: Locale, s: string) => words(l, outcomeWords, s),
  proposalKind: (l: Locale, s: string) => words(l, proposalKindWords, s),
  proposalState: (l: Locale, s: string) => words(l, proposalStateWords, s),
  migrationModule: (l: Locale, s: string) => words(l, migrationWords, s),
  connection: (l: Locale, s: string) => words(l, connectionWords, s),
  shown: (l: Locale, s: string) => words(l, shownWords, s),
  result: (l: Locale, s: string) => words(l, resultWords, s),

  form: {
    rationale: (l: Locale) => t(l, "Reason", "사유", "السبب"),
    rationaleHint: (l: Locale, n: number) => t(l, `${n} of 256 bytes. Required. Do not enter customer data.`, `256바이트 중 ${n}바이트. 필수. 고객 정보를 입력하지 마세요.`, `${n} من 256 بايت. مطلوب. لا تُدخل بيانات العملاء.`),
    rationaleMissing: (l: Locale) => t(l, "Enter a reason.", "사유를 입력하세요.", "أدخل سببًا."),
    checkReason: (l: Locale) => t(l, "Check the reason and try again.", "사유를 확인하고 다시 시도하세요.", "تحقّق من السبب وحاول مجددًا."),
    rationaleLong: (l: Locale) => t(l, "The reason is too long. Use at most 256 bytes.", "사유가 너무 깁니다. 256바이트 이하로 입력하세요.", "السبب طويل جدًا. استخدم 256 بايت على الأكثر."),
    review: (l: Locale) => t(l, "Review", "검토", "مراجعة"),
    cancel: (l: Locale) => t(l, "Cancel", "취소", "إلغاء"),
    backToEdit: (l: Locale) => t(l, "Back to edit", "수정으로 돌아가기", "العودة إلى التعديل"),
    sending: (l: Locale) => t(l, "Sending…", "보내는 중…", "جارٍ الإرسال…"),
  },
};
