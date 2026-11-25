import { holdCountKeys, type HoldRow } from "../../../../service/retention";
import { AdminError } from "../../../../service/admin";
import { useOlderRows } from "../../../../shared/older-rows";
import type { Locale } from "../../../../shared/types";
import { plurals } from "./live-codes";
import { k } from "./live-copy";
import {
  blockerPolicy, duplicateIn, holdTarget, keptKey, pendingNow, pendingReleaseFor, refusedNow, rowId, stateSentence, tenantTarget, textProblem,
} from "./live-model";
import { Code, Free, HoldFacts, Rich, Stamp, Who, coversText, holdContext, holdTextError, href, ltr, ItemsFoot, MoreLine } from "./live-format";
import { CommandForm, RowNote, line, useOpen, type Blocker, type FieldErrors, type Fields, type Shared } from "./LiveCommand";
import { HoldCommand, TextField } from "./LiveHoldForm";
import { Strip } from "./LiveExpiry";
import { Proposals } from "./LiveProposals";

function releaseFields(l: Locale): Fields {
  return {
    keys: ["reason"],
    saved: ["reason"],
    check: (d) => {
      const e: FieldErrors = {};
      const text = holdTextError(l, d.reason ?? "", true);
      if (text) e.reason = text;
      return e;
    },
    values: (d) => ({ reason: textProblem(d.reason ?? "") === "control" ? d.reason ?? "" : (d.reason ?? "").trim() }),
    render: (d, set, errors, id) => (
      <TextField l={l} id={id + "-reason"} label={k.form.releaseReason(l)} value={d.reason ?? ""} error={errors.reason ?? ""} set={(v) => set("reason", v)} />
    ),
  };
}

function countLabel(l: Locale, key: string): string {
  return key === "outcome_pending" ? k.beingDeleted(l) : k.holdOutcome(l, key);
}

