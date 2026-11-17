import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export type Words = Record<string, [string, string, string]>;

export const unknownCode = (l: Locale, code: string) => t(l, `Unknown (${code})`, `알 수 없음(${code})`, `غير معروف (${code})`);

export function words(l: Locale, table: Words, key: string): string {
  const w = Object.prototype.hasOwnProperty.call(table, key) ? table[key] : undefined;
  return w ? t(l, w[0], w[1], w[2]) : unknownCode(l, key);
}

export const classes = ["claim", "conversation", "final", "identity", "projection", "source", "temporary"];

export const classWords: Words = {
  source: ["Source records", "원천 자료", "سجلات المصدر"],
  final: ["Final documents and manifests", "확정 문서·매니페스트", "المستندات والبيانات النهائية"],
  claim: ["Duplicate-prevention records", "중복 방지 기록", "سجلات منع التكرار"],
  identity: ["Identity and access records", "신원·접근 기록", "سجلات الهوية والوصول"],
  conversation: ["Assistant conversations", "Assistant 대화", "محادثات المساعد"],
  temporary: ["Temporary files", "임시 파일", "الملفات المؤقتة"],
  projection: ["Caches and projections", "캐시·프로젝션", "ذاكرات التخزين المؤقت والإسقاطات"],
};

export const standingWords: Words = {
  eligible: ["Due for deletion", "삭제 대상", "مستحق للحذف"],
  held: ["On hold", "보존 보류 중", "معلّق الحذف"],
  referenced: ["Required by another item", "다른 항목이 참조", "مطلوب لعنصر آخر"],
  not_due: ["Not due yet", "아직 기한 전", "لم يحن موعده"],
  no_policy: ["No policy", "정책 없음", "لا توجد سياسة"],
  in_run: ["In a deletion run", "삭제 작업 중", "ضمن عملية حذف"],
  deleted: ["Recorded as deleted", "삭제로 기록됨", "مسجّل كمحذوف"],
};

export const accessWords: Words = {
  ok: ["Accessible", "접근 가능", "متاح"],
  blocked: ["Blocked", "차단됨", "محجوب"],
  deleted: ["Recorded as deleted", "삭제로 기록됨", "مسجّل كمحذوف"],
};

export const holdOutcomeWords: Words = {
  not_preserved: ["Already deleted (not preserved)", "이미 삭제됨(보존 불가)", "محذوف بالفعل (غير محفوظ)"],
  outcome_pending: ["Deletion outcome pending", "삭제 결과 대기", "نتيجة الحذف معلّقة"],
  partially_deleted: ["Partly deleted", "일부 삭제됨", "محذوف جزئيًا"],
};

export const storeResultWords: Words = {
  deleted: ["deleted", "삭제됨", "محذوف"],
  failed: ["failed", "실패", "فشل"],
};

export const runStateWords: Words = {
  running: ["Running (no worker connected)", "실행 중(작업자 미연결)", "قيد التشغيل (لا توجد جهة متصلة)"],
  stopped: ["Stopped (a claimed chunk is still open)", "중지됨(맡은 묶음 진행 중)", "موقوفة (دفعة مستلمة ما زالت مفتوحة)"],
  done: ["Finished, not complete", "종료됨(미완료)", "انتهت، غير مكتملة"],
  complete: ["Closed without failures", "실패 없이 종료됨", "أُغلقت دون إخفاقات"],
};

export const runStateNotes: Words = {
  done: [
    "Not every item was deleted, or backup expiry is not recorded for every deleted item.",
    "모든 항목이 삭제되지 않았거나, 삭제된 모든 항목의 백업 만료가 기록되지 않았습니다.",
    "لم تُحذف كل العناصر، أو لم يُسجَّل انتهاء النسخ الاحتياطية لكل عنصر محذوف.",
  ],
  complete: [
    "This does not by itself mean any item was deleted: a run whose items were all skipped or stopped also closes this way.",
    "이것만으로 항목이 삭제되었다는 뜻은 아닙니다. 모든 항목이 제외되거나 중지된 작업도 이렇게 종료됩니다.",
    "هذا لا يعني بحد ذاته أن أي عنصر قد حُذف: العملية التي استُبعدت كل عناصرها أو أوقفت تُغلق بهذه الطريقة أيضًا.",
  ],
};

