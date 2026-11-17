import { useCallback, useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import { ownersOf, type ArtifactRow, type Owner } from "../../../../service/retention";
import { classes } from "./live-codes";
import { k } from "./live-copy";
import { holdMatch, holdValues, ownerOption, partialCounts, runNumber, utf8Bytes, type FieldReaction } from "./live-model";
import { Code, Rich, holdTextError, href, ltr } from "./live-format";
import { CommandForm, type Draft, type FieldErrors, type Fields, type Shared } from "./LiveCommand";

export type Owners = { state: "loading" | "failed" | "ready"; list: Owner[] };

export function useOwners(api: AdminClient) {
  const [owners, setOwners] = useState<Owners>({ state: "loading", list: [] });
  const seq = useRef(0);
  const read = useCallback(async () => {
    const n = ++seq.current;
    setOwners((o) => ({ state: "loading", list: o.list }));
    try {
      const got = await api.access();
      if (n !== seq.current) return;
      setOwners({ state: "ready", list: ownersOf(got.subjects) });
    } catch {
      if (n !== seq.current) return;
      setOwners({ state: "failed", list: [] });
    }
  }, [api]);
  useEffect(() => { void read(); }, [read]);
  return { owners, read };
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

function holdFields(l: Locale, tenant: boolean, owners: Owners, retry: () => void): Fields {
  const ready = owners.state === "ready" && owners.list.length > 0;
  return {
    keys: tenant ? ["owner", "reason", "class"] : ["owner", "reason"],
    saved: tenant ? ["reason", "class"] : ["reason"],
    ready,
    check: (d) => {
      const e: FieldErrors = {};
      if (!d.owner || !owners.list.some((o) => o.id === d.owner)) e.owner = k.form.ownerMissing(l);
      const text = holdTextError(l, d.reason ?? "");
      if (text) e.reason = text;
      return e;
    },
    values: (d) => holdValues(d, tenant),
    render: (d, set, errors, id) => (
      <>
        <div className="service-field p04-select-field">
          <label htmlFor={id + "-owner"}>{k.form.owner(l)}</label>
          {errors.owner && <p className="service-field-error" id={id + "-owner-error"}>{errors.owner}</p>}
          {owners.state === "failed" ? (
            <div className="live-error p04-owners-failed" role="alert">
              <p>{k.form.ownersFailed(l)}</p>
              <button type="button" className="live-secondary" onClick={retry}>{k.form.tryAgain(l)}</button>
            </div>
          ) : owners.state === "loading" && owners.list.length === 0 ? (
            <p className="live-meta" role="status">{k.form.ownersLoading(l)}</p>
          ) : owners.list.length === 0 ? (
            <p className="live-empty" role="status">{k.form.ownersEmpty(l)}</p>
          ) : (
            <select
              id={id + "-owner"} value={owners.list.some((o) => o.id === d.owner) ? d.owner : ""} aria-invalid={!!errors.owner || undefined}
              aria-describedby={[id + "-owner-hint", errors.owner ? id + "-owner-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set("owner", e.target.value)}
            >
              <option value="">{k.form.ownerPlaceholder(l)}</option>
              {owners.list.map((o) => <option key={o.id} value={o.id}>{ownerOption(o)}</option>)}
            </select>
          )}
          <p className="service-hint" id={id + "-owner-hint"}>{k.form.ownerHint(l)}</p>
        </div>
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
  const { owners, read } = useOwners(api);
  const ownerLogin = (id: string) => owners.list.find((o) => o.id === id)?.login ?? "";
  return (
    <CommandForm
      {...shared} command="retention.place_hold" target={target} fields={holdFields(l, tenant, owners, () => void read())}
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
      hooks={{ field: (how: FieldReaction) => { if (how === "owner") void read(); } }}
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
