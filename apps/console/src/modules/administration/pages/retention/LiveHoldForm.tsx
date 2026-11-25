import { useCallback, useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import type { ArtifactRow, Owner } from "../../../../service/retention";
import { classes } from "./live-codes";
import { k } from "./live-copy";
import { holdMatch, holdValues, partialCounts, runNumber, utf8Bytes, type FieldReaction } from "./live-model";
import { Code, Rich, holdTextError, href, ltr } from "./live-format";
import { CommandForm, type Draft, type FieldErrors, type Fields, type Shared } from "./LiveCommand";

export type OwnerSearch = {
  term: string;
  status: "searching" | "ready" | "failed";
  rows: Owner[];
  matching: number;
  more: boolean;
  seen: Record<string, string>;
  kept: Record<string, "reading" | "failed" | "gone">;
  opened: number;
  search: (term: string) => void;
  confirm: (id: string) => void;
  reopen: () => void;
  forget: (id: string) => void;
};

function withinBytes(text: string, limit: number): string {
  let out = "";
  for (const ch of text) {
    if (utf8Bytes(out + ch) > limit) break;
    out += ch;
  }
  return out;
}

export function useOwnerSearch(api: AdminClient): OwnerSearch {
  const [found, setFound] = useState<{ term: string; status: "searching" | "ready" | "failed"; rows: Owner[]; matching: number; more: boolean }>({ term: "", status: "searching", rows: [], matching: 0, more: false });
  const [seen, setSeen] = useState<Record<string, string>>({});
  const [kept, setKept] = useState<Record<string, "reading" | "failed" | "gone">>({});
  const [opened, setOpened] = useState(0);
  const seq = useRef(0);
  const note = (rows: Owner[]) => setSeen((was) => ({ ...was, ...Object.fromEntries(rows.map((o) => [o.id, o.login])) }));
  const search = useCallback(async (term: string) => {
    const n = ++seq.current;
    setFound((was) => ({ ...was, term, status: "searching" }));
    try {
      const got = await api.retentionOwners({ q: term });
      if (n !== seq.current) return;
      note(got.rows);
      setFound({ term, status: "ready", rows: got.rows, matching: got.matching, more: got.more });
    } catch {
      if (n !== seq.current) return;
      setFound((was) => ({ ...was, term, status: "failed" }));
    }
  }, [api]);
  const confirm = useCallback(async (id: string) => {
    setKept((was) => ({ ...was, [id]: "reading" }));
    try {
      const got = await api.retentionOwners({ id });
      note(got.rows);
      setKept((was) => {
        const next = { ...was };
        if (got.rows.some((o) => o.id === id)) delete next[id];
        else next[id] = "gone";
        return next;
      });
    } catch {
      setKept((was) => ({ ...was, [id]: "failed" }));
    }
  }, [api]);
  useEffect(() => { void search(""); }, [search]);
  return { ...found, seen, kept, opened, search: (term) => { void search(term); }, confirm: (id) => { void confirm(id); }, reopen: () => { setOpened((x) => x + 1); void search(""); },
    forget: (id) => setKept((was) => { if (!(id in was)) return was; const next = { ...was }; delete next[id]; return next; }) };
}

function Person({ o }: { o: Owner }) {
  return (
    <>
      {o.login && <bdi dir="auto" className="p04-owner-login">{o.login}</bdi>}
      <bdi dir="ltr" className="p04-owner-id">{o.id}</bdi>
    </>
  );
}

function OwnerField({ l, id, value, error, set, owners }: { l: Locale; id: string; value: string; error: string; set: (v: string) => void; owners: OwnerSearch }) {
  const [changing, setChanging] = useState(false);
  const [draft, setDraft] = useState("");
  const [focus, setFocus] = useState<{ at: string; n: number } | null>(null);
  const [gone, setGone] = useState(false);
  const field = id + "-owner";
  const state = value ? owners.kept[value] : undefined;
  const { confirm, seen, opened } = owners;
  useEffect(() => { if (focus) document.getElementById(focus.at)?.focus(); }, [focus]);
  useEffect(() => { if (value && !(value in seen) && !owners.kept[value]) confirm(value); }, [value]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { if (opened > 0) { setChanging(true); setFocus((f) => ({ at: field, n: (f?.n ?? 0) + 1 })); } }, [opened]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { if (state === "gone") { set(""); setChanging(false); setGone(true); } }, [state]); // eslint-disable-line react-hooks/exhaustive-deps
  const shownError = error || (gone && !value ? k.form.ownerGone(l) : "");
  const open = !value || changing;
  const none = owners.status === "ready" && owners.term === "" && owners.matching === 0;
  const run = () => owners.search(withinBytes(draft.trim(), 64));
  const go = (at: string) => setFocus((f) => ({ at, n: (f?.n ?? 0) + 1 }));
  const described = [open && !none ? field + "-search-hint" : "", field + "-hint", shownError ? field + "-error" : ""].filter(Boolean).join(" ");
  const status = !open ? "" : owners.status === "searching" ? k.form.ownerSearching(l) : owners.status === "failed" || none ? ""
    : owners.term === "" ? (owners.more ? k.form.ownerFirst(l, owners.rows.length, owners.matching) : k.form.ownerAll(l, owners.matching))
      : owners.matching === 0 ? k.form.ownerNone(l, owners.term) : owners.more ? k.form.ownerMore(l, owners.rows.length, owners.matching) : k.form.ownerMatches(l, owners.matching);
  return (
    <div className="service-field p04-owner-field">
      {open && !none ? <label htmlFor={field} id={field + "-label"}>{k.form.owner(l)}</label> : <span id={field + "-label"} className="p04-owner-label">{k.form.owner(l)}</span>}
      {shownError && <p className="service-field-error" id={field + "-error"}>{shownError}</p>}
      {value && (
        <p className="p04-owner-chosen">
          <span className="p04-owner-name" id={field + "-name"}><Person o={{ id: value, login: seen[value] ?? "" }} /></span>
          {open
            ? <button type="button" id={field + "-keep"} className="service-button-link p04-owner-toggle" onClick={() => { setChanging(false); go(field); }}>{k.form.ownerKeep(l)}</button>
            : <button type="button" id={field} className="service-button-link p04-owner-toggle" aria-labelledby={`${field} ${field}-label ${field}-name`} aria-describedby={described} aria-invalid={!!shownError || undefined} onClick={() => { setChanging(true); owners.search(""); setDraft(""); go(field); }}>{k.form.ownerChange(l)}</button>}
        </p>
      )}
      {state === "failed" && (
        <div className="p04-owner-unread" role="alert">
          <p>{k.form.ownerLoginFailed(l)}</p>
          <button type="button" className="live-secondary" onClick={() => confirm(value)}>{k.form.tryAgain(l)}</button>
        </div>
      )}
      {open && none && <p className="live-empty" role="status">{k.form.ownersEmpty(l)}</p>}
      {open && !none && <p className="service-hint p04-owner-search-hint" id={field + "-search-hint"}>{k.form.ownerSearchHint(l)}</p>}
      {open && !none && (
        <div className="p04-owner-search">
          <input id={field} type="search" dir="auto" autoComplete="off" value={draft} maxLength={64} aria-invalid={!!shownError || undefined} aria-describedby={described} onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); run(); } }} />
          <button type="button" className="live-secondary p04-owner-search-button" onClick={run}>{k.form.ownerSearch(l)}</button>
        </div>
      )}
      {open && <p className="p04-small live-quiet p04-owner-status" role="status">{status}</p>}
      {open && owners.status === "failed" && (
        <div className="live-error p04-owners-failed" role="alert">
          <p>{k.form.ownersFailed(l)}{value ? " " + k.form.ownerKept(l) : ""}</p>
          <button type="button" className="live-secondary" onClick={() => { owners.search(owners.term); go(field); }}>{k.form.tryAgain(l)}</button>
        </div>
      )}
      {open && owners.status !== "failed" && owners.rows.length > 0 && (
        <ul className="p04-owner-results" aria-label={k.form.ownerResults(l)} aria-busy={owners.status === "searching" || undefined}>
          {owners.rows.map((o) => (
            <li key={o.id}>
              <button type="button" className="p04-owner-result" aria-current={(o.id === value && !error) || undefined} onClick={() => { owners.forget(o.id); set(o.id); setChanging(false); setGone(false); go(field); }}><Person o={o} /></button>
            </li>
          ))}
        </ul>
      )}
      <p className="service-hint" id={field + "-hint"}>{k.form.ownerHint(l)}</p>
    </div>
  );
}