export const runOutcomeWords: Words = {
  pending: ["Not yet claimed", "아직 맡지 않음", "لم يُتسلَّم بعد"],
  deleting: ["Being deleted", "삭제 중", "قيد الحذف"],
  deleted: ["Deleted", "삭제됨", "محذوف"],
  failed: ["Failed", "실패", "فشل"],
  held: ["Skipped: on hold", "제외: 보존 보류", "مستبعد: معلّق الحذف"],
  referenced: ["Skipped: required", "제외: 참조됨", "مستبعد: مطلوب"],
  not_due: ["Skipped: not due", "제외: 기한 전", "مستبعد: لم يحن موعده"],
  stopped: ["Stopped", "중지됨", "موقوف"],
};

export const holdStateWords: Words = {
  in_force: ["In force", "유지 중", "سارٍ"],
  released: ["Released", "해제됨", "مرفوع"],
};

export const originWords: Words = {
  artifact_anchor: ["Each item's recorded start time", "항목별 기록된 기산 시각", "وقت البدء المسجّل لكل عنصر"],
};

export const connectionWords: Words = {
  not_connected: ["Not connected", "연결되지 않음", "غير متصل"],
};

export const ledgerWords: Words = {
  in_memory: ["In memory (not durable)", "메모리 내(비내구)", "في الذاكرة (غير دائم)"],
};

export const decisionWords: Words = {
  not_connected: ["Decision record: not connected (not confirmed)", "결정 기록: 미연결(미확정)", "سجل القرار: غير متصل (غير مؤكد)"],
};

export const populationWords: Words = {
  registered_sample_artifacts: ["Registered sample items in this workspace", "이 업무 공간에 등록된 샘플 항목", "عناصر العينة المسجّلة في مساحة العمل هذه"],
};

export const proposalKindWords: Words = {
  policy: ["Retention policy", "보관 정책", "سياسة الاحتفاظ"],
  plan: ["Deletion plan", "삭제 계획", "خطة حذف"],
  release: ["Hold release", "보존 보류 해제", "رفع تعليق الحذف"],
};

export const proposalStateWords: Words = {
  pending: ["Pending", "승인 대기", "بانتظار الاعتماد"],
  applied: ["Applied", "적용됨", "مطبّق"],
  withdrawn: ["Withdrawn", "철회됨", "مسحوب"],
  retired: ["Retired", "폐기됨", "مُسقَط"],
};

export const resultWords: Words = {
  held: ["In force", "유지 중", "سارٍ"],
  partially_preserved: ["In force, not all preserved", "유지 중(일부 보존 불가)", "سارٍ مع عدم حفظ الكل"],
};

export const summaryWords: Words = {
  policy: [
    "Set {class} to {p} days plus {g} days grace (total {t} days)",
    "{class} 보관 기간을 {p}일, 유예 {g}일(합계 {t}일)로 설정",
    "ضبط {class} على {p} يومًا مع فترة سماح {g} يومًا (المجموع {t} يومًا)",
  ],
  plan: [
    "Delete {n} {class} items in chunks of {c} (lease {s} s): {reason}",
    "{class} 항목 {n}건을 {c}건 단위로 삭제(점유 {s}초): {reason}",
    "حذف {n} من عناصر {class} على دفعات من {c} (مدة الحجز {s} ث): {reason}",
  ],
  release: ["Release hold {hold}: {reason}", "보존 보류 {hold} 해제: {reason}", "رفع تعليق الحذف {hold}: {reason}"],
};

