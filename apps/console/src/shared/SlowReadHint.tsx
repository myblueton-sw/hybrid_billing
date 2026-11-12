import { useEffect, useState } from "react";
import type { Locale } from "./types";
import { text } from "./copy";
export const SLOW_READ_DELAY_MS = 4000;
export const slowReadText = (locale: Locale) =>
  text(
    locale,
    "Still working on this query. Other queries may be running first, so this one may be waiting its turn.",
    "아직 조회 중입니다. 먼저 진행 중인 조회가 있으면 차례를 기다린 뒤 조회합니다.",
    "ما زال الاستعلام جارياً. قد تكون استعلامات أخرى قيد التنفيذ قبله، فينتظر دوره.",
  );
export function scheduleSlowRead(show: () => void) {
  const id = setTimeout(show, SLOW_READ_DELAY_MS);
  return () => clearTimeout(id);
}
export default function SlowReadHint({ locale }: { locale: Locale }) {
  const [shown, setShown] = useState(false);
  useEffect(() => scheduleSlowRead(() => setShown(true)), []);
  return shown ? (
    <span className="slow-read-hint" data-slow-read>
      {slowReadText(locale)}
    </span>
  ) : null;
}
