import type { Locale } from "../../types";
export type Words = readonly [string, string, string];
export const t = (l: Locale, words: Words) =>
  words[l === "ko" ? 0 : l === "en" ? 1 : 2];
export type Props = {
  screenId: string;
  locale: Locale;
  params: URLSearchParams;
  view: string;
  onState: (state: string, items: string[]) => void;
  onDirty: (dirty: boolean) => void;
};
export type Field = {
  id: string;
  label: Words;
  kind?: "text" | "date" | "datetime-local" | "textarea";
  required?: boolean;
  options?: readonly Words[];
  hint?: Words;
  pattern?: RegExp;
};
export const unknown: Words = ["미확인", "Unknown", "غير معروف"];
export const unconnected: Words = [
  "서비스 미연결",
  "Service not connected",
  "الخدمة غير متصلة",
];
export const localOnly: Words = [
  "브라우저에서만 검토합니다. 서버에 저장하거나 업무를 실행하지 않습니다.",
  "Review in this browser only. Nothing is saved to a server or executed.",
  "مراجعة داخل المتصفح فقط. لا يتم الحفظ على الخادم أو تنفيذ أي إجراء.",
];
