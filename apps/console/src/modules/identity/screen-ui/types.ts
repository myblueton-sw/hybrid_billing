import type { Locale } from "../../../shared/types";
export type Props = {
  screenId: string;
  locale: Locale;
  params: URLSearchParams;
  view: string;
  onState: (state: string, items: string[]) => void;
  onDirty: (dirty: boolean) => void;
};
export type Words = readonly [string, string, string];
export const say = (l: Locale, w: Words) =>
  w[l === "en" ? 0 : l === "ko" ? 1 : 2];
