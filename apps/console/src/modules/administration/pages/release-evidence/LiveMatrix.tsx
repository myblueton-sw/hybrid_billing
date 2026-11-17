import type { Locale } from "../../../../shared/types";
import type { Candidate, CountOf, EvidencePage, PathRow, Problem } from "../../../../service/release-evidence";
import { k } from "./live-copy";
import { categories, categoryWords, counted, pathStateWords, plurals } from "./live-codes";
import { blockersUnexpected, pathState, proofUnexpected, requirementByTarget, requirementNumber, rowId, timeKnown } from "./live-model";
import { Absent, Code, DetailShell, Fixed, State, EvidenceLink, Gap, Known, Label, ListOr, MoreLine, Num, Stamp, Template, Unexpected, gapOf, href, ltr } from "./live-format";
import type { Shared } from "./LiveCommand";
import { ProblemWords, StateText, Subject, TestBlocks } from "./LiveRequirements";

function CountText({ l, c, template }: { l: Locale; c: CountOf; template?: string }) {
  if (!c) return <Absent>{k.unknown(l)}</Absent>;
  return <Template text={template ?? k.countOf(l)} slots={{ count: <Num l={l} n={c.count} />, of: <Num l={l} n={c.of} /> }} />;
}

function TestCell({ l, p, candidate }: { l: Locale; p: PathRow; candidate: Candidate }) {
  const c = p.columns.test;
  if (!c) return <Gap l={l} why={gapOf(candidate)} />;
  const failing = p.columns.failing;
  return (
    <>
      <span className="p05-line"><CountText l={l} c={c} template={k.testedOf(l)} /></span>
      <span className={"p05-line" + (failing !== null && failing > 0 ? " p05-weight" : "")}>{failing === null ? <Gap l={l} why={gapOf(candidate)} /> : counted(l, failing, plurals.failing)}</span>
    </>
  );
}

function ProofCell({ l, p, candidate, full = false }: { l: Locale; p: PathRow; candidate: Candidate; full?: boolean }) {
  const c = p.columns.customer_proof;
  if (!c) return <Gap l={l} why={gapOf(candidate)} />;
  if (proofUnexpected(c)) return <Unexpected l={l} code="customer_proof.count" />;
  return (
    <>
      <span className="p05-line"><CountText l={l} c={c} /></span>
      <span className="p05-line p05-small p05-muted">{p.customer_proof_intake !== "not_connected" ? <Unexpected l={l} code="intake" /> : full ? k.intakeFull(l) : k.intakeShort(l)}</span>
    </>
  );
}

function LastObserved({ l, p, candidate }: { l: Locale; p: PathRow; candidate: Candidate }) {
  if (p.last_observed === null) return candidate === null ? <Absent>{k.notEvaluated(l)}</Absent> : <>{k.noResultForPath(l)}</>;
  if (!timeKnown(p.last_observed)) return <Absent>{k.unknown(l)}</Absent>;
  return <Stamp l={l} at={p.last_observed} />;
}

export function BlockerList({ l, list, candidate }: { l: Locale; list: Problem[] | null; candidate: Candidate }) {
  if (list === null) return <Gap l={l} why={gapOf(candidate)} />;
  if (blockersUnexpected(list)) return <Unexpected l={l} code="path.blockers" />;
  return (
    <ul className="p05-problems">
      {list.map((p, i) => (
        <li key={i}>
          <ProblemWords l={l} p={p} />
          <span className="p05-line p05-small"><Subject l={l} p={p} /></span>
        </li>
      ))}
    </ul>
  );
}

function BlockerCount({ l, list, candidate }: { l: Locale; list: Problem[] | null; candidate: Candidate }) {
  if (list === null) return <Gap l={l} why={gapOf(candidate)} />;
  if (blockersUnexpected(list)) return <Unexpected l={l} code="path.blockers" />;
  return <strong>{counted(l, list.length, plurals.problems)}</strong>;
}

function PathStateText({ l, p }: { l: Locale; p: PathRow }) {
  const s = pathState(p);
  if (s.kind === "unexpected") return <Unexpected l={l} code={s.code} />;
  return <State l={l} value={s.code} table={pathStateWords} say={k.pathStateWord} locator="path.state" />;
}

