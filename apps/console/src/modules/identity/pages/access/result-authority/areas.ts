import type {Locale} from "../../../../../shared/types";
import {frozen,issuanceDomains,resultDomain,type AuthorityDomain} from "./contract";
type Words=readonly [string,string,string];
export type AuthorityArea=Readonly<{param:string;item:string;domain:AuthorityDomain;title:Words;intro:Words;refusal:Words;capabilities:Readonly<Record<string,Words>>}>;
const intro=(en:string,ko:string,ar:string):Words=>[
 `Manage proposals for console permission ceilings and individual permissions for ${en}. Each permission is time-limited and is granted only after another eligible governor approves the proposal. This is a synthetic internal workflow.`,
 `${ko}에 대한 콘솔 권한 상한과 사용자별 권한 제안을 관리합니다. 권한에는 유효기간이 있으며, 다른 적격 관리자가 제안을 승인해야 부여됩니다. 합성 데이터 기반의 내부 절차입니다.`,
 `أدر مقترحات سقوف صلاحيات وحدة التحكم وصلاحيات الأفراد الخاصة ب${ar}. لكل صلاحية مدة محددة، ولا تُمنح إلا بعد أن يعتمد المقترح مسؤول آخر مؤهل. هذا مسار داخلي ببيانات اصطناعية.`];
const refusal=(en:string,ko:string,ar:string):Words=>[`This client cannot be provisioned through console ${en}.`,`이 클라이언트는 콘솔 ${ko} 부여 대상이 아닙니다.`,`لا يمكن منح هذا العميل ${ar} الخاصة بوحدة التحكم.`];
export const resultArea:AuthorityArea=frozen({param:"result",item:"result-authority",domain:resultDomain,
 title:["ERP result permissions","ERP 결과 권한","صلاحيات نتائج ERP"],
 intro:["Manage proposals for console permission ceilings and individual result permissions. This synthetic internal workflow does not send commands to an external ERP.","콘솔 권한 상한과 사용자별 결과 권한 제안을 관리합니다. 합성 데이터 기반의 내부 절차이며 외부 ERP로 명령을 전송하지 않습니다.","أدر مقترحات سقوف صلاحيات وحدة التحكم وصلاحيات النتائج للأفراد. هذا مسار داخلي ببيانات اصطناعية ولا يرسل أوامر إلى نظام ERP خارجي."],
 refusal:["This client cannot be provisioned through console result permissions.","이 클라이언트는 콘솔 결과 권한 부여 대상이 아닙니다.","لا يمكن منح هذا العميل صلاحيات النتائج الخاصة بوحدة التحكم."],
 capabilities:{propose:["Propose result evidence","결과 근거 제안","اقتراح أدلة النتيجة"],confirm:["Confirm result evidence","결과 근거 확인","تأكيد أدلة النتيجة"],revoke:["Revoke result evidence","결과 근거 철회","إلغاء أدلة النتيجة"],apply:["Apply ERP result evidence internally","ERP 결과 근거 내부 반영","تطبيق أدلة نتائج ERP داخلياً"],inspect:["Inspect result evidence","결과 근거 조회","عرض أدلة النتيجة"]}});
const issuance:AuthorityArea[]=[
 {param:"issuance-protected",item:"issuance-protected-authority",domain:issuanceDomains.protected,
  title:["Protected issuance settings permissions","보호된 발행 설정 권한","صلاحيات إعدادات الإصدار المحمية"],
  intro:intro("protected issuance settings","보호된 발행 설정","إعدادات الإصدار المحمية"),refusal:refusal("protected issuance settings permissions","보호된 발행 설정 권한","صلاحيات إعدادات الإصدار المحمية"),
  capabilities:{register_channel:["Register verification channel","검증 채널 등록","تسجيل قناة التحقق"],admit_channel:["Accept verification channel","검증 채널 수락","قبول قناة التحقق"],request_change:["Request protected settings change","보호된 설정 변경 요청","طلب تغيير الإعدادات المحمية"],verify_change:["Verify protected settings change","보호된 설정 변경 검증","التحقق من تغيير الإعدادات المحمية"],approve_change:["Approve protected settings change","보호된 설정 변경 승인","اعتماد تغيير الإعدادات المحمية"],inspect:["Inspect protected settings evidence","보호된 설정 근거 조회","عرض أدلة الإعدادات المحمية"]}},
 {param:"issuance-terms",item:"issuance-terms-authority",domain:issuanceDomains.term,
  title:["Payment terms permissions","납부 조건 권한","صلاحيات شروط الدفع"],
  intro:intro("payment terms","납부 조건","شروط الدفع"),refusal:refusal("payment terms permissions","납부 조건 권한","صلاحيات شروط الدفع"),
  capabilities:{record:["Record payment terms","납부 조건 기록","تسجيل شروط الدفع"],approve:["Approve payment terms","납부 조건 승인","اعتماد شروط الدفع"],bind:["Apply payment terms to a contract","계약에 납부 조건 적용","تطبيق شروط الدفع على عقد"],approve_binding:["Approve applying payment terms to a contract","계약에 납부 조건 적용 승인","اعتماد تطبيق شروط الدفع على عقد"],inspect:["Inspect payment terms evidence","납부 조건 근거 조회","عرض أدلة شروط الدفع"]}},
 {param:"issuance-drafts",item:"issuance-drafts-authority",domain:issuanceDomains.draft,
  title:["Issuance draft permissions","발행 초안 권한","صلاحيات مسودات الإصدار"],
  intro:intro("issuance drafts","발행 초안","مسودات الإصدار"),refusal:refusal("issuance draft permissions","발행 초안 권한","صلاحيات مسودات الإصدار"),
  capabilities:{submit:["Submit issuance draft for approval","발행 초안 승인 요청","إرسال مسودة الإصدار للاعتماد"],approve:["Approve issuance draft","발행 초안 승인","اعتماد مسودة الإصدار"],inspect:["Inspect issuance draft evidence","발행 초안 근거 조회","عرض أدلة مسودة الإصدار"]}},
];
export const issuanceAreas:readonly AuthorityArea[]=frozen(issuance);
export const authorityAreas:readonly AuthorityArea[]=frozen([resultArea,...issuanceAreas]);
export const areaOf=(param:string|null):AuthorityArea|undefined=>authorityAreas.find(area=>area.param===param);
export const words=(l:Locale,value:Words)=>l==="ko"?value[1]:l==="ar"?value[2]:value[0];
export const capabilityLabel=(l:Locale,area:AuthorityArea,value:string)=>{const found=Object.hasOwn(area.capabilities,value)?area.capabilities[value]:undefined;return found?words(l,found):value;};
