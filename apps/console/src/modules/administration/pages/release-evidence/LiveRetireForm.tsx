import type { Locale } from "../../../../shared/types";
import { useEffect, useRef, useState } from "react";
import type { AdminClient } from "../../../../service/admin";
import type { EvidencePage, RequirementRow, TargetRow } from "../../../../service/release-evidence";
import { c as sc } from "../../../../service/copy";
import { f, k } from "./live-copy";
import { noReplacement, replacementOptions, replacementProblem, requirementByTarget, retireValues, textProblem, utf8Bytes } from "./live-model";
import type { FieldErrors, Fields } from "./LiveCommand";

export function retireTextError(l: Locale, v: string): string {
  const p = textProblem(v);
  if (p === "missing") return f.retireReasonMissing(l);
  if (p === "long") return f.retireReasonLong(l);
  if (p === "control") return f.retireReasonControl(l);
  return "";
}

export function replacementError(l: Locale, problem: string): string {
  return problem === "required" ? f.replacementRequired(l) : problem === "choose" ? f.replacementChoose(l) : "";
}

export type ReplacementSearch = {
  term: string;
  status: "idle" | "searching" | "failed";
  rows: TargetRow[] | null;
  matching: number;
  more: boolean;
  seen: Record<string, string>;
  detail: Record<string, RequirementRow | "loading" | "failed">;
  choose: (target: string) => void;
  focus: number;
  search: (term: string) => void;
};

function withinBytes(text: string, limit: number): string {
  let out = "";
  for (const ch of text) {
    if (utf8Bytes(out + ch) > limit) break;
    out += ch;
  }
  return out;
}

export function useReplacementSearch(api: AdminClient, requirement: string, page: EvidencePage): ReplacementSearch {
  const [found, setFound] = useState<{ key: string; term: string; status: "idle" | "searching" | "failed"; rows: TargetRow[] | null; matching: number; more: boolean }>({ key: requirement, term: "", status: "idle", rows: null, matching: 0, more: false });
  const [seen, setSeen] = useState<Record<string, string>>({});
  const [detail, setDetail] = useState<Record<string, RequirementRow | "loading" | "failed">>({});
  const [focus, setFocus] = useState(0);
  const choose = (target: string) => {
    const held = detail[target];
    if (!target || target === noReplacement || requirementByTarget(page, target) || (held && held !== "failed")) return;
    setDetail((was) => ({ ...was, [target]: "loading" }));
    api.evidenceRequirements({ targets: [target] })
      .then((got) => setDetail((was) => ({ ...was, [target]: got.rows.find((x) => x.target === target) ?? "failed" })))
      .catch(() => setDetail((was) => ({ ...was, [target]: "failed" })));
  };
  const seq = useRef(0);
  const now = found.key === requirement ? found : { key: requirement, term: "", status: "idle" as const, rows: null, matching: 0, more: false };
  const search = (raw: string) => {
    const term = withinBytes(raw.trim(), 64);
    const n = ++seq.current;
    if (term === "") {
      setFound({ key: requirement, term: "", status: "idle", rows: null, matching: 0, more: false });
      return;
    }
    setFound({ ...now, status: "searching" });
    api.evidenceReplacements(requirement, { q: term })
      .then((got) => {
        if (n !== seq.current) return;
        setSeen((was) => ({ ...was, ...Object.fromEntries(got.rows.map((o) => [o.target, o.label])) }));
        setFound({ key: requirement, term, status: "idle", rows: got.rows, matching: got.matching, more: got.more });
        setFocus((x) => x + 1);
      })
      .catch(() => {
        if (n !== seq.current) return;
        setFound((was) => ({ ...(was.key === requirement ? was : now), status: "failed", term: was.key === requirement && was.rows !== null ? was.term : term }));
      });
  };
  return { ...now, seen, detail, choose, focus, search };
}