function RequirementRefs({ l, page, p }: { l: Locale; page: EvidencePage; p: PathRow }) {
  const sample = page.requirements.origin === "sample_seed";
  return (
    <>
      <ListOr l={l} items={p.requirements.rows} empty={k.none(l)}>
        {(rows) => (
          <ul className="p05-refs-list">
            {rows.map((x, i) => {
              const n = x.target ? requirementNumber(x.target) : "";
              return (
                <li key={x.label + i}>
                  {n ? <a className="p05-link" href={href(l, "requirements", { requirement: n })}><Label value={x.label} /></a> : <Label value={x.label} />}
                  <span className="p05-line p05-small"><StateText l={l} state={x.state} candidate={page.candidate} sample={sample} /></span>
                </li>
              );
            })}
          </ul>
        )}
      </ListOr>
      <MoreLine l={l} more={p.requirements.more} total={p.requirements.total} forms={plurals.showing50} />
    </>
  );
}

function ComboFacts({ l, p }: { l: Locale; p: PathRow }) {
  return (
    <dl className="p05-combo">
      <div><dt>{k.provider(l)}</dt><dd><Known l={l} value={p.provider} /></dd></div>
      <div><dt>{k.contract(l)}</dt><dd><Known l={l} value={p.contract} /></dd></div>
      <div><dt>{k.role(l)}</dt><dd><Known l={l} value={p.role} /></dd></div>
      <div><dt>{k.version(l)}</dt><dd><Known l={l} value={p.version} /></dd></div>
      <div><dt>{k.issuance(l)}</dt><dd><Known l={l} value={p.issuance} /></dd></div>
      <div><dt>{k.template(l)}</dt><dd>{p.template === null ? k.noTemplate(l) : <Known l={l} value={p.template} />}</dd></div>
    </dl>
  );
}

function PathBody({ p, reduced, ...shared }: Shared & { p: PathRow; reduced: boolean }) {
  const { l, page, selection, mode } = shared;
  const c = page.candidate;
  const current = selection.path !== "" && selection.path === p.id;
  const evidenceId = rowId("evidence-path", p.id);
  const link = p.id ? <EvidenceLink l={l} label={p.id} id={evidenceId} to={href(l, "matrix", { path: p.id })} controls={mode === "split" && selection.path ? "p05-detail" : undefined} /> : null;
  const title = (
    <>
      <span className="p05-row-title">{p.id ? ltr(p.id) : <Absent>{k.unknown(l)}</Absent>}</span>
      {current && <span className="p05-tag p05-selected-tag" aria-hidden="true">{k.selected(l)}</span>}
    </>
  );
  if (reduced) {
    return (
      <tbody className={"p05-row" + (current ? " live-selected" : "")} id={p.id ? rowId("path", p.id) : undefined} aria-current={current || undefined}>
        <tr>
          <td className="p05-first">
            {title}
            <span className="p05-line p05-small"><Known l={l} value={p.provider} /> · <Known l={l} value={p.issuance} /></span>
            <span className="p05-line"><PathStateText l={l} p={p} /></span>
            <span className="p05-line">{link}</span>
          </td>
          <td><TestCell l={l} p={p} candidate={c} /></td>
          <td><ProofCell l={l} p={p} candidate={c} /></td>
          <td><BlockerCount l={l} list={p.blockers} candidate={c} /></td>
        </tr>
      </tbody>
    );
  }
  return (
    <tbody className={"p05-row" + (current ? " live-selected" : "")} id={p.id ? rowId("path", p.id) : undefined} aria-current={current || undefined}>
      <tr>
        <td className="p05-first">
          {title}
          <ComboFacts l={l} p={p} />
          <span className="p05-line"><PathStateText l={l} p={p} /></span>
          <span className="p05-line">{link}</span>
        </td>
        <td data-label={k.colRequirementIds(l)}><RequirementRefs l={l} page={page} p={p} /></td>
        <td data-label={k.colDesign(l)}><CountText l={l} c={p.columns.design} /></td>
        <td data-label={k.colImplementation(l)}>
          <span className="p05-line"><CountText l={l} c={p.columns.implementation} /></span>
          <span className="p05-line p05-small p05-muted">{k.withCommit(l)}</span>
        </td>
        <td data-label={k.colTest(l)}>
          <TestCell l={l} p={p} candidate={c} />
          <span className="p05-fold p05-small"><span className="p05-line p05-weight">{k.colLastObserved(l)}</span><LastObserved l={l} p={p} candidate={c} /></span>
        </td>
        <td data-label={k.colCustomerProof(l)}><ProofCell l={l} p={p} candidate={c} /></td>
        <td className="p05-last" data-label={k.colLastObserved(l)}><LastObserved l={l} p={p} candidate={c} /></td>
        <td data-label={k.colBlockers(l)}><BlockerList l={l} list={p.blockers} candidate={c} /></td>
      </tr>
    </tbody>
  );
}

