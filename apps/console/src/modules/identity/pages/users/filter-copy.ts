import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { counted } from "../../../../shared/plural";
import { tabLabel } from "../../../../planning/copy";
import screens from "../../screens.json";

const m01Tabs = (screens as { id: string; tabs?: { key: string; labelKo: string }[] }[]).find((s) => s.id === "M01")?.tabs ?? [];

export function tabName(l: Locale, key: string): string {
  return tabLabel(l, key, m01Tabs.find((x) => x.key === key)?.labelKo ?? key);
}

export type AuthMethods = { connected: string[]; not_connected: string[] };

function batchim(word: string): boolean {
  const base = word.replace(/\s*\([^)]*\)\s*$/, "").trim() || word.trim();
  const ch = base.slice(-1);
  const code = ch.charCodeAt(0);
  if (code >= 0xac00 && code <= 0xd7a3) return (code - 0xac00) % 28 !== 0;
  if (/[0-9]/.test(ch)) return "013678".includes(ch);
  return "LMNR".includes(ch.toUpperCase());
}

export const josa = (word: string, withFinal: string, withoutFinal: string) => word + (batchim(word) ? withFinal : withoutFinal);

const list = (l: Locale, items: string[]) => {
  if (items.length <= 1) return items.join("");
  const last = items[items.length - 1], rest = items.slice(0, -1);
  if (l === "ko") return `${rest.slice(0, -1).map((x) => x + ", ").join("")}${josa(rest[rest.length - 1], "과", "와")} ${last}`;
  return l === "ar" ? `${rest.join("، ")} و${last}` : `${rest.join(", ")} and ${last}`;
};

const capital = (text: string) => text.charAt(0).toUpperCase() + text.slice(1);

export const bare = (text: string) => text.replace(/[.。]\s*$/, "");

function methodLine(l: Locale, m: AuthMethods): string {
  const parts: string[] = [];
  const own = m.connected.filter((x) => x !== "password");
  if (m.connected.includes("password") && own.length === 0) {
    parts.push(t(l, "Everyone signs in with a password set at this installation.", "모든 사용자는 이 설치에서 설정한 비밀번호로 로그인합니다.", "يسجّل الجميع الدخول بكلمة مرور معيّنة في هذا التثبيت."));
  } else if (m.connected.length > 0) {
    const names = m.connected.map((x) => x.toUpperCase());
    parts.push(t(l, `Connected sign-in methods: ${list(l, names)}.`, `연결된 로그인 방식: ${list(l, names)}.`, `طرق تسجيل الدخول المتصلة: ${list(l, names)}.`));
  }
  const sso = ["saml", "oidc"].filter((x) => m.not_connected.includes(x)).map((x) => x.toUpperCase());
  const off: string[] = [];
  if (sso.length) off.push(t(l, `Single sign-on (${sso.join(", ")})`, `SSO(${sso.join("·")})`, `الدخول الموحد (${sso.join(" و")})`));
  if (m.not_connected.includes("ldap")) off.push(t(l, "directory sign-in (LDAP)", "디렉터리 로그인(LDAP)", "الدخول عبر الدليل (LDAP)"));
  const known = new Set(["saml", "oidc", "ldap"]);
  for (const x of m.not_connected.filter((x) => !known.has(x))) off.push(x.toUpperCase());
  if (off.length) {
    const names = list(l, off);
    const ar = off.length === 1 ? "غير متصل" : off.length === 2 ? "غير متصلَين" : "غير متصلة";
    parts.push(t(l, `${capital(names)} ${off.length > 1 ? "are" : "is"} not connected.`, `${josa(names, "은", "는")} 연결되어 있지 않습니다.`, `${names} ${ar}.`));
  }
  return parts.join(" ");
}

const people = (l: Locale, n: number) => {
  if (l === "en") return `${n}`;
  if (l === "ko") return `${n}명`;
  return `${n}`;
};

