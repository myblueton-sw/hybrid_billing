import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { counted } from "../../../../shared/plural";
import type { PolicyPage, RevisionRow, TrustChange } from "../../../../service/admin";
import { w } from "../users/live-copy";
import { p } from "./live-copy";
import { duration, effectiveStep, methodsText, stale as pastWindow, stepText, when } from "./live-model";
import { ty } from "./trust-copy";
import {
  actions, blocked, canPropose, checkItems, conditions, conditionText, currentVersion, enrolled, epochText, findingReason, firstSentence, grouped, nextStep, rightsList, stale,
  stateSentence, unavailable, versionOf, whyNot, type ActionKey, type Condition,
} from "./trust-model";

export const ltr = (v: string, mono = true) => <bdi dir="ltr" className={mono ? "live-mono" : undefined}>{v}</bdi>;
export const actor = (v: string) => <bdi dir="ltr" className="live-mono m02-actor">{v}</bdi>;

function Person({ l, id }: { l: Locale; id: string }) {
  return id ? actor(id) : <span className="live-quiet">{ty.notRecorded(l)}</span>;
}

function Lead({ text }: { text: string }) {
  const [lead, rest] = firstSentence(text);
  return <><p><strong>{lead}</strong></p>{rest && <p>{rest}</p>}</>;
}

export function Conditions({ l, page, onOpen }: { l: Locale; page: PolicyPage; onOpen: (id: string) => void }) {
  const list = conditions(page);
  if (!list.length) return null;
  const at = (c: Condition) => (c.kind === "reset" ? when(l, page.trust.reset_at) : "");
  const unknown = page.changes.find((c) => c.state === "activation_unknown");
  return (
    <section className="m02-conditions" aria-labelledby="m02-notices">
      <h2 id="m02-notices" className="visually-hidden">{ty.notices(l)}</h2>
      {list.map((c) => {
        const parts = conditionText(l, c, at(c));
        return (
          <div className="m02-condition" key={c.kind} data-condition={c.kind}>
            {parts.length === 1 ? <Lead text={parts[0]} /> : <><p><strong>{parts[0]}</strong></p>{parts.slice(1).map((x) => <p key={x}>{x}</p>)}</>}
            {c.kind === "hold" && page.trust.hold && (
              <p className="m02-note">
                {ty.since(l, when(l, page.trust.hold.since))}
                {unknown && <> · <button type="button" className="service-button-link m02-open" onClick={() => onOpen(unknown.id)}>{ty.change(l, unknown.number)}</button></>}
              </p>
            )}
          </div>
        );
      })}
    </section>
  );
}

export function ShortConditions({ l, page, skipHold = false }: { l: Locale; page: PolicyPage; skipHold?: boolean }) {
  return <>{conditions(page).filter((c) => !(skipHold && c.kind === "hold")).map((c) => <p key={c.kind} className="m02-condition-short"><strong>{firstSentence(conditionText(l, c, "")[0])[0]}</strong></p>)}</>;
}