function PathDetail({ l, page, p, mode }: { l: Locale; page: EvidencePage; p: PathRow; mode: "split" | "step" }) {
  const c = page.candidate;
  const recon = p.reconciliations;
  return (
    <DetailShell l={l} mode={mode} title={ltr(p.id)} sub={<><Known l={l} value={p.provider} /> · <Known l={l} value={p.issuance} /></>} closeTo={href(l, "matrix")} opener={rowId("evidence-path", p.id)}>
      <dl className="p05-sum">
        <div><dt>{k.provider(l)}</dt><dd><Known l={l} value={p.provider} /></dd></div>
        <div><dt>{k.contract(l)}</dt><dd><Known l={l} value={p.contract} /></dd></div>
        <div><dt>{k.role(l)}</dt><dd><Known l={l} value={p.role} /></dd></div>
        <div><dt>{k.version(l)}</dt><dd><Known l={l} value={p.version} /></dd></div>
        <div><dt>{k.issuance(l)}</dt><dd><Known l={l} value={p.issuance} /></dd></div>
        <div><dt>{k.template(l)}</dt><dd>{p.template === null ? k.noTemplate(l) : <Known l={l} value={p.template} />}</dd></div>
        <div><dt>{k.pathState(l)}</dt><dd><PathStateText l={l} p={p} /></dd></div>
        <div className="p05-sum-wide"><dt>{k.colRequirementIds(l)}</dt><dd><RequirementRefs l={l} page={page} p={p} /></dd></div>
        <div><dt>{k.colDesign(l)}</dt><dd><CountText l={l} c={p.columns.design} /></dd></div>
        <div><dt>{k.colImplementation(l)}</dt><dd><CountText l={l} c={p.columns.implementation} /> <span className="p05-small p05-muted">{k.withCommit(l)}</span></dd></div>
        <div><dt>{k.colTest(l)}</dt><dd><TestCell l={l} p={p} candidate={c} /></dd></div>
        <div><dt>{k.colLastObserved(l)}</dt><dd><LastObserved l={l} p={p} candidate={c} /></dd></div>
        <div className="p05-sum-wide"><dt>{k.colCustomerProof(l)}</dt><dd><ProofCell l={l} p={p} candidate={c} full /></dd></div>
        <div className="p05-sum-wide"><dt>{k.colBlockers(l)}</dt><dd><BlockerList l={l} list={p.blockers} candidate={c} /></dd></div>
        <div className="p05-sum-wide"><dt>{k.supportSince(l)}</dt><dd>{k.supportSinceValue(l)}</dd></div>
      </dl>
      <h3>{k.testCasesTitle(l)}</h3>
      <ListOr l={l} items={p.requirements.rows} empty={k.none(l)}>
        {(rows) => (
          <div className="p05-cases">
            {rows.map((x, i) => {
              const r = requirementByTarget(page, x.target);
              return (
                <div key={x.label + i} className="p05-case">
                  <h4><Label value={x.label} /></h4>
                  {!r ? <p><Template text={k.testCasesMissing(l)} slots={{ label: <Label value={x.label} /> }} /></p>
                    : r.tests === null ? <p><Gap l={l} why={gapOf(c)} /></p>
                      : <ListOr l={l} items={r.tests.rows} empty={k.noTests(l)}>{(tests) => <TestBlocks l={l} rows={tests} />}</ListOr>}
                </div>
              );
            })}
          </div>
        )}
      </ListOr>
      <h3>{k.categoriesTitle(l)}</h3>
      {p.categories === null ? <p><Gap l={l} why={gapOf(c)} /></p> : (
        <dl className="p05-cats">
          {categories.map((key) => {
            const v = p.categories?.[key];
            return (
              <div key={key}>
                <dt><Code l={l} value={key} table={categoryWords} say={k.category} /></dt>
                <dd>{v === true ? k.catTrue(l) : v === false ? k.catFalse(l) : <Absent>{k.unknown(l)}</Absent>}</dd>
              </div>
            );
          })}
        </dl>
      )}
      <h3>{k.reconTitle(l)}</h3>
      <ListOr l={l} items={recon.rows} empty={k.noRecon(l)}>
        {(rows) => (
          <ul className="p05-plain p05-recon">
            {rows.map((x, i) => (
              <li key={x.period + i}>
                {x.period ? ltr(x.period) : <Absent>{k.unknown(l)}</Absent>} · {x.matched === null ? <Absent>{k.unknown(l)}</Absent> : x.matched ? k.matched(l) : <strong>{k.notMatched(l)}</strong>} · {x.differences === null ? <Absent>{k.unknown(l)}</Absent> : counted(l, x.differences, plurals.differences)}
              </li>
            ))}
          </ul>
        )}
      </ListOr>
      <MoreLine l={l} more={recon.more} total={recon.total} forms={plurals.showingNewest12} />
      <p className="p05-small p05-muted">{k.reconNote(l)}</p>
      <h3>{k.ncTitle(l)}</h3>
      <ul className="p05-nc">
        <li><strong>{k.activation(l)}</strong> <Fixed l={l} value={p.activation} locator="activation" /></li>
        <li><strong>{k.supportDecision(l)}</strong> <Fixed l={l} value={p.support_decision} locator="support_decision" /></li>
        <li><strong>{k.customerProofIntake(l)}</strong> <Fixed l={l} value={p.customer_proof_intake} locator="intake" /></li>
      </ul>
    </DetailShell>
  );
}

