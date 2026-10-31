import {esc} from './widgets.js';
const copy={
 "title": [
  "Contribute",
  "기여",
  "المساهمة"
 ],
 "intro": [
  "Individuals, companies and organizations can sponsor at any tier. Sponsor type is separate from the monthly tier.",
  "개인·기업·단체 모두 원하는 등급으로 후원할 수 있습니다. 후원 유형과 월 후원 등급은 별개입니다.",
  "يمكن للأفراد والشركات والمؤسسات الدعم بأي مستوى. نوع الداعم مستقل عن مستوى الدعم الشهري."
 ],
 "tiers": [
  "Monthly sponsorship",
  "월 후원",
  "الدعم الشهري"
 ],
 "draft": [
  "Design preview: these agreed tiers have not been verified as published on GitHub Sponsors. Check the terms on GitHub before sponsoring.",
  "시안: 합의한 등급이며 GitHub Sponsors 등록 여부는 미확인입니다. 후원 전 GitHub에 표시된 조건을 확인하세요.",
  "معاينة تصميم: لم يتم التحقق من نشر هذه المستويات المتفق عليها على GitHub Sponsors. راجع الشروط على GitHub قبل الدعم."
 ],
 "month": [
  "per month",
  "매월",
  "شهرياً"
 ],
 "terms": [
  "Benefits concern development priority, not exclusive access. Adopted work goes to the public branch. Requests are assessed for product fit, security and effort; delivery is not guaranteed. Sponsorship does not grant a commercial license or an SLA.",
  "혜택은 개발 우선순위에 관한 것입니다. 결과물은 공개 브랜치에 반영합니다. 요청은 제품 방향·보안·개발 규모에 따라 채택하며 구현을 보장하지 않습니다. 상용 라이선스·SLA는 포함하지 않습니다.",
  "تتعلق المزايا بأولوية التطوير وليس بالوصول الحصري. تُنشر النتائج في الفرع العام. تُقيّم الطلبات حسب ملاءمة المنتج والأمان والجهد؛ ولا يُضمن تنفيذها. لا يمنح الدعم ترخيصاً تجارياً أو اتفاقية مستوى خدمة."
 ],
 "sponsor": [
  "View GitHub Sponsors",
  "GitHub Sponsors에서 후원하기",
  "عرض GitHub Sponsors"
 ],
 "roster": [
  "Sponsors",
  "후원자",
  "الداعمون"
 ],
 "empty": [
  "Sponsor roster not connected. No verified public listings have been loaded into this preview.",
  "후원 명단 미연결. 이 시안에는 검증된 공개 명단을 불러오지 않았습니다.",
  "قائمة الداعمين غير متصلة. لم يتم تحميل قائمة عامة موثقة في هذه المعاينة."
 ],
 "people": [
  "Individuals",
  "개인 후원자",
  "الأفراد"
 ],
 "companies": [
  "Companies and organizations",
  "기업·단체 후원자",
  "الشركات والمؤسسات"
 ],
 "peopleInfo": [
  "With consent: display name and approved profile link.",
  "동의한 표시명·프로필 링크를 게재합니다.",
  "بالموافقة: اسم العرض ورابط الملف المعتمد."
 ],
 "companyInfo": [
  "With authorized consent: company or organization name, approved logo and website. Contacts and payment details remain private.",
  "권한 있는 담당자의 동의를 받아 회사·단체명, 승인된 로고·홈페이지를 게재합니다. 담당자·결제 정보는 공개하지 않습니다.",
  "بموافقة مخولة: اسم الشركة أو المؤسسة وشعارها المعتمد وموقعها. تبقى بيانات الاتصال والدفع خاصة."
 ],
 "credit": [
  "Contributors",
  "작업 기여자",
  "المساهمون"
 ],
 "creditInfo": [
  "Code, documentation, design, translation and testing contributions are credited through their work evidence, independently of sponsorship.",
  "코드·문서·디자인·번역·테스트 기여는 후원 여부와 관계없이 작업 근거로 인정합니다.",
  "تُقدّر مساهمات الشفرة والوثائق والتصميم والترجمة والاختبار بأدلة العمل، بصورة مستقلة عن الدعم المالي."
 ],
 "contribute": [
  "Contribution guide",
  "기여 방법",
  "دليل المساهمة"
 ],
 "history": [
  "Repository contributors",
  "저장소 기여자 보기",
  "مساهمو المستودع"
 ],
 "manage": [
  "Request a listing, correction or removal",
  "명단 등재·수정·삭제 요청",
  "طلب الإدراج أو التصحيح أو الإزالة"
 ],
 "manageInfo": [
  "Contact the maintainer with the desired public name and link. Do not send payment details. Company logos require permission from an authorized representative.",
  "원하는 공개 이름·링크를 관리자에게 알려주세요. 결제 정보는 보내지 마세요. 기업 로고는 권한 있는 담당자의 사용 허가를 확인합니다.",
  "أرسل للمشرف اسم العرض والرابط المطلوبين دون بيانات الدفع. يتطلب شعار الشركة إذناً من ممثل مخول."
 ]
};
const tiers=[["Supporter", 5, ["Optional public recognition and development updates.", "선택적 후원자 등재, 개발 소식", "إدراج اختياري ضمن الداعمين وأخبار التطوير."]], ["Backer", 25, ["Supporter benefits plus roadmap opinions and surveys.", "Supporter 혜택 + 로드맵 의견·설문 참여", "مزايا Supporter مع المشاركة في الآراء والاستبيانات حول خارطة الطريق."]], ["Team", 100, ["Backer benefits plus priority review of feature and improvement requests, with review results.", "Backer 혜택 + 기능·개선 요청 우선 검토 및 결과 안내", "مزايا Backer مع أولوية مراجعة طلبات الميزات والتحسينات وإبلاغ نتائج المراجعة."]], ["Organization", 300, ["Team benefits plus development priority for adopted features and compatibility improvements.", "Team 혜택 + 채택된 기능·호환성 개선의 개발 우선순위 반영", "مزايا Team مع أولوية تطوير الميزات المعتمدة وتحسينات التوافق."]], ["Partner", 1000, ["Organization benefits plus roadmap discussion and discussion of agreed feature scope and delivery schedules.", "Organization 혜택 + 로드맵 협의, 합의한 기능의 개발·적용 일정 협의", "مزايا Organization مع مناقشة خارطة الطريق ونطاق الميزات المتفق عليها وجداول تطويرها وتطبيقها."]]];
export function community(locale){
 const index=locale==='ko'?1:locale==='ar'?2:0;
 const c=key=>esc(copy[key][index]);
 return `<section class="panel"><h2>${c('tiers')}</h2><p>${c('intro')}</p><p class="notice">${c('draft')}</p><div class="screen-index">${tiers.map(([name,amount,benefits])=>`<article class="panel"><h3><bdi>${name}</bdi></h3><p><bdi dir="ltr">${amount} USD</bdi> · ${c('month')}</p><p>${esc(benefits[index])}</p></article>`).join('')}</div><p>${c('terms')}</p><a href="https://github.com/sponsors/myblueton-sw">${c('sponsor')}</a></section>
 <section class="panel"><h2>${c('roster')}</h2><p class="notice">${c('empty')}</p><div class="screen-index"><article><h3>${c('people')}</h3><p>${c('peopleInfo')}</p></article><article><h3>${c('companies')}</h3><p>${c('companyInfo')}</p></article></div><p>${c('manageInfo')}</p><a href="mailto:myblueton@gmail.com?subject=Hybrid%20Billing%20sponsor%20listing">${c('manage')}</a></section>
 <section class="panel"><h2>${c('credit')}</h2><p>${c('creditInfo')}</p><div class="related"><a href="https://github.com/myblueton-sw/hybrid_billing/blob/main/CONTRIBUTING.md">${c('contribute')}</a><a href="https://github.com/myblueton-sw/hybrid_billing/graphs/contributors">${c('history')}</a></div></section>`;
}
