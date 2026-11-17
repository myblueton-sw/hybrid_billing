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
};

export const shownWords: Words = {
  not_shown: ["Not shown", "표시하지 않음", "غير معروض"],
};

export const installStateWords: Words = {
  serving: ["Serving", "운영 중", "قيد التشغيل"],
  locked: ["Locked", "잠김", "مقفل"],
  recovering: ["In recovery", "복구 중", "قيد الاسترداد"],
};

export const publicLockWords: Words = {
  serving: ["Not locked", "잠기지 않음", "غير مقفل"],
  locked: ["Writes locked", "쓰기 잠김", "الكتابة مقفلة"],
  recovering: ["Writes locked (in recovery)", "쓰기 잠김(복구 중)", "الكتابة مقفلة (قيد الاسترداد)"],
};

export const recoveryWords: Words = {
  open: ["Open", "진행 중", "مفتوح"],
  none: ["No recovery open", "진행 중인 복구 없음", "لا يوجد استرداد مفتوح"],
};

export const phaseWords: Words = {
  verify: ["Verifying the restore", "복원 검증 중", "التحقق من الاستعادة"],
  reapply: ["Reapplying decision events", "결정 기록 재적용 중", "إعادة تطبيق أحداث القرار"],
  ready: ["Ready for a resume decision", "재개 판정 대기", "جاهز لقرار الاستئناف"],
  not_recovering: ["Not in recovery", "복구 중 아님", "ليس قيد الاسترداد"],
};

export const modeWords: Words = {
  recovery_resume: ["Resume after recovery", "복구 후 재개", "الاستئناف بعد الاسترداد"],
  lock_release: ["Lock release", "잠금 해제(복구 없음)", "رفع القفل"],
  none: ["Nothing to resume", "재개할 대상 없음", "لا شيء للاستئناف"],
};

export const domainWords: Words = {
  database: ["Database", "데이터베이스", "قاعدة البيانات"],
  evidence: ["Evidence files", "증빙 파일", "ملفات الأدلة"],
  keys: ["Keys", "키", "المفاتيح"],
  policies: ["Policies", "정책", "السياسات"],
  modules: ["Modules", "모듈", "الوحدات"],
  numbering: ["Document numbering", "문서 번호", "ترقيم المستندات"],
  submissions: ["Submissions", "제출 기록", "الإرسالات"],
  outbox: ["Outbox", "발신 대기열(outbox)", "صندوق الصادر"],
  deletions: ["Deletion records", "삭제 기록", "سجلات الحذف"],
};

export const domainStatusWords: Words = {
  pending: ["Not checked yet", "검증 전", "لم يُتحقق بعد"],
  verified: ["Verified", "검증됨", "تم التحقق"],
  mismatch: ["Mismatch", "불일치", "عدم تطابق"],
};

export const writerKindWords: Words = {
  erp: ["ERP", "ERP", "ERP"],
  issuance: ["Invoice issuance", "청구서 발행", "إصدار الفواتير"],
  delivery: ["Delivery", "전달", "التسليم"],
  payment: ["Payment", "결제", "الدفع"],
  message: ["Messaging", "메시지 발송", "المراسلة"],
};

export const externalStatusWords: Words = {
  pending: ["Not reconciled", "대사 전", "لم تتم المطابقة"],
  reconciled: ["Reconciled", "대사 완료", "تمت المطابقة"],
};

export const fencedWords: Words = {
  true: ["Fenced", "차단됨", "محجوب"],
  false: ["Not fenced", "차단 안 됨", "غير محجوب"],
};

export const callsSourceWords: Words = {
  recovery: ["From the open recovery", "진행 중인 복구 기준", "من الاسترداد المفتوح"],
};

export const eventKindWords: Words = {
  delete: ["Deletion", "삭제", "حذف"],
  purge: ["Purge", "영구 삭제", "محو نهائي"],
  hold: ["Hold", "보존 보류", "تعليق الحذف"],
  extend: ["Retention extension", "보존 기간 연장", "تمديد الاحتفاظ"],
  revoke_key: ["Key revocation", "키 회수", "إلغاء مفتاح"],
  revoke_access: ["Access revocation", "접근 권한 회수", "إلغاء الوصول"],
};

