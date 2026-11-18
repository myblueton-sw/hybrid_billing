import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { dateParts } from "./period-model";

export function Stamp({ l, iso, date = true, time = true, zone = true }: { l: Locale; iso: string; date?: boolean; time?: boolean; zone?: boolean }) {
  if (!iso) return <>—</>;
  const d = dateParts(l, iso);
  const parts = [date && <bdi key="d" className="m01-part">{d.date}</bdi>, time && <bdi key="t" className="m01-part">{d.time}</bdi>].filter(Boolean);
  return (
    <span className="m01-stamp">
      {parts.length === 2 ? <>{parts[0]}{l === "ar" ? "، " : ", "}{parts[1]}</> : parts}
      {zone && <> <bdi dir="ltr" className="m01-part">({d.zone})</bdi></>}
    </span>
  );
}

export function fill(template: string, node: ReactNode, token = "{at}"): ReactNode {
  const [before, after = ""] = template.split(token);
  return <>{before}{node}{after}</>;
}

export function At({ l, template, iso, date, time, zone }: { l: Locale; template: string; iso: string; date?: boolean; time?: boolean; zone?: boolean }) {
  return fill(template, <Stamp l={l} iso={iso} date={date} time={time} zone={zone} />);
}
