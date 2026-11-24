import type { Locale } from "../../../../shared/types";
import { letterActions, type Delivery, type LetterAction, type RequestValue } from "../../../../service/integrations";
import { k } from "./live-copy";
import { actor, blockerPolicy, deliveriesCut, deliveriesShown, letterTarget, pendingNow, refusedNow, subscriptionRevoked, waitsForResume, type CommandKey } from "./live-model";
import { Code, Known, ListOr, Num, Prose, Who, href, ltr } from "./live-format";
import { CommandForm, RowNote, blockerLines, plain, useOpen, type Blocker, type Shared } from "./LiveCommand";
import { letterRequestTarget, redeliveryTarget } from "./live-model";
import { letterApplies, lostFor, lostRight } from "./live-rights";
import { r } from "./rights-copy";

const commandOf = (a: LetterAction) => ("integrations.letter_" + a) as CommandKey;
const order: LetterAction[] = ["approve_redeliver", "request_redeliver", "request_dismiss", "approve_dismiss", "withdraw_redeliver", "withdraw_dismiss"];

function actionLabel(l: Locale, a: LetterAction): string {
  return a === "withdraw_redeliver" ? r.withdrawRedeliver(l) : a === "withdraw_dismiss" ? r.withdrawDismiss(l) : k.letterAction(l, a);
}

function Outbox({ l, page }: Shared) {
  const o = page.events.outbox;
  return (
    <dl className="p02-outbox">
      <div><dt>{k.oldestEvent(l)}</dt><dd className="p02-num"><Num l={l} n={o.floor} /></dd></div>
      <div><dt>{k.newestEvent(l)}</dt><dd className="p02-num"><Num l={l} n={o.head} /></dd></div>
      <div><dt>{k.dispatcher(l)}</dt><dd><Code l={l} value={page.events.dispatcher} say={k.connection} /></dd></div>
    </dl>
  );
}