export const outcomeWords: Words = {
  completed: ["Completed", "완료", "مكتمل"],
  abandoned: ["Abandoned", "중단됨", "متروك"],
};

export const proposalKindWords: Words = {
  installation: ["Installation", "설치", "التثبيت"],
  writer: ["Writer", "쓰기 주체", "جهة الكتابة"],
  external: ["External system", "외부 시스템", "نظام خارجي"],
  verifier: ["Verifier", "검증 주체", "جهة التحقق"],
  begin: ["Recovery opened (isolated restore record)", "복구 개시(격리 복원 기록)", "فتح استرداد (سجل استعادة معزولة)"],
  unlock: ["Unlock", "잠금 해제 제안", "إلغاء القفل"],
  prune: ["Prune decision events", "결정 기록 정리", "تقليم أحداث القرار"],
  abandon: ["Abandon recovery", "복구 중단", "التخلي عن الاسترداد"],
};

export const proposalStateWords: Words = {
  pending: ["Pending", "승인 대기", "بانتظار الاعتماد"],
  applied: ["Applied", "적용됨", "مطبّق"],
  withdrawn: ["Withdrawn", "철회됨", "مسحوب"],
  retired: ["Retired", "폐기됨", "مُسقَط"],
};

export const migrationWords: Words = {
  migration: ["Customer migration import", "고객 이관 반입", "استيراد ترحيل العملاء"],
  balances: ["Opening balances", "기초 잔액", "الأرصدة الافتتاحية"],
};

export const resultWords: Words = {
  applied: proposalStateWords.applied,
  withdrawn: proposalStateWords.withdrawn,
  pending: proposalStateWords.pending,
  locked: installStateWords.locked,
};

export const summaryWords: Words = {
  installation: ["Register installation {installation} (clock skew {skew})", "설치 {installation} 등록(시계 오차 {skew})", "تسجيل التثبيت {installation} (انحراف الساعة {skew})"],
  writer: ["Register writer {writer} for {installation}", "{installation}에 쓰기 주체 {writer} 등록", "تسجيل جهة الكتابة {writer} لـ {installation}"],
  external: ["Register external system {external} ({kind}) for {installation}", "{installation}에 외부 시스템 {external}({kind}) 등록", "تسجيل النظام الخارجي {external} ({kind}) لـ {installation}"],
  verifier: ["Register verifier {verifier} for {installation}", "{installation}에 검증 주체 {verifier} 등록", "تسجيل جهة التحقق {verifier} لـ {installation}"],
  begin: [
    "Open recovery of {installation} against the restore point as of {as_of} (generation {point_generation}, manifest {manifest_digest}); recorded in the registry, no data was restored",
    "{as_of} 기준 복원 지점(generation {point_generation}, manifest {manifest_digest})에 대해 {installation} 복구 개시. 레지스트리에 기록만 되었으며 데이터는 복원되지 않음",
    "فتح استرداد {installation} مقابل نقطة الاستعادة بتاريخ {as_of} (الجيل {point_generation}، البيان {manifest_digest})؛ سُجّل في السجل ولم تُستعد أي بيانات",
  ],
  unlock: [
    "Unlock {installation} at generation {unlock_generation}, confirming lost-sequence digest {lost_digest}",
    "{installation}을(를) generation {unlock_generation}에서 잠금 해제, 유실 순번 요약값 {lost_digest} 확인",
    "إلغاء قفل {installation} عند الجيل {unlock_generation}، مع تأكيد ملخص التسلسلات المفقودة {lost_digest}",
  ],
  prune: ["Prune decision events of {installation}", "{installation}의 결정 기록 정리", "تقليم أحداث القرار لـ {installation}"],
  abandon: ["Abandon the recovery of {installation}", "{installation}의 복구 중단", "التخلي عن استرداد {installation}"],
};