export function CurrentRules({ l, page, onReload, loading }: { l: Locale; page: PolicyPage; onReload: () => void; loading: boolean }) {
  const cur = page.policy;
  const tr = page.trust;
  const fact = (label: string, value: ReactNode) => <div><dt>{label}</dt><dd>{value}</dd></div>;
  return (
    <section className="m02-current" aria-labelledby="m02-current">
      <div className="m02-toolbar">
        <div className="live-heading">
          <h2 id="m02-current" tabIndex={-1}>{ty.currentTitle(l)}</h2>
          <p className="live-meta">{w.loadedAt(l, when(l, page.now))}</p>
        </div>
        <div className="live-tools"><button type="button" className="live-secondary" onClick={onReload} aria-busy={loading || undefined}>{w.reload(l)}</button></div>
      </div>
      <div className="m02-current-grid">
        {cur ? (
          <div className="m02-group">
            <h3>{ty.rules(l)}</h3>
            <dl className="m02-facts">
              {fact(p.methods(l), methodsText(l, cur))}
              {fact(p.mfa(l), cur.mfa === "none" ? p.notRequired(l) : ltr(cur.mfa))}
              {fact(p.maxAge(l), duration(l, cur.max_age_seconds))}
              {fact(p.stepUp(l), stepText(l, cur))}
              {fact(p.effective(l), duration(l, effectiveStep(page)))}
            </dl>
          </div>
        ) : <p className="live-empty">{ty.noPolicy(l)}</p>}
        <div className="m02-group">
          <h3>{ty.trust(l)}</h3>
          <dl className="m02-facts">
            {fact(ty.version(l), <bdi dir="ltr" className="live-mono m02-epoch">{currentVersion(page)}</bdi>)}
            {fact(ty.policyRevision(l), cur ? String(tr.policy_number) : <span className="live-quiet">{ty.notRecorded(l)}</span>)}
            {fact(ty.origin(l), ty.originWord(l, tr.origin))}
            {fact(ty.stricterFields(l), tr.stricter_fields == null ? ty.fieldsUnknown(l) : tr.stricter_fields.length === 0 ? ty.noStricterFields(l) : tr.stricter_fields.map((field, index) => <span className="live-sub" key={index}>{index > 0 && <br />}{field === "methods" ? p.methods(l) : field === "mfa" ? p.mfa(l) : field === "max_age" ? p.maxAge(l) : field === "step_up" ? p.stepUp(l) : ltr(field)}</span>))}
            {fact(ty.activatedAt(l), tr.activated_at ? when(l, tr.activated_at) : <span className="live-quiet">{ty.notRecorded(l)}</span>)}
            {fact(ty.proposedBy(l), <Person l={l} id={tr.proposer} />)}
            {fact(ty.validatedBy(l), <Person l={l} id={tr.validator} />)}
            {fact(ty.approvedBy(l), <Person l={l} id={tr.approver} />)}
            {fact(ty.activatedBy(l), <Person l={l} id={tr.activator} />)}
          </dl>
        </div>
      </div>
      <p className="m02-ledger">{ty.ledgerLine(l, counted(l, page.ledger.records, ty.records()), page.ledger.generation)}</p>
      {tr.origin === "carried" && tr.carried_from && <p className="m02-note">{ty.originWord(l, "carried")}: {ltr(tr.carried_from)}</p>}
      {page.act_until && <p className="live-meta live-window">{pastWindow(page) ? p.staleLine(l) : p.actUntil(l, when(l, page.act_until))}</p>}
    </section>
  );
}

export function ChangesHead({ l, page, onPropose }: { l: Locale; page: PolicyPage; onPropose: () => void }) {
  const notKept = page.changes.some((c) => c.earlier_run) || page.trust.origin === "ledger_replay" || page.trust.origin === "carried";
  return (
    <div className="m02-changes-head">
      <div className="m02-toolbar">
        <h2 id="m02-changes" tabIndex={-1}>{ty.changesTitle(l)}</h2>
        {canPropose(page) && <button type="button" className="live-primary m02-propose" onClick={onPropose}>{ty.proposeOpen(l)}</button>}
      </div>
      {enrolled(page) ? (
        <p className="m02-holders">{ty.holders(l, page.holders.qualifying_people)}</p>
      ) : (
        <div className="m02-enrolment" data-enrolment="blocked">
          <p>
            <strong>{ty.enrolmentLead(l)}</strong>
            {page.holders.missing.length > 0 && <> {ty.missingRoles(l, page.holders.missing.map((r) => ty.role(l, r)).join(l === "ar" ? "، " : ", "))}</>}
            {" "}{ty.enrolmentRest(l, page.holders.qualifying_people)}
          </p>
        </div>
      )}
      <p className="m02-rights">{rightsList(l, page)}</p>
      {unavailable(page) && <p className="m02-unavailable-line">{(() => { const [lead, rest] = firstSentence(ty.unavailableLine(l)); return <><strong>{lead}</strong>{rest && " " + rest}</>; })()}</p>}
      {notKept && <p className="m02-note">{ty.notKept(l, when(l, page.started_at))}</p>}
    </div>
  );
}

