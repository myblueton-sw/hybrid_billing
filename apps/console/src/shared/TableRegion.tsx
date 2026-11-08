import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import type { Locale } from "./types";
export default function TableRegion({
  locale,
  label,
  className,
  children,
}: {
  locale: Locale;
  label: string;
  className: string;
  children: ReactNode;
}) {
  const region = useRef<HTMLDivElement>(null),
    id = useId();
  const [overflow, setOverflow] = useState(false);
  useEffect(() => {
    const target = region.current;
    if (!target) return;
    const measure = () =>
      setOverflow(target.scrollWidth > target.clientWidth + 1);
    measure();
    if (typeof ResizeObserver === "undefined") {
      window.addEventListener("resize", measure);
      return () => window.removeEventListener("resize", measure);
    }
    const observer = new ResizeObserver(measure);
    observer.observe(target);
    const table = target.querySelector("table");
    if (table) observer.observe(table);
    return () => observer.disconnect();
  }, [locale, label]);
  const hint =
    locale === "ko"
      ? "표의 나머지 열을 보려면 가로로 이동하세요."
      : locale === "en"
        ? "Scroll horizontally to see the remaining columns."
        : "مرّر أفقيًا لعرض الأعمدة المتبقية.";
  return (
    <>
      <p id={id} className="remaining-table-hint" hidden={!overflow}>
        {hint}
      </p>
      <div
        ref={region}
        className={className}
        tabIndex={0}
        role="region"
        aria-label={label}
        aria-describedby={overflow ? id : undefined}
        data-horizontal-overflow={overflow}
      >
        {children}
      </div>
    </>
  );
}
