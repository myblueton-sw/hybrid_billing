import { useState } from "react";
import type { Locale } from "../../../../shared/types";
import { subscriptionActions, type HistoryEntry, type Subscription, type SubscriptionAction } from "../../../../service/integrations";
import { AdminError } from "../../../../service/admin";
import { useOlderRows, type OlderPage } from "../../../../shared/older-rows";
import { k } from "./live-copy";
import {
  blockerPolicy, coveredBy, draftOf, duplicateIn, frozenValues, keepDraft, normalTypes, pendingNow, refusedNow, subscribeBlocker, subscribeProblems, subscriptionTarget,
  validTarget, webhooksTarget, type CommandKey,
} from "./live-model";
import { Code, Known, ListOr, Num, Prose, Who, fsi, href, ltr, stamp } from "./live-format";
import { CommandForm, RowNote, blockerLines, plain, useOpen, type Blocker, type Shared } from "./LiveCommand";
import { resumeRequestTarget } from "./live-model";
import { lostFor, lostRight, subscriptionApplies } from "./live-rights";
import { r } from "./rights-copy";
import { Marked } from "./LiveProposals";

const commandOf = (a: SubscriptionAction) => ("integrations.subscription_" + a) as CommandKey;
const order: SubscriptionAction[] = ["approve", "approve_resume", "pause", "request_resume", "revoke", "withdraw", "withdraw_resume"];

function SubState({ l, s }: { l: Locale; s: Subscription }) {
  if (s.state === "revoked" && s.reason === "requester_lost_right") return <>{r.closedState(l)}</>;
  if (s.state === "revoked" && s.reason === "withdrawn") return <>{r.withdrawnState(l)}</>;
  return <Code l={l} value={s.state} say={k.subscriptionState} />;
}

function actionLabel(l: Locale, a: SubscriptionAction): string {
  return a === "withdraw" ? r.withdrawSubscription(l) : a === "withdraw_resume" ? r.withdrawResume(l) : k.subAction(l, a);
}

