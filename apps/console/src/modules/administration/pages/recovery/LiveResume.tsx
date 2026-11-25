import type { Locale } from "../../../../shared/types";
import type { ResumeRow } from "../../../../service/recovery";
import { eventOf } from "../../../../service/recovery";
import { RowsFoot, useRows } from "./LiveRows";
import { k } from "./live-copy";
import {
  alreadyProposed, basisContext, basisFrom, basisOf, blockerPolicy, duplicateIn, frozenTargetOk, frozenValues, generationNumber, installTarget, isolationOf, lostList, more,
  pendingNow, proposeNarrow, refusedNow, rowId, type Basis,
} from "./live-model";
import { Code, Known, ListOr, LostText, ModeImpact, Num, Stamp, href } from "./live-format";
import { CommandForm, RowNote, line, useOpen, type Blocker, type Shared } from "./LiveCommand";
import { Proposals } from "./LiveProposals";

function ProposeFacts({ l, b }: { l: Locale; b: Basis }) {
  const g = generationNumber(b);
  return (
    <>
      <div><dt>{k.factInstallation(l)}</dt><dd><Known l={l} value={b.installation} /></dd></div>
      <div><dt>{k.mode(l)}</dt><dd><Code l={l} value={b.mode} say={k.modeWord} /></dd></div>
      <div><dt>{k.factState(l)}</dt><dd><Code l={l} value={b.state} say={k.installState} /></dd></div>
      <div><dt>{k.colGeneration(l)}</dt><dd>{g === null ? k.unknown(l) : <bdi dir="ltr">{g}</bdi>}</dd></div>
      <div><dt>{k.lostSequences(l)}</dt><dd>{b.mode === "lock_release" ? k.none(l) : <LostText l={l} list={lostList(b)} />}</dd></div>
      <div className="p03-span"><dt>{k.digest(l)}</dt><dd><Known l={l} value={b.digest} wrap /></dd></div>
    </>
  );
}

function Propose({ r, ...shared }: Shared & { r: ResumeRow }) {
  const { l, page, canCommand } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<"propose">();
  const target = installTarget(r.installation);
  const pending = target ? pendingNow("recovery.propose_unlock", target) : undefined;
  const narrow = proposeNarrow(r);
  const usable = canCommand && !!target && ((r.can_propose_unlock && narrow === "") || (!!pending && frozenTargetOk("recovery.propose_unlock", target)));
  const blockers: Blocker[] = [];
  let link = "";
  if (!target) blockers.push(line("", k.narrow(l, "no_command")));
  else if (!usable && !r.can_propose_unlock && blockerPolicy(r.unlock_blocked, "propose_unlock", { mode: r.mode }) === "explain") {
    if (r.unlock_blocked === "already_proposed") {
      const a = alreadyProposed(page, r.installation);
      blockers.push(line(k.proposeUnlock(l), k.already(l, a?.number ?? "", a?.own ?? false)));
      link = a?.number ?? "";
    } else blockers.push(line(k.proposeUnlock(l), k.blocker(l, r.unlock_blocked)));
  } else if (!usable && canCommand && r.can_propose_unlock && narrow) blockers.push(line(k.proposeUnlock(l), k.narrow(l, narrow)));
  const kept = target && refusedNow("recovery.propose_unlock", target) ? k.kept(l, "recovery.propose_unlock") : "";
  const b = pending ? basisFrom(pending.context) : basisOf(page, r.installation);
  const frozen = frozenValues(pending);
  const values = { lost: frozen ? frozen.lost ?? "" : r.lost_digest };
  return (
    <div className="p03-actions">
      <RowNote l={l} diagnostics={diagnostics(target)}
        kept={kept} blockers={blockers} focus={focusNote}
        extra={link ? <p className="p03-goto"><a className="p03-link" href={href(l, "resume", { proposal: link })}>{k.goToProposal(l, link)}</a></p> : null}
      />
      {usable && !open && (
        <div className="live-buttons">
          <button type="button" className="live-secondary" aria-expanded={false} data-opener={mark("propose")} onClick={() => toggle("propose")}>{k.proposeUnlock(l)}</button>
        </div>
      )}
      {open && usable && (
        <CommandForm
          {...shared} command="recovery.propose_unlock" target={target} values={values}
          title={k.confirmPropose(l)}
          facts={<ProposeFacts l={l} b={b} />}
          impact={<ModeImpact l={l} b={b} lead />}
          verb={k.proposeVerb(l, r.installation)}
          context={basisContext(b)}
          done={(fields, state, snapshot) =>
            state !== "pending" ? k.resultUnknown(l, state) : duplicateIn(snapshot, fields.proposal ?? "") ? k.duplicate(l, fields.proposal ?? "") : k.proposedDone(l, fields.proposal || k.unknown(l))
          }
          onClose={close}
        />
      )}
    </div>
  );
}

