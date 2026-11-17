import type { Locale } from "../../../../shared/types";
import type { RecoveryPage, SetRow } from "../../../../service/recovery";
import { k } from "./live-copy";
import { chunksOf, evidenceOf, historyOf, noRestorePoint, reconciledOf, reconciliationOf, rowId, watermarkOf } from "./live-model";
import { plain } from "./live-copy";
import { Code, Known, ListOr, Num, Rich, Stamp, Template, href, ltr } from "./live-format";
import type { Shared } from "./LiveCommand";

function Evidence({ l, row }: { l: Locale; row: SetRow }) {
  const e = evidenceOf(row);
  if (e.kind === "not_applicable") return <>{k.notApplicable(l)}</>;
  if (e.kind === "no_point") return <>{k.noPoint(l)}</>;
  if (e.kind === "unknown") return <>{k.unknown(l)}</>;
  return (
    <>
      {k.verifiedOf(l, e.verified, e.total)}
      {e.mismatched > 0 && <span className="live-sub">{k.mismatched(l, e.mismatched)}</span>}
      {e.pending > 0 && <span className="live-sub">{k.notChecked(l, e.pending)}</span>}
    </>
  );
}

function Watermark({ l, page, id }: { l: Locale; page: RecoveryPage; id: string }) {
  const w = watermarkOf(page, id);
  if (w.kind === "unknown") return <>{k.unknown(l)}</>;
  if (w.kind === "outside") return <>{k.outsideRecovery(l)}</>;
  if (w.kind === "mode") return <Code l={l} value={w.mode} say={k.modeWord} />;
  return (
    <>
      {w.total === null ? k.unknown(l) : w.total === 0 ? k.noEventsPending(l) : k.eventsPending(l, w.total)}
      {w.lost === null ? <span className="live-sub">{k.lostUnknown(l)}</span> : w.lost > 0 ? <span className="live-sub">{k.declaredLost(l, w.lost)}</span> : null}
    </>
  );
}

function Reconciled({ l, page, row }: { l: Locale; page: RecoveryPage; row: SetRow }) {
  const r = reconciledOf(page, row);
  if (r.kind === "none") return <>{k.notApplicable(l)}</>;
  if (r.kind === "no_point") return <>{k.noPoint(l)}</>;
  if (r.kind === "unknown") return <>{k.unknown(l)}</>;
  return <>{k.reconciledOf(l, r.reconciled, r.total)}</>;
}

function Generation({ l, row }: { l: Locale; row: SetRow }) {
  if (row.recovery.state !== "open") return <span className="p03-num"><Num l={l} n={row.generation} /></span>;
  return (
    <>
      <span className="live-sub">{k.installationGen(l)} <Num l={l} n={row.generation} /></span>
      <span className="live-sub">{k.pointGen(l)} {noRestorePoint(row) ? k.noPoint(l) : <Num l={l} n={row.recovery.point_generation} />}</span>
    </>
  );
}

function ChunkText({ l, status, list }: { l: Locale; status: string; list: number[] | null }) {
  const c = chunksOf(status, list);
  if (c.kind === "none") return <span className="live-quiet p03-small">{k.none(l)}</span>;
  if (c.kind === "unknown") return <>{k.unknown(l)}</>;
  if (c.kind === "count_differs") return <>{k.countDiffers(l)}</>;
  const text = c.first100 ? k.chunkFirst100(l) : c.indices.length === 1 ? k.chunkOne(l) : k.chunkMany(l);
  return <Template text={text} slots={{ i: ltr(c.indices.join(", "), true) }} />;
}