function Destinations({ l, page }: Shared) {
  return (
    <section className="p02-destinations" aria-labelledby="p02-destinations-title">
      <h3 id="p02-destinations-title">{k.destinationsTitle(l)}</h3>
      <ListOr l={l} items={page.channels.destinations} empty={k.noDestinations(l)}>
        {(list) => (
          <div className="live-table">
            <table className="p02-table">
              <colgroup><col className="p02-c40" /><col className="p02-c20" /><col className="p02-c40" /></colgroup>
              <thead><tr><th scope="col">{k.colDestination(l)}</th><th scope="col">{k.colState(l)}</th><th scope="col">{k.colSource(l)}</th></tr></thead>
              <tbody>
                {list.map((d, i) => (
                  <tr key={d.id || i}>
                    <td className="p02-first"><Known l={l} value={d.id} /></td>
                    <td data-label={k.colState(l)}><span className="p02-tag"><Code l={l} value={d.state} say={k.destinationState} /></span></td>
                    <td data-label={k.colSource(l)}><Code l={l} value={d.source} say={k.source} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
      <p className="p02-small live-quiet p02-band-note">{k.destinationsNote(l)}</p>
    </section>
  );
}

function Fold({ l, s, revision, read, onRefused }: { l: Locale; s: Subscription; revision: string; read?: (subscription: string, after: string) => Promise<OlderPage<HistoryEntry>>; onRefused?: (e: unknown) => boolean }) {
  const older = useOlderRows<HistoryEntry>(s.id + "\u0000" + revision, s.history_facts, s.history, read ? (after) => read(s.id, after) : undefined, onRefused ?? (() => false));
  const failed = older.status === "failed";
  return (
    <details className="p02-fold">
      <summary>{k.historyFold(l)}</summary>
      <h4>{k.historyTitle(l)}</h4>
      {older.cut && <p className="p02-small live-quiet p02-history-count" role="status" ref={older.countRef} tabIndex={-1}>{older.all ? k.historyAll(l, older.total) : k.historyShown(l, older.rows?.length ?? 0, older.total)}</p>}
      {older.reset && <p className="p02-small p02-history-reset" role="status">{k.historyReset(l)}</p>}
      <ListOr l={l} items={older.rows} empty={k.noHistory(l)}>
        {(list) => (
          <div className="live-table">
            <table className="p02-table">
              <colgroup><col className="p02-c30" /><col className="p02-c30" /><col className="p02-c40" /></colgroup>
              <thead><tr><th scope="col">{k.colAction(l)}</th><th scope="col">{k.colBy(l)}</th><th scope="col">{k.colReason(l)}</th></tr></thead>
              <tbody>
                {list.map((h, i) => (
                  <tr key={h.number ?? "n" + i} className="p02-history-row">
                    <td className="p02-first"><Code l={l} value={h.action} say={k.history} /></td>
                    <td data-label={k.colBy(l)}>{h.reason === "requester_lost_right" ? r.closedAutomatically(l) : <Who l={l} login={h.actor_login} id={h.actor} />}</td>
                    <td data-label={k.colReason(l)}>
                      {h.reason === "requester_lost_right" ? <Marked text={r.closedFound(l, h.actor_login || h.actor || k.unknown(l), h.request || k.unknown(l))} /> : <Prose l={l} value={h.rationale} empty={k.noReason(l)} />}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
      {older.more && (
        <div className="p02-history-more">
          {failed && <p role="alert">{k.historyFailed(l)}</p>}
          <button type="button" className="live-secondary p02-history-more-button" aria-disabled={older.status === "loading" || undefined} onClick={older.load}>
            {failed ? k.historyTryAgain(l) : older.status === "loading" ? k.historyLoading(l) : k.historyMore(l)}
          </button>
        </div>
      )}
      <h4>{k.keysTitle(l)}</h4>
      <ListOr l={l} items={s.keys} empty={k.noKeys(l)}>
        {(list) => (
          <div className="live-table">
            <table className="p02-table">
              <colgroup><col className="p02-c20" /><col className="p02-c16" /><col className="p02-c16" /><col className="p02-c14" /><col className="p02-c16" /><col className="p02-c18" /></colgroup>
              <thead>
                <tr>
                  <th scope="col">{k.colKey(l)}</th><th scope="col">{k.colFrom(l)}</th><th scope="col">{k.colUntil(l)}</th><th scope="col">{k.colState(l)}</th>
                  <th scope="col">{k.colProposedBy(l)}</th><th scope="col">{k.colApprovedBy(l)}</th>
                </tr>
              </thead>
              <tbody>
                {list.map((key, i) => (
                  <tr key={key.version || i}>
                    <td className="p02-first"><Known l={l} value={key.version} /></td>
                    <td data-label={k.colFrom(l)}>{fsi(stamp(l, key.not_before))}</td>
                    <td data-label={k.colUntil(l)}>{fsi(stamp(l, key.not_after))}</td>
                    <td data-label={k.colState(l)}><span className="p02-tag"><Code l={l} value={key.state} say={k.keyState} /></span></td>
                    <td data-label={k.colProposedBy(l)}><Who l={l} login={key.proposer_login} id={key.proposer} /></td>
                    <td data-label={k.colApprovedBy(l)}><Who l={l} login={key.approver_login} id={key.approver} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
      <p className="p02-small live-quiet">{k.signerNote(l, s.signer)}</p>
    </details>
  );
}

function Types({ l, types }: { l: Locale; types: string[] | null }) {
  return <ListOr l={l} items={types}>{(list) => list.map((t) => <span key={t} className="live-sub">{ltr(t)}</span>)}</ListOr>;
}

function SubFacts({ l, s }: { l: Locale; s: Subscription }) {
  return (
    <>
      <div><dt>{k.colSubscription(l)}</dt><dd><Known l={l} value={s.id} /></dd></div>
      <div><dt>{k.subscriberIs(l)}</dt><dd><Known l={l} value={s.subscriber} /></dd></div>
      <div><dt>{k.destinationIs(l)}</dt><dd><Known l={l} value={s.destination} /></dd></div>
      <div><dt>{k.eventTypes(l)}</dt><dd><Types l={l} types={s.types} /></dd></div>
      <div><dt>{k.colState(l)}</dt><dd><SubState l={l} s={s} /></dd></div>
      <div><dt>{k.factProposerReason(l)}</dt><dd><Prose l={l} value={s.rationale} empty={k.noReason(l)} /></dd></div>
    </>
  );
}

function SubRow({ s, ...shared }: Shared & { s: Subscription }) {
  const { l, selection } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<SubscriptionAction>();
  const target = subscriptionTarget(s);
  const resumeRequest = s.resume_request !== null && s.resume_request !== "unknown" ? resumeRequestTarget(s.resume_request) : "";
  const targetOf = (a: SubscriptionAction) => (a === "withdraw_resume" ? resumeRequest : target);
  const usable = (a: SubscriptionAction) => shared.canCommand && !!targetOf(a) && (s.actions[a].allowed || !!pendingNow(commandOf(a), targetOf(a)));
  const lost = target ? lostFor(subscriptionActions.map((action) => [commandOf(action), targetOf(action)])) : "";
  const kept = !target ? "" : lost ? r.noLongerHeld(l, lost) : subscriptionActions.some((a) => refusedNow(commandOf(a), targetOf(a))) ? k.kept(l, "subscription") : "";
  const explains = (a: SubscriptionAction) => {
    const code = s.actions[a].blocked;
    if (a === "withdraw" || a === "withdraw_resume") return false;
    return blockerPolicy(code) === "explain" && (code !== "no_right" && code !== "requester_lost_right" || subscriptionApplies(s, a));
  };
  const blockers: Blocker[] = [];
  if (!target) blockers.push(plain(k.noCommand(l)));
  else for (const a of order) if (!usable(a) && !s.actions[a].allowed && explains(a)) blockers.push(...blockerLines(l, s.actions[a].blocked, a, k.subAction(l, a)));
  const shown = open ? [] : order.filter(usable);
  const current = selection.subscription !== "" && selection.subscription === s.id;
  const people = [
    k.proposedBy(l, s.proposer_login || s.proposer || k.unknown(l)),
    s.approver || s.approver_login ? k.approvedBy(l, s.approver_login || s.approver) : "",
    s.resumer || s.resumer_login ? k.resumeBy(l, s.resumer_login || s.resumer) : "",
  ].filter(Boolean);
  return (
    <li className={"p02-sub" + (current ? " live-selected" : "")} id={s.id ? "p02-sub-" + s.id : undefined} aria-current={current || undefined}>
      <div className="p02-meta">
        <p className="p02-strong"><Known l={l} value={s.id} /></p>
        <p>
          <span className="p02-tag"><SubState l={l} s={s} /></span>
          {s.reason && <span className="live-sub p02-small"><Code l={l} value={s.reason} say={k.subscriptionReason} /></span>}
        </p>
        <p className="p02-small">{k.subscriberIs(l)} <Known l={l} value={s.subscriber} /></p>
        <p className="p02-small">{k.destinationIs(l)} <Known l={l} value={s.destination} /></p>
        <p className="p02-small live-quiet">{k.created(l, stamp(l, s.created_at))}</p>
      </div>
      <div className="p02-body">
        <dl className="p02-sub-facts">
          <div><dt>{k.eventTypes(l)}</dt><dd><Types l={l} types={s.types} /></dd></div>
          <div>
            <dt>{k.cursorLag(l)}</dt>
            <dd className="p02-num">{k.cursorWord(l)} <Num l={l} n={s.cursor} /> · <Num l={l} n={s.lag} /> {k.behind(l)}</dd>
          </div>
          <div>
            <dt>{k.gaps(l)}</dt>
            <dd>
              <ListOr l={l} items={s.gaps} empty={k.noGaps(l)}>
                {(list) => list.map((g, i) => (
                  <span key={i} className="live-sub">{k.gapLine(l)} {ltr((g.after === null ? k.unknown(l) : String(g.after)) + "–" + (g.to === null ? k.unknown(l) : String(g.to)))}</span>
                ))}
              </ListOr>
            </dd>
          </div>
        </dl>
        <p className="p02-small">{people.join(" · ")}</p>
        <p className="p02-small">{k.reason(l)}: <Prose l={l} value={s.rationale} empty={k.noReason(l)} /></p>
        {s.id && <p className="p02-small"><a className="p02-link" href={href(l, "events", { subscription: s.id })}>{k.deliveriesLink(l)}</a></p>}
        <Fold l={l} s={s} revision={shared.page.revision} read={shared.api.subscriptionHistory} onRefused={(e) => {
          if (e instanceof AdminError && e.failure === "denied") { shared.onLostAccess(); return true; }
          if (e instanceof AdminError && e.failure === "unauthenticated") { shared.onSignedOut(); return true; }
          return false;
        }} />
        <RowNote l={l} diagnostics={diagnostics(subscriptionActions.map((action) => [commandOf(action), targetOf(action)]))} kept={kept} blockers={blockers} focus={focusNote} />
        {shown.length > 0 && (
          <div className="live-buttons">
            {shown.map((a) => (
              <button
                key={a} type="button" className={a === "approve" || a === "approve_resume" ? "live-primary" : "live-secondary"} aria-expanded={open === a}
                data-opener={mark(a)} onClick={() => toggle(a)}
              >
                {actionLabel(l, a)}
              </button>
            ))}
          </div>
        )}
        {open && targetOf(open) && (
          <CommandForm
            key={open} {...shared} command={commandOf(open)} target={targetOf(open)} values={{}}
            title={open === "withdraw" || open === "withdraw_resume" ? r.confirmWithdraw(l) : k.subConfirm(l, open)}
            facts={<SubFacts l={l} s={s} />}
            impact={open === "withdraw" || open === "withdraw_resume" ? r.withdrawImpact(l) : k.subImpact(l, open)}
            verb={open === "withdraw" || open === "withdraw_resume" ? actionLabel(l, open) : k.subVerb(l, open, s.id)}
            done={(fields, state) => (open === "withdraw" || open === "withdraw_resume" ? r.withdrawnDone(l, actionLabel(l, open), s.id) : k.subDone(l, fields.subscription || s.id, k.result(l, state)))}
            object={r.subscriptionObject(l, s.id)}
            onClose={close}
          />
        )}
      </div>
    </li>
  );
}

function initialChoice(): { destination: string; types: string[] } {
  const from = frozenValues(pendingNow("integrations.subscription_propose", webhooksTarget)) ?? draftOf("integrations.subscription_propose", webhooksTarget);
  return { destination: from.destination ?? "", types: from.types ? from.types.split(",").filter(Boolean) : [] };
}

function ProposeSub(shared: Shared) {
  const { l, page, me } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<"propose">();
  const noRight = page.channels.actions.propose.blocked === "no_right";
  const blocked = subscribeBlocker(page);
  const kept = pendingNow("integrations.subscription_propose", webhooksTarget);
  const usable = !noRight && shared.canCommand && (blocked === null || !!kept);
  const lost = lostRight("integrations.subscription_propose", webhooksTarget);
  const [choice, setChoice] = useState(initialChoice);
  const refused = refusedNow("integrations.subscription_propose", webhooksTarget);
  const destinations = (page.channels.destinations ?? []).filter((d) => validTarget(d.id));
  const active = destinations.filter((d) => d.state === "active");
  const inactive = destinations.filter((d) => d.state !== "active");
  const types = (page.events.types ?? []).filter((t) => validTarget(t.name));
  const subscriber = (page.channels.subscribers ?? [])[0] ?? "";
  const chosen = normalTypes(choice.types);
  const covered = choice.destination && chosen.length > 0 ? coveredBy(page, me, choice.destination, chosen) : null;
  const values = { destination: choice.destination, types: chosen.join(",") };
  const id = "p02-propose-sub";

  const update = (next: { destination: string; types: string[] }) => {
    setChoice(next);
    keepDraft("integrations.subscription_propose", webhooksTarget, { destination: next.destination, types: normalTypes(next.types).join(",") });
  };

  if (!usable && !noRight && !refused && !lost && !blocked) return null;
  return (
    <section className="p02-section p02-propose-sub" aria-labelledby="p02-propose-sub-title">
      <h3 id="p02-propose-sub-title">{k.proposeSubTitle(l)}</h3>
      <RowNote l={l} diagnostics={diagnostics(webhooksTarget)}
        kept={lost ? r.noLongerHeld(l, lost) : refused ? k.kept(l, "propose_subscription") : ""}
        blockers={noRight ? blockerLines(l, "no_right", "propose_subscription") : !usable && blocked ? [plain(k.subBlocker(l, blocked))] : []}
        focus={focusNote}
      />
      {usable && !open && (
        <div className="live-buttons">
          <button type="button" className="live-secondary" aria-expanded={open === "propose"} data-opener={mark("propose")} onClick={() => toggle("propose")}>{k.proposeSubTitle(l)}</button>
        </div>
      )}
      {open && usable && (
        <>
          <CommandForm
            {...shared} command="integrations.subscription_propose" target={webhooksTarget} values={values} untitled
            title={k.confirmSub(l)}
            fieldIds={{ destination: id + "-destination-first", types: id + "-types-first" }}
            choices={
              <>
                <dl className="live-facts"><div><dt>{k.subscriberIs(l)}</dt><dd><Known l={l} value={subscriber} /></dd></div></dl>
                <fieldset className="p02-choices">
                  <legend>{k.destinationIs(l)}</legend>
                  {active.map((d, n) => (
                    <label key={d.id} className="p02-choice">
                      <input
                        type="radio" name={id + "-destination"} id={n === 0 ? id + "-destination-first" : undefined} checked={choice.destination === d.id}
                        onChange={() => update({ ...choice, destination: d.id })}
                      />
                      <span>{ltr(d.id)}</span>
                    </label>
                  ))}
                  {inactive.length > 0 && (
                    <ul className="p02-inactive">
                      {inactive.map((d) => <li key={d.id}>{ltr(d.id)} — {d.state === "inactive" || d.state === "revoked" ? k.inactiveListed(l) : <Code l={l} value={d.state} say={k.destinationState} />}</li>)}
                    </ul>
                  )}
                </fieldset>
                <fieldset className="p02-choices">
                  <legend>{k.eventTypes(l)}</legend>
                  {types.map((t, n) => (
                    <label key={t.name} className="p02-choice">
                      <input
                        type="checkbox" id={n === 0 ? id + "-types-first" : undefined} checked={choice.types.includes(t.name)}
                        onChange={(e) => update({ ...choice, types: e.target.checked ? [...choice.types, t.name] : choice.types.filter((x) => x !== t.name) })}
                      />
                      <span>{ltr(t.name)}</span>
                    </label>
                  ))}
                  <p className="service-hint">{k.typesHint(l, chosen.length)}</p>
                </fieldset>
              </>
            }
            choiceProblems={() => {
              const out = subscribeProblems(page, choice.destination, chosen).map((p) => ({
                field: p === "destination" ? id + "-destination-first" : id + "-types-first",
                message: p === "destination" ? k.chooseDestination(l) : p === "types_none" ? k.chooseTypes(l) : k.tooManyTypes(l),
              }));
              if (out.length === 0 && covered && !kept) out.push({ field: id + "-destination-first", message: covered.id === "" ? k.subBlocker(l, "unknown_list") : covered.own ? k.coveredOwn(l, covered.id) : k.coveredOther(l, covered.id) });
              return out;
            }}
            facts={
              <>
                <div><dt>{k.subscriberIs(l)}</dt><dd><Known l={l} value={subscriber} /></dd></div>
                <div><dt>{k.destinationIs(l)}</dt><dd><Known l={l} value={choice.destination} /></dd></div>
                <div><dt>{k.eventTypes(l)}</dt><dd><Types l={l} types={chosen} /></dd></div>
              </>
            }
            impact={k.proposeSubImpact(l)}
            verb={k.proposeSubVerb(l)}
            done={(fields, _, snapshot) => (duplicateIn(snapshot, fields.subscription ?? "") ? k.duplicateSub(l, fields.subscription ?? "") : k.proposedSubDone(l, fields.subscription || k.unknown(l)))}
            onClose={close}
          />
        </>
      )}
    </section>
  );
}

export function ChannelsView(props: Shared & { section: string }) {
  const { section, ...shared } = props;
  const { l, page } = shared;
  const list = page.channels.subscriptions;
  return (
    <>
      {section === "configured" ? (
        <>
          <Destinations {...shared} />
          <section className="p02-section p02-subscriptions" aria-labelledby="p02-subscriptions-title">
            <h3 id="p02-subscriptions-title">{k.subscriptionsTitle(l)}</h3>
            <ListOr l={l} items={list} empty={k.noSubscriptions(l)}>
              {(rows) => <ul className="p02-list">{rows.map((s, i) => <SubRow key={s.id || i} s={s} {...shared} />)}</ul>}
            </ListOr>
          </section>
        </>
      ) : section === "not_configured" ? (
        <p className="live-empty">{k.sectionEmpty(l, "channels")}</p>
      ) : null}
      <ProposeSub {...shared} />
    </>
  );
}