export const blockerWords: Words = {
  own_proposal: ["You proposed this. Another administrator must approve it.", "본인이 제안했습니다. 다른 관리자가 승인해야 합니다.", "أنت من اقترح هذا. يجب أن يعتمده مسؤول آخر."],
  not_pending: ["This is no longer waiting for a decision. Reload.", "더 이상 결정을 기다리고 있지 않습니다. 새로 고침하세요.", "لم يعد هذا بانتظار قرار. أعد التحميل."],
  retired: [
    "The recovery registry no longer holds this proposal. Nothing can be done with it here.",
    "복구 레지스트리에 이 제안이 더 이상 없습니다. 여기에서 할 수 있는 작업이 없습니다.",
    "لم يعد سجل الاسترداد يحتفظ بهذا المقترح. لا يمكن فعل أي شيء به هنا.",
  ],
  not_console_kind: [
    "This kind of change cannot be approved from the console in this build. Its proposer can still withdraw it.",
    "이 빌드에서는 이 종류의 변경을 콘솔에서 승인할 수 없습니다. 제안자는 철회할 수 있습니다.",
    "لا يمكن اعتماد هذا النوع من التغيير من وحدة التحكم في هذا الإصدار. لا يزال بإمكان مقترِحه سحبه.",
  ],
  recovering: [
    "The installation is in recovery and already locked for writes in the recovery registry. It cannot be locked separately.",
    "설치가 복구 중이며 복구 레지스트리에서 이미 쓰기가 잠겨 있습니다. 따로 잠글 수 없습니다.",
    "التثبيت قيد الاسترداد ومقفل للكتابة بالفعل في سجل الاسترداد. لا يمكن قفله بشكل منفصل.",
  ],
  locked: ["The installation is already locked.", "설치가 이미 잠겨 있습니다.", "التثبيت مقفل بالفعل."],
  not_recovering: [
    "The installation is not locked or in recovery, so there is nothing to unlock.",
    "설치가 잠겨 있거나 복구 중이 아니므로 잠금 해제할 대상이 없습니다.",
    "التثبيت غير مقفل وليس قيد الاسترداد، لذا لا يوجد ما يُلغى قفله.",
  ],
  open_calls: [
    "Writer calls are still open. They close when the writer reports them or at their deadline.",
    "쓰기 주체의 호출이 아직 열려 있습니다. 쓰기 주체가 결과를 보고하거나 마감 시각이 되면 닫힙니다.",
    "لا تزال استدعاءات جهة الكتابة مفتوحة. تُغلق عندما تبلّغ عنها جهة الكتابة أو عند موعدها النهائي.",
  ],
  no_restore_point: ["The recovery has no restore point, so it cannot be resumed.", "복구에 복원 지점이 없어 재개할 수 없습니다.", "لا توجد نقطة استعادة للاسترداد، لذا لا يمكن استئنافه."],
  incomplete: [
    "The recovery is not complete: a domain is not verified, an external system is not reconciled, or decision events are still pending.",
    "복구가 완료되지 않았습니다. 검증되지 않은 영역이 있거나, 대사되지 않은 외부 시스템이 있거나, 아직 처리되지 않은 결정 기록이 있습니다.",
    "الاسترداد غير مكتمل: هناك مجال لم يُتحقق منه، أو نظام خارجي لم تتم مطابقته، أو أحداث قرار لا تزال معلّقة.",
  ],
  lost: [
    "The confirmed lost sequences do not match the recovery's current lost sequences.",
    "확인한 유실 순번이 복구의 현재 유실 순번과 일치하지 않습니다.",
    "التسلسلات المفقودة المؤكدة لا تطابق التسلسلات المفقودة الحالية للاسترداد.",
  ],
  kind_coverage: ["A writer kind has no external system to reconcile against.", "대사할 외부 시스템이 없는 쓰기 주체 종류가 있습니다.", "يوجد نوع من جهات الكتابة لا يقابله نظام خارجي للمطابقة."],
  generation: ["The installation's generation changed after this was proposed.", "제안 이후 설치의 generation이 바뀌었습니다.", "تغيّر جيل التثبيت بعد اقتراح هذا."],
  not_found: ["The installation is no longer registered in the recovery registry.", "설치가 더 이상 복구 레지스트리에 등록되어 있지 않습니다.", "لم يعد التثبيت مسجّلًا في سجل الاسترداد."],
  invalid: ["The recovery registry could not check this request.", "복구 레지스트리가 이 요청을 확인할 수 없었습니다.", "تعذّر على سجل الاسترداد التحقق من هذا الطلب."],
  limit: ["A recovery registry limit is reached.", "복구 레지스트리 한도에 도달했습니다.", "تم بلوغ حد في سجل الاسترداد."],
  conflict: ["Another record already uses this identifier.", "다른 기록이 이미 이 식별자를 사용합니다.", "يستخدم سجل آخر هذا المعرّف بالفعل."],
};