function PointFold({ l, row, open }: { l: Locale; row: SetRow; open: boolean }) {
  const r = row.recovery;
  const domains = r.domains;
  return (
    <details className="p03-fold" open={open || (domains ?? []).some((d) => d.status === "mismatch") || undefined}>
      <summary>{r.id ? <Rich text={k.pointFold(l, r.id)} /> : k.pointFold(l, k.unknown(l))}</summary>
      <div className="p03-point-grid">
        <div>
          <dl className="p03-point-facts">
            <div><dt>{k.phase(l)}</dt><dd><Code l={l} value={r.phase} say={k.phaseWord} /></dd></div>
            <div><dt>{k.backup(l)}</dt><dd>{noRestorePoint(row) ? k.noPoint(l) : <Known l={l} value={r.backup_reference} wrap />}</dd></div>
            <div><dt>{k.manifest(l)}</dt><dd>{noRestorePoint(row) ? k.noPoint(l) : <Known l={l} value={r.manifest_digest} wrap />}</dd></div>
          </dl>
          <p className="p03-small live-quiet">{k.noChecksums(l)}</p>
        </div>
        <section aria-label={k.domainChecks(l)}>
          <h4>{k.domainChecks(l)}</h4>
          <ListOr l={l} items={domains} empty={k.noPoint(l)}>
            {(list) => (
              <div className="live-table">
                <table className="p03-table">
                  <colgroup><col className="p03-c40" /><col className="p03-c24" /><col className="p03-c36" /></colgroup>
                  <thead><tr><th scope="col">{k.colDomain(l)}</th><th scope="col">{k.colStatus(l)}</th><th scope="col">{k.colChunks(l)}</th></tr></thead>
                  <tbody>
                    {list.map((d, n) => (
                      <tr key={d.domain || n}>
                        <td className="p03-first"><Code l={l} value={d.domain} say={k.domain} /></td>
                        <td data-label={k.colStatus(l)} className={d.status === "mismatch" ? "p03-strong-cell" : undefined}><Code l={l} value={d.status} say={k.domainStatus} /></td>
                        <td data-label={k.colChunks(l)}><ChunkText l={l} status={d.status} list={d.mismatch_chunks} /></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </ListOr>
        </section>
      </div>
    </details>
  );
}

function History({ l, row }: { l: Locale; row: SetRow }) {
  const h = historyOf(row);
  return (
    <section className="p03-history" aria-label={k.historyTitle(l)}>
      <h4>{k.historyTitle(l)}</h4>
      {h.kind === "more" && <p className="p03-small live-quiet">{k.historyMore(l, h.total)}</p>}
      {h.kind === "unknown" ? <p>{k.unknown(l)}</p> : (
      <ListOr l={l} items={row.history} empty={k.noHistory(l)}>
        {(list) => (
          <div className="live-table">
            <table className="p03-table">
              <colgroup><col className="p03-c20" /><col className="p03-c44" /><col className="p03-c12" /><col className="p03-c12" /><col className="p03-c12" /></colgroup>
              <thead>
                <tr>
                  <th scope="col">{k.colRecovery(l)}</th><th scope="col">{k.manifest(l)}</th><th scope="col" className="p03-num">{k.colGeneration(l)}</th>
                  <th scope="col">{k.colOutcome(l)}</th><th scope="col" className="p03-num">{k.colLost(l)}</th>
                </tr>
              </thead>
              <tbody>
                {list.map((h, n) => (
                  <tr key={h.recovery || n}>
                    <td className="p03-first"><Known l={l} value={h.recovery} /></td>
                    <td data-label={k.manifest(l)} className="p03-wrap"><Known l={l} value={h.manifest_digest} wrap /></td>
                    <td data-label={k.colGeneration(l)} className="p03-num"><Num l={l} n={h.generation} /></td>
                    <td data-label={k.colOutcome(l)}><span className="p03-tag"><Code l={l} value={h.outcome} say={k.outcome} /></span></td>
                    <td data-label={k.colLost(l)} className="p03-num"><Num l={l} n={h.lost_count} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
      )}
    </section>
  );
}

export function SetsView({ l, page, selection }: Shared) {
  const list = page.sets.rows;
  return (
    <ListOr l={l} items={list} empty={k.notConfigured(l)}>
      {(rows) => (
        <div className="live-table">
          <table className="p03-table p03-sets" aria-labelledby="p03-view-title">
            <colgroup>
              <col className="p03-c14" /><col className="p03-c8" /><col className="p03-c14" /><col className="p03-c10" />
              <col className="p03-c16" /><col className="p03-c12" /><col className="p03-c12" /><col className="p03-c14" />
            </colgroup>
            <thead>
              <tr>
                <th scope="col">{k.colInstallation(l)}</th><th scope="col">{k.colRestore(l)}</th><th scope="col">{k.colAsOf(l)}</th><th scope="col">{k.colGeneration(l)}</th>
                <th scope="col">{k.colEvidence(l)}</th><th scope="col">{k.colWatermark(l)}</th><th scope="col">{k.colExternal(l)}</th><th scope="col">{k.colLock(l)}</th>
              </tr>
            </thead>
            {rows.map((row, n) => {
              const id = row.installation;
              const current = !!id && selection.installation === id;
              const open = row.recovery.state === "open";
              return (
                <tbody key={id || n} className={"p03-group" + (current ? " live-selected" : "")} id={id ? rowId("set", id) : undefined}>
                  <tr aria-current={current || undefined}>
                    <td className="p03-first"><Known l={l} value={id} /></td>
                    <td data-label={k.colRestore(l)}>{open ? <Known l={l} value={row.recovery.id} /> : row.recovery.state === "none" ? k.recoveryState(l, "none") : <Code l={l} value={row.recovery.state} say={k.recoveryState} />}</td>
                    <td data-label={k.colAsOf(l)}>{!open ? k.dash() : noRestorePoint(row) ? k.noPoint(l) : <Stamp l={l} at={row.recovery.as_of} />}</td>
                    <td data-label={k.colGeneration(l)}><Generation l={l} row={row} /></td>
                    <td data-label={k.colEvidence(l)}><Evidence l={l} row={row} /></td>
                    <td data-label={k.colWatermark(l)}><Watermark l={l} page={page} id={id} /></td>
                    <td data-label={k.colExternal(l)}><Reconciled l={l} page={page} row={row} /></td>
                    <td data-label={k.colLock(l)}><span className="p03-tag"><Code l={l} value={row.state} say={k.publicLock} /></span></td>
                  </tr>
                  <tr className="p03-follow">
                    <td colSpan={8}>
                      {id && (
                        <nav className="p03-jumps" aria-label={plain(k.jumpsLabel(l, id))}>
                          <a className="p03-link" href={href(l, "isolation", { installation: id })}>{k.viewTitle(l, "isolation")}</a>
                          {reconciliationOf(page, id) && <a className="p03-link" href={href(l, "reconciliation", { installation: id })}>{k.viewTitle(l, "reconciliation")}</a>}
                          <a className="p03-link" href={href(l, "resume", { installation: id })}>{k.viewTitle(l, "resume")}</a>
                        </nav>
                      )}
                      <p className="p03-small live-quiet p03-later">
                        {k.notConnectedLine(l)}: <Code l={l} value={row.begin} say={k.connection} /> · {k.abandonLine(l)}: <Code l={l} value={row.abandon} say={k.connection} />
                      </p>
                      {open && <PointFold l={l} row={row} open={current} />}
                      <History l={l} row={row} />
                    </td>
                  </tr>
                </tbody>
              );
            })}
          </table>
        </div>
      )}
    </ListOr>
  );
}