function ReplacementFinder({ l, id, r, find }: { l: Locale; id: string; r: RequirementRow; find: ReplacementSearch }) {
  const [draft, setDraft] = useState("");
  const [tried, setTried] = useState("");
  const input = id + "-replacement-search";
  useEffect(() => { if (find.focus > 0) document.getElementById(input)?.focus(); }, [find.focus]); // eslint-disable-line react-hooks/exhaustive-deps
  const run = () => { setTried(draft); find.search(draft); };
  const listed = replacementOptions(r).length;
  const status = find.status === "searching" ? f.replacementSearching(l)
    : find.rows === null ? (r.replacements.total !== null ? f.replacementFirst(l, listed, r.replacements.total) : f.replacementFirstUncounted(l))
      : find.matching === 0 ? f.replacementNone(l, find.term) : find.more ? f.replacementMoreMatch(l, find.rows.length, find.matching) : f.replacementMatches(l, find.matching);
  return (
    <>
      <p className="p05-small p05-muted p05-replacement-status" role="status" id={id + "-replacement-status"}>{status}</p>
      <label className="p05-replacement-search-label" htmlFor={input}>{f.replacementSearch(l)}</label>
      <div className="p05-replacement-search">
        <input id={input} type="search" dir="auto" autoComplete="off" maxLength={64} value={draft} aria-describedby={id + "-replacement-status"} onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); run(); } }} />
        <button type="button" className="live-secondary p05-replacement-search-button" onClick={run}>{f.replacementSearchButton(l)}</button>
      </div>
      {find.status === "failed" && (
        <div className="p05-replacement-failed" role="alert">
          <p>{f.replacementSearchFailed(l)}</p>
          <button type="button" className="live-secondary" onClick={() => find.search(tried)}>{k.reqTryAgain(l)}</button>
        </div>
      )}
    </>
  );
}

export function retireFields(l: Locale, r: RequirementRow, frozen = "", find?: ReplacementSearch): Fields {
  const required = r.replacement_required === true;
  const first = replacementOptions(r);
  const searchable = !!find && r.replacements.more === true;
  const listed = searchable && find.rows !== null ? find.rows.filter((o) => o.target !== r.target && !!o.label) : first;
  const known = (target: string) => (find?.seen[target] ?? first.find((o) => o.target === target)?.label ?? target);
  const withChoice = (choice: string) => (choice && choice !== noReplacement && !listed.some((o) => o.target === choice) && (choice === frozen || !!find?.seen[choice] || first.some((o) => o.target === choice))
    ? [...listed, { target: choice, label: known(choice) }] : listed);
  const options = withChoice(frozen);
  return {
    keys: ["reason", "replacement"],
    saved: ["reason"],
    check: (d) => {
      const e: FieldErrors = {};
      const text = retireTextError(l, d.reason ?? "");
      if (text) e.reason = text;
      const choice = replacementError(l, replacementProblem(d.replacement ?? "", required, withChoice(d.replacement ?? "")));
      if (choice) e.replacement = choice;
      return e;
    },
    values: (d) => retireValues(d),
    render: (d, set, errors, id) => (
      <>
        <div className="service-field p05-text-field">
          <label htmlFor={id + "-reason"}>{f.retireReason(l)}</label>
          <p className="service-hint" id={id + "-reason-hint"}>{f.retireReasonHint(l)}</p>
          {errors.reason && <p className="service-field-error" id={id + "-reason-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{errors.reason}</p>}
          <input
            id={id + "-reason"} type="text" autoComplete="off" dir="auto" value={d.reason ?? ""} aria-invalid={!!errors.reason || undefined}
            aria-describedby={[id + "-reason-hint", errors.reason ? id + "-reason-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set("reason", e.target.value)}
          />
        </div>
        <div className="service-field p05-select-field">
          <label htmlFor={id + "-replacement"}>{f.replacement(l)}</label>
          <p className="service-hint" id={id + "-replacement-hint"}>{f.replacementHint(l)}</p>
          {r.replacements.more === true && !searchable && <p className="service-hint">{f.replacementMore(l)}</p>}
          {errors.replacement && <p className="service-field-error" id={id + "-replacement-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{errors.replacement}</p>}
          <select
            id={id + "-replacement"} value={d.replacement ?? ""} aria-invalid={!!errors.replacement || undefined}
            aria-describedby={[id + "-replacement-hint", errors.replacement ? id + "-replacement-error" : "", searchable ? id + "-replacement-status" : ""].filter(Boolean).join(" ")} onChange={(e) => { set("replacement", e.target.value); find?.choose(e.target.value); }}
          >
            <option value="">{f.choose(l)}</option>
            {!required && <option value={noReplacement}>{f.noReplacementOption(l)}</option>}
            {(searchable ? withChoice(d.replacement ?? "") : options).map((o) => <option key={o.target} value={o.target}>{"\u2066" + o.label + "\u2069"}</option>)}
          </select>
          {searchable && find && <ReplacementFinder l={l} id={id} r={r} find={find} />}
        </div>
      </>
    ),
  };
}