export const blockerWords: Words = {
  own_proposal: ["You proposed this. Another administrator must approve it.", "직접 제안한 건입니다. 다른 관리자가 승인해야 합니다.", "أنت من اقترح هذا. يجب أن يعتمده مسؤول آخر."],
  not_pending: ["This is no longer waiting for a decision. Reload.", "더 이상 결정을 기다리는 건이 아닙니다. 새로 고침하세요.", "لم يعد هذا بانتظار قرار. أعد التحميل."],
  retired: [
    "The retention register no longer holds this proposal. Nothing can be done with it here.",
    "보관 레지스트리에 이 제안이 더 이상 없습니다. 여기에서 할 수 있는 작업이 없습니다.",
    "لم يعد سجل الاحتفاظ يحتفظ بهذا المقترح. لا يمكن فعل شيء به هنا.",
  ],
  not_console_kind: [
    "Deletion plans cannot be approved or withdrawn from the console in this build.",
    "이 빌드에서는 콘솔에서 삭제 계획을 승인하거나 철회할 수 없습니다.",
    "لا يمكن اعتماد خطط الحذف أو سحبها من وحدة التحكم في هذا الإصدار.",
  ],
  no_eligible_approver: [
    "Fewer than two administrators besides the person who placed this hold can act, so its release cannot be proposed and approved.",
    "이 보존 보류를 등록한 사람 외에 조치할 수 있는 관리자가 두 명보다 적어 해제를 제안하고 승인할 수 없습니다.",
    "يقل عدد المسؤولين القادرين على التصرف، عدا من وضع هذا التعليق، عن اثنين، لذا لا يمكن اقتراح رفعه واعتماده.",
  ],
  released: ["This hold is already released.", "이 보존 보류는 이미 해제되었습니다.", "رُفع تعليق الحذف هذا بالفعل."],
  impact_changed: [
    "The impact changed after it was reviewed. Reload and review it again.",
    "검토 이후 영향이 바뀌었습니다. 새로 고침한 뒤 다시 검토하세요.",
    "تغيّر الأثر بعد مراجعته. أعد التحميل وراجعه مجددًا.",
  ],
  target_too_long: [
    "This item's identifier is too long to name in a command, so it cannot be held here.",
    "이 항목의 식별자가 너무 길어 명령에 지정할 수 없으므로 여기에서 보존 보류를 등록할 수 없습니다.",
    "معرّف هذا العنصر أطول من أن يُذكر في أمر، لذا لا يمكن تعليق حذفه هنا.",
  ],
  not_found: ["The retention register no longer knows this hold or proposal.", "보관 레지스트리에 이 보존 보류나 제안이 더 이상 없습니다.", "لم يعد سجل الاحتفاظ يعرف هذا التعليق أو المقترح."],
  limit: ["A retention register limit is reached.", "보관 레지스트리 한도에 도달했습니다.", "تم بلوغ حد في سجل الاحتفاظ."],
  conflict: ["Another record already uses this identifier.", "다른 기록이 이미 이 식별자를 사용합니다.", "يستخدم سجل آخر هذا المعرّف بالفعل."],
  invalid: ["The retention register could not check this request.", "보관 레지스트리가 이 요청을 확인할 수 없었습니다.", "تعذّر على سجل الاحتفاظ التحقق من هذا الطلب."],
};

export const placerWords: Words = {
  approve: ["You placed this hold, so you cannot approve its release.", "직접 등록한 보존 보류이므로 해제를 승인할 수 없습니다.", "أنت من وضع هذا التعليق، لذا لا يمكنك اعتماد رفعه."],
  propose_release: [
    "You placed this hold, so you cannot propose its release. Two other administrators must propose and approve it.",
    "직접 등록한 보존 보류이므로 해제를 제안할 수 없습니다. 다른 관리자 두 명이 제안하고 승인해야 합니다.",
    "أنت من وضع هذا التعليق، لذا لا يمكنك اقتراح رفعه. يجب أن يقترحه ويعتمده مسؤولان آخران.",
  ],
};

export const stateSentenceWords: Words = {
  not_proposer: ["Only the proposer can withdraw it.", "제안자만 철회할 수 있습니다.", "لا يستطيع سحبه إلا المقترِح."],
  released: ["This hold is released; there is nothing to release.", "이 보존 보류는 해제되었으므로 해제할 것이 없습니다.", "رُفع هذا التعليق؛ لا يوجد ما يُرفع."],
  stopped: ["This run is stopped; a claimed chunk is still open.", "이 작업은 중지되었으며 맡은 묶음이 아직 진행 중입니다.", "هذه العملية موقوفة؛ ما زالت دفعة مستلمة مفتوحة."],
  finished: ["This run has finished, so it cannot be stopped.", "이 작업은 종료되었으므로 중지할 수 없습니다.", "انتهت هذه العملية، لذا لا يمكن إيقافها."],
};

export const narrowWords: Words = {
  no_command: ["No command is available for this row.", "이 행에는 사용할 수 있는 명령이 없습니다.", "لا يتوفر أمر لهذا الصف."],
  impact_unreadable: [
    "The impact could not be read, so this cannot be approved here. Reload.",
    "영향을 읽을 수 없어 여기에서 승인할 수 없습니다. 새로 고침하세요.",
    "تعذّرت قراءة الأثر، لذا لا يمكن اعتماد هذا هنا. أعد التحميل.",
  ],
};

