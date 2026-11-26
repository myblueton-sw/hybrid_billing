import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import type { AccessSubject, SubjectPage } from "../../../../service/access-lists";
import { c } from "../../../../service/copy";
import { a } from "./live-copy";
import { pc } from "./person-copy";

const pageSize = 25;
type Read = { state: "searching" } | { state: "failed"; term: string } | { state: "ready"; term: string; page: SubjectPage };

export function PersonPicker({ l, api, id, self, chosen, error, disabled, flagLine, onChoose }: {
  l: Locale; api: AdminClient; id: string; self: string; chosen: AccessSubject | null; error: string; disabled?: boolean; flagLine: string;
  onChoose: (person: AccessSubject) => void;
}) {
  const [draft, setDraft] = useState("");
  const [term, setTerm] = useState("");
  const [tries, setTries] = useState(0);
  const [changing, setChanging] = useState(false);
  const [answer, setRead] = useState<Read>({ state: "searching" });
  const input = useRef<HTMLInputElement>(null);
  const toggle = useRef<HTMLButtonElement>(null);
  const focus = useRef<"" | "toggle" | "input">("");
  const open = !chosen || changing;
  useEffect(() => {
    if (!open) return;
    let active = true;
    setRead({ state: "searching" });
    api.accessSubjects({ limit: pageSize, ...(term ? { q: term } : {}) }).then(
      (page) => { if (active) setRead({ state: "ready", term, page }); },
      () => { if (active) setRead({ state: "failed", term }); },
    );
    return () => { active = false; };
  }, [api, term, tries, open]);
  useEffect(() => {
    if (focus.current === "toggle") toggle.current?.focus();
    if (focus.current === "input") input.current?.focus();
    focus.current = "";
  });
  const run = () => { const next = draft.trim(); if (next === term) setTries((n) => n + 1); else setTerm(next); };
  const read: Read = answer.state !== "searching" && answer.term !== term ? { state: "searching" } : answer;
  const page = read.state === "ready" ? read.page : null;
  const rows = page ? page.rows.filter((row) => row.user_id !== self) : [];
  const nobody = !!page && term === "" && rows.length === 0 && !page.next;
  const status = read.state === "searching" ? pc.searching(l) : !page ? ""
    : rows.length === 0 ? (term === "" ? "" : page.rows.length > 0 ? pc.onlyYou(l, term) : pc.noMatch(l, term))
      : page.next ? (term === "" ? pc.firstPage(l, pageSize) : pc.refine(l)) : pc.count(l, rows.length);
  const described = [id + "-hint", id + "-status", error ? id + "-error" : ""].filter(Boolean).join(" ");
  const name = (person: AccessSubject) => (
    <>
      {person.login && <span className="person-login"><bdi dir="ltr">{person.login}</bdi></span>}
      <span className="person-id" translate="no"><bdi dir="ltr">{person.user_id}</bdi></span>
    </>
  );
  if (nobody && !chosen) return <p className="live-note" data-person-none="true" id={id} tabIndex={-1}>{a.noneEligible(l)}</p>;
  return (
    <div className="service-field person-picker" data-person={chosen?.user_id ?? ""}>
      <label htmlFor={id} id={id + "-label"}>{a.person(l)}</label>
      <p className="service-hint" id={id + "-hint"}>{pc.hint(l)}</p>
      {error && <p className="service-field-error" id={id + "-error"}><span className="visually-hidden">{c.errorPrefix(l)} </span>{error}</p>}
      {chosen && (
        <div className="person-chosen" role="group" aria-labelledby={id + "-label"}>
          <span className="person-name person-name-chosen">{name(chosen)}</span>
          <button type="button" className="person-toggle" ref={toggle} disabled={disabled}
            onClick={() => { focus.current = changing ? "toggle" : "input"; setChanging(!changing); }}>{changing ? pc.keep(l) : pc.change(l)}</button>
        </div>
      )}
      {chosen?.revocation_unconfirmed && flagLine && <p className="live-strong" data-unconfirmed="true">{flagLine}</p>}
      {open && (
        <div className="person-search">
          <input id={id} ref={input} type="search" dir="auto" autoComplete="off" spellCheck={false} value={draft} maxLength={64} disabled={disabled}
            aria-invalid={error ? true : undefined} aria-describedby={described} onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); run(); } }} />
          <button type="button" className="live-secondary person-search-button" disabled={disabled} onClick={run}>{pc.search(l)}</button>
        </div>
      )}
      {open && <p className="live-meta person-status" role="status" id={id + "-status"}>{status}</p>}
      {open && read.state === "failed" && (
        <div className="live-error person-failed" role="alert">
          <p>{pc.failed(l)}{chosen ? " " + pc.kept(l) : ""}</p>
          <button type="button" className="live-secondary" onClick={() => { focus.current = "input"; setTries((n) => n + 1); }}>{c.tryAgain(l)}</button>
        </div>
      )}
      {open && rows.length > 0 && (
        <ul className="person-results" aria-label={pc.results(l)}>
          {rows.map((row) => (
            <li key={row.user_id}>
              <button type="button" className="person-result" data-user={row.user_id} disabled={disabled} aria-current={row.user_id === chosen?.user_id || undefined}
                onClick={() => { focus.current = "toggle"; setChanging(false); onChoose(row); }}>
                <span className="person-name">{name(row)}</span>
                {row.revocation_unconfirmed && <span className="person-flag">{a.stUnconfirmed(l)}</span>}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
