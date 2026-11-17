import { counted, plurals } from "./live-codes";
import { k } from "./live-copy";
import { artifactOf, rowId, runNumber } from "./live-model";
import { Code, Num, Stamp, href, ltr } from "./live-format";
import type { Shared } from "./LiveCommand";

export function DecisionsView({ l, page }: Shared) {
  const d = page.decisions;
  const t = d.tombstones;
  return (
    <>
      <dl className="p04-facts p04-facts-5 p04-decision-facts">
        <div><dt>{k.factWatermark(l)}</dt><dd><Num l={l} n={d.watermark} /></dd></div>
        <div><dt>{k.factLedger(l)}</dt><dd><Code l={l} value={d.ledger} say={k.ledger} /></dd></div>
        <div><dt>{k.factEpoch(l)}</dt><dd><Code l={l} value={d.epoch} say={k.connection} /></dd></div>
        <div><dt>{k.factDurable(l)}</dt><dd><Code l={l} value={d.durable_commit} say={k.connection} /></dd></div>
        <div><dt>{k.factRestoreCheck(l)}</dt><dd><Code l={l} value={d.restore_check} say={k.connection} /></dd></div>
      </dl>
      <section className="p04-records" aria-labelledby="p04-records-title">
        <h3 id="p04-records-title">{k.deletionRecords(l)}</h3>
        <p className="p04-small">
          {t.window === null ? k.unknown(l) : counted(l, t.window, plurals.window, { t: t.total === null ? k.unknown(l) : String(t.total) })}
          {t.more === true && <> {k.newest50(l)}</>}
          {t.more === null && <> {k.maybeMore(l)}</>}
        </p>
        {t.rows === null ? (
          <p className="live-empty">{k.unknown(l)}</p>
        ) : t.rows.length === 0 ? (
          <p className="live-empty">{t.total === 0 ? k.noRecords(l) : k.unknown(l)}</p>
        ) : (
          <div className="live-table">
            <table className="p04-table p04-records-table">
              <caption className="visually-hidden">{k.recordsCaption(l)}</caption>
              <colgroup><col className="p04-c10" /><col className="p04-c24" /><col className="p04-c18" /><col className="p04-c16" /><col className="p04-c16" /><col className="p04-c16" /></colgroup>
              <thead>
                <tr>
                  <th scope="col" className="p04-num">{k.colSequence(l)}</th>
                  <th scope="col">{k.colItem(l)}</th>
                  <th scope="col">{k.colClass(l)}</th>
                  <th scope="col">{k.colRun(l)}</th>
                  <th scope="col">{k.colRecorded(l)}</th>
                  <th scope="col">{k.colDecision(l)}</th>
                </tr>
              </thead>
              <tbody>
                {t.rows.map((x, i) => {
                  const n = runNumber(x.run);
                  const found = x.artifact ? artifactOf(page, x.artifact) : undefined;
                  return (
                    <tr key={x.sequence ?? "s" + i} id={x.sequence === null ? undefined : rowId("seq", String(x.sequence))}>
                      <td className="p04-first p04-num"><Num l={l} n={x.sequence} /></td>
                      <td data-label={k.colItem(l)}>
                        {!x.artifact ? k.unknown(l) : found ? <a className="p04-link" href={href(l, "expiry", { class: found.group.class, artifact: x.artifact })}>{ltr(x.artifact)}</a> : ltr(x.artifact)}
                      </td>
                      <td data-label={k.colClass(l)}><Code l={l} value={x.class} say={k.className} /></td>
                      <td data-label={k.colRun(l)}>
                        {x.run && n && x.other_scope !== true ? <a className="p04-link" href={href(l, "deletion", { run: n })}>{ltr(x.run.label)}</a>
                          : x.other_scope === true ? <span className="p04-small-text">{k.runOutside(l)}</span>
                            : x.run || x.other_scope === null ? k.unknown(l) : k.none(l)}
                      </td>
                      <td data-label={k.colRecorded(l)}><Stamp l={l} at={x.at} /></td>
                      <td data-label={k.colDecision(l)}><Code l={l} value={x.decision} say={k.connection} /></td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
        <p className="p04-small live-quiet">{k.skipNote(l)}</p>
      </section>
    </>
  );
}
