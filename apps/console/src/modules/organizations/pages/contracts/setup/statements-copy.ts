import type { Locale } from "../../../../../shared/types";
import type { SetupReference } from "./attestation";
import { stepError } from "./step-copy";

const pick = (l: Locale, v: readonly [string, string, string]) => v[l === "ko" ? 1 : l === "ar" ? 2 : 0];
const words = {
 purpose: ["Accountable claims written by the people who acted. A statement is its author's claim, not proof.", "행위를 한 담당자가 직접 작성하고 책임지는 진술입니다. 진술은 작성자의 주장이며 증명이 아닙니다.", "إفادات يتحمل مسؤوليتها الأشخاص الذين قاموا بالإجراء. الإفادة قول صاحبها وليست إثباتاً."],
 assurance: ["Written by a person. This service does not check what a statement says.", "사람이 작성했습니다. 이 서비스는 진술 내용을 확인하지 않습니다.", "كتبها شخص. لا تتحقق هذه الخدمة مما تقوله الإفادة."],
 total: ["Statements in scope of these settings", "이 설정 범위의 진술", "الإفادات ضمن نطاق هذه الإعدادات"],
 showing: ["Rows on this page", "현재 페이지 행 수", "صفوف هذه الصفحة"],
 empty: ["No statement is recorded for these settings.", "이 설정에 기록된 진술이 없습니다.", "لا توجد إفادة مسجلة لهذه الإعدادات."],
 author: ["Written by", "작성자", "كتبها"], recorded: ["Recorded", "기록 시각", "وقت التسجيل"], step: ["Statement for", "진술 대상 단계", "الإفادة عن"], names: ["Record named", "지목한 기록", "السجل المسمّى"], state: ["State", "상태", "الحالة"],
 channel_registration: ["Channel registration", "채널 등록", "تسجيل القناة"], channel_admission: ["Channel acceptance", "채널 수락", "قبول القناة"],
 change_instruction: ["Change instruction", "변경 지시", "تعليمات التغيير"], change_verification: ["Change verification", "변경 검증", "التحقق من التغيير"],
 channel: ["Channel", "채널", "القناة"], change: ["Change", "변경", "التغيير"],
 unconsumed: ["Not used — supports no record", "미사용 — 어떤 기록도 뒷받침하지 않음", "غير مستخدمة — لا تدعم أي سجل"],
 consumed: ["Used", "사용됨", "مستخدمة"],
 withdrawn: ["Withdrawn — supports no record", "철회됨 — 어떤 기록도 뒷받침하지 않음", "مسحوبة — لا تدعم أي سجل"],
 text: ["Statement text", "진술 내용", "نص الإفادة"],
 locator: ["Where the source is kept", "원본 보관 위치", "مكان حفظ المصدر"],
 noLocator: ["Not stated", "기재되지 않음", "غير مذكور"],
 action: ["Action", "작업", "الإجراء"],
 actAttest: ["Record statement", "진술 기록", "تسجيل الإفادة"],
 actWithdraw: ["Withdraw statement", "진술 철회", "سحب الإفادة"],
 targetStatement: ["Statement", "진술", "الإفادة"],
 fieldStatement: ["Your statement", "진술", "إفادتك"],
 hintStatement: ["Describe what you did. Do not enter account, contact or identifier values.", "수행한 내용을 적으세요. 계좌, 연락처, 식별자 값은 입력하지 마세요.", "صف ما قمت به. لا تُدخل قيم حسابات أو جهات اتصال أو معرّفات."],
 fieldLocator: ["Where the source is kept (optional)", "원본 보관 위치(선택)", "مكان حفظ المصدر (اختياري)"],
 hintLocator: ["A place outside this product, for example a folder.", "이 제품 밖의 위치입니다. 예: 폴더.", "مكان خارج هذا المنتج، مثل مجلد."],
 groupStatement: ["Statement you will record", "기록할 진술", "الإفادة التي ستسجّلها"],
 groupReuse: ["Statement you recorded in this session", "이 세션에서 기록한 진술", "الإفادة التي سجّلتها في هذه الجلسة"],
 groupAction: ["Step it is written for", "진술 대상 단계", "الخطوة التي كُتبت لها"],
 twoRequests: ["Two requests are sent. The statement is recorded first and stays on record even if the action is refused. It can be withdrawn only while unused.", "요청을 두 번 전송합니다. 진술을 먼저 기록하며, 단계가 거절되어도 진술은 기록으로 남습니다. 사용되지 않은 동안에만 철회할 수 있습니다.", "يُرسل طلبان. تُسجَّل الإفادة أولاً وتبقى في السجل حتى لو رُفض الإجراء. لا يمكن سحبها إلا ما دامت غير مستخدمة."],
 oneRequest: ["One request is sent. The statement is already on record and was unused when this session recorded it.", "요청을 한 번 전송합니다. 진술은 이미 기록되어 있으며, 이 세션에서 기록할 당시 사용되지 않은 상태였습니다.", "يُرسل طلب واحد. الإفادة مسجلة سلفاً وكانت غير مستخدمة عند تسجيلها في هذه الجلسة."],
 noticeUsed: ["A statement used by a step stays used.", "단계에 사용된 진술은 사용된 상태로 남습니다.", "الإفادة التي استخدمتها خطوة تبقى مستخدمة."],
 noticeWithdraw: ["A withdrawn statement stays on record as withdrawn and cannot be used.", "철회한 진술은 철회됨 상태로 기록에 남으며 사용할 수 없습니다.", "تبقى الإفادة المسحوبة في السجل بحالة مسحوبة ولا يمكن استخدامها."],
 notUsedNote: ["The statement recorded for this step was not used by this request. It is sent again with the same step in this session, or can be withdrawn under Statements.", "이 단계를 위해 기록한 진술은 이 요청에서 사용되지 않았습니다. 이 세션에서 같은 단계를 다시 보내면 함께 전송되며, 진술 항목에서 철회할 수도 있습니다.", "الإفادة المسجلة لهذه الخطوة لم يستخدمها هذا الطلب. تُرسل مجدداً مع الخطوة نفسها في هذه الجلسة، أو يمكن سحبها من قسم الإفادات."],
 unknownUseNote: ["Whether the statement recorded for this step was used is not known.", "이 단계를 위해 기록한 진술이 사용되었는지는 확인되지 않았습니다.", "لا يُعرف هل استُخدمت الإفادة المسجلة لهذه الخطوة."],
 stepNotSent: ["The step this statement was written for has not been sent. Settle this request first, then start the step again.", "이 진술의 대상 단계는 아직 전송되지 않았습니다. 이 요청을 먼저 해결한 뒤 단계를 다시 시작하세요.", "الخطوة التي كُتبت لها هذه الإفادة لم تُرسل بعد. احسم هذا الطلب أولاً ثم ابدأ الخطوة من جديد."],
 stepNotSentRefused: ["The step this statement was written for was not sent.", "이 진술의 대상 단계는 전송되지 않았습니다.", "الخطوة التي كُتبت لها هذه الإفادة لم تُرسل."],
 usedInSession: ["Used by a step recorded in this session", "이 세션에서 기록된 단계에 사용됨", "استُخدمت في خطوة مسجلة في هذه الجلسة"],
 stateWhenRecorded: ["State when this request was recorded", "이 요청 기록 시점의 상태", "الحالة عند تسجيل هذا الطلب"],
 groupFields: ["Statement recorded with this step", "이 단계와 함께 기록하는 진술", "الإفادة المسجلة مع هذه الخطوة"],
 errSelf: ["The person who registered the channel or requested the change cannot write the statement of the next person's step.", "채널을 등록했거나 변경을 요청한 사람은 다음 담당자 단계의 진술을 작성할 수 없습니다.", "لا يمكن لمن سجّل القناة أو طلب التغيير أن يكتب إفادة خطوة الشخص التالي."],
 errShape: ["The service refused the request as written. The text may contain characters it does not accept.", "서비스가 작성된 요청을 거절했습니다. 텍스트에 허용되지 않는 문자가 있을 수 있습니다.", "رفضت الخدمة الطلب كما كُتب. قد يحتوي النص على محارف لا تقبلها."],
 ownOnly: ["Only its author can withdraw a statement, and only while it is unused.", "진술은 작성자만, 사용되지 않은 동안에만 철회할 수 있습니다.", "لا يسحب الإفادة إلا كاتبها، وما دامت غير مستخدمة."],
 statementResults: ["Statement requests sent in this session", "이 세션에서 전송한 진술 요청", "طلبات الإفادات المرسلة في هذه الجلسة"],
 errPermission: ["Recording or withdrawing a statement may need the permission of the step it is written for, only its author can withdraw it, or the service refused for a reason it does not disclose.", "진술을 기록하거나 철회하려면 해당 단계의 권한이 필요할 수 있고, 철회는 작성자만 할 수 있으며, 서비스가 밝히지 않는 사유로 거절했을 수도 있습니다.", "قد يتطلب تسجيل الإفادة أو سحبها صلاحية الخطوة التي كُتبت لها، ولا يسحبها إلا كاتبها، أو أن الخدمة رفضت لسبب لا تفصح عنه."],
 errStale: ["Possible causes: the record the statement names changed, or it is not in the state this step needs. Reload and review again.", "가능한 원인: 진술이 지목한 기록이 변경됨, 기록이 이 단계에 필요한 상태가 아님. 새로 조회한 뒤 다시 검토하세요.", "الأسباب المحتملة: تغير السجل الذي تسميه الإفادة، أو أنه ليس في الحالة التي تتطلبها هذه الخطوة. أعد التحميل ثم راجع من جديد."],
 errPrerequisite: ["Possible causes: you already hold ten unused statements, the total limit of statements was reached, the statement is already used or withdrawn, or the selected settings are no longer confirmed. The service does not say which.", "가능한 원인: 미사용 진술을 이미 10건 보유함, 진술 전체 한도 도달, 진술이 이미 사용되었거나 철회됨, 선택한 설정이 더 이상 확인 완료 상태가 아님. 서비스는 어느 것인지 알려 주지 않습니다.", "الأسباب المحتملة: لديك عشر إفادات غير مستخدمة، أو بُلغ الحد الإجمالي للإفادات، أو أن الإفادة مستخدمة أو مسحوبة، أو لم تعد الإعدادات المحددة مؤكدة. لا تحدد الخدمة أيها."],
 errText: ["Write a statement of at most 2000 bytes without control characters.", "제어 문자가 없는 2000바이트 이하의 진술을 작성하세요.", "اكتب إفادة لا تتجاوز 2000 بايت وبلا محارف تحكم."],
 samples: ["Sample references", "샘플 참조", "المراجع التجريبية"],
 samplesNote: ["Protected settings and channels are chosen from these named samples. They stand for no real account, address or template.", "보호된 설정과 채널은 이름이 붙은 이 샘플 중에서 선택합니다. 실제 계좌, 주소, 템플릿을 나타내지 않습니다.", "تُختار الإعدادات المحمية والقنوات من هذه النماذج المسماة. لا تمثل حساباً أو عنواناً أو قالباً حقيقياً."],
 use_document_recipient: ["Document recipient", "문서 수신자", "مستلم المستند"], use_document_template: ["Document template", "문서 템플릿", "قالب المستند"], use_payment_account: ["Payment account", "결제 계좌", "حساب الدفع"],
 use_channel_customer: ["Verification channel of a customer", "고객의 검증 채널", "قناة تحقق لعميل"], use_channel_document_kind: ["Verification channel of a document kind", "문서 종류의 검증 채널", "قناة تحقق لنوع مستند"], use_channel_entity: ["Verification channel of an issuing entity", "발행 법인의 검증 채널", "قناة تحقق لجهة مصدرة"],
 use_unregistered_sender: ["Instruction from a sender that is not a registered channel", "등록된 채널이 아닌 발신자의 지시", "تعليمات من مرسل ليس قناة مسجلة"],
} as const;
export type StatementWord = keyof typeof words;
export function statementText(l: Locale, key: StatementWord): string { return pick(l, words[key]); }

