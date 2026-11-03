import { useEffect, useState } from "react";
import type { Locale } from "../shared/types";
import type { PlanningScreen } from "../planning/registry";
import { local, titles } from "../planning/copy";
import { planningUrl } from "../planning/routing";
import {
  businessSections,
  administrationSection,
  navigationLabels,
  type ServiceSection,
} from "./service-ia";
export default function ServiceNavigation({
  locale,
  entries,
  active,
  query,
  onNavigate,
  developer,
}: {
  locale: Locale;
  entries: PlanningScreen[];
  active?: string;
  query: URLSearchParams;
  onNavigate: (path: string) => void;
  developer: boolean;
}) {
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  useEffect(() => {
    const group = [...businessSections, administrationSection].find((s) =>
      s.screens.includes(active ?? ""),
    );
    if (group)
      setExpanded((old) =>
        old[group.key] ? old : { ...old, [group.key]: true },
      );
  }, [active]);
  const render = (s: PlanningScreen) => (
    <a
      key={s.id}
      href={planningUrl(s.path, query)}
      aria-current={s.id === active ? "page" : undefined}
      onClick={(e) => {
        e.preventDefault();
        onNavigate(s.path);
      }}
    >
      <span>{local(locale, navigationLabels[s.id] ?? titles[s.id])}</span>
      {developer && <small>{s.id}</small>}
    </a>
  );
  const section = (group: ServiceSection) => {
    const pages = group.screens.flatMap((id) => {
      const found = entries.find((s) => s.id === id);
      return found ? [found] : [];
    });
    if (!pages.length) return null;
    return (
      <details
        key={group.key}
        data-section={group.key}
        open={
          expanded[group.key] ??
          (developer || pages.some((s) => s.id === active))
        }
      >
        <summary
          onClick={(e) => {
            e.preventDefault();
            const current = e.currentTarget.parentElement as HTMLDetailsElement;
            setExpanded((old) => ({ ...old, [group.key]: !current.open }));
          }}
        >
          {local(locale, group.label)}
        </summary>
        {pages.map(render)}
      </details>
    );
  };
  return (
    <nav
      className="service-navigation"
      aria-label={local(locale, [
        "서비스 메뉴",
        "Service navigation",
        "تنقل الخدمة",
      ])}
    >
      <div className="service-business">{businessSections.map(section)}</div>
      <div className="service-administration">
        {section(administrationSection)}
      </div>
    </nav>
  );
}