function BasedOn({ l, c }: { l: Locale; c: TrustChange }) {
  return <>{ltr(versionOf(c.old_epoch))}<br /><span className="m02-sub">{ty.policyNumber(l, c.old_policy)}</span></>;
}

function StateCell({ l, c }: { l: Locale; c: TrustChange }) {
  const strong = c.state === "activation_unknown" || (c.state === "draft" && blocked(c));
  return (
    <>
      <span className={"m02-state-word" + (strong ? " live-strong" : "")}>{ty.state(l, c.state)}</span>
      {stale(c) && <><br /><span className="m02-sub live-strong">{ty.evidenceChanged(l)}</span></>}
    </>
  );
}

function Rows({ l, rows, title, selected, reduced, onOpen }: {
  l: Locale; rows: TrustChange[]; title: string; selected: string; reduced: boolean; onOpen: (id: string) => void;
}) {
  return (
    <div className="security-table-scroll m02-table-wrap">
      <table className={"m02-table" + (reduced ? " m02-reduced" : "")} role="table">
        <caption className="visually-hidden">{ty.caption(l, title)}</caption>
        {!reduced && <colgroup><col className="m02-c1" /><col className="m02-c2" /><col className="m02-c3" /><col className="m02-c4" /><col className="m02-c5" /></colgroup>}
        <thead role="rowgroup" className={reduced ? "visually-hidden" : undefined}>
          <tr role="row">
            <th scope="col" role="columnheader">{ty.colChange(l)}</th>
            {!reduced && <><th scope="col" role="columnheader">{ty.colState(l)}</th><th scope="col" role="columnheader">{ty.proposedBy(l)}</th><th scope="col" role="columnheader">{ty.colBased(l)}</th><th scope="col" role="columnheader">{ty.colNext(l)}</th></>}
          </tr>
        </thead>
        <tbody role="rowgroup">
          {rows.map((c) => {
            const opener = (
              <>
                <button type="button" className="m02-open" data-change={c.id} onClick={() => onOpen(c.id)}>{ty.change(l, c.number)}</button>
                {selected === c.id && <span className="m02-tag" aria-hidden="true">{ty.selected(l)}</span>}
                {c.earlier_run && <><br /><span className="m02-sub">{ty.earlierRun(l, when(l, c.run_started_at))}</span></>}
              </>
            );
            return (
              <tr key={c.id} role="row" data-state={c.state} aria-current={selected === c.id ? "true" : undefined}>
                {reduced ? (
                  <td role="cell">{opener}<br /><StateCell l={l} c={c} /><br /><span className="m02-sub">{nextStep(l, c)}</span></td>
                ) : (
                  <>
                    <td role="cell">{opener}</td>
                    <td role="cell" data-label={ty.colState(l)}><StateCell l={l} c={c} /></td>
                    <td role="cell" data-label={ty.proposedBy(l)}>{actor(c.proposer)}</td>
                    <td role="cell" data-label={ty.colBased(l)}><BasedOn l={l} c={c} /></td>
                    <td role="cell" data-label={ty.colNext(l)}>{nextStep(l, c)}</td>
                  </>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export function Lists({ l, page, selected, reduced, onOpen }: { l: Locale; page: PolicyPage; selected: string; reduced: boolean; onOpen: (id: string) => void }) {
  const g = grouped(page);
  return (
    <div className="m02-lists">
      <h3 id="m02-open-list">{counted(l, g.open.length, ty.openCount())}</h3>
      {g.open.length ? <Rows l={l} rows={g.open} title={ty.open(l)} selected={selected} reduced={reduced} onOpen={onOpen} /> : <p className="live-empty">{ty.noOpen(l)}</p>}
      <h3 id="m02-closed-list">{counted(l, g.closed.length, ty.closedCount())}</h3>
      {g.closed.length ? <Rows l={l} rows={g.closed} title={ty.closed(l)} selected={selected} reduced={reduced} onOpen={onOpen} /> : <p className="live-empty">{ty.noClosed(l)}</p>}
    </div>
  );
}

export function Compare({ l, current, next, older }: { l: Locale; current: RevisionRow | null; next: RevisionRow; older?: boolean }) {
  const none = ty.notRecorded(l);
  const rows: [string, string, string][] = [
    [p.maxAge(l), current ? duration(l, current.max_age_seconds) : none, duration(l, next.max_age_seconds)],
    [p.stepUp(l), current ? stepText(l, current) : none, stepText(l, next)],
    [p.mfa(l), current ? (current.mfa === "none" ? p.notRequired(l) : current.mfa) : none, next.mfa === "none" ? p.notRequired(l) : next.mfa],
  ];
  return (
    <>
      <div className="security-table-scroll m02-compare-table">
        <table role="table">
          <colgroup><col style={{ inlineSize: "36%" }} /><col style={{ inlineSize: "32%" }} /><col style={{ inlineSize: "32%" }} /></colgroup>
          <thead role="rowgroup"><tr role="row"><th scope="col" role="columnheader">{p.setting(l)}</th><th scope="col" role="columnheader">{p.current(l)}</th><th scope="col" role="columnheader">{p.proposed(l)}</th></tr></thead>
          <tbody role="rowgroup">
            {rows.map(([label, before, after]) => (
              <tr key={label} role="row">
                <td role="cell">{label}</td>
                <td role="cell" data-label={p.current(l)}>{before}</td>
                <td role="cell" data-label={p.proposed(l)} className={before !== after ? "live-strong" : ""}>{after}{before !== after && <> {p.changed(l)}</>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {older && <p className="m02-sub">{ty.mayBeOlder(l)}</p>}
    </>
  );
}

function Findings({ l, c }: { l: Locale; c: TrustChange }) {
  const v = c.validation;
  if (!v) return <p className="m02-result live-quiet">{ty.notValidated(l)}</p>;
  const items = new Map(v.items.map((i) => [i.item, i]));
  const k = v.items.filter((i) => i.result === "blocked").length;
  const info = (key: string) => items.get(key);
  return (
    <>
      {v.result === "blocked"
        ? <p className="m02-result live-strong">{ty.blockedAt(l, when(l, v.at), c.validator, k)}</p>
        : <ValidatedLine text={ty.validatedAt(l, when(l, v.at), "\u2063")} who={c.validator} />}
      <div className="security-table-scroll m02-findings-wrap">
        <table className="m02-findings" role="table">
          <caption className="visually-hidden">{ty.checks(l)}</caption>
          <colgroup><col style={{ inlineSize: "34%" }} /><col style={{ inlineSize: "22%" }} /><col style={{ inlineSize: "44%" }} /></colgroup>
          <thead role="rowgroup"><tr role="row"><th scope="col" role="columnheader">{ty.colCheck(l)}</th><th scope="col" role="columnheader">{ty.colResult(l)}</th><th scope="col" role="columnheader">{ty.colReason(l)}</th></tr></thead>
          <tbody role="rowgroup">
            {checkItems.map((key) => {
              const it = items.get(key);
              if (!it) return null;
              return (
                <tr key={key} role="row" data-item={key} data-result={it.result}>
                  <td role="cell">{ty.item(l, key)}</td>
                  <td role="cell" data-label={ty.colResult(l)} className={"m02-word" + (it.result === "blocked" ? " live-strong" : "")}>{ty.result(l, it.result)}</td>
                  {findingReason(l, it) || it.reason || it.detail ? (
                    <td role="cell" data-label={ty.colReason(l)}>
                      {findingReason(l, it)}
                      {(it.reason || it.detail) && <><br /><span className="m02-sub">{ltr(it.reason || it.detail)}</span></>}
                    </td>
                  ) : <td role="cell" />}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <dl className="m02-sum">
        {info("mfa_weakening") && <div><dt>{ty.item(l, "mfa_weakening")}</dt><dd className={info("mfa_weakening")?.reason === "weakens_mfa" ? "live-strong" : ""}>{ty.reason(l, info("mfa_weakening")?.reason ?? "")}</dd></div>}
        {info("affected_subjects") && <div><dt>{ty.item(l, "affected_subjects")}</dt><dd>{info("affected_subjects")?.detail}</dd></div>}
        {info("session_impact") && <div><dt>{ty.item(l, "session_impact")}</dt><dd>{info("session_impact")?.detail}</dd></div>}
      </dl>
    </>
  );
}

function Impact({ l, c }: { l: Locale; c: TrustChange }) {
  const impact = c.impact;
  if (!impact) {
    return (
      <section className="m02-impact" aria-labelledby={"m02-impact-" + c.number}>
        <h3 id={"m02-impact-" + c.number}>{ty.impactTitle(l)}</h3>
        <p className="live-strong">{ty.signsOut(l)}</p>
        <p className="live-quiet">{ty.impactNotComputed(l)}</p>
      </section>
    );
  }
  const ids = impact.affected_ids;
  return (
    <section className="m02-impact" aria-labelledby={"m02-impact-" + c.number}>
      <h3 id={"m02-impact-" + c.number}>{ty.impactTitle(l)}</h3>
      {impact.signs_out_everyone && <p className="live-strong">{ty.signsOut(l)}</p>}
      <dl className="m02-sum">
        <div><dt>{ty.members(l)}</dt><dd>{impact.members}</dd></div>
        <div><dt>{ty.sessionsNow(l)}</dt><dd>{impact.sessions}</dd></div>
      </dl>
      {ids.length > 0 && (
        <details className="m02-members">
          <summary>{ty.affected(l, impact.members)}</summary>
          <ul className="m02-ids">{ids.map((id) => <li key={id}>{actor(id)}</li>)}</ul>
          {impact.members > ids.length && <p className="m02-sub">{ty.showing(l, impact.members)}</p>}
        </details>
      )}
    </section>
  );
}

function History({ l, c }: { l: Locale; c: TrustChange }) {
  return (
    <section className="m02-history-section" aria-labelledby={"m02-history-" + c.number}>
      <h3 id={"m02-history-" + c.number}>{ty.history(l)}</h3>
      <ol className="m02-history">
        {c.history.map((h, i) => (
          <li key={i}>
            <span className="live-strong">{h.rationale === "approval_invalidated" ? ty.invalidated(l) : ty.state(l, h.state)}</span> · {actor(h.by)} · {when(l, h.at)}
            {h.rationale && h.rationale !== "approval_invalidated" && <><br /><span className="m02-free" dir="auto">{h.rationale}</span></>}
          </li>
        ))}
      </ol>
    </section>
  );
}

function Outcome({ l, c, onOpen, page }: { l: Locale; c: TrustChange; onOpen: (id: string) => void; page: PolicyPage }) {
  if (c.state === "activated" && c.activation) {
    return (
      <section className="m02-activation" aria-labelledby={"m02-act-" + c.number}>
        <dl className="m02-sum">
          <div><dt>{ty.newVersion(l)}</dt><dd>{ltr(versionOf(c.activation.epoch))}</dd></div>
          <div><dt>{ty.ledgerSequence(l)}</dt><dd>{c.activation.ledger_sequence}</dd></div>
          <div><dt>{ty.activatedAt(l)}</dt><dd>{when(l, c.activation.at)}</dd></div>
        </dl>
        {c.invalidation && (
          <>
            <h3 id={"m02-act-" + c.number}>{ty.invalidationTitle(l)}</h3>
            <div className="security-table-scroll">
              <table className="m02-invalidation" role="table">
                <colgroup><col style={{ inlineSize: "40%" }} /><col style={{ inlineSize: "20%" }} /><col style={{ inlineSize: "40%" }} /></colgroup>
                <thead role="rowgroup"><tr role="row"><th scope="col" role="columnheader">{ty.colItem(l)}</th><th scope="col" role="columnheader">{ty.colResult(l)}</th><th scope="col" role="columnheader">{ty.colDetail(l)}</th></tr></thead>
                <tbody role="rowgroup">
                  {c.invalidation.items.map((it) => (
                    <tr key={it.item} role="row" data-item={it.item}>
                      <td role="cell">{ty.progress(l, it.item)}</td>
                      <td role="cell" data-label={ty.colResult(l)} className={"m02-word" + (it.result === "in_progress" || it.result === "unknown" ? " live-strong" : "")}>{ty.result(l, it.result)}</td>
                      <td role="cell" data-label={ty.colDetail(l)}>
                        {it.item === "console_sessions" && <>{ty.sessionsEnded(l)}: {it.ended} · {ty.sessionsRemaining(l)}: {it.remaining}<br /></>}
                        {it.reason && ltr(it.reason)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </section>
    );
  }
  if (c.state === "activation_unknown") {
    return (
      <section className="m02-waiting" aria-label={ty.waitingTitle(l)}>
        <dl className="m02-sum">
          <div><dt>{ty.job(l)}</dt><dd>{ltr(c.job?.id ?? "")}</dd></div>
          <div><dt>{ty.jobState(l)}</dt><dd>{ty.waiting(l)}</dd></div>
          <div><dt>{ty.colReason(l)}</dt><dd>{ty.jobReason(l, c.job?.reason ?? "")} {ltr(c.job?.reason ?? "")}</dd></div>
        </dl>
      </section>
    );
  }
  if (c.state === "superseded") {
    const by = page.changes.find((x) => x.id === c.superseded_by);
    return (
      <dl className="m02-sum">
        <div>
          <dt>{ty.supersededBy(l)}</dt>
          <dd>
            {by ? <button type="button" className="service-button-link m02-open" onClick={() => onOpen(by.id)}>{ty.change(l, by.number)}</button> : ltr(c.superseded_by)}
            {c.superseded_epoch && <> · {ltr(versionOf(c.superseded_epoch))}</>}
          </dd>
        </div>
      </dl>
    );
  }
  return null;
}

export function actionLabel(l: Locale, k: ActionKey): string {
  return { activate: ty.activate, approve: ty.approve, validate: ty.validate, revise: ty.revise, reject: ty.reject, cancel: ty.cancel }[k](l);
}

export function Detail({ l, page, c, split, onClose, onAction, onOpen, headingRef }: {
  l: Locale; page: PolicyPage; c: TrustChange; split: boolean; onClose: () => void; onAction: (k: ActionKey, opener: HTMLElement) => void;
  onOpen: (id: string) => void; headingRef?: (el: HTMLHeadingElement | null) => void;
}) {
  const act = actions(c);
  const strongState = c.state === "activation_unknown" || c.state === "superseded" || (c.state === "draft" && blocked(c)) || stale(c);
  const button = (k: ActionKey, primary: boolean) => (
    <button key={k} type="button" className={primary ? "live-primary" : "live-secondary"} data-action={k} onClick={(e) => onAction(k, e.currentTarget)}>{actionLabel(l, k)}</button>
  );
  const isolated = page.ledger.state === "isolated" || page.ledger.state === "stopped";
  const fact = (label: string, value: ReactNode) => <div className="m02-wide"><dt>{label}</dt><dd>{value}</dd></div>;
  return (
    <div className="m02-detail-body" data-change={c.id} data-state={c.state}>
      <div className="m02-detail-head">
        <h2 className="m02-detail-title" tabIndex={-1} ref={headingRef}>
          {ty.change(l, c.number)}
          {c.earlier_run && <span className="m02-sub"> {ty.earlierRun(l, when(l, c.run_started_at))}</span>}
        </h2>
        {split && <button type="button" className="service-button-link m02-close" onClick={onClose}>{ty.close(l)}</button>}
      </div>
      {c.state === "activation_unknown" && split && page.trust.hold ? <p className="m02-state-line live-strong">{ty.holdSeeNotice(l)}</p>
        : c.state === "activation_unknown" ? <div className="m02-state-line"><Lead text={stateSentence(l, c)} /></div>
        : <p className={"m02-state-line" + (strongState ? " live-strong" : "") + (stale(c) ? " m02-stale" : "")}>{stateSentence(l, c)}</p>}
      <p className="m02-next">{ty.nextLabel(l)} {nextStep(l, c)}</p>
      {(act.primary || act.secondary.length > 0) && (
        <div className="m02-actions">
          {act.primary && button(act.primary, true)}
          {act.secondary.map((k) => button(k, false))}
        </div>
      )}
      {whyNot(l, c).map((x) => <p key={x} className="m02-why">{x}</p>)}
      <Outcome l={l} c={c} onOpen={onOpen} page={page} />
      <section className="m02-compare" aria-labelledby={"m02-settings-" + c.number}>
        <h3 id={"m02-settings-" + c.number}>{ty.settings(l)}</h3>
        <Compare l={l} current={page.policy} next={c.candidate} older={isolated} />
      </section>
      <section className="m02-validation" aria-labelledby={"m02-validation-" + c.number}>
        <h3 id={"m02-validation-" + c.number}>{ty.validationTitle(l)}</h3>
        <Findings l={l} c={c} />
      </section>
      {(c.state !== "activated" && c.state !== "rejected" && c.state !== "cancelled" && c.state !== "superseded" && c.state !== "activation_unknown") && <Impact l={l} c={c} />}
      <section className="m02-people" aria-labelledby={"m02-people-" + c.number}>
        <h3 id={"m02-people-" + c.number}>{ty.people(l)}</h3>
        <dl className="m02-sum">
          {[[ty.proposedBy(l), c.proposer], [ty.validatedBy(l), c.validator], [ty.approvedBy(l), c.approver], [c.state === "activation_unknown" ? ty.attemptedBy(l) : ty.activatedBy(l), c.activator]].map(([label, id]) => (
            <div key={label}><dt>{label}</dt><dd>{id ? actor(id) : <span className="live-quiet">{ty.notYet(l)}</span>}</dd></div>
          ))}
        </dl>
      </section>
      <History l={l} c={c} />
      <details className="m02-expert">
        <summary>{ty.technical(l)}</summary>
        <dl className="m02-sum">
          {fact(ty.changeId(l), ltr(c.id))}
          {fact(ty.changeRevision(l), String(c.change_revision))}
          {fact(ty.baseVersion(l), <>{ltr(versionOf(c.old_epoch))} ({ty.rawEpoch(l)} {ltr(c.old_epoch.value)})</>)}
          {fact(ty.basePolicy(l), String(c.old_policy))}
          {fact(ty.oldDigest(l), ltr(c.old_digest))}
          {fact(ty.newDigest(l), ltr(c.new_digest))}
          {fact(ty.changeDigest(l), ltr(c.change_digest))}
          {c.validation && fact(ty.evidenceDigest(l), ltr(c.validation.evidence_digest))}
          {c.activation && fact(ty.ledgerSequence(l), <>{c.activation.ledger_sequence} · {ltr(c.activation.watermark)}</>)}
          {c.superseded_by && fact(ty.supersededBy(l), <>{ltr(c.superseded_by)} · {ltr(epochText(c.superseded_epoch))}</>)}
        </dl>
      </details>
    </div>
  );
}

export function Revoked({ l, onReload }: { l: Locale; onReload: () => void }) {
  return (
    <div className="live-users">
      <section className="live-unavailable m02-revoked" aria-labelledby="m02-revoked">
        <h2 id="m02-revoked">{ty.revokedTitle(l)}</h2>
        <p>{ty.revoked(l)}</p>
        <button type="button" className="live-secondary" onClick={onReload}>{w.reload(l)}</button>
      </section>
    </div>
  );
}

function ValidatedLine({ text, who }: { text: string; who: string }) {
  const [before, after = ""] = text.split("\u2063");
  return <p className="m02-result">{before}{actor(who)}{after}</p>;
}
