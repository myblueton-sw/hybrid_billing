import type { Locale } from "../../../../shared/types";
import type { BuildRow, ResultRow } from "../../../../service/release-evidence";
import { AdminError } from "../../../../service/admin";
import { useOlderRows } from "../../../../shared/older-rows";
import { k } from "./live-copy";
import { moduleWords, outcomeWords, plurals, reasonWords } from "./live-codes";
import { commitPattern, requirementByTest, requirementNumber, rowId } from "./live-model";
import { Absent, Code, Digest, Fixed, State, Known, ListOr, MoreLine, Num, Stamp, href, ltr } from "./live-format";
import type { Shared } from "./LiveCommand";
import { CountedText } from "./LiveRequirements";

function CommitText({ l, value }: { l: Locale; value: string }) {
  return commitPattern.test(value) ? ltr(value, true) : <Absent>{k.malformed(l)}</Absent>;
}

function ResultBody({ x, ...shared }: Shared & { x: ResultRow }) {
  const { l, page } = shared;
  const r = x.namespace && x.test ? requirementByTest(page, x.namespace, x.test) : undefined;
  const n = r ? requirementNumber(r.target) : "";
  return (
    <tbody className="p05-row">
      <tr>
        <td className="p05-first">
          <span className="p05-row-title">{n ? <a className="p05-link" href={href(l, "requirements", { requirement: n })}>{ltr(x.test)}</a> : <Known l={l} value={x.test} />}</span>
          <span className="p05-line p05-small"><Known l={l} value={x.namespace} /></span>
        </td>
        <td data-label={k.colBuild(l)}><Known l={l} value={x.build} /></td>
        <td data-label={k.colOutcome(l)}><State l={l} value={x.latest_outcome} table={outcomeWords} say={k.outcome} locator="test.outcome" /></td>
        <td data-label={k.colCounted(l)}>
          <span className="p05-line"><CountedText l={l} v={x.counted} /></span>
          {x.counted === false && (
            <span className="p05-line p05-small">{x.not_counted_reason ? <Code l={l} value={x.not_counted_reason} table={reasonWords} say={k.reason} /> : <Absent>{k.reasonUnknown(l)}</Absent>}</span>
          )}
        </td>
        <td className="p05-num" data-label={k.passes(l)}><Num l={l} n={x.passes} /></td>
        <td className="p05-num" data-label={k.failures(l)}><Num l={l} n={x.failures} /></td>
        <td data-label={k.colHarness(l)}><Known l={l} value={x.harness} wrap /></td>
        <td data-label={k.colRecordedAt(l)}><Stamp l={l} at={x.at} /></td>
      </tr>
      <tr className="p05-follow">
        <td colSpan={8}>
          <dl className="p05-digest"><div><dt>{k.evidenceDigest(l)}</dt><dd><Digest l={l} value={x.evidence} /></dd></div></dl>
        </td>
      </tr>
    </tbody>
  );
}