function Outcomes({ l, h }: { l: Locale; h: HoldRow }) {
  const list = h.outcomes;
  return (
    <>
      {list === null ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : list.length === 0 ? (
        <p className="p04-small">{k.noneListed(l)}</p>
      ) : (
        <div className="live-table">
          <table className="p04-table p04-outcomes">
            <caption className="visually-hidden">{k.outcomesCaption(l)}</caption>
            <colgroup><col className="p04-c36" /><col className="p04-c30" /><col className="p04-c34" /></colgroup>
            <thead><tr><th scope="col">{k.colItem(l)}</th><th scope="col">{k.colOutcome(l)}</th><th scope="col">{k.colStoreResults(l)}</th></tr></thead>
            <tbody>
              {list.map((o, n) => (
                <tr key={o.artifact || n}>
                  <td className="p04-first">{o.artifact ? ltr(o.artifact) : k.unknown(l)}</td>
                  <td data-label={k.colOutcome(l)}><Code l={l} value={o.outcome} say={k.holdOutcome} /></td>
                  <td data-label={k.colStoreResults(l)}>
                    {o.stores === null ? k.unknown(l) : o.stores.length === 0 ? k.none(l) : o.stores.map((s) => (
                      <span key={s.store} className="live-sub">{ltr(s.store)} <Code l={l} value={s.result} say={k.storeResult} /></span>
                    ))}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="p04-small live-quiet">{k.onlyListed(l)}</p>
      {h.outcomes_more === true && <p className="p04-small live-quiet">{k.moreOutcomes(l)}</p>}
      {h.outcomes_more === null && <p className="p04-small live-quiet">{k.maybeMore(l)}</p>}
    </>
  );
}

function Hold({ h, ...shared }: Shared & { h: HoldRow }) {
  const { l, page, selection, canCommand } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<"release">();
  const target = holdTarget(h);
  const pending = target ? pendingNow("retention.propose_release", target) : undefined;
  const usable = canCommand && !!target && (h.can_propose_release || !!pending);
  const already = pendingReleaseFor(page, h);
  const refused = target ? refusedNow("retention.propose_release", target) : "";
  const kept = refused ? k.kept(l, keptKey("retention.propose_release", refused)) : "";
  const blockers: Blocker[] = [];
  let state = "";
  if (!target) blockers.push(line("", k.narrow(l, "no_command")));
  else if (!usable && !already && !h.can_propose_release) {
    const code = h.release_blocked;
    if (blockerPolicy(code, "propose_release") === "explain") {
      blockers.push(line(k.proposeRelease(l), code === "placer" ? k.placer(l, "propose_release") : code === "already_proposed" ? k.alreadyRelease(l, "", false) : k.blocker(l, code)));
    } else if (stateSentence(code, "propose_release")) state = k.stateSentence(l, stateSentence(code, "propose_release"));
  }
  const current = selection.hold !== "" && selection.hold === h.id;
  const head = "p04-hold-h-" + rowId("x", h.id || "x");
  const scope = h.counts_scope;
  return (
    <article className={"p04-hold" + (current ? " live-selected" : "")} id={h.id ? rowId("hold", h.id) : undefined} aria-labelledby={head} aria-current={current || undefined}>
      <div className="p04-head">
        <h3 id={head}>{h.id ? ltr(h.id) : k.unknown(l)}</h3>
        <span className="p04-tag">{h.released === null ? k.unknown(l) : k.holdState(l, h.released ? "released" : "in_force")}</span>
      </div>
      <div className="p04-hold-grid">
        <dl className="p04-facts p04-hold-facts">
          <div><dt>{k.factOwner(l)}</dt><dd><Who l={l} p={h.owner} /></dd></div>
          <div><dt>{k.factPlacedBy(l)}</dt><dd><Who l={l} p={h.placer} /></dd></div>
          <div><dt>{k.factPlaced(l)}</dt><dd><Stamp l={l} at={h.placed_at} /></dd></div>
          <div><dt>{k.factHoldReason(l)}</dt><dd><Free l={l} value={h.reason} /></dd></div>
          <div><dt>{k.factPlacingReason(l)}</dt><dd><Free l={l} value={h.rationale} /></dd></div>
        </dl>
        <div className="p04-hold-end">
          <h4>{k.covers(l)}</h4>
          <p className="p04-covers"><Rich text={coversText(l, h)} /></p>
          <h4>{h.released === true ? k.couldNotPreserve(l) : k.cannotPreserve(l)}</h4>
          {scope === "workspace" && h.counts ? (
            <Strip l={l} counts={h.counts} keys={holdCountKeys} label={countLabel} cls="p04-strip-3" />
          ) : scope === "other_scope" ? (
            <p>{k.otherScopeCounts(l)}</p>
          ) : (
            <p>{k.unknown(l)}</p>
          )}
          <Outcomes l={l} h={h} />
        </div>
      </div>
      <div className="p04-actions">
        <RowNote l={l} diagnostics={diagnostics(target)}
          kept={kept} blockers={blockers} focus={focusNote} state={state}
          extra={already ? <p><a className="p04-link" href={href(l, "holds", { proposal: already.number })}><span><Rich text={k.alreadyRelease(l, already.number, already.own)} /></span></a></p> : undefined}
        />
        {usable && !open && !already && (
          <div className="live-buttons">
            <button type="button" className="live-secondary p04-opener" aria-expanded={false} data-opener={mark("release")} onClick={() => toggle("release")}>{k.proposeRelease(l)}</button>
          </div>
        )}
        {open && usable && (
          <CommandForm
            {...shared} command="retention.propose_release" target={target} fields={releaseFields(l)}
            heading={k.releaseHeading(l, h.id)} title={k.confirmRelease(l)} verb={k.releaseVerb(l)}
            context={() => ({ view: "holds", ...holdContext(l, h) })}
            facts={(ctx, d) => (
              <>
                <HoldFacts l={l} ctx={ctx} />
                <div className="p04-span"><dt>{k.factReleaseReason(l)}</dt><dd><Free l={l} value={(d.reason ?? "").trim()} /></dd></div>
              </>
            )}
            impact={() => <p>{k.releaseProposeImpact(l)}</p>}
            done={(fields, st, snapshot) => {
              const n = fields.proposal ?? "";
              if (st !== "pending" || !n) return k.resultUnknown(l, st);
              return duplicateIn(snapshot, n) ? k.duplicate(l, n) : k.releaseProposed(l, n, h.id);
            }}
            onClose={close}
          />
        )}
      </div>
    </article>
  );
}

function Tenant(shared: Shared) {
  const { l, page, workspace, canCommand } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<"tenant">();
  const target = tenantTarget(page, workspace);
  const pending = target ? pendingNow("retention.place_hold", target) : undefined;
  const usable = canCommand && !!target;
  const refused = target ? refusedNow("retention.place_hold", target) : "";
  const kept = refused ? k.kept(l, keptKey("retention.place_hold", refused)) : "";
  return (
    <div className="p04-tenant">
      <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={target ? [] : [line("", k.narrow(l, "no_command"))]} focus={focusNote} />
      {usable && !open && (
        <>
          <button type="button" className="live-secondary p04-opener" aria-expanded={false} data-opener={mark("tenant")} onClick={() => toggle("tenant")}>{k.placeWorkspaceHold(l)}</button>
          <p className="p04-small live-quiet">{k.tenantNote(l)}</p>
          {pending && <p className="p04-small">{k.pendingAttempt(l)}</p>}
        </>
      )}
      {open && usable && (
        <HoldCommand {...shared} target={target} tenant base={{ view: "holds", target_kind: "workspace" }} heading={k.workspaceHoldHeading(l)} onClose={close} />
      )}
    </div>
  );
}

export function HoldsView(shared: Shared) {
  const { l, page } = shared;
  const holds = page.holds;
  const refs = holds.references;
  const facts = holds.rows === null ? null : holds.released;
  const older = useOlderRows<HoldRow>(page.revision, facts, holds.rows, shared.api.retentionHolds, (e) => {
    if (e instanceof AdminError && e.failure === "denied") { void shared.onReload(); return true; }
    if (e instanceof AdminError && e.failure === "unauthenticated") { shared.onSignedOut(); return true; }
    return false;
  }, false);
  const rows = older.rows;
  const released = (rows ?? []).filter((h) => h.released === true);
  const inForce = (rows ?? []).filter((h) => h.released !== true);
  const cut = facts !== null && older.counted > (holds.rows ?? []).filter((h) => h.released === true).length;
  const failed = older.status === "failed";
  return (
    <>
      <Tenant {...shared} />
      {facts === null && <MoreLine l={l} more={holds.more} total={holds.total} forms={plurals.showingNewest} />}
      {rows === null ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : rows.length === 0 && holds.total !== 0 && !older.more ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : rows.length === 0 && !older.more ? (
        <p className="live-empty">{k.noHolds(l)}</p>
      ) : (
        <>
          <section className="p04-held-group p04-held-in-force" aria-labelledby="p04-held-in-force">
            <h4 id="p04-held-in-force">{k.heldInForce(l, inForce.length)}</h4>
            {inForce.length === 0 ? <p className="live-empty">{k.heldNoneInForce(l)}</p> : inForce.map((h, n) => <Hold key={h.id || "f" + n} h={h} {...shared} />)}
          </section>
          <section className="p04-held-group p04-held-released" aria-labelledby="p04-held-released">
            <h4 id="p04-held-released">{k.heldReleased(l, released.length)}</h4>
            {cut && <p className="p04-small live-quiet p04-held-count" role="status" ref={older.countRef} tabIndex={-1}>{!older.more && released.length >= older.counted ? k.heldAll(l, older.counted) : k.heldShown(l, released.length, older.counted)}</p>}
            {older.reset && <p className="p04-small p04-held-reset" role="status">{k.heldReset(l)}</p>}
            {released.length === 0 && !older.more ? <p className="live-empty">{k.heldNoneReleased(l)}</p> : released.map((h, n) => <Hold key={h.id || "r" + n} h={h} {...shared} />)}
            {older.more && (
              <div className="p04-held-more">
                {failed && <p role="alert">{k.heldFailed(l)}</p>}
                <button type="button" className="live-secondary p04-held-more-button" aria-disabled={older.status === "loading" || undefined} onClick={older.load}>
                  {failed ? k.heldTryAgain(l) : older.status === "loading" ? k.heldLoading(l) : k.heldMore(l)}
                </button>
              </div>
            )}
          </section>
        </>
      )}
      <section className="p04-section p04-required" aria-labelledby="p04-required-title">
        <h3 id="p04-required-title">{k.requiredTitle(l)}</h3>
        <p className="p04-prose">{k.requiredLead(l)}</p>
        <p className="p04-small live-quiet"><Rich text={k.populationNote(l, refs.population)} /></p>
        {!shared.listOf && <MoreLine l={l} more={refs.more} total={refs.total} forms={plurals.showingOf} />}
        {refs.rows === null ? (
          <p className="live-empty">{k.unknown(l)}</p>
        ) : refs.rows.length === 0 ? (
          <p className="live-empty">{refs.total === 0 ? k.noneDot(l) : k.unknown(l)}</p>
        ) : (
          <div className="live-table p04-cap-720">
            <table className="p04-table">
              <caption className="visually-hidden">{k.requiredCaption(l)}</caption>
              <colgroup><col className="p04-c60" /><col className="p04-c40" /></colgroup>
              <thead><tr><th scope="col">{k.colItem(l)}</th><th scope="col">{k.colClass(l)}</th></tr></thead>
              <tbody>
                {refs.rows.map((r, n) => (
                  <tr key={r.id || n}>
                    <td className="p04-first">{r.id ? ltr(r.id) : k.unknown(l)}</td>
                    <td data-label={k.colClass(l)}><Code l={l} value={r.class} say={k.className} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {shared.listOf && <ItemsFoot l={l} kind="refs" list={shared.listOf("refs", refs.next)} shown={refs.rows?.length ?? 0} total={refs.total} />}
      </section>
      <Proposals kind="release" {...shared} />
    </>
  );
}
