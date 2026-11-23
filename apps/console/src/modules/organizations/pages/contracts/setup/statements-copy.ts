import type { Locale } from "../../../../../shared/types";
import type { SetupReference } from "./attestation";

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