export const narrowWords: Words = {
  no_command: ["No command is available for this row.", "이 행에는 사용할 수 있는 명령이 없습니다.", "لا يتوفر أمر لهذا الصف."],
  unreadable: [
    "The lost sequences or the mode of this installation could not be read, so an unlock cannot be proposed here. Reload.",
    "이 설치의 유실 순번이나 모드를 읽을 수 없어 여기에서 잠금 해제를 제안할 수 없습니다. 새로 고침하세요.",
    "تعذّرت قراءة التسلسلات المفقودة أو وضع هذا التثبيت، لذا لا يمكن اقتراح إلغاء القفل هنا. أعد التحميل.",
  ],
  resume_unknown: [
    "The installation's resume state is unknown, so this cannot be approved here.",
    "설치의 재개 상태를 알 수 없어 여기에서 승인할 수 없습니다.",
    "حالة الاستئناف للتثبيت غير معروفة، لذا لا يمكن اعتماد هذا هنا.",
  ],
  generation_unknown: [
    "The installation's current generation is unknown, so this cannot be approved here. Reload.",
    "설치의 현재 generation을 알 수 없어 여기에서 승인할 수 없습니다. 새로 고침하세요.",
    "الجيل الحالي للتثبيت غير معروف، لذا لا يمكن اعتماد هذا هنا. أعد التحميل.",
  ],
  changed: [
    "The installation changed after this was proposed. Reload or ask the proposer to withdraw it.",
    "제안 이후 설치가 바뀌었습니다. 새로 고침하거나 제안자에게 철회를 요청하세요.",
    "تغيّر التثبيت بعد اقتراح هذا. أعد التحميل أو اطلب من المقترِح سحبه.",
  ],
};

export const keptWords: Words = {
  "recovery.approve": [
    "The recovery registry refused this approval. Nothing changed. Review the proposal and the installation's state.",
    "복구 레지스트리가 이 승인을 거부했습니다. 바뀐 것은 없습니다. 제안과 설치 상태를 검토하세요.",
    "رفض سجل الاسترداد هذا الاعتماد. لم يتغيّر شيء. راجع المقترح وحالة التثبيت.",
  ],
  "recovery.withdraw": ["The recovery registry refused this withdrawal. Nothing changed.", "복구 레지스트리가 이 철회를 거부했습니다. 바뀐 것은 없습니다.", "رفض سجل الاسترداد هذا السحب. لم يتغيّر شيء."],
  "recovery.lock": [
    "The lock was refused. Nothing changed. The installation may already be locked or in recovery.",
    "잠금이 거부되었습니다. 바뀐 것은 없습니다. 설치가 이미 잠겨 있거나 복구 중일 수 있습니다.",
    "رُفض القفل. لم يتغيّر شيء. ربما يكون التثبيت مقفلًا بالفعل أو قيد الاسترداد.",
  ],
  "recovery.propose_unlock": [
    "The unlock proposal was refused. Nothing changed. Another unlock may already be waiting, or you may have too many pending proposals.",
    "잠금 해제 제안이 거부되었습니다. 바뀐 것은 없습니다. 다른 잠금 해제 제안이 이미 대기 중이거나 승인 대기 중인 제안이 너무 많을 수 있습니다.",
    "رُفض مقترح إلغاء القفل. لم يتغيّر شيء. ربما يوجد إلغاء قفل آخر بالانتظار، أو لديك عدد كبير من المقترحات المعلّقة.",
  ],
};

export const allWordTables: Record<string, Words> = {
  sectionWords, connectionWords, shownWords, installStateWords, publicLockWords, recoveryWords, phaseWords, modeWords, domainWords, domainStatusWords, writerKindWords,
  externalStatusWords, fencedWords, callsSourceWords, eventKindWords, outcomeWords, proposalKindWords, proposalStateWords, migrationWords, resultWords, summaryWords,
  blockerWords, narrowWords, keptWords,
};