function Events({ l, r, shared }: { l: Locale; r: ResumeRow; shared: Shared }) {
  const paged = r.pending_next !== null;
  const further = useRows(shared, r.installation, "pending", r.pending_events, r.pending_next, eventOf, (x) => (x.sequence === null ? "" : String(x.sequence)));
  const list = paged ? further.rows : r.pending_events;
  return (
    <section className="p03-events" aria-label={k.pendingEvents(l)}>
      <h4>{k.pendingEvents(l)}</h4>
      {!paged && list !== null && more(list.length, r.pending_total) && <p className="p03-small live-quiet">{k.showingOf(l, r.pending_total ?? 0)}</p>}
      {list !== null && list.length === 0 && r.pending_total !== 0 ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : (
        <ListOr l={l} items={list} empty={k.noPendingEvents(l)}>
          {(rows) => (
            <div className="live-table">
              <table className="p03-table">
                <colgroup><col className="p03-c14" /><col className="p03-c28" /><col className="p03-c28" /><col className="p03-c30" /></colgroup>
                <thead>
                  <tr><th scope="col" className="p03-num">{k.colSequence(l)}</th><th scope="col">{k.colKind(l)}</th><th scope="col">{k.colDomain(l)}</th><th scope="col">{k.colRecorded(l)}</th></tr>
                </thead>
                <tbody>
                  {rows.map((e, n) => (
                    <tr key={n} className="p03-event">
                      <td className="p03-first p03-num"><Num l={l} n={e.sequence} /></td>
                      <td data-label={k.colKind(l)}><Code l={l} value={e.kind} say={k.eventKind} /></td>
                      <td data-label={k.colDomain(l)}><Code l={l} value={e.domain} say={k.domain} /></td>
                      <td data-label={k.colRecorded(l)}><Stamp l={l} at={e.at} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </ListOr>
      )}
      <p className="p03-small live-quiet">{k.noEventKeys(l)}</p>
      {paged && <RowsFoot l={l} kind="events" rows={further} total={r.pending_total} moves more={k.rowsMoreEvents(l)} />}
    </section>
  );
}

export function ResumeView(shared: Shared) {
  const { l, page, selection } = shared;
  return (
    <>
      <ListOr l={l} items={page.resume.rows} empty={k.notConfigured(l)}>
        {(rows) => (
          <>
            {rows.map((r, n) => {
              const current = !!r.installation && selection.installation === r.installation;
              const iso = isolationOf(page, r.installation);
              const head = "p03-res-h-" + n;
              return (
                <article
                  key={r.installation || n} className={"p03-install" + (current ? " live-selected" : "")} id={r.installation ? rowId("res", r.installation) : undefined}
                  aria-labelledby={head} aria-current={current || undefined}
                >
                  <div className="p03-install-head">
                    <h3 id={head}><Known l={l} value={r.installation} /></h3>
                    {iso && <span className="p03-tag"><Code l={l} value={iso.state} say={k.installState} /></span>}
                  </div>
                  <dl className="p03-facts p03-facts-4">
                    <div><dt>{k.mode(l)}</dt><dd><Code l={l} value={r.mode} say={k.modeWord} /></dd></div>
                    <div><dt>{k.phase(l)}</dt><dd><Code l={l} value={r.phase} say={k.phaseWord} /></dd></div>
                    <div><dt>{k.colGeneration(l)}</dt><dd>{iso ? <Num l={l} n={iso.generation} /> : k.unknown(l)}</dd></div>
                    <div><dt>{k.lostSequences(l)}</dt><dd><LostText l={l} list={r.lost} /></dd></div>
                  </dl>
                  {r.mode === "none" && <p className="p03-mode-note">{k.modeNone(l)}</p>}
                  {r.mode === "lock_release" && <p className="p03-mode-note">{k.modeRelease(l)} <strong>{k.modeReleaseStrong(l)}</strong></p>}
                  {r.mode === "recovery_resume" && <Events l={l} r={r} shared={shared} />}
                  <dl className="p03-digest">
                    <div><dt>{k.digest(l)}</dt><dd><Known l={l} value={r.lost_digest} wrap /></dd></div>
                  </dl>
                  <Propose {...shared} r={r} />
                </article>
              );
            })}
          </>
        )}
      </ListOr>
      <Proposals {...shared} />
    </>
  );
}