export function MatrixView(shared: Shared) {
  const { l, page, selection, mode } = shared;
  const list = page.matrix.paths;
  const selected = selection.path ? (list.rows ?? []).find((p) => p.id === selection.path) : undefined;
  if (selected && mode === "step") return <PathDetail l={l} page={page} p={selected} mode="step" />;
  const open = !!selected;
  const reduced = open && mode === "split";
  const blocked = (list.rows ?? []).some((p) => p.blockers === null || p.blockers.length > 0 || pathState(p).kind === "unexpected" || p.state !== "active");
  return (
    <>
      <p className="p05-lead">{k.matrixLead(l)}</p>
      {blocked && <p className="p05-lead"><strong>{k.blockedLead(l)}</strong> {k.blockedRest(l)}</p>}
      <p className="p05-note p05-intake">{(list.rows ?? []).every((p) => p.customer_proof_intake === "not_connected") ? k.intakeFull(l) : <Unexpected l={l} code="intake" />}</p>
      {!open && <p className="p05-hint">{k.hintMatrix(l)}</p>}
      <div className="p05-split" data-open={reduced ? "true" : "false"}>
        <div className="p05-list">
          <ListOr l={l} items={list.rows} empty={k.noPaths(l)}>
            {(rows) => (
              <div className="live-table">
                <table className={"p05-table p05-matrix" + (reduced ? " p05-reduced" : " p05-full")}>
                  <caption className="visually-hidden">{k.matrixCaption(l)}</caption>
                  <thead>
                    {reduced ? (
                      <tr>
                        <th scope="col" className="p05-w40">{k.colCombination(l)}</th>
                        <th scope="col" className="p05-w24">{k.colTest(l)}</th>
                        <th scope="col" className="p05-w16">{k.colCustomerProof(l)}</th>
                        <th scope="col" className="p05-w20">{k.colBlockers(l)}</th>
                      </tr>
                    ) : (
                      <tr>
                        <th scope="col" className="p05-m-combo">{k.colCombination(l)}</th>
                        <th scope="col" className="p05-m-reqs">{k.colRequirementIds(l)}</th>
                        <th scope="col" className="p05-m-design">{k.colDesign(l)}</th>
                        <th scope="col" className="p05-m-impl">{k.colImplementation(l)}</th>
                        <th scope="col" className="p05-m-test">{k.colTest(l)}</th>
                        <th scope="col" className="p05-m-proof">{k.colCustomerProof(l)}</th>
                        <th scope="col" className="p05-m-last p05-last">{k.colLastObserved(l)}</th>
                        <th scope="col" className="p05-m-block">{k.colBlockers(l)}</th>
                      </tr>
                    )}
                  </thead>
                  {rows.map((p, i) => <PathBody key={p.id || "p" + i} p={p} reduced={reduced} {...shared} />)}
                </table>
              </div>
            )}
          </ListOr>
          <MoreLine l={l} more={list.more} total={list.total} forms={plurals.showing50} />
        </div>
        {reduced && selected && <PathDetail l={l} page={page} p={selected} mode="split" />}
      </div>
      <section className="p05-section" aria-labelledby="p05-nc-title">
        <h3 id="p05-nc-title">{k.ncTitle(l)}</h3>
        <ul className="p05-nc">
          <li><strong>{k.linkEvidence(l)}</strong> <Absent>{k.ncTitle(l)}</Absent></li>
          <li><strong>{k.recordSupport(l)}</strong> <Absent>{k.ncTitle(l)}</Absent></li>
          <li><strong>{k.requestActivation(l)}</strong> <Absent>{k.ncTitle(l)}</Absent></li>
        </ul>
      </section>
    </>
  );
}