export const keptWords: Words = {
  "retention.approve": [
    "The retention register refused this approval. Nothing changed. For a shorter period, the impact may have changed or be more than an hour old: review it again.",
    "보관 레지스트리가 이 승인을 거부했습니다. 바뀐 것은 없습니다. 기간을 줄이는 경우 영향이 바뀌었거나 1시간이 지났을 수 있으니 다시 검토하세요.",
    "رفض سجل الاحتفاظ هذا الاعتماد. لم يتغيّر شيء. عند تقصير المدة، ربما تغيّر الأثر أو مضت عليه أكثر من ساعة: راجعه مجددًا.",
  ],
  "retention.withdraw": ["The retention register refused this withdrawal. Nothing changed.", "보관 레지스트리가 이 철회를 거부했습니다. 바뀐 것은 없습니다.", "رفض سجل الاحتفاظ هذا السحب. لم يتغيّر شيء."],
  "retention.propose_policy": [
    "The policy proposal was refused. Nothing changed. Another proposal may already be waiting, the values may equal the current policy, or you may have too many pending proposals.",
    "정책 제안이 거부되었습니다. 바뀐 것은 없습니다. 다른 제안이 이미 대기 중이거나, 값이 현재 정책과 같거나, 승인 대기 중인 제안이 너무 많을 수 있습니다.",
    "رُفض مقترح السياسة. لم يتغيّر شيء. ربما يوجد مقترح آخر بالانتظار، أو تساوي القيم السياسة الحالية، أو لديك عدد كبير من المقترحات المعلّقة.",
  ],
  "retention.place_hold:refused": [
    "The hold was not placed. Nothing changed. This workspace already has 100 holds in force, or the retention register's hold limit is reached.",
    "보존 보류가 등록되지 않았습니다. 바뀐 것은 없습니다. 이 업무 공간에 유지 중인 보존 보류가 이미 100건이거나 보관 레지스트리의 보존 보류 한도에 도달했습니다.",
    "لم يُوضع التعليق. لم يتغيّر شيء. لدى مساحة العمل هذه 100 تعليق سارٍ بالفعل، أو بلغ سجل الاحتفاظ حد التعليقات.",
  ],
  "retention.place_hold:stage": [
    "The hold was not placed. Nothing changed. Reload and try again.",
    "보존 보류가 등록되지 않았습니다. 바뀐 것은 없습니다. 새로 고침한 뒤 다시 시도하세요.",
    "لم يُوضع التعليق. لم يتغيّر شيء. أعد التحميل وحاول مجددًا.",
  ],
  "retention.propose_release": [
    "The release proposal was refused. Nothing changed. The hold may already be released, a release may already be waiting, or you placed it.",
    "해제 제안이 거부되었습니다. 바뀐 것은 없습니다. 보존 보류가 이미 해제되었거나, 해제 제안이 이미 대기 중이거나, 직접 등록한 보존 보류일 수 있습니다.",
    "رُفض مقترح الرفع. لم يتغيّر شيء. ربما رُفع التعليق بالفعل، أو يوجد رفع بالانتظار، أو أنت من وضعه.",
  ],
  "retention.stop_run": [
    "The stop was refused. Nothing changed. The run may already be stopped or finished.",
    "중지가 거부되었습니다. 바뀐 것은 없습니다. 작업이 이미 중지되었거나 종료되었을 수 있습니다.",
    "رُفض الإيقاف. لم يتغيّر شيء. ربما تكون العملية موقوفة أو منتهية بالفعل.",
  ],
};

export type { Plural } from "../../../../shared/plural";
export { category, counted } from "../../../../shared/plural";
import type { Plural } from "../../../../shared/plural";

