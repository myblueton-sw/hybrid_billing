import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export const pc = {
  hint: (l: Locale) => t(l, "Search by login or user ID.", "로그인 ID 또는 사용자 ID로 검색합니다.", "ابحث باسم الدخول أو معرّف المستخدم."),
  search: (l: Locale) => t(l, "Search", "검색", "بحث"),
  searching: (l: Locale) => t(l, "Searching…", "검색하는 중…", "جارٍ البحث…"),
  count: (l: Locale, n: number) => t(l, n === 1 ? "1 person" : `${n} people`, `${n}명`, `عدد الأشخاص: ${n}`),
  firstPage: (l: Locale, n: number) => t(l, `Showing the first ${n} by user ID. Search to find others.`, `사용자 ID순으로 처음 ${n}명을 표시합니다. 다른 사람은 검색하세요.`, `يُعرض أول ${n} حسب معرّف المستخدم. ابحث للعثور على غيرهم.`),
  refine: (l: Locale) => t(l, "More people match than are shown. Type more of the login or user ID.", "표시된 것보다 일치하는 사람이 더 있습니다. 로그인 ID나 사용자 ID를 더 입력하세요.", "يطابق أشخاص أكثر مما هو معروض. اكتب المزيد من اسم الدخول أو معرّف المستخدم."),
  noMatch: (l: Locale, term: string) => t(l, `No one matches “${term}”.`, `“${term}”와 일치하는 사람이 없습니다.`, `لا يطابق «${term}» أحدًا.`),
  onlyYou: (l: Locale, term: string) => t(l, `Only you match “${term}”. You can't request access for yourself.`, `“${term}”와 일치하는 사람은 본인뿐입니다. 자신에 대한 권한은 요청할 수 없습니다.`, `لا يطابق «${term}» غيرك، ولا يمكنك طلب صلاحية لنفسك.`),
  failed: (l: Locale) => t(l, "People could not be searched.", "사람을 검색하지 못했습니다.", "تعذّر البحث عن الأشخاص."),
  kept: (l: Locale) => t(l, "The person you chose is kept.", "선택한 사람은 그대로입니다.", "يبقى الشخص الذي اخترته كما هو."),
  change: (l: Locale) => t(l, "Change", "변경", "تغيير"),
  keep: (l: Locale) => t(l, "Keep this person", "이 사람 유지", "الإبقاء على هذا الشخص"),
  results: (l: Locale) => t(l, "Matching people", "검색된 사람", "الأشخاص المطابقون"),
  requestUnconfirmed: (l: Locale) => t(l, "A revocation for this person is not confirmed; see the Revocations section.", "이 사람에 대한 회수가 확정되지 않았습니다. '회수' 섹션을 확인하세요.", "إبطال يخص هذا الشخص غير مؤكَّد؛ راجع قسم «الإبطال»."),
};