export const fc = {
  scopeLabel: (l: Locale) => t(l, "Workspace:", "업무 공간:", "مساحة العمل:"),
  filters: (l: Locale) => t(l, "Filters", "필터", "عوامل التصفية"),
  filtersApplied: (l: Locale, n: number) => t(l, `Filters · ${n} applied`, `필터 · ${n}개 적용`, `عوامل التصفية · المطبّقة: ${n}`),
  search: (l: Locale) => t(l, "Search by login, contact email or user ID", "로그인 ID·연락용 이메일·사용자 ID로 검색", "البحث باسم الدخول أو البريد الإلكتروني للتواصل أو معرّف المستخدم"),
  searchButton: (l: Locale) => t(l, "Search", "검색", "بحث"),
  searchTooLong: (l: Locale) => t(l, "The search text is too long. Shorten it and search again.", "검색어가 너무 깁니다. 줄여서 다시 검색하세요.", "نص البحث طويل جدًا. اختصره وابحث مجددًا."),
  searchChip: (l: Locale) => t(l, "Search", "검색", "بحث"),
  all: (l: Locale) => t(l, "All", "전체", "الكل"),
  membership: (l: Locale) => t(l, "Membership in this workspace", "이 업무 공간 소속", "العضوية في مساحة العمل هذه"),
  membershipEnded: (l: Locale) => t(l, "Role period ended, suspension pending", "역할 기간 종료(정지 처리 대기)", "انتهت مدة الدور (بانتظار التعليق)"),
  msSuspended: (l: Locale) => t(l, "Suspended", "정지됨", "معلّقة"),
  msActive: (l: Locale) => t(l, "Active", "활성", "نشطة"),
  setLinkAt: (l: Locale) => t(l, "Password set link issued, expires {at}", "비밀번호 설정 링크 발급됨 · {at} 만료", "صدر رابط تعيين كلمة المرور، ينتهي في {at}"),
  resetLinkAt: (l: Locale) => t(l, "Password reset link issued, expires {at}", "비밀번호 재설정 링크 발급됨 · {at} 만료", "صدر رابط إعادة تعيين كلمة المرور، ينتهي في {at}"),
  periodOpen: (l: Locale) => t(l, "No end date", "종료일 없음", "بلا تاريخ انتهاء"),
  periodBounded: (l: Locale, n: number) => counted(l, n, { en: ["Ends in more than {n} day", "Ends in more than {n} days"], ko: "{n}일 이후 종료",
    ar: ["تنتهي بعد أكثر من {n} يوم", "تنتهي بعد أكثر من يوم واحد", "تنتهي بعد أكثر من يومين", "تنتهي بعد أكثر من {n} أيام", "تنتهي بعد أكثر من {n} يومًا", "تنتهي بعد أكثر من {n} يوم"] }),
  periodSoon: (l: Locale, n: number) => counted(l, n, { en: ["Ends within {n} day", "Ends within {n} days"], ko: "{n}일 안에 종료",
    ar: ["تنتهي خلال {n} يوم", "تنتهي خلال يوم واحد", "تنتهي خلال يومين", "تنتهي خلال {n} أيام", "تنتهي خلال {n} يومًا", "تنتهي خلال {n} يوم"] }),
  passwordSet: (l: Locale) => t(l, "Password set", "비밀번호 설정됨", "تم تعيين كلمة المرور"),
  setLink: (l: Locale) => t(l, "Password set link issued", "비밀번호 설정 링크 발급됨", "صدر رابط تعيين كلمة المرور"),
  resetLink: (l: Locale) => t(l, "Password reset link issued", "비밀번호 재설정 링크 발급됨", "صدر رابط إعادة تعيين كلمة المرور"),
  notSet: (l: Locale) => t(l, "No password and no link", "비밀번호·링크 없음", "لا كلمة مرور ولا رابط"),
  holder: (l: Locale) => t(l, "Installation administrators only", "설치 관리자만", "مسؤولو التثبيت فقط"),
  methodLabel: (l: Locale) => t(l, "Sign-in method", "로그인 방식", "طريقة تسجيل الدخول"),
  method: methodLine,
  appliedFilters: (l: Locale) => t(l, "Applied filters:", "적용한 필터:", "عوامل التصفية المطبّقة:"),
  clearAll: (l: Locale) => t(l, "Clear all filters", "필터 모두 제거", "مسح كل عوامل التصفية"),
  removeFilter: (l: Locale, chip: string) => t(l, `Remove filter ${chip}`, `필터 제거: ${chip}`, `إزالة عامل التصفية ${chip}`),
  chip: (field: string, value: string) => `${field}: ${value}`,
  withMembership: (l: Locale) => t(l, "(people with a membership)", "(소속이 있는 사람)", "(الأشخاص الذين لديهم عضوية)"),
  order: (l: Locale) => t(l, "By login. Departed people last.", "로그인 ID순. 퇴사한 사람은 맨 뒤.", "حسب اسم الدخول. يأتي المغادرون أخيرًا."),
  noneShown: (l: Locale) => t(l, "No people shown.", "표시된 사람이 없습니다.", "لا يوجد أشخاص معروضون."),
  shownMore: (l: Locale, n: number) => t(l, `${people(l, n)} shown; more may exist`, `${people(l, n)} 표시, 더 있을 수 있음`, `المعروض: ${people(l, n)}؛ قد يوجد المزيد`),
  shownAll: (l: Locale, n: number) => t(l, `${people(l, n)} shown; end of list`, `${people(l, n)} 표시, 목록 끝`, `المعروض: ${people(l, n)}؛ نهاية القائمة`),
  searchedUpTo: (l: Locale) => t(l, "Searched up to {login}.", "{login}까지 검색했습니다.", "تم البحث حتى {login}."),
  searchedUser: (l: Locale) => t(l, "Searched up to user {id}.", "사용자 {id}까지 검색했습니다.", "تم البحث حتى المستخدم {id}."),
  continueSearching: (l: Locale) => t(l, "Continue searching", "이어서 검색", "متابعة البحث"),
  loadingMore: (l: Locale) => t(l, "Loading…", "불러오는 중…", "جارٍ التحميل…"),
  noMatchesYet: (l: Locale) => t(l, "No matches yet.", "지금까지 일치하는 사람이 없습니다.", "لا توجد نتائج مطابقة حتى الآن."),
  filteredEmpty: (l: Locale) => t(l, "No people match these filters in your permitted scope.", "허용 범위에서 이 필터에 맞는 사람이 없습니다.", "لا يوجد أشخاص يطابقون عوامل التصفية هذه ضمن نطاقك المسموح."),
  nextFailed: (l: Locale) => t(l, "The next page could not be loaded. The rows shown are kept.", "다음 페이지를 불러오지 못했습니다. 표시된 행은 유지됩니다.", "تعذر تحميل الصفحة التالية. تم الاحتفاظ بالصفوف المعروضة."),
  changed: (l: Locale) => [
    t(l, "The list changed after the first page.", "첫 페이지 이후 목록이 바뀌었습니다.", "تغيّرت القائمة بعد الصفحة الأولى."),
    t(l, "Reload for a consistent list.", "일관된 목록을 보려면 다시 불러오세요.", "أعد التحميل للحصول على قائمة متسقة."),
    t(l, "Changes that come only from time passing, such as a role period ending, are not detected.", "역할 기간 종료처럼 시간이 지나서 생기는 변화는 감지하지 않습니다.",
      "التغييرات الناتجة عن مرور الوقت فقط، مثل انتهاء مدة الدور، لا تُكتشف."),
  ],
  restart: (l: Locale) => [
    t(l, "The list changed while you were paging.", "페이지를 넘기는 동안 목록이 바뀌었습니다.", "تغيّرت القائمة أثناء التنقل بين الصفحات."),
    t(l, "Start again from the first page.", "첫 페이지부터 다시 시작하세요.", "ابدأ من جديد من الصفحة الأولى."),
  ],
  startAgain: (l: Locale) => t(l, "Start again", "처음부터 다시", "البدء من جديد"),
  departedRow: (l: Locale) => t(l, "Departed (login and email removed)", "퇴사(로그인 ID·이메일 삭제됨)", "مغادر (أُزيل اسم الدخول والبريد الإلكتروني)"),
  linkIssued: (l: Locale) => t(l, "Issued, expires {at}", "발급됨 · {at} 만료", "صادر، ينتهي في {at}"),
  linkWithInvitation: (l: Locale) => t(l, "Issued, expires with the invitation", "발급됨 · 초대와 함께 만료", "صادر، وينتهي مع الدعوة"),
  jobsNote: (l: Locale) => {
    const [a, b, c] = ["users", "invitations", "membership"].map((k) => tabName(l, k));
    return t(l, `Filters apply to the ${a}, ${b} and ${c} tabs.`, `필터는 '${a}', '${b}', '${c}' 탭에 적용됩니다.`, `تُطبَّق عوامل التصفية على علامات التبويب «${a}» و«${b}» و«${c}».`);
  },
  badLink: (l: Locale) => t(l, "One or more filters in the link were not valid and were removed.", "링크의 필터 중 올바르지 않은 값을 제거했습니다.", "أُزيل من الرابط ما كان غير صالح من عوامل التصفية."),
  updated: (l: Locale, count: string, k: number) => t(l, `List updated. ${bare(count)}. Filters applied: ${k}.`, `목록을 새로 고쳤습니다. ${bare(count)}. 적용된 필터 ${k}개.`, `تم تحديث القائمة. ${bare(count)}. عوامل التصفية المطبّقة: ${k}.`),
  removed: (l: Locale, count: string) => t(l, `Filter removed. ${bare(count)}.`, `필터를 제거했습니다. ${bare(count)}.`, `أُزيل عامل التصفية. ${bare(count)}.`),
  openNotLoaded: (l: Locale) => t(l, "The person you had open is not in the loaded rows.", "열어 둔 사람이 불러온 행에 없습니다.", "الشخص الذي فتحته ليس ضمن الصفوف المحمَّلة."),
  openFiltered: (l: Locale) => t(l, "The person you had open is not in the filtered list.", "열어 둔 사람이 필터 결과 목록에 없습니다.", "الشخص الذي فتحته ليس في القائمة المصفّاة."),
};