export const plurals = {
  items: {
    en: ["1 item", "{n} items"],
    ko: "{n}건",
    ar: ["{n} عنصر", "عنصر واحد", "عنصران", "{n} عناصر", "{n} عنصرًا", "{n} عنصر"],
  },
  showingOf: {
    en: ["Showing 50 of 1.", "Showing 50 of {n}."],
    ko: "{n}건 중 50건을 표시합니다.",
    ar: ["عرض 50 من أصل {n}.", "عرض 50 من أصل عنصر واحد.", "عرض 50 من أصل عنصرين.", "عرض 50 من أصل {n} عناصر.", "عرض 50 من أصل {n} عنصرًا.", "عرض 50 من أصل {n} عنصر."],
  },
  showingNewest: {
    en: ["Showing the newest 50 of 1.", "Showing the newest 50 of {n}."],
    ko: "{n}건 중 최신 50건을 표시합니다.",
    ar: ["عرض أحدث 50 من أصل {n}.", "عرض أحدث 50 من أصل عنصر واحد.", "عرض أحدث 50 من أصل عنصرين.", "عرض أحدث 50 من أصل {n} عناصر.", "عرض أحدث 50 من أصل {n} عنصرًا.", "عرض أحدث 50 من أصل {n} عنصر."],
  },
  chunks: {
    en: ["{next} of 1 chunk finished (a stop finishes unclaimed chunks)", "{next} of {n} chunks finished (a stop finishes unclaimed chunks)"],
    ko: "묶음 {n}개 중 {next}개 종료(중지하면 맡지 않은 묶음이 종료됨)",
    ar: [
      "انتهت {next} من {n} دفعة (الإيقاف يُنهي الدفعات غير المستلمة)",
      "انتهت {next} من دفعة واحدة (الإيقاف يُنهي الدفعات غير المستلمة)",
      "انتهت {next} من دفعتين (الإيقاف يُنهي الدفعات غير المستلمة)",
      "انتهت {next} من {n} دفعات (الإيقاف يُنهي الدفعات غير المستلمة)",
      "انتهت {next} من {n} دفعة (الإيقاف يُنهي الدفعات غير المستلمة)",
      "انتهت {next} من {n} دفعة (الإيقاف يُنهي الدفعات غير المستلمة)",
    ],
  },
  stopDone: {
    en: ["{run} stopped: 1 unclaimed item was released, not deleted. Nothing was restored.", "{run} stopped: {n} unclaimed items were released, not deleted. Nothing was restored."],
    ko: "{run}을(를) 중지했습니다. 맡지 않은 항목 {n}건이 삭제되지 않고 작업에서 풀려났습니다. 아무것도 복원되지 않았습니다.",
    ar: [
      "أُوقفت {run}: لم يُحرَّر أي عنصر غير مستلم. لم يُستعد شيء.",
      "أُوقفت {run}: حُرّر عنصر واحد غير مستلم ولم يُحذف. لم يُستعد شيء.",
      "أُوقفت {run}: حُرّر عنصران غير مستلمين ولم يُحذفا. لم يُستعد شيء.",
      "أُوقفت {run}: حُرّرت {n} عناصر غير مستلمة ولم تُحذف. لم يُستعد شيء.",
      "أُوقفت {run}: حُرّر {n} عنصرًا غير مستلم ولم يُحذف. لم يُستعد شيء.",
      "أُوقفت {run}: حُرّر {n} عنصر غير مستلم ولم يُحذف. لم يُستعد شيء.",
    ],
  },
  impactNow: {
    en: [
      "Impact now: 1 item in the installation becomes due earlier ({w} in this workspace; computed {at}).",
      "Impact now: {n} items in the installation become due earlier, {w} of them in this workspace (computed {at}).",
    ],
    ko: "현재 영향: 설치 전체에서 {n}건이 더 일찍 삭제 대상이 되며, 그중 {w}건이 이 업무 공간의 항목입니다({at} 계산).",
    ar: [
      "الأثر الآن: لا يصبح أي عنصر في التثبيت مستحقًا مبكرًا، ومنها {w} في مساحة العمل هذه (حُسب {at}).",
      "الأثر الآن: يصبح عنصر واحد في التثبيت مستحقًا مبكرًا، ومنه {w} في مساحة العمل هذه (حُسب {at}).",
      "الأثر الآن: يصبح عنصران في التثبيت مستحقين مبكرًا، منهما {w} في مساحة العمل هذه (حُسب {at}).",
      "الأثر الآن: تصبح {n} عناصر في التثبيت مستحقة مبكرًا، منها {w} في مساحة العمل هذه (حُسب {at}).",
      "الأثر الآن: يصبح {n} عنصرًا في التثبيت مستحقًا مبكرًا، منها {w} في مساحة العمل هذه (حُسب {at}).",
      "الأثر الآن: يصبح {n} عنصر في التثبيت مستحقًا مبكرًا، منها {w} في مساحة العمل هذه (حُسب {at}).",
    ],
  },
  impactReview: {
    en: [
      "As of {at}, 1 item in the installation that is not due now would become due under this period ({w} in this workspace). Held and required items are included in the count and stay protected. Approval sends exactly these values and the digest above; the retention register refuses them if the impact has changed or is more than one hour old. Nothing is deleted: deletion plans are not connected.",
      "As of {at}, {n} items in the installation that are not due now would become due under this period, {w} of them in this workspace. Held and required items are included in the count and stay protected. Approval sends exactly these values and the digest above; the retention register refuses them if the impact has changed or is more than one hour old. Nothing is deleted: deletion plans are not connected.",
    ],
    ko: "{at} 기준으로 현재 삭제 대상이 아닌 항목 중 {n}건(설치 전체)이 이 기간에서는 삭제 대상이 되며, 그중 {w}건이 이 업무 공간의 항목입니다. 보류·참조 중인 항목도 건수에 포함되며 계속 보호됩니다. 승인하면 이 값과 위의 요약값을 그대로 보냅니다. 영향이 바뀌었거나 1시간이 지났으면 보관 레지스트리가 거부합니다. 삭제 계획이 연결되어 있지 않아 아무것도 삭제되지 않습니다.",
    ar: [
      "اعتبارًا من {at}، لن يصبح أي عنصر في التثبيت مستحقًا بموجب هذه المدة، ومنها {w} في مساحة العمل هذه. تشمل الأعداد العناصر المعلّقة والمطلوبة وتبقى محمية. ترسل الموافقة هذه القيم والملخص أعلاه كما هي، ويرفضها سجل الاحتفاظ إذا تغيّر الأثر أو مضت عليه أكثر من ساعة. لا يُحذف أي شيء: خطط الحذف غير متصلة.",
      "اعتبارًا من {at}، سيصبح عنصر واحد في التثبيت غير مستحق الآن مستحقًا بموجب هذه المدة، ومنه {w} في مساحة العمل هذه. تشمل الأعداد العناصر المعلّقة والمطلوبة وتبقى محمية. ترسل الموافقة هذه القيم والملخص أعلاه كما هي، ويرفضها سجل الاحتفاظ إذا تغيّر الأثر أو مضت عليه أكثر من ساعة. لا يُحذف أي شيء: خطط الحذف غير متصلة.",
      "اعتبارًا من {at}، سيصبح عنصران في التثبيت غير مستحقين الآن مستحقين بموجب هذه المدة، منهما {w} في مساحة العمل هذه. تشمل الأعداد العناصر المعلّقة والمطلوبة وتبقى محمية. ترسل الموافقة هذه القيم والملخص أعلاه كما هي، ويرفضها سجل الاحتفاظ إذا تغيّر الأثر أو مضت عليه أكثر من ساعة. لا يُحذف أي شيء: خطط الحذف غير متصلة.",
      "اعتبارًا من {at}، ستصبح {n} عناصر في التثبيت غير مستحقة الآن مستحقة بموجب هذه المدة، منها {w} في مساحة العمل هذه. تشمل الأعداد العناصر المعلّقة والمطلوبة وتبقى محمية. ترسل الموافقة هذه القيم والملخص أعلاه كما هي، ويرفضها سجل الاحتفاظ إذا تغيّر الأثر أو مضت عليه أكثر من ساعة. لا يُحذف أي شيء: خطط الحذف غير متصلة.",
      "اعتبارًا من {at}، سيصبح {n} عنصرًا في التثبيت غير مستحق الآن مستحقًا بموجب هذه المدة، منها {w} في مساحة العمل هذه. تشمل الأعداد العناصر المعلّقة والمطلوبة وتبقى محمية. ترسل الموافقة هذه القيم والملخص أعلاه كما هي، ويرفضها سجل الاحتفاظ إذا تغيّر الأثر أو مضت عليه أكثر من ساعة. لا يُحذف أي شيء: خطط الحذف غير متصلة.",
      "اعتبارًا من {at}، سيصبح {n} عنصر في التثبيت غير مستحق الآن مستحقًا بموجب هذه المدة، منها {w} في مساحة العمل هذه. تشمل الأعداد العناصر المعلّقة والمطلوبة وتبقى محمية. ترسل الموافقة هذه القيم والملخص أعلاه كما هي، ويرفضها سجل الاحتفاظ إذا تغيّر الأثر أو مضت عليه أكثر من ساعة. لا يُحذف أي شيء: خطط الحذف غير متصلة.",
    ],
  },
  impactAtProposal: {
    en: ["Impact when proposed (installation-wide): 1 item as of {at}", "Impact when proposed (installation-wide): {n} items as of {at}"],
    ko: "제안 시점 영향(설치 전체): {at} 기준 {n}건",
    ar: [
      "الأثر عند الاقتراح (على مستوى التثبيت): لا عناصر اعتبارًا من {at}",
      "الأثر عند الاقتراح (على مستوى التثبيت): عنصر واحد اعتبارًا من {at}",
      "الأثر عند الاقتراح (على مستوى التثبيت): عنصران اعتبارًا من {at}",
      "الأثر عند الاقتراح (على مستوى التثبيت): {n} عناصر اعتبارًا من {at}",
      "الأثر عند الاقتراح (على مستوى التثبيت): {n} عنصرًا اعتبارًا من {at}",
      "الأثر عند الاقتراح (على مستوى التثبيت): {n} عنصر اعتبارًا من {at}",
    ],
  },
  notPreservedPart: {
    en: ["1 item already deleted", "{n} items already deleted"],
    ko: "이미 삭제된 항목 {n}건",
    ar: ["{n} عنصر محذوف بالفعل", "عنصرًا واحدًا محذوفًا بالفعل", "عنصرين محذوفين بالفعل", "{n} عناصر محذوفة بالفعل", "{n} عنصرًا محذوفًا بالفعل", "{n} عنصر محذوف بالفعل"],
  },
  partlyPart: {
    en: ["1 item partly deleted", "{n} items partly deleted"],
    ko: "일부 삭제된 항목 {n}건",
    ar: ["{n} عنصر محذوف جزئيًا", "عنصرًا واحدًا محذوفًا جزئيًا", "عنصرين محذوفين جزئيًا", "{n} عناصر محذوفة جزئيًا", "{n} عنصرًا محذوفًا جزئيًا", "{n} عنصر محذوف جزئيًا"],
  },
  pendingPart: {
    en: ["1 item has a deletion outcome pending.", "{n} items have a deletion outcome pending."],
    ko: "{n}건은 삭제 결과가 대기 중입니다.",
    ar: [
      "لا يوجد عنصر نتيجة حذفه معلّقة.",
      "عنصر واحد نتيجة حذفه معلّقة.",
      "عنصران نتيجة حذفهما معلّقة.",
      "{n} عناصر نتيجة حذفها معلّقة.",
      "{n} عنصرًا نتيجة حذفها معلّقة.",
      "{n} عنصر نتيجة حذفها معلّقة.",
    ],
  },
  window: {
    en: ["Only the last deletion record of the installation is scanned; {t} of them are in this workspace.", "Only the last {n} deletion records of the installation are scanned; {t} of them are in this workspace."],
    ko: "설치의 최근 삭제 기록 {n}건만 조회하며, 그중 {t}건이 이 업무 공간의 기록입니다.",
    ar: [
      "لا يُفحص أي سجل حذف للتثبيت؛ ومنها {t} في مساحة العمل هذه.",
      "يُفحص آخر سجل حذف للتثبيت فقط؛ ومنه {t} في مساحة العمل هذه.",
      "يُفحص آخر سجلَّي حذف للتثبيت فقط؛ منهما {t} في مساحة العمل هذه.",
      "تُفحص آخر {n} سجلات حذف للتثبيت فقط؛ منها {t} في مساحة العمل هذه.",
      "يُفحص آخر {n} سجلًا للحذف في التثبيت فقط؛ منها {t} في مساحة العمل هذه.",
      "يُفحص آخر {n} سجل حذف للتثبيت فقط؛ منها {t} في مساحة العمل هذه.",
    ],
  },
} satisfies Record<string, Plural>;

export const allWordTables: Record<string, Words> = {
  classWords, standingWords, accessWords, holdOutcomeWords, storeResultWords, runStateWords, runStateNotes, runOutcomeWords, holdStateWords, originWords, connectionWords,
  ledgerWords, decisionWords, populationWords, proposalKindWords, proposalStateWords, resultWords, summaryWords, blockerWords, placerWords, stateSentenceWords, narrowWords,
  keptWords,
};
