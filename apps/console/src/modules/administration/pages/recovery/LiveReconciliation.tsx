import type { Locale } from "../../../../shared/types";
import type { ReconciliationRow } from "../../../../service/recovery";
import { k } from "./live-copy";
import { highWaterOf, more, noRestorePoint, rowId, setOf } from "./live-model";
import { Code, Known, ListOr, ltr } from "./live-format";
import type { Shared } from "./LiveCommand";

function Externals({ l, r, noPoint }: { l: Locale; r: ReconciliationRow; noPoint: boolean }) {
  return (
    <section aria-label={k.externalsTitle(l)}>
      <h4>{k.externalsTitle(l)}</h4>
      <ListOr l={l} items={r.externals} empty={noPoint ? k.noPoint(l) : k.noExternals(l)}>
        {(list) => (
          <div className="live-table">
            <table className="p03-table">
              <colgroup><col className="p03-c40" /><col className="p03-c30" /><col className="p03-c30" /></colgroup>
              <thead><tr><th scope="col">{k.colSystem(l)}</th><th scope="col">{k.colKind(l)}</th><th scope="col">{k.colStatus(l)}</th></tr></thead>
              <tbody>
                {list.map((x, n) => (
                  <tr key={x.id || n}>
                    <td className="p03-first"><Known l={l} value={x.id} /></td>
                    <td data-label={k.colKind(l)}><Code l={l} value={x.kind} say={k.writerKind} /></td>
                    <td data-label={k.colStatus(l)}><span className="p03-tag"><Code l={l} value={x.status} say={k.externalStatus} /></span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
    </section>
  );
}

function Targets({ l, r }: { l: Locale; r: ReconciliationRow }) {
  const list = r.unknown_targets;
  return (
    <section aria-label={k.targetsTitle(l)}>
      <h4>{k.targetsTitle(l)}</h4>
      {list !== null && more(list.length, r.unknown_total) && <p className="p03-small live-quiet">{k.showingOf(l, r.unknown_total ?? 0)}</p>}
      {list !== null && list.length === 0 && r.unknown_total !== 0 ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : (
        <ListOr l={l} items={list} empty={k.noTargets(l)}>
          {(rows) => (
            <div className="live-table">
              <table className="p03-table">
                <colgroup><col className="p03-c28" /><col className="p03-c28" /><col className="p03-c14" /><col className="p03-c14" /><col className="p03-c16" /></colgroup>
                <thead>
                  <tr>
                    <th scope="col">{k.colKey(l)}</th><th scope="col">{k.colSeries(l)}</th><th scope="col" className="p03-num">{k.colNumber(l)}</th>
                    <th scope="col">{k.colFenced(l)}</th><th scope="col" className="p03-num">{k.colHighWater(l)}</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((x, n) => {
                    const hw = highWaterOf(x.series, x.high_water);
                    return (
                      <tr key={x.key || n} className="p03-target">
                        <td className="p03-first p03-wrap"><Known l={l} value={x.key} wrap /></td>
                        <td data-label={k.colSeries(l)} className="p03-wrap">{x.series ? ltr(x.series, true) : k.noSeries(l)}</td>
                        <td data-label={k.colNumber(l)} className="p03-num">{x.number === null ? k.unknown(l) : <bdi dir="ltr">{x.number}</bdi>}</td>
                        <td data-label={k.colFenced(l)}>{x.fenced === null ? k.unknown(l) : <span className="p03-tag">{k.fenced(l, String(x.fenced))}</span>}</td>
                        <td data-label={k.colHighWater(l)} className="p03-num">{hw.kind === "number" ? <bdi dir="ltr">{hw.n}</bdi> : hw.kind === "no_series" ? k.noSeries(l) : k.unknown(l)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </ListOr>
      )}
      {r.other_scope_present === null ? <p className="p03-small live-quiet">{k.otherTargetsUnknown(l)}</p> : r.other_scope_present ? <p className="p03-small live-quiet">{k.otherTargets(l)}</p> : null}
    </section>
  );
}

export function ReconciliationView({ l, page, selection }: Shared) {
  return (
    <>
      <ListOr l={l} items={page.reconciliation.rows} empty={k.reconciliationEmpty(l)}>
        {(rows) => (
          <>
            {rows.map((r, n) => {
              const current = !!r.installation && selection.installation === r.installation;
              const set = setOf(page, r.installation);
              const noPoint = !!set && noRestorePoint(set);
              const head = "p03-rec-h-" + n;
              return (
                <article
                  key={r.installation || n} className={"p03-install" + (current ? " live-selected" : "")} id={r.installation ? rowId("rec", r.installation) : undefined}
                  aria-labelledby={head} aria-current={current || undefined}
                >
                  <div className="p03-install-head"><h3 id={head}><Known l={l} value={r.installation} /></h3></div>
                  <div className="p03-recon-grid">
                    <Externals l={l} r={r} noPoint={noPoint} />
                    <Targets l={l} r={r} />
                  </div>
                  <dl className="p03-facts p03-facts-2">
                    <div><dt>{k.skipped(l)}</dt><dd>{r.skipped === null ? k.unknown(l) : <bdi dir="ltr">{r.skipped}</bdi>}</dd></div>
                    <div><dt>{k.reconciler(l)}</dt><dd><Code l={l} value={r.reconciler} say={k.connection} /></dd></div>
                  </dl>
                </article>
              );
            })}
          </>
        )}
      </ListOr>
      <p className="p03-small live-quiet p03-closing">{k.reconcilerNote(l)}</p>
    </>
  );
}