const samples: Record<SetupReference["use"], readonly [string, string, string]> = {
 document_recipient: ["Sample recipient", "샘플 수신자", "مستلم تجريبي"],
 document_template: ["Sample document template", "샘플 문서 템플릿", "قالب مستند تجريبي"],
 payment_account: ["Sample payment account", "샘플 결제 계좌", "حساب دفع تجريبي"],
 channel_customer: ["Sample customer channel", "샘플 고객 채널", "قناة عميل تجريبية"],
 channel_document_kind: ["Sample document kind channel", "샘플 문서 종류 채널", "قناة نوع مستند تجريبية"],
 channel_entity: ["Sample issuing entity channel", "샘플 발행 법인 채널", "قناة جهة مصدرة تجريبية"],
 unregistered_sender: ["Sample unregistered sender", "샘플 미등록 발신자", "مرسل تجريبي غير مسجل"],
};
export function sampleLabel(l: Locale, entry: Pick<SetupReference, "code" | "use">): string {
 const letter = entry.use === "unregistered_sender" ? "" : /_([a-z])$/.exec(entry.code)?.[1].toUpperCase() ?? "";
 return pick(l, samples[entry.use]) + (letter ? " " + letter : "");
}

export function statementError(l: Locale, code: string): string {
 const key: StatementWord | "" = code === "permission_denied" ? "errPermission" : code === "evidence_stale" || code === "revision_conflict" ? "errStale" : code === "prerequisite_failed" ? "errPrerequisite" : code === "self_approval_denied" ? "errSelf" : code === "invalid_request" ? "errShape" : code === "invalid_text" ? "errText" : "";
 return key ? pick(l, words[key]) : stepError(l, code);
}