export function ArtifactsView(shared: Shared) {
  const { l, page, selection } = shared;
  const a = page.artifacts;
  const facts = a.results.rows === null || a.results_next === null || a.results.total === null ? null : { total: a.results.total, next: a.results_next };
  const older = useOlderRows<ResultRow>(page.revision, facts, a.results.rows, shared.api.evidenceResults, (e) => {
    if (e instanceof AdminError && e.failure === "denied") { void shared.onReload(); return true; }
    if (e instanceof AdminError && e.failure === "unauthenticated") { shared.onSignedOut(); return true; }
    return false;
  }, false);
  const failed = older.status === "failed";
  const buildFacts = a.builds.rows === null || a.builds_next === null || a.builds.total === null ? null : { total: a.builds.total, next: a.builds_next };
  const builds = useOlderRows<BuildRow>(page.revision, buildFacts, a.builds.rows, shared.api.evidenceBuilds, (e) => {
    if (e instanceof AdminError && e.failure === "denied") { void shared.onReload(); return true; }
    if (e instanceof AdminError && e.failure === "unauthenticated") { shared.onSignedOut(); return true; }
    return false;
  }, false, (b) => b.order);
  const buildsFailed = builds.status === "failed";
  const nc: [string, string][] = [[k.files(l), a.files], [k.logs(l), a.logs], [k.environment(l), a.environment], [k.content(l), a.content]];
  return (
    <>
      <p className="p05-lead">{k.artifactsLead(l)}</p>
      <dl className="p05-nc-strip">
        {nc.map(([label, v]) => <div key={label}><dt>{label}</dt><dd><Fixed l={l} value={v} locator="artifacts" /></dd></div>)}
      </dl>
      <div className="p05-regs">
        <section className="p05-reg" aria-labelledby="p05-builds-title">
          <h3 id="p05-builds-title">{k.buildsTitle(l)}</h3>
          {builds.cut && <p className="p05-small p05-muted p05-builds-count" role="status" ref={builds.countRef} tabIndex={-1}>{builds.all ? k.buildsAll(l, builds.total) : k.buildsShown(l, builds.rows?.length ?? 0, builds.total)}</p>}
          {builds.reset && <p className="p05-small p05-builds-reset" role="status">{k.buildsReset(l)}</p>}
          {builds.cut && !builds.more && !builds.all && builds.status === "idle" && <p className="p05-small p05-builds-moved" role="status">{k.buildsMoved(l)}</p>}
          <ListOr l={l} items={builds.rows} empty={k.noBuilds(l)}>
            {(rows) => (
              <div className="live-table">
                <table className="p05-table p05-small-reg p05-builds">
                  <caption className="visually-hidden">{k.buildsTitle(l)}</caption>
                  <thead><tr><th scope="col">{k.colBuild(l)}</th><th scope="col" className="p05-w70">{k.colCommit(l)}</th><th scope="col" className="p05-num">{k.colOrder(l)}</th></tr></thead>
                  {rows.map((b, i) => {
                    const current = selection.build !== "" && selection.build === b.build;
                    return (
                      <tbody key={b.order ?? b.build + i} className={"p05-row" + (current ? " live-selected" : "")} id={b.build ? rowId("build", b.build) : undefined} aria-current={current || undefined}>
                        <tr>
                          <td className="p05-first"><Known l={l} value={b.build} />{current && <span className="p05-tag p05-selected-tag" aria-hidden="true">{k.selected(l)}</span>}</td>
                          <td data-label={k.colCommit(l)}><CommitText l={l} value={b.commit} /></td>
                          <td className="p05-num p05-nobr" data-label={k.colOrder(l)}>{b.order === null ? <Absent>{k.unknown(l)}</Absent> : k.registered(l, String(b.order))}</td>
                        </tr>
                      </tbody>
                    );
                  })}
                </table>
              </div>
            )}
          </ListOr>
          {buildFacts === null && <MoreLine l={l} more={a.builds.more} total={a.builds.total} forms={plurals.showing50} />}
          {builds.more && (
            <div className="p05-req-more p05-builds-more">
              {buildsFailed && <p role="alert">{k.buildsFailed(l)}</p>}
              <button type="button" className="live-secondary p05-req-more-button p05-builds-more-button" aria-disabled={builds.status === "loading" || undefined} onClick={builds.load}>
                {buildsFailed ? k.resultsTryAgain(l) : builds.status === "loading" ? k.resultsLoading(l) : k.buildsMore(l)}
              </button>
            </div>
          )}
        </section>
        <section className="p05-reg" aria-labelledby="p05-harnesses-title">
          <h3 id="p05-harnesses-title">{k.harnessesTitle(l)}</h3>
          <ListOr l={l} items={a.harnesses.rows} empty={k.noHarnesses(l)}>
            {(rows) => (
              <div className="live-table">
                <table className="p05-table p05-small-reg p05-harnesses">
                  <caption className="visually-hidden">{k.harnessesTitle(l)}</caption>
                  <thead><tr><th scope="col" className="p05-w30">{k.colRegister(l)}</th><th scope="col" className="p05-w45">{k.colId(l)}</th><th scope="col" className="p05-w25">{k.colRevocation(l)}</th></tr></thead>
                  <tbody>
                    {rows.map((h, i) => (
                      <tr key={h.id + i}>
                        <td className="p05-first"><Code l={l} value={h.module} table={moduleWords} say={k.module} /></td>
                        <td data-label={k.colId(l)}><Known l={l} value={h.id} wrap /></td>
                        <td data-label={k.colRevocation(l)}><Fixed l={l} value={h.revocation} locator="revocation" /></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </ListOr>
          <MoreLine l={l} more={a.harnesses.more} total={a.harnesses.total} forms={plurals.showing50} />
        </section>
      </div>
      <section className="p05-section p05-list" aria-labelledby="p05-results-title">
        <h3 id="p05-results-title">{k.resultsTitle(l)}</h3>
        {older.cut && <p className="p05-small p05-muted p05-results-count" role="status" ref={older.countRef} tabIndex={-1}>{older.all ? k.resultsAll(l, older.total) : k.resultsShown(l, older.rows?.length ?? 0, older.total)}</p>}
        {older.reset && <p className="p05-small p05-results-reset" role="status">{k.resultsReset(l)}</p>}
        {older.cut && !older.more && !older.all && older.status === "idle" && <p className="p05-small p05-results-moved" role="status">{k.resultsMoved(l)}</p>}
        <ListOr l={l} items={older.rows} empty={k.noResults(l)}>
          {(rows) => (
            <div className="live-table">
              <table className="p05-table p05-results">
                <caption className="visually-hidden">{k.resultsTitle(l)}</caption>
                <thead>
                  <tr>
                    <th scope="col" className="p05-r-test">{k.colTestId(l)}</th>
                    <th scope="col" className="p05-r-build">{k.colBuild(l)}</th>
                    <th scope="col" className="p05-r-outcome">{k.colOutcome(l)}</th>
                    <th scope="col" className="p05-r-counted">{k.colCounted(l)}</th>
                    <th scope="col" className="p05-r-num p05-num">{k.passes(l)}</th>
                    <th scope="col" className="p05-r-num p05-num">{k.failures(l)}</th>
                    <th scope="col" className="p05-r-harness">{k.colHarness(l)}</th>
                    <th scope="col" className="p05-r-at">{k.colRecordedAt(l)}</th>
                  </tr>
                </thead>
                {rows.map((x, i) => <ResultBody key={x.number ?? x.test + x.build + i} x={x} {...shared} />)}
              </table>
            </div>
          )}
        </ListOr>
        {facts === null && <MoreLine l={l} more={a.results.more} total={a.results.total} forms={plurals.showing50} />}
        {older.more && (
          <div className="p05-req-more p05-results-more">
            {failed && <p role="alert">{k.resultsFailed(l)}</p>}
            <button type="button" className="live-secondary p05-req-more-button p05-results-more-button" aria-disabled={older.status === "loading" || undefined} onClick={older.load}>
              {failed ? k.resultsTryAgain(l) : older.status === "loading" ? k.resultsLoading(l) : k.resultsMore(l)}
            </button>
          </div>
        )}
      </section>
    </>
  );
}
