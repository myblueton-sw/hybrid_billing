import type { ReactNode } from "react";
import type { Locale } from "../../../../../shared/types";
import { day, ltr } from "../live-format";
import { text } from "./copy";
import { partText } from "./part-copy";
import { setupParts, type SetupPart, type SetupScope } from "./scope";
import "./parts.css";

export function partHref(part: SetupPart): string {
 if (typeof location === "undefined") return "#";
 const [path, query = ""] = location.hash.replace(/^#/, "").split("?"), q = new URLSearchParams(query);
 if (part === "settings") q.delete("part"); else q.set("part", part);
 return "#" + path + "?" + q.toString();
}

type NavProps = { l: Locale; part: SetupPart; onPart: (part: SetupPart) => void };
export function PartNav({ l, part, onPart }: NavProps) {
 return <nav className="o03-parts" aria-label={partText(l, "nav")}><ul>{setupParts.map(item => {
  const label = partText(l, item);
  return <li key={item}><a href={partHref(item)} aria-current={item === part ? "page" : undefined} data-label={label} onClick={event => { event.preventDefault(); onPart(item); }}><span>{label}</span></a></li>;
 })}</ul></nav>;
}

export function ScopeBar({ l, scope }: { l: Locale; scope: SetupScope }) {
 return <section className="o03-scope-bar" aria-labelledby="o03-scope-bar-title">
  <h3 id="o03-scope-bar-title">{partText(l, "scope")}</h3>
  <dl className="live-facts live-facts-grid">
   <Fact label={partText(l, "setupDate")}>{day(l, scope.setupOn)}</Fact>
   <Fact label={text(l, "recordedContractRevision")}>{ltr(scope.contractRevision)}</Fact>
   <Fact label={text(l, "recordedProfileVersion")}>{ltr(scope.billingRevision)}</Fact>
   <Fact label={text(l, "state")}><span className="o03-tag">{text(l, "confirmed")}</span></Fact>
  </dl>
  <p className="live-sub">{partText(l, "scopeNote")}</p>
  <details><summary>{text(l, "details")}</summary><p className="live-mono">{ltr(scope.configuration)}</p><p className="live-mono">{ltr(scope.digest)}</p></details>
 </section>;
}

export function PartFrame({ l, part, scope, loading, feedback, onPart, children }: NavProps & { scope: SetupScope | undefined; loading: boolean; feedback: ReactNode; children?: (scope: SetupScope) => ReactNode }) {
 if (!scope) return <section className="o03-part" aria-labelledby="o03-part-title">
  <h3 id="o03-part-title">{partText(l, part)}</h3>
  {loading ? <p role="status">{text(l, "loading")}</p> : feedback ? feedback : <><p>{partText(l, "noScope")}</p><p><a href={partHref("settings")} onClick={event => { event.preventDefault(); onPart("settings"); }}>{partText(l, "toSettings")}</a></p></>}
 </section>;
 const content = children?.(scope);
 if (content) return <><ScopeBar l={l} scope={scope} />{content}</>;
 return <><ScopeBar l={l} scope={scope} /><section className="o03-part" aria-labelledby="o03-part-title"><h3 id="o03-part-title">{partText(l, part)}</h3><p>{partText(l, "unconnected")}</p></section></>;
}

function Fact({ label, children }: { label: string; children: ReactNode }) { return <div><dt>{label}</dt><dd>{children}</dd></div>; }