function Types({ l, page }: Shared) {
  return (
    <section className="p02-section p02-types" aria-labelledby="p02-types-title">
      <h3 id="p02-types-title">{k.typesTitle(l)}</h3>
      <ListOr l={l} items={page.events.types} empty={k.noTypes(l)}>
        {(list) => (
          <div className="live-table">
            <table className="p02-table">
              <colgroup><col className="p02-c20" /><col className="p02-c10" /><col className="p02-c20" /><col className="p02-c10" /><col className="p02-c20" /><col className="p02-c20" /></colgroup>
              <thead>
                <tr>
                  <th scope="col">{k.colEventType(l)}</th><th scope="col" className="p02-num">{k.colSchema(l)}</th><th scope="col">{k.colFields(l)}</th>
                  <th scope="col">{k.colState(l)}</th><th scope="col">{k.colProposedBy(l)}</th><th scope="col">{k.colApprovedBy(l)}</th>
                </tr>
              </thead>
              <tbody>
                {list.map((t, i) => (
                  <tr key={t.name || i}>
                    <td className="p02-first"><Known l={l} value={t.name} /></td>
                    <td data-label={k.colSchema(l)} className="p02-num"><Known l={l} value={t.schema_version} /></td>
                    <td data-label={k.colFields(l)}><ListOr l={l} items={t.fields}>{(f) => f.map((x) => <span key={x} className="live-sub">{ltr(x)}</span>)}</ListOr></td>
                    <td data-label={k.colState(l)}><span className="p02-tag"><Code l={l} value={t.state} say={k.eventState} /></span></td>
                    <td data-label={k.colProposedBy(l)}><Who l={l} login={t.proposer_login} id={t.proposer} /></td>
                    <td data-label={k.colApprovedBy(l)}><Who l={l} login={t.approver_login} id={t.approver} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
      <p className="p02-small live-quiet p02-band-note">{k.typesNote(l)}</p>
    </section>
  );
}

function RequestLine({ l, r, lead }: { l: Locale; r: RequestValue; lead: (l: Locale) => string }) {
  if (r === null) return null;
  if (r === "unknown") return <p className="p02-small">{k.requestUnknown(l)}</p>;
  const who = actor(r.by_login, r.by);
  return (
    <p className="p02-small p02-request">
      <span>{lead(l)}</span> {who ? ltr(who) : k.unknown(l)}
      <span className="live-sub">{k.reason(l)}: <Prose l={l} value={r.rationale} empty={k.noReason(l)} /></span>
    </p>
  );
}

function EventCell({ l, d }: { l: Locale; d: Delivery }) {
  return (
    <>
      <span className="live-sub"><Known l={l} value={d.source} /></span>
      <span className="live-sub"><Known l={l} value={d.event_id} /></span>
    </>
  );
}

function Letter({ d, ...shared }: Shared & { d: Delivery }) {
  const { l } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<LetterAction>();
  const target = letterTarget(d);
  const requestTarget = (v: RequestValue) => (v !== null && v !== "unknown" ? letterRequestTarget(v) : "");
  const targetOf = (a: LetterAction) => (a === "withdraw_redeliver" ? requestTarget(d.redeliver_request) : a === "withdraw_dismiss" ? requestTarget(d.dismiss_request) : target);
  const usable = (a: LetterAction) => shared.canCommand && !!targetOf(a) && (d.actions[a].allowed || !!pendingNow(commandOf(a), targetOf(a)));
  const lost = target ? lostFor(letterActions.filter((action) => action !== "reapprove").map((action) => [commandOf(action), targetOf(action)])) : "";
  const kept = !target ? "" : lost ? r.noLongerHeld(l, lost) : letterActions.some((a) => a !== "reapprove" && refusedNow(commandOf(a), targetOf(a))) ? k.kept(l, "letter") : "";
  const explains = (a: LetterAction) => {
    const code = d.actions[a].blocked;
    if (a === "withdraw_redeliver" || a === "withdraw_dismiss") return false;
    return blockerPolicy(code) === "explain" && (code !== "no_right" && code !== "requester_lost_right" || letterApplies(d, a));
  };
  const blockers: Blocker[] = [];
  if (!target) blockers.push(plain(k.noCommand(l)));
  else for (const a of order) if (!usable(a) && !d.actions[a].allowed && explains(a)) blockers.push(...blockerLines(l, d.actions[a].blocked, a, k.letterAction(l, a)));
  const shown = open ? [] : order.filter(usable);
  const existing = (a: LetterAction) => (a.endsWith("redeliver") ? d.redeliver_request : d.dismiss_request);
  return (
    <div className="p02-letter">
      <h4>{k.letterTitle(l)}</h4>
      {d.redeliver_request === null && d.dismiss_request === null && <p className="p02-small">{k.noRequests(l)}</p>}
      <RequestLine l={l} r={d.redeliver_request} lead={k.redeliverRequested} />
      <RequestLine l={l} r={d.dismiss_request} lead={k.dismissRequested} />
      <RowNote l={l} diagnostics={diagnostics(letterActions.filter((action) => action !== "reapprove").map((action) => [commandOf(action), targetOf(action)]))} kept={kept} blockers={blockers} focus={focusNote} />
      {shown.length > 0 && (
        <div className="live-buttons">
          {shown.map((a) => (
            <button key={a} type="button" className={a === "approve_redeliver" ? "live-primary" : "live-secondary"} aria-expanded={open === a} data-opener={mark(a)} onClick={() => toggle(a)}>
              {actionLabel(l, a)}
            </button>
          ))}
        </div>
      )}
      {open && targetOf(open) && (
        <CommandForm
          key={open} {...shared} command={commandOf(open)} target={targetOf(open)} values={{}} context={{ event: d.event_id }}
          object={r.letterObject(l, d.id)}
          title={open === "withdraw_redeliver" || open === "withdraw_dismiss" ? r.confirmWithdraw(l) : k.letterConfirm(l, open)}
          facts={
            <>
              <div><dt>{k.colDelivery(l)}</dt><dd><Known l={l} value={d.id} /></dd></div>
              <div><dt>{k.colSubscription(l)}</dt><dd><Known l={l} value={d.subscription} /></dd></div>
              <div><dt>{k.colSequence(l)}</dt><dd><Num l={l} n={d.sequence} /></dd></div>
              <div><dt>{k.colEvent(l)}</dt><dd><EventCell l={l} d={d} /></dd></div>
              <div><dt>{k.colReason(l)}</dt><dd><Code l={l} value={d.reason} say={k.deliveryReason} /></dd></div>
              {existing(open) !== null && <div><dt>{k.existingRequest(l)}</dt><dd><RequestLine l={l} r={existing(open)} lead={open.endsWith("redeliver") ? k.redeliverRequested : k.dismissRequested} /></dd></div>}
            </>
          }
          impact={open === "withdraw_redeliver" || open === "withdraw_dismiss" ? r.withdrawImpact(l) : k.letterImpact(l, open)}
          verb={open === "withdraw_redeliver" || open === "withdraw_dismiss" ? actionLabel(l, open) : k.letterVerb(l, open, d.id)}
          done={(fields) => (open === "withdraw_redeliver" || open === "withdraw_dismiss" ? r.withdrawnDone(l, actionLabel(l, open), d.id) : k.letterDone(l, open, fields.redelivery || k.unknown(l)))}
          onClose={close}
        />
      )}
    </div>
  );
}

function Held({ d, ...shared }: Shared & { d: Delivery }) {
  const { l } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<"reapprove">();
  const target = redeliveryTarget(d);
  const allowed = d.actions.reapprove.allowed;
  const usable = shared.canCommand && !!target && (allowed || !!pendingNow("integrations.redelivery_reapprove", target));
  const lost = target ? lostRight("integrations.redelivery_reapprove", target) : "";
  const kept = lost ? r.noLongerHeld(l, lost) : target && refusedNow("integrations.redelivery_reapprove", target) ? k.kept(l, "letter") : "";
  const code = d.actions.reapprove.blocked;
  const blockers: Blocker[] = [];
  if (!target) blockers.push(plain(k.noCommand(l)));
  else if (!usable && !allowed) {
    if (code === "no_right" || code === "requester_lost_right") blockers.push(...blockerLines(l, code, "reapprove", r.againButton(l)));
    else if (code === "own_request") blockers.push({ label: r.againButton(l), text: d.requester === shared.me ? r.againOwnRequest(l) : r.againOwnApproval(l) });
    else if (blockerPolicy(code) === "explain") blockers.push(...blockerLines(l, code, "approve_redeliver", r.againButton(l)));
  }
  return (
    <div className="p02-letter p02-held">
      <h4>{r.heldTitle(l)}</h4>
      <p className="p02-small">{r.requestedBy(l, actor(d.requester_login, d.requester) || k.unknown(l))}</p>
      <p className="p02-small">{k.approvedBy(l, actor(d.approver_login, d.approver) || k.unknown(l))}</p>
      <p className="p02-held-status">{r.heldStatus(l)}</p>
      <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={blockers} focus={focusNote} />
      {usable && !open && (
        <div className="live-buttons">
          <button type="button" className="live-primary" aria-expanded={false} data-opener={mark("reapprove")} onClick={() => toggle("reapprove")}>{r.againButton(l)}</button>
        </div>
      )}
      {open && target && (
        <CommandForm
          {...shared} command="integrations.redelivery_reapprove" target={target} values={{}} object={r.redeliveryObject(l, d.id)}
          title={r.confirmAgain(l)}
          facts={
            <>
              <div><dt>{k.colDelivery(l)}</dt><dd><Known l={l} value={d.id} /></dd></div>
              <div><dt>{k.colEvent(l)}</dt><dd><EventCell l={l} d={d} /></dd></div>
              <div><dt>{k.colSubscription(l)}</dt><dd><Known l={l} value={d.subscription} /></dd></div>
              <div><dt>{r.requestedBy(l, "").trim()}</dt><dd><Who l={l} login={d.requester_login} id={d.requester} /></dd></div>
              <div><dt>{r.previouslyApprovedBy(l)}</dt><dd><Who l={l} login={d.approver_login} id={d.approver} /></dd></div>
            </>
          }
          impact={r.againImpact(l)}
          verb={r.againVerb(l, d.id)}
          done={() => r.againDone(l, d.id)}
          onClose={close}
        />
      )}
    </div>
  );
}

function StateTag({ l, d, waits }: { l: Locale; d: Delivery; waits: boolean }) {
  if (d.hold) return <>{r.held(l)}</>;
  if (d.kind === "cancelled" && d.reason === "requester_lost_right") return <>{r.closedState(l)}</>;
  return waits ? <>{k.waitingResume(l)}</> : <Code l={l} value={d.state} say={k.job} />;
}

function Deliveries(shared: Shared) {
  const { l, page, selection } = shared;
  const list = deliveriesShown(page, selection.subscription);
  return (
    <section className="p02-section p02-deliveries" aria-labelledby="p02-deliveries-title">
      <h3 id="p02-deliveries-title">{k.deliveriesTitle(l)}</h3>
      <p className="p02-small live-quiet">{k.deliveriesNote(l)}</p>
      {list !== null && deliveriesCut(page, selection.subscription) !== "complete" && <p className="p02-small" data-deliveries-cut="true">{deliveriesCut(page, selection.subscription) === "cut" ? k.deliveriesCut(l) : k.deliveriesMaybeCut(l)}</p>}
      {selection.subscription && (
        <p className="p02-filter">
          {k.filterLine(l)} {ltr(selection.subscription)} · <a className="p02-link" href={href(l, "events")}>{k.showAll(l)}</a>
        </p>
      )}
      <ListOr l={l} items={list} empty={k.noDeliveries(l)}>
        {(rows) => (
          <div className="live-table">
            <table className="p02-table">
              <colgroup>
                <col className="p02-c10" /><col className="p02-c16" /><col className="p02-c10" /><col className="p02-c18" /><col className="p02-c10" /><col className="p02-c10" />
                <col className="p02-c14" /><col className="p02-c12" />
              </colgroup>
              <thead>
                <tr>
                  <th scope="col">{k.colDelivery(l)}</th><th scope="col">{k.colSubscription(l)}</th><th scope="col" className="p02-num">{k.colSequence(l)}</th>
                  <th scope="col">{k.colEvent(l)}</th><th scope="col">{k.colKind(l)}</th><th scope="col">{k.colState(l)}</th><th scope="col">{k.colReason(l)}</th>
                  <th scope="col" className="p02-num">{k.colAttempts(l)}</th>
                </tr>
              </thead>
              {rows.map((d, i) => {
                const current = selection.delivery !== "" && selection.delivery === d.id;
                const revoked = subscriptionRevoked(page, d);
                return (
                  <tbody key={d.id || i} className={"p02-group" + (current ? " live-selected" : "")}>
                    <tr id={d.id ? "p02-delivery-" + d.id : undefined} aria-current={current || undefined}>
                      <td className="p02-first"><Known l={l} value={d.id} /></td>
                      <td data-label={k.colSubscription(l)}>{d.subscription ? <a className="p02-link" href={href(l, "channels", { subscription: d.subscription })}>{ltr(d.subscription)}</a> : k.unknown(l)}</td>
                      <td data-label={k.colSequence(l)} className="p02-num"><Num l={l} n={d.sequence} /></td>
                      <td data-label={k.colEvent(l)}><EventCell l={l} d={d} /></td>
                      <td data-label={k.colKind(l)}><Code l={l} value={d.kind} say={k.deliveryKind} /></td>
                      <td data-label={k.colState(l)}><span className="p02-tag"><StateTag l={l} d={d} waits={waitsForResume(page, d)} /></span></td>
                      <td data-label={k.colReason(l)}>{d.hold ? r.heldReason(l) : <Code l={l} value={d.reason} say={k.deliveryReason} />}</td>
                      <td data-label={k.colAttempts(l)} className="p02-num"><Num l={l} n={d.attempts} /></td>
                    </tr>
                    {(d.outcome_unknown !== false || revoked || d.kind === "dead" || d.hold !== "") && (
                      <tr className="p02-follow">
                        <td colSpan={8}>
                          {d.outcome_unknown === true && <p className="p02-outcome">{k.outcomeUnknown(l)}</p>}
                          {d.outcome_unknown === null && <p className="p02-small">{k.outcomeMissing(l)}</p>}
                          {revoked && <p className="p02-small">{k.revokedRow(l)}</p>}
                          {d.kind === "dead" && <Letter {...shared} d={d} />}
                          {d.hold !== "" && <Held {...shared} d={d} />}
                        </td>
                      </tr>
                    )}
                  </tbody>
                );
              })}
            </table>
          </div>
        )}
      </ListOr>
    </section>
  );
}

export function EventsView(shared: Shared) {
  return (
    <>
      <Outbox {...shared} />
      <Types {...shared} />
      <Deliveries {...shared} />
    </>
  );
}