export function TextField({ l, id, label, value, error, set }: { l: Locale; id: string; label: string; value: string; error: string; set: (v: string) => void }) {
  return (
    <div className="service-field p04-text-field">
      <label htmlFor={id}>{label}</label>
      {error && <p className="service-field-error" id={id + "-error"}>{error}</p>}
      <input
        id={id} type="text" autoComplete="off" value={value} aria-invalid={!!error || undefined}
        aria-describedby={[id + "-hint", error ? id + "-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set(e.target.value)}
      />
      <p className="service-hint" id={id + "-hint"}>{k.form.textHint(l, utf8Bytes(value.trim()))}</p>
    </div>
  );
}

function holdFields(l: Locale, tenant: boolean, owners: OwnerSearch): Fields {
  const ready = !(owners.status === "ready" && owners.term === "" && owners.matching === 0);
  return {
    keys: tenant ? ["owner", "reason", "class"] : ["owner", "reason"],
    saved: tenant ? ["reason", "class"] : ["reason"],
    ready,
    check: (d) => {
      const e: FieldErrors = {};
      if (!d.owner || owners.kept[d.owner] === "gone") e.owner = k.form.ownerMissing(l);
      const text = holdTextError(l, d.reason ?? "");
      if (text) e.reason = text;
      return e;
    },
    values: (d) => holdValues(d, tenant),
    render: (d, set, errors, id) => (
      <>
        <OwnerField l={l} id={id} value={d.owner ?? ""} error={errors.owner ?? ""} set={(v) => set("owner", v)} owners={owners} />
        <TextField l={l} id={id + "-reason"} label={k.form.holdReason(l)} value={d.reason ?? ""} error={errors.reason ?? ""} set={(v) => set("reason", v)} />
        {tenant && (
          <div className="service-field p04-class-field">
            <label htmlFor={id + "-class"}>{k.form.classLabel(l)}</label>
            <select id={id + "-class"} value={classes.includes(d.class ?? "") ? d.class : ""} onChange={(e) => set("class", e.target.value)}>
              <option value="">{k.form.allClasses(l)}</option>
              {classes.map((c) => <option key={c} value={c}>{k.className(l, c)}</option>)}
            </select>
          </div>
        )}
      </>
    ),
  };
}

export function itemContext(a: ArtifactRow, cls: string): Record<string, string> {
  return {
    view: "expiry",
    target_kind: "item",
    item: a.id,
    class: cls,
    access: a.access,
    standing: a.standing,
    damaged: a.damaged === true ? "1" : "",
    run: a.run && runNumber(a.run) && a.other_scope !== true ? a.run.label : "",
    run_other: a.other_scope === true ? "1" : "",
  };
}

function HoldTargetFacts({ l, ctx }: { l: Locale; ctx: Record<string, string> }) {
  if (ctx.target_kind === "workspace") {
    return <>{k.thisWorkspace(l)} · {ctx.choice ? k.className(l, ctx.choice) : k.allClasses(l)}</>;
  }
  return (
    <>
      <Rich text={k.itemTarget(l, ctx.item ?? "")} /> · <Code l={l} value={ctx.class ?? ""} say={k.className} /> · <Code l={l} value={ctx.access ?? ""} say={k.access} /> ·{" "}
      <Code l={l} value={ctx.standing ?? ""} say={k.standing} />
    </>
  );
}

function HoldImpact({ l, ctx }: { l: Locale; ctx: Record<string, string> }) {
  return (
    <>
      <p><strong>{k.holdStrong(l)}</strong> {k.holdBody(l)}</p>
      {ctx.target_kind === "item" && (ctx.access === "deleted" || ctx.standing === "deleted") && <p>{k.itemDeleted(l)}</p>}
      {ctx.target_kind === "item" && ctx.damaged === "1" && <p>{k.itemDamaged(l)}</p>}
      {ctx.target_kind === "item" && ctx.run && <p><Rich text={k.itemInRun(l, ctx.run)} /></p>}
      {ctx.target_kind === "item" && ctx.run_other === "1" && <p>{k.itemInOtherRun(l)}</p>}
      {ctx.target_kind === "workspace" && <p>{k.workspaceCovers(l, ctx.choice ? k.className(l, ctx.choice) : "")}</p>}
      <p>{k.releaseLater(l)}</p>
    </>
  );
}

export function HoldCommand({ target, tenant, base, heading, onClose, ...shared }: Shared & {
  target: string;
  tenant: boolean;
  base: Record<string, string>;
  heading: string;
  onClose: (how?: "done" | "refused") => void;
}) {
  const { l, page, api, me } = shared;
  const owners = useOwnerSearch(api);
  const ownerLogin = (id: string) => owners.seen[id] ?? "";
  return (
    <CommandForm
      {...shared} command="retention.place_hold" target={target} fields={holdFields(l, tenant, owners)}
      heading={heading} title={k.confirmHold(l)} verb={k.holdVerb(l)}
      context={(d) => ({ ...base, owner: d.owner ?? "", owner_login: ownerLogin(d.owner ?? ""), reason: (d.reason ?? "").trim(), choice: tenant ? d.class ?? "" : "" })}
      facts={(ctx) => (
        <>
          <div className="p04-span"><dt>{k.factTarget(l)}</dt><dd><HoldTargetFacts l={l} ctx={ctx} /></dd></div>
          <div><dt>{k.factOwner(l)}</dt><dd>{ctx.owner ? ltr(ctx.owner_login ? `${ctx.owner_login} (${ctx.owner})` : ctx.owner) : k.unknown(l)}</dd></div>
          <div className="p04-span"><dt>{k.factHoldReason(l)}</dt><dd><span dir="auto" className="p04-free">{ctx.reason}</span></dd></div>
        </>
      )}
      impact={(ctx) => <HoldImpact l={l} ctx={ctx} />}
      editNote={(pending) => (pending ? <div className="p04-edit-note"><p>{k.editWarning(l)}</p><p className="p04-small">{k.holdPendingNote(l)}</p></div> : null)}
      stale={(frozen) => {
        const id = holdMatch(page, me, target, frozen);
        return id ? (
          <>
            <p><Rich text={k.staleMatch(l, id)} /></p>
            <p><a className="p04-link" href={href(l, "holds", { hold: id })}><Rich text={k.goToHold(l, id)} /></a></p>
          </>
        ) : <p>{k.staleNoMatch(l)}</p>;
      }}
      hooks={{ field: (how: FieldReaction) => { if (how === "owner") owners.reopen(); } }}
      done={(fields, st) => {
        const hold = fields.hold ?? "";
        if (st === "held" && hold) return k.holdInForce(l, hold);
        if (st === "partially_preserved" && hold) {
          const c = partialCounts(fields);
          return k.holdPartial(l, hold, c.a, c.b, c.c);
        }
        return k.resultUnknown(l, st);
      }}
      onClose={onClose}
    />
  );
}
